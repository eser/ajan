package logfx_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eser/ajan/logfx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFailWriter struct{}

func (m *mockFailWriter) Write(p []byte) (int, error) {
	return 0, errors.New("failed to write") //nolint:err113
}

func TestNewHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		writer      *bytes.Buffer
		config      *logfx.Config
		expectedErr error
	}{
		{
			name:   "ValidConfig",
			writer: &bytes.Buffer{},
			config: &logfx.Config{
				Level:        "INFO",
				PrettyMode:   true,
				AddSource:    false,
				OTLPEndpoint: "",
				OTLPInsecure: false,
				LokiURI:      "",
				LokiLabel:    "",
			},
			expectedErr: nil,
		},
		{
			name:   "InvalidLogLevel",
			writer: &bytes.Buffer{},
			config: &logfx.Config{
				Level:        "INVALID",
				PrettyMode:   true,
				AddSource:    false,
				OTLPEndpoint: "",
				OTLPInsecure: false,
				LokiURI:      "",
				LokiLabel:    "",
			},
			expectedErr: logfx.ErrFailedToParseLogLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := logfx.NewHandler(tt.writer, tt.config)

			if tt.expectedErr != nil {
				require.Error(t, handler.InitError)
				require.ErrorIs(t, handler.InitError, tt.expectedErr)

				return
			}

			require.NoError(t, handler.InitError)
			assert.NotNil(t, handler)
		})
	}
}

func TestHandler_Handle(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []struct {
		name     string
		level    string
		record   slog.Record
		expected string
	}{
		{
			name:     "Trace",
			level:    "trace",
			record:   slog.NewRecord(time.Time{}, logfx.LevelTrace, "test", 0),
			expected: "\x1b[90m00:00:00.000\x1b[0m \x1b[94mTRACE\x1b[0m test {}\n",
		},
		{
			name:     "Debug",
			level:    "debug",
			record:   slog.NewRecord(time.Time{}, logfx.LevelDebug, "test", 0),
			expected: "\x1b[90m00:00:00.000\x1b[0m \x1b[94mDEBUG\x1b[0m test {}\n",
		},
		{
			name:     "Info",
			level:    "info",
			record:   slog.NewRecord(time.Time{}, logfx.LevelInfo, "test", 0),
			expected: "\x1b[90m00:00:00.000\x1b[0m \x1b[32mINFO\x1b[0m test {}\n",
		},
		{
			name:     "Warn",
			level:    "warn",
			record:   slog.NewRecord(time.Time{}, logfx.LevelWarn, "test", 0),
			expected: "\x1b[90m00:00:00.000\x1b[0m \x1b[33mWARN\x1b[0m test {}\n",
		},
		{
			name:     "Error",
			level:    "error",
			record:   slog.NewRecord(time.Time{}, logfx.LevelError, "test", 0),
			expected: "\x1b[90m00:00:00.000\x1b[0m \x1b[31mERROR\x1b[0m test {}\n",
		},
		{
			name:     "Fatal",
			level:    "fatal",
			record:   slog.NewRecord(time.Time{}, logfx.LevelFatal, "test", 0),
			expected: "\x1b[90m00:00:00.000\x1b[0m \x1b[31mFATAL\x1b[0m test {}\n",
		},
		{
			name:     "Panic",
			level:    "panic",
			record:   slog.NewRecord(time.Time{}, logfx.LevelPanic, "test", 0),
			expected: "\x1b[90m00:00:00.000\x1b[0m \x1b[31mPANIC\x1b[0m test {}\n",
		},
		{
			name:     "UnknownLevel",
			level:    "panic",
			record:   slog.NewRecord(time.Time{}, 77, "test", 0),
			expected: "\x1b[90m00:00:00.000\x1b[0m \x1b[31mPANIC+61\x1b[0m test {}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			writer := &bytes.Buffer{}
			handler := logfx.NewHandler(writer, &logfx.Config{ //nolint:exhaustruct
				Level:      tt.level,
				PrettyMode: true,
			})

			err := handler.Handle(t.Context(), tt.record)
			require.NoError(t, err)

			assert.Contains(t, writer.String(), tt.expected)
		})
	}

	t.Run("failed to write log", func(t *testing.T) {
		t.Parallel()

		handler := logfx.NewHandler(&mockFailWriter{}, &logfx.Config{ //nolint:exhaustruct
			Level:      "info",
			PrettyMode: true,
		})
		err := handler.Handle(t.Context(), slog.NewRecord(time.Time{}, slog.LevelInfo, "test", 0))
		assert.EqualError(t, err, "failed to write log: failed to write")
	})
}

func TestHandler_WithAttrs(t *testing.T) {
	t.Parallel()

	handler := logfx.NewHandler(&bytes.Buffer{}, &logfx.Config{ //nolint:exhaustruct
		Level: "info",
	})
	newHandler := handler.WithAttrs(make([]slog.Attr, 0))
	// FIXME(@eser) should equal or not?
	assert.Equal(t, handler, newHandler)
}

func TestHandler_WithGroup(t *testing.T) {
	t.Parallel()

	handler := logfx.NewHandler(&bytes.Buffer{}, &logfx.Config{ //nolint:exhaustruct
		Level: "info",
	})
	newHandler := handler.WithGroup("test")
	assert.NotEqual(t, handler, newHandler)
}

