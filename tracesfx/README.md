# 🔍 **tracesfx** - Distributed Tracing with OpenTelemetry

**tracesfx** provides a clean, lightweight wrapper around OpenTelemetry tracing that integrates seamlessly with `logfx` and `metricsfx` to create a complete observability stack.

## ✨ **Key Features**

- 🚀 **Zero-Config Start** - Works out of the box with sensible defaults
- 🔗 **Seamless Integration** - Native correlation with logs and metrics
- 📊 **OTLP Support** - Built-in OpenTelemetry Protocol export
- 🎯 **Clean API** - Simple, focused interface without unnecessary complexity
- 🔧 **Configurable Sampling** - Control trace volume with ratio-based sampling
- 📝 **Auto-Correlation** - Automatic correlation ID propagation from `logfx`

## 🏗️ **Architecture Integration**

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   logfx     │◄──►│  tracesfx   │◄──►│ metricsfx   │
│ (Logging)   │    │ (Tracing)   │    │ (Metrics)   │
└─────────────┘    └─────────────┘    └─────────────┘
       │                   │                   │
       └───────────────────┼───────────────────┘
                           │
               ┌─────────────────────┐
               │  OpenTelemetry      │
               │  Collector/Backend  │
               └─────────────────────┘
```

## 📦 **Installation**

```bash
go get go.opentelemetry.io/otel
```

The required OpenTelemetry dependencies are automatically included.

## 🚀 **Quick Start**

### Basic Setup

```go
package main

import (
    "context"
    "log"
    
    "github.com/eser/ajan/tracesfx"
)

func main() {
    // Create and initialize traces provider
    provider := tracesfx.NewTracesProvider(&tracesfx.Config{
        ServiceName:    "my-service",
        ServiceVersion: "1.0.0",
        OTLPEndpoint:   "http://localhost:4318",
        OTLPInsecure:   true,
        SampleRatio:    1.0, // Sample 100% for development
    })
    
    if err := provider.Init(); err != nil {
        log.Fatal(err)
    }
    defer provider.Shutdown(context.Background())
    
    // Get a tracer
    tracer := provider.Tracer("my-component")
    
    // Create spans
    ctx, span := tracer.Start(context.Background(), "do-work")
    defer span.End()
    
    // Your business logic here
    doWork(ctx)
}

func doWork(ctx context.Context) {
    // Spans automatically propagate through context
    span := trace.SpanFromContext(ctx)
    span.SetAttributes(attribute.String("work.type", "important"))
}
```

### Integration with logfx

```go
import (
    "github.com/eser/ajan/logfx"
    "github.com/eser/ajan/tracesfx"
)

func setupObservability() {
    // Setup tracing
    tracesProvider := tracesfx.NewTracesProvider(&tracesfx.Config{
        ServiceName:  "my-service",
        OTLPEndpoint: "http://localhost:4318",
    })
    tracesProvider.Init()
    
    // Setup logging
    logger := logfx.NewLogger(os.Stdout, &logfx.Config{
        Level:        "INFO",
        OTLPEndpoint: "http://localhost:4318", // Same endpoint
    })
    
    // Use together - trace IDs automatically appear in logs
    tracer := tracesProvider.Tracer("my-service")
    ctx, span := tracer.Start(context.Background(), "user-request")
    
    // This log will include trace_id and span_id
    logger.InfoContext(ctx, "Processing user request", 
        slog.String("user_id", "12345"))
    
    span.End()
}
```

### Correlation ID Integration

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    // Extract or generate correlation ID (typically done by middleware)
    correlationID := r.Header.Get("X-Correlation-ID")
    if correlationID == "" {
        correlationID = generateCorrelationID()
    }
    
    // Add to context
    ctx := tracesfx.SetCorrelationIDInContext(r.Context(), correlationID)
    
    // Start span with automatic correlation
    tracer := getTracer()
    ctx, span := tracer.StartSpanWithCorrelation(ctx, "handle-request")
    defer span.End()
    
    // Both traces and logs will include correlation_id
    processRequest(ctx)
}
```

## ⚙️ **Configuration**

### Config Structure

