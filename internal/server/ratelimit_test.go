package server_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestLoginIsSlowedAfterFiveFailuresForAnEmail(t *testing.T) {
	app := startApp(t, tokyoMorning)
	email := uniqueEmail("guessed")
	app.browser().signup(email, "UTC")

	app.clock.Advance(3 * time.Minute)
	for i := range 5 {
		res := app.browser().do(http.MethodPost, "/api/v1/auth/login", creds{Email: email, Password: "wrong guess"})
		if res.status != http.StatusUnauthorized {
			t.Fatalf("failed login %d = %d %s, want 401", i+1, res.status, res.body)
		}
	}

	// Even the right password, from a fresh IP, waits out the window.
	res := app.browser().do(http.MethodPost, "/api/v1/auth/login", creds{Email: email, Password: goodPassword})
	if res.status != http.StatusTooManyRequests || res.errorCode(t) != "rate_limited" {
		t.Fatalf("login after 5 failures = %d %s, want 429 rate_limited", res.status, res.body)
	}
	if got := res.header.Get("Retry-After"); got != "720" {
		t.Errorf("Retry-After = %q, want 720 (12 minutes left in the 15-minute window)", got)
	}

	// No lockout: once the window ends, the owner gets straight in.
	app.clock.Advance(12 * time.Minute)
	app.browser().login(email)
}

func TestLoginIsSlowedAfterTwentyFailuresFromAnIP(t *testing.T) {
	app := startApp(t, tokyoMorning)
	email := uniqueEmail("sprayed")
	app.browser().signup(email, "UTC")

	attacker := app.browser()
	for i := range 20 {
		res := attacker.do(http.MethodPost, "/api/v1/auth/login", creds{Email: uniqueEmail("spray"), Password: "wrong guess"})
		if res.status != http.StatusUnauthorized {
			t.Fatalf("failed login %d = %d %s, want 401", i+1, res.status, res.body)
		}
	}

	res := attacker.do(http.MethodPost, "/api/v1/auth/login", creds{Email: email, Password: goodPassword})
	if res.status != http.StatusTooManyRequests || res.errorCode(t) != "rate_limited" {
		t.Fatalf("login after 20 failures from one IP = %d %s, want 429 rate_limited", res.status, res.body)
	}
	if got := res.header.Get("Retry-After"); got != "900" {
		t.Errorf("Retry-After = %q, want 900", got)
	}
	// Other IPs aren't slowed.
	app.browser().login(email)
}

func TestSignupIsSlowedAfterFivePerIPInAnHour(t *testing.T) {
	app := startApp(t, tokyoMorning)
	ip := randomIP()
	device := func() *browser {
		b := app.browser()
		b.ip = ip
		return b
	}
	app.clock.Advance(10 * time.Minute)
	for range 5 {
		device().signup(uniqueEmail("farm"), "UTC")
	}

	res := device().do(http.MethodPost, "/api/v1/auth/signup", creds{uniqueEmail("farm"), goodPassword, "UTC"})
	if res.status != http.StatusTooManyRequests || res.errorCode(t) != "rate_limited" {
		t.Fatalf("6th signup from one IP = %d %s, want 429 rate_limited", res.status, res.body)
	}
	if got := res.header.Get("Retry-After"); got != "3000" {
		t.Errorf("Retry-After = %q, want 3000 (50 minutes left in the hour)", got)
	}

	app.browser().signup(uniqueEmail("neighbour"), "UTC") // other IPs aren't slowed
	app.clock.Advance(50 * time.Minute)
	device().signup(uniqueEmail("farm"), "UTC") // nor this one, next hour
}

func TestTheAPIServesAUserAtMost300RequestsAMinute(t *testing.T) {
	app := startApp(t, tokyoMorning)
	email := uniqueEmail("runaway")
	laptop := app.browser()
	laptop.signup(email, "UTC")
	phone := app.browser()
	phone.login(email)
	bystander := app.browser()
	bystander.signup(uniqueEmail("bystander"), "UTC")

	app.clock.Advance(15 * time.Second)
	for i := range 300 {
		if got := laptop.meStatus(); got != http.StatusOK {
			t.Fatalf("GET /me #%d = %d, want 200", i+1, got)
		}
	}

	// The limit is per user, so another of their devices is slowed too.
	_, res := phone.me()
	if res.status != http.StatusTooManyRequests || res.errorCode(t) != "rate_limited" {
		t.Fatalf("GET /me #301 = %d %s, want 429 rate_limited", res.status, res.body)
	}
	if got := res.header.Get("Retry-After"); got != "45" {
		t.Errorf("Retry-After = %q, want 45", got)
	}
	if got := bystander.meStatus(); got != http.StatusOK {
		t.Errorf("another user's GET /me = %d, want 200", got)
	}
	app.clock.Advance(45 * time.Second)
	if got := laptop.meStatus(); got != http.StatusOK {
		t.Errorf("GET /me in the next minute = %d, want 200", got)
	}
}

// Railway's proxy appends the address it saw, so only the last X-Forwarded-For entry is trusted.
func TestASpoofedForwardedHeaderDoesntDodgeTheIPLimit(t *testing.T) {
	app := startApp(t, tokyoMorning)
	attacker := app.browser()
	for i := range 20 {
		spoof := fmt.Sprintf("203.0.113.%d, %s", i, attacker.ip)
		res := attacker.do(http.MethodPost, "/api/v1/auth/login", creds{Email: uniqueEmail("spoof"), Password: "wrong guess"},
			"X-Forwarded-For", spoof)
		if res.status != http.StatusUnauthorized {
			t.Fatalf("failed login %d = %d %s, want 401", i+1, res.status, res.body)
		}
	}
	res := attacker.do(http.MethodPost, "/api/v1/auth/login", creds{Email: uniqueEmail("spoof"), Password: "wrong guess"},
		"X-Forwarded-For", "198.51.100.7, "+attacker.ip)
	if res.status != http.StatusTooManyRequests {
		t.Errorf("21st failure behind a new spoofed address = %d %s, want 429", res.status, res.body)
	}
}
