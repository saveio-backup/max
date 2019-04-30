package max

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	cid "gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"

	"github.com/saveio/max/max/fsstore"
	"github.com/saveio/themis/common"
	"github.com/saveio/themis/crypto/pdp"
)

// start the PDP prove service, if it is called first time for a file, it should submit one immediately
func (this *MaxService) StartPDPVerify(fileHash string, luckyNum, bakHeight, bakNum uint64, brokenWalletAddr common.Address) error {
	if this.IsFileStore() {
		return errors.New("cannot start pdp verify with filestore")
	}

	fsContract := this.chain.Native.Fs
	rootCid, err := cid.Decode(fileHash)
	if err != nil {
		return err
	}

	// check if stored in FS
	_, err = this.GetBlock(rootCid)
	if err != nil {
		return err
	}

	//check if already started a service
	if _, exist := this.provetasks.Load(fileHash); exist {
		return fmt.Errorf("PDP verify task for filehash: %s already started", fileHash)
	}

	once.Do(func() {
		go this.proveFileService()
	})

	if !this.loadingtasks {
		// store the task to db
		err = this.saveProveTask(fileHash, luckyNum, bakHeight, bakNum, brokenWalletAddr)
		if err != nil {
			return err
		}
	}
	this.provetasks.Store(fileHash, struct{}{})

	fileProveDetails, err := fsContract.GetFileProveDetails(fileHash)
	if err != nil {
		// not found prove for the first time
		go this.proveFile(true, fileHash, luckyNum, bakHeight, bakNum, brokenWalletAddr)
	} else {
		var found bool
		for _, detail := range fileProveDetails.ProveDetails {
			if detail.WalletAddr.ToBase58() == fsContract.DefAcc.Address.ToBase58() {
				found = true
				break
			}
		}

		// first prove, when prove detail found but not provetask, it means the fs node has restarted
		if !found {
			go this.proveFile(true, fileHash, luckyNum, bakHeight, bakNum, brokenWalletAddr)
		}
	}

	return nil
}

func (this *MaxService) saveProveTask(fileHash string, luckyNum, bakHeight, bakNum uint64, brokenWalletAddr common.Address) error {
	return this.fsstore.PutProveParam(fileHash, fsstore.NewProveParam(fileHash, luckyNum, bakHeight, bakNum, brokenWalletAddr))
}

func (this *MaxService) getProveTasks() ([]*fsstore.ProveParam, error) {
	return this.fsstore.GetProveParams()
}

func (this *MaxService) deleteProveTask(fileHash string) error {
	return this.fsstore.DeleteProveParam(fileHash)
}

func (this *MaxService) loadPDPTasksOnStartup() error {
	this.loadingtasks = true

	tasks, err := this.getProveTasks()
	if err != nil {
		return err
	}

	for _, param := range tasks {
		err = this.StartPDPVerify(param.FileHash, param.LuckyNum, param.BakHeight, param.BakNum, param.BrokenWalletAddr)
		if err != nil {
			return err
		}
	}

	this.loadingtasks = false
	return nil
}

func (this *MaxService) proveFileService() {
	ticker := time.NewTicker(time.Duration(PROVE_FILE_INTERVAL) * time.Second)
	for {
		select {
		case <-ticker.C:
			if this.chain == nil {
				break
			}

			var fileHashes []string
			this.provetasks.Range(func(key, value interface{}) bool {
				fileHashes = append(fileHashes, key.(string))
				return true
			})

			wg := sync.WaitGroup{}
			count := 0
			for _, hash := range fileHashes {
				wg.Add(1)
				count++
				go func(fileHash string) {
					this.proveFile(false, fileHash, 0, 0, 0, common.ADDRESS_EMPTY)
					wg.Done()
					count--
				}(hash)
				if count >= MAX_PROVE_FILE_ROUTINES {
					wg.Wait()
				}
			}
		}
	}
}

