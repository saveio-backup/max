package max

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	cid "gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"

	"github.com/saveio/themis/common/log"

	"github.com/saveio/max/max/fsstore"
	"github.com/saveio/themis/common"
	"github.com/saveio/themis/crypto/pdp"
)

// start the PDP prove service, if it is called first time for a file, it should submit one immediately
func (this *MaxService) StartPDPVerify(fileHash string, luckyNum, bakHeight, bakNum uint64, brokenWalletAddr common.Address) error {
	log.Debugf("[StartPDPVerify] fileHash : %s, luckyNum : %d, bakHeight : %d, bakNum : %d, brokenWalletAddr : %s",
		fileHash, luckyNum, bakHeight, bakNum, brokenWalletAddr.ToBase58())

	if this.IsFileStore() {
		log.Errorf("[StartPDPVerify] cannot start pdp verify with filestore")
		return errors.New("cannot start pdp verify with filestore")
	}

	fsContract := this.chain.Native.Fs
	rootCid, err := cid.Decode(fileHash)
	if err != nil {
		log.Errorf("[StartPDPVerify] decode filehash %s error : %s", fileHash, err)
		return err
	}

	// check if stored in FS
	_, err = this.GetBlock(rootCid)
	if err != nil {
		log.Errorf("[StartPDPVerify] GetBlock for %s error : %s", rootCid.String(), err)
		return err
	}

	//check if already started a service
	if _, exist := this.provetasks.Load(fileHash); exist {
		log.Errorf("[StartPDPVerify] PDP verify task for filehash: %s already started", fileHash)
		return fmt.Errorf("PDP verify task for filehash: %s already started", fileHash)
	}

	once.Do(func() {
		go this.proveFileService()
	})

	fileProveDetails, err := fsContract.GetFileProveDetails(fileHash)
	if err != nil {
		// not found prove for the first time
		log.Debugf("[StartPDPVerify] first prove for filehash: %s, GetFileProveDetails error : %s", fileHash, err)
		err = this.proveFile(true, fileHash, luckyNum, bakHeight, bakNum, brokenWalletAddr)
		if err != nil {
			log.Debugf("[StartPDPVerify] proveFile for filehash: %s, error : %s", fileHash, err)
			return err
		}
	} else {
		var found bool
		for _, detail := range fileProveDetails.ProveDetails {
			if detail.WalletAddr.ToBase58() == fsContract.DefAcc.Address.ToBase58() {
				found = true
				log.Debugf("[StartPDPVerify] prove detail found with matching address for filehash: %s", fileHash)
				break
			}
		}

		// first prove, when prove detail found but not provetask, it means the fs node has restarted
		if !found {
			log.Debugf("[StartPDPVerify] first prove when prove detail found for filehash: %s", fileHash)
			err = this.proveFile(true, fileHash, luckyNum, bakHeight, bakNum, brokenWalletAddr)
			if err != nil {
				log.Debugf("[StartPDPVerify] proveFile for filehash: %s, error : %s", fileHash, err)
				return err
			}
		}
	}

	// save prove task only when first prove success
	if !this.loadingtasks {
		// store the task to db
		err = this.saveProveTask(fileHash, luckyNum, bakHeight, bakNum, brokenWalletAddr)
		if err != nil {
			log.Errorf("[StartPDPVerify] saveProveTask for filehash: %s error : %s", fileHash, err)
			return err
		}
	}
	this.provetasks.Store(fileHash, struct{}{})

	return nil
}

func (this *MaxService) saveProveTask(fileHash string, luckyNum, bakHeight, bakNum uint64, brokenWalletAddr common.Address) error {
	err := this.fsstore.PutProveParam(fileHash, fsstore.NewProveParam(fileHash, luckyNum, bakHeight, bakNum, brokenWalletAddr))
	if err != nil {
		log.Errorf("[saveProveTask] PutProveParam error: %v, for fileHash : %s, luckyNum : %d, bakHeight : %d, bakNum : %d, brokenWalletAddr : %s",
			err, fileHash, luckyNum, bakHeight, bakNum, brokenWalletAddr.ToBase58())
		return err
	}
	return nil
}

