package dbstore

import (
	"errors"
	"sync"
	"sync/atomic"

	//logging "gx/ipfs/QmRb5jh8z2E8hMGN2tkvs1yHynUanqnZ3UeKwgN1i9P1F8/go-log"
	"github.com/saveio/themis/common/log"

	ds "gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore"
	dsns "gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore/namespace"
	query "gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore/query"
)

//var log = logging.Logger("dbstore")

var ErrValueTypeMismatch = errors.New("the retrieved value is not match")

var ErrHashMismatch = errors.New("data in storage has different hash than requested")

var ErrNotFound = errors.New("dbstore: data not found")

var DBStorePrefix = ds.NewKey("dbstore")

type DBStore interface {
	Delete(string) error
	Has(string) (bool, error)
	Get(string) ([]byte, error)
	Put(string, []byte) error
	Query(q query.Query) (query.Results, error)
}

type GCLocker interface {
	GCLock() Unlocker

	PinLock() Unlocker

	GCRequested() bool
}

type GCDBStore interface {
	DBStore
	GCLocker
}

func NewGCDBStore(dbs DBStore, gcl GCLocker) GCDBStore {
	return gcDBstore{dbs, gcl}
}

type gcDBstore struct {
	DBStore
	GCLocker
}

func NewDBstore(d ds.Batching) DBStore {
	var dsb ds.Batching
	dd := dsns.Wrap(d, DBStorePrefix)
	dsb = dd
	return &dbstore{
		datastore: dsb,
	}
}

type dbstore struct {
	datastore ds.Batching
}

func (dbs *dbstore) Get(k string) ([]byte, error) {
	if len(k) == 0 {
		log.Error("nil key")
		return nil, ErrNotFound
	}

	maybeData, err := dbs.datastore.Get(ds.NewKey(k))
	if err == ds.ErrNotFound {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	bdata, ok := maybeData.([]byte)
	if !ok {
		return nil, ErrValueTypeMismatch
	}
	return bdata, nil
}

// Put a data into storage
func (dbs *dbstore) Put(key string, data []byte) error {
	k := ds.NewKey(key)
	return dbs.datastore.Put(k, data)
}

func (dbs *dbstore) Has(k string) (bool, error) {
	return dbs.datastore.Has(ds.NewKey(k))
}

func (dbs *dbstore) Delete(k string) error {
	err := dbs.datastore.Delete(ds.NewKey(k))
	if err == ds.ErrNotFound {
		return ErrNotFound
	}
	return err
}

func (dbs *dbstore) Query(q query.Query) (query.Results, error) {
	return dbs.datastore.Query(q)
}

// NewGCLocker returns a default implementation of
// GCLocker using standard [RW] mutexes.
func NewGCLocker() GCLocker {
	return &gclocker{}
}

type gclocker struct {
	lk    sync.RWMutex
	gcreq int32
}

// Unlocker represents an object which can Unlock
// something.
type Unlocker interface {
	Unlock()
}

type unlocker struct {
	unlock func()
}

func (u *unlocker) Unlock() {
	u.unlock()
	u.unlock = nil // ensure its not called twice
}

func (bs *gclocker) GCLock() Unlocker {
	atomic.AddInt32(&bs.gcreq, 1)
	bs.lk.Lock()
	atomic.AddInt32(&bs.gcreq, -1)
	return &unlocker{bs.lk.Unlock}
}

func (bs *gclocker) PinLock() Unlocker {
	bs.lk.RLock()
	return &unlocker{bs.lk.RUnlock}
}

func (bs *gclocker) GCRequested() bool {
	return atomic.LoadInt32(&bs.gcreq) > 0
}
