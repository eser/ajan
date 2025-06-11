package tracesfx

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.32.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	ErrFailedToCreateOTLPExporter = errors.New("failed to create OTLP trace exporter")
	ErrFailedToCreateResource     = errors.New("failed to create resource")
	ErrFailedToShutdownProvider   = errors.New("failed to shutdown trace provider")
	ErrTracesNotConfigured        = errors.New("traces not configured")
)

// TracesProvider manages OpenTelemetry tracing infrastructure.
type TracesProvider struct {
	config         *Config
	tracerProvider *trace.TracerProvider
	shutdown       func(context.Context) error
}

// NewTracesProvider creates a new traces provider with the given configuration.
func NewTracesProvider(config *Config) *TracesProvider {
	return &TracesProvider{
		config:         config,
		tracerProvider: nil,
		shutdown:       nil,
	}
}

// Init initializes the traces provider.
func (tp *TracesProvider) Init() error {
	// If no OTLP endpoint is configured, use a no-op tracer
	if tp.config.OTLPEndpoint == "" {
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

	// Create OTLP trace exporter
	exporter, err := createOTLPTraceExporter(tp.config)
	if err != nil {
		return err
	}

	// Create batch span processor
	processor := trace.NewBatchSpanProcessor(
		exporter,
		trace.WithBatchTimeout(tp.config.BatchTimeout),
		trace.WithMaxExportBatchSize(tp.config.BatchSize),
	)

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

func createOTLPTraceExporter(config *Config) (*otlptrace.Exporter, error) {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(config.OTLPEndpoint),
	}

	if config.OTLPInsecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToCreateOTLPExporter, err)
	}

	return exporter, nil
}
