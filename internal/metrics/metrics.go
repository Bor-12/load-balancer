package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Bor-12/load-balancer/internal/backend"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
)

type Recorder struct {
	registry               *prometheus.Registry
	requestsTotal          *prometheus.CounterVec
	backendErrorsTotal     *prometheus.CounterVec
	backendHealthy         *prometheus.GaugeVec
	backendActiveRequests  *prometheus.GaugeVec
	backendRequestDuration *prometheus.HistogramVec
	retriesTotal           *prometheus.CounterVec
}

func NewRecorder(backends []*backend.Backend) *Recorder {
	recorder := &Recorder{
		registry: prometheus.NewRegistry(),
		requestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cloudbalancer_requests_total",
			Help: "Total proxied backend requests.",
		}, []string{"method", "backend", "status"}),
		backendErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cloudbalancer_backend_errors_total",
			Help: "Total backend error responses.",
		}, []string{"backend", "status"}),
		backendHealthy: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cloudbalancer_backend_healthy",
			Help: "Backend health state. 1 is healthy, 0 is unhealthy.",
		}, []string{"backend"}),
		backendActiveRequests: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "cloudbalancer_backend_active_requests",
			Help: "Current active requests per backend.",
		}, []string{"backend"}),
		backendRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "cloudbalancer_backend_request_duration_seconds",
			Help:    "Backend request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"backend"}),
		retriesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cloudbalancer_retries_total",
			Help: "Total backend request retries.",
		}, []string{"method"}),
	}

	recorder.registry.MustRegister(
		recorder.requestsTotal,
		recorder.backendErrorsTotal,
		recorder.backendHealthy,
		recorder.backendActiveRequests,
		recorder.backendRequestDuration,
		recorder.retriesTotal,
	)

	for _, configuredBackend := range backends {
		recorder.UpdateBackendState(configuredBackend)
	}

	return recorder
}

func (recorder *Recorder) Handler() http.Handler {
	return promhttp.HandlerFor(recorder.registry, promhttp.HandlerOpts{})
}

func (recorder *Recorder) RecordRequest(method string, backendID string, statusCode int, duration time.Duration) {
	status := strconv.Itoa(statusCode)
	recorder.requestsTotal.WithLabelValues(method, backendID, status).Inc()
	recorder.backendRequestDuration.WithLabelValues(backendID).Observe(duration.Seconds())
	if statusCode >= http.StatusInternalServerError {
		recorder.backendErrorsTotal.WithLabelValues(backendID, status).Inc()
	}
}

func (recorder *Recorder) RecordRetry(method string) {
	recorder.retriesTotal.WithLabelValues(method).Inc()
}

func (recorder *Recorder) UpdateBackendState(updatedBackend *backend.Backend) {
	healthy := 0.0
	if updatedBackend.IsAlive() {
		healthy = 1
	}

	recorder.backendHealthy.WithLabelValues(updatedBackend.ID).Set(healthy)
	recorder.backendActiveRequests.WithLabelValues(updatedBackend.ID).Set(float64(updatedBackend.ActiveCount()))
}

func (recorder *Recorder) Gather() ([]*dto.MetricFamily, error) {
	return recorder.registry.Gather()
}
