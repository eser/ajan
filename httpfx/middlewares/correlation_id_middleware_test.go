package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eser/ajan/httpfx"
	"github.com/eser/ajan/httpfx/middlewares"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCorrelationIdMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		existingHeaderValue string
		wantHeaderExists    bool
	}{
		{ //nolint:exhaustruct
			name:             "no_existing_correlation_id",
			wantHeaderExists: true,
		},
		{
			name:                "existing_correlation_id",
			existingHeaderValue: "test-correlation-id",
			wantHeaderExists:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.existingHeaderValue != "" {
				req.Header.Set(middlewares.CorrelationIdHeader, tt.existingHeaderValue)
			}

			// Create a test response recorder
			w := httptest.NewRecorder()

			// Create a test context
			ctx := &httpfx.Context{
				Request:        req,
				ResponseWriter: w,
				Results:        httpfx.Results{},
			}

			// Create and execute the middleware
			middleware := middlewares.CorrelationIdMiddleware()
			result := middleware(ctx)
			require.NotNil(t, result)

			// Check the response header
			correlationID := w.Header().Get(middlewares.CorrelationIdHeader)
			if tt.wantHeaderExists {
				assert.NotEmpty(t, correlationID)

				if tt.existingHeaderValue != "" {
					assert.Equal(t, tt.existingHeaderValue, correlationID)
				}
			} else {
				assert.Empty(t, correlationID)
			}
		})
	}
}
