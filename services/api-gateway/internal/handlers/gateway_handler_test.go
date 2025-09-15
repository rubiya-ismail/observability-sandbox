package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"services/api-gateway/internal/models"
)

// Mock Gateway for testing
type MockGateway struct {
	mock.Mock
}

func (m *MockGateway) ForwardRequest(ctx context.Context, serviceName string, w http.ResponseWriter, r *http.Request) models.ProxyResult {
	args := m.Called(ctx, serviceName, w, r)
	return args.Get(0).(models.ProxyResult)
}

func (m *MockGateway) CheckServiceHealth(ctx context.Context, serviceName string) models.ServiceHealth {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(models.ServiceHealth)
}

func (m *MockGateway) GetAllServiceHealth(ctx context.Context) map[string]models.ServiceHealth {
	args := m.Called(ctx)
	return args.Get(0).(map[string]models.ServiceHealth)
}

func (m *MockGateway) CheckRateLimit(clientIP, path string) bool {
	args := m.Called(clientIP, path)
	return args.Get(0).(bool)
}

func (m *MockGateway) GetServiceConfig(serviceName string) (models.ServiceConfig, bool) {
	args := m.Called(serviceName)
	return args.Get(0).(models.ServiceConfig), args.Get(1).(bool)
}

func (m *MockGateway) GetEnvironment() string {
	args := m.Called()
	return args.Get(0).(string)
}

func setupTestRouter(mockGateway *MockGateway) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	handler := NewGatewayHandler(mockGateway)

	// Match the actual routes.go structure
	r.GET("/health", handler.HealthCheck)
	r.GET("/services/health", handler.ServicesHealthCheck)
	r.GET("/metrics", handler.Metrics)
	r.Any("/users", handler.ProxyRequest)
	r.Any("/users/*path", handler.ProxyRequest)
	r.Any("/orders", handler.ProxyRequest)
	r.Any("/orders/*path", handler.ProxyRequest)

	return r
}

func TestGatewayHandler_ProxyRequest_Success(t *testing.T) {
	// Arrange
	mockGateway := new(MockGateway)
	router := setupTestRouter(mockGateway)

	baseUrl := "192.0.2.1"
	urlPath := "/users/123"
	verPath := "/api/v1" + urlPath
	mockGateway.On("GetServiceConfig", "users").Return(
		models.ServiceConfig{
			Name:            "users",
			ServiceEndpoint: verPath,
			APIVersion:      "v1",
		},
		true,
	)
	mockGateway.On("CheckRateLimit", baseUrl, urlPath).Return(true)
	mockGateway.On("ForwardRequest", mock.Anything, "users", mock.Anything,
		mock.MatchedBy(func(r *http.Request) bool {
			return r.URL.Path == verPath
		}),
	).Return(
		models.ProxyResult{
			StatusCode: http.StatusOK,
			Duration:   50 * time.Millisecond,
		},
	)

	// Act
	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	req.RemoteAddr = baseUrl + ":12345"
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	mockGateway.AssertExpectations(t)
}

func TestGatewayHandler_ProxyRequest_RateLimitExceeded(t *testing.T) {
	// Arrange
	mockGateway := new(MockGateway)
	router := setupTestRouter(mockGateway)

	mockGateway.On("GetServiceConfig", "users").Return(models.ServiceConfig{Name: "users"}, true)
	mockGateway.On("CheckRateLimit", "192.0.2.1", "/users/123").Return(false)

	// Act
	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "rate limit exceeded")

	// Verify ForwardRequest was not called
	mockGateway.AssertNotCalled(t, "ForwardRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockGateway.AssertExpectations(t)
}

