# ajan/metricsfx

## Overview

**metricsfx** package provides a comprehensive metrics solution built on OpenTelemetry SDK, designed for modern observability pipelines. The package emphasizes **OpenTelemetry collector integration** as the preferred approach, while maintaining backward compatibility with direct Prometheus exports.

### Key Features

- 🎯 **OpenTelemetry Collector Ready** - Native OTLP export to unified pipelines
- 📊 **Simplified Metrics Builder** - Intuitive API for creating metrics
- ⚡ **High Performance** - Optimized for production workloads
- 🔄 **Unified Observability** - Works seamlessly with logfx and tracesfx
- 🎛️ **Flexible Configuration** - Multiple export options and custom intervals

## Quick Start

### Basic Usage with OpenTelemetry Collector

```go
package main

import (
    "context"
    "time"
    
    "github.com/eser/ajan/metricsfx"
)

func main() {
    ctx := context.Background()
    
    // Configure for OpenTelemetry collector (recommended)
    config := &metricsfx.Config{
        ServiceName:    "my-service",
        ServiceVersion: "1.0.0",
        OTLPEndpoint:   "http://otel-collector:4318",
        OTLPInsecure:   true,
        ExportInterval: 30 * time.Second,
    }
    
    // Create provider with collector integration
    provider := metricsfx.NewMetricsProvider(config)
    if err := provider.Init(); err != nil {
        panic(err)
    }
    defer provider.Shutdown(ctx)
    
    // Create metrics using the builder
    builder := provider.NewBuilder()
    
    requestCounter, err := builder.Counter(
        "requests_total",
        "Total number of requests",
    ).WithUnit("{request}").Build()
    if err != nil {
        panic(err)
    }
    
    // Use metrics in your application
    requestCounter.Inc(ctx, metricsfx.StringAttr("endpoint", "/api/users"))
}
```

### HTTP Integration with Correlation

```go
package main

import (
    "context"
    "net/http"
    "time"
    
    "github.com/eser/ajan/httpfx"
    "github.com/eser/ajan/httpfx/middlewares"
    "github.com/eser/ajan/metricsfx"
)

func main() {
    ctx := context.Background()
    
    // Setup metrics with OpenTelemetry collector
    provider := metricsfx.NewMetricsProvider(&metricsfx.Config{
        ServiceName:    "api-service",
        ServiceVersion: "1.0.0", 
        OTLPEndpoint:   "http://otel-collector:4318",
        ExportInterval: 15 * time.Second,
    })
    _ = provider.Init()
    defer provider.Shutdown(ctx)
    
    // Create HTTP metrics
    httpMetrics, _ := metricsfx.NewHTTPMetrics(provider, "api-service", "1.0.0")
    
    // Setup HTTP router with observability middleware
    router := httpfx.NewRouter("/api")
    router.Use(middlewares.CorrelationIDMiddleware())        // Request correlation
    router.Use(middlewares.MetricsMiddleware(httpMetrics))   // Automatic metrics
    
    router.Route("GET /users/{id}", func(ctx *httpfx.Context) httpfx.Result {
        // Custom business metrics
        httpMetrics.RequestsTotal.Inc(ctx.Request.Context(),
            metricsfx.StringAttr("operation", "get_user"),
            metricsfx.StringAttr("user_type", "premium"),
        )
        
        return ctx.Results.JSON(map[string]string{"status": "success"})
    })
    
    http.ListenAndServe(":8080", router.GetMux())
}
```

## Configuration

```go
type Config struct {
	// Service information for resource attribution
	ServiceName    string        `conf:"service_name"    default:""`
	ServiceVersion string        `conf:"service_version" default:""`

	// OpenTelemetry Collector configuration (preferred)
	OTLPEndpoint string        `conf:"otlp_endpoint" default:""`
	OTLPInsecure bool          `conf:"otlp_insecure" default:"true"`

	// Export interval for batching
	ExportInterval time.Duration `conf:"export_interval" default:"30s"`

	// Legacy direct exporters (still supported)
	PrometheusEndpoint string `conf:"prometheus_endpoint" default:""`
}
```

### Export Priority

The package automatically chooses the best export method:

1. **🥇 OTLP Collector** (`OTLPEndpoint`) - Preferred for production
2. **🥈 Direct Prometheus** (`PrometheusEndpoint`) - Legacy/fallback option
3. **Both can run simultaneously** if needed

## OpenTelemetry Collector Integration

### Why Use OpenTelemetry Collector?

The collector provides a unified observability pipeline that offers significant advantages over direct exports:

