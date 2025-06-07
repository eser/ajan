package grpcfx

import (
	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	mp MetricsProvider

	RequestsTotal   metric.Int64Counter
	RequestDuration metric.Float64Histogram
}

func NewMetrics(metricsProvider MetricsProvider) *Metrics {
	meter := metricsProvider.GetMeterProvider().Meter("github.com/eser/ajan/grpcfx")

	requestsTotal, err := meter.Int64Counter(
		"grpc_requests_total",
		metric.WithDescription("Total number of gRPC requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		panic(err) // Handle error appropriately in your application
	}

	requestDuration, err := meter.Float64Histogram(
		"grpc_request_duration_seconds",
		metric.WithDescription("gRPC request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err) // Handle error appropriately in your application
	}

	return &Metrics{
		mp:              metricsProvider,
		RequestsTotal:   requestsTotal,
		RequestDuration: requestDuration,
	}
}
