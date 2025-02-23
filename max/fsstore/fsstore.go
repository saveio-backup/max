package fsstore

import (
	"encoding/json"
	"errors"
	"github.com/saveio/max/max/leveldbstore"
)

const (
	BLOCK_ATTR_PREFIX      = "blockattr:"
	BLOCK_TAG_INDEX_PREFIX = "blocktagindex:"
	FILE_PREFIX_KEY        = "fileprefix:"
	PROVE_PARAM_KEY        = "proveparam"
	FILE_BLOCKHASH_KEY     = "fileblockhash:"
	DATA_KEY               = "data:"
)

type FsStore struct {
	db *leveldbstore.LevelDBStore
}

func NewFsStore(file string) (*FsStore, error) {
	db, err := leveldbstore.NewLevelDBStore(file)
	if err != nil {
		return nil, err
	}

	return &FsStore{db}, nil
}

type BlockAttr struct {
	Index    uint64 `json:"index"`
	Hash     string `json:"hash"`
	Tag      []byte `json:"tag"`
	FileHash string `json:"filehash"`
}

func NewBlockAttr(blockHashStr, fileHashStr string, index uint64, tag []byte) *BlockAttr {
	return &BlockAttr{
		Index:    index,
		Hash:     blockHashStr,
		Tag:      tag,
		FileHash: fileHashStr,
	}
}

func (p *BlockAttr) Serialization() ([]byte, error) {
	return json.Marshal(p)
}

func (p *BlockAttr) Deserialization(raw []byte) error {
	return json.Unmarshal(raw, p)
}

type FilePrefix struct {
	Path   string `json:"path"`
	Prefix string `json:"prefix"`
}

func NewFilePrefix(path, prefix string) *FilePrefix {
	return &FilePrefix{
		Path:   path,
		Prefix: prefix,
	}
}

func (p *FilePrefix) Serialization() ([]byte, error) {
	return json.Marshal(p)
}

func (p *FilePrefix) Deserialization(raw []byte) error {
	return json.Unmarshal(raw, p)
}

type ProveParam struct {
	FileHash         string `json:"filehash"`
	FirstProveHeight uint64 `json:"firstproveheight"`
	PDPParam         []byte `json:"pdpparam"`
}

func NewProveParam(fileHashStr string, firstProveHeight uint64, pdpParam []byte) *ProveParam {
	return &ProveParam{
		FileHash:         fileHashStr,
		FirstProveHeight: firstProveHeight,
		PDPParam:         pdpParam,
	}
}

func (p *ProveParam) Serialization() ([]byte, error) {
	return json.Marshal(p)
}

func (p *ProveParam) Deserialization(raw []byte) error {
	return json.Unmarshal(raw, p)
}

type FileBlockHash struct {
	FileHash    string   `json:"filehash"`
	BlockHashes []string `json:"blockhahses"`
}

func (p *FileBlockHash) Serialization() ([]byte, error) {
	return json.Marshal(p)
}

func (p *FileBlockHash) Deserialization(raw []byte) error {
	return json.Unmarshal(raw, p)
}

func (fss *FsStore) Close() error {
	return fss.db.Close()
}

func (fss *FsStore) GetBlockAttr(key string) (*BlockAttr, error) {
	if len(key) == 0 {
		return nil, errors.New("blockattr: key is nil")
	}
	bdata, err := fss.db.Get(genBlockAttrKey(key))
	if err != nil {
		return nil, err
	}
	p := &BlockAttr{}
	err = p.Deserialization(bdata)
	if err != nil {
		return nil, errors.New("the retrieved value is not a blockattr")
	}
	return p, nil
}

// Put a BlockAttr into storage, key is blockhash-filehash
func (fss *FsStore) PutBlockAttr(key string, p *BlockAttr) error {
	if len(key) == 0 {
		return errors.New("blockattr: key is nil")
	}
	data, err := p.Serialization()
	if err != nil {
		return err
	}
	return fss.db.Put(genBlockAttrKey(key), data)
}

// DeleteBlockAttr delete a block attributes value
func (fss *FsStore) DeleteBlockAttr(key string) error {
	if len(key) == 0 {
		return errors.New("blockattr: key is nil")
	}
	err := fss.db.Delete(genBlockAttrKey(key))
	return err
}

func (fss *FsStore) GetBlockAttrsWithPrefix(prefix string) ([]*BlockAttr, error) {
	var blockAttrs []*BlockAttr

	iter := fss.db.NewIterator(genBlockAttrKey(prefix))

	for iter.Next() {
		p := &BlockAttr{}
		err := p.Deserialization(iter.Value())
		if err != nil {
			return nil, errors.New("the retrieved value is not a blockattr")
		}

		blockAttrs = append(blockAttrs, p)
	}

	return blockAttrs, nil
}

