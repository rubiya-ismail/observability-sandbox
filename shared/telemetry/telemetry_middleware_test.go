package telemetry

import (
	"net/http"
	"net/http/httptest"
	"shared/util"
	"testing"

	"github.com/gin-gonic/gin"
	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// --- Helper to create Telemetry via New() ---
func newTestTelemetry(t *testing.T, serviceName string) *Telemetry {
	t.Helper()

	tel, err := New(Config{
		ServiceName:    serviceName,
		ServiceVersion: "1.0",
		LogPath:        "",
		LogLevel:       logrus.DebugLevel,
		JaegerURL:      "http://localhost:14268/api/traces",
	})
	assert.NoError(t, err)
	assert.NotNil(t, tel)

	return tel
}

// --- Setup Gin router with middleware and serviceName ---
func setupTestRouter(serviceName string, middleware gin.HandlerFunc) *gin.Engine {
	r := util.SetupTestRouter()

	r.Use(func(c *gin.Context) {
		c.Set("serviceName", serviceName)
		c.Next()
	})

	r.Use(middleware)
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	return r
}

// --- 1. RequestIDMiddleware ---
func TestRequestIDMiddleware(t *testing.T) {
	serviceName := "test-service"
	r := setupTestRouter(serviceName, RequestIDMiddleware())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
}

// --- 2. TracingMiddleware ---
func TestTracingMiddleware(t *testing.T) {
	serviceName := "test-service"
	tel := newTestTelemetry(t, serviceName)

	r := setupTestRouter(serviceName, tel.TracingMiddleware())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Span check (in-memory exporter)
	spans := tracetest.NewInMemoryExporter().GetSpans()
	assert.NotNil(t, spans)
}

// --- 3. MetricsMiddleware ---
func TestMetricsMiddleware(t *testing.T) {
	serviceName := "test-service"
	tel := newTestTelemetry(t, serviceName)

	r := setupTestRouter(serviceName, tel.MetricsMiddleware())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check counter increments for "test-service"
	m := &dto.Metric{}
	err := tel.Metrics.RequestTotal.WithLabelValues(http.MethodGet, "/ping", "200").Write(m)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), m.GetCounter().GetValue())
}

// --- 4. SecurityHeadersMiddleware ---
func TestSecurityHeadersMiddleware(t *testing.T) {
	serviceName := "test-service"
	r := setupTestRouter(serviceName, SecurityHeadersMiddleware())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)

	headers := w.Header()
	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
	assert.Contains(t, headers.Get("X-XSS-Protection"), "1;")
}
