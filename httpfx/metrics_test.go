package httpfx_test

import (
	"testing"

	"github.com/eser/ajan/httpfx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
)

func TestNewMetrics(t *testing.T) {
	t.Parallel()

	provider := newMockMetricsProvider()
	metrics := httpfx.NewMetrics(provider)
	require.NotNil(t, metrics)

	// Test that we can use the counter (basic smoke test)
	ctx := t.Context()
	attrs := metric.WithAttributes(
		attribute.String("method", "GET"),
		attribute.String("endpoint", "/test"),
		attribute.String("status", "200"),
	)

	// This should not panic
	metrics.RequestsTotal.Add(ctx, 1, attrs)
}

func TestMetrics_RequestsTotal(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []struct {
		name     string
		method   string
		endpoint string
		status   string
		count    int64
	}{
		{
			name:     "successful_get",
			method:   "GET",
			endpoint: "/test",
			status:   "200",
			count:    1,
		},
		{
			name:     "not_found",
			method:   "GET",
			endpoint: "/missing",
			status:   "404",
			count:    2,
		},
		{
			name:     "bad_request",
			method:   "POST",
			endpoint: "/api",
			status:   "400",
			count:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a new metrics provider with a manual reader for each test case
			reader := sdkmetric.NewManualReader()
			meterProvider := sdkmetric.NewMeterProvider(
				sdkmetric.WithResource(resource.Default()),
				sdkmetric.WithReader(reader),
			)

			provider := &mockMetricsProvider{meterProvider: meterProvider}
			metrics := httpfx.NewMetrics(provider)
			require.NotNil(t, metrics)

			// Add to the counter
			ctx := t.Context()
			attrs := metric.WithAttributes(
				attribute.String("method", tt.method),
				attribute.String("endpoint", tt.endpoint),
				attribute.String("status", tt.status),
			)

			metrics.RequestsTotal.Add(ctx, tt.count, attrs)

			// Collect metrics to verify
			var rm metricdata.ResourceMetrics
			err := reader.Collect(ctx, &rm)
			require.NoError(t, err)

			// Verify we have scope metrics
			require.Len(t, rm.ScopeMetrics, 1)

			// Verify we have the counter metric
			require.Len(t, rm.ScopeMetrics[0].Metrics, 1)
			metric := rm.ScopeMetrics[0].Metrics[0]

			assert.Equal(t, "http_requests_total", metric.Name)
			assert.Equal(t, "Total number of HTTP requests", metric.Description)

			// Verify the data points
			sumData, ok := metric.Data.(metricdata.Sum[int64])
			require.True(t, ok, "expected Sum[int64] metric data")
			assert.Len(t, sumData.DataPoints, 1)
			assert.Equal(t, tt.count, sumData.DataPoints[0].Value)
		})
	}
}