```
┌─────────────────┐    ┌─────────────────────┐    ┌─────────────────┐
│                 │    │                     │    │                 │
│   Your App      │───▶│ OpenTelemetry      │───▶│   Backends      │
│                 │    │ Collector           │    │                 │
│ • metricsfx     │    │                     │    │ • Prometheus    │
│ • logfx         │    │ • Receives all 3    │    │ • Grafana       │
│ • tracesfx      │    │   pillars (L+M+T)   │    │ • DataDog       │
│                 │    │ • Routes & filters  │    │ • New Relic     │
│                 │    │ • Transforms        │    │ • Custom        │
└─────────────────┘    └─────────────────────┘    └─────────────────┘
```

**Benefits:**
- **🔄 Unified Pipeline** - All observability data flows through one point
- **🎛️ Flexibility** - Change backends without code changes
- **⚡ Performance** - Built-in batching, compression, and retries
- **💰 Cost Optimization** - Sampling and filtering before expensive exports
- **🔧 Data Processing** - Transform, enrich, and route telemetry data

### Recommended Configuration

```go
// Production setup
config := &metricsfx.Config{
    ServiceName:    "my-service",
    ServiceVersion: "1.2.3",
    OTLPEndpoint:   "http://otel-collector:4318",
    OTLPInsecure:   false,  // Use TLS in production
    ExportInterval: 30 * time.Second,
}

// Development setup  
devConfig := &metricsfx.Config{
    ServiceName:    "my-service",
    ServiceVersion: "dev",
    OTLPEndpoint:   "http://localhost:4318",
    OTLPInsecure:   true,
    ExportInterval: 5 * time.Second,  // Faster for development
}
```

### Example Collector Configuration

```yaml
# otel-collector-config.yaml
receivers:
  otlp:
    protocols:
      http:
        endpoint: 0.0.0.0:4318
      grpc:
        endpoint: 0.0.0.0:4317

processors:
  batch:
    timeout: 10s
    send_batch_size: 1024
  
  resource:
    attributes:
      - key: environment
        value: production
        action: upsert

exporters:
  prometheus:
    endpoint: "0.0.0.0:8889"
  
  # Add other exporters as needed
  datadog:
    api:
      key: ${DD_API_KEY}

service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [batch, resource]
      exporters: [prometheus, datadog]
```

## Metrics Builder API

The simplified metrics builder provides an intuitive interface for creating metrics:

### Counter Metrics

```go
builder := provider.NewBuilder()

// Simple counter
requestCounter, err := builder.Counter(
    "requests_total",
    "Total number of HTTP requests",
).Build()

// Counter with unit
bytesCounter, err := builder.Counter(
    "bytes_processed_total", 
    "Total bytes processed",
).WithUnit("byte").Build()

// Usage
requestCounter.Inc(ctx, 
    metricsfx.StringAttr("method", "GET"),
    metricsfx.StringAttr("status", "200"),
)
```

### Histogram Metrics

```go
// Request duration with custom buckets
requestDuration, err := builder.Histogram(
    "request_duration_seconds",
    "HTTP request processing time",
).WithDurationBuckets().Build()

// Custom buckets
responseSizeHist, err := builder.Histogram(
    "response_size_bytes",
    "HTTP response sizes",
).WithBuckets([]float64{100, 1024, 10240, 102400}).Build()

// Usage
requestDuration.RecordDuration(ctx, duration, 
    metricsfx.StringAttr("endpoint", "/api/users"),
)
```

### Gauge Metrics

```go
// Current active connections
activeConnections, err := builder.Gauge(
    "active_connections",
    "Number of active connections", 
).Build()

// Memory usage
memoryUsage, err := builder.Gauge(
    "memory_usage_bytes",
    "Current memory usage",
).WithUnit("byte").Build()

// Usage
activeConnections.Set(ctx, 42,
    metricsfx.StringAttr("pool", "database"),
)
```

## HTTP Metrics Integration

### Automatic HTTP Metrics

```go
// Create HTTP metrics
httpMetrics, err := metricsfx.NewHTTPMetrics(provider, "web-service", "1.0.0")
if err != nil {
    panic(err)
}

// Add to router - automatically tracks all requests
router.Use(middlewares.MetricsMiddleware(httpMetrics))
```

**Automatically tracked metrics:**
- `http_requests_total` - Counter of HTTP requests by method, path, status code
- `http_request_duration_seconds` - Histogram of request durations

### Custom HTTP Metrics

