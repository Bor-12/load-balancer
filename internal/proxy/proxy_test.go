package proxy

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Bor-12/load-balancer/internal/backend"
	"github.com/Bor-12/load-balancer/internal/balancer"
)

func TestProxy_ForwardsRequestToBackend(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/hello" {
			t.Fatalf("expected path /hello, got %s", request.URL.Path)
		}

		responseWriter.WriteHeader(http.StatusCreated)
		_, _ = responseWriter.Write([]byte("backend response"))
	}))
	defer backend.Close()

	reverseProxy := newTestProxy(t, backend.URL)
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/hello", nil)

	reverseProxy.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, responseRecorder.Code)
	}

	if strings.TrimSpace(responseRecorder.Body.String()) != "backend response" {
		t.Fatalf("expected backend response, got %q", responseRecorder.Body.String())
	}
}

func TestProxy_ForwardsQueryString(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if request.URL.RawQuery != "q=cloud" {
			t.Fatalf("expected query q=cloud, got %s", request.URL.RawQuery)
		}

		responseWriter.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	reverseProxy := newTestProxy(t, backend.URL)
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/search?q=cloud", nil)

	reverseProxy.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, responseRecorder.Code)
	}
}

func TestProxy_PreservesMethod(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("expected method POST, got %s", request.Method)
		}

		responseWriter.WriteHeader(http.StatusAccepted)
	}))
	defer backend.Close()

	reverseProxy := newTestProxy(t, backend.URL)
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader("payload"))

	reverseProxy.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, responseRecorder.Code)
	}
}

func TestProxy_ReturnsBadGatewayWhenBackendUnavailable(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(http.StatusOK)
	}))
	backendURL := backend.URL
	backend.Close()

	reverseProxy := newTestProxy(t, backendURL)
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	reverseProxy.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, responseRecorder.Code)
	}
}

func TestNewProxy_RejectsInvalidURL(t *testing.T) {
	invalidURLs := []string{
		"",
		"://bad-url",
		"localhost:8081",
		"ftp://localhost:8081",
	}

	for _, invalidURL := range invalidURLs {
		t.Run(invalidURL, func(t *testing.T) {
			_, err := New(invalidURL, testLogger())
			if err == nil {
				t.Fatalf("expected error for URL %q", invalidURL)
			}
		})
	}
}

func TestProxy_RoundRobinDistribution(t *testing.T) {
	backendA := newBackendServer("A")
	defer backendA.Close()
	backendB := newBackendServer("B")
	defer backendB.Close()
	backendC := newBackendServer("C")
	defer backendC.Close()

	backends := []*backend.Backend{
		newTestBackend(t, "A", backendA.URL),
		newTestBackend(t, "B", backendB.URL),
		newTestBackend(t, "C", backendC.URL),
	}

	roundRobin, err := balancer.NewRoundRobin(backends)
	if err != nil {
		t.Fatalf("failed to create round robin: %v", err)
	}

	reverseProxy := NewWithBalancer(roundRobin, testLogger())
	expectedInstances := []string{"A", "B", "C", "A", "B", "C"}

	for _, expectedInstance := range expectedInstances {
		responseRecorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)

		reverseProxy.ServeHTTP(responseRecorder, request)

		if responseRecorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, responseRecorder.Code)
		}

		var response testBackendResponse
		if err := json.NewDecoder(responseRecorder.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response.Instance != expectedInstance {
			t.Fatalf("expected instance %s, got %s", expectedInstance, response.Instance)
		}
	}
}

func TestProxy_ReturnsServiceUnavailableWhenNoBackendIsAlive(t *testing.T) {
	backendA := newBackendServer("A")
	defer backendA.Close()

	testBackend := newTestBackend(t, "A", backendA.URL)
	testBackend.SetAlive(false)

	roundRobin, err := balancer.NewRoundRobin([]*backend.Backend{testBackend})
	if err != nil {
		t.Fatalf("failed to create round robin: %v", err)
	}

	reverseProxy := NewWithBalancer(roundRobin, testLogger())
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	reverseProxy.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, responseRecorder.Code)
	}
}

func TestProxy_ReturnsGatewayTimeoutWhenBackendIsSlow(t *testing.T) {
	slowBackend := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		time.Sleep(50 * time.Millisecond)
		responseWriter.WriteHeader(http.StatusOK)
	}))
	defer slowBackend.Close()

	testBackend := newTestBackend(t, "A", slowBackend.URL)
	roundRobin, err := balancer.NewRoundRobin([]*backend.Backend{testBackend})
	if err != nil {
		t.Fatalf("failed to create round robin: %v", err)
	}

	reverseProxy := NewWithBalancerWithTimeout(roundRobin, testLogger(), 10*time.Millisecond)
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	reverseProxy.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected status %d, got %d", http.StatusGatewayTimeout, responseRecorder.Code)
	}
}

func newTestProxy(t *testing.T, targetURL string) *Proxy {
	t.Helper()

	reverseProxy, err := New(targetURL, testLogger())
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}

	return reverseProxy
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type testBackendResponse struct {
	Instance string `json:"instance"`
}

func newBackendServer(instance string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(responseWriter).Encode(testBackendResponse{Instance: instance})
	}))
}

func newTestBackend(t *testing.T, id string, rawURL string) *backend.Backend {
	t.Helper()

	testBackend, err := backend.New(id, rawURL)
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	return testBackend
}
