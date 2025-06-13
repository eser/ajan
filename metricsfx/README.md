# ajan/metricsfx

## Overview

**metricsfx** provides a comprehensive metrics solution built on OpenTelemetry with **centralized OTLP connection management** through `connfx`. It offers a simplified metrics builder API, automatic runtime metrics collection, and native integration with HTTP services and other observability packages for complete telemetry coverage.

### Key Features

- 🎯 **Simplified Metrics Builder** - Fluent API for creating counters, gauges, and histograms
- 📊 **Runtime Metrics** - Automatic Go runtime metrics (memory, GC, goroutines)
- 🔄 **HTTP Metrics** - Built-in request/response metrics for HTTP services
- 🌐 **Centralized OTLP Integration** - Uses `connfx` registry for shared OTLP connections
- ⚡ **Performance Optimized** - Efficient periodic exports with configurable intervals
- 🎛️ **Flexible Configuration** - Environment-based configuration with sensible defaults
- 🔗 **Service Integration** - Seamless integration with `httpfx`, `grpcfx`, and other packages

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "time"

    "github.com/eser/ajan/connfx"
    "github.com/eser/ajan/logfx"
    "github.com/eser/ajan/metricsfx"
)

func main() {
    ctx := context.Background()

    // Create logger and connection registry
    logger := logfx.NewLogger()
    registry := connfx.NewRegistryWithDefaults(logger)

    // Configure OTLP connection once, use everywhere
    _, err := registry.AddConnection(ctx, "otel", &connfx.ConfigTarget{
        Protocol: "otlp",
        DSN:      "otel-collector:4318",
        Properties: map[string]any{
            "service_name":    "my-service",
            "service_version": "1.0.0",
            "insecure":        true,
        },
    })
    if err != nil {
        panic(err)
    }

    // Create metrics provider with connection registry
    provider := metricsfx.NewMetricsProvider(&metricsfx.Config{
        ServiceName:        "my-service",
        ServiceVersion:     "1.0.0",
        OTLPConnectionName: "otel", // Reference the connection
        ExportInterval:     30 * time.Second,
    }, registry) // Pass the registry

    // Initialize provider (enables OTLP export and runtime metrics)
    err = provider.Init()
    if err != nil {
        panic(err)
    }
    defer provider.Shutdown(ctx)

    // Create custom application metrics
    builder := provider.NewBuilder()

    // Counter - tracks total events
    requestCounter, _ := builder.Counter(
        "http_requests_total",
        "Total HTTP requests processed",
    ).WithUnit("{request}").Build()

    // Gauge - tracks current state
    activeConnections, _ := builder.Gauge(
        "active_connections",
        "Current number of active connections",
    ).WithUnit("{connection}").Build()

    // Histogram - tracks distributions
    requestDuration, _ := builder.Histogram(
        "http_request_duration_seconds",
        "HTTP request processing duration",
    ).WithDurationBuckets().Build()

    // Use metrics in your application
    requestCounter.Inc(ctx, metricsfx.StringAttr("method", "GET"))
    activeConnections.Set(ctx, 42)
    requestDuration.Record(ctx, 0.123, metricsfx.StringAttr("status", "200"))
}
```

### Complete Observability Stack Integration

```go
package main

import (
    "context"
    "net/http"
    "time"

    "github.com/eser/ajan/connfx"
    "github.com/eser/ajan/httpfx"
    "github.com/eser/ajan/httpfx/middlewares"
    "github.com/eser/ajan/logfx"
    "github.com/eser/ajan/metricsfx"
    "github.com/eser/ajan/tracesfx"
)