```go
type Config struct {
    // Service identification
    ServiceName    string        `conf:"service_name"`
    ServiceVersion string        `conf:"service_version"`
    
    // OpenTelemetry Collector
    OTLPEndpoint   string        `conf:"otlp_endpoint"`     // e.g. "http://localhost:4318"
    OTLPInsecure   bool          `conf:"otlp_insecure"`     // default: true
    
    // Sampling
    SampleRatio    float64       `conf:"sample_ratio"`      // 0.0 to 1.0, default: 1.0
    
    // Batching
    BatchTimeout   time.Duration `conf:"batch_timeout"`     // default: 5s
    BatchSize      int           `conf:"batch_size"`        // default: 512
}
```

### Environment Variables

```bash
# Service information
SERVICE_NAME=my-service
SERVICE_VERSION=1.0.0

# OTLP configuration
OTLP_ENDPOINT=http://localhost:4318
OTLP_INSECURE=true

# Sampling configuration
SAMPLE_RATIO=0.1  # Sample 10% in production

# Batch configuration
BATCH_TIMEOUT=5s
BATCH_SIZE=512
```

## 🔗 **Integration Patterns**

### Complete Observability Stack

```go
type ObservabilityStack struct {
    Traces  *tracesfx.TracesProvider
    Metrics *metricsfx.MetricsProvider
    Logger  *logfx.Logger
}

func NewObservabilityStack(config *Config) *ObservabilityStack {
    // Shared OTLP endpoint for all signals
    otlpEndpoint := config.OTLPEndpoint
    
    return &ObservabilityStack{
        Traces: tracesfx.NewTracesProvider(&tracesfx.Config{
            ServiceName:  config.ServiceName,
            OTLPEndpoint: otlpEndpoint,
            SampleRatio:  config.TraceSampleRatio,
        }),
        Metrics: metricsfx.NewMetricsProvider(&metricsfx.Config{
            ServiceName:  config.ServiceName,
            OTLPEndpoint: otlpEndpoint,
        }),
        Logger: logfx.NewLogger(os.Stdout, &logfx.Config{
            Level:        config.LogLevel,
            OTLPEndpoint: otlpEndpoint,
        }),
    }
}

func (o *ObservabilityStack) Init() error {
    if err := o.Traces.Init(); err != nil {
        return err
    }
    if err := o.Metrics.Init(); err != nil {
        return err
    }
    return nil
}
```

### HTTP Middleware Integration

```go
func TracingMiddleware(tracer *tracesfx.Tracer) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx, span := tracer.Start(r.Context(), fmt.Sprintf("%s %s", r.Method, r.URL.Path))
            defer span.End()
            
            // Add HTTP attributes
            span.SetAttributes(
                attribute.String("http.method", r.Method),
                attribute.String("http.url", r.URL.String()),
                attribute.String("http.user_agent", r.UserAgent()),
            )
            
            // Continue with tracing context
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

## 📊 **Span Management**

### Creating Spans

```go
// Basic span
ctx, span := tracer.Start(ctx, "operation-name")
defer span.End()

// With attributes
ctx, span := tracer.Start(ctx, "database-query",
    trace.WithAttributes(
        attribute.String("db.table", "users"),
        attribute.String("db.operation", "SELECT"),
    ))
defer span.End()

// With correlation (automatic)
ctx, span := tracer.StartSpanWithCorrelation(ctx, "api-call")
defer span.End()
```

### Span Operations

```go
// Set attributes
span.SetAttributes(
    attribute.String("user.id", "12345"),
    attribute.Int("batch.size", 100),
)

// Add events
span.AddEvent("cache-miss")
span.AddEvent("retry-attempt", 
    attribute.Int("attempt", 2))

// Record errors
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, "operation failed")
}

// Success
span.SetStatus(codes.Ok, "completed successfully")
```

## 🔄 **Context Propagation**

### Automatic Propagation

```go
func parentOperation(ctx context.Context) {
    tracer := getTracer()
    ctx, span := tracer.Start(ctx, "parent")
    defer span.End()
    
    // Child operations automatically inherit trace context
    childOperation(ctx)  // Will be a child span
}

func childOperation(ctx context.Context) {
    tracer := getTracer()
    ctx, span := tracer.Start(ctx, "child")
    defer span.End()
    
    // This span will be a child of "parent"
}
```

### Cross-Service Propagation

```go
func makeHTTPRequest(ctx context.Context, url string) {
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    
    // OpenTelemetry automatically injects trace headers
    client := &http.Client{}
    resp, err := client.Do(req)
    // Trace context is propagated to the remote service
}
```

## 🎯 **Best Practices**

### 1. Span Naming

```go
// ✅ Good - descriptive, hierarchical
tracer.Start(ctx, "user.authentication.validate")
tracer.Start(ctx, "database.users.query")
tracer.Start(ctx, "cache.redis.get")

