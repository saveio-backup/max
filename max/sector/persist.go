package sector

import (
	"encoding/json"
	"fmt"
	ldb "github.com/saveio/max/max/leveldbstore"
	"github.com/saveio/themis/common/log"
	"strconv"
)

type DB interface {
	PutData(key string, data []byte) error
	GetData(key string) ([]byte, error)
	DeleteData(key string) error
	//GetDataWithPrefix(prefix string) ([][]byte, error)
}

const (
	SECTOR_LIST_KEY                 = "sectorlist:"
	SECTOR_FILE_LIST_KEY            = "sectorfilelist:"
	SECTOR_PROVE_PARAM_KEY          = "sectorproveparam:"
	SECTOR_CANDIDIATE_FILE_LIST_KEY = "sectorcandidatefilelist:"
)

type DBSectorInfo struct {
	SectorId   uint64 `json:"sectorid"`
	ProveLevel uint64 `json:"provelevel"`
	Size       uint64 `json:"size"`
	IsPlots    bool   `json:"isplots"`
}

type DBSectorList struct {
	SectorInfos []*DBSectorInfo `json:"sectorinfos"`
}

// need to run with lock to for data consistency
func (this *SectorManager) saveSectorList() error {
	if this.isOnStartup() {
		return nil
	}

	if this.db == nil {
		return nil
	}

	sectorInfos := make([]*DBSectorInfo, 0)
	for level, sectors := range this.sectors {
		for _, sector := range sectors {
			sectorInfos = append(sectorInfos, &DBSectorInfo{
				SectorId:   sector.GetSectorID(),
				ProveLevel: level,
				Size:       sector.SectorSize,
				IsPlots:    sector.IsPlots,
			})
		}
	}

	sectorList := DBSectorList{SectorInfos: sectorInfos}

	data, err := json.Marshal(sectorList)
	if err != nil {
		return err
	}

	return this.db.PutData(genSectorListKey(), data)
}

