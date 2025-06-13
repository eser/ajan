package tracesfx

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.32.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	ErrFailedToCreateResource   = errors.New("failed to create resource")
	ErrFailedToShutdownProvider = errors.New("failed to shutdown trace provider")
	ErrTracesNotConfigured      = errors.New("traces not configured")
	ErrConnectionNotFound       = errors.New("connection not found")
	ErrConnectionNotOTLP        = errors.New("connection is not an OTLP connection")
	ErrOTLPBridgeNotAvailable   = errors.New("no OTLP bridge available")
	ErrTraceExporterNotFound    = errors.New("failed to get trace exporter")
)

// TracesProvider manages OpenTelemetry tracing infrastructure.
type TracesProvider struct {
	config         *Config
	bridge         *OTLPBridge
	tracerProvider *trace.TracerProvider
	shutdown       func(context.Context) error
}

// NewTracesProvider creates a new traces provider with the given configuration.
func NewTracesProvider(config *Config, registry ConnectionRegistry) *TracesProvider {
	var bridge *OTLPBridge
	if registry != nil {
		bridge = NewOTLPBridge(registry)
	}

	return &TracesProvider{
		config:         config,
		bridge:         bridge,
		tracerProvider: nil,
		shutdown:       nil,
	}
}

// Init initializes the traces provider.
func (tp *TracesProvider) Init() error {
	// If no connection is configured, use a no-op tracer
	if tp.config.OTLPConnectionName == "" {
		tp.tracerProvider = trace.NewTracerProvider()
		tp.shutdown = func(ctx context.Context) error { return nil }

		// Set global tracer provider to no-op
		otel.SetTracerProvider(noop.NewTracerProvider())

		return nil
	}

	// Create resource with service information
	res, err := createTraceResource(tp.config)
	if err != nil {
		return err
	}

	// Try to create OTLP trace exporter from connection
	exporter, err := tp.createOTLPTraceExporter()
	if err != nil {
		// If OTLP connection is not available, fall back to no-op tracer
		tp.tracerProvider = trace.NewTracerProvider()
		tp.shutdown = func(ctx context.Context) error { return nil }

		// Set global tracer provider to no-op
		otel.SetTracerProvider(noop.NewTracerProvider())

		return nil //nolint:nilerr // Intentional graceful fallback to no-op tracer
	}

	// Create batch span processor only if we have a valid exporter
	var processor trace.SpanProcessor
	if exporter != nil {
		processor = trace.NewBatchSpanProcessor(
			exporter,
			trace.WithBatchTimeout(tp.config.BatchTimeout),
			trace.WithMaxExportBatchSize(tp.config.BatchSize),
		)
	} else {
		// Use a simple processor with nil exporter for no-op behavior
		processor = trace.NewSimpleSpanProcessor(nil)
	}

	// Create tracer provider
	tp.tracerProvider = trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithSpanProcessor(processor),
		trace.WithSampler(trace.TraceIDRatioBased(tp.config.SampleRatio)),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp.tracerProvider)

	// Set global text map propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tp.shutdown = func(ctx context.Context) error {
		if err := tp.tracerProvider.Shutdown(ctx); err != nil {
			return fmt.Errorf("%w: %w", ErrFailedToShutdownProvider, err)
		}

		return nil
	}

	return nil
}

// Shutdown gracefully shuts down the traces provider.
func (tp *TracesProvider) Shutdown(ctx context.Context) error {
	if tp.shutdown != nil {
		return tp.shutdown(ctx)
	}

	return nil
}

// Tracer returns a tracer with the given name.
func (tp *TracesProvider) Tracer(name string) *Tracer {
	var otelTracer oteltrace.Tracer
	if tp.tracerProvider != nil {
		otelTracer = tp.tracerProvider.Tracer(name)
	} else {
		otelTracer = noop.NewTracerProvider().Tracer(name)
	}

	return &Tracer{
		tracer: otelTracer,
	}
}

func createTraceResource(config *Config) (*resource.Resource, error) {
	attributes := []attribute.KeyValue{}

	if config.ServiceName != "" {
		attributes = append(attributes, semconv.ServiceName(config.ServiceName))
	}

	if config.ServiceVersion != "" {
		attributes = append(attributes, semconv.ServiceVersion(config.ServiceVersion))
	}

	// Create resource without explicit schema URL to avoid conflicts
	customResource := resource.NewWithAttributes("", attributes...)

	res, err := resource.Merge(resource.Default(), customResource)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToCreateResource, err)
	}

	return res, nil
}

func (tp *TracesProvider) createOTLPTraceExporter() (trace.SpanExporter, error) {
	// Get OTLP connection from bridge
	if tp.bridge == nil {
		return nil, ErrOTLPBridgeNotAvailable
	}

	exporter, err := tp.bridge.GetTraceExporter(tp.config.OTLPConnectionName)
	if err != nil {
		return nil, fmt.Errorf(
			"%w (connection=%q): %w",
			ErrTraceExporterNotFound,
			tp.config.OTLPConnectionName,
			err,
		)
	}

	return exporter, nil
}
