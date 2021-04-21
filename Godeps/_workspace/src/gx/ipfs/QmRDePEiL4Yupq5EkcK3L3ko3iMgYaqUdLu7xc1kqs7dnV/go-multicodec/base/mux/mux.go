package basemux

import (
	mc "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRDePEiL4Yupq5EkcK3L3ko3iMgYaqUdLu7xc1kqs7dnV/go-multicodec"
	mux "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRDePEiL4Yupq5EkcK3L3ko3iMgYaqUdLu7xc1kqs7dnV/go-multicodec/mux"

	b64 "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRDePEiL4Yupq5EkcK3L3ko3iMgYaqUdLu7xc1kqs7dnV/go-multicodec/base/b64"
	bin "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRDePEiL4Yupq5EkcK3L3ko3iMgYaqUdLu7xc1kqs7dnV/go-multicodec/base/bin"
	hex "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRDePEiL4Yupq5EkcK3L3ko3iMgYaqUdLu7xc1kqs7dnV/go-multicodec/base/hex"
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
