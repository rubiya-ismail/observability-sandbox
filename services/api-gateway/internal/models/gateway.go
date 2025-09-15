package models

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"shared/telemetry"
)

// IGateway defines gateway operations
type IGateway interface {
	ForwardRequest(ctx context.Context, serviceName string, w http.ResponseWriter, r *http.Request) ProxyResult
	CheckServiceHealth(ctx context.Context, serviceName string) ServiceHealth
	GetAllServiceHealth(ctx context.Context) map[string]ServiceHealth
	CheckRateLimit(clientIP, path string) bool
	GetServiceConfig(serviceName string) (ServiceConfig, bool)
	GetEnvironment() string
}

// Gateway struct with configuration
type Gateway struct {
	config  *GatewayConfig
	proxies map[string]*httputil.ReverseProxy
	mu      sync.RWMutex
	tel     *telemetry.Telemetry
}

type ProxyResult struct {
	StatusCode int           `json:"status_code"`
	Duration   time.Duration `json:"duration"`
	Error      error         `json:"error,omitempty"`
}

type ServiceHealth struct {
	Status       string        `json:"status"`
	Latency      time.Duration `json:"latency,omitempty"`
	ResponseTime time.Duration `json:"response_time,omitempty"`
	Error        string        `json:"error,omitempty"`
	Timestamp    time.Time     `json:"timestamp"`
}

// NewGateway creates a new gateway instance with provided configuration
func NewGateway(config *GatewayConfig, tel *telemetry.Telemetry) (*Gateway, error) {
	g := &Gateway{
		config:  config,
		proxies: make(map[string]*httputil.ReverseProxy),
		tel:     tel,
	}

	if err := g.initProxies(); err != nil {
		return nil, err
	}

	return g, nil
}

// GetServiceConfig returns configuration for a specific service
func (g *Gateway) GetServiceConfig(serviceName string) (ServiceConfig, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.config == nil {
		return ServiceConfig{}, false
	}

	config, exists := g.config.Services[serviceName]
	return config, exists
}

// ForwardRequest forwards HTTP requests to the appropriate service
func (g *Gateway) ForwardRequest(ctx context.Context, serviceName string, w http.ResponseWriter, r *http.Request) ProxyResult {
	start := time.Now()

	// Optional tracing
	if g.tel != nil {
		_, span := g.tel.Tracer.Start(ctx, fmt.Sprintf("gateway.proxy.%s", serviceName))
		defer span.End()
	}

	g.mu.RLock()
	proxy, exists := g.proxies[serviceName]
	g.mu.RUnlock()

	if !exists {
		return ProxyResult{
			StatusCode: http.StatusNotFound,
			Duration:   time.Since(start),
			Error:      fmt.Errorf("service %s not found", serviceName),
		}
	}

	// Create a custom ResponseWriter to capture status code
	responseCapture := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	proxy.ServeHTTP(responseCapture, r.WithContext(ctx))

	return ProxyResult{
		StatusCode: responseCapture.statusCode,
		Duration:   time.Since(start),
	}
}

// CheckServiceHealth checks health of a specific service
func (g *Gateway) CheckServiceHealth(ctx context.Context, serviceName string) ServiceHealth {
	g.mu.RLock()
	serviceConfig, exists := g.config.Services[serviceName]
	g.mu.RUnlock()

	if !exists {
		return ServiceHealth{
			Status:    "unknown",
			Error:     "service not configured",
			Timestamp: time.Now(),
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	healthURL := FormatServiceURL(serviceConfig, serviceConfig.HealthEndpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return ServiceHealth{
			Status:    "unhealthy",
			Error:     err.Error(),
			Timestamp: time.Now(),
		}
	}

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)

	if err != nil {
		return ServiceHealth{
			Status:       "unhealthy",
			Latency:      latency,
			ResponseTime: latency,
			Error:        err.Error(),
			Timestamp:    time.Now(),
		}
	}
	defer resp.Body.Close()

	status := "healthy"
	var errorMsg string
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		status = "unhealthy"
		errorMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return ServiceHealth{
		Status:       status,
		Latency:      latency,
		ResponseTime: latency,
		Error:        errorMsg,
		Timestamp:    time.Now(),
	}
}

// GetAllServiceHealth checks health of all configured services
func (g *Gateway) GetAllServiceHealth(ctx context.Context) map[string]ServiceHealth {
	g.mu.RLock()
	services := make(map[string]ServiceHealth)
	for service := range g.config.Services {
		services[service] = g.CheckServiceHealth(ctx, service)
	}
	g.mu.RUnlock()

	return services
}

// CheckRateLimit checks if request is within rate limits
func (g *Gateway) CheckRateLimit(clientIP, path string) bool {
	// Simple in-memory rate limiting
	// In production, use Redis or similar
	return true // Simplified for example
}

// initProxies initializes reverse proxies for all configured services
func (g *Gateway) initProxies() error {
	if g.config == nil {
		return fmt.Errorf("Gateway configuration cannot be nil")
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// Clear existing proxies
	g.proxies = make(map[string]*httputil.ReverseProxy)

	for service, config := range g.config.Services {
		target, err := url.Parse(config.BaseURL)
		if err != nil {
			return fmt.Errorf("invalid upstream URL for %s: %w", service, err)
		}

		proxy := httputil.NewSingleHostReverseProxy(target)

		// Enhanced error handling with service config
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			if g.tel != nil {
				g.tel.Logger.WithContext(r.Context()).WithError(err).WithField("service", service).Error("Proxy error")
			}
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		}

		// Set custom transport with timeout from config
		proxy.Transport = &http.Transport{
			MaxIdleConns:        100,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		}

		g.proxies[service] = proxy
	}

	return nil
}

func (g *Gateway) GetEnvironment() string {
	return g.config.Environment
}

// responseWriter is a wrapper to capture the response status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
