package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/eser/ajan/configfx"
	"github.com/eser/ajan/connfx"
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
	appContext.Logger = logfx.NewLogger(
		logfx.WithWriter(os.Stdout),
		logfx.WithConfig(&appContext.Config.Log),
	)
	appContext.Logger.SetAsDefault()

	// metrics
	appContext.Metrics = metricsfx.NewMetricsProvider(appContext.Metrics)

	err = appContext.Metrics.RegisterNativeCollectors()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInitFailed, err)
	}

	// connections
	appContext.Connections = connfx.NewRegistryWithDefaults(appContext.Logger)

	err = appContext.Connections.LoadFromConfig(ctx, &appContext.Config.Conn)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInitFailed, err)
	}

	return appContext, nil
}
