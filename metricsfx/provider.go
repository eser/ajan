package metricsfx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.32.0"
)

const (
	minimumReadMemStatsInterval = 15 * time.Second
)

var (
	ErrFailedToCreateOTLPExporter       = errors.New("failed to create OTLP exporter")
	ErrFailedToCreatePrometheusExporter = errors.New("failed to create Prometheus exporter")
	ErrFailedToCreateResource           = errors.New("failed to create resource")
	ErrFailedToShutdownProvider         = errors.New("failed to shutdown metrics provider")
)

type MetricsProvider struct {
	config *Config

	meterProvider *sdkmetric.MeterProvider
	shutdown      func(context.Context) error
}

// NewMetricsProvider creates a new metrics provider with the given configuration.
func NewMetricsProvider(config *Config) *MetricsProvider {
	return &MetricsProvider{
		config: config,

		meterProvider: nil,
		shutdown:      nil,
	}
}

func (mp *MetricsProvider) Init() error {
	// Create resource with service information
	res, err := createResource(mp.config)
	if err != nil {
		return err
	}

	// Create readers based on configuration
	readers, shutdownFuncs, err := createReaders(mp.config)
	if err != nil {
		return err
	}

	// Create meter provider with readers
	providerOptions := []sdkmetric.Option{
		sdkmetric.WithResource(res),
	}

	for _, reader := range readers {
		providerOptions = append(providerOptions, sdkmetric.WithReader(reader))
	}

	mp.meterProvider = sdkmetric.NewMeterProvider(providerOptions...)

	mp.shutdown = func(ctx context.Context) error {
		// Shutdown meter provider first
		if err := mp.meterProvider.Shutdown(ctx); err != nil {
			return fmt.Errorf("%w: %w", ErrFailedToShutdownProvider, err)
		}

		// Then shutdown exporters
		for _, shutdownFunc := range shutdownFuncs {
			if err := shutdownFunc(ctx); err != nil {
				return err
			}
		}

		return nil
	}

	if mp.config.RegisterNativeCollectors {
		return mp.registerNativeCollectors()
	}

	return nil
}

// Shutdown gracefully shuts down the metrics provider.
func (mp *MetricsProvider) Shutdown(ctx context.Context) error {
	return mp.shutdown(ctx)
}

func (mp *MetricsProvider) NewBuilder() *MetricsBuilder {
	return NewMetricsBuilder(mp)
}

func (mp *MetricsProvider) registerNativeCollectors() error {
	// Set the global meter provider to enable runtime metrics collection
	otel.SetMeterProvider(mp.meterProvider)

	// Start runtime metrics collection with the same interval as Prometheus default
	err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(minimumReadMemStatsInterval))
	if err != nil {
		return err //nolint:wrapcheck
	}

	return nil
}

func createResource(config *Config) (*resource.Resource, error) {
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

func createReaders(config *Config) ([]sdkmetric.Reader, []func(context.Context) error, error) {
	var readers []sdkmetric.Reader

	var shutdownFuncs []func(context.Context) error

	// OTLP Exporter (preferred for OpenTelemetry Collector)
	if config.OTLPEndpoint != "" {
		otlpReader, shutdownFunc, err := createOTLPReader(config)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"%w (endpoint=%q): %w",
				ErrFailedToCreateOTLPExporter,
				config.OTLPEndpoint,
				err,
			)
		}

		readers = append(readers, otlpReader)
		shutdownFuncs = append(shutdownFuncs, shutdownFunc)
	}

	// Prometheus Exporter (legacy support)
	if config.PrometheusEndpoint != "" {
		promReader, err := createPrometheusReader()
		if err != nil {
			return nil, nil, fmt.Errorf("%w: %w", ErrFailedToCreatePrometheusExporter, err)
		}

		readers = append(readers, promReader)
	}

	// Add runtime metrics reader
	runtimeReader := sdkmetric.NewManualReader(
		sdkmetric.WithProducer(runtime.NewProducer()),
	)
	readers = append(readers, runtimeReader)

	// If no exporters configured, use manual reader for backward compatibility
	if len(readers) == 1 { // Only runtime reader
		return readers, shutdownFuncs, nil
	}

	return readers, shutdownFuncs, nil
}

func createOTLPReader(config *Config) (sdkmetric.Reader, func(context.Context) error, error) {
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(config.OTLPEndpoint),
	}

	if config.OTLPInsecure {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}

	exporter, err := otlpmetrichttp.New(context.Background(), opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ErrFailedToCreateOTLPExporter, err)
	}

	reader := sdkmetric.NewPeriodicReader(
		exporter,
		sdkmetric.WithInterval(config.ExportInterval),
	)

	shutdownFunc := func(ctx context.Context) error {
		return exporter.Shutdown(ctx)
	}

	return reader, shutdownFunc, nil
}

func createPrometheusReader() (sdkmetric.Reader, error) {
	exporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToCreatePrometheusExporter, err)
	}

	return exporter, nil
}
