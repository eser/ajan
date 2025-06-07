package queuefx

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AmqpBroker struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	dialect    Dialect
}

func NewAmqpBroker(ctx context.Context, dialect Dialect, dsn string) (*AmqpBroker, error) {
	connection, err := amqp.Dial(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open broker connection: %w", err)
	}

	channel, err := connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open broker channel: %w", err)
	}

	return &AmqpBroker{
		connection: connection,
		channel:    channel,
		dialect:    dialect,
	}, nil
}

func (broker *AmqpBroker) GetDialect() Dialect {
	return broker.dialect
}

func (broker *AmqpBroker) QueueDeclare(ctx context.Context, name string) (string, error) {
	queue, err := broker.channel.QueueDeclare(
		name,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("failed to declare queue: %w", err)
	}

	return queue.Name, nil
}

func (broker *AmqpBroker) Publish(ctx context.Context, name string, body []byte) error {
	err := broker.channel.Publish(
		name,
		"",
		false,
		false,
		amqp.Publishing{ //nolint:exhaustruct
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// ConsumerConfig holds configuration for message consumption.
type ConsumerConfig struct {
	// Args additional arguments for declaration
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

// Message represents a consumed message with its metadata and acknowledgment functions.
type Message struct {
	Headers map[string]any
	ack     func() error
	nack    func(requeue bool) error
	Body    []byte
}

// Ack acknowledges the message.
func (m *Message) Ack() error {
	return m.ack()
}

// Nack negatively acknowledges the message.
func (m *Message) Nack(requeue bool) error {
	return m.nack(requeue)
}

// The consumer will automatically reconnect on connection failures.
func (broker *AmqpBroker) Consume( //nolint:cyclop,gocognit,funlen
	ctx context.Context,
	queueName string,
	config ConsumerConfig,
) (<-chan Message, <-chan error) {
	messages := make(chan Message)
	errors := make(chan error)

	go func() {
		defer close(messages)
		defer close(errors)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Ensure we have an open channel
				if broker.channel == nil {
					if err := broker.reconnect(); err != nil {
						errors <- fmt.Errorf("failed to reconnect: %w", err)
						// Add exponential backoff here if needed
						continue
					}
				}

				// Start consuming
				deliveries, err := broker.channel.Consume(
					queueName,
					"", // Consumer name (empty for auto-generated)
					config.AutoAck,
					config.Exclusive,
					config.NoLocal,
					config.NoWait,
					config.Args,
				)
				if err != nil {
					errors <- fmt.Errorf("failed to start consuming: %w", err)

					continue
				}

				// Monitor channel closure
				chanClose := broker.channel.NotifyClose(make(chan *amqp.Error, 1))

				// Process messages
				for {
					select {
					case <-ctx.Done():
						return
					case err := <-chanClose:
						errors <- fmt.Errorf("channel closed: %w", err)

						broker.channel = nil

						goto RECONNECT
					case delivery, ok := <-deliveries:
						if !ok {
							goto RECONNECT
						}

						msg := Message{
							Body:    delivery.Body,
							Headers: delivery.Headers,
							ack: func() error {
								return delivery.Ack(false)
							},
							nack: func(requeue bool) error {
								return delivery.Nack(false, requeue)
							},
						}

						select {
						case messages <- msg:
						case <-ctx.Done():
							return
						}
					}
				}
			RECONNECT:
				continue
			}
		}
	}()

	return messages, errors
}

// reconnect attempts to recreate the channel.
func (broker *AmqpBroker) reconnect() error {
	if broker.channel != nil {
		err := broker.channel.Close()
		if err != nil {
			return fmt.Errorf("failed to close channel: %w", err)
		}
	}

	channel, err := broker.connection.Channel()
	if err != nil {
		return fmt.Errorf("failed to recreate channel: %w", err)
	}

	broker.channel = channel

	return nil
}
