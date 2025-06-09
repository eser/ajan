package cachefx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/eser/ajan/logfx"
)

var (
	ErrFailedToDetermineDialect = errors.New("failed to determine dialect")
	ErrFailedToAddConnection    = errors.New("failed to add connection")
)

const DefaultCache = "default"

type Cache interface {
	GetDialect() Dialect

	Set(ctx context.Context, key string, value any, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
}

type Registry struct {
	caches map[string]Cache
	logger *logfx.Logger
}

func NewRegistry(logger *logfx.Logger) *Registry {
	caches := make(map[string]Cache)

	return &Registry{
		caches: caches,
		logger: logger,
	}
}

func (registry *Registry) GetDefault() Cache {
	return registry.caches[DefaultCache]
}

func (registry *Registry) GetNamed(name string) Cache {
	return registry.caches[name]
}

func (registry *Registry) AddConnection(
	ctx context.Context,
	name string,
	provider string,
	dsn string,
) error {
	dialect, err := DetermineDialect(provider, dsn)
	if err != nil {
		return fmt.Errorf("%w (name=%q): %w", ErrFailedToDetermineDialect, name, err)
	}

	registry.logger.Info(
		"adding cache connection",
		slog.String("name", name),
		slog.String("dialect", string(dialect)),
	)

	cache := NewRedisCache(ctx, dialect, dsn)
	if err := cache.EnsureConnection(ctx); err != nil {
		registry.logger.Error(
			"failed to open cache connection",
			slog.String("error", err.Error()),
			slog.String("name", name),
			slog.String("dialect", string(dialect)),
			slog.String("dsn", dsn),
		)

		return fmt.Errorf(
			"%w (name=%q, dialect=%q, dsn=%q): %w",
			ErrFailedToAddConnection,
			name,
			dialect,
			dsn,
			err,
		)
	}

	registry.caches[name] = cache

	registry.logger.Info("successfully added cache connection", slog.String("name", name))

	return nil
}

func (registry *Registry) LoadFromConfig(ctx context.Context, config *Config) error {
	for name, source := range config.Caches {
		nameLower := strings.ToLower(name)

		err := registry.AddConnection(ctx, nameLower, source.Provider, source.DSN)
		if err != nil {
			return fmt.Errorf("%w (name=%q): %w", ErrFailedToAddConnection, nameLower, err)
		}
	}

	return nil
}
