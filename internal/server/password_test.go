package server_test

import (
	"net/http"
	"testing"
	"time"
)

const newPassword = "a whole new passphrase"

type resetBody struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (b *browser) forgot(email string) response {
	b.app.t.Helper()
	return b.do(http.MethodPost, "/api/v1/auth/password/forgot", emailBody{email})
}

func (b *browser) reset(token, password string) response {
	b.app.t.Helper()
	return b.do(http.MethodPost, "/api/v1/auth/password/reset", resetBody{token, password})
}

func TestAForgottenPasswordCanBeResetByEmail(t *testing.T) {
	app := startApp(t, tokyoMorning)
	email := lower(uniqueEmail("forgot"))
	app.browser().signup(email, "UTC")

	b := app.browser()
	if res := b.forgot(email); res.status != http.StatusAccepted {
		t.Fatalf("forgot = %d %s, want 202", res.status, res.body)
	}
	link := app.emailLink(email)
	if link.Path != "/reset" || link.Query().Get("token") == "" {
		t.Fatalf("reset link = %s, want %s/reset?token=…", link, app.srv.URL)
	}
	if res := b.reset(link.Query().Get("token"), newPassword); res.status != http.StatusNoContent {
		t.Fatalf("reset = %d %s, want 204", res.status, res.body)
	}

	old := app.browser().do(http.MethodPost, "/api/v1/auth/login", creds{Email: email, Password: goodPassword})
	if old.status != http.StatusUnauthorized {
		t.Errorf("login with the old password = %d, want 401", old.status)
	}
	res := app.browser().do(http.MethodPost, "/api/v1/auth/login", creds{Email: email, Password: newPassword})
	if res.status != http.StatusOK {
		t.Errorf("login with the new password = %d %s, want 200", res.status, res.body)
	}
}

func TestAResetSignsOutEveryDevice(t *testing.T) {
	app := startApp(t, tokyoMorning)
	laptop := app.browser()
	email := lower(uniqueEmail("devices"))
	laptop.signup(email, "UTC")
	phone := app.browser()
	phone.login(email)
	stolen := laptop.stubborn() // a device that won't let go of its cookie

	app.browser().forgot(email)
	if res := app.browser().reset(app.emailLink(email).Query().Get("token"), newPassword); res.status != http.StatusNoContent {
		t.Fatalf("reset = %d %s, want 204", res.status, res.body)
	}
	for name, b := range map[string]*browser{"laptop": laptop, "phone": phone, "stolen copy": stolen} {
		if got := b.meStatus(); got != http.StatusUnauthorized {
			t.Errorf("GET /me on the %s after a reset = %d, want 401", name, got)
		}
	}
}

func TestForgotDoesntRevealWhichEmailsHaveAccounts(t *testing.T) {
	app := startApp(t, tokyoMorning)
	known := lower(uniqueEmail("known"))
	app.browser().signup(known, "UTC")
	before := len(app.mail.Sent())

	forKnown := app.browser().forgot(known)
	forUnknown := app.browser().forgot(lower(uniqueEmail("nobody")))
	if forKnown.status != forUnknown.status || string(forKnown.body) != string(forUnknown.body) {
		t.Errorf("forgot answered %d %q for an account but %d %q for none, want the same",
			forKnown.status, forKnown.body, forUnknown.status, forUnknown.body)
	}
	if sent := len(app.mail.Sent()) - before; sent != 1 {
		t.Errorf("%d emails sent, want 1 (only to the real account)", sent)
	}
}

func TestAResetLinkWorksOnceWithinAnHour(t *testing.T) {
	app := startApp(t, tokyoMorning)
	email := lower(uniqueEmail("reset-once"))
	app.browser().signup(email, "UTC")

	app.browser().forgot(email)
	used := app.emailLink(email).Query().Get("token")
	if res := app.browser().reset(used, newPassword); res.status != http.StatusNoContent {
		t.Fatalf("reset = %d %s, want 204", res.status, res.body)
	}
	if res := app.browser().reset(used, "yet another passphrase"); res.status != http.StatusBadRequest || res.errorCode(t) != "bad_token" {
		t.Errorf("second reset = %d %s, want 400 bad_token", res.status, res.body)
	}

	app.browser().forgot(email)
	late := app.emailLink(email).Query().Get("token")
	app.clock.Advance(time.Hour)
	if res := app.browser().reset(late, "yet another passphrase"); res.status != http.StatusBadRequest || res.errorCode(t) != "bad_token" {
		t.Errorf("reset after an hour = %d %s, want 400 bad_token", res.status, res.body)
	}
	res := app.browser().do(http.MethodPost, "/api/v1/auth/login", creds{Email: email, Password: newPassword})
	if res.status != http.StatusOK {
		t.Errorf("login with the first reset's password = %d %s, want 200", res.status, res.body)
	}
}

func TestAResetNeedsAGoodPassword(t *testing.T) {
	app := startApp(t, tokyoMorning)
	email := lower(uniqueEmail("short"))
	app.browser().signup(email, "UTC")
	app.browser().forgot(email)
	token := app.emailLink(email).Query().Get("token")

	if res := app.browser().reset(token, "short"); res.status != http.StatusBadRequest || res.errorCode(t) != "validation" {
		t.Fatalf("reset with a short password = %d %s, want 400 validation", res.status, res.body)
	}
	if res := app.browser().reset(token, newPassword); res.status != http.StatusNoContent {
		t.Errorf("reset after fixing the password = %d %s, want 204 (the token wasn't used up)", res.status, res.body)
	}
}

func TestAGoogleOnlyUserSetsAPasswordThroughAReset(t *testing.T) {
	app := startApp(t, tokyoMorning)
	who := googleUser("google-only")
	email := lower(who.Email)
	app.browser().googleSignIn(who, "UTC")

	app.browser().forgot(email)
	if res := app.browser().reset(app.emailLink(email).Query().Get("token"), newPassword); res.status != http.StatusNoContent {
		t.Fatalf("reset = %d %s, want 204", res.status, res.body)
	}

	withPassword := app.browser()
	if res := withPassword.do(http.MethodPost, "/api/v1/auth/login", creds{Email: email, Password: newPassword}); res.status != http.StatusOK {
		t.Fatalf("login with the new password = %d %s, want 200", res.status, res.body)
	}
	if got, _ := withPassword.me(); len(got.SignInMethods) != 2 || got.SignInMethods[0] != "google" || got.SignInMethods[1] != "password" {
		t.Errorf("signInMethods = %v, want [google password]", got.SignInMethods)
	}
	if landed, _ := app.browser().googleSignIn(who, "UTC"); landed != "/" {
		t.Errorf("Google sign-in after the reset lands on %q, want / (still works)", landed)
	}
}
