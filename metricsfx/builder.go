package metricsfx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	ErrFailedToCreateCounter   = errors.New("failed to create counter")
	ErrFailedToCreateGauge     = errors.New("failed to create gauge")
	ErrFailedToCreateHistogram = errors.New("failed to create histogram")
)

// Attribute represents a key-value pair for metric attributes.
// This wraps OpenTelemetry's attribute.KeyValue to hide the dependency.
type Attribute = attribute.KeyValue

// AttributeBuilder provides methods to create metric attributes without exposing OpenTelemetry.
type AttributeBuilder struct{}

// NewAttributeBuilder creates a new attribute builder.
func NewAttributeBuilder() *AttributeBuilder {
	return &AttributeBuilder{}
}

// String creates a string attribute.
func (ab *AttributeBuilder) String(key, value string) Attribute {
	return attribute.String(key, value)
}

// Int creates an integer attribute.
func (ab *AttributeBuilder) Int(key string, value int) Attribute {
	return attribute.Int(key, value)
}

// Int64 creates an int64 attribute.
func (ab *AttributeBuilder) Int64(key string, value int64) Attribute {
	return attribute.Int64(key, value)
}

// Float64 creates a float64 attribute.
func (ab *AttributeBuilder) Float64(key string, value float64) Attribute {
	return attribute.Float64(key, value)
}

// Bool creates a boolean attribute.
func (ab *AttributeBuilder) Bool(key string, value bool) Attribute {
	return attribute.Bool(key, value)
}

// Convenience functions for creating attributes without needing the builder.
func StringAttr(key, value string) Attribute {
	return attribute.String(key, value)
}

func IntAttr(key string, value int) Attribute {
	return attribute.Int(key, value)
}

func Int64Attr(key string, value int64) Attribute {
	return attribute.Int64(key, value)
}

func Float64Attr(key string, value float64) Attribute {
	return attribute.Float64(key, value)
}

func BoolAttr(key string, value bool) Attribute {
	return attribute.Bool(key, value)
}

// MetricsBuilder provides a fluent interface for creating and managing metrics.
type MetricsBuilder struct {
	meter metric.Meter

	counters   map[string]metric.Int64Counter
	gauges     map[string]metric.Int64Gauge
	histograms map[string]metric.Float64Histogram
	name       string
}

// NewMetricsBuilder creates a new metrics builder with the given meter and name prefix.
func NewMetricsBuilder(provider *MetricsProvider, name, version string) *MetricsBuilder {
	meter := provider.GetMeterProvider().Meter(name, metric.WithInstrumentationVersion(version))

	return &MetricsBuilder{
		meter:      meter,
		name:       name,
		counters:   make(map[string]metric.Int64Counter),
		gauges:     make(map[string]metric.Int64Gauge),
		histograms: make(map[string]metric.Float64Histogram),
	}
}

// Counter creates a new counter metric.
func (mb *MetricsBuilder) Counter(name, description string) *CounterBuilder {
	return &CounterBuilder{
		builder:     mb,
		name:        name,
		description: description,
		unit:        "1", // default unit
	}
}

// Gauge creates a new gauge metric.
func (mb *MetricsBuilder) Gauge(name, description string) *GaugeBuilder {
	return &GaugeBuilder{
		builder:     mb,
		name:        name,
		description: description,
		unit:        "1", // default unit
	}
}

// Histogram creates a new histogram metric.
func (mb *MetricsBuilder) Histogram(name, description string) *HistogramBuilder {
	return &HistogramBuilder{
		builder:     mb,
		name:        name,
		description: description,
		unit:        "s", // default unit for duration
		buckets:     nil, // explicitly initialize
	}
}

// CounterBuilder provides a fluent interface for building counter metrics.
type CounterBuilder struct {
	builder     *MetricsBuilder
	name        string
	description string
	unit        string
}

// WithUnit sets the unit for the counter.
func (cb *CounterBuilder) WithUnit(unit string) *CounterBuilder {
	cb.unit = unit

	return cb
}

// Build creates the counter metric and returns a CounterMetric wrapper.
func (cb *CounterBuilder) Build() (*CounterMetric, error) {
	counter, err := cb.builder.meter.Int64Counter(
		cb.name,
		metric.WithDescription(cb.description),
		metric.WithUnit(cb.unit),
	)
	if err != nil {
		return nil, fmt.Errorf("%w (cb_name=%q): %w", ErrFailedToCreateCounter, cb.name, err)
	}

	cb.builder.counters[cb.name] = counter

	return &CounterMetric{counter: counter}, nil
}

// GaugeBuilder provides a fluent interface for building gauge metrics.
type GaugeBuilder struct {
	builder     *MetricsBuilder
	name        string
	description string
	unit        string
}

// WithUnit sets the unit for the gauge.
func (gb *GaugeBuilder) WithUnit(unit string) *GaugeBuilder {
	gb.unit = unit

	return gb
}

// Build creates the gauge metric and returns a GaugeMetric wrapper.
func (gb *GaugeBuilder) Build() (*GaugeMetric, error) {
	gauge, err := gb.builder.meter.Int64Gauge(
		gb.name,
		metric.WithDescription(gb.description),
		metric.WithUnit(gb.unit),
	)
	if err != nil {
		return nil, fmt.Errorf("%w (gb_name=%q): %w", ErrFailedToCreateGauge, gb.name, err)
	}

	gb.builder.gauges[gb.name] = gauge

	return &GaugeMetric{gauge: gauge}, nil
}

