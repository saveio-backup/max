package main

import (
	"context"
	"os"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmTjNRVt2fvaRFu93keEC7z5M1GS1iH6qZ9227htQioTUY/go-ipfs-cmds/examples/adder"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmTjNRVt2fvaRFu93keEC7z5M1GS1iH6qZ9227htQioTUY/go-ipfs-cmds"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmTjNRVt2fvaRFu93keEC7z5M1GS1iH6qZ9227htQioTUY/go-ipfs-cmds/cli"
)

func main() {
	// parse the command path, arguments and options from the command line
	req, err := cli.Parse(context.TODO(), os.Args[1:], os.Stdin, adder.RootCmd)
	if err != nil {
		panic(err)
	}

	req.Options["encoding"] = cmds.Text

	// create an emitter
	re, retCh := cli.NewResponseEmitter(os.Stdout, os.Stderr, req.Command.Encoders["Text"], req)

	if pr, ok := req.Command.PostRun[cmds.CLI]; ok {
		re = pr(req, re)
	}

	wait := make(chan struct{})
	// call command in background
	go func() {
		defer close(wait)

		adder.RootCmd.Call(req, re, nil)
	}()

	// wait until command has returned and exit
	ret := <-retCh
	<-wait

	os.Exit(ret)
}
