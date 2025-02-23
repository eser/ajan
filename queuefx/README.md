# ajan/queuefx

## Overview

The **queuefx** package is a flexible message queue package that provides a
unified interface for different message queue backends. Currently, it supports
RabbitMQ (AMQP) as a message queue backend.

The documentation below provides an overview of the package, its types,
functions, and usage examples. For more detailed information, refer to the
source code and tests.

## Configuration

Configuration struct for the message queue:

```go
type Config struct {
  Brokers map[string]ConfigBroker `conf:"BROKERS"`
}

type ConfigBroker struct {
  Provider string `conf:"PROVIDER"`
  DSN      string `conf:"DSN"`
}

// Consumer configuration
type ConsumerConfig struct {
  Args      map[string]any // Additional arguments for declaration
  AutoAck   bool          // Automatic message acknowledgment
  Exclusive bool          // Exclusive queue access
  NoLocal   bool          // Don't receive messages published by this connection
  NoWait    bool          // Don't wait for server confirmation
}
```

Example configuration:

```go
config := &queuefx.Config{
  Brokers: map[string]queuefx.ConfigBroker{
    "default": {
      Provider: "amqp",
      DSN:      "amqp://guest:guest@localhost:5672",
    },
    "events": {
      Provider: "amqp",
      DSN:      "amqp://user:pass@events:5672",
    },
  },
}

// Consumer configuration
consumerConfig := queuefx.DefaultConsumerConfig() // Get defaults
consumerConfig.AutoAck = true                     // Customize as needed
```

## Features

- RabbitMQ (AMQP) message queue backend support
- Configurable queue dialects
- Automatic reconnection handling
- Message acknowledgment control
- Flexible consumer configuration
- Easy to extend for additional message queue backends

## API

### Usage

```go
import "github.com/eser/ajan/queuefx"

// Create a new AMQP broker instance
broker, err := queuefx.NewAmqpBroker(ctx, queuefx.DialectAmqp, "amqp://localhost:5672")
if err != nil {
  log.Fatal(err)
}

// Declare a queue
queueName, err := broker.QueueDeclare(ctx, "my-queue")
if err != nil {
  log.Fatal(err)
}

// Publish a message
err = broker.Publish(ctx, queueName, []byte("Hello, World!"))
if err != nil {
  log.Fatal(err)
}

// Configure consumer
config := queuefx.DefaultConsumerConfig()
config.AutoAck = false

// Start consuming messages
messages, errors := broker.Consume(ctx, queueName, config)

// Handle messages and errors
go func() {
  for {
    select {
    case msg := <-messages:
      fmt.Printf("Received: %s\n", string(msg.Body))
      msg.Ack() // Acknowledge the message
    case err := <-errors:
      fmt.Printf("Error: %v\n", err)
    case <-ctx.Done():
      return
    }
  }
}()
```

### Consumer Configuration

The package provides flexible consumer configuration through the
`ConsumerConfig` struct:

```go
type ConsumerConfig struct {
  Args      map[string]any // Additional arguments for declaration
  AutoAck   bool          // Automatic message acknowledgment
  Exclusive bool          // Exclusive queue access
  NoLocal   bool          // Don't receive messages published by this connection
  NoWait    bool          // Don't wait for server confirmation
}
```

You can use `DefaultConsumerConfig()` to get started with sensible defaults and
customize as needed.
