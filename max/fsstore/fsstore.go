package fsstore

import (
	"encoding/json"
	"errors"
	ds "gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore"
	query "gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore/query"

	logging "gx/ipfs/QmRb5jh8z2E8hMGN2tkvs1yHynUanqnZ3UeKwgN1i9P1F8/go-log"

	"github.com/saveio/max/max/dbstore"
	"github.com/saveio/themis/common"
)

var log = logging.Logger("fsstore")

const (
	BLOCK_ATTR_PREFIX      = "blockattr:"
	BLOCK_TAG_INDEX_PREFIX = "blocktagindex:"
	FILE_PREFIX_KEY        = "fileprefix:"
	PROVE_PARAM_KEY        = "proveparam"
)

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

// TODO: using protobuf serialize
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
	FileHash         string         `json:"filehash"`
	LuckyNum         uint64         `json:"luckynum"`
	BakHeight        uint64         `json:"bakheight"`
	BakNum           uint64         `json:"baknum"`
	BrokenWalletAddr common.Address `json:"brokenwalletaddr"`
}

func NewProveParam(fileHashStr string, luckyNum, bakHeight, bakNum uint64, brokenWalletAddr common.Address) *ProveParam {
	return &ProveParam{
		FileHash:         fileHashStr,
		LuckyNum:         luckyNum,
		BakHeight:        bakHeight,
		BakNum:           bakNum,
		BrokenWalletAddr: brokenWalletAddr,
	}
}

func (p *ProveParam) Serialization() ([]byte, error) {
	return json.Marshal(p)
}

func (p *ProveParam) Deserialization(raw []byte) error {
	return json.Unmarshal(raw, p)
}

type FsStore struct {
	db dbstore.DBStore
}

func NewFsStore(d ds.Batching) *FsStore {
	dbs := dbstore.NewDBstore(d)
	db := dbstore.NewGCDBStore(dbs, dbstore.NewGCLocker())

	fss := &FsStore{}
	fss.db = db

	return fss
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

	q := query.Query{Prefix: genBlockAttrKey(prefix)}
	result, err := fss.db.Query(q)
	if err != nil {
		return nil, err
	}

	for entry := range result.Next() {
		if entry.Error != nil {
			return nil, entry.Error
		}

		if _, ok := entry.Value.([]byte); !ok {
			return nil, errors.New("value not byte slice")
		}

		p := &BlockAttr{}
		err = p.Deserialization(entry.Value.([]byte))
		if err != nil {
			return nil, errors.New("the retrieved value is not a blockattr")
		}

		blockAttrs = append(blockAttrs, p)
	}

	return blockAttrs, nil
}

func (fss *FsStore) GetFilePrefixes() ([]*FilePrefix, error) {
	var filePrefixes []*FilePrefix

	q := query.Query{Prefix: FILE_PREFIX_KEY}
	result, err := fss.db.Query(q)
	if err != nil {
		return nil, err
	}

	for entry := range result.Next() {
		if entry.Error != nil {
			return nil, entry.Error
		}

		if _, ok := entry.Value.([]byte); !ok {
			return nil, errors.New("value not byte slice")
		}

		p := &FilePrefix{}
		err = p.Deserialization(entry.Value.([]byte))
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

	q := query.Query{Prefix: PROVE_PARAM_KEY}
	result, err := fss.db.Query(q)
	if err != nil {
		return nil, err
	}

	for entry := range result.Next() {
		if entry.Error != nil {
			return nil, entry.Error
		}

		if _, ok := entry.Value.([]byte); !ok {
			return nil, errors.New("value not byte slice")
		}

		p := &ProveParam{}
		err = p.Deserialization(entry.Value.([]byte))
		if err != nil {
			return nil, errors.New("the retrieved value is not a prove param")
		}

		params = append(params, p)
	}

	return params, nil
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

func genBlockAttrKey(k string) string {
	return BLOCK_ATTR_PREFIX + k
}

func genBlockTagIndexKey(k string) string {
	return BLOCK_TAG_INDEX_PREFIX + k
}

func genFilePrefixesKey(k string) string {
	return FILE_PREFIX_KEY + k
}

func genProveParamKey(k string) string {
	return PROVE_PARAM_KEY + k
}
