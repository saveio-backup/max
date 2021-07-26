package sector

import (
	"fmt"
	"github.com/saveio/themis/common/log"
	//fs "github.com/saveio/themis/smartcontract/service/native/savefs"
	"sort"
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
	lock                sync.RWMutex
	manager             *SectorManager
	SectorId            uint64
	SectorSize          uint64
	FileList            *sync.Map // fileHash -> *SectorFileInfo
	CandidateFileList   *sync.Map // fileHash -> *SectorFileInfo
	isCandidateLocked   bool
	candidateUnlockChan chan struct{}
	candidateEmptyChan  chan struct{}
	TotalFileSize       uint64 // sum of file size for files in the sector
	TotalBlockCount     uint64 // all blocks in the sector
	ProveParam          SectorProveParam
	IsPlots             bool
}

func InitSector(manager *SectorManager, id uint64, size uint64, isPlots bool) *Sector {
	return &Sector{
		manager:           manager,
		SectorId:          id,
		SectorSize:        size,
		FileList:          new(sync.Map),
		CandidateFileList: new(sync.Map),
		IsPlots:           isPlots,
	}
}

func (this *Sector) GetSectorID() uint64 {
	//no need for lock, sector id never change
	return this.SectorId
}

func (this *Sector) GetSectorSize() uint64 {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.SectorSize
}

func (this *Sector) GetTotalFileSize() uint64 {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.TotalFileSize
}

func (this *Sector) IsPlotSector() bool {
	return this.IsPlots
}

func (this *Sector) GetProveLevel() uint64 {
	return this.GetProveParam().ProveLevel
}

func (this *Sector) GetTotalBlockCount() uint64 {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.TotalBlockCount
}

func (this *Sector) GetTotalBlockCountNoLock() uint64 {
	return this.TotalBlockCount
}

func (this *Sector) GetProveParam() *SectorProveParam {
	return &this.ProveParam
}

func (this *Sector) GetNextProveHeight() uint64 {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.GetProveParam().NextProveHeight
}

func (this *Sector) SetNextProveHeight(height uint64) error {
	this.lock.Lock()
	defer this.lock.Unlock()

	curHeight := this.GetProveParam().NextProveHeight
	if height < curHeight {
		log.Errorf("height %d is smaller than next prove height %d", height, curHeight)
		return fmt.Errorf("height %d is smaller than next prove height %d", height, curHeight)
	}
	this.GetProveParam().NextProveHeight = height
	log.Debugf("SetNextProveHeight for sector %d as %d", this.SectorId, height)
	return this.saveProveParam()
}

func (this *Sector) GetLastProveHeight() uint64 {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.GetProveParam().LastProveHeight
}
func (this *Sector) SetLastProveHeight(height uint64) error {
	this.lock.Lock()

	curHeight := this.GetProveParam().LastProveHeight
	if height < curHeight {
		log.Errorf("height %d is smaller than last prove height %d", height, curHeight)
		return fmt.Errorf("height %d is smaller than last prove height %d", height, curHeight)
	}
	this.GetProveParam().LastProveHeight = height

	err := this.saveProveParam()
	if err != nil {
		return err
	}

	this.lock.Unlock()

	if this.GetProveParam().FirstProveHeight == 0 {
		return this.SetFirstProveHeight(height)
	}
	return nil
}

func (this *Sector) GetFirstProveHeight() uint64 {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.GetProveParam().FirstProveHeight
}

func (this *Sector) SetFirstProveHeight(height uint64) error {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.GetProveParam().FirstProveHeight = height

	return this.saveProveParam()
}

func (this *Sector) GetProveInterval() uint64 {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.GetProveParam().Interval
}

func (this *Sector) AddFileToSector(fileHash string, blockCount uint64, blockSize uint64, isPlots bool) error {
	if this.IsFileInSector(fileHash) {
		return fmt.Errorf("addFileToSector, file %s already in sector", fileHash)
	}

	if this.IsCandidateFile(fileHash) {
		return fmt.Errorf("addFileToSector, file %s is candiate file", fileHash)
	}

	if isPlots != this.IsPlotSector() {
		return fmt.Errorf("addFileToSector, file %s plot type not match", fileHash)
	}

	fileSize := blockCount * blockSize

	this.lock.Lock()
	defer this.lock.Unlock()

	if this.GetSectorRemainingSizeNoLock() < fileSize {
		return fmt.Errorf("addFileToSector, not enought space left in sector for file %s", fileHash)
	}

	this.FileList.Store(fileHash, &SectorFileInfo{
		FileHash:   fileHash,
		BlockCount: blockCount,
		BlockSize:  blockSize,
	})
	this.TotalFileSize += fileSize
	this.TotalBlockCount += blockCount

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

	value, ok := this.FileList.Load(fileHash)
	if !ok {
		log.Errorf("deleteFileFromSector error, file %s sectorFileInfo not found", fileHash)
		return fmt.Errorf("deleteFileFromSector error, file %s sectorFileInfo not found", fileHash)
	}

	sectorFileInfo := value.(*SectorFileInfo)

	this.TotalFileSize = this.TotalFileSize - sectorFileInfo.BlockCount*sectorFileInfo.BlockSize
	this.TotalBlockCount = this.TotalBlockCount - sectorFileInfo.BlockCount

	this.FileList.Delete(fileHash)

	err := this.saveFileList()
	if err != nil {
		return err
	}

	return nil
}

