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
