# ajan/cachefx

## Overview

**cachefx** is a comprehensive caching framework for Go applications that provides a unified interface for different cache backends with connection management, configuration, and observability features.

## Features

- **Unified Cache Interface**: Common API for different cache backends
- **Redis Support**: Built-in Redis implementation with connection pooling
- **Connection Registry**: Centralized management of multiple cache connections
- **Configuration-Driven**: Easy setup through configuration files
- **Error Handling**: Comprehensive error handling with sentinel errors
- **Logging Integration**: Built-in logging with structured output

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "time"

    "github.com/eser/ajan/cachefx"
    "github.com/eser/ajan/logfx"
)

func main() {
    logger := logfx.NewLogger(os.Stdout, &logfx.Config{Level: "info"})

    // Create a cache registry
    registry := cachefx.NewRegistry(logger)

    // Add a Redis cache connection
    ctx := context.Background()
    err := registry.AddConnection(ctx, "default", "redis", "redis://localhost:6379")
    if err != nil {
        log.Fatal(err)
    }

    // Get the cache instance
    cache := registry.GetDefault()

    // Set a value with expiration
    err = cache.Set(ctx, "user:123", "john_doe", 5*time.Minute)
    if err != nil {
        log.Fatal(err)
    }

    // Get a value
    value, err := cache.Get(ctx, "user:123")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("User: %s\n", value)

    // Delete a key
    err = cache.Delete(ctx, "user:123")
    if err != nil {
        log.Fatal(err)
    }
}
```

### Configuration-Based Setup

```go
import "github.com/eser/ajan/cachefx"

// Define configuration
config := &cachefx.Config{
    Caches: map[string]cachefx.ConfigCache{
        "default": {
            Provider: "redis",
            DSN:      "redis://localhost:6379/0",
        },
        "sessions": {
            Provider: "redis",
            DSN:      "redis://localhost:6379/1",
        },
    },
}

// Load configuration
registry := cachefx.NewRegistry(logger)
err := registry.LoadFromConfig(ctx, config)
if err != nil {
    log.Fatal(err)
}

// Use named caches
defaultCache := registry.GetDefault()
sessionCache := registry.GetNamed("sessions")
```

## API Reference

### Registry

The `Registry` manages multiple cache connections and provides centralized access.

#### Creating a Registry

```go
func NewRegistry(logger *logfx.Logger) *Registry
```

#### Adding Connections

```go
func (registry *Registry) AddConnection(
    ctx context.Context,
    name string,
    provider string,
    dsn string,
) error
```

Supported providers:
- `redis`: Redis cache backend

DSN formats:
- Redis: `redis://localhost:6379/0`, `redis://user:pass@localhost:6379/0`

#### Getting Cache Instances

```go
// Get the default cache (named "default")
func (registry *Registry) GetDefault() Cache

// Get a named cache
func (registry *Registry) GetNamed(name string) Cache
```

#### Configuration Loading

```go
func (registry *Registry) LoadFromConfig(ctx context.Context, config *Config) error
```

### Cache Interface

All cache implementations provide a common interface:

```go
type Cache interface {
    GetDialect() Dialect
    Set(ctx context.Context, key string, value any, expiration time.Duration) error
    Get(ctx context.Context, key string) (string, error)
    Delete(ctx context.Context, key string) error
}
```

#### Methods

**`Set(ctx, key, value, expiration)`**
- Stores a value with optional expiration
- `expiration`: Use `0` for no expiration, or `time.Duration` for TTL

**`Get(ctx, key)`**
- Retrieves a value by key
- Returns empty string if key doesn't exist (no error)

**`Delete(ctx, key)`**
- Removes a key from the cache

**`GetDialect()`**
- Returns the cache backend type

## Configuration

### Structure

```go
type Config struct {
    Caches map[string]ConfigCache `conf:"caches"`
}

type ConfigCache struct {
    Provider string `conf:"provider"`
    DSN      string `conf:"dsn"`
}
```

### Example Configuration File

**YAML:**
```yaml
caches:
  default:
    provider: redis
    dsn: redis://localhost:6379/0
  sessions:
    provider: redis
    dsn: redis://localhost:6379/1
  temp:
    provider: redis
    dsn: redis://cache-server:6379/2
```

