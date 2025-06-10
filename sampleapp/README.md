# Sample Application

A comprehensive example application demonstrating the ajan framework capabilities, including modern connection management, data persistence, caching, message queues, logging, and metrics collection.

## Architecture

This sample application showcases the **separation of concerns** principle in ajan:

```
Application Layer (main.go)
    ↓ uses
Business Layer (datafx)
    ↓ depends on
Adapter Layer (connfx)
    ↓ implements
Infrastructure (SQLite, Redis, AMQP, etc.)
```

## Features Demonstrated

- **🔌 Connection Management**: Multi-protocol connection registry with health checks
- **📊 Data Operations**: Technology-agnostic CRUD operations with automatic JSON marshaling
- **💾 Transaction Support**: ACID transactions for compatible storage backends
- **⚡ Cache Operations**: High-performance caching with TTL/expiration support
- **📨 Message Queues**: Reliable message publishing and consumption with acknowledgments
- **📝 Structured Logging**: JSON-formatted logs with configurable levels
- **📈 Metrics Collection**: Runtime and custom metrics with Prometheus integration
- **🐳 Docker Support**: Multi-stage builds for development and production

## Quick Start

### Local Development

```bash
# Clone and enter directory
git clone <repository>
cd ajan/sampleapp

# Install dependencies
go mod download

# Run the application
go run .
```

### Docker Development

```bash
# Start with hot-reloading
docker compose up --build

# Or run detached
docker compose up -d
```

## Project Structure

```
sampleapp/
├── main.go              # Application entry point and business logic
├── appcontext.go        # Application context initialization
├── appconfig.go         # Configuration structure
├── config.json          # Default configuration
├── Dockerfile           # Multi-stage Docker build
├── compose.yml          # Docker Compose setup
├── go.mod               # Go module dependencies
└── README.md            # This file
```

## Configuration

The application uses a layered configuration system supporting JSON files and environment variables.

### Configuration File (config.json)

```json
{
  "app_name": "sample-app",
  "app_env": "development",
  "log": {
    "level": "info",
    "pretty": true,
    "add_source": false
  },
  "conn": {
    "targets": {
      "default": {
        "protocol": "sqlite",
        "dsn": ":memory:"
      },
      "redis-cache": {
        "protocol": "redis",
        "host": "localhost",
        "port": 6379,
        "dsn": "redis://localhost:6379"
      },
      "amqp-queue": {
        "protocol": "amqp",
        "host": "localhost",
        "port": 5672,
        "dsn": "amqp://guest:guest@localhost:5672/"
      }
    }
  }
}
```

### Environment Variables

Override any configuration using environment variables with double underscore notation:

```bash
# Application
export app_name=my-app
export app_env=production

# Logging
export log__level=debug
export log__pretty=false
export log__add_source=true

# Connections
export conn__targets__default__dsn="file:app.db?cache=shared&mode=rwc"
export conn__targets__redis_cache__host=redis.example.com
export conn__targets__redis_cache__port=6380
```

## Data Operations

The application demonstrates various data persistence patterns using the modern datafx API.

### Basic CRUD Operations

```go
// Create data instance from connection
data, err := datafx.NewStore(connection)
if err != nil {
    return fmt.Errorf("failed to create data instance: %w", err)
}

// Set data (automatic JSON marshaling)
user := &datafx.User{
    ID:    "user123",
    Name:  "John Doe",
    Email: "john@example.com",
}
err = data.Set(ctx, "user:123", user)

// Get data (automatic JSON unmarshalling)
var retrievedUser datafx.User
err = data.Get(ctx, "user:123", &retrievedUser)

// Update existing data
retrievedUser.Name = "John Smith"
err = data.Update(ctx, "user:123", &retrievedUser)

// Check existence
exists, err := data.Exists(ctx, "user:123")

// Remove data
err = data.Remove(ctx, "user:123")
```

### Transaction Management

For storage backends that support ACID transactions:

```go
// Create transactional store instance
txData, err := datafx.NewTransactionalStore(connection)
if err != nil {
    return fmt.Errorf("transactions not supported: %w", err)
}

// Execute operations within a transaction
err = txData.ExecuteTransaction(ctx, func(tx *datafx.TransactionStore) error {
    // All operations within this function are transactional
    user := &datafx.User{ID: "tx-user-123", Name: "Transaction User"}
    if err := tx.Set(ctx, "tx-user:123", user); err != nil {
        return err // Transaction will be rolled back
    }

    product := &datafx.Product{ID: "tx-product-456", Name: "Widget", Price: 19.99}
    if err := tx.Set(ctx, "tx-product:456", product); err != nil {
        return err // Transaction will be rolled back
    }

    return nil // Transaction will be committed
})
```

## Cache Operations

For connections that support caching (Redis, Memcached, etc.):

```go
// Create cache instance
cache, err := datafx.NewCache(connection)
if err != nil {
    return fmt.Errorf("cache operations not supported: %w", err)
}

// Set with expiration
sessionData := map[string]any{
    "user_id":    "user123",
    "session_id": "sess_abc123",
    "expires_at": time.Now().Add(5 * time.Minute),
}
err = cache.Set(ctx, "session:abc123", sessionData, 5*time.Minute)

// Get cached value
var retrievedSession map[string]any
err = cache.Get(ctx, "session:abc123", &retrievedSession)

// Check TTL
ttl, err := cache.GetTTL(ctx, "session:abc123")

// Extend expiration
err = cache.Expire(ctx, "session:abc123", 10*time.Minute)

// Cache raw data
rawData := []byte("temporary data")
err = cache.SetRaw(ctx, "temp:data", rawData, 1*time.Minute)

// Delete from cache
err = cache.Delete(ctx, "session:abc123")
```

