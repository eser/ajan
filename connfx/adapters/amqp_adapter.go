package adapters

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/eser/ajan/connfx"
	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	ErrAMQPClientNotInitialized = errors.New("AMQP client not initialized")
	ErrFailedToOpenConnection   = errors.New("failed to open AMQP connection")
	ErrFailedToOpenChannel      = errors.New("failed to open AMQP channel")
	ErrFailedToCloseConnection  = errors.New("failed to close AMQP connection")
	ErrFailedToCloseChannel     = errors.New("failed to close AMQP channel")
	ErrFailedToDeclareQueue     = errors.New("failed to declare queue")
	ErrFailedToPublishMessage   = errors.New("failed to publish message")
	ErrFailedToStartConsuming   = errors.New("failed to start consuming")
	ErrChannelClosed            = errors.New("channel closed")
	ErrFailedToReconnect        = errors.New("failed to reconnect")
	ErrDeliveryChannelClosed    = errors.New("delivery channel closed")
)

// AMQPAdapter implements the QueueRepository interface for AMQP-based message queues.
type AMQPAdapter struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	dsn        string
}

// AMQPConnection implements the connfx.Connection interface for AMQP connections.
type AMQPConnection struct {
	adapter  *AMQPAdapter
	protocol string
	state    connfx.ConnectionState
}

// NewAMQPConnection creates a new AMQP connection.
func NewAMQPConnection(protocol, dsn string) *AMQPConnection {
	adapter := &AMQPAdapter{
		dsn:        dsn,
		connection: nil,
		channel:    nil,
	}

	return &AMQPConnection{
		adapter:  adapter,
		protocol: protocol,
		state:    connfx.ConnectionStateConnected,
	}
}

// Connection interface implementation.
func (ac *AMQPConnection) GetBehaviors() []connfx.ConnectionBehavior {
	return []connfx.ConnectionBehavior{
		connfx.ConnectionBehaviorStateful,
		connfx.ConnectionBehaviorStreaming,
		connfx.ConnectionBehaviorQueue,
	}
}

func (ac *AMQPConnection) GetProtocol() string {
	return ac.protocol
}

func (ac *AMQPConnection) GetState() connfx.ConnectionState {
	return ac.state
}

func (ac *AMQPConnection) HealthCheck(ctx context.Context) *connfx.HealthStatus {
	start := time.Now()

	status := &connfx.HealthStatus{
		Timestamp: start,
		State:     ac.state,
		Error:     nil,
		Message:   "",
		Latency:   0,
	}

	if ac.adapter.connection != nil && !ac.adapter.connection.IsClosed() {
		status.Message = "AMQP connection healthy"
	} else {
		status.Error = ErrAMQPClientNotInitialized
		status.Message = "AMQP connection not initialized or closed"
		status.State = connfx.ConnectionStateError
	}

	status.Latency = time.Since(start)

	return status
}

func (ac *AMQPConnection) Close(ctx context.Context) error {
	if ac.adapter.channel != nil {
		if err := ac.adapter.channel.Close(); err != nil {
			return fmt.Errorf("%w: %w", ErrFailedToCloseChannel, err)
		}
	}

	if ac.adapter.connection != nil {
		if err := ac.adapter.connection.Close(); err != nil {
			return fmt.Errorf("%w: %w", ErrFailedToCloseConnection, err)
		}
	}

	return nil
}

func (ac *AMQPConnection) GetRawConnection() any {
	return ac.adapter
}

// QueueRepository interface implementation.
func (aa *AMQPAdapter) QueueDeclare(ctx context.Context, name string) (string, error) {
	if err := aa.ensureConnection(); err != nil {
		return "", fmt.Errorf("%w (queue=%q): %w", ErrAMQPClientNotInitialized, name, err)
	}

	queue, err := aa.channel.QueueDeclare(
		name,  // queue name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return "", fmt.Errorf("%w (queue=%q): %w", ErrFailedToDeclareQueue, name, err)
	}

	return queue.Name, nil
}

func (aa *AMQPAdapter) Publish(ctx context.Context, queueName string, body []byte) error {
	if err := aa.ensureConnection(); err != nil {
		return fmt.Errorf("%w (queue=%q): %w", ErrAMQPClientNotInitialized, queueName, err)
	}

	err := aa.channel.Publish(
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			Headers:         nil,
			ContentType:     "application/json",
			ContentEncoding: "",
			Body:            body,
			DeliveryMode:    0,
			Priority:        0,
			CorrelationId:   "",
			ReplyTo:         "",
			Expiration:      "",
			MessageId:       "",
			Timestamp:       time.Time{},
			Type:            "",
			UserId:          "",
			AppId:           "",
		},
	)
	if err != nil {
		return fmt.Errorf("%w (queue=%q): %w", ErrFailedToPublishMessage, queueName, err)
	}

	return nil
}

