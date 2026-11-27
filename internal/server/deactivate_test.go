package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/boinkkitty/zoop/internal/google/googletest"
)

func (b *browser) deactivate(email string) response {
	b.app.t.Helper()
	return b.do(http.MethodDelete, "/api/v1/me", emailBody{email})
}

func TestDeactivatingSignsOutEveryDeviceAndEmailsTheUser(t *testing.T) {
	app := startApp(t, tokyoMorning)
	email := lower(uniqueEmail("leaving"))
	laptop := app.browser()
	laptop.signup(email, "UTC")
	phone := app.browser()
	phone.login(email)
	stolen := laptop.stubborn()

	res := laptop.deactivate(upper(email))
	if res.status != http.StatusNoContent {
		t.Fatalf("DELETE /me = %d %s, want 204", res.status, res.body)
	}
	if _, kept := laptop.cookies["zoop_session"]; kept {
		t.Error("the laptop still has its session cookie, want it cleared")
	}
	for name, b := range map[string]*browser{"laptop": laptop, "phone": phone, "stolen copy": stolen} {
		if got := b.meStatus(); got != http.StatusUnauthorized {
			t.Errorf("GET /me on the %s after deactivating = %d, want 401", name, got)
		}
	}
	sent := app.mail.Sent()
	last := sent[len(sent)-1]
	if last.To != email || !strings.Contains(last.Subject, "deactivated") {
		t.Errorf("last email = to %q subject %q, want the deactivation email to %q", last.To, last.Subject, email)
	}
}

func TestDeactivatingNeedsTheAccountsEmailTypedToConfirm(t *testing.T) {
	app := startApp(t, tokyoMorning)
	email := lower(uniqueEmail("stay"))
	b := app.browser()
	b.signup(email, "UTC")
	sent := len(app.mail.Sent())

	for _, typed := range []string{"", "someone@example.com", email + "x"} {
		res := b.deactivate(typed)
		if res.status != http.StatusBadRequest || res.errorCode(t) != "validation" {
			t.Errorf("DELETE /me typing %q = %d %s, want 400 validation", typed, res.status, res.body)
		}
	}
	if got := b.meStatus(); got != http.StatusOK {
		t.Errorf("GET /me after refused deactivations = %d, want 200 (still signed in)", got)
	}
	if got := len(app.mail.Sent()); got != sent {
		t.Errorf("%d emails sent by refused deactivations, want none", got-sent)
	}
}

// deactivated signs up an account with a password and deactivates it, returning its email.
func (a *testApp) deactivated(prefix string) string {
	a.t.Helper()
	email := lower(uniqueEmail(prefix))
	b := a.browser()
	b.signup(email, "UTC")
	if res := b.deactivate(email); res.status != http.StatusNoContent {
		a.t.Fatalf("DELETE /me = %d %s, want 204", res.status, res.body)
	}
	return email
}

func TestSigningInToADeactivatedAccountSaysSoOnlyWithTheRightPassword(t *testing.T) {
	app := startApp(t, tokyoMorning)
	email := app.deactivated("gone")

	b := app.browser()
	res := b.do(http.MethodPost, "/api/v1/auth/login", creds{Email: email, Password: goodPassword})
	if res.status != http.StatusForbidden || res.errorCode(t) != "account_deactivated" {
		t.Errorf("login with the right password = %d %s, want 403 account_deactivated", res.status, res.body)
	}
	if got := b.meStatus(); got != http.StatusUnauthorized {
		t.Errorf("GET /me after signing in to a Deactivated Account = %d, want 401", got)
	}

	// A stranger guessing can't tell a Deactivated Account from no account.
	res = app.browser().do(http.MethodPost, "/api/v1/auth/login", creds{Email: email, Password: goodPassword + "!"})
	if res.status != http.StatusUnauthorized || res.errorCode(t) != "unauthenticated" {
		t.Errorf("login with a wrong password = %d %s, want 401 unauthenticated", res.status, res.body)
	}
}

func TestGoogleSignInToADeactivatedAccountSaysSo(t *testing.T) {
	app := startApp(t, tokyoMorning)
	const landing = "/signin?error=account_deactivated"

	t.Run("linked to the Google account", func(t *testing.T) {
		who := googleUser("gone-google")
		b := app.browser()
		b.googleSignIn(who, "UTC")
		if res := b.deactivate(who.Email); res.status != http.StatusNoContent {
			t.Fatalf("DELETE /me = %d %s, want 204", res.status, res.body)
		}
		again := app.browser()
		if landed, _ := again.googleSignIn(who, "UTC"); landed != landing {
			t.Errorf("Google sign-in sends the browser to %q, want %q", landed, landing)
		}
		if got := again.meStatus(); got != http.StatusUnauthorized {
			t.Errorf("GET /me after = %d, want 401", got)
		}
	})

	t.Run("by the email of a password account", func(t *testing.T) {
		email := app.deactivated("gone-password")
		who := googletest.Identity{Subject: "sub-" + email, Email: email, EmailVerified: true}
		b := app.browser()
		if landed, _ := b.googleSignIn(who, "UTC"); landed != landing {
			t.Errorf("Google sign-in sends the browser to %q, want %q", landed, landing)
		}
		if got := b.meStatus(); got != http.StatusUnauthorized {
			t.Errorf("GET /me after = %d, want 401", got)
		}
	})
}

func TestADeactivatedAccountIsSentNoMoreLinks(t *testing.T) {
	app := startApp(t, tokyoMorning)
	email := app.deactivated("no-links")
	sent := len(app.mail.Sent())

	b := app.browser()
	if res := b.forgot(email); res.status != http.StatusAccepted {
		t.Errorf("forgot = %d %s, want 202 as for any email", res.status, res.body)
	}
	if res := b.do(http.MethodPost, "/api/v1/auth/verify/resend", emailBody{email}); res.status != http.StatusAccepted {
		t.Errorf("resend = %d %s, want 202 as for any email", res.status, res.body)
	}
	if got := len(app.mail.Sent()); got != sent {
		t.Errorf("%d emails sent to a Deactivated Account, want none", got-sent)
	}
}

func TestADeactivatedEmailCantSignUpAgain(t *testing.T) {
	app := startApp(t, tokyoMorning)
	email := app.deactivated("again")

	res := app.browser().do(http.MethodPost, "/api/v1/auth/signup", creds{email, goodPassword, "UTC"})
	if res.status != http.StatusConflict || res.errorCode(t) != "email_taken" {
		t.Errorf("signup with a deactivated email = %d %s, want 409 email_taken", res.status, res.body)
	}
}
