# ajan/httpclient

## Overview

**httpclient** is a resilient HTTP client that is 100% compatible with the
standard `net/http` interfaces while providing additional features for
improved reliability and fault tolerance.

## Features

- Drop-in replacement for `net/http.Client`
- Circuit breaker pattern implementation
- Exponential backoff retry mechanism with jitter
- Context-aware request handling
- Configurable failure thresholds and timeouts
- Support for HTTP request body retries (when `GetBody` is implemented)

## Usage

### Basic Usage

```go
// Create a client with default settings
client := httpclient.DefaultClient()

// Make requests as you would with http.Client
resp, err := client.Get("https://api.example.com")
if err != nil {
  // Handle error
}
defer resp.Body.Close()
```

### Custom Configuration

```go
// Configure circuit breaker
cb := httpclient.NewCircuitBreaker(
  5,              // Failure threshold
  10*time.Second, // Reset timeout
  2,              // Half-open success needed
)

// Configure retry strategy
rs := httpclient.NewRetryStrategy(
  3,                    // Max attempts
  100*time.Millisecond, // Initial interval
  10*time.Second,       // Max interval
  2.0,                  // Multiplier
  0.1,                  // Random factor
)

// Create client with custom settings
client := httpclient.NewClient(cb, rs)
```

### Context Support

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.example.com", nil)
if err != nil {
  // Handle error
}

resp, err := client.Do(req)
if err != nil {
  // Handle error
}
defer resp.Body.Close()
```

## Circuit Breaker States

The circuit breaker has three states:

1. **Closed** (default): Requests flow normally
2. **Open**: Requests are immediately rejected
3. **Half-Open**: Limited requests are allowed to test the service

## Retry Strategy

The retry mechanism implements exponential backoff with optional jitter:

- Initial retry interval grows exponentially with each attempt
- Random jitter helps prevent thundering herd problems
- Maximum interval caps the exponential growth
- Configurable maximum number of attempts

## Error Handling

The client provides specific error types for different failure scenarios:

- `ErrCircuitOpen`: When the circuit breaker is open
- `ErrMaxRetries`: When maximum retry attempts are exceeded
- `ErrRequestBodyNotRetriable`: When request body cannot be retried

## Best Practices

1. Always use context for request timeouts
2. Close response bodies
3. Implement `GetBody` for POST/PUT requests that need retry support
4. Configure circuit breaker thresholds based on your service's characteristics
5. Use appropriate retry settings to avoid overwhelming downstream services

## Thread Safety

The client is safe for concurrent use by multiple goroutines.
