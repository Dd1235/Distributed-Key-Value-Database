package db

import (
	bolt "go.etcd.io/bbolt"
)

var defaultBucket = []byte("default")
var replicaBucket = []byte("replication")

type Database struct {
	db       *bolt.DB
	readOnly bool
}
