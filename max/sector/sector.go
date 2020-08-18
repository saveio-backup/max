package sector

import (
	"fmt"
	"github.com/saveio/themis/common/log"
	fs "github.com/saveio/themis/smartcontract/service/native/savefs"
	"strings"
	"sync"
)

type SectorProveParam struct {
	proveLevel       uint64 // now prove level will decide the interval
	interval         uint64
	firstProveHeight uint64
	lastProveHeight  uint64
	nextProveHeight  uint64
}

type Sector struct {
	lock          sync.RWMutex
	sectorId      uint64
	sectorSize    uint64
	fileList      []string // TODO: may not be enough just to know the fileHash, some info maybe needed like when to expire
	fileMap       map[string]struct{}
	fileTotalSize uint64 // sum of file size for files in the sector
	proveParam    SectorProveParam
}

func initSector(id uint64, size uint64) *Sector {
	return &Sector{
		sectorId:   id,
		sectorSize: size,
		fileList:   make([]string, 0),
		fileMap:    make(map[string]struct{}),
	}
}

func (this *Sector) addFileToSector(fileInfo *fs.FileInfo) error {
	if fileInfo == nil {
		return fmt.Errorf("addFileToSector, fileInfo is nil")
	}

	fileHash := string(fileInfo.FileHash)

	if !fileInfo.ValidFlag {
		return fmt.Errorf("addFileToSector, file %s is invalid", fileHash)
	}

	if this.isFileInSector(fileHash) {
		return fmt.Errorf("addFileToSector, file %s already in sector", fileHash)
	}

	if this.getSectorRemainingSize() < fileInfo.FileBlockNum*fileInfo.FileBlockSize {
		return fmt.Errorf("addFileToSector, not enought space left in sector for file %s", fileHash)
	}

	this.lock.Lock()
	this.fileList = append(this.fileList, fileHash)
	this.fileMap[fileHash] = struct{}{}
	this.lock.Unlock()

	return nil
}

func (this *Sector) deleteFileFromSector(fileHash string) error {
	// no need to check if valid in case file is deleted by user

	if !this.isFileInSector(fileHash) {
		return fmt.Errorf("deleteFileFromSector file %s is not in sector", fileHash)
	}

	this.lock.Lock()

	for i := 0; i < len(this.fileList); i++ {
		if strings.Compare(this.fileList[i], fileHash) == 0 {
			this.fileList = append(this.fileList[:i], this.fileList[i+1:]...)
		}
	}
	delete(this.fileMap, fileHash)
	this.lock.Unlock()
	return nil
}

func (this *Sector) isFileInSector(fileHash string) bool {
	this.lock.RLock()
	defer this.lock.RUnlock()
	_, exist := this.fileMap[fileHash]
	return exist
}

func (this *Sector) getSectorRemainingSize() uint64 {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.sectorSize - this.fileTotalSize
}

func (this *Sector) getProveLevel() uint64 {
	return this.proveParam.proveLevel
}

func (this *Sector) updateLastProveHeight(height uint64) error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if height <= this.proveParam.lastProveHeight {
		log.Errorf("updateLastProveHeight error, height %d is no larger than last prove height %d", height, this.proveParam.lastProveHeight)
		return fmt.Errorf("updateLastProveHeight error, height %d is no larger than last prove height %d", height, this.proveParam.lastProveHeight)
	}
	this.proveParam.lastProveHeight = height
	return nil
}
