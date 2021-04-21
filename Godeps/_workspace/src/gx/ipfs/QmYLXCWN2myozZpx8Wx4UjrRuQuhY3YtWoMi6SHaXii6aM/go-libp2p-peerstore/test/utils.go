package testutil

import (
	"io"
	"math/rand"
	"time"

	mh "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmPnFwZ2JXKnXgMw8CdBPxn7FWh6LLdjUjxV1fKHuJnkr8/go-multihash"
	ci "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmPvyPwuCgJ7pDmrKDxRtsScJgBaM5h4EpRL2qQJsmXf4n/go-libp2p-crypto"
	peer "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZSzKEM5yDfpZbeEEZaVmaZ1zXm6JWTbrQZSB8hCVPzk/go-libp2p-peer"
)

func timeSeededRand() io.Reader {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

func RandPeerID() (peer.ID, error) {
	buf := make([]byte, 16)
	if _, err := io.ReadFull(timeSeededRand(), buf); err != nil {
		return "", err
	}
	h, err := mh.Sum(buf, mh.SHA2_256, -1)
	if err != nil {
		return "", err
	}

	return peer.ID(h), nil
}

func RandTestKeyPair(bits int) (ci.PrivKey, ci.PubKey, error) {
	return ci.GenerateKeyPairWithReader(ci.RSA, bits, timeSeededRand())
}

func SeededTestKeyPair(seed int64) (ci.PrivKey, ci.PubKey, error) {
	return ci.GenerateKeyPairWithReader(ci.RSA, 512, rand.New(rand.NewSource(seed)))
}
