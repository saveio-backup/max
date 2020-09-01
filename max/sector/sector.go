package sector

import (
	"fmt"
	"github.com/saveio/themis/common/log"
	//fs "github.com/saveio/themis/smartcontract/service/native/savefs"
	"strings"
	"sync"
)

type SectorFileInfo struct {
	FileHash   string `json:"filehash"`
	BlockCount uint64 `json:"blockcount"`
	BlockSize  uint64 `json:"blocksize"`
}

type SectorProveParam struct {
	ProveLevel       uint64 // now prove level will decide the interval
	Interval         uint64
	ProveBlockNum    uint64
	FirstProveHeight uint64
	LastProveHeight  uint64
	NextProveHeight  uint64
}

type Sector struct {
	lock            sync.RWMutex
	manager         *SectorManager
	sectorId        uint64
	sectorSize      uint64
	fileList        []*SectorFileInfo // TODO: may not be enough just to know the fileHash, some info maybe needed like when to expire
	fileMap         map[string]struct{}
	totalFileSize   uint64 // sum of file size for files in the sector
	totalBlockCount uint64 // all blocks in the sector
	proveParam      SectorProveParam
}

func InitSector(manager *SectorManager, id uint64, size uint64) *Sector {
	return &Sector{
		manager:    manager,
		sectorId:   id,
		sectorSize: size,
		fileList:   make([]*SectorFileInfo, 0),
		fileMap:    make(map[string]struct{}),
	}
}

func (this *Sector) GetSectorID() uint64 {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.sectorId
}

func (this *Sector) GetNextProveHeight() uint64 {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.proveParam.NextProveHeight
}

func (this *Sector) SetNextProveHeight(height uint64) error {
	this.lock.Lock()
	defer this.lock.Unlock()

	curHeight := this.proveParam.NextProveHeight
	if height < curHeight {
		log.Errorf("height %d is smaller than next prove height %d", height, curHeight)
		return fmt.Errorf("height %d is smaller than next prove height %d", height, curHeight)
	}
	this.proveParam.NextProveHeight = height
	log.Debugf("SetNextProveHeight for sector %d as %d", this.sectorId, height)
	return this.saveProveParam()
}

func (this *Sector) GetLastProveHeight() uint64 {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.proveParam.LastProveHeight
}
func (this *Sector) SetLastProveHeight(height uint64) error {
	this.lock.Lock()

	curHeight := this.proveParam.LastProveHeight
	if height < curHeight {
		log.Errorf("height %d is smaller than last prove height %d", height, curHeight)
		return fmt.Errorf("height %d is smaller than last prove height %d", height, curHeight)
	}
	this.proveParam.LastProveHeight = height

	err := this.saveProveParam()
	if err != nil {
		return err
	}

	this.lock.Unlock()

	if this.proveParam.FirstProveHeight == 0 {
		return this.SetFirstProveHeight(height)
	}
	return nil
}

func (this *Sector) GetFirstProveHeight() uint64 {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.proveParam.FirstProveHeight
}

func (this *Sector) SetFirstProveHeight(height uint64) error {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.proveParam.FirstProveHeight = height

	return this.saveProveParam()
}

func (this *Sector) GetProveInterval() uint64 {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.proveParam.Interval
}

func (this *Sector) AddFileToSector(fileHash string, blockCount uint64, blockSize uint64) error {
	if this.IsFileInSector(fileHash) {
		return fmt.Errorf("addFileToSector, file %s already in sector", fileHash)
	}

	fileSize := blockCount * blockSize

	if this.GetSectorRemainingSize() < fileSize {
		return fmt.Errorf("addFileToSector, not enought space left in sector for file %s", fileHash)
	}

	this.lock.Lock()
	defer this.lock.Unlock()

	this.fileList = append(this.fileList, &SectorFileInfo{
		FileHash:   fileHash,
		BlockCount: blockCount,
		BlockSize:  blockSize,
	})
	this.fileMap[fileHash] = struct{}{}
	this.totalFileSize += fileSize
	this.totalBlockCount += blockCount

	err := this.saveFileList()
	if err != nil {
		return err
	}

	return nil
}

