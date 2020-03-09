package max

import (
	"fmt"
	"github.com/saveio/themis/common"
	"github.com/saveio/themis/common/log"
	fs "github.com/saveio/themis/smartcontract/service/native/savefs"
)

type PDPCalItem struct {
	FileHash       string
	FileInfo       *fs.FileInfo
	NextChalHeight uint32
	BlockHash      common.Uint256
	NextSubHeight  uint32
	BakParam       BakParam
	ExpireState    ExpireState
	FirstProve     bool
}

type BakParam struct {
	LuckyNum          uint64
	BakHeight         uint64
	BakNum            uint64
	BadNodeWalletAddr common.Address
}

type PDPResult struct {
	MultiRes []byte
	AddRes   string
}

func (this *MaxService) startPdpCalculationService() {
	log.Debugf("start pdp calculation service")

	for {
		select {
		case <-this.pdpQueue.GetPushNotifyChan():
			this.processPdpCalculationQueue()
		case <-this.kill:
			log.Debug("quit pdp calculation service")
			return
		}
	}
}

// process item in the pdp calculation queue
func (this *MaxService) processPdpCalculationQueue() {
	for this.pdpQueue.Len() > 0 {
		item := this.pdpQueue.FirstItem()
		if item == nil {
			log.Debugf("no item in calculation queue")
			return
		}

		pdpCalItem := item.Value.(*PDPCalItem)
		fileHash := pdpCalItem.FileHash

		pdpResult, err := this.doPdpCalculation(pdpCalItem)
		if err != nil {
			log.Errorf("do pdp calculation for file %s error %s", fileHash, err)
			this.deleteAndNotify(pdpCalItem.FileHash, PROVE_TASK_REMOVAL_REASON_PDP_CALCULATION)
		} else {
			this.ScheduleForPdpSubmission(pdpCalItem, pdpResult)
		}

		// NOTE: cannot use pop here, since queue might have been updated
		removed := this.pdpQueue.Remove(item.Key)
		if !removed {
			log.Errorf("error to remove item with key %v from pdp queue", item.Key)
		}
	}
	return
}

func (this *MaxService) IsScheduledForPdpCalculationOrSubmission(fileHash string) bool {
	if this.pdpQueue.IndexByKey(fileHash) != -1 {
		log.Debugf("file %s has already been scheduled for pdp calculation", fileHash)
		return true
	}
	if this.submitQueue.IndexByKey(fileHash) != -1 {
		log.Debugf("file %s has already been scheduled for pdp submission", fileHash)
		return true
	}

	if _, ok := this.submitting.Load(fileHash); ok {
		log.Debugf("file %s is in submitting queue")
		return true
	}

	log.Debugf("file %s has not been scheduled for pdp calculation or submission", fileHash)
	return false
}

// push to pdp queue to schedule for pdp calculation
func (this *MaxService) scheduleForProve(fileInfo *fs.FileInfo, bakParam *BakParam, nextPDPChalHeight uint32, blockHash common.Uint256,
	nextPDPSubHeight uint32, expireState ExpireState, firstProve bool) error {
	if fileInfo == nil {
		log.Errorf("scheduleForProve error, fileInfo is nil")
		return fmt.Errorf("scheduleForProve error, fileInfo is nil")
	}
	fileHash := string(fileInfo.FileHash)
	item := &Item{
		Key: fileHash,
		Value: &PDPCalItem{
			FileHash:       fileHash,
			FileInfo:       fileInfo,
			NextChalHeight: nextPDPChalHeight,
			BlockHash:      blockHash,
			NextSubHeight:  nextPDPSubHeight,
			BakParam:       *bakParam,
			ExpireState:    expireState,
			FirstProve:     firstProve,
		},
		Priority: int(nextPDPChalHeight),
	}
	return this.pdpQueue.Push(item)
}

type PdpSubItem struct {
	FileHash       string
	PdpResult      PDPResult
	BakParam       BakParam
	NextChalHeight uint32
	NextSubHeight  uint32
	FirstProve     bool
	ExpireState    ExpireState
}

func (this *MaxService) ScheduleForPdpSubmission(item *PDPCalItem, pdpResult *PDPResult) error {
	subItem := &Item{
		Key: item.FileHash,
		Value: &PdpSubItem{
			FileHash:       item.FileHash,
			PdpResult:      *pdpResult,
			BakParam:       item.BakParam,
			NextChalHeight: item.NextChalHeight,
			NextSubHeight:  item.NextSubHeight,
			FirstProve:     item.FirstProve,
			ExpireState:    item.ExpireState,
		},
		Priority: int(item.NextSubHeight),
		Index:    0,
	}
	return this.submitQueue.Push(subItem)
}

func (this *MaxService) startPdpSubmissionService() {
	log.Debugf("start pdp submission service")

	for {
		select {
		case <-this.submitQueue.GetPushNotifyChan():
			this.processPdpSubmissionQueue()
		case <-this.kill:
			log.Debug("quit pdp submission service")
			return
		}
	}
}

func (this *MaxService) processPdpSubmissionQueue() {
	for this.submitQueue.Len() > 0 {
		item := this.submitQueue.FirstItem()
		if item == nil {
			log.Errorf("no item in submission queue")
			return
		}

		pdpSubItem := item.Value.(*PdpSubItem)
		fileHash := pdpSubItem.FileHash

		currentHeight, _ := this.getCurrentBlockHeightAndHash()
		if currentHeight >= pdpSubItem.NextSubHeight {
			log.Debugf("time to submit file prove for file %s", fileHash)

			// put in the submission map since the waitForConfirmation will block
			// following submission if not run in a go routine, and make sure it
			// can not be scheduled before submission finish
			this.submitting.Store(fileHash, struct{}{})

			go func() {
				defer this.submitting.Delete(fileHash)

				err := this.doPdpSubmission(pdpSubItem)
				if err != nil {
					// TODO : on prove error need to delete the file
					log.Errorf("doPdpSubmission for file %s error %s", fileHash, err)
				} else {
					err = this.waitForConfirmation(uint64(currentHeight))
					if err != nil {
						log.Errorf("waitForConfirmation for file %s error %s", fileHash, err)
					} else {
						log.Debugf("prove success for fileHash : %s", fileHash)
						err = this.onSuccessPdpSubmission(pdpSubItem)
						if err != nil {
							log.Errorf("onSuccessPdpSubmission for file %s error %s", fileHash, err)
						}
					}
				}
			}()

			removed := this.submitQueue.Remove(item.Key)
			if !removed {
				log.Errorf("error to remove item with key %v from submit queue", item.Key)
			}
		} else {
			log.Debugf("not time to submit file prove")
			return
		}
	}
	return
}
