package middlewares

import (
	"log/slog"
	"time"

	"github.com/eser/ajan/httpfx"
	"github.com/eser/ajan/logfx"
)

const (
	// HTTP status code threshold for error logging.
	httpErrorThreshold = 400
)

// LoggingMiddleware creates HTTP request logging middleware that integrates with correlation ID.
func LoggingMiddleware(logger *logfx.Logger) httpfx.Handler {
	return func(ctx *httpfx.Context) httpfx.Result {
		startTime := time.Now()

		// Get correlation ID from context if available
		correlationID := GetCorrelationIDFromContext(ctx.Request.Context())

		// Log request start
		startArgs := []any{
			slog.String("method", ctx.Request.Method),
			slog.String("path", ctx.Request.URL.Path),
			slog.String("user_agent", ctx.Request.UserAgent()),
			slog.String("remote_addr", ctx.Request.RemoteAddr),
		}

		if correlationID != "" {
			startArgs = append(startArgs, slog.String("correlation_id", correlationID))
		}

		logger.InfoContext(ctx.Request.Context(), "HTTP request started", startArgs...)

		// Process the request
		result := ctx.Next()

		// Calculate duration
		duration := time.Since(startTime)

		// Log request completion
		endArgs := []any{
			slog.String("method", ctx.Request.Method),
			slog.String("path", ctx.Request.URL.Path),
			slog.Int("status_code", result.StatusCode()),
			slog.Duration("duration", duration),
		}

		if correlationID != "" {
			endArgs = append(endArgs, slog.String("correlation_id", correlationID))
		}

		if result.StatusCode() >= httpErrorThreshold {
			logger.WarnContext(
				ctx.Request.Context(),
				"HTTP request completed with error",
				endArgs...)
		} else {
			logger.InfoContext(ctx.Request.Context(), "HTTP request completed", endArgs...)
		}

		return result
	}
}
