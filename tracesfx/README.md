# ajan/tracesfx

## Overview

**tracesfx** provides comprehensive distributed tracing capabilities built on OpenTelemetry with **centralized OTLP connection management** through `connfx`. It offers request correlation, automatic span management, and seamless integration with other observability packages for complete telemetry coverage.

### Key Features

- 🔍 **Distributed Tracing** - Full request tracing across service boundaries
- 🔄 **Automatic Correlation** - Request correlation with logs and metrics
- 🌐 **Centralized OTLP Integration** - Uses `connfx` registry for shared OTLP connections
- ⚡ **Performance Optimized** - Efficient batch exports with configurable sampling
- 🎛️ **Flexible Configuration** - Environment-based configuration with sensible defaults
- 🔗 **Service Integration** - Seamless integration with `httpfx`, `grpcfx`, and other packages
- 📊 **Context Propagation** - Automatic trace context propagation across HTTP calls

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "time"

    "github.com/eser/ajan/connfx"
    "github.com/eser/ajan/logfx"
    "github.com/eser/ajan/tracesfx"
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

    // Create traces provider with connection registry
    provider := tracesfx.NewTracesProvider(&tracesfx.Config{
        ServiceName:        "my-service",
        ServiceVersion:     "1.0.0",
        OTLPConnectionName: "otel", // Reference the connection
        SampleRatio:        1.0,    // Sample 100% of traces for development
        BatchTimeout:       5 * time.Second,
        BatchSize:          512,
    }, registry) // Pass the registry

    // Initialize provider (enables OTLP export)
    err = provider.Init()
    if err != nil {
        panic(err)
    }
    defer provider.Shutdown(ctx)

    // Create tracer for your service
    tracer := provider.Tracer("my-service")

    // Start tracing operations
    spanCtx, span := tracer.Start(ctx, "main_operation")
    defer span.End()

    // Add attributes to spans
    span.SetAttributes(
        tracesfx.StringAttr("user_id", "123"),
        tracesfx.StringAttr("operation", "data_processing"),
    )

    // Nested operations are automatically child spans
    processData(spanCtx, tracer)

    // Add events to spans
    span.AddEvent("operation_completed")
}

func processData(ctx context.Context, tracer *tracesfx.Tracer) {
    // Child spans are automatically created with proper parent relationship
    _, span := tracer.Start(ctx, "process_data")
    defer span.End()

    // Simulate work
    time.Sleep(100 * time.Millisecond)

    span.SetAttributes(tracesfx.IntAttr("records_processed", 42))
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
            "sample_ratio":     1.0,
            "batch_timeout":    "5s",
            "batch_size":       512,
        },
    })
    if err != nil {
        panic(err)
    }

    // Step 2: Create observability stack using shared connection

    // Traces for request tracing
    tracesProvider := tracesfx.NewTracesProvider(&tracesfx.Config{
        ServiceName:        "my-api",
        ServiceVersion:     "1.0.0",
        OTLPConnectionName: "otel",
        SampleRatio:        1.0,
        BatchTimeout:       5 * time.Second,
        BatchSize:          512,
    }, registry)
    _ = tracesProvider.Init()

    // Metrics with automatic correlation
    metricsProvider := metricsfx.NewMetricsProvider(&metricsfx.Config{
        ServiceName:        "my-api",
        ServiceVersion:     "1.0.0",
        OTLPConnectionName: "otel",
        ExportInterval:     15 * time.Second,
    }, registry)
    _ = metricsProvider.Init()

    // Logs with trace correlation
    logger = logfx.NewLogger(
        logfx.WithConfig(&logfx.Config{
            Level:              "INFO",
            OTLPConnectionName: "otel",
        }),
        logfx.WithRegistry(registry),
    )

    // Step 3: Setup HTTP service with tracing middleware
    router := httpfx.NewRouter("/api")

    // Get tracer for HTTP service
    tracer := tracesProvider.Tracer("my-api-http")

    // Add observability middleware
    router.Use(middlewares.CorrelationIDMiddleware())
    router.Use(middlewares.TracingMiddleware(tracer))  // Automatic request tracing
    router.Use(middlewares.LoggingMiddleware(logger))  // Logs include trace context

    // Create custom business metrics that are correlated with traces
    builder := metricsProvider.NewBuilder()
    userOperations, _ := builder.Counter(
        "user_operations_total",
        "Total user operations performed",
    ).WithUnit("{operation}").Build()

    router.Route("GET /users/{id}", func(ctx *httpfx.Context) httpfx.Result {
        // Get tracer from context (set by tracing middleware)
        span := tracesfx.SpanFromContext(ctx.Request.Context())

        // Add business attributes to the span
        span.SetAttributes(
            tracesfx.StringAttr("user_id", "123"),
            tracesfx.StringAttr("operation", "lookup"),
        )

        // Business metrics automatically correlated with trace
        userOperations.Inc(ctx.Request.Context(),
            tracesfx.StringAttr("operation", "lookup"),
            tracesfx.StringAttr("user_id", "123"),
        )

        // Nested operation creates child span automatically
        userData := fetchUserData(ctx.Request.Context(), tracer, "123")

        // Add event to span
        span.AddEvent("user_data_retrieved")

        return ctx.Results.JSON(userData)
    })

    http.ListenAndServe(":8080", router.GetMux())
}

