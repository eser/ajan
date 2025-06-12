# ajan/logfx

## Overview

**logfx** package is a configurable logging solution that leverages the
`log/slog` of the standard library for structured logging. It includes
pretty-printing options and **OpenTelemetry collector integration** as the
preferred export method, with optional direct Loki export for legacy setups.
The package supports OpenTelemetry-compatible severity levels and provides
extensive test coverage to ensure reliability and correctness.

### Key Features

- 🎯 **Extended Log Levels** - OpenTelemetry-compatible levels while using standard `log/slog` under the hood
- 🔄 **Automatic Correlation IDs** - Request tracing across your entire application
- 🌐 **OpenTelemetry Integration** - Native OTLP export to OpenTelemetry collectors
- 📊 **Multiple Export Formats** - JSON logs, Loki export, OTLP export
- 🎨 **Pretty Printing** - Colored output for development
- ⚡ **Performance Optimized** - Asynchronous exports, structured logging

## 🚀 **Extended Log Levels**

**The Problem**: Go's standard `log/slog` package provides only 4 log levels (Debug, Info, Warn, Error), which is insufficient for modern observability and OpenTelemetry compatibility.

**The Solution**: logfx extends the standard library to provide **7 OpenTelemetry-compatible log levels** while maintaining full compatibility with `log/slog`:

```go
// Standard Go slog levels (limited)
slog.LevelDebug  // -4
slog.LevelInfo   //  0
slog.LevelWarn   //  4
slog.LevelError  //  8

// logfx extended levels (OpenTelemetry compatible)
logfx.LevelTrace // -8  ← Additional
logfx.LevelDebug // -4
logfx.LevelInfo  //  0
logfx.LevelWarn  //  4
logfx.LevelError //  8
logfx.LevelFatal // 12  ← Additional
logfx.LevelPanic // 16  ← Additional
```

### Why This Matters

1. **OpenTelemetry Compatibility** - Maps perfectly to OpenTelemetry log severity levels
2. **Better Observability** - More granular log levels for better debugging and monitoring
3. **Standard Library Foundation** - Built on `log/slog`, not a replacement
4. **Zero Breaking Changes** - Existing slog code works unchanged
5. **Proper Severity Mapping** - Correct OTLP export with appropriate severity levels

### Extended Level Usage

```go
import "github.com/eser/ajan/logfx"

logger := logfx.NewLogger(
    logfx.WithLevel(logfx.LevelTrace), // Now supports all 7 levels
)

// Use all OpenTelemetry-compatible levels
logger.Trace("Detailed debugging info")           // Most verbose
logger.Debug("Debug information")                 // Development debugging
logger.Info("General information")                // Standard info
logger.Warn("Warning message")                    // Potential issues
logger.Error("Error occurred")                    // Errors that don't stop execution
logger.Fatal("Fatal error")                       // Critical errors
logger.Panic("Panic condition")                   // Most severe
```

**Colored Output** (development mode):
```bash
23:45:12.123 TRACE Detailed debugging info
23:45:12.124 DEBUG Debug information
23:45:12.125 INFO General information
23:45:12.126 WARN Warning message
23:45:12.127 ERROR Error occurred
23:45:12.128 FATAL Fatal error
23:45:12.129 PANIC Panic condition
```

**Structured Output** (production mode):
```json
{"time":"2024-01-15T23:45:12.123Z","level":"TRACE","msg":"Detailed debugging info"}
{"time":"2024-01-15T23:45:12.124Z","level":"DEBUG","msg":"Debug information"}
{"time":"2024-01-15T23:45:12.125Z","level":"INFO","msg":"General information"}
{"time":"2024-01-15T23:45:12.126Z","level":"WARN","msg":"Warning message"}
{"time":"2024-01-15T23:45:12.127Z","level":"ERROR","msg":"Error occurred"}
{"time":"2024-01-15T23:45:12.128Z","level":"FATAL","msg":"Fatal error"}
{"time":"2024-01-15T23:45:12.129Z","level":"PANIC","msg":"Panic condition"}
```

