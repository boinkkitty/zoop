package app_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/boinkkitty/zoop/internal/app"
	"github.com/boinkkitty/zoop/internal/clock"
	"github.com/boinkkitty/zoop/internal/mail/mailtest"
	"github.com/boinkkitty/zoop/internal/testenv"
)

// This package has its own database, so Sweep's counts are only this file's rows.
func TestMain(m *testing.M) { testenv.Run(m) }

const day = 24 * time.Hour

var start = time.Date(2026, 11, 25, 20, 0, 0, 0, time.UTC)

func TestSweepDeletesOnlyExpiredSessionsRateLimitsAndEmailTokens(t *testing.T) {
	c := clock.NewFake(start)
	deps := app.Deps{
		DB:     testenv.Pool(t),
		Clock:  c,
		Mailer: &mailtest.Fake{},
		Assets: fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}},
	}
	srv := httptest.NewServer(app.New(deps))
	t.Cleanup(srv.Close)

	// A session lasting 30 days, a signup-per-IP count kept for the hour, and a verify token lasting a day.
	laptop := post(t, srv, "/api/v1/auth/signup", `{"email":"ada@example.com","password":"correct horse","timezone":"UTC"}`, "")
	// Per-email and per-IP failure counts, kept for 15 minutes.
	post(t, srv, "/api/v1/auth/login", `{"email":"ada@example.com","password":"wrong"}`, "")

	sweep := func(want app.Swept) {
		t.Helper()
		got, err := app.Sweep(context.Background(), deps)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("Sweep at %s = %+v, want %+v", c.Now(t.Context()).Sub(start), got, want)
		}
	}

	sweep(app.Swept{})
	c.Advance(15 * time.Minute)
	sweep(app.Swept{RateLimits: 2})
	c.Advance(45 * time.Minute)
	sweep(app.Swept{RateLimits: 1})

	phone := post(t, srv, "/api/v1/auth/login", `{"email":"ada@example.com","password":"correct horse"}`, "")
	c.Set(start.Add(day))
	sweep(app.Swept{EmailTokens: 1})
	c.Set(start.Add(30 * day))
	sweep(app.Swept{Sessions: 1}) // the laptop's; the phone's has an hour left
	sweep(app.Swept{})

	if got := get(t, srv, "/api/v1/me", phone); got != http.StatusOK {
		t.Errorf("GET /me on the phone after the sweep = %d, want 200", got)
	}
	if got := get(t, srv, "/api/v1/me", laptop); got != http.StatusUnauthorized {
		t.Errorf("GET /me on the laptop after the sweep = %d, want 401", got)
	}
}

// post sends body from one IP and returns the session cookie it set, if any.
func post(t *testing.T, srv *httptest.Server, path, body, cookie string) *http.Cookie {
	t.Helper()
	res := send(t, srv, http.MethodPost, path, bytes.NewBufferString(body), cookie)
	for _, c := range res.Cookies() {
		return c
	}
	return nil
}

func get(t *testing.T, srv *httptest.Server, path string, session *http.Cookie) int {
	t.Helper()
	return send(t, srv, http.MethodGet, path, nil, session.Value).StatusCode
}

func send(t *testing.T, srv *httptest.Server, method, path string, body *bytes.Buffer, cookie string) *http.Response {
	t.Helper()
	var r io.Reader = http.NoBody
	if body != nil {
		r = body
	}
	req, err := http.NewRequest(method, srv.URL+path, r)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "192.0.2.1")
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: "zoop_session", Value: cookie})
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, res.Body)
	res.Body.Close()
	return res
}
