package server_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/boinkkitty/zoop/internal/app"
	"github.com/boinkkitty/zoop/internal/clock"
	"github.com/boinkkitty/zoop/internal/mail/mailtest"
	"github.com/boinkkitty/zoop/internal/testenv"
)

// logBuffer collects the app's log output. The server writes from many goroutines.
type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *logBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

// lines parses every log line as JSON, failing the test on any that isn't.
func (l *logBuffer) lines(t *testing.T) []map[string]any {
	t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []map[string]any
	sc := bufio.NewScanner(bytes.NewReader(l.buf.Bytes()))
	for sc.Scan() {
		var line map[string]any
		if err := json.Unmarshal(sc.Bytes(), &line); err != nil {
			t.Fatalf("log line %q is not JSON: %v", sc.Text(), err)
		}
		out = append(out, line)
	}
	return out
}

func startAppWithLogs(t *testing.T, deps app.Deps) (*testApp, *logBuffer) {
	t.Helper()
	logs := &logBuffer{}
	c := clock.NewFake(tokyoMorning)
	if deps.DB == nil {
		deps.DB = testenv.Pool(t)
	}
	deps.Clock, deps.Mailer, deps.Assets, deps.Logs = c, &mailtest.Fake{}, assets, logs
	srv := httptest.NewServer(app.New(deps))
	t.Cleanup(srv.Close)
	return &testApp{t: t, srv: srv, clock: c}, logs
}

// The only keys a log line may carry. Anything else could be someone's writing.
var allowedLogKeys = map[string]bool{
	"time": true, "level": true, "msg": true,
	"method": true, "route": true, "status": true, "latency_ms": true,
	"user_id": true, "request_id": true, "err": true,
}

func TestLogsCarryOnlyAllowlistedFieldsAndNoPersonalData(t *testing.T) {
	app, logs := startAppWithLogs(t, app.Deps{})
	b := app.browser()
	email := uniqueEmail("grace")

	if res := b.do(http.MethodPost, "/api/v1/auth/signup", creds{email, goodPassword, "Asia/Tokyo"}); res.status != http.StatusCreated {
		t.Fatalf("signup = %d %s", res.status, res.body)
	}
	session := b.cookies["zoop_session"].Value
	b.me()
	b.do(http.MethodPost, "/api/v1/auth/login", creds{Email: email, Password: "wrong password, so wrong"})
	b.do(http.MethodGet, "/api/v1/search/dear-diary?q=dear+diary", nil)
	b.do(http.MethodGet, "/day/2026-11-26/my-journal?q=dear+diary", nil)
	b.do(http.MethodPost, "/api/v1/auth/logout", nil)

	lines := logs.lines(t)
	var requests []map[string]any
	for _, line := range lines {
		for key := range line {
			if !allowedLogKeys[key] {
				t.Errorf("log line has key %q, which isn't on the allowlist: %v", key, line)
			}
		}
		raw, _ := json.Marshal(line)
		for _, secret := range []string{lower(email), goodPassword, "so wrong", session, "dear", "diary"} {
			if strings.Contains(strings.ToLower(string(raw)), strings.ToLower(secret)) {
				t.Errorf("log line contains %q: %s", secret, raw)
			}
		}
		if line["msg"] == "request" {
			requests = append(requests, line)
		}
	}
	if len(requests) != 6 {
		t.Fatalf("got %d request lines, want one per request (6): %v", len(requests), lines)
	}

	meLine := requests[1]
	if meLine["method"] != "GET" || meLine["route"] != "/api/v1/me" || meLine["status"] != float64(200) {
		t.Errorf("GET /me logged as %v, want method GET, route /api/v1/me, status 200", meLine)
	}
	if id, _ := meLine["user_id"].(string); len(id) != 36 {
		t.Errorf("GET /me user_id = %v, want the signed-in user's uuid", meLine["user_id"])
	}
	if _, ok := meLine["latency_ms"].(float64); !ok {
		t.Errorf("GET /me latency_ms = %v, want a number", meLine["latency_ms"])
	}
	ids := map[any]bool{}
	for _, line := range requests {
		if id, _ := line["request_id"].(string); id == "" {
			t.Errorf("request line without a request_id: %v", line)
		}
		ids[line["request_id"]] = true
	}
	if len(ids) != len(requests) {
		t.Errorf("request ids aren't unique: %v", requests)
	}
}
