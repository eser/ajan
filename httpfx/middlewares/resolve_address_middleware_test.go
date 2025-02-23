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

func TestResolveAddressMiddleware(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []struct {
		name           string
		remoteAddr     string
		headers        map[string]string
		expectedOrigin string
		expectedAddr   string
		expectedStatus int
	}{
		{ //nolint:exhaustruct
			name:           "local_request",
			remoteAddr:     "127.0.0.1:12345",
			expectedOrigin: "local",
			expectedAddr:   "127.0.0.1:12345",
			expectedStatus: http.StatusNoContent,
		},
		{ //nolint:exhaustruct
			name:           "remote_request",
			remoteAddr:     "203.0.113.1:54321",
			expectedOrigin: "remote",
			expectedAddr:   "203.0.113.1:54321",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:       "request_with_x_forwarded_for",
			remoteAddr: "10.0.0.1:54321",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1",
			},
			expectedOrigin: "remote",
			expectedAddr:   "203.0.113.1",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:       "request_with_x_forwarded_for_priority",
			remoteAddr: "10.0.0.1:54321",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1",
				"X-Real-IP":       "203.0.113.3",
			},
			expectedOrigin: "remote",
			expectedAddr:   "203.0.113.1",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:       "request_with_x_real_ip",
			remoteAddr: "10.0.0.1:54321",
			headers: map[string]string{
				"X-Real-IP": "203.0.113.3",
			},
			expectedOrigin: "remote",
			expectedAddr:   "10.0.0.1:54321",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:       "request_with_multiple_proxies",
			remoteAddr: "10.0.0.1:54321",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.4, 10.0.0.2",
			},
			expectedOrigin: "remote",
			expectedAddr:   "203.0.113.4, 10.0.0.2",
			expectedStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a router with the resolve address middleware
			router := httpfx.NewRouter("/")
			router.Use(middlewares.ResolveAddressMiddleware())

			// Add a test route that returns the client address from context
			router.Route("GET /test", func(c *httpfx.Context) httpfx.Result {
				addr := c.Request.Context().Value(middlewares.ClientAddr).(string)         //nolint:forcetypeassert
				origin := c.Request.Context().Value(middlewares.ClientAddrOrigin).(string) //nolint:forcetypeassert

				assert.Equal(t, tt.expectedAddr, addr)
				assert.Equal(t, tt.expectedOrigin, origin)

				return c.Results.Ok()
			})

			// Create test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr

			// Add headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Execute request
			w := httptest.NewRecorder()
			router.GetMux().ServeHTTP(w, req)

			// Verify response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedOrigin == "local" {
				assert.Equal(t, "local: "+tt.expectedAddr, w.Header().Get("X-Request-Origin"))
			} else {
				assert.Equal(t, tt.expectedAddr, w.Header().Get("X-Request-Origin"))
			}
		})
	}
}

func TestGetClientAddrs(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []struct {
		name         string
		remoteAddr   string
		headers      map[string]string
		expectedAddr string
	}{
		{ //nolint:exhaustruct
			name:         "remote_addr_only",
			remoteAddr:   "203.0.113.1:54321",
			expectedAddr: "203.0.113.1:54321",
		},
		{
			name:       "with_x_forwarded_for",
			remoteAddr: "10.0.0.1:54321",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1",
			},
			expectedAddr: "203.0.113.1",
		},
		{
			name:       "with_x_forwarded_for_priority",
			remoteAddr: "10.0.0.1:54321",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1",
				"X-Real-IP":       "203.0.113.3",
			},
			expectedAddr: "203.0.113.1",
		},
		{
			name:       "with_x_real_ip",
			remoteAddr: "10.0.0.1:54321",
			headers: map[string]string{
				"X-Real-IP": "203.0.113.3",
			},
			expectedAddr: "10.0.0.1:54321",
		},
		{
			name:       "with_multiple_proxies",
			remoteAddr: "10.0.0.1:54321",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.4, 10.0.0.2",
			},
			expectedAddr: "203.0.113.4, 10.0.0.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr

			// Add headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Get client address
			addr := middlewares.GetClientAddrs(req)
			assert.Equal(t, tt.expectedAddr, addr)
		})
	}
}

func TestDetectLocalNetwork(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		addr      string
		wantLocal bool
		wantErr   bool
	}{
		{ //nolint:exhaustruct
			name:      "loopback_ipv4",
			addr:      "127.0.0.1",
			wantLocal: true,
		},
		{ //nolint:exhaustruct
			name:      "loopback_ipv4_with_port",
			addr:      "127.0.0.1:8080",
			wantLocal: true,
		},
		{ //nolint:exhaustruct
			name:      "remote_ipv4",
			addr:      "203.0.113.1",
			wantLocal: false,
		},
		{ //nolint:exhaustruct
			name:      "remote_ipv4_with_port",
			addr:      "203.0.113.1:8080",
			wantLocal: false,
		},
		{ //nolint:exhaustruct
			name:    "invalid_addr",
			addr:    "not-an-ip",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			isLocal, err := middlewares.DetectLocalNetwork(tt.addr)
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantLocal, isLocal)
		})
	}
}
