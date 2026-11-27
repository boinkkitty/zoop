package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/boinkkitty/zoop/internal/db/dbgen"
)

// SweepExpired deletes the sessions and rate-limit rows that have expired at now,
// returning how many of each it deleted. Email tokens join the sweep in #33.
func SweepExpired(ctx context.Context, pool *pgxpool.Pool, now time.Time) (sessions, rateLimits int64, err error) {
	q := dbgen.New(pool)
	if sessions, err = q.DeleteExpiredSessions(ctx, now); err != nil {
		return 0, 0, fmt.Errorf("delete expired sessions: %w", err)
	}
	if rateLimits, err = q.DeleteExpiredRateLimits(ctx, now); err != nil {
		return sessions, 0, fmt.Errorf("delete expired rate limits: %w", err)
	}
	return sessions, rateLimits, nil
}
