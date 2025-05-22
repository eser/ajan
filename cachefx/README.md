# ajan/cachefx

## Overview

The **cachefx** package is a flexible caching package that provides a unified
interface for different caching backends. Currently, it supports Redis as a
caching backend.

The documentation below provides an overview of the package, its types,
functions, and usage examples. For more detailed information, refer to the
source code and tests.

## Configuration

Configuration struct for the cache:

```go
type Config struct {
  Caches map[string]ConfigCache `conf:"caches"`
}

type ConfigCache struct {
  Provider string `conf:"provider"`
  DSN      string `conf:"dsn"`
}
```

Example configuration:

```go
config := &cachefx.Config{
  Caches: map[string]cachefx.ConfigCache{
    "default": {
      Provider: "redis",
      DSN:      "redis://localhost:6379",
    },
    "session": {
      Provider: "redis",
      DSN:      "redis://localhost:6380",
    },
  },
}
```

## Features

- Redis caching backend support
- Configurable cache dialects
- Registry pattern for managing multiple cache instances
- Easy to extend for additional caching backends

## API

### Usage

```go
import "github.com/eser/ajan/cachefx"

// Create a new Redis cache instance
cache, err := cachefx.NewRedisCache(ctx, cachefx.DialectRedis, "redis://localhost:6379")
if err != nil {
  log.Fatal(err)
}

// Set a value with expiration
err = cache.Set(ctx, "key", "value", time.Hour)

// Get a value
value, err := cache.Get(ctx, "key")

// Delete a value
err = cache.Delete(ctx, "key")
```