```go
router.Route("POST /users", func(ctx *httpfx.Context) httpfx.Result {
    // Track business-specific metrics
    httpMetrics.RequestsTotal.Inc(ctx.Request.Context(),
        metricsfx.StringAttr("operation", "create_user"),
        metricsfx.StringAttr("user_type", "premium"),
        metricsfx.StringAttr("plan", "enterprise"),
    )
    
    // Track custom durations
    start := time.Now()
    // ... business logic ...
    businessDuration := time.Since(start)
    
    businessTimer.RecordDuration(ctx.Request.Context(), businessDuration,
        metricsfx.StringAttr("operation", "user_creation"),
    )
    
    return ctx.Results.JSON(response)
})
```

## Advanced Usage

### Multiple Export Destinations

```go
config := &metricsfx.Config{
    ServiceName:        "my-service",
    OTLPEndpoint:       "http://otel-collector:4318",  // Primary
    PrometheusEndpoint: "/metrics",                     // Direct fallback
    ExportInterval:     30 * time.Second,
}
```

### Resource Attribution

```go
// Metrics automatically include resource information
config := &metricsfx.Config{
    ServiceName:    "user-service",      // service.name
    ServiceVersion: "2.1.0",             // service.version  
    OTLPEndpoint:   "http://collector:4318",
}

// All metrics will include these resource attributes automatically
```

### Correlation with Logs and Traces

When used with the complete `ajan` observability stack:

```go
// All using the same collector endpoint
observabilityConfig := struct {
    ServiceName  string
    ServiceVersion string  
    OTLPEndpoint string
}{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0", 
    OTLPEndpoint:   "http://otel-collector:4318",
}

// Metrics
metricsProvider := metricsfx.NewMetricsProvider(&metricsfx.Config{
    ServiceName:    observabilityConfig.ServiceName,
    ServiceVersion: observabilityConfig.ServiceVersion,
    OTLPEndpoint:   observabilityConfig.OTLPEndpoint,
})
_ = metricsProvider.Init()

// Logs (with automatic correlation)
logger := logfx.NewLogger(os.Stdout, &logfx.Config{
    Level:        "INFO",
    OTLPEndpoint: observabilityConfig.OTLPEndpoint,
})

// All telemetry data flows to the same collector with:
// - Consistent resource attribution (service name/version)
// - Automatic correlation via trace context
// - HTTP correlation IDs in logs
// - Unified export pipeline
```

## Error Handling

The metrics provider handles export failures gracefully:

```go
provider := metricsfx.NewMetricsProvider(config)
if err := provider.Init(); err != nil {
    log.Printf("Failed to create metrics provider: %v", err)
    // Fallback to basic provider or handle appropriately
}

// Metrics continue to work even if exports fail
// Failures are logged but don't affect application performance
```

## Migration from Direct Exports

### Before (Direct Prometheus)
```go
// Old approach - direct to Prometheus
config := &metricsfx.Config{
    PrometheusEndpoint: "/metrics",
}
```

### After (OpenTelemetry Collector)
```go
// New approach - unified pipeline  
config := &metricsfx.Config{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
    OTLPEndpoint:   "http://otel-collector:4318",
}
```

The collector configuration handles routing to Prometheus and other backends without any application code changes!

## Performance Characteristics

- **Low Overhead** - Optimized for production workloads
- **Async Exports** - Non-blocking metric collection
- **Batching** - Efficient data transmission
- **Memory Efficient** - Bounded resource usage
- **High Throughput** - Suitable for high-traffic applications

## Best Practices

1. **Use OpenTelemetry Collector** for production deployments
2. **Set appropriate export intervals** (15-60 seconds typically)
3. **Include resource attribution** (service name/version)
4. **Use consistent labeling** across metrics
5. **Monitor export health** via collector metrics
6. **Combine with logfx** for complete observability

---

## Legacy Interface Reference

The package maintains backward compatibility with the original interface:

### Types Overview

The package defines several key types for metrics collection:

```go
type MetricsProvider struct {
    // Internal implementation
}

type MetricsBuilder struct {
    // Builder for creating metrics
}

// Metric types
type Counter interface { Inc(ctx context.Context, attrs ...attribute.KeyValue) }
type Histogram interface { Record(ctx context.Context, value float64, attrs ...attribute.KeyValue) }
type Gauge interface { Set(ctx context.Context, value float64, attrs ...attribute.KeyValue) }
```

### Provider Creation

```go
provider := metricsfx.NewMetricsProvider(config)

err := provider.Init()
if err != nil {
    log.Fatal("Failed to register metrics collectors:", err)
}
```

The documentation below provides additional details about the package types,
functions, and usage examples. For more detailed information, refer to the
source code and tests.
