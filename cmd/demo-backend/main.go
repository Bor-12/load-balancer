package main

import (
	"encoding/json"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type backendResponse struct {
	Instance string `json:"instance"`
	Path     string `json:"path"`
}

func main() {
	port := valueOrDefault(os.Getenv("PORT"), "8081")
	instanceID := valueOrDefault(os.Getenv("BACKEND_ID"), os.Getenv("INSTANCE_ID"))
	instanceID = valueOrDefault(instanceID, "backend-1")
	responseDelay := durationOrDefault(os.Getenv("RESPONSE_DELAY"), 0)
	responseDelayJitter := durationOrDefault(os.Getenv("RESPONSE_DELAY_JITTER"), 0)
	address := ":" + port

	http.HandleFunc("/health", func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/", func(responseWriter http.ResponseWriter, request *http.Request) {
		delay := responseDuration(responseDelay, responseDelayJitter)
		if delay > 0 {
			time.Sleep(delay)
		}

		responseWriter.Header().Set("Content-Type", "application/json")
		response := backendResponse{
			Instance: instanceID,
			Path:     request.URL.Path,
		}

		if err := json.NewEncoder(responseWriter).Encode(response); err != nil {
			slog.Error("failed to write response", "error", err)
		}
	})

	slog.Info("demo backend listening", "address", address, "instance_id", instanceID, "response_delay", responseDelay, "response_delay_jitter", responseDelayJitter)

	if err := http.ListenAndServe(address, nil); err != nil {
		slog.Error("demo backend stopped", "error", err)
		os.Exit(1)
	}
}

func durationOrDefault(value string, defaultValue time.Duration) time.Duration {
	if value == "" {
		return defaultValue
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}

	return duration
}

func responseDuration(baseDelay time.Duration, jitter time.Duration) time.Duration {
	if jitter <= 0 {
		return baseDelay
	}

	return baseDelay + time.Duration(rand.Int63n(int64(jitter)+1))
}

func valueOrDefault(value string, defaultValue string) string {
	if value != "" {
		return value
	}

	return defaultValue
}
