package metricsfx

import (
	"context"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.32.0"
)

const (
	minimumReadMemStatsInterval = 15 * time.Second
)

type MetricsProvider struct {
	meterProvider *sdkmetric.MeterProvider
	shutdown      func(context.Context) error
}

func NewMetricsProvider() *MetricsProvider {
	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			// TODO(@eser) fill with namespace information from deployment
			semconv.ServiceName(""),
			semconv.ServiceVersion(""),
		),
	)
	if err != nil {
		// Log error but continue with default resource
		res = resource.Default()
	}

	// Create a manual reader with runtime producer for collecting metrics
	reader := sdkmetric.NewManualReader(
		sdkmetric.WithProducer(runtime.NewProducer()),
	)

	// Create meter provider with the reader
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
		// Add additional readers as needed for your exporters
	)

	return &MetricsProvider{
		meterProvider: meterProvider,
		shutdown: func(ctx context.Context) error {
			return meterProvider.Shutdown(ctx)
		},
	}
}

func (mp *MetricsProvider) RegisterNativeCollectors() error {
	// Set the global meter provider to enable runtime metrics collection
	otel.SetMeterProvider(mp.meterProvider)

	// Start runtime metrics collection with the same interval as Prometheus default
	err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(minimumReadMemStatsInterval))
	if err != nil {
		return err //nolint:wrapcheck
	}

	return nil
}

func (mp *MetricsProvider) GetMeterProvider() metric.MeterProvider {
	return mp.meterProvider
}

// GetRegistry is kept for backward compatibility but now returns the MeterProvider
// For systems that need a registry-like interface, use GetMeterProvider instead.
func (mp *MetricsProvider) GetRegistry() metric.MeterProvider {
	return mp.meterProvider
}

// Shutdown gracefully shuts down the metrics provider.
func (mp *MetricsProvider) Shutdown(ctx context.Context) error {
	return mp.shutdown(ctx)
}
