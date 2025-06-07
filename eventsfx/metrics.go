package eventsfx

import (
	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	mp MetricsProvider

	RequestsTotal metric.Int64Counter
}

func NewMetrics(mp MetricsProvider) *Metrics {
	meter := mp.GetMeterProvider().Meter("github.com/eser/ajan/eventsfx")

	requestsTotal, err := meter.Int64Counter(
		"event_dispatches_total",
		metric.WithDescription("Total number of event dispatches"),
		metric.WithUnit("{dispatch}"),
	)
	if err != nil {
		panic(err) // Handle error appropriately in your application
	}

	return &Metrics{
		mp:            mp,
		RequestsTotal: requestsTotal,
	}
}
