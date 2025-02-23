package httpfx_test

import (
	"testing"

	"github.com/eser/ajan/httpfx"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetrics(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	provider := &mockMetricsProvider{registry: registry}

	metrics := httpfx.NewMetrics(provider)
	require.NotNil(t, metrics)

	// Increment the counter to ensure it's registered
	metrics.RequestsTotal.WithLabelValues("GET", "/test", "200").Inc()

	// Verify that metrics were registered
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	// Find our http_requests_total metric
	var foundMetric *dto.MetricFamily

	for _, mf := range metricFamilies {
		if mf.GetName() == "http_requests_total" {
			foundMetric = mf

			break
		}
	}

	require.NotNil(t, foundMetric, "http_requests_total metric not found")
	assert.Equal(t, dto.MetricType_COUNTER, foundMetric.GetType())

	// Verify the metric has the expected labels
	expectedLabels := []string{"method", "endpoint", "status"}
	metric := foundMetric.GetMetric()[0]

	for _, label := range metric.GetLabel() {
		assert.Contains(t, expectedLabels, label.GetName())
	}
}

func TestMetrics_RequestsTotal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		method   string
		endpoint string
		status   string
	}{
		{
			name:     "successful_get",
			method:   "GET",
			endpoint: "/test",
			status:   "200",
		},
		{
			name:     "not_found",
			method:   "GET",
			endpoint: "/missing",
			status:   "404",
		},
		{
			name:     "bad_request",
			method:   "POST",
			endpoint: "/api",
			status:   "400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a new registry and metrics for each test case
			registry := prometheus.NewRegistry()
			provider := &mockMetricsProvider{registry: registry}
			metrics := httpfx.NewMetrics(provider)
			require.NotNil(t, metrics)

			// Increment the counter
			metrics.RequestsTotal.WithLabelValues(tt.method, tt.endpoint, tt.status).Inc()

			// Get the metric value
			metric := &dto.Metric{} //nolint:exhaustruct
			err := metrics.RequestsTotal.WithLabelValues(tt.method, tt.endpoint, tt.status).Write(metric)
			require.NoError(t, err)

			// Verify the counter value
			assert.InDelta(t, float64(1), metric.GetCounter().GetValue(), 0.0001)
		})
	}
}
