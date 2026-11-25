// Package mail sends the app's emails (verify, reset, deactivation).
package mail

import (
	"context"
	"errors"
)

// Message is one plain email.
type Message struct {
	To      string
	Subject string
	Text    string
}

// Mailer sends email. Production uses Resend; tests use mailtest.Fake.
type Mailer interface {
	Send(ctx context.Context, m Message) error
}

// Unconfigured is the Mailer used until a real provider is wired in (Resend, #33).
// It refuses to send rather than silently dropping mail.
type Unconfigured struct{}

func (Unconfigured) Send(context.Context, Message) error {
	return errors.New("mail: no mailer configured")
}
