package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
)

// reporter records failures the visitor can't fix: an error log line and, when
// Sentry is configured, an event tagged with the same request id.
type reporter struct {
	log    *slog.Logger
	sentry *sentry.Client // nil when SENTRY_DSN isn't set
}

// internal reports err and answers with a generic 500.
func (rp *reporter) internal(w http.ResponseWriter, r *http.Request, err error) {
	rp.report(r, err)
	writeError(w, http.StatusInternalServerError, "internal", "Something went wrong.")
}

func (rp *reporter) report(r *http.Request, err error) {
	var reqID string
	userID := uuid.Nil
	if rl := requestLogFrom(r.Context()); rl != nil {
		reqID, userID = rl.id, rl.userID
	}
	rp.log.ErrorContext(r.Context(), "request failed", "err", err, "request_id", reqID)
	if rp.sentry == nil {
		return
	}
	hub := sentry.NewHub(rp.sentry, sentry.NewScope())
	hub.WithScope(func(s *sentry.Scope) {
		s.SetTag("route", routePattern(r))
		s.SetTag("method", r.Method)
		s.SetTag("request_id", reqID)
		if userID != uuid.Nil {
			s.SetUser(sentry.User{ID: userID.String()})
		}
		hub.CaptureException(err)
	})
}

// recoverPanics turns a panicking handler into a reported 500.
func recoverPanics(rp *reporter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				v := recover()
				if v == nil {
					return
				}
				if v == http.ErrAbortHandler {
					panic(v)
				}
				rp.internal(w, r, fmt.Errorf("panic: %v", v))
			}()
			next.ServeHTTP(w, r)
		})
	}
}
