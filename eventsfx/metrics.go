package eventsfx

import (
	"errors"
	"fmt"

	"github.com/eser/ajan/metricsfx"
)

var ErrFailedToBuildEventDispatchesCounter = errors.New(
	"failed to build event dispatches counter",
)

// Metrics holds event-specific metrics using the simplified MetricsBuilder approach.
type Metrics struct {
	EventDispatchesTotal *metricsfx.CounterMetric
}

// NewMetrics creates event metrics using the simplified MetricsBuilder.
func NewMetrics(provider *metricsfx.MetricsProvider) (*Metrics, error) {
	builder := metricsfx.NewMetricsBuilder(provider, "github.com/eser/ajan/eventsfx", "1.0.0")

	eventDispatchesTotal, err := builder.Counter(
		"event_dispatches_total",
		"Total number of event dispatches",
	).WithUnit("{dispatch}").Build()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToBuildEventDispatchesCounter, err)
	}

	return &Metrics{
		EventDispatchesTotal: eventDispatchesTotal,
	}, nil
}
