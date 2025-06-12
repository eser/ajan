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
    logger := logfx.NewLoggerWithDefaults()

    // Create connection registry
    registry := connfx.NewRegistryWithDefaults(logger)

    // Load configuration
    config := &connfx.Config{
        Connections: map[string]connfx.ConfigTarget{
            "default": {
                Protocol: "sqlite",
                DSN:      ":memory:",
            },
            "cache": {
                Protocol: "redis",
                DSN:     "redis://localhost:6379",
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
    dbConn := registry.GetDefault()
    if dbConn == nil {
        log.Fatal("Database connection not found")
    }

    cacheConn := registry.GetNamed("cache")
    if cacheConn == nil {
        log.Fatal("Cache connection not found")
    }

    apiConn := registry.GetNamed("api")
    if apiConn == nil {
        log.Fatal("API connection not found")
    }

    // Check behaviors determined by providers
    log.Printf("DB behaviors: %v", dbConn.GetBehaviors())     // [stateful]
    log.Printf("Cache behaviors: %v", dbConn.GetBehaviors())  // [stateful streaming]
    log.Printf("API behaviors: %v", apiConn.GetBehaviors())   // [stateless]

    // Check capabilities determined by providers
    log.Printf("DB capabilities: %v", dbConn.GetCapabilities())         // [transactional relational]
    log.Printf("Cache capabilities: %v", cacheConn.GetCapabilities())   // [key-value cache queue]
    log.Printf("API capabilities: %v", apiConn.GetBehaviors())          // []

    // Use the connections with type safety
    db, err := connfx.GetTypedConnection[*sql.DB](dbConn)
    if err != nil {
        log.Fatal("Failed to get SQL DB:", err)
    }

    cache, err := connfx.GetTypedConnection[*redis.Client](cacheConn)
    if err != nil {
        log.Fatal("Failed to get Redis Client:", err)
    }

    httpClient, err := connfx.GetTypedConnection[*http.Client](apiConn)
    if err != nil {
        log.Fatal("Failed to get HTTP client:", err)
    }

    // Now use the typed connections safely
    _ = db         // *sql.DB
    _ = cache      // *redis.Client
    _ = httpClient // *http.Client
}
```

### Service Integration Pattern

```go
func NewService(logger *logfx.Logger) *Service {
    registry := connfx.NewRegistry(logger)

    // Register required adapters
    registry.RegisterFactory(connfx.NewSQLConnectionFactory("sqlite"))
    registry.RegisterFactory(connfx.NewRedisConnectionFactory("redis")) // Supports both stateful + streaming
    registry.RegisterFactory(connfx.NewHTTPConnectionFactory("http"))

    // Load config
    registry.LoadFromConfig(ctx, config)

    return &Service{
        connRegistry: registry,
    }
}

func (s *Service) DoSomething(ctx context.Context) error {
    conn := s.connRegistry.GetDefault()
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

func (c *CustomConnection) GetCapabilities() []connfx.ConnectionCapability {
    // Your adapter defines what capabilities it supports
    return []connfx.ConnectionCapability{
        connfx.ConnectionCapabilityRelational,
        connfx.ConnectionCapabilityTransactional, // Example: supports multiple capabilities
    }
}
// ... implement other Connection interface methods

// CustomConnectionFactory implements connfx.ConnectionFactory
type CustomConnectionFactory struct{}

func (f *CustomConnectionFactory) CreateConnection(ctx context.Context, config connfx.ConnectionConfig) (connfx.Connection, error) {
    // Create your custom connection
}

func (f *CustomConnectionFactory) GetProtocol() string { return "myprotocol" }

// Register the adapter
func RegisterCustomAdapter(registry *connfx.Registry) {
    factory := &CustomConnectionFactory{}
    registry.RegisterFactory(factory)
}
```

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
fmt.Printf("Protocol: %s, Behaviors: %v, Capabilities: %v\n",
    dbConn.GetProtocol(), dbConn.GetBehaviors(), dbConn.GetCapabilities())
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

### Filtering by Capabilities

```go
// Get all relational connections (databases, etc.)
relationalConns := registry.GetByCapability(connfx.ConnectionCapabilityRelational)

// Get all transactional connections (databases, etc.)
transactionalConns := registry.GetByCapability(connfx.ConnectionCapabilityTransactional)

// Get all cache connections (redis, etc.)
cacheConns := registry.GetByCapability(connfx.ConnectionCapabilityCache)

// Get all queue connections (amqp, etc.)
queueConns := registry.GetByCapability(connfx.ConnectionCapabilityQueue)

// Get all key-value connections (redis, etc.)
keyValueConns := registry.GetByCapability(connfx.ConnectionCapabilityKeyValue)

// Redis appears in cache, queue and key-value lists!
for _, conn := range queueConns {
    fmt.Printf("Queue connection: %s\n", conn.GetProtocol())
}
```

### Filtering by Protocol

```go
// Get all Postgres connections
postgresConns := registry.GetByProtocol("postgres")

// Get all Redis connections (support multiple behaviors and capabilities)
redisConns := registry.GetByProtocol("redis")
for _, conn := range redisConns {
    fmt.Printf("Redis connection supports: %v and %v\n", conn.GetBehaviors(), conn.GetCapabilities())
    // Output: Redis connection supports: [stateful streaming] and [cache queue key-value]
}
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
conn := registry.GetNamed("primary")
status := conn.HealthCheck(ctx)
fmt.Printf("Status: %s, Message: %s\n", status.State, status.Message)
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
func (r *Registry) AddConnection(ctx context.Context, config *ConnectionConfig) (Connection, error)
func (r *Registry) RemoveConnection(ctx context.Context, name string) error
func (r *Registry) LoadFromConfig(ctx context.Context, config *Config) error

// Health monitoring
func (r *Registry) HealthCheck(ctx context.Context) map[string]*HealthStatus

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
    GetCapabilities() []ConnectionCapability
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
