package max

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	cid "gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"

	"github.com/saveio/themis/common/log"

	"github.com/saveio/max/max/fsstore"
	"github.com/saveio/themis/common"
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

	// no need for periodically file prove, after first successful file prove,
	// it will be covered by sector prove
	/*
		once.Do(func() {
			go this.proveFileService()
		})
	*/

	fileProveDetails, err := this.getFileProveDetails(fileHash)
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

	proveParam, err := this.getProveTask(fileHash)
	if proveParam == nil {
		err = this.saveProveTask(fileHash, 0, 0, 0,
			common.ADDRESS_EMPTY, 0, nil)
		if err != nil {
			log.Errorf("saveProveTask for fileHash %s error : %s", fileHash, err)
			return err
		}
		log.Debugf("saveProveTask when task not found")
	}

	this.provetasks.Store(fileHash, struct{}{})

	return nil
}

func (this *MaxService) saveProveTask(fileHash string, luckyNum, bakHeight, bakNum uint64,
	brokenWalletAddr common.Address, firstProveHeight uint64, pdpParam []byte) error {
	err := this.fsstore.PutProveParam(fileHash, fsstore.NewProveParam(fileHash, luckyNum, bakHeight, bakNum,
		brokenWalletAddr, firstProveHeight, pdpParam))
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

func (this *MaxService) getProveTask(fileHash string) (*fsstore.ProveParam, error) {
	param, err := this.fsstore.GetProveParam(fileHash)
	if err != nil {
		log.Errorf("[getProveTask] error : %s", err)
		return nil, err
	}
	return param, nil
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

func (this *MaxService) deleteProveTask(fileHash string, removeDB bool) error {
	log.Debugf("[deleteProveTask] delete task for fileHash : %s", fileHash)

	if _, ok := this.provetasks.Load(fileHash); ok {
		this.provetasks.Delete(fileHash)

		if removeDB {
			err := this.fsstore.DeleteProveParam(fileHash)
			if err != nil {
				log.Errorf("[deleteProveTask] delete prove task for fileHash: %s error : %s", fileHash, err)
				return err
			}
		}
	} else {
		log.Debugf("[deleteProveTask] task has already been deleted")
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
		case <-this.kill:
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

func (this *MaxService) deleteAndNotify(fileHash string, reason string) error {
	err := this.DeleteFile(fileHash)
	if err != nil {
		log.Errorf("[deleteAndNotify] DeleteFile for fileHash %s error : %s", fileHash, err)
		return err
	}
	this.notifyProveTaskDeletion(fileHash, reason)
	log.Debugf("[deleteAndNotify] success for %s, reason : %s", fileHash, reason)
	return nil
}

func (this *MaxService) proveFile(first bool, fileHash string, luckyNum, bakHeight, bakNum uint64, brokenWalletAddr common.Address) error {
	log.Debugf("[proveFile] first: %v, fileHash : %s, luckyNum : %d, bakHeight : %d, bakNum : %d, brokenWallet : %s",
		first, fileHash, luckyNum, bakHeight, bakNum, brokenWalletAddr.ToBase58())

	if this.IsScheduledForPdpCalculationOrSubmission(fileHash) {
		return nil
	}

	fsContract := this.chain.Native.Fs

	fileInfo, err := this.getFileInfo(fileHash)
	if err != nil {
		log.Errorf("[proveFile] GetFileInfo for fileHash : %s error : %s", fileHash, err)

		// prove task should be deleted when fileInfo is deleted
		if strings.Contains(err.Error(), "FsGetFileInfo not found!") {
			log.Debugf("[proveFile] GetFileInfo for fileHash : %s, fileInfo is deleted, remove prove task", fileHash)
			this.deleteAndNotify(fileHash, PROVE_TASK_REMOVAL_REASON_DELETE)
			return nil
		}
		return err
	}

	param, err := this.getProveTask(fileHash)
	if err == nil && param != nil {
		if param.FirstProveHeight != 0 && fileInfo.BlockHeight > param.FirstProveHeight {
			log.Debugf("[proveFile] fileInfo is renewed for %s, blockheight %d larger than first prove height %d ,remove prove task",
				fileHash, fileInfo.BlockHeight, param.FirstProveHeight)
			this.deleteAndNotify(fileHash, PROVE_TASK_REMOVAL_REASON_FILE_RENEWED)
			return nil
		}

		if param.FirstProveHeight != 0 {
			log.Debugf("no need prove file when first prove done")
			return nil
		}
	}

	height, hash := this.getCurrentBlockHeightAndHash()
	if height == 0 {
		log.Errorf("getCurrentBlockHeightAndHash error, block height is 0")
		return fmt.Errorf("getCurrentBlockHeightAndHash error, block height is 0")
	}

	expireState := (ExpireState)(EXPIRE_NONE)
	if !first {
		var times uint64
		var firstProveHeight uint64

		log.Debugf("[proveFile] not first prove for fileHash %s", fileHash)
		fileProveDetails, err := this.getFileProveDetails(fileHash)
		if err != nil {
			log.Errorf("[proveFile] GetFileProveDetails for fileHash %s error : %s", fileHash, err)
			if strings.Contains(err.Error(), "FsGetFileProveDetails not found!") {
				this.deleteAndNotify(fileHash, PROVE_TASK_REMOVAL_REASON_DELETE)
				return nil
			}
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

		addedToSector := false
		// process the case when prove record not found on first pdp submission
		sectorId := this.sectorManager.GetFileSectorId(fileHash)
		if sectorId == 0 {
			for _, sectorRef := range fileInfo.SectorRefs {
				if sectorRef.NodeAddr.ToBase58() == fsContract.DefAcc.Address.ToBase58() {
					// find matching file info, add  to sector
					_, err = this.sectorManager.AddFileToSector(fileInfo.ProveLevel, fileHash, fileInfo.FileBlockNum, fileInfo.FileBlockSize, sectorRef.SectorID)
					if err != nil {
						log.Errorf("AddFileToSector for file %s error %s", fileHash, err)
						return fmt.Errorf("AddFileToSector for file %s error %s", fileHash, err)
					}
					addedToSector = true
				}
			}
		} else {
			sector := this.sectorManager.GetSectorBySectorId(sectorId)
			if sector == nil {
				log.Errorf("sector %d not exist", sectorId)
				return fmt.Errorf("sector %d not exist", sectorId)
			}
			if sector.IsCandidateFile(fileHash) {
				err = this.sectorManager.MoveCandidateFileToSector(fileHash)
				if err != nil {
					log.Errorf("MoveCandidateFileToSector for file %s error %s", fileHash, err)
					return fmt.Errorf("MoveCandidateFileToSector for file %s error %s", fileHash, err)
				}
				addedToSector = true
			}
		}

		if addedToSector {
			log.Debugf("file %s has been added to sector %d when prove record found", fileHash, sectorId)

			// saves the first prove height to check if fileinfo is deleted then added agian
			err = this.saveProveTask(fileHash, 0, 0, 0,
				common.ADDRESS_EMPTY, uint64(height), fileInfo.FileProveParam)
			if err != nil {
				log.Errorf("saveProveTask for fileHash %s error : %s", fileHash, err)
				return err
			}
			log.Debugf("saveProveTask for file % with firstProveHeight %d", fileHash, height)
			return nil
		}

		log.Debugf("[proveFile]  fileHash : %s, times :%d, challengeTimes : %d", fileHash, times, fileInfo.ProveTimes)
		if times == fileInfo.ProveTimes+1 {
			log.Debugf("[proveFile] finish file prove for %s", fileHash)
			this.deleteAndNotify(fileHash, PROVE_TASK_REMOVAL_REASON_NORMAL)
			return nil
		}

		expireState = checkProveExpire(uint64(height), firstProveHeight, times, fileInfo.ProveInterval, fileInfo.ExpiredHeight)
		switch expireState {
		case EXPIRE_NEED_PROVE:
			log.Debugf("[proveFile] time to prove for fileHash :%s", fileHash)
			break
		case EXPIRE_LAST_PROVE:
			log.Debugf("[proveFile] last prove after reaching expired height for fileHash :%s, ", fileHash)
			break
		case EXPIRE_AFTER_MAX:
			log.Warnf("[proveFile] delete file and prove task for fileHash %s after prove task expire", fileHash)
			this.deleteAndNotify(fileHash, PROVE_TASK_REMOVAL_REASON_EXPIRE)
			return nil

		case EXPIRE_BEFORE_MIN:
			log.Debugf("[proveFile] file prove too early for file %s", fileHash)
			return nil
		default:
			log.Errorf("[proveFile] invalid expire state")
			return fmt.Errorf("invalid expire state")
		}
	}
	bakParam := BakParam{
		LuckyNum:          luckyNum,
		BakHeight:         bakHeight,
		BakNum:            bakNum,
		BadNodeWalletAddr: brokenWalletAddr,
	}

	filePdpItem := &FilePDPItem{
		FileHash:       fileHash,
		FileInfo:       fileInfo,
		NextChalHeight: height,
		BlockHash:      hash,
		NextSubHeight:  height,
		BakParam:       bakParam,
		ExpireState:    expireState,
		FirstProve:     first,
		max:            this,
	}
	err = this.scheduleForProve(filePdpItem)
	if err != nil {
		log.Errorf("failed to schedule for prove for file %s error %s", fileHash, err)
		return err
	}
	log.Debugf(" schedule file %s for pdp calculation", fileHash)
	return nil
}

func (this *MaxService) pollForTxConfirmed(timeout time.Duration, txHash []byte) (bool, error) {
	fsContract := this.chain.Native.Fs
	return fsContract.PollForTxConfirmed(timeout, txHash)
}

type ExpireState int

const (
	EXPIRE_NONE       = iota
	EXPIRE_BEFORE_MIN // before min time to submit the prove
	EXPIRE_AFTER_MAX  // after max time to submit the prove
	EXPIRE_NEED_PROVE // need to submit the prove
	EXPIRE_LAST_PROVE // last prove after reach expied height
)

func checkProveExpire(currBlockHeight uint64, firstProveHeight uint64, provedTimes uint64, challengeRate uint64, expiredHeight uint64) ExpireState {
	expireMinHeight := firstProveHeight + provedTimes*challengeRate
	expireMaxHeight := firstProveHeight + (provedTimes+1)*challengeRate

	log.Debugf("[checkProveExpire] currBlockHeight :%d, firstProveHeight :%d, provedTimes :%d, challengeRate :%d, expiredHeight : %d",
		currBlockHeight, firstProveHeight, provedTimes, challengeRate, expiredHeight)
	if currBlockHeight > expireMaxHeight {
		return EXPIRE_AFTER_MAX
	} else if currBlockHeight < expireMinHeight {
		// dont wait for next interval if expiredHeight has reached
		if currBlockHeight > expiredHeight {
			return EXPIRE_LAST_PROVE
		}
		return EXPIRE_BEFORE_MIN
	} else {
		return EXPIRE_NEED_PROVE
	}
}
func getTxHashString(txHash []byte) string {
	hash, err := common.Uint256ParseFromBytes(txHash)
	if err != nil {
		log.Errorf("parse tx hash error")
		return "error parsing tx hash"
	}
	return hash.ToHexString()
}
