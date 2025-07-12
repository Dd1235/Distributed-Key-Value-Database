package replication

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"kv/db"
	"log"
	"net/http"
	"net/url"
	"time"
)

// asynchronous pull based replication
// sync (k,v) from master/leader to a follower/replica
// increase availability or fault tolerance
// eventually consistent
// contacts the leader/master server
// requests the next key-value pair from replication queue
// apply it to local db
// inform the leader to remove the key from replication queue

// ClientLoop() continuously polls for updates from the leader
// loop() - executes one replication cycle
// deleteFromReplicationQueue() - informs the leader to delete the replicated key
// NextKeyValue - JSON struct for communication

// there is no batching here though, fetches one key at a time
// need to add validation of response body
// need auth for hitting the leader endpoint

// if replica writes successfully, and but crashes before deleting key, on next startup key is still in the queue

// there is a single point of failure with hardCoded single leader, no leader selection implemented
// essentially fetch next and ack delete

type NextKeyValue struct {
	Key   string
	Value string
	Err   error
}

type client struct {
	db         *db.Database
	leaderAddr string
}

func ClientLoop(db *db.Database, leaderAddr string) {
	if db == nil {
		log.Fatalf("replication.ClientLoop: nil database passed for leader %s", leaderAddr)
	}
	if leaderAddr == "" {
		log.Fatalf("replication.ClientLoop: empty leader address")
	}

	c := &client{db: db, leaderAddr: leaderAddr}
	for {
		present, err := c.loop()
		if err != nil {
			log.Printf("Loop error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		if !present {
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func (c *client) loop() (present bool, err error) {
	const maxRetries = 10          // Retry up to 10 times before giving up
	const retryDelay = time.Second // Wait 1s between retries

	var resp *http.Response
	for i := 0; i < maxRetries; i++ {
		resp, err = http.Get("http://" + c.leaderAddr + "/next-replication-key")
		if err != nil {
			log.Printf("Loop error: could not connect to leader at %s (attempt %d/%d): %v", c.leaderAddr, i+1, maxRetries, err)
			time.Sleep(retryDelay)
			continue
		}
		break // connection succeeded
	}
	if err != nil {
		return false, fmt.Errorf("replica failed to contact leader %s after %d retries: %w", c.leaderAddr, maxRetries, err)
	}
	defer resp.Body.Close()

	var res NextKeyValue
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return false, fmt.Errorf("failed to decode response from leader: %w", err)
	}

	// Server might return an error inside response (e.g., no more keys to replicate)
	if res.Err != nil {
		return false, fmt.Errorf("server-side error during replication: %v", res.Err)
	}

	if res.Key == "" {
		// Nothing to replicate currently
		return false, nil
	}

	if err := c.db.SetKeyOnReplica(res.Key, []byte(res.Value)); err != nil {
		return false, fmt.Errorf("failed to set key on replica: %w", err)
	}

	if err := c.deleteFromReplicationQueue(res.Key, res.Value); err != nil {
		log.Printf("Warning: DeleteKeyFromReplication failed for key %q: %v", res.Key, err)
	}

	return true, nil
}

func (c *client) deleteFromReplicationQueue(key, value string) error {
	u := url.Values{}
	u.Set("key", key)
	u.Set("value", value)

	// log.Printf("Deleting key=%q, value=%q from replication queue on %q", key, value, c.leaderAddr)

	resp, err := http.Get("http://" + c.leaderAddr + "/delete-replication-key?" + u.Encode())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if !bytes.Equal(result, []byte("ok")) {
		return errors.New(string(result))
	}

	return nil
}
