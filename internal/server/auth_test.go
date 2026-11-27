package server_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/boinkkitty/zoop/internal/app"
	"github.com/boinkkitty/zoop/internal/clock"
	"github.com/boinkkitty/zoop/internal/google"
	"github.com/boinkkitty/zoop/internal/google/googletest"
	"github.com/boinkkitty/zoop/internal/mail/mailtest"
	"github.com/boinkkitty/zoop/internal/testenv"
)

// testApp is the real router behind a real HTTP server, with a Clock the test can move.
// Its Google is a stub OIDC provider.
type testApp struct {
	t      *testing.T
	srv    *httptest.Server
	clock  *clock.Fake
	google *googletest.Provider
}

func startApp(t *testing.T, now time.Time) *testApp {
	t.Helper()
	c := clock.NewFake(now)
	stub := googletest.Start(t, func() time.Time { return c.Now(t.Context()) })
	// The app needs its own URL for Google's redirect, so it's built once the server is listening.
	var h http.Handler
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { h.ServeHTTP(w, r) }))
	t.Cleanup(srv.Close)
	h = app.New(app.Deps{
		DB:     testenv.Pool(t),
		Clock:  c,
		Mailer: &mailtest.Fake{},
		Assets: assets,
		Google: &google.Config{
			Issuer:       stub.Issuer(),
			ClientID:     googletest.ClientID,
			ClientSecret: googletest.ClientSecret,
			RedirectURL:  srv.URL + "/api/v1/auth/google/callback",
		},
	})
	return &testApp{t: t, srv: srv, clock: c, google: stub}
}

// browser is one device with its own cookies and its own IP address. Unlike
// net/http/cookiejar it expires cookies by the app's Clock, so moving the Clock ages them too.
type browser struct {
	app     *testApp
	cookies map[string]*http.Cookie
	ip      string // what Railway's proxy reports in X-Forwarded-For
}

func (a *testApp) browser() *browser {
	return &browser{app: a, cookies: map[string]*http.Cookie{}, ip: randomIP()}
}

// randomIP keeps per-IP rate limits apart between tests in the shared database.
func randomIP() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return fmt.Sprintf("10.%d.%d.%d", b[0], b[1], b[2])
}

func (b *browser) keep(c *http.Cookie) {
	expired := c.MaxAge < 0 || (!c.Expires.IsZero() && !c.Expires.After(b.app.clock.Now(context.Background())))
	if expired {
		delete(b.cookies, c.Name)
		return
	}
	b.cookies[c.Name] = c
}

type response struct {
	status int
	header http.Header
	body   []byte
}

func (r response) json(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.body, v); err != nil {
		t.Fatalf("response body %q is not the expected JSON: %v", r.body, err)
	}
}

func (r response) errorCode(t *testing.T) string {
	t.Helper()
	var body struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	r.json(t, &body)
	return body.Error.Code
}

func (b *browser) do(method, path string, body any, headers ...string) response {
	b.app.t.Helper()
	var r io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			b.app.t.Fatal(err)
		}
		r = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(method, b.app.srv.URL+path, r)
	if err != nil {
		b.app.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Forwarded-For", b.ip)
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	for name, c := range b.cookies {
		if !c.Expires.IsZero() && !c.Expires.After(b.app.clock.Now(context.Background())) {
			delete(b.cookies, name)
			continue
		}
		req.AddCookie(&http.Cookie{Name: c.Name, Value: c.Value})
	}
	res, err := noRedirects.Do(req)
	if err != nil {
		b.app.t.Fatal(err)
	}
	defer res.Body.Close()
	for _, c := range res.Cookies() {
		b.keep(c)
	}
	out, err := io.ReadAll(res.Body)
	if err != nil {
		b.app.t.Fatal(err)
	}
	return response{status: res.StatusCode, header: res.Header, body: out}
}

// noRedirects hands each redirect back to the test, so the browser can carry cookies across them.
var noRedirects = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}}

type me struct {
	Email           string   `json:"email"`
	EmailVerified   bool     `json:"emailVerified"`
	Plan            string   `json:"plan"`
	Timezone        string   `json:"timezone"`
	PendingTimezone *string  `json:"pendingTimezone"`
	FirstDay        string   `json:"firstDay"`
	Today           string   `json:"today"`
	GraceUntil      *string  `json:"graceUntil"`
	SignInMethods   []string `json:"signInMethods"`
}

func (b *browser) me() (me, response) {
	b.app.t.Helper()
	res := b.do(http.MethodGet, "/api/v1/me", nil)
	var m me
	if res.status == http.StatusOK {
		res.json(b.app.t, &m)
	}
	return m, res
}

// uniqueEmail keeps tests apart in the package's shared database.
func uniqueEmail(prefix string) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return prefix + "+" + hex.EncodeToString(b) + "@example.com"
}

