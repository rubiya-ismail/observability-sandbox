package telemetry

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/natefinch/lumberjack"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// Config contains all configurable options for telemetry
type Config struct {
	ServiceName    string
	ServiceVersion string
	JaegerURL      string // e.g., http://localhost:14268/api/traces
	LogPath        string // Path to log file
	LogLevel       logrus.Level

	// Optional logger overrides
	LogWriter   io.Writer // If set, use this writer instead of lumberjack file
	MaxLogSize  int       // MB
	MaxBackups  int
	MaxAgeDays  int
	CompressLog bool
}

// Metrics holds Prometheus metric collectors
type Metrics struct {
	RequestTotal    *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	ActiveRequests  *prometheus.GaugeVec
}

// Telemetry encapsulates logger, metrics, and tracing
type Telemetry struct {
	Logger         *logrus.Logger
	Tracer         trace.Tracer
	TracerProvider *sdktrace.TracerProvider
	Metrics        *Metrics
	Propagator     propagation.TextMapPropagator
	Registry       *prometheus.Registry // NEW
}

// Shutdown gracefully shuts down telemetry (tracing and logging)
func (t *Telemetry) Shutdown(ctx context.Context) error {
	var errs []error

	if t.TracerProvider != nil {
		if err := t.TracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer shutdown: %w", err))
		}
	}

	if t.Logger != nil {
		if logFile, ok := t.Logger.Out.(*lumberjack.Logger); ok {
			logFile.Rotate()
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}
	return nil
}

// New creates a new Telemetry instance
// New creates a new Telemetry instance
func New(config Config) (*Telemetry, error) {
	// Setup logger
	logger, err := setupLogger(config)
	if err != nil {
		return nil, err
	}

	// Setup tracer
	tracer, tp, err := setupTracer(config)
	if err != nil {
		return nil, err
	}

	// Setup metrics
	metrics, registry, err := setupMetrics(config)
	if err != nil {
		return nil, err
	}

	// Optionally, you can store the registry in Telemetry if you need to expose /metrics
	return &Telemetry{
		Logger:         logger,
		Tracer:         tracer,
		TracerProvider: tp,
		Metrics:        metrics,
		Propagator:     propagation.TraceContext{},
		Registry:       registry,
	}, nil
}

// setupLogger initializes Logrus with JSON formatting and optional rotation
func setupLogger(config Config) (*logrus.Logger, error) {
	logPath := config.LogPath
	if logPath == "" && config.LogWriter == nil {
		logPath = filepath.Join("./logs", config.ServiceName+".log")
	}

	// Ensure directory exists only if writing to file
	if logPath != "" {
		if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create log dir: %w", err)
		}
	}

	logger := logrus.New()
	logger.SetLevel(config.LogLevel)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	// Use provided writer or lumberjack file
	if config.LogWriter != nil {
		logger.SetOutput(config.LogWriter)
	} else {
		// Use defaults if not set
		maxSize := 50
		if config.MaxLogSize > 0 {
			maxSize = config.MaxLogSize
		}
		maxBackups := 5
		if config.MaxBackups > 0 {
			maxBackups = config.MaxBackups
		}
		maxAge := 28
		if config.MaxAgeDays > 0 {
			maxAge = config.MaxAgeDays
		}

		logger.SetOutput(&lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     maxAge,
			Compress:   config.CompressLog,
		})
	}

	return logger, nil
}

// setupTracer initializes OpenTelemetry tracer with Jaeger exporter
func setupTracer(config Config) (trace.Tracer, *sdktrace.TracerProvider, error) {
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.JaegerURL)))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(1.0))), // configurable in future
	)

	// Optional: do not set global provider here if library is used in larger apps
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp.Tracer(config.ServiceName), tp, nil
}

// setupMetrics creates Prometheus metric collectors
// setupMetrics creates Prometheus metric collectors with best practices:
// - Uses a custom registry (not global)
// - Supports namespace (service-specific metrics)
// - Supports custom histogram buckets
// Returns both Metrics struct and the registry
func setupMetrics(config Config) (*Metrics, *prometheus.Registry, error) {
	reg := prometheus.NewRegistry() // custom registry

	namespace := config.ServiceName // use service name as namespace

	// Default histogram buckets (can be customized if needed)
	defaultBuckets := prometheus.DefBuckets

	requestTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "Histogram of response time for handler",
			Buckets:   defaultBuckets,
		},

		[]string{"method", "path", "status"},
	)

	activeRequests := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "http_active_requests",
			Help:      "Number of active requests",
		},
		nil,
	)

	// Register collectors in custom registry
	if err := reg.Register(requestTotal); err != nil {
		return nil, nil, fmt.Errorf("failed to register requestTotal: %w", err)
	}
	if err := reg.Register(requestDuration); err != nil {
		return nil, nil, fmt.Errorf("failed to register requestDuration: %w", err)
	}
	if err := reg.Register(activeRequests); err != nil {
		return nil, nil, fmt.Errorf("failed to register activeRequests: %w", err)
	}

	return &Metrics{
		RequestTotal:    requestTotal,
		RequestDuration: requestDuration,
		ActiveRequests:  activeRequests,
	}, reg, nil
}

func (t *Telemetry) WrapGinHandler(r *gin.Engine) {
	r.Use(func(c *gin.Context) {
		// Safety check - this should prevent the crash
		if t == nil || t.Propagator == nil || t.Tracer == nil {
			logrus.Warn("Telemetry components not available for request")
			c.Next()
			return
		}

		ctx := t.Propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
		ctx, span := t.Tracer.Start(ctx, c.FullPath())
		defer span.End()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
}