func (aa *AMQPAdapter) Consume(
	ctx context.Context,
	queueName string,
	config connfx.ConsumerConfig,
) (<-chan connfx.Message, <-chan error) {
	messages := make(chan connfx.Message)
	errors := make(chan error)

	go func() {
		defer close(messages)
		defer close(errors)

		aa.consumeLoop(ctx, queueName, config, messages, errors)
	}()

	return messages, errors
}

// ensureConnection ensures we have an active AMQP connection.
func (aa *AMQPAdapter) ensureConnection() error {
	if aa.connection != nil && !aa.connection.IsClosed() {
		return nil
	}

	connection, err := amqp.Dial(aa.dsn)
	if err != nil {
		return fmt.Errorf("%w (dsn=%q): %w", ErrFailedToOpenConnection, aa.dsn, err)
	}

	channel, err := connection.Channel()
	if err != nil {
		return fmt.Errorf("%w (dsn=%q): %w", ErrFailedToOpenChannel, aa.dsn, err)
	}

	aa.connection = connection
	aa.channel = channel

	return nil
}

// consumeLoop handles the message consumption loop with reconnection logic.
func (aa *AMQPAdapter) consumeLoop(
	ctx context.Context,
	queueName string,
	config connfx.ConsumerConfig,
	messages chan<- connfx.Message,
	errors chan<- error,
) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := aa.ensureConnection(); err != nil {
				errors <- fmt.Errorf("%w: %w", ErrFailedToReconnect, err)

				continue
			}

			if err := aa.processMessages(ctx, queueName, config, messages, errors); err != nil {
				// Connection lost, reset channel and retry
				aa.channel = nil
			}
		}
	}
}

// processMessages handles message processing for a single connection session.
func (aa *AMQPAdapter) processMessages(
	ctx context.Context,
	queueName string,
	config connfx.ConsumerConfig,
	messages chan<- connfx.Message,
	errors chan<- error,
) error {
	deliveries, err := aa.channel.Consume(
		queueName,        // queue
		"",               // consumer
		config.AutoAck,   // auto-ack
		config.Exclusive, // exclusive
		config.NoLocal,   // no-local
		config.NoWait,    // no-wait
		config.Args,      // args
	)
	if err != nil {
		errors <- fmt.Errorf("%w (queue=%q): %w", ErrFailedToStartConsuming, queueName, err)

		return fmt.Errorf("%w: %w", ErrFailedToStartConsuming, err)
	}

	// Monitor channel closure
	chanClose := aa.channel.NotifyClose(make(chan *amqp.Error, 1))

	// Process messages
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-chanClose:
			errors <- fmt.Errorf("%w: %w", ErrChannelClosed, err)

			return err
		case delivery, ok := <-deliveries:
			if !ok {
				return ErrDeliveryChannelClosed
			}

			msg := aa.createMessage(delivery)
			select {
			case messages <- msg:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

// createMessage creates a connfx.Message from an AMQP delivery.
func (aa *AMQPAdapter) createMessage(delivery amqp.Delivery) connfx.Message {
	msg := connfx.Message{
		Body:    delivery.Body,
		Headers: delivery.Headers,
	}

	// Set acknowledgment functions
	msg.SetAckFunc(func() error {
		return delivery.Ack(false)
	})
	msg.SetNackFunc(func(requeue bool) error {
		return delivery.Nack(false, requeue)
	})

	return msg
}

// AMQPConnectionFactory creates AMQP connections.
type AMQPConnectionFactory struct {
	protocol string
}

// NewAMQPConnectionFactory creates a new AMQP connection factory for a specific protocol.
func NewAMQPConnectionFactory(protocol string) *AMQPConnectionFactory {
	return &AMQPConnectionFactory{
		protocol: protocol,
	}
}

// NewAMQPFactory creates a new AMQP factory with default "amqp" protocol.
func NewAMQPFactory() *AMQPConnectionFactory {
	return NewAMQPConnectionFactory("amqp")
}

func (f *AMQPConnectionFactory) CreateConnection(
	ctx context.Context,
	config *connfx.ConfigTarget,
) (connfx.Connection, error) {
	dsn := config.DSN
	if dsn == "" {
		// Build DSN from config components using net.JoinHostPort
		hostPort := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))
		dsn = "amqp://" + hostPort
	}

	// Create the connection
	conn := NewAMQPConnection(f.protocol, dsn)

	return conn, nil
}

func (f *AMQPConnectionFactory) GetProtocol() string {
	return f.protocol
}

func (f *AMQPConnectionFactory) GetSupportedBehaviors() []connfx.ConnectionBehavior {
	return []connfx.ConnectionBehavior{
		connfx.ConnectionBehaviorStateful,
		connfx.ConnectionBehaviorStreaming,
		connfx.ConnectionBehaviorQueue,
	}
}
