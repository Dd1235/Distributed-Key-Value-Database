package transport

import (
	"encoding/json"
	"fmt"
	"io"
	"kv/config"
	"kv/db"
	"kv/replication"
	"net/http"
)

type Server struct {
	db       *db.Database
	shards   *config.Shards
	serverId string // this is simply to be able to identify the server in logs
}

func NewServer(db *db.Database, s *config.Shards, id string) *Server {
	return &Server{
		db:       db,
		shards:   s,
		serverId: id,
	}
}

func (s *Server) redirect(shard int, w http.ResponseWriter, r *http.Request) {
	url := "http://" + s.shards.Addrs[shard] + r.RequestURI
	fmt.Fprintf(w, "redirecting from shard %d to shard %d (%q)\n", s.shards.CurIdx, shard, url)

	resp, err := http.Get(url)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Error redirecting the request: %v", err)
		return
	}
	defer resp.Body.Close()

	io.Copy(w, resp.Body)
}

func (s *Server) GetHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	key := r.Form.Get("key")
	shard := s.shards.Index(key)

	// fmt.Printf("➡️ GET /get?key=%s → target shard: %d | current shard: %d\n", key, shard, s.shards.CurIdx)

	if shard != s.shards.CurIdx {
		// fmt.Println("🔁 Redirecting GET request to correct shard")
		s.redirect(shard, w, r)
		return
	}

	value, err := s.db.GetKey(key)
	// fmt.Printf("✅ GET served locally: key=%s, value=%s, error=%v\n", key, value, err)

	fmt.Fprintf(w, "Shard = %d, current shard = %d, addr = %q, Value = %q, error = %v", shard, s.shards.CurIdx, s.shards.Addrs[shard], value, err)
}

func (s *Server) SetHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	key := r.Form.Get("key")
	value := r.Form.Get("value")
	shard := s.shards.Index(key)

	// fmt.Printf("➡️ PUT /set?key=%s&value=%s → target shard: %d | current shard: %d\n", key, value, shard, s.shards.CurIdx)

	if shard != s.shards.CurIdx {
		// fmt.Println("🔁 Redirecting SET request to correct shard")
		s.redirect(shard, w, r)
		return
	}

	err := s.db.SetKey(key, []byte(value))
	// fmt.Printf("✅ SET served locally: key=%s, value=%s, error=%v\n", key, value, err)

	fmt.Fprintf(w, "Error = %v, shardIdx = %d, current shard = %d", err, shard, s.shards.CurIdx)
}

func (s *Server) DeleteExtraKeysHandler(w http.ResponseWriter, r *http.Request) {
	// fmt.Printf("🧹 PURGE: Checking for foreign keys on shard %d...\n", s.shards.CurIdx)
	err := s.db.DeleteExtraKeys(func(key string) bool {
		shouldDelete := s.shards.Index(key) != s.shards.CurIdx
		// if shouldDelete {
		// 	// fmt.Printf("🗑️  Purging key=%s (belongs to shard %d)\n", key, s.shards.Index(key))
		// }
		return shouldDelete
	})
	fmt.Fprintf(w, "Error = %v", err)
}

func (s *Server) GetNextKeyForReplication(w http.ResponseWriter, r *http.Request) {
	k, v, err := s.db.GetNextKeyForReplication()
	// fmt.Printf("📤 %s: REPLICATION PULL: key=%s, value=%s, err=%v\n", s.serverId, k, v, err)
	// this gets printed a lot because of the polling, so skipping it

	enc := json.NewEncoder(w)
	enc.Encode(&replication.NextKeyValue{
		Key:   string(k),
		Value: string(v),
		Err:   err,
	})
}

func (s *Server) DeleteReplicationKey(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	key := r.Form.Get("key")
	value := r.Form.Get("value")

	// fmt.Printf("ACK: Deleting key=%s, value=%s from queue\n", key, value)

	err := s.db.DeleteReplicationKey([]byte(key), []byte(value))
	if err != nil {
		fmt.Printf("❌ REPLICATION DELETE failed: %v\n", err)
		w.WriteHeader(http.StatusExpectationFailed)
		fmt.Fprintf(w, "error: %v", err)
		return
	}

	// fmt.Println("✅ REPLICATION DELETE successful")
	fmt.Fprintf(w, "ok")
}
