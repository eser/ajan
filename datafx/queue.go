package datafx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/eser/ajan/connfx"
)

var (
	ErrQueueNotSupported = errors.New("connection does not support queue operations")
	ErrMessageProcessing = errors.New("message processing failed")
	ErrContextCanceled   = errors.New("context canceled")
)

// Queue provides high-level message queue operations.
type Queue struct {
	conn       connfx.Connection
	repository connfx.QueueRepository
}

// NewQueue creates a new Queue instance from a connfx connection.
// The connection must support queue operations.
func NewQueue(conn connfx.Connection) (*Queue, error) {
	if conn == nil {
		return nil, fmt.Errorf("%w: connection is nil", ErrConnectionNotSupported)
	}

	// Check if the connection supports queue operations
	behaviors := conn.GetBehaviors()
	supportsQueue := slices.Contains(behaviors, connfx.ConnectionBehaviorQueue)

	if !supportsQueue {
		return nil, fmt.Errorf("%w: connection does not support queue operations (protocol=%q)",
			ErrQueueNotSupported, conn.GetProtocol())
	}

	// Get the queue repository from the raw connection
	repo, ok := conn.GetRawConnection().(connfx.QueueRepository)
	if !ok {
		return nil, fmt.Errorf(
			"%w: connection does not implement QueueRepository interface (protocol=%q)",
			ErrQueueNotSupported,
			conn.GetProtocol(),
		)
	}

	return &Queue{
		conn:       conn,
		repository: repo,
	}, nil
}

// DeclareQueue declares a queue and returns its name.
func (q *Queue) DeclareQueue(ctx context.Context, name string) (string, error) {
	queueName, err := q.repository.QueueDeclare(ctx, name)
	if err != nil {
		return "", fmt.Errorf("failed to declare queue %q: %w", name, err)
	}

	return queueName, nil
}

// Publish sends a message to a queue after marshaling it to JSON.
func (q *Queue) Publish(ctx context.Context, queueName string, message any) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("%w (queue=%q): %w", ErrFailedToMarshal, queueName, err)
	}

	if err := q.repository.Publish(ctx, queueName, data); err != nil {
		return fmt.Errorf("failed to publish message to queue %q: %w", queueName, err)
	}

	return nil
}

// PublishRaw sends raw bytes to a queue.
func (q *Queue) PublishRaw(ctx context.Context, queueName string, data []byte) error {
	if err := q.repository.Publish(ctx, queueName, data); err != nil {
		return fmt.Errorf("failed to publish raw message to queue %q: %w", queueName, err)
	}

	return nil
}

// Consume starts consuming messages from a queue with the given configuration.
// Returns channels for messages and errors.
func (q *Queue) Consume(
	ctx context.Context,
	queueName string,
	config connfx.ConsumerConfig,
) (<-chan connfx.Message, <-chan error) {
	return q.repository.Consume(ctx, queueName, config)
}

// ConsumeWithDefaults starts consuming messages from a queue with default configuration.
func (q *Queue) ConsumeWithDefaults(
	ctx context.Context,
	queueName string,
) (<-chan connfx.Message, <-chan error) {
	config := connfx.DefaultConsumerConfig()

	return q.repository.Consume(ctx, queueName, config)
}

// ProcessMessages provides a convenient way to process messages with automatic unmarshalling.
// The messageHandler function receives the unmarshaled message and should return true to acknowledge
// the message, or false to negatively acknowledge it.
func (q *Queue) ProcessMessages(
	ctx context.Context,
	queueName string,
	config connfx.ConsumerConfig,
	messageHandler func(ctx context.Context, message any) bool,
	messageType any,
) error {
	messages, errors := q.repository.Consume(ctx, queueName, config)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w: %w", ErrContextCanceled, ctx.Err())
		case err := <-errors:
			if err != nil {
				return fmt.Errorf("%w (queue=%q): %w", ErrMessageProcessing, queueName, err)
			}
		case msg, ok := <-messages:
			if !ok {
				return nil // Channel closed
			}

			if err := q.processMessage(ctx, msg, messageHandler, messageType); err != nil {
				return err
			}
		}
	}
}

// ProcessMessagesWithDefaults processes messages with default consumer configuration.
func (q *Queue) ProcessMessagesWithDefaults(
	ctx context.Context,
	queueName string,
	messageHandler func(ctx context.Context, message any) bool,
	messageType any,
) error {
	config := connfx.DefaultConsumerConfig()

	return q.ProcessMessages(ctx, queueName, config, messageHandler, messageType)
}

// GetConnection returns the underlying connfx connection.
func (q *Queue) GetConnection() connfx.Connection {
	return q.conn
}

// GetRepository returns the underlying queue repository.
func (q *Queue) GetRepository() connfx.QueueRepository {
	return q.repository
}

// processMessage handles the processing of a single message.
func (q *Queue) processMessage(
	ctx context.Context,
	msg connfx.Message,
	messageHandler func(ctx context.Context, message any) bool,
	messageType any,
) error {
	// Create a new instance of the message type
	messageValue := q.createMessageInstance(messageType)

	// Unmarshal the message
	if err := json.Unmarshal(msg.Body, &messageValue); err != nil {
		// Nack the message due to unmarshalling error
		if nackErr := msg.Nack(false); nackErr != nil {
			return fmt.Errorf("failed to nack message after unmarshal error: %w", nackErr)
		}

		return nil // Continue processing other messages
	}

	// Process the message
	success := messageHandler(ctx, messageValue)

	// Acknowledge or nack based on processing result
	return q.acknowledgeMessage(msg, success)
}

// createMessageInstance creates an instance for unmarshalling the message.
func (q *Queue) createMessageInstance(messageType any) any {
	if messageType != nil {
		// Use the provided message type
		return messageType
	}

	// Default to generic map
	return make(map[string]any)
}

// acknowledgeMessage handles message acknowledgment based on processing success.
func (q *Queue) acknowledgeMessage(msg connfx.Message, success bool) error {
	if success {
		if err := msg.Ack(); err != nil {
			return fmt.Errorf("failed to acknowledge message: %w", err)
		}
	} else {
		if err := msg.Nack(true); err != nil { // Requeue on failure
			return fmt.Errorf("failed to nack message: %w", err)
		}
	}

	return nil
}
