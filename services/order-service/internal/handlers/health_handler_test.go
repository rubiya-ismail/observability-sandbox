package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"shared/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHealthHandler_CreateWithStartTime(t *testing.T) {
	// Arrange
	beforeCreation := time.Now()

	// Act
	handler := NewHealthHandler()

	// Assert
	assert.NotNil(t, handler)
	assert.False(t, handler.startTime.IsZero())
	assert.True(t, handler.startTime.After(beforeCreation.Add(-time.Second)))
	assert.True(t, handler.startTime.Before(time.Now().Add(time.Second)))
}

func TestHealthCheck_WithCorrectStatusAndStruct(t *testing.T) {
	// Arrange
	handler := NewHealthHandler()
	router := util.SetupTestRouter()

	const url = "/health"
	router.GET(url, handler.HealthCheck)

	w := util.ProcessTestRequest(http.MethodGet, url, nil, router)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify response structure
	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "order-service", response["service"])
	assert.Equal(t, "1.0.0", response["version"])
	assert.Contains(t, response, "timestamp")
	assert.Contains(t, response, "uptime")

	// Verify timestamp is recent
	timestampStr, ok := response["timestamp"].(string)
	require.True(t, ok)
	timestamp, err := time.Parse(time.RFC3339Nano, timestampStr)
	require.NoError(t, err)
	assert.True(t, time.Since(timestamp) < time.Minute)

	// Verify uptime format
	uptime, ok := response["uptime"].(string)
	require.True(t, ok)
	assert.NotEmpty(t, uptime)
	assert.Contains(t, uptime, "s") // Should contain seconds unit
}

func TestHealthCheck_UptimeIncreaseOverTime(t *testing.T) {
	// Arrange
	handler := NewHealthHandler()
	router := util.SetupTestRouter()

	const url = "/health"
	router.GET(url, handler.HealthCheck)

	// Act - First call
	w1 := util.ProcessTestRequest(http.MethodGet, url, nil, router)

	var response1 map[string]interface{}
	err := json.Unmarshal(w1.Body.Bytes(), &response1)
	require.NoError(t, err)
	uptime1 := response1["uptime"].(string)

	// Wait a small amount
	time.Sleep(10 * time.Millisecond)

	// Act - Second call
	w2 := util.ProcessTestRequest(http.MethodGet, url, nil, router)

	var response2 map[string]interface{}
	err = json.Unmarshal(w2.Body.Bytes(), &response2)
	require.NoError(t, err)
	uptime2 := response2["uptime"].(string)

	// Assert
	assert.NotEqual(t, uptime1, uptime2, "Uptime should increase between calls")
}

func TestReadinessCheck_Ready(t *testing.T) {
	// Arrange
	handler := NewHealthHandler()
	router := util.SetupTestRouter()

	const url = "/ready"
	router.GET(url, handler.ReadinessCheck)

	w := util.ProcessTestRequest(http.MethodGet, url, nil, router)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	expected := map[string]interface{}{
		"status":  "ready",
		"service": "order-service",
	}

	assert.Equal(t, expected, response)
}

func TestLivenessCheck_Alive(t *testing.T) {
	// Arrange
	handler := NewHealthHandler()
	router := util.SetupTestRouter()

	const url = "/live"
	router.GET(url, handler.LivenessCheck)

	w := util.ProcessTestRequest(http.MethodGet, url, nil, router)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	expected := map[string]interface{}{
		"status":  "alive",
		"service": "order-service",
	}

	assert.Equal(t, expected, response)
}

func TestHealthHandler_MultInstanceDiffStartTime(t *testing.T) {
	// Arrange & Act
	handler1 := NewHealthHandler()
	time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	handler2 := NewHealthHandler()

	// Assert
	assert.True(t, handler2.startTime.After(handler1.startTime),
		"Second handler should have later start time")
}

func TestHealthHandler_AsyncHealthChecks(t *testing.T) {
	// Arrange
	handler := NewHealthHandler()
	router := util.SetupTestRouter()

	const url = "/health"
	router.GET(url, handler.HealthCheck)

	const numGoroutines = 10
	responses := make(chan *httptest.ResponseRecorder, numGoroutines)

	// Act - Concurrent requests
	for range numGoroutines {
		go func() {
			w := util.ProcessTestRequest(http.MethodGet, url, nil, router)
			responses <- w
		}()
	}

	// Assert - All responses should be successful
	for range numGoroutines {
		w := <-responses
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "healthy", response["status"])
		assert.Equal(t, "order-service", response["service"])
	}
}
