package merkledag

import (
	"context"

	cid "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
	ipld "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmZtNq8dArGfnpCZfx2pUNY7UcjGhVp5qqwQ4hH6mpTMRQ/go-ipld-format"
)

// ErrorService implements ipld.DAGService, returning 'Err' for every call.
type ErrorService struct {
	Err error
}

var _ ipld.DAGService = (*ErrorService)(nil)

// Add returns the cs.Err.
func (cs *ErrorService) Add(ctx context.Context, nd ipld.Node) error {
	return cs.Err
}

// AddMany returns the cs.Err.
func (cs *ErrorService) AddMany(ctx context.Context, nds []ipld.Node) error {
	return cs.Err
}

// Get returns the cs.Err.
func (cs *ErrorService) Get(ctx context.Context, c *cid.Cid) (ipld.Node, error) {
	return nil, cs.Err
}

// GetMany many returns the cs.Err.
func (cs *ErrorService) GetMany(ctx context.Context, cids []*cid.Cid) <-chan *ipld.NodeOption {
	ch := make(chan *ipld.NodeOption)
	close(ch)
	return ch
}

// Remove returns the cs.Err.
func (cs *ErrorService) Remove(ctx context.Context, c *cid.Cid) error {
	return cs.Err
}

// RemoveMany returns the cs.Err.
func (cs *ErrorService) RemoveMany(ctx context.Context, cids []*cid.Cid) error {
	return cs.Err
}
