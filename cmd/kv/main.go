package main

import (
	"flag"
	"log"
	"net/http"

	"kv/config"
	"kv/db"
	"kv/replication"
	"kv/transport"
)

// command line flags
var (
	dbLocation = flag.String("db-location", "", "Path to the BoltDB file for this shard")
	httpAddr   = flag.String("http-addr", "127.0.0.1:8080", "Address this HTTP server should listen on")
	configFile = flag.String("config-file", "sharding.toml", "Path to the TOML config defining all shards")
	shardName  = flag.String("shard", "", "Name of the current shard (must match one in config)")
	replica    = flag.Bool("replica", false, "Run in read-only replica mode (pull from leader)")
)

func parseFlags() {
	flag.Parse()

	if *dbLocation == "" {
		log.Fatalf("Must provide --db-location")
	}
	if *shardName == "" {
		log.Fatalf("Must provide --shard (e.g. Hyderabad, Bangalore)")
	}
}

func main() {
	parseFlags()

	// Parse TOML file containing all shard configs
	cfg, err := config.ParseFile(*configFile)
	if err != nil {
		log.Fatalf("Error parsing config file %q: %v", *configFile, err)
	}

	// Extract current shard's index, address, and global shard map
	shards, err := config.ParseShards(cfg.Shards, *shardName)
	if err != nil {
		log.Fatalf("Error parsing shard metadata: %v", err)
	}

	log.Printf("Loaded shard config: %q (Index: %d) | Total shards: %d", *shardName, shards.CurIdx, shards.Count)

	// Open BoltDB (read-only if --replica)
	dbInstance, closeFn, err := db.NewDatabase(*dbLocation, *replica)
	if err != nil {
		log.Fatalf("Failed to open DB %q: %v", *dbLocation, err)
	}
	defer closeFn()

	// If this is a replica, start replication client loop
	if *replica {
		leaderAddr, ok := shards.Addrs[shards.CurIdx]
		if !ok {
			log.Fatalf("Could not determine leader address for shard %d", shards.CurIdx)
		}
		log.Printf("Running in replica mode — syncing from leader %q", leaderAddr)
		go replication.ClientLoop(dbInstance, leaderAddr)
	}

	shorthand := map[string]string{
		"Hyderabad": "Hyd",
		"Bangalore": "Blr",
		"Mumbai":    "Bom",
		"Delhi":     "Del",
	}

	label := shorthand[*shardName]
	if label == "" {
		label = *shardName
	}
	if *replica {
		label += " Replica"
	}

	// Create HTTP server handlers
	srv := transport.NewServer(dbInstance, shards, label)

	http.HandleFunc("/get", srv.GetHandler)
	http.HandleFunc("/set", srv.SetHandler)
	http.HandleFunc("/purge", srv.DeleteExtraKeysHandler)
	http.HandleFunc("/next-replication-key", srv.GetNextKeyForReplication)
	http.HandleFunc("/delete-replication-key", srv.DeleteReplicationKey)

	log.Printf("Serving on http://%s ...", *httpAddr)
	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}
