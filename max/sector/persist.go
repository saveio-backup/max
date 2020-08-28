package sector

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type DB interface {
	PutData(key string, data []byte) error
	GetData(key string) ([]byte, error)
	DeleteData(key string) error
	//GetDataWithPrefix(prefix string) ([][]byte, error)
}

const (
	SECTOR_LIST_KEY  = "sectorlist:"
	SECTOR_FILE_LIST = "sectorfilelist:"
)

type DBSectorInfo struct {
	SectorId   uint64 `json:"sectorid"`
	ProveLevel uint64 `json:"provelevel"`
	Size       uint64 `json:"size"`
}

type DBSectorList struct {
	SectorInfos []*DBSectorInfo `json:"sectorinfos"`
}

// need to run with lock to for data consistency
func (this *SectorManager) saveSectorList() error {
	if this.isLoading {
		return nil
	}

	sectorInfos := make([]*DBSectorInfo, 0)
	for level, sectors := range this.sectors {
		for _, sector := range sectors {
			sectorInfos = append(sectorInfos, &DBSectorInfo{
				SectorId:   sector.sectorId,
				ProveLevel: level,
				Size:       sector.sectorSize,
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
		return nil, err
	}

	sectorList := new(DBSectorList)

	err = json.Unmarshal(data, sectorList)
	if err != nil {
		return nil, err
	}
	return sectorList, nil
}

type DBSectorFileInfo struct {
	FileHash   string `json:"filehash"`
	BlockCount uint64 `json:"blockcount"`
	BlockSize  uint64 `json:"blocksize"`
}

type DBSectorFileList struct {
	SectorFileInfos []*DBSectorFileInfo `json:"sectorfileinfos"`
}

func (this *SectorManager) saveSectorFileList(sectorId uint64) error {
	if this.isLoading {
		return nil
	}

	sector := this.GetSectorBySectorId(sectorId)
	if sector == nil {
		return fmt.Errorf("saveSectorFileList, no sector found with id %d", sectorId)
	}

	sectorFileInfos := make([]*DBSectorFileInfo, 0)
	for _, file := range sector.fileList {
		sectorFileInfos = append(sectorFileInfos, &DBSectorFileInfo{
			FileHash:   file.fileHash,
			BlockCount: file.blockCount,
			BlockSize:  file.blockSize,
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
	data, err := this.db.GetData(genSectorFileListKey(sectorId))
	if err != nil {
		return nil, err
	}

	sectorFileList := new(DBSectorFileList)
	err = json.Unmarshal(data, sectorFileList)
	if err != nil {
		return nil, err
	}
	return sectorFileList, nil
}

func (this *SectorManager) LoadSectorsOnStartup() error {
	this.isLoading = true

	defer func() {
		this.isLoading = false
	}()

	sectorList, err := this.loadSectorList()
	if err != nil {
		return err
	}

	// load all the sectors and create sector
	for _, sectorInfo := range sectorList.SectorInfos {
		_, err = this.CreateSector(sectorInfo.SectorId, sectorInfo.ProveLevel, sectorInfo.Size)
		if err != nil {
			return err
		}

		fileList, err := this.loadSectorFileList(sectorInfo.SectorId)
		if err != nil {
			return err
		}

		// load file list in the sector and add file to sector
		for _, fileInfo := range fileList.SectorFileInfos {
			_, err = this.AddFile(sectorInfo.ProveLevel, fileInfo.FileHash, fileInfo.BlockCount, fileInfo.BlockSize)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func genSectorListKey() string {
	return SECTOR_LIST_KEY
}

func genSectorFileListKey(sectorId uint64) string {
	return SECTOR_FILE_LIST + strconv.Itoa(int(sectorId))
}
