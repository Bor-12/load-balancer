package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
)

type backendResponse struct {
	Instance string `json:"instance"`
	Path     string `json:"path"`
}

func main() {
	port := valueOrDefault(os.Getenv("PORT"), "8081")
	instanceID := valueOrDefault(os.Getenv("BACKEND_ID"), os.Getenv("INSTANCE_ID"))
	instanceID = valueOrDefault(instanceID, "backend-1")
	address := ":" + port

	http.HandleFunc("/health", func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/", func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("Content-Type", "application/json")
		response := backendResponse{
			Instance: instanceID,
			Path:     request.URL.Path,
		}

		if err := json.NewEncoder(responseWriter).Encode(response); err != nil {
			slog.Error("failed to write response", "error", err)
		}
	})

	slog.Info("demo backend listening", "address", address, "instance_id", instanceID)

	if err := http.ListenAndServe(address, nil); err != nil {
		slog.Error("demo backend stopped", "error", err)
		os.Exit(1)
	}
}

func valueOrDefault(value string, defaultValue string) string {
	if value != "" {
		return value
	}

	return defaultValue
}
