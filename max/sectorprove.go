package max

import (
	"fmt"
	"github.com/saveio/max/max/sector"
	"github.com/saveio/themis/common/log"
	"sort"
	"time"
)

func (this *MaxService) startSectorProveService() {
	log.Debugf("[startSectorProveService] start service")
	ticker := time.NewTicker(time.Duration(PROVE_SECTOR_INTERVAL) * time.Second)

	err := this.loadSectorProveTasks()
	if err != nil {
		log.Errorf("[startSectorProveService] loadSectorProveTasks error %s", err)
		return
	}

	for {
		select {
		case <-this.kill:
			log.Debugf("[startSectorProveService] service killed")
			return
		case <-ticker.C:
			if this.chain == nil {
				break
			}

			this.sectorProveTasks.Range(func(key, value interface{}) bool {
				go func() {
					sectorId := key.(uint64)
					err := this.proveSector(sectorId)
					if err != nil {
						log.Errorf("[startSectorProveService] proveSector for sector %d error %s", sectorId, err)
					}
					// TODO: check the error type and retry
				}()
				return true
			})
		}
	}
}

func (this *MaxService) loadSectorProveTasks() error {
	sectorIds, err := this.sectorManager.GetAllSectorIds()
	if err != nil {
		return err
	}

	for _, sectorId := range sectorIds {
		sector := this.sectorManager.GetSectorBySectorId(sectorId)
		if sector == nil {
			log.Errorf("sector %d not found", sectorId)
			continue
		}

		err = this.processCandidateFileSynchronization(sector)
		if err != nil {
			log.Errorf("processCandidateFileSynchronization error %s", err)
			continue
		}

		if sector.GetFirstProveHeight() != 0 && sector.GetTotalBlockCount() != 0 {
			err = this.addSectorProveTask(sectorId)
			if err != nil {
				return fmt.Errorf("addSectorProveTask error %s", err)
			}
		}
	}
	return nil
}

// process sector status synchronization with chain in case candidateFileList is not empty
func (this *MaxService) processCandidateFileSynchronization(sector *sector.Sector) error {
	candidateFileList := sector.GetCandidateFileList()

	if len(candidateFileList) == 0 {
		return nil
	}

	sectorId := sector.GetSectorID()

	sectorInfo, err := this.getFsContract().GetSectorInfo(sectorId)
	if err != nil {
		log.Errorf("sector %d getSectorInfo error %s", sectorId, err)
		return fmt.Errorf("sector %d getSectorInfo error %s", sectorId, err)
	}

	fileMap := make(map[string]int, 0)
	for index, fileHash := range sectorInfo.FileList.List {
		fileMap[string(fileHash.Hash)] = index
	}

	fileIndex := make([]int, 0)
	for _, file := range candidateFileList {
		index, exist := fileMap[file.FileHash]
		if !exist {
			err = this.sectorManager.DeleteCandidateFile(file.FileHash)
			if err != nil {
				log.Errorf("sector %d deleteCandidateFile error %s", sectorId, err)
			}
			continue
		}

		fileIndex = append(fileIndex, index)
	}

	if len(fileIndex) != 0 {
		sort.SliceStable(fileIndex, func(i, j int) bool {
			return fileIndex[i] < fileIndex[j]
		})

		// add candidate file to sector in same order as on chain
		for _, index := range fileIndex {
			fileHash := string(sectorInfo.FileList.List[index].Hash)
			err := sector.MoveCandidateFileToFileList(fileHash)
			if err != nil {
				log.Errorf("sector %d moveCandidateFileToFileList error %s", sectorId, err)
				return fmt.Errorf("sector %d moveCandidateFileToFileList error %s", sectorId, err)
			}
		}
	}
	return nil
}

// check if time to do pdp for the sector
func (this *MaxService) proveSector(sectorId uint64) error {
	log.Debugf("proveSector for sector %d", sectorId)
	sector := this.sectorManager.GetSectorBySectorId(sectorId)
	if sector == nil {
		log.Errorf("proveSector error sector %d not exist", sectorId)
		return fmt.Errorf("proveSector error sector %d not exist", sectorId)
	}

	// get next prove height from chain, next prove height may not be updated correctly
	sectorInfo, err := this.getFsContract().GetSectorInfo(sectorId)
	if err != nil {
		log.Errorf("sector %d getSectorInfo error %s", sectorId, err)
		return fmt.Errorf("sector %d getSectorInfo error %s", sectorId, err)
	}

	challengeHeight := uint32(sectorInfo.NextProveHeight)

	height, _ := this.getCurrentBlockHeightAndHash()
	if height < challengeHeight {
		log.Debugf("proveSector, not reach next prove height %d for sector %d", challengeHeight, sectorId)
		return nil
	}

	if this.IsScheduledForPdpCalculationOrSubmission(getSectorIdString(sectorId)) {
		log.Debugf("proveSector, sector %d has been scheduled for pdp calculation or submission", sectorId)
		return nil
	}

	hash, err := this.getFsContract().Client.GetBlockHash(challengeHeight)
	if err != nil {
		log.Errorf("proveSector, getBlockHash for height %d error %s", height, err)
		return fmt.Errorf("proveSector, getBlockHash for height %d error %s", height, err)
	}

	sectorPdpItem := &SectorPDPItem{
		SectorId:        sectorId,
		Sector:          sector,
		NextProveHeight: challengeHeight,
		BlockHash:       hash,
		max:             this,
	}

	err = this.scheduleForProve(sectorPdpItem)
	if err != nil {
		log.Errorf("proveSector, scheduleForProve for sector %d error %s", sectorId, err)
		return err
	}

	log.Debugf("proveSector, scheduleForProve for sector %d", sectorId)
	return nil
}

func (this *MaxService) isSectorProveTaskExist(sectorId uint64) bool {
	if _, exist := this.sectorProveTasks.Load(sectorId); exist {
		return true
	}
	return false
}

func (this *MaxService) addSectorProveTask(sectorId uint64) error {
	log.Debugf("addSectorProveTask for sectorId %d", sectorId)
	this.sectorProveTasks.Store(sectorId, struct{}{})
	return nil
}

func (this *MaxService) deleteSectorProveTask(sectorId uint64) error {
	log.Debugf("deleteSectorProveTask for sectorId %d", sectorId)
	this.sectorProveTasks.Delete(sectorId)
	return nil
}
