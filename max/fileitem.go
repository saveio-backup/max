package max

import (
	"bytes"
	"context"
	"fmt"
	fscontract "github.com/saveio/themis-go-sdk/fs"
	"github.com/saveio/themis/common"
	"github.com/saveio/themis/common/log"
	fs "github.com/saveio/themis/smartcontract/service/native/savefs"
	"github.com/saveio/themis/smartcontract/service/native/savefs/pdp"
	cid "gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
)

type FilePDPItem struct {
	FileHash       string
	FileInfo       *fs.FileInfo
	NextChalHeight uint32
	BlockHash      common.Uint256
	NextSubHeight  uint32
	BakParam       BakParam
	ExpireState    ExpireState
	FirstProve     bool
	PdpResult      []byte
	max            *MaxService
}

func (this *FilePDPItem) doPdpCalculation() error {
	if this == nil || this.FileInfo == nil {
		return fmt.Errorf("pdp cal item is invalid")
	}

	fileHash := this.FileHash
	fileInfo := this.FileInfo
	bakParm := this.BakParam
	blockHeight := this.NextChalHeight
	blockHash := this.BlockHash
	fsContract := this.getFsContract()

	log.Debugf("[doPdpCalculation] fileHash : %s, blockNum : %d, proveBlockNum : %d, fileProveParam : %v, hash : %d, height : %d, luckyNum :%d, bakNum : %d, badNodeWalletAddr : %s",
		fileHash, fileInfo.FileBlockNum, fileInfo.ProveBlockNum, fileInfo.FileProveParam, blockHash.ToHexString(), blockHeight,
		bakParm.LuckyNum, bakParm.BakHeight, bakParm.BakNum, bakParm.BadNodeWalletAddr.ToBase58())

	challenges := fsContract.GenChallenge(this.getAccountAddress(), blockHash,
		this.FileInfo.FileBlockNum, this.FileInfo.ProveBlockNum)

	log.Debugf("[doPdpCalculation] challenges : %v", challenges)

	proveData, err := this.doPdpCalculationForFile(challenges)
	if err != nil {
		return err
	}

	this.PdpResult = proveData
	return nil
}

func (this *FilePDPItem) onFailedPdpCalculation(err error) error {
	fileHash := this.FileHash
	log.Errorf("do pdp calculation for file %s error %s", fileHash, err)
	return this.getMaxService().deleteAndNotify(fileHash, PROVE_TASK_REMOVAL_REASON_PDP_CALCULATION)
}

func (this *FilePDPItem) doPdpSubmission() ([]byte, error) {
	if this == nil {
		log.Errorf("item for pdp submission is nil")
		return nil, fmt.Errorf("item for pdp submission is nil")
	}

	log.Debugf("pdpSubmission item: %+v", this)

	fsContract := this.getFsContract()

	fileHash := this.FileHash
	bakParam := this.BakParam
	proveData := this.PdpResult
	height := uint64(this.NextChalHeight)

	var err error
	var txHash []byte

	if bakParam.BakNum == 0 {
		txHash, err = fsContract.FileProve(fileHash, proveData, height)
	} else {
		txHash, err = fsContract.FileBackProve(fileHash, proveData, height,
			bakParam.LuckyNum, bakParam.BakHeight, bakParam.BakNum, bakParam.BadNodeWalletAddr)
	}
	if err != nil {
		log.Errorf("file prove error : %s bakNum : %d", err, bakParam.BakNum)
		return nil, err
	}

	log.Debugf("call fileProve for file %s success with txHash %s", fileHash, getTxHashString(txHash))
	return txHash, nil
}
func (this *FilePDPItem) onSuccessfulPdpSubmission() error {
	fsContract := this.getFsContract()
	fileHash := this.FileHash
	height := this.NextChalHeight

	max := this.getMaxService()
	proveDetails, err := fsContract.GetFileProveDetails(fileHash)
	log.Debugf("proveDetails for file %s: %+v", fileHash, proveDetails)
	if err != nil {
		log.Errorf("get prove details after success prove error : %s", err)
		// delete cached prove details to force getting prove details from contract for next prove
		// when proveDetail not found, fileInfo may have been deleted
		max.rpcCache.deleteProveDetails(fileHash)

		return fmt.Errorf("get prove details after success prove error : %s", err)
	} else {
		log.Debugf("try add prove details to cache after success prove")
		found := false
		address := this.getAccountAddress()
		for _, detail := range proveDetails.ProveDetails {
			if detail.WalletAddr.ToBase58() == address.ToBase58() {
				found = true
				break
			}
		}
		if found {
			log.Debugf("matching prove detail found, add prove detail to cache")
			max.rpcCache.addProveDetails(fileHash, proveDetails)
		} else {
			log.Debugf("no matching prove detail found, delete cached prove details")
			max.rpcCache.deleteProveDetails(fileHash)

			return fmt.Errorf("no matching prove detail found, delete cached prove details")
		}
	}

	if this.ExpireState == EXPIRE_LAST_PROVE {
		log.Infof("delete file and prove task for fileHash %s after last prove", fileHash)
		max.deleteAndNotify(fileHash, PROVE_TASK_REMOVAL_REASON_NORMAL)
		return nil
	}

	if this.FirstProve {
		// saves the first prove height to check if fileinfo is deleted then added agian
		err := max.saveProveTask(fileHash, 0, 0, 0,
			common.ADDRESS_EMPTY, uint64(height), this.FileInfo.FileProveParam)
		if err != nil {
			log.Errorf("saveProveTask for fileHash %s error : %s", fileHash, err)
			return err
		}
		log.Debugf("save prove task for fileHash %s with first prove height %d after first prove", fileHash, height)

		err = this.processForSectorProve()
		if err != nil {
			log.Errorf("processForSectorProve for fileHash %s error : %s", fileHash, err)
			return err
		}
	}
	return nil
}

