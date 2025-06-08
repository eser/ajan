package httpfx

import (
	"errors"
	"fmt"

	"github.com/eser/ajan/metricsfx"
)

var (
	ErrFailedToBuildHTTPRequestsCounter = errors.New(
		"failed to build HTTP requests counter",
	)
	ErrFailedToBuildHTTPRequestDurationHistogram = errors.New(
		"failed to build HTTP request duration histogram",
	)
)

// Metrics holds HTTP-specific metrics using the simplified MetricsBuilder approach.
type Metrics struct {
	RequestsTotal   *metricsfx.CounterMetric
	RequestDuration *metricsfx.HistogramMetric
}

// NewMetrics creates HTTP metrics using the simplified MetricsBuilder.
func NewMetrics(provider *metricsfx.MetricsProvider) (*Metrics, error) {
	builder := metricsfx.NewMetricsBuilder(provider, "github.com/eser/ajan/httpfx", "1.0.0")

	requestsTotal, err := builder.Counter(
		"http_requests_total",
		"Total number of HTTP requests",
	).WithUnit("{request}").Build()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToBuildHTTPRequestsCounter, err)
	}

	requestDuration, err := builder.Histogram(
		"http_request_duration_seconds",
		"HTTP request duration in seconds",
	).WithDurationBuckets().Build()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToBuildHTTPRequestDurationHistogram, err)
	}

	return &Metrics{
		RequestsTotal:   requestsTotal,
		RequestDuration: requestDuration,
	}, nil
}
