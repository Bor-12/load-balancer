package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRunLoadTest_RecordsStatusCodes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	result := runLoadTest(server.URL, 10, 2, time.Second)

	if result.Errors != 0 {
		t.Fatalf("expected errors %d, got %d", 0, result.Errors)
	}
	if result.StatusCounts[http.StatusAccepted] != 10 {
		t.Fatalf("expected status count %d, got %d", 10, result.StatusCounts[http.StatusAccepted])
	}
	if len(result.Latencies) != 10 {
		t.Fatalf("expected latencies %d, got %d", 10, len(result.Latencies))
	}
}

func TestRunLoadTest_RecordsErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(http.StatusOK)
	}))
	targetURL := server.URL
	server.Close()

	result := runLoadTest(targetURL, 5, 2, 50*time.Millisecond)

	if result.Errors != 5 {
		t.Fatalf("expected errors %d, got %d", 5, result.Errors)
	}
}

func TestPercentile(t *testing.T) {
	latencies := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
	}

	if percentile(latencies, 50) != 20*time.Millisecond {
		t.Fatalf("expected p50 %s, got %s", 20*time.Millisecond, percentile(latencies, 50))
	}
	if percentile(latencies, 95) != 40*time.Millisecond {
		t.Fatalf("expected p95 %s, got %s", 40*time.Millisecond, percentile(latencies, 95))
	}
}
