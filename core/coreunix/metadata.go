package coreunix

import (
	cid "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"

	core "github.com/saveio/max/core"
	dag "github.com/saveio/max/merkledag"
	ft "github.com/saveio/max/unixfs"
)

func AddMetadataTo(n *core.IpfsNode, skey string, m *ft.Metadata) (string, error) {
	c, err := cid.Decode(skey)
	if err != nil {
		return "", err
	}

	nd, err := n.DAG.Get(n.Context(), c)
	if err != nil {
		return "", err
	}

	mdnode := new(dag.ProtoNode)
	mdata, err := ft.BytesForMetadata(m)
	if err != nil {
		return "", err
	}

	mdnode.SetData(mdata)
	if err := mdnode.AddNodeLink("file", nd); err != nil {
		return "", err
	}

	err = n.DAG.Add(n.Context(), mdnode)
	if err != nil {
		return "", err
	}

	return mdnode.Cid().String(), nil
}

func Metadata(n *core.IpfsNode, skey string) (*ft.Metadata, error) {
	c, err := cid.Decode(skey)
	if err != nil {
		return nil, err
	}

	nd, err := n.DAG.Get(n.Context(), c)
	if err != nil {
		return nil, err
	}

	pbnd, ok := nd.(*dag.ProtoNode)
	if !ok {
		return nil, dag.ErrNotProtobuf
	}

	return ft.MetadataFromBytes(pbnd.Data())
}
