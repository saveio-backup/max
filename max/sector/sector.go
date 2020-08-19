package sector

import (
	"fmt"
	"github.com/saveio/themis/common/log"
	//fs "github.com/saveio/themis/smartcontract/service/native/savefs"
	"strings"
	"sync"
)

type SectorFileInfo struct {
	fileHash   string
	blockCount uint64
	blockSize  uint64
}

type SectorProveParam struct {
	proveLevel       uint64 // now prove level will decide the interval
	interval         uint64
	proveBlockNum    uint64
	firstProveHeight uint64
	lastProveHeight  uint64
	nextProveHeight  uint64
}

type Sector struct {
	lock            sync.RWMutex
	sectorId        uint64
	sectorSize      uint64
	fileList        []*SectorFileInfo // TODO: may not be enough just to know the fileHash, some info maybe needed like when to expire
	fileMap         map[string]struct{}
	totalFileSize   uint64 // sum of file size for files in the sector
	totalBlockCount uint64 // all blocks in the sector
	proveParam      SectorProveParam
}

func InitSector(id uint64, size uint64) *Sector {
	return &Sector{
		sectorId:   id,
		sectorSize: size,
		fileList:   make([]*SectorFileInfo, 0),
		fileMap:    make(map[string]struct{}),
	}
}

func (this *Sector) GetSectorID() uint64 {
	return this.sectorId
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
	this.fileList = append(this.fileList, &SectorFileInfo{
		fileHash:   fileHash,
		blockCount: blockCount,
		blockSize:  blockSize,
	})
	this.fileMap[fileHash] = struct{}{}
	this.totalFileSize += fileSize
	this.totalBlockCount += blockCount
	this.lock.Unlock()

	return nil
}

func (this *Sector) DeleteFileFromSector(fileHash string) error {
	// no need to check if valid in case file is deleted by user

	if !this.IsFileInSector(fileHash) {
		return fmt.Errorf("deleteFileFromSector file %s is not in sector", fileHash)
	}

	this.lock.Lock()

	for i := 0; i < len(this.fileList); i++ {
		file := this.fileList[i]
		if strings.Compare(file.fileHash, fileHash) == 0 {
			this.totalFileSize = this.totalFileSize - file.blockCount*file.blockSize
			this.totalBlockCount = this.totalBlockCount - file.blockCount
			this.fileList = append(this.fileList[:i], this.fileList[i+1:]...)
			break
		}
	}
	delete(this.fileMap, fileHash)

	this.lock.Unlock()
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
	return this.proveParam.proveLevel
}

func (this *Sector) UpdateLastProveHeight(height uint64) error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if height <= this.proveParam.lastProveHeight {
		log.Errorf("updateLastProveHeight error, height %d is no larger than last prove height %d", height, this.proveParam.lastProveHeight)
		return fmt.Errorf("updateLastProveHeight error, height %d is no larger than last prove height %d", height, this.proveParam.lastProveHeight)
	}
	this.proveParam.lastProveHeight = height
	return nil
}

func (this *Sector) GetTotalBlockCount() uint64 {
	return this.totalBlockCount
}

func (this *Sector) GetProveBlockNum() uint64 {
	return this.proveParam.proveBlockNum
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
		blockCount := fileInfo.blockCount

		start := offset
		end := offset + blockCount - 1

		pos := &FilePos{
			FileHash:     fileInfo.fileHash,
			BlockIndexes: make([]uint64, 0),
		}
		for i := curIndex; i < len(indexes); i++ {
			index := indexes[curIndex]
			if index >= start && index < end {
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
