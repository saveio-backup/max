package bstest

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmeZMtdkNG7u2CohGSL8mzAdZY2c3B1coYE91wvbzip1pF/go-blockservice"

	u "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmPdKqUcHGFdeSpvjVoaTRPPstGif9GBZb5Q56RVw9o69A/go-ipfs-util"
	blockstore "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRNFh4wm6FgTDrtsWmnvEP9NTuEa3Ykf72y1LXCyevbGW/go-ipfs-blockstore"
	blocks "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVzK524a2VWLqyvtBeiHKsUAWYgeAk4DBeZoY7vpNPNRx/go-block-format"
	offline "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmWURzU3XRY4wYBsu2LHukKKHp5skkYB1K357nzpbEvRY4/go-ipfs-exchange-offline"
	cid "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
	ds "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore"
	dssync "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore/sync"
)

func newObject(data []byte) blocks.Block {
	return blocks.NewBlock(data)
}

func TestBlocks(t *testing.T) {
	bstore := blockstore.NewBlockstore(dssync.MutexWrap(ds.NewMapDatastore()))
	bs := New(bstore, offline.Exchange(bstore))
	defer bs.Close()

	o := newObject([]byte("beep boop"))
	h := cid.NewCidV0(u.Hash([]byte("beep boop")))
	if !o.Cid().Equals(h) {
		t.Error("Block key and data multihash key not equal")
	}

	err := bs.AddBlock(o)
	if err != nil {
		t.Error("failed to add block to BlockService", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	b2, err := bs.GetBlock(ctx, o.Cid())
	if err != nil {
		t.Error("failed to retrieve block from BlockService", err)
		return
	}

	if !o.Cid().Equals(b2.Cid()) {
		t.Error("Block keys not equal.")
	}

	if !bytes.Equal(o.RawData(), b2.RawData()) {
		t.Error("Block data is not equal.")
	}
}

func makeObjects(n int) []blocks.Block {
	var out []blocks.Block
	for i := 0; i < n; i++ {
		out = append(out, newObject([]byte(fmt.Sprintf("object %d", i))))
	}
	return out
}

func TestGetBlocksSequential(t *testing.T) {
	var servs = Mocks(4)
	for _, s := range servs {
		defer s.Close()
	}
	objs := makeObjects(50)

	var cids []*cid.Cid
	for _, o := range objs {
		cids = append(cids, o.Cid())
		servs[0].AddBlock(o)
	}

	t.Log("one instance at a time, get blocks concurrently")

	for i := 1; i < len(servs); i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*50)
		defer cancel()
		out := servs[i].GetBlocks(ctx, cids)
		gotten := make(map[string]blocks.Block)
		for blk := range out {
			if _, ok := gotten[blk.Cid().KeyString()]; ok {
				t.Fatal("Got duplicate block!")
			}
			gotten[blk.Cid().KeyString()] = blk
		}
		if len(gotten) != len(objs) {
			t.Fatalf("Didnt get enough blocks back: %d/%d", len(gotten), len(objs))
		}
	}
}
