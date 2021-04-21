package swarm_test

import (
	"testing"

	"context"
	inet "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVwU7Mgwg6qaPn9XXz93ANfq1PTxcduGRzfe41Sygg4mR/go-libp2p-net"
	pstore "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYLXCWN2myozZpx8Wx4UjrRuQuhY3YtWoMi6SHaXii6aM/go-libp2p-peerstore"
	ma "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYmsdtJ3HsodkePE3eU3TsCaP2YvPZJ4LoXnNkDE5Tpt7/go-multiaddr"
	peer "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZSzKEM5yDfpZbeEEZaVmaZ1zXm6JWTbrQZSB8hCVPzk/go-libp2p-peer"

	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmdjC8HtKZpEufBL1u7WxvQn78Lqq2Wk31NJS8WvFX3crB/go-libp2p-swarm"
)

func TestPeers(t *testing.T) {

	ctx := context.Background()
	swarms := makeSwarms(ctx, t, 2)
	s1 := swarms[0]
	s2 := swarms[1]

	connect := func(s *Swarm, dst peer.ID, addr ma.Multiaddr) {
		// TODO: make a DialAddr func.
		s.Peerstore().AddAddr(dst, addr, pstore.PermanentAddrTTL)
		// t.Logf("connections from %s", s.LocalPeer())
		// for _, c := range s.ConnsToPeer(dst) {
		// 	t.Logf("connection from %s to %s: %v", s.LocalPeer(), dst, c)
		// }
		// t.Logf("")
		if _, err := s.DialPeer(ctx, dst); err != nil {
			t.Fatal("error swarm dialing to peer", err)
		}
		// t.Log(s.swarm.Dump())
	}

	s1GotConn := make(chan struct{}, 0)
	s2GotConn := make(chan struct{}, 0)
	s1.SetConnHandler(func(c inet.Conn) {
		s1GotConn <- struct{}{}
	})
	s2.SetConnHandler(func(c inet.Conn) {
		s2GotConn <- struct{}{}
	})

	connect(s1, s2.LocalPeer(), s2.ListenAddresses()[0])
	<-s2GotConn // have to wait here so the other side catches up.
	connect(s2, s1.LocalPeer(), s1.ListenAddresses()[0])

	for i := 0; i < 100; i++ {
		connect(s1, s2.LocalPeer(), s2.ListenAddresses()[0])
		connect(s2, s1.LocalPeer(), s1.ListenAddresses()[0])
	}

	for _, s := range swarms {
		log.Infof("%s swarm routing table: %s", s.LocalPeer(), s.Peers())
	}

	test := func(s *Swarm) {
		expect := 1
		actual := len(s.Peers())
		if actual != expect {
			t.Errorf("%s has %d peers, not %d: %v", s.LocalPeer(), actual, expect, s.Peers())
		}
		actual = len(s.Conns())
		if actual != expect {
			t.Errorf("%s has %d conns, not %d: %v", s.LocalPeer(), actual, expect, s.Conns())
		}
	}

	test(s1)
	test(s2)
}
