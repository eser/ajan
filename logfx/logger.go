package logfx

import (
	"context"
	"io"
	"log/slog"
)

type Logger struct {
	*slog.Logger
}

func NewLogger(w io.Writer, config *Config) (*Logger, error) {
	handler, err := NewHandler(w, config)
	if err != nil {
		return nil, err
	}

	return &Logger{Logger: slog.New(handler)}, nil
}

func NewLoggerAsDefault(w io.Writer, config *Config) (*Logger, error) {
	logger, err := NewLogger(w, config)
	if err != nil {
		return nil, err
	}

	slog.SetDefault(logger.Logger)

	return logger, nil
}

func NewLoggerFromSlog(slog *slog.Logger) *Logger {
	return &Logger{Logger: slog}
}

// Trace logs at [LevelTrace].
func (l *Logger) Trace(msg string, args ...any) {
	l.Log(context.Background(), LevelTrace, msg, args...)
}

// TraceContext logs at [LevelTrace] with the given context.
func (l *Logger) TraceContext(ctx context.Context, msg string, args ...any) {
	l.Log(ctx, LevelTrace, msg, args...)
}

// Fatal logs at [LevelFatal].
func (l *Logger) Fatal(msg string, args ...any) {
	l.Log(context.Background(), LevelFatal, msg, args...)
}

// FatalContext logs at [LevelFatal] with the given context.
func (l *Logger) FatalContext(ctx context.Context, msg string, args ...any) {
	l.Log(ctx, LevelFatal, msg, args...)
}

// Panic logs at [LevelPanic].
func (l *Logger) Panic(msg string, args ...any) {
	l.Log(context.Background(), LevelPanic, msg, args...)
}

// PanicContext logs at [LevelPanic] with the given context.
func (l *Logger) PanicContext(ctx context.Context, msg string, args ...any) {
	l.Log(ctx, LevelPanic, msg, args...)
}
