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

func TestCorsMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		method         string
		expectedOrigin string
	}{
		{
			name:           "get_request",
			method:         http.MethodGet,
			expectedOrigin: "*",
		},
		{
			name:           "post_request",
			method:         http.MethodPost,
			expectedOrigin: "*",
		},
		{
			name:           "options_request",
			method:         http.MethodOptions,
			expectedOrigin: "*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a test request
			req := httptest.NewRequest(tt.method, "/test", nil)

			// Create a test response recorder
			w := httptest.NewRecorder()

			// Create a test context
			ctx := &httpfx.Context{
				Request:        req,
				ResponseWriter: w,
				Results:        httpfx.Results{},
			}

			// Create and execute the middleware
			middleware := middlewares.CorsMiddleware()
			result := middleware(ctx)
			require.NotNil(t, result)

			// Check the CORS headers
			assert.Equal(t, tt.expectedOrigin, w.Header().Get(middlewares.AccessControlAllowOriginHeader))
		})
	}
}
