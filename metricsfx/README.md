# ajan/metricsfx

## Overview

The **metricsfx** package provides metrics collection and monitoring utilities
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

## API

### MetricsProvider

The main interface for metrics functionality:

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

// Create custom metrics
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
