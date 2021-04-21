package stream

import (
	"fmt"

	transport "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmUMTtHxeyVJPrpcpvEQppH3uTf3g1NnkRC6C36LpXy2no/go-libp2p-transport"
	inet "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVwU7Mgwg6qaPn9XXz93ANfq1PTxcduGRzfe41Sygg4mR/go-libp2p-net"
	smux "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmY9JXR3FupnYAYJWK9aMr9bCpqWKcToQ1tz8DVGTrHpHw/go-stream-muxer"
)

type transportConn struct {
	smux.Conn
	inet.ConnMultiaddrs
	inet.ConnSecurity
	transport transport.Transport
}

func (t *transportConn) Transport() transport.Transport {
	return t.transport
}

func (t *transportConn) String() string {
	ts := ""
	if s, ok := t.transport.(fmt.Stringer); ok {
		ts = "[" + s.String() + "]"
	}
	return fmt.Sprintf(
		"<stream.Conn%s %s (%s) <-> %s (%s)>",
		ts,
		t.LocalMultiaddr(),
		t.LocalPeer(),
		t.RemoteMultiaddr(),
		t.RemotePeer(),
	)
}
