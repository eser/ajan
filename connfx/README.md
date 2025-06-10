# ajan/connfx

## Overview

**connfx** provides centralized connection management based on connection behaviors rather than specific technologies. It allows adapters to register themselves and define their own supported behaviors, making the system extensible and non-opinionated about specific connection types.

## Features

- **Behavior-Based Design**: Connections are categorized by behavior (stateful, stateless, streaming)
- **Provider-Defined Behaviors**: Adapters determine their own supported behaviors (no user configuration needed)
- **Multiple Behaviors per Provider**: Providers like Redis can support multiple behaviors simultaneously
- **Self-Registering Adapters**: Adapters register themselves with the connection registry
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

## Quick Start

### Basic Usage with Registry

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
    logger := logfx.NewLogger(os.Stdout, &logfx.Config{
        Level:      "INFO",
        PrettyMode: true,
    })

    // Create connection registry
    registry := connfx.NewRegistryWithDefaults(logger)

    // Load configuration - no behavior field needed!
    config := &connfx.Config{
        Connections: map[string]connfx.ConfigTarget{
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
    if err := registry.LoadFromConfig(ctx, config); err != nil {
        log.Fatal(err)
    }

    // Use connections
    dbConn := registry.GetNamed("default")
    if dbConn == nil {
        log.Fatal("Database connection not found")
    }

    apiConn := registry.GetNamed("api")
    if apiConn == nil {
        log.Fatal("API connection not found")
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

### Service Integration Pattern

```go
func NewService(logger *logfx.Logger) *Service {
    registry := connfx.NewRegistry(logger)

    // Register required adapters
    registry.RegisterFactory(connfx.NewSQLConnectionFactory("sqlite"))
    registry.RegisterFactory(connfx.NewHTTPConnectionFactory("http"))
    registry.RegisterFactory(connfx.NewRedisConnectionFactory("redis")) // Supports both stateful + streaming

    // Load config
    registry.LoadFromConfig(ctx, config)

    return &Service{
        connRegistry: registry,
    }
}

func (s *Service) DoSomething(ctx context.Context) error {
    conn := s.connRegistry.GetNamed("primary")
    if conn == nil {
        return errors.New("connection not found")
    }

    // Verify protocol if needed
    if conn.GetProtocol() != "postgres" {
        return errors.New("expected postgres connection")
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
func RegisterCustomAdapter(registry *connfx.Registry) {
    factory := &CustomConnectionFactory{}
    registry.RegisterFactory(factory)
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

### Basic Connection Retrieval

```go
// Get default connection
defaultConn := registry.GetDefault()
if defaultConn == nil {
    return errors.New("default connection not available")
}

// Get named connection
dbConn := registry.GetNamed("primary_db")
if dbConn == nil {
    return errors.New("database connection not found")
}

// Check connection properties
fmt.Printf("Protocol: %s, Behaviors: %v\n",
    dbConn.GetProtocol(), dbConn.GetBehaviors())
```

### Filtering by Behavior

```go
// Get all stateful connections (databases, Redis for key-value, etc.)
statefulConns := registry.GetByBehavior(connfx.ConnectionBehaviorStateful)

// Get all stateless connections (APIs, etc.)
statelessConns := registry.GetByBehavior(connfx.ConnectionBehaviorStateless)

// Get all streaming connections (queues, Redis for pub/sub, etc.)
streamingConns := registry.GetByBehavior(connfx.ConnectionBehaviorStreaming)

// Redis appears in both stateful and streaming lists!
for _, conn := range statefulConns {
    fmt.Printf("Stateful connection: %s\n", conn.GetProtocol())
}
```

### Filtering by Protocol

```go
// Get all Postgres connections
postgresConns := registry.GetByProtocol("postgres")

// Get all Redis connections (support multiple behaviors)
redisConns := registry.GetByProtocol("redis")
for _, conn := range redisConns {
    fmt.Printf("Redis connection supports: %v\n", conn.GetBehaviors())
    // Output: Redis connection supports: [stateful streaming]
}
```

### Type-Safe Connection Extraction

The `GetTypedConnection` generic function provides type-safe extraction of raw connections without manual type assertions:

```go
import "database/sql"

// Get connection
conn := registry.GetNamed("database")
if conn == nil {
    return errors.New("database connection not found")
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
httpConn := registry.GetNamed("api")
if httpConn == nil {
    return errors.New("API connection not found")
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
func getDatabase(registry *connfx.Registry, name string) (*sql.DB, error) {
    conn := registry.GetNamed(name)
    if conn == nil {
        return nil, fmt.Errorf("connection %q not found", name)
    }

    return connfx.GetTypedConnection[*sql.DB](conn)
}

// Usage
db, err := getDatabase(registry, "primary")
if err != nil {
    return err
}
// db is now *sql.DB
```

## Health Checks

```go
// Check all connections
statuses := registry.HealthCheck(ctx)
for name, status := range statuses {
    conn := registry.GetNamed(name)
    if conn != nil {
        fmt.Printf("Connection %s (%s/%v): %s (latency: %v)\n",
            name, conn.GetProtocol(), conn.GetBehaviors(),
            status.State, status.Latency)
    }
}

// Check specific connection
status, err := registry.HealthCheckNamed(ctx, "primary")
if err != nil {
    log.Printf("Health check failed: %v", err)
} else {
    fmt.Printf("Status: %s, Message: %s\n", status.State, status.Message)
}
```

## Available Adapters

### SQL Databases (Stateful)
- Postgres: `registry.RegisterFactory(postgresFactory)` → `[stateful]`
- MySQL: `registry.RegisterFactory(mysqlFactory)` → `[stateful]`
- SQLite: `registry.RegisterFactory(sqliteFactory)` → `[stateful]`

### HTTP APIs (Stateless)
- HTTP: `registry.RegisterFactory(httpFactory)` → `[stateless]`
- GraphQL: `registry.RegisterFactory(graphqlFactory)` → `[stateless]`

### Multiple Behavior Providers
- Redis: `registry.RegisterFactory(redisFactory)` → `[stateful, streaming]`

## Integration Examples

### datafx Integration

```go
// datafx can get SQL connections by behavior
statefulConns := registry.GetByBehavior(connfx.ConnectionBehaviorStateful)
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
streamingConns := registry.GetByBehavior(connfx.ConnectionBehaviorStreaming)
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
redisConn := registry.GetNamed("cache")
if redisConn == nil {
    return errors.New("cache connection not found")
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

## API Reference

### Registry

```go
type Registry struct {
    // ... internal fields
}

// Core methods for connection retrieval
func (r *Registry) GetDefault() Connection
func (r *Registry) GetNamed(name string) Connection

// Behavior and protocol filtering
func (r *Registry) GetByBehavior(behavior ConnectionBehavior) []Connection
func (r *Registry) GetByProtocol(protocol string) []Connection

// Connection management
func (r *Registry) AddConnection(ctx context.Context, config *ConnectionConfig) error
func (r *Registry) RemoveConnection(ctx context.Context, name string) error
func (r *Registry) LoadFromConfig(ctx context.Context, config *Config) error

// Health monitoring
func (r *Registry) HealthCheck(ctx context.Context) map[string]*HealthStatus
func (r *Registry) HealthCheckNamed(ctx context.Context, name string) (*HealthStatus, error)

// Administrative methods
func (r *Registry) ListConnections() []string
func (r *Registry) ListRegisteredProtocols() []string
func (r *Registry) Close(ctx context.Context) error

// Adapter registration
func (r *Registry) RegisterFactory(factory ConnectionFactory) error
```

### Connection Interface

```go
type Connection interface {
    GetBehaviors() []ConnectionBehavior
    GetProtocol() string
    GetState() ConnectionState
    HealthCheck(ctx context.Context) *HealthStatus
    Close(ctx context.Context) error
    GetRawConnection() any
}
```

### Type-Safe Connection Extraction

```go
func GetTypedConnection[T any](conn Connection) (T, error)
```

## Error Handling

```go
// Check for nil connections
conn := registry.GetNamed("nonexistent")
if conn == nil {
    // Handle missing connection
    return errors.New("connection not found")
}

// Type extraction errors
db, err := connfx.GetTypedConnection[*sql.DB](conn)
if errors.Is(err, connfx.ErrInvalidType) {
    // Handle type mismatch
}
```

## Best Practices

1. **Register Adapters Early**: Register all needed adapters during application startup
2. **Let Providers Define Behaviors**: Don't specify behaviors in config - let adapters define them
3. **Use Behavior Filtering**: Filter connections by behavior for generic operations
4. **Check for Nil**: Always check if `GetNamed()` returns nil before using connections
5. **Type-Safe Extraction**: Use `GetTypedConnection[T]()` instead of manual type assertions for better error handling
6. **Multi-Behavior Awareness**: Remember that some providers (like Redis) support multiple behaviors
7. **Health Monitoring**: Regularly check connection health for monitoring
8. **Graceful Shutdown**: Call `registry.Close(ctx)` during shutdown
9. **Configuration**: Use configuration files for connection management
10. **Adapter Separation**: Keep adapters in separate packages for modularity

## Architecture Benefits

- **Open/Closed Principle**: Easy to add new connection types without modifying core code
- **Dependency Inversion**: Core module doesn't depend on specific connection implementations
- **Single Responsibility**: Each adapter handles one protocol/technology
- **Provider Autonomy**: Adapters define their own supported behaviors
- **Extensibility**: Anyone can create and register custom adapters with any behavior combination
- **Non-Opinionated**: Core module doesn't make assumptions about specific technologies
- **Simplified API**: Focus on essential operations with clear, predictable behavior

## Thread Safety

connfx is fully thread-safe and can be used concurrently from multiple goroutines. All operations on the registry and connections are protected by appropriate synchronization mechanisms.
