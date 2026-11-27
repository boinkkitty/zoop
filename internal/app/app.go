// Package app is the composition root (ADR 0002): it builds the adapters,
// hands them to the core, and returns the HTTP handler.
package app

import (
	"context"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/boinkkitty/zoop/internal/auth"
	"github.com/boinkkitty/zoop/internal/clock"
	"github.com/boinkkitty/zoop/internal/google"
	"github.com/boinkkitty/zoop/internal/logging"
	"github.com/boinkkitty/zoop/internal/mail"
	"github.com/boinkkitty/zoop/internal/pages"
	"github.com/boinkkitty/zoop/internal/postgres"
	"github.com/boinkkitty/zoop/internal/ratelimit"
	"github.com/boinkkitty/zoop/internal/search"
	"github.com/boinkkitty/zoop/internal/server"
)

// Deps is the outside world: production passes the real ones, tests pass fakes.
type Deps struct {
	DB     *pgxpool.Pool
	Clock  clock.Clock
	Mailer mail.Mailer
	AppURL string         // where emailed links point, e.g. https://zoop.example
	Assets fs.FS          // the built frontend; must contain index.html
	Logs   io.Writer      // where JSON log lines go; nil discards them
	Sentry *sentry.Client // from errreport.NewClient; nil turns reporting off
	Google *google.Config // nil when Google sign-in isn't configured

	TrustedProxies int // proxies in front that append to X-Forwarded-For; see server.Deps
}

// Swept is how many expired rows one Sweep deleted.
type Swept = postgres.Swept

// Sweep deletes what has expired by the Clock: sessions, rate-limit rows and email tokens.
func Sweep(ctx context.Context, d Deps) (Swept, error) {
	return postgres.SweepExpired(ctx, d.DB, d.Clock.Now(ctx))
}

// KeepSwept runs Sweep now and then every interval until ctx ends, so no separate cron is needed.
// A failed sweep is logged and retried on the next tick.
func KeepSwept(ctx context.Context, d Deps, every time.Duration) {
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		if swept, err := Sweep(ctx, d); err != nil {
			slog.Error("sweep expired rows", "err", err)
		} else {
			slog.Info("swept expired rows", "sessions", swept.Sessions, "rateLimits", swept.RateLimits, "emailTokens", swept.EmailTokens)
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// New wires the whole app and returns its root handler.
func New(d Deps) http.Handler {
	var g auth.GoogleProvider // left nil, not a typed nil, when Google isn't configured
	if d.Google != nil {
		g = google.New(*d.Google, d.Clock)
	}
	logs := d.Logs
	if logs == nil {
		logs = io.Discard
	}
	limits := ratelimit.New(postgres.NewRateLimitStore(d.DB), d.Clock)
	return server.New(server.Deps{
		Log:    logging.New(logs),
		Sentry: d.Sentry,
		Auth: auth.New(auth.Deps{
			Store: postgres.NewAuthStore(d.DB), Clock: d.Clock, Limits: limits, Google: g,
			Mailer: d.Mailer, AppURL: d.AppURL,
		}),
		Pages:  pages.New(postgres.NewPageStore(d.DB), d.Clock),
		Search: search.New(postgres.NewSearchStore(d.DB)),
		Limits: limits,
		Ping:   postgres.Ping(d.DB),
		Assets: d.Assets,

		TrustedProxies: d.TrustedProxies,
	})
}
