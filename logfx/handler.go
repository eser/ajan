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
	ErrFailedToInitLoki      = errors.New("failed to initialize loki client")
	ErrFailedToInitOTLP      = errors.New("failed to initialize OTLP client")
)

type Handler struct {
	InitError error

	InnerHandler slog.Handler

	InnerWriter io.Writer
	InnerConfig *Config

	// Export clients (prioritized: OTLP -> Loki)
	OTLPClient *OTLPClient
	LokiClient *LokiClient
}

func NewHandler(w io.Writer, config *Config) *Handler {
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

	// Initialize OTLP client if configured (preferred)
	var otlpClient *OTLPClient
	if config.OTLPEndpoint != "" {
		otlpClient, err = NewOTLPClient(config.OTLPEndpoint, config.OTLPInsecure)
		if err != nil {
			if initError != nil {
				initError = fmt.Errorf("%w; %w: %w", initError, ErrFailedToInitOTLP, err)
			} else {
				initError = fmt.Errorf("%w: %w", ErrFailedToInitOTLP, err)
			}
		}
	}

	// Initialize Loki client if configured (fallback or additional)
	var lokiClient *LokiClient
	if config.LokiURI != "" {
		lokiClient, err = NewLokiClient(config.LokiURI, config.LokiLabel)
		if err != nil {
			if initError != nil {
				initError = fmt.Errorf("%w; %w: %w", initError, ErrFailedToInitLoki, err)
			} else {
				initError = fmt.Errorf("%w: %w", ErrFailedToInitLoki, err)
			}
		}
	}

	return &Handler{
		InitError: initError,

		InnerHandler: innerHandler,
		InnerWriter:  w,
		InnerConfig:  config,

		OTLPClient: otlpClient,
		LokiClient: lokiClient,
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

	// Send to OTLP collector if configured (preferred for OpenTelemetry)
	if h.OTLPClient != nil {
		h.OTLPClient.SendLog(ctx, rec)
	}

	// Send to Loki if configured (legacy/additional option)
	if h.LokiClient != nil {
		h.LokiClient.SendLog(ctx, rec)
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

		OTLPClient: h.OTLPClient,
		LokiClient: h.LokiClient,
	}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		InitError: h.InitError,

		InnerHandler: h.InnerHandler.WithGroup(name),

		InnerWriter: h.InnerWriter,
		InnerConfig: h.InnerConfig,

		OTLPClient: h.OTLPClient,
		LokiClient: h.LokiClient,
	}
}

// Shutdown gracefully shuts down any active export clients.
func (h *Handler) Shutdown(ctx context.Context) error {
	var err error

	if h.OTLPClient != nil {
		if shutdownErr := h.OTLPClient.Shutdown(ctx); shutdownErr != nil {
			err = fmt.Errorf("failed to shutdown OTLP client: %w", shutdownErr)
		}
	}

	// Note: Loki client doesn't need shutdown as it uses simple HTTP client

	return err
}
