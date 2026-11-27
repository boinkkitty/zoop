package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Ping reports whether the database answers SELECT 1. It backs /api/healthz.
func Ping(pool *pgxpool.Pool) func(context.Context) error {
	return func(ctx context.Context) error {
		var one int
		return pool.QueryRow(ctx, "SELECT 1").Scan(&one)
	}
}
