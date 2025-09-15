package handlers

import (
	"context"
	"net/http"
	"shared/util"
	"strings"
	"time"

	"services/api-gateway/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type GatewayHandler struct {
	gateway models.IGateway
}

func NewGatewayHandler(g models.IGateway) *GatewayHandler {
	return &GatewayHandler{
		gateway: g,
	}
}

// ProxyRequest handles forwarding requests to appropriate services based on path prefix
func (h *GatewayHandler) ProxyRequest(c *gin.Context) {
	path := strings.TrimPrefix(c.Request.URL.Path, "/")

	// Determine service name from path prefix
	var serviceName string
	if strings.HasPrefix(path, "users/") || path == "users" {
		serviceName = "users"
	} else if strings.HasPrefix(path, "orders/") || path == "orders" {
		serviceName = "orders"
	} else {
		util.SendErrorResponse(c, http.StatusNotFound, "service not found - supported paths: /users/* or /orders/*")
		return
	}

	// Fetch service config
	serviceConfig, exists := h.gateway.GetServiceConfig(serviceName)
	if !exists {
		util.SendErrorResponse(c, http.StatusNotFound, "service not configured")
		return
	}

	// Store service name for logging/metrics
	c.Set("serviceName", serviceName)

	// Rate limiting
	if !h.gateway.CheckRateLimit(c.ClientIP(), c.Request.URL.Path) {
		util.SendErrorResponse(c, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}

	// Forward the versioned request
	originalPath := c.Request.URL.Path
	c.Request.URL.Path = serviceConfig.ServiceEndpoint
	result := h.gateway.ForwardRequest(c.Request.Context(), serviceName, c.Writer, c.Request)
	c.Request.URL.Path = originalPath // Restore original path

	if result.Error != nil {
		util.SendErrorResponse(c, result.StatusCode, result.Error.Error())
		return
	}
}

func (h *GatewayHandler) ServicesHealthCheck(c *gin.Context) {
	// Create timeout context for service health checks
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Get all service health status
	serviceHealth := h.gateway.GetAllServiceHealth(ctx)

	// Determine overall gateway health based on service health
	overallStatus := "healthy"
	unhealthyServices := make([]string, 0)

	for serviceName, health := range serviceHealth {
		if health.Status != "healthy" {
			unhealthyServices = append(unhealthyServices, serviceName)
		}
	}

	// Gateway is degraded if any services are unhealthy
	if len(unhealthyServices) > 0 {
		overallStatus = "degraded"
	}

	health := gin.H{
		"status":      overallStatus,
		"service":     "gateway",
		"timestamp":   time.Now().Format(time.RFC3339),
		"environment": h.gateway.GetEnvironment(),
		"services":    serviceHealth,
	}

	// Add summary for quick overview
	healthyCount := len(serviceHealth) - len(unhealthyServices)
	health["summary"] = gin.H{
		"total_services":     len(serviceHealth),
		"healthy_services":   healthyCount,
		"unhealthy_services": len(unhealthyServices),
	}

	// Include unhealthy services list if any
	if len(unhealthyServices) > 0 {
		health["unhealthy_services"] = unhealthyServices
	}

	// Determine HTTP status code based on gateway health
	httpStatus := http.StatusOK
	if overallStatus == "degraded" {
		httpStatus = http.StatusServiceUnavailable
	}

	util.SendJSONResponse(c, httpStatus, health)
}

func (h *GatewayHandler) HealthCheck(ctxt *gin.Context) {
	util.SendJSONResponse(ctxt, http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "gateway",
	})
}

// Metrics exposes Prometheus metrics
func (h *GatewayHandler) Metrics(c *gin.Context) {
	promhttp.Handler().ServeHTTP(c.Writer, c.Request)
}
