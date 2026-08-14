package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	targetURL := flag.String("url", "http://localhost:8080/", "target URL")
	requests := flag.Int("requests", 100, "total number of requests")
	concurrency := flag.Int("concurrency", 10, "number of concurrent workers")
	timeout := flag.Duration("timeout", 10*time.Second, "request timeout")
	flag.Parse()

	if *requests < 1 {
		fmt.Fprintln(os.Stderr, "requests must be at least 1")
		os.Exit(1)
	}
	if *concurrency < 1 {
		fmt.Fprintln(os.Stderr, "concurrency must be at least 1")
		os.Exit(1)
	}

	result := runLoadTest(*targetURL, *requests, *concurrency, *timeout)
	printResult(result)

	if result.Errors > 0 {
		os.Exit(1)
	}
}

type loadTestResult struct {
	Requests     int
	Concurrency  int
	Duration     time.Duration
	Errors       int64
	StatusCounts map[int]int
	Latencies    []time.Duration
}

func runLoadTest(targetURL string, requests int, concurrency int, timeout time.Duration) loadTestResult {
	client := &http.Client{Timeout: timeout}
	jobs := make(chan struct{})
	results := make(chan requestResult, requests)

	startedAt := time.Now()
	var waitGroup sync.WaitGroup
	for range concurrency {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for range jobs {
				results <- executeRequest(client, targetURL)
			}
		}()
	}

	go func() {
		for range requests {
			jobs <- struct{}{}
		}
		close(jobs)
		waitGroup.Wait()
		close(results)
	}()

	loadResult := loadTestResult{
		Requests:     requests,
		Concurrency:  concurrency,
		StatusCounts: make(map[int]int),
		Latencies:    make([]time.Duration, 0, requests),
	}

	for result := range results {
		if result.Err != nil {
			atomic.AddInt64(&loadResult.Errors, 1)
			continue
		}

		loadResult.StatusCounts[result.StatusCode]++
		loadResult.Latencies = append(loadResult.Latencies, result.Latency)
	}

	loadResult.Duration = time.Since(startedAt)

	return loadResult
}

type requestResult struct {
	StatusCode int
	Latency    time.Duration
	Err        error
}

func executeRequest(client *http.Client, targetURL string) requestResult {
	startedAt := time.Now()
	response, err := client.Get(targetURL)
	if err != nil {
		return requestResult{Err: err}
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)

	return requestResult{
		StatusCode: response.StatusCode,
		Latency:    time.Since(startedAt),
	}
}

func printResult(result loadTestResult) {
	successfulRequests := len(result.Latencies)
	requestsPerSecond := float64(result.Requests) / result.Duration.Seconds()

	fmt.Printf("requests=%d concurrency=%d duration=%s rps=%.2f errors=%d\n", result.Requests, result.Concurrency, result.Duration.Round(time.Millisecond), requestsPerSecond, result.Errors)
	fmt.Println("status_codes:")

	statusCodes := make([]int, 0, len(result.StatusCounts))
	for statusCode := range result.StatusCounts {
		statusCodes = append(statusCodes, statusCode)
	}
	sort.Ints(statusCodes)
	for _, statusCode := range statusCodes {
		fmt.Printf("  %d=%d\n", statusCode, result.StatusCounts[statusCode])
	}

	if successfulRequests == 0 {
		return
	}

	sort.Slice(result.Latencies, func(left int, right int) bool {
		return result.Latencies[left] < result.Latencies[right]
	})

	fmt.Printf("latency_min=%s\n", result.Latencies[0].Round(time.Millisecond))
	fmt.Printf("latency_p50=%s\n", percentile(result.Latencies, 50).Round(time.Millisecond))
	fmt.Printf("latency_p95=%s\n", percentile(result.Latencies, 95).Round(time.Millisecond))
	fmt.Printf("latency_max=%s\n", result.Latencies[len(result.Latencies)-1].Round(time.Millisecond))
}

func percentile(latencies []time.Duration, percentileValue int) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	index := (len(latencies)*percentileValue + 99) / 100
	if index < 1 {
		index = 1
	}
	if index > len(latencies) {
		index = len(latencies)
	}

	return latencies[index-1]
}
