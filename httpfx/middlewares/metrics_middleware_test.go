package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/eser/ajan/httpfx"
	"github.com/eser/ajan/httpfx/middlewares"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
)

type mockMetricsProvider struct {
	meterProvider metric.MeterProvider
}

func (m *mockMetricsProvider) GetMeterProvider() metric.MeterProvider {
	return m.meterProvider
}

func TestMetricsMiddleware(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []struct {
		name           string
		method         string
		path           string
		handler        httpfx.Handler
		expectedStatus int
	}{
		{
			name:   "success_request",
			method: http.MethodGet,
			path:   "/test",
			handler: func(c *httpfx.Context) httpfx.Result {
				return c.Results.Ok()
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:   "error_request",
			method: http.MethodPost,
			path:   "/error",
			handler: func(c *httpfx.Context) httpfx.Result {
				return c.Results.Error(http.StatusBadRequest, httpfx.WithPlainText("bad request"))
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create metrics with manual reader for testing
			reader := sdkmetric.NewManualReader()
			meterProvider := sdkmetric.NewMeterProvider(
				sdkmetric.WithResource(resource.Default()),
				sdkmetric.WithReader(reader),
			)
			metricsProvider := &mockMetricsProvider{meterProvider: meterProvider}
			metrics := httpfx.NewMetrics(metricsProvider)

			// Create a router with the metrics middleware
			router := httpfx.NewRouter("/")
			router.Use(middlewares.MetricsMiddleware(metrics))

			// Add a test route
			router.Route(tt.method+" "+tt.path, tt.handler)

			// Create and execute test request
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			router.GetMux().ServeHTTP(w, req)

			// Verify the response status
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Collect metrics to verify they were recorded
			ctx := t.Context()

			var rm metricdata.ResourceMetrics
			err := reader.Collect(ctx, &rm)
			require.NoError(t, err)

			// Verify we have scope metrics
			require.Len(t, rm.ScopeMetrics, 1)

			// Verify we have the counter metric
			require.Len(t, rm.ScopeMetrics[0].Metrics, 1)
			metric := rm.ScopeMetrics[0].Metrics[0]

			assert.Equal(t, "http_requests_total", metric.Name)

			// Verify the data points - should have one measurement
			sumData, ok := metric.Data.(metricdata.Sum[int64])
			require.True(t, ok, "expected Sum[int64] metric data")
			require.Len(t, sumData.DataPoints, 1)
			assert.Equal(t, int64(1), sumData.DataPoints[0].Value)

			// Verify the attributes
			attrs := sumData.DataPoints[0].Attributes
			expectedAttrs := attribute.NewSet(
				attribute.String("method", tt.method),
				attribute.String("endpoint", tt.path),
				attribute.String("status", strconv.Itoa(tt.expectedStatus)),
			)
			assert.Equal(t, expectedAttrs, attrs)
		})
	}
}
