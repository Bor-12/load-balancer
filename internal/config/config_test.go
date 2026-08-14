package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfig_LoadValidYAML(t *testing.T) {
	loadedConfig := loadConfigFromString(t, `
server:
  listen_address: ":9090"
  request_timeout: 3s
  shutdown_timeout: 7s
balancer:
  strategy: weighted_round_robin
health_check:
  enabled: true
  interval: 1s
  timeout: 250ms
  path: /ready
retries:
  max_attempts: 3
rate_limit:
  enabled: true
  requests_per_second: 7.5
  burst: 15
backends:
  - id: backend-1
    url: http://localhost:9001
    weight: 5
  - id: backend-2
    url: http://localhost:9002
    weight: 3
`)

	if loadedConfig.Server.ListenAddress != ":9090" {
		t.Fatalf("expected listen address %q, got %q", ":9090", loadedConfig.Server.ListenAddress)
	}
	if loadedConfig.Server.RequestTimeout != 3*time.Second {
		t.Fatalf("expected request timeout %s, got %s", 3*time.Second, loadedConfig.Server.RequestTimeout)
	}
	if loadedConfig.Server.ShutdownTimeout != 7*time.Second {
		t.Fatalf("expected shutdown timeout %s, got %s", 7*time.Second, loadedConfig.Server.ShutdownTimeout)
	}
	if loadedConfig.Balancer.Strategy != StrategyWeightedRoundRobin {
		t.Fatalf("expected strategy %q, got %q", StrategyWeightedRoundRobin, loadedConfig.Balancer.Strategy)
	}
	if loadedConfig.HealthCheck.Interval != time.Second {
		t.Fatalf("expected health interval %s, got %s", time.Second, loadedConfig.HealthCheck.Interval)
	}
	if loadedConfig.HealthCheck.Timeout != 250*time.Millisecond {
		t.Fatalf("expected health timeout %s, got %s", 250*time.Millisecond, loadedConfig.HealthCheck.Timeout)
	}
	if loadedConfig.HealthCheck.Path != "/ready" {
		t.Fatalf("expected health path %q, got %q", "/ready", loadedConfig.HealthCheck.Path)
	}
	if loadedConfig.Retries.MaxAttempts != 3 {
		t.Fatalf("expected max attempts %d, got %d", 3, loadedConfig.Retries.MaxAttempts)
	}
	if !loadedConfig.RateLimit.Enabled {
		t.Fatal("expected rate limit to be enabled")
	}
	if loadedConfig.RateLimit.RequestsPerSecond != 7.5 {
		t.Fatalf("expected rate limit rps %f, got %f", 7.5, loadedConfig.RateLimit.RequestsPerSecond)
	}
	if loadedConfig.RateLimit.Burst != 15 {
		t.Fatalf("expected rate limit burst %d, got %d", 15, loadedConfig.RateLimit.Burst)
	}
	if loadedConfig.Backends[0].Weight != 5 {
		t.Fatalf("expected first backend weight %d, got %d", 5, loadedConfig.Backends[0].Weight)
	}
}

