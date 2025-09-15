package models

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function for creating test config
func createTestConfig() *GatewayConfig {
	return &GatewayConfig{
		Environment: "test",
		Services: map[string]ServiceConfig{
			"users": {
				Name:            "user-service",
				BaseURL:         "http://localhost:8081",
				Timeout:         Duration(5 * time.Second),
				Retries:         2,
				ServiceEndpoint: "/users",
				HealthEndpoint:  "/health",
				MetricsEndpoint: "/metrics",
				APIVersion:      "v1",
			},
			"orders": {
				Name:            "order-service",
				BaseURL:         "http://localhost:8082",
				Timeout:         Duration(3 * time.Second),
				Retries:         1,
				ServiceEndpoint: "/orders",
				HealthEndpoint:  "/health",
				MetricsEndpoint: "/metrics",
				APIVersion:      "v1",
			},
		},
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 100,
			WindowSize:        Duration(time.Minute),
		},
		Gateway: GatewayServerConfig{
			Port:         8080,
			ReadTimeout:  Duration(10 * time.Second),
			WriteTimeout: Duration(10 * time.Second),
		},
	}
}

// Test Duration custom unmarshaling
func TestDuration_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Duration
		wantErr  bool
	}{
		{
			name:     "valid seconds",
			input:    `"30s"`,
			expected: Duration(30 * time.Second),
			wantErr:  false,
		},
		{
			name:     "valid minutes",
			input:    `"5m"`,
			expected: Duration(5 * time.Minute),
			wantErr:  false,
		},
		{
			name:     "valid milliseconds",
			input:    `"500ms"`,
			expected: Duration(500 * time.Millisecond),
			wantErr:  false,
		},
		{
			name:    "invalid format",
			input:   `"invalid"`,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   `""`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Duration
			err := json.Unmarshal([]byte(tt.input), &d)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, d)
			}
		})
	}
}

func TestDuration_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		duration Duration
		expected string
	}{
		{
			name:     "seconds",
			duration: Duration(30 * time.Second),
			expected: `"30s"`,
		},
		{
			name:     "minutes",
			duration: Duration(5 * time.Minute),
			expected: `"5m0s"`,
		},
		{
			name:     "milliseconds",
			duration: Duration(500 * time.Millisecond),
			expected: `"500ms"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.duration)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, string(data))
		})
	}
}

func TestDuration_ToDuration(t *testing.T) {
	d := Duration(45 * time.Second)
	result := d.ToDuration()
	assert.Equal(t, 45*time.Second, result)
}

// Test ServiceConfig
func TestServiceConfig_JSONMarshaling(t *testing.T) {
	config := ServiceConfig{
		Name:           "test-service",
		BaseURL:        "http://localhost:8080",
		Timeout:        Duration(30 * time.Second),
		Retries:        3,
		HealthEndpoint: "/health",
		APIVersion:     "v1",
	}

	// Test marshaling
	data, err := json.Marshal(config)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"name":"test-service"`)
	assert.Contains(t, string(data), `"timeout":"30s"`)

	// Test unmarshaling
	var unmarshaled ServiceConfig
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, config.Name, unmarshaled.Name)
	assert.Equal(t, config.BaseURL, unmarshaled.BaseURL)
	assert.Equal(t, config.Timeout, unmarshaled.Timeout)
	assert.Equal(t, config.Retries, unmarshaled.Retries)
	assert.Equal(t, config.HealthEndpoint, unmarshaled.HealthEndpoint)
	assert.Equal(t, config.APIVersion, unmarshaled.APIVersion)
}

func TestServiceConfig_GetTimeout(t *testing.T) {
	config := ServiceConfig{
		Timeout: Duration(45 * time.Second),
	}

	assert.Equal(t, 45*time.Second, config.GetTimeout())
}

// Test RateLimitConfig
func TestRateLimitConfig_JSONMarshaling(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerMinute: 500,
		WindowSize:        Duration(2 * time.Minute),
	}

	// Test marshaling
	data, err := json.Marshal(config)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"requests_per_minute":500`)
	assert.Contains(t, string(data), `"window_size":"2m0s"`)

	// Test unmarshaling
	var unmarshaled RateLimitConfig
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, 500, unmarshaled.RequestsPerMinute)
	assert.Equal(t, Duration(2*time.Minute), unmarshaled.WindowSize)
}

func TestRateLimitConfig_GetWindowSize(t *testing.T) {
	config := RateLimitConfig{
		WindowSize: Duration(90 * time.Second),
	}

	assert.Equal(t, 90*time.Second, config.GetWindowSize())
}

// Test GatewayServerConfig
func TestGatewayServerConfig_JSONMarshaling(t *testing.T) {
	config := GatewayServerConfig{
		Port:         9090,
		ReadTimeout:  Duration(15 * time.Second),
		WriteTimeout: Duration(20 * time.Second),
	}

	// Test marshaling
	data, err := json.Marshal(config)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"port":9090`)
	assert.Contains(t, string(data), `"read_timeout":"15s"`)
	assert.Contains(t, string(data), `"write_timeout":"20s"`)

	// Test unmarshaling
	var unmarshaled GatewayServerConfig
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, 9090, unmarshaled.Port)
	assert.Equal(t, Duration(15*time.Second), unmarshaled.ReadTimeout)
	assert.Equal(t, Duration(20*time.Second), unmarshaled.WriteTimeout)
}

