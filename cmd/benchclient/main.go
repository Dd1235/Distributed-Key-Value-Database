package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// ---- CLI Flags ----
var (
	serverAddr  = flag.String("addr", "localhost:8080", "Address of the HTTP server to benchmark.")
	writeIters  = flag.Int("iterations", 1000, "Number of key-value pairs to write.")
	readIters   = flag.Int("read-iterations", 100000, "Number of reads to perform.")
	concurrency = flag.Int("concurrency", 1, "Number of concurrent goroutines for write/read.")
)

// ---- Custom HTTP client with high connection reuse ----
var httpClient = &http.Client{
	Transport: &http.Transport{
		IdleConnTimeout:     60 * time.Second,
		MaxIdleConns:        300,
		MaxConnsPerHost:     300,
		MaxIdleConnsPerHost: 300,
	},
}

// benchmark runs `iter` iterations of the given function and reports timing stats.
func benchmark(name string, iter int, fn func() string) (qps float64, results []string) {
	var max, min time.Duration = 0, time.Hour
	start := time.Now()

	for i := 0; i < iter; i++ {
		iterStart := time.Now()
		result := fn()
		results = append(results, result)
		delta := time.Since(iterStart)

		if delta > max {
			max = delta
		}
		if delta < min {
			min = delta
		}
	}

	totalTime := time.Since(start)
	avg := totalTime / time.Duration(iter)
	qps = float64(iter) / totalTime.Seconds()

	fmt.Printf("→ [%s] Avg: %s | QPS: %.1f | Max: %s | Min: %s\n", name, avg, qps, max, min)
	return qps, results
}

// sendWrite performs a single /set request with a random key-value pair.
func sendWrite() string {
	key := fmt.Sprintf("key-%d", rand.Intn(1_000_000))
	value := fmt.Sprintf("value-%d", rand.Intn(1_000_000))

	query := url.Values{}
	query.Set("key", key)
	query.Set("value", value)

	resp, err := httpClient.Get("http://" + *serverAddr + "/set?" + query.Encode())
	if err != nil {
		log.Fatalf("Write failed: %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return key
}

// sendRead performs a single /get request on a randomly chosen key.
func sendRead(keys []string) string {
	key := keys[rand.Intn(len(keys))]

	query := url.Values{}
	query.Set("key", key)

	resp, err := httpClient.Get("http://" + *serverAddr + "/get?" + query.Encode())
	if err != nil {
		log.Fatalf("Read failed: %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return key
}

// benchmarkWrite performs concurrent /set operations and returns all keys written.
func benchmarkWrite() []string {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allKeys []string
	var totalQPS float64

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			qps, keys := benchmark("write", *writeIters, sendWrite)
			mu.Lock()
			totalQPS += qps
			allKeys = append(allKeys, keys...)
			mu.Unlock()
		}()
	}

	wg.Wait()
	log.Printf("✔ Write Phase Complete: %.1f QPS total | %d keys written\n", totalQPS, len(allKeys))
	return allKeys
}

// benchmarkRead performs concurrent /get operations using the keys written earlier.
func benchmarkRead(keys []string) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalQPS float64

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			qps, _ := benchmark("read", *readIters, func() string {
				return sendRead(keys)
			})
			mu.Lock()
			totalQPS += qps
			mu.Unlock()
		}()
	}

	wg.Wait()
	log.Printf("✔ Read Phase Complete: %.1f QPS total\n", totalQPS)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	flag.Parse()

	fmt.Printf(" Benchmarking http://%s | Writes: %d × %d threads | Reads: %d\n",
		*serverAddr, *writeIters, *concurrency, *readIters)

	// Phase 1: Write
	keys := benchmarkWrite()

	// Phase 2 (optional): More writes in background
	go benchmarkWrite()

	// Phase 3: Read
	benchmarkRead(keys)
}
