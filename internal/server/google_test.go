package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/boinkkitty/zoop/internal/google/googletest"
)

// googleSignIn walks this browser through Google sign-in, with the stub signing in as who.
// It returns where the app finally sends the browser, and what the app asked the provider for.
func (b *browser) googleSignIn(who googletest.Identity, timezone string) (string, googletest.AuthRequest) {
	b.app.t.Helper()
	b.app.google.As(who)
	return b.followFromGoogle(b.googleStart(timezone)), b.app.google.LastAuthRequest()
}

// googleStart begins Google sign-in and returns the provider URL the app sends the browser to.
func (b *browser) googleStart(timezone string) string {
	b.app.t.Helper()
	start := b.do(http.MethodGet, "/api/v1/auth/google/start?timezone="+url.QueryEscape(timezone), nil)
	if start.status != http.StatusFound {
		b.app.t.Fatalf("google start = %d %s, want 302", start.status, start.body)
	}
	toGoogle := start.header.Get("Location")
	if !strings.HasPrefix(toGoogle, b.app.google.Issuer()+"/authorize?") {
		b.app.t.Fatalf("google start redirects to %q, want the provider's authorize endpoint", toGoogle)
	}
	return toGoogle
}

// followFromGoogle visits the provider (which keeps none of our cookies), then the app's callback,
// returning where the callback sends the browser.
func (b *browser) followFromGoogle(authorizeURL string) string {
	b.app.t.Helper()
	return b.callback(b.app.authorize(authorizeURL))
}

// authorize visits the provider and returns the callback path it sends the browser back to.
func (a *testApp) authorize(authorizeURL string) string {
	a.t.Helper()
	res, err := noRedirects.Get(authorizeURL)
	if err != nil {
		a.t.Fatal(err)
	}
	res.Body.Close()
	back := res.Header.Get("Location")
	if !strings.HasPrefix(back, a.srv.URL+"/") {
		a.t.Fatalf("provider redirected to %q, want the app", back)
	}
	return strings.TrimPrefix(back, a.srv.URL)
}

func (b *browser) callback(path string) string {
	b.app.t.Helper()
	cb := b.do(http.MethodGet, path, nil)
	if cb.status != http.StatusFound {
		b.app.t.Fatalf("google callback = %d %s, want 302", cb.status, cb.body)
	}
	return cb.header.Get("Location")
}

func googleUser(prefix string) googletest.Identity {
	email := uniqueEmail(prefix)
	return googletest.Identity{Subject: "sub-" + email, Email: email, EmailVerified: true}
}

func TestANewVisitorSignsUpWithGoogle(t *testing.T) {
	app := startApp(t, tokyoMorning)
	b := app.browser()
	who := googleUser("Grace.Hopper")
	who.Email = upper(who.Email)

	landed, asked := b.googleSignIn(who, "Asia/Tokyo")

	if landed != "/" {
		t.Errorf("after Google sign-in the app sends the browser to %q, want /", landed)
	}
	if asked.CodeChallengeMethod != "S256" || asked.CodeChallenge == "" {
		t.Errorf("authorize asked with PKCE method %q challenge %q, want S256", asked.CodeChallengeMethod, asked.CodeChallenge)
	}
	if asked.State == "" || asked.Nonce == "" {
		t.Errorf("authorize asked with state %q nonce %q, want both", asked.State, asked.Nonce)
	}
	got, res := b.me()
	if res.status != http.StatusOK {
		t.Fatalf("GET /me after Google signup = %d %s, want 200", res.status, res.body)
	}
	if got.Email != lower(who.Email) || !got.EmailVerified {
		t.Errorf("email %q verified %v, want %q verified (Google vouched for it)", got.Email, got.EmailVerified, lower(who.Email))
	}
	if got.Timezone != "Asia/Tokyo" || got.FirstDay != "2026-11-26" {
		t.Errorf("timezone %q firstDay %q, want Asia/Tokyo and 2026-11-26 (Tokyo's date)", got.Timezone, got.FirstDay)
	}
	if len(got.SignInMethods) != 1 || got.SignInMethods[0] != "google" {
		t.Errorf("signInMethods = %v, want [google]", got.SignInMethods)
	}
}