**OpenTelemetry Export** (automatic severity mapping):
```json
{
  "logRecords": [
    {"body": {"stringValue": "Detailed debugging info"}, "severityNumber": 1, "severityText": "TRACE"},
    {"body": {"stringValue": "Debug information"}, "severityNumber": 5, "severityText": "DEBUG"},
    {"body": {"stringValue": "General information"}, "severityNumber": 9, "severityText": "INFO"},
    {"body": {"stringValue": "Warning message"}, "severityNumber": 13, "severityText": "WARN"},
    {"body": {"stringValue": "Error occurred"}, "severityNumber": 17, "severityText": "ERROR"},
    {"body": {"stringValue": "Fatal error"}, "severityNumber": 21, "severityText": "FATAL"},
    {"body": {"stringValue": "Panic condition"}, "severityNumber": 24, "severityText": "PANIC"}
  ]
}
```

## Quick Start

### Basic Usage

```go
package main

import (
    "log/slog"
    "os"

    "github.com/eser/ajan/logfx"
)

func main() {
    // Create logger with OpenTelemetry collector export
    logger := logfx.NewLogger(
        logfx.WithOTLP("http://otel-collector:4318", true),
    )

    // Use structured logging with extended levels
    logger.Info("Application started",
        slog.String("service", "my-service"),
        slog.String("version", "1.0.0"),
    )

    // Extended levels for better observability
    logger.Trace("Connection pool initialized")     // Very detailed
    logger.Debug("Processing user request")         // Debug info
    logger.Warn("High memory usage detected")       // Warnings
    logger.Fatal("Database connection failed")      // Critical errors
}
```

### With HTTP Correlation IDs

```go
package main

import (
    "log/slog"
    "net/http"
    "os"

    "github.com/eser/ajan/httpfx"
    "github.com/eser/ajan/httpfx/middlewares"
    "github.com/eser/ajan/logfx"
)

func main() {
    logger := logfx.NewLogger(
        logfx.WithWriter(os.Stdout),
        logfx.WithConfig(&logfx.Config{
            Level:        "TRACE",  // Use extended levels for comprehensive logging
            PrettyMode:   false,
            OTLPEndpoint: "http://otel-collector:4318",
        }),
    )

    router := httpfx.NewRouter("/api")

    // Add correlation middleware for automatic request tracking
    router.Use(middlewares.CorrelationIDMiddleware())
    router.Use(middlewares.LoggingMiddleware(logger))

    router.Route("GET /users/{id}", func(ctx *httpfx.Context) httpfx.Result {
        // All logs automatically include correlation_id from HTTP headers
        logger.TraceContext(ctx.Request.Context(), "Starting user lookup")
        logger.InfoContext(ctx.Request.Context(), "Processing user request",
            slog.String("user_id", "123"),
        )

        return ctx.Results.JSON(map[string]string{"status": "success"})
    })

    http.ListenAndServe(":8080", router.GetMux())
}
```

**Log Output with Correlation:**
```json
{"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"HTTP request started","method":"GET","path":"/api/users/123","correlation_id":"abc-123-def"}
{"time":"2024-01-15T10:30:00Z","level":"TRACE","msg":"Starting user lookup","correlation_id":"abc-123-def"}
{"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"Processing user request","user_id":"123","correlation_id":"abc-123-def"}
{"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"HTTP request completed","method":"GET","status_code":200,"correlation_id":"abc-123-def"}
```

## Configuration

```go
type Config struct {
	Level      string `conf:"level"      default:"INFO"`        // Supports: TRACE, DEBUG, INFO, WARN, ERROR, FATAL, PANIC
	PrettyMode bool   `conf:"pretty"     default:"true"`
	AddSource  bool   `conf:"add_source" default:"false"`

	// OpenTelemetry Collector configuration (preferred)
	OTLPEndpoint string `conf:"otlp_endpoint" default:""`
	OTLPInsecure bool   `conf:"otlp_insecure" default:"true"`

	// Direct Loki export (legacy/additional option)
	LokiURI   string `conf:"loki_uri" default:""`
	LokiLabel string `conf:"loki_label" default:""`
}
```

### Export Priority

The package automatically chooses the best export method:

1. **🥇 OTLP Collector** (`OTLPEndpoint`) - Preferred for production
2. **🥈 Direct Loki** (`LokiURI`) - Legacy/fallback option
3. **Both can run simultaneously** if needed

## OpenTelemetry Collector Integration

### Recommended Setup

```go
config := &logfx.Config{
    Level:        "INFO",           // Use any of the 7 extended levels
    PrettyMode:   false,
    OTLPEndpoint: "http://otel-collector:4318",
    OTLPInsecure: true,
}
logger := logfx.NewLogger(
    logfx.WithWriter(os.Stdout),
    logfx.WithConfig(config),
)
```

