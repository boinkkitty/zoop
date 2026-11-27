// Package server is the HTTP adapter: it turns requests into calls on the core
// and the core's results into JSON, cookies and status codes (ADR 0002).
package server

import (
	"context"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/boinkkitty/zoop/internal/auth"
)

// Deps is what the router drives. internal/app builds it.
type Deps struct {
	Auth   *auth.Service
	Ping   func(context.Context) error // backs /api/healthz
	Assets fs.FS                       // the built frontend; must contain index.html
}

// New returns the app's root handler.
func New(d Deps) http.Handler {
	authn := d.Auth
	r := chi.NewRouter()
	r.Route("/api", func(api chi.Router) {
		api.Get("/healthz", healthz(d.Ping))
		api.Route("/v1", func(v1 chi.Router) {
			v1.Post("/auth/signup", signup(authn))
			v1.Post("/auth/login", login(authn))
			v1.Post("/auth/logout", logout(authn))
			v1.Get("/auth/google/start", googleStart(authn))
			v1.Get("/auth/google/callback", googleCallback(authn))
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
func healthz(ping func(context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if err := ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