func (this *SectorManager) loadSectorList() (*DBSectorList, error) {
	data, err := this.db.GetData(genSectorListKey())
	if err != nil {
		if err == ldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	if data == nil {
		return nil, nil
	}

	sectorList := new(DBSectorList)

	err = json.Unmarshal(data, sectorList)
	if err != nil {
		return nil, err
	}
	return sectorList, nil
}

type DBSectorFileList struct {
	SectorFileInfos []*SectorFileInfo `json:"sectorfileinfos"`
}

func (this *SectorManager) saveSectorFileList(sectorId uint64) error {
	if this.isOnStartup() {
		return nil
	}

	if this.db == nil {
		return nil
	}

	sector := this.GetSectorBySectorId(sectorId)
	if sector == nil {
		return fmt.Errorf("saveSectorFileList, no sector found with id %d", sectorId)
	}

	sectorFileInfos := make([]*SectorFileInfo, 0)
	for _, file := range sector.GetFileList() {
		sectorFileInfos = append(sectorFileInfos, &SectorFileInfo{
			FileHash:   file.FileHash,
			BlockCount: file.BlockCount,
			BlockSize:  file.BlockSize,
		})
	}

	sectorFileList := &DBSectorFileList{SectorFileInfos: sectorFileInfos}
	data, err := json.Marshal(sectorFileList)
	if err != nil {
		return err
	}
	return this.db.PutData(genSectorFileListKey(sectorId), data)
}

func (this *SectorManager) loadSectorFileList(sectorId uint64) (*DBSectorFileList, error) {
	if this.db == nil {
		return nil, nil
	}

	data, err := this.db.GetData(genSectorFileListKey(sectorId))
	if err != nil {
		if err == ldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	if data == nil {
		return nil, nil
	}

	sectorFileList := new(DBSectorFileList)
	err = json.Unmarshal(data, sectorFileList)
	if err != nil {
		return nil, err
	}
	return sectorFileList, nil
}

func (this *SectorManager) deleteSectorFileList(sectorId uint64) error {
	if this.db == nil {
		return nil
	}
	return this.db.DeleteData(genSectorFileListKey(sectorId))
}

func (this *SectorManager) saveSectorProveParam(sectorId uint64) error {
	if this.isOnStartup() {
		return nil
	}

	if this.db == nil {
		return nil
	}

	sector := this.GetSectorBySectorId(sectorId)
	if sector == nil {
		return fmt.Errorf("saveSectorProveParam, no sector found with id %d", sectorId)
	}

	data, err := json.Marshal(sector.GetProveParam())
	if err != nil {
		return err
	}

	return this.db.PutData(genSectorProveParamKey(sectorId), data)
}

func (this *SectorManager) loadSectorProveParam(sectorId uint64) (*SectorProveParam, error) {
	data, err := this.db.GetData(genSectorProveParamKey(sectorId))
	if err != nil {
		if err == ldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	if data == nil {
		return nil, nil
	}

	sectorProveParam := new(SectorProveParam)

	err = json.Unmarshal(data, sectorProveParam)
	if err != nil {
		return nil, err
	}
	return sectorProveParam, nil
}

func (this *SectorManager) deleteSectorProveParam(sectorId uint64) error {
	if this.db == nil {
		return nil
	}
	return this.db.DeleteData(genSectorProveParamKey(sectorId))
}

func (this *SectorManager) saveSectorCandidateFileList(sectorId uint64) error {
	if this.isOnStartup() {
		return nil
	}

	if this.db == nil {
		return nil
	}

	sector := this.GetSectorBySectorId(sectorId)
	if sector == nil {
		return fmt.Errorf("saveSectorCandidateFileList, no sector found with id %d", sectorId)
	}

	sectorFileInfos := sector.GetCandidateFileList()

	sectorFileList := &DBSectorFileList{SectorFileInfos: sectorFileInfos}
	data, err := json.Marshal(sectorFileList)
	if err != nil {
		return err
	}
	return this.db.PutData(genSectorCandidateFileListKey(sectorId), data)
}

func (this *SectorManager) loadSectorCandidateFileList(sectorId uint64) (*DBSectorFileList, error) {
	if this.db == nil {
		return nil, nil
	}

	data, err := this.db.GetData(genSectorCandidateFileListKey(sectorId))
	if err != nil {
		return nil, err
	}

	if data == nil {
		return nil, nil
	}

	sectorFileList := new(DBSectorFileList)
	err = json.Unmarshal(data, sectorFileList)
	if err != nil {
		return nil, err
	}
	return sectorFileList, nil
}

func (this *SectorManager) deleteSectorCandidateFileList(sectorId uint64) error {
	if this.db == nil {
		return nil
	}
	return this.db.DeleteData(genSectorCandidateFileListKey(sectorId))
}

func (this *SectorManager) LoadSectorsOnStartup() error {
	this.isLoading = true

	defer func() {
		this.isLoading = false
	}()

	sectorList, err := this.loadSectorList()
	if err != nil {
		log.Debugf("LoadSectorsOnStartup, loadSectorList error %s", err)
		return err
	}

	if sectorList == nil {
		log.Debugf("LoadSectorsOnStartup, sectorList is nil")
		return nil
	}

	// load all the sectors and create sector
	for _, sectorInfo := range sectorList.SectorInfos {
		sectorId := sectorInfo.SectorId
		isPlots := sectorInfo.IsPlots
		sector, err := this.CreateSector(sectorId, sectorInfo.ProveLevel, sectorInfo.Size, isPlots)
		if err != nil {
			log.Debugf("LoadSectorsOnStartup, createSector err %s", err)
			return err
		}

		proveParam, err := this.loadSectorProveParam(sectorId)
		if err != nil || proveParam == nil {
			log.Debugf("LoadSectorsOnStartup, loadSectorProveParam err %s", err)
			return err
		}

		err = sector.SetFirstProveHeight(proveParam.FirstProveHeight)
		if err != nil {
			return err
		}

		err = sector.SetLastProveHeight(proveParam.LastProveHeight)
		if err != nil {
			return err
		}

		err = sector.SetNextProveHeight(proveParam.NextProveHeight)
		if err != nil {
			return err
		}

		fileList, err := this.loadSectorFileList(sectorId)
		if err != nil {
			return err
		}

		if fileList == nil {
			continue
		}

		// load file list in the sector and add file to sector
		for _, fileInfo := range fileList.SectorFileInfos {
			_, err = this.AddFileToSector(sectorInfo.ProveLevel, fileInfo.FileHash, fileInfo.BlockCount,
				fileInfo.BlockSize, sectorId)
			if err != nil {
				return err
			}
		}

		candidateFileList, err := this.loadSectorCandidateFileList(sectorId)
		if err != nil {
			return err
		}

		if candidateFileList == nil {
			continue
		}

		for _, fileInfo := range candidateFileList.SectorFileInfos {
			_, err = this.AddCandidateFileToSector(sectorInfo.ProveLevel, fileInfo.FileHash, fileInfo.BlockCount,
				fileInfo.BlockSize, sectorId)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (this *SectorManager) isOnStartup() bool {
	if this.isLoading {
		return true
	}
	return false
}

func genSectorListKey() string {
	return SECTOR_LIST_KEY
}

func genSectorFileListKey(sectorId uint64) string {
	return SECTOR_FILE_LIST_KEY + strconv.Itoa(int(sectorId))
}

func genSectorCandidateFileListKey(sectorId uint64) string {
	return SECTOR_CANDIDIATE_FILE_LIST_KEY + strconv.Itoa(int(sectorId))
}

func genSectorProveParamKey(sectorId uint64) string {
	return SECTOR_PROVE_PARAM_KEY + strconv.Itoa(int(sectorId))
}
