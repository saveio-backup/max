package badger

import (
	"bufio"
	"encoding/binary"
	"io"
	"sync"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmdKhi5wUQyV9i3GcTyfUmpfTntWjXu8DcyT9HyNbznYrn/badger/y"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmdKhi5wUQyV9i3GcTyfUmpfTntWjXu8DcyT9HyNbznYrn/badger/protos"
)

func writeTo(entry *protos.KVPair, w io.Writer) error {
	if err := binary.Write(w, binary.LittleEndian, uint64(entry.Size())); err != nil {
		return err
	}
	buf, err := entry.Marshal()
	if err != nil {
		return err
	}
	_, err = w.Write(buf)
	return err
}

// Backup dumps a protobuf-encoded list of all entries in the database into the
// given writer, that are newer than the specified version. It returns a
// timestamp indicating when the entries were dumped which can be passed into a
// later invocation to generate an incremental dump, of entries that have been
// added/modified since the last invocation of DB.Backup()
//
// This can be used to backup the data in a database at a given point in time.
func (db *DB) Backup(w io.Writer, since uint64) (uint64, error) {
	var tsNew uint64
	err := db.View(func(txn *Txn) error {
		opts := DefaultIteratorOptions
		opts.AllVersions = true
		it := txn.NewIterator(opts)
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			if item.Version() < since {
				// Ignore versions less than given timestamp
				continue
			}
			val, err := item.Value()
			if err != nil {
				return err
			}

			entry := &protos.KVPair{
				Key:       y.Copy(item.Key()),
				Value:     y.Copy(val),
				UserMeta:  []byte{item.UserMeta()},
				Version:   item.Version(),
				ExpiresAt: item.ExpiresAt(),
			}

			// Write entries to disk
			if err := writeTo(entry, w); err != nil {
				return err
			}
		}
		tsNew = txn.readTs
		return nil
	})
	return tsNew, err
}

// Load reads a protobuf-encoded list of all entries from a reader and writes
// them to the database. This can be used to restore the database from a backup
// made by calling DB.Dump().
//
// DB.Load() should be called on a database that is not running any other
// concurrent transactions while it is running.
func (db *DB) Load(r io.Reader) error {
	br := bufio.NewReaderSize(r, 16<<10)
	unmarshalBuf := make([]byte, 1<<10)
	var entries []*Entry
	var wg sync.WaitGroup
	errChan := make(chan error, 1)

	// func to check for pending error before sending off a batch for writing
	batchSetAsyncIfNoErr := func(entries []*Entry) error {
		select {
		case err := <-errChan:
			return err
		default:
			wg.Add(1)
			return db.batchSetAsync(entries, func(err error) {
				defer wg.Done()
				if err != nil {
					select {
					case errChan <- err:
					default:
					}
				}
			})
		}
	}

	for {
		var sz uint64
		err := binary.Read(br, binary.LittleEndian, &sz)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if cap(unmarshalBuf) < int(sz) {
			unmarshalBuf = make([]byte, sz)
		}

		e := &protos.KVPair{}
		if _, err = io.ReadFull(br, unmarshalBuf[:sz]); err != nil {
			return err
		}
		if err = e.Unmarshal(unmarshalBuf[:sz]); err != nil {
			return err
		}
		entries = append(entries, &Entry{
			Key:       y.KeyWithTs(e.Key, e.Version),
			Value:     e.Value,
			UserMeta:  e.UserMeta[0],
			ExpiresAt: e.ExpiresAt,
		})

		if len(entries) == 1000 {
			if err := batchSetAsyncIfNoErr(entries); err != nil {
				return err
			}
			entries = entries[:0]
		}
	}

	if len(entries) > 0 {
		if err := batchSetAsyncIfNoErr(entries); err != nil {
			return err
		}
	}

	wg.Wait()

	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}
