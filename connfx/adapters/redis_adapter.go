package adapters

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/eser/ajan/connfx"
)

const (
	DefaultRedisPort        = 6379
	DefaultRedisTimeout     = 5 * time.Second
	RedisHealthCheckTimeout = 2 * time.Second
)

var (
	ErrFailedToConnectRedis = errors.New("failed to connect to Redis")
	ErrRedisNotImplemented  = errors.New(
		"redis client not implemented - placeholder for demonstration",
	)
	ErrInvalidConfigTypeRedis  = errors.New("invalid config type for Redis connection")
	ErrFailedToCloseConnection = errors.New("failed to close connection")
)

// RedisConnection represents a Redis connection that supports both stateful and streaming behaviors.
type RedisConnection struct {
	lastHealth time.Time
	conn       net.Conn // Simplified for demonstration - would use Redis client library
	protocol   string
	host       string
	port       int
	state      int32 // atomic field for connection state
}

// RedisConnectionFactory creates Redis connections.
type RedisConnectionFactory struct {
	protocol string
}

// NewRedisConnectionFactory creates a new Redis connection factory.
func NewRedisConnectionFactory(protocol string) *RedisConnectionFactory {
	return &RedisConnectionFactory{
		protocol: protocol,
	}
}

func (f *RedisConnectionFactory) CreateConnection(
	ctx context.Context,
	config *connfx.ConfigTarget,
) (connfx.Connection, error) {
	// Default Redis port
	port := DefaultRedisPort
	if config.Port > 0 {
		port = config.Port
	}

	host := "localhost"
	if config.Host != "" {
		host = config.Host
	}

	// For demonstration - create a simple TCP connection to Redis
	// In a real implementation, you'd use a Redis client library like go-redis
	address := fmt.Sprintf("%s:%d", host, port)

	// Set timeout for connection attempt
	timeout := DefaultRedisTimeout
	if config.Timeout > 0 {
		timeout = config.Timeout
	}

	dialer := net.Dialer{Timeout: timeout} //nolint:exhaustruct

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToConnectRedis, err)
	}

	redisConn := &RedisConnection{
		protocol:   f.protocol,
		host:       host,
		port:       port,
		state:      int32(connfx.ConnectionStateConnected),
		conn:       conn,
		lastHealth: time.Time{},
	}

	return redisConn, nil
}

func (f *RedisConnectionFactory) GetProtocol() string {
	return f.protocol
}

func (f *RedisConnectionFactory) GetSupportedBehaviors() []connfx.ConnectionBehavior {
	// Redis supports both stateful (key-value operations) and streaming (pub/sub) behaviors
	return []connfx.ConnectionBehavior{
		connfx.ConnectionBehaviorStateful,
		connfx.ConnectionBehaviorStreaming,
	}
}

// Connection interface implementation

func (c *RedisConnection) GetBehaviors() []connfx.ConnectionBehavior {
	// Redis connections support both stateful and streaming behaviors
	return []connfx.ConnectionBehavior{
		connfx.ConnectionBehaviorStateful,  // For GET/SET/HGET operations
		connfx.ConnectionBehaviorStreaming, // For PUBLISH/SUBSCRIBE operations
	}
}

func (c *RedisConnection) GetProtocol() string {
	return c.protocol
}

func (c *RedisConnection) GetState() connfx.ConnectionState {
	state := atomic.LoadInt32(&c.state)

	return connfx.ConnectionState(state)
}

func (c *RedisConnection) HealthCheck(ctx context.Context) *connfx.HealthStatus {
	start := time.Now()
	status := &connfx.HealthStatus{ //nolint:exhaustruct
		Timestamp: start,
	}

	// Simple ping test by checking if connection is alive
	if c.conn == nil {
		atomic.StoreInt32(&c.state, int32(connfx.ConnectionStateDisconnected))
		status.State = connfx.ConnectionStateDisconnected
		status.Message = "Connection is nil"
		status.Latency = time.Since(start)

		return status
	}

	// Set a short deadline for the health check
	deadline := time.Now().Add(RedisHealthCheckTimeout)
	if err := c.conn.SetDeadline(deadline); err != nil {
		atomic.StoreInt32(&c.state, int32(connfx.ConnectionStateError))
		status.State = connfx.ConnectionStateError
		status.Error = err
		status.Message = fmt.Sprintf("Failed to set deadline: %v", err)
		status.Latency = time.Since(start)

		return status
	}

	// Send a PING command (simplified)
	_, err := c.conn.Write([]byte("PING\r\n"))
	status.Latency = time.Since(start)

	if err != nil {
		atomic.StoreInt32(&c.state, int32(connfx.ConnectionStateError))
		status.State = connfx.ConnectionStateError
		status.Error = err
		status.Message = fmt.Sprintf("Health check failed: %v", err)

		return status
	}

	// In a real implementation, you'd read the response
	// For demo purposes, assume success if write succeeded
	atomic.StoreInt32(&c.state, int32(connfx.ConnectionStateConnected))
	status.State = connfx.ConnectionStateConnected
	status.Message = fmt.Sprintf("Connected to Redis at %s:%d (supports: stateful + streaming)",
		c.host, c.port)

	c.lastHealth = start

	return status
}

func (c *RedisConnection) Close(ctx context.Context) error {
	atomic.StoreInt32(&c.state, int32(connfx.ConnectionStateDisconnected))

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return fmt.Errorf("%w: %w", ErrFailedToCloseConnection, err)
		}
	}

	return nil
}

func (c *RedisConnection) GetRawConnection() any {
	return c.conn
}

// Additional Redis-specific methods (simplified for demonstration)

// GetTCPConnection returns the underlying TCP connection (for demo purposes).
func (c *RedisConnection) GetTCPConnection() net.Conn {
	return c.conn
}

// GetAddress returns the Redis server address.
func (c *RedisConnection) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.host, c.port)
}

// IsStatefulReady returns true if the connection can handle stateful operations.
func (c *RedisConnection) IsStatefulReady() bool {
	return c.GetState() == connfx.ConnectionStateConnected
}

// IsStreamingReady returns true if the connection can handle streaming operations.
func (c *RedisConnection) IsStreamingReady() bool {
	return c.GetState() == connfx.ConnectionStateConnected
}