func (this *Sector) DeleteFileFromSector(fileHash string) error {
	// no need to check if valid in case file is deleted by user

	if !this.IsFileInSector(fileHash) {
		return fmt.Errorf("deleteFileFromSector file %s is not in sector", fileHash)
	}

	this.lock.Lock()
	defer this.lock.Unlock()

	for i := 0; i < len(this.fileList); i++ {
		file := this.fileList[i]
		if strings.Compare(file.FileHash, fileHash) == 0 {
			this.totalFileSize = this.totalFileSize - file.BlockCount*file.BlockSize
			this.totalBlockCount = this.totalBlockCount - file.BlockCount
			this.fileList = append(this.fileList[:i], this.fileList[i+1:]...)
			break
		}
	}
	delete(this.fileMap, fileHash)

	err := this.saveFileList()
	if err != nil {
		return err
	}

	return nil
}

func (this *Sector) IsFileInSector(fileHash string) bool {
	this.lock.RLock()
	defer this.lock.RUnlock()
	_, exist := this.fileMap[fileHash]
	return exist
}

func (this *Sector) GetSectorRemainingSize() uint64 {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.sectorSize - this.totalFileSize
}

func (this *Sector) GetProveLevel() uint64 {
	return this.proveParam.ProveLevel
}

func (this *Sector) GetTotalBlockCount() uint64 {
	return this.totalBlockCount
}

func (this *Sector) GetProveBlockNum() uint64 {
	return this.proveParam.ProveBlockNum
}

type FilePos struct {
	FileHash     string
	BlockIndexes []uint64
}

// find files and indexes by indexes in the sector
func (this *Sector) GetFilePosBySectorIndexes(indexes []uint64) ([]*FilePos, error) {
	indexCount := len(indexes)
	if indexCount == 0 {
		return nil, fmt.Errorf("GetFilesBySectorIndexes, no indexes is provided")
	}

	// check if indexes is in order
	var preIndex uint64
	for _, index := range indexes {
		if index >= this.totalBlockCount {
			return nil, fmt.Errorf("GetFilesBySectorIndexes, index %d is larger than sector blockCount %d", index, this.totalBlockCount)
		}

		if preIndex != 0 && index <= preIndex {
			return nil, fmt.Errorf("GetFilesBySectorIndexes, index %d is no larger than preIndex %d", index, preIndex)
		}
		preIndex = index
	}

	var offset uint64
	var curIndex = 0

	filePos := make([]*FilePos, 0)
	for _, fileInfo := range this.fileList {
		blockCount := fileInfo.BlockCount

		start := offset
		end := offset + blockCount - 1

		pos := &FilePos{
			FileHash:     fileInfo.FileHash,
			BlockIndexes: make([]uint64, 0),
		}
		for i := curIndex; i < len(indexes); i++ {
			index := indexes[curIndex]
			if index >= start && index <= end {
				pos.BlockIndexes = append(pos.BlockIndexes, index-start)

				curIndex++
				// reach end of indexes, return the result
				if curIndex >= indexCount {
					filePos = append(filePos, pos)
					return filePos, nil
				}
				continue
			}
			break
		}

		if len(pos.BlockIndexes) != 0 {
			filePos = append(filePos, pos)
		}
		offset += blockCount
	}

	return filePos, nil
}

func (this *Sector) saveFileList() error {
	if this.manager == nil {
		return nil
	}
	return this.manager.saveSectorFileList(this.sectorId)
}

func (this *Sector) saveProveParam() error {
	if this.manager == nil {
		return nil
	}
	return this.manager.saveSectorProveParam(this.sectorId)
}

func (this *Sector) LockSector() {
	this.lock.Lock()
}

func (this *Sector) UnLockSector() {
	this.lock.Unlock()
}
