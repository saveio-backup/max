package main

import (
	nethttp "net/http"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmTjNRVt2fvaRFu93keEC7z5M1GS1iH6qZ9227htQioTUY/go-ipfs-cmds/examples/adder"

	http "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmTjNRVt2fvaRFu93keEC7z5M1GS1iH6qZ9227htQioTUY/go-ipfs-cmds/http"
)

func main() {
	h := http.NewHandler(nil, adder.RootCmd, http.NewServerConfig())

	// create http rpc server
	err := nethttp.ListenAndServe(":6798", h)
	if err != nil {
		panic(err)
	}
}
