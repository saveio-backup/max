package filestore

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	pb "github.com/saveio/max/filestore/pb"

	proto "gx/ipfs/QmT6n4mspWYEya864BhCUJEgyxiRfmiSY9ruQwTUNpRKaM/protobuf/proto"
	dshelp "gx/ipfs/QmTmqJGRQfuH8eKWD1FjThwPRipt1QhqJQNZ8MpzmfAAxo/go-ipfs-ds-help"
	ds "gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore"
	dsns "gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore/namespace"
	dsq "gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore/query"
	blockstore "gx/ipfs/QmaG4DZ4JaqEfvPWt5nPPgoTzhc1tr1T3f4Nu9Jpdm8ymY/go-ipfs-blockstore"
	posinfo "gx/ipfs/Qmb3jLEFAQrqdVgWUajqEyuuDoavkSq1XQXz6tWdFWF995/go-ipfs-posinfo"
	cid "gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
	blocks "gx/ipfs/Qmej7nf81hi2x2tvjRBF3mcp74sQyuDH4VMYDGd1YtXjb2/go-block-format"
	mpool "gx/ipfs/QmWBug6eBS7AxRdCDVuSY5CnSit7cS2XnPFYJWqWDumhCG/go-msgio/mpool"
)

// FilestorePrefix identifies the key prefix for FileManager blocks.
var FilestorePrefix = ds.NewKey("filestore")

// FileManager is a blockstore implementation which stores special
// blocks FilestoreNode type. These nodes only contain a reference
// to the actual location of the block data in the filesystem
// (a path and an offset).
type FileManager struct {
	ds            ds.Batching
	pathPrefixMap map[string]string // filepath -> filePrefix, need to persist
	sync.Mutex
}

// CorruptReferenceError implements the error interface.
// It is used to indicate that the block contents pointed
// by the referencing blocks cannot be retrieved (i.e. the
// file is not found, or the data changed as it was being read).
type CorruptReferenceError struct {
	Code Status
	Err  error
}

// Error() returns the error message in the CorruptReferenceError
// as a string.
func (c CorruptReferenceError) Error() string {
	return c.Err.Error()
}

// NewFileManager initializes a new file manager with the given
// datastore and root. All FilestoreNodes paths are relative to the
// root path given here, which is prepended for any operations.
func NewFileManager(ds ds.Batching) *FileManager {
	return &FileManager{ds: dsns.Wrap(ds, FilestorePrefix), pathPrefixMap: make(map[string]string)}
}

