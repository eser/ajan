package datafx

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/eser/ajan/logfx"
)

const DefaultDatasource = "default"

type SqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type Datasource interface {
	GetDialect() Dialect
	GetConnection() SqlExecutor
	ExecuteUnitOfWork(ctx context.Context, fn func(uow *UnitOfWork) error) error
}

type Registry struct {
	datasources map[string]Datasource
	logger      *logfx.Logger
}

func NewRegistry(logger *logfx.Logger) *Registry {
	datasources := make(map[string]Datasource)

	return &Registry{
		datasources: datasources,
		logger:      logger,
	}
}

func (registry *Registry) GetDefault() Datasource { //nolint:ireturn
	return registry.datasources[DefaultDatasource]
}

func (registry *Registry) GetNamed(name string) Datasource { //nolint:ireturn
	return registry.datasources[name]
}

func (registry *Registry) AddConnection(ctx context.Context, name string, provider string, dsn string) error {
	dialect, err := DetermineDialect(provider, dsn)
	if err != nil {
		return fmt.Errorf("failed to determine dialect for %q: %w", name, err)
	}

	registry.logger.Info(
		"adding datasource connection",
		slog.String("name", name),
		slog.String("dialect", string(dialect)),
	)

	// var db Datasource

	// var err error

	// if dialect == DialectPostgresPgx {
	// 	db, err = NewPgxDatasource(ctx, dialect, dsn)
	// } else {
	db, err := NewSqlDatasource(ctx, dialect, dsn) //nolint:varnamelen
	// }
	if err != nil {
		registry.logger.Error(
			"failed to open datasource connection",
			slog.String("error", err.Error()),
			slog.String("name", name),
		)

		return fmt.Errorf("failed to add connection for %q: %w", name, err)
	}

	registry.datasources[name] = db

	registry.logger.Info("successfully added datasource connection", slog.String("name", name))

	return nil
}

func (registry *Registry) LoadFromConfig(ctx context.Context, config *Config) error {
	for name, source := range config.Sources {
		nameLower := strings.ToLower(name)

		err := registry.AddConnection(ctx, nameLower, source.Provider, source.DSN)
		if err != nil {
			return fmt.Errorf("failed to add connection for %q: %w", nameLower, err)
		}
	}

	return nil
}
