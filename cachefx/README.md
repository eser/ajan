# ajan/configfx

## Overview

The **configfx** package is a flexible caching package that provides a unified interface for different caching backends. Currently, it supports Redis as a caching backend.

The documentation below provides an overview of the package, its types, functions, and usage examples. For more detailed
information, refer to the source code and tests.

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
