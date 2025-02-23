# ajan/metricsfx

## Overview

The **metricsfx** package provides metrics collection and monitoring utilities
using Prometheus. It integrates with other components to provide metrics for
HTTP services, gRPC services, and custom metrics.

## Features

- Prometheus metrics integration
- HTTP metrics collection
- gRPC metrics collection
- Custom metrics support
- Integration with dependency injection
- Automatic metric registration

## API

### MetricsProvider

The main interface for metrics functionality:

```go
// Create a new metrics registry
registry := prometheus.NewRegistry()

// Create metrics provider
metricsProvider := metricsfx.NewMetricsProvider(registry)

// Register custom metrics
counter := prometheus.NewCounter(prometheus.CounterOpts{
  Name: "my_counter",
  Help: "Example counter",
})
metricsProvider.GetRegistry().MustRegister(counter)
```

### HTTP Integration

```go
// Add metrics middleware to your HTTP router
router.Use(metricsfx.NewMetricsMiddleware(metricsProvider))
```

### gRPC Integration

```go
// Add metrics interceptor to your gRPC server
grpcServer := grpc.NewServer(
  grpc.UnaryInterceptor(metricsfx.NewMetricsInterceptor(metricsProvider)),
)
```
