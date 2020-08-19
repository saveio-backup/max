package max

import (
	"fmt"
	"github.com/saveio/max/max/sector"
	"github.com/saveio/themis/common/log"
	"github.com/saveio/themis/smartcontract/service/native/utils"
	"reflect"
	"time"
)

func (this *MaxService) StartEventFilter(interval uint32) error {
	if this.chain == nil {
		return nil
	}

	// make sure this run before load pdp task to get block height and hash
	_, _, err := this.getCurrentBlockHeightAndHashFromChainAndUpdateCache()
	if err != nil {
		return err
	}

	go func() {
		var latestHeight uint32
		var currHeight uint32
		var err error

		fsContractAddr := utils.OntFSContractAddress.ToHexString()

		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		log.Debugf("start event filter")

		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				currHeight, _, err = this.getCurrentBlockHeightAndHashFromChainAndUpdateCache()
				if err != nil {
					log.Errorf("get block height and hash from chain error %s", err)
					continue
				}

				if currHeight <= latestHeight {
					continue
				}

				for i := latestHeight + 1; i <= currHeight; i++ {
					// get events by height
					events, err := this.getContractEvents(i, fsContractAddr)
					if err != nil {
						log.Errorf("getContractEvents error %s", err)
						continue
					}
					for _, event := range events {
						this.processEvent(event)
					}
				}

				latestHeight = currHeight

			case <-this.kill:
				log.Debugf("stop event filter")
				return
			}
		}
	}()
	return nil
}

func (this *MaxService) getContractEvents(blockHeight uint32, contractAddress string) ([]map[string]interface{}, error) {
	log.Debugf("getContractEvents for height %d, contractAddress %s", blockHeight, contractAddress)
	var eventRe = make([]map[string]interface{}, 0)

	raws, err := this.chain.GetSmartContractEventsByBlock(blockHeight)
	if err != nil {
		log.Errorf("GetSmartContractEventsByBlock for height %d error %s", blockHeight, err)
		return nil, err
	}

	if len(raws) == 0 {
		log.Debugf("getContractEvents no event found")
		return nil, nil
	}
	for _, raw := range raws {
		if raw == nil {
			continue
		}
		for _, notify := range raw.Notify {
			if notify.ContractAddress != contractAddress {
				continue
			}
			if _, ok := notify.States.(map[string]interface{}); !ok {
				continue
			}
			eventRe = append(eventRe, notify.States.(map[string]interface{}))
		}
	}
	log.Debugf("getContractEvents event found: %v", eventRe)
	return eventRe, nil
}

func (this *MaxService) processEvent(event map[string]interface{}) {
	var eventName string
	if _, ok := event["eventName"].(string); ok == false {
		log.Debugf("eventName not found")
		return
	}

	eventName = event["eventName"].(string)

	parsedEvent, err := parseEvent(event)
	if err != nil {
		log.Errorf("parse event error : %s", err)
		return
	}

	switch eventName {
	case "deleteFile":
		this.processDeleteFileEvent(parsedEvent)
	case "deleteFiles":
		this.processDeleteFilesEvent(parsedEvent)
	case "createSector":
		this.processCreateSectorEvent(parsedEvent)
	default:
		return
	}
	return
}

func (this *MaxService) processCreateSectorEvent(event map[string]interface{}) {
	walletAddr := event["walletAddr"].(string)
	sectorId := event["sectorId"].(uint64)
	size := event["size"].(uint64)
	proveLevel := event["proveLevel"].(uint64)

	address := this.getAccoutAddress()
	if walletAddr == address.ToBase58() {
		this.notifySectorEvent(&sector.SectorEvent{
			Event:      sector.SECTOR_EVENT_CREATE,
			SectorID:   sectorId,
			ProveLevel: proveLevel,
			Size:       size,
		})
	}
}

func (this *MaxService) notifySectorEvent(event *sector.SectorEvent) {
	go func() {
		this.sectorManager.NotifySectorEvent(event)
	}()
}

func (this *MaxService) processDeleteFileEvent(event map[string]interface{}) {
	fileHash := event["fileHash"].(string)
	walletAddr := event["walletAddr"].(string)

	log.Debugf("process deleteFile event: fileHash %s, walletAddr %s", fileHash, walletAddr)

	this.processOneFileDelete(fileHash)
}

func (this *MaxService) processDeleteFilesEvent(event map[string]interface{}) {
	fileHashes := event["fileHashes"].([]string)
	walletAddr := event["walletAddr"].(string)

	log.Debugf("process deleteFiles event: fileHashes %v, walletAddr %s", fileHashes, walletAddr)

	for _, fileHash := range fileHashes {
		this.processOneFileDelete(fileHash)
	}
}

func (this *MaxService) processOneFileDelete(fileHash string) {
	_, ok := this.provetasks.Load(fileHash)
	if ok {
		err := this.deleteAndNotify(fileHash, PROVE_TASK_REMOVAL_REASON_DELETE)
		if err != nil {
			log.Errorf("processDeleteFile error %s", err)
			return
		}
	}
}
func parseEvent(event map[string]interface{}) (map[string]interface{}, error) {
	events := make(map[string]interface{})

	stopLoop := false
	for key, value := range event {
		switch key {
		case "fileHash":
			fileHash, ok := value.(string)
			if !ok {
				stopLoop = true
				break
			}
			events[key] = fileHash
		case "fileHashes":
			fileHashesSlice, ok := value.([]interface{})
			if !ok {
				stopLoop = true
				break
			}
			fileHashes, err := parseStringSliceFromInterfaceSlice(fileHashesSlice)
			if err != nil {
				return nil, fmt.Errorf("Parse fileHashes error %s", err.Error())
			}
			events[key] = fileHashes
		case "walletAddr":
			walletAddr, ok := value.(string)
			if !ok {
				stopLoop = true
				break
			}
			events[key] = walletAddr
		case "sectorId":
			sectorId, ok := value.(float64)
			if !ok {
				stopLoop = true
				break
			}
			events[key] = uint64(sectorId)
		case "size":
			size, ok := value.(float64)
			if !ok {
				stopLoop = true
				break
			}
			events[key] = uint64(size)
		case "proveLevel":
			proveLevel, ok := value.(float64)
			if !ok {
				stopLoop = true
				break
			}
			events[key] = uint64(proveLevel)
		}

		if stopLoop {
			return nil, fmt.Errorf("Parse key %s error, type %s",
				key, reflect.TypeOf(value).String())
		}
	}
	return events, nil
}
func parseStringSliceFromInterfaceSlice(slice []interface{}) ([]string, error) {
	result := []string{}
	for _, item := range slice {
		itemString, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("invalid type %s", reflect.TypeOf(item).String())
		}
		result = append(result, itemString)
	}
	return result, nil
}
