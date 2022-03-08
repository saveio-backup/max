package max

import (
	"fmt"
	"github.com/saveio/themis/common"
	"github.com/saveio/themis/common/log"
	fs "github.com/saveio/themis/smartcontract/service/native/savefs"
	"sync"
)

type Cache struct {
	infoLock sync.RWMutex
	fileInfo map[string]*fs.FileInfo // fileHash -> fileInfo

	detailLock   sync.RWMutex
	proveDetails map[string]*fs.FsProveDetails // fileHash -> proveDetails

	blockHashLock sync.RWMutex
	currentHeight uint32
	blockHash     common.Uint256
}

func NewCache() *Cache {
	return &Cache{
		fileInfo:     make(map[string]*fs.FileInfo),
		proveDetails: make(map[string]*fs.FsProveDetails),
	}
}

func (this *Cache) getFileInfo(fileHash string) (*fs.FileInfo, error) {
	this.infoLock.RLock()
	defer this.infoLock.RUnlock()

	if fileInfo, ok := this.fileInfo[fileHash]; ok {
		return fileInfo, nil
	}
	return nil, fmt.Errorf("fileInfo not found for file %s in cache", fileHash)
}

func (this *Cache) addFileInfo(fileHash string, fileInfo *fs.FileInfo) {
	this.infoLock.Lock()
	defer this.infoLock.Unlock()
	this.fileInfo[fileHash] = fileInfo
}

func (this *Cache) deleteFileInfo(fileHash string) {
	this.infoLock.Lock()
	defer this.infoLock.Unlock()

	if _, ok := this.fileInfo[fileHash]; ok {
		delete(this.fileInfo, fileHash)
	}
}

func (this *Cache) getProveDetails(fileHash string) (*fs.FsProveDetails, error) {
	this.detailLock.RLock()
	defer this.detailLock.RUnlock()

	if proveDetails, ok := this.proveDetails[fileHash]; ok {
		return proveDetails, nil
	}
	return nil, fmt.Errorf("proveDetails not found for file %s in cache", fileHash)
}

func (this *Cache) addProveDetails(fileHash string, proveDetails *fs.FsProveDetails) {
	this.detailLock.Lock()
	defer this.detailLock.Unlock()
	this.proveDetails[fileHash] = proveDetails
}

func (this *Cache) deleteProveDetails(fileHash string) {
	this.detailLock.Lock()
	defer this.detailLock.Unlock()

	if _, ok := this.proveDetails[fileHash]; ok {
		delete(this.proveDetails, fileHash)
	}
}

func (this *Cache) updateCurrentHeightAndHash(height uint32, blockHash common.Uint256) {
	this.blockHashLock.Lock()
	defer this.blockHashLock.Unlock()

	this.currentHeight = height
	this.blockHash = blockHash
}

func (this *Cache) getCurrentHeightAndHash() (uint32, common.Uint256) {
	this.blockHashLock.RLock()
	defer this.blockHashLock.RUnlock()
	return this.currentHeight, this.blockHash
}

func (this *MaxService) getFileProveDetails(fileHash string) (*fs.FsProveDetails, error) {
	// may get wrong prove detail when it is updated
	/*
		proveDetails, err := this.rpcCache.getProveDetails(fileHash)
		if err == nil {
			return proveDetails, nil
		}
	*/

	fsContract := this.chain.Native.Fs
	proveDetails, err := fsContract.GetFileProveDetails(fileHash)
	if err != nil {
		return nil, err
	}

	this.rpcCache.addProveDetails(fileHash, proveDetails)
	return proveDetails, nil
}

func (this *MaxService) getFileInfo(fileHash string) (*fs.FileInfo, error) {
	fileInfo, err := this.rpcCache.getFileInfo(fileHash)
	if err == nil {
		return fileInfo, nil
	}

	fsContract := this.chain.Native.Fs
	fileInfo, err = fsContract.GetFileInfo(fileHash)
	if err != nil {
		return nil, err
	}

	this.rpcCache.addFileInfo(fileHash, fileInfo)
	return fileInfo, nil
}

// get current block height and hash from cache, height value 0 means failed to get height
func (this *MaxService) getCurrentBlockHeightAndHash() (uint32, common.Uint256) {
	return this.rpcCache.getCurrentHeightAndHash()
}

func (this *MaxService) updateCurrentBlockHeightAndHash(height uint32, blockHash common.Uint256) {
	this.rpcCache.updateCurrentHeightAndHash(height, blockHash)
}

func (this *MaxService) getCurrentBlockHeightAndHashFromChainAndUpdateCache() (uint32, common.Uint256, error) {
	var emptyHash common.Uint256

	fsContract := this.chain.Native.Fs

	height, err := fsContract.Client.GetClient().GetCurrentBlockHeight()
	if err != nil {
		log.Errorf("GetCurrentBlockHeight error : %s", err)
		return 0, emptyHash, err
	}

	blockHash, err := fsContract.Client.GetClient().GetBlockHash(height)
	if err != nil {
		log.Errorf("GetBlockHash error : %s", err)
		return 0, emptyHash, err
	}
	this.updateCurrentBlockHeightAndHash(height, blockHash)
	return height, blockHash, nil
}