func TestGoogleTakesOverAnUnverifiedEmailAccountSoAPreRegistrantIsShutOut(t *testing.T) {
	app := startApp(t, tokyoMorning)
	victim := googleUser("victim")
	attacker := app.browser()
	attacker.signup(victim.Email, "Asia/Tokyo") // never verified: the attacker can't read the inbox

	app.clock.Advance(2 * day)
	owner := app.browser()
	if landed, _ := owner.googleSignIn(victim, "Europe/London"); landed != "/" {
		t.Fatalf("owner's Google sign-in sends the browser to %q, want /", landed)
	}

	got, res := owner.me()
	if res.status != http.StatusOK {
		t.Fatalf("GET /me for the owner = %d %s, want 200", res.status, res.body)
	}
	if got.Email != lower(victim.Email) || !got.EmailVerified {
		t.Errorf("owner's account %q verified %v, want %q verified", got.Email, got.EmailVerified, lower(victim.Email))
	}
	if len(got.SignInMethods) != 1 || got.SignInMethods[0] != "google" {
		t.Errorf("signInMethods = %v, want [google] (the unverified password removed)", got.SignInMethods)
	}
	if got := attacker.meStatus(); got != http.StatusUnauthorized {
		t.Errorf("GET /me in the pre-registrant's session = %d, want 401 (signed out)", got)
	}
	res = app.browser().do(http.MethodPost, "/api/v1/auth/login", creds{Email: victim.Email, Password: goodPassword})
	if res.status != http.StatusUnauthorized {
		t.Errorf("login with the pre-registered password = %d, want 401", res.status)
	}
}

func TestGoogleSignInIsRefusedWhenGoogleHasNotVerifiedTheEmail(t *testing.T) {
	app := startApp(t, tokyoMorning)

	t.Run("an account has the email", func(t *testing.T) {
		email := uniqueEmail("holder")
		holder := app.browser()
		holder.signup(email, "UTC")
		claimant := googletest.Identity{Subject: "sub-" + email, Email: email, EmailVerified: false}

		b := app.browser()
		if landed, _ := b.googleSignIn(claimant, "UTC"); landed != "/signin?error=google_unverified" {
			t.Errorf("Google sign-in sends the browser to %q, want /signin?error=google_unverified", landed)
		}
		if got := b.meStatus(); got != http.StatusUnauthorized {
			t.Errorf("GET /me after the refused sign-in = %d, want 401", got)
		}
		got, _ := holder.me()
		if len(got.SignInMethods) != 1 || got.SignInMethods[0] != "password" {
			t.Errorf("holder's signInMethods = %v, want [password] (nothing linked or removed)", got.SignInMethods)
		}
		app.browser().login(email)
	})

	t.Run("no account has the email", func(t *testing.T) {
		email := uniqueEmail("unclaimed")
		b := app.browser()
		landed, _ := b.googleSignIn(googletest.Identity{Subject: "sub-" + email, Email: email}, "UTC")
		if landed != "/signin?error=google_unverified" {
			t.Errorf("Google sign-in sends the browser to %q, want /signin?error=google_unverified", landed)
		}
		if got := b.meStatus(); got != http.StatusUnauthorized {
			t.Errorf("GET /me after the refused sign-in = %d, want 401", got)
		}
		app.browser().signup(email, "UTC") // no account was made, so the real owner can still sign up
	})
}

