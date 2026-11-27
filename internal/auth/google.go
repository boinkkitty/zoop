package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/boinkkitty/zoop/internal/days"
)

var (
	// ErrGoogleUnavailable means Google sign-in isn't configured on this server.
	ErrGoogleUnavailable = errors.New("auth: google sign-in is not configured")
	// ErrGoogleFailed covers a sign-in that can't be trusted or didn't complete:
	// a missing or mismatched state, a bad code, or an ID token that fails verification.
	ErrGoogleFailed = errors.New("auth: google sign-in failed")
	// ErrGoogleUnverified means Google hasn't verified the email on a Google account we don't know yet.
	ErrGoogleUnverified = errors.New("auth: google has not verified this email")
)

// GoogleIdentity is who Google says signed in, from a verified ID token.
type GoogleIdentity struct {
	Subject       string // Google's stable account ID ("sub"); identities are keyed on it
	Email         string
	EmailVerified bool
}

// GoogleProvider is the port to Google. internal/google implements it over OpenID Connect.
type GoogleProvider interface {
	// AuthURL is where to send the browser, asking for an ID token bound to nonce,
	// with a PKCE S256 challenge derived from verifier.
	AuthURL(ctx context.Context, state, nonce, verifier string) (string, error)
	// Exchange redeems code with the PKCE verifier and returns the verified identity.
	// It fails unless the ID token is valid for this client and carries nonce.
	Exchange(ctx context.Context, code, verifier, nonce string) (GoogleIdentity, error)
}

// GoogleFlow is one sign-in in progress. The browser keeps it between start and callback;
// it is used once.
type GoogleFlow struct {
	State    string `json:"s"`
	Nonce    string `json:"n"`
	Verifier string `json:"v"`
	Timezone string `json:"tz"` // used if this sign-in creates the account
}

// StartGoogle begins a Google sign-in from a browser in timezone.
// It returns the URL to send the browser to and the flow it must keep until the callback.
func (s *Service) StartGoogle(ctx context.Context, timezone string) (string, GoogleFlow, error) {
	if s.google == nil {
		return "", GoogleFlow{}, ErrGoogleUnavailable
	}
	if _, err := loadTimezone(timezone); err != nil {
		return "", GoogleFlow{}, err
	}
	flow := GoogleFlow{State: randomString(), Nonce: randomString(), Verifier: randomString(), Timezone: timezone}
	u, err := s.google.AuthURL(ctx, flow.State, flow.Nonce, flow.Verifier)
	if err != nil {
		return "", GoogleFlow{}, fmt.Errorf("%w: %v", ErrGoogleUnavailable, err)
	}
	return u, flow, nil
}

// FinishGoogle completes the sign-in flow began, given the state and code Google sent back.
func (s *Service) FinishGoogle(ctx context.Context, flow GoogleFlow, state, code string) (Session, error) {
	if s.google == nil {
		return Session{}, ErrGoogleUnavailable
	}
	if flow.State == "" || subtle.ConstantTimeCompare([]byte(flow.State), []byte(state)) != 1 {
		return Session{}, fmt.Errorf("%w: state does not match", ErrGoogleFailed)
	}
	who, err := s.google.Exchange(ctx, code, flow.Verifier, flow.Nonce)
	if err != nil {
		return Session{}, fmt.Errorf("%w: %v", ErrGoogleFailed, err)
	}

	// Identities are keyed on Google's sub: the email on a Google account can change.
	userID, err := s.store.GoogleLogin(ctx, who.Subject)
	if err == nil {
		return s.signIn(ctx, userID)
	}
	if !errors.Is(err, ErrNotFound) {
		return Session{}, err
	}

	// Past here the email decides which account this is, so Google must vouch for it:
	// otherwise anyone could put someone else's address on a Google account and claim it.
	if !who.EmailVerified {
		return Session{}, ErrGoogleUnverified
	}
	email := normalizeEmail(who.Email)
	now := s.clock.Now(ctx)
	token, first, err := newSession(now)
	if err != nil {
		return Session{}, err
	}
	signedIn := func(userID uuid.UUID) Session {
		return Session{Token: token, UserID: userID, ExpiresAt: first.ExpiresAt, Renewed: true}
	}

	// Google vouches for the email, so whoever holds it here may sign in with Google.
	// If it was never verified here, the password on it may be a pre-registrant's: it goes.
	owner, err := s.store.UserByEmail(ctx, email)
	if err == nil {
		link := GoogleLink{UserID: owner.UserID, Subject: who.Subject, TakeOver: !owner.EmailVerified, Now: now}
		if err := s.store.LinkGoogle(ctx, link, first); err != nil {
			return Session{}, err
		}
		return signedIn(owner.UserID), nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Session{}, err
	}

	loc, err := loadTimezone(flow.Timezone)
	if err != nil {
		return Session{}, err
	}
	userID, err = s.store.CreateGoogleAccount(ctx, NewGoogleAccount{
		Email: email, Subject: who.Subject,
		FirstDay: days.Today(now, loc), Timezone: flow.Timezone, CreatedAt: now,
	}, first)
	if err != nil {
		return Session{}, err
	}
	return signedIn(userID), nil
}

func randomString() string {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		panic(err) // crypto/rand never fails on supported platforms
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}
