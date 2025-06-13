# ajan/grpcfx

## Overview

**grpcfx** package provides a framework for building gRPC services with
support for reflection, graceful shutdown, and integration with the dependency
injection system.

## Configuration

Configuration struct for the gRPC service:

```go
type Config struct {
  Addr                    string        `conf:"addr"             default:":9090"`
  Reflection              bool          `conf:"reflection"       default:"true"`
  InitializationTimeout   time.Duration `conf:"init_timeout"     default:"25s"`
  GracefulShutdownTimeout time.Duration `conf:"shutdown_timeout" default:"5s"`
}
```

Example configuration:

```go
config := &grpcfx.Config{
  Addr:                    ":50051",
  Reflection:              true,
  InitializationTimeout:   30 * time.Second,
  GracefulShutdownTimeout: 10 * time.Second,
}
```

## Key Features

- gRPC service setup and configuration
- Server reflection support
- Graceful shutdown handling
- Integration with dependency injection
- Configurable timeouts
- Support for multiple services
- Middleware support

## API

### GRPCService

The main component for gRPC service handling:

```go
// Create a new gRPC service
service := grpcfx.NewGRPCService(config)

// Register your gRPC service implementations
pb.RegisterYourServiceServer(service.GetServer(), &YourServiceImpl{})

// Start the service
if err := service.Start(); err != nil {
  log.Fatal(err)
}
```
