package websocket

import (
	"net/url"
	"testing"

	ma "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
)

func TestMultiaddrParsing(t *testing.T) {
	addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/5555/ws")
	if err != nil {
		t.Fatal(err)
	}

	wsaddr, err := parseMultiaddr(addr)
	if err != nil {
		t.Fatal(err)
	}
	if wsaddr != "ws://127.0.0.1:5555" {
		t.Fatalf("expected ws://127.0.0.1:5555, got %s", wsaddr)
	}
}

type httpAddr struct {
	*url.URL
}

func (addr *httpAddr) Network() string {
	return "http"
}

func TestParseWebsocketNetAddr(t *testing.T) {
	notWs := &httpAddr{&url.URL{Host: "http://127.0.0.1:1234"}}
	_, err := ParseWebsocketNetAddr(notWs)
	if err.Error() != "not a websocket address" {
		t.Fatalf("expect \"not a websocket address\", got \"%s\"", err)
	}

	wsAddr := NewAddr("127.0.0.1:5555")
	parsed, err := ParseWebsocketNetAddr(wsAddr)
	if err != nil {
		t.Fatal(err)
	}

	if parsed.String() != "/ip4/127.0.0.1/tcp/5555/ws" {
		t.Fatalf("expected \"/ip4/127.0.0.1/tcp/5555/ws\", got \"%s\"", parsed.String())
	}
}

func TestConvertWebsocketMultiaddrToNetAddr(t *testing.T) {
	addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/5555/ws")
	if err != nil {
		t.Fatal(err)
	}

	wsaddr, err := ConvertWebsocketMultiaddrToNetAddr(addr)
	if err != nil {
		t.Fatal(err)
	}
	if wsaddr.String() != "//127.0.0.1:5555" {
		t.Fatalf("expected //127.0.0.1:5555, got %s", wsaddr)
	}
	if wsaddr.Network() != "websocket" {
		t.Fatalf("expected network: \"websocket\", got \"%s\"", wsaddr.Network())
	}
}