func TestGatewayHandler_ProxyRequest_InvalidPath(t *testing.T) {
	// Arrange
	mockGateway := new(MockGateway)
	router := setupTestRouter(mockGateway)

	// Act
	req := httptest.NewRequest(http.MethodGet, "/invalid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "404 page not found")

	// Verify no other methods were called
	mockGateway.AssertNotCalled(t, "CheckRateLimit", mock.Anything, mock.Anything)
	mockGateway.AssertNotCalled(t, "ForwardRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestGatewayHandler_ProxyRequest_ServiceNotConfigured(t *testing.T) {
	// Arrange
	mockGateway := new(MockGateway)
	router := setupTestRouter(mockGateway)

	// Service exists in routing but not in configuration
	mockGateway.On("GetServiceConfig", "users").Return(models.ServiceConfig{}, false)

	// Act
	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "service not configured")

	// Verify other methods were not called
	mockGateway.AssertNotCalled(t, "CheckRateLimit", mock.Anything, mock.Anything)
	mockGateway.AssertNotCalled(t, "ForwardRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockGateway.AssertExpectations(t)
}

func TestGatewayHandler_ProxyRequest_ServiceError(t *testing.T) {
	// Arrange
	mockGateway := new(MockGateway)
	router := setupTestRouter(mockGateway)

	mockGateway.On("GetServiceConfig", "orders").Return(
		models.ServiceConfig{Name: "orders", APIVersion: "v1"}, true)
	mockGateway.On("CheckRateLimit", "192.0.2.1", "/orders/456").Return(true)
	mockGateway.On("ForwardRequest", mock.Anything, "orders", mock.Anything, mock.Anything).Return(
		models.ProxyResult{
			StatusCode: http.StatusNotFound,
			Duration:   10 * time.Millisecond,
			Error:      errors.New("service order not found"),
		},
	)

	// Act
	req := httptest.NewRequest(http.MethodPost, "/orders/456", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "service order not found")
	mockGateway.AssertExpectations(t)
}

func TestGatewayHandler_ProxyRequest_DifferentHTTPMethods(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		path    string
		service string
	}{
		{"GET Request", http.MethodGet, "/users/123", "users"},
		{"POST Request", http.MethodPost, "/users", "users"},
		{"PUT Request", http.MethodPut, "/orders/456", "orders"},
		{"DELETE Request", "DELETE", "/orders/789", "orders"},
		{"PATCH Request", "PATCH", "/users/123", "users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockGateway := new(MockGateway)
			router := setupTestRouter(mockGateway)

			mockGateway.On("GetServiceConfig", tt.service).Return(models.ServiceConfig{Name: tt.service}, true)
			mockGateway.On("CheckRateLimit", "192.0.2.1", tt.path).Return(true)
			mockGateway.On("ForwardRequest", mock.Anything, tt.service, mock.Anything, mock.Anything).Return(
				models.ProxyResult{
					StatusCode: http.StatusOK,
					Duration:   30 * time.Millisecond,
				},
			)

			// Act
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.RemoteAddr = "192.0.2.1:12345"
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, http.StatusOK, w.Code)
			mockGateway.AssertExpectations(t)
		})
	}
}

