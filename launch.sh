#!/bin/bash
set -e

cd $(dirname "$0")

# Clean up any existing kv server processes
pkill -f "go run ./cmd/kv" || true
pkill -f "./kvserver" || true
sleep 0.2


mkdir -p data

# Hyderabad
go run ./cmd/kv -db-location=data/hyderabad.db -http-addr=127.0.0.2:8080 -config-file=sharding.toml -shard=Hyderabad &
go run ./cmd/kv -db-location=data/hyderabad-r.db -http-addr=127.0.0.22:8080 -config-file=sharding.toml -shard=Hyderabad -replica &

# Bangalore
go run ./cmd/kv -db-location=data/bangalore.db -http-addr=127.0.0.3:8080 -config-file=sharding.toml -shard=Bangalore &
go run ./cmd/kv -db-location=data/bangalore-r.db -http-addr=127.0.0.33:8080 -config-file=sharding.toml -shard=Bangalore -replica &

# Mumbai
go run ./cmd/kv -db-location=data/mumbai.db -http-addr=127.0.0.4:8080 -config-file=sharding.toml -shard=Mumbai &
go run ./cmd/kv -db-location=data/mumbai-r.db -http-addr=127.0.0.44:8080 -config-file=sharding.toml -shard=Mumbai -replica &

# Delhi
go run ./cmd/kv -db-location=data/delhi.db -http-addr=127.0.0.5:8080 -config-file=sharding.toml -shard=Delhi &
go run ./cmd/kv -db-location=data/delhi-r.db -http-addr=127.0.0.55:8080 -config-file=sharding.toml -shard=Delhi -replica &

wait