func main() {
    ctx := context.Background()

    // Step 1: Create shared OTLP connection
    logger := logfx.NewLogger()
    registry := connfx.NewRegistryWithDefaults(logger)

    _, err := registry.AddConnection(ctx, "otel", &connfx.ConfigTarget{
        Protocol: "otlp",
        DSN:      "otel-collector:4318",
        Properties: map[string]any{
            "service_name":     "my-api",
            "service_version":  "1.0.0",
            "insecure":         true,
            "export_interval":  "15s",
            "batch_timeout":    "5s",
        },
    })
    if err != nil {
        panic(err)
    }

    // Step 2: Create observability stack using shared connection

    // Metrics with automatic runtime collection
    metricsProvider := metricsfx.NewMetricsProvider(&metricsfx.Config{
        ServiceName:        "my-api",
        ServiceVersion:     "1.0.0",
        OTLPConnectionName: "otel",
        ExportInterval:     15 * time.Second,
        NoNativeCollectorRegistration: false, // Enable runtime metrics
    }, registry)
    _ = metricsProvider.Init()

    // Traces for request tracing
    tracesProvider := tracesfx.NewTracesProvider(&tracesfx.Config{
        ServiceName:        "my-api",
        ServiceVersion:     "1.0.0",
        OTLPConnectionName: "otel",
        SampleRatio:        1.0,
    }, registry)
    _ = tracesProvider.Init()

    // Logs with correlation
    logger = logfx.NewLogger(
        logfx.WithConfig(&logfx.Config{
            Level:              "INFO",
            OTLPConnectionName: "otel",
        }),
        logfx.WithRegistry(registry),
    )

    // Step 3: Setup HTTP service with metrics middleware
    router := httpfx.NewRouter("/api")

    // Create HTTP metrics using the provider
    httpMetrics := metricsfx.NewHTTPMetrics(metricsProvider)
    err = httpMetrics.Init()
    if err != nil {
        panic(err)
    }

    // Add observability middleware
    router.Use(middlewares.CorrelationIDMiddleware())
    router.Use(middlewares.LoggingMiddleware(logger))
    router.Use(middlewares.MetricsMiddleware(httpMetrics))

    // Create custom business metrics
    builder := metricsProvider.NewBuilder()
    userOperations, _ := builder.Counter(
        "user_operations_total",
        "Total user operations performed",
    ).WithUnit("{operation}").Build()

    router.Route("GET /users/{id}", func(ctx *httpfx.Context) httpfx.Result {
        // Business metrics automatically correlated with HTTP metrics
        userOperations.Inc(ctx.Request.Context(),
            metricsfx.StringAttr("operation", "lookup"),
            metricsfx.StringAttr("user_id", "123"),
        )

        return ctx.Results.JSON(map[string]string{"status": "success"})
    })

    http.ListenAndServe(":8080", router.GetMux())
}
```

**Metrics Output (automatically exported to OTLP):**
```json
{
  "resourceMetrics": [
    {
      "resource": {
        "attributes": [
          {"key": "service.name", "value": {"stringValue": "my-api"}},
          {"key": "service.version", "value": {"stringValue": "1.0.0"}}
        ]
      },
      "scopeMetrics": [
        {
          "metrics": [
            {
              "name": "http_requests_total",
              "unit": "{request}",
              "sum": {"dataPoints": [{"asInt": "42", "attributes": [{"key": "method", "value": {"stringValue": "GET"}}]}]}
            },
            {
              "name": "go_memstats_heap_objects",
              "unit": "{object}",
              "gauge": {"dataPoints": [{"asInt": "1234567"}]}
            },
            {
              "name": "http_request_duration_seconds",
              "unit": "s",
              "histogram": {"dataPoints": [{"count": "1", "sum": 0.123, "bucketCounts": ["0", "1", "0"]}]}
            }
          ]
        }
      ]
    }
  ]
}
```

## Configuration

```go
type Config struct {
    // Service identification (automatically applied to all metrics)
    ServiceName    string `conf:"service_name"    default:""`
    ServiceVersion string `conf:"service_version" default:""`

    // Connection-based OTLP configuration (replaces direct endpoint config)
    OTLPConnectionName string `conf:"otlp_connection_name" default:""`

    // Export configuration
    ExportInterval time.Duration `conf:"export_interval" default:"30s"`

    // Runtime metrics collection
    NoNativeCollectorRegistration bool `conf:"no_native_collector_registration" default:"false"`
}
```

## Centralized Connection Management

### Why Use connfx for OTLP Connections?

The new architecture centralizes OTLP connection management through `connfx`, providing significant advantages:

**Before (Old Architecture):**
```go
// Each package configured separately - duplicated configuration
metrics := metricsfx.NewMetricsProvider(&metricsfx.Config{
    OTLPEndpoint: "otel-collector:4318",
    ServiceName:  "my-service",
})
traces := tracesfx.NewTracesProvider(&tracesfx.Config{
    OTLPEndpoint: "otel-collector:4318",
    ServiceName:  "my-service",
})
```

**After (New Architecture):**
```go
// Single OTLP connection shared across all packages
registry.AddConnection(ctx, "otel", &connfx.ConfigTarget{
    Protocol: "otlp",
    DSN:      "otel-collector:4318",
    Properties: map[string]any{
        "service_name": "my-service",
        "service_version": "1.0.0",
    },
})

