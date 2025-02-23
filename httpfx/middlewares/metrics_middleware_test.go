package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/eser/ajan/httpfx"
	"github.com/eser/ajan/httpfx/middlewares"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockMetricsProvider struct {
	registry *prometheus.Registry
}

func (m *mockMetricsProvider) GetRegistry() *prometheus.Registry {
	return m.registry
}

func TestMetricsMiddleware(t *testing.T) {
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
				return c.Results.Error(http.StatusBadRequest, []byte("bad request"))
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create metrics
			registry := prometheus.NewRegistry()
			metricsProvider := &mockMetricsProvider{registry: registry}
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

			// Verify metrics were recorded
			metric := &dto.Metric{} //nolint:exhaustruct
			err := metrics.RequestsTotal.WithLabelValues(tt.method, tt.path, strconv.Itoa(tt.expectedStatus)).Write(metric)
			require.NoError(t, err)

			// The counter should have been incremented once
			assert.InDelta(t, float64(1), metric.GetCounter().GetValue(), 0.0001)
		})
	}
}
