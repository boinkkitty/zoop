package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/boinkkitty/zoop/internal/app"
	"github.com/boinkkitty/zoop/internal/errreport"
	"github.com/boinkkitty/zoop/internal/testenv"
)

// sentryRecorder stands in for Sentry's servers: it keeps every event the client would send.
type sentryRecorder struct {
	mu     sync.Mutex
	events []*sentry.Event
}

func (s *sentryRecorder) Configure(sentry.ClientOptions)        {}
func (s *sentryRecorder) Flush(time.Duration) bool              { return true }
func (s *sentryRecorder) FlushWithContext(context.Context) bool { return true }
func (s *sentryRecorder) Close()                                {}
func (s *sentryRecorder) SendEvent(e *sentry.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)
}

func (s *sentryRecorder) sent() []*sentry.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*sentry.Event(nil), s.events...)
}

func TestServerErrorsReachSentryWithoutPersonalData(t *testing.T) {
	// A closed pool makes every database call fail, so signup ends in a 500.
	broken, err := pgxpool.New(context.Background(), testenv.URL(t))
	if err != nil {
		t.Fatal(err)
	}
	broken.Close()
	recorder := &sentryRecorder{}
	client, err := errreport.NewClient(errreport.Options{
		DSN:       "https://public@o0.ingest.sentry.io/0",
		Transport: recorder,
	})
	if err != nil {
		t.Fatal(err)
	}
	app, logs := startAppWithLogs(t, app.Deps{DB: broken, Sentry: client})
	b := app.browser()
	b.cookies["zoop_session"] = &http.Cookie{Name: "zoop_session", Value: "tok_s3cr3t-cookie"}
	email := uniqueEmail("hopper")

	res := b.do(http.MethodPost, "/api/v1/auth/signup", creds{email, goodPassword, "Asia/Tokyo"},
		"Authorization", "Bearer tok_s3cr3t-header")
	if res.status != http.StatusInternalServerError {
		t.Fatalf("signup on a broken database = %d %s, want 500", res.status, res.body)
	}

	events := recorder.sent()
	if len(events) != 1 {
		t.Fatalf("Sentry got %d events, want 1", len(events))
	}
	ev := events[0]
	raw, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{lower(email), goodPassword, "tok_s3cr3t"} {
		if strings.Contains(strings.ToLower(string(raw)), strings.ToLower(secret)) {
			t.Errorf("Sentry event contains %q: %s", secret, raw)
		}
	}
	if ev.Request != nil {
		t.Errorf("Sentry event has request data %+v, want none", ev.Request)
	}
	if len(ev.Breadcrumbs) != 0 {
		t.Errorf("Sentry event has breadcrumbs %v, want none", ev.Breadcrumbs)
	}
	if len(ev.Exception) == 0 {
		t.Error("Sentry event has no exception, want the database error")
	}
	if ev.Tags["route"] != "/api/v1/auth/signup" {
		t.Errorf("Sentry route tag = %q, want /api/v1/auth/signup", ev.Tags["route"])
	}

	// The event and the access log share a request id, so one leads to the other.
	var reqID any
	for _, line := range logs.lines(t) {
		if line["msg"] == "request" {
			reqID = line["request_id"]
		}
	}
	if reqID == nil || ev.Tags["request_id"] != reqID {
		t.Errorf("Sentry request_id tag = %q, want the access log's %v", ev.Tags["request_id"], reqID)
	}
}
