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
// Redis/Valkey configuration example
redisConfig := &connfx.ConfigTarget{
    Protocol: "redis",
    DSN:      "redis://localhost:6379",
}

// PostgreSQL configuration example
postgresConfig := &connfx.ConfigTarget{
    Protocol: "postgres",
    DSN:      "postgres://user:pass@localhost:5432/dbname",
}

// MongoDB configuration example
mongoConfig := &connfx.ConfigTarget{
    Protocol: "mongodb",
    DSN:      "mongodb://localhost:27017/mydb",
}

// AMQP/RabbitMQ configuration example
amqpConfig := &connfx.ConfigTarget{
    Protocol: "amqp",
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
    OrderID    string    `json:"order_id"`
    CustomerID string    `json:"customer_id"`
    Amount     float64   `json:"amount"`
    Timestamp  time.Time `json:"timestamp"`
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
    logger := logfx.NewLoggerWithDefaults()
    registry := connfx.NewRegistryWithDefaults(logger)

    // Configure connection
    config := &connfx.ConfigTarget{
        Protocol: "redis",
        DSN:      "redis://localhost:6379",
    }

    // Add connection to registry
    conn, err := registry.AddConnection(ctx, connfx.DefaultConnection, config)
    if err != nil {
        log.Fatal(err)
    }

    // Create datafx.Store instance
    data, err := datafx.NewStore(conn)
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
// Create transactional store instance
txData, err := datafx.NewTransactionalStore(conn)
if err != nil {
    log.Fatal(err)
}

// Execute operations within a transaction
err = txData.ExecuteTransaction(ctx, func(tx *datafx.TransactionStore) error {
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
cacheConn, err = registry.AddConnection(ctx, "redis-cache", redisConfig)
dbConn, err = registry.AddConnection(ctx, "postgres-main", postgresConfig)

// Create separate data instances
cache, _ := datafx.NewCache(cacheConn)
database, _ := datafx.NewStore(dbConn)

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
    cache, _ := datafx.NewStore(kvConnections[0])
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

For connections that support message queues (e.g., AMQP/RabbitMQ, Redis Streams):

```go
import (
    "context"
    "github.com/eser/ajan/connfx"
    "github.com/eser/ajan/connfx/adapters"
    "github.com/eser/ajan/datafx"
    "github.com/eser/ajan/logfx"
    "time"
)

func main() {
    ctx := context.Background()

    // Setup connection registry
    logger := logfx.NewLoggerWithDefaults()
    registry := connfx.NewRegistryWithDefaults(logger)

    // Configure AMQP connection
    config := &connfx.ConfigTarget{
        Protocol: "amqp",
        Host:     "localhost",
        Port:     5672,
        DSN:      "amqp://guest:guest@localhost:5672/",
    }

    // Add connection to registry
    conn, err := registry.AddConnection(ctx, "message-broker", config)
    if err != nil {
        log.Fatal(err)
    }

    // Create queue instance
    queue, err := datafx.NewQueue(conn)
    if err != nil {
        log.Fatal(err)
    }

    // Declare a queue with configuration
    queueConfig := connfx.DefaultQueueConfig()
    queueConfig.Durable = true
    queueConfig.MaxLength = 1000
    queueConfig.MessageTTL = 24 * time.Hour

    queueName, err := queue.DeclareQueueWithConfig(ctx, "user-events", queueConfig)
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

    // Publish with headers
    headers := map[string]any{
        "content-type": "application/json",
        "priority":     "high",
        "source":       "order-service",
    }

    err = queue.PublishWithHeaders(ctx, queueName, orderEvent, headers)
    if err != nil {
        log.Fatal(err)
    }

    // Configure consumer
    consumerConfig := connfx.DefaultConsumerConfig()
    consumerConfig.AutoAck = false // Manual acknowledgment
    consumerConfig.PrefetchCount = 10
    consumerConfig.MaxRetries = 3

    // Consumer group processing (for Redis Streams)
    if queue.IsStreamSupported() {
        messages, errors := queue.ConsumeWithGroup(ctx, queueName, "order-processors", "worker-1", consumerConfig)

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
    }

    // Standard queue consumption
    messages, errors := queue.Consume(ctx, queueName, consumerConfig)

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
    err = queue.ProcessMessages(ctx, queueName, consumerConfig,
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

#### Consumer Group Processing with Retry Logic

For advanced queue systems like Redis Streams that support consumer groups:

```go
func ProcessMessagesWithRetry(ctx context.Context, queue *datafx.Queue) error {
    queueName := "orders-stream"
    consumerGroup := "order-processors"
    consumerName := "worker-1"

    // Configure consumer with retry settings
    config := connfx.DefaultConsumerConfig()
    config.AutoAck = false
    config.PrefetchCount = 5
    config.BlockTimeout = 2 * time.Second
    config.MaxRetries = 3
    config.RetryDelay = 1 * time.Second

    // Create consumer group if using streams
    if queue.IsStreamSupported() {
        streamRepo, err := queue.GetStreamRepository()
        if err != nil {
            return err
        }

        // Create consumer group (starts from latest messages)
        err = streamRepo.CreateConsumerGroup(ctx, queueName, consumerGroup, "$")
        if err != nil {
            log.Printf("Consumer group might already exist: %v", err)
        }
    }

    // Background goroutine for claiming pending messages
    go func() {
        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                // Claim messages that have been pending for more than 1 minute
                pendingMessages, err := queue.ClaimPendingMessages(
                    ctx, queueName, consumerGroup, consumerName,
                    1*time.Minute, 10,
                )
                if err != nil {
                    log.Printf("Failed to claim pending messages: %v", err)
                    continue
                }

                log.Printf("Claimed %d pending messages for retry", len(pendingMessages))

                // Process claimed messages
                for _, msg := range pendingMessages {
                    // Process and acknowledge the message
                    if processMessage(msg) {
                        msg.Ack()
                    } else {
                        msg.Nack(true) // Requeue for retry
                    }
                }
            }
        }
    }()

    // Main message processing loop
    return queue.ProcessMessagesWithGroup(
        ctx, queueName, consumerGroup, consumerName, config,
        func(ctx context.Context, message any) bool {
            return processMessage(message)
        },
        &OrderEvent{},
    )
}

func processMessage(message any) bool {
    // Simulate message processing with potential failures
    event, ok := message.(*OrderEvent)
    if !ok {
        log.Printf("Invalid message type")
        return false // Don't requeue malformed messages
    }

    log.Printf("Processing order: %s", event.OrderID)

    // Simulate processing logic
    if event.Amount > 1000 {
        log.Printf("High-value order requires manual review: %s", event.OrderID)
        return false // Nack - will be retried
    }

    // Process successfully
    log.Printf("Order processed successfully: %s", event.OrderID)
    return true
}
```

#### Queue Stream Operations

For advanced streaming capabilities (Redis Streams, Kafka):

```go
func StreamOperationsExample(ctx context.Context) error {
    // Setup Redis connection for streams
    redisConfig := connfx.NewDefaultRedisConfig()
    redisConfig.Address = "localhost:6379"
    redisConn := connfx.NewRedisConnection("redis", redisConfig)

    // Create queue stream instance
    stream, err := datafx.NewQueueStream(redisConn)
    if err != nil {
        return fmt.Errorf("failed to create queue stream: %w", err)
    }

    streamName := "user-activity"
    consumerGroup := "analytics-group"

    // Create consumer group
    err = stream.CreateConsumerGroup(ctx, streamName, consumerGroup, "0")
    if err != nil {
        log.Printf("Consumer group might already exist: %v", err)
    }

    // Send messages to stream
    for i := 1; i <= 10; i++ {
        messageID, err := stream.SendMessage(ctx, streamName, map[string]any{
            "userID":    fmt.Sprintf("user-%d", i),
            "action":    "login",
            "timestamp": time.Now(),
            "metadata":  map[string]any{"ip": "192.168.1.1", "device": "mobile"},
        })
        if err != nil {
            return err
        }
        log.Printf("Message sent with ID: %s", messageID)
    }

    // Send message with headers
    messageID, err := stream.SendMessageWithHeaders(ctx, streamName,
        map[string]any{
            "userID": "premium-user-1",
            "action": "purchase",
            "amount": 299.99,
        },
        map[string]any{
            "priority": "high",
            "category": "revenue",
            "region":   "us-west",
        },
    )
    if err != nil {
        return err
    }
    log.Printf("Priority message sent with ID: %s", messageID)

    // Process messages from consumer group
    config := connfx.DefaultConsumerConfig()
    config.PrefetchCount = 5
    config.BlockTimeout = 2 * time.Second

    processCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    err = stream.ProcessMessagesFromGroup(
        processCtx, streamName, consumerGroup, "analytics-worker-1", config,
        func(ctx context.Context, message any) bool {
            msgMap := message.(map[string]any)
            log.Printf("Processing user activity: %v", msgMap)

            // Simulate analytics processing
            userID := msgMap["userID"].(string)
            action := msgMap["action"].(string)

            log.Printf("Analytics: User %s performed %s", userID, action)
            return true // Acknowledge
        },
        map[string]any{}, // Message type template
    )

    if err != nil && err != datafx.ErrContextCanceled {
        return err
    }

    // Get stream statistics
    info, err := stream.GetStreamInfo(ctx, streamName)
    if err != nil {
        return err
    }

    log.Printf("Stream info - Length: %d, Groups: %d, Last ID: %s",
        info.Length, info.Groups, info.LastGeneratedID)

    // Get consumer group information
    groupInfo, err := stream.GetConsumerGroupInfo(ctx, streamName)
    if err != nil {
        return err
    }

    for _, group := range groupInfo {
        log.Printf("Group: %s, Consumers: %d, Pending: %d, Lag: %d",
            group.Name, group.Consumers, group.Pending, group.Lag)
    }

    // Trim stream to keep only last 1000 messages
    err = stream.TrimStream(ctx, streamName, 1000)
    if err != nil {
        return err
    }

    return nil
}
```

#### Reliable Message Processing with Error Handling

```go
func ReliableProcessingExample(ctx context.Context) error {
    // Setup connection
    redisConfig := connfx.NewDefaultRedisConfig()
    redisConn := connfx.NewRedisConnection("redis", redisConfig)

    stream, err := datafx.NewQueueStream(redisConn)
    if err != nil {
        return err
    }

    streamName := "critical-orders"
    consumerGroup := "order-processors"
    consumerName := "processor-1"

    // Create consumer group
    stream.CreateConsumerGroup(ctx, streamName, consumerGroup, "0")

    // Configure for reliable processing
    config := connfx.DefaultConsumerConfig()
    config.AutoAck = false
    config.PrefetchCount = 3
    config.BlockTimeout = 1 * time.Second
    config.MaxRetries = 5
    config.RetryDelay = 2 * time.Second

    // Statistics tracking
    stats := struct {
        processed int
        failed    int
        retried   int
    }{}

    // Background pending message claimer
    go func() {
        ticker := time.NewTicker(10 * time.Second)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                claimed, err := stream.ClaimPendingMessages(
                    ctx, streamName, consumerGroup, consumerName,
                    30*time.Second, 10,
                )
                if err != nil {
                    continue
                }

                if len(claimed) > 0 {
                    log.Printf("Claimed %d pending messages for retry", len(claimed))
                    stats.retried += len(claimed)
                }
            }
        }
    }()

    // Send test messages
    for i := 1; i <= 20; i++ {
        stream.SendMessage(ctx, streamName, map[string]any{
            "orderID":     fmt.Sprintf("order-%d", i),
            "amount":      float64(i * 10),
            "customerID":  fmt.Sprintf("customer-%d", i%5),
            "timestamp":   time.Now(),
            "shouldFail":  i%7 == 0, // Every 7th message will fail
        })
    }

    // Process with timeout
    processCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
    defer cancel()

    err = stream.ProcessMessagesFromGroup(
        processCtx, streamName, consumerGroup, consumerName, config,
        func(ctx context.Context, message any) bool {
            order := message.(map[string]any)
            orderID := order["orderID"].(string)
            shouldFail, _ := order["shouldFail"].(bool)

            log.Printf("Processing order: %s", orderID)

            // Simulate processing that might fail
            if shouldFail {
                log.Printf("Order processing failed: %s", orderID)
                stats.failed++
                return false // Nack - will be retried
            }

            // Simulate processing time
            time.Sleep(100 * time.Millisecond)

            log.Printf("Order processed successfully: %s", orderID)
            stats.processed++
            return true // Ack
        },
        map[string]any{},
    )

    if err != nil && err != datafx.ErrContextCanceled {
        return err
    }

    log.Printf("Processing complete - Processed: %d, Failed: %d, Retried: %d",
        stats.processed, stats.failed, stats.retried)

    return nil
}
```

#### Queue Consumer Configuration

```go
// Default configuration
config := connfx.DefaultConsumerConfig()

