package telemetry

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// Helpers
func forceLoggerWrite(logger *logrus.Logger) {
	logger.Info("log entry")
	if lj, ok := logger.Out.(*lumberjack.Logger); ok {
		_ = lj.Rotate()
	}
}

// clear registry (for old tests if needed)
func resetPrometheus() {
	// No longer necessary if we always use custom registry
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	prometheus.DefaultGatherer = prometheus.DefaultRegisterer.(prometheus.Gatherer)
}

// ------------------- Logger Tests -------------------
func TestSetupLogger_ValidPath(t *testing.T) {
	config := Config{
		ServiceName: "test-service",
		LogPath:     "/tmp/test-service.log",
	}

	logger, err := setupLogger(config)
	assert.NoError(t, err)
	assert.NotNil(t, logger)

	forceLoggerWrite(logger)

	stat, err := os.Stat(config.LogPath)
	assert.NoError(t, err)
	assert.False(t, stat.IsDir())

	_ = os.Remove(config.LogPath)
}

func TestSetupLogger_EmptyLogPath(t *testing.T) {
	config := Config{
		ServiceName: "empty-path-service",
		LogPath:     "",
	}

	logger, err := setupLogger(config)
	assert.NoError(t, err)
	assert.NotNil(t, logger)

	forceLoggerWrite(logger)

	defaultPath := filepath.Join("./logs", "empty-path-service.log")
	_, err = os.Stat(defaultPath)
	assert.NoError(t, err)

	_ = os.Remove(defaultPath)
	_ = os.RemoveAll("./logs")
}

func TestSetupLogger_NonWritableDir(t *testing.T) {
	config := Config{
		ServiceName: "readonly-service",
		LogPath:     "/root/test.log",
	}

	logger, err := setupLogger(config)
	assert.Error(t, err)
	assert.Nil(t, logger)
}

func TestSetupLogger_CreateNestedDir(t *testing.T) {
	config := Config{
		ServiceName: "nested-service",
		LogPath:     "./tmp/logs/nested-service.log",
	}

	logger, err := setupLogger(config)
	assert.NoError(t, err)
	assert.NotNil(t, logger)

	stat, err := os.Stat("./tmp/logs")
	assert.NoError(t, err)
	assert.True(t, stat.IsDir())

	_ = os.Remove(config.LogPath)
	_ = os.RemoveAll("./tmp")
}

func TestSetupLogger_InMemoryWriter(t *testing.T) {
	buf := &bytes.Buffer{}
	config := Config{
		ServiceName: "in-memory-service",
		LogLevel:    logrus.InfoLevel,
		LogWriter:   buf,
	}

	logger, err := setupLogger(config)
	assert.NoError(t, err)
	assert.NotNil(t, logger)

	// Write a log entry
	logger.Info("test log entry")

	assert.Contains(t, buf.String(), "test log entry")
}

func TestSetupLogger_CustomRotation(t *testing.T) {
	logPath := "/tmp/custom-rotation.log"
	config := Config{
		ServiceName: "custom-service",
		LogPath:     logPath,
		LogLevel:    logrus.InfoLevel,
		MaxLogSize:  1,
		MaxBackups:  2,
		MaxAgeDays:  3,
		CompressLog: true,
	}

	logger, err := setupLogger(config)
	assert.NoError(t, err)
	assert.NotNil(t, logger)

	// Ensure lumberjack is configured correctly
	lumberjackLogger, ok := logger.Out.(*lumberjack.Logger)
	assert.True(t, ok)
	assert.Equal(t, 1, lumberjackLogger.MaxSize)
	assert.Equal(t, 2, lumberjackLogger.MaxBackups)
	assert.Equal(t, 3, lumberjackLogger.MaxAge)
	assert.True(t, lumberjackLogger.Compress)

	// Cleanup
	_ = os.Remove(logPath)
}

func TestSetupLogger_Defaults(t *testing.T) {
	config := Config{
		ServiceName: "default-service",
		LogLevel:    logrus.InfoLevel,
	}

	logger, err := setupLogger(config)
	assert.NoError(t, err)
	assert.NotNil(t, logger)

	// Ensure lumberjack is defaulted
	lumberjackLogger, ok := logger.Out.(*lumberjack.Logger)
	assert.True(t, ok)
	assert.Equal(t, 50, lumberjackLogger.MaxSize)
	assert.Equal(t, 5, lumberjackLogger.MaxBackups)
	assert.Equal(t, 28, lumberjackLogger.MaxAge)
	assert.False(t, lumberjackLogger.Compress)

	// Cleanup
	defaultPath := filepath.Join("./logs", "default-service.log")
	_ = os.Remove(defaultPath)
	_ = os.RemoveAll("./logs")
}

