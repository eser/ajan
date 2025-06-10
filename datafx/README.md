# ajan/datafx

## Overview

**datafx** package provides a high-level, technology-agnostic data persistence abstraction layer. It sits on top of **connfx** (the connection/adapter layer) and offers simple, consistent data operations that work with any storage technology - whether Redis, PostgreSQL, MongoDB, DynamoDB, or others.

The key principle is **separation of concerns**: connfx handles infrastructure (connections, drivers, protocols) while datafx handles business logic (data operations, transactions, queues). This architecture allows you to write storage-agnostic code that can easily switch between different backends without changing your business logic.

## Architecture

```
datafx (Business Layer)
    ↓ depends on
connfx (Adapter Layer)
    ↓ implements
Storage Technologies (Redis, PostgreSQL, MongoDB, AMQP/RabbitMQ, etc.)
```

## Configuration

datafx depends on connfx for connection management. You configure connections through connfx and then create datafx instances from those connections.

### Connection Configuration (via connfx)

```go
// Redis configuration example
redisConfig := &connfx.ConfigTarget{
    Protocol: "redis",
    Host:     "localhost",
    Port:     6379,
    DSN:      "redis://localhost:6379",
}

// PostgreSQL configuration example
postgresConfig := &connfx.ConfigTarget{
    Protocol: "postgres",
    Host:     "localhost",
    Port:     5432,
    DSN:      "postgres://user:pass@localhost:5432/dbname",
}

// MongoDB configuration example
mongoConfig := &connfx.ConfigTarget{
    Protocol: "mongodb",
    Host:     "localhost",
    Port:     27017,
    DSN:      "mongodb://localhost:27017/mydb",
}

// AMQP/RabbitMQ configuration example
amqpConfig := &connfx.ConfigTarget{
    Protocol: "amqp",
    Host:     "localhost",
    Port:     5672,
    DSN:      "amqp://guest:guest@localhost:5672/",
}
```

### Data Types

datafx works with standard Go types and automatically handles JSON marshaling:

```go
// Example user type
type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Example product type
type Product struct {
    ID       string  `json:"id"`
    Name     string  `json:"name"`
    Price    float64 `json:"price"`
    Category string  `json:"category"`
}

// Example message types for queues
type OrderEvent struct {
    OrderID   string    `json:"order_id"`
    CustomerID string   `json:"customer_id"`
    Amount    float64   `json:"amount"`
    Timestamp time.Time `json:"timestamp"`
}

type NotificationMessage struct {
    UserID  string `json:"user_id"`
    Type    string `json:"type"`
    Content string `json:"content"`
}
```

## Features

- **Technology Agnostic**: Same API works with Redis, SQL databases, document stores, message queues, etc.
- **Automatic Marshaling**: Handles JSON serialization/deserialization transparently
- **Transaction Support**: ACID transactions for compatible storage backends
- **Cache Support**: High-performance caching with TTL/expiration support
- **Queue Support**: Message queue operations with automatic reconnection and acknowledgment
- **Connection Behaviors**: Automatic capability detection (key-value, document, relational, transactional, cache, queue)
- **Type Safety**: Compile-time interface verification and generic type support
- **Error Handling**: Comprehensive error context with sentinel errors
- **Raw Data Support**: Work with `[]byte` directly when needed
- **Extensible**: Easy to add new storage adapters without changing business code

## API

### Basic Usage

