package proxy

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
