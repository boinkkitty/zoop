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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/boinkkitty/zoop/internal/auth/password"
	"github.com/boinkkitty/zoop/internal/clock"
	"github.com/boinkkitty/zoop/internal/days"
	"github.com/boinkkitty/zoop/internal/db/dbgen"
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

// Service is the auth module. Time comes from its Clock.
type Service struct {
	pool  *pgxpool.Pool
	q     *dbgen.Queries
	clock clock.Clock
}

func New(pool *pgxpool.Pool, c clock.Clock) *Service {
	return &Service{pool: pool, q: dbgen.New(pool), clock: c}
}

// Session is a signed-in device.
type Session struct {
	Token     string // only known when the session is created
	UserID    uuid.UUID
	ExpiresAt time.Time
	Renewed   bool // the expiry moved, so the cookie should be re-sent
}

// Signup creates a user with a password identity, their first Day being today in timezone, and signs them in.
func (s *Service) Signup(ctx context.Context, email, plain, timezone string) (Session, error) {
	email = normalizeEmail(email)
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
	hash, err := password.Hash(plain)
	if err != nil {
		return Session{}, err
	}
	now := s.clock.Now()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback(ctx)
	q := s.q.WithTx(tx)

	userID, err := q.CreateUser(ctx, dbgen.CreateUserParams{
		Email: email, FirstDay: days.Today(now, loc), Timezone: timezone, Now: now,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrEmailTaken
	}
	if err != nil {
		return Session{}, fmt.Errorf("create user: %w", err)
	}
	if err := q.CreatePasswordIdentity(ctx, dbgen.CreatePasswordIdentityParams{
		UserID: userID, Subject: email, PasswordHash: &hash, Now: now,
	}); err != nil {
		return Session{}, fmt.Errorf("create password identity: %w", err)
	}
	sess, err := createSession(ctx, q, userID, now)
	if err != nil {
		return Session{}, err
	}
	return sess, tx.Commit(ctx)
}

// Login signs a new device in with an email and password.
// Every failure is ErrBadLogin, so it doesn't reveal which emails have accounts.
func (s *Service) Login(ctx context.Context, email, plain string) (Session, error) {
	if len(plain) > password.MaxBytes {
		return Session{}, ErrBadLogin
	}
	row, err := s.q.GetPasswordLogin(ctx, normalizeEmail(email))
	if errors.Is(err, pgx.ErrNoRows) {
		// Spend the same time as a real check, so timing doesn't reveal it either.
		_, _ = password.Verify(dummyHash, plain)
		return Session{}, ErrBadLogin
	}
	if err != nil {
		return Session{}, err
	}
	ok, err := password.Verify(*row.PasswordHash, plain)
	if err != nil {
		return Session{}, fmt.Errorf("verify stored hash: %w", err)
	}
	if !ok {
		return Session{}, ErrBadLogin
	}
	return createSession(ctx, s.q, row.ID, s.clock.Now())
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
func (s *Service) Authenticate(ctx context.Context, token string) (Session, error) {
	if token == "" {
		return Session{}, ErrNoSession
	}
	now := s.clock.Now()
	row, err := s.q.GetLiveSession(ctx, dbgen.GetLiveSessionParams{TokenHash: hashToken(token), Now: now})
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNoSession
	}
	if err != nil {
		return Session{}, err
	}
	sess := Session{UserID: row.UserID, ExpiresAt: row.ExpiresAt}
	if now.Sub(row.LastSeenAt) >= RenewEvery {
		sess.ExpiresAt = now.Add(SessionLifetime)
		sess.Renewed = true
		if err := s.q.RenewSession(ctx, dbgen.RenewSessionParams{ID: row.ID, Now: now, ExpiresAt: sess.ExpiresAt}); err != nil {
			return Session{}, err
		}
	}
	return sess, nil
}

// Logout ends the session for token. Ending a session that doesn't exist is not an error.
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.q.DeleteSession(ctx, hashToken(token))
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
	u, err := s.q.GetUser(ctx, userID)
	if err != nil {
		return Account{}, err
	}
	methods, err := s.q.ListSignInMethods(ctx, userID)
	if err != nil {
		return Account{}, err
	}
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		return Account{}, fmt.Errorf("stored timezone %q: %w", u.Timezone, err)
	}
	now := s.clock.Now()
	return Account{
		Email:           u.Email,
		EmailVerified:   u.EmailVerifiedAt != nil,
		Plan:            u.Plan,
		Timezone:        u.Timezone,
		PendingTimezone: u.NextTimezone,
		FirstDay:        u.FirstDay,
		Today:           days.Today(now, loc),
		GraceUntil:      days.GraceUntil(now, loc),
		SignInMethods:   methods,
	}, nil
}

func createSession(ctx context.Context, q *dbgen.Queries, userID uuid.UUID, now time.Time) (Session, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return Session{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	expires := now.Add(SessionLifetime)
	if err := q.CreateSession(ctx, dbgen.CreateSessionParams{
		TokenHash: hashToken(token), UserID: userID, Now: now, ExpiresAt: expires,
	}); err != nil {
		return Session{}, fmt.Errorf("create session: %w", err)
	}
	return Session{Token: token, UserID: userID, ExpiresAt: expires, Renewed: true}, nil
}

// hashToken is what's stored, so a leaked sessions table can't be replayed.
func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func normalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

// MinPasswordLen is the shortest password a new account may use.
const MinPasswordLen = 8

func validateEmail(email string) error {
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || len(email) > 254 {
		return &ValidationError{"Enter a valid email address."}
	}
	return nil
}

func validateNewPassword(plain string) error {
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
