package main

import (
	"context"
	"fmt"
	"time"

	"github.com/eser/ajan/connfx"
	"github.com/eser/ajan/datafx"
)

func main() {
	ctx := context.Background()

	appContext, err := NewAppContext(ctx)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize app context: %v", err))
	}

	appContext.Logger.Info("app context initialized",
		"name", appContext.Config.AppName,
		"env", appContext.Config.AppEnv,
	)

	if err := business(ctx, appContext); err != nil {
		panic(fmt.Sprintf("business logic failed: %v", err))
	}
}

func business(ctx context.Context, appContext *AppContext) error {
	// Example: Basic data operations using Redis connection
	redisConnection := appContext.Connections.GetNamed("redis-cache")
	if redisConnection != nil {
		data, err := datafx.NewCache(redisConnection)
		if err != nil {
			appContext.Logger.Warn("failed to create data instance from Redis", "error", err)
		} else {
			if err := performBasicOperations(ctx, data); err != nil {
				appContext.Logger.Warn("basic data operations failed", "error", err)
			}
		}
	} else {
		appContext.Logger.Info("Redis connection not available, skipping basic data operations")
	}

	// Example: Transactional operations (use default connection if it supports transactions)
	defaultConnection := appContext.Connections.GetDefault()
	if err := performTransactionalOperations(ctx, defaultConnection); err != nil {
		appContext.Logger.Warn("transactional operations not supported or failed", "error", err)
	}

	// Example: Cache operations using Redis connection
	if redisConnection != nil {
		if err := performCacheOperations(ctx, redisConnection, appContext); err != nil {
			appContext.Logger.Warn("cache operations failed", "error", err)
		}
	} else {
		appContext.Logger.Info("Redis cache connection not available, skipping cache operations")
	}

	// Example: Queue operations using AMQP connection
	amqpConnection := appContext.Connections.GetNamed("amqp-queue")
	if amqpConnection != nil {
		if err := performQueueOperations(ctx, amqpConnection, appContext); err != nil {
			appContext.Logger.Warn("queue operations failed", "error", err)
		}
	} else {
		appContext.Logger.Info("AMQP queue connection not available, skipping queue operations")
	}

	return nil
}

func performBasicOperations(ctx context.Context, store *datafx.Store) error {
	// Example data operations
	user := &datafx.User{
		ID:    "user123",
		Name:  "John Doe",
		Email: "john@example.com",
	}

	// Set data
	if err := store.Set(ctx, "user:123", user); err != nil {
		return fmt.Errorf("failed to set user: %w", err)
	}

	// Get data
	var retrievedUser datafx.User
	if err := store.Get(ctx, "user:123", &retrievedUser); err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Check existence
	exists, err := store.Exists(ctx, "user:123")
	if err != nil {
		return fmt.Errorf("failed to check existence: %w", err)
	}

	if exists {
		// Update data
		retrievedUser.Name = "John Smith"
		if err := store.Update(ctx, "user:123", &retrievedUser); err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}
	}

	return nil
}

func performTransactionalOperations(ctx context.Context, connection connfx.Connection) error {
	// Try to create transactional store instance
	txData, err := datafx.NewTransactionalStore(connection)
	if err != nil {
		return fmt.Errorf("transactional operations not supported: %w", err)
	}

	// Execute operations within a transaction
	err = txData.ExecuteTransaction(ctx, func(tx *datafx.TransactionStore) error {
		// All operations within this function are transactional
		user := &datafx.User{ID: "tx-user-123", Name: "Transaction User", Email: "tx@example.com"}
		if err := tx.Set(ctx, "tx-user:123", user); err != nil {
			return err // Transaction will be rolled back
		}

		product := &datafx.Product{ID: "tx-product-456", Name: "Transaction Widget", Price: 19.99}
		if err := tx.Set(ctx, "tx-product:456", product); err != nil {
			return err // Transaction will be rolled back
		}

		return nil // Transaction will be committed
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

func performCacheOperations(ctx context.Context, connection connfx.Connection, appContext *AppContext) error {
	// Try to create cache instance
	cache, err := datafx.NewCache(connection)
	if err != nil {
		return fmt.Errorf("cache operations not supported: %w", err)
	}

	appContext.Logger.Info("performing cache operations")

	// Cache user session with 5 minute expiration
	sessionData := map[string]any{
		"user_id":    "user123",
		"session_id": "sess_abc123",
		"expires_at": time.Now().Add(5 * time.Minute),
	}

	if err := cache.Set(ctx, "session:abc123", sessionData, 5*time.Minute); err != nil {
		return fmt.Errorf("failed to cache session: %w", err)
	}

	// Retrieve cached session
	var retrievedSession map[string]any
	if err := cache.Get(ctx, "session:abc123", &retrievedSession); err != nil {
		return fmt.Errorf("failed to get cached session: %w", err)
	}

	// Check TTL
	ttl, err := cache.GetTTL(ctx, "session:abc123")
	if err != nil {
		return fmt.Errorf("failed to get TTL: %w", err)
	}

	appContext.Logger.Info("cache session retrieved",
		"session_id", retrievedSession["session_id"],
		"ttl", ttl)

	// Extend expiration
	if err := cache.Expire(ctx, "session:abc123", 10*time.Minute); err != nil {
		return fmt.Errorf("failed to extend expiration: %w", err)
	}

	// Cache raw data
	rawData := []byte("temporary data")
	if err := cache.SetRaw(ctx, "temp:data", rawData, 1*time.Minute); err != nil {
		return fmt.Errorf("failed to cache raw data: %w", err)
	}

	return nil
}

func performQueueOperations(ctx context.Context, connection connfx.Connection, appContext *AppContext) error {
	// Try to create queue instance
	queue, err := datafx.NewQueue(connection)
	if err != nil {
		return fmt.Errorf("queue operations not supported: %w", err)
	}

	appContext.Logger.Info("performing queue operations")

	// Declare a queue
	queueName, err := queue.DeclareQueue(ctx, "app-events")
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Publish a message
	eventMessage := map[string]any{
		"event_type": "user_login",
		"user_id":    "user123",
		"timestamp":  time.Now(),
		"metadata": map[string]string{
			"ip_address": "192.168.1.100",
			"user_agent": "Mozilla/5.0...",
		},
	}

	if err := queue.Publish(ctx, queueName, eventMessage); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	appContext.Logger.Info("message published", "queue", queueName, "event_type", "user_login")

	// Publish raw message
	rawMessage := []byte(`{"raw": "event", "data": "some binary data"}`)
	if err := queue.PublishRaw(ctx, queueName, rawMessage); err != nil {
		return fmt.Errorf("failed to publish raw message: %w", err)
	}

	// Example: Start a consumer (commented out to avoid blocking in sample app)
	// This would typically be in a separate goroutine or service
	/*
		messages, errors := queue.ConsumeWithDefaults(ctx, queueName)

		go func() {
			for {
				select {
				case msg := <-messages:
					var event map[string]any
					if err := json.Unmarshal(msg.Body, &event); err != nil {
						appContext.Logger.Error("failed to unmarshal message", "error", err)
						msg.Nack(false) // Don't requeue invalid messages
						continue
					}

					appContext.Logger.Info("processing event", "event", event)

					// Process the event...
					// if successful:
					msg.Ack()

				case err := <-errors:
					appContext.Logger.Error("queue error", "error", err)
				case <-ctx.Done():
					return
				}
			}
		}()
	*/

	return nil
}
