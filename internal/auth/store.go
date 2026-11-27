package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Store is the port auth needs from persistence. internal/postgres implements it.
// Each method is one thing auth needs done, so transactions stay inside the adapter.
type Store interface {
	// CreateAccount stores a new user with their password identity and first session, all or nothing.
	// It returns ErrEmailTaken if the email already has an account, including a Deactivated one.
	CreateAccount(ctx context.Context, a NewAccount, first NewSession) (uuid.UUID, error)

	// CreateGoogleAccount stores a new user, verified by Google, with their Google identity
	// and first session, all or nothing. It returns ErrEmailTaken if the email has an account.
	CreateGoogleAccount(ctx context.Context, a NewGoogleAccount, first NewSession) (uuid.UUID, error)

	// GoogleLogin finds the live account linked to a Google subject, or ErrNotFound.
	GoogleLogin(ctx context.Context, subject string) (uuid.UUID, error)

	// UserByEmail finds the live account with this email, or ErrNotFound.
	UserByEmail(ctx context.Context, email string) (EmailOwner, error)

	// LinkGoogle adds a Google identity to an account and starts a session on it, all or nothing.
	// With TakeOver, and only while the account's email is still unverified, it first marks the
	// email verified, removes the password identity and ends every other session.
	LinkGoogle(ctx context.Context, l GoogleLink, first NewSession) error

	// PasswordLogin finds the password hash for a live account's email, or ErrNotFound.
	PasswordLogin(ctx context.Context, email string) (userID uuid.UUID, passwordHash string, err error)

	CreateSession(ctx context.Context, userID uuid.UUID, s NewSession) error

	// LiveSession finds the session for tokenHash if it hasn't expired at now and its account is live, or ErrNotFound.
	LiveSession(ctx context.Context, tokenHash []byte, now time.Time) (StoredSession, error)

	RenewSession(ctx context.Context, id uuid.UUID, lastSeenAt, expiresAt time.Time) error

	// DeleteSession ends the session for tokenHash. Deleting one that doesn't exist is not an error.
	DeleteSession(ctx context.Context, tokenHash []byte) error

	// User describes a live account, or returns ErrNotFound.
	User(ctx context.Context, id uuid.UUID) (StoredUser, error)
}

// ErrNotFound is how a Store says there's no such row.
var ErrNotFound = errors.New("auth: not found")

// NewAccount is a user to create, with the hash of their password.
type NewAccount struct {
	Email        string
	FirstDay     time.Time
	Timezone     string
	PasswordHash string
	CreatedAt    time.Time
}

// NewGoogleAccount is a user to create from a Google sign-in. Their email counts as verified at CreatedAt.
type NewGoogleAccount struct {
	Email     string
	Subject   string // Google's "sub"
	FirstDay  time.Time
	Timezone  string
	CreatedAt time.Time
}

// EmailOwner is the account that holds an email.
type EmailOwner struct {
	UserID        uuid.UUID
	EmailVerified bool
}

// GoogleLink links a Google subject to an existing account.
type GoogleLink struct {
	UserID   uuid.UUID
	Subject  string
	TakeOver bool // the email was never verified: whoever set its password may not own it
	Now      time.Time
}

// NewSession is a session to create. Only the token's hash is stored.
type NewSession struct {
	TokenHash []byte
	CreatedAt time.Time
	ExpiresAt time.Time
}

type StoredSession struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	LastSeenAt time.Time
	ExpiresAt  time.Time
}

type StoredUser struct {
	Email           string
	EmailVerifiedAt *time.Time
	Plan            string
	FirstDay        time.Time
	Timezone        string
	NextTimezone    *string
	SignInMethods   []string
}
