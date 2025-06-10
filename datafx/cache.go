package datafx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/eser/ajan/connfx"
)

var (
	ErrCacheNotSupported = errors.New("connection does not support cache operations")
	ErrKeyExpired        = errors.New("key has expired")
)

// Cache provides high-level cache operations with expiration support.
type Cache struct {
	conn       connfx.Connection
	repository connfx.CacheRepository
}

// NewCache creates a new Cache instance from a connfx connection.
// The connection must support cache operations.
func NewCache(conn connfx.Connection) (*Cache, error) {
	if conn == nil {
		return nil, fmt.Errorf("%w: connection is nil", ErrConnectionNotSupported)
	}

	// Check if the connection supports cache operations
	behaviors := conn.GetBehaviors()
	supportsCache := slices.Contains(behaviors, connfx.ConnectionBehaviorCache)

	if !supportsCache {
		return nil, fmt.Errorf("%w: connection does not support cache operations (protocol=%q)",
			ErrCacheNotSupported, conn.GetProtocol())
	}

	// Get the cache repository from the raw connection
	repo, ok := conn.GetRawConnection().(connfx.CacheRepository)
	if !ok {
		return nil, fmt.Errorf(
			"%w: connection does not implement CacheRepository interface (protocol=%q)",
			ErrCacheNotSupported,
			conn.GetProtocol(),
		)
	}

	return &Cache{
		conn:       conn,
		repository: repo,
	}, nil
}

// Set stores a value with the given key and expiration time after marshaling it to JSON.
func (c *Cache) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%w (key=%q): %w", ErrFailedToMarshal, key, err)
	}

	if err := c.repository.SetWithExpiration(ctx, key, data, expiration); err != nil {
		return fmt.Errorf("failed to set cache key %q: %w", key, err)
	}

	return nil
}

// SetRaw stores raw bytes with the given key and expiration time.
func (c *Cache) SetRaw(
	ctx context.Context,
	key string,
	value []byte,
	expiration time.Duration,
) error {
	if err := c.repository.SetWithExpiration(ctx, key, value, expiration); err != nil {
		return fmt.Errorf("failed to set raw cache key %q: %w", key, err)
	}

	return nil
}

// Get retrieves a value by key and unmarshals it into the provided destination.
func (c *Cache) Get(ctx context.Context, key string, dest any) error {
	data, err := c.repository.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get cache key %q: %w", key, err)
	}

	if data == nil {
		return fmt.Errorf("%w (key=%q)", ErrKeyNotFound, key)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("%w (key=%q): %w", ErrFailedToUnmarshal, key, err)
	}

	return nil
}

// GetRaw retrieves raw bytes by key.
func (c *Cache) GetRaw(ctx context.Context, key string) ([]byte, error) {
	data, err := c.repository.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get raw cache key %q: %w", key, err)
	}

	if data == nil {
		return nil, fmt.Errorf("%w (key=%q)", ErrKeyNotFound, key)
	}

	return data, nil
}

// Delete removes a key from the cache.
func (c *Cache) Delete(ctx context.Context, key string) error {
	if err := c.repository.Remove(ctx, key); err != nil {
		return fmt.Errorf("failed to delete cache key %q: %w", key, err)
	}

	return nil
}

// Exists checks if a key exists in the cache.
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := c.repository.Exists(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to check if cache key %q exists: %w", key, err)
	}

	return exists, nil
}

// GetTTL returns the time-to-live for a key.
func (c *Cache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := c.repository.GetTTL(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL for cache key %q: %w", key, err)
	}

	return ttl, nil
}

// Expire sets an expiration time for an existing key.
func (c *Cache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if err := c.repository.Expire(ctx, key, expiration); err != nil {
		return fmt.Errorf("failed to set expiration for cache key %q: %w", key, err)
	}

	return nil
}

// GetConnection returns the underlying connfx connection.
func (c *Cache) GetConnection() connfx.Connection {
	return c.conn
}

// GetRepository returns the underlying cache repository.
func (c *Cache) GetRepository() connfx.CacheRepository {
	return c.repository
}