func (this *FilePDPItem) processForSectorProve() error {
	max := this.getMaxService()

	proveLevel := this.FileInfo.ProveLevel
	fileHash := string(this.FileInfo.FileHash)
	blockCount := this.FileInfo.FileBlockNum
	blockSize := this.FileInfo.FileBlockSize

	sector, err := max.sectorManager.AddFile(proveLevel, fileHash, blockCount, blockSize)
	if err != nil {
		log.Errorf("addFileToSector for file %s error %s", fileHash, err)
		return err
	}

	sectorId := sector.GetSectorID()
	// if sector prove task not exist, create a sector prove task
	if !max.isSectorProveTaskExist(sectorId) {
		err := max.addSectorProveTask(sectorId)
		if err != nil {
			return err
		}
		log.Debugf("addProveTask for sector %d", sectorId)
	}

	// remove the task no more file prove needed after first success file prove
	// keep prove param in db for sector prove
	err = max.deleteProveTask(fileHash, false)
	if err != nil {
		log.Errorf("processForSectorProve, deleteProveTask for file %s error %s", fileHash, err)
		return nil
	}
	return nil
}

func (this *FilePDPItem) onFailedPdpSubmission(err error) error {
	// delete the task if first prove error
	if this.FirstProve {
		return this.getMaxService().deleteAndNotify(this.FileHash, err.Error())
	}
	return nil
}

func (this *FilePDPItem) getItemKey() string {
	return this.FileHash
}

func (this *FilePDPItem) getPdpCalculationHeight() uint32 {
	return this.NextChalHeight
}
func (this *FilePDPItem) getPdpSubmissionHeight() uint32 {
	return this.NextSubHeight
}

func (this *FilePDPItem) getMaxService() *MaxService {
	if this.max == nil {
		panic("max service is nil")
	}
	return this.max
}
func (this *FilePDPItem) doPdpCalculationForFile(challenges []pdp.Challenge) ([]byte, error) {
	fileHash := this.FileHash
	// get all cids
	rootCid, err := cid.Decode(fileHash)
	if err != nil {
		log.Errorf("[doPdpCalculation] Decode for fileHash %s error : %s", fileHash, err)
		return nil, err
	}

	cids, err := this.getMaxService().GetFileAllCids(context.TODO(), rootCid)
	if err != nil {
		log.Errorf("[doPdpCalculation] GetFileAllCids for rootCid %s error : %s", rootCid.String(), err)
		return nil, err
	}

	param := new(fs.ProveParam)
	reader := bytes.NewReader(this.FileInfo.FileProveParam)
	err = param.Deserialize(reader)
	if err != nil {
		log.Errorf("get prove param for file %s error %s", fileHash, err)
		return nil, fmt.Errorf("get prove param for file %s error %s", fileHash, err)
	}

	prover := pdp.NewPdp(0)

	err = this.initProverWithTags(prover, param, cids)
	if err != nil {
		log.Errorf("[doPdpCalculation] initProverWithTags for rootCid %s error : %s", rootCid.String(), err)
		return nil, err
	}

	tags, blocks, err := this.prepareForPdpCal(cids, challenges)
	if err != nil {
		return nil, err
	}

	return this.generateProve(prover, param, challenges, tags, blocks)
}

