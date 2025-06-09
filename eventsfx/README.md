# ajan/eventsfx

## Overview

**eventsfx** package provides an event handling and pub/sub system for Go
applications. It supports synchronous and asynchronous event handling with
configurable timeouts and buffer sizes.

## Configuration

Configuration struct for the event bus:

```go
type Config struct {
  DefaultBufferSize int           `conf:"default_buffer_size" default:"100"`
  ReplyTimeout      time.Duration `conf:"reply_timeout"       default:"5s"`
}
```

Example configuration:

```go
config := &eventsfx.Config{
  DefaultBufferSize: 1000,    // Buffer size for event queue
  ReplyTimeout:     10 * time.Second,  // Timeout for synchronous event replies
}
```

## Features

- Synchronous and asynchronous event handling
- Event buffering with configurable queue size
- Timeout handling for synchronous events
- Support for multiple subscribers per event
- Metrics integration with Prometheus
- Integration with dependency injection

## API

### EventBus

The main component for event handling:

```go
// Create a new event bus
bus := eventsfx.NewEventBus(config, metricsProvider, logger)
if err := bus.Init(); err != nil {
  panic("unable to initialize event bus")
}

// Subscribe to events
bus.Subscribe("user.created", func(event Event) {
  // Handle event
})

// Publish events asynchronously
bus.Publish(Event{
  Name: "user.created",
  Data: userData,
})

// Publish events synchronously with reply
reply, err := bus.PublishSync(Event{
  Name: "user.validate",
  Data: userData,
})
```
