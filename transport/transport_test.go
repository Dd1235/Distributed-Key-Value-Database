package transport_test

import (
	"fmt"
	"io"
	"kv/config"
	"kv/db"
	"kv/transport"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// createShardDB creates a new shard database for testing.
func createShardDB(t *testing.T, idx int) *db.Database {
	t.Helper() // tells go that this is a helper, if there is a failure, it will show the line number of the test that called this function

	dbPath := fmt.Sprintf("%s/db%d.bolt", t.TempDir(), idx)
	database, closeFunc, err := db.NewDatabase(dbPath, false) // create new database that is not read only
	require.NoError(t, err)
	// registers clean up function after the test ends
	t.Cleanup(func() {
		closeFunc()
	})

	return database
}

// initializes a shard server with a database and shard configuration.
func createShardServer(t *testing.T, idx int, addrs map[int]string) (*db.Database, *transport.Server) {
	t.Helper()

	db := createShardDB(t, idx)
	shards := &config.Shards{
		Addrs:  addrs,
		Count:  len(addrs),
		CurIdx: idx,
	}

	server := transport.NewServer(db, shards)
	return db, server
}

func TestWebServer_ShardsAndRedirect(t *testing.T) {
	var ts1GetHandler, ts1SetHandler func(http.ResponseWriter, *http.Request)
	var ts2GetHandler, ts2SetHandler func(http.ResponseWriter, *http.Request)

	ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.RequestURI, "/get"):
			ts1GetHandler(w, r)
		case strings.HasPrefix(r.RequestURI, "/set"):
			ts1SetHandler(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts1.Close()

	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.RequestURI, "/get"):
			ts2GetHandler(w, r)
		case strings.HasPrefix(r.RequestURI, "/set"):
			ts2SetHandler(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts2.Close()

	addrs := map[int]string{
		0: strings.TrimPrefix(ts1.URL, "http://"),
		1: strings.TrimPrefix(ts2.URL, "http://"),
	}

	db1, srv1 := createShardServer(t, 0, addrs)
	db2, srv2 := createShardServer(t, 1, addrs)

	keys := map[string]int{
		"Hyd": 0,
		"Blr": 1,
	}

	ts1GetHandler = srv1.GetHandler
	ts1SetHandler = srv1.SetHandler
	ts2GetHandler = srv2.GetHandler
	ts2SetHandler = srv2.SetHandler

	// Set keys via shard 0 (ts1), expect redirect to correct shard
	for key := range keys {
		url := fmt.Sprintf("%s/set?key=%s&value=value-%s", ts1.URL, key, key)
		resp, err := http.Get(url)
		require.NoError(t, err)
		resp.Body.Close()
	}

	// Get keys via shard 0, expect correct values via redirect
	for key := range keys {
		url := fmt.Sprintf("%s/get?key=%s", ts1.URL, key)
		resp, err := http.Get(url)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		require.NoError(t, err)

		expected := fmt.Sprintf("value-%s", key)
		require.Contains(t, string(body), expected, "Unexpected body for key %q", key)

		log.Printf("Fetched %q => %s", key, string(body))
	}

	// Direct DB validation (no HTTP)
	valHyd, err := db1.GetKey("Hyd")
	require.NoError(t, err)
	require.Equal(t, []byte("value-Hyd"), valHyd)

	valBlr, err := db2.GetKey("Blr")
	require.NoError(t, err)
	require.Equal(t, []byte("value-Blr"), valBlr)
}