// ------------------- Tracer Tests -------------------
func TestSetupTracer_ValidConfig(t *testing.T) {
	config := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0",
		JaegerURL:      "http://localhost:14268/api/traces",
	}

	mockExporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(mockExporter),
	)
	tracer := tp.Tracer(config.ServiceName)

	_, span := tracer.Start(context.Background(), "test-operation")
	span.End()

	defer func() {
		err := tp.Shutdown(context.Background())
		assert.NoError(t, err)
	}()

	spans := mockExporter.GetSpans()
	assert.Len(t, spans, 1)
	assert.Equal(t, "test-operation", spans[0].Name)
	assert.Equal(t, config.ServiceName, spans[0].InstrumentationLibrary.Name)
}

func TestSetupTracer_InvalidURL(t *testing.T) {
	config := Config{
		ServiceName:    "bad-tracer",
		ServiceVersion: "1.0",
		JaegerURL:      "http://invalid-url",
	}

	tracer, tp, err := setupTracer(config)
	assert.NoError(t, err)
	assert.NotNil(t, tracer)
	assert.NotNil(t, tp)

	_ = tp.Shutdown(context.Background())
}

// ------------------- Metrics Tests -------------------
func TestSetupMetrics_ValidConfig(t *testing.T) {
	config := Config{ServiceName: "test-service"}
	metrics, registry, err := setupMetrics(config)

	assert.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.NotNil(t, metrics.RequestTotal)
	assert.NotNil(t, metrics.RequestDuration)
	assert.NotNil(t, metrics.ActiveRequests)
	assert.NotNil(t, registry)

	// Pre-populate metrics so registry returns them
	metrics.RequestTotal.WithLabelValues(http.MethodGet, "/ping", "200").Add(0)
	metrics.RequestDuration.WithLabelValues(http.MethodGet, "/ping", "200").Observe(0)
	metrics.ActiveRequests.WithLabelValues().Set(0)

	// Ensure metrics are registered
	mfs, err := registry.Gather()
	assert.NoError(t, err)
	assert.NotEmpty(t, mfs)
}

func TestMetrics_UpdateValues(t *testing.T) {
	config := Config{ServiceName: "test-service"}
	metrics, registry, err := setupMetrics(config)
	assert.NoError(t, err)

	// --- Counter ---
	metrics.RequestTotal.WithLabelValues(http.MethodGet, "/ping", "200").Inc()

	m := &dto.Metric{}
	err = metrics.RequestTotal.WithLabelValues(http.MethodGet, "/ping", "200").Write(m)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), m.GetCounter().GetValue())

	// --- Histogram ---
	hist := metrics.RequestDuration.WithLabelValues(http.MethodGet, "/ping", "200")
	hist.Observe(0.5)

	collected := make(chan prometheus.Metric, 1)
	metrics.RequestDuration.Collect(collected)
	close(collected)

	m2 := &dto.Metric{}
	for c := range collected {
		err = c.Write(m2)
		assert.NoError(t, err)
		if m2.Histogram != nil {
			assert.Equal(t, uint64(1), m2.GetHistogram().GetSampleCount())
		}
	}

	// --- Gauge ---
	metrics.ActiveRequests.WithLabelValues().Inc()
	m3 := &dto.Metric{}
	err = metrics.ActiveRequests.WithLabelValues().Write(m3)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), m3.GetGauge().GetValue())

	// Gather from custom registry
	mfs, err := registry.Gather()
	assert.NoError(t, err)
	assert.Len(t, mfs, 3) // counter, histogram, gauge
}

// ------------------- Telemetry Tests -------------------
func TestNewTelemetry(t *testing.T) {
	config := Config{ServiceName: "test-service"}
	tel, err := New(config)

	assert.NoError(t, err)
	assert.NotNil(t, tel)
	assert.NotNil(t, tel.Logger)
	assert.NotNil(t, tel.Tracer)
	assert.NotNil(t, tel.TracerProvider)
	assert.NotNil(t, tel.Metrics)
	assert.NotNil(t, tel.Registry) // new
	assert.IsType(t, propagation.TraceContext{}, tel.Propagator)

	// verify metric increments
	m := &dto.Metric{}
	tel.Metrics.ActiveRequests.WithLabelValues().Inc()
	err = tel.Metrics.ActiveRequests.WithLabelValues().Write(m)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), m.GetGauge().GetValue())
}

func TestTelemetryShutdown(t *testing.T) {
	config := Config{ServiceName: "test-service"}
	tel, err := New(config)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = tel.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestTelemetryShutdownWithCanceledContext(t *testing.T) {
	config := Config{ServiceName: "test-service"}
	tel, err := New(config)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = tel.Shutdown(ctx)
	assert.Error(t, err)
}

func TestTelemetryShutdownLoggerRotation(t *testing.T) {
	tel := &Telemetry{
		Logger: logrus.New(),
	}
	tel.Logger.Out = &lumberjack.Logger{
		Filename:   "test.log",
		MaxSize:    1,
		MaxBackups: 1,
		MaxAge:     1,
	}

	ctx := context.Background()
	err := tel.Shutdown(ctx)
	assert.NoError(t, err)
}
