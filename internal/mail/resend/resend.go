// Package resend is the Mailer adapter for Resend, over its official Go SDK.
package resend

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	sdk "github.com/resend/resend-go/v3"

	"github.com/boinkkitty/zoop/internal/mail"
)

// Config is a Resend account to send from.
type Config struct {
	APIKey  string
	From    string // a sender on a domain verified in Resend, e.g. "Zoop <hello@zoop.app>"
	BaseURL string // empty for Resend's own API; tests point it at a fake
}

// Mailer sends through Resend.
type Mailer struct {
	client *sdk.Client
	from   string
}

var _ mail.Mailer = (*Mailer)(nil)

func New(c Config) *Mailer {
	client := sdk.NewClient(c.APIKey)
	if c.BaseURL != "" {
		u, err := url.Parse(strings.TrimSuffix(c.BaseURL, "/") + "/")
		if err != nil {
			panic(fmt.Sprintf("resend: bad base URL %q: %v", c.BaseURL, err))
		}
		client.BaseURL = u
	}
	return &Mailer{client: client, from: c.From}
}

func (m *Mailer) Send(ctx context.Context, msg mail.Message) error {
	_, err := m.client.Emails.SendWithContext(ctx, &sdk.SendEmailRequest{
		From:    m.from,
		To:      []string{msg.To},
		Subject: msg.Subject,
		Text:    msg.Text,
	})
	if err != nil {
		return fmt.Errorf("resend: %w", err)
	}
	return nil
}
