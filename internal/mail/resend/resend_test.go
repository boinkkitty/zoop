package resend_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/boinkkitty/zoop/internal/mail"
	"github.com/boinkkitty/zoop/internal/mail/resend"
)

// fakeResend stands in for api.resend.com, answering every request with status and body.
type fakeResend struct {
	*httptest.Server
	method, path, auth string
	email              map[string]any
}

func startFakeResend(t *testing.T, status int, body string) *fakeResend {
	t.Helper()
	f := &fakeResend{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.method, f.path, f.auth = r.Method, r.URL.Path, r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &f.email)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(f.Close)
	return f
}

func TestSendPostsTheEmailToResend(t *testing.T) {
	api := startFakeResend(t, http.StatusOK, `{"id":"49a3999c-0ce1-4ea6-ab68-afcd6dc2e794"}`)
	m := resend.New(resend.Config{APIKey: "re_test_key", From: "Zoop <hello@zoop.test>", BaseURL: api.URL})

	err := m.Send(t.Context(), mail.Message{To: "ada@example.com", Subject: "Verify your email", Text: "Open this link"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if api.method != http.MethodPost || api.path != "/emails" {
		t.Errorf("request = %s %s, want POST /emails", api.method, api.path)
	}
	if api.auth != "Bearer re_test_key" {
		t.Errorf("Authorization = %q, want Bearer re_test_key", api.auth)
	}
	want := map[string]any{
		"from":    "Zoop <hello@zoop.test>",
		"to":      []any{"ada@example.com"},
		"subject": "Verify your email",
		"text":    "Open this link",
	}
	for k, v := range want {
		if got, _ := json.Marshal(api.email[k]); string(got) != mustJSON(t, v) {
			t.Errorf("email %s = %s, want %s", k, got, mustJSON(t, v))
		}
	}
}

func TestSendFailsWhenResendRefuses(t *testing.T) {
	api := startFakeResend(t, http.StatusUnprocessableEntity,
		`{"statusCode":422,"name":"validation_error","message":"Invalid from field."}`)
	m := resend.New(resend.Config{APIKey: "re_test_key", From: "nope", BaseURL: api.URL})

	if err := m.Send(t.Context(), mail.Message{To: "ada@example.com", Subject: "s", Text: "t"}); err == nil {
		t.Fatal("Send succeeded although Resend answered 422")
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
