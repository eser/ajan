# ajan/connfx

## Overview

**connfx** provides centralized connection management based on connection behaviors rather than specific technologies. It allows adapters to register themselves and define their own supported behaviors, making the system extensible and non-opinionated about specific connection types.

## Features

- **Behavior-Based Design**: Connections are categorized by behavior (stateful, stateless, streaming)
- **Provider-Defined Behaviors**: Adapters determine their own supported behaviors (no user configuration needed)
- **Multiple Behaviors per Provider**: Providers like Redis can support multiple behaviors simultaneously
- **Self-Registering Adapters**: Adapters register themselves with the connection manager
- **Extensible Architecture**: Easy to add new connection types without modifying core code
- **Health Checks**: Built-in health monitoring for all connections
- **Context Support**: Full context.Context support for cancellation and timeouts
- **Shared Connections**: Multiple modules can share the same connection instance
- **Configuration-driven**: Load connections from configuration files
- **Thread Safety**: All operations are thread-safe

## Connection Behaviors

- **Stateful**: Persistent connections that maintain state (databases, connection pools)
- **Stateless**: Connections that don't maintain state (HTTP APIs, REST services)
- **Streaming**: Real-time/streaming connections (message queues, event streams, websockets)

**Note**: Behaviors are determined by the connection provider/adapter, not by user configuration. Some providers like Redis support multiple behaviors simultaneously.

## Quick Start

### Basic Usage with Adapter Registration

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/eser/ajan/connfx"
    "github.com/eser/ajan/connfx/adapters"
    "github.com/eser/ajan/logfx"
    "database/sql"
    "net/http"
)