```go
import (
    "context"
    "github.com/eser/ajan/connfx"
    "github.com/eser/ajan/connfx/adapters"
    "github.com/eser/ajan/datafx"
    "github.com/eser/ajan/logfx"
)

func main() {
    ctx := context.Background()

    // Setup connection registry
    logger := logfx.NewLogger(os.Stdout, &logfx.Config{Level: logfx.LevelInfo})
    registry := connfx.NewRegistry(logger)

    // Register storage adapter (Redis example)
    redisFactory := adapters.NewRedisFactory()
    registry.RegisterFactory(redisFactory)

    // Configure connection
    config := &connfx.ConfigTarget{
        Protocol: "redis",
        Host:     "localhost",
        Port:     6379,
        DSN:      "redis://localhost:6379",
    }

    // Add connection to registry
    err := registry.AddConnection(ctx, connfx.DefaultConnection, config)
    if err != nil {
        log.Fatal(err)
    }

    // Get connection
    conn := registry.GetDefault()

    // Create datafx instance
    data, err := datafx.New(conn)
    if err != nil {
        log.Fatal(err)
    }

    // Use data operations
    user := &User{
        ID:    "user123",
        Name:  "John Doe",
        Email: "john@example.com",
    }

    // Set data (auto-marshaled to JSON)
    err = data.Set(ctx, "user:123", user)

    // Get data (auto-unmarshaled from JSON)
    var retrievedUser User
    err = data.Get(ctx, "user:123", &retrievedUser)

    // Check existence
    exists, err := data.Exists(ctx, "user:123")

    // Update data
    user.Name = "John Smith"
    err = data.Update(ctx, "user:123", user)

    // Remove data
    err = data.Remove(ctx, "user:123")
}
```

### Core Operations

#### JSON Operations (Recommended)
```go
// Set data - automatically marshaled to JSON
err := data.Set(ctx, "user:123", user)

// Get data - automatically unmarshaled from JSON
var user User
err := data.Get(ctx, "user:123", &user)

// Update existing data
err := data.Update(ctx, "user:123", updatedUser)

// Remove data
err := data.Remove(ctx, "user:123")

// Check if key exists
exists, err := data.Exists(ctx, "user:123")
```

#### Raw Byte Operations
```go
// Set raw bytes
err := data.SetRaw(ctx, "key", []byte("raw data"))

// Get raw bytes
rawData, err := data.GetRaw(ctx, "key")

// Update raw bytes
err := data.UpdateRaw(ctx, "key", []byte("updated raw data"))
```

### Transactional Operations

For storage backends that support transactions:

```go
// Create transactional data instance
txData, err := datafx.NewTransactional(conn)
if err != nil {
    log.Fatal(err)
}

// Execute operations within a transaction
err = txData.ExecuteTransaction(ctx, func(tx *datafx.TransactionData) error {
    // All operations within this function are transactional
    user := &User{ID: "123", Name: "John"}

    if err := tx.Set(ctx, "user:123", user); err != nil {
        return err // Transaction will be rolled back
    }

    product := &Product{ID: "456", Name: "Widget", Price: 19.99}
    if err := tx.Set(ctx, "product:456", product); err != nil {
        return err // Transaction will be rolled back
    }

    return nil // Transaction will be committed
})

if err != nil {
    log.Printf("Transaction failed: %v", err)
}
```

### Working with Multiple Connections

```go
// Add multiple connections
err = registry.AddConnection(ctx, "redis-cache", redisConfig)
err = registry.AddConnection(ctx, "postgres-main", postgresConfig)

// Get specific connections
cacheConn := registry.GetNamed("redis-cache")
dbConn := registry.GetNamed("postgres-main")

// Create separate data instances
cache, _ := datafx.New(cacheConn)
database, _ := datafx.New(dbConn)

// Use them independently
cache.Set(ctx, "session:abc", sessionData)  // Goes to Redis
database.Set(ctx, "user:123", userData)    // Goes to PostgreSQL
```

### Connection Discovery by Behavior

```go
// Find all key-value storage connections
kvConnections := registry.GetByBehavior(connfx.ConnectionBehaviorKeyValue)

// Find all transactional connections
txConnections := registry.GetByBehavior(connfx.ConnectionBehaviorTransactional)

// Find all relational database connections
sqlConnections := registry.GetByBehavior(connfx.ConnectionBehaviorRelational)

// Use the first available key-value store
if len(kvConnections) > 0 {
    cache, _ := datafx.New(kvConnections[0])
    cache.Set(ctx, "temp:data", someData)
}
```

### Cache Operations

For connections that support caching (e.g., Redis):

