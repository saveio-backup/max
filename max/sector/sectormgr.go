package sector

import (
	"fmt"
	"github.com/saveio/themis/common/log"
	fs "github.com/saveio/themis/smartcontract/service/native/savefs"
	"sort"
	"sync"
)

const (
	MIN_SECTOR_SIZE = 1024 * 1024 // 1G as min sector size, uint is KB
)

type SectorManager struct {
	lock            sync.RWMutex
	sectors         map[uint64]map[uint64]*Sector // proveLevel -> sectorId -> sector
	sectorIdMap     map[uint64]*Sector            // sector id -> sector
	fileSectorIdMap map[string]uint64             // fileHash -> sector id
	db              DB                            // db for persist sector data
	kill            chan struct{}
	sectorEventChan chan *SectorEvent // channel for sector create/delete event
	isLoading       bool
}

const (
	SECTOR_EVENT_CREATE = "createsector"
	SECTOR_EVENT_DELETE = "deletesector"
)

type SectorEvent struct {
	Event      string
	SectorID   uint64
	ProveLevel uint64
	Size       uint64
}

// need to add more generic interface for saving/retriving data from db like getData
func InitSectorManager(db DB) *SectorManager {
	return &SectorManager{
		sectors:         make(map[uint64]map[uint64]*Sector),
		sectorIdMap:     make(map[uint64]*Sector),
		fileSectorIdMap: make(map[string]uint64),
		db:              db,
		kill:            make(chan struct{}),
		sectorEventChan: make(chan *SectorEvent, 100),
		isLoading:       false,
	}
}

// sector manager serivce handles sector creation
func (this *SectorManager) StartSectorManagerService() {
	log.Debugf("[SectorManagerService] start service")
	for {
		select {
		case <-this.kill:
			log.Debugf("[SectorManagerService] service killed")
			return
		case event := <-this.sectorEventChan:
			switch event.Event {
			case SECTOR_EVENT_CREATE:
				go func() {
					_, err := this.CreateSector(event.SectorID, event.ProveLevel, event.Size)
					if err != nil {
						log.Errorf("[SectorManagerService] create sector error : %s", err)
					}
				}()
			default:
				log.Errorf("[SectorManagerService] unknown event %s", event.Event)
			}
		}
	}
}

func (this *SectorManager) NotifySectorEvent(event *SectorEvent) {
	log.Debugf("NotifySectorEvent:%v", event)
	this.sectorEventChan <- event
}

// create a new sector with given sector size, sector id is allocated automatically
func (this *SectorManager) CreateSector(sectorId uint64, proveLevel uint64, size uint64) (*Sector, error) {
	sector := this.GetSectorBySectorId(sectorId)
	if sector != nil {
		return nil, fmt.Errorf("create sector error, sector with id %d already exist", sectorId)
	}
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

	sector = InitSector(this, sectorId, size)
	sectors[sector.sectorId] = sector

	this.sectorIdMap[sectorId] = sector

	err := this.setProveParam(sector, proveLevel)
	if err != nil {
		log.Errorf("[CreateSector] setProveParm error %s", err)
		return nil, err
	}
	err = this.saveSectorList()
	if err != nil {
		log.Errorf("[CreateSector] saveSectorList error %s", err)
		return nil, err
	}

	log.Debugf("[CreateSector] Sector created with sectorid %d, proveLevel %d, size %d", sectorId, proveLevel, size)
	return sector, nil
}

func (this *SectorManager) DeleteSector(sectorId uint64) error {
	var sector *Sector

	this.lock.Lock()
	defer this.lock.Unlock()

	if sector = this.GetSectorBySectorId(sectorId); sector == nil {
		return fmt.Errorf("deleteSector no sector found with id %d", sectorId)
	}

	proveLevel := sector.GetProveLevel()
	sectors, exist := this.sectors[proveLevel]
	if !exist {
		return fmt.Errorf("deleteSector no sector found with proveLevel %d id %d", proveLevel, sectorId)
	}

	delete(sectors, sector.sectorId)
	delete(this.sectorIdMap, sector.sectorId)

	err := this.saveSectorList()
	if err != nil {
		log.Errorf("[DeleteSector] saveSectorList error %s", err)
		return err
	}

	log.Debugf("Sector deleted with sectorid %d", sectorId)
	return nil
}

func (this *SectorManager) GetSectorBySectorId(sectorId uint64) *Sector {
	sector, exist := this.sectorIdMap[sectorId]
	if !exist {
		return nil
	}
	return sector
}

