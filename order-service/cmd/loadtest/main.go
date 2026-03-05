package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

type result struct {
	latency    time.Duration
	statusCode int
	err        error
}

func main() {
	url := flag.String("url", "http://localhost:8081", "base URL of the server")
	concurrency := flag.Int("concurrency", 50, "number of parallel workers")
	duration := flag.Duration("duration", 10*time.Second, "test duration")
	uidsFile := flag.String("uids-file", "/tmp/order_uids.txt", "path to file with UIDs")
	flag.Parse()

	uids, err := loadUids(*uidsFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load UIDs: %v\n", err)
		os.Exit(1)
	}
	if len(uids) == 0 {
		fmt.Fprintln(os.Stderr, "no UIDs loaded from file")
		os.Exit(1)
	}
	fmt.Printf("Loaded %d UIDs\n", len(uids))

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: *concurrency,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	results := make(chan result, *concurrency)

	wg := &sync.WaitGroup{}
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(ctx, client, *url, uids, results)
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var all []result
	for v := range results {
		all = append(all, v)
	}

	printStats(all, *duration)
}

func loadUids(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var res []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			res = append(res, line)
		}
	}

	return res, nil
}

func worker(ctx context.Context, client *http.Client, url string, uids []string, results chan<- result) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		uid := uids[rand.Intn(len(uids))]
		reqURL := url + "/order/" + uid

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			results <- result{err: err}
			continue
		}

		start := time.Now()
		resp, err := client.Do(req)
		latency := time.Since(start)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			results <- result{latency: latency, err: err}
			continue
		}

		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		results <- result{latency: latency, statusCode: resp.StatusCode}
	}
}

func printStats(all []result, duration time.Duration) {
	if len(all) == 0 {
		fmt.Println("No requests completed")
		return
	}
	var (
		latencies  []time.Duration
		totalTime  time.Duration
		statusDist = make(map[int]int)
		errCount   int
	)
	for _, r := range all {
		if r.err != nil {
			errCount++
			continue
		}
		latencies = append(latencies, r.latency)
		totalTime += r.latency
		statusDist[r.statusCode]++
	}
	fmt.Printf("\n=== Load Test Results ===\n")
	fmt.Printf("Total requests: %d\n", len(all))
	fmt.Printf("Duration:       %s\n", duration)
	fmt.Printf("RPS:            %.2f\n", float64(len(all))/duration.Seconds())
	fmt.Printf("Errors:         %d\n", errCount)

	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
		avg := totalTime / time.Duration(len(latencies))
		fmt.Printf("\nLatency:\n")
		fmt.Printf("  avg: %s\n", avg)
		fmt.Printf("  p50: %s\n", percentile(latencies, 50))
		fmt.Printf("  p95: %s\n", percentile(latencies, 95))
		fmt.Printf("  p99: %s\n", percentile(latencies, 99))
	}

	fmt.Printf("\nStatus codes:\n")
	for code, count := range statusDist {
		fmt.Printf("  %d: %d\n", code, count)
	}
}
func percentile(sorted []time.Duration, p int) time.Duration {
	idx := len(sorted) * p / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
