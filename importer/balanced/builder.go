// Package balanced provides methods to build balanced DAGs.
// In a balanced DAG, nodes are added to a single root
// until the maximum number of links is reached (with leaves
// being at depth 0). Then, a new root is created, and points to the
// old root, and incorporates a new child, which proceeds to be
// filled up (link) to more leaves. In all cases, the Data (chunks)
// is stored only at the leaves, with the rest of nodes only
// storing links to their children.
//
// In a balanced DAG, nodes fill their link capacity before
// creating new ones, thus depth only increases when the
// current tree is completely full.
//
// Balanced DAGs are generalistic DAGs in which all leaves
// are at the same distance from the root.
package balanced

import (
	"errors"

	h "github.com/saveio/max/importer/helpers"

	ipld "gx/ipfs/Qme5bWv7wtjUNGsK2BNGVUFPKiuxWrsqrtvYwCLRw8YFES/go-ipld-format"
)

// Layout builds a balanced DAG. Data is stored at the leaves
// and depth only increases when the tree is full, that is, when
// the root node has reached the maximum number of links.
func Layout(db *h.DagBuilderHelper) (ipld.Node, error) {
	var offset uint64
	var root *h.UnixfsNode
	for level := 0; !db.Done(); level++ {

		nroot := db.NewUnixfsNode()
		db.SetPosInfo(nroot, 0)

		// add our old root as a child of the new root.
		if root != nil { // nil if it's the first node.
			if err := nroot.AddChild(root, db); err != nil {
				return nil, err
			}
		}

		// fill it up.
		if _, err := fillNodeRec(db, nroot, level, offset); err != nil {
			return nil, err
		}

		offset = nroot.FileSize()
		root = nroot

	}
	if root == nil {
		root = db.NewUnixfsNode()
	}

	out, err := db.Add(root)
	if err != nil {
		return nil, err
	}

	err = db.Close()
	if err != nil {
		return nil, err
	}

	return out, nil
}

func LayoutAndGetUnixFsNodes(db *h.DagBuilderHelper) (*h.UnixfsNode, []*h.UnixfsNode, error) {
	var offset uint64
	var root *h.UnixfsNode

	list := make([]*h.UnixfsNode, 0)
	offset = db.GetOffset()
	for level := 0; !db.Done() && level <= db.GetMaxlevel(); level++ {

		nroot := db.NewUnixfsNode()
		db.SetPosInfo(nroot, offset)

		// add our old root as a child of the new root.
		if root != nil { // nil if it's the first node.
			if err := nroot.AddChild(root, db); err != nil {
				return nil, nil, err
			}
		}

		// fill it up.
		c, err := fillNodeRec(db, nroot, level, offset)
		if err != nil {
			return nil, nil, err
		}
		if c != nil && len(c) > 0 {
			list = append(list, c...)
		}

		offset += nroot.FileSize()
		root = nroot
	}
	if root == nil {
		root = db.NewUnixfsNode()
	}

	return root, list, nil
}

func LayoutAndGetNodes(db *h.DagBuilderHelper) (ipld.Node, []*h.UnixfsNode, error) {
	var offset uint64
	var root *h.UnixfsNode
	list := make([]*h.UnixfsNode, 0)
	for level := 0; !db.Done(); level++ {

		nroot := db.NewUnixfsNode()
		db.SetPosInfo(nroot, 0)

		// add our old root as a child of the new root.
		if root != nil { // nil if it's the first node.
			if err := nroot.AddChild(root, db); err != nil {
				return nil, nil, err
			}
		}

		// fill it up.
		c, err := fillNodeRec(db, nroot, level, offset)
		if err != nil {
			return nil, nil, err
		}
		if c != nil && len(c) > 0 {
			list = append(list, c...)
		}

		offset = nroot.FileSize()
		root = nroot
	}
	if root == nil {
		root = db.NewUnixfsNode()
	}

	out, err := root.GetDagNode()
	if err != nil {
		return nil, nil, err
	}

	return out, list, nil
}

// fillNodeRec will fill the given node with data from the dagBuilders input
// source down to an indirection depth as specified by 'depth'
// it returns the total dataSize of the node, and a potential error
//
// warning: **children** pinned indirectly, but input node IS NOT pinned.
func fillNodeRec(db *h.DagBuilderHelper, node *h.UnixfsNode, depth int, offset uint64) ([]*h.UnixfsNode, error) {
	if depth < 0 {
		return nil, errors.New("attempt to fillNode at depth < 0")
	}

	list := make([]*h.UnixfsNode, 0)
	// Base case
	if depth <= 0 { // catch accidental -1's in case error above is removed.
		child, err := db.GetNextDataNode()
		if err != nil {
			return nil, err
		}
		node.Set(child)
		list = append(list, node)
		return list, nil
	}

	// while we have room AND we're not done
	for node.NumChildren() < db.Maxlinks() && !db.Done() {
		child := db.NewUnixfsNode()
		db.SetPosInfo(child, offset)

		c, err := fillNodeRec(db, child, depth-1, offset)
		if err != nil {
			return nil, err
		}

		if err := node.AddChild(child, db); err != nil {
			return nil, err
		}

		if !isContain(list, node) {
			list = append(list, node)
		}
		offset += child.FileSize()

		if c != nil && len(c) > 0 {
			list = append(list, c...)
		}
	}

	return list, nil
}

func isContain(nodes []*h.UnixfsNode, news *h.UnixfsNode) bool {
	for _, n := range nodes {
		if n == news {
			return true
		}
	}
	return false
}
