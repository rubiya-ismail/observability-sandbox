package telemetry

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/propagation"
)

// -------------------------
// Request ID Middleware
// -------------------------

const RequestIDKey = "requestID"

// RequestIDMiddleware generates a UUID per request and adds it to context and headers
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := uuid.New().String()
		ctx := context.WithValue(c.Request.Context(), RequestIDKey, id)
		c.Request = c.Request.WithContext(ctx)
		c.Writer.Header().Set("X-Request-ID", id)

		c.Next()
	}
}

// -------------------------
// Tracing Middleware (Telemetry method)
// -------------------------

func (t *Telemetry) TracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		service, ok := c.Get("serviceName")
		if !ok {
			service = "unknown"
		}
		serviceName := service.(string)

		if t.Propagator == nil || t.Tracer == nil {
			c.Next()
			return
		}

		ctx := t.Propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
		ctx, span := t.Tracer.Start(ctx, serviceName+"/"+c.FullPath())
		defer span.End()

		c.Request = c.Request.WithContext(ctx)

		// Process request
		c.Next()
	}
}

// -------------------------
// Metrics Middleware (Telemetry method)
// -------------------------

func (t *Telemetry) MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// service, ok := c.Get("serviceName")
		// if !ok {
		// 	service = "unknown"
		// }
		// serviceName := service.(string)

		// Track active request
		t.Metrics.ActiveRequests.WithLabelValues().Inc()
		defer t.Metrics.ActiveRequests.WithLabelValues().Dec()

		// Process request
		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		duration := time.Since(start).Seconds()

		// Update metrics
		t.Metrics.RequestTotal.WithLabelValues(c.Request.Method, c.FullPath(), status).Inc()
		t.Metrics.RequestDuration.WithLabelValues(c.Request.Method, c.FullPath(), status).Observe(duration)
	}
}

// -------------------------
// Security Headers Middleware
// -------------------------

func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
		c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'")

		// Process request
		c.Next()
	}
}
