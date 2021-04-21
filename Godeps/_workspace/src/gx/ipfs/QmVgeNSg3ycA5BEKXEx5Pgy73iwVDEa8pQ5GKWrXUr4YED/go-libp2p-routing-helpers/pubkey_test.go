package routinghelpers

import (
	"context"
	"testing"

	peert "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZSzKEM5yDfpZbeEEZaVmaZ1zXm6JWTbrQZSB8hCVPzk/go-libp2p-peer/test"
	routing "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmewrvpGvgK9qkCtXsGNwXiQzyux4jcHNjoyVrGdsgtNK5/go-libp2p-routing"
)

func TestGetPublicKey(t *testing.T) {
	d := Parallel{
		Parallel{
			&Compose{
				ValueStore: &LimitedValueStore{
					ValueStore: new(dummyValueStore),
					Namespaces: []string{"other"},
				},
			},
		},
		Tiered{
			&Compose{
				ValueStore: &LimitedValueStore{
					ValueStore: new(dummyValueStore),
					Namespaces: []string{"pk"},
				},
			},
		},
		&Compose{
			ValueStore: &LimitedValueStore{
				ValueStore: new(dummyValueStore),
				Namespaces: []string{"other", "pk"},
			},
		},
		&struct{ Compose }{Compose{ValueStore: &LimitedValueStore{ValueStore: Null{}}}},
		&struct{ Compose }{},
	}

	pid, _ := peert.RandPeerID()

	ctx := context.Background()
	if _, err := d.GetPublicKey(ctx, pid); err != routing.ErrNotFound {
		t.Fatal(err)
	}
}
