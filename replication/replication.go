package replication

import (
	"bytes"
	"encoding/json"
	"errors"
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
	resp, err := http.Get("http://" + c.leaderAddr + "/next-replication-key")
	var res NextKeyValue
	// json format to go struct
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if res.Err != nil {
		return false, err
	}
	if res.Key == "" {
		return false, nil
	}
	if err := c.db.SetKeyOnReplica(res.Key, []byte(res.Value)); err != nil {
		return false, err
	}
	if err := c.deleteFromReplicationQueue(res.Key, res.Value); err != nil {
		log.Printf("DeleteKeyFromReplication failed: %v", err)
	}
	return true, nil
}

func (c *client) deleteFromReplicationQueue(key, value string) error {
	u := url.Values{}
	u.Set("key", key)
	u.Set("value", value)

	log.Printf("Deleting key=%q, value=%q from replication queue on %q", key, value, c.leaderAddr)

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
