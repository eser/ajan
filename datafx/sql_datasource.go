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

	dialect Dialect
	dsn     string
}

func NewSQLDatasource(ctx context.Context, dialect Dialect, dsn string) *SQLDatasource {
	return &SQLDatasource{
		connection: nil,

		dialect: dialect,
		dsn:     dsn,
	}
}

func (datasource *SQLDatasource) EnsureConnection(ctx context.Context) error {
	if datasource.connection != nil {
		return nil
	}

	connection, err := sql.Open(string(datasource.dialect), datasource.dsn)
	if err != nil {
		return fmt.Errorf(
			"%w (dialect=%q, dsn=%q): %w",
			ErrFailedToOpenDatasourceConnection,
			datasource.dialect,
			datasource.dsn,
			err,
		)
	}

	if err := connection.PingContext(ctx); err != nil {
		return fmt.Errorf(
			"%w (dialect=%q, dsn=%q): %w",
			ErrFailedToPingDatasource,
			datasource.dialect,
			datasource.dsn,
			err,
		)
	}

	datasource.connection = connection

	return nil
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
