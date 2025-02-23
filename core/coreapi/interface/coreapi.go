// Package iface defines ONT-IPFS Core API which is a set of interfaces used to
// interact with ONT-IPFS nodes.
package iface

import (
	"context"

	ipld "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qme5bWv7wtjUNGsK2BNGVUFPKiuxWrsqrtvYwCLRw8YFES/go-ipld-format"
)

// CoreAPI defines an unified interface to ONT-IPFS for Go programs
type CoreAPI interface {
	// Unixfs returns an implementation of Unixfs API
	Unixfs() UnixfsAPI

	// Block returns an implementation of Block API
	Block() BlockAPI

	// Dag returns an implementation of Dag API
	Dag() DagAPI

	// Name returns an implementation of Name API
	Name() NameAPI

	// Key returns an implementation of Key API
	Key() KeyAPI

	// Pin returns an implementation of Pin API
	Pin() PinAPI

	// ObjectAPI returns an implementation of Object API
	Object() ObjectAPI

	// ResolvePath resolves the path using Unixfs resolver
	ResolvePath(context.Context, Path) (Path, error)

	// ResolveNode resolves the path (if not resolved already) using Unixfs
	// resolver, gets and returns the resolved Node
	ResolveNode(context.Context, Path) (ipld.Node, error)
}
