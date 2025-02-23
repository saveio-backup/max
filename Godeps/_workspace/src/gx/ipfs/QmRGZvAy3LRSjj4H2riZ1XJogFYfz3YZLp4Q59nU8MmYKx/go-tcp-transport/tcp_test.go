package tcp

import (
	"testing"

	tpt "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVxtCwKFMmwcjhQXsGj6m4JAW7nGb9hRoErH9jpgqcLxA/go-libp2p-transport"
	utils "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVxtCwKFMmwcjhQXsGj6m4JAW7nGb9hRoErH9jpgqcLxA/go-libp2p-transport/test"
	ma "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
)

func TestTcpTransport(t *testing.T) {
	ta := NewTCPTransport()
	tb := NewTCPTransport()

	zero := "/ip4/127.0.0.1/tcp/0"
	utils.SubtestTransport(t, ta, tb, zero)
}

func TestTcpTransportCantListenUtp(t *testing.T) {
	utpa, err := ma.NewMultiaddr("/ip4/127.0.0.1/udp/0/utp")
	if err != nil {
		t.Fatal(err)
	}

	tpt := NewTCPTransport()
	_, err = tpt.Listen(utpa)
	if err == nil {
		t.Fatal("shouldnt be able to listen on utp addr with tcp transport")
	}
}

func TestCorrectIPVersionMatching(t *testing.T) {
	ta := NewTCPTransport()

	addr4, err := ma.NewMultiaddr("/ip4/0.0.0.0/tcp/0")
	if err != nil {
		t.Fatal(err)
	}
	addr6, err := ma.NewMultiaddr("/ip6/::1/tcp/0")
	if err != nil {
		t.Fatal(err)
	}

	d4, err := ta.Dialer(addr4, tpt.ReuseportOpt(true))
	if err != nil {
		t.Fatal(err)
	}

	d6, err := ta.Dialer(addr6, tpt.ReuseportOpt(true))
	if err != nil {
		t.Fatal(err)
	}

	if d4.Matches(addr6) {
		t.Fatal("tcp4 dialer should not match ipv6 address")
	}

	if d6.Matches(addr4) {
		t.Fatal("tcp4 dialer should not match ipv6 address")
	}
}
