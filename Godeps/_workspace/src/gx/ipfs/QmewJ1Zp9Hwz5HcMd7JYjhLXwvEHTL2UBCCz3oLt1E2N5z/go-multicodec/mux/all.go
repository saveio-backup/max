package muxcodec

import (
	mc "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmewJ1Zp9Hwz5HcMd7JYjhLXwvEHTL2UBCCz3oLt1E2N5z/go-multicodec"
	cbor "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmewJ1Zp9Hwz5HcMd7JYjhLXwvEHTL2UBCCz3oLt1E2N5z/go-multicodec/cbor"
	json "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmewJ1Zp9Hwz5HcMd7JYjhLXwvEHTL2UBCCz3oLt1E2N5z/go-multicodec/json"
)

func StandardMux() *Multicodec {
	return MuxMulticodec([]mc.Multicodec{
		cbor.Multicodec(),
		json.Multicodec(false),
		json.Multicodec(true),
	}, SelectFirst)
}
