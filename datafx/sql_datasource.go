package datafx

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
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

func (dataSource *SqlDatasource) GetDialect() Dialect {
	return dataSource.dialect
}

func (dataSource *SqlDatasource) GetConnection() SqlExecutor { //nolint:ireturn
	return dataSource.connection
}

func (dataSource *SqlDatasource) UseUnitOfWork(ctx context.Context) (*UnitOfWork, error) {
	uow, err := UseUnitOfWork(ctx, dataSource.connection)
	if err != nil {
		return &UnitOfWork{}, err
	}

	return uow, err
}
