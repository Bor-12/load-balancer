package integration

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Bor-12/load-balancer/internal/backend"
	"github.com/Bor-12/load-balancer/internal/balancer"
	"github.com/Bor-12/load-balancer/internal/health"
	"github.com/Bor-12/load-balancer/internal/metrics"
	"github.com/Bor-12/load-balancer/internal/proxy"
	dto "github.com/prometheus/client_model/go"
)

type backendResponse struct {
	Instance string `json:"instance"`
	Path     string `json:"path"`
}

func TestIntegration_ProxyDistributesRequestsWithRoundRobin(t *testing.T) {
	backendA := newDemoBackend(t, "backend-1", 0, http.StatusOK)
	defer backendA.Close()
	backendB := newDemoBackend(t, "backend-2", 0, http.StatusOK)
	defer backendB.Close()
	backendC := newDemoBackend(t, "backend-3", 0, http.StatusOK)
	defer backendC.Close()

	backends := []*backend.Backend{
		newBackend(t, "backend-1", backendA.URL),
		newBackend(t, "backend-2", backendB.URL),
		newBackend(t, "backend-3", backendC.URL),
	}
	roundRobin := newRoundRobin(t, backends)
	loadBalancer := httptest.NewServer(proxy.NewWithBalancer(roundRobin, testLogger()))
	defer loadBalancer.Close()

	expectedInstances := []string{"backend-1", "backend-2", "backend-3", "backend-1", "backend-2", "backend-3"}
	for _, expectedInstance := range expectedInstances {
		response := getBackendResponse(t, loadBalancer.URL+"/hello")
		if response.Instance != expectedInstance {
			t.Fatalf("expected instance %s, got %s", expectedInstance, response.Instance)
		}
		if response.Path != "/hello" {
			t.Fatalf("expected path %s, got %s", "/hello", response.Path)
		}
	}
}

func TestIntegration_FailoverSkipsUnhealthyBackend(t *testing.T) {
	backendA := newDemoBackend(t, "backend-1", 0, http.StatusOK)
	defer backendA.Close()
	backendB := newDemoBackend(t, "backend-2", 0, http.StatusOK)
	backendB.Close()
	backendC := newDemoBackend(t, "backend-3", 0, http.StatusOK)
	defer backendC.Close()

	backends := []*backend.Backend{
		newBackend(t, "backend-1", backendA.URL),
		newBackend(t, "backend-2", backendB.URL),
		newBackend(t, "backend-3", backendC.URL),
	}
	checker := health.NewChecker(backends, "/health", time.Second, 25*time.Millisecond, testLogger())
	checker.CheckOnce(context.Background())

	roundRobin := newRoundRobin(t, backends)
	loadBalancer := httptest.NewServer(proxy.NewWithBalancer(roundRobin, testLogger()))
	defer loadBalancer.Close()

	for range 8 {
		response := getBackendResponse(t, loadBalancer.URL+"/")
		if response.Instance == "backend-2" {
			t.Fatal("expected unhealthy backend-2 to be skipped")
		}
	}
}

func TestIntegration_BackendRecoveryReturnsToPool(t *testing.T) {
	backendA := newDemoBackend(t, "backend-1", 0, http.StatusOK)
	defer backendA.Close()
	recoverableBackend := newToggleBackend(t, "backend-2")
	defer recoverableBackend.server.Close()

	backendB := newBackend(t, "backend-2", recoverableBackend.server.URL)
	backends := []*backend.Backend{
		newBackend(t, "backend-1", backendA.URL),
		backendB,
	}
	checker := health.NewChecker(backends, "/health", time.Second, 25*time.Millisecond, testLogger())

	recoverableBackend.SetHealthy(false)
	checker.CheckOnce(context.Background())
	if backendB.IsAlive() {
		t.Fatal("expected backend-2 to be unhealthy")
	}

	recoverableBackend.SetHealthy(true)
	checker.CheckOnce(context.Background())
	if !backendB.IsAlive() {
		t.Fatal("expected backend-2 to recover")
	}

	roundRobin := newRoundRobin(t, backends)
	loadBalancer := httptest.NewServer(proxy.NewWithBalancer(roundRobin, testLogger()))
	defer loadBalancer.Close()

	seenRecoveredBackend := false
	for range 4 {
		response := getBackendResponse(t, loadBalancer.URL+"/")
		if response.Instance == "backend-2" {
			seenRecoveredBackend = true
		}
	}

	if !seenRecoveredBackend {
		t.Fatal("expected recovered backend to receive traffic")
	}
}

func TestIntegration_RetriesIdempotentRequestAfterBackendFailure(t *testing.T) {
	unavailableBackend := newDemoBackend(t, "backend-1", 0, http.StatusOK)
	unavailableBackendURL := unavailableBackend.URL
	unavailableBackend.Close()

	healthyBackend := newDemoBackend(t, "backend-2", 0, http.StatusOK)
	defer healthyBackend.Close()

	backends := []*backend.Backend{
		newBackend(t, "backend-1", unavailableBackendURL),
		newBackend(t, "backend-2", healthyBackend.URL),
	}
	recorder := metrics.NewRecorder(backends)
	roundRobin := newRoundRobin(t, backends)
	loadBalancer := httptest.NewServer(proxy.NewWithBalancerWithTimeoutRetriesAndObserver(roundRobin, testLogger(), time.Second, 2, recorder))
	defer loadBalancer.Close()

	response := getBackendResponse(t, loadBalancer.URL+"/")
	if response.Instance != "backend-2" {
		t.Fatalf("expected retry to reach backend-2, got %s", response.Instance)
	}

	metricFamilies, err := recorder.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}
	if metricValue(metricFamilies, "cloudbalancer_retries_total", map[string]string{"method": http.MethodGet}) != 1 {
		t.Fatal("expected GET retry metric to be incremented")
	}
}

