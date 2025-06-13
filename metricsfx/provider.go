package metricsfx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.32.0"
)

const (
	minimumReadMemStatsInterval = 15 * time.Second
)

var (
	ErrFailedToCreateResource            = errors.New("failed to create resource")
	ErrFailedToShutdownProvider          = errors.New("failed to shutdown metrics provider")
	ErrOTLPBridgeNotAvailable            = errors.New("no OTLP bridge available")
	ErrMetricExporterNotAvailable        = errors.New("no metric exporter available")
	ErrFailedToCreateMeterProvider       = errors.New("failed to create meter provider")
	ErrFailedToInitializeMetricsProvider = errors.New("failed to initialize metrics provider")
)

type MetricsProvider struct {
	config *Config
	bridge *OTLPBridge

	meterProvider *sdkmetric.MeterProvider
	shutdown      func(context.Context) error
}

// NewMetricsProvider creates a new metrics provider with the given configuration.
func NewMetricsProvider(config *Config, registry ConnectionRegistry) *MetricsProvider {
	var bridge *OTLPBridge
	if registry != nil {
		bridge = NewOTLPBridge(registry)
	}

	return &MetricsProvider{
		config: config,
		bridge: bridge,

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
	readers, shutdownFuncs, err := mp.createReaders()
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

	// Set global meter provider to enable both runtime and custom metrics
	otel.SetMeterProvider(mp.meterProvider)

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

	if !mp.config.NoNativeCollectorRegistration {
		return mp.registerNativeCollectors()
	}

	return nil
}

// Shutdown gracefully shuts down the metrics provider.
func (mp *MetricsProvider) Shutdown(ctx context.Context) error {
	if mp.shutdown != nil {
		return mp.shutdown(ctx)
	}

	return nil
}

func (mp *MetricsProvider) NewBuilder() *MetricsBuilder {
	return NewMetricsBuilder(mp)
}

func (mp *MetricsProvider) registerNativeCollectors() error {
	// Start runtime metrics collection - they will use the same meter provider
	// and be exported through the same readers (OTLP)
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

func (mp *MetricsProvider) createReaders() ([]sdkmetric.Reader, []func(context.Context) error, error) {
	var readers []sdkmetric.Reader

	var shutdownFuncs []func(context.Context) error

	// OTLP Connection
	if mp.config.OTLPConnectionName != "" {
		otlpReader, shutdownFunc, err := mp.createOTLPReader()
		if err != nil {
			return nil, nil, fmt.Errorf(
				"failed to create OTLP reader (connection=%q): %w",
				mp.config.OTLPConnectionName,
				err,
			)
		}

		readers = append(readers, otlpReader)
		shutdownFuncs = append(shutdownFuncs, shutdownFunc)
	}

	// If no exporter configured, use manual reader for backward compatibility
	if len(readers) == 0 {
		// Create a manual reader that can be used for testing or when no export is needed
		manualReader := sdkmetric.NewManualReader()
		readers = append(readers, manualReader)
	}

	return readers, shutdownFuncs, nil
}

func (mp *MetricsProvider) createOTLPReader() (sdkmetric.Reader, func(context.Context) error, error) {
	// Get OTLP connection from bridge
	if mp.bridge == nil {
		return nil, nil, fmt.Errorf("%w", ErrOTLPBridgeNotAvailable)
	}

	exporter, err := mp.bridge.GetMetricExporter(mp.config.OTLPConnectionName)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"%w (connection=%q): %w",
			ErrMetricExporterNotAvailable,
			mp.config.OTLPConnectionName,
			err,
		)
	}

	if exporter == nil {
		return nil, nil, fmt.Errorf(
			"%w (connection=%q)",
			ErrMetricExporterNotAvailable,
			mp.config.OTLPConnectionName,
		)
	}

	reader := sdkmetric.NewPeriodicReader(
		exporter,
		sdkmetric.WithInterval(mp.config.ExportInterval),
	)

	shutdownFunc := func(ctx context.Context) error {
		// Don't shutdown the exporter here - it's owned by the connection
		// The connection registry will handle shutdown
		return nil
	}

	return reader, shutdownFunc, nil
}
