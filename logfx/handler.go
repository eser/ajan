package logfx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

var (
	ErrFailedToParseLogLevel = errors.New("failed to parse log level")
	ErrFailedToWriteLog      = errors.New("failed to write log")
	ErrFailedToHandleLog     = errors.New("failed to handle log")
	ErrConnectionNotFound    = errors.New("connection not found")
	ErrConnectionNotOTLP     = errors.New("connection is not an OTLP connection")
)

type Handler struct {
	InitError error

	InnerHandler slog.Handler

	InnerWriter io.Writer
	InnerConfig *Config

	// OTLP bridge for sending logs
	OTLPBridge *OTLPBridge
}

func NewHandler(w io.Writer, config *Config, registry ConnectionRegistry) *Handler {
	var initError error

	var l slog.Level

	level, err := ParseLevel(config.Level, false)
	if err != nil {
		initError = fmt.Errorf("%w (level=%q): %w", ErrFailedToParseLogLevel, config.Level, err)

		// FIXME(@eser) default to info level
		l = LevelInfo
		level = &l
	}

	opts := &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: ReplacerGenerator(config.PrettyMode),
		AddSource:   config.AddSource,
	}

	innerHandler := slog.NewJSONHandler(w, opts)

	// Create OTLP bridge if registry is provided
	var otlpBridge *OTLPBridge
	if registry != nil {
		otlpBridge = NewOTLPBridge(registry)
	}

	return &Handler{
		InitError: initError,

		InnerHandler: innerHandler,
		InnerWriter:  w,
		InnerConfig:  config,

		OTLPBridge: otlpBridge,
	}
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.InnerHandler.Enabled(ctx, level)
}

func (h *Handler) Handle(ctx context.Context, rec slog.Record) error {
	// Add correlation ID from context to the record if available
	if correlationID := getCorrelationIDFromContext(ctx); correlationID != "" {
		rec.AddAttrs(slog.String("correlation_id", correlationID))
	}

	// Send to OTLP collector if configured
	if h.InnerConfig.OTLPConnectionName != "" && h.OTLPBridge != nil {
		h.sendToOTLP(ctx, rec)
	}

	if h.InnerConfig.PrettyMode {
		out := strings.Builder{}

		timeStr := rec.Time.Format("15:04:05.000")

		out.WriteString(Colored(ColorDimGray, timeStr))
		out.WriteRune(' ')

		out.WriteString(LevelEncoderColored(rec.Level))

		out.WriteRune(' ')
		out.WriteString(rec.Message)
		out.WriteRune(' ')

		_, err := io.WriteString(h.InnerWriter, out.String())
		if err != nil {
			return fmt.Errorf("%w: %w", ErrFailedToWriteLog, err)
		}
	}

	err := h.InnerHandler.Handle(ctx, rec)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToHandleLog, err)
	}

	return nil
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		InitError: h.InitError,

		InnerHandler: h.InnerHandler.WithAttrs(attrs),

		InnerWriter: h.InnerWriter,
		InnerConfig: h.InnerConfig,

		OTLPBridge: h.OTLPBridge,
	}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		InitError: h.InitError,

		InnerHandler: h.InnerHandler.WithGroup(name),

		InnerWriter: h.InnerWriter,
		InnerConfig: h.InnerConfig,

		OTLPBridge: h.OTLPBridge,
	}
}

// Shutdown gracefully shuts down any active export clients.
func (h *Handler) Shutdown(ctx context.Context) error {
	// The connection registry handles shutdown of connections
	// No need to shutdown OTLP client directly
	return nil
}

// sendToOTLP sends a log record to the OTLP connection asynchronously.
func (h *Handler) sendToOTLP(ctx context.Context, rec slog.Record) {
	go func() {
		if err := h.OTLPBridge.SendLog(ctx, h.InnerConfig.OTLPConnectionName, rec); err != nil {
			// Use slog for error logging to avoid infinite recursion
			slog.Error("Failed to send log to OTLP collector", "error", err)
		}
	}()
}
