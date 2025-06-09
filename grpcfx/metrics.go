package grpcfx

import (
	"errors"
	"fmt"

	"github.com/eser/ajan/metricsfx"
)

var (
	ErrFailedToBuildGRPCRequestsCounter = errors.New(
		"failed to build gRPC requests counter",
	)
	ErrFailedToBuildGRPCRequestDurationHistogram = errors.New(
		"failed to build gRPC request duration histogram",
	)
)

// Metrics holds gRPC-specific metrics using the simplified MetricsBuilder approach.
type Metrics struct {
	builder *metricsfx.MetricsBuilder

	RequestsTotal   *metricsfx.CounterMetric
	RequestDuration *metricsfx.HistogramMetric
}

// NewMetrics creates gRPC metrics using the simplified MetricsBuilder.
func NewMetrics(provider *metricsfx.MetricsProvider) *Metrics {
	builder := metricsfx.NewMetricsBuilder(provider, "github.com/eser/ajan/grpcfx", "1.0.0")

	return &Metrics{
		builder: builder,

		RequestsTotal:   nil,
		RequestDuration: nil,
	}
}

func (metrics *Metrics) Init() error {
	requestsTotal, err := metrics.builder.Counter(
		"grpc_requests_total",
		"Total number of gRPC requests",
	).WithUnit("{request}").Build()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToBuildGRPCRequestsCounter, err)
	}

	metrics.RequestsTotal = requestsTotal

	requestDuration, err := metrics.builder.Histogram(
		"grpc_request_duration_seconds",
		"gRPC request duration in seconds",
	).WithDurationBuckets().Build()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToBuildGRPCRequestDurationHistogram, err)
	}

	metrics.RequestDuration = requestDuration

	return nil
}