func TestHandler_LokiIntegration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		lokiURI      string
		lokiLabel    string
		expectLoki   bool
		expectError  bool
		responseCode int
	}{
		{
			name:         "loki configured and working",
			lokiURI:      "", // Will be set to test server URL
			lokiLabel:    "app=test,env=dev",
			expectLoki:   true,
			expectError:  false,
			responseCode: 200,
		},
		{
			name:         "loki not configured",
			lokiURI:      "",
			lokiLabel:    "",
			expectLoki:   false,
			expectError:  false,
			responseCode: 0,
		},
		{
			name:         "loki server error",
			lokiURI:      "", // Will be set to test server URL
			lokiLabel:    "app=test",
			expectLoki:   true,
			expectError:  false, // Handler should not fail even if Loki fails
			responseCode: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			testLokiIntegrationCase(t, tt)
		})
	}
}

func testLokiIntegrationCase(t *testing.T, tt struct {
	name         string
	lokiURI      string
	lokiLabel    string
	expectLoki   bool
	expectError  bool
	responseCode int
},
) {
	t.Helper()

	var lokiRequestReceived atomic.Bool

	var receivedPayload *logfx.LokiPayload

	// Create test server for Loki if needed
	server := createTestServerIfNeeded(t, tt, &lokiRequestReceived, &receivedPayload)
	if server != nil {
		defer server.Close()
		tt.lokiURI = server.URL
	}

	// Create and test handler
	handler, buf := createAndTestHandler(t, tt)

	// Create and handle a log record
	testLogRecord(t, handler, buf, tt, &lokiRequestReceived, receivedPayload)
}

func createTestServerIfNeeded(t *testing.T, tt struct {
	name         string
	lokiURI      string
	lokiLabel    string
	expectLoki   bool
	expectError  bool
	responseCode int
}, lokiRequestReceived *atomic.Bool, receivedPayload **logfx.LokiPayload,
) *httptest.Server {
	t.Helper()

	if !tt.expectLoki {
		return nil
	}

	return httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lokiRequestReceived.Store(true)

			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			if tt.responseCode == 200 {
				var payload logfx.LokiPayload

				err := json.NewDecoder(r.Body).Decode(&payload)
				if err == nil {
					*receivedPayload = &payload
				}
			}

			w.WriteHeader(tt.responseCode)
		}),
	)
}

func createAndTestHandler(t *testing.T, tt struct {
	name         string
	lokiURI      string
	lokiLabel    string
	expectLoki   bool
	expectError  bool
	responseCode int
},
) (*logfx.Handler, *bytes.Buffer) {
	t.Helper()

	// Create config
	config := &logfx.Config{
		Level:        "INFO",
		PrettyMode:   false,
		AddSource:    false,
		LokiURI:      tt.lokiURI,
		LokiLabel:    tt.lokiLabel,
		OTLPEndpoint: "",
		OTLPInsecure: false,
	}

	// Create handler
	var buf bytes.Buffer
	handler := logfx.NewHandler(&buf, config)

	// Verify initialization
	if tt.expectError {
		require.Error(t, handler.InitError)
	} else {
		if tt.expectLoki {
			assert.NotNil(t, handler.LokiClient)
		} else {
			assert.Nil(t, handler.LokiClient)
		}
	}

	return handler, &buf
}

func testLogRecord(t *testing.T, handler *logfx.Handler, buf *bytes.Buffer, tt struct {
	name         string
	lokiURI      string
	lokiLabel    string
	expectLoki   bool
	expectError  bool
	responseCode int
}, lokiRequestReceived *atomic.Bool, receivedPayload *logfx.LokiPayload,
) {
	t.Helper()

	// Create and handle a log record
	ctx := t.Context()
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "test log message", 0)
	rec.AddAttrs(
		slog.String("user_id", "12345"),
		slog.String("action", "login"),
	)

	err := handler.Handle(ctx, rec)
	require.NoError(t, err, "Handler.Handle should not fail even if Loki fails")

	// Verify log was written to buffer (normal logging should always work)
	logOutput := buf.String()
	assert.Contains(t, logOutput, "test log message")

	// Give time for async Loki call to complete
	if tt.expectLoki {
		time.Sleep(200 * time.Millisecond)

		// Verify Loki received the request
		assert.True(t, lokiRequestReceived.Load(), "Loki should have received a request")

		if tt.responseCode == 200 && receivedPayload != nil {
			verifyLokiPayload(t, receivedPayload)
		}
	} else {
		assert.False(t, lokiRequestReceived.Load(), "Loki should not have received a request")
	}
}

func verifyLokiPayload(t *testing.T, receivedPayload *logfx.LokiPayload) {
	t.Helper()

	// Verify payload structure
	assert.Len(t, receivedPayload.Streams, 1)
	stream := receivedPayload.Streams[0]

	// Verify labels
	assert.Equal(t, "test", stream.Stream["app"])
	assert.Equal(t, "dev", stream.Stream["env"])
	assert.Equal(t, "INFO", stream.Stream["level"])

	// Verify log entry
	assert.Len(t, stream.Values, 1)
	assert.Len(t, stream.Values[0], 2) // timestamp and log line

	// Parse the log line
	var logData map[string]any
	err := json.Unmarshal([]byte(stream.Values[0][1]), &logData)
	require.NoError(t, err)

	assert.Equal(t, "test log message", logData["msg"])
	assert.Equal(t, "INFO", logData["level"])
	assert.Equal(t, "12345", logData["user_id"])
	assert.Equal(t, "login", logData["action"])
}
