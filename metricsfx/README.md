# ajan/metricsfx

## Overview

**metricsfx** package provides metrics collection and monitoring utilities
using OpenTelemetry. It integrates with other components to provide metrics for
HTTP services, gRPC services, and native Go runtime metrics.

## Features

- OpenTelemetry metrics integration
- Native Go runtime metrics collection (memory, GC, goroutines)
- HTTP metrics collection
- gRPC metrics collection
- Event metrics collection
- Multiple exporter support (OTLP, Prometheus, StatsD, etc.)
- Integration with dependency injection
- Vendor-neutral observability
- **🆕 Simplified Metrics Builder** - Fluent interface for creating metrics with minimal boilerplate

## Quick Start with MetricsBuilder

The **MetricsBuilder** provides a clean, fluent interface for creating metrics without boilerplate:

### Simple Example

```go
// Create a metrics provider
provider := metricsfx.NewMetricsProvider()
defer provider.Shutdown(context.Background())

// Create a metrics builder
builder := metricsfx.NewMetricsBuilder(provider, "my-service", "1.0.0")

// Create metrics with fluent interface
counter, err := builder.Counter(
    "requests_total",
    "Total number of requests",
).WithUnit("requests").Build()

gauge, err := builder.Gauge(
    "connections_active",
    "Number of active connections",
).Build()

histogram, err := builder.Histogram(
    "request_duration_seconds",
    "Request processing time",
).WithDurationBuckets().Build()

// Use metrics easily
ctx := context.Background()
counter.Inc(ctx, metricsfx.WorkerAttrs("worker-1")...)
gauge.Set(ctx, 42)
histogram.RecordDuration(ctx, 250*time.Millisecond)
```

### Complex Example: Worker Metrics

Before (lots of boilerplate):
```go
// 200+ lines of initialization code...
meter := otel.Meter("bfo.workers", metric.WithInstrumentationVersion("1.0.0"))

workerTicksTotal, err := meter.Int64Counter(
    "bfo_worker_ticks_total",
    metric.WithDescription("Total number of worker ticks executed"),
    metric.WithUnit("1"),
)
// ... repeat for 8+ more metrics
```

After (clean and simple):
```go
// Create worker metrics with the builder
func NewWorkerMetrics(provider *metricsfx.MetricsProvider, logger *logfx.Logger) (*WorkerMetrics, error) {
    builder := metricsfx.NewMetricsBuilder(provider, "bfo.workers", "1.0.0")

    ticksTotal, err := builder.Counter(
        "bfo_worker_ticks_total",
        "Total number of worker ticks executed",
    ).Build()
    if err != nil {
        return nil, err
    }

    processingTime, err := builder.Histogram(
        "bfo_worker_processing_duration_seconds",
        "Time spent processing worker ticks",
    ).WithDurationBuckets().Build()
    if err != nil {
        return nil, err
    }

    // ... much cleaner metric creation

    return &WorkerMetrics{
        ticksTotal:     ticksTotal,
        processingTime: processingTime,
        // ...
    }, nil
}

// Clean usage with helper functions
func (m *WorkerMetrics) RecordWorkerTick(ctx context.Context, workerName string, duration time.Duration) {
    attrs := metricsfx.WorkerAttrs(workerName)

    m.ticksTotal.Inc(ctx, attrs...)
    m.processingTime.RecordDuration(ctx, duration, attrs...)
}
```

## MetricsBuilder API

### Creating Metrics

```go
builder := metricsfx.NewMetricsBuilder(provider, "service-name", "version")

// Counters
counter, err := builder.Counter("metric_name", "description").
    WithUnit("requests").
    Build()

// Gauges
gauge, err := builder.Gauge("metric_name", "description").
    WithUnit("connections").
    Build()

// Histograms
histogram, err := builder.Histogram("metric_name", "description").
    WithDurationBuckets().              // Predefined duration buckets
    Build()

// Custom histogram buckets
histogram, err := builder.Histogram("metric_name", "description").
    WithBuckets(0.1, 0.5, 1.0, 2.0, 5.0).
    Build()
```

### Using Metrics