// AllKeysChan returns a channel from which to read the keys stored in
// the FileManager. If the given context is cancelled the channel will be
// closed.
func (f *FileManager) AllKeysChan(ctx context.Context) (<-chan *cid.Cid, error) {
	q := dsq.Query{KeysOnly: true}

	res, err := f.ds.Query(q)
	if err != nil {
		return nil, err
	}

	out := make(chan *cid.Cid, dsq.KeysOnlyBufSize)
	go func() {
		defer close(out)
		for {
			v, ok := res.NextSync()
			if !ok {
				return
			}

			k := ds.RawKey(v.Key)
			c, err := dshelp.DsKeyToCid(k)
			if err != nil {
				log.Errorf("decoding cid from filestore: %s", err)
				continue
			}

			select {
			case out <- c:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

// DeleteBlock deletes the reference-block from the underlying
// datastore. It does not touch the referenced data.
func (f *FileManager) DeleteBlock(c *cid.Cid) error {
	err := f.ds.Delete(dshelp.CidToDsKey(c))
	if err == ds.ErrNotFound {
		return blockstore.ErrNotFound
	}
	return err
}

// Get reads a block from the datastore. Reading a block
// is done in two steps: the first step retrieves the reference
// block from the datastore. The second step uses the stored
// path and offsets to read the raw block data directly from disk.
func (f *FileManager) Get(c *cid.Cid) (blocks.Block, error) {
	dobj, err := f.getDataObj(c)
	if err != nil {
		return nil, err
	}

	out, err := f.readDataObj(c, dobj)
	if err != nil {
		return nil, err
	}

	return blocks.NewBlockWithCid(out, c)
}

func (f *FileManager) getDataObj(c *cid.Cid) (*pb.DataObj, error) {
	o, err := f.ds.Get(dshelp.CidToDsKey(c))
	switch err {
	case ds.ErrNotFound:
		return nil, blockstore.ErrNotFound
	default:
		return nil, err
	case nil:
		//
	}

	return unmarshalDataObj(o)
}

// find one valid DataObj and return
func unmarshalDataObj(o interface{}) (*pb.DataObj, error) {
	dobjs, err := unmarshalDataObjs(o)
	if err != nil {
		return nil, err
	}

	for _, d := range dobjs {
		abspath := filepath.FromSlash(d.GetFilePath())
		_, err := os.Stat(abspath)
		if err == nil {
			return d, nil
		}
	}

	return nil, fmt.Errorf("no valid data objcet found")
}

func (f *FileManager) getDataObjs(c *cid.Cid) ([]*pb.DataObj, error) {
	o, err := f.ds.Get(dshelp.CidToDsKey(c))
	switch err {
	case ds.ErrNotFound:
		return nil, blockstore.ErrNotFound
	default:
		return nil, err
	case nil:
		//
	}

	return unmarshalDataObjs(o)
}

func unmarshalDataObjs(o interface{}) ([]*pb.DataObj, error) {
	data, ok := o.([]byte)
	if !ok {
		return nil, fmt.Errorf("stored filestore dataobj was not a []byte")
	}

	var dobjs pb.DataObjs

	if err := proto.Unmarshal(data, &dobjs); err != nil {
		return nil, err
	}
	return dobjs.GetObjects(), nil
}

// reads and verifies the block
func (f *FileManager) readDataObj(c *cid.Cid, d *pb.DataObj) ([]byte, error) {
	abspath := filepath.FromSlash(d.GetFilePath())

	fi, err := os.Open(abspath)
	if os.IsNotExist(err) {
		return nil, &CorruptReferenceError{StatusFileNotFound, err}
	} else if err != nil {
		return nil, &CorruptReferenceError{StatusFileError, err}
	}
	defer fi.Close()

	// calculate the offset taking into account the prefix
	offset := d.GetOffset()
	size := d.GetSize_()
	f.Lock()
	defer f.Unlock()
	prefix, ok := f.pathPrefixMap[abspath]
	if !ok {
		return nil, fmt.Errorf("error get prefix for file : %s", abspath)
	}
	if offset == 0 {
		size = size - uint64(len(prefix))
	} else {
		offset = offset - uint64(len(prefix))
	}

	_, err = fi.Seek(int64(offset), io.SeekStart)
	if err != nil {
		return nil, &CorruptReferenceError{StatusFileError, err}
	}

	//outbuf := make([]byte, d.GetSize_())
	outbuf := mpool.ByteSlicePool.Get(uint32(d.GetSize_())).([]byte)[:d.GetSize_()]
	if offset == 0 {
		buf := make([]byte, size)
		_, err = io.ReadFull(fi, buf)

		copy(outbuf[0:len(prefix)], []byte(prefix))
		copy(outbuf[len(prefix):], buf)
	} else {
		_, err = io.ReadFull(fi, outbuf)
	}
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return nil, &CorruptReferenceError{StatusFileChanged, err}
	} else if err != nil {
		return nil, &CorruptReferenceError{StatusFileError, err}
	}

	outcid, err := c.Prefix().Sum(outbuf)
	if err != nil {
		return nil, err
	}

	if !c.Equals(outcid) {
		return nil, &CorruptReferenceError{StatusFileChanged,
			fmt.Errorf("data in file did not match. %s offset %d", d.GetFilePath(), d.GetOffset())}
	}

	return outbuf, nil
}

// Has returns if the FileManager is storing a block reference. It does not
// validate the data, nor checks if the reference is valid.
func (f *FileManager) Has(c *cid.Cid) (bool, error) {
	// NOTE: interesting thing to consider. Has doesnt validate the data.
	// So the data on disk could be invalid, and we could think we have it.
	dsk := dshelp.CidToDsKey(c)
	return f.ds.Has(dsk)
}

type putter interface {
	Put(ds.Key, interface{}) error
}

// Put adds a new reference block to the FileManager. It does not check
// that the reference is valid.
func (f *FileManager) Put(b *posinfo.FilestoreNode) error {
	return f.putTo(b, f.ds)
}

func (f *FileManager) putTo(b *posinfo.FilestoreNode, to putter) error {
	var dobj pb.DataObj

	// the filepaht.HasPrefix is depreciated, need to check for the case eg, root: ./fs, fullpath ./fstext.txt
	// need to gurantee the fullpath and root is absolute path
	/*
			if !filepath.HasPrefix(b.PosInfo.FullPath, f.root) || (b.PosInfo.FullPath[len(f.root)] != filepath.Separator && f.root[len(f.root)-1] != filepath.Separator) {
				return fmt.Errorf("cannot add filestore references outside ont-ipfs root (%s)", f.root)
			}

		p, err := filepath.Rel(f.root, b.PosInfo.FullPath)
		if err != nil {
			return err
		}
	*/

	abspath, err := filepath.Abs(b.PosInfo.FullPath)
	if err != nil {
		return err
	}

	// check if same path already exist
	dobjs, err := f.getDataObjs(b.Cid())
	if err != nil && err != blockstore.ErrNotFound {
		return fmt.Errorf("get data objects error : %s", err)
	}

	for _, d := range dobjs {
		// path exist, no need to store
		if filepath.ToSlash(abspath) == filepath.ToSlash(d.GetFilePath()) {
			return nil
		}
	}

	dobj.FilePath = filepath.ToSlash(abspath)
	dobj.Offset = b.PosInfo.Offset
	dobj.Size_ = uint64(len(b.RawData()))

	var objs []*pb.DataObj

	if dobjs != nil {
		objs = append(objs, dobjs[:]...)
	}
	objs = append(objs, &dobj)

	data, err := proto.Marshal(&pb.DataObjs{objs})
	if err != nil {
		return err
	}

	return to.Put(dshelp.CidToDsKey(b.Cid()), data)
}

// PutMany is like Put() but takes a slice of blocks instead,
// allowing it to create a batch transaction.
func (f *FileManager) PutMany(bs []*posinfo.FilestoreNode) error {
	batch, err := f.ds.Batch()
	if err != nil {
		return err
	}

	for _, b := range bs {
		if err := f.putTo(b, batch); err != nil {
			return err
		}
	}

	return batch.Commit()
}

func (f *FileManager) SetPrefix(path string, prefix string) {
	f.Lock()
	defer f.Unlock()
	f.pathPrefixMap[path] = prefix
}

func (f *FileManager) GetPrefix(path string) string {
	f.Lock()
	defer f.Unlock()

	prefix, ok := f.pathPrefixMap[path]
	if ok {
		return prefix
	} else {
		return ""
	}
}
