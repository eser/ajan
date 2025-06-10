package connfx

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrRedisClientNotInitialized = errors.New("redis client not initialized")
	ErrFailedToCloseRedisClient  = errors.New("failed to close Redis client")
	ErrRedisOperation            = errors.New("redis operation failed")
)

// RedisAdapter is an example adapter that implements the Repository interface.
// This would typically wrap a real Redis client like go-redis/redis.
type RedisAdapter struct {
	client   RedisClient // This would be a real Redis client
	host     string
	password string
	port     int
	db       int
}

// RedisClient represents a Redis client interface (would be implemented by actual Redis library).
type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, keys ...string) (int64, error)
	Close() error
	Ping(ctx context.Context) error
	TTL(ctx context.Context, key string) (time.Duration, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
}

// RedisConnection implements the connfx.Connection interface.
type RedisConnection struct {
	adapter  *RedisAdapter
	protocol string
	state    ConnectionState
}

// NewRedisConnection creates a new Redis connection.
func NewRedisConnection(protocol, host string, port int, password string, db int) *RedisConnection {
	adapter := &RedisAdapter{
		host:     host,
		port:     port,
		password: password,
		db:       db,
		client:   nil, // Will be initialized with real Redis client
	}

	return &RedisConnection{
		adapter:  adapter,
		protocol: protocol,
		state:    ConnectionStateConnected,
	}
}

// Connection interface implementation.
func (rc *RedisConnection) GetBehaviors() []ConnectionBehavior {
	return []ConnectionBehavior{
		ConnectionBehaviorStateful,
		ConnectionBehaviorStreaming,
		ConnectionBehaviorKeyValue,
		ConnectionBehaviorCache,
	}
}

func (rc *RedisConnection) GetProtocol() string {
	return rc.protocol
}

func (rc *RedisConnection) GetState() ConnectionState {
	return rc.state
}

func (rc *RedisConnection) HealthCheck(ctx context.Context) *HealthStatus {
	start := time.Now()

	status := &HealthStatus{
		Timestamp: start,
		State:     rc.state,
		Error:     nil,
		Message:   "",
		Latency:   0,
	}

	if rc.adapter.client != nil {
		err := rc.adapter.client.Ping(ctx)
		if err != nil {
			status.Error = err
			status.Message = "Redis ping failed"
			status.State = ConnectionStateError
		} else {
			status.Message = "Redis connection healthy"
		}
	} else {
		status.Error = ErrRedisClientNotInitialized
		status.Message = "Redis client not initialized"
		status.State = ConnectionStateError
	}

	status.Latency = time.Since(start)

	return status
}

func (rc *RedisConnection) Close(ctx context.Context) error {
	if rc.adapter.client != nil {
		if err := rc.adapter.client.Close(); err != nil {
			return fmt.Errorf("%w: %w", ErrFailedToCloseRedisClient, err)
		}
	}

	return nil
}

func (rc *RedisConnection) GetRawConnection() any {
	return rc.adapter
}

// StoreRepository interface implementation.
func (ra *RedisAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	if ra.client == nil {
		return nil, fmt.Errorf("%w (key=%q)", ErrRedisClientNotInitialized, key)
	}

	value, err := ra.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("%w (operation=get, key=%q): %w", ErrRedisOperation, key, err)
	}

	return []byte(value), nil
}

func (ra *RedisAdapter) Set(ctx context.Context, key string, value []byte) error {
	if ra.client == nil {
		return fmt.Errorf("%w (key=%q)", ErrRedisClientNotInitialized, key)
	}

	err := ra.client.Set(ctx, key, string(value), 0) // 0 means no expiration
	if err != nil {
		return fmt.Errorf("%w (operation=set, key=%q): %w", ErrRedisOperation, key, err)
	}

	return nil
}

func (ra *RedisAdapter) Remove(ctx context.Context, key string) error {
	if ra.client == nil {
		return fmt.Errorf("%w (key=%q)", ErrRedisClientNotInitialized, key)
	}

	err := ra.client.Del(ctx, key)
	if err != nil {
		return fmt.Errorf("%w (operation=remove, key=%q): %w", ErrRedisOperation, key, err)
	}

	return nil
}

func (ra *RedisAdapter) Update(ctx context.Context, key string, value []byte) error {
	// For Redis, update is the same as set
	return ra.Set(ctx, key, value)
}

func (ra *RedisAdapter) Exists(ctx context.Context, key string) (bool, error) {
	if ra.client == nil {
		return false, fmt.Errorf("%w (key=%q)", ErrRedisClientNotInitialized, key)
	}

	count, err := ra.client.Exists(ctx, key)
	if err != nil {
		return false, fmt.Errorf("%w (operation=exists, key=%q): %w", ErrRedisOperation, key, err)
	}

	return count > 0, nil
}

// CacheRepository interface implementation.
func (ra *RedisAdapter) SetWithExpiration(
	ctx context.Context,
	key string,
	value []byte,
	expiration time.Duration,
) error {
	if ra.client == nil {
		return fmt.Errorf("%w (key=%q)", ErrRedisClientNotInitialized, key)
	}

	err := ra.client.Set(ctx, key, string(value), expiration)
	if err != nil {
		return fmt.Errorf(
			"%w (operation=set_with_expiration, key=%q): %w",
			ErrRedisOperation,
			key,
			err,
		)
	}

	return nil
}

func (ra *RedisAdapter) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	if ra.client == nil {
		return 0, fmt.Errorf("%w (key=%q)", ErrRedisClientNotInitialized, key)
	}

	ttl, err := ra.client.TTL(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("%w (operation=get_ttl, key=%q): %w", ErrRedisOperation, key, err)
	}

	return ttl, nil
}

func (ra *RedisAdapter) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if ra.client == nil {
		return fmt.Errorf("%w (key=%q)", ErrRedisClientNotInitialized, key)
	}

	err := ra.client.Expire(ctx, key, expiration)
	if err != nil {
		return fmt.Errorf("%w (operation=expire, key=%q): %w", ErrRedisOperation, key, err)
	}

	return nil
}

// RedisConnectionFactory creates Redis connections - following the same pattern as SQLConnectionFactory.
type RedisConnectionFactory struct {
	protocol string
}

// NewRedisConnectionFactory creates a new Redis connection factory for a specific protocol.
func NewRedisConnectionFactory(protocol string) *RedisConnectionFactory {
	return &RedisConnectionFactory{
		protocol: protocol,
	}
}

func (f *RedisConnectionFactory) CreateConnection(
	ctx context.Context,
	config *ConfigTarget,
) (Connection, error) {
	// Parse Redis-specific configuration
	host := config.Host
	if host == "" {
		host = "localhost"
	}

	port := config.Port
	if port == 0 {
		port = 6379
	}

	// For Redis, we can extract password from DSN if needed
	// For now, use empty password as default
	password := ""
	db := 0 // Default Redis DB

	// Create the connection
	conn := NewRedisConnection(f.protocol, host, port, password, db)

	return conn, nil
}

func (f *RedisConnectionFactory) GetProtocol() string {
	return f.protocol
}

func (f *RedisConnectionFactory) GetSupportedBehaviors() []ConnectionBehavior {
	return []ConnectionBehavior{
		ConnectionBehaviorStateful,
		ConnectionBehaviorStreaming,
		ConnectionBehaviorKeyValue,
	}
}