func (this *Sector) AddCandidateFile(fileHash string, blockCount uint64, blockSize uint64, isPlots bool) error {
	if this.IsFileInSector(fileHash) {
		log.Errorf("addCandidateFile, file %s is already in sector %d", fileHash, this.SectorId)
		return fmt.Errorf("addCandidateFile, file %s is already in sector %d", fileHash, this.SectorId)
	}

	if this.IsCandidateFile(fileHash) {
		log.Errorf("addCandidateFile, file %s is already candidate file in sector %d", fileHash, this.SectorId)
		return fmt.Errorf("addCandidateFile, file %s is already candidate file in sector %d", fileHash, this.SectorId)
	}

	if isPlots != this.IsPlotSector() {
		return fmt.Errorf("addCandidateFile, file %s plot type not match with sector %d", fileHash, this.SectorId)
	}

	// block if sector prvoe is ongoing, since prove task needs to block until it has result for file pdp submission
	// to have the same view of files in sector as fs contract
	this.BlockUntilCandidateListUnlocked()

	this.lock.Lock()
	defer this.lock.Unlock()

	if this.GetSectorRemainingSizeNoLock() < blockCount*blockSize {
		return fmt.Errorf("addCandidateFile, not enought space left in sector for candidate file %s", fileHash)
	}

	this.CandidateFileList.Store(fileHash, &SectorFileInfo{
		FileHash:   fileHash,
		BlockCount: blockCount,
		BlockSize:  blockSize,
	})

	err := this.saveCandidateFileList()
	if err != nil {
		return err
	}

	log.Debugf("candidate file %s is added to sector %d", fileHash, this.SectorId)
	return nil
}

func (this *Sector) IsCandidateFile(fileHash string) bool {
	_, exist := this.CandidateFileList.Load(fileHash)
	return exist
}

func (this *Sector) GetCandidateFileTotalSize() uint64 {
	var totalsize uint64

	this.CandidateFileList.Range(func(key, value interface{}) bool {
		sectorFileInfo := value.(*SectorFileInfo)
		totalsize += sectorFileInfo.BlockCount * sectorFileInfo.BlockSize
		return true
	})

	log.Debugf("candidate files total size is %d for sector %d", totalsize, this.SectorId)
	return totalsize
}

// NOTE: use with caution, delete candidate file will not update file map
func (this *Sector) DeleteCandidateFile(fileHash string) error {
	if !this.IsCandidateFile(fileHash) {
		log.Errorf("deleteCandidateFile, file %s is not candidate file in sector %d", fileHash, this.SectorId)
		return fmt.Errorf("deleteCandidateFile, file %s is not candidate file in sector %d", fileHash, this.SectorId)
	}

	this.lock.Lock()
	defer this.lock.Unlock()

	this.CandidateFileList.Delete(fileHash)

	if len(this.GetCandidateFileList()) == 0 && this.candidateEmptyChan != nil {
		close(this.candidateEmptyChan)
		this.candidateEmptyChan = nil
	}

	err := this.saveCandidateFileList()
	if err != nil {
		return err
	}
	log.Debugf("candidate file %s is deleted from sector %d", fileHash, this.SectorId)
	return nil
}

func (this *Sector) MoveCandidateFileToFileList(fileHash string) error {
	if !this.IsCandidateFile(fileHash) {
		log.Errorf("moveCandidateFileToFileList, file %s is not candidate file in sector %d", fileHash, this.SectorId)
		return fmt.Errorf("moveCandidateFileToFileList, file %s is not candidate file in sector %d", fileHash, this.SectorId)
	}

	this.lock.Lock()
	defer this.lock.Unlock()

	value, ok := this.CandidateFileList.Load(fileHash)
	if !ok {
		log.Errorf("moveCandidateFileToFileList, load file %s from candidate list of sector %d failed", fileHash, this.SectorId)
		return fmt.Errorf("moveCandidateFileToFileList, load file %s from candidate list of sector %d failed", fileHash, this.SectorId)
	}

	sectorFileInfo := value.(*SectorFileInfo)

	this.FileList.Store(fileHash, sectorFileInfo)

	this.TotalFileSize += sectorFileInfo.BlockSize * sectorFileInfo.BlockCount
	this.TotalBlockCount += sectorFileInfo.BlockCount

	this.CandidateFileList.Delete(fileHash)

	if len(this.GetCandidateFileList()) == 0 && this.candidateEmptyChan != nil {
		close(this.candidateEmptyChan)
		this.candidateEmptyChan = nil
	}

	err := this.saveCandidateFileList()
	if err != nil {
		return err
	}

	err = this.saveFileList()
	if err != nil {
		return err
	}

	log.Debugf("candidate file %s is moved to file list in sector %d", fileHash, this.SectorId)
	return nil
}

