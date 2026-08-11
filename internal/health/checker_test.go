package health

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Bor-12/load-balancer/internal/backend"
)

func TestHealthChecker_MarksHealthyBackendAlive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	testBackend := newTestBackend(t, server.URL)
	testBackend.SetAlive(false)

	checker := newTestChecker(testBackend)
	checker.CheckOnce(context.Background())

	if !testBackend.IsAlive() {
		t.Fatal("expected backend to be alive")
	}
}

func TestHealthChecker_Marks500BackendUnhealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	testBackend := newTestBackend(t, server.URL)

	checker := newTestChecker(testBackend)
	checker.CheckOnce(context.Background())

	if testBackend.IsAlive() {
		t.Fatal("expected backend to be unhealthy")
	}
}

func TestHealthChecker_MarksUnreachableBackendUnhealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(http.StatusOK)
	}))
	backendURL := server.URL
	server.Close()

	testBackend := newTestBackend(t, backendURL)

	checker := newTestChecker(testBackend)
	checker.CheckOnce(context.Background())

	if testBackend.IsAlive() {
		t.Fatal("expected backend to be unhealthy")
	}
}

func TestHealthChecker_MarksTimeoutBackendUnhealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		time.Sleep(50 * time.Millisecond)
		responseWriter.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	testBackend := newTestBackend(t, server.URL)
	checker := NewChecker([]*backend.Backend{testBackend}, "/health", time.Second, 10*time.Millisecond, testLogger())
	checker.CheckOnce(context.Background())

	if testBackend.IsAlive() {
		t.Fatal("expected backend to be unhealthy")
	}
}

func TestHealthChecker_RecoversBackend(t *testing.T) {
	healthy := false
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if !healthy {
			responseWriter.WriteHeader(http.StatusInternalServerError)
			return
		}

		responseWriter.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	testBackend := newTestBackend(t, server.URL)
	checker := newTestChecker(testBackend)

	checker.CheckOnce(context.Background())
	if testBackend.IsAlive() {
		t.Fatal("expected backend to become unhealthy")
	}

	healthy = true
	checker.CheckOnce(context.Background())
	if !testBackend.IsAlive() {
		t.Fatal("expected backend to recover")
	}
}

func TestHealthChecker_StopsWhenContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	testBackend := newTestBackend(t, server.URL)
	checker := NewChecker([]*backend.Backend{testBackend}, "/health", time.Hour, 100*time.Millisecond, testLogger())
	runContext, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})

	go func() {
		checker.Run(runContext)
		close(stopped)
	}()

	cancel()

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("expected health checker to stop after context cancellation")
	}
}

func newTestChecker(testBackend *backend.Backend) *Checker {
	return NewChecker([]*backend.Backend{testBackend}, "/health", time.Second, 100*time.Millisecond, testLogger())
}

func newTestBackend(t *testing.T, rawURL string) *backend.Backend {
	t.Helper()

	testBackend, err := backend.New("backend-1", rawURL)
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	return testBackend
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
