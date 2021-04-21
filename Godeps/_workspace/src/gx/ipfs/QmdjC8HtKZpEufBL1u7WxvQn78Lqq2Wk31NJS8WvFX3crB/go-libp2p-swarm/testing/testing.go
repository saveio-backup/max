package testing

import (
	"context"
	"testing"

	tptu "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmP7znopdZogwxPJyRKEZSNnP7HfnUCaQjaMNDmPw8VE2Y/go-libp2p-transport-upgrader"
	tcp "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmSYxVNTGJ4MEy7UH26qgLbidfSgdYiEkV6affSitghk4S/go-tcp-transport"
	inet "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVwU7Mgwg6qaPn9XXz93ANfq1PTxcduGRzfe41Sygg4mR/go-libp2p-net"
	metrics "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmWne2EKHBvVpSTYuWuWch3D9KqAx78Te83UXWFKQDcksJ/go-libp2p-metrics"
	msmux "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmWzjXAyBTygw6CeCTUnhJzhFucfxY5FJivSoiGuiSbPjS/go-smux-multistream"
	tu "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmXG74iiKQnDstVQq9fPFQEB6JTNSWBbAWE1qsq6L4E5sR/go-testutil"
	pstore "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYLXCWN2myozZpx8Wx4UjrRuQuhY3YtWoMi6SHaXii6aM/go-libp2p-peerstore"
	secio "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmZ69yKFZ7V5hLCYFWzJF1Nz4dH1Es4LaE8jGXAFucLe3K/go-libp2p-secio"
	csms "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qmahr4UDXGj8zXFhXzQZpmoJgeVi6XX8pEnuVi2bFkENqj/go-conn-security-multistream"
	yamux "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcsgrV3nCAKjiHKZhKVXWc4oY3WBECJCqahXEMpHeMrev/go-smux-yamux"

	swarm "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmdjC8HtKZpEufBL1u7WxvQn78Lqq2Wk31NJS8WvFX3crB/go-libp2p-swarm"
)

type config struct {
	disableReuseport bool
	dialOnly         bool
}

// Option is an option that can be passed when constructing a test swarm.
type Option func(*testing.T, *config)

// OptDisableReuseport disables reuseport in this test swarm.
var OptDisableReuseport Option = func(_ *testing.T, c *config) {
	c.disableReuseport = true
}

// OptDialOnly prevents the test swarm from listening.
var OptDialOnly Option = func(_ *testing.T, c *config) {
	c.dialOnly = true
}

// GenUpgrader creates a new connection upgrader for use with this swarm.
func GenUpgrader(n *swarm.Swarm) *tptu.Upgrader {
	id := n.LocalPeer()
	pk := n.Peerstore().PrivKey(id)
	secMuxer := new(csms.SSMuxer)
	secMuxer.AddTransport(secio.ID, &secio.Transport{
		LocalID:    id,
		PrivateKey: pk,
	})

	stMuxer := msmux.NewBlankTransport()
	stMuxer.AddTransport("/yamux/1.0.0", yamux.DefaultTransport)

	return &tptu.Upgrader{
		Secure:  secMuxer,
		Muxer:   stMuxer,
		Filters: n.Filters,
	}

}

// GenSwarm generates a new test swarm.
func GenSwarm(t *testing.T, ctx context.Context, opts ...Option) *swarm.Swarm {
	var cfg config
	for _, o := range opts {
		o(t, &cfg)
	}

	p := tu.RandPeerNetParamsOrFatal(t)

	ps := pstore.NewPeerstore()
	ps.AddPubKey(p.ID, p.PubKey)
	ps.AddPrivKey(p.ID, p.PrivKey)
	s := swarm.NewSwarm(ctx, p.ID, ps, metrics.NewBandwidthCounter())

	tcpTransport := tcp.NewTCPTransport(GenUpgrader(s))
	tcpTransport.DisableReuseport = cfg.disableReuseport

	if err := s.AddTransport(tcpTransport); err != nil {
		t.Fatal(err)
	}

	if !cfg.dialOnly {
		if err := s.Listen(p.Addr); err != nil {
			t.Fatal(err)
		}

		s.Peerstore().AddAddrs(p.ID, s.ListenAddresses(), pstore.PermanentAddrTTL)
	}

	return s
}

// DivulgeAddresses adds swarm a's addresses to swarm b's peerstore.
func DivulgeAddresses(a, b inet.Network) {
	id := a.LocalPeer()
	addrs := a.Peerstore().Addrs(id)
	b.Peerstore().AddAddrs(id, addrs, pstore.PermanentAddrTTL)
}