// try add a file with a fileInfo to blocks which has enough size for file
func (this *SectorManager) AddFile(proveLevel uint64, fileHash string, blockCount uint64, blockSize uint64) (*Sector, error) {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.IsFileAdded(fileHash) {
		return nil, fmt.Errorf("addFile error, file %s is already added", fileHash)
	}

	sectors, exist := this.sectors[proveLevel]
	if !exist {
		return nil, fmt.Errorf("addFile error, no sector with prove level %d found for file %s", proveLevel, fileHash)
	}

	//to find the sector which is most suitable for the file storage
	// eg, if one sector has remaining size 2G, and we want to store a file with 1G, it should be
	// stored in this sector instead of putting it in a empty sector

	fileSize := blockCount * blockSize
	sectorId := this.FindMatchingSectorIdWithSize(sectors, fileSize)
	if sectorId == 0 {
		return nil, fmt.Errorf("addFile error, no matching sector found for file %s with size %d", fileHash, fileSize)
	}

	sector := this.GetSectorBySectorId(sectorId)
	err := sector.AddFileToSector(fileHash, blockCount, blockSize)
	if err != nil {
		return nil, fmt.Errorf("addFile error, addFileToSector error %v", err)
	}

	this.UpdateFileMap(fileHash, sector.sectorId, true)
	log.Debugf("Sector AddFile: file %s is added to sector %d", fileHash, sectorId)
	return sector, nil

}

func (this *SectorManager) DeleteFile(fileHash string) error {
	var sectorId uint64

	this.lock.Lock()
	defer this.lock.Unlock()

	if sectorId = this.GetFileSectorId(fileHash); sectorId == 0 {
		return fmt.Errorf("deleteFile, file %s is not in sectors", fileHash)
	}

	sector := this.GetSectorBySectorId(sectorId)
	if sector == nil {
		return fmt.Errorf("deleteFile, sector with id %d not found", sectorId)
	}

	err := sector.DeleteFileFromSector(fileHash)
	if err != nil {
		return fmt.Errorf("deleteFile, deleteFileFromSector for file %s error %s", fileHash, err)
	}

	this.UpdateFileMap(fileHash, sectorId, false)
	log.Debugf("Sector DeleteFile: file %s is deleted from sector %d", fileHash, sectorId)
	return nil
}

func (this *SectorManager) FindMatchingSectorIdWithSize(sectors map[uint64]*Sector, fileSize uint64) uint64 {
	candidates := make([]uint64, 0)
	for sectorId, sector := range sectors {
		if sector.GetSectorRemainingSize() >= fileSize {
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

func (this *SectorManager) IsFileAdded(fileHash string) bool {
	_, exist := this.fileSectorIdMap[fileHash]
	if exist {
		return true
	}
	return false
}

func (this *SectorManager) GetFileSectorId(fileHash string) uint64 {
	sectorId, exist := this.fileSectorIdMap[fileHash]
	if !exist {
		return 0
	}
	return sectorId
}

// update file map no lock
func (this *SectorManager) UpdateFileMap(fileHash string, sectorId uint64, isAdd bool) {
	if isAdd {
		this.fileSectorIdMap[fileHash] = sectorId
	} else {
		delete(this.fileSectorIdMap, fileHash)
	}
}

// find files and indexes by indexes in the sector
func (this *SectorManager) GetFilePosBySectorIndexes(sectorId uint64, indexes []uint64) ([]*FilePos, error) {
	sector := this.GetSectorBySectorId(sectorId)
	if sector == nil {
		return nil, fmt.Errorf("GetFilesBySectorIndexes, no sector found with id %d", sectorId)
	}
	return sector.GetFilePosBySectorIndexes(indexes)
}

func (this *SectorManager) setProveParam(sector *Sector, proveLevel uint64) error {
	sector.proveParam.ProveLevel = proveLevel
	sector.proveParam.Interval = getIntervalByProveLevel(proveLevel)
	sector.proveParam.ProveBlockNum = fs.SECTOR_PROVE_BLOCK_NUM

	return this.saveSectorProveParam(sector.sectorId)
}

func (this *SectorManager) GetAllSectorIds() ([]uint64, error) {
	this.lock.RLock()
	defer this.lock.RUnlock()

	sectorIds := make([]uint64, 0)
	for sectorId, _ := range this.sectorIdMap {
		sectorIds = append(sectorIds, sectorId)
	}
	return sectorIds, nil
}

func getIntervalByProveLevel(proveLevel uint64) uint64 {
	switch proveLevel {
	case fs.PROVE_LEVEL_HIGH:
		return fs.PROVE_PERIOD_HIGHT
	case fs.PROVE_LEVEL_MEDIEUM:
		return fs.PROVE_PERIOD_MEDIEUM
	case fs.PROVE_LEVEL_LOW:
		return fs.PROVE_PERIOD_LOW
	default:
	    panic("unknown prove level")
	}
}
