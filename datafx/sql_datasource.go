package datafx

import (
	"context"
	"database/sql"
	"fmt"
)

type SqlDatasource struct {
	connection *sql.DB
	dialect    Dialect
}

func NewSqlDatasource(ctx context.Context, dialect Dialect, dsn string) (*SqlDatasource, error) {
	connection, err := sql.Open(string(dialect), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open datasource connection: %w", err)
	}

	if err := connection.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping datasource: %w", err)
	}

	return &SqlDatasource{
		connection: connection,
		dialect:    dialect,
	}, nil
}

func (datasource *SqlDatasource) GetDialect() Dialect {
	return datasource.dialect
}

func (datasource *SqlDatasource) GetConnection() SqlExecutor {
	return datasource.connection
}

func (datasource *SqlDatasource) ExecuteUnitOfWork(
	ctx context.Context,
	fn func(uow *UnitOfWork) error,
) error {
	uow := NewUnitOfWork(datasource.connection)

	return uow.Execute(ctx, fn)
}