// All packages reference the same connection
metrics := metricsfx.NewMetricsProvider(&metricsfx.Config{OTLPConnectionName: "otel"}, registry)
traces := tracesfx.NewTracesProvider(&tracesfx.Config{OTLPConnectionName: "otel"}, registry)
```

**Benefits:**
- 🔧 **Single Configuration Point** - Configure OTLP once, use everywhere
- 🔄 **Shared Connections** - Efficient resource usage and connection pooling
- 🎛️ **Centralized Management** - Health checks, monitoring, and lifecycle management
- 🔗 **Consistent Attribution** - Service name/version automatically applied to all metrics
- 💰 **Cost Optimization** - Single connection reduces overhead
- 🛡️ **Error Handling** - Graceful fallbacks when connections are unavailable

### OTLP Connection Configuration

```go
// Configure OTLP connection with metrics-specific options
otlpConfig := &connfx.ConfigTarget{
    Protocol: "otlp",
    DSN:      "otel-collector:4318",
    Properties: map[string]any{
        // Service identification (applied to all metrics automatically)
        "service_name":    "my-service",
        "service_version": "1.0.0",

        // Connection settings
        "insecure":        true,                    // Use HTTP instead of HTTPS

        // Metrics-specific export configuration
        "export_interval": 30 * time.Second,       // How often to export metrics
        "batch_timeout":   5 * time.Second,        // Maximum time to wait for batch
        "batch_size":      512,                    // Maximum batch size
    },
}

_, err := registry.AddConnection(ctx, "otel", otlpConfig)
```

### Environment-Based Configuration

```bash
# Connection configuration via environment
CONN_TARGETS_OTEL_PROTOCOL=otlp
CONN_TARGETS_OTEL_DSN=otel-collector:4318
CONN_TARGETS_OTEL_PROPERTIES_SERVICE_NAME=my-service
CONN_TARGETS_OTEL_PROPERTIES_SERVICE_VERSION=1.0.0
CONN_TARGETS_OTEL_PROPERTIES_EXPORT_INTERVAL=30s

# Package configuration references the connection
METRICS_SERVICE_NAME=my-service
METRICS_SERVICE_VERSION=1.0.0
METRICS_OTLP_CONNECTION_NAME=otel
METRICS_EXPORT_INTERVAL=30s
```

### Multiple OTLP Endpoints

```go
// Different endpoints for different metric types
_, err := registry.AddConnection(ctx, "otel-business", &connfx.ConfigTarget{
    Protocol: "otlp",
    URL:      "http://business-metrics-collector:4318",
    Properties: map[string]any{
        "service_name":    "my-service",
        "export_interval": 60 * time.Second,  // Less frequent export for business metrics
    },
})

_, err = registry.AddConnection(ctx, "otel-system", &connfx.ConfigTarget{
    Protocol: "otlp",
    URL:      "http://system-metrics-collector:4318",
    Properties: map[string]any{
        "service_name":    "my-service",
        "export_interval": 15 * time.Second,  // More frequent export for system metrics
    },
})

// Use different connections for different metric types
businessMetrics := metricsfx.NewMetricsProvider(&metricsfx.Config{
    OTLPConnectionName: "otel-business",
}, registry)

systemMetrics := metricsfx.NewMetricsProvider(&metricsfx.Config{
    OTLPConnectionName: "otel-system",
}, registry)
```

## Metrics Builder API

### Counter Metrics

Track cumulative values that only increase:

```go
builder := provider.NewBuilder()

// Basic counter
requests, _ := builder.Counter(
    "http_requests_total",
    "Total HTTP requests processed",
).Build()

// Counter with custom unit
bytes, _ := builder.Counter(
    "data_processed_bytes",
    "Total bytes processed",
).WithUnit("By").Build()

