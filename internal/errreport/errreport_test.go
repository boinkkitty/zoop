package errreport_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"

	"github.com/boinkkitty/zoop/internal/errreport"
)

type recorder struct{ events []*sentry.Event }

func (r *recorder) Configure(sentry.ClientOptions)        {}
func (r *recorder) Flush(time.Duration) bool              { return true }
func (r *recorder) FlushWithContext(context.Context) bool { return true }
func (r *recorder) Close()                                {}
func (r *recorder) SendEvent(e *sentry.Event)             { r.events = append(r.events, e) }

// Whatever ends up on the scope, by a future integration or a careless call,
// only the allowlisted fields leave for Sentry.
func TestEventsKeepOnlyAllowlistedFields(t *testing.T) {
	rec := &recorder{}
	client, err := errreport.NewClient(errreport.Options{DSN: "https://public@o0.ingest.sentry.io/0", Transport: rec})
	if err != nil {
		t.Fatal(err)
	}
	hub := sentry.NewHub(client, sentry.NewScope())

	req := httptest.NewRequest("POST", "/api/v1/pages/2026-11-26?q=secret-search", strings.NewReader(`{"text":"secret-page"}`))
	req.Header.Set("Cookie", "zoop_session=secret-cookie")
	hub.WithScope(func(s *sentry.Scope) {
		s.SetRequest(req)
		s.SetRequestBody([]byte(`{"text":"secret-page"}`))
		s.SetUser(sentry.User{ID: "user-1", Email: "secret@example.com", IPAddress: "203.0.113.9", Username: "secret-name"})
		s.SetTag("route", "/api/v1/pages/{day}")
		s.SetTag("title", "secret-title")
		s.SetContext("page", sentry.Context{"text": "secret-page"})
		hub.AddBreadcrumb(&sentry.Breadcrumb{Category: "console", Message: "secret-breadcrumb"}, nil)
		hub.CaptureException(errors.New("boom"))
	})

	if len(rec.events) != 1 {
		t.Fatalf("got %d events, want 1", len(rec.events))
	}
	ev := rec.events[0]
	raw, _ := json.Marshal(ev)
	if strings.Contains(string(raw), "secret") || strings.Contains(string(raw), "203.0.113.9") {
		t.Errorf("event leaks personal data: %s", raw)
	}
	if ev.Request != nil {
		t.Errorf("event.Request = %+v, want nil", ev.Request)
	}
	if u := ev.User; u.ID != "user-1" || u.Email != "" || u.IPAddress != "" || u.Username != "" || u.Name != "" || len(u.Data) != 0 {
		t.Errorf("event.User = %+v, want only the id", ev.User)
	}
	if ev.Tags["route"] != "/api/v1/pages/{day}" {
		t.Errorf("route tag = %q, want it kept", ev.Tags["route"])
	}
	if len(ev.Exception) == 0 || ev.Exception[0].Value != "boom" {
		t.Errorf("exception = %+v, want boom kept", ev.Exception)
	}
}
