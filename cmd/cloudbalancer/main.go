package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/Bor-12/load-balancer/internal/proxy"
)

func main() {
	logger := slog.Default()
	backendURL := os.Getenv("BACKEND_URL")

	reverseProxy, err := proxy.New(backendURL, logger)
	if err != nil {
		logger.Error("invalid backend configuration", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:    ":8080",
		Handler: reverseProxy,
	}

	logger.Info("CloudBalancer listening", "address", server.Addr, "backend_url", backendURL)

	if err := server.ListenAndServe(); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
