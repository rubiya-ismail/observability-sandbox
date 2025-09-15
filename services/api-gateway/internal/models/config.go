package models

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Duration wrapper for custom JSON unmarshaling
type Duration time.Duration

func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	duration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	*d = Duration(duration)
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// Convert to time.Duration for use in standard library
func (d Duration) ToDuration() time.Duration {
	return time.Duration(d)
}

// ServiceConfig with custom Duration type
type ServiceConfig struct {
	Name            string   `json:"name"`
	BaseURL         string   `json:"base_url"`
	Timeout         Duration `json:"timeout"`
	Retries         int      `json:"retries"`
	ServiceEndpoint string   `json:"service_endpoint"`
	HealthEndpoint  string   `json:"health_endpoint"`
	MetricsEndpoint string   `json:"metrics_endpoint"`
	APIVersion      string   `json:"api_version"`
}

// RateLimitConfig with custom Duration type
type RateLimitConfig struct {
	RequestsPerMinute int      `json:"requests_per_minute"`
	WindowSize        Duration `json:"window_size"`
}

type GatewayServerConfig struct {
	Port         int      `json:"port"`
	ReadTimeout  Duration `json:"read_timeout"`
	WriteTimeout Duration `json:"write_timeout"`
}

// GatewayConfig with custom Duration type
type GatewayConfig struct {
	Environment string                   `json:"environment"`
	Services    map[string]ServiceConfig `json:"services"`
	RateLimits  RateLimitConfig          `json:"rate_limits"`
	Gateway     GatewayServerConfig      `json:"gateway"`
}

// FormatServiceURL returns full service URL
// - This excludes query parameters, etc
func FormatServiceURL(svcConfig ServiceConfig, urlPath string) string {
	// Ensure BaseURL does not end with "/"
	baseURL := strings.TrimRight(svcConfig.BaseURL, "/")

	// Ensure path starts with "/"
	servicePath := urlPath
	if !strings.HasPrefix(servicePath, "/") {
		servicePath = "/" + servicePath
	}

	// Concatenate base URL + path
	return baseURL + servicePath
}

// LoadConfig reads a JSON config file and returns a GatewayConfig
func LoadConfig(path string) (*GatewayConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %s: %w", path, err)
	}
	defer file.Close()

	var cfg GatewayConfig
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config JSON: %w", err)
	}

	return &cfg, nil
}

// Convenience methods for accessing time.Duration values
func (s ServiceConfig) GetTimeout() time.Duration {
	return s.Timeout.ToDuration()
}

func (r RateLimitConfig) GetWindowSize() time.Duration {
	return r.WindowSize.ToDuration()
}

func (g GatewayConfig) GetReadTimeout() time.Duration {
	return g.Gateway.ReadTimeout.ToDuration()
}

func (g GatewayConfig) GetWriteTimeout() time.Duration {
	return g.Gateway.WriteTimeout.ToDuration()
}