type creds struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Timezone string `json:"timezone,omitempty"`
}

const goodPassword = "correct horse battery staple"

// 20:00 UTC on 25 Nov is already 05:00 on 26 Nov in Tokyo.
var tokyoMorning = time.Date(2026, 11, 25, 20, 0, 0, 0, time.UTC)

func TestSignupSignsTheVisitorInAndMeDescribesThem(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	email := uniqueEmail("Ada.Lovelace")

	res := b.do(http.MethodPost, "/api/v1/auth/signup", creds{email, goodPassword, "Asia/Tokyo"})
	if res.status != http.StatusCreated {
		t.Fatalf("signup = %d %s, want 201", res.status, res.body)
	}

	got, res := b.me()
	if res.status != http.StatusOK {
		t.Fatalf("GET /me after signup = %d %s, want 200", res.status, res.body)
	}
	if want := lower(email); got.Email != want {
		t.Errorf("email = %q, want %q (lowercased)", got.Email, want)
	}
	if got.EmailVerified {
		t.Error("emailVerified = true, want false right after signup")
	}
	if got.Plan != "free" {
		t.Errorf("plan = %q, want free", got.Plan)
	}
	if got.Timezone != "Asia/Tokyo" || got.PendingTimezone != nil {
		t.Errorf("timezone = %q pending %v, want Asia/Tokyo with none pending", got.Timezone, got.PendingTimezone)
	}
	if got.FirstDay != "2026-11-26" || got.Today != "2026-11-26" {
		t.Errorf("firstDay %q today %q, want both 2026-11-26 (Tokyo's date)", got.FirstDay, got.Today)
	}
	if got.GraceUntil != nil {
		t.Errorf("graceUntil = %q at 05:00, want null", *got.GraceUntil)
	}
	if len(got.SignInMethods) != 1 || got.SignInMethods[0] != "password" {
		t.Errorf("signInMethods = %v, want [password]", got.SignInMethods)
	}
}

func lower(s string) string { return string(bytes.ToLower([]byte(s))) }

func TestMeShowsWhenTheGraceWindowEnds(t *testing.T) {
	// 16:30 UTC is 01:30 on 26 Nov in Tokyo: yesterday is still open until 02:00 local.
	app := startApp(t, time.Date(2026, 11, 25, 16, 30, 0, 0, time.UTC))
	b := app.browser()
	b.signup(uniqueEmail("grace"), "Asia/Tokyo")

	got, _ := b.me()
	if got.GraceUntil == nil {
		t.Fatal("graceUntil = null at 01:30, want 02:00 Tokyo")
	}
	if *got.GraceUntil != "2026-11-25T17:00:00Z" {
		t.Errorf("graceUntil = %q, want 2026-11-25T17:00:00Z (02:00 Tokyo)", *got.GraceUntil)
	}
}

func TestTheSessionCookieIsHttpOnlyAndSameSiteLax(t *testing.T) {
	app := startApp(t, tokyoMorning)
	res := app.browser().do(http.MethodPost, "/api/v1/auth/signup", creds{uniqueEmail("cookie"), goodPassword, "UTC"})
	cookies := (&http.Response{Header: res.header}).Cookies()
	if len(cookies) != 1 {
		t.Fatalf("signup set %d cookies, want 1", len(cookies))
	}
	c := cookies[0]
	if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode {
		t.Errorf("cookie HttpOnly=%v SameSite=%v, want HttpOnly and Lax", c.HttpOnly, c.SameSite)
	}
	if want := tokyoMorning.Add(30 * 24 * time.Hour); !c.Expires.Equal(want) {
		t.Errorf("cookie expires %v, want %v (30 days)", c.Expires, want)
	}
}

func TestSignupWithAnEmailThatHasAnAccountIsRefused(t *testing.T) {
	app := startApp(t, tokyoMorning)
	email := uniqueEmail("taken")
	app.browser().signup(email, "UTC")

	b := app.browser()
	res := b.do(http.MethodPost, "/api/v1/auth/signup", creds{"  " + upper(email) + " ", goodPassword, "UTC"})
	if res.status != http.StatusConflict || res.errorCode(t) != "email_taken" {
		t.Errorf("second signup = %d %s, want 409 email_taken", res.status, res.body)
	}
	if _, res := b.me(); res.status != http.StatusUnauthorized {
		t.Errorf("GET /me after refused signup = %d, want 401", res.status)
	}
}

func TestMeWithoutASessionIsUnauthenticated(t *testing.T) {
	_, res := startApp(t, tokyoMorning).browser().me()
	if res.status != http.StatusUnauthorized || res.errorCode(t) != "unauthenticated" {
		t.Errorf("GET /me signed out = %d %s, want 401 unauthenticated", res.status, res.body)
	}
}

