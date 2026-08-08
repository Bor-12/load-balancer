package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestLoadConfigFromFile(t *testing.T) {
	tempDirectory := t.TempDir()
	configPath := filepath.Join(tempDirectory, "config.json")
	configContent := `{
		"listen_address": ":9090",
		"backend_urls": ["http://localhost:9001", "http://localhost:9002"],
		"request_timeout": "3s",
		"health_check_path": "/ready",
		"health_check_interval": "1s",
		"health_check_timeout": "250ms",
		"max_attempts": 3
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CONFIG_FILE", configPath)

	loadedConfig, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}

	expectedBackendURLs := []string{"http://localhost:9001", "http://localhost:9002"}
	if !reflect.DeepEqual(loadedConfig.BackendURLs, expectedBackendURLs) {
		t.Fatalf("expected %v, got %v", expectedBackendURLs, loadedConfig.BackendURLs)
	}
	if loadedConfig.ListenAddress != ":9090" {
		t.Fatalf("expected listen address %q, got %q", ":9090", loadedConfig.ListenAddress)
	}
	if loadedConfig.RequestTimeout != 3*time.Second {
		t.Fatalf("expected request timeout %s, got %s", 3*time.Second, loadedConfig.RequestTimeout)
	}
	if loadedConfig.HealthCheckPath != "/ready" {
		t.Fatalf("expected health check path %q, got %q", "/ready", loadedConfig.HealthCheckPath)
	}
	if loadedConfig.HealthCheckInterval != time.Second {
		t.Fatalf("expected health check interval %s, got %s", time.Second, loadedConfig.HealthCheckInterval)
	}
	if loadedConfig.HealthCheckTimeout != 250*time.Millisecond {
		t.Fatalf("expected health check timeout %s, got %s", 250*time.Millisecond, loadedConfig.HealthCheckTimeout)
	}
	if loadedConfig.MaxAttempts != 3 {
		t.Fatalf("expected max attempts %d, got %d", 3, loadedConfig.MaxAttempts)
	}
}

func TestLoadConfigEnvironmentOverridesFile(t *testing.T) {
	tempDirectory := t.TempDir()
	configPath := filepath.Join(tempDirectory, "config.json")
	configContent := `{
		"backend_urls": ["http://localhost:9001"],
		"request_timeout": "3s"
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CONFIG_FILE", configPath)
	t.Setenv("BACKEND_URLS", "http://localhost:9101,http://localhost:9102")
	t.Setenv("REQUEST_TIMEOUT", "1s")

	loadedConfig, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}

	expectedBackendURLs := []string{"http://localhost:9101", "http://localhost:9102"}
	if !reflect.DeepEqual(loadedConfig.BackendURLs, expectedBackendURLs) {
		t.Fatalf("expected %v, got %v", expectedBackendURLs, loadedConfig.BackendURLs)
	}
	if loadedConfig.RequestTimeout != time.Second {
		t.Fatalf("expected request timeout %s, got %s", time.Second, loadedConfig.RequestTimeout)
	}
}

func TestLoadConfigRequiresBackendURLs(t *testing.T) {
	t.Setenv("CONFIG_FILE", filepath.Join(t.TempDir(), "missing.json"))

	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