```go
ctx := context.Background()

// Counters
counter.Inc(ctx)                              // Increment by 1
counter.Add(ctx, 5)                          // Add 5
counter.Inc(ctx, attribute.String("key", "value"))  // With attributes

// Gauges
gauge.Set(ctx, 42)                           // Set value
gauge.SetBool(ctx, true)                     // Set boolean (1/0)

// Histograms
histogram.Record(ctx, 1.5)                   // Record value
histogram.RecordDuration(ctx, 250*time.Millisecond)  // Record duration
```

### Attribute Helpers

Pre-built attribute helpers for common patterns:

```go
// Worker attributes
attrs := metricsfx.WorkerAttrs("worker-1")
// → [worker_name="worker-1"]

// Error attributes
attrs := metricsfx.ErrorAttrs(err)
// → [error_type="*errors.errorString"]

// Combined worker + error
attrs := metricsfx.WorkerErrorAttrs("worker-1", err)
// → [worker_name="worker-1", error_type="*errors.errorString"]

// Status attributes
attrs := metricsfx.StatusAttrs("active")
// → [status="active"]

// Type attributes
attrs := metricsfx.TypeAttrs("batch_processing")
// → [type="batch_processing"]
```

## Traditional API (Still Available)

### MetricsProvider

The traditional lower-level interface is still available:

```go
// Create a new metrics provider
metricsProvider := metricsfx.NewMetricsProvider()

// Register native Go runtime collectors
err := metricsProvider.RegisterNativeCollectors()
if err != nil {
    log.Fatal(err)
}

// Get the meter provider for creating custom metrics
meterProvider := metricsProvider.GetMeterProvider()
meter := meterProvider.Meter("my-service")

// Create custom metrics manually
counter, err := meter.Int64Counter(
    "my_counter",
    metric.WithDescription("Example counter"),
    metric.WithUnit("{request}"),
)
if err != nil {
    log.Fatal(err)
}

// Use the counter with attributes
ctx := context.Background()
counter.Add(ctx, 1, metric.WithAttributes(
    attribute.String("method", "GET"),
    attribute.String("status", "200"),
))
```

### HTTP Integration

```go
// Create HTTP metrics
httpMetrics := httpfx.NewMetrics(metricsProvider)

// Add metrics middleware to your HTTP router
router.Use(middlewares.MetricsMiddleware(httpMetrics))
```

### gRPC Integration

```go
// Create gRPC metrics
grpcMetrics := grpcfx.NewMetrics(metricsProvider)

// Add metrics interceptor to your gRPC server
grpcServer := grpc.NewServer(
    grpc.UnaryInterceptor(grpcfx.MetricsInterceptor(grpcMetrics)),
)
```

### Event Integration

```go
// Create event metrics
eventMetrics := eventsfx.NewMetrics(metricsProvider)

// Metrics are automatically recorded when events are dispatched
```

## Runtime Metrics

The package automatically collects Go runtime metrics:

- **Memory**: `go.memory.used`, `go.memory.limit`, `go.memory.allocated`
- **Garbage Collection**: `go.memory.gc.goal`
- **Goroutines**: `go.goroutine.count`
- **Processor**: `go.processor.limit`
- **Scheduler**: `go.schedule.duration`

## Exporters

Configure exporters to send metrics to your monitoring backend:

### OTLP Exporter (Recommended)

```go
import "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"

exporter, err := otlpmetrichttp.New(ctx,
    otlpmetrichttp.WithEndpoint("http://your-otel-collector:4318"),
)
```

### Prometheus Exporter

```go
import "go.opentelemetry.io/otel/exporters/prometheus"

exporter, err := prometheus.New()
```

## Shutdown

Always properly shutdown the metrics provider:

```go
defer func() {
    if err := metricsProvider.Shutdown(context.Background()); err != nil {
        log.Printf("Error shutting down metrics: %v", err)
    }
}()
```

## Benefits of MetricsBuilder

- **90% less boilerplate** - No more repetitive OpenTelemetry setup code
- **Type-safe** - Compile-time guarantees for metric usage
- **Fluent interface** - Chain configuration methods for readability
- **Best practices** - Follows OpenTelemetry conventions automatically
- **Helper functions** - Common attribute patterns pre-built
- **Easy testing** - Simple mocking and testing patterns
- **Backward compatible** - Traditional API still available