func main() {
    // Create logger using logfx
    logger, err := logfx.NewLogger(os.Stdout, &logfx.Config{
        Level:      "INFO",
        PrettyMode: true,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Initialize connfx
    connfx.Initialize(logger)

    // Register adapters - they define their own behaviors
    if err := connfx.RegisterAdapter(adapters.NewSQLConnectionFactory("sqlite")); err != nil {
        log.Fatal(err)
    }
    if err := connfx.RegisterAdapter(adapters.NewHTTPConnectionFactory("http")); err != nil {
        log.Fatal(err)
    }

    // Load configuration - no behavior field needed!
    config := &connfx.Config{
        Connections: map[string]connfx.ConnectionConfigData{
            "default": {
                Protocol: "sqlite",
                Database: ":memory:",
            },
            "api": {
                Protocol: "http",
                URL:     "https://api.example.com",
            },
        },
    }

    ctx := context.Background()
    if err := connfx.LoadConfig(ctx, config); err != nil {
        log.Fatal(err)
    }

    // Use connections
    dbConn, err := connfx.GetConnection("default")
    if err != nil {
        log.Fatal(err)
    }

    apiConn, err := connfx.GetConnection("api")
    if err != nil {
        log.Fatal(err)
    }

    // Check behaviors determined by providers
    log.Printf("DB behaviors: %v", dbConn.GetBehaviors())     // [stateful]
    log.Printf("API behaviors: %v", apiConn.GetBehaviors())   // [stateless]

    // Use the connections with type safety
    db, err := connfx.GetTypedConnection[*sql.DB](dbConn)
    if err != nil {
        log.Fatal("Failed to get SQL DB:", err)
    }

    httpClient, err := connfx.GetTypedConnection[*http.Client](apiConn)
    if err != nil {
        log.Fatal("Failed to get HTTP client:", err)
    }

    // Now use the typed connections safely
    _ = db        // *sql.DB
    _ = httpClient // *http.Client
}
```

### Manager Instance (Recommended for DI)

```go
func NewService(logger *logfx.Logger) *Service {
    manager := connfx.NewManager(logger)

    // Register required adapters
    adapters.RegisterSQLiteAdapter(manager)
    adapters.RegisterHTTPAdapter(manager)
    adapters.RegisterRedisAdapter(manager) // Supports both stateful + streaming

    // Load config
    manager.LoadFromConfig(ctx, config)

    return &Service{
        connManager: manager,
    }
}

func (s *Service) DoSomething(ctx context.Context) error {
    conn, err := s.connManager.GetConnectionByProtocol("primary", "postgres")
    if err != nil {
        return err
    }

    // Use connection...
    return nil
}
```

### Creating Custom Adapters

```go
package myadapter

import (
    "context"
    "github.com/eser/ajan/connfx"
)

// CustomConnection implements connfx.Connection
type CustomConnection struct {
    name     string
    protocol string
    // ... your connection-specific fields
}

func (c *CustomConnection) GetName() string { return c.name }
func (c *CustomConnection) GetProtocol() string { return c.protocol }
func (c *CustomConnection) GetBehaviors() []connfx.ConnectionBehavior {
    // Your adapter defines what behaviors it supports
    return []connfx.ConnectionBehavior{
        connfx.ConnectionBehaviorStateful,
        connfx.ConnectionBehaviorStreaming, // Example: supports multiple behaviors
    }
}
// ... implement other Connection interface methods

// CustomConnectionFactory implements connfx.ConnectionFactory
type CustomConnectionFactory struct{}

func (f *CustomConnectionFactory) CreateConnection(ctx context.Context, config connfx.ConnectionConfig) (connfx.Connection, error) {
    // Create your custom connection
}

func (f *CustomConnectionFactory) GetProtocol() string { return "myprotocol" }
func (f *CustomConnectionFactory) GetSupportedBehaviors() []connfx.ConnectionBehavior {
    return []connfx.ConnectionBehavior{
        connfx.ConnectionBehaviorStateful,
        connfx.ConnectionBehaviorStreaming,
    }
}

// Register the adapter
func RegisterCustomAdapter(manager *connfx.Manager) error {
    factory := &CustomConnectionFactory{}
    return manager.RegisterAdapter(factory)
}
```

## Configuration

### Provider-Based Configuration

```yaml
connections:
  primary_db:
    protocol: postgres  # Provider determines this is stateful
    host: localhost
    port: 5432
    database: myapp
    username: user
    password: secret

  cache:
    protocol: redis     # Provider determines this supports stateful + streaming
    host: localhost
    port: 6379

  external_api:
    protocol: http      # Provider determines this is stateless
    url: "https://api.example.com"
    timeout: 30s
    properties:
      headers:
        Authorization: "Bearer TOKEN"

  event_stream:
    protocol: kafka     # Provider determines this is streaming
    host: localhost
    port: 9092
```

### Provider Behavior Examples

- **SQL Databases** (postgres/mysql/sqlite): `[stateful]`
- **HTTP APIs** (http/https/graphql): `[stateless]`
- **Redis**: `[stateful, streaming]` - supports both key-value and pub/sub
- **Message Queues** (kafka/rabbitmq): `[streaming]`

## Working with Connections

### Filtering by Behavior

```go
// Get all stateful connections (databases, Redis for key-value, etc.)
statefulConns := connfx.Default().GetStatefulConnections()

// Get all stateless connections (APIs, etc.)
statelessConns := connfx.Default().GetStatelessConnections()

// Get all streaming connections (queues, Redis for pub/sub, etc.)
streamingConns := connfx.Default().GetStreamingConnections()

// Redis appears in both stateful and streaming lists!
```

### Filtering by Protocol

```go
// Get all PostgreSQL connections
postgresConns := connfx.Default().GetConnectionsByProtocol("postgres")

// Get all Redis connections (support multiple behaviors)
redisConns := connfx.Default().GetConnectionsByProtocol("redis")
for _, conn := range redisConns {
    fmt.Printf("Redis connection supports: %v\n", conn.GetBehaviors())
    // Output: Redis connection supports: [stateful streaming]
}
```

### Type-Safe Retrieval

```go
// Get connection and verify protocol
dbConn, err := connfx.Default().GetConnectionByProtocol("primary", "postgres")
if err != nil {
    return err
}

// Get connection and verify it supports a specific behavior
cacheConn, err := connfx.Default().GetConnectionByBehavior("cache", connfx.ConnectionBehaviorStateful)
if err != nil {
    return err // Will succeed for Redis since it supports stateful
}

// Redis supports streaming too
streamConn, err := connfx.Default().GetConnectionByBehavior("cache", connfx.ConnectionBehaviorStreaming)
if err != nil {
    return err // Will also succeed for Redis
}
```

### Type-Safe Connection Extraction

The `GetTypedConnection` generic function provides type-safe extraction of raw connections without manual type assertions:

```go
import "database/sql"

// Get connection
conn, err := connfx.GetConnection("database")
if err != nil {
    return err
}

// Extract typed connection safely
db, err := connfx.GetTypedConnection[*sql.DB](conn)
if err != nil {
    return fmt.Errorf("failed to get SQL database: %w", err)
}

// Now db is *sql.DB and can be used safely
rows, err := db.QueryContext(ctx, "SELECT * FROM users")
if err != nil {
    return err
}
defer rows.Close()

// Works with any connection type
httpConn, err := connfx.GetConnection("api")
if err != nil {
    return err
}

client, err := connfx.GetTypedConnection[*http.Client](httpConn)
if err != nil {
    return fmt.Errorf("failed to get HTTP client: %w", err)
}

resp, err := client.Get("https://api.example.com/data")
```

### Combined Usage Pattern

```go
// Get and extract in one pattern
func getDatabase(name string) (*sql.DB, error) {
    conn, err := connfx.GetConnection(name)
    if err != nil {
        return nil, err
    }

    return connfx.GetTypedConnection[*sql.DB](conn)
}

// Usage
db, err := getDatabase("primary")
if err != nil {
    return err
}
// db is now *sql.DB
```

## Health Checks

```go
// Check all connections
statuses := connfx.HealthCheck(ctx)
for name, status := range statuses {
    conn, _ := connfx.GetConnection(name)
    fmt.Printf("Connection %s (%s/%v): %s (latency: %v)\n",
        name, conn.GetProtocol(), conn.GetBehaviors(),
        status.State, status.Latency)
}

// Check specific connection
status, err := connfx.Default().HealthCheckNamed(ctx, "primary")
if err != nil {
    log.Printf("Health check failed: %v", err)
} else {
    fmt.Printf("Status: %s, Message: %s\n", status.State, status.Message)
}
```

## Available Adapters

### SQL Databases (Stateful)
- PostgreSQL: `adapters.RegisterPostgreSQLAdapter(manager)` → `[stateful]`
- MySQL: `adapters.RegisterMySQLAdapter(manager)` → `[stateful]`
- SQLite: `adapters.RegisterSQLiteAdapter(manager)` → `[stateful]`

### HTTP APIs (Stateless)
- HTTP: `adapters.RegisterHTTPAdapter(manager)` → `[stateless]`
- HTTPS: `adapters.RegisterHTTPSAdapter(manager)` → `[stateless]`
- GraphQL: `adapters.RegisterGraphQLAdapter(manager)` → `[stateless]`

### Multiple Behavior Providers
- Redis: `adapters.RegisterRedisAdapter(manager)` → `[stateful, streaming]`

## Integration Examples

### datafx Integration

```go
// datafx can get SQL connections by behavior
statefulConns := connfx.GetStatefulConnections()
for _, conn := range statefulConns {
    if conn.GetProtocol() == "postgres" {
        db, err := connfx.GetTypedConnection[*sql.DB](conn)
        if err != nil {
            log.Printf("Failed to get SQL DB: %v", err)
            continue
        }
        // Use db for datafx operations
    }
}
```

### queuefx Integration

```go
// queuefx can get streaming connections
streamingConns := connfx.GetStreamingConnections()
for _, conn := range streamingConns {
    switch conn.GetProtocol() {
    case "redis":
        // Type-safe Redis connection extraction
        client, err := connfx.GetTypedConnection[redis.Client](conn) // Assuming redis.Client type
        if err != nil {
            log.Printf("Failed to get Redis client: %v", err)
            continue
        }
        // Use client for queue operations
    case "kafka":
        // Type-safe Kafka connection extraction
        kafkaConn, err := connfx.GetTypedConnection[kafka.Connection](conn) // Assuming kafka.Connection type
        if err != nil {
            log.Printf("Failed to get Kafka connection: %v", err)
            continue
        }
        // Handle Kafka connections
    }
}
```

### Multi-Purpose Redis Usage

```go
// Use Redis for both caching (stateful) and pub/sub (streaming)
redisConn, err := connfx.GetConnection("cache")
if err != nil {
    return err
}

// Check what behaviors this Redis connection supports
behaviors := redisConn.GetBehaviors()
fmt.Printf("Redis supports: %v\n", behaviors) // [stateful streaming]

// Extract Redis client safely
redisClient, err := connfx.GetTypedConnection[redis.Client](redisConn) // Assuming redis.Client type
if err != nil {
    return fmt.Errorf("failed to get Redis client: %w", err)
}

// Use for caching (stateful behavior)
if hasStateful(behaviors) {
    // Use redisClient for GET/SET operations
    err := redisClient.Set("key", "value")
    if err != nil {
        return err
    }
}

// Use for pub/sub (streaming behavior)
if hasStreaming(behaviors) {
    // Use redisClient for PUBLISH/SUBSCRIBE operations
    err := redisClient.Publish("channel", "message")
    if err != nil {
        return err
    }
}
```

## Error Handling

```go
conn, err := connfx.GetConnection("nonexistent")
if errors.Is(err, connfx.ErrConnectionNotFound) {
    // Handle missing connection
}

// Adapter registration errors
err := manager.RegisterAdapter(factory)
if errors.Is(err, connfx.ErrFactoryAlreadyRegistered) {
    // Handle duplicate registration
}

// Behavior checking
_, err = manager.GetConnectionByBehavior("http_conn", connfx.ConnectionBehaviorStateful)
if err != nil {
    // HTTP connections don't support stateful behavior
}
```

## Best Practices

1. **Register Adapters Early**: Register all needed adapters during application startup
2. **Let Providers Define Behaviors**: Don't specify behaviors in config - let adapters define them
3. **Use Behavior Filtering**: Filter connections by behavior for generic operations
4. **Protocol Validation**: Use protocol-specific getters when you need specific connection types
5. **Type-Safe Extraction**: Use `GetTypedConnection[T]()` instead of manual type assertions for better error handling
6. **Multi-Behavior Awareness**: Remember that some providers (like Redis) support multiple behaviors
7. **Health Monitoring**: Regularly check connection health for monitoring
8. **Graceful Shutdown**: Call `connfx.Default().Close(ctx)` during shutdown
9. **Configuration**: Use configuration files for connection management
10. **Adapter Separation**: Keep adapters in separate packages for modularity

## Architecture Benefits

- **Open/Closed Principle**: Easy to add new connection types without modifying core code
- **Dependency Inversion**: Core module doesn't depend on specific connection implementations
- **Single Responsibility**: Each adapter handles one protocol/technology
- **Provider Autonomy**: Adapters define their own supported behaviors
- **Extensibility**: Anyone can create and register custom adapters with any behavior combination
- **Non-Opinionated**: Core module doesn't make assumptions about specific technologies

## Thread Safety

connfx is fully thread-safe and can be used concurrently from multiple goroutines. All operations on the registry and connections are protected by appropriate synchronization mechanisms.
