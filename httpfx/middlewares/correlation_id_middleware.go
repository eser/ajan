package middlewares

import (
	"context"

	"github.com/eser/ajan/httpfx"
	"github.com/eser/ajan/lib"
	"github.com/eser/ajan/logfx"
)

const CorrelationIDHeader = "X-Correlation-ID"

// GetCorrelationIDFromContext extracts correlation ID from context.
func GetCorrelationIDFromContext(ctx context.Context) string {
	if correlationID, ok := ctx.Value(logfx.CorrelationIDContextKey{}).(string); ok {
		return correlationID
	}

	return ""
}

func CorrelationIDMiddleware() httpfx.Handler {
	return func(ctx *httpfx.Context) httpfx.Result {
		// FIXME(@eser) no need to check if the header is specified
		correlationID := ctx.Request.Header.Get(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = lib.IDsGenerateUnique()
		}

		// Inject correlation ID into request context for use by logging and other middleware
		newContext := context.WithValue(
			ctx.Request.Context(),
			logfx.CorrelationIDContextKey{},
			correlationID,
		)
		ctx.UpdateContext(newContext)

		result := ctx.Next()

		ctx.ResponseWriter.Header().Set(CorrelationIDHeader, correlationID)

		return result
	}
}
