package bitswap

import (
	"context"
	"errors"
	"sync"
	"time"

	bsmsg "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNQQEYL3Vpj4beteqyeRpVpivuX1wBP6Q5GZMdBPPTV3S/go-bitswap/message"
	bsnet "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNQQEYL3Vpj4beteqyeRpVpivuX1wBP6Q5GZMdBPPTV3S/go-bitswap/network"

	delay "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRJVNatYJwTAHgdSM1Xef9QVQ1Ch3XHdmcrykjP5Y4soL/go-ipfs-delay"
	logging "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRREK2CAZ5Re2Bd9zZFG6FeYDppUWt5cMgsoUEp3ktgSr/go-log"
	testutil "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmXG74iiKQnDstVQq9fPFQEB6JTNSWBbAWE1qsq6L4E5sR/go-testutil"
	cid "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
	mockrouting "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qma19TdQ7W26jbfuPgdo9Zi4qtjks1zeXzX86mtEYWYCiw/go-ipfs-routing/mock"
	peer "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZSzKEM5yDfpZbeEEZaVmaZ1zXm6JWTbrQZSB8hCVPzk/go-libp2p-peer"
	ifconnmgr "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmeJbAMK4cZc1RMChb68h9t2jqvK8miqE8oQiwGAf4EdQq/go-libp2p-interface-connmgr"
	routing "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmewrvpGvgK9qkCtXsGNwXiQzyux4jcHNjoyVrGdsgtNK5/go-libp2p-routing"
)

var log = logging.Logger("bstestnet")

func VirtualNetwork(rs mockrouting.Server, d delay.D) Network {
	return &network{
		clients:       make(map[peer.ID]*receiverQueue),
		delay:         d,
		routingserver: rs,
		conns:         make(map[string]struct{}),
	}
}

type network struct {
	mu            sync.Mutex
	clients       map[peer.ID]*receiverQueue
	routingserver mockrouting.Server
	delay         delay.D
	conns         map[string]struct{}
}

type message struct {
	from       peer.ID
	msg        bsmsg.BitSwapMessage
	shouldSend time.Time
}

// receiverQueue queues up a set of messages to be sent, and sends them *in
// order* with their delays respected as much as sending them in order allows
// for
type receiverQueue struct {
	receiver bsnet.Receiver
	queue    []*message
	active   bool
	lk       sync.Mutex
}

func (n *network) Adapter(p testutil.Identity) bsnet.BitSwapNetwork {
	n.mu.Lock()
	defer n.mu.Unlock()

	client := &networkClient{
		local:   p.ID(),
		network: n,
		routing: n.routingserver.Client(p),
	}
	n.clients[p.ID()] = &receiverQueue{receiver: client}
	return client
}

func (n *network) HasPeer(p peer.ID) bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	_, found := n.clients[p]
	return found
}

// TODO should this be completely asynchronous?
// TODO what does the network layer do with errors received from services?
func (n *network) SendMessage(
	ctx context.Context,
	from peer.ID,
	to peer.ID,
	mes bsmsg.BitSwapMessage) error {

	n.mu.Lock()
	defer n.mu.Unlock()

	receiver, ok := n.clients[to]
	if !ok {
		return errors.New("cannot locate peer on network")
	}

	// nb: terminate the context since the context wouldn't actually be passed
	// over the network in a real scenario

	msg := &message{
		from:       from,
		msg:        mes,
		shouldSend: time.Now().Add(n.delay.Get()),
	}
	receiver.enqueue(msg)

	return nil
}

func (n *network) deliver(
	r bsnet.Receiver, from peer.ID, message bsmsg.BitSwapMessage) error {
	if message == nil || from == "" {
		return errors.New("invalid input")
	}

	n.delay.Wait()

	r.ReceiveMessage(context.TODO(), from, message)
	return nil
}

type networkClient struct {
	local peer.ID
	bsnet.Receiver
	network *network
	routing routing.IpfsRouting
}

func (nc *networkClient) SendMessage(
	ctx context.Context,
	to peer.ID,
	message bsmsg.BitSwapMessage) error {
	return nc.network.SendMessage(ctx, nc.local, to, message)
}

// FindProvidersAsync returns a channel of providers for the given key
func (nc *networkClient) FindProvidersAsync(ctx context.Context, k *cid.Cid, max int) <-chan peer.ID {

	// NB: this function duplicates the PeerInfo -> ID transformation in the
	// bitswap network adapter. Not to worry. This network client will be
	// deprecated once the ipfsnet.Mock is added. The code below is only
	// temporary.

	out := make(chan peer.ID)
	go func() {
		defer close(out)
		providers := nc.routing.FindProvidersAsync(ctx, k, max)
		for info := range providers {
			select {
			case <-ctx.Done():
			case out <- info.ID:
			}
		}
	}()
	return out
}

func (nc *networkClient) ConnectionManager() ifconnmgr.ConnManager {
	return &ifconnmgr.NullConnMgr{}
}

type messagePasser struct {
	net    *network
	target peer.ID
	local  peer.ID
	ctx    context.Context
}

func (mp *messagePasser) SendMsg(ctx context.Context, m bsmsg.BitSwapMessage) error {
	return mp.net.SendMessage(ctx, mp.local, mp.target, m)
}

func (mp *messagePasser) Close() error {
	return nil
}

func (mp *messagePasser) Reset() error {
	return nil
}

func (n *networkClient) NewMessageSender(ctx context.Context, p peer.ID) (bsnet.MessageSender, error) {
	return &messagePasser{
		net:    n.network,
		target: p,
		local:  n.local,
		ctx:    ctx,
	}, nil
}

// Provide provides the key to the network
func (nc *networkClient) Provide(ctx context.Context, k *cid.Cid) error {
	return nc.routing.Provide(ctx, k, true)
}

func (nc *networkClient) SetDelegate(r bsnet.Receiver) {
	nc.Receiver = r
}

func (nc *networkClient) ConnectTo(_ context.Context, p peer.ID) error {
	nc.network.mu.Lock()

	otherClient, ok := nc.network.clients[p]
	if !ok {
		nc.network.mu.Unlock()
		return errors.New("no such peer in network")
	}

	tag := tagForPeers(nc.local, p)
	if _, ok := nc.network.conns[tag]; ok {
		nc.network.mu.Unlock()
		log.Warning("ALREADY CONNECTED TO PEER (is this a reconnect? test lib needs fixing)")
		return nil
	}
	nc.network.conns[tag] = struct{}{}
	nc.network.mu.Unlock()

	// TODO: add handling for disconnects

	otherClient.receiver.PeerConnected(nc.local)
	nc.Receiver.PeerConnected(p)
	return nil
}

func (rq *receiverQueue) enqueue(m *message) {
	rq.lk.Lock()
	defer rq.lk.Unlock()
	rq.queue = append(rq.queue, m)
	if !rq.active {
		rq.active = true
		go rq.process()
	}
}

func (rq *receiverQueue) process() {
	for {
		rq.lk.Lock()
		if len(rq.queue) == 0 {
			rq.active = false
			rq.lk.Unlock()
			return
		}
		m := rq.queue[0]
		rq.queue = rq.queue[1:]
		rq.lk.Unlock()

		time.Sleep(time.Until(m.shouldSend))
		rq.receiver.ReceiveMessage(context.TODO(), m.from, m.msg)
	}
}

func tagForPeers(a, b peer.ID) string {
	if a < b {
		return string(a + b)
	}
	return string(b + a)
}
