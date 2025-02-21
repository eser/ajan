package cachefx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client  *redis.Client
	dialect Dialect
}

func NewRedisCache(ctx context.Context, dialect Dialect, dsn string) (*RedisCache, error) {
	opt, err := redis.ParseURL(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test the connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client:  client,
		dialect: dialect,
	}, nil
}

func (cache *RedisCache) GetDialect() Dialect {
	return cache.dialect
}

func (cache *RedisCache) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	err := cache.client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache key: %w", err)
	}

	return nil
}

func (cache *RedisCache) Get(ctx context.Context, key string) (string, error) {
	val, err := cache.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil // Key does not exist
	}

	if err != nil {
		return "", fmt.Errorf("failed to get cache key: %w", err)
	}

	return val, nil
}

func (cache *RedisCache) Delete(ctx context.Context, key string) error {
	err := cache.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete cache key: %w", err)
	}

	return nil
}
