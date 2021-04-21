// Package importer implements utilities used to create IPFS DAGs from files
// and readers.
package importer

import (
	bal "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qmdqe1sKBpz6W8xFDptGfmzgCPQ5CXNuQPhZeELqMowgsQ/go-unixfs/importer/balanced"
	h "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qmdqe1sKBpz6W8xFDptGfmzgCPQ5CXNuQPhZeELqMowgsQ/go-unixfs/importer/helpers"
	trickle "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qmdqe1sKBpz6W8xFDptGfmzgCPQ5CXNuQPhZeELqMowgsQ/go-unixfs/importer/trickle"

	ipld "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmZtNq8dArGfnpCZfx2pUNY7UcjGhVp5qqwQ4hH6mpTMRQ/go-ipld-format"
	chunker "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qmc3UwSvJkntxu2gKDPCqJEzmhqVeJtTbrVxJm6tsdmMF1/go-ipfs-chunker"
)

// BuildDagFromReader creates a DAG given a DAGService and a Splitter
// implementation (Splitters are io.Readers), using a Balanced layout.
func BuildDagFromReader(ds ipld.DAGService, spl chunker.Splitter) (ipld.Node, error) {
	dbp := h.DagBuilderParams{
		Dagserv:  ds,
		Maxlinks: h.DefaultLinksPerBlock,
	}

	return bal.Layout(dbp.New(spl))
}

// BuildTrickleDagFromReader creates a DAG given a DAGService and a Splitter
// implementation (Splitters are io.Readers), using a Trickle Layout.
func BuildTrickleDagFromReader(ds ipld.DAGService, spl chunker.Splitter) (ipld.Node, error) {
	dbp := h.DagBuilderParams{
		Dagserv:  ds,
		Maxlinks: h.DefaultLinksPerBlock,
	}

	return trickle.Layout(dbp.New(spl))
}
