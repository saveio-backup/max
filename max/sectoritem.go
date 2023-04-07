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

type SectorPDPItem struct {
	SectorId        uint64
	Sector          *sector.Sector
	BlockHash       common.Uint256
	NextProveHeight uint32
	ProveData       []byte
	max             *MaxService
}

func (this *SectorPDPItem) doPdpCalculation() error {
	if this.Sector == nil {
		return fmt.Errorf("doPdpCalculation, no sector in the item")
	}

	log.Debugf("doPdpCalculation for sector %d", this.SectorId)

	sector := this.Sector

	// lock the candidate list and wait until candidate list is empty
	// after lock the candidate list, no more files can be added to candidate list, but candidate can be moved
	// to sector when file pdp succeed.
	// file that needs to be submitted will need to wait until the candidate list is unlocked to add to candidate list
	sector.BlockUntilCandidateListEmpty()

	log.Debugf("doPdpCalculation for sector %d, gen challenges param %v, %v, %v, %v", this.SectorId,
		this.getAccountAddress(), this.BlockHash, uint32(sector.GetTotalBlockCount()), uint32(sector.GetProveBlockNum()))

	// generate challenges
	challenges := fs.GenChallenge(this.getAccountAddress(), this.BlockHash,
		uint32(sector.GetTotalBlockCount()), uint32(sector.GetProveBlockNum()))

	log.Debugf("doPdpCalculation for sector %d, challenges %v", this.SectorId, challenges)

	indexes := make([]uint64, 0)

	for _, challenge := range challenges {
		indexes = append(indexes, uint64(challenge.Index))
	}

	filePos, err := sector.GetFilePosBySectorIndexes(indexes)
	if err != nil {
		return fmt.Errorf("doPdpCalculation, GetFilePosBySectorIndexes for sector %d error %s", this.SectorId, err)
	}

	proofs, err := this.doPdpCalculationForSector(filePos, challenges)
	if err != nil {
		return fmt.Errorf("doPdpCalculation, doPdpCalculationForSector for sector %d error %s", this.SectorId, err)
	}

	this.ProveData = proofs
	return nil
}

func (this *SectorPDPItem) onFailedPdpCalculation(err error) error {
	this.Sector.UnlockCandidateList()
	log.Errorf("onFailedPdpCalculation for %s, error %s", this.getItemKey(), err)
	return nil
}

func (this *SectorPDPItem) doPdpSubmission(proveData []byte) (txHash []byte, err error) {
	return this.getFsContract().SectorProve(this.SectorId, uint64(this.getPdpCalculationHeight()), proveData)
}

func (this *SectorPDPItem) onSuccessfulPdpSubmission() error {
	this.Sector.UnlockCandidateList()

	sectorInfo, err := this.getFsContract().GetSectorInfo(this.SectorId)
	if err != nil {
		log.Errorf("getSectorInfo error %s", err)
		return fmt.Errorf("getSectorInfo error %s", err)
	}

	curProveHeight := this.Sector.GetNextProveHeight()
	nextProveHeight := sectorInfo.NextProveHeight

	if nextProveHeight <= curProveHeight {
		log.Errorf("nextProveHeight in sectorInfo is %d, is no bigger than current prove height %d",
			nextProveHeight, curProveHeight)
	}

	// recalculate next challenge height
	//nextHeight := this.Sector.GetNextProveHeight() + this.Sector.GetProveInterval()
	log.Debugf("onSuccessfulPdpSubmission, set nextProveHeight for sector %d as %d", this.SectorId, nextProveHeight)

	err = this.Sector.SetLastProveHeight(curProveHeight)
	if err != nil {
		return nil
	}

	err = this.Sector.SetNextProveHeight(nextProveHeight)
	if err != nil {
		return nil
	}
	return nil
}

func (this *SectorPDPItem) onFailedPdpSubmission(err error) error {
	this.Sector.UnlockCandidateList()

	log.Errorf("onFailedPdpSubmission, pdp submission for sector %d error %s", this.SectorId, err)
	return fmt.Errorf("onFailedPdpSubmission, pdp submission for sector %d error %s", this.SectorId, err)
}

func (this *SectorPDPItem) getItemKey() string {
	return getSectorIdString(this.SectorId)
}

func (this *SectorPDPItem) getPdpCalculationHeight() uint32 {
	return this.NextProveHeight
}

// submission height same as calculation height
func (this *SectorPDPItem) getPdpSubmissionHeight() uint32 {
	return this.NextProveHeight
}