// Usage
ctx := context.Background()
requests.Inc(ctx, metricsfx.StringAttr("method", "GET"))
requests.Add(ctx, 5, metricsfx.StringAttr("method", "POST"))
bytes.Add(ctx, 1024, metricsfx.StringAttr("type", "upload"))
```

### Gauge Metrics

Track current values that can go up and down:

```go
// Basic gauge
connections, _ := builder.Gauge(
    "active_connections",
    "Number of active connections",
).Build()

// Gauge with custom unit
temperature, _ := builder.Gauge(
    "cpu_temperature_celsius",
    "CPU temperature",
).WithUnit("Cel").Build()

// Usage
connections.Set(ctx, 42, metricsfx.StringAttr("pool", "main"))
connections.Add(ctx, 5, metricsfx.StringAttr("pool", "cache"))
temperature.Set(ctx, 65.5)

// Boolean convenience method
healthy, _ := builder.Gauge("system_healthy", "System health status").Build()
healthy.SetBool(ctx, true)
```

### Histogram Metrics

Track distributions of values:

```go
// Duration histogram with pre-configured buckets
duration, _ := builder.Histogram(
    "http_request_duration_seconds",
    "HTTP request processing duration",
).WithDurationBuckets().Build()

// Custom histogram buckets
responseSize, _ := builder.Histogram(
    "http_response_size_bytes",
    "HTTP response size distribution",
).WithBuckets([]float64{100, 1000, 10000, 100000}).Build()

// Usage
duration.Record(ctx, 0.123, metricsfx.StringAttr("method", "GET"))
responseSize.Record(ctx, 2048, metricsfx.StringAttr("endpoint", "/api/users"))

// Convenience method for duration tracking
start := time.Now()
// ... do work ...
duration.RecordDuration(ctx, time.Since(start), metricsfx.StringAttr("operation", "db_query"))
```

### Attributes

Add dimensions to your metrics:

```go
// String attributes
metricsfx.StringAttr("method", "GET")
metricsfx.StringAttr("status", "success")

// Numeric attributes
metricsfx.IntAttr("status_code", 200)
metricsfx.Float64Attr("cpu_usage", 0.85)

// Boolean attributes
metricsfx.BoolAttr("cache_hit", true)

// Multiple attributes
counter.Inc(ctx,
    metricsfx.StringAttr("method", "POST"),
    metricsfx.IntAttr("status_code", 201),
    metricsfx.StringAttr("endpoint", "/api/users"),
)
```

## HTTP Metrics Integration

### Automatic HTTP Metrics

Create standardized HTTP metrics for your services:

```go
// Create HTTP metrics using the provider
httpMetrics := metricsfx.NewHTTPMetrics(metricsProvider)
err := httpMetrics.Init()
if err != nil {
    panic(err)
}

// Use with httpfx middleware
router := httpfx.NewRouter("/api")
router.Use(middlewares.MetricsMiddleware(httpMetrics))

// Automatically tracks:
// - http_requests_total (counter)
// - http_request_duration_seconds (histogram)
// - http_response_size_bytes (histogram)
// With attributes: method, status_code, endpoint
```

### Custom HTTP Metrics

```go
// Create custom HTTP-related metrics
builder := provider.NewBuilder()

// Track API endpoints separately
apiCalls, _ := builder.Counter(
    "api_calls_total",
    "Total API calls by endpoint",
).Build()

authFailures, _ := builder.Counter(
    "auth_failures_total",
    "Authentication failures",
).Build()

// Use in handlers
router.Route("GET /users/{id}", func(ctx *httpfx.Context) httpfx.Result {
    apiCalls.Inc(ctx.Request.Context(),
        metricsfx.StringAttr("endpoint", "get_user"),
        metricsfx.StringAttr("version", "v1"),
    )

    // ... handler logic ...

    return ctx.Results.JSON(user)
})

