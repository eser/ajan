package logfx

import (
	"context"
	"io"
	"log/slog"
)

type LogfxLogger struct {
	*slog.Logger
}

func NewLogger(w io.Writer, config *Config) (*LogfxLogger, error) {
	handler, err := NewHandler(w, config)
	if err != nil {
		return nil, err
	}

	return &LogfxLogger{Logger: slog.New(handler)}, nil
}

func NewLoggerAsDefault(w io.Writer, config *Config) (*LogfxLogger, error) {
	logger, err := NewLogger(w, config)
	if err != nil {
		return nil, err
	}

	slog.SetDefault(logger.Logger)

	return logger, nil
}

// Trace logs at [LevelTrace].
func (l *LogfxLogger) Trace(msg string, args ...any) {
	l.Log(context.Background(), LevelTrace, msg, args...)
}

// TraceContext logs at [LevelTrace] with the given context.
func (l *LogfxLogger) TraceContext(ctx context.Context, msg string, args ...any) {
	l.Log(ctx, LevelTrace, msg, args...)
}

// Fatal logs at [LevelFatal].
func (l *LogfxLogger) Fatal(msg string, args ...any) {
	l.Log(context.Background(), LevelFatal, msg, args...)
}

// FatalContext logs at [LevelFatal] with the given context.
func (l *LogfxLogger) FatalContext(ctx context.Context, msg string, args ...any) {
	l.Log(ctx, LevelFatal, msg, args...)
}

// Panic logs at [LevelPanic].
func (l *LogfxLogger) Panic(msg string, args ...any) {
	l.Log(context.Background(), LevelPanic, msg, args...)
}

// PanicContext logs at [LevelPanic] with the given context.
func (l *LogfxLogger) PanicContext(ctx context.Context, msg string, args ...any) {
	l.Log(ctx, LevelPanic, msg, args...)
}
