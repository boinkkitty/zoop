package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// requestLog is what this request's access line will say, filled in as the request runs.
type requestLog struct {
	id     string
	userID uuid.UUID // uuid.Nil until requireUser finds a session
}

type requestLogKey struct{}

func requestLogFrom(ctx context.Context) *requestLog {
	rl, _ := ctx.Value(requestLogKey{}).(*requestLog)
	return rl
}

// accessLog writes one line per request with only the allowlisted fields. It
// runs inside the chi router so it can log the matched route pattern rather
// than the raw URL, which can hold dates, search queries and tokens.
func accessLog(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rl := &requestLog{id: newRequestID()}
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r.WithContext(context.WithValue(r.Context(), requestLogKey{}, rl)))

			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("route", routePattern(r)),
				slog.Int("status", sw.status),
				slog.Float64("latency_ms", float64(time.Since(start).Microseconds())/1000),
				slog.String("request_id", rl.id),
			}
			if rl.userID != uuid.Nil {
				attrs = append(attrs, slog.String("user_id", rl.userID.String()))
			}
			log.LogAttrs(r.Context(), slog.LevelInfo, "request", attrs...)
		})
	}
}

// routePattern is the chi pattern the request matched, such as /api/v1/me.
func routePattern(r *http.Request) string {
	if rc := chi.RouteContext(r.Context()); rc != nil {
		if p := rc.RoutePattern(); p != "" {
			return p
		}
	}
	return "unmatched"
}

func newRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// statusWriter remembers the status code the handler sent.
type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.status, w.wroteHeader = code, true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(p []byte) (int, error) {
	w.wroteHeader = true
	return w.ResponseWriter.Write(p)
}

// Unwrap lets http.ResponseController reach the real writer.
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