### Benefits of OpenTelemetry Collector

- **🔄 Unified Pipeline** - All logs, metrics, and traces flow through one point
- **🎛️ Flexibility** - Change backends without code changes
- **⚡ Performance** - Built-in batching, retries, and buffering
- **💰 Cost Optimization** - Sampling and filtering before export
- **🔧 Processing** - Transform, enrich, and route data

### Example Collector Configuration

```yaml
# otel-collector-config.yaml
receivers:
  otlp:
    protocols:
      http:
        endpoint: 0.0.0.0:4318

exporters:
  loki:
    endpoint: http://loki:3100/loki/api/v1/push

service:
  pipelines:
    logs:
      receivers: [otlp]
      exporters: [loki]
```

## Correlation IDs

### Automatic HTTP Correlation

When using with `httpfx`, correlation IDs are automatically:

- ✅ **Extracted** from `X-Correlation-ID` headers
- ✅ **Generated** if missing
- ✅ **Propagated** through Go context
- ✅ **Added** to all log entries
- ✅ **Included** in response headers

### Manual Correlation Access

```go
import "github.com/eser/ajan/httpfx/middlewares"

func MyHandler(ctx *httpfx.Context) httpfx.Result {
    correlationID := middlewares.GetCorrelationIDFromContext(ctx.Request.Context())

    // Use in external service calls
    externalReq.Header.Set("X-Correlation-ID", correlationID)

    return ctx.Results.JSON(map[string]string{
        "correlation_id": correlationID,
    })
}
```

## Advanced Usage

### Multiple Export Destinations

```go
config := &logfx.Config{
    Level:        "DEBUG",                          // Extended level support
    OTLPEndpoint: "http://otel-collector:4318",     // Primary
    LokiURI:      "http://backup-loki:3100",        // Backup
    LokiLabel:    "backup=true,service=my-app",
}
```

### Level Configuration Examples

```go
// Development - verbose logging with all levels
devConfig := &logfx.Config{
    Level:      "TRACE",    // Most verbose - see everything
    PrettyMode: true,
    AddSource:  true,
}

// Production - structured output with appropriate level
prodConfig := &logfx.Config{
    Level:        "INFO",   // Production appropriate
    PrettyMode:   false,
    OTLPEndpoint: "http://otel-collector:4318",
}

// Debug production issues - temporary verbose logging
debugConfig := &logfx.Config{
    Level:        "DEBUG",  // More detail for troubleshooting
    PrettyMode:   false,
    OTLPEndpoint: "http://otel-collector:4318",
}
```

### Standard Library Compatibility

```go
// logfx extends slog.Level, so standard slog works unchanged
import "log/slog"

// This works exactly as before
slog.Info("Standard slog message")
slog.Debug("Debug with standard slog")

// But you can also use extended levels through logfx
logger.Trace("Extended trace level")    // Not available in standard slog
logger.Fatal("Extended fatal level")    // Not available in standard slog
logger.Panic("Extended panic level")    // Not available in standard slog
```

## Error Handling

The logger handles export failures gracefully:

```go
// Logger continues working even if exports fail
logger := logfx.NewLogger(
    logfx.WithWriter(os.Stdout),
    logfx.WithConfig(config),
)

// Check for initialization errors
if handler, ok := logger.Handler.(*logfx.Handler); ok && handler.InitError != nil {
    log.Printf("Logger init warning: %v", handler.InitError)
}

// Logs always go to the primary writer (stdout/file)
// Export failures are logged to stderr without affecting your app
```

## Observability Integration

### Complete Observability Stack

Use with other `ajan` packages for full observability:

```go
import (
    "github.com/eser/ajan/logfx"     // Extended logs
    "github.com/eser/ajan/metricsfx" // Metrics
    // "github.com/eser/ajan/tracesfx"  // Traces
)

// All export to the same OpenTelemetry collector
config := &Config{
    OTLPEndpoint: "http://otel-collector:4318",
}
```

### Log Correlation with Traces

When OpenTelemetry tracing is active, logs automatically include:

```jsonc
{
  "time": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "msg": "Processing request",
  "correlation_id": "abc-123-def",  // HTTP correlation ID
  "trace_id": "4bf92f3577b34da6",   // OpenTelemetry trace ID
  "span_id": "00f067aa0bb902b7"     // OpenTelemetry span ID
}
```

