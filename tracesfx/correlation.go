package tracesfx

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// CorrelationIDContextKey is the context key for correlation IDs.
// This should match the one used in logfx for seamless integration.
type CorrelationIDContextKey struct{}

// GetCorrelationIDFromContext extracts the correlation ID from context.
// This integrates with logfx's correlation system.
func GetCorrelationIDFromContext(ctx context.Context) string {
	if correlationID, ok := ctx.Value(CorrelationIDContextKey{}).(string); ok {
		return correlationID
	}

	return ""
}

// SetCorrelationIDInContext sets the correlation ID in context.
func SetCorrelationIDInContext(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationIDContextKey{}, correlationID)
}

// AddCorrelationToSpan adds correlation ID as an attribute to the span if available.
func AddCorrelationToSpan(ctx context.Context, span *Span) {
	if correlationID := GetCorrelationIDFromContext(ctx); correlationID != "" {
		span.SetAttributes(attribute.String("correlation_id", correlationID))
	}
}

// StartSpanWithCorrelation starts a new span and automatically adds correlation ID.
func (t *Tracer) StartSpanWithCorrelation(
	ctx context.Context,
	name string,
	opts ...trace.SpanStartOption,
) (context.Context, *Span) {
	ctx, span := t.Start(ctx, name, opts...)
	AddCorrelationToSpan(ctx, span)

	return ctx, span
}

// GetTraceIDFromContext extracts the trace ID from the current span context.
// This can be used by logfx to include trace IDs in log entries.
func GetTraceIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}

	return ""
}

// GetSpanIDFromContext extracts the span ID from the current span context.
func GetSpanIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}

	return ""
}
