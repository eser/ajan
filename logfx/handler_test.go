package logfx_test

import (
	"bytes"
	"errors"
	"log/slog"
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
				Level:              "INFO",
				PrettyMode:         true,
				AddSource:          false,
				DefaultLogger:      false,
				OTLPConnectionName: "", // No connection for testing
			},
			expectedErr: nil,
		},
		{
			name:   "InvalidLogLevel",
			writer: &bytes.Buffer{},
			config: &logfx.Config{
				Level:              "INVALID",
				PrettyMode:         true,
				AddSource:          false,
				DefaultLogger:      false,
				OTLPConnectionName: "", // No connection for testing
			},
			expectedErr: logfx.ErrFailedToParseLogLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := logfx.NewHandler(tt.writer, tt.config, nil) // nil registry for testing

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
			}, nil)

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
		}, nil)
		err := handler.Handle(t.Context(), slog.NewRecord(time.Time{}, slog.LevelInfo, "test", 0))
		assert.EqualError(t, err, "failed to write log: failed to write")
	})
}

func TestHandler_WithAttrs(t *testing.T) {
	t.Parallel()

	handler := logfx.NewHandler(&bytes.Buffer{}, &logfx.Config{ //nolint:exhaustruct
		Level: "info",
	}, nil)
	newHandler := handler.WithAttrs(make([]slog.Attr, 0))
	// FIXME(@eser) should equal or not?
	assert.Equal(t, handler, newHandler)
}

func TestHandler_WithGroup(t *testing.T) {
	t.Parallel()

	handler := logfx.NewHandler(&bytes.Buffer{}, &logfx.Config{ //nolint:exhaustruct
		Level: "info",
	}, nil)
	newHandler := handler.WithGroup("test")
	assert.NotEqual(t, handler, newHandler)
}
