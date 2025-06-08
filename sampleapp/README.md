# SampleApp

SampleApp is a complete example application that demonstrates how to build a production-ready Go application using the ajan framework. It showcases best practices for application structure, configuration management, logging, metrics, and database integration.

## Features

- **Complete Application Structure**: Demonstrates proper layering and organization of a Go application
- **Configuration Management**: Shows how to use configfx for environment-aware configuration
- **Structured Logging**: Integrates logfx for comprehensive application logging
- **Metrics Collection**: Uses metricsfx for application observability
- **Database Integration**: Demonstrates datafx for database connection management and transactions
- **Docker Support**: Includes multi-stage Dockerfile for development and production builds
- **Docker Compose**: Complete development environment setup
- **Graceful Error Handling**: Proper error handling patterns throughout the application

## Quick Start

### Prerequisites

- Go 1.21 or later
- Docker and Docker Compose (for containerized setup)
- PostgreSQL database (for database examples)

### Running Locally

1. **Clone and navigate to the sample app:**
```bash
cd sampleapp
```

2. **Install dependencies:**
```bash
go mod download
```

3. **Set up the database (using Docker Compose):**
```bash
docker-compose up -d postgres
```

4. **Run the application:**
```bash
go run .
```

### Running with Docker

**Development mode:**
```bash
docker-compose up app-dev
```

**Production mode:**
```bash
docker-compose up app-prod
```

## Application Structure

### Core Components

The application demonstrates the layered architecture pattern:

```
sampleapp/
├── main.go              # Application entry point
├── appcontext.go        # Application context and dependency setup
├── appconfig.go         # Configuration structure
├── config.json          # Base configuration file
├── Dockerfile           # Multi-stage Docker build
├── compose.yml          # Development environment
└── volumes/             # Docker volumes for data persistence
```

### Application Context

The `AppContext` struct centralizes all application dependencies:

```go
type AppContext struct {
    Config  *AppConfig                    // Application configuration
    Logger  *logfx.Logger                // Structured logger
    Metrics *metricsfx.MetricsProvider   // Metrics collection
    Data    *datafx.Registry             // Database connections
}
```

## Configuration

The application uses the ajan configfx module for configuration management with support for multiple sources.

### Configuration Structure

```go
type AppConfig struct {
    ajan.BaseConfig  // Includes common ajan framework configuration
}
```

The `BaseConfig` includes:
- `AppName`: Application name
- `AppEnv`: Environment (development, production, etc.)
- `Log`: Logging configuration
- `Data`: Database configuration
- `Metrics`: Metrics configuration

### Configuration Files

**config.json** (base configuration):
```json
{
  "data": {
    "sources": {
      "default": {
        "provider": "postgres",
        "dsn": "postgres://user:user123@localhost:5432/userdb?sslmode=disable"
      }
    }
  }
}
```

**Environment-specific overrides:**
- `config.development.json` - Development settings
- `config.production.json` - Production settings
- `config.local.json` - Local developer overrides

**Environment variables:**
```bash
export APP_NAME=MyApplication
export APP_ENV=production
export LOG__LEVEL=info
export DATA__SOURCES__DEFAULT__DSN=postgres://prod-host:5432/proddb
```

## Database Integration

The application demonstrates proper database integration using DataFX:

### Connection Management

```go
// Database registry setup
appContext.Data = datafx.NewRegistry(appContext.Logger)
err = appContext.Data.LoadFromConfig(ctx, &appContext.Config.Data)
```

### Transaction Management

Using the Unit of Work pattern for database operations:

```go
func business(ctx context.Context, appContext *AppContext) {
    datasource := appContext.Data.GetDefault()

    err := datasource.ExecuteUnitOfWork(ctx, func(uow *datafx.UnitOfWork) error {
        // All database operations within this function
        // are part of a single transaction

        // service1.DoSomething(uow)
        // service2.DoSomething(uow)

        return nil // Transaction commits automatically
        // return err // Transaction rolls back on error
    })

    if err != nil {
        // Handle transaction failure
        log.Printf("Transaction failed: %v", err)
    }
}
```

## Logging

The application uses structured logging with LogFX:

### Logger Setup

```go
// Create logger with configuration
logger, err := logfx.NewLoggerAsDefault(os.Stdout, &config.Log)

// Use throughout the application
logger.Info("Application started",
    "name", config.AppName,
    "env", config.AppEnv,
)
```

### Log Levels

Configure log level via configuration:

```json
{
  "log": {
    "level": "info",
    "pretty": true,
    "add_source": false
  }
}
```

Or environment variables:
```bash
export LOG__LEVEL=debug
export LOG__PRETTY=false
export LOG__ADD_SOURCE=true
```

## Metrics

The application includes metrics collection using MetricsFX:

### Metrics Setup

```go
// Initialize metrics provider
metrics := metricsfx.NewMetricsProvider()

// Register native Go runtime metrics
err := metrics.RegisterNativeCollectors()
if err != nil {
    log.Fatal("Failed to register metrics collectors:", err)
}
```

