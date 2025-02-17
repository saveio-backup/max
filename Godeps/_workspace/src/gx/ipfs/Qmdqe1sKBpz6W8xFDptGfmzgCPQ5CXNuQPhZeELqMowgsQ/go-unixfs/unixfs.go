// Package unixfs implements a data format for files in the IPFS filesystem It
// is not the only format in ipfs, but it is the one that the filesystem
// assumes
package unixfs

import (
	"errors"

	proto "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmdxUuburamoF6zF9qjeQC4WYcWGbWuRmdLacMEsW8ioD8/gogo-protobuf/proto"

	dag "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmXkZeJmx4c3ddjw81DQMUpM1e5LjAack5idzZYWUb2qAJ/go-merkledag"
	pb "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qmdqe1sKBpz6W8xFDptGfmzgCPQ5CXNuQPhZeELqMowgsQ/go-unixfs/pb"
)

// Shorthands for protobuffer types
const (
	TRaw       = pb.Data_Raw
	TFile      = pb.Data_File
	TDirectory = pb.Data_Directory
	TMetadata  = pb.Data_Metadata
	TSymlink   = pb.Data_Symlink
	THAMTShard = pb.Data_HAMTShard
)

// Common errors
var (
	ErrMalformedFileFormat = errors.New("malformed data in file format")
	ErrUnrecognizedType    = errors.New("unrecognized node type")
)

// FromBytes unmarshals a byte slice as protobuf Data.
func FromBytes(data []byte) (*pb.Data, error) {
	pbdata := new(pb.Data)
	err := proto.Unmarshal(data, pbdata)
	if err != nil {
		return nil, err
	}
	return pbdata, nil
}

// FilePBData creates a protobuf File with the given
// byte slice and returns the marshaled protobuf bytes representing it.
func FilePBData(data []byte, totalsize uint64) []byte {
	pbfile := new(pb.Data)
	typ := pb.Data_File
	pbfile.Type = &typ
	pbfile.Data = data
	pbfile.Filesize = proto.Uint64(totalsize)

	data, err := proto.Marshal(pbfile)
	if err != nil {
		// This really shouldnt happen, i promise
		// The only failure case for marshal is if required fields
		// are not filled out, and they all are. If the proto object
		// gets changed and nobody updates this function, the code
		// should panic due to programmer error
		panic(err)
	}
	return data
}

//FolderPBData returns Bytes that represent a Directory.
func FolderPBData() []byte {
	pbfile := new(pb.Data)
	typ := pb.Data_Directory
	pbfile.Type = &typ

	data, err := proto.Marshal(pbfile)
	if err != nil {
		//this really shouldnt happen, i promise
		panic(err)
	}
	return data
}

//WrapData marshals raw bytes into a `Data_Raw` type protobuf message.
func WrapData(b []byte) []byte {
	pbdata := new(pb.Data)
	typ := pb.Data_Raw
	pbdata.Data = b
	pbdata.Type = &typ
	pbdata.Filesize = proto.Uint64(uint64(len(b)))

	out, err := proto.Marshal(pbdata)
	if err != nil {
		// This shouldnt happen. seriously.
		panic(err)
	}

	return out
}

