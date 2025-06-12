package logfx

import (
	"io"
	"log/slog"
)

type NewLoggerOption func(*Logger)

func WithConfig(config *Config) NewLoggerOption {
	return func(logger *Logger) {
		logger.Config = config
	}
}

func WithWriter(writer io.Writer) NewLoggerOption {
	return func(logger *Logger) {
		logger.Writer = writer
	}
}

func WithFromSlog(slog *slog.Logger) NewLoggerOption {
	return func(logger *Logger) {
		logger.Logger = slog
	}
}

func WithLevel(level slog.Level) NewLoggerOption {
	return func(logger *Logger) {
		logger.Config.Level = level.String()
	}
}

func WithDefaultLogger() NewLoggerOption {
	return func(logger *Logger) {
		logger.Config.DefaultLogger = true
	}
}

func WithPrettyMode(pretty bool) NewLoggerOption {
	return func(logger *Logger) {
		logger.Config.PrettyMode = pretty
	}
}

func WithAddSource(addSource bool) NewLoggerOption {
	return func(logger *Logger) {
		logger.Config.AddSource = addSource
	}
}

func WithOTLP(endpoint string, insecure bool) NewLoggerOption {
	return func(logger *Logger) {
		logger.Config.OTLPEndpoint = endpoint
		logger.Config.OTLPInsecure = insecure
	}
}

func WithLoki(uri string, label string) NewLoggerOption {
	return func(logger *Logger) {
		logger.Config.LokiURI = uri
		logger.Config.LokiLabel = label
	}
}
