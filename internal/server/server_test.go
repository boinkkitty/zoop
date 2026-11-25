package server_test

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/boinkkitty/zoop/internal/clock"
	"github.com/boinkkitty/zoop/internal/mail/mailtest"
	"github.com/boinkkitty/zoop/internal/server"
	"github.com/boinkkitty/zoop/internal/testenv"
)

func TestMain(m *testing.M) { testenv.Run(m) }

const shell = `<!doctype html><title>Zoop</title><div id="root"></div>`

// newApp builds the real router against this package's migrated test database.
func newApp(t *testing.T) http.Handler {
	t.Helper()
	return server.New(server.Deps{
		DB:     testenv.Pool(t),
		Clock:  clock.NewFake(time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)),
		Mailer: &mailtest.Fake{},
		Assets: assets,
	})
}

var assets = fstest.MapFS{
	"index.html":        {Data: []byte(shell)},
	"assets/app-abc.js": {Data: []byte("console.log('zoop')")},
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestHealthzIsOKWhenPostgresIsUp(t *testing.T) {
	rec := get(t, newApp(t), "/api/healthz")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/healthz = %d, want 200", rec.Code)
	}
}

// A database that accepts connections but never answers: the worst kind of "down".
func TestHealthzFailsFastWhenPostgresHangs(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			t.Cleanup(func() { conn.Close() })
		}
	}()

	pool, err := pgxpool.New(context.Background(), "postgres://zoop:zoop@"+ln.Addr().String()+"/zoop?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	app := server.New(server.Deps{
		DB:     pool,
		Clock:  clock.NewFake(time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)),
		Mailer: &mailtest.Fake{},
		Assets: fstest.MapFS{"index.html": {Data: []byte(shell)}},
	})

	start := time.Now()
	rec := get(t, app, "/api/healthz")
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("healthz took %v with a hung database, want about 1s", elapsed)
	}
	if rec.Code < 500 {
		t.Errorf("GET /api/healthz = %d with a hung database, want 5xx", rec.Code)
	}
}

func TestAppShellIsServedForRootAndUnknownNonAPIPaths(t *testing.T) {
	app := newApp(t)
	for _, path := range []string{"/", "/day/2026-11-24", "/settings/whatever"} {
		rec := get(t, app, path)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, rec.Code)
		}
		if rec.Body.String() != shell {
			t.Errorf("GET %s body = %q, want the app shell", path, rec.Body.String())
		}
		if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Errorf("GET %s Content-Type = %q, want text/html", path, ct)
		}
	}
}

func TestUnknownAPIPathsAreNotFoundNotTheShell(t *testing.T) {
	app := newApp(t)
	for _, path := range []string{"/api", "/api/v1/nope", "/api/healthz/extra"} {
		rec := get(t, app, path)
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", path, rec.Code)
		}
		var body struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Errorf("GET %s body %q is not JSON: %v", path, rec.Body.String(), err)
			continue
		}
		if body.Error.Code != "not_found" {
			t.Errorf("GET %s error code = %q, want not_found", path, body.Error.Code)
		}
	}
}

func TestBuiltFilesAreServedAsThemselves(t *testing.T) {
	rec := get(t, newApp(t), "/assets/app-abc.js")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /assets/app-abc.js = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "console.log('zoop')" {
		t.Errorf("body = %q, want the JS file", rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/javascript; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/javascript", ct)
	}
}
