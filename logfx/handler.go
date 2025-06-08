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
)

type Handler struct {
	InnerHandler slog.Handler

	InnerWriter io.Writer
	InnerConfig *Config
}

func NewHandler(w io.Writer, config *Config) (*Handler, error) {
	level, err := ParseLevel(config.Level, false)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToParseLogLevel, err)
	}

	opts := &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: ReplacerGenerator(config.PrettyMode),
		AddSource:   config.AddSource,
	}

	innerHandler := slog.NewJSONHandler(w, opts)

	return &Handler{
		InnerHandler: innerHandler,
		InnerWriter:  w,
		InnerConfig:  config,
	}, nil
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.InnerHandler.Enabled(ctx, level)
}

func (h *Handler) Handle(ctx context.Context, rec slog.Record) error {
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
		InnerHandler: h.InnerHandler.WithAttrs(attrs),
		InnerWriter:  h.InnerWriter,
		InnerConfig:  h.InnerConfig,
	}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		InnerHandler: h.InnerHandler.WithGroup(name),
		InnerWriter:  h.InnerWriter,
		InnerConfig:  h.InnerConfig,
	}
}
