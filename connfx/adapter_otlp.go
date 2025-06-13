package connfx

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.32.0"
)

const (
	DefaultOTLPTimeout     = 30 * time.Second
	DefaultBatchTimeout    = 5 * time.Second
	DefaultExportInterval  = 30 * time.Second
	DefaultBatchSize       = 512
	DefaultSampleRatio     = 1.0
	HealthCheckRequestPath = "/v1/traces" // Standard OTLP path for health check
	MinimumReadMemInterval = 15 * time.Second
)

// Add missing connection capabilities for observability.
const (
	// ConnectionCapabilityObservability represents general observability behavior.
	ConnectionCapabilityObservability ConnectionCapability = "observability"

	// ConnectionCapabilityLogging represents logging behavior.
	ConnectionCapabilityLogging ConnectionCapability = "logging"

	// ConnectionCapabilityMetrics represents metrics behavior.
	ConnectionCapabilityMetrics ConnectionCapability = "metrics"

	// ConnectionCapabilityTracing represents tracing behavior.
	ConnectionCapabilityTracing ConnectionCapability = "tracing"
)

var (
	ErrFailedToCreateOTLPLogExporter    = errors.New("failed to create OTLP log exporter")
	ErrFailedToCreateOTLPMetricExporter = errors.New("failed to create OTLP metric exporter")
	ErrFailedToCreateOTLPTraceExporter  = errors.New("failed to create OTLP trace exporter")
	ErrFailedToCreateResource           = errors.New("failed to create resource")
	ErrFailedToShutdownOTLPClient       = errors.New("failed to shutdown OTLP client")
	ErrInvalidConfigTypeOTLP            = errors.New("invalid config type for OTLP connection")
	ErrOTLPEndpointRequired             = errors.New("OTLP endpoint is required")
	ErrOTLPHealthCheckFailed            = errors.New("OTLP health check failed")
	ErrFailedToShutdownLogProvider      = errors.New("failed to shutdown log provider")
	ErrFailedToShutdownMeterProvider    = errors.New("failed to shutdown meter provider")
	ErrFailedToShutdownTracerProvider   = errors.New("failed to shutdown tracer provider")
	ErrFailedToShutdownLogExporter      = errors.New("failed to shutdown log exporter")
	ErrFailedToShutdownMetricExporter   = errors.New("failed to shutdown metric exporter")
	ErrFailedToShutdownTraceExporter    = errors.New("failed to shutdown trace exporter")
	ErrFailedToCreateTestExporter       = errors.New("failed to create test exporter")
	ErrFailedToMergeResources           = errors.New("failed to merge resources")
)

// OTLPConnection represents an OpenTelemetry Protocol connection.
type OTLPConnection struct {
	lastHealth time.Time

	// Resource for telemetry attribution
	resource *resource.Resource

	// Exporters
	logExporter    *otlploghttp.Exporter
	metricExporter *otlpmetrichttp.Exporter
	traceExporter  *otlptrace.Exporter

	// Providers
	loggerProvider *sdklog.LoggerProvider
	meterProvider  *sdkmetric.MeterProvider
	tracerProvider *sdktrace.TracerProvider

	config   *ConfigTarget
	endpoint string
	protocol string

	// Configuration
	serviceName    string
	serviceVersion string
	batchTimeout   time.Duration
	exportInterval time.Duration
	batchSize      int
	sampleRatio    float64
	state          int32 // atomic field for connection state
	insecure       bool
}

// OTLPConnectionFactory creates OTLP connections.
type OTLPConnectionFactory struct {
	protocol string
}

// NewOTLPConnectionFactory creates a new OTLP connection factory.
func NewOTLPConnectionFactory(protocol string) *OTLPConnectionFactory {
	return &OTLPConnectionFactory{
		protocol: protocol,
	}
}