## Message Queue Operations

For connections that support message queues (AMQP/RabbitMQ, Kafka, etc.):

```go
// Create queue instance
queue, err := datafx.NewQueue(connection)
if err != nil {
    return fmt.Errorf("queue operations not supported: %w", err)
}

// Declare a queue
queueName, err := queue.DeclareQueue(ctx, "app-events")

// Publish structured message (automatic JSON marshaling)
eventMessage := map[string]any{
    "event_type": "user_login",
    "user_id":    "user123",
    "timestamp":  time.Now(),
    "metadata": map[string]string{
        "ip_address": "192.168.1.100",
        "user_agent": "Mozilla/5.0...",
    },
}
err = queue.Publish(ctx, queueName, eventMessage)

// Publish raw message
rawMessage := []byte(`{"raw": "event", "data": "some binary data"}`)
err = queue.PublishRaw(ctx, queueName, rawMessage)

// Consume messages
messages, errors := queue.ConsumeWithDefaults(ctx, queueName)

go func() {
    for {
        select {
        case msg := <-messages:
            var event map[string]any
            if err := json.Unmarshal(msg.Body, &event); err != nil {
                logger.Error("failed to unmarshal message", "error", err)
                msg.Nack(false) // Don't requeue invalid messages
                continue
            }

            logger.Info("processing event", "event", event)

            // Process the event...
            msg.Ack() // Acknowledge successful processing

        case err := <-errors:
            logger.Error("queue error", "error", err)
        case <-ctx.Done():
            return
        }
    }
}()
```

## Logging

The application uses structured logging with LogFX:

### Logger Setup

```go
// Create logger with configuration
logger := logfx.NewLoggerAsDefault(os.Stdout, &config.Log)

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
export log__level=debug
export log__pretty=false
export log__add_source=true
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
- **Build Stage**: Optimized binary compilation
- **Production Stage**: Minimal runtime container

### Docker Compose

The `compose.yml` provides:
- **Application**: Hot-reloading development environment
- **Dependencies**: Redis, RabbitMQ for testing cache and queue operations

```bash
# Start all services
docker compose up

# Start specific services
docker compose up redis rabbitmq

# View logs
docker compose logs -f app
```

## Connection Behaviors

The application automatically detects and works with different storage capabilities:

- **Key-Value Storage**: Redis, Memcached
- **Document Storage**: MongoDB, CouchDB
- **Relational Database**: PostgreSQL, MySQL, SQLite
- **Transactional**: Any storage supporting ACID transactions
- **Cache**: Redis, Memcached (with TTL/expiration support)
- **Message Queue**: RabbitMQ, Apache Kafka, AWS SQS

## Error Handling

The application demonstrates proper error handling using ajan's sentinel errors:

```go
import "errors"

// Check for specific errors
if errors.Is(err, datafx.ErrKeyNotFound) {
    // Handle key not found
}

if errors.Is(err, datafx.ErrConnectionNotSupported) {
    // Handle unsupported connection
}

if errors.Is(err, datafx.ErrTransactionFailed) {
    // Handle transaction failure
}

if errors.Is(err, datafx.ErrCacheNotSupported) {
    // Handle cache not supported
}

if errors.Is(err, datafx.ErrQueueNotSupported) {
    // Handle queue not supported
}
```

## Best Practices Demonstrated

1. **Separation of Concerns**: Clean architecture with distinct layers
2. **Configuration Management**: Environment-based configuration with defaults
3. **Error Handling**: Comprehensive error context with sentinel errors
4. **Structured Logging**: Consistent, queryable log output
5. **Metrics Collection**: Observability for production monitoring
6. **Resource Management**: Proper connection lifecycle management
7. **Technology Agnostic**: Same API across different storage types
8. **Type Safety**: Compile-time interface verification

## Production Deployment

### Build Production Image

```bash
# Build optimized production image
docker build --target production -t sample-app:latest .

# Run production container
docker run -p 8080:8080 sample-app:latest
```

### Environment Configuration

```bash
# Production environment variables
export app_env=production
export log__level=warn
export log__pretty=false
export conn__targets__default__dsn="postgres://user:pass@db:5432/app"
export conn__targets__redis_cache__host=redis-cluster
export conn__targets__amqp_queue__host=rabbitmq-cluster
```

## Extending the Sample

To add new functionality:

1. **New Storage Type**: Register additional connection factories in `appcontext.go`
2. **New Operations**: Add business logic functions in `main.go`
3. **Custom Metrics**: Register application-specific metrics collectors
4. **Additional Logging**: Use structured logging throughout business logic

## Learning Resources

- [ajan Documentation](../README.md)
- [datafx Guide](../datafx/README.md)
- [connfx Architecture](../connfx/README.md)
- [Configuration System](../configfx/README.md)
- [Logging Framework](../logfx/README.md)
