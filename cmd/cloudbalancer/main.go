package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Bor-12/load-balancer/internal/backend"
	"github.com/Bor-12/load-balancer/internal/balancer"
	"github.com/Bor-12/load-balancer/internal/config"
	"github.com/Bor-12/load-balancer/internal/health"
	"github.com/Bor-12/load-balancer/internal/metrics"
	"github.com/Bor-12/load-balancer/internal/proxy"
	serverrunner "github.com/Bor-12/load-balancer/internal/server"
	"github.com/Bor-12/load-balancer/internal/status"
)

const defaultConfigPath = "configs/config.local.yaml"

func main() {
	logger := slog.Default()
	configPath := flag.String("config", defaultConfigPath, "path to YAML configuration file")
	flag.Parse()
	runContext, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	loadedConfig, err := config.Load(*configPath)
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	backends, err := buildBackends(loadedConfig.Backends)
	if err != nil {
		logger.Error("invalid backend configuration", "error", err)
		os.Exit(1)
	}

	requestBalancer, err := buildBalancer(loadedConfig.Balancer.Strategy, backends)
	if err != nil {
		logger.Error("invalid balancer configuration", "error", err)
		os.Exit(1)
	}

	metricsRecorder := metrics.NewRecorder(backends)
	reverseProxy := proxy.NewWithBalancerWithTimeoutRetriesAndObserver(requestBalancer, logger, loadedConfig.Server.RequestTimeout, loadedConfig.Retries.MaxAttempts, metricsRecorder)
	if loadedConfig.HealthCheck.Enabled {
		healthChecker := health.NewCheckerWithObserver(backends, loadedConfig.HealthCheck.Path, loadedConfig.HealthCheck.Interval, loadedConfig.HealthCheck.Timeout, logger, metricsRecorder)
		go healthChecker.Run(runContext)
	}

	router := http.NewServeMux()
	router.Handle("/metrics", metricsRecorder.Handler())
	router.Handle("/healthz", status.HealthzHandler())
	router.Handle("/readyz", status.ReadyzHandler(backends))
	router.Handle("/", reverseProxy)

	server := &http.Server{
		Addr:    loadedConfig.Server.ListenAddress,
		Handler: router,
	}

	logger.Info("CloudBalancer listening", "address", server.Addr, "backend_count", len(backends))

	runner := serverrunner.NewRunner(server, logger, loadedConfig.Server.ShutdownTimeout)
	if err := runner.Run(runContext); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func buildBackends(configuredBackends []config.BackendConfig) ([]*backend.Backend, error) {
	backends := make([]*backend.Backend, 0, len(configuredBackends))
	for _, configuredBackend := range configuredBackends {
		parsedBackend, err := backend.NewWithWeight(configuredBackend.ID, configuredBackend.URL, configuredBackend.Weight)
		if err != nil {
			return nil, err
		}

		backends = append(backends, parsedBackend)
	}

	return backends, nil
}

func buildBalancer(strategy string, backends []*backend.Backend) (balancer.Balancer, error) {
	switch strategy {
	case config.StrategyRoundRobin:
		return balancer.NewRoundRobin(backends)
	case config.StrategyWeightedRoundRobin:
		return balancer.NewWeightedRoundRobin(backends)
	case config.StrategyLeastConnections:
		return balancer.NewLeastConnections(backends)
	default:
		return nil, fmt.Errorf("unsupported balancer strategy %q", strategy)
	}
}
