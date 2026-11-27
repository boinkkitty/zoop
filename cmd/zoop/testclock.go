//go:build testclock

// The test-only clock override that drives the Playwright 02:00 lock test.
// It exists only in binaries built with -tags testclock, never in production.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/boinkkitty/zoop/internal/clock"
)

// offsetCookie holds how many milliseconds this browser's "now" is ahead of the real one.
const offsetCookie = "zoop_test_now_offset"

type offsetKey struct{}

// shiftedClock is base, moved per request by the offset its browser asked for.
type shiftedClock struct{ base clock.Clock }

func (c shiftedClock) Now(ctx context.Context) time.Time {
	d, _ := ctx.Value(offsetKey{}).(time.Duration)
	return c.base.Now(ctx).Add(d)
}

// withTestClock returns the shifted clock and the middleware that reads each request's offset.
// Every response says the override is on, so the e2e test can tell.
func withTestClock(c clock.Clock) (clock.Clock, func(http.Handler) http.Handler) {
	slog.Warn("test clock override compiled in (-tags testclock): never run this build in production")
	return shiftedClock{c}, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Zoop-Test-Clock", "on")
			if ck, err := r.Cookie(offsetCookie); err == nil {
				if ms, err := strconv.ParseInt(ck.Value, 10, 64); err == nil {
					d := time.Duration(ms) * time.Millisecond
					r = r.WithContext(context.WithValue(r.Context(), offsetKey{}, d))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
