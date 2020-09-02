package max

import (
	"github.com/saveio/themis/common/log"
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
