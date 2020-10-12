package max

import (
	"github.com/saveio/themis/common/log"
	"strconv"
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
	for this.pdpQueue.Len() > 0 {
		item := this.pdpQueue.FirstItem()
		if item == nil {
			log.Debugf("no item in calculation queue")
			return
		}

		pdpItem := item.Value.(PDPItem)

		itemKey := pdpItem.getItemKey()
		nextChalHeight := pdpItem.getPdpCalculationHeight()

		needSubmit := false
		// try find saved but not submitted result from db to reduce duplicate pdp calculation
		proveData, err := this.getPdpCalculationResult(pdpItem, nextChalHeight)
		if err != nil || proveData == nil {
			err = pdpItem.doPdpCalculation()
			if err != nil {
				err = pdpItem.onFailedPdpCalculation(err)
				if err != nil {
					log.Errorf("onFailedPdpCalculation error: %s", err)
				}
			} else {
				err = this.savePdpCalculationResult(pdpItem, pdpItem.getPdpCalculationHeight(), pdpItem.getPdpCalculationResult())
				if err != nil {
					log.Errorf("save pdp calculation result for pdp item %s error %s", itemKey, err)
				} else {
					log.Debugf("calculate and save pdp result success for pdp item %s", itemKey, item)
					needSubmit = true
				}
			}
		} else {
			needSubmit = true
			log.Debugf("cached pdp result found for pdp item %s, height %d", itemKey, nextChalHeight)
		}

		if needSubmit {
			err = this.ScheduleForPdpSubmission(pdpItem)
			if err != nil {
				log.Errorf("schedule for pdp submission for pdp item %s error %s", itemKey, err)
			} else {
				log.Debugf("schedule for pdp submission for pdp item %s success", itemKey)
			}
		}

		// NOTE: cannot use pop here, since queue might have been updated
		removed := this.pdpQueue.Remove(item.Key)
		if !removed {
			log.Errorf("error to remove item with key %v from pdp queue", item.Key)
		}
	}
	return
}

func (this *MaxService) IsScheduledForPdpCalculationOrSubmission(key string) bool {
	if this.pdpQueue.IndexByKey(key) != -1 {
		log.Debugf("item with key %s has already been scheduled for pdp calculation", key)
		return true
	}
	if this.submitQueue.IndexByKey(key) != -1 {
		log.Debugf("item with key %s has already been scheduled for pdp submission", key)
		return true
	}

	if _, ok := this.submitting.Load(key); ok {
		log.Debugf("item with key %s is in submitting queue", key)
		return true
	}

	log.Debugf("item with key %s has not been scheduled for pdp calculation or submission", key)
	return false
}

// push to pdp queue to schedule for pdp calculation
func (this *MaxService) scheduleForProve(pdpItem PDPItem) error {
	log.Debugf("scheduleForProve for pdpItem %v", pdpItem)
	log.Debugf("scheduleForProve  with key %s, challengeHeight %d", pdpItem.getItemKey(), pdpItem.getPdpCalculationHeight())
	item := &Item{
		Key:      pdpItem.getItemKey(),
		Value:    pdpItem,
		Priority: int(pdpItem.getPdpCalculationHeight()),
	}
	return this.pdpQueue.Push(item)
}

func (this *MaxService) ScheduleForPdpSubmission(pdpItem PDPItem) error {
	log.Debugf("ScheduleForPdpSubmission for pdpItem %v", pdpItem)
	log.Debugf("ScheduleForPdpSubmission with key %s, submissionHeight %d", pdpItem.getItemKey(), pdpItem.getPdpSubmissionHeight())
	subItem := &Item{
		Key:      pdpItem.getItemKey(),
		Value:    pdpItem,
		Priority: int(pdpItem.getPdpSubmissionHeight()),
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

		log.Debugf("process pdp submission for item %s", itemKey)
		currentHeight, _ := this.getCurrentBlockHeightAndHash()
		if currentHeight >= pdpItem.getPdpSubmissionHeight() {
			log.Debugf("time to submit prove for item %s", itemKey)

			// put in the submission map since the waitForConfirmation will block
			// following submission if not run in a go routine, and make sure it
			// can not be scheduled before submission finish
			this.submitting.Store(itemKey, struct{}{})

			go func() {
				defer this.submitting.Delete(itemKey)

				var proveData []byte
				var err error

				height := pdpItem.getPdpSubmissionHeight()

				proveData, err = this.getPdpCalculationResult(pdpItem, height)
				if err != nil || proveData == nil {
					log.Errorf("get pdp calculation result for pdp item %s error %s", itemKey, err)
					return
				}

				txHash, err := pdpItem.doPdpSubmission(proveData)
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

func (this *MaxService) savePdpCalculationResult(pdpItem PDPItem, height uint32, proveData []byte) error {
	itemKey := pdpItem.getItemKey()
	if !pdpItem.shouldSavePdpResult() {
		log.Debugf("savePdpCalculationResult, no need to save to db for %v", itemKey)
		return nil
	}

	log.Debugf("save pdp calculation result for pdp item %s, height %d, proveData %v", pdpItem, height, proveData)
	return this.savePdpResultToDB(pdpItem, height, proveData)
}

func (this *MaxService) getPdpCalculationResult(pdpItem PDPItem, height uint32) (proveData []byte, err error) {
	itemKey := pdpItem.getItemKey()
	if !pdpItem.shouldSavePdpResult() {
		log.Debugf("getPdpCalculationResult, no need to save to db for %v", itemKey)
		return pdpItem.getPdpCalculationResult(), nil
	}

	proveData, err = this.getPdpResultFromDB(pdpItem, height)
	if err == nil {
		log.Debugf("get pdp calculation result for pdp item %s, proveData %v", itemKey, proveData)
	}
	return proveData, err
}

const FILE_PDP_RESULT_KEY = "pdpresult:"

func genPdpResultKey(pdpItem PDPItem, height uint32) string {
	return FILE_PDP_RESULT_KEY + strconv.Itoa(int(height))
}

func (this *MaxService) savePdpResultToDB(pdpItem PDPItem, height uint32, proveData []byte) error {
	return this.fsstore.PutData(genPdpResultKey(pdpItem, height), proveData)
}

func (this *MaxService) getPdpResultFromDB(pdpItem PDPItem, height uint32) ([]byte, error) {
	return this.fsstore.GetData(genPdpResultKey(pdpItem, height))
}