func (this *Sector) GetCandidateFileList() []*SectorFileInfo {
	sectorFileInfos := make([]*SectorFileInfo, 0)

	this.CandidateFileList.Range(func(key, value interface{}) bool {
		info := value.(*SectorFileInfo)
		sectorFileInfos = append(sectorFileInfos, &SectorFileInfo{
			FileHash:   info.FileHash,
			BlockCount: info.BlockCount,
			BlockSize:  info.BlockSize,
		})
		return true
	})
	return sectorFileInfos
}

func (this *Sector) UnlockCandidateList() {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.isCandidateLocked = false

	if this.candidateUnlockChan != nil {
		close(this.candidateUnlockChan)
		this.candidateUnlockChan = nil
	}
	log.Debugf("UnlockCandidateList for sector %d", this.SectorId)
}

func (this *Sector) BlockUntilCandidateListEmpty() {
	this.lock.Lock()
	this.isCandidateLocked = true

	log.Debugf("BlockUntilCandidateListEmpty for sector %d", this.SectorId)
	if len(this.GetCandidateFileList()) == 0 {
		this.lock.Unlock()
		log.Debugf("BlockUntilCandidateListEmpty for sector %d, no need to wait", this.SectorId)
		return
	}
	if this.candidateEmptyChan == nil {
		this.candidateEmptyChan = make(chan struct{})
	}
	this.lock.Unlock()

	<-this.candidateEmptyChan
	log.Debugf("BlockUntilCandidateListEmpty for sector %d done", this.SectorId)
}

func (this *Sector) BlockUntilCandidateListUnlocked() {
	this.lock.Lock()

	log.Debugf("BlockUntilCandidateListUnlocked for sector %d", this.SectorId)
	if !this.isCandidateLocked {
		this.lock.Unlock()
		log.Debugf("BlockUntilCandidateListUnlocked for sector %d, no need to wait", this.SectorId)
		return
	}

	if this.candidateUnlockChan == nil {
		this.candidateUnlockChan = make(chan struct{})
	}

	this.lock.Unlock()

	<-this.candidateUnlockChan
	log.Debugf("BlockUntilCandidateListUnlocked for sector %d done", this.SectorId)
}

// return the file in alphabetic order
func (this *Sector) GetFileList() []*SectorFileInfo {
	sectorFileInfos := make([]*SectorFileInfo, 0)

	this.FileList.Range(func(key, value interface{}) bool {
		sectorFileInfos = append(sectorFileInfos, value.(*SectorFileInfo))
		return true
	})

	sort.SliceStable(sectorFileInfos, func(i, j int) bool {
		return strings.Compare(sectorFileInfos[i].FileHash, sectorFileInfos[j].FileHash) == -1
	})
	return sectorFileInfos
}

func (this *Sector) GetFileHashList() []string {
	fileList := make([]string, 0)

	for _, file := range this.GetFileList() {
		fileList = append(fileList, file.FileHash)
	}
	return fileList
}

func (this *Sector) IsFileInSector(fileHash string) bool {
	_, exist := this.FileList.Load(fileHash)
	return exist
}

func (this *Sector) GetSectorRemainingSize() uint64 {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.GetSectorRemainingSizeNoLock()
}

func (this *Sector) GetSectorRemainingSizeNoLock() uint64 {
	candidateSize := this.GetCandidateFileTotalSize()
	remainingSize := this.SectorSize - this.TotalFileSize - candidateSize
	log.Debugf("sector %d remaining size is %d", remainingSize)
	return remainingSize
}

func (this *Sector) GetProveBlockNum() uint64 {
	return this.GetProveParam().ProveBlockNum
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
		if index >= this.TotalBlockCount {
			return nil, fmt.Errorf("GetFilesBySectorIndexes, index %d is larger than sector blockCount %d", index, this.TotalBlockCount)
		}

		if preIndex != 0 && index <= preIndex {
			return nil, fmt.Errorf("GetFilesBySectorIndexes, index %d is no larger than preIndex %d", index, preIndex)
		}
		preIndex = index
	}

	var offset uint64
	var curIndex = 0

	filePos := make([]*FilePos, 0)
	for _, fileInfo := range this.GetFileList() {
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
	return this.manager.saveSectorFileList(this.SectorId)
}

func (this *Sector) saveProveParam() error {
	if this.manager == nil {
		return nil
	}
	return this.manager.saveSectorProveParam(this.SectorId)
}

func (this *Sector) saveCandidateFileList() error {
	if this.manager == nil {
		return nil
	}
	return this.manager.saveSectorCandidateFileList(this.SectorId)
}
