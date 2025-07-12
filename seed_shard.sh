#!/bin/bash

for shard in 127.0.0.2:8080; do
  echo "Seeding shard $shard"
  for i in $(seq 1 10); do
    k="key-$((RANDOM*RANDOM))"
    curl -s "http://$shard/set?key=$k&value=value-$i" > /dev/null
  done
done
