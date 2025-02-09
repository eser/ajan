package eventsfx

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	mp MetricsProvider

	RequestsTotal *prometheus.CounterVec
}

func NewMetrics(mp MetricsProvider) *Metrics { //nolint:varnamelen
	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{ //nolint:exhaustruct
			Name: "event_dispatches_total",
			Help: "Total number of event dispatches",
		},
		[]string{"event"},
	)

	mp.GetRegistry().MustRegister(requestsTotal)

	return &Metrics{
		mp:            mp,
		RequestsTotal: requestsTotal,
	}
}
