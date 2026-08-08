package main

import (
	"context"
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

	backends, err := buildBackends(loadedConfig.BackendURLs)
	if err != nil {
		logger.Error("invalid backend configuration", "error", err)
		os.Exit(1)
	}

	roundRobin, err := balancer.NewRoundRobin(backends)
	if err != nil {
		logger.Error("invalid backend configuration", "error", err)
		os.Exit(1)
	}

	reverseProxy := proxy.NewWithBalancerWithTimeoutAndRetries(roundRobin, logger, loadedConfig.RequestTimeout, loadedConfig.MaxAttempts)
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

func buildBackends(backendURLs []string) ([]*backend.Backend, error) {
	backends := make([]*backend.Backend, 0, len(backendURLs))
	for index, backendURL := range backendURLs {
		backendID := fmt.Sprintf("backend-%d", index+1)
		parsedBackend, err := backend.New(backendID, backendURL)
		if err != nil {
			return nil, err
		}

		backends = append(backends, parsedBackend)
	}

	return backends, nil
}
