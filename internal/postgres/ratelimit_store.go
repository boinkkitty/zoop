package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/boinkkitty/zoop/internal/db/dbgen"
	"github.com/boinkkitty/zoop/internal/ratelimit"
)

// RateLimitStore implements ratelimit.Store.
type RateLimitStore struct{ q *dbgen.Queries }

var _ ratelimit.Store = (*RateLimitStore)(nil)

func NewRateLimitStore(pool *pgxpool.Pool) *RateLimitStore {
	return &RateLimitStore{q: dbgen.New(pool)}
}

func (s *RateLimitStore) Hit(ctx context.Context, key string, start, expiresAt time.Time) (int, error) {
	n, err := s.q.AddRateLimitHit(ctx, dbgen.AddRateLimitHitParams{Key: key, WindowStart: start, ExpiresAt: expiresAt})
	return int(n), err
}

func (s *RateLimitStore) Count(ctx context.Context, key string, start time.Time) (int, error) {
	n, err := s.q.GetRateLimitCount(ctx, dbgen.GetRateLimitCountParams{Key: key, WindowStart: start})
	return int(n), err
}
