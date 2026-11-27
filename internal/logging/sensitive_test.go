package logging_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/boinkkitty/zoop/internal/auth"
	"github.com/boinkkitty/zoop/internal/pages"
)

// Even a careless log line that names a sensitive value outright prints [redacted].
func TestSensitiveValuesRedactThemselvesWhenLogged(t *testing.T) {
	const (
		email    = "ada@example.com"
		token    = "tok_s3cr3t-session-token"
		password = "correct horse battery staple"
		page     = `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Dear diary, today I cried"}]}]}`
	)
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	log.Info("oops",
		"email", auth.Email(email),
		"token", auth.Token(token),
		"password", auth.Password(password),
		"page", pages.Content(page),
	)

	out := buf.String()
	for _, secret := range []string{email, token, password, "Dear diary"} {
		if strings.Contains(out, secret) {
			t.Errorf("log line contains %q: %s", secret, out)
		}
	}
	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("log line %q is not JSON: %v", out, err)
	}
	for _, key := range []string{"email", "token", "password", "page"} {
		if line[key] != "[redacted]" {
			t.Errorf("%s logged as %v, want [redacted]", key, line[key])
		}
	}
}
