package tracesfx

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Tracer provides a clean interface for creating and managing spans.
type Tracer struct {
	tracer oteltrace.Tracer
}

// Start creates a new span with the given name and options.
// The returned span must be ended by the caller.
func (t *Tracer) Start(
	ctx context.Context,
	name string,
	opts ...oteltrace.SpanStartOption,
) (context.Context, *Span) { //nolint:spancheck
	ctx, span := t.tracer.Start(ctx, name, opts...)

	return ctx, &Span{span: span} //nolint:spancheck
}

// Span wraps an OpenTelemetry span with additional convenience methods.
type Span struct {
	span oteltrace.Span
}

// SetAttributes sets attributes on the span.
func (s *Span) SetAttributes(attrs ...attribute.KeyValue) {
	s.span.SetAttributes(attrs...)
}

// SetStatus sets the status of the span.
func (s *Span) SetStatus(code codes.Code, description string) {
	s.span.SetStatus(code, description)
}

// AddEvent adds an event to the span.
func (s *Span) AddEvent(name string, attrs ...attribute.KeyValue) {
	s.span.AddEvent(name, oteltrace.WithAttributes(attrs...))
}

// RecordError records an error as a span event.
func (s *Span) RecordError(err error, attrs ...attribute.KeyValue) {
	s.span.RecordError(err, oteltrace.WithAttributes(attrs...))
}

// End ends the span.
func (s *Span) End() {
	s.span.End()
}

// SpanContext returns the span context.
func (s *Span) SpanContext() oteltrace.SpanContext {
	return s.span.SpanContext()
}

// IsRecording returns true if the span is recording.
func (s *Span) IsRecording() bool {
	return s.span.IsRecording()
}

// Unwrap returns the underlying OpenTelemetry span.
func (s *Span) Unwrap() oteltrace.Span {
	return s.span
}
