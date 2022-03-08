package max

import (
	"bytes"
	"context"
	"fmt"
	cid "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
	"github.com/saveio/max/max/sector"
	fscontract "github.com/saveio/themis-go-sdk/fs"
	"github.com/saveio/themis/common"
	"github.com/saveio/themis/common/log"
	fs "github.com/saveio/themis/smartcontract/service/native/savefs"
	"github.com/saveio/themis/smartcontract/service/native/savefs/pdp"
	"strings"
)

type FilePDPItem struct {
	FileHash       string
	FileInfo       *fs.FileInfo
	NextChalHeight uint32
	BlockHash      common.Uint256
	NextSubHeight  uint32
	ExpireState    ExpireState
	FirstProve     bool
	PdpResult      []byte
	max            *MaxService
	sectorId       uint64
}

func (this *FilePDPItem) doPdpCalculation() error {
	if this == nil || this.FileInfo == nil {
		return fmt.Errorf("pdp cal item is invalid")
	}

	fileHash := this.FileHash
	fileInfo := this.FileInfo
	blockHeight := this.NextChalHeight
	blockHash := this.BlockHash
	fsContract := this.getFsContract()

	log.Debugf("[doPdpCalculation] fileHash : %s, blockNum : %d, proveBlockNum : %d, fileProveParam : %v, hash : %d, height : %d",
		fileHash, fileInfo.FileBlockNum, fileInfo.ProveBlockNum, fileInfo.FileProveParam, blockHash.ToHexString(), blockHeight)

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

func (this *FilePDPItem) doPdpSubmission(proveData []byte) ([]byte, error) {
	var err error
	var sectorId uint64

	if this == nil {
		log.Errorf("item for pdp submission is nil")
		return nil, fmt.Errorf("item for pdp submission is nil")
	}

	log.Debugf("pdpSubmission item: %+v", this)

	fsContract := this.getFsContract()

	fileHash := this.FileHash
	height := uint64(this.NextChalHeight)

	if this.ExpireState == EXPIRE_LAST_PROVE {
		// sector id should be 0 when submit last prove for file
		sectorId = this.max.sectorManager.GetFileSectorId(fileHash)
		if sectorId == 0 {
			log.Errorf("file %s not found in any sector when try to do last prove", fileHash)
			return nil, fmt.Errorf("file %s not found in any sector when try to do last prove", fileHash)
		}
	} else {
		sector, err := this.assignSectorForFile()
		if err != nil {
			return nil, err
		}

		sectorId = sector.GetSectorID()

		log.Debugf("file %s has been assign to sector %d", fileHash, sectorId)
	}

	this.sectorId = sectorId

	txHash, err := fsContract.FileProve(fileHash, proveData, height, sectorId)
	if err != nil {
		log.Errorf("file prove error : %s", err)
		return nil, err
	}

	log.Debugf("call fileProve for file %s success with txHash %s", fileHash, getTxHashString(txHash))

	max := this.getMaxService()
	proveDetails, err := fsContract.GetFileProveDetails(fileHash)
	log.Debugf("proveDetails for file %s: %+v", fileHash, proveDetails)
	if err != nil {
		log.Errorf("get prove details after success prove error : %s", err)
		// delete cached prove details to force getting prove details from contract for next prove
		// when proveDetail not found, fileInfo may have been deleted
		max.rpcCache.deleteProveDetails(fileHash)
		// delete the file info to get file info from chain for sectorRef
		max.rpcCache.deleteFileInfo(fileHash)

		return nil, fmt.Errorf("get prove details after success prove error : %s", err)
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
			// delete the file info to get file info from chain for sectorRef
			max.rpcCache.deleteFileInfo(fileHash)

			return nil, fmt.Errorf("no matching prove detail found, delete cached prove details")
		}
	}
	return txHash, nil
}
func (this *FilePDPItem) onSuccessfulPdpSubmission() error {
	fileHash := this.FileHash
	height := this.NextChalHeight
	max := this.getMaxService()
	sectorId := this.sectorId

	if this.ExpireState == EXPIRE_LAST_PROVE {
		log.Infof("delete file and prove task for fileHash %s after last prove", fileHash)
		/*
			err := max.sectorManager.DeleteFile(this.FileHash)
			if err != nil {
				log.Errorf("deleteFile for file %s error %s", this.FileHash, err)
			}
		*/
		max.deleteAndNotify(fileHash, PROVE_TASK_REMOVAL_REASON_NORMAL)
		return nil
	}

	sector := max.sectorManager.GetSectorBySectorId(sectorId)
	if sector == nil {
		panic("sector not exist")
	}

	if this.FirstProve {
		err := sector.MoveCandidateFileToFileList(this.FileHash)
		if err != nil {
			log.Errorf("MoveCandidateFileToFileList for file %s onSuccessfulPdpSubmission error %s", this.FileHash, err)
			return fmt.Errorf("MoveCandidateFileToFileList for file %s onSuccessfulPdpSubmission error %s", this.FileHash, err)
		}

		// saves the first prove height to check if fileinfo is deleted then added agian
		err = max.saveProveTask(fileHash, uint64(height), this.FileInfo.FileProveParam)
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

func (this *FilePDPItem) assignSectorForFile() (*sector.Sector, error) {
	max := this.getMaxService()

	proveLevel := this.FileInfo.ProveLevel
	fileHash := string(this.FileInfo.FileHash)
	blockCount := this.FileInfo.FileBlockNum
	blockSize := this.FileInfo.FileBlockSize
	isPlots := this.FileInfo.IsPlotFile

	sector, err := max.sectorManager.AddCandidateFile(proveLevel, fileHash, blockCount, blockSize, isPlots)
	if err != nil {
		log.Errorf("assignSectorForFile for file %s error %s", fileHash, err)
		return nil, err
	}
	return sector, nil
}

func (this *FilePDPItem) processForSectorProve() error {
	max := this.getMaxService()

	sectorId := this.sectorId

	sector := max.sectorManager.GetSectorBySectorId(sectorId)
	if sector == nil {
		panic("sector not exist")
	}

	// if sector prove task not exist, create a sector prove task
	if !max.isSectorProveTaskExist(sectorId) {
		err := max.addSectorProveTask(sectorId)
		if err != nil {
			return err
		}
		log.Debugf("addProveTask for sector %d", sectorId)

		err = sector.SetNextProveHeight(uint64(this.NextChalHeight) + sector.GetProveInterval())
		if err != nil {
			log.Errorf("addFileToSector setNextProveHeight for sector %d error %s", sectorId, err)
			return err
		}
	}

	return nil
}

func (this *FilePDPItem) onFailedPdpSubmission(err error) error {
	log.Debugf("onFailedPdpSubmission for file %s with error %s", this.FileHash, err)

	max := this.getMaxService()

	if this.sectorId != 0 && this.ExpireState != EXPIRE_LAST_PROVE {
		err := max.sectorManager.DeleteCandidateFile(this.FileHash)
		if err != nil {
			log.Errorf("deleteCandidateFile for file %s onFailedPdpSubmission error %s", this.FileHash, err)
		}
	}

	// delete the task when the error is check pdp error
	// in case there is a rpc error or pdp record not found, the pdp submission may be successful
	// need to check when next prove is scheduled
	if strings.Contains(err.Error(), "CheckProve error") {
		return max.deleteAndNotify(this.FileHash, err.Error())
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

	for _, path := range mpath {
		fmt.Println("printPath")
		path.PrintPath()
	}

	p2 := pdp.NewPdp(0)
	err = p2.VerifyProofWithMerklePathForFile(0, proofs, proveParam.FileID, pdpTags, challenges, mpath, proveParam.RootHash)
	if err != nil {
		log.Errorf("self verify pdp error %s", err)
		return nil, err
	}

	log.Debugf("self verify pdp success")

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
	return this.getMaxService().getFsContract()
}

func (this *FilePDPItem) getAccountAddress() common.Address {
	return this.getFsContract().Client.GetDefaultAccount().Address
}

func (this *FilePDPItem) shouldSavePdpResult() bool {
	return true
}

func (this *FilePDPItem) getPdpCalculationResult() []byte {
	return this.PdpResult
}

var _ PDPItem = (*FilePDPItem)(nil)
