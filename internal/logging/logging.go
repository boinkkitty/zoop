// Package logging builds the app's logger: slog JSON lines that carry only an
// allowlist of fields, so the operator can debug without seeing anyone's writing.
package logging

import (
	"io"
	"log/slog"
)

// allowed are the only keys a log line keeps. Every other attribute is dropped,
// so a stray "email" or "body" field never reaches the logs. Sensitive values
// that do get passed under an allowed key still print as [redacted] through
// their LogValue (auth.Email, auth.Token, auth.Password, pages.Content).
var allowed = map[string]bool{
	slog.TimeKey:    true,
	slog.LevelKey:   true,
	slog.MessageKey: true,
	"method":        true,
	"route":         true, // the route pattern, never the raw URL
	"status":        true,
	"latency_ms":    true,
	"user_id":       true,
	"request_id":    true,
	"err":           true,
}

// New returns a logger writing JSON lines to w.
func New(w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) > 0 || !allowed[a.Key] {
				return slog.Attr{}
			}
			return a
		},
	}))
}