```go
// Create cache instance
cache, err := datafx.NewCache(conn)
if err != nil {
    log.Fatal(err)
}

// Set with expiration
user := &User{ID: "123", Name: "John"}
err = cache.Set(ctx, "user:123", user, 5*time.Minute)

// Get cached value
var cachedUser User
err = cache.Get(ctx, "user:123", &cachedUser)

// Check TTL
ttl, err := cache.GetTTL(ctx, "user:123")
fmt.Printf("TTL: %v\n", ttl)

// Set expiration on existing key
err = cache.Expire(ctx, "user:123", 10*time.Minute)

// Delete from cache
err = cache.Delete(ctx, "user:123")

// Raw cache operations
err = cache.SetRaw(ctx, "session:abc", []byte("session-data"), time.Hour)
rawData, err := cache.GetRaw(ctx, "session:abc")
```

### Queue Operations

For connections that support message queues (e.g., AMQP/RabbitMQ):

```go
import (
    "context"
    "github.com/eser/ajan/connfx"
    "github.com/eser/ajan/connfx/adapters"
    "github.com/eser/ajan/datafx"
    "github.com/eser/ajan/logfx"
)

func main() {
    ctx := context.Background()

    // Setup connection registry
    logger := logfx.NewLogger(os.Stdout, &logfx.Config{Level: logfx.LevelInfo})
    registry := connfx.NewRegistry(logger)

    // Register AMQP adapter
    amqpFactory := adapters.NewAMQPFactory()
    registry.RegisterFactory(amqpFactory)

    // Configure AMQP connection
    config := &connfx.ConfigTarget{
        Protocol: "amqp",
        Host:     "localhost",
        Port:     5672,
        DSN:      "amqp://guest:guest@localhost:5672/",
    }

    // Add connection to registry
    err := registry.AddConnection(ctx, "message-broker", config)
    if err != nil {
        log.Fatal(err)
    }

    // Get connection
    conn := registry.GetNamed("message-broker")

    // Create queue instance
    queue, err := datafx.NewQueue(conn)
    if err != nil {
        log.Fatal(err)
    }

    // Declare a queue
    queueName, err := queue.DeclareQueue(ctx, "user-events")
    if err != nil {
        log.Fatal(err)
    }

    // Publish messages
    orderEvent := &OrderEvent{
        OrderID:    "order-123",
        CustomerID: "customer-456",
        Amount:     99.99,
        Timestamp:  time.Now(),
    }

    err = queue.Publish(ctx, queueName, orderEvent)
    if err != nil {
        log.Fatal(err)
    }

    // Consume messages with custom configuration
    config := connfx.DefaultConsumerConfig()
    config.AutoAck = false // Manual acknowledgment

    messages, errors := queue.Consume(ctx, queueName, config)

    // Handle messages
    go func() {
        for {
            select {
            case msg := <-messages:
                var event OrderEvent
                if err := json.Unmarshal(msg.Body, &event); err != nil {
                    log.Printf("Failed to unmarshal message: %v", err)
                    msg.Nack(false) // Don't requeue invalid messages
                    continue
                }

                // Process the event
                log.Printf("Processing order: %s for customer: %s",
                    event.OrderID, event.CustomerID)

                // Acknowledge successful processing
                if err := msg.Ack(); err != nil {
                    log.Printf("Failed to ack message: %v", err)
                }

            case err := <-errors:
                log.Printf("Queue error: %v", err)
            case <-ctx.Done():
                return
            }
        }
    }()

    // Or use the convenient ProcessMessages method
    err = queue.ProcessMessages(ctx, queueName, config,
        func(ctx context.Context, message any) bool {
            event := message.(*OrderEvent)
            log.Printf("Processing order: %s", event.OrderID)

            // Return true to acknowledge, false to nack with requeue
            return true
        },
        &OrderEvent{}, // Message type for unmarshalling
    )
}
```

#### Queue Consumer Configuration

```go
// Default configuration
config := connfx.DefaultConsumerConfig()

// Custom configuration
config := connfx.ConsumerConfig{
    AutoAck:   false, // Manual acknowledgment
    Exclusive: true,  // Exclusive access to queue
    NoLocal:   false, // Receive messages from this connection
    NoWait:    false, // Wait for server response
    Args:      nil,   // Additional arguments
}

// Start consuming with configuration
messages, errors := queue.Consume(ctx, "my-queue", config)
```

