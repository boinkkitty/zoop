//go:build !testclock

package main

import (
	"net/http"

	"github.com/boinkkitty/zoop/internal/mail"
)

// withTestOutbox is a no-op in production builds: the outbox is compiled out (see testoutbox.go).
func withTestOutbox(m mail.Mailer) (mail.Mailer, func(http.Handler) http.Handler) {
	return m, func(h http.Handler) http.Handler { return h }
}
