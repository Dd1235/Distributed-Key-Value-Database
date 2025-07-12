# Distributed Key-Value Database

This project implements a distributed key-value database in Go, featuring sharding, and replication.

## System Overview

This distributed key-value database implements a horizontally scalable architecture with the following key components and concepts:

### Architecture Overview

The system consists of **4 shards** (Hyderabad, Bangalore, Mumbai, Delhi), each with a **leader-replica pair** for high availability. Data is automatically partitioned across shards using consistent hashing, and each shard maintains its own local database with asynchronous replication to its replica. Each shard is a standalone key-value server that stores data in a BoltDB file, listens on a specific address, and optionally runs as a replica (polls the leader). Replicas are read only, this is just for safety incase there are multiple replicas and you don't write to both at the same time.

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Hyderabad     │    │   Bangalore     │    │     Mumbai      │    │      Delhi      │
│                 │    │                 │    │                 │    │                 │
│ Leader: 127.0.0.2   │ Leader: 127.0.0.3   │ Leader: 127.0.0.4   │ Leader: 127.0.0.5   │
│ Replica: 127.0.0.22  │ Replica: 127.0.0.33  │ Replica: 127.0.0.44  │ Replica: 127.0.0.55  │
└─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Key Concepts

#### 1. **Sharding Strategy**

- **Consistent Hashing**: Uses FNV-64 hash function to deterministically map keys to shards
- **Key Distribution**: `shard_index = hash(key) % total_shards`
- **Automatic Routing**: Requests are automatically redirected to the correct shard
- **Load Balancing**: Keys are evenly distributed across all shards

#### 2. **Replication Model**

- **Leader-Replica Pattern**: Each shard has one leader and one replica
- **Asynchronous Replication**: Pull-based replication from leader to replica
- **Eventually Consistent**: Replicas may lag behind leaders but will eventually catch up
- **Fault Tolerance**: If a leader fails, the replica can take over (manual failover required)

#### 3. **Data Storage**

- **BoltDB**: Embedded key-value store using B+ tree for efficient range queries
- **Dual Buckets**:
  - `default` bucket: Main data storage
  - `replication` bucket: Queue for pending replication operations
- **ACID Transactions**: All operations are atomic and consistent

#### 4. **Communication Protocol**

- **HTTP API**: RESTful interface for client operations
- **Custom Binary Protocol**: Internal communication between nodes
- **JSON Encoding**: For replication coordination messages

### Data Flow Examples

#### Write Operation (SET)

```
1. Client sends: PUT /set?key=user:123&value=john_doe
2. System hashes "user:123" → determines shard (e.g., Bangalore)
3. Request routed to Bangalore leader (127.0.0.3:8080)
4. Leader writes to local BoltDB (both default and replication buckets)
5. Replica (127.0.0.33:8080) polls leader for replication queue
6. Replica applies change and acknowledges deletion from queue
```

#### Read Operation (GET)

```
1. Client sends: GET /get?key=user:123
2. System hashes "user:123" → determines shard (e.g., Bangalore)
3. Request routed to Bangalore leader (127.0.0.3:8080)
4. Leader reads from local BoltDB and returns value
```

#### Cross-Shard Request

```
1. Client sends: GET /get?key=user:456 to Hyderabad (127.0.0.2:8080)
2. Hyderabad hashes "user:456" → determines it belongs to Mumbai
3. Hyderabad redirects request to Mumbai (127.0.0.4:8080)
4. Mumbai processes request and returns response
```

### Replication Process

The replication system uses a **pull-based model** with the following steps:

1. **Leader writes** data to both `default` and `replication` buckets
2. **Replica polls** leader every 100ms for new replication keys
3. **Leader responds** with next key-value pair from replication queue
4. **Replica applies** the change to its local database
5. **Replica acknowledges** successful replication by requesting key deletion
6. **Leader removes** the key from replication queue

### Configuration

The system is configured via `sharding.toml`:

```toml
[[shards]]
name = "Hyderabad"
idx = 0
address = "127.0.0.2:8080"
replicas = ["127.0.0.22:8080"]
```

### Performance Characteristics

- **Horizontal Scaling**: Add more shards to increase capacity
- **High Availability**: Replica nodes provide fault tolerance
- **Low Latency**: Local reads and writes within each shard
- **Eventually Consistent**: Replicas may have slight data lag
- **No Cross-Shard Transactions**: Each shard operates independently

### Limitations and Considerations

- **Single Point of Failure**: No automatic leader election
- **Manual Failover**: Replica promotion requires manual intervention
- **No Cross-Shard Transactions**: Cannot guarantee consistency across shards
- **Fixed Shard Count**: Adding/removing shards requires rehashing all data
- **Network Partition Handling**: Limited handling of network splits

## Features

- **Sharding:** Data is partitioned across multiple shards, allowing for horizontal scaling and improved performance.
- **Replication:** Each shard has a leader and a set of replicas, ensuring data redundancy and high availability.
- **Custom Binary Protocol:** A custom binary protocol is used for efficient data transfer between nodes.
- **HTTP API:** A simple HTTP API is provided for setting, getting, and deleting keys.
- **Benchmarking Tool:** A benchmarking tool is included for performance testing.

## Getting Started

### Prerequisites

- Go 1.18 or higher

### Installation

0.

Ensure you have Go installed on your machine.
`echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc`
`source ~/.zshrc`
`go install ./cmd/kv`
`go install ./cmd/benchclient`
`which kv` # should show the path to the binary

```bash
kv             ← used by launch.sh (server)
benchclient    ← used for load testing
```

You might need to do this

```bash
sudo ifconfig lo0 alias 127.0.0.2
sudo ifconfig lo0 alias 127.0.0.3
sudo ifconfig lo0 alias 127.0.0.4
sudo ifconfig lo0 alias 127.0.0.5
sudo ifconfig lo0 alias 127.0.0.22
sudo ifconfig lo0 alias 127.0.0.33
sudo ifconfig lo0 alias 127.0.0.44
sudo ifconfig lo0 alias 127.0.0.55
```

and

clean up ports incase of existing processes

```bash
for ip in 2 22 3 33 4 44 5 55; do lsof -nP -i@127.0.0.$ip:8080 | awk 'NR>1 {print $2}' | xargs kill -9 2>/dev/null || true; done

```

You can play around running a single leader like this

```bash
go run ./cmd/kv \
  -db-location=data/hyderabad.db \
  -http-addr=127.0.0.2:8080 \
  -config-file=sharding.toml \
  -shard=Hyderabad
```

1.  Clone the repository:

    ```bash
    git clone https://github.com/dd1235/distributed-key-value-database.git
    ```

2.  Install the dependencies:

    ```bash
    go mod tidy
    ```

### Usage

1.  Launch the database:

    ```bash
    ./launch.sh
    ```

2.  Seed the database with data:

    ```bash
    ./seed_shard.sh
    ```

3.  Interact with the database using the HTTP API:

    ```bash
    # Set a key
    curl "http://127.0.0.2:8080/set?key=my-key&value=my-value"

    # Get a key
    curl "http://127.0.0.2:8080/get?key=my-key"
    ```

## Project Structure

```
.
├── cmd
│   └── benchclient
│       └── main.go
|   └── kv
│       └── main.go
├── config
│   ├── config.go
│   └── config_test.go
├── db
│   ├── db.go
│   └── db_test.go
├── replication
│   └── replication.go
├── transport
│   ├── transport.go
│   └── transport_test.go
├── .gitignore
├── go.mod
├── go.sum
├── launch.sh
├── main.go
├── notes.md
├── readme.md
├── seed_shard.sh
└── sharding.toml
```
