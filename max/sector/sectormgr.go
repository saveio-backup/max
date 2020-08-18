package sector

import (
	"fmt"
	fs "github.com/saveio/themis/smartcontract/service/native/savefs"
	"sort"
	"sync"
)

const (
	MIN_SECTOR_SIZE = 4 * 1024 * 1024 * 1024 // 4G as min sector size
)

type SectorManager struct {
	lock            sync.RWMutex
	sectors         map[uint64]map[uint64]*Sector // proveLevel -> sectorId -> sector
	sectorIdMap     map[uint64]*Sector            // sector id -> sector
	fileSectorIdMap map[string]uint64             // fileHash -> sector id
	idCounter       uint64                        // global idCounter for sector id allocation
}

// TODO : may need to recover the last idCounter by local storage or from chain
// need to add more generic interface for saving/retriving data from db like getData
func InitSectors() *SectorManager {
	return &SectorManager{
		sectors:         make(map[uint64]map[uint64]*Sector),
		sectorIdMap:     make(map[uint64]*Sector),
		fileSectorIdMap: make(map[string]uint64),
		idCounter:       0,
	}
}

// TODO: load all sector info from DB
func (this *SectorManager) loadSectorsOnStartup() {
}

// create a new sector with given sector size, sector id is allocated automatically
func (this *SectorManager) createSector(proveLevel uint64, size uint64) (*Sector, error) {
	switch proveLevel {
	case fs.PROVE_LEVEL_LOW:
	case fs.PROVE_LEVEL_MEDIEUM:
	case fs.PROVE_LEVEL_HIGH:
	default:
		return nil, fmt.Errorf("create sector error, unknown prove level")
	}

	if size < MIN_SECTOR_SIZE {
		return nil, fmt.Errorf("create sector error, size %d is smaller than min sector size", size)
	}

	this.lock.Lock()
	defer this.lock.Unlock()

	sectors, exist := this.sectors[proveLevel]
	if !exist {
		sectors = make(map[uint64]*Sector)
		this.sectors[proveLevel] = sectors
	}

	this.idCounter++
	sector := initSector(this.idCounter, size)
	sector.proveParam.proveLevel = proveLevel
	sectors[sector.sectorId] = sector

	this.sectorIdMap[this.idCounter] = sector

	return sector, nil
}

func (this *SectorManager) deleteSector(id uint64) error {
	var sector *Sector

	this.lock.Lock()
	defer this.lock.Unlock()

	if sector = this.getSectorBySectorId(id); sector == nil {
		return fmt.Errorf("deleteSector no sector found with id %d", id)
	}

	proveLevel := sector.getProveLevel()
	sectors, exist := this.sectors[proveLevel]
	if !exist {
		return fmt.Errorf("deleteSector no sector found with proveLevel %d id %d", proveLevel, id)
	}

	delete(sectors, sector.sectorId)
	delete(this.sectorIdMap, sector.sectorId)
	return nil
}

func (this *SectorManager) getSectorBySectorId(id uint64) *Sector {
	sector, exist := this.sectorIdMap[id]
	if !exist {
		return nil
	}
	return sector
}

// try add a file with a fileInfo to blocks which has enough size for file
func (this *SectorManager) addFile(fileInfo *fs.FileInfo) (*Sector, error) {
	if fileInfo == nil {
		return nil, fmt.Errorf("addFile error, fileInfo is nil")
	}

	fileHash := string(fileInfo.FileHash)
	if !fileInfo.ValidFlag {
		return nil, fmt.Errorf("addFile error, file %s is invalid", fileHash)
	}

	this.lock.Lock()
	defer this.lock.Unlock()

	if this.isFileAdded(fileHash) {
		return nil, fmt.Errorf("addFile error, file %s is already added", fileHash)
	}

	proveLevel := fileInfo.ProveLevel

	sectors, exist := this.sectors[proveLevel]
	if !exist {
		return nil, fmt.Errorf("addFile error, no sector with prove level %d found for file %s", proveLevel, fileHash)
	}

	//to find the sector which is most suitable for the file storage
	// eg, if one sector has remaining size 2G, and we want to store a file with 1G, it should be
	// stored in this sector instead of putting it in a empty sector

	fileSize := fileInfo.FileBlockNum * fileInfo.FileBlockSize
	sectorId := this.findMatchingSectorIdWithSize(sectors, fileSize)
	if sectorId == 0 {
		return nil, fmt.Errorf("addFile error, no matching sector found for file %s with size %d", fileHash, fileSize)
	}

	sector := this.getSectorBySectorId(sectorId)
	err := sector.addFileToSector(fileInfo)
	if err != nil {
		return nil, fmt.Errorf("addFile error, addFileToSector error %v", err)
	}

	this.updateFileMap(fileHash, sector.sectorId, true)
	return sector, nil

}

func (this *SectorManager) deleteFile(fileHash string) error {
	var sectorId uint64

	this.lock.Lock()
	defer this.lock.Unlock()

	if sectorId = this.getFileSectorId(fileHash); sectorId != 0 {
		return fmt.Errorf("deleteFile, file %s is not in sectors", fileHash)
	}

	sector := this.getSectorBySectorId(sectorId)
	if sector == nil {
		return fmt.Errorf("deleteFile, sector with id %d not found", sectorId)
	}

	err := sector.deleteFileFromSector(fileHash)
	if err != nil {
		return fmt.Errorf("deleteFile, deleteFileFromSector for file %s error %s", fileHash, err)
	}

	this.updateFileMap(fileHash, sectorId, false)
	return nil
}

func (this *SectorManager) findMatchingSectorIdWithSize(sectors map[uint64]*Sector, fileSize uint64) uint64 {
	candidates := make([]uint64, 0)
	for sectorId, sector := range sectors {
		if sector.getSectorRemainingSize() >= fileSize {
			candidates = append(candidates, sectorId)
		}
	}

	if len(candidates) == 0 {
		return 0
	}

	// sort by sectorId to make sure we fill sector allocated earlier first
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i] < candidates[j]
	})
	return candidates[0]
}

func (this *SectorManager) isFileAdded(fileHash string) bool {
	_, exist := this.fileSectorIdMap[fileHash]
	if exist {
		return true
	}
	return false
}

func (this *SectorManager) getFileSectorId(fileHash string) uint64 {
	sectorId, exist := this.fileSectorIdMap[fileHash]
	if !exist {
		return 0
	}
	return sectorId
}

// update file map no lock
func (this *SectorManager) updateFileMap(fileHash string, sectorId uint64, isAdd bool) {
	if isAdd {
		this.fileSectorIdMap[fileHash] = sectorId
	} else {
		delete(this.fileSectorIdMap, fileHash)
	}
}
