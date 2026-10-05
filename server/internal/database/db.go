package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool opens a connection pool to the database at the given DSN.
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, dsn)
}
