package main

import (
	nethttp "net/http"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmUQb3xtNzkQCgTj2NjaqcJZNv2nfSSub2QAdy9DtQMRBT/go-ipfs-cmds/examples/adder"

	http "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmUQb3xtNzkQCgTj2NjaqcJZNv2nfSSub2QAdy9DtQMRBT/go-ipfs-cmds/http"
)

func main() {
	h := http.NewHandler(nil, adder.RootCmd, http.NewServerConfig())

	// create http rpc server
	err := nethttp.ListenAndServe(":6798", h)
	if err != nil {
		panic(err)
	}
}
