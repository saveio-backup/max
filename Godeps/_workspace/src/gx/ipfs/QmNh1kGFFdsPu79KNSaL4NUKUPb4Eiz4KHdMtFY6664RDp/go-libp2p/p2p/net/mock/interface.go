// Package mocknet provides a mock net.Network to test with.
//
// - a Mocknet has many inet.Networks
// - a Mocknet has many Links
// - a Link joins two inet.Networks
// - inet.Conns and inet.Streams are created by inet.Networks
package mocknet

import (
	"io"
	"time"

	host "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNmJZL7FQySMtE2BQuLMuZg2EB2CLEunJJUSVSc9YnnbV/go-libp2p-host"

	ma "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
	pstore "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmXauCuJzmzapetmC6W4TuDJLL1yFFrVzSHoWv8YdbmnxH/go-libp2p-peerstore"
	inet "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmXfkENeeBvh3zYA51MaSdGUdBjhQ99cP5WQe8zgr6wchG/go-libp2p-net"
	peer "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
	ic "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
)

type Mocknet interface {

	// GenPeer generates a peer and its inet.Network in the Mocknet
	GenPeer() (host.Host, error)

	// AddPeer adds an existing peer. we need both a privkey and addr.
	// ID is derived from PrivKey
	AddPeer(ic.PrivKey, ma.Multiaddr) (host.Host, error)
	AddPeerWithPeerstore(peer.ID, pstore.Peerstore) (host.Host, error)

	// retrieve things (with randomized iteration order)
	Peers() []peer.ID
	Net(peer.ID) inet.Network
	Nets() []inet.Network
	Host(peer.ID) host.Host
	Hosts() []host.Host
	Links() LinkMap
	LinksBetweenPeers(a, b peer.ID) []Link
	LinksBetweenNets(a, b inet.Network) []Link

	// Links are the **ability to connect**.
	// think of Links as the physical medium.
	// For p1 and p2 to connect, a link must exist between them.
	// (this makes it possible to test dial failures, and
	// things like relaying traffic)
	LinkPeers(peer.ID, peer.ID) (Link, error)
	LinkNets(inet.Network, inet.Network) (Link, error)
	Unlink(Link) error
	UnlinkPeers(peer.ID, peer.ID) error
	UnlinkNets(inet.Network, inet.Network) error

	// LinkDefaults are the default options that govern links
	// if they do not have thier own option set.
	SetLinkDefaults(LinkOptions)
	LinkDefaults() LinkOptions

	// Connections are the usual. Connecting means Dialing.
	// **to succeed, peers must be linked beforehand**
	ConnectPeers(peer.ID, peer.ID) (inet.Conn, error)
	ConnectNets(inet.Network, inet.Network) (inet.Conn, error)
	DisconnectPeers(peer.ID, peer.ID) error
	DisconnectNets(inet.Network, inet.Network) error
	LinkAll() error
	ConnectAllButSelf() error
}

// LinkOptions are used to change aspects of the links.
// Sorry but they dont work yet :(
type LinkOptions struct {
	Latency   time.Duration
	Bandwidth float64 // in bytes-per-second
	// we can make these values distributions down the road.
}

// Link represents the **possibility** of a connection between
// two peers. Think of it like physical network links. Without
// them, the peers can try and try but they won't be able to
// connect. This allows constructing topologies where specific
// nodes cannot talk to each other directly. :)
type Link interface {
	Networks() []inet.Network
	Peers() []peer.ID

	SetOptions(LinkOptions)
	Options() LinkOptions

	// Metrics() Metrics
}

// LinkMap is a 3D map to give us an easy way to track links.
// (wow, much map. so data structure. how compose. ahhh pointer)
type LinkMap map[string]map[string]map[Link]struct{}

// Printer lets you inspect things :)
type Printer interface {
	// MocknetLinks shows the entire Mocknet's link table :)
	MocknetLinks(mn Mocknet)
	NetworkConns(ni inet.Network)
}

// PrinterTo returns a Printer ready to write to w.
func PrinterTo(w io.Writer) Printer {
	return &printer{w}
}
