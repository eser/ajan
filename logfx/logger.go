package logfx

import (
	"context"
	"io"
	"log/slog"
	"os"
)

type Logger struct {
	*slog.Logger
}

func NewLogger(w io.Writer, config *Config) *Logger {
	handler := NewHandler(w, config)

	logger := &Logger{Logger: slog.New(handler)}

	if handler.InitError != nil {
		logger.Warn(
			"an error occurred while initializing the logger",
			slog.String("error", handler.InitError.Error()),
			slog.Any("config", config),
		)
	}

	return logger
}

func NewLoggerWithDefaults() *Logger {
	return NewLogger(os.Stdout, &Config{ //nolint:exhaustruct
		Level: DefaultLogLevel,
	})
}

func NewLoggerFromSlog(slog *slog.Logger) *Logger {
	return &Logger{Logger: slog}
}

func (l *Logger) SetAsDefault() {
	slog.SetDefault(l.Logger)
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