router.Route("POST /auth/login", func(ctx *httpfx.Context) httpfx.Result {
    if !authenticate(ctx) {
        authFailures.Inc(ctx.Request.Context(),
            metricsfx.StringAttr("reason", "invalid_credentials"),
        )
        return ctx.Results.Unauthorized()
    }

    return ctx.Results.JSON(token)
})
```

## Runtime Metrics

### Automatic Go Runtime Metrics

Enable automatic collection of Go runtime metrics:

```go
provider := metricsfx.NewMetricsProvider(&metricsfx.Config{
    ServiceName:                   "my-service",
    OTLPConnectionName:            "otel",
    NoNativeCollectorRegistration: false, // Enable runtime metrics (default)
}, registry)

_ = provider.Init()

// Automatically collects:
// - go_memstats_* (memory statistics)
// - go_gc_* (garbage collection metrics)
// - go_goroutines (goroutine count)
// - go_threads (thread count)
// - runtime_* (runtime performance metrics)
```

**Available Runtime Metrics:**
- `go_memstats_alloc_bytes` - Currently allocated bytes
- `go_memstats_heap_objects` - Number of heap objects
- `go_memstats_gc_count_total` - Total number of GC cycles
- `go_memstats_gc_duration_seconds` - GC pause duration
- `go_goroutines` - Current number of goroutines
- `go_threads` - Current number of OS threads
- `runtime_uptime_milliseconds` - Process uptime

### Disable Runtime Metrics

```go
// Disable runtime metrics for lightweight deployments
provider := metricsfx.NewMetricsProvider(&metricsfx.Config{
    ServiceName:                   "my-service",
    OTLPConnectionName:            "otel",
    NoNativeCollectorRegistration: true, // Disable runtime metrics
}, registry)
```

## Advanced Usage

### Migration from Direct OTLP Configuration

**Old Code:**
```go
// Before: Direct OTLP configuration
provider := metricsfx.NewMetricsProvider(&metricsfx.Config{
    ServiceName:    "my-service",
    OTLPEndpoint:   "otel-collector:4318",
    ExportInterval: 30 * time.Second,
})
```

**New Code:**
```go
// After: Connection-based configuration
registry := connfx.NewRegistryWithDefaults(logger)
_, err := registry.AddConnection(ctx, "otel", &connfx.ConfigTarget{
    Protocol: "otlp",
    DSN:      "otel-collector:4318",
    Properties: map[string]any{
        "service_name":    "my-service",
        "export_interval": 30 * time.Second,
    },
})

provider := metricsfx.NewMetricsProvider(&metricsfx.Config{
    ServiceName:        "my-service",
    OTLPConnectionName: "otel",
    ExportInterval:     30 * time.Second,
}, registry)
```

### Multiple Metrics Providers

```go
// Different providers for different purposes
businessMetrics := metricsfx.NewMetricsProvider(&metricsfx.Config{
    ServiceName:        "my-service",
    OTLPConnectionName: "otel-business",
    ExportInterval:     60 * time.Second,
}, registry)

systemMetrics := metricsfx.NewMetricsProvider(&metricsfx.Config{
    ServiceName:        "my-service",
    OTLPConnectionName: "otel-system",
    ExportInterval:     15 * time.Second,
}, registry)

// Business metrics - less frequent, more detailed
businessBuilder := businessMetrics.NewBuilder()
revenue, _ := businessBuilder.Counter("revenue_total", "Total revenue").Build()
orders, _ := businessBuilder.Counter("orders_total", "Total orders").Build()

// System metrics - more frequent, operational focus
systemBuilder := systemMetrics.NewBuilder()
requests, _ := systemBuilder.Counter("requests_total", "Total requests").Build()
errors, _ := systemBuilder.Counter("errors_total", "Total errors").Build()
```

### Custom Metric Buckets

```go
// Latency histogram with custom buckets optimized for your use case
latency, _ := builder.Histogram(
    "api_latency_seconds",
    "API request latency distribution",
).WithBuckets([]float64{
    0.001, 0.005, 0.010, 0.025, 0.050,  // sub-50ms buckets
    0.100, 0.250, 0.500,                // normal response times
    1.0, 2.5, 5.0, 10.0,               // slow responses
}).Build()

// Response size histogram
responseSize, _ := builder.Histogram(
    "response_size_bytes",
    "HTTP response size distribution",
).WithBuckets([]float64{
    100, 1000, 10000, 100000, 1000000, // 100B to 1MB
}).Build()