//SymlinkData returns a `Data_Symlink` protobuf message for the path you specify.
func SymlinkData(path string) ([]byte, error) {
	pbdata := new(pb.Data)
	typ := pb.Data_Symlink
	pbdata.Data = []byte(path)
	pbdata.Type = &typ

	out, err := proto.Marshal(pbdata)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// UnwrapData unmarshals a protobuf messages and returns the contents.
func UnwrapData(data []byte) ([]byte, error) {
	pbdata := new(pb.Data)
	err := proto.Unmarshal(data, pbdata)
	if err != nil {
		return nil, err
	}
	return pbdata.GetData(), nil
}

// DataSize returns the size of the contents in protobuf wrapped slice.
// For raw data it simply provides the length of it. For Data_Files, it
// will return the associated filesize. Note that Data_Directories will
// return an error.
func DataSize(data []byte) (uint64, error) {
	pbdata := new(pb.Data)
	err := proto.Unmarshal(data, pbdata)
	if err != nil {
		return 0, err
	}

	switch pbdata.GetType() {
	case pb.Data_Directory:
		return 0, errors.New("can't get data size of directory")
	case pb.Data_File:
		return pbdata.GetFilesize(), nil
	case pb.Data_Raw:
		return uint64(len(pbdata.GetData())), nil
	default:
		return 0, errors.New("unrecognized node data type")
	}
}

// An FSNode represents a filesystem object using the UnixFS specification.
//
// The `NewFSNode` constructor should be used instead of just calling `new(FSNode)`
// to guarantee that the required (`Type` and `Filesize`) fields in the `format`
// structure are initialized before marshaling (in `GetBytes()`).
type FSNode struct {

	// UnixFS format defined as a protocol buffers message.
	format pb.Data
}

// FSNodeFromBytes unmarshal a protobuf message onto an FSNode.
func FSNodeFromBytes(b []byte) (*FSNode, error) {
	n := new(FSNode)
	err := proto.Unmarshal(b, &n.format)
	if err != nil {
		return nil, err
	}

	return n, nil
}

// NewFSNode creates a new FSNode structure with the given `dataType`.
//
// It initializes the (required) `Type` field (that doesn't have a `Set()`
// accessor so it must be specified at creation), otherwise the `Marshal()`
// method in `GetBytes()` would fail (`required field "Type" not set`).
//
// It also initializes the `Filesize` pointer field to ensure its value
// is never nil before marshaling, this is not a required field but it is
// done to be backwards compatible with previous `go-ipfs` versions hash.
// (If it wasn't initialized there could be cases where `Filesize` could
// have been left at nil, when the `FSNode` was created but no data or
// child nodes were set to adjust it, as is the case in `NewLeaf()`.)
func NewFSNode(dataType pb.Data_DataType) *FSNode {
	n := new(FSNode)
	n.format.Type = &dataType

	// Initialize by `Filesize` by updating it with a dummy (zero) value.
	n.UpdateFilesize(0)

	return n
}

// AddBlockSize adds the size of the next child block of this node
func (n *FSNode) AddBlockSize(s uint64) {
	n.UpdateFilesize(int64(s))
	n.format.Blocksizes = append(n.format.Blocksizes, s)
}

// RemoveBlockSize removes the given child block's size.
func (n *FSNode) RemoveBlockSize(i int) {
	n.UpdateFilesize(-int64(n.format.Blocksizes[i]))
	n.format.Blocksizes = append(n.format.Blocksizes[:i], n.format.Blocksizes[i+1:]...)
}

// BlockSize returns the block size indexed by `i`.
// TODO: Evaluate if this function should be bounds checking.
func (n *FSNode) BlockSize(i int) uint64 {
	return n.format.Blocksizes[i]
}

// RemoveAllBlockSizes removes all the child block sizes of this node.
func (n *FSNode) RemoveAllBlockSizes() {
	n.format.Blocksizes = []uint64{}
	n.format.Filesize = proto.Uint64(uint64(len(n.Data())))
}

// GetBytes marshals this node as a protobuf message.
func (n *FSNode) GetBytes() ([]byte, error) {
	return proto.Marshal(&n.format)
}

// FileSize returns the total size of this tree. That is, the size of
// the data in this node plus the size of all its children.
func (n *FSNode) FileSize() uint64 {
	return n.format.GetFilesize()
}

// NumChildren returns the number of child blocks of this node
func (n *FSNode) NumChildren() int {
	return len(n.format.Blocksizes)
}

// Data retrieves the `Data` field from the internal `format`.
func (n *FSNode) Data() []byte {
	return n.format.GetData()
}

// SetData sets the `Data` field from the internal `format`
// updating its `Filesize`.
func (n *FSNode) SetData(newData []byte) {
	n.UpdateFilesize(int64(len(newData) - len(n.Data())))
	n.format.Data = newData
}

// UpdateFilesize updates the `Filesize` field from the internal `format`
// by a signed difference (`filesizeDiff`).
// TODO: Add assert to check for `Filesize` > 0?
func (n *FSNode) UpdateFilesize(filesizeDiff int64) {
	n.format.Filesize = proto.Uint64(uint64(
		int64(n.format.GetFilesize()) + filesizeDiff))
}

// Type retrieves the `Type` field from the internal `format`.
func (n *FSNode) Type() pb.Data_DataType {
	return n.format.GetType()
}

// Metadata is used to store additional FSNode information.
type Metadata struct {
	MimeType string
	Size     uint64
}

// MetadataFromBytes Unmarshals a protobuf Data message into Metadata.
// The provided slice should have been encoded with BytesForMetadata().
func MetadataFromBytes(b []byte) (*Metadata, error) {
	pbd := new(pb.Data)
	err := proto.Unmarshal(b, pbd)
	if err != nil {
		return nil, err
	}
	if pbd.GetType() != pb.Data_Metadata {
		return nil, errors.New("incorrect node type")
	}

	pbm := new(pb.Metadata)
	err = proto.Unmarshal(pbd.Data, pbm)
	if err != nil {
		return nil, err
	}
	md := new(Metadata)
	md.MimeType = pbm.GetMimeType()
	return md, nil
}

// Bytes marshals Metadata as a protobuf message of Metadata type.
func (m *Metadata) Bytes() ([]byte, error) {
	pbm := new(pb.Metadata)
	pbm.MimeType = &m.MimeType
	return proto.Marshal(pbm)
}

// BytesForMetadata wraps the given Metadata as a profobuf message of Data type,
// setting the DataType to Metadata. The wrapped bytes are itself the
// result of calling m.Bytes().
func BytesForMetadata(m *Metadata) ([]byte, error) {
	pbd := new(pb.Data)
	pbd.Filesize = proto.Uint64(m.Size)
	typ := pb.Data_Metadata
	pbd.Type = &typ
	mdd, err := m.Bytes()
	if err != nil {
		return nil, err
	}

	pbd.Data = mdd
	return proto.Marshal(pbd)
}

// EmptyDirNode creates an empty folder Protonode.
func EmptyDirNode() *dag.ProtoNode {
	return dag.NodeWithData(FolderPBData())
}
