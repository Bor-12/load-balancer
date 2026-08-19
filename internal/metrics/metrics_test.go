package metrics

import (
	"net/http"
	"testing"
	"time"

	"github.com/Bor-12/load-balancer/internal/backend"
)

func TestMetrics_RequestCounterIncrements(t *testing.T) {
	testBackend := newTestBackend(t, "backend-1")
	recorder := NewRecorder([]*backend.Backend{testBackend})

	recorder.RecordRequest(http.MethodGet, "backend-1", http.StatusOK, time.Millisecond)

	if metricValue(t, recorder, "loadbalancer_requests_total") != 1 {
		t.Fatalf("expected request counter to increment")
	}
}

func TestMetrics_BackendErrorCounterIncrements(t *testing.T) {
	testBackend := newTestBackend(t, "backend-1")
	recorder := NewRecorder([]*backend.Backend{testBackend})

	recorder.RecordRequest(http.MethodGet, "backend-1", http.StatusBadGateway, time.Millisecond)

	if metricValue(t, recorder, "loadbalancer_backend_errors_total") != 1 {
		t.Fatalf("expected backend error counter to increment")
	}
}

func TestMetrics_HealthyGaugeChanges(t *testing.T) {
	testBackend := newTestBackend(t, "backend-1")
	recorder := NewRecorder([]*backend.Backend{testBackend})

	testBackend.SetAlive(false)
	recorder.UpdateBackendState(testBackend)

	if metricValue(t, recorder, "loadbalancer_backend_healthy") != 0 {
		t.Fatalf("expected healthy gauge to be 0")
	}
}

func TestMetrics_RetryCounterIncrements(t *testing.T) {
	testBackend := newTestBackend(t, "backend-1")
	recorder := NewRecorder([]*backend.Backend{testBackend})

	recorder.RecordRetry(http.MethodGet)

	if metricValue(t, recorder, "loadbalancer_retries_total") != 1 {
		t.Fatalf("expected retry counter to increment")
	}
}

func metricValue(t *testing.T, recorder *Recorder, name string) float64 {
	t.Helper()

	metricFamilies, err := recorder.Gather()
	if err != nil {
		t.Fatal(err)
	}

	for _, metricFamily := range metricFamilies {
		if metricFamily.GetName() != name {
			continue
		}

		metric := metricFamily.GetMetric()[0]
		if metric.GetCounter() != nil {
			return metric.GetCounter().GetValue()
		}
		if metric.GetGauge() != nil {
			return metric.GetGauge().GetValue()
		}
	}

	t.Fatalf("metric %s not found", name)
	return 0
}

func newTestBackend(t *testing.T, id string) *backend.Backend {
	t.Helper()

	testBackend, err := backend.New(id, "http://localhost:9001")
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	return testBackend
}