// Database connection pool size
poolSize, _ := builder.Histogram(
    "db_pool_size",
    "Database connection pool size distribution",
).WithBuckets([]float64{
    1, 5, 10, 25, 50, 100, // pool sizes
}).Build()
```

### Error Handling

```go
// Metrics provider handles connection failures gracefully
provider := metricsfx.NewMetricsProvider(&metricsfx.Config{
    ServiceName:        "my-service",
    OTLPConnectionName: "nonexistent-connection", // Connection doesn't exist
}, registry)

// Init may return error, but provider still works locally
err := provider.Init()
if err != nil {
    log.Printf("Metrics export disabled: %v", err)
    // Provider still works, just no OTLP export
}

// Metrics continue to work even without export
builder := provider.NewBuilder()
counter, _ := builder.Counter("local_counter", "Local counter").Build()
counter.Inc(context.Background()) // Works fine locally
```

## Integration with Other Services

### gRPC Metrics

```go
import "github.com/eser/ajan/grpcfx"

// Create gRPC metrics
grpcMetrics := grpcfx.NewMetrics(metricsProvider)
err := grpcMetrics.Init()
if err != nil {
    panic(err)
}

// Use with gRPC service
grpcService := grpcfx.NewGRPCService(
    &grpcfx.Config{Port: 9090},
    metricsProvider,
    logger,
)

// Automatically tracks:
// - grpc_requests_total
// - grpc_request_duration_seconds
```

### Event System Metrics

```go
import "github.com/eser/ajan/eventsfx"

// Create event metrics
eventMetrics := eventsfx.NewMetrics(metricsProvider)
err := eventMetrics.Init()
if err != nil {
    panic(err)
}

// Use with event dispatcher
dispatcher := eventsfx.NewDispatcher(eventMetrics)

// Automatically tracks:
// - event_dispatches_total
```

### Database Metrics

```go
// Custom database metrics
builder := provider.NewBuilder()

queryDuration, _ := builder.Histogram(
    "db_query_duration_seconds",
    "Database query execution time",
).WithDurationBuckets().Build()

connections, _ := builder.Gauge(
    "db_connections_active",
    "Active database connections",
).Build()

queryErrors, _ := builder.Counter(
    "db_query_errors_total",
    "Database query errors",
).Build()

// Use in database layer
func (db *Database) Query(ctx context.Context, sql string) (*Rows, error) {
    start := time.Now()
    defer queryDuration.RecordDuration(ctx, time.Since(start))

    connections.Set(ctx, float64(db.pool.ActiveCount()))

    rows, err := db.conn.QueryContext(ctx, sql)
    if err != nil {
        queryErrors.Inc(ctx,
            metricsfx.StringAttr("error", err.Error()),
            metricsfx.StringAttr("query_type", "select"),
        )
        return nil, err
    }

    return rows, nil
}
```

## Best Practices

1. **Use Centralized Connections**: Configure OTLP connections once in `connfx`, use everywhere
2. **Connection Health Monitoring**: Use `registry.HealthCheck(ctx)` to monitor OTLP connection health
3. **Graceful Degradation**: Metrics provider works with or without OTLP connections
4. **Consistent Naming**: Use standard metric naming conventions (`_total` for counters, `_seconds` for durations)
5. **Meaningful Attributes**: Add relevant dimensions but avoid high cardinality
6. **Resource Attribution**: Set service name/version in connection properties for proper attribution
7. **Export Intervals**: Balance between timeliness and overhead (15-60 seconds typically)
8. **Runtime Metrics**: Enable for production monitoring, disable for lightweight deployments
9. **Connection Lifecycle**: Use `registry.Close(ctx)` during shutdown to properly cleanup connections
10. **Environment-Specific Config**: Use different connection names and intervals for dev/staging/prod

## Architecture Benefits

- **Unified Configuration** - Single place to configure OTLP connections for all observability signals
- **Shared Resources** - Efficient connection pooling and resource utilization
- **Consistent Attribution** - Service information automatically applied to all metrics
- **Health Monitoring** - Built-in connection health checks and monitoring
- **Graceful Fallbacks** - Continue working even when OTLP connections fail
- **Environment Flexibility** - Easy switching between different collectors/environments
- **Import Cycle Prevention** - Bridge pattern avoids circular dependencies
- **Thread Safety** - All connection operations are thread-safe
