package queuefx

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/eser/ajan/logfx"
)

const DefaultBroker = "default"

type Broker interface {
	GetDialect() Dialect

	QueueDeclare(ctx context.Context, name string) (string, error)
	Publish(ctx context.Context, name string, body []byte) error
	// Consume(ctx context.Context, name string)
}

type Registry struct {
	brokers map[string]Broker
	logger  *logfx.Logger
}

func NewRegistry(logger *logfx.Logger) *Registry {
	brokers := make(map[string]Broker)

	return &Registry{
		brokers: brokers,
		logger:  logger,
	}
}

func (registry *Registry) GetDefault() Broker {
	return registry.brokers[DefaultBroker]
}

func (registry *Registry) GetNamed(name string) Broker {
	if db, exists := registry.brokers[name]; exists {
		return db
	}

	return nil
}

func (registry *Registry) AddConnection(ctx context.Context, name string, provider string, dsn string) error {
	dialect, err := DetermineDialect(provider, dsn)
	if err != nil {
		return fmt.Errorf("failed to determine dialect for %q: %w", name, err)
	}

	registry.logger.Info(
		"adding broker connection",
		slog.String("name", name),
		slog.String("dialect", string(dialect)),
	)

	db, err := NewAmqpBroker(ctx, dialect, dsn) //nolint:varnamelen
	if err != nil {
		registry.logger.Error(
			"failed to open broker connection",
			slog.String("error", err.Error()),
			slog.String("name", name),
		)

		return fmt.Errorf("failed to add connection for %q: %w", name, err)
	}

	registry.brokers[name] = db

	registry.logger.Info("successfully added broker connection", slog.String("name", name))

	return nil
}

func (registry *Registry) LoadFromConfig(ctx context.Context, config *Config) error {
	for name, source := range config.Brokers {
		nameLower := strings.ToLower(name)

		err := registry.AddConnection(ctx, nameLower, source.Provider, source.DSN)
		if err != nil {
			return fmt.Errorf("failed to add connection for %q: %w", nameLower, err)
		}
	}

	return nil
}