func (fss *FsStore) GetFilePrefixes() ([]*FilePrefix, error) {
	var filePrefixes []*FilePrefix

	iter := fss.db.NewIterator(genFilePrefixesKey(""))

	for iter.Next() {
		p := &FilePrefix{}
		err := p.Deserialization(iter.Value())
		if err != nil {
			return nil, errors.New("the retrieved value is not a fileprefixes")
		}

		filePrefixes = append(filePrefixes, p)
	}

	return filePrefixes, nil
}

func (fss *FsStore) PutFilePrefix(key string, p *FilePrefix) error {
	if len(key) == 0 {
		return errors.New("fileprefix: key is nil")
	}

	data, err := p.Serialization()
	if err != nil {
		return err
	}
	return fss.db.Put(genFilePrefixesKey(key), data)
}

func (fss *FsStore) GetProveParams() ([]*ProveParam, error) {
	var params []*ProveParam

	iter := fss.db.NewIterator(genProveParamKey(""))

	for iter.Next() {
		p := &ProveParam{}
		err := p.Deserialization(iter.Value())
		if err != nil {
			return nil, errors.New("the retrieved value is not a prove param")
		}

		params = append(params, p)
	}

	return params, nil
}

func (fss *FsStore) GetProveParam(key string) (*ProveParam, error) {
	if len(key) == 0 {
		return nil, errors.New("proveparam: key is nil")
	}
	bdata, err := fss.db.Get(genProveParamKey(key))
	if err != nil {
		return nil, err
	}
	p := &ProveParam{}
	err = p.Deserialization(bdata)
	if err != nil {
		return nil, errors.New("the retrieved value is not a proveparam")
	}
	return p, nil
}

func (fss *FsStore) PutProveParam(key string, p *ProveParam) error {
	if p == nil {
		return errors.New("proveparam: key is nil")
	}
	data, err := p.Serialization()
	if err != nil {
		return err
	}
	return fss.db.Put(genProveParamKey(key), data)
}

func (fss *FsStore) DeleteProveParam(key string) error {
	if len(key) == 0 {
		return errors.New("proveparam: key is nil")
	}
	err := fss.db.Delete(genProveParamKey(key))
	return err
}

func (fss *FsStore) GetFileBlockHashes(key string) (*FileBlockHash, error) {
	if len(key) == 0 {
		return nil, errors.New("fileblockhash: key is nil")
	}
	bdata, err := fss.db.Get(genFileBlockHashKey(key))
	if err != nil {
		return nil, err
	}
	p := &FileBlockHash{}
	err = p.Deserialization(bdata)
	if err != nil {
		return nil, errors.New("the retrieved value is not a fileblockhash")
	}
	return p, nil
}

// Put a BlockAttr into storage, key is blockhash-filehash
func (fss *FsStore) PutFileBlockHash(key string, p *FileBlockHash) error {
	if len(key) == 0 {
		return errors.New("fileblockhash: key is nil")
	}
	data, err := p.Serialization()
	if err != nil {
		return err
	}
	return fss.db.Put(genFileBlockHashKey(key), data)
}

// DeleteBlockAttr delete a block attributes value
func (fss *FsStore) DeleteFileBlockHash(key string) error {
	if len(key) == 0 {
		return errors.New("fileblockhash: key is nil")
	}
	err := fss.db.Delete(genFileBlockHashKey(key))
	return err
}

func (fss *FsStore) GetData(key string) ([]byte, error) {
	if len(key) == 0 {
		return nil, errors.New("get data: key is nil")
	}
	bdata, err := fss.db.Get(genDataKey(key))
	if err != nil {
		return nil, err
	}
	return bdata, nil
}

// Put a user defined data into storage
func (fss *FsStore) PutData(key string, data []byte) error {
	if len(key) == 0 {
		return errors.New("put data : key is nil")
	}
	return fss.db.Put(genDataKey(key), data)
}

// Delete user defined data from storage
func (fss *FsStore) DeleteData(key string) error {
	if len(key) == 0 {
		return errors.New("deleteData: key is nil")
	}
	return fss.db.Delete(genDataKey(key))
}

func (fss *FsStore) GetDataWithPrefix(prefix string) ([][]byte, error) {
	var out [][]byte

	iter := fss.db.NewIterator(genDataKey(prefix))

	defer iter.Release()
	for iter.Next() {
		data := make([]byte, len(iter.Value()))
		copy(data, iter.Value())
		out = append(out, data)
	}
	return out, nil
}

func genBlockAttrKey(key string) []byte {
	return ([]byte)(BLOCK_ATTR_PREFIX + key)
}

func genBlockTagIndexKey(key string) []byte {
	return ([]byte)(BLOCK_TAG_INDEX_PREFIX + key)
}

func genFilePrefixesKey(key string) []byte {
	return ([]byte)(FILE_PREFIX_KEY + key)
}

func genProveParamKey(key string) []byte {
	return ([]byte)(PROVE_PARAM_KEY + key)
}

func genFileBlockHashKey(key string) []byte {
	return ([]byte)(FILE_BLOCKHASH_KEY + key)
}

func genDataKey(key string) []byte {
	return ([]byte)(DATA_KEY + key)
}
