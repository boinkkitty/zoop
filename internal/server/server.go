// Package server builds the HTTP router: the JSON API plus the app shell.
package server

import (
	"context"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/boinkkitty/zoop/internal/auth"
	"github.com/boinkkitty/zoop/internal/clock"
	"github.com/boinkkitty/zoop/internal/mail"
)

// Deps is everything the router needs from the outside world.
type Deps struct {
	DB     *pgxpool.Pool
	Clock  clock.Clock
	Mailer mail.Mailer
	Assets fs.FS // the built frontend; must contain index.html
}

// New returns the app's root handler.
func New(d Deps) http.Handler {
	authn := auth.New(d.DB, d.Clock)
	r := chi.NewRouter()
	r.Route("/api", func(api chi.Router) {
		api.Get("/healthz", healthz(d.DB))
		api.Route("/v1", func(v1 chi.Router) {
			v1.Post("/auth/signup", signup(authn))
			v1.Post("/auth/login", login(authn))
			v1.Post("/auth/logout", logout(authn))
			v1.Group(func(signedIn chi.Router) {
				signedIn.Use(requireUser(authn))
				signedIn.Get("/me", getMe(authn))
			})
		})
		api.NotFound(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, http.StatusNotFound, "not_found", "No such endpoint.")
		})
	})
	r.NotFound(appShell(d.Assets))

	// CSRF: browsers label cross-site requests, so state-changing ones from other sites are refused.
	csrf := http.NewCrossOriginProtection()
	csrf.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusForbidden, "cross_origin", "Cross-origin requests aren't allowed.")
	}))
	return csrf.Handler(r)
}

// appShell serves a built file when the path names one, and otherwise
// index.html so the client router can handle the path.
func appShell(assets fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if info, err := fs.Stat(assets, name); err == nil && !info.IsDir() && name != "index.html" {
			http.ServeFileFS(w, r, assets, name)
			return
		}
		page, err := fs.ReadFile(assets, "index.html")
		if err != nil {
			http.Error(w, "app shell missing", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page)
	}
}

// healthz returns a bare 200 if SELECT 1 succeeds within ~1s.
func healthz(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		var one int
		if err := pool.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
