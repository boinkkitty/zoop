// Package auth signs users up and in, and keeps their sessions.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/boinkkitty/zoop/internal/auth/password"
	"github.com/boinkkitty/zoop/internal/clock"
	"github.com/boinkkitty/zoop/internal/days"
	"github.com/boinkkitty/zoop/internal/ratelimit"
)

const (
	// SessionLifetime is how long a session lasts after it was last renewed.
	SessionLifetime = 30 * 24 * time.Hour
	// RenewEvery is the most often a session's expiry is pushed forward.
	RenewEvery = time.Hour
)

var (
	ErrEmailTaken = errors.New("auth: email already has an account")
	ErrBadLogin   = errors.New("auth: wrong email or password")
	ErrNoSession  = errors.New("auth: no live session")
)

// ValidationError is a problem with what the visitor typed, safe to show them.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

// Service is the auth module. It persists through its Store, tells time by its Clock,
// slows down guessing with its RateLimiter, and signs people in with Google through
// its GoogleProvider (nil if not configured).
type Service struct {
	store  Store
	clock  clock.Clock
	limits *ratelimit.RateLimiter
	google GoogleProvider
}

func New(store Store, c clock.Clock, limits *ratelimit.RateLimiter, google GoogleProvider) *Service {
	return &Service{store: store, clock: c, limits: limits, google: google}
}

// Session is a signed-in device.
type Session struct {
	Token     Token // only known when the session is created
	UserID    uuid.UUID
	ExpiresAt time.Time
	Renewed   bool // the expiry moved, so the cookie should be re-sent
}

// Signup creates a user with a password identity, their first Day being today in timezone, and signs them in.
// Every attempt counts toward SignupsPerIP, so one address can't farm accounts.
func (s *Service) Signup(ctx context.Context, rawEmail Email, plain Password, timezone, clientIP string) (Session, error) {
	if err := s.limits.Take(ctx, ratelimit.SignupsPerIP, clientIP); err != nil {
		return Session{}, err
	}
	email := normalizeEmail(rawEmail)
	if err := validateEmail(email); err != nil {
		return Session{}, err
	}
	if err := validateNewPassword(plain); err != nil {
		return Session{}, err
	}
	loc, err := loadTimezone(timezone)
	if err != nil {
		return Session{}, err
	}
	hash, err := password.Hash(string(plain))
	if err != nil {
		return Session{}, err
	}
	now := s.clock.Now(ctx)
	token, first, err := newSession(now)
	if err != nil {
		return Session{}, err
	}
	userID, err := s.store.CreateAccount(ctx, NewAccount{
		Email: string(email), FirstDay: days.Today(now, loc), Timezone: timezone, PasswordHash: hash, CreatedAt: now,
	}, first)
	if err != nil {
		return Session{}, err
	}
	return Session{Token: token, UserID: userID, ExpiresAt: first.ExpiresAt, Renewed: true}, nil
}

// Login signs a new device in with an email and password.
// Every failure is ErrBadLogin, so it doesn't reveal which emails have accounts.
// Too many failures return a ratelimit.ExceededError until the window ends, even for the right password.
func (s *Service) Login(ctx context.Context, rawEmail Email, plain Password, clientIP string) (Session, error) {
	email := normalizeEmail(rawEmail)
	if err := s.limits.Check(ctx, ratelimit.LoginFailuresPerEmail, string(email)); err != nil {
		return Session{}, err
	}
	if err := s.limits.Check(ctx, ratelimit.LoginFailuresPerIP, clientIP); err != nil {
		return Session{}, err
	}
	sess, err := s.login(ctx, email, plain)
	if errors.Is(err, ErrBadLogin) {
		if err := s.limits.Record(ctx, ratelimit.LoginFailuresPerEmail, string(email)); err != nil {
			return Session{}, err
		}
		if err := s.limits.Record(ctx, ratelimit.LoginFailuresPerIP, clientIP); err != nil {
			return Session{}, err
		}
	}
	return sess, err
}

func (s *Service) login(ctx context.Context, email Email, plain Password) (Session, error) {
	if len(plain) > password.MaxBytes {
		return Session{}, ErrBadLogin
	}
	userID, hash, err := s.store.PasswordLogin(ctx, string(email))
	if errors.Is(err, ErrNotFound) {
		// Spend the same time as a real check, so timing doesn't reveal it either.
		_, _ = password.Verify(dummyHash, string(plain))
		return Session{}, ErrBadLogin
	}
	if err != nil {
		return Session{}, err
	}
	ok, err := password.Verify(hash, string(plain))
	if err != nil {
		return Session{}, fmt.Errorf("verify stored hash: %w", err)
	}
	if !ok {
		return Session{}, ErrBadLogin
	}
	return s.signIn(ctx, userID)
}

