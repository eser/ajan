# ajan/httpfx

## Overview

The **httpfx** package provides a framework for building HTTP services with support for routing, middleware, and OpenAPI
documentation generation. The package is designed to work seamlessly with the `ajan/di` package.

The documentation below provides an overview of the package, its types, functions, and usage examples. For more detailed
information, refer to the source code and tests.

## Configuration

Configuration struct for the HTTP service:

```go
type Config struct {
	Addr string `conf:"ADDR" default:":8080"`

	CertString        string        `conf:"CERT_STRING"`
	KeyString         string        `conf:"KEY_STRING"`
	ReadHeaderTimeout time.Duration `conf:"READ_HEADER_TIMEOUT" default:"5s"`
	ReadTimeout       time.Duration `conf:"READ_TIMEOUT"        default:"10s"`
	WriteTimeout      time.Duration `conf:"WRITE_TIMEOUT"       default:"10s"`
	IdleTimeout       time.Duration `conf:"IDLE_TIMEOUT"        default:"120s"`

	InitializationTimeout   time.Duration `conf:"INIT_TIMEOUT"     default:"25s"`
	GracefulShutdownTimeout time.Duration `conf:"SHUTDOWN_TIMEOUT" default:"5s"`

	SelfSigned bool `conf:"SELF_SIGNED" default:"false"`

	HealthCheckEnabled bool `conf:"HEALTH_CHECK" default:"true"`
	OpenApiEnabled     bool `conf:"OPENAPI"      default:"true"`
	ProfilingEnabled   bool `conf:"PROFILING"    default:"false"`
}
```

Example configuration:
```go
config := &httpfx.Config{
	Addr:            ":8080",
	ReadTimeout:     15 * time.Second,
	WriteTimeout:    15 * time.Second,
	IdleTimeout:     60 * time.Second,
	OpenApiEnabled:  true,
	SelfSigned:      false,
}
```

## API

### NewRouter function

Create a new `Router` object.

```go
// func NewRouter(path string) *RouterImpl

router := httpfx.NewRouter("/")
```

### NewHttpService function

Creates a new `HttpService` object based on the provided configuration.

```go
// func NewHttpService(config *Config, router Router) *HttpServiceImpl

router := httpfx.NewRouter("/")
hs := httpfx.NewHttpService(config, router)
```

## Features

- HTTP routing with support for path parameters and wildcards
- Middleware support for request/response processing
- OpenAPI documentation generation
- Graceful shutdown handling
- Configurable timeouts and server settings
- Integration with dependency injection
- Support for CORS and security headers
- Request logging and metrics

## Example Usage

```go
func main() {
	// Create router
	router := httpfx.NewRouter("/api")

	// Add routes
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Add middleware
	router.Use(httpfx.LoggerMiddleware())
	router.Use(httpfx.RecoveryMiddleware())

	// Create and start service
	config := &httpfx.Config{
		Addr: ":8080",
	}
	service := httpfx.NewHttpService(config, router)

	if err := service.Start(); err != nil {
		log.Fatal(err)
	}
}
```
