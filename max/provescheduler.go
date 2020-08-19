package max

import (
	"fmt"
	"github.com/saveio/themis/common"
	"github.com/saveio/themis/common/log"
	fs "github.com/saveio/themis/smartcontract/service/native/savefs"
	"time"
)

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
	var err error
	for this.pdpQueue.Len() > 0 {
		item := this.pdpQueue.FirstItem()
		if item == nil {
			log.Debugf("no item in calculation queue")
			return
		}

		pdpItem := item.Value.(PDPItem)

		err = pdpItem.doPdpCalculation()
		if err != nil {
			err = pdpItem.onFailedPdpCalculation(err)
			if err != nil {
				log.Errorf("onFailedPdpCalculation error: %s", err)
			}
		} else {
			this.ScheduleForPdpSubmission(pdpItem)
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
	pdpItem := &FilePDPItem{
		FileHash:       fileHash,
		FileInfo:       fileInfo,
		NextChalHeight: nextPDPChalHeight,
		BlockHash:      blockHash,
		NextSubHeight:  nextPDPSubHeight,
		BakParam:       *bakParam,
		ExpireState:    expireState,
		FirstProve:     firstProve,
		max:            this,
	}

	item := &Item{
		Key:      fileHash,
		Value:    pdpItem,
		Priority: int(nextPDPChalHeight),
	}
	return this.pdpQueue.Push(item)
}

func (this *MaxService) ScheduleForPdpSubmission(item PDPItem) error {
	subItem := &Item{
		Key:      item.getItemKey(),
		Value:    item,
		Priority: int(item.getPdpSubmissionHeight()),
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

		pdpItem := item.Value.(PDPItem)
		itemKey := pdpItem.getItemKey()

		currentHeight, _ := this.getCurrentBlockHeightAndHash()
		if currentHeight >= pdpItem.getPdpSubmissionHeight() {
			log.Debugf("time to submit file prove for item %s", itemKey)

			// put in the submission map since the waitForConfirmation will block
			// following submission if not run in a go routine, and make sure it
			// can not be scheduled before submission finish
			this.submitting.Store(itemKey, struct{}{})

			go func() {
				defer this.submitting.Delete(itemKey)

				txHash, err := pdpItem.doPdpSubmission()
				if err != nil {
					// TODO : on prove error need to delete the file
					log.Errorf("doPdpSubmission for item %s error %s", itemKey, err)
					err = pdpItem.onFailedPdpSubmission(err)
					if err != nil {
						log.Errorf("onFailedPdpSubmission for item %s error %s", itemKey, err)
					}
				} else {
					_, err := this.pollForTxConfirmed(POLL_TX_CONFIRMED_TIMEOUT*time.Second, txHash)
					if err != nil {
						log.Errorf("pollForTxConfirmed for item %s error %s", itemKey, err)
						err = pdpItem.onFailedPdpSubmission(err)
						if err != nil {
							log.Errorf("onFailedPdpSubmission for item %s error %s", itemKey, err)
						}
					} else {
						log.Debugf("prove success for item : %s", itemKey)
						err = pdpItem.onSuccessfulPdpSubmission()
						if err != nil {
							log.Errorf("onSuccessPdpSubmission for item %s error %s", itemKey, err)
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
