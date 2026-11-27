package server_test

import (
	"net/http"
	"net/url"
	"regexp"
	"testing"
	"time"
)

var linkPattern = regexp.MustCompile(`https?://\S+`)

// emailLink is the one link in the newest email to `to`, failing if there's none.
func (a *testApp) emailLink(to string) *url.URL {
	a.t.Helper()
	sent := a.mail.Sent()
	for i := len(sent) - 1; i >= 0; i-- {
		if sent[i].To != to {
			continue
		}
		raw := linkPattern.FindString(sent[i].Text)
		if raw == "" {
			a.t.Fatalf("email to %s has no link: %q", to, sent[i].Text)
		}
		u, err := url.Parse(raw)
		if err != nil {
			a.t.Fatal(err)
		}
		return u
	}
	a.t.Fatalf("no email sent to %s; sent: %+v", to, sent)
	return nil
}

func TestSignupEmailsAVerifyLink(t *testing.T) {
	app := startApp(t, tokyoMorning)
	email := lower(uniqueEmail("verify"))
	app.browser().signup(email, "UTC")

	link := app.emailLink(email)
	if link.Path != "/verify" || link.Query().Get("token") == "" {
		t.Fatalf("verify link = %s, want %s/verify?token=…", link, app.srv.URL)
	}
	if got := link.Scheme + "://" + link.Host; got != app.srv.URL {
		t.Errorf("link host = %s, want the app's own URL %s", got, app.srv.URL)
	}
}

type tokenBody struct {
	Token string `json:"token"`
}

func (b *browser) verify(token string) response {
	b.app.t.Helper()
	return b.do(http.MethodPost, "/api/v1/auth/verify", tokenBody{token})
}

func TestTheVerifyLinksTokenVerifiesTheEmail(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	email := lower(uniqueEmail("verify"))
	b.signup(email, "UTC")

	// The link may be opened in another browser, signed in or not.
	res := app.browser().verify(app.emailLink(email).Query().Get("token"))
	if res.status != http.StatusNoContent {
		t.Fatalf("verify = %d %s, want 204", res.status, res.body)
	}
	if got, _ := b.me(); !got.EmailVerified {
		t.Error("emailVerified = false after verifying, want true")
	}
}

func TestAVerifyLinkWorksOnlyOnce(t *testing.T) {
	app := startApp(t, tokyoMorning)
	email := lower(uniqueEmail("once"))
	app.browser().signup(email, "UTC")
	token := app.emailLink(email).Query().Get("token")

	if res := app.browser().verify(token); res.status != http.StatusNoContent {
		t.Fatalf("first verify = %d %s, want 204", res.status, res.body)
	}
	res := app.browser().verify(token)
	if res.status != http.StatusBadRequest || res.errorCode(t) != "bad_token" {
		t.Errorf("second verify = %d %s, want 400 bad_token", res.status, res.body)
	}
}

func TestAVerifyLinkLastsADay(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	email := lower(uniqueEmail("expiry"))
	b.signup(email, "UTC")
	token := app.emailLink(email).Query().Get("token")

	app.clock.Advance(24 * time.Hour)
	res := app.browser().verify(token)
	if res.status != http.StatusBadRequest || res.errorCode(t) != "bad_token" {
		t.Fatalf("verify after 24h = %d %s, want 400 bad_token", res.status, res.body)
	}
	b.login(email) // the session may have aged too
	if got, _ := b.me(); got.EmailVerified {
		t.Error("emailVerified = true from an expired link")
	}
}

func TestAnUnknownVerifyTokenIsRefused(t *testing.T) {
	app := startApp(t, tokyoMorning)
	for _, token := range []string{"", "not-a-real-token"} {
		res := app.browser().verify(token)
		if res.status != http.StatusBadRequest || res.errorCode(t) != "bad_token" {
			t.Errorf("verify %q = %d %s, want 400 bad_token", token, res.status, res.body)
		}
	}
}

type emailBody struct {
	Email string `json:"email"`
}

func TestResendSendsAFreshVerifyLink(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	email := lower(uniqueEmail("resend"))
	b.signup(email, "UTC")
	first := app.emailLink(email).Query().Get("token")

	res := b.do(http.MethodPost, "/api/v1/auth/verify/resend", emailBody{email})
	if res.status != http.StatusAccepted {
		t.Fatalf("resend = %d %s, want 202", res.status, res.body)
	}
	again := app.emailLink(email).Query().Get("token")
	if again == first {
		t.Fatal("resend sent the same token again, want a fresh one")
	}
	if res := b.verify(again); res.status != http.StatusNoContent {
		t.Fatalf("verify resent link = %d %s, want 204", res.status, res.body)
	}
	if got, _ := b.me(); !got.EmailVerified {
		t.Error("emailVerified = false after the resent link, want true")
	}
}

func TestResendDoesntRevealWhichEmailsHaveAccounts(t *testing.T) {
	app := startApp(t, tokyoMorning)
	known := lower(uniqueEmail("known"))
	app.browser().signup(known, "UTC")
	before := len(app.mail.Sent())

	forKnown := app.browser().do(http.MethodPost, "/api/v1/auth/verify/resend", emailBody{known})
	forUnknown := app.browser().do(http.MethodPost, "/api/v1/auth/verify/resend", emailBody{lower(uniqueEmail("nobody"))})
	if forKnown.status != forUnknown.status || string(forKnown.body) != string(forUnknown.body) {
		t.Errorf("resend answered %d %q for an account but %d %q for none, want the same",
			forKnown.status, forKnown.body, forUnknown.status, forUnknown.body)
	}
	if sent := len(app.mail.Sent()) - before; sent != 1 {
		t.Errorf("%d emails sent, want 1 (only to the real account)", sent)
	}
}

func TestResendSendsNothingOnceVerified(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	email := lower(uniqueEmail("done"))
	b.signup(email, "UTC")
	b.verify(app.emailLink(email).Query().Get("token"))
	before := len(app.mail.Sent())

	res := b.do(http.MethodPost, "/api/v1/auth/verify/resend", emailBody{email})
	if res.status != http.StatusAccepted {
		t.Fatalf("resend = %d %s, want 202", res.status, res.body)
	}
	if sent := len(app.mail.Sent()) - before; sent != 0 {
		t.Errorf("%d emails sent to a verified address, want 0", sent)
	}
}

func TestEmailRequestsAreLimitedPerEmail(t *testing.T) {
	app := startApp(t, tokyoMorning)
	email := lower(uniqueEmail("flood"))
	app.browser().signup(email, "UTC")

	for i := range 3 {
		if res := app.browser().do(http.MethodPost, "/api/v1/auth/verify/resend", emailBody{email}); res.status != http.StatusAccepted {
			t.Fatalf("resend %d = %d %s, want 202", i+1, res.status, res.body)
		}
	}
	res := app.browser().do(http.MethodPost, "/api/v1/auth/verify/resend", emailBody{email})
	if res.status != http.StatusTooManyRequests {
		t.Fatalf("4th resend in an hour = %d %s, want 429", res.status, res.body)
	}
	app.clock.Advance(time.Hour)
	if res := app.browser().do(http.MethodPost, "/api/v1/auth/verify/resend", emailBody{email}); res.status != http.StatusAccepted {
		t.Errorf("resend an hour later = %d %s, want 202", res.status, res.body)
	}
}
