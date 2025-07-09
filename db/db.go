package db

import (
	"bytes"
	"errors"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

var defaultBucket = []byte("default")
var replicaBucket = []byte("replication")

type Database struct {
	db       *bolt.DB
	readOnly bool
}

// make a new database constructor

// return the close function is a common go idiom, give the caller full control is closing

// idiomatic go uses factory like functions hence the func NewThing(...) (*Thing, error) { ... }

func NewDatabase(dbPath string, readOnly bool) (db *Database, closeFunc func() error, err error) {
	// only owner has access, read write
	boltDb, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, nil, err
	}

	db = &Database{db: boltDb, readOnly: readOnly}
	closeFunc = boltDb.Close

	if err := db.createBuckets(); err != nil {
		closeFunc()
		return nil, nil, fmt.Errorf("creating default bucket: %w", err)
	}

	return db, closeFunc, nil
}

// pointer receiver, methods on the Database type, to createBuckets, and returns an error is creation of buckets failed
func (d *Database) createBuckets() error {

	// anonymous function initBuckets
	return d.db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(defaultBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(replicaBucket); err != nil {
			return err
		}
		return nil // success, commit the transaction
	})
}

// SetKey sets the key to the requested value into the default database or returns an error.
// []byte(key) creates a new byte slice with the same underlying content
// string is immutable, []byte is mutable
// you copy over the values

func (d *Database) SetKey(key string, value []byte) error {
	if d.readOnly {
		return errors.New("read-only mode")
	}

	return d.db.Update(func(tx *bolt.Tx) error {
		if err := tx.Bucket(defaultBucket).Put([]byte(key), value); err != nil {
			return err
		}

		return tx.Bucket(replicaBucket).Put([]byte(key), value)
	})
}

// Even after data is written to the database, it's not considered fully processed until it's delivered (replicated) — so you queue it for delivery first.

// SetKeyOnReplica sets the key to the requested value into the default database and does not write
// to the replication queue.
// This method is intended to be used only on replicas.
func (d *Database) SetKeyOnReplica(key string, value []byte) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(defaultBucket).Put([]byte(key), value)
	})
}

// defensive copying of slices
// in go, slices are references
// a := []byte("hello"); b:=a; shared memory

func copyByteSlice(b []byte) []byte {
	if b == nil {
		return nil
	}
	res := make([]byte, len(b))
	copy(res, b)
	return res
}

func (d *Database) GetNextKeyForReplication() (key, value []byte, err error) {
	err = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(replicaBucket)
		k, v := b.Cursor().First()
		key = copyByteSlice(k)
		value = copyByteSlice(v)
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return key, value, nil
}

// DeleteReplicationKey deletes the key from the replication queue
// if the value matches the contents or if the key is already absent.
// buckets are just nodes in the B+ tree, buckets are transaction scoped views of the data
func (d *Database) DeleteReplicationKey(key, value []byte) (err error) {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(replicaBucket)

		v := b.Get(key)
		if v == nil {
			return errors.New("key does not exist")
		}

		if !bytes.Equal(v, value) {
			return errors.New("value does not match")
		}

		return b.Delete(key)
	})
}

// GetKey get the value of the requested from a default database.
func (d *Database) GetKey(key string) ([]byte, error) {
	var result []byte
	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(defaultBucket)
		result = copyByteSlice(b.Get([]byte(key)))
		return nil
	})

	if err == nil {
		return result, nil
	}
	return nil, err
}

// DeleteExtraKeys deletes the keys that do not belong to this shard.
// isExtra - predicate function - tells you if it belongs to a differnt shard
// View is read only and non blocking
// now there is a problem, view and update are not in the same transaction
// handle later
func (d *Database) DeleteExtraKeys(isExtra func(string) bool) error {
	var keys []string

	// read only phase collect the keys
	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(defaultBucket)
		return b.ForEach(func(k, v []byte) error {
			ks := string(k)
			if isExtra(ks) {
				keys = append(keys, ks)
			}
			return nil
		})
	})

	if err != nil {
		return err
	}

	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(defaultBucket)

		for _, k := range keys {
			if err := b.Delete([]byte(k)); err != nil {
				return err
			}
		}
		return nil
	})
}
