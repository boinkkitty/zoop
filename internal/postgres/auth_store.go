// Package postgres is the database adapter: it implements the core's store ports with sqlc.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/boinkkitty/zoop/internal/auth"
	"github.com/boinkkitty/zoop/internal/db/dbgen"
)

// AuthStore implements auth.Store.
type AuthStore struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

var _ auth.Store = (*AuthStore)(nil)

func NewAuthStore(pool *pgxpool.Pool) *AuthStore {
	return &AuthStore{pool: pool, q: dbgen.New(pool)}
}

func (s *AuthStore) CreateAccount(ctx context.Context, a auth.NewAccount, first auth.NewSession) (uuid.UUID, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)
	q := s.q.WithTx(tx)

	userID, err := q.CreateUser(ctx, dbgen.CreateUserParams{
		Email: a.Email, FirstDay: a.FirstDay, Timezone: a.Timezone, Now: a.CreatedAt,
	})
	if errors.Is(err, pgx.ErrNoRows) { // ON CONFLICT (email) DO NOTHING
		return uuid.Nil, auth.ErrEmailTaken
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("create user: %w", err)
	}
	if err := q.CreatePasswordIdentity(ctx, dbgen.CreatePasswordIdentityParams{
		UserID: userID, Subject: a.Email, PasswordHash: &a.PasswordHash, Now: a.CreatedAt,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("create password identity: %w", err)
	}
	if err := createSession(ctx, q, userID, first); err != nil {
		return uuid.Nil, err
	}
	return userID, tx.Commit(ctx)
}

func (s *AuthStore) PasswordLogin(ctx context.Context, email string) (uuid.UUID, string, error) {
	row, err := s.q.GetPasswordLogin(ctx, email)
	if err != nil {
		return uuid.Nil, "", notFound(err)
	}
	if row.PasswordHash == nil { // ruled out by the auth_identities CHECK
		return uuid.Nil, "", fmt.Errorf("password identity for user %s has no hash", row.ID)
	}
	return row.ID, *row.PasswordHash, nil
}

func (s *AuthStore) CreateSession(ctx context.Context, userID uuid.UUID, sess auth.NewSession) error {
	return createSession(ctx, s.q, userID, sess)
}

func (s *AuthStore) LiveSession(ctx context.Context, tokenHash []byte, now time.Time) (auth.StoredSession, error) {
	row, err := s.q.GetLiveSession(ctx, dbgen.GetLiveSessionParams{TokenHash: tokenHash, Now: now})
	if err != nil {
		return auth.StoredSession{}, notFound(err)
	}
	return auth.StoredSession{ID: row.ID, UserID: row.UserID, LastSeenAt: row.LastSeenAt, ExpiresAt: row.ExpiresAt}, nil
}

func (s *AuthStore) RenewSession(ctx context.Context, id uuid.UUID, lastSeenAt, expiresAt time.Time) error {
	return s.q.RenewSession(ctx, dbgen.RenewSessionParams{ID: id, Now: lastSeenAt, ExpiresAt: expiresAt})
}

func (s *AuthStore) DeleteSession(ctx context.Context, tokenHash []byte) error {
	return s.q.DeleteSession(ctx, tokenHash)
}

func (s *AuthStore) User(ctx context.Context, id uuid.UUID) (auth.StoredUser, error) {
	u, err := s.q.GetUser(ctx, id)
	if err != nil {
		return auth.StoredUser{}, notFound(err)
	}
	methods, err := s.q.ListSignInMethods(ctx, id)
	if err != nil {
		return auth.StoredUser{}, err
	}
	return auth.StoredUser{
		Email:           u.Email,
		EmailVerifiedAt: u.EmailVerifiedAt,
		Plan:            u.Plan,
		FirstDay:        u.FirstDay,
		Timezone:        u.Timezone,
		NextTimezone:    u.NextTimezone,
		SignInMethods:   methods,
	}, nil
}

func createSession(ctx context.Context, q *dbgen.Queries, userID uuid.UUID, s auth.NewSession) error {
	if err := q.CreateSession(ctx, dbgen.CreateSessionParams{
		TokenHash: s.TokenHash, UserID: userID, Now: s.CreatedAt, ExpiresAt: s.ExpiresAt,
	}); err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// notFound turns pgx's "no rows" into the core's ErrNotFound.
func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.ErrNotFound
	}
	return err
}
