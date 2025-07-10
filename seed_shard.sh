#!/bin/bash

for shard in 127.0.0.2:8080; do
    echo "Seeding shard $shard"
    for i in {1...10000}; do
        curl "http://$shard/set?key=key-$RANDOM&value=value-$RANDOM"
    done
done