// Package server is the HTTP adapter: it turns requests into calls on the core
// and the core's results into JSON, cookies and status codes (ADR 0002).
package server

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"

	"github.com/boinkkitty/zoop/internal/auth"
	"github.com/boinkkitty/zoop/internal/pages"
	"github.com/boinkkitty/zoop/internal/ratelimit"
)

// Deps is what the router drives. internal/app builds it.
type Deps struct {
	Auth   *auth.Service
	Pages  *pages.Service
	Limits *ratelimit.RateLimiter
	Ping   func(context.Context) error // backs /api/healthz
	Assets fs.FS                       // the built frontend; must contain index.html
	Log    *slog.Logger                // allowlisting; see internal/logging
	Sentry *sentry.Client              // nil turns error reporting off
}

// New returns the app's root handler.
func New(d Deps) http.Handler {
	authn := d.Auth
	rep := &reporter{log: d.Log, sentry: d.Sentry}
	r := chi.NewRouter()
	r.Use(accessLog(d.Log), recoverPanics(rep))

	// CSRF: browsers label cross-site requests, so state-changing ones from other sites are refused.
	csrf := http.NewCrossOriginProtection()
	csrf.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusForbidden, "cross_origin", "Cross-origin requests aren't allowed.")
	}))
	r.Use(csrf.Handler)

	r.Route("/api", func(api chi.Router) {
		api.Get("/healthz", healthz(d.Ping))
		api.Route("/v1", func(v1 chi.Router) {
			v1.Post("/auth/signup", signup(authn, rep))
			v1.Post("/auth/login", login(authn, rep))
			v1.Post("/auth/logout", logout(authn, rep))
			v1.Post("/auth/verify", verifyEmail(authn, rep))
			v1.Post("/auth/verify/resend", resendVerify(authn, rep))
			v1.Post("/auth/password/forgot", forgotPassword(authn, rep))
			v1.Post("/auth/password/reset", resetPassword(authn, rep))
			v1.Get("/auth/google/start", googleStart(authn, rep))
			v1.Get("/auth/google/callback", googleCallback(authn, rep))
			v1.Group(func(signedIn chi.Router) {
				signedIn.Use(requireUser(authn, rep), limitPerUser(d.Limits, rep))
				signedIn.Get("/me", getMe(authn, rep))
				signedIn.Get("/pages/{day}/{notebook}", getPage(d.Pages, rep))
				signedIn.Put("/pages/{day}/{notebook}", putPage(d.Pages, rep))
			})
		})
		api.NotFound(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, http.StatusNotFound, "not_found", "No such endpoint.")
		})
	})
	r.NotFound(appShell(d.Assets))
	return r
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
