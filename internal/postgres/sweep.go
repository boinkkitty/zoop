package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/boinkkitty/zoop/internal/db/dbgen"
)

// Swept is how many expired rows of each kind one sweep deleted.
type Swept struct {
	Sessions    int64
	RateLimits  int64
	EmailTokens int64
}

// SweepExpired deletes the sessions, rate-limit rows and email tokens that have expired at now.
func SweepExpired(ctx context.Context, pool *pgxpool.Pool, now time.Time) (Swept, error) {
	q := dbgen.New(pool)
	var s Swept
	var err error
	if s.Sessions, err = q.DeleteExpiredSessions(ctx, now); err != nil {
		return s, fmt.Errorf("delete expired sessions: %w", err)
	}
	if s.RateLimits, err = q.DeleteExpiredRateLimits(ctx, now); err != nil {
		return s, fmt.Errorf("delete expired rate limits: %w", err)
	}
	if s.EmailTokens, err = q.DeleteExpiredEmailTokens(ctx, now); err != nil {
		return s, fmt.Errorf("delete expired email tokens: %w", err)
	}
	return s, nil
}
