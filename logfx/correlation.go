package logfx

import "context"

// CorrelationIDContextKey is the context key for correlation ID - shared with httpfx middleware.
type CorrelationIDContextKey struct{}

// getCorrelationIDFromContext extracts correlation ID from various context sources.
func getCorrelationIDFromContext(ctx context.Context) string {
	// Try HTTP correlation ID context key (most common in HTTP requests)
	if correlationID, ok := ctx.Value(CorrelationIDContextKey{}).(string); ok &&
		correlationID != "" {
		return correlationID
	}

	// Could add other correlation ID sources here in the future
	// e.g., gRPC metadata, other middleware context keys, etc.

	return ""
}