// Test GatewayConfig
func TestGatewayConfig_JSONMarshaling(t *testing.T) {
	config := createTestConfig()

	// Test marshaling
	data, err := json.Marshal(config)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"environment":"test"`)

	// Test unmarshaling
	var unmarshaled GatewayConfig
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, config.Environment, unmarshaled.Environment)
	assert.Len(t, unmarshaled.Services, 2)
	assert.Equal(t, config.RateLimits.RequestsPerMinute, unmarshaled.RateLimits.RequestsPerMinute)
	assert.Equal(t, config.Gateway.Port, unmarshaled.Gateway.Port)
}

func TestGatewayConfig_GetTimeouts(t *testing.T) {
	config := createTestConfig()

	assert.Equal(t, 10*time.Second, config.GetReadTimeout())
	assert.Equal(t, 10*time.Second, config.GetWriteTimeout())
}

// TestFormatServiceURL
func TestFormatServiceURL(t *testing.T) {

	const baseUrl = "http://localhost:8080"

	tests := []struct {
		name     string
		config   ServiceConfig
		path     string
		expected string
	}{
		{
			name:     "Versionless API",
			config:   ServiceConfig{BaseURL: baseUrl},
			path:     "/users",
			expected: "http://localhost:8080/users",
		},
		{
			name:     "Versioned API",
			config:   ServiceConfig{BaseURL: baseUrl},
			path:     "/api/v1/users",
			expected: "http://localhost:8080/api/v1/users",
		},
		{
			name:     "base URL with trailing slash",
			config:   ServiceConfig{BaseURL: baseUrl + "/"},
			path:     "/users",
			expected: "http://localhost:8080/users",
		},
		{
			name:     "path without leading slash",
			config:   ServiceConfig{BaseURL: baseUrl},
			path:     "users",
			expected: "http://localhost:8080/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatServiceURL(tt.config, tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test LoadConfig
func TestLoadConfig_ValidFile(t *testing.T) {
	// Create temporary config file
	configData := `{
		"environment": "test",
		"services": {
			"users": {
				"name": "user-service",
				"base_url": "http://localhost:8081",
				"timeout": "30s",
				"retries": 3,
				"health_endpoint": "/health",
				"api_version": "v1"
			}
		},
		"rate_limits": {
			"requests_per_minute": 1000,
			"window_size": "1m"
		},
		"gateway": {
			"port": 8080,
			"read_timeout": "30s",
			"write_timeout": "30s"
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.json")
	err := os.WriteFile(configPath, []byte(configData), 0644)
	require.NoError(t, err)

	// Test loading
	config, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, config)

	assert.Equal(t, "test", config.Environment)
	assert.Len(t, config.Services, 1)
	assert.Equal(t, "user-service", config.Services["users"].Name)
	assert.Equal(t, Duration(30*time.Second), config.Services["users"].Timeout)
	assert.Equal(t, 1000, config.RateLimits.RequestsPerMinute)
	assert.Equal(t, Duration(time.Minute), config.RateLimits.WindowSize)
	assert.Equal(t, 8080, config.Gateway.Port)
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.json")
	err := os.WriteFile(configPath, []byte(`{invalid json`), 0644)
	require.NoError(t, err)

	config, err := LoadConfig(configPath)
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to decode config JSON")
}

func TestLoadConfig_MissingFile(t *testing.T) {
	config, err := LoadConfig("/nonexistent/path/config.json")
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to open config file")
}

func TestLoadConfig_InvalidDuration(t *testing.T) {
	configData := `{
		"environment": "test",
		"services": {
			"users": {
				"name": "user-service",
				"base_url": "http://localhost:8081",
				"timeout": "invalid-duration",
				"retries": 3,
				"health_endpoint": "/health",
				"api_version": "v1"
			}
		},
		"rate_limits": {
			"requests_per_minute": 1000,
			"window_size": "1m"
		},
		"gateway": {
			"port": 8080,
			"read_timeout": "30s",
			"write_timeout": "30s"
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid_duration.json")
	err := os.WriteFile(configPath, []byte(configData), 0644)
	require.NoError(t, err)

	config, err := LoadConfig(configPath)
	assert.Error(t, err)
	assert.Nil(t, config)
}

// Test complete config roundtrip
func TestGatewayConfig_CompleteJSONRoundtrip(t *testing.T) {
	original := createTestConfig()

	// Marshal to JSON
	data, err := json.Marshal(original)
	assert.NoError(t, err)

	// Unmarshal back
	var parsed GatewayConfig
	err = json.Unmarshal(data, &parsed)
	assert.NoError(t, err)

	// Verify all components
	assert.Equal(t, original.Environment, parsed.Environment)
	assert.Equal(t, len(original.Services), len(parsed.Services))

	for name, originalService := range original.Services {
		parsedService, exists := parsed.Services[name]
		assert.True(t, exists)
		assert.Equal(t, originalService.Name, parsedService.Name)
		assert.Equal(t, originalService.BaseURL, parsedService.BaseURL)
		assert.Equal(t, originalService.Timeout, parsedService.Timeout)
		assert.Equal(t, originalService.Retries, parsedService.Retries)
		assert.Equal(t, originalService.HealthEndpoint, parsedService.HealthEndpoint)
		assert.Equal(t, originalService.APIVersion, parsedService.APIVersion)
	}

	assert.Equal(t, original.RateLimits.RequestsPerMinute, parsed.RateLimits.RequestsPerMinute)
	assert.Equal(t, original.RateLimits.WindowSize, parsed.RateLimits.WindowSize)
	assert.Equal(t, original.Gateway.Port, parsed.Gateway.Port)
	assert.Equal(t, original.Gateway.ReadTimeout, parsed.Gateway.ReadTimeout)
	assert.Equal(t, original.Gateway.WriteTimeout, parsed.Gateway.WriteTimeout)
}
