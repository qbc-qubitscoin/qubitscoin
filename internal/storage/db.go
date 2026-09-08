package storage

import (
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// DB is a thin wrapper around a LevelDB instance.
// All higher-level stores (BlockStore, StateStore) embed a *DB.
type DB struct {
	db *leveldb.DB
}

// Open opens (or creates) a LevelDB database at path.
func Open(path string) (*DB, error) {
	ldb, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, fmt.Errorf("open leveldb %s: %w", path, err)
	}
	return &DB{db: ldb}, nil
}

// Close releases all held resources.
func (d *DB) Close() error { return d.db.Close() }

// Put stores key → value.
func (d *DB) Put(key, value []byte) error { return d.db.Put(key, value, nil) }

// Get returns the value for key, or (nil, nil) when the key does not exist.
func (d *DB) Get(key []byte) ([]byte, error) {
	val, err := d.db.Get(key, nil)
	if err == leveldb.ErrNotFound {
		return nil, nil
	}
	return val, err
}

// Has reports whether key exists in the database.
func (d *DB) Has(key []byte) (bool, error) { return d.db.Has(key, nil) }

// Delete removes key from the database.
func (d *DB) Delete(key []byte) error { return d.db.Delete(key, nil) }

// NewBatch returns a fresh write batch.
func (d *DB) NewBatch() *leveldb.Batch { return new(leveldb.Batch) }

// Write commits a batch atomically.
func (d *DB) Write(batch *leveldb.Batch) error { return d.db.Write(batch, nil) }

// IterPrefix calls fn for every key-value pair whose key starts with prefix.
// Iteration stops early if fn returns a non-nil error.
func (d *DB) IterPrefix(prefix []byte, fn func(key, val []byte) error) error {
	iter := d.db.NewIterator(util.BytesPrefix(prefix), nil)
	defer iter.Release()
	for iter.Next() {
		// Copy key/value — they are only valid for the life of the iterator.
		k := make([]byte, len(iter.Key()))
		v := make([]byte, len(iter.Value()))
		copy(k, iter.Key())
		copy(v, iter.Value())
		if err := fn(k, v); err != nil {
			return err
		}
	}
	return iter.Error()
}