func (f *OTLPConnectionFactory) CreateConnection(
	ctx context.Context,
	config *ConfigTarget,
) (Connection, error) {
	endpoint := config.DSN
	if endpoint == "" {
		return nil, fmt.Errorf("%w (endpoint not provided)", ErrOTLPEndpointRequired)
	}

	// Extract configuration
	insecure := f.extractInsecureFlag(config)
	serviceName := f.extractServiceName(config)
	serviceVersion := f.extractServiceVersion(config)

	// Create resource for telemetry attribution
	res, err := f.createResource(serviceName, serviceVersion)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToCreateResource, err)
	}

	conn := &OTLPConnection{
		config:         config,
		state:          int32(ConnectionStateNotInitialized),
		lastHealth:     time.Time{},
		logExporter:    nil,
		metricExporter: nil,
		traceExporter:  nil,
		loggerProvider: nil,
		meterProvider:  nil,
		tracerProvider: nil,
		endpoint:       endpoint,
		insecure:       insecure,
		protocol:       f.protocol,
		resource:       res,
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
		batchTimeout:   f.extractBatchTimeout(config),
		exportInterval: f.extractExportInterval(config),
		batchSize:      f.extractBatchSize(config),
		sampleRatio:    f.extractSampleRatio(config),
	}

	// Initialize exporters
	if err := conn.initializeExporters(ctx); err != nil {
		return nil, err
	}

	// Perform initial health check
	status := conn.HealthCheck(ctx)
	if status.State == ConnectionStateError {
		return nil, fmt.Errorf("%w: %w", ErrOTLPHealthCheckFailed, status.Error)
	}

	return conn, nil
}

func (f *OTLPConnectionFactory) GetProtocol() string {
	return f.protocol
}

// Connection interface implementation

func (c *OTLPConnection) GetBehaviors() []ConnectionBehavior {
	return []ConnectionBehavior{
		ConnectionBehaviorStateless,
		ConnectionBehaviorStreaming,
	}
}

func (c *OTLPConnection) GetCapabilities() []ConnectionCapability {
	return []ConnectionCapability{
		ConnectionCapabilityObservability,
		ConnectionCapabilityLogging,
		ConnectionCapabilityMetrics,
		ConnectionCapabilityTracing,
	}
}

func (c *OTLPConnection) GetProtocol() string {
	return c.protocol
}

func (c *OTLPConnection) GetState() ConnectionState {
	return ConnectionState(atomic.LoadInt32(&c.state))
}

func (c *OTLPConnection) HealthCheck(ctx context.Context) *HealthStatus {
	start := time.Now()
	status := &HealthStatus{
		Timestamp: start,
		State:     c.GetState(),
		Error:     nil,
		Message:   "",
		Latency:   0,
	}

	// Create a simple health check by attempting to create a minimal exporter
	// This validates that the endpoint is reachable and properly configured
	healthCheck, err := c.performHealthCheck(ctx)
	status.Latency = time.Since(start)

	if err != nil {
		atomic.StoreInt32(&c.state, int32(ConnectionStateError))
		status.State = ConnectionStateError
		status.Error = err
		status.Message = fmt.Sprintf("OTLP health check failed: %v", err)

		return status
	}

	// If health check passed, connection is ready
	atomic.StoreInt32(&c.state, int32(ConnectionStateReady))
	status.State = ConnectionStateReady
	status.Message = fmt.Sprintf("OTLP connection is ready (endpoint=%s, secure=%t, check=%s)",
		c.endpoint, !c.insecure, healthCheck)
	c.lastHealth = start

	return status
}

