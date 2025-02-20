package logfx

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

type Handler struct {
	InnerHandler slog.Handler

	InnerWriter io.Writer
	InnerConfig *Config
}

func NewHandler(w io.Writer, config *Config) (*Handler, error) {
	level, err := ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("failed to parse log level: %w", err)
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
			return fmt.Errorf("failed to write log: %w", err)
		}
	}

	err := h.InnerHandler.Handle(ctx, rec)
	if err != nil {
		return fmt.Errorf("failed to handle log: %w", err)
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