// ❌ Avoid - too generic
tracer.Start(ctx, "process")
tracer.Start(ctx, "operation")
```

### 2. Attribute Management

```go
// ✅ Good - semantic conventions
span.SetAttributes(
    semconv.HTTPMethodKey.String("GET"),
    semconv.HTTPURLKey.String(url),
    semconv.HTTPStatusCodeKey.Int(200),
)

// ✅ Good - business context
span.SetAttributes(
    attribute.String("user.role", "admin"),
    attribute.String("tenant.id", "org-123"),
)
```

### 3. Error Handling

```go
func operationWithTracing(ctx context.Context) error {
    ctx, span := tracer.Start(ctx, "risky-operation")
    defer span.End()
    
    result, err := riskyOperation()
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, "operation failed")
        return err
    }
    
    span.SetStatus(codes.Ok, "success")
    span.SetAttributes(attribute.String("result", result))
    return nil
}
```

### 4. Sampling Configuration

```go
// Development - trace everything
config.SampleRatio = 1.0

// Production - sample based on load
config.SampleRatio = 0.1  // 10%

// High-volume services
config.SampleRatio = 0.01 // 1%
```

## 🔧 **Advanced Usage**

### Custom Attributes

```go
// Business-specific attributes
span.SetAttributes(
    attribute.String("order.id", orderID),
    attribute.String("customer.tier", "premium"),
    attribute.Int("item.count", len(items)),
    attribute.Float64("order.total", 299.99),
)
```

### Span Links

```go
// Link to related operations
ctx, span := tracer.Start(ctx, "batch-process",
    trace.WithLinks(trace.Link{
        SpanContext: relatedSpanContext,
    }))
```

### Manual Instrumentation

```go
func instrumentedFunction(ctx context.Context) {
    tracer := otel.Tracer("my-component")
    ctx, span := tracer.Start(ctx, "custom-operation")
    defer span.End()
    
    // Add custom logic
    span.AddEvent("starting-phase-1")
    phase1(ctx)
    
    span.AddEvent("starting-phase-2") 
    phase2(ctx)
}
```

## 🐛 **Debugging**

### Trace Verification

```go
// Check if tracing is active
if span := trace.SpanFromContext(ctx); span.IsRecording() {
    // Tracing is active
    traceID := span.SpanContext().TraceID().String()
    fmt.Printf("Trace ID: %s\n", traceID)
}

// Get trace/span IDs for logging
traceID := tracesfx.GetTraceIDFromContext(ctx)
spanID := tracesfx.GetSpanIDFromContext(ctx)
```

### No-Op Mode

```go
// When OTLPEndpoint is empty, tracesfx uses no-op tracers
config := &tracesfx.Config{
    ServiceName:  "my-service",
    OTLPEndpoint: "", // No tracing overhead
}
```

## 📈 **Performance Considerations**

- **Sampling**: Use appropriate sample ratios for production workloads
- **Batch Configuration**: Tune batch size and timeout for your throughput
- **Attribute Limits**: Be mindful of attribute cardinality
- **Context Overhead**: Minimal overhead when properly configured

## 🔗 **Integration Examples**

### With HTTP Server

```go
func main() {
    provider := tracesfx.NewTracesProvider(&tracesfx.Config{
        ServiceName:  "web-api",
        OTLPEndpoint: "http://localhost:4318",
    })
    provider.Init()
    defer provider.Shutdown(context.Background())
    
    tracer := provider.Tracer("http-server")
    
    http.HandleFunc("/users", TracingMiddleware(tracer)(UsersHandler))
    http.ListenAndServe(":8080", nil)
}
```

### With gRPC

```go
import "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

func setupGRPCServer() {
    provider := tracesfx.NewTracesProvider(config)
    provider.Init()
    
    server := grpc.NewServer(
        grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
        grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
    )
}
```

## 🏷️ **Related Packages**

- **[logfx](../logfx/README.md)** - Structured logging with automatic trace correlation
- **[metricsfx](../metricsfx/README.md)** - Metrics collection and export
- **[httpfx](../httpfx/README.md)** - HTTP server with observability middleware

---

**tracesfx** provides the distributed tracing foundation for your observability stack, seamlessly integrating with logs and metrics to give you complete visibility into your application's behavior. 
