package basemux

import (
	mc "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmewJ1Zp9Hwz5HcMd7JYjhLXwvEHTL2UBCCz3oLt1E2N5z/go-multicodec"
	mux "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmewJ1Zp9Hwz5HcMd7JYjhLXwvEHTL2UBCCz3oLt1E2N5z/go-multicodec/mux"

	b64 "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmewJ1Zp9Hwz5HcMd7JYjhLXwvEHTL2UBCCz3oLt1E2N5z/go-multicodec/base/b64"
	bin "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmewJ1Zp9Hwz5HcMd7JYjhLXwvEHTL2UBCCz3oLt1E2N5z/go-multicodec/base/bin"
	hex "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmewJ1Zp9Hwz5HcMd7JYjhLXwvEHTL2UBCCz3oLt1E2N5z/go-multicodec/base/hex"
)

func AllBasesMux() *mux.Multicodec {
	m := mux.MuxMulticodec([]mc.Multicodec{
		hex.Multicodec(),
		b64.Multicodec(),
		bin.Multicodec(),
	}, mux.SelectFirst)
	m.Wrap = false
	return m
}
