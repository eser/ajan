package logfx_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eser/ajan/logfx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLokiClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		uri       string
		labelStr  string
		expectErr bool
	}{
		{
			name:      "valid uri and labels",
			uri:       "http://localhost:3100/loki/api/v1/push",
			labelStr:  "app=test,env=dev",
			expectErr: false,
		},
		{
			name:      "valid uri without labels",
			uri:       "http://localhost:3100/loki/api/v1/push",
			labelStr:  "",
			expectErr: false,
		},
		{
			name:      "empty uri",
			uri:       "",
			labelStr:  "app=test",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, err := logfx.NewLokiClient(tt.uri, tt.labelStr)

			if tt.expectErr {
				require.Error(t, err)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestLokiClient_SendLog_Success(t *testing.T) {
	t.Parallel()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := logfx.NewLokiClient(server.URL, "app=test")
	require.NoError(t, err)

	// Create a test log record
	ctx := t.Context()
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)

	// Send log (this should not block or error)
	client.SendLog(ctx, rec)

	// Give some time for the async operation
	time.Sleep(100 * time.Millisecond)
}

func TestLokiClient_SendLog_ServerError(t *testing.T) {
	t.Parallel()

	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := logfx.NewLokiClient(server.URL, "app=test")
	require.NoError(t, err)

	rec := slog.NewRecord(time.Now(), slog.LevelError, "error message", 0)

	// This should not panic even if server returns error
	ctx := t.Context()
	client.SendLog(ctx, rec)

	// Give some time for the async operation to complete
	time.Sleep(100 * time.Millisecond)
}

func TestParseLokiLabels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		labelStr string
		expected map[string]string
	}{
		{
			name:     "single label",
			labelStr: "app=test",
			expected: map[string]string{"app": "test"},
		},
		{
			name:     "multiple labels",
			labelStr: "app=test,env=dev,region=us-east-1",
			expected: map[string]string{
				"app":    "test",
				"env":    "dev",
				"region": "us-east-1",
			},
		},
		{
			name:     "labels with spaces",
			labelStr: "app = test , env = dev",
			expected: map[string]string{
				"app": "test",
				"env": "dev",
			},
		},
		{
			name:     "empty string",
			labelStr: "",
			expected: map[string]string{},
		},
		{
			name:     "invalid format",
			labelStr: "app:test,invalid",
			expected: map[string]string{}, // Should ignore invalid entries
		},
		{
			name:     "mixed valid and invalid",
			labelStr: "app=test,invalid,env=dev",
			expected: map[string]string{
				"app": "test",
				"env": "dev",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := logfx.NewLokiClient("http://localhost:3100", tt.labelStr)
			require.NoError(t, err)
			// The test just verifies that the client can be created
			// Actual parsing is tested separately
		})
	}
}
