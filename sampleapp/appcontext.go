package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/eser/ajan/configfx"
	"github.com/eser/ajan/connfx"
	"github.com/eser/ajan/connfx/adapters"
	"github.com/eser/ajan/logfx"
	"github.com/eser/ajan/metricsfx"
	_ "modernc.org/sqlite"
)

var ErrInitFailed = errors.New("failed to initialize app context")

type AppContext struct {
	Config      *AppConfig
	Logger      *logfx.Logger
	Metrics     *metricsfx.MetricsProvider
	Connections *connfx.Registry
}

func NewAppContext(ctx context.Context) (*AppContext, error) {
	appContext := &AppContext{} //nolint:exhaustruct

	// config
	cl := configfx.NewConfigManager()

	appContext.Config = &AppConfig{} //nolint:exhaustruct

	err := cl.LoadDefaults(appContext.Config)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInitFailed, err)
	}

	// logger
	appContext.Logger = logfx.NewLoggerAsDefault(os.Stdout, &appContext.Config.Log)

	// metrics
	appContext.Metrics = metricsfx.NewMetricsProvider()

	err = appContext.Metrics.RegisterNativeCollectors()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInitFailed, err)
	}

	// connections
	appContext.Connections = connfx.NewRegistry(appContext.Logger)

	// Register factory for SQLite
	sqliteFactory := adapters.NewSQLConnectionFactory("sqlite")
	appContext.Connections.RegisterFactory(sqliteFactory)

	// Register factory for Redis (cache operations)
	redisFactory := adapters.NewRedisFactory()
	appContext.Connections.RegisterFactory(redisFactory)

	// Register factory for AMQP (queue operations)
	amqpFactory := adapters.NewAMQPFactory()
	appContext.Connections.RegisterFactory(amqpFactory)

	err = appContext.Connections.LoadFromConfig(ctx, &appContext.Config.Conn)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInitFailed, err)
	}

	return appContext, nil
}
