package mockrouting

import (
	"context"
	"time"

	logging "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRREK2CAZ5Re2Bd9zZFG6FeYDppUWt5cMgsoUEp3ktgSr/go-log"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmXG74iiKQnDstVQq9fPFQEB6JTNSWBbAWE1qsq6L4E5sR/go-testutil"
	pstore "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYLXCWN2myozZpx8Wx4UjrRuQuhY3YtWoMi6SHaXii6aM/go-libp2p-peerstore"
	cid "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
	ma "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYmsdtJ3HsodkePE3eU3TsCaP2YvPZJ4LoXnNkDE5Tpt7/go-multiaddr"
	peer "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZSzKEM5yDfpZbeEEZaVmaZ1zXm6JWTbrQZSB8hCVPzk/go-libp2p-peer"
	routing "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmewrvpGvgK9qkCtXsGNwXiQzyux4jcHNjoyVrGdsgtNK5/go-libp2p-routing"
	ropts "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmewrvpGvgK9qkCtXsGNwXiQzyux4jcHNjoyVrGdsgtNK5/go-libp2p-routing/options"
)

var log = logging.Logger("mockrouter")

type client struct {
	vs     routing.ValueStore
	server server
	peer   testutil.Identity
}

// FIXME(brian): is this method meant to simulate putting a value into the network?
func (c *client) PutValue(ctx context.Context, key string, val []byte, opts ...ropts.Option) error {
	log.Debugf("PutValue: %s", key)
	return c.vs.PutValue(ctx, key, val, opts...)
}

// FIXME(brian): is this method meant to simulate getting a value from the network?
func (c *client) GetValue(ctx context.Context, key string, opts ...ropts.Option) ([]byte, error) {
	log.Debugf("GetValue: %s", key)
	return c.vs.GetValue(ctx, key, opts...)
}

func (c *client) FindProviders(ctx context.Context, key *cid.Cid) ([]pstore.PeerInfo, error) {
	return c.server.Providers(key), nil
}

func (c *client) FindPeer(ctx context.Context, pid peer.ID) (pstore.PeerInfo, error) {
	log.Debugf("FindPeer: %s", pid)
	return pstore.PeerInfo{}, nil
}

func (c *client) FindProvidersAsync(ctx context.Context, k *cid.Cid, max int) <-chan pstore.PeerInfo {
	out := make(chan pstore.PeerInfo)
	go func() {
		defer close(out)
		for i, p := range c.server.Providers(k) {
			if max <= i {
				return
			}
			select {
			case out <- p:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// Provide returns once the message is on the network. Value is not necessarily
// visible yet.
func (c *client) Provide(_ context.Context, key *cid.Cid, brd bool) error {
	if !brd {
		return nil
	}
	info := pstore.PeerInfo{
		ID:    c.peer.ID(),
		Addrs: []ma.Multiaddr{c.peer.Address()},
	}
	return c.server.Announce(info, key)
}

func (c *client) Ping(ctx context.Context, p peer.ID) (time.Duration, error) {
	return 0, nil
}

func (c *client) Bootstrap(context.Context) error {
	return nil
}

var _ routing.IpfsRouting = &client{}
