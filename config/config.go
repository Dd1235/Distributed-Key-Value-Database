package config

import (
	"fmt"
	"hash/fnv"

	"github.com/BurntSushi/toml"
)

// the sharding.toml matches thi structure
// shard describes a shard that holds the appropriate set of keys
type Shard struct {
	Name    string
	Idx     int
	Address string
}

// all the shards
type Config struct {
	Shards []Shard
}

// all the [[shard]] blocks fill the Shards slice
func ParseFile(filename string) (Config, error) {
	var c Config
	// automatically decode the toml file into the config struct
	if _, err := toml.DecodeFile(filename, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// run time friendly, total number of shards, the current shard, and a map of shard index to address
type Shards struct {
	Count  int
	CurIdx int // which shard this machine is
	Addrs  map[int]string
}

func ParseShards(shards []Shard, curShardName string) (*Shards, error) {
	shardCount := len(shards)
	shardIdx := -1
	addrs := make(map[int]string)

	for _, s := range shards {
		if _, ok := addrs[s.Idx]; ok {
			return nil, fmt.Errorf("duplicate shard index: %d", s.Idx)
		}
		// map shard index to its address
		addrs[s.Idx] = s.Address
		if s.Name == curShardName {
			shardIdx = s.Idx
		}
	}

	for i := 0; i < shardCount; i++ {
		if _, ok := addrs[i]; !ok {
			return nil, fmt.Errorf("shard %d is not found", i)
		}
	}

	if shardIdx < 0 {
		return nil, fmt.Errorf("shard %q was not found", curShardName)
	}

	return &Shards{
		Addrs:  addrs,
		Count:  shardCount,
		CurIdx: shardIdx,
	}, nil
}

// Index, returns the shard number for the corresponding key.
func (s *Shards) Index(key string) int {
	h := fnv.New64()
	h.Write([]byte(key))
	return int(h.Sum64() % uint64(s.Count))
}