func (this *MaxService) getProveTasks() ([]*fsstore.ProveParam, error) {
	params, err := this.fsstore.GetProveParams()
	if err != nil {
		log.Errorf("[getProveTasks] error : %s", err)
		return nil, err
	}
	return params, nil
}

func (this *MaxService) notifyProveTaskDeletion(fileHash string, reason string) {
	notify := &ProveTaskRemovalNotify{
		FileHash: fileHash,
		Reason:   reason,
	}

	go func() {
		log.Debugf("[notifyProveTaskDeletion] notify remove prove task for fileHash : %s, reason : %s", fileHash, reason)
		this.Notify <- notify
	}()
}

func (this *MaxService) deleteProveTask(fileHash string) error {
	this.provetasks.Delete(fileHash)

	err := this.fsstore.DeleteProveParam(fileHash)
	if err != nil {
		log.Errorf("[deleteProveTask] delete prove task for fileHash: %s error : %s", fileHash, err)
		return err
	}

	return nil
}

func (this *MaxService) loadPDPTasksOnStartup() error {
	log.Debugf("[loadPDPTasksOnStartup] start")
	this.loadingtasks = true

	tasks, err := this.getProveTasks()
	if err != nil {
		log.Errorf("[loadPDPTasksOnStartup] getProveTasks error : %s", err)
		return err
	}

	for _, param := range tasks {
		err = this.StartPDPVerify(param.FileHash, param.LuckyNum, param.BakHeight, param.BakNum, param.BrokenWalletAddr)
		if err != nil {
			log.Errorf("[loadPDPTasksOnStartup] StartPDPVerify for fileHash %s error : %s", param.FileHash, err)
		}
	}

	this.loadingtasks = false
	log.Debugf("[loadPDPTasksOnStartup] finish")
	return nil
}

