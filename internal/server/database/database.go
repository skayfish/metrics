package database

import (
	"context"
	"database/sql"
)

// SF TODO
type Database interface {
	// SF TODO
	PingContext(ctx context.Context) error

	// SF TODO
	Query(query string, args ...interface{}) (*sql.Rows, error)

	// SF TODO
	QueryRow(query string, args ...interface{}) *sql.Row

	// SF TODO
	Exec(query string, args ...interface{}) (sql.Result, error)
}