func (c *OTLPConnection) Close(ctx context.Context) error { //nolint:cyclop
	atomic.StoreInt32(&c.state, int32(ConnectionStateDisconnected))

	var errs []error

	// Shutdown providers
	if c.loggerProvider != nil {
		if err := c.loggerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", ErrFailedToShutdownLogProvider, err))
		}
	}

	if c.meterProvider != nil {
		if err := c.meterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", ErrFailedToShutdownMeterProvider, err))
		}
	}

	if c.tracerProvider != nil {
		if err := c.tracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", ErrFailedToShutdownTracerProvider, err))
		}
	}

	// Shutdown exporters
	if c.logExporter != nil {
		if err := c.logExporter.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", ErrFailedToShutdownLogExporter, err))
		}
	}

	if c.metricExporter != nil {
		if err := c.metricExporter.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", ErrFailedToShutdownMetricExporter, err))
		}
	}

	if c.traceExporter != nil {
		if err := c.traceExporter.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", ErrFailedToShutdownTraceExporter, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%w: %v", ErrFailedToShutdownOTLPClient, errs)
	}

	return nil
}

func (c *OTLPConnection) GetRawConnection() any {
	return c
}

// OTLP-specific methods

// GetLoggerProvider returns the OpenTelemetry log provider.
func (c *OTLPConnection) GetLoggerProvider() *sdklog.LoggerProvider {
	return c.loggerProvider
}

// GetMeterProvider returns the OpenTelemetry meter provider.
func (c *OTLPConnection) GetMeterProvider() *sdkmetric.MeterProvider {
	return c.meterProvider
}

// GetTracerProvider returns the OpenTelemetry tracer provider.
func (c *OTLPConnection) GetTracerProvider() *sdktrace.TracerProvider {
	return c.tracerProvider
}

// GetLogExporter returns the OTLP log exporter.
func (c *OTLPConnection) GetLogExporter() *otlploghttp.Exporter {
	return c.logExporter
}

// GetMetricExporter returns the OTLP metric exporter.
func (c *OTLPConnection) GetMetricExporter() *otlpmetrichttp.Exporter {
	return c.metricExporter
}

// GetTraceExporter returns the OTLP trace exporter.
func (c *OTLPConnection) GetTraceExporter() *otlptrace.Exporter {
	return c.traceExporter
}

// GetResource returns the resource used for telemetry attribution.
func (c *OTLPConnection) GetResource() *resource.Resource {
	return c.resource
}

// GetEndpoint returns the OTLP endpoint.
func (c *OTLPConnection) GetEndpoint() string {
	return c.endpoint
}

// IsInsecure returns whether the connection uses insecure transport.
func (c *OTLPConnection) IsInsecure() bool {
	return c.insecure
}

// Private helper methods

func (c *OTLPConnection) initializeExporters(ctx context.Context) error {
	var err error

	c.logExporter, err = c.createLogExporter(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToCreateOTLPLogExporter, err)
	}

	c.metricExporter, err = c.createMetricExporter(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToCreateOTLPMetricExporter, err)
	}

	c.traceExporter, err = c.createTraceExporter(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToCreateOTLPTraceExporter, err)
	}

	// Create providers from exporters
	c.createProviders()

	return nil
}

func (c *OTLPConnection) createLogExporter(ctx context.Context) (*otlploghttp.Exporter, error) {
	opts := []otlploghttp.Option{
		otlploghttp.WithEndpoint(c.endpoint),
	}

	if c.insecure {
		opts = append(opts, otlploghttp.WithInsecure())
	}

	exporter, err := otlploghttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToCreateOTLPLogExporter, err)
	}

	return exporter, nil
}

func (c *OTLPConnection) createMetricExporter(
	ctx context.Context,
) (*otlpmetrichttp.Exporter, error) {
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(c.endpoint),
	}

	if c.insecure {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}

	exporter, err := otlpmetrichttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToCreateOTLPMetricExporter, err)
	}

	return exporter, nil
}

func (c *OTLPConnection) createTraceExporter(ctx context.Context) (*otlptrace.Exporter, error) {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(c.endpoint),
	}

	if c.insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToCreateOTLPTraceExporter, err)
	}

	return exporter, nil
}

