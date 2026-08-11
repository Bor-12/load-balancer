package status

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Bor-12/load-balancer/internal/backend"
)

func TestHealthz_Returns200(t *testing.T) {
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	HealthzHandler().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, responseRecorder.Code)
	}
}

func TestReadyz_Returns200WhenBackendAvailable(t *testing.T) {
	testBackend := newTestBackend(t, "backend-1")
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	ReadyzHandler([]*backend.Backend{testBackend}).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, responseRecorder.Code)
	}
}

func TestReadyz_Returns503WhenNoBackendAvailable(t *testing.T) {
	testBackend := newTestBackend(t, "backend-1")
	testBackend.SetAlive(false)
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	ReadyzHandler([]*backend.Backend{testBackend}).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, responseRecorder.Code)
	}
}

func newTestBackend(t *testing.T, id string) *backend.Backend {
	t.Helper()

	testBackend, err := backend.New(id, "http://localhost:9001")
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	return testBackend
}
