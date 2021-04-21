package network

import (
	"context"

	bsmsg "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNQQEYL3Vpj4beteqyeRpVpivuX1wBP6Q5GZMdBPPTV3S/go-bitswap/message"

	cid "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
	protocol "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	peer "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZSzKEM5yDfpZbeEEZaVmaZ1zXm6JWTbrQZSB8hCVPzk/go-libp2p-peer"
	ifconnmgr "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmeJbAMK4cZc1RMChb68h9t2jqvK8miqE8oQiwGAf4EdQq/go-libp2p-interface-connmgr"
)

var (
	// These two are equivalent, legacy
	ProtocolBitswapOne    protocol.ID = "/ipfs/bitswap/1.0.0"
	ProtocolBitswapNoVers protocol.ID = "/ipfs/bitswap"

	ProtocolBitswap protocol.ID = "/ipfs/bitswap/1.1.0"
)

// BitSwapNetwork provides network connectivity for BitSwap sessions
type BitSwapNetwork interface {

	// SendMessage sends a BitSwap message to a peer.
	SendMessage(
		context.Context,
		peer.ID,
		bsmsg.BitSwapMessage) error

	// SetDelegate registers the Reciver to handle messages received from the
	// network.
	SetDelegate(Receiver)

	ConnectTo(context.Context, peer.ID) error

	NewMessageSender(context.Context, peer.ID) (MessageSender, error)

	ConnectionManager() ifconnmgr.ConnManager

	Routing
}

type MessageSender interface {
	SendMsg(context.Context, bsmsg.BitSwapMessage) error
	Close() error
	Reset() error
}

// Implement Receiver to receive messages from the BitSwapNetwork
type Receiver interface {
	ReceiveMessage(
		ctx context.Context,
		sender peer.ID,
		incoming bsmsg.BitSwapMessage)

	ReceiveError(error)

	// Connected/Disconnected warns bitswap about peer connections
	PeerConnected(peer.ID)
	PeerDisconnected(peer.ID)
}

type Routing interface {
	// FindProvidersAsync returns a channel of providers for the given key
	FindProvidersAsync(context.Context, *cid.Cid, int) <-chan peer.ID

	// Provide provides the key to the network
	Provide(context.Context, *cid.Cid) error
}