This provides multiple correlation dimensions for complete request traceability.

## API Reference

### Logger Creation

#### NewLogger (Options Pattern)

```go
func NewLogger(options ...NewLoggerOption) *Logger
```

Create a logger using the flexible options pattern:

```go
// Basic logger with default configuration
logger := logfx.NewLogger()

// Logger with custom writer and config
logger := logfx.NewLogger(
    logfx.WithWriter(os.Stdout),
    logfx.WithConfig(&logfx.Config{
        Level:        "INFO",
        PrettyMode:   false,
        OTLPEndpoint: "http://otel-collector:4318",
    }),
)

// Logger with individual options
logger := logfx.NewLogger(
    logfx.WithLevel(slog.LevelDebug),
    logfx.WithPrettyMode(true),
    logfx.WithAddSource(true),
    logfx.WithOTLP("http://otel-collector:4318", true),
    logfx.WithDefaultLogger(), // Set as default logger
)
```

#### Available Options

```go
// Configuration options
WithConfig(config *Config)                    // Full configuration
WithLevel(level slog.Level)                   // Set log level
WithPrettyMode(pretty bool)                   // Enable/disable pretty printing
WithAddSource(addSource bool)                 // Include source code location
WithDefaultLogger()                           // Set as default logger

// Output options
WithWriter(writer io.Writer)                  // Set output writer
WithFromSlog(slog *slog.Logger)              // Wrap existing slog.Logger

// Export options
WithOTLP(endpoint string, insecure bool)      // OpenTelemetry collector export
WithLoki(uri string, label string)           // Direct Loki export
```

#### Convenience Functions

```go
// Quick default setup
logger := logfx.NewLogger(
    logfx.WithConfig(&logfx.Config{
        Level:      "INFO",
        PrettyMode: true,
        AddSource:  false,
    }),
)

// Wrap existing slog logger
slogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
logger := logfx.NewLogger(logfx.WithFromSlog(slogger))

// Production setup
logger := logfx.NewLogger(
    logfx.WithConfig(&logfx.Config{
        Level:        "INFO",
        PrettyMode:   false,
        OTLPEndpoint: "http://otel-collector:4318",
    }),
)

// Development setup
logger := logfx.NewLogger(
    logfx.WithLevel(slog.LevelDebug),
    logfx.WithPrettyMode(true),
    logfx.WithAddSource(true),
    logfx.WithDefaultLogger(),
)
```

**Usage Examples:**

```go
// Quick default setup
logger := logfx.NewLogger(
    logfx.WithConfig(&logfx.Config{
        Level:      "INFO",
        PrettyMode: true,
        AddSource:  false,
    }),
)

// Wrap existing slog logger
slogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
logger := logfx.NewLogger(logfx.WithFromSlog(slogger))

// Production setup
logger := logfx.NewLogger(
    logfx.WithConfig(&logfx.Config{
        Level:        "INFO",
        PrettyMode:   false,
        OTLPEndpoint: "http://otel-collector:4318",
    }),
)

// Development setup
logger := logfx.NewLogger(
    logfx.WithLevel(slog.LevelDebug),
    logfx.WithPrettyMode(true),
    logfx.WithAddSource(true),
    logfx.WithDefaultLogger(),
)
```

## Ideal Architecture

**🎯 Recommended Setup**: Use OpenTelemetry Collector as the central hub for all observability data.

```
┌─────────────────┐    ┌─────────────────────┐    ┌─────────────────┐
│                 │    │                     │    │                 │
│   Your App      │───▶│ OpenTelemetry       │───▶│  Backends       │
│                 │    │ Collector           │    │                 │
│ • logfx         │    │                     │    │ • Loki (logs)   │
│ • metricsfx     │    │ • Receives all 3    │    │ • Prometheus    │
│ • tracesfx      │    │   pillars (L+M+T)   │    │   (metrics)     │
│                 │    │ • Routes & filters  │    │ • Tempo         │
│                 │    │ • Transforms        │    │   (traces)      │
└─────────────────┘    └─────────────────────┘    └─────────────────┘
```

Benefits:
- **Unified observability pipeline** - All logs, metrics, and traces flow through one point
- **Flexibility** - Change backends without touching application code
- **Processing** - Filter, transform, sample, and enrich data before export
- **Reliability** - Built-in buffering, retries, and load balancing
- **Cost optimization** - Sampling and filtering reduce backend costs
