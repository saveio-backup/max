package merkledag_test

import (
	"context"
	"testing"

	. "github.com/saveio/max/merkledag"
	dstest "github.com/saveio/max/merkledag/test"

	cid "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
	ipld "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qme5bWv7wtjUNGsK2BNGVUFPKiuxWrsqrtvYwCLRw8YFES/go-ipld-format"
)

func TestReadonlyProperties(t *testing.T) {
	ds := dstest.Mock()
	ro := NewReadOnlyDagService(ds)

	ctx := context.Background()
	nds := []ipld.Node{
		NewRawNode([]byte("foo1")),
		NewRawNode([]byte("foo2")),
		NewRawNode([]byte("foo3")),
		NewRawNode([]byte("foo4")),
	}
	cids := []*cid.Cid{
		nds[0].Cid(),
		nds[1].Cid(),
		nds[2].Cid(),
		nds[3].Cid(),
	}

	// add to the actual underlying datastore
	if err := ds.Add(ctx, nds[2]); err != nil {
		t.Fatal(err)
	}
	if err := ds.Add(ctx, nds[3]); err != nil {
		t.Fatal(err)
	}

	if err := ro.Add(ctx, nds[0]); err != ErrReadOnly {
		t.Fatal("expected ErrReadOnly")
	}
	if err := ro.Add(ctx, nds[2]); err != ErrReadOnly {
		t.Fatal("expected ErrReadOnly")
	}

	if err := ro.AddMany(ctx, nds[0:1]); err != ErrReadOnly {
		t.Fatal("expected ErrReadOnly")
	}

	if err := ro.Remove(ctx, cids[3]); err != ErrReadOnly {
		t.Fatal("expected ErrReadOnly")
	}
	if err := ro.RemoveMany(ctx, cids[1:2]); err != ErrReadOnly {
		t.Fatal("expected ErrReadOnly")
	}

	if _, err := ro.Get(ctx, cids[0]); err != ipld.ErrNotFound {
		t.Fatal("expected ErrNotFound")
	}
	if _, err := ro.Get(ctx, cids[3]); err != nil {
		t.Fatal(err)
	}
}