func fetchUserData(ctx context.Context, tracer *tracesfx.Tracer, userID string) map[string]string {
    // Create child span for database operation
    _, span := tracer.Start(ctx, "fetch_user_from_db")
    defer span.End()

    // Add database-specific attributes
    span.SetAttributes(
        tracesfx.StringAttr("db.operation", "SELECT"),
        tracesfx.StringAttr("db.table", "users"),
        tracesfx.StringAttr("user_id", userID),
    )

    // Simulate database operation
    time.Sleep(50 * time.Millisecond)

    return map[string]string{"name": "John Doe", "email": "john@example.com"}
}
```

**Trace Output (automatically exported to OTLP):**
```json
{
  "resourceSpans": [
    {
      "resource": {
        "attributes": [
          {"key": "service.name", "value": {"stringValue": "my-api"}},
          {"key": "service.version", "value": {"stringValue": "1.0.0"}}
        ]
      },
      "scopeSpans": [
        {
          "spans": [
            {
              "traceId": "4bf92f3577b34da6a3ce929d0e0e4736",
              "spanId": "00f067aa0bb902b7",
              "name": "GET /api/users/{id}",
              "kind": "SPAN_KIND_SERVER",
              "attributes": [
                {"key": "http.method", "value": {"stringValue": "GET"}},
                {"key": "http.route", "value": {"stringValue": "/api/users/{id}"}},
                {"key": "user_id", "value": {"stringValue": "123"}},
                {"key": "operation", "value": {"stringValue": "lookup"}}
              ],
              "events": [
                {"name": "user_data_retrieved", "timeUnixNano": "1641024000000000000"}
              ]
            },
            {
              "traceId": "4bf92f3577b34da6a3ce929d0e0e4736",
              "spanId": "b7ad6b7169203331",
              "parentSpanId": "00f067aa0bb902b7",
              "name": "fetch_user_from_db",
              "attributes": [
                {"key": "db.operation", "value": {"stringValue": "SELECT"}},
                {"key": "db.table", "value": {"stringValue": "users"}},
                {"key": "user_id", "value": {"stringValue": "123"}}
              ]
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
    // Service identification (automatically applied to all spans)
    ServiceName    string `conf:"service_name"    default:""`
    ServiceVersion string `conf:"service_version" default:""`

    // Connection-based OTLP configuration (replaces direct endpoint config)
    OTLPConnectionName string `conf:"otlp_connection_name" default:""`

    // Sampling configuration
    SampleRatio float64 `conf:"sample_ratio" default:"1.0"`

    // Batch export configuration
    BatchTimeout time.Duration `conf:"batch_timeout" default:"5s"`
    BatchSize    int           `conf:"batch_size"    default:"512"`
}
```

## Centralized Connection Management

### Why Use connfx for OTLP Connections?

The new architecture centralizes OTLP connection management through `connfx`, providing significant advantages:

**Before (Old Architecture):**
```go
// Each package configured separately - duplicated configuration
traces := tracesfx.NewTracesProvider(&tracesfx.Config{
    OTLPEndpoint: "otel-collector:4318",
    ServiceName:  "my-service",
})
metrics := metricsfx.NewMetricsProvider(&metricsfx.Config{
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
traces := tracesfx.NewTracesProvider(&tracesfx.Config{OTLPConnectionName: "otel"}, registry)
metrics := metricsfx.NewMetricsProvider(&metricsfx.Config{OTLPConnectionName: "otel"}, registry)
```

**Benefits:**
- 🔧 **Single Configuration Point** - Configure OTLP once, use everywhere
- 🔄 **Shared Connections** - Efficient resource usage and connection pooling
- 🎛️ **Centralized Management** - Health checks, monitoring, and lifecycle management
- 🔗 **Consistent Attribution** - Service name/version automatically applied to all spans
- 💰 **Cost Optimization** - Single connection reduces overhead
- 🛡️ **Error Handling** - Graceful fallbacks when connections are unavailable

### OTLP Connection Configuration

```go
// Configure OTLP connection with traces-specific options
otlpConfig := &connfx.ConfigTarget{
    Protocol: "otlp",
    DSN:      "otel-collector:4318",
    Properties: map[string]any{
        // Service identification (applied to all spans automatically)
        "service_name":    "my-service",
        "service_version": "1.0.0",

        // Connection settings
        "insecure":        true,                    // Use HTTP instead of HTTPS

        // Traces-specific export configuration
        "sample_ratio":    1.0,                    // Trace sampling ratio (0.0-1.0)
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
CONN_TARGETS_OTEL_PROPERTIES_SAMPLE_RATIO=1.0
CONN_TARGETS_OTEL_PROPERTIES_BATCH_TIMEOUT=5s

# Package configuration references the connection
TRACES_SERVICE_NAME=my-service
TRACES_SERVICE_VERSION=1.0.0
TRACES_OTLP_CONNECTION_NAME=otel
TRACES_SAMPLE_RATIO=1.0
```

### Multiple OTLP Endpoints

```go
// Different endpoints for different environments
_, err := registry.AddConnection(ctx, "otel-dev", &connfx.ConfigTarget{
    Protocol: "otlp",
    URL:      "http://dev-collector:4318",
    Properties: map[string]any{
        "service_name":  "my-service-dev",
        "sample_ratio":  1.0,  // Sample all traces in development
    },
})

_, err = registry.AddConnection(ctx, "otel-prod", &connfx.ConfigTarget{
    Protocol: "otlp",
    URL:      "https://prod-collector:4317",
    TLS:      true,
    Properties: map[string]any{
        "service_name":  "my-service",
        "sample_ratio":  0.1,  // Sample 10% of traces in production
        "insecure":      false,
    },
})

// Use different connections for different environments
devTraces := tracesfx.NewTracesProvider(&tracesfx.Config{
    OTLPConnectionName: "otel-dev",
}, registry)

prodTraces := tracesfx.NewTracesProvider(&tracesfx.Config{
    OTLPConnectionName: "otel-prod",
}, registry)
```

## Best Practices

1. **Use Centralized Connections**: Configure OTLP connections once in `connfx`, use everywhere
2. **Connection Health Monitoring**: Use `registry.HealthCheck(ctx)` to monitor OTLP connection health
3. **Graceful Degradation**: Traces provider works with or without OTLP connections
4. **Meaningful Span Names**: Use descriptive names that indicate what the span does
5. **Appropriate Attributes**: Add relevant context but avoid high cardinality values
6. **Resource Attribution**: Set service name/version in connection properties for proper attribution
7. **Sampling Strategy**: Use appropriate sampling ratios for your environment (higher for dev, lower for prod)
8. **Batch Configuration**: Balance between latency and efficiency with batch settings
9. **Error Recording**: Always record errors and set appropriate span status
10. **Context Propagation**: Use context.Context consistently to maintain trace relationships
11. **Connection Lifecycle**: Use `registry.Close(ctx)` during shutdown to properly cleanup connections
12. **Environment-Specific Config**: Use different connection names and sampling for dev/staging/prod

## Architecture Benefits

- **Unified Configuration** - Single place to configure OTLP connections for all observability signals
- **Shared Resources** - Efficient connection pooling and resource utilization
- **Consistent Attribution** - Service information automatically applied to all spans
- **Health Monitoring** - Built-in connection health checks and monitoring
- **Graceful Fallbacks** - Continue working even when OTLP connections fail
- **Environment Flexibility** - Easy switching between different collectors/environments
- **Import Cycle Prevention** - Bridge pattern avoids circular dependencies
- **Thread Safety** - All connection operations are thread-safe
- **Context Propagation** - Automatic trace context propagation across service boundaries
- **Correlation Integration** - Seamless integration with correlation IDs and logging
