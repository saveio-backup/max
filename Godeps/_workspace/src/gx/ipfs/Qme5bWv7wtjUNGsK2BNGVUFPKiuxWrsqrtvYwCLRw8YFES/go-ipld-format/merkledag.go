package format

import (
	"context"
	"fmt"

	cid "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
)

var ErrNotFound = fmt.Errorf("merkledag: not found")

// Either a node or an error.
type NodeOption struct {
	Node Node
	Err  error
}

// The basic Node resolution service.
type NodeGetter interface {
	// Get retrieves nodes by CID. Depending on the NodeGetter
	// implementation, this may involve fetching the Node from a remote
	// machine; consider setting a deadline in the context.
	Get(context.Context, *cid.Cid) (Node, error)

	// GetMany returns a channel of NodeOptions given a set of CIDs.
	GetMany(context.Context, []*cid.Cid) <-chan *NodeOption
}

// NodeGetters can optionally implement this interface to make finding linked
// objects faster.
type LinkGetter interface {
	NodeGetter

	// TODO(ipfs/go-ipld-format#9): This should return []*cid.Cid

	// GetLinks returns the children of the node refered to by the given
	// CID.
	GetLinks(ctx context.Context, nd *cid.Cid) ([]*Link, error)
}

// DAGService is an IPFS Merkle DAG service.
type DAGService interface {
	NodeGetter

	// Add adds a node to this DAG.
	Add(context.Context, Node) error

	// Remove removes a node from this DAG.
	//
	// Remove returns no error if the requested node is not present in this DAG.
	Remove(context.Context, *cid.Cid) error

	// AddMany adds many nodes to this DAG.
	//
	// Consider using NewBatch instead of calling this directly if you need
	// to add an unbounded number of nodes to avoid buffering too much.
	AddMany(context.Context, []Node) error

	// RemoveMany removes many nodes from this DAG.
	//
	// It returns success even if the nodes were not present in the DAG.
	RemoveMany(context.Context, []*cid.Cid) error
}
