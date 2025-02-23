package floodsub

import (
	"context"

	pb "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRXefkwjreRT6XfYh3Ag4hsVnWBbpcUicGJPcg8TWbhBK/go-libp2p-floodsub/pb"

	host "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmQ1hwb95uSSZR8jSPJysnfHxBDQAykSXsmz5TwTzxjq2Z/go-libp2p-host"
	protocol "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	peer "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZSzKEM5yDfpZbeEEZaVmaZ1zXm6JWTbrQZSB8hCVPzk/go-libp2p-peer"
)

const (
	FloodSubID = protocol.ID("/floodsub/1.0.0")
)

// NewFloodsubWithProtocols returns a new floodsub-enabled PubSub objecting using the protocols specified in ps
func NewFloodsubWithProtocols(ctx context.Context, h host.Host, ps []protocol.ID, opts ...Option) (*PubSub, error) {
	rt := &FloodSubRouter{
		protocols: ps,
	}
	return NewPubSub(ctx, h, rt, opts...)
}

// NewFloodSub returns a new PubSub object using the FloodSubRouter
func NewFloodSub(ctx context.Context, h host.Host, opts ...Option) (*PubSub, error) {
	return NewFloodsubWithProtocols(ctx, h, []protocol.ID{FloodSubID}, opts...)
}

type FloodSubRouter struct {
	p         *PubSub
	protocols []protocol.ID
}

func (fs *FloodSubRouter) Protocols() []protocol.ID {
	return fs.protocols
}

func (fs *FloodSubRouter) Attach(p *PubSub) {
	fs.p = p
}

func (fs *FloodSubRouter) AddPeer(peer.ID, protocol.ID) {}

func (fs *FloodSubRouter) RemovePeer(peer.ID) {}

func (fs *FloodSubRouter) HandleRPC(rpc *RPC) {}

func (fs *FloodSubRouter) Publish(from peer.ID, msg *pb.Message) {
	tosend := make(map[peer.ID]struct{})
	for _, topic := range msg.GetTopicIDs() {
		tmap, ok := fs.p.topics[topic]
		if !ok {
			continue
		}

		for p := range tmap {
			tosend[p] = struct{}{}
		}
	}

	out := rpcWithMessages(msg)
	for pid := range tosend {
		if pid == from || pid == peer.ID(msg.GetFrom()) {
			continue
		}

		mch, ok := fs.p.peers[pid]
		if !ok {
			continue
		}

		select {
		case mch <- out:
		default:
			log.Infof("dropping message to peer %s: queue full", pid)
			// Drop it. The peer is too slow.
		}
	}
}

func (fs *FloodSubRouter) Join(topic string) {}

func (fs *FloodSubRouter) Leave(topic string) {}
