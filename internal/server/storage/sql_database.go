package storage

import (
	"context"
	"database/sql"
)

// SF TODO
type SQLExecutor interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// SF TODO
type SQLTransaction interface {
	SQLExecutor

	Commit() error
	Rollback() error
}

// SF TODO
var _ SQLTransaction = (*sql.Tx)(nil)

// Интерфейс базы данных
type SQLDatabase interface {
	SQLExecutor

	PingContext(ctx context.Context) error
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
	Close() error
}

// SF TODO
var _ SQLDatabase = (*sql.DB)(nil)