func (this *MaxService) proveFile(first bool, fileHash string, luckyNum, bakHeight, bakNum uint64, brokenWalletAddr common.Address) error {
	fsContract := this.chain.Native.Fs

	fileInfo, err := fsContract.GetFileInfo(fileHash)
	if err != nil {
		return err
	}

	height, err := fsContract.Client.GetCurrentBlockHeight()
	if err != nil {
		return err
	}

	hash, err := fsContract.Client.GetBlockHash(height)
	if err != nil {
		return err
	}

	if !first {
		var times uint64

		fileProveDetails, err := fsContract.GetFileProveDetails(fileHash)
		if err != nil {
			return err
		}

		for _, detail := range fileProveDetails.ProveDetails {
			if detail.WalletAddr.ToBase58() == fsContract.DefAcc.Address.ToBase58() {
				times = detail.ProveTimes
				break
			}
		}

		if times == fileInfo.ChallengeTimes+1 {
			err = this.DeleteFile(fileHash)
			if err != nil {
				return err
			}
			err = this.deleteProveTask(fileHash)
			if err != nil {
				return err
			}
			return nil
		}

		left := fileInfo.BlockHeight + times*fileInfo.ChallengeRate
		if uint64(height) < left {
			return nil
		}
	}

	_, err = this.internalProveFile(fileHash, fileInfo.FileBlockNum, fileInfo.ProveBlockNum, fileInfo.FileProveParam, hash, height, luckyNum, bakHeight, bakNum, brokenWalletAddr)
	if err != nil {
		return nil
	}

	// for backup node, after first prove, following prove parmeter should use default value
	if brokenWalletAddr != common.ADDRESS_EMPTY {
		err = this.saveProveTask(fileHash, 0, 0, 0, common.ADDRESS_EMPTY)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *MaxService) internalProveFile(fileHash string, blockNum, proveBlockNum uint64, fileProveParam []byte,
	hash common.Uint256, height uint32, luckyNum, bakHeight, bakNum uint64, badNodeWalletAddr common.Address) (bool, error) {
	fsContract := this.chain.Native.Fs

	challenges := fsContract.GenChallenge(fsContract.DefAcc.Address, hash, blockNum, proveBlockNum)

	tags := make([][]byte, 0)
	blocks := make([][]byte, 0)

	// get all cids
	rootCid, err := cid.Decode(fileHash)
	if err != nil {
		return false, err
	}
	cids, err := this.GetFileAllCids(context.TODO(), rootCid)
	if err != nil {
		return false, err
	}

	attrs := make(map[uint64]*fsstore.BlockAttr, 0)
	for _, cid := range cids {
		blockHash := cid.String()
		// get all indexes for a blockHash, blocks with same cid but differnt index have differnt tags
		indexes, err := this.getTagIndexes(blockHash, fileHash)
		if err != nil {
			return false, err
		}

		for _, index := range indexes {
			attrKey := fileHash + blockHash + strconv.FormatUint(index, 10)
			attr, err := this.fsstore.GetBlockAttr(attrKey)
			if err != nil {
				return false, err
			}
			attrs[attr.Index] = attr
		}
	}

	for _, c := range challenges {
		attr, ok := attrs[uint64(c.Index-1)]
		if !ok {
			return false, fmt.Errorf("file:%s, tag not found for index:%d", fileHash, c.Index)
		}
		tags = append(tags, attr.Tag)
		blockCId, err := cid.Decode(attr.Hash)
		if err != nil {
			return false, err
		}
		blk, err := this.GetBlock(blockCId)
		if err != nil {
			return false, err
		}
		blocks = append(blocks, blk.RawData())
	}

	err = this.proveFileStore(fileHash, uint64(height), challenges, tags, blocks, luckyNum, bakHeight, bakNum, badNodeWalletAddr)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (this *MaxService) proveFileStore(fileHash string, height uint64, challenges []pdp.Challenge, tags [][]byte, blocks [][]byte, luckyNum, bakHeight, bakNum uint64, badNodeWalletAddr common.Address) error {
	fsContract := this.chain.Native.Fs
	byteTags := make([]pdp.Element, 0)
	byteBlocks := make([]pdp.Block, 0)
	for _, tag := range tags {
		byteTags = append(byteTags, pdp.Element{
			Buffer: tag,
		})
	}
	for _, block := range blocks {
		byteBlocks = append(byteBlocks, pdp.Block{
			Buffer: block,
		})
	}

	multiRes, addRes := pdp.ProofGenerate(challenges, byteTags, byteBlocks)
	var proveErr error
	if bakNum == 0 {
		_, proveErr = fsContract.FileProve(fileHash, multiRes, addRes, height)
	} else {
		_, proveErr = fsContract.FileBackProve(fileHash, multiRes, addRes, height, luckyNum, bakHeight, bakNum, badNodeWalletAddr)
	}
	if proveErr != nil {
		return proveErr
	}
	// wait one confirmation
	return this.waitOneConfirmation(height)
}

func (this *MaxService) waitOneConfirmation(curBlockHeight uint64) error {
	fsContract := this.chain.Native.Fs
	retry := 0
	for {
		if retry > MAX_RETRY_REQUEST_TIMES {
			return errors.New("wait timeout")
		}
		height, _ := fsContract.Client.GetCurrentBlockHeight()
		if uint64(height) >= curBlockHeight+1 {
			return nil
		}
		retry++
		time.Sleep(time.Duration(MAX_REQUEST_TIMEWAIT) * time.Second)
	}
}
