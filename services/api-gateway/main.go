package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"services/api-gateway/internal/handlers"
	"services/api-gateway/internal/models"
	"services/api-gateway/internal/routes"
	"shared/telemetry"
)

func main() {
	// Determine environment from ENV variable (default to dev)
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "dev"
	}

	// Load environment-specific config
	configPath := fmt.Sprintf("config/%s.json", env)
	cfg, err := models.LoadConfig(configPath)
	if err != nil {
		logrus.WithError(err).Fatalf("Failed to load config from %s", configPath)
	}

	// Initialize telemetry
	tel, err := initTelemetry(env)
	if err != nil {
		logrus.WithError(err).Warn("Telemetry initialization failed, continuing without telemetry")

		// Additional detailed reporting
		logrus.WithFields(logrus.Fields{
			"service":            "api-gateway",
			"env":                env,
			"telemetry_disabled": true,
		}).Warn("Service running in degraded mode - no metrics/tracing available")
	} else {
		logrus.Info("Telemetry initialized successfully")
	}

	// Initialize gateway with loaded config
	gateway, err := models.NewGateway(cfg, tel)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize gateway")
	}

	// Initialize handlers
	gatewayHandler := handlers.NewGatewayHandler(gateway)

	// Setup routes
	router := routes.SetupRoutes(tel, gatewayHandler)

	// Start HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Gateway.Port),
		Handler:      router,
		ReadTimeout:  cfg.Gateway.ReadTimeout.ToDuration(),
		WriteTimeout: cfg.Gateway.WriteTimeout.ToDuration(),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatal("Server failed to start")
		}
	}()

	logrus.Infof("API Gateway started on :%d", cfg.Gateway.Port)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down server...")

	// Create shutdown context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server
	if err := srv.Shutdown(ctx); err != nil {
		logrus.WithError(err).Error("Server forced to shutdown")
	}

	// Shutdown telemetry
	if tel != nil {
		if err := tel.Shutdown(ctx); err != nil {
			logrus.WithError(err).Error("Telemetry shutdown failed")
		}
	}

	logrus.Info("Server shutdown complete")
}

func initTelemetry(env string) (*telemetry.Telemetry, error) {
	// Configure based on environment
	logLevel := logrus.InfoLevel
	if env == "dev" {
		logLevel = logrus.DebugLevel
	}

	config := telemetry.Config{
		ServiceName:    "api-gateway",
		ServiceVersion: "1.0.0",
		JaegerURL:      "http://jaeger:14268/api/traces",
		LogPath:        fmt.Sprintf("./logs/api-gateway-%s.log", env),
		LogLevel:       logLevel,
		MaxLogSize:     100, // 100MB
		MaxBackups:     10,
		MaxAgeDays:     30,
		CompressLog:    true,
	}

	return telemetry.New(config)
}
