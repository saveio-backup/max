package mfs

import (
	"context"
	"fmt"
	"sync"

	dag "github.com/saveio/max/merkledag"
	ft "github.com/saveio/max/unixfs"
	mod "github.com/saveio/max/unixfs/mod"

	chunker "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmWo8jYc19ppG7YoTsrr2kEtLRbARTJho5oNXFTR6B7Peq/go-ipfs-chunker"
	ipld "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qme5bWv7wtjUNGsK2BNGVUFPKiuxWrsqrtvYwCLRw8YFES/go-ipld-format"
)

type File struct {
	parent childCloser

	name string

	desclock sync.RWMutex

	dserv  ipld.DAGService
	node   ipld.Node
	nodelk sync.Mutex

	RawLeaves bool
}

// NewFile returns a NewFile object with the given parameters.  If the
// Cid version is non-zero RawLeaves will be enabled.
func NewFile(name string, node ipld.Node, parent childCloser, dserv ipld.DAGService) (*File, error) {
	fi := &File{
		dserv:  dserv,
		parent: parent,
		name:   name,
		node:   node,
	}
	if node.Cid().Prefix().Version > 0 {
		fi.RawLeaves = true
	}
	return fi, nil
}

const (
	OpenReadOnly = iota
	OpenWriteOnly
	OpenReadWrite
)

func (fi *File) Open(flags int, sync bool) (FileDescriptor, error) {
	fi.nodelk.Lock()
	node := fi.node
	fi.nodelk.Unlock()

	switch node := node.(type) {
	case *dag.ProtoNode:
		fsn, err := ft.FSNodeFromBytes(node.Data())
		if err != nil {
			return nil, err
		}

		switch fsn.Type {
		default:
			return nil, fmt.Errorf("unsupported fsnode type for 'file'")
		case ft.TSymlink:
			return nil, fmt.Errorf("symlinks not yet supported")
		case ft.TFile, ft.TRaw:
			// OK case
		}
	case *dag.RawNode:
		// Ok as well.
	}

	switch flags {
	case OpenReadOnly:
		fi.desclock.RLock()
	case OpenWriteOnly, OpenReadWrite:
		fi.desclock.Lock()
	default:
		// TODO: support other modes
		return nil, fmt.Errorf("mode not supported")
	}

	dmod, err := mod.NewDagModifier(context.TODO(), node, fi.dserv, chunker.DefaultSplitter)
	if err != nil {
		return nil, err
	}
	dmod.RawLeaves = fi.RawLeaves

	return &fileDescriptor{
		inode: fi,
		perms: flags,
		sync:  sync,
		mod:   dmod,
	}, nil
}

// Size returns the size of this file
func (fi *File) Size() (int64, error) {
	fi.nodelk.Lock()
	defer fi.nodelk.Unlock()
	switch nd := fi.node.(type) {
	case *dag.ProtoNode:
		pbd, err := ft.FromBytes(nd.Data())
		if err != nil {
			return 0, err
		}
		return int64(pbd.GetFilesize()), nil
	case *dag.RawNode:
		return int64(len(nd.RawData())), nil
	default:
		return 0, fmt.Errorf("unrecognized node type in mfs/file.Size()")
	}
}

// GetNode returns the dag node associated with this file
func (fi *File) GetNode() (ipld.Node, error) {
	fi.nodelk.Lock()
	defer fi.nodelk.Unlock()
	return fi.node, nil
}

func (fi *File) Flush() error {
	// open the file in fullsync mode
	fd, err := fi.Open(OpenWriteOnly, true)
	if err != nil {
		return err
	}

	defer fd.Close()

	return fd.Flush()
}

func (fi *File) Sync() error {
	// just being able to take the writelock means the descriptor is synced
	fi.desclock.Lock()
	fi.desclock.Unlock()
	return nil
}

// Type returns the type FSNode this is
func (fi *File) Type() NodeType {
	return TFile
}