// Custom configuration for high-throughput processing
config := connfx.ConsumerConfig{
    AutoAck:       false,          // Manual acknowledgment for reliability
    Exclusive:     true,           // Exclusive access to queue
    NoLocal:       false,          // Receive messages from this connection
    NoWait:        false,          // Wait for server response
    PrefetchCount: 50,             // Prefetch more messages for performance
    BlockTimeout:  5 * time.Second, // Wait up to 5 seconds for new messages
    MaxRetries:    5,              // Retry failed messages up to 5 times
    RetryDelay:    2 * time.Second, // Wait 2 seconds between retries
    Args:          nil,            // Additional arguments
}

// Custom configuration for low-latency processing
config := connfx.ConsumerConfig{
    AutoAck:       true,           // Auto-acknowledge for speed
    PrefetchCount: 1,              // Process one message at a time
    BlockTimeout:  100 * time.Millisecond, // Short timeout for responsiveness
    MaxRetries:    1,              // Minimal retries for speed
    RetryDelay:    100 * time.Millisecond,
}

// Start consuming with configuration
messages, errors := queue.Consume(ctx, "my-queue", config)
```

#### Raw Queue Operations

```go
// Publish raw bytes
rawData := []byte(`{"type": "raw", "data": "some data"}`)
err = queue.PublishRaw(ctx, queueName, rawData)

// Publish raw bytes with headers
headers := map[string]any{
    "encoding": "gzip",
    "version":  "1.0",
}
err = queue.PublishRawWithHeaders(ctx, queueName, rawData, headers)

// Consume with defaults
messages, errors := queue.ConsumeWithDefaults(ctx, queueName)

// Process raw messages
for msg := range messages {
    log.Printf("Received raw message: %s", string(msg.Body))

    // Access headers
    if encoding, ok := msg.Headers["encoding"]; ok {
        log.Printf("Encoding: %v", encoding)
    }

    // Check message metadata
    log.Printf("Message ID: %s, Timestamp: %v, Delivery Count: %d",
        msg.MessageID, msg.Timestamp, msg.DeliveryCount)

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
kafkaFactory := connfx.NewKafkaFactory()
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
