#!/bin/bash
set -e

echo "Cleaning up any old servers..."
pkill -f "kv" || true
sleep 0.2
rm -rf data/
mkdir -p data/

echo "Rebuilding all Go binaries..."
go install ./cmd/kv
go install ./cmd/benchclient

KV_BIN=$(which kv)
BENCH_BIN=$(which benchclient)

if [[ ! -x "$KV_BIN" || ! -x "$BENCH_BIN" ]]; then
  echo "Could not find kv or benchclient binaries in \$PATH"
  exit 1
fi

echo "Launching all shards and replicas..."

"$KV_BIN" -db-location=data/hyd.db -http-addr=127.0.0.2:8080 -config-file=sharding.toml -shard=Hyderabad &
"$KV_BIN" -db-location=data/hyd-r.db -http-addr=127.0.0.22:8080 -config-file=sharding.toml -shard=Hyderabad -replica &

"$KV_BIN" -db-location=data/blr.db -http-addr=127.0.0.3:8080 -config-file=sharding.toml -shard=Bangalore &
"$KV_BIN" -db-location=data/blr-r.db -http-addr=127.0.0.33:8080 -config-file=sharding.toml -shard=Bangalore -replica &

"$KV_BIN" -db-location=data/bom.db -http-addr=127.0.0.4:8080 -config-file=sharding.toml -shard=Mumbai &
"$KV_BIN" -db-location=data/bom-r.db -http-addr=127.0.0.44:8080 -config-file=sharding.toml -shard=Mumbai -replica &

"$KV_BIN" -db-location=data/del.db -http-addr=127.0.0.5:8080 -config-file=sharding.toml -shard=Delhi &
"$KV_BIN" -db-location=data/del-r.db -http-addr=127.0.0.55:8080 -config-file=sharding.toml -shard=Delhi -replica &

echo "Waiting for all nodes to boot..."
sleep 2

echo "Running benchmark on Hyderabad (127.0.0.2:8080)..."
"$BENCH_BIN" --addr=127.0.0.2:8080 --iterations=1000 --read-iterations=10000 --concurrency=4

echo "Cleaning up all servers and data..."
pkill -f "kv" || true
rm -rf data/