func TestIntegration_LeastConnectionsPrefersFastBackendUnderConcurrentLoad(t *testing.T) {
	slowBackend := newDemoBackend(t, "slow", 80*time.Millisecond, http.StatusOK)
	defer slowBackend.Close()
	fastBackend := newDemoBackend(t, "fast", 0, http.StatusOK)
	defer fastBackend.Close()

	backends := []*backend.Backend{
		newBackend(t, "slow", slowBackend.URL),
		newBackend(t, "fast", fastBackend.URL),
	}
	leastConnections, err := balancer.NewLeastConnections(backends)
	if err != nil {
		t.Fatalf("failed to create least connections balancer: %v", err)
	}
	loadBalancer := httptest.NewServer(proxy.NewWithBalancerWithTimeout(leastConnections, testLogger(), time.Second))
	defer loadBalancer.Close()

	results := make(chan string, 30)
	var waitGroup sync.WaitGroup
	for range 30 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			results <- getBackendResponse(t, loadBalancer.URL+"/").Instance
		}()
	}
	waitGroup.Wait()
	close(results)

	counts := map[string]int{}
	for result := range results {
		counts[result]++
	}

	if counts["fast"] <= counts["slow"] {
		t.Fatalf("expected fast backend to receive more requests, got fast=%d slow=%d", counts["fast"], counts["slow"])
	}
}

func newDemoBackend(t *testing.T, instance string, responseDelay time.Duration, healthStatus int) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/health" {
			responseWriter.WriteHeader(healthStatus)
			return
		}

		if responseDelay > 0 {
			time.Sleep(responseDelay)
		}

		responseWriter.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(responseWriter).Encode(backendResponse{
			Instance: instance,
			Path:     request.URL.Path,
		})
	}))
}

type toggleBackend struct {
	server  *httptest.Server
	mutex   sync.RWMutex
	healthy bool
}

func newToggleBackend(t *testing.T, instance string) *toggleBackend {
	t.Helper()

	toggle := &toggleBackend{healthy: true}
	toggle.server = httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/health" {
			if toggle.IsHealthy() {
				responseWriter.WriteHeader(http.StatusOK)
				return
			}

			responseWriter.WriteHeader(http.StatusInternalServerError)
			return
		}

		responseWriter.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(responseWriter).Encode(backendResponse{
			Instance: instance,
			Path:     request.URL.Path,
		})
	}))

	return toggle
}

func (toggle *toggleBackend) SetHealthy(healthy bool) {
	toggle.mutex.Lock()
	defer toggle.mutex.Unlock()

	toggle.healthy = healthy
}

func (toggle *toggleBackend) IsHealthy() bool {
	toggle.mutex.RLock()
	defer toggle.mutex.RUnlock()

	return toggle.healthy
}

func newBackend(t *testing.T, id string, rawURL string) *backend.Backend {
	t.Helper()

	createdBackend, err := backend.New(id, rawURL)
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	return createdBackend
}

func newRoundRobin(t *testing.T, backends []*backend.Backend) *balancer.RoundRobin {
	t.Helper()

	roundRobin, err := balancer.NewRoundRobin(backends)
	if err != nil {
		t.Fatalf("failed to create round robin: %v", err)
	}

	return roundRobin
}

func getBackendResponse(t *testing.T, url string) backendResponse {
	t.Helper()

	response, err := http.Get(url)
	if err != nil {
		t.Fatalf("failed to get %s: %v", url, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}

	var decodedResponse backendResponse
	if err := json.NewDecoder(response.Body).Decode(&decodedResponse); err != nil {
		t.Fatalf("failed to decode backend response: %v", err)
	}

	return decodedResponse
}

func metricValue(metricFamilies []*dto.MetricFamily, name string, labels map[string]string) float64 {
	for _, metricFamily := range metricFamilies {
		if metricFamily.GetName() != name {
			continue
		}

		for _, metric := range metricFamily.GetMetric() {
			if !metricHasLabels(metric, labels) {
				continue
			}

			if metric.GetCounter() != nil {
				return metric.GetCounter().GetValue()
			}
			if metric.GetGauge() != nil {
				return metric.GetGauge().GetValue()
			}
		}
	}

	return 0
}

func metricHasLabels(metric *dto.Metric, expectedLabels map[string]string) bool {
	foundLabels := make(map[string]string, len(metric.GetLabel()))
	for _, label := range metric.GetLabel() {
		foundLabels[label.GetName()] = label.GetValue()
	}

	for expectedName, expectedValue := range expectedLabels {
		if foundLabels[expectedName] != expectedValue {
			return false
		}
	}

	return true
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
