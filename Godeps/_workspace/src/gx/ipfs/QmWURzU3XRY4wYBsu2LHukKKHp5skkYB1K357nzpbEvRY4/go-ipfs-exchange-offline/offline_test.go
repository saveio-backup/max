package offline

import (
	"context"
	"testing"

	u "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmPdKqUcHGFdeSpvjVoaTRPPstGif9GBZb5Q56RVw9o69A/go-ipfs-util"
	blockstore "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRNFh4wm6FgTDrtsWmnvEP9NTuEa3Ykf72y1LXCyevbGW/go-ipfs-blockstore"
	blocks "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVzK524a2VWLqyvtBeiHKsUAWYgeAk4DBeZoY7vpNPNRx/go-block-format"
	cid "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
	blocksutil "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYqPGpZ9Yemr55xus9DiEztkns6Jti5XJ7hC94JbvkdqZ/go-ipfs-blocksutil"
	ds "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore"
	ds_sync "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore/sync"
)

func TestBlockReturnsErr(t *testing.T) {
	off := Exchange(bstore())
	c := cid.NewCidV0(u.Hash([]byte("foo")))
	_, err := off.GetBlock(context.Background(), c)
	if err != nil {
		return // as desired
	}
	t.Fail()
}

func TestHasBlockReturnsNil(t *testing.T) {
	store := bstore()
	ex := Exchange(store)
	block := blocks.NewBlock([]byte("data"))

	err := ex.HasBlock(block)
	if err != nil {
		t.Fail()
	}

	if _, err := store.Get(block.Cid()); err != nil {
		t.Fatal(err)
	}
}

func TestGetBlocks(t *testing.T) {
	store := bstore()
	ex := Exchange(store)
	g := blocksutil.NewBlockGenerator()

	expected := g.Blocks(2)

	for _, b := range expected {
		if err := ex.HasBlock(b); err != nil {
			t.Fail()
		}
	}

	request := func() []*cid.Cid {
		var ks []*cid.Cid

		for _, b := range expected {
			ks = append(ks, b.Cid())
		}
		return ks
	}()

	received, err := ex.GetBlocks(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}

	var count int
	for range received {
		count++
	}
	if len(expected) != count {
		t.Fail()
	}
}

func bstore() blockstore.Blockstore {
	return blockstore.NewBlockstore(ds_sync.MutexWrap(ds.NewMapDatastore()))
}