func (this *FilePDPItem) initProverWithTags(prover *pdp.Pdp, proveParam *fs.ProveParam, cids []*cid.Cid) error {
	fileHash := this.FileHash
	merkleNodes := make([]*pdp.MerkleNode, 0)
	for i, blockCid := range cids {
		tag, err := this.getMaxService().GetTag(blockCid.String(), fileHash, uint64(i))
		if err != nil {
			log.Errorf("[initProverWithTags] GetTag for file %s with index %d error: %s", fileHash, i, err)
			return fmt.Errorf("GetTag for file %s with index %d error: %s", fileHash, i, err)
		}

		node := pdp.InitNodeWithData(tag, uint64(i))
		merkleNodes = append(merkleNodes, node)
	}

	err := prover.InitMerkleTreeForFile(proveParam.FileID, merkleNodes)
	if err != nil {
		log.Errorf("[initProverWithTags] initMerkleTree for file %s error: %s", fileHash, err)
		return fmt.Errorf("initMerkleTree for file %s error: %s", fileHash, err)
	}
	return nil
}
func (this *FilePDPItem) prepareForPdpCal(cids []*cid.Cid, challenges []pdp.Challenge) (tags [][]byte, blocks [][]byte, err error) {
	var blockCid *cid.Cid
	var index uint64
	var tag []byte

	fileHash := this.FileHash
	max := this.getMaxService()
	cidsLen := uint64(len(cids))
	for _, c := range challenges {
		index = uint64(c.Index)
		if index >= cidsLen {
			log.Errorf("[prepareForPdpCal] invalid index for fileHash %s index %d", fileHash, index)
			return nil, nil, fmt.Errorf("file:%s, invalid index:%d", fileHash, index)
		}

		blockCid = cids[index]
		tag, err = max.GetTag(blockCid.String(), fileHash, index)
		if err != nil {
			log.Errorf("[prepareForPdpCal] GetBlockAttr for blockHash %s fileHash %s index %d error : %s", blockCid.String(), fileHash, index, err)
			return nil, nil, err
		}

		tags = append(tags, tag)
		blk, err := max.GetBlock(blockCid)
		if err != nil {
			log.Errorf("[prepareForPdpCal] GetBlock for block %s error : %s", blockCid.String(), err)
			return nil, nil, err
		}
		blocks = append(blocks, blk.RawData())
	}

	return tags, blocks, nil
}

func (this *FilePDPItem) generateProve(prover *pdp.Pdp, proveParam *fs.ProveParam, challenges []pdp.Challenge, tags [][]byte, blocks [][]byte) ([]byte, error) {
	fileHash := this.FileHash

	pdpTags := make([]pdp.Tag, 0)
	pdpBlocks := make([]pdp.Block, 0)

	for _, tag := range tags {
		var tmp pdp.Tag
		copy(tmp[:], tag[:])
		pdpTags = append(pdpTags, tmp)
	}
	for _, block := range blocks {
		pdpBlocks = append(pdpBlocks, block)
	}

	if len(challenges) != len(pdpTags) || len(challenges) != len(pdpBlocks) {
		log.Errorf("length of challenges, byteTags and byteBlocks no match for fileHash %s", fileHash)
		return nil, fmt.Errorf("length of challenges, byteTags and byteBlocks no match for fileHash %s", fileHash)
	}

	proofs, mpath, err := prover.GenerateProofWithMerklePathForFile(0, pdpBlocks, proveParam.FileID, pdpTags, challenges)
	if err != nil {
		log.Errorf("GenerateProofWithMerklePath for file %s error %s", fileHash, err)
		return nil, fmt.Errorf("GenerateProofWithMerklePath for file %s error %s", fileHash, err)
	}

	proveData := &fs.ProveData{
		Proofs:     proofs,
		BlockNum:   uint64(len(pdpBlocks)),
		Tags:       pdpTags,
		MerklePath: mpath,
	}

	buf := new(bytes.Buffer)
	err = proveData.Serialize(buf)
	if err != nil {
		log.Errorf("ProveData serialize for file %s error %s", fileHash, err)
		return nil, fmt.Errorf("ProveData serialize for file %s error %s", fileHash, err)
	}
	return buf.Bytes(), nil
}

func (this *FilePDPItem) getFsContract() *fscontract.Fs {
	return this.getMaxService().chain.Native.Fs
}

func (this *FilePDPItem) getAccountAddress() common.Address {
	return this.getFsContract().DefAcc.Address
}

var _ PDPItem = (*FilePDPItem)(nil)
