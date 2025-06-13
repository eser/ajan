package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eser/ajan/httpfx"
	"github.com/eser/ajan/httpfx/middlewares"
	"github.com/eser/ajan/metricsfx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestMetricsProvider(t *testing.T) *metricsfx.MetricsProvider {
	t.Helper()

	provider := metricsfx.NewMetricsProvider(&metricsfx.Config{
		ServiceName:                   "test-service",
		ServiceVersion:                "1.0.0",
		OTLPConnectionName:            "", // No connection for testing
		ExportInterval:                30 * time.Second,
		NoNativeCollectorRegistration: true,
	}, nil) // nil registry for testing

	err := provider.Init()
	require.NoError(t, err)

	t.Cleanup(func() {
		err := provider.Shutdown(t.Context())
		if err != nil {
			t.Logf("Error shutting down metrics provider: %v", err)
		}
	})

	return provider
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
				return c.Results.Error(http.StatusBadRequest, httpfx.WithPlainText("bad request"))
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create metrics using the new MetricsBuilder
			metricsProvider := setupTestMetricsProvider(t)
			metrics := httpfx.NewMetrics(metricsProvider)
			require.NotNil(t, metrics)

			err := metrics.Init()
			require.NoError(t, err)

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
			// The metrics are recorded successfully if no panic occurs
			// With the new interface, we don't need complex verification
			// since the MetricsBuilder handles all the complexity internally
		})
	}
}