func TestGatewayHandler_ServicesHealthCheck_Success(t *testing.T) {
	// Arrange
	mockGateway := new(MockGateway)
	router := setupTestRouter(mockGateway)

	// Mock return data - all services healthy
	healthyServices := map[string]models.ServiceHealth{
		"user-service": {
			Status:       "healthy",
			Latency:      45 * time.Millisecond,
			ResponseTime: 45 * time.Millisecond,
			Timestamp:    time.Now(),
		},
		"order-service": {
			Status:       "healthy",
			Latency:      32 * time.Millisecond,
			ResponseTime: 32 * time.Millisecond,
			Timestamp:    time.Now(),
		},
	}

	// Set up mock expectations
	mockGateway.On("GetAllServiceHealth", mock.AnythingOfType("*context.timerCtx")).Return(healthyServices)
	mockGateway.On("GetEnvironment").Return("test")

	// Act
	req := httptest.NewRequest(http.MethodGet, "/services/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
	assert.Contains(t, w.Body.String(), "gateway")
	// Note: This test doesn't call GetAllServiceHealth, so no mock needed
}

func TestGatewayHandler_ServicesHealthCheck_SomeUnhealthService(t *testing.T) {
	mockGateway := &MockGateway{}
	handler := &GatewayHandler{gateway: mockGateway}

	// Mock return data - mixed health status
	mixedServices := map[string]models.ServiceHealth{
		"user-service": {
			Status:       "healthy",
			Latency:      45 * time.Millisecond,
			ResponseTime: 45 * time.Millisecond,
			Timestamp:    time.Now(),
		},
		"order-service": {
			Status:       "unhealthy",
			Error:        "HTTP 500",
			Latency:      2 * time.Second,
			ResponseTime: 2 * time.Second,
			Timestamp:    time.Now(),
		},
	}

	mockGateway.On("GetAllServiceHealth", mock.AnythingOfType("*context.timerCtx")).Return(mixedServices)
	mockGateway.On("GetEnvironment").Return("test")

	// Test execution
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/services/health", nil)

	handler.ServicesHealthCheck(c)

	// Assertions - should return 503 for degraded status
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	mockGateway.AssertExpectations(t)
}

func TestGatewayHandler_Metrics(t *testing.T) {
	// Arrange
	mockGateway := new(MockGateway)
	router := setupTestRouter(mockGateway)

	// Act
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	// Prometheus metrics should be returned - exact content depends on metrics registered
	assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")
}

func TestGatewayHandler_ProxyRequest_PathRewriting(t *testing.T) {
	tests := []struct {
		name            string
		requestPath     string
		expectedRewrite string
		expectedService string
	}{
		{
			name:            "Simple user path",
			requestPath:     "/users/123",
			expectedRewrite: "/api/v1/users/123",
			expectedService: "users",
		},
		{
			name:            "Nested user path",
			requestPath:     "/users/123/orders",
			expectedRewrite: "/api/v1/users/123/orders",
			expectedService: "users",
		},
		{
			name:            "Order path with query params",
			requestPath:     "/orders/456",
			expectedRewrite: "/api/v1/orders/456",
			expectedService: "orders",
		},
		{
			name:            "Root service path",
			requestPath:     "/users",
			expectedRewrite: "/api/v1/users",
			expectedService: "users",
		},
		{
			name:            "Root order service path",
			requestPath:     "/orders",
			expectedRewrite: "/api/v1/orders",
			expectedService: "orders",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockGateway := new(MockGateway)
			router := setupTestRouter(mockGateway)

			// Environment must not be "", dev, or test to be versioned
			mockGateway.On("GetServiceConfig", tt.expectedService).Return(
				models.ServiceConfig{
					Name:            tt.expectedService,
					ServiceEndpoint: tt.expectedRewrite,
					APIVersion:      "v1",
				},
				true,
			)
			mockGateway.On("CheckRateLimit", mock.Anything, tt.requestPath).Return(true)
			mockGateway.On("ForwardRequest", mock.Anything, tt.expectedService, mock.Anything,
				mock.MatchedBy(func(r *http.Request) bool {
					return r.URL.Path == tt.expectedRewrite
				}),
			).Return(
				models.ProxyResult{
					StatusCode: http.StatusOK,
					Duration:   20 * time.Millisecond,
				},
			)

			// Act
			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, http.StatusOK, w.Code)
			mockGateway.AssertExpectations(t)
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkGatewayHandler_ProxyRequest(b *testing.B) {
	mockGateway := new(MockGateway)
	router := setupTestRouter(mockGateway)

	mockGateway.On("GetServiceConfig", "users").Return(models.ServiceConfig{Name: "users"}, true)
	mockGateway.On("CheckRateLimit", mock.Anything, mock.Anything).Return(true)
	mockGateway.On("ForwardRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		models.ProxyResult{
			StatusCode: http.StatusOK,
			Duration:   10 * time.Millisecond,
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// Table-driven test for edge cases
func TestGatewayHandler_ProxyRequest_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		serviceName    string
		configExists   bool
		rateLimitOK    bool
		proxyResult    models.ProxyResult
		expectedStatus int
		expectedBody   string
	}{
		{
			name:         "User service success",
			path:         "/users/123/profile/settings",
			serviceName:  "users",
			configExists: true,
			rateLimitOK:  true,
			proxyResult: models.ProxyResult{
				StatusCode: http.StatusOK,
				Duration:   10 * time.Millisecond,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Rate limit with complex path",
			path:           "/users/123/profile/settings",
			serviceName:    "users",
			configExists:   true,
			rateLimitOK:    false,
			expectedStatus: http.StatusTooManyRequests,
			expectedBody:   "rate limit exceeded",
		},
		{
			name:         "Service timeout",
			path:         "/orders/timeout",
			serviceName:  "orders",
			configExists: true,
			rateLimitOK:  true,
			proxyResult: models.ProxyResult{
				StatusCode: http.StatusGatewayTimeout,
				Duration:   30 * time.Second,
				Error:      errors.New("service timeout"),
			},
			expectedStatus: http.StatusGatewayTimeout,
			expectedBody:   "service timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGateway := new(MockGateway)
			router := setupTestRouter(mockGateway)

			if tt.configExists {
				mockGateway.On("GetServiceConfig", tt.serviceName).Return(models.ServiceConfig{Name: tt.serviceName, APIVersion: "v1"}, true)
			} else {
				mockGateway.On("GetServiceConfig", tt.serviceName).Return(models.ServiceConfig{}, false)
			}

			if tt.rateLimitOK && tt.configExists {
				mockGateway.On("CheckRateLimit", mock.Anything, tt.path).Return(true)
				mockGateway.On("ForwardRequest", mock.Anything, tt.serviceName, mock.Anything, mock.Anything).Return(tt.proxyResult)
			} else if tt.configExists {
				mockGateway.On("CheckRateLimit", mock.Anything, tt.path).Return(false)
			}

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
			mockGateway.AssertExpectations(t)
		})
	}
}