func (this *SectorPDPItem) doPdpCalculationForSector(filePos []*sector.FilePos, challenges []pdp.Challenge) ([]byte, error) {
	prover := pdp.NewPdp(0)

	blocksForPdp := make([][]byte, 0)
	tagsForPdp := make([][]byte, 0)
	fileIdsForPdp := make([]pdp.FileID, 0)
	// save updated challenge with index in file range other than global index
	updatedChallenges := make([]pdp.Challenge, 0)
	max := this.getMaxService()

	log.Debugf("[doPdpCalculationForSector] with challenges %+v", challenges)
	for _, pos := range filePos {
		log.Debugf("[doPdpCalculationForSector] with filePos %+v", pos)
	}

	var scoopData []byte
	curIndex := 0
	for _, pos := range filePos {
		fileHash := pos.FileHash
		fileChallenges := make([]pdp.Challenge, 0)

		for _, index := range pos.BlockIndexes {
			challenge := pdp.Challenge{
				Index: uint32(index),
				Rand:  challenges[curIndex].Rand,
			}
			// NOTE: the file challenges has its index within file index range, needed for merkle path calculation
			// for current pdp implementation, index is only used to select the blocks to be challenged, its value
			// have no impact for the pdp calculation/verification
			fileChallenges = append(fileChallenges, challenge)
			curIndex++
		}

		proveParam, err := this.getPDPProveParamForFile(fileHash)
		if err != nil {
			log.Errorf("[doPdpCalculationForSector] getPDPProveParamForFile for file %s error %s", fileHash, err)
			return nil, err
		}

		rootCid, err := cid.Decode(fileHash)
		if err != nil {
			log.Errorf("[doPdpCalculationForSector] Decode for fileHash %s error : %s", fileHash, err)
			return nil, err
		}

		cids, err := max.GetFileAllCids(context.TODO(), rootCid)
		if err != nil {
			log.Errorf("[doPdpCalculationForSector] GetFileAllCids for fileHash %s error : %s", fileHash, err)
			return nil, err
		}

		err = this.initProverWithTagsForFile(prover, fileHash, proveParam, cids)
		if err != nil {
			log.Errorf("[doPdpCalculationForSector] initProverWithTagsForFile for fileHash %s error : %s", fileHash, err)
			return nil, err
		}

		fileIds, tags, blocks, err := this.prepareForPdpCal(prover, fileHash, proveParam, fileChallenges)
		if err != nil {
			return nil, err
		}

		if this.Sector.IsPlots && len(scoopData) == 0 {
			fileInfo, err := max.getFileInfo(fileHash)
			if err != nil {
				log.Errorf("getFileInfo for file %s error %s", fileHash, err)
				return nil, err
			}

			if !fileInfo.IsPlotFile || fileInfo.PlotInfo == nil {
				log.Errorf("file %s not a plot file %s", fileHash, err)
				return nil, err
			}

			index := pos.BlockIndexes[0] % fileInfo.PlotInfo.Nonces
			count := uint64(0)

			cidPrefix := cid.SaveCidPrefix
			for _, cid := range cids {
				cidStr := cid.String()
				if strings.HasPrefix(cidStr, cidPrefix+"Qm") {
					continue
				}
				count++
				// found the block
				if index+1 == count {
					block, err := max.GetBlock(cid)
					if err != nil {
						log.Errorf("get block error %s", err)
						return nil, err
					}

					scoopData = make([]byte, 64)
					copy(scoopData, block.RawData())
				}
			}
		}

		fileIdsForPdp = append(fileIdsForPdp, fileIds...)
		tagsForPdp = append(tagsForPdp, tags...)
		blocksForPdp = append(blocksForPdp, blocks...)
		updatedChallenges = append(updatedChallenges, fileChallenges...)
	}

	proveData, err := this.generateProve(prover, uint64(len(filePos)), fileIdsForPdp, updatedChallenges, tagsForPdp, blocksForPdp)
	if err != nil {
		return nil, err
	}

	// add first scoop from the fist block in prove data
	if this.Sector.IsPlots {
		proveData.PlotData = scoopData
	}

	buf := new(bytes.Buffer)
	err = proveData.Serialize(buf)
	if err != nil {
		log.Errorf("SectorProveData serialize for sector %d error %s", this.SectorId, err)
		return nil, fmt.Errorf("SectorProveData serialize for sector %d error %s", this.SectorId, err)
	}
	return buf.Bytes(), nil
}

