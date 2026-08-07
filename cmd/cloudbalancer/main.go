package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Bor-12/load-balancer/internal/backend"
	"github.com/Bor-12/load-balancer/internal/balancer"
	"github.com/Bor-12/load-balancer/internal/health"
	"github.com/Bor-12/load-balancer/internal/proxy"
)

func main() {
	logger := slog.Default()
	backendURLs := backendURLsFromEnvironment()

	backends, err := buildBackends(backendURLs)
	if err != nil {
		logger.Error("invalid backend configuration", "error", err)
		os.Exit(1)
	}

	roundRobin, err := balancer.NewRoundRobin(backends)
	if err != nil {
		logger.Error("invalid backend configuration", "error", err)
		os.Exit(1)
	}

	reverseProxy := proxy.NewWithBalancer(roundRobin, logger)
	healthChecker := health.NewChecker(backends, "/health", 2*time.Second, 500*time.Millisecond, logger)
	go healthChecker.Run(context.Background())

	server := &http.Server{
		Addr:    ":8080",
		Handler: reverseProxy,
	}

	logger.Info("CloudBalancer listening", "address", server.Addr, "backend_count", len(backends))

	if err := server.ListenAndServe(); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func backendURLsFromEnvironment() []string {
	rawBackendURLs := os.Getenv("BACKEND_URLS")
	if rawBackendURLs == "" {
		rawBackendURLs = os.Getenv("BACKEND_URL")
	}

	backendURLs := strings.Split(rawBackendURLs, ",")
	cleanBackendURLs := make([]string, 0, len(backendURLs))
	for _, backendURL := range backendURLs {
		cleanBackendURL := strings.TrimSpace(backendURL)
		if cleanBackendURL != "" {
			cleanBackendURLs = append(cleanBackendURLs, cleanBackendURL)
		}
	}

	return cleanBackendURLs
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
