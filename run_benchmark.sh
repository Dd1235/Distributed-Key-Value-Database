#!/bin/bash
set -e

echo "Cleaning up..."
pkill -f "kv" || true
sleep 0.2
rm -rf data/
mkdir -p data/

echo "Installing server binary..."
go install ./cmd/kv

echo "Installing benchmark client..."
go install ./cmd/benchclient

KV_BIN=$(which kv)
BENCH_BIN=$(which benchclient)

if [[ ! -x "$KV_BIN" || ! -x "$BENCH_BIN" ]]; then
  echo "❌ Failed to locate built binaries in \$PATH"
  exit 1
fi

echo "Launching single KV server on 127.0.0.1:8080 (Hyderabad)..."
"$KV_BIN" -db-location=data/hyd.db -http-addr=127.0.0.1:8080 -config-file=only_one_shard.toml -shard=Hyderabad &
pid=$!
sleep 1

echo "Running benchmark..."
"$BENCH_BIN" --addr=127.0.0.1:8080 --iterations=500 --read-iterations=5000 --concurrency=2

echo "Cleaning up..."
kill $pid
rm -rf data/