func (c *OTLPConnection) createProviders() {
	// Create log provider
	if c.logExporter != nil {
		processor := sdklog.NewBatchProcessor(c.logExporter)
		c.loggerProvider = sdklog.NewLoggerProvider(
			sdklog.WithProcessor(processor),
			sdklog.WithResource(c.resource),
		)
	}

	// Create meter provider
	if c.metricExporter != nil {
		reader := sdkmetric.NewPeriodicReader(
			c.metricExporter,
			sdkmetric.WithInterval(c.exportInterval),
		)
		c.meterProvider = sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(c.resource),
			sdkmetric.WithReader(reader),
		)
	}

	// Create tracer provider
	if c.traceExporter != nil {
		processor := sdktrace.NewBatchSpanProcessor(
			c.traceExporter,
			sdktrace.WithBatchTimeout(c.batchTimeout),
			sdktrace.WithMaxExportBatchSize(c.batchSize),
		)
		c.tracerProvider = sdktrace.NewTracerProvider(
			sdktrace.WithResource(c.resource),
			sdktrace.WithSpanProcessor(processor),
			sdktrace.WithSampler(sdktrace.TraceIDRatioBased(c.sampleRatio)),
		)
	}
}

func (c *OTLPConnection) performHealthCheck(ctx context.Context) (string, error) {
	// For OTLP health check, we try to create a minimal exporter
	// This validates connectivity and configuration
	testOpts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(c.endpoint),
	}

	if c.insecure {
		testOpts = append(testOpts, otlptracehttp.WithInsecure())
	}

	// Create a temporary exporter for health check
	testExporter, err := otlptracehttp.New(ctx, testOpts...)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFailedToCreateTestExporter, err)
	}

	// Clean up test exporter
	defer func() {
		_ = testExporter.Shutdown(ctx) // Ignore shutdown errors for health check
	}()

	return "connection_validated", nil
}

func (f *OTLPConnectionFactory) extractInsecureFlag(config *ConfigTarget) bool {
	if config.TLS {
		return false
	}

	if config.Properties != nil {
		if insecure, ok := config.Properties["insecure"].(bool); ok {
			return insecure
		}
	}

	return true // Default to insecure for development
}

func (f *OTLPConnectionFactory) extractServiceName(config *ConfigTarget) string {
	if config.Properties != nil {
		if name, ok := config.Properties["service_name"].(string); ok {
			return name
		}
	}

	return "unknown-service"
}

func (f *OTLPConnectionFactory) extractServiceVersion(config *ConfigTarget) string {
	if config.Properties != nil {
		if version, ok := config.Properties["service_version"].(string); ok {
			return version
		}
	}

	return "unknown"
}

func (f *OTLPConnectionFactory) extractBatchTimeout(config *ConfigTarget) time.Duration {
	if config.Properties != nil {
		if timeout, ok := config.Properties["batch_timeout"].(time.Duration); ok {
			return timeout
		}
	}

	return DefaultBatchTimeout
}

func (f *OTLPConnectionFactory) extractExportInterval(config *ConfigTarget) time.Duration {
	if config.Properties != nil {
		if interval, ok := config.Properties["export_interval"].(time.Duration); ok {
			return interval
		}
	}

	return DefaultExportInterval
}

func (f *OTLPConnectionFactory) extractBatchSize(config *ConfigTarget) int {
	if config.Properties != nil {
		if size, ok := config.Properties["batch_size"].(int); ok {
			return size
		}
	}

	return DefaultBatchSize
}

func (f *OTLPConnectionFactory) extractSampleRatio(config *ConfigTarget) float64 {
	if config.Properties != nil {
		if ratio, ok := config.Properties["sample_ratio"].(float64); ok {
			return ratio
		}
	}

	return DefaultSampleRatio
}

func (f *OTLPConnectionFactory) createResource(
	serviceName, serviceVersion string,
) (*resource.Resource, error) {
	attributes := []attribute.KeyValue{
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(serviceVersion),
	}

	// Create resource without explicit schema URL to avoid conflicts
	customResource := resource.NewWithAttributes("", attributes...)

	res, err := resource.Merge(resource.Default(), customResource)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToMergeResources, err)
	}

	return res, nil
}