func (this *MaxService) proveFileService() {
	log.Debugf("[proveFileService] start service")
	ticker := time.NewTicker(time.Duration(PROVE_FILE_INTERVAL) * time.Second)
	for {
		select {
		case <-this.killprove:
			log.Debugf("[proveFileService] service killed")
			return
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
	log.Debugf("[proveFile] first: %v, fileHash : %s, luckyNum : %d, bakHeight : %d, bakNum : %d, brokenWallet : %s",
		first, fileHash, luckyNum, bakHeight, bakNum, brokenWalletAddr.ToBase58())
	fsContract := this.chain.Native.Fs

	fileInfo, err := fsContract.GetFileInfo(fileHash)
	if err != nil {
		log.Errorf("[proveFile] GetFileInfo for fileHash : %s error : %s", fileHash, err)

		// prove task should be deleted when fileInfo is deleted
		if strings.Contains(err.Error(), "FsGetFileInfo not found!") {
			log.Debugf("[proveFile] GetFileInfo for fileHash : %s, fileInfo is deleted, remove prove task", fileHash)
			err = this.DeleteFile(fileHash)
			if err != nil {
				log.Errorf("[proveFile] DeleteFile for fileHash %s error : %s", fileHash, err)
				return err
			}
			log.Debugf("[proveFile] file info deleted, remove file prove for %s", fileHash)
			this.notifyProveTaskDeletion(fileHash, PROVE_TASK_REMOVAL_REASON_DELETE)
			return nil
		}
		return err
	}

	height, err := fsContract.Client.GetCurrentBlockHeight()
	if err != nil {
		log.Errorf("[proveFile] GetCurrentBlockHeight error : %s", err)
		return err
	}

	hash, err := fsContract.Client.GetBlockHash(height)
	if err != nil {
		log.Errorf("[proveFile] GetBlockHash for height %d error : %s", height, err)
		return err
	}

	if !first {
		var times uint64
		var firstProveHeight uint64

		log.Debugf("[proveFile] not first prove for fileHash %s", fileHash)
		fileProveDetails, err := fsContract.GetFileProveDetails(fileHash)
		if err != nil {
			log.Errorf("[proveFile] GetFileProveDetails for fileHash %s error : %s", fileHash, err)
			return err
		}

		found := false
		for _, detail := range fileProveDetails.ProveDetails {
			if detail.WalletAddr.ToBase58() == fsContract.DefAcc.Address.ToBase58() {
				times = detail.ProveTimes
				firstProveHeight = detail.BlockHeight
				log.Debugf("[proveFile] find matching prove detail for fileHash : %s, times :%d", fileHash, times)
				found = true
				break
			}
		}

		if !found {
			log.Debugf("[proveFile] cannot find matching prove detail for fileHash : %s", fileHash)
			return fmt.Errorf("prove detail not found for fileHash %s", fileHash)
		}

		log.Debugf("[proveFile]  fileHash : %s, times :%d, challengeTimes : %d", fileHash, times, fileInfo.ProveTimes)
		if times == fileInfo.ProveTimes+1 {
			err = this.DeleteFile(fileHash)
			if err != nil {
				log.Errorf("[proveFile] DeleteFile for fileHash %s error : %s", fileHash, err)
				return err
			}
			log.Debugf("[proveFile] finish file prove for %s", fileHash)
			this.notifyProveTaskDeletion(fileHash, PROVE_TASK_REMOVAL_REASON_NORMAL)
			return nil
		}

		expireState := checkProveExpire(uint64(height), firstProveHeight, times, fileInfo.ProveInterval)
		switch expireState {
		case EXPIRE_NEED_PROVE:
			log.Debugf("[proveFile] time to prove for fileHash :%s", fileHash)
			break
		case EXPIRE_AFTER_MAX:
			err = this.DeleteFile(fileHash)
			if err != nil {
				log.Errorf("[proveFile] DeleteFile for fileHash %s after prove task expire error : %s", fileHash, err)
				return err
			}
			log.Warnf("[proveFile] delete file and prove task for fileHash %s after prove task expire", fileHash)
			this.notifyProveTaskDeletion(fileHash, PROVE_TASK_REMOVAL_REASON_EXPIRE)
			return nil

		case EXPIRE_BEFORE_MIN:
			log.Debugf("[proveFile] file prove too early for file %s", fileHash)
			return nil
		default:
			log.Errorf("[proveFile] invalid expire state")
			return fmt.Errorf("invalid expire state")
		}
	}

	_, err = this.internalProveFile(fileHash, fileInfo.FileBlockNum, fileInfo.ProveBlockNum, fileInfo.FileProveParam, hash, height, luckyNum, bakHeight, bakNum, brokenWalletAddr)
	if err != nil {
		log.Errorf("[proveFile] internalProveFile for fileHash %s error : %s", fileHash, err)
		return err
	}

	// for backup node, after first prove, following prove parmeter should use default value
	if brokenWalletAddr != common.ADDRESS_EMPTY {
		err = this.saveProveTask(fileHash, 0, 0, 0, common.ADDRESS_EMPTY)
		if err != nil {
			log.Errorf("[proveFile] saveProveTask for fileHash %s error : %s", fileHash, err)
			return err
		}
		log.Debugf("[proveFile] save prove task with updated parameter for backup node after first prove")
	}
	return nil
}

func (this *MaxService) internalProveFile(fileHash string, blockNum, proveBlockNum uint64, fileProveParam []byte,
	hash common.Uint256, height uint32, luckyNum, bakHeight, bakNum uint64, badNodeWalletAddr common.Address) (bool, error) {
	log.Debugf("[internalProveFile] fileHash : %s, blockNum : %d, proveBlockNum : %d, fileProveParam : %v, hash : %d, height : %d, luckyNum :%d, bakNum : %d, badNodeWalletAddr : %s",
		fileHash, blockNum, proveBlockNum, fileProveParam, hash.ToHexString(), height, luckyNum, bakHeight, bakNum, badNodeWalletAddr.ToBase58())
	fsContract := this.chain.Native.Fs

	challenges := fsContract.GenChallenge(fsContract.DefAcc.Address, hash, blockNum, proveBlockNum)

	tags := make([][]byte, 0)
	blocks := make([][]byte, 0)

	// get all cids
	rootCid, err := cid.Decode(fileHash)
	if err != nil {
		log.Errorf("[internalProveFile] Decode for fileHash %s error : %s", fileHash, err)
		return false, err
	}
	cids, err := this.GetFileAllCids(context.TODO(), rootCid)
	if err != nil {
		log.Errorf("[internalProveFile] GetFileAllCids for rootCid %s error : %s", rootCid.String(), err)
		return false, err
	}

	var blockCid *cid.Cid
	var index uint64
	var attrKey string
	var attr *fsstore.BlockAttr

	cidsLen := uint64(len(cids))
	for _, c := range challenges {
		index = uint64(c.Index - 1)
		if index+1 > cidsLen {
			log.Errorf("[internalProveFile] invalid index for fileHash %s index %d", fileHash, index)
			return false, fmt.Errorf("file:%s, invalid index:%d", fileHash, index)
		}

		blockCid = cids[index]
		attrKey = fileHash + blockCid.String() + strconv.FormatUint(index, 10)
		attr, err = this.fsstore.GetBlockAttr(attrKey)
		if err != nil {
			log.Errorf("[internalProveFile] GetBlockAttr for blockHash %s fileHash %s index %d error : %s", blockCid.String(), fileHash, index, err)
			return false, err
		}

		tags = append(tags, attr.Tag)
		blk, err := this.GetBlock(blockCid)
		if err != nil {
			log.Errorf("[internalProveFile] GetBlock for block %s error : %s", blockCid.String(), err)
			return false, err
		}
		blocks = append(blocks, blk.RawData())
	}

	err = this.proveFileStore(fileHash, uint64(height), challenges, tags, blocks, luckyNum, bakHeight, bakNum, badNodeWalletAddr)
	if err != nil {
		log.Errorf("[internalProveFile] proveFileStore for fileHash %s error : %s", fileHash, err)
		return false, err
	}
	log.Debugf("[internalProveFile] prove succsess for fileHash : %s, blockNum : %d, proveBlockNum : %d, fileProveParam : %v, hash : %d, height : %d, luckyNum :%d, bakNum : %d, badNodeWalletAddr : %s",
		fileHash, blockNum, proveBlockNum, fileProveParam, hash.ToHexString(), height, luckyNum, bakHeight, bakNum, badNodeWalletAddr.ToBase58())
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
		log.Errorf("[proveFileStore] file prove error : %s bakNum : %d", proveErr, bakNum)
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
			log.Errorf("[waitOneConfirmation] wait timeout")
			return errors.New("wait timeout")
		}
		height, _ := fsContract.Client.GetCurrentBlockHeight()
		if uint64(height) >= curBlockHeight+1 {
			log.Debugf("[waitOneConfirmation] wait ok")
			return nil
		}
		retry++
		time.Sleep(time.Duration(MAX_REQUEST_TIMEWAIT) * time.Second)
	}
}

type ExpireState int

const (
	EXPIRE_NODE       = iota
	EXPIRE_BEFORE_MIN // before min time to submit the prove
	EXPIRE_AFTER_MAX  // after max time to submit the prove
	EXPIRE_NEED_PROVE // need to submit the prove
)

func checkProveExpire(currBlockHeight uint64, firstProveHeight uint64, provedTimes uint64, challengeRate uint64) ExpireState {
	expireMinHeight := firstProveHeight + provedTimes*challengeRate
	expireMaxHeight := firstProveHeight + (provedTimes+1)*challengeRate

	log.Debugf("[checkProveExpire] currBlockHeight :%d, firstProveHeight :%d, provedTimes :%d, challengeRate :%d",
		currBlockHeight, firstProveHeight, provedTimes, challengeRate)
	if currBlockHeight > expireMaxHeight {
		return EXPIRE_AFTER_MAX
	} else if currBlockHeight < expireMinHeight {
		return EXPIRE_BEFORE_MIN
	} else {
		return EXPIRE_NEED_PROVE
	}
}
