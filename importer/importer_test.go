package importer

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"testing"

	mdtest "github.com/saveio/max/merkledag/test"
	uio "github.com/saveio/max/unixfs/io"

	u "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNiJuT8Ja3hMVpBHXv3Q6dwmperaQ6JjLtpMQgMCD7xvx/go-ipfs-util"
	chunker "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmWo8jYc19ppG7YoTsrr2kEtLRbARTJho5oNXFTR6B7Peq/go-ipfs-chunker"
	ipld "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qme5bWv7wtjUNGsK2BNGVUFPKiuxWrsqrtvYwCLRw8YFES/go-ipld-format"
)

func getBalancedDag(t testing.TB, size int64, blksize int64) (ipld.Node, ipld.DAGService) {
	ds := mdtest.Mock()
	r := io.LimitReader(u.NewTimeSeededRand(), size)
	nd, err := BuildDagFromReader(ds, chunker.NewSizeSplitter(r, blksize))
	if err != nil {
		t.Fatal(err)
	}
	return nd, ds
}

func getTrickleDag(t testing.TB, size int64, blksize int64) (ipld.Node, ipld.DAGService) {
	ds := mdtest.Mock()
	r := io.LimitReader(u.NewTimeSeededRand(), size)
	nd, err := BuildTrickleDagFromReader(ds, chunker.NewSizeSplitter(r, blksize))
	if err != nil {
		t.Fatal(err)
	}
	return nd, ds
}

func TestBalancedDag(t *testing.T) {
	ds := mdtest.Mock()
	buf := make([]byte, 10000)
	u.NewTimeSeededRand().Read(buf)
	r := bytes.NewReader(buf)

	nd, err := BuildDagFromReader(ds, chunker.DefaultSplitter(r))
	if err != nil {
		t.Fatal(err)
	}

	dr, err := uio.NewDagReader(context.Background(), nd, ds)
	if err != nil {
		t.Fatal(err)
	}

	out, err := ioutil.ReadAll(dr)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(out, buf) {
		t.Fatal("bad read")
	}
}

func BenchmarkBalancedReadSmallBlock(b *testing.B) {
	b.StopTimer()
	nbytes := int64(10000000)
	nd, ds := getBalancedDag(b, nbytes, 4096)

	b.SetBytes(nbytes)
	b.StartTimer()
	runReadBench(b, nd, ds)
}

func BenchmarkTrickleReadSmallBlock(b *testing.B) {
	b.StopTimer()
	nbytes := int64(10000000)
	nd, ds := getTrickleDag(b, nbytes, 4096)

	b.SetBytes(nbytes)
	b.StartTimer()
	runReadBench(b, nd, ds)
}

func BenchmarkBalancedReadFull(b *testing.B) {
	b.StopTimer()
	nbytes := int64(10000000)
	nd, ds := getBalancedDag(b, nbytes, chunker.DefaultBlockSize)

	b.SetBytes(nbytes)
	b.StartTimer()
	runReadBench(b, nd, ds)
}

func BenchmarkTrickleReadFull(b *testing.B) {
	b.StopTimer()
	nbytes := int64(10000000)
	nd, ds := getTrickleDag(b, nbytes, chunker.DefaultBlockSize)

	b.SetBytes(nbytes)
	b.StartTimer()
	runReadBench(b, nd, ds)
}

func runReadBench(b *testing.B, nd ipld.Node, ds ipld.DAGService) {
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		read, err := uio.NewDagReader(ctx, nd, ds)
		if err != nil {
			b.Fatal(err)
		}

		_, err = read.WriteTo(ioutil.Discard)
		if err != nil && err != io.EOF {
			b.Fatal(err)
		}
		cancel()
	}
}