func (b *browser) signup(email, timezone string) {
	b.app.t.Helper()
	res := b.do(http.MethodPost, "/api/v1/auth/signup", creds{email, goodPassword, timezone})
	if res.status != http.StatusCreated {
		b.app.t.Fatalf("signup %s = %d %s, want 201", email, res.status, res.body)
	}
}

func upper(s string) string { return string(bytes.ToUpper([]byte(s))) }

func TestSignupRefusesInvalidInputWithoutCreatingAnAccount(t *testing.T) {
	app := startApp(t, tokyoMorning)
	email := uniqueEmail("valid")
	cases := map[string]creds{
		"no email":          {"", goodPassword, "UTC"},
		"not an email":      {"ada.example.com", goodPassword, "UTC"},
		"empty password":    {email, "", "UTC"},
		"short password":    {email, "1234567", "UTC"},
		"too long password": {email, strings.Repeat("a", 257), "UTC"},
		"no timezone":       {email, goodPassword, ""},
		"unknown timezone":  {email, goodPassword, "Mars/Olympus_Mons"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			res := app.browser().do(http.MethodPost, "/api/v1/auth/signup", c)
			if res.status != http.StatusBadRequest || res.errorCode(t) != "validation" {
				t.Errorf("signup = %d %s, want 400 validation", res.status, res.body)
			}
		})
	}
	// None of those created the account, so the email is still free.
	app.browser().signup(email, "UTC")
}

func TestLoginSignsANewDeviceIn(t *testing.T) {
	app := startApp(t, tokyoMorning)
	email := uniqueEmail("login")
	app.browser().signup(email, "Asia/Tokyo")

	other := app.browser()
	res := other.do(http.MethodPost, "/api/v1/auth/login", creds{Email: upper(email), Password: goodPassword})
	if res.status != http.StatusOK {
		t.Fatalf("login = %d %s, want 200", res.status, res.body)
	}
	got, res := other.me()
	if res.status != http.StatusOK || got.Email != lower(email) {
		t.Errorf("GET /me after login = %d email %q, want 200 %q", res.status, got.Email, lower(email))
	}
}

func TestLoginWithAWrongPasswordOrUnknownEmailLooksTheSame(t *testing.T) {
	app := startApp(t, tokyoMorning)
	email := uniqueEmail("wrong")
	app.browser().signup(email, "UTC")

	for name, c := range map[string]creds{
		"wrong password": {Email: email, Password: goodPassword + "!"},
		"unknown email":  {Email: uniqueEmail("nobody"), Password: goodPassword},
		"empty":          {},
		"too long":       {Email: email, Password: strings.Repeat("a", 10_000)},
	} {
		t.Run(name, func(t *testing.T) {
			b := app.browser()
			res := b.do(http.MethodPost, "/api/v1/auth/login", c)
			if res.status != http.StatusUnauthorized || res.errorCode(t) != "unauthenticated" {
				t.Errorf("login = %d %s, want 401 unauthenticated", res.status, res.body)
			}
			if _, res := b.me(); res.status != http.StatusUnauthorized {
				t.Errorf("GET /me after failed login = %d, want 401", res.status)
			}
		})
	}
}

func TestLogoutSignsOutOnlyThisDeviceAndKillsItsToken(t *testing.T) {
	app := startApp(t, tokyoMorning)
	email := uniqueEmail("logout")
	laptop := app.browser()
	laptop.signup(email, "UTC")
	phone := app.browser()
	phone.login(email)
	stolen := laptop.sessionCookie()

	res := laptop.do(http.MethodPost, "/api/v1/auth/logout", nil)
	if res.status != http.StatusNoContent {
		t.Fatalf("logout = %d %s, want 204", res.status, res.body)
	}
	if _, res := laptop.me(); res.status != http.StatusUnauthorized {
		t.Errorf("GET /me on the signed-out laptop = %d, want 401", res.status)
	}
	if _, res := phone.me(); res.status != http.StatusOK {
		t.Errorf("GET /me on the phone = %d, want 200 (still signed in)", res.status)
	}

	replay := app.browser()
	replay.setSessionCookie(stolen)
	if _, res := replay.me(); res.status != http.StatusUnauthorized {
		t.Errorf("GET /me with the logged-out token = %d, want 401", res.status)
	}
}

func (b *browser) login(email string) {
	b.app.t.Helper()
	res := b.do(http.MethodPost, "/api/v1/auth/login", creds{Email: email, Password: goodPassword})
	if res.status != http.StatusOK {
		b.app.t.Fatalf("login %s = %d %s, want 200", email, res.status, res.body)
	}
}

func (b *browser) sessionCookie() *http.Cookie {
	b.app.t.Helper()
	c, ok := b.cookies["zoop_session"]
	if !ok {
		b.app.t.Fatal("this browser has no zoop_session cookie")
	}
	return c
}

