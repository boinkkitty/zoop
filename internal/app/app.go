// Package app is the composition root (ADR 0002): it builds the adapters,
// hands them to the core, and returns the HTTP handler.
package app

import (
	"io/fs"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/boinkkitty/zoop/internal/auth"
	"github.com/boinkkitty/zoop/internal/clock"
	"github.com/boinkkitty/zoop/internal/mail"
	"github.com/boinkkitty/zoop/internal/postgres"
	"github.com/boinkkitty/zoop/internal/server"
)

// Deps is the outside world: production passes the real ones, tests pass fakes.
type Deps struct {
	DB     *pgxpool.Pool
	Clock  clock.Clock
	Mailer mail.Mailer // unused until email verification (#33)
	Assets fs.FS       // the built frontend; must contain index.html
}

// New wires the whole app and returns its root handler.
func New(d Deps) http.Handler {
	return server.New(server.Deps{
		Auth:   auth.New(postgres.NewAuthStore(d.DB), d.Clock),
		Ping:   postgres.Ping(d.DB),
		Assets: d.Assets,
	})
}