// HistogramBuilder provides a fluent interface for building histogram metrics.
type HistogramBuilder struct {
	builder     *MetricsBuilder
	name        string
	description string
	unit        string
	buckets     []float64
}

// WithUnit sets the unit for the histogram.
func (hb *HistogramBuilder) WithUnit(unit string) *HistogramBuilder {
	hb.unit = unit

	return hb
}

// WithBuckets sets custom bucket boundaries for the histogram.
func (hb *HistogramBuilder) WithBuckets(buckets ...float64) *HistogramBuilder {
	hb.buckets = buckets

	return hb
}

// WithDurationBuckets sets predefined duration buckets for the histogram.
func (hb *HistogramBuilder) WithDurationBuckets() *HistogramBuilder {
	hb.buckets = []float64{
		0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0,
	}

	return hb
}

// Build creates the histogram metric and returns a HistogramMetric wrapper.
func (hb *HistogramBuilder) Build() (*HistogramMetric, error) {
	opts := []metric.Float64HistogramOption{
		metric.WithDescription(hb.description),
		metric.WithUnit(hb.unit),
	}

	if len(hb.buckets) > 0 {
		opts = append(opts, metric.WithExplicitBucketBoundaries(hb.buckets...))
	}

	histogram, err := hb.builder.meter.Float64Histogram(hb.name, opts...)
	if err != nil {
		return nil, fmt.Errorf("%w (hb_name=%q): %w", ErrFailedToCreateHistogram, hb.name, err)
	}

	hb.builder.histograms[hb.name] = histogram

	return &HistogramMetric{histogram: histogram}, nil
}

// CounterMetric wraps a counter with convenient methods.
type CounterMetric struct {
	counter metric.Int64Counter
}

// Add increments the counter by the given value with optional attributes.
func (cm *CounterMetric) Add(ctx context.Context, value int64, attrs ...Attribute) {
	cm.counter.Add(ctx, value, metric.WithAttributes(attrs...))
}

// Inc increments the counter by 1 with optional attributes.
func (cm *CounterMetric) Inc(ctx context.Context, attrs ...Attribute) {
	cm.Add(ctx, 1, attrs...)
}

// GaugeMetric wraps a gauge with convenient methods.
type GaugeMetric struct {
	gauge metric.Int64Gauge
}

// Set sets the gauge value with optional attributes.
func (gm *GaugeMetric) Set(ctx context.Context, value int64, attrs ...Attribute) {
	gm.gauge.Record(ctx, value, metric.WithAttributes(attrs...))
}

// SetBool sets the gauge to 1 for true, 0 for false with optional attributes.
func (gm *GaugeMetric) SetBool(ctx context.Context, value bool, attrs ...Attribute) {
	var intValue int64
	if value {
		intValue = 1
	}

	gm.Set(ctx, intValue, attrs...)
}

// HistogramMetric wraps a histogram with convenient methods.
type HistogramMetric struct {
	histogram metric.Float64Histogram
}

// Record records a value in the histogram with optional attributes.
func (hm *HistogramMetric) Record(ctx context.Context, value float64, attrs ...Attribute) {
	hm.histogram.Record(ctx, value, metric.WithAttributes(attrs...))
}

// RecordDuration records a duration in seconds with optional attributes.
func (hm *HistogramMetric) RecordDuration(
	ctx context.Context,
	duration time.Duration,
	attrs ...Attribute,
) {
	hm.Record(ctx, duration.Seconds(), attrs...)
}

// Helper functions for common attribute patterns

// WorkerAttrs creates common worker attributes.
func WorkerAttrs(workerName string) []Attribute {
	return []Attribute{
		attribute.String("worker_name", workerName),
	}
}

// ErrorAttrs creates common error attributes.
func ErrorAttrs(err error) []Attribute {
	return []Attribute{
		attribute.String("error_type", fmt.Sprintf("%T", err)),
	}
}

// WorkerErrorAttrs combines worker and error attributes.
func WorkerErrorAttrs(workerName string, err error) []Attribute {
	return []Attribute{
		attribute.String("worker_name", workerName),
		attribute.String("error_type", fmt.Sprintf("%T", err)),
	}
}

// StatusAttrs creates status attributes.
func StatusAttrs(status string) []Attribute {
	return []Attribute{
		attribute.String("status", status),
	}
}

// TypeAttrs creates type attributes.
func TypeAttrs(typ string) []Attribute {
	return []Attribute{
		attribute.String("type", typ),
	}
}

// HTTPAttrs creates common HTTP request attributes.
func HTTPAttrs(method, endpoint, status string) []Attribute {
	return []Attribute{
		attribute.String("method", method),
		attribute.String("endpoint", endpoint),
		attribute.String("status", status),
	}
}

// HTTPMethodAttrs creates HTTP method attributes.
func HTTPMethodAttrs(method string) []Attribute {
	return []Attribute{
		attribute.String("method", method),
	}
}

// HTTPEndpointAttrs creates HTTP endpoint attributes.
func HTTPEndpointAttrs(endpoint string) []Attribute {
	return []Attribute{
		attribute.String("endpoint", endpoint),
	}
}

// GRPCAttrs creates common gRPC request attributes.
func GRPCAttrs(method, code string) []Attribute {
	return []Attribute{
		attribute.String("method", method),
		attribute.String("code", code),
	}
}

// GRPCMethodAttrs creates gRPC method attributes.
func GRPCMethodAttrs(method string) []Attribute {
	return []Attribute{
		attribute.String("method", method),
	}
}
