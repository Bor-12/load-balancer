package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	StrategyRoundRobin         = "round_robin"
	StrategyWeightedRoundRobin = "weighted_round_robin"
	StrategyLeastConnections   = "least_connections"
	DefaultListenAddress       = ":8080"
	DefaultRequestTimeout      = 5 * time.Second
	DefaultBalancerStrategy    = StrategyRoundRobin
	DefaultHealthCheckEnabled  = true
	DefaultHealthCheckInterval = 2 * time.Second
	DefaultHealthCheckTimeout  = 500 * time.Millisecond
	DefaultHealthCheckPath     = "/health"
	DefaultRetryMaxAttempts    = 2
)

type Config struct {
	Server      ServerConfig
	Balancer    BalancerConfig
	HealthCheck HealthCheckConfig
	Retries     RetryConfig
	Backends    []BackendConfig
}

type ServerConfig struct {
	ListenAddress  string
	RequestTimeout time.Duration
}

type BalancerConfig struct {
	Strategy string `yaml:"strategy"`
}

type HealthCheckConfig struct {
	Enabled  bool
	Interval time.Duration
	Timeout  time.Duration
	Path     string
}

type RetryConfig struct {
	MaxAttempts int `yaml:"max_attempts"`
}

type BackendConfig struct {
	ID     string `yaml:"id"`
	URL    string `yaml:"url"`
	Weight int    `yaml:"weight"`
}

type rawConfig struct {
	Server      rawServerConfig      `yaml:"server"`
	Balancer    BalancerConfig       `yaml:"balancer"`
	HealthCheck rawHealthCheckConfig `yaml:"health_check"`
	Retries     RetryConfig          `yaml:"retries"`
	Backends    []BackendConfig      `yaml:"backends"`
}

type rawServerConfig struct {
	ListenAddress  string `yaml:"listen_address"`
	RequestTimeout string `yaml:"request_timeout"`
}

type rawHealthCheckConfig struct {
	Enabled  *bool  `yaml:"enabled"`
	Interval string `yaml:"interval"`
	Timeout  string `yaml:"timeout"`
	Path     string `yaml:"path"`
}

func Load(path string) (Config, error) {
	fileContent, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var rawConfig rawConfig
	if err := yaml.Unmarshal(fileContent, &rawConfig); err != nil {
		return Config{}, err
	}

	loadedConfig, err := parseRawConfig(rawConfig)
	if err != nil {
		return Config{}, err
	}

	if err := Validate(loadedConfig); err != nil {
		return Config{}, err
	}

	return loadedConfig, nil
}

func Validate(loadedConfig Config) error {
	if loadedConfig.Balancer.Strategy != StrategyRoundRobin &&
		loadedConfig.Balancer.Strategy != StrategyWeightedRoundRobin &&
		loadedConfig.Balancer.Strategy != StrategyLeastConnections {
		return fmt.Errorf("unsupported balancer strategy %q", loadedConfig.Balancer.Strategy)
	}

	if loadedConfig.Server.RequestTimeout <= 0 {
		return errors.New("server request timeout must be greater than zero")
	}

	if loadedConfig.HealthCheck.Interval <= 0 {
		return errors.New("health check interval must be greater than zero")
	}

	if loadedConfig.HealthCheck.Timeout <= 0 {
		return errors.New("health check timeout must be greater than zero")
	}

	if loadedConfig.Retries.MaxAttempts < 1 {
		return errors.New("retry max attempts must be at least 1")
	}

	if len(loadedConfig.Backends) == 0 {
		return errors.New("at least one backend is required")
	}

	backendIDs := make(map[string]bool, len(loadedConfig.Backends))
	for _, configuredBackend := range loadedConfig.Backends {
		if configuredBackend.ID == "" {
			return errors.New("backend ID is required")
		}
		if backendIDs[configuredBackend.ID] {
			return fmt.Errorf("duplicate backend ID %q", configuredBackend.ID)
		}
		backendIDs[configuredBackend.ID] = true

		if err := validateBackendURL(configuredBackend.URL); err != nil {
			return err
		}

		if configuredBackend.Weight <= 0 {
			return errors.New("backend weight must be greater than zero")
		}
	}

	return nil
}

func parseRawConfig(rawConfig rawConfig) (Config, error) {
	requestTimeout, err := parseDurationOrDefault(rawConfig.Server.RequestTimeout, DefaultRequestTimeout)
	if err != nil {
		return Config{}, err
	}

	healthInterval, err := parseDurationOrDefault(rawConfig.HealthCheck.Interval, DefaultHealthCheckInterval)
	if err != nil {
		return Config{}, err
	}

	healthTimeout, err := parseDurationOrDefault(rawConfig.HealthCheck.Timeout, DefaultHealthCheckTimeout)
	if err != nil {
		return Config{}, err
	}

	healthEnabled := DefaultHealthCheckEnabled
	if rawConfig.HealthCheck.Enabled != nil {
		healthEnabled = *rawConfig.HealthCheck.Enabled
	}

	loadedConfig := Config{
		Server: ServerConfig{
			ListenAddress:  valueOrDefault(rawConfig.Server.ListenAddress, DefaultListenAddress),
			RequestTimeout: requestTimeout,
		},
		Balancer: BalancerConfig{
			Strategy: valueOrDefault(rawConfig.Balancer.Strategy, DefaultBalancerStrategy),
		},
		HealthCheck: HealthCheckConfig{
			Enabled:  healthEnabled,
			Interval: healthInterval,
			Timeout:  healthTimeout,
			Path:     valueOrDefault(rawConfig.HealthCheck.Path, DefaultHealthCheckPath),
		},
		Retries: RetryConfig{
			MaxAttempts: rawConfig.Retries.MaxAttempts,
		},
		Backends: rawConfig.Backends,
	}

	if loadedConfig.Retries.MaxAttempts == 0 {
		loadedConfig.Retries.MaxAttempts = DefaultRetryMaxAttempts
	}
	for index := range loadedConfig.Backends {
		if loadedConfig.Backends[index].Weight == 0 {
			loadedConfig.Backends[index].Weight = 1
		}
	}

	return loadedConfig, nil
}

func parseDurationOrDefault(rawDuration string, defaultDuration time.Duration) (time.Duration, error) {
	if rawDuration == "" {
		return defaultDuration, nil
	}

	return time.ParseDuration(rawDuration)
}

func valueOrDefault(value string, defaultValue string) string {
	if value == "" {
		return defaultValue
	}

	return value
}

func validateBackendURL(rawURL string) error {
	if rawURL == "" {
		return errors.New("backend URL is required")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return err
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("backend URL scheme must be http or https")
	}

	if parsedURL.Host == "" {
		return errors.New("backend URL host is required")
	}

	return nil
}
