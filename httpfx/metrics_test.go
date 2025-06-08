package httpfx_test

import (
	"testing"
	"time"

	"github.com/eser/ajan/httpfx"
	"github.com/eser/ajan/metricsfx"
	"github.com/stretchr/testify/require"
)

func TestNewMetrics(t *testing.T) {
	t.Parallel()

	provider := setupTestMetricsProvider(t)
	metrics, err := httpfx.NewMetrics(provider)
	require.NoError(t, err)
	require.NotNil(t, metrics)

	// Test that we can use the metrics (basic smoke test)
	ctx := t.Context()
	attrs := metricsfx.HTTPAttrs("GET", "/test", "200")

	// This should not panic
	metrics.RequestsTotal.Inc(ctx, attrs...)
	metrics.RequestDuration.RecordDuration(ctx, 100*time.Millisecond, attrs...)
}

func TestMetrics_Integration(t *testing.T) {
	t.Parallel()

	provider := setupTestMetricsProvider(t)
	metrics, err := httpfx.NewMetrics(provider)
	require.NoError(t, err)
	require.NotNil(t, metrics)

	ctx := t.Context()

	// Test various HTTP scenarios
	testCases := []struct {
		method   string
		endpoint string
		status   string
		duration time.Duration
	}{
		{"GET", "/api/users", "200", 150 * time.Millisecond},
		{"POST", "/api/users", "201", 250 * time.Millisecond},
		{"GET", "/api/users/123", "404", 50 * time.Millisecond},
		{"PUT", "/api/users/123", "500", 300 * time.Millisecond},
	}

	for _, tc := range testCases {
		attrs := metricsfx.HTTPAttrs(tc.method, tc.endpoint, tc.status)

		metrics.RequestsTotal.Inc(ctx, attrs...)
		metrics.RequestDuration.RecordDuration(ctx, tc.duration, attrs...)
	}
}
