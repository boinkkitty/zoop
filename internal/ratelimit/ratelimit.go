// Package ratelimit slows down repeated attempts with fixed-window counters.
// It never locks anyone out: a refused caller is told when the window ends.
package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/boinkkitty/zoop/internal/clock"
)

// Limit is at most Max attempts per subject in each Window.
type Limit struct {
	Name   string
	Max    int
	Window time.Duration
}

var (
	LoginFailuresPerEmail = Limit{"login-email", 5, 15 * time.Minute}
	LoginFailuresPerIP    = Limit{"login-ip", 20, 15 * time.Minute}
	SignupsPerIP          = Limit{"signup-ip", 5, time.Hour}
	APIRequestsPerUser    = Limit{"api-user", 300, time.Minute}

	// EmailRequestsPerEmail and EmailRequestsPerIP cover forgot password and resend verification together.
	// They're wired up with those endpoints in #33.
	EmailRequestsPerEmail = Limit{"email-request-email", 3, time.Hour}
	EmailRequestsPerIP    = Limit{"email-request-ip", 10, time.Hour}
)

// Store is the port ratelimit needs: counters keyed by (key, window start). internal/postgres implements it.
type Store interface {
	// Hit adds one to key's count in the window starting at start, keeping it until expiresAt, and returns the new count.
	Hit(ctx context.Context, key string, start, expiresAt time.Time) (int, error)
	// Count is key's count in the window starting at start, 0 if nothing was counted.
	Count(ctx context.Context, key string, start time.Time) (int, error)
}

// ExceededError refuses an attempt until RetryAfter has passed.
type ExceededError struct{ RetryAfter time.Duration }

func (e *ExceededError) Error() string {
	return fmt.Sprintf("ratelimit: try again in %s", e.RetryAfter)
}

// RateLimiter applies Limits, counting in its Store and telling time by its Clock.
type RateLimiter struct {
	store Store
	clock clock.Clock
}

func New(store Store, c clock.Clock) *RateLimiter {
	return &RateLimiter{store: store, clock: c}
}

// Check returns an ExceededError if subject has already used up l in this window, without counting anything.
func (r *RateLimiter) Check(ctx context.Context, l Limit, subject string) error {
	now := r.clock.Now()
	start := now.Truncate(l.Window)
	n, err := r.store.Count(ctx, key(l, subject), start)
	if err != nil {
		return err
	}
	return refuseOver(l, n, start, now)
}

// Take counts this attempt against l for subject, and returns an ExceededError if it's one too many.
func (r *RateLimiter) Take(ctx context.Context, l Limit, subject string) error {
	now := r.clock.Now()
	start := now.Truncate(l.Window)
	n, err := r.store.Hit(ctx, key(l, subject), start, start.Add(l.Window))
	if err != nil {
		return err
	}
	return refuseOver(l, n-1, start, now)
}

// Record counts one attempt against l for subject.
func (r *RateLimiter) Record(ctx context.Context, l Limit, subject string) error {
	start := r.clock.Now().Truncate(l.Window)
	_, err := r.store.Hit(ctx, key(l, subject), start, start.Add(l.Window))
	return err
}

func refuseOver(l Limit, n int, start, now time.Time) error {
	if n < l.Max {
		return nil
	}
	return &ExceededError{RetryAfter: start.Add(l.Window).Sub(now)}
}

func key(l Limit, subject string) string { return l.Name + ":" + subject }
