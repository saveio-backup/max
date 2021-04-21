package blockservice

import (
	"testing"

	blockstore "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRNFh4wm6FgTDrtsWmnvEP9NTuEa3Ykf72y1LXCyevbGW/go-ipfs-blockstore"
	blocks "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVzK524a2VWLqyvtBeiHKsUAWYgeAk4DBeZoY7vpNPNRx/go-block-format"
	offline "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmWURzU3XRY4wYBsu2LHukKKHp5skkYB1K357nzpbEvRY4/go-ipfs-exchange-offline"
	butil "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYqPGpZ9Yemr55xus9DiEztkns6Jti5XJ7hC94JbvkdqZ/go-ipfs-blocksutil"
	ds "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore"
	dssync "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore/sync"
)

func TestWriteThroughWorks(t *testing.T) {
	bstore := &PutCountingBlockstore{
		blockstore.NewBlockstore(dssync.MutexWrap(ds.NewMapDatastore())),
		0,
	}
	bstore2 := blockstore.NewBlockstore(dssync.MutexWrap(ds.NewMapDatastore()))
	exch := offline.Exchange(bstore2)
	bserv := NewWriteThrough(bstore, exch)
	bgen := butil.NewBlockGenerator()

	block := bgen.Next()

	t.Logf("PutCounter: %d", bstore.PutCounter)
	bserv.AddBlock(block)
	if bstore.PutCounter != 1 {
		t.Fatalf("expected just one Put call, have: %d", bstore.PutCounter)
	}

	bserv.AddBlock(block)
	if bstore.PutCounter != 2 {
		t.Fatalf("Put should have called again, should be 2 is: %d", bstore.PutCounter)
	}
}

var _ blockstore.Blockstore = (*PutCountingBlockstore)(nil)

type PutCountingBlockstore struct {
	blockstore.Blockstore
	PutCounter int
}

func (bs *PutCountingBlockstore) Put(block blocks.Block) error {
	bs.PutCounter++
	return bs.Blockstore.Put(block)
}