func TestTheGoogleCallbackOnlyCompletesTheSignInThisBrowserStartedOnce(t *testing.T) {
	// State, PKCE and the nonce each tie the callback to the browser that started it,
	// and each alone stops these; this proves the app refuses, whichever catches it first.
	app := startApp(t, tokyoMorning)
	refused := "/signin?error=google_failed"

	t.Run("a callback this browser never started", func(t *testing.T) {
		// Login CSRF: the attacker signs in at Google and hands the victim their callback URL.
		app.google.As(googleUser("attacker"))
		attackerCallback := app.authorize(app.browser().googleStart("UTC"))
		victim := app.browser()
		if landed := victim.callback(attackerCallback); landed != refused {
			t.Errorf("callback without a started sign-in sends the browser to %q, want %s", landed, refused)
		}
		if got := victim.meStatus(); got != http.StatusUnauthorized {
			t.Errorf("GET /me for the victim = %d, want 401 (not signed in as the attacker)", got)
		}
	})

	t.Run("a callback for a different sign-in than this browser started", func(t *testing.T) {
		app.google.As(googleUser("someone"))
		otherCallback := app.authorize(app.browser().googleStart("UTC"))
		b := app.browser()
		b.googleStart("UTC")
		if landed := b.callback(otherCallback); landed != refused {
			t.Errorf("callback with another sign-in's state sends the browser to %q, want %s", landed, refused)
		}
		if got := b.meStatus(); got != http.StatusUnauthorized {
			t.Errorf("GET /me = %d, want 401", got)
		}
	})

	t.Run("the same sign-in completed twice", func(t *testing.T) {
		app.google.As(googleUser("twice"))
		b := app.browser()
		toGoogle := b.googleStart("UTC")
		if landed := b.followFromGoogle(toGoogle); landed != "/" {
			t.Fatalf("first callback sends the browser to %q, want /", landed)
		}
		b.do(http.MethodPost, "/api/v1/auth/logout", nil)
		// Google would hand out a fresh code for the same state; the state is spent.
		if landed := b.followFromGoogle(toGoogle); landed != refused {
			t.Errorf("second callback for the same state sends the browser to %q, want %s", landed, refused)
		}
		if got := b.meStatus(); got != http.StatusUnauthorized {
			t.Errorf("GET /me after the replay = %d, want 401", got)
		}
	})
}

func TestGoogleSignInWithoutGoogleConfiguredSendsTheBrowserBackToSignIn(t *testing.T) {
	rec := get(t, newApp(t), "/api/v1/auth/google/start?timezone=UTC")
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/signin?error=google_unavailable" {
		t.Errorf("start = %d to %q, want 302 to /signin?error=google_unavailable", rec.Code, rec.Header().Get("Location"))
	}
}

func TestAnIDTokenNotIssuedForThisSignInIsRefused(t *testing.T) {
	app := startApp(t, tokyoMorning)
	app.google.IssueWrongNonce() // e.g. a token replayed from another sign-in
	b := app.browser()

	if landed, _ := b.googleSignIn(googleUser("replayed"), "UTC"); landed != "/signin?error=google_failed" {
		t.Errorf("sign-in with a foreign nonce sends the browser to %q, want /signin?error=google_failed", landed)
	}
	if got := b.meStatus(); got != http.StatusUnauthorized {
		t.Errorf("GET /me = %d, want 401", got)
	}
}

func TestAReturningGoogleUserIsFoundByTheirGoogleAccountNotTheirEmail(t *testing.T) {
	app := startApp(t, tokyoMorning)
	who := googleUser("returning")
	app.browser().googleSignIn(who, "Asia/Tokyo")

	// They changed their Google email since; the Google account (sub) is the same.
	renamed := who
	renamed.Email = uniqueEmail("renamed")
	app.clock.Advance(3 * day)
	laptop := app.browser()
	if landed, _ := laptop.googleSignIn(renamed, "Europe/London"); landed != "/" {
		t.Fatalf("second Google sign-in sends the browser to %q, want /", landed)
	}

	got, res := laptop.me()
	if res.status != http.StatusOK {
		t.Fatalf("GET /me = %d %s, want 200", res.status, res.body)
	}
	if got.Email != lower(who.Email) || got.FirstDay != "2026-11-26" || got.Timezone != "Asia/Tokyo" {
		t.Errorf("got account %q first day %q in %q, want the original %q from 2026-11-26 in Asia/Tokyo",
			got.Email, got.FirstDay, got.Timezone, lower(who.Email))
	}
}
