package merkledag

import (
	"encoding/json"
	"fmt"
	cid "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
	ipld "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qme5bWv7wtjUNGsK2BNGVUFPKiuxWrsqrtvYwCLRw8YFES/go-ipld-format"
)

type DirNode struct {
	links []*ipld.Link
	data  []byte

	// cache encoded/marshaled value
	encoded []byte

	cached *cid.Cid

	// Prefix specifies cid version and hashing function
	Prefix cid.Prefix
}

var _ ipld.Node = (*DirNode)(nil)

// Copy returns a copy of the node.
// NOTE: Does not make copies of Node objects in the links.
func (n *DirNode) Copy() ipld.Node {
	nnode := new(DirNode)
	if len(n.data) > 0 {
		nnode.data = make([]byte, len(n.data))
		copy(nnode.data, n.data)
	}

	if len(n.links) > 0 {
		nnode.links = make([]*ipld.Link, len(n.links))
		copy(nnode.links, n.links)
	}

	nnode.Prefix = n.Prefix

	return nnode
}

// RawData returns the protobuf-encoded version of the node.
func (n *DirNode) RawData() []byte {
	return nil
}

// Size returns the total size of the data addressed by node,
// including the total sizes of references.
func (n *DirNode) Size() (uint64, error) {
	return 0, nil
}

// Stat returns statistics on the node.
func (n *DirNode) Stat() (*ipld.NodeStat, error) {
	cumSize, err := n.Size()
	if err != nil {
		return nil, err
	}
	return &ipld.NodeStat{
		Hash:           n.Cid().String(),
		NumLinks:       len(n.links),
		BlockSize:      0,
		LinksSize:      0, // includes framing.
		DataSize:       len(n.data),
		CumulativeSize: int(cumSize),
	}, nil
}

// Loggable implements the ipfs/go-log.Loggable interface.
func (n *DirNode) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"node": n.String(),
	}
}

// UnmarshalJSON reads the node fields from a JSON-encoded byte slice.
func (n *DirNode) UnmarshalJSON(b []byte) error {
	s := struct {
		Data  []byte       `json:"data"`
		Links []*ipld.Link `json:"links"`
	}{}

	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	n.data = s.Data
	n.links = s.Links
	return nil
}

// MarshalJSON returns a JSON representation of the node.
func (n *DirNode) MarshalJSON() ([]byte, error) {
	out := map[string]interface{}{
		"data":  n.data,
		"links": n.links,
	}

	return json.Marshal(out)
}

// SetPrefix sets the CID prefix if it is non nil, if prefix is nil then
// it resets it the default value
func (n *DirNode) SetPrefix(prefix *cid.Prefix) {
	if prefix == nil {
		n.Prefix = v0CidPrefix
	} else {
		n.Prefix = *prefix
		n.Prefix.Codec = cid.DagProtobuf
		n.encoded = nil
		n.cached = nil
	}
}

// Cid returns the node's Cid, calculated according to its prefix
// and raw data contents.
func (n *DirNode) Cid() *cid.Cid {
	if n.encoded != nil && n.cached != nil {
		return n.cached
	}

	if n.Prefix.Codec == 0 {
		n.SetPrefix(nil)
	}

	c, err := n.Prefix.Sum(n.RawData())
	if err != nil {
		// programmer error
		err = fmt.Errorf("invalid CID of length %d: %x: %v", len(n.RawData()), n.RawData(), err)
		panic(err)
	}

	n.cached = c
	return c
}

// String prints the node's Cid.
func (n *DirNode) String() string {
	return n.Cid().String()
}

// Links returns the node links.
func (n *DirNode) Links() []*ipld.Link {
	return n.links
}

// SetLinks replaces the node links with the given ones.
func (n *DirNode) SetLinks(links []*ipld.Link) {
	n.links = links
}

// Resolve is an alias for ResolveLink.
func (n *DirNode) Resolve(path []string) (interface{}, []string, error) {
	return n.ResolveLink(path)
}

// ResolveLink consumes the first element of the path and obtains the link
// corresponding to it from the node. It returns the link
// and the path without the consumed element.
func (n *DirNode) ResolveLink(path []string) (*ipld.Link, []string, error) {
	if len(path) == 0 {
		return nil, nil, fmt.Errorf("end of path, no more links to resolve")
	}

	lnk, err := n.GetNodeLink(path[0])
	if err != nil {
		return nil, nil, err
	}

	return lnk, path[1:], nil
}

// GetNodeLink returns a copy of the link with the given name.
func (n *DirNode) GetNodeLink(name string) (*ipld.Link, error) {
	for _, l := range n.links {
		if l.Name == name {
			return &ipld.Link{
				Name: l.Name,
				Size: l.Size,
				Cid:  l.Cid,
			}, nil
		}
	}
	return nil, ErrLinkNotFound
}

// Tree returns the link names of the DirNode.
// DirNodes are only ever one path deep, so anything different than an empty
// string for p results in nothing. The depth parameter is ignored.
func (n *DirNode) Tree(p string, depth int) []string {
	if p != "" {
		return nil
	}

	out := make([]string, 0, len(n.links))
	for _, lnk := range n.links {
		out = append(out, lnk.Name)
	}
	return out
}
