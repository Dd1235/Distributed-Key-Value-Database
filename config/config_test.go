package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFile_Success(t *testing.T) {
	configFile := "test_sharding.toml"
	err := os.WriteFile(configFile, []byte(`
[[shards]]
name = "Hyderabad"
idx = 0
address = "127.0.0.2:8080"

[[shards]]
name = "Bangalore"
idx = 1
address = "127.0.0.3:8080"
`), 0644)
	require.NoError(t, err)
	defer os.Remove(configFile)

	conf, err := ParseFile(configFile)
	require.NoError(t, err)
	require.Len(t, conf.Shards, 2)
	require.Equal(t, "Hyderabad", conf.Shards[0].Name)
	require.Equal(t, 1, conf.Shards[1].Idx)
}

func TestParseShards_ValidConfig(t *testing.T) {
	shards := []Shard{
		{Name: "Hyderabad", Idx: 0, Address: "127.0.0.2:8080"},
		{Name: "Bangalore", Idx: 1, Address: "127.0.0.3:8080"},
	}

	parsed, err := ParseShards(shards, "Hyderabad")
	require.NoError(t, err)
	require.Equal(t, 2, parsed.Count)
	require.Equal(t, 0, parsed.CurIdx)
	require.Equal(t, "127.0.0.3:8080", parsed.Addrs[1])
}

func TestParseShards_DuplicateIndex(t *testing.T) {
	shards := []Shard{
		{Name: "Hyderabad", Idx: 0, Address: "127.0.0.2:8080"},
		{Name: "Bangalore", Idx: 0, Address: "127.0.0.3:8080"},
	}

	_, err := ParseShards(shards, "Hyderabad")
	require.Error(t, err)
}

func TestParseShards_MissingShard(t *testing.T) {
	shards := []Shard{
		{Name: "Hyderabad", Idx: 0, Address: "127.0.0.2:8080"},
		{Name: "Bangalore", Idx: 2, Address: "127.0.0.3:8080"},
	}

	_, err := ParseShards(shards, "Hyderabad")
	require.Error(t, err)
}

func TestParseShards_UnknownCurrentNode(t *testing.T) {
	shards := []Shard{
		{Name: "Hyderabad", Idx: 0, Address: "127.0.0.2:8080"},
		{Name: "Bangalore", Idx: 1, Address: "127.0.0.3:8080"},
	}

	_, err := ParseShards(shards, "Chennai")
	require.Error(t, err)
}

func TestShards_Index(t *testing.T) {
	s := &Shards{
		Count:  3,
		CurIdx: 1,
		Addrs: map[int]string{
			0: "127.0.0.1:8080",
			1: "127.0.0.2:8080",
			2: "127.0.0.3:8080",
		},
	}

	keys := []string{"key1", "key2", "some-longer-key", "test-key-123"}
	for _, k := range keys {
		idx := s.Index(k)
		require.True(t, idx >= 0 && idx < s.Count, "shard index out of bounds for key: %s", k)
	}
}