func (this *SectorPDPItem) initProverWithTagsForFile(prover *pdp.Pdp, fileHash string, proveParam *fs.ProveParam, cids []*cid.Cid) error {
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
		log.Errorf("[initProverWithTagsForFile] initMerkleTree for file %s error: %s", fileHash, err)
		return fmt.Errorf("initMerkleTree for file %s error: %s", fileHash, err)
	}
	return nil
}
func (this *SectorPDPItem) prepareForPdpCal(prover *pdp.Pdp, fileHash string, proveParam *fs.ProveParam, challenges []pdp.Challenge) (fileIDs []pdp.FileID, tags [][]byte, blocks [][]byte, err error) {
	rootCid, err := cid.Decode(fileHash)
	if err != nil {
		log.Errorf("[prepareForPdpCal] Decode for fileHash %s error : %s", fileHash, err)
		return nil, nil, nil, err
	}

	max := this.getMaxService()
	cids, err := max.GetFileAllCids(context.TODO(), rootCid)
	if err != nil {
		log.Errorf("[prepareForPdpCal] GetFileAllCids for fileHash %s error : %s", fileHash, err)
		return nil, nil, nil, err
	}

	fileId := proveParam.FileID

	var blockCid *cid.Cid
	var index uint64
	var tag []byte

	cidsLen := uint64(len(cids))
	for _, c := range challenges {
		index = uint64(c.Index)
		if index >= cidsLen {
			log.Errorf("[prepareForPdpCal] invalid index for fileHash %s index %d", fileHash, index)
			return nil, nil, nil, fmt.Errorf("file:%s, invalid index:%d", fileHash, index)
		}

		blockCid = cids[index]
		tag, err = max.GetTag(blockCid.String(), fileHash, index)
		if err != nil {
			log.Errorf("[prepareForPdpCal] GetBlockAttr for blockHash %s fileHash %s index %d error : %s", blockCid.String(), fileHash, index, err)
			return nil, nil, nil, err
		}

		tags = append(tags, tag)
		blk, err := max.GetBlock(blockCid)
		if err != nil {
			log.Errorf("[prepareForPdpCal] GetBlock for block %s error : %s", blockCid.String(), err)
			return nil, nil, nil, err
		}
		blocks = append(blocks, blk.RawData())

		fileIDs = append(fileIDs, fileId)
	}

	return fileIDs, tags, blocks, nil
}

func (this *SectorPDPItem) generateProve(prover *pdp.Pdp, fileNum uint64, fileIds []pdp.FileID, challenges []pdp.Challenge, tags [][]byte, blocks [][]byte) (*fs.SectorProveData, error) {
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

	if len(fileIds) != len(challenges) || len(fileIds) != len(pdpTags) || len(fileIds) != len(pdpBlocks) {
		log.Errorf("length of fileIds, challenges, byteTags and byteBlocks no match for sector %d", this.SectorId)
		return nil, fmt.Errorf("length of fileIds, challenges, byteTags and byteBlocks no match for sector %d", this.SectorId)
	}

	proofs, mpath, err := prover.GenerateProofWithMerklePath(0, pdpBlocks, fileIds, pdpTags, challenges)
	if err != nil {
		log.Errorf("GenerateProofWithMerklePath for sector %d error %s", this.SectorId, err)
		return nil, fmt.Errorf("GenerateProofWithMerklePath for sector %d error %s", this.SectorId, err)
	}

	return &fs.SectorProveData{
		ProveFileNum: fileNum,
		BlockNum:     uint64(len(blocks)),
		Proofs:       proofs,
		Tags:         pdpTags,
		MerklePath:   mpath,
	}, nil
}

// from the challenge indexes
func (this *SectorPDPItem) getFilesInSectorForPdpCalculation(indexes []uint64) ([]*sector.FilePos, error) {
	return this.getSectorManager().GetFilePosBySectorIndexes(this.SectorId, indexes)
}

func (this *SectorPDPItem) getMaxService() *MaxService {
	if this.max == nil {
		panic("max service is nil")
	}
	return this.max
}

func (this *SectorPDPItem) getSectorManager() *sector.SectorManager {
	return this.getMaxService().sectorManager
}

func (this *SectorPDPItem) getFsContract() *fscontract.Fs {
	return this.getMaxService().getFsContract()
}

func (this *SectorPDPItem) getAccountAddress() common.Address {
	return this.getMaxService().getAccoutAddress()
}

func (this *SectorPDPItem) getPDPProveParamForFile(fileHash string) (*fs.ProveParam, error) {
	param, err := this.getMaxService().getProveTask(fileHash)
	if err != nil {
		return nil, err
	}

	if param == nil {
		return nil, fmt.Errorf("param for file %s is nil", fileHash)
	}

	var pdpParam fs.ProveParam
	paramReader := bytes.NewReader(param.PDPParam)
	err = pdpParam.Deserialize(paramReader)
	if err != nil {
		return nil, err
	}
	return &pdpParam, nil
}

func (this *SectorPDPItem) shouldSavePdpResult() bool {
	return false
}

func (this *SectorPDPItem) getPdpCalculationResult() []byte {
	return this.ProveData
}

func getSectorIdString(sectorId uint64) string {
	return fmt.Sprintf("Secotr %d", sectorId)
}

var _ PDPItem = (*SectorPDPItem)(nil)
