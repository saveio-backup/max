package stream

import (
	"context"
	"errors"
	"fmt"
	"net"

	filter "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmSW4uNHbvQia8iZDXzbwjiyHQtnyo9aFqfQAMasj3TJ6Y/go-maddr-filter"
	transport "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmUMTtHxeyVJPrpcpvEQppH3uTf3g1NnkRC6C36LpXy2no/go-libp2p-transport"
	manet "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmV6FjemM1K8oXjrvuq3wuVWWoU2TLDPmNnKrxHzY3v6Ai/go-multiaddr-net"
	pnet "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmW7Ump7YyBMr712Ta3iEVh3ZYcfVvJaPryfbCnyE826b4/go-libp2p-interface-pnet"
	smux "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmY9JXR3FupnYAYJWK9aMr9bCpqWKcToQ1tz8DVGTrHpHw/go-stream-muxer"
	ss "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmaKSpLZuCobQa8tfcKkZYdabfTPuihz113WM7RT9moeVS/go-conn-security"
	peer "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZSzKEM5yDfpZbeEEZaVmaZ1zXm6JWTbrQZSB8hCVPzk/go-libp2p-peer"
)

// ErrNilPeer is returned when attempting to upgrade an outbound connection
// without specifying a peer ID.
var ErrNilPeer = errors.New("nil peer")

// AcceptQueueLength is the number of connections to fully setup before not accepting any new connections
var AcceptQueueLength = 16

// Upgrader is a multistream upgrader that can upgrade an underlying connection
// to a full transport connection (secure and multiplexed).
type Upgrader struct {
	Protector pnet.Protector
	Secure    ss.Transport
	Muxer     smux.Transport
	Filters   *filter.Filters
}

// UpgradeListener upgrades the passed multiaddr-net listener into a full libp2p-transport listener.
func (u *Upgrader) UpgradeListener(t transport.Transport, list manet.Listener) transport.Listener {
	ctx, cancel := context.WithCancel(context.Background())
	l := &listener{
		Listener:  list,
		upgrader:  u,
		transport: t,
		threshold: newThreshold(AcceptQueueLength),
		incoming:  make(chan transport.Conn),
		cancel:    cancel,
		ctx:       ctx,
	}
	go l.handleIncoming()
	return l
}

// UpgradeOutbound upgrades the given outbound multiaddr-net connection into a
// full libp2p-transport connection.
func (u *Upgrader) UpgradeOutbound(ctx context.Context, t transport.Transport, maconn manet.Conn, p peer.ID) (transport.Conn, error) {
	if p == "" {
		return nil, ErrNilPeer
	}
	return u.upgrade(ctx, t, maconn, p)
}

// UpgradeInbound upgrades the given inbound multiaddr-net connection into a
// full libp2p-transport connection.
func (u *Upgrader) UpgradeInbound(ctx context.Context, t transport.Transport, maconn manet.Conn) (transport.Conn, error) {
	return u.upgrade(ctx, t, maconn, "")
}

func (u *Upgrader) upgrade(ctx context.Context, t transport.Transport, maconn manet.Conn, p peer.ID) (transport.Conn, error) {
	if u.Filters != nil && u.Filters.AddrBlocked(maconn.RemoteMultiaddr()) {
		log.Debugf("blocked connection from %s", maconn.RemoteMultiaddr())
		maconn.Close()
		return nil, fmt.Errorf("blocked connection from %s", maconn.RemoteMultiaddr())
	}

	var conn net.Conn = maconn
	if u.Protector != nil {
		pconn, err := u.Protector.Protect(conn)
		if err != nil {
			conn.Close()
			return nil, err
		}
		conn = pconn
	} else if pnet.ForcePrivateNetwork {
		log.Error("tried to dial with no Private Network Protector but usage" +
			" of Private Networks is forced by the enviroment")
		return nil, pnet.ErrNotInPrivateNetwork
	}
	sconn, err := u.setupSecurity(ctx, conn, p)
	if err != nil {
		conn.Close()
		return nil, err
	}
	smconn, err := u.setupMuxer(ctx, sconn, p)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return &transportConn{
		Conn:           smconn,
		ConnMultiaddrs: maconn,
		ConnSecurity:   sconn,
		transport:      t,
	}, nil
}

func (u *Upgrader) setupSecurity(ctx context.Context, conn net.Conn, p peer.ID) (ss.Conn, error) {
	if p == "" {
		return u.Secure.SecureInbound(ctx, conn)
	}
	return u.Secure.SecureOutbound(ctx, conn, p)
}

func (u *Upgrader) setupMuxer(ctx context.Context, conn net.Conn, p peer.ID) (smux.Conn, error) {
	// TODO: The muxer should take a context.
	done := make(chan struct{})

	var smconn smux.Conn
	var err error
	go func() {
		defer close(done)
		smconn, err = u.Muxer.NewConn(conn, p == "")
	}()

	select {
	case <-done:
		return smconn, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