func TestConfig_RejectsUnknownStrategy(t *testing.T) {
	_, err := loadConfigErrorFromString(t, `
balancer:
  strategy: random
backends:
  - id: backend-1
    url: http://localhost:9001
`)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestConfig_RejectsDuplicateBackendID(t *testing.T) {
	_, err := loadConfigErrorFromString(t, `
backends:
  - id: backend-1
    url: http://localhost:9001
  - id: backend-1
    url: http://localhost:9002
`)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestConfig_RejectsInvalidBackendURL(t *testing.T) {
	_, err := loadConfigErrorFromString(t, `
backends:
  - id: backend-1
    url: ftp://localhost:9001
`)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestConfig_RejectsInvalidWeight(t *testing.T) {
	_, err := loadConfigErrorFromString(t, `
backends:
  - id: backend-1
    url: http://localhost:9001
    weight: -1
`)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestConfig_RejectsNoBackends(t *testing.T) {
	_, err := loadConfigErrorFromString(t, `
balancer:
  strategy: round_robin
`)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestConfig_RejectsInvalidRateLimitRPS(t *testing.T) {
	_, err := loadConfigErrorFromString(t, `
rate_limit:
  enabled: true
  requests_per_second: -1
  burst: 10
backends:
  - id: backend-1
    url: http://localhost:9001
`)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestConfig_RejectsInvalidRateLimitBurst(t *testing.T) {
	_, err := loadConfigErrorFromString(t, `
rate_limit:
  enabled: true
  requests_per_second: 10
  burst: -1
backends:
  - id: backend-1
    url: http://localhost:9001
`)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestConfig_AppliesDefaults(t *testing.T) {
	loadedConfig := loadConfigFromString(t, `
backends:
  - id: backend-1
    url: http://localhost:9001
`)

	if loadedConfig.Server.ListenAddress != DefaultListenAddress {
		t.Fatalf("expected listen address %q, got %q", DefaultListenAddress, loadedConfig.Server.ListenAddress)
	}
	if loadedConfig.Server.RequestTimeout != DefaultRequestTimeout {
		t.Fatalf("expected request timeout %s, got %s", DefaultRequestTimeout, loadedConfig.Server.RequestTimeout)
	}
	if loadedConfig.Server.ShutdownTimeout != DefaultShutdownTimeout {
		t.Fatalf("expected shutdown timeout %s, got %s", DefaultShutdownTimeout, loadedConfig.Server.ShutdownTimeout)
	}
	if loadedConfig.Balancer.Strategy != DefaultBalancerStrategy {
		t.Fatalf("expected strategy %q, got %q", DefaultBalancerStrategy, loadedConfig.Balancer.Strategy)
	}
	if loadedConfig.HealthCheck.Interval != DefaultHealthCheckInterval {
		t.Fatalf("expected health interval %s, got %s", DefaultHealthCheckInterval, loadedConfig.HealthCheck.Interval)
	}
	if loadedConfig.HealthCheck.Timeout != DefaultHealthCheckTimeout {
		t.Fatalf("expected health timeout %s, got %s", DefaultHealthCheckTimeout, loadedConfig.HealthCheck.Timeout)
	}
	if loadedConfig.HealthCheck.Path != DefaultHealthCheckPath {
		t.Fatalf("expected health path %q, got %q", DefaultHealthCheckPath, loadedConfig.HealthCheck.Path)
	}
	if loadedConfig.Retries.MaxAttempts != DefaultRetryMaxAttempts {
		t.Fatalf("expected max attempts %d, got %d", DefaultRetryMaxAttempts, loadedConfig.Retries.MaxAttempts)
	}
	if loadedConfig.RateLimit.Enabled != DefaultRateLimitEnabled {
		t.Fatalf("expected rate limit enabled %t, got %t", DefaultRateLimitEnabled, loadedConfig.RateLimit.Enabled)
	}
	if loadedConfig.RateLimit.RequestsPerSecond != DefaultRateLimitRPS {
		t.Fatalf("expected rate limit rps %f, got %f", DefaultRateLimitRPS, loadedConfig.RateLimit.RequestsPerSecond)
	}
	if loadedConfig.RateLimit.Burst != DefaultRateLimitBurst {
		t.Fatalf("expected rate limit burst %d, got %d", DefaultRateLimitBurst, loadedConfig.RateLimit.Burst)
	}
	if loadedConfig.Backends[0].Weight != 1 {
		t.Fatalf("expected backend weight %d, got %d", 1, loadedConfig.Backends[0].Weight)
	}
}

func loadConfigFromString(t *testing.T, content string) Config {
	t.Helper()

	loadedConfig, err := loadConfigErrorFromString(t, content)
	if err != nil {
		t.Fatal(err)
	}

	return loadedConfig
}

func loadConfigErrorFromString(t *testing.T, content string) (Config, error) {
	t.Helper()

	tempDirectory := t.TempDir()
	configPath := filepath.Join(tempDirectory, "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	return Load(configPath)
}