// signIn starts a new session for an existing account.
func (s *Service) signIn(ctx context.Context, userID uuid.UUID) (Session, error) {
	token, sess, err := newSession(s.clock.Now(ctx))
	if err != nil {
		return Session{}, err
	}
	if err := s.store.CreateSession(ctx, userID, sess); err != nil {
		return Session{}, err
	}
	return Session{Token: token, UserID: userID, ExpiresAt: sess.ExpiresAt, Renewed: true}, nil
}

// dummyHash is checked when there's no account, so a miss costs as much as a wrong password.
var dummyHash = func() string {
	h, err := password.Hash("zoop-no-such-account")
	if err != nil {
		panic(err)
	}
	return h
}()

// Authenticate finds the live session for token, pushing its expiry forward if it hasn't been for RenewEvery.
func (s *Service) Authenticate(ctx context.Context, token Token) (Session, error) {
	if token == "" {
		return Session{}, ErrNoSession
	}
	now := s.clock.Now(ctx)
	row, err := s.store.LiveSession(ctx, hashToken(token), now)
	if errors.Is(err, ErrNotFound) {
		return Session{}, ErrNoSession
	}
	if err != nil {
		return Session{}, err
	}
	sess := Session{UserID: row.UserID, ExpiresAt: row.ExpiresAt}
	if now.Sub(row.LastSeenAt) >= RenewEvery {
		sess.ExpiresAt = now.Add(SessionLifetime)
		sess.Renewed = true
		if err := s.store.RenewSession(ctx, row.ID, now, sess.ExpiresAt); err != nil {
			return Session{}, err
		}
	}
	return sess, nil
}

// Logout ends the session for token. Ending a session that doesn't exist is not an error.
func (s *Service) Logout(ctx context.Context, token Token) error {
	if token == "" {
		return nil
	}
	return s.store.DeleteSession(ctx, hashToken(token))
}

// Account is what a signed-in user can see about themselves.
type Account struct {
	Email           string
	EmailVerified   bool
	Plan            string
	Timezone        string
	PendingTimezone *string
	FirstDay        time.Time
	Today           time.Time
	GraceUntil      *time.Time
	SignInMethods   []string
}

// Account describes userID as of now.
func (s *Service) Account(ctx context.Context, userID uuid.UUID) (Account, error) {
	u, err := s.store.User(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return Account{}, ErrNoSession
	}
	if err != nil {
		return Account{}, err
	}
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		return Account{}, fmt.Errorf("stored timezone %q: %w", u.Timezone, err)
	}
	now := s.clock.Now(ctx)
	return Account{
		Email:           u.Email,
		EmailVerified:   u.EmailVerifiedAt != nil,
		Plan:            u.Plan,
		Timezone:        u.Timezone,
		PendingTimezone: u.NextTimezone,
		FirstDay:        u.FirstDay,
		Today:           days.Today(now, loc),
		GraceUntil:      days.GraceUntil(now, loc),
		SignInMethods:   u.SignInMethods,
	}, nil
}

// newSession makes a fresh random token, returning it and the session to store under its hash.
func newSession(now time.Time) (Token, NewSession, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", NewSession{}, err
	}
	token := Token(base64.RawURLEncoding.EncodeToString(raw))
	return token, NewSession{TokenHash: hashToken(token), CreatedAt: now, ExpiresAt: now.Add(SessionLifetime)}, nil
}

// hashToken is what's stored, so a leaked sessions table can't be replayed.
func hashToken(token Token) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func normalizeEmail(email Email) Email {
	return Email(strings.ToLower(strings.TrimSpace(string(email))))
}

// MinPasswordLen is the shortest password a new account may use.
const MinPasswordLen = 8

func validateEmail(email Email) error {
	addr, err := mail.ParseAddress(string(email))
	if err != nil || addr.Address != string(email) || len(email) > 254 {
		return &ValidationError{"Enter a valid email address."}
	}
	return nil
}

func validateNewPassword(plain Password) error {
	if len(plain) < MinPasswordLen {
		return &ValidationError{fmt.Sprintf("Use at least %d characters for your password.", MinPasswordLen)}
	}
	if len(plain) > password.MaxBytes {
		return &ValidationError{fmt.Sprintf("Use at most %d characters for your password.", password.MaxBytes)}
	}
	return nil
}

func loadTimezone(name string) (*time.Location, error) {
	if name == "" || name == "Local" {
		return nil, &ValidationError{"Unknown timezone."}
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, &ValidationError{"Unknown timezone."}
	}
	return loc, nil
}
