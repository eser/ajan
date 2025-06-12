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
	Provider *metricsfx.MetricsProvider

	EventDispatchesTotal *metricsfx.CounterMetric
}

// NewMetrics creates event metrics using the simplified MetricsBuilder.
func NewMetrics(provider *metricsfx.MetricsProvider) *Metrics {
	return &Metrics{
		Provider: provider,

		EventDispatchesTotal: nil,
	}
}

func (metrics *Metrics) Init() error {
	builder := metrics.Provider.NewBuilder()

	eventDispatchesTotal, err := builder.Counter(
		"event_dispatches_total",
		"Total number of event dispatches",
	).WithUnit("{dispatch}").Build()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToBuildEventDispatchesCounter, err)
	}

	metrics.EventDispatchesTotal = eventDispatchesTotal

	return nil
}
