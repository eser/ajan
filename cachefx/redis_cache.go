package cachefx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrFailedToParseRedisURL  = errors.New("failed to parse Redis URL")
	ErrFailedToConnectToRedis = errors.New("failed to connect to Redis")
	ErrFailedToSetCacheKey    = errors.New("failed to set cache key")
	ErrFailedToGetCacheKey    = errors.New("failed to get cache key")
	ErrFailedToDeleteCacheKey = errors.New("failed to delete cache key")
)

type RedisCache struct {
	clientError error
	client      *redis.Client

	dialect   Dialect
	hasPinged bool
}

func NewRedisCache(ctx context.Context, dialect Dialect, dsn string) *RedisCache {
	opt, err := redis.ParseURL(dsn)
	if err != nil {
		return &RedisCache{
			client: nil,
			clientError: fmt.Errorf(
				"%w (dialect=%q, dsn=%q): %w",
				ErrFailedToParseRedisURL,
				dialect,
				dsn,
				err,
			),
			hasPinged: false,

			dialect: dialect,
		}
	}

	return &RedisCache{
		client:      redis.NewClient(opt),
		clientError: nil,
		hasPinged:   false,

		dialect: dialect,
	}
}

func (cache *RedisCache) EnsureConnection(ctx context.Context) error {
	if cache.clientError != nil {
		return cache.clientError
	}

	if !cache.hasPinged {
		if err := cache.client.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("%w: %w", ErrFailedToConnectToRedis, err)
		}

		cache.hasPinged = true
	}

	return nil
}

func (cache *RedisCache) GetDialect() Dialect {
	return cache.dialect
}

func (cache *RedisCache) Set(
	ctx context.Context,
	key string,
	value any,
	expiration time.Duration,
) error {
	err := cache.client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToSetCacheKey, err)
	}

	return nil
}

func (cache *RedisCache) Get(ctx context.Context, key string) (string, error) {
	val, err := cache.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil // Key does not exist
	}

	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFailedToGetCacheKey, err)
	}

	return val, nil
}

func (cache *RedisCache) Delete(ctx context.Context, key string) error {
	err := cache.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToDeleteCacheKey, err)
	}

	return nil
}
