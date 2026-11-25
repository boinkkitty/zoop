// Package mailtest provides an in-memory Mailer for tests.
package mailtest

import (
	"context"
	"sync"

	"github.com/boinkkitty/zoop/internal/mail"
)

// Fake records every message instead of sending it.
type Fake struct {
	mu   sync.Mutex
	sent []mail.Message
}

func (f *Fake) Send(_ context.Context, m mail.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, m)
	return nil
}

// Sent returns a copy of every message sent so far, oldest first.
func (f *Fake) Sent() []mail.Message {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]mail.Message(nil), f.sent...)
}
