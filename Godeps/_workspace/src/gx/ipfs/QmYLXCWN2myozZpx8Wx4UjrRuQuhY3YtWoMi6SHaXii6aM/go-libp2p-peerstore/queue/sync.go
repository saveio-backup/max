package queue

import (
	"context"

	logging "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRREK2CAZ5Re2Bd9zZFG6FeYDppUWt5cMgsoUEp3ktgSr/go-log"
	peer "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZSzKEM5yDfpZbeEEZaVmaZ1zXm6JWTbrQZSB8hCVPzk/go-libp2p-peer"
)

var log = logging.Logger("peerqueue")

// ChanQueue makes any PeerQueue synchronizable through channels.
type ChanQueue struct {
	Queue   PeerQueue
	EnqChan chan<- peer.ID
	DeqChan <-chan peer.ID
}

// NewChanQueue creates a ChanQueue by wrapping pq.
func NewChanQueue(ctx context.Context, pq PeerQueue) *ChanQueue {
	cq := &ChanQueue{Queue: pq}
	cq.process(ctx)
	return cq
}

func (cq *ChanQueue) process(ctx context.Context) {
	// construct the channels here to be able to use them bidirectionally
	enqChan := make(chan peer.ID)
	deqChan := make(chan peer.ID)

	cq.EnqChan = enqChan
	cq.DeqChan = deqChan

	go func() {
		log.Debug("processing")
		defer log.Debug("closed")
		defer close(deqChan)

		var next peer.ID
		var item peer.ID
		var more bool

		for {
			if cq.Queue.Len() == 0 {
				// log.Debug("wait for enqueue")
				select {
				case next, more = <-enqChan:
					if !more {
						return
					}
					// log.Debug("got", next)

				case <-ctx.Done():
					return
				}

			} else {
				next = cq.Queue.Dequeue()
				// log.Debug("peek", next)
			}

			select {
			case item, more = <-enqChan:
				if !more {
					if cq.Queue.Len() > 0 {
						return // we're done done.
					}
					enqChan = nil // closed, so no use.
				}
				// log.Debug("got", item)
				cq.Queue.Enqueue(item)
				cq.Queue.Enqueue(next) // order may have changed.
				next = ""

			case deqChan <- next:
				// log.Debug("dequeued", next)
				next = ""

			case <-ctx.Done():
				return
			}
		}

	}()
}
