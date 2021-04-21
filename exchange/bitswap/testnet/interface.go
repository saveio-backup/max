package bitswap

import (
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVvkK7s5imCiq3JVbL3pGfnhcCnf3LrFJPF4GE2sAoGZf/go-testutil"
	peer "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"

	bsnet "github.com/saveio/max/exchange/bitswap/network"
)

type Network interface {
	Adapter(testutil.Identity) bsnet.BitSwapNetwork

	HasPeer(peer.ID) bool
}
