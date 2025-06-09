package datafx

import (
	"context"
	"database/sql"

	"github.com/eser/ajan/connfx"
)

type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type DataConnection interface {
	connfx.Connection
	// GetDB() *sql.DB
	// ExecuteUnitOfWork(ctx context.Context, fn func(uow *UnitOfWork) error) error
}