func (b *browser) setSessionCookie(c *http.Cookie) {
	b.cookies[c.Name] = &http.Cookie{Name: c.Name, Value: c.Value}
}

// stubborn copies b's session token into a device that never drops it,
// so only the server can decide the session is over.
func (b *browser) stubborn() *browser {
	b.app.t.Helper()
	other := b.app.browser()
	other.setSessionCookie(b.sessionCookie())
	return other
}

func (b *browser) meStatus() int {
	b.app.t.Helper()
	_, res := b.me()
	return res.status
}

const day = 24 * time.Hour

func TestASessionLasts30DaysFromItsLastUse(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	b.signup(uniqueEmail("sliding"), "UTC")
	device := b.stubborn()

	app.clock.Advance(29 * day)
	if got := device.meStatus(); got != http.StatusOK {
		t.Fatalf("GET /me on day 29 = %d, want 200", got)
	}
	app.clock.Advance(29 * day) // day 58: past the original expiry, within 30 days of last use
	if got := device.meStatus(); got != http.StatusOK {
		t.Fatalf("GET /me on day 58 = %d, want 200 (renewed on day 29)", got)
	}
	app.clock.Advance(30 * day) // 30 days of silence
	if got := device.meStatus(); got != http.StatusUnauthorized {
		t.Errorf("GET /me 30 days after last use = %d, want 401", got)
	}
}

func TestAnUnusedSessionEndsAfter30Days(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	b.signup(uniqueEmail("unused"), "UTC")
	device := b.stubborn()

	app.clock.Advance(30 * day)
	if got := device.meStatus(); got != http.StatusUnauthorized {
		t.Errorf("GET /me exactly 30 days after signup = %d, want 401", got)
	}
}

func TestRenewalHappensAtMostHourlyAndResendsTheCookie(t *testing.T) {
	app := startApp(t, tokyoMorning)

	recent := app.browser()
	recent.signup(uniqueEmail("hourly"), "UTC")
	app.clock.Advance(59 * time.Minute)
	if res := recent.do(http.MethodGet, "/api/v1/me", nil); len((&http.Response{Header: res.header}).Cookies()) != 0 {
		t.Errorf("GET /me 59 minutes after signup re-sent the cookie, want no renewal yet")
	}
	device := recent.stubborn()
	app.clock.Set(tokyoMorning.Add(30 * day))
	if got := device.meStatus(); got != http.StatusUnauthorized {
		t.Errorf("GET /me on day 30 after only a 59-minute use = %d, want 401 (not renewed)", got)
	}

	app.clock.Set(tokyoMorning)
	hourly := app.browser()
	hourly.signup(uniqueEmail("hourly"), "UTC")
	app.clock.Advance(time.Hour)
	res := hourly.do(http.MethodGet, "/api/v1/me", nil)
	cookies := (&http.Response{Header: res.header}).Cookies()
	if want := tokyoMorning.Add(time.Hour + 30*day); len(cookies) != 1 || !cookies[0].Expires.Equal(want) {
		t.Errorf("GET /me an hour after signup set cookies %v, want one expiring %v", cookies, want)
	}
	device = hourly.stubborn()
	app.clock.Set(tokyoMorning.Add(30 * day))
	if got := device.meStatus(); got != http.StatusOK {
		t.Errorf("GET /me on day 30 after a use at hour 1 = %d, want 200 (renewed)", got)
	}
}

func TestCrossOriginStateChangesAreRefused(t *testing.T) {
	app := startApp(t, tokyoMorning)
	email := uniqueEmail("csrf")
	victim := app.browser()
	victim.signup(email, "UTC")

	attacks := map[string][]string{
		"browser marks it cross-site": {"Sec-Fetch-Site", "cross-site"},
		"foreign Origin":              {"Origin", "https://evil.example"},
	}
	for name, headers := range attacks {
		t.Run(name, func(t *testing.T) {
			for _, path := range []string{"/api/v1/auth/logout", "/api/v1/auth/login", "/api/v1/auth/signup"} {
				res := victim.do(http.MethodPost, path, creds{email, goodPassword, "UTC"}, headers...)
				if res.status != http.StatusForbidden || res.errorCode(t) != "cross_origin" {
					t.Errorf("POST %s = %d %s, want 403 cross_origin", path, res.status, res.body)
				}
			}
			if got := victim.meStatus(); got != http.StatusOK {
				t.Errorf("GET /me after the refused logout = %d, want 200 (still signed in)", got)
			}
		})
	}

	res := victim.do(http.MethodPost, "/api/v1/auth/logout", nil, "Sec-Fetch-Site", "same-origin")
	if res.status != http.StatusNoContent {
		t.Errorf("same-origin logout = %d %s, want 204", res.status, res.body)
	}
}
