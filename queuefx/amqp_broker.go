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
