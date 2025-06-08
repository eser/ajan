package datafx

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrFailedToOpenDatasourceConnection = errors.New("failed to open datasource connection")
	ErrFailedToPingDatasource           = errors.New("failed to ping datasource")
)

type SQLDatasource struct {
	connection *sql.DB
	dialect    Dialect
}

func NewSQLDatasource(ctx context.Context, dialect Dialect, dsn string) (*SQLDatasource, error) {
	connection, err := sql.Open(string(dialect), dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToOpenDatasourceConnection, err)
	}

	if err := connection.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToPingDatasource, err)
	}

	return &SQLDatasource{
		connection: connection,
		dialect:    dialect,
	}, nil
}

func (datasource *SQLDatasource) GetDialect() Dialect {
	return datasource.dialect
}

func (datasource *SQLDatasource) GetConnection() SQLExecutor {
	return datasource.connection
}

func (datasource *SQLDatasource) ExecuteUnitOfWork(
	ctx context.Context,
	fn func(uow *UnitOfWork) error,
) error {
	uow := NewUnitOfWork(datasource.connection)

	return uow.Execute(ctx, fn)
}
