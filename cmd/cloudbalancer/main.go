package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/Bor-12/load-balancer/internal/backend"
	"github.com/Bor-12/load-balancer/internal/balancer"
	"github.com/Bor-12/load-balancer/internal/health"
	"github.com/Bor-12/load-balancer/internal/proxy"
)

func main() {
	logger := slog.Default()
	loadedConfig, err := loadConfig()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	backends, err := buildBackends(loadedConfig.BackendURLs, loadedConfig.BackendWeights)
	if err != nil {
		logger.Error("invalid backend configuration", "error", err)
		os.Exit(1)
	}

	requestBalancer, err := buildBalancer(loadedConfig.BalancerStrategy, backends)
	if err != nil {
		logger.Error("invalid balancer configuration", "error", err)
		os.Exit(1)
	}

	reverseProxy := proxy.NewWithBalancerWithTimeoutAndRetries(requestBalancer, logger, loadedConfig.RequestTimeout, loadedConfig.MaxAttempts)
	healthChecker := health.NewChecker(backends, loadedConfig.HealthCheckPath, loadedConfig.HealthCheckInterval, loadedConfig.HealthCheckTimeout, logger)
	go healthChecker.Run(context.Background())

	server := &http.Server{
		Addr:    loadedConfig.ListenAddress,
		Handler: reverseProxy,
	}

	logger.Info("CloudBalancer listening", "address", server.Addr, "backend_count", len(backends))

	if err := server.ListenAndServe(); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func buildBackends(backendURLs []string, backendWeights []int) ([]*backend.Backend, error) {
	if len(backendWeights) > 0 && len(backendWeights) != len(backendURLs) {
		return nil, errors.New("backend weights count must match backend URLs count")
	}

	backends := make([]*backend.Backend, 0, len(backendURLs))
	for index, backendURL := range backendURLs {
		backendID := fmt.Sprintf("backend-%d", index+1)
		backendWeight := 1
		if len(backendWeights) > 0 {
			backendWeight = backendWeights[index]
		}

		parsedBackend, err := backend.NewWithWeight(backendID, backendURL, backendWeight)
		if err != nil {
			return nil, err
		}

		backends = append(backends, parsedBackend)
	}

	return backends, nil
}

func buildBalancer(strategy string, backends []*backend.Backend) (balancer.Balancer, error) {
	switch strategy {
	case "round_robin":
		return balancer.NewRoundRobin(backends)
	case "weighted_round_robin":
		return balancer.NewWeightedRoundRobin(backends)
	case "least_connections":
		return balancer.NewLeastConnections(backends)
	default:
		return nil, fmt.Errorf("unsupported balancer strategy %q", strategy)
	}
}
