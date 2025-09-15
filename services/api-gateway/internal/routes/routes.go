package routes

import (
	"services/api-gateway/internal/handlers"
	"shared/telemetry"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func SetupRoutes(tel *telemetry.Telemetry, handler *handlers.GatewayHandler) *gin.Engine {
	router := gin.New()

	// Middleware stack in order
	// 1. Recover from panics early
	router.Use(gin.Recovery())

	// Conditional telemetry middleware with safety check
	if tel != nil {
		tel.WrapGinHandler(router)
		logrus.Info("Telemetry middleware enabled")
	} else {
		logrus.Warn("Telemetry middleware disabled - running without observability")
	}

	// 2. Assign request ID
	router.Use(telemetry.RequestIDMiddleware())

	// 3. Start tracing spans
	router.Use(tel.TracingMiddleware())

	// 4. Record Prometheus metrics
	router.Use(tel.MetricsMiddleware())

	// 5. Add security headers
	router.Use(telemetry.SecurityHeadersMiddleware())

	// 6. Gin default logger (can replace with logrus)
	router.Use(gin.Logger())

	// Gateway system routes - these have priority over proxy routes
	router.GET("/health", handler.HealthCheck)
	router.GET("/metrics", handler.Metrics)
	router.GET("/services/health", handler.ServicesHealthCheck)

	// Service proxy routes - must come after system routes
	// ANY /users/* -> Forward to user-service
	router.Any("/users", handler.ProxyRequest)
	router.Any("/users/*path", handler.ProxyRequest)

	// ANY /orders/* -> Forward to order-service
	router.Any("/orders", handler.ProxyRequest)
	router.Any("/orders/*path", handler.ProxyRequest)

	return router
}