#### Raw Queue Operations

```go
// Publish raw bytes
rawData := []byte(`{"type": "raw", "data": "some data"}`)
err = queue.PublishRaw(ctx, queueName, rawData)

// Consume with defaults
messages, errors := queue.ConsumeWithDefaults(ctx, queueName)

// Process raw messages
for msg := range messages {
    log.Printf("Received raw message: %s", string(msg.Body))

    // Access headers
    if contentType, ok := msg.Headers["content-type"]; ok {
        log.Printf("Content-Type: %v", contentType)
    }

    // Acknowledge message
    msg.Ack()
}
```

## Connection Behaviors

datafx automatically detects and works with different storage capabilities:

- **ConnectionBehaviorKeyValue**: Redis, Memcached, etc.
- **ConnectionBehaviorDocument**: MongoDB, CouchDB, etc.
- **ConnectionBehaviorRelational**: PostgreSQL, MySQL, SQLite, etc.
- **ConnectionBehaviorTransactional**: Any storage supporting ACID transactions
- **ConnectionBehaviorCache**: Redis, Memcached, etc. (with TTL/expiration support)
- **ConnectionBehaviorQueue**: RabbitMQ, Apache Kafka, AWS SQS, etc.

## Error Handling

datafx uses sentinel errors for consistent error handling:

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

if errors.Is(err, datafx.ErrQueueNotSupported) {
    // Handle queue not supported
}

if errors.Is(err, datafx.ErrMessageProcessing) {
    // Handle message processing failure
}
```

## Extending with New Storage Types

To add support for a new storage technology (e.g., Apache Kafka):

### 1. Implement connfx.Connection Interface
```go
type KafkaConnection struct {
    client KafkaClient
}

func (kc *KafkaConnection) GetBehaviors() []connfx.ConnectionBehavior {
    return []connfx.ConnectionBehavior{
        connfx.ConnectionBehaviorStreaming,
        connfx.ConnectionBehaviorQueue,
    }
}
// ... implement other Connection methods
```

### 2. Implement connfx.QueueRepository Interface
```go
func (kc *KafkaConnection) QueueDeclare(ctx context.Context, name string) (string, error) {
    // Kafka topic creation implementation
}

func (kc *KafkaConnection) Publish(ctx context.Context, queueName string, body []byte) error {
    // Kafka producer implementation
}

func (kc *KafkaConnection) Consume(ctx context.Context, queueName string, config connfx.ConsumerConfig) (<-chan connfx.Message, <-chan error) {
    // Kafka consumer implementation
}
```

### 3. Create Factory and Register
```go
kafkaFactory := adapters.NewKafkaFactory()
registry.RegisterFactory(kafkaFactory)

// Now datafx works with Kafka!
queue, _ := datafx.NewQueue(registry.GetDefault())
queue.Publish(ctx, "my-topic", message) // Publishes to Kafka
```

## Best Practices

1. **Use JSON Operations**: Prefer `Publish()`/`ProcessMessages()` over `PublishRaw()` for automatic marshaling
2. **Handle Errors**: Always check for specific sentinel errors
3. **Use Transactions**: For operations that require consistency across multiple keys
4. **Connection Pooling**: Let connfx handle connection lifecycle and pooling
5. **Behavior-Based Selection**: Use `GetByBehavior()` for flexible connection selection
6. **Queue Management**: Always declare queues before publishing/consuming
7. **Acknowledgment**: Use manual acknowledgment for reliable message processing
8. **Separation of Concerns**: Keep business logic in datafx, infrastructure concerns in connfx

## Benefits

- **Vendor Independence**: Switch storage backends without code changes
- **Consistent API**: Same operations across all storage types
- **Type Safety**: Compile-time verification and generic support
- **Transaction Support**: ACID transactions where available
- **Queue Reliability**: Automatic reconnection and message acknowledgment
- **Easy Testing**: Mock connfx interfaces for unit tests
- **Extensible**: Add new storage types without touching existing code
- **Performance**: Direct adapter implementations without unnecessary layers
