package max

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"time"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/saveio/dsp-go-sdk/consts"
	"github.com/saveio/themis-go-sdk/common"
	"github.com/saveio/themis-go-sdk/fs"

	ldb "github.com/saveio/max/max/leveldbstore"
	"github.com/saveio/max/max/sector"
	"github.com/saveio/themis/common/log"
	"github.com/saveio/themis/smartcontract/service/native/utils"
)

const LATEST_HEIGHT_KEY = "latestheight:"

func (this *MaxService) StartEventFilter(interval uint32) error {
	if this.chain == nil {
		return nil
	}

	// make sure this run before load pdp task to get block height and hash
	height, _, err := this.getCurrentBlockHeightAndHashFromChainAndUpdateCache()
	if err != nil {
		return err
	}

	go func() {
		var latestHeight uint32
		var currHeight uint32
		var err error

		fsContractAddr := utils.OntFSContractAddress.ToHexString()

		if this.chain.GetChainType() == consts.DspModeOp {
			fsContractAddr = fs.FileAddress.String()
		}

		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		log.Debugf("start event filter, fsContractAddr %s", fsContractAddr)

		latestHeight, err = this.loadLatestHeight()
		if err != nil {
			log.Errorf("loadLatestHeight error %s", err)
		}

		if latestHeight == 0 {
			latestHeight = height
			log.Debugf("latestHeight not found, set latest height to %d", height)
		}

		log.Debugf("eventfilter latestHeight is %d", latestHeight)

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
						this.processEvent(i, event)
					}
				}

				latestHeight = currHeight
				err = this.saveLatestHeight(latestHeight)
				if err != nil {
					log.Errorf("SaveLatestHeight error %s", err)
				}

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

	var raws []*common.SmartContactEvent
	var err error
	switch this.chain.GetChainType() {
	case consts.DspModeOp:
		events, err := this.chain.GetSDK().EVM.Fs.GetEventsByBlockHeight(big.NewInt(int64(blockHeight)))
		if err != nil {
			log.Errorf("GetSmartContractEventsByBlock for height %d error %s", blockHeight, err)
			return nil, err
		}
		for _, event := range events {
			raws = append(raws, &common.SmartContactEvent{
				Notify: []*common.NotifyEventInfo{
					{
						ContractAddress: contractAddress,
						States:          event,
					},
				},
			})
		}
	default:
		raws, err = this.chain.GetSDK().GetSmartContractEventsByBlock(blockHeight)
		if err != nil {
			log.Errorf("GetSmartContractEventsByBlock for height %d error %s", blockHeight, err)
			return nil, err
		}
	}

	if len(raws) == 0 {
		// log.Debugf("getContractEvents no event found")
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

func (this *MaxService) processEvent(height uint32, event map[string]interface{}) {
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

	parsedEvent["eventName"] = eventName
	parsedEvent["blockHeight"] = height

	fmt.Println("===234===", eventName, parsedEvent)
	fmt.Println("===234===2", eventName, event)

	this.notifyChainEvent(parsedEvent)

	switch eventName {
	case "deleteFile":
		this.processDeleteFileEvent(parsedEvent)
	case "deleteFiles":
		this.processDeleteFilesEvent(parsedEvent)
	case "createSector":
		this.processCreateSectorEvent(parsedEvent)
	case "deleteSector":
		this.processDeleteSectorEvent(parsedEvent)
	default:
		return
	}
	return
}

func (this *MaxService) RegChainEventNotificationChannel(moduleId string) chan map[string]interface{} {
	ch := make(chan map[string]interface{}, 10)
	this.chainEventNotifyChannels.Store(moduleId, ch)
	log.Debugf("register chain event notification for module %s", moduleId)
	return ch
}

func (this *MaxService) notifyChainEvent(event map[string]interface{}) {
	this.chainEventNotifyChannels.Range(func(key, value interface{}) bool {
		log.Debugf("notify chain event for module %s, event %v", event["eventName"], event)
		go func() {
			ch, ok := value.(chan map[string]interface{})
			if !ok {
				log.Errorf("event channel type error")
				return
			}
			ch <- event
		}()
		return true
	})
}

func (this *MaxService) processCreateSectorEvent(event map[string]interface{}) {

	walletAddr := event["walletAddr"].(string)
	sectorId := event["sectorId"].(uint64)
	size := event["size"].(uint64)
	proveLevel := event["proveLevel"].(uint64)
	isPlots := event["isPlots"].(bool)

	address := this.getAccoutAddress()
	ethAddr := ethCommon.BytesToAddress(address[:])
	if walletAddr == address.ToBase58() || walletAddr == ethAddr.String() {
		this.notifySectorEvent(&sector.SectorEvent{
			Event:      sector.SECTOR_EVENT_CREATE,
			SectorID:   sectorId,
			ProveLevel: proveLevel,
			Size:       size,
			IsPlots:    isPlots,
		})
	}
}

func (this *MaxService) processDeleteSectorEvent(event map[string]interface{}) {
	walletAddr := event["walletAddr"].(string)
	sectorId := event["sectorId"].(uint64)

	address := this.getAccoutAddress()
	if walletAddr == address.ToBase58() {
		this.notifySectorEvent(&sector.SectorEvent{
			Event:    sector.SECTOR_EVENT_DELETE,
			SectorID: sectorId,
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

type latestHeight struct {
	Height uint32 `json:"height"`
}

func (this *MaxService) loadLatestHeight() (uint32, error) {
	data, err := this.fsstore.GetData(generateLatestHeightKey())
	if err != nil {
		if err == ldb.ErrNotFound {
			return 0, nil
		}
		log.Errorf("loadLatestHeight error %s", err)
		return 0, err
	}

	height := new(latestHeight)
	err = json.Unmarshal(data, height)
	if err != nil {
		return 0, err
	}
	log.Debugf("loadLatestHeight: %d", height.Height)
	return height.Height, nil
}

func (this *MaxService) saveLatestHeight(height uint32) error {
	latest := &latestHeight{Height: height}

	data, err := json.Marshal(latest)
	if err != nil {
		return err
	}

	return this.fsstore.PutData(generateLatestHeightKey(), data)
}

func generateLatestHeightKey() string {
	return LATEST_HEIGHT_KEY
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
				address, ok := value.(ethCommon.Address)
				if !ok {
					stopLoop = true
					break
				}
				walletAddr = address.String()
			}
			events[key] = walletAddr
		case "sectorId":
			sectorId, ok := value.(float64)
			if !ok {
				tmp, ok := value.(uint64)
				if !ok {
					stopLoop = true
					break
				} else {
					sectorId = float64(tmp)
				}
			}
			events[key] = uint64(sectorId)
		case "size":
			size, ok := value.(float64)
			if !ok {
				tmp, ok := value.(uint64)
				if !ok {
					stopLoop = true
					break
				} else {
					size = float64(tmp)
				}
			}
			events[key] = uint64(size)
		case "proveLevel":
			proveLevel, ok := value.(float64)
			if !ok {
				tmp, ok := value.(uint8)
				if !ok {
					stopLoop = true
					break
				} else {
					proveLevel = float64(tmp)
				}
			}
			events[key] = uint64(proveLevel)
		case "isPlots":
			isPlots, ok := value.(bool)
			if !ok {
				stopLoop = true
				break
			}
			events[key] = isPlots
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
