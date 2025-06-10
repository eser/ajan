package connfx

import (
	"context"
	"time"
)

// Repository defines the port for data access operations.
// This interface will be implemented by adapters in connfx for different storage technologies.
type Repository interface {
	// Get retrieves a value by key
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value with the given key
	Set(ctx context.Context, key string, value []byte) error

	// Remove deletes a value by key
	Remove(ctx context.Context, key string) error

	// Update updates an existing value by key
	Update(ctx context.Context, key string, value []byte) error

	// Exists checks if a key exists
	Exists(ctx context.Context, key string) (bool, error)
}

// CacheRepository extends Repository with cache-specific operations.
type CacheRepository interface {
	Repository

	// SetWithExpiration stores a value with the given key and expiration time
	SetWithExpiration(ctx context.Context, key string, value []byte, expiration time.Duration) error

	// GetTTL returns the time-to-live for a key
	GetTTL(ctx context.Context, key string) (time.Duration, error)

	// Expire sets an expiration time for an existing key
	Expire(ctx context.Context, key string, expiration time.Duration) error
}

// TransactionalRepository extends Repository with transaction support.
type TransactionalRepository interface {
	Repository

	// BeginTransaction starts a new transaction
	BeginTransaction(ctx context.Context) (TransactionContext, error)
}

// TransactionContext represents a transaction context for data operations.
type TransactionContext interface {
	// Commit commits the transaction
	Commit() error

	// Rollback rolls back the transaction
	Rollback() error

	// GetRepository returns a repository bound to this transaction
	GetRepository() Repository
}

// QueryRepository defines the port for query operations (for SQL-like storages).
type QueryRepository interface {
	// Query executes a query and returns raw results
	Query(ctx context.Context, query string, args ...any) (QueryResult, error)

	// Execute runs a command (INSERT, UPDATE, DELETE)
	Execute(ctx context.Context, command string, args ...any) (ExecuteResult, error)
}

// QueryResult represents query results.
type QueryResult interface {
	// Next advances to the next row
	Next() bool

	// Scan scans the current row into destinations
	Scan(dest ...any) error

	// Close closes the result set
	Close() error
}

// ExecuteResult represents execution results.
type ExecuteResult interface {
	// RowsAffected returns the number of rows affected
	RowsAffected() (int64, error)

	// LastInsertId returns the last insert ID (if applicable)
	LastInsertId() (int64, error)
}

// QueueRepository defines the port for message queue operations.
type QueueRepository interface {
	// QueueDeclare declares a queue and returns its name
	QueueDeclare(ctx context.Context, name string) (string, error)

	// Publish sends a message to a queue
	Publish(ctx context.Context, queueName string, body []byte) error

	// Consume starts consuming messages from a queue
	Consume(
		ctx context.Context,
		queueName string,
		config ConsumerConfig,
	) (<-chan Message, <-chan error)
}

// ConsumerConfig holds configuration for message consumption.
type ConsumerConfig struct {
	// Args additional arguments for queue declaration
	Args map[string]any
	// AutoAck when true, the server will automatically acknowledge messages
	AutoAck bool
	// Exclusive when true, only this consumer can access the queue
	Exclusive bool
	// NoLocal when true, the server will not send messages to the connection that published them
	NoLocal bool
	// NoWait when true, the server will not respond to the declare
	NoWait bool
}

// Message represents a consumed message with its metadata and acknowledgment functions.
type Message struct {
	// Headers contains message headers
	Headers map[string]any
	// ack acknowledges the message
	ack func() error
	// nack negatively acknowledges the message
	nack func(requeue bool) error
	// Body contains the message payload
	Body []byte
}

// Ack acknowledges the message.
func (m *Message) Ack() error {
	return m.ack()
}

// Nack negatively acknowledges the message.
func (m *Message) Nack(requeue bool) error {
	return m.nack(requeue)
}

// SetAckFunc sets the acknowledgment function.
func (m *Message) SetAckFunc(ackFunc func() error) {
	m.ack = ackFunc
}

// SetNackFunc sets the negative acknowledgment function.
func (m *Message) SetNackFunc(nackFunc func(requeue bool) error) {
	m.nack = nackFunc
}

// DefaultConsumerConfig returns a default configuration for consuming messages.
func DefaultConsumerConfig() ConsumerConfig {
	return ConsumerConfig{
		AutoAck:   false,
		Exclusive: false,
		NoLocal:   false,
		NoWait:    false,
		Args:      nil,
	}
}

const (
	// ConnectionBehaviorKeyValue represents key-value storage behavior.
	ConnectionBehaviorKeyValue ConnectionBehavior = "key-value"

	// ConnectionBehaviorDocument represents document storage behavior.
	ConnectionBehaviorDocument ConnectionBehavior = "document"

	// ConnectionBehaviorRelational represents relational database behavior.
	ConnectionBehaviorRelational ConnectionBehavior = "relational"

	// ConnectionBehaviorTransactional represents transactional behavior.
	ConnectionBehaviorTransactional ConnectionBehavior = "transactional"

	// ConnectionBehaviorCache represents caching behavior with expiration support.
	ConnectionBehaviorCache ConnectionBehavior = "cache"

	// ConnectionBehaviorQueue represents message queue behavior.
	ConnectionBehaviorQueue ConnectionBehavior = "queue"
)
