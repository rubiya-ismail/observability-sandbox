package models

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestGatewayConfig() *GatewayConfig {
	return &GatewayConfig{
		Environment: "test",
		Services: map[string]ServiceConfig{
			"users": {
				Name:           "user-service",
				BaseURL:        "http://localhost:8081",
				Timeout:        Duration(5 * time.Second),
				Retries:        2,
				HealthEndpoint: "/health",
				APIVersion:     "v1",
			},
			"orders": {
				Name:           "order-service",
				BaseURL:        "http://localhost:8082",
				Timeout:        Duration(3 * time.Second),
				Retries:        1,
				HealthEndpoint: "/status",
				APIVersion:     "v1",
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

func TestNewGateway(t *testing.T) {
	config := createTestGatewayConfig()
	gateway, err := NewGateway(config, nil)

	assert.NoError(t, err)
	assert.NotNil(t, gateway)

	// Verify configuration was loaded
	userConfig, exists := gateway.GetServiceConfig("users")
	assert.True(t, exists)
	assert.Equal(t, "http://localhost:8081", userConfig.BaseURL)
	assert.Equal(t, Duration(5*time.Second), userConfig.Timeout)
	assert.Equal(t, "/health", userConfig.HealthEndpoint)
}

func TestNewGateway_NilConfig(t *testing.T) {
	gateway, err := NewGateway(nil, nil)

	// Should handle nil config gracefully or return error
	assert.Error(t, err)
	assert.Nil(t, gateway)
}

func TestGetServiceConfig(t *testing.T) {
	config := createTestGatewayConfig()
	gateway, err := NewGateway(config, nil)
	require.NoError(t, err)

	tests := []struct {
		name        string
		serviceName string
		expectFound bool
		expectedURL string
	}{
		{
			name:        "existing service",
			serviceName: "users",
			expectFound: true,
			expectedURL: "http://localhost:8081",
		},
		{
			name:        "another existing service",
			serviceName: "orders",
			expectFound: true,
			expectedURL: "http://localhost:8082",
		},
		{
			name:        "non-existent service",
			serviceName: "payments",
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, found := gateway.GetServiceConfig(tt.serviceName)

			assert.Equal(t, tt.expectFound, found)
			if tt.expectFound {
				assert.Equal(t, tt.expectedURL, config.BaseURL)
			}
		})
	}
}

func TestGetEnvironment(t *testing.T) {
	config := createTestGatewayConfig()
	gateway, err := NewGateway(config, nil)
	require.NoError(t, err)

	assert.Equal(t, "test", gateway.GetEnvironment())
}

func TestForwardRequest_ServiceNotFound(t *testing.T) {
	config := createTestGatewayConfig()
	gateway, err := NewGateway(config, nil)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	result := gateway.ForwardRequest(context.Background(), "nonexistent", w, req)

	assert.Equal(t, http.StatusNotFound, result.StatusCode)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "service nonexistent not found")
}

func TestForwardRequest_Success(t *testing.T) {
	// Create a test server to act as the upstream service
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer testServer.Close()

	// Create config with the test server URL
	config := &GatewayConfig{
		Environment: "test",
		Services: map[string]ServiceConfig{
			"test": {
				Name:           "test-service",
				BaseURL:        testServer.URL,
				Timeout:        Duration(5 * time.Second),
				Retries:        2,
				HealthEndpoint: "/health",
				APIVersion:     "v1",
			},
		},
	}

	gateway, err := NewGateway(config, nil)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	result := gateway.ForwardRequest(context.Background(), "test", w, req)

	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.NoError(t, result.Error)
	assert.Greater(t, result.Duration, time.Duration(0))
}

func TestForwardRequest_WithContext(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify context was passed through
		select {
		case <-r.Context().Done():
			t.Error("Context should not be cancelled")
		default:
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	config := &GatewayConfig{
		Environment: "test",
		Services: map[string]ServiceConfig{
			"test": {
				Name:    "test-service",
				BaseURL: testServer.URL,
				Timeout: Duration(5 * time.Second),
			},
		},
		Gateway: GatewayServerConfig{
			Port: 8080,
		},
	}

	gateway, err := NewGateway(config, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	result := gateway.ForwardRequest(ctx, "test", w, req)
	assert.Equal(t, http.StatusOK, result.StatusCode)
}

func TestCheckServiceHealth_ServiceNotConfigured(t *testing.T) {
	config := createTestGatewayConfig()
	gateway, err := NewGateway(config, nil)
	require.NoError(t, err)

	health := gateway.CheckServiceHealth(context.Background(), "nonexistent")

	assert.Equal(t, "unknown", health.Status)
	assert.Equal(t, "service not configured", health.Error)
}

func TestCheckServiceHealth_Success(t *testing.T) {
	// Create a test server with health endpoint
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer testServer.Close()

	config := &GatewayConfig{
		Environment: "test",
		Services: map[string]ServiceConfig{
			"test": {
				Name:           "test-service",
				BaseURL:        testServer.URL,
				Timeout:        Duration(5 * time.Second),
				Retries:        2,
				HealthEndpoint: "/api/v1/health",
				APIVersion:     "v1",
			},
		},
	}

	gateway, err := NewGateway(config, nil)
	require.NoError(t, err)

	health := gateway.CheckServiceHealth(context.Background(), "test")

	assert.Equal(t, "healthy", health.Status)
	assert.Empty(t, health.Error)
	assert.Greater(t, health.Latency, time.Duration(0))
}

func TestCheckServiceHealth_Unhealthy(t *testing.T) {
	// Create a test server that returns 500
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer testServer.Close()

	config := &GatewayConfig{
		Environment: "test",
		Services: map[string]ServiceConfig{
			"test": {
				Name:           "test-service",
				BaseURL:        testServer.URL,
				Timeout:        Duration(5 * time.Second),
				Retries:        2,
				HealthEndpoint: "/health",
				APIVersion:     "v1",
			},
		},
	}

	gateway, err := NewGateway(config, nil)
	require.NoError(t, err)

	health := gateway.CheckServiceHealth(context.Background(), "test")

	assert.Equal(t, "unhealthy", health.Status)
	assert.Contains(t, health.Error, "HTTP 500")
}

func TestCheckServiceHealth_NetworkError(t *testing.T) {
	config := &GatewayConfig{
		Environment: "test",
		Services: map[string]ServiceConfig{
			"test": {
				Name:           "test-service",
				BaseURL:        "http://localhost:99999", // Invalid port
				Timeout:        Duration(1 * time.Second),
				HealthEndpoint: "/health",
			},
		},
		Gateway: GatewayServerConfig{
			Port: 8080,
		},
	}

	gateway, err := NewGateway(config, nil)
	require.NoError(t, err)

	health := gateway.CheckServiceHealth(context.Background(), "test")

	assert.Equal(t, "unhealthy", health.Status)
	assert.NotEmpty(t, health.Error)
	assert.Greater(t, health.Latency, time.Duration(0))
}

func TestGetAllServiceHealth(t *testing.T) {
	config := createTestGatewayConfig()
	gateway, err := NewGateway(config, nil)
	require.NoError(t, err)

	healthMap := gateway.GetAllServiceHealth(context.Background())

	assert.Len(t, healthMap, 2)
	assert.Contains(t, healthMap, "users")
	assert.Contains(t, healthMap, "orders")

	// Since we don't have actual services running, they should be unhealthy
	assert.Equal(t, "unhealthy", healthMap["users"].Status)
	assert.Equal(t, "unhealthy", healthMap["orders"].Status)
}

func TestCheckRateLimit(t *testing.T) {
	config := createTestGatewayConfig()
	gateway, err := NewGateway(config, nil)
	require.NoError(t, err)

	// Simple test - current implementation always returns true
	allowed := gateway.CheckRateLimit("127.0.0.1", "/users/123")
	assert.True(t, allowed)
}

// Test concurrent access to gateway
func TestGateway_ConcurrentAccess(t *testing.T) {
	config := createTestGatewayConfig()
	gateway, err := NewGateway(config, nil)
	require.NoError(t, err)

	// Test concurrent GetServiceConfig calls
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			config, found := gateway.GetServiceConfig("users")
			assert.True(t, found)
			assert.Equal(t, "http://localhost:8081", config.BaseURL)
		}()
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}
