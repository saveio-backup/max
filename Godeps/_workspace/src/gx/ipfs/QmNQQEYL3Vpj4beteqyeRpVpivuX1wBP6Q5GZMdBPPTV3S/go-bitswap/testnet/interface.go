package bitswap

import (
	bsnet "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNQQEYL3Vpj4beteqyeRpVpivuX1wBP6Q5GZMdBPPTV3S/go-bitswap/network"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmXG74iiKQnDstVQq9fPFQEB6JTNSWBbAWE1qsq6L4E5sR/go-testutil"
	peer "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZSzKEM5yDfpZbeEEZaVmaZ1zXm6JWTbrQZSB8hCVPzk/go-libp2p-peer"
)

type Network interface {
	Adapter(testutil.Identity) bsnet.BitSwapNetwork

	HasPeer(peer.ID) bool
}
