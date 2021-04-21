package routinghelpers

import (
	"context"
	"errors"
	"strings"
	"sync"

	pstore "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYLXCWN2myozZpx8Wx4UjrRuQuhY3YtWoMi6SHaXii6aM/go-libp2p-peerstore"
	cid "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
	peer "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZSzKEM5yDfpZbeEEZaVmaZ1zXm6JWTbrQZSB8hCVPzk/go-libp2p-peer"
	routing "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmewrvpGvgK9qkCtXsGNwXiQzyux4jcHNjoyVrGdsgtNK5/go-libp2p-routing"
	ropts "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmewrvpGvgK9qkCtXsGNwXiQzyux4jcHNjoyVrGdsgtNK5/go-libp2p-routing/options"
)

type dummyValueStore sync.Map

func (d *dummyValueStore) PutValue(ctx context.Context, key string, value []byte, opts ...ropts.Option) error {
	if strings.HasPrefix(key, "/notsupported/") {
		return routing.ErrNotSupported
	}
	if strings.HasPrefix(key, "/error/") {
		return errors.New(key[len("/error/"):])
	}
	if strings.HasPrefix(key, "/stall/") {
		<-ctx.Done()
		return ctx.Err()
	}
	(*sync.Map)(d).Store(key, value)
	return nil
}

func (d *dummyValueStore) GetValue(ctx context.Context, key string, opts ...ropts.Option) ([]byte, error) {
	if strings.HasPrefix(key, "/error/") {
		return nil, errors.New(key[len("/error/"):])
	}
	if strings.HasPrefix(key, "/stall/") {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if v, ok := (*sync.Map)(d).Load(key); ok {
		return v.([]byte), nil
	}
	return nil, routing.ErrNotFound
}

type dummyProvider map[string][]peer.ID

func (d dummyProvider) FindProvidersAsync(ctx context.Context, c *cid.Cid, count int) <-chan pstore.PeerInfo {
	peers := d[c.KeyString()]
	if len(peers) > count {
		peers = peers[:count]
	}
	out := make(chan pstore.PeerInfo)
	go func() {
		defer close(out)
		for _, p := range peers {
			if p == "stall" {
				<-ctx.Done()
				return
			}
			select {
			case out <- pstore.PeerInfo{ID: p}:
			case <-ctx.Done():
			}
		}
	}()
	return out
}

func (d dummyProvider) Provide(ctx context.Context, c *cid.Cid, local bool) error {
	return routing.ErrNotSupported
}

type cbProvider func(c *cid.Cid, local bool) error

func (d cbProvider) Provide(ctx context.Context, c *cid.Cid, local bool) error {
	return d(c, local)
}

func (d cbProvider) FindProvidersAsync(ctx context.Context, c *cid.Cid, count int) <-chan pstore.PeerInfo {
	ch := make(chan pstore.PeerInfo)
	close(ch)
	return ch
}

type dummyPeerRouter map[peer.ID]struct{}

func (d dummyPeerRouter) FindPeer(ctx context.Context, p peer.ID) (pstore.PeerInfo, error) {
	if _, ok := d[p]; ok {
		return pstore.PeerInfo{ID: p}, nil
	}
	return pstore.PeerInfo{}, routing.ErrNotFound
}
