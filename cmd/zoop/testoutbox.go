//go:build testclock

package main

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/boinkkitty/zoop/internal/mail"
	"github.com/boinkkitty/zoop/internal/mail/mailtest"
)

// withTestOutbox swaps the Mailer for an in-memory outbox that the Playwright tests read
// at GET /api/test/outbox?to=<email>, newest message first. Only in -tags testclock builds.
func withTestOutbox(mail.Mailer) (mail.Mailer, func(http.Handler) http.Handler) {
	slog.Warn("test outbox compiled in (-tags testclock): no email leaves this server")
	box := &mailtest.Fake{}
	return box, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/api/test/outbox" {
				next.ServeHTTP(w, r)
				return
			}
			to := r.URL.Query().Get("to")
			out := []mail.Message{}
			sent := box.Sent()
			for i := len(sent) - 1; i >= 0; i-- {
				if sent[i].To == to {
					out = append(out, sent[i])
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(out)
		})
	}
}