### Available Metrics

The application automatically collects:
- **Runtime Metrics**: Memory usage, GC stats, goroutine counts
- **Application Metrics**: Custom business metrics (add as needed)

## Docker Setup

### Multi-Stage Dockerfile

The application includes a production-ready multi-stage Dockerfile:

- **Development Stage**: Hot-reloading with `go run`
- **Production Stage**: Optimized binary in distroless container

### Development Environment

The Docker Compose setup provides:

```yaml
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: userdb
      POSTGRES_USER: user
      POSTGRES_PASSWORD: user123
    ports:
      - "5432:5432"

  app-dev:
    build:
      target: runner-development
    ports:
      - "8080:8080"
    depends_on:
      - postgres

  app-prod:
    build:
      target: runner-production
    ports:
      - "8080:8080"
    depends_on:
      - postgres
```

### Running Services

```bash
# Start database only
docker-compose up -d postgres

# Start development environment
docker-compose up app-dev

# Start production environment
docker-compose up app-prod

# View logs
docker-compose logs -f app-dev
```

## Error Handling

The application demonstrates proper error handling patterns:

### Initialization Errors

```go
func NewAppContext(ctx context.Context) (*AppContext, error) {
    // ... initialization code ...

    if err != nil {
        return nil, fmt.Errorf("%w: %w", ErrInitFailed, err)
    }

    return appContext, nil
}
```

### Business Logic Errors

```go
func main() {
    appContext, err := NewAppContext(ctx)
    if err != nil {
        panic(fmt.Sprintf("failed to initialize app context: %v", err))
    }

    // Continue with business logic
    business(ctx, appContext)
}
```

## Extending the Sample App

### Adding Services

1. **Create service interface and implementation:**
```go
type UserService interface {
    CreateUser(ctx context.Context, user *User) error
    GetUser(ctx context.Context, id string) (*User, error)
}

type userService struct {
    data *datafx.Registry
    logger *logfx.Logger
}
```

2. **Add to application context:**
```go
type AppContext struct {
    // ... existing fields ...
    UserService UserService
}
```

3. **Initialize in NewAppContext:**
```go
appContext.UserService = NewUserService(appContext.Data, appContext.Logger)
```

### Adding HTTP API

1. **Add HTTP server configuration:**
```go
type AppConfig struct {
    ajan.BaseConfig
    HTTP httpfx.Config `conf:"http"`
}
```

2. **Add HTTP server to context:**
```go
type AppContext struct {
    // ... existing fields ...
    HTTP *httpfx.Server
}
```

3. **Set up routes:**
```go
server := httpfx.NewServer(&config.HTTP, logger)
server.GET("/users/:id", userHandler.GetUser)
server.POST("/users", userHandler.CreateUser)
```

### Adding Background Workers

1. **Use processfx for worker management:**
```go
import "github.com/eser/ajan/processfx"

process := processfx.New(ctx, logger)

// Start background worker
process.StartGoroutine("data-processor", func(ctx context.Context) error {
    // Worker logic here
    return nil
})
```

## Best Practices Demonstrated

1. **Dependency Injection**: All dependencies are injected through the AppContext
2. **Configuration Management**: Environment-aware configuration with sensible defaults
3. **Error Handling**: Consistent error wrapping and propagation
4. **Logging**: Structured logging with context
5. **Database Transactions**: Proper transaction management with Unit of Work
6. **Container Ready**: Production-ready Docker setup
7. **Observability**: Built-in metrics and logging

## Development Workflow

### Local Development

1. **Start dependencies:**
```bash
docker-compose up -d postgres
```

2. **Run application:**
```bash
go run .
```

3. **Make changes and restart**

### Testing

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with race detection
go test -race ./...
```

### Building

```bash
# Local build
go build -o app .

# Docker build
docker build -t sampleapp .

# Multi-architecture build
docker buildx build --platform linux/amd64,linux/arm64 -t sampleapp .
```

## Production Deployment

### Environment Variables

Set these environment variables for production:

```bash
export APP_ENV=production
export LOG__LEVEL=info
export LOG__PRETTY=false
export DATA__SOURCES__DEFAULT__DSN=postgres://prod-host:5432/proddb
```

### Health Checks

The application automatically includes health checks for:
- Database connections
- Application metrics

### Monitoring

Access metrics at runtime:
- Application logs: Structured JSON output
- Runtime metrics: Available via metricsfx integration

## Dependencies

- **github.com/eser/ajan**: Main ajan framework
- **github.com/eser/ajan/configfx**: Configuration management
- **github.com/eser/ajan/logfx**: Structured logging
- **github.com/eser/ajan/datafx**: Database integration
- **github.com/eser/ajan/metricsfx**: Metrics collection

## Contributing

This sample application serves as a reference implementation. When contributing:

1. Follow the established patterns
2. Add comprehensive error handling
3. Include appropriate logging
4. Update configuration as needed
5. Add tests for new functionality

## License

This sample application is part of the ajan framework and follows the same license terms.
