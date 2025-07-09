package db

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*Database, func()) {
	_ = os.Remove("test.db")
	// create test db
	db, closeFunc, err := NewDatabase("test.db", false)
	require.NoError(t, err)

	return db, func() {
		closeFunc()
		_ = os.Remove("test.db")
	}
}

// require will require it to not have errors, code wont continue running
func TestSetGetKey(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := db.SetKey("hello", []byte("world"))
	require.NoError(t, err)

	val, err := db.GetKey("hello")
	require.NoError(t, err)
	require.Equal(t, []byte("world"), val)
}

// t *testing.T is test runners handle
func TestReplicationQueue(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	key := "replicaKey"
	value := []byte("some data")

	// Write key
	err := db.SetKey(key, value)
	require.NoError(t, err)

	// Check replication queue
	k, v, err := db.GetNextKeyForReplication()
	require.NoError(t, err)
	require.Equal(t, key, string(k))
	require.Equal(t, value, v)

	// Delete from replication queue
	err = db.DeleteReplicationKey(k, v)
	require.NoError(t, err)

	// Now it should be gone
	k, v, err = db.GetNextKeyForReplication()
	require.NoError(t, err)
	require.Nil(t, k)
	require.Nil(t, v)
}

func TestReadOnlyMode(t *testing.T) {
	dbPath := "test_readonly.db"
	_ = os.Remove(dbPath)

	// Create normally first
	db, closeFunc, err := NewDatabase(dbPath, false)
	require.NoError(t, err)
	_ = db.SetKey("foo", []byte("bar"))
	closeFunc()

	// Open as read-only
	db, closeFunc, err = NewDatabase(dbPath, true)
	require.NoError(t, err)
	defer func() {
		closeFunc()
		_ = os.Remove(dbPath)
	}()
	var v []byte
	v, err = db.GetKey("foo")
	require.NoError(t, err)
	require.Equal(t, v, []byte("bar"))

	err = db.SetKey("foo", []byte("baz"))
	require.Error(t, err)
}

func TestDeleteExtraKeys(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Add 3 keys
	_ = db.SetKey("a", []byte("1"))
	_ = db.SetKey("b", []byte("2"))
	_ = db.SetKey("c", []byte("3"))

	// Delete keys not equal to "a"
	err := db.DeleteExtraKeys(func(key string) bool {
		return key != "a"
	})
	require.NoError(t, err)

	// Check only "a" remains
	v, err := db.GetKey("a")
	require.NoError(t, err)
	require.Equal(t, []byte("1"), v)

	_, err = db.GetKey("b")
	require.NoError(t, err) // Exists but should be nil
}
