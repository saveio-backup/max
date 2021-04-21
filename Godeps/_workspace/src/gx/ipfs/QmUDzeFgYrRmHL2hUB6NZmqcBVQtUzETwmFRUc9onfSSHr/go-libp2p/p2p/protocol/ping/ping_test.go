package ping

import (
	"context"
	"testing"
	"time"

	bhost "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmUDzeFgYrRmHL2hUB6NZmqcBVQtUzETwmFRUc9onfSSHr/go-libp2p/p2p/host/basic"
	pstore "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYLXCWN2myozZpx8Wx4UjrRuQuhY3YtWoMi6SHaXii6aM/go-libp2p-peerstore"
	peer "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZSzKEM5yDfpZbeEEZaVmaZ1zXm6JWTbrQZSB8hCVPzk/go-libp2p-peer"
	swarmt "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmdjC8HtKZpEufBL1u7WxvQn78Lqq2Wk31NJS8WvFX3crB/go-libp2p-swarm/testing"
)

func TestPing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h1 := bhost.New(swarmt.GenSwarm(t, ctx))
	h2 := bhost.New(swarmt.GenSwarm(t, ctx))

	err := h1.Connect(ctx, pstore.PeerInfo{
		ID:    h2.ID(),
		Addrs: h2.Addrs(),
	})

	if err != nil {
		t.Fatal(err)
	}

	ps1 := NewPingService(h1)
	ps2 := NewPingService(h2)

	testPing(t, ps1, h2.ID())
	testPing(t, ps2, h1.ID())
}

func testPing(t *testing.T, ps *PingService, p peer.ID) {
	pctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ts, err := ps.Ping(pctx, p)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		select {
		case took := <-ts:
			t.Log("ping took: ", took)
		case <-time.After(time.Second * 4):
			t.Fatal("failed to receive ping")
		}
	}

}
