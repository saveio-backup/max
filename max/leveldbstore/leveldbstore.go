package leveldbstore

import (
	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/storage"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var ErrNotFound = errors.New("leveldbstore: data not found")

type LevelDBStore struct {
	db    *leveldb.DB // LevelDB instance
	batch *leveldb.Batch
}

// used to compute the size of bloom filter bits array
// too small will lead to high positive rate.
const BITSPERKEY = 10

// NewLevelDBStore return LevelDBStore instance
func NewLevelDBStore(file string) (*LevelDBStore, error) {
	openFileCache := opt.DefaultOpenFilesCacheCapacity
	maxOpenFiles, err := fdlimit.Current()
	if err == nil && maxOpenFiles < openFileCache*5 {
		openFileCache = maxOpenFiles / 5
	}

	if openFileCache < 16 {
		openFileCache = 16
	}

	// default Options
	o := opt.Options{
		NoSync:                 false,
		OpenFilesCacheCapacity: openFileCache,
		Filter:                 filter.NewBloomFilter(BITSPERKEY),
	}

	db, err := leveldb.OpenFile(file, &o)

	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		db, err = leveldb.RecoverFile(file, nil)
	}

	if err != nil {
		return nil, err
	}

	return &LevelDBStore{
		db:    db,
		batch: nil,
	}, nil
}

func NewMemLevelDBStore() (*LevelDBStore, error) {
	store := storage.NewMemStorage()
	// default Options
	o := opt.Options{
		NoSync: false,
		Filter: filter.NewBloomFilter(BITSPERKEY),
	}

	db, err := leveldb.Open(store, &o)
	if err != nil {
		return nil, err
	}

	return &LevelDBStore{
		db:    db,
		batch: nil,
	}, nil
}

//Put a key-value pair to leveldb
func (this *LevelDBStore) Put(key []byte, value []byte) error {
	return this.db.Put(key, value, nil)
}

//Get the value of a key from leveldb
func (this *LevelDBStore) Get(key []byte) ([]byte, error) {
	data, err := this.db.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

//Has return whether the key exist in leveldb
func (this *LevelDBStore) Has(key []byte) (bool, error) {
	return this.db.Has(key, nil)
}

//Delete the data with the key in leveldb
func (this *LevelDBStore) Delete(key []byte) error {
	return this.db.Delete(key, nil)
}

//NewBatch start commit batch
func (this *LevelDBStore) NewBatch() {
	this.batch = new(leveldb.Batch)
}

//BatchPut put a key-value pair to leveldb batch
func (this *LevelDBStore) BatchPut(key []byte, value []byte) {
	this.batch.Put(key, value)
}

//BatchDelete delete a key to leveldb batch
func (this *LevelDBStore) BatchDelete(key []byte) {
	this.batch.Delete(key)
}

//BatchCommit commit batch to leveldb
func (this *LevelDBStore) BatchCommit() error {
	err := this.db.Write(this.batch, nil)
	if err != nil {
		return err
	}
	this.batch = nil
	return nil
}

//Close leveldb
func (this *LevelDBStore) Close() error {
	return this.db.Close()
}

//Store iterator for iterate store
type StoreIterator interface {
	Next() bool    //Next item. If item available return true, otherwise return false
	First() bool   //First item. If item available return true, otherwise return false
	Key() []byte   //Return the current item key
	Value() []byte //Return the current item value
	Release()      //Close iterator
	Error() error  // Error returns any accumulated error.
}

//NewIterator return a iterator of leveldb with the key prefix
func (this *LevelDBStore) NewIterator(prefix []byte) StoreIterator {
	return this.db.NewIterator(util.BytesPrefix(prefix), nil)
}
