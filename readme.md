# Distributed Key-Value Store

- [] implement with static sharding

- Bolt DB is a pure go key-value database stored in the filesystem, not server based like Redis or Cassandra.
- ACID compliant, supports transactions.
- All writes must occur in a read write transaction adn all reads must occur in a read transaction
- [docs](https://pkg.go.dev/go.etcd.io/bbolt)
- MVCC - multi verison concurrency control, instead of locking a row, keep multiple versions, readers see a consistent screenshot while writers create new versions. Lock free reads are great for OLAP workloads.
- Better than 2PL

- B+ trees are used like SQL (umm they fan out, like binary tree but multiple ways, linked list at the leaf and doubly linked, think depth 4, but 100 ways, data only at the leaf, insertion can split a node and increase depth, they are specialy useful when you have disc based storage, binary are great for memory, but when dealing with disc based storage, you can put one node of the B+ tree in memory, so operations happen by bringing part of it to memory, and writing back to disc etc. It is self balancing, making sure minimum number of keys per node)

- bolt uses memory mapped files, mmap, to load database into virtual memory. _single level_, just mapping no caching in between, zero-copy, read from memory as it is reading from the disk. No extra copy.

- bolt is append only for writes, and uses copy on write at page level. During a write, a new version of page is created, commits flush to disk. so no recovery needed.

`void *mmap(void *addr, size_t length, int prot, int flags, int fd, off_t offset);`

- so the files disc blocks mapped to virtual memory
- load page contents lazily based on page faults.
- bolt implements its own structure on top, with meta pages, root page node - top of B+tree,

- buckets let you use a key-value store within a key-value store, like a namespace.
- shard is a partition of your database
- shards are building blocks of horizontal scaling
- server may hold one or more shards

- use as if its a slice of bytes instead of read and write operations.
- cow, and single writer transactions

- used in initial ETCD versions
- []byte
- Go through Dynamodb, supposed to be very good (also combine this with something like my url shortener instead of aws and dynamo? can explore)

Implementation techniques;

- Sharding: split across nodes
- Replication: copy data across nodes
- Consistency strategies:
  - Strong consistency (etcd, zookeeper)
  - Eventual consistency (dynamo, cassandra)
- Membership management: eg. Gossip protocol
- Write coordination: quorum, vector clocks, CRDTs

Static Sharding:

- fix assignment of keys to shards, instead of dynamic assignment. I can already see that this is bad if there are lots of collisions, it is also possible that your servers receive a certain type of data at a certain time where one type of shard is hashed to more. But it is simple assuming that your data is uniformly Distributed

- It's kind of like saying all books that start with A - D go to cupboard 1, E - H go to cupboard 2, etc. If you have a lot of books starting with A - D, cupboard 1 will be full and cupboard 2 will be empty.
- But this is how real world analogies ish work right?

- But resharding is hard ie adding nodes. Hotspots possible.

- This contrasts with virtual node based dynamic partitioning, or auto rebalancing systems.
- eg, Zipfian workloads - this would not work.
- But I can already see that if you do know it is something like zipfian already, there would be a work around maybe? you know one shard would be overloaded, so just add more servers to that one? I don't konw how they do it irl.
- global remapping and hot partition redistribution for adding and deleting is a problem

Consistent hashing with virtual nodes - dynamo or redis
CockroadDB, Yugabyte db - dynamic Sharding

# Some scribbles, will refactor after completing project

Static Sharding

server("key") = hash("key")% (num shards) -> server number
num_shards-power of two

Resharding

Two server - mod2 = 0, mod2 = 1
For dividing intwo 2, consider mode 4. So the server that is 0mod2 will be 0mod4 and 2mod4, and the server that is 1mod2 will be 1mod4 and 3mod4.

Purge the rest