**JSON:**
```json
{
  "caches": {
    "default": {
      "provider": "redis",
      "dsn": "redis://localhost:6379/0"
    },
    "sessions": {
      "provider": "redis",
      "dsn": "redis://localhost:6379/1"
    }
  }
}
```

## Error Handling

cachefx uses sentinel errors for consistent error handling:

```go
import "errors"

// Check for specific errors
if errors.Is(err, cachefx.ErrFailedToAddConnection) {
    // Handle connection error
}

if errors.Is(err, cachefx.ErrFailedToSetCacheKey) {
    // Handle set operation error
}
```

### Available Errors

- `ErrFailedToDetermineDialect`: Unknown or unsupported provider
- `ErrFailedToAddConnection`: Connection setup failed
- `ErrFailedToParseRedisURL`: Invalid Redis URL format
- `ErrFailedToConnectToRedis`: Redis connection failed
- `ErrFailedToSetCacheKey`: Set operation failed
- `ErrFailedToGetCacheKey`: Get operation failed
- `ErrFailedToDeleteCacheKey`: Delete operation failed
- `ErrUnknownProvider`: Unsupported cache provider
- `ErrUnableToDetermineDialect`: Cannot auto-detect provider from DSN

## Best Practices

### 1. Connection Management

```go
// Create registry once, reuse throughout application
registry := cachefx.NewRegistry(logger)

// Add connections during application startup
err := registry.LoadFromConfig(ctx, config)
if err != nil {
    log.Fatal("Failed to setup cache connections:", err)
}
```

### 2. Error Handling

```go
// Always handle cache errors gracefully
value, err := cache.Get(ctx, key)
if err != nil {
    logger.Error("Cache get failed", slog.String("key", key), slog.String("error", err.Error()))
    // Fallback to other data source
    return getFromDatabase(ctx, key)
}
```

### 3. Expiration Strategies

```go
// Short-lived data
cache.Set(ctx, "temp:token", token, 15*time.Minute)

// Session data
cache.Set(ctx, "session:"+sessionID, data, 24*time.Hour)

// Permanent cache (manual invalidation)
cache.Set(ctx, "config:app", config, 0)
```

### 4. Key Naming Conventions

```go
// Use prefixes for different data types
userKey := fmt.Sprintf("user:%d", userID)
sessionKey := fmt.Sprintf("session:%s", sessionID)
configKey := fmt.Sprintf("config:%s", configName)
```

## Integration with Other Modules

cachefx integrates seamlessly with other ajan modules:

### With configfx

```go
import (
    "github.com/eser/ajan/configfx"
    "github.com/eser/ajan/cachefx"
)

type AppConfig struct {
    Cache cachefx.Config `conf:"cache"`
}

config := &AppConfig{}
err := configfx.Load(config)
if err != nil {
    log.Fatal(err)
}

registry := cachefx.NewRegistry(logger)
err = registry.LoadFromConfig(ctx, &config.Cache)
```

### With LogFX

```go
// cachefx automatically uses structured logging
registry := cachefx.NewRegistry(logger)

// All operations are logged with context
cache.Set(ctx, "key", "value", time.Hour)
// Logs: level=INFO msg="cache operation" operation=set key=key
```

## Advanced Usage

### Multiple Cache Backends

```go
// Setup different caches for different purposes
config := &cachefx.Config{
    Caches: map[string]cachefx.ConfigCache{
        "fast": {
            Provider: "redis",
            DSN:      "redis://localhost:6379/0", // Local Redis
        },
        "persistent": {
            Provider: "redis",
            DSN:      "redis://redis-cluster:6379/0", // Persistent Redis
        },
    },
}

registry.LoadFromConfig(ctx, config)

// Use appropriate cache for each use case
fastCache := registry.GetNamed("fast")       // For temporary data
persistentCache := registry.GetNamed("persistent") // For important data
```

### Custom Cache Implementation

```go
// Implement the Cache interface for custom backends
type MyCustomCache struct {
    // Your implementation
}

func (c *MyCustomCache) GetDialect() cachefx.Dialect {
    return "custom"
}

func (c *MyCustomCache) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
    // Your implementation
    return nil
}

// ... implement other methods
```

## Dependencies

- `github.com/redis/go-redis/v9`: Redis client
- `github.com/eser/ajan/logfx`: Logging framework

## Thread Safety

All cache implementations are thread-safe and can be used concurrently from multiple goroutines.
