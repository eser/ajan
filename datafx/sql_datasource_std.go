package datafx

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type SqlDatasourceStd struct {
	connection *sql.DB
	dialect    Dialect
}

func NewSqlDatasourceStd(ctx context.Context, dialect Dialect, dsn string) (*SqlDatasourceStd, error) {
	connection, err := sql.Open(string(dialect), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := connection.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &SqlDatasourceStd{
		connection: connection,
		dialect:    dialect,
	}, nil
}

func (dataSource *SqlDatasourceStd) GetDialect() Dialect {
	return dataSource.dialect
}

func (dataSource *SqlDatasourceStd) GetConnection() SqlExecutor { //nolint:ireturn
	return dataSource.connection
}

func (dataSource *SqlDatasourceStd) UseUnitOfWork(ctx context.Context) (*UnitOfWork, error) {
	uow, err := UseUnitOfWork(ctx, dataSource.connection)
	if err != nil {
		return &UnitOfWork{}, err
	}

	return uow, err
}
