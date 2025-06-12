package middlewares_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eser/ajan/httpfx"
	"github.com/eser/ajan/httpfx/middlewares"
	"github.com/eser/ajan/logfx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCorrelationIDIntegration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		incomingCorrelationID  string
		expectCorrelationID    bool
		expectInLogs           bool
		expectInResponseHeader bool
	}{
		{
			name:                   "with existing correlation ID",
			incomingCorrelationID:  "test-correlation-123",
			expectCorrelationID:    true,
			expectInLogs:           true,
			expectInResponseHeader: true,
		},
		{
			name:                   "without existing correlation ID - should generate one",
			incomingCorrelationID:  "",
			expectCorrelationID:    true,
			expectInLogs:           true,
			expectInResponseHeader: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			correlationIDFromResponse, logOutput := executeTestRequest(t, tt.incomingCorrelationID)

			// Verify response header
			if tt.expectInResponseHeader {
				assert.NotEmpty(t, correlationIDFromResponse)

				if tt.incomingCorrelationID != "" {
					assert.Equal(t, tt.incomingCorrelationID, correlationIDFromResponse)
				}
			}

			// Verify logs contain correlation ID
			if tt.expectInLogs {
				verifyCorrelationInLogs(
					t,
					logOutput,
					tt.incomingCorrelationID,
					correlationIDFromResponse,
				)
			}
		})
	}
}

func executeTestRequest(t *testing.T, incomingCorrelationID string) (string, string) {
	t.Helper()

	// Capture logs
	var logBuffer bytes.Buffer

	// Create logger with JSON output for easy parsing
	logConfig := &logfx.Config{
		Level:         "DEBUG",
		PrettyMode:    false,
		AddSource:     false,
		DefaultLogger: false,
		OTLPEndpoint:  "",
		OTLPInsecure:  false,
		LokiURI:       "",
		LokiLabel:     "",
	}
	logger := logfx.NewLogger(
		logfx.WithWriter(&logBuffer),
		logfx.WithConfig(logConfig),
	)

	// Create router with middleware chain
	router := httpfx.NewRouter("/")

	// Add correlation middleware first
	router.Use(middlewares.CorrelationIDMiddleware())

	// Add logging middleware second
	router.Use(middlewares.LoggingMiddleware(logger))

	// Create a test handler that logs something
	testHandler := func(ctx *httpfx.Context) httpfx.Result {
		// This simulates application logging within the request handler
		logger.InfoContext(ctx.Request.Context(), "Processing business logic",
			slog.String("operation", "test-operation"),
			slog.String("user_id", "user123"),
		)

		return ctx.Results.PlainText([]byte("success"))
	}

	// Add the route
	router.Route("GET /test", testHandler)

	// Create a test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	if incomingCorrelationID != "" {
		req.Header.Set(middlewares.CorrelationIDHeader, incomingCorrelationID)
	}

	// Create a test response recorder
	w := httptest.NewRecorder()

	// Execute the request through the router
	router.GetMux().ServeHTTP(w, req)

	correlationIDFromResponse := w.Header().Get(middlewares.CorrelationIDHeader)
	logOutput := logBuffer.String()

	return correlationIDFromResponse, logOutput
}

func verifyCorrelationInLogs(
	t *testing.T,
	logOutput, incomingCorrelationID, correlationIDFromResponse string,
) {
	t.Helper()

	assert.NotEmpty(t, logOutput)

	// Split log entries (each line is a JSON log entry)
	logLines := strings.Split(strings.TrimSpace(logOutput), "\n")
	assert.GreaterOrEqual(t, len(logLines), 2) // At least request start and business logic

	// Verify correlation ID is present in all log entries
	expectedCorrelationID := incomingCorrelationID
	if expectedCorrelationID == "" {
		expectedCorrelationID = correlationIDFromResponse // Should match generated one
	}

	for i, logLine := range logLines {
		if strings.TrimSpace(logLine) == "" {
			continue
		}

		var logEntry map[string]any
		err := json.Unmarshal([]byte(logLine), &logEntry)
		require.NoError(t, err, "Failed to parse log line %d: %s", i, logLine)

		// Check that correlation_id is present
		correlationIDInLog, hasCorrelationID := logEntry["correlation_id"]
		assert.True(t, hasCorrelationID, "Log entry %d missing correlation_id: %s", i, logLine)

		if hasCorrelationID {
			assert.Equal(t, expectedCorrelationID, correlationIDInLog,
				"Correlation ID mismatch in log entry %d", i)
		}
	}

	// Verify specific log entries contain expected information
	foundEntries := verifyLogEntries(t, logLines)
	assert.True(t, foundEntries.businessLogic, "Business logic log entry not found")
	assert.True(t, foundEntries.requestStart, "Request start log entry not found")
	assert.True(t, foundEntries.requestEnd, "Request end log entry not found")
}

type foundLogEntries struct {
	businessLogic bool
	requestStart  bool
	requestEnd    bool
}

func verifyLogEntries(t *testing.T, logLines []string) foundLogEntries {
	t.Helper()

	var found foundLogEntries

	for _, logLine := range logLines {
		if strings.TrimSpace(logLine) == "" {
			continue
		}

		var logEntry map[string]any
		err := json.Unmarshal([]byte(logLine), &logEntry)
		require.NoError(t, err)

		msg, hasMsg := logEntry["msg"]
		if !hasMsg {
			continue
		}

		switch msg {
		case "Processing business logic":
			found.businessLogic = true

			assert.Equal(t, "test-operation", logEntry["operation"])
			assert.Equal(t, "user123", logEntry["user_id"])
		case "HTTP request started":
			found.requestStart = true

			assert.Equal(t, "GET", logEntry["method"])
			assert.Equal(t, "/test", logEntry["path"])
		case "HTTP request completed":
			found.requestEnd = true

			assert.Equal(t, "GET", logEntry["method"])
			assert.Equal(t, "/test", logEntry["path"])
			assert.InEpsilon(t, 200.0, logEntry["status_code"], 0.01)
		}
	}

	return found
}

func TestCorrelationIDFromContext(t *testing.T) {
	t.Parallel()

	// Test the correlation ID context extraction
	ctx := t.Context()

	// Without correlation ID
	correlationID := middlewares.GetCorrelationIDFromContext(ctx)
	assert.Empty(t, correlationID)

	// With correlation ID
	expectedID := "test-123"
	ctxWithID := context.WithValue(ctx, logfx.CorrelationIDContextKey{}, expectedID)
	correlationID = middlewares.GetCorrelationIDFromContext(ctxWithID)
	assert.Equal(t, expectedID, correlationID)
}
