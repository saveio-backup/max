package helpers

import (
	"context"
	"io"
	"os"

	dag "github.com/saveio/max/merkledag"
	ft "github.com/saveio/max/unixfs"

	chunker "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmWo8jYc19ppG7YoTsrr2kEtLRbARTJho5oNXFTR6B7Peq/go-ipfs-chunker"
	cid "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
	files "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmceUdzxkimdYsgtX733uNgzf1DLHyBKN6ehGSp85ayppM/go-ipfs-cmdkit/files"
	ipld "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qme5bWv7wtjUNGsK2BNGVUFPKiuxWrsqrtvYwCLRw8YFES/go-ipld-format"
)

// DagBuilderHelper wraps together a bunch of objects needed to
// efficiently create unixfs dag trees
type DagBuilderHelper struct {
	dserv     ipld.DAGService
	spl       chunker.Splitter
	recvdErr  error
	rawLeaves bool
	nextData  []byte // the next item to return.
	maxlinks  int
	maxlevel  int
	offset    uint64
	batch     *ipld.Batch
	fullPath  string
	stat      os.FileInfo
	prefix    *cid.Prefix
}

// DagBuilderParams wraps configuration options to create a DagBuilderHelper
// from a chunker.Splitter.
type DagBuilderParams struct {
	// Maximum number of links per intermediate node
	Maxlinks int

	// Maximum number of level per 'root' node
	// Use this and Maxlinks to control the size of single tree when handling large file
	Maxlevel int

	// RawLeaves signifies that the importer should use raw ipld nodes as leaves
	// instead of using the unixfs TRaw type
	RawLeaves bool

	// CID Prefix to use if set
	Prefix *cid.Prefix

	// DAGService to write blocks to (required)
	Dagserv ipld.DAGService

	// NoCopy signals to the chunker that it should track fileinfo for
	// filestore adds
	NoCopy bool
}

// New generates a new DagBuilderHelper from the given params and a given
// chunker.Splitter as data source.
func (dbp *DagBuilderParams) New(spl chunker.Splitter) *DagBuilderHelper {
	db := &DagBuilderHelper{
		dserv:     dbp.Dagserv,
		spl:       spl,
		rawLeaves: dbp.RawLeaves,
		prefix:    dbp.Prefix,
		maxlinks:  dbp.Maxlinks,
		maxlevel:  dbp.Maxlevel,
		batch:     ipld.NewBatch(context.TODO(), dbp.Dagserv),
	}
	if fi, ok := spl.Reader().(files.FileInfo); dbp.NoCopy && ok {
		db.fullPath = fi.AbsPath()
		db.stat = fi.Stat()
	}
	return db
}

// prepareNext consumes the next item from the splitter and puts it
// in the nextData field. it is idempotent-- if nextData is full
// it will do nothing.
func (db *DagBuilderHelper) prepareNext() {
	// if we already have data waiting to be consumed, we're ready
	if db.nextData != nil || db.recvdErr != nil {
		return
	}

	db.nextData, db.recvdErr = db.spl.NextBytes()
	if db.recvdErr == io.EOF {
		db.recvdErr = nil
	}
}

// Done returns whether or not we're done consuming the incoming data.
func (db *DagBuilderHelper) Done() bool {
	// ensure we have an accurate perspective on data
	// as `done` this may be called before `next`.
	db.prepareNext() // idempotent
	if db.recvdErr != nil {
		return false
	}
	return db.nextData == nil
}

// Next returns the next chunk of data to be inserted into the dag
// if it returns nil, that signifies that the stream is at an end, and
// that the current building operation should finish.
func (db *DagBuilderHelper) Next() ([]byte, error) {
	db.prepareNext() // idempotent
	d := db.nextData
	db.nextData = nil // signal we've consumed it
	if db.recvdErr != nil {
		return nil, db.recvdErr
	}
	return d, nil
}

// GetDagServ returns the dagservice object this Helper is using
func (db *DagBuilderHelper) GetDagServ() ipld.DAGService {
	return db.dserv
}

// NewUnixfsNode creates a new Unixfs node to represent a file.
func (db *DagBuilderHelper) NewUnixfsNode() *UnixfsNode {
	n := &UnixfsNode{
		node: new(dag.ProtoNode),
		ufmt: &ft.FSNode{Type: ft.TFile},
	}
	n.SetPrefix(db.prefix)
	return n
}

// newUnixfsBlock creates a new Unixfs node to represent a raw data block
func (db *DagBuilderHelper) newUnixfsBlock() *UnixfsNode {
	n := &UnixfsNode{
		node: new(dag.ProtoNode),
		ufmt: &ft.FSNode{Type: ft.TRaw},
	}
	n.SetPrefix(db.prefix)
	return n
}

// FillNodeLayer will add datanodes as children to the give node until
// at most db.indirSize nodes are added.
func (db *DagBuilderHelper) FillNodeLayer(node *UnixfsNode) error {

	// while we have room AND we're not done
	for node.NumChildren() < db.maxlinks && !db.Done() {
		child, err := db.GetNextDataNode()
		if err != nil {
			return err
		}

		if err := node.AddChild(child, db); err != nil {
			return err
		}
	}

	return nil
}

// GetNextDataNode builds a UnixFsNode with the data obtained from the
// Splitter, given the constraints (BlockSizeLimit, RawLeaves) specified
// when creating the DagBuilderHelper.
func (db *DagBuilderHelper) GetNextDataNode() (*UnixfsNode, error) {
	data, err := db.Next()
	if err != nil {
		return nil, err
	}

	if data == nil { // we're done!
		return nil, nil
	}

	if len(data) > BlockSizeLimit {
		return nil, ErrSizeLimitExceeded
	}

	if db.rawLeaves {
		if db.prefix == nil {
			return &UnixfsNode{
				rawnode: dag.NewRawNode(data),
				raw:     true,
			}, nil
		}
		rawnode, err := dag.NewRawNodeWPrefix(data, *db.prefix)
		if err != nil {
			return nil, err
		}
		return &UnixfsNode{
			rawnode: rawnode,
			raw:     true,
		}, nil
	}

	blk := db.newUnixfsBlock()
	blk.SetData(data)
	return blk, nil
}

// SetPosInfo sets the offset information of a node using the fullpath and stat
// from the DagBuilderHelper.
func (db *DagBuilderHelper) SetPosInfo(node *UnixfsNode, offset uint64) {
	if db.fullPath != "" {
		node.SetPosInfo(offset, db.fullPath, db.stat)
	}
}

// Add sends a node to the DAGService, and returns it.
func (db *DagBuilderHelper) Add(node *UnixfsNode) (ipld.Node, error) {
	dn, err := node.GetDagNode()
	if err != nil {
		return nil, err
	}

	err = db.dserv.Add(context.TODO(), dn)
	if err != nil {
		return nil, err
	}

	return dn, nil
}

// Maxlinks returns the configured maximum number for links
// for nodes built with this helper.
func (db *DagBuilderHelper) Maxlinks() int {
	return db.maxlinks
}

// Close has the DAGService perform a batch Commit operation.
// It should be called at the end of the building process to make
// sure all data is persisted.
func (db *DagBuilderHelper) Close() error {
	return db.batch.Commit()
}

func (db *DagBuilderHelper) SetOffset(offset uint64) {
	db.offset = offset
	return
}

func (db *DagBuilderHelper) GetOffset() uint64 {
	return db.offset
}

func (db *DagBuilderHelper) GetMaxlevel() int {
	return db.maxlevel
}
