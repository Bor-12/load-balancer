package main

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultConfigPath = "config.json"

type config struct {
	ListenAddress       string   `json:"listen_address"`
	BalancerStrategy    string   `json:"balancer_strategy"`
	BackendURLs         []string `json:"backend_urls"`
	BackendWeights      []int    `json:"backend_weights"`
	RequestTimeout      string   `json:"request_timeout"`
	HealthCheckPath     string   `json:"health_check_path"`
	HealthCheckInterval string   `json:"health_check_interval"`
	HealthCheckTimeout  string   `json:"health_check_timeout"`
	MaxAttempts         int      `json:"max_attempts"`
}

type runtimeConfig struct {
	ListenAddress       string
	BalancerStrategy    string
	BackendURLs         []string
	BackendWeights      []int
	RequestTimeout      time.Duration
	HealthCheckPath     string
	HealthCheckInterval time.Duration
	HealthCheckTimeout  time.Duration
	MaxAttempts         int
}

func loadConfig() (runtimeConfig, error) {
	loadedConfig := defaultConfig()
	configPath := configPathFromEnvironment()

	fileConfig, err := loadConfigFile(configPath)
	if err != nil {
		return runtimeConfig{}, err
	}

	loadedConfig = mergeConfig(loadedConfig, fileConfig)
	loadedConfig = mergeConfig(loadedConfig, configFromEnvironment())

	if len(loadedConfig.BackendURLs) == 0 {
		return runtimeConfig{}, errors.New("at least one backend URL is required")
	}

	return parseRuntimeConfig(loadedConfig)
}

func defaultConfig() config {
	return config{
		ListenAddress:       ":8080",
		BalancerStrategy:    "round_robin",
		RequestTimeout:      "5s",
		HealthCheckPath:     "/health",
		HealthCheckInterval: "2s",
		HealthCheckTimeout:  "500ms",
		MaxAttempts:         2,
	}
}

func configPathFromEnvironment() string {
	configPath := strings.TrimSpace(os.Getenv("CONFIG_FILE"))
	if configPath == "" {
		return defaultConfigPath
	}

	return configPath
}

func loadConfigFile(configPath string) (config, error) {
	file, err := os.Open(configPath)
	if errors.Is(err, os.ErrNotExist) && configPath == defaultConfigPath {
		return config{}, nil
	}
	if err != nil {
		return config{}, err
	}
	defer file.Close()

	var loadedConfig config
	if err := json.NewDecoder(file).Decode(&loadedConfig); err != nil {
		return config{}, err
	}

	return loadedConfig, nil
}

func configFromEnvironment() config {
	return config{
		ListenAddress:       strings.TrimSpace(os.Getenv("LISTEN_ADDRESS")),
		BalancerStrategy:    strings.TrimSpace(os.Getenv("BALANCER_STRATEGY")),
		BackendURLs:         backendURLsFromEnvironment(),
		BackendWeights:      backendWeightsFromEnvironment(),
		RequestTimeout:      strings.TrimSpace(os.Getenv("REQUEST_TIMEOUT")),
		HealthCheckPath:     strings.TrimSpace(os.Getenv("HEALTH_CHECK_PATH")),
		HealthCheckInterval: strings.TrimSpace(os.Getenv("HEALTH_CHECK_INTERVAL")),
		HealthCheckTimeout:  strings.TrimSpace(os.Getenv("HEALTH_CHECK_TIMEOUT")),
		MaxAttempts:         intFromEnvironment("MAX_ATTEMPTS"),
	}
}

func backendURLsFromEnvironment() []string {
	rawBackendURLs := os.Getenv("BACKEND_URLS")
	if rawBackendURLs == "" {
		rawBackendURLs = os.Getenv("BACKEND_URL")
	}

	return splitBackendURLs(rawBackendURLs)
}

func backendWeightsFromEnvironment() []int {
	return splitInts(os.Getenv("BACKEND_WEIGHTS"))
}

func splitBackendURLs(rawBackendURLs string) []string {
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

func splitInts(rawValues string) []int {
	values := strings.Split(rawValues, ",")
	parsedValues := make([]int, 0, len(values))
	for _, rawValue := range values {
		value, err := strconv.Atoi(strings.TrimSpace(rawValue))
		if err == nil {
			parsedValues = append(parsedValues, value)
		}
	}

	return parsedValues
}

func intFromEnvironment(name string) int {
	rawValue := strings.TrimSpace(os.Getenv(name))
	if rawValue == "" {
		return 0
	}

	value, err := strconv.Atoi(rawValue)
	if err != nil {
		return 0
	}

	return value
}

func mergeConfig(baseConfig config, overrideConfig config) config {
	if overrideConfig.ListenAddress != "" {
		baseConfig.ListenAddress = overrideConfig.ListenAddress
	}
	if overrideConfig.BalancerStrategy != "" {
		baseConfig.BalancerStrategy = overrideConfig.BalancerStrategy
	}
	if len(overrideConfig.BackendURLs) > 0 {
		baseConfig.BackendURLs = overrideConfig.BackendURLs
	}
	if len(overrideConfig.BackendWeights) > 0 {
		baseConfig.BackendWeights = overrideConfig.BackendWeights
	}
	if overrideConfig.RequestTimeout != "" {
		baseConfig.RequestTimeout = overrideConfig.RequestTimeout
	}
	if overrideConfig.HealthCheckPath != "" {
		baseConfig.HealthCheckPath = overrideConfig.HealthCheckPath
	}
	if overrideConfig.HealthCheckInterval != "" {
		baseConfig.HealthCheckInterval = overrideConfig.HealthCheckInterval
	}
	if overrideConfig.HealthCheckTimeout != "" {
		baseConfig.HealthCheckTimeout = overrideConfig.HealthCheckTimeout
	}
	if overrideConfig.MaxAttempts > 0 {
		baseConfig.MaxAttempts = overrideConfig.MaxAttempts
	}

	return baseConfig
}

func parseRuntimeConfig(loadedConfig config) (runtimeConfig, error) {
	requestTimeout, err := time.ParseDuration(loadedConfig.RequestTimeout)
	if err != nil {
		return runtimeConfig{}, err
	}

	healthCheckInterval, err := time.ParseDuration(loadedConfig.HealthCheckInterval)
	if err != nil {
		return runtimeConfig{}, err
	}

	healthCheckTimeout, err := time.ParseDuration(loadedConfig.HealthCheckTimeout)
	if err != nil {
		return runtimeConfig{}, err
	}

	return runtimeConfig{
		ListenAddress:       loadedConfig.ListenAddress,
		BalancerStrategy:    loadedConfig.BalancerStrategy,
		BackendURLs:         loadedConfig.BackendURLs,
		BackendWeights:      loadedConfig.BackendWeights,
		RequestTimeout:      requestTimeout,
		HealthCheckPath:     loadedConfig.HealthCheckPath,
		HealthCheckInterval: healthCheckInterval,
		HealthCheckTimeout:  healthCheckTimeout,
		MaxAttempts:         loadedConfig.MaxAttempts,
	}, nil
}
