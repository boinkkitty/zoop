package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/boinkkitty/zoop/internal/auth"
	"github.com/boinkkitty/zoop/internal/db/dbgen"
)

func (s *AuthStore) CreateEmailToken(ctx context.Context, userID uuid.UUID, t auth.NewEmailToken) error {
	if err := s.q.CreateEmailToken(ctx, dbgen.CreateEmailTokenParams{
		TokenHash: t.TokenHash, UserID: userID, Purpose: string(t.Purpose), Now: t.CreatedAt, ExpiresAt: t.ExpiresAt,
	}); err != nil {
		return fmt.Errorf("create email token: %w", err)
	}
	return nil
}

func (s *AuthStore) VerifyEmail(ctx context.Context, tokenHash []byte, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := s.q.WithTx(tx)

	userID, err := q.UseEmailToken(ctx, dbgen.UseEmailTokenParams{
		TokenHash: tokenHash, Purpose: string(auth.PurposeVerify), Now: now,
	})
	if err != nil {
		return notFound(err)
	}
	if err := q.MarkEmailVerified(ctx, dbgen.MarkEmailVerifiedParams{ID: userID, Now: now}); err != nil {
		return fmt.Errorf("mark email verified: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *AuthStore) ResetPassword(ctx context.Context, tokenHash []byte, passwordHash string, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := s.q.WithTx(tx)

	userID, err := q.UseEmailToken(ctx, dbgen.UseEmailTokenParams{
		TokenHash: tokenHash, Purpose: string(auth.PurposeReset), Now: now,
	})
	if err != nil {
		return notFound(err)
	}
	if err := q.SetPassword(ctx, dbgen.SetPasswordParams{UserID: userID, PasswordHash: &passwordHash, Now: now}); err != nil {
		return fmt.Errorf("set password: %w", err)
	}
	if err := q.MarkEmailVerified(ctx, dbgen.MarkEmailVerifiedParams{ID: userID, Now: now}); err != nil {
		return fmt.Errorf("mark email verified: %w", err)
	}
	if err := q.DeleteUserSessions(ctx, userID); err != nil {
		return fmt.Errorf("delete sessions: %w", err)
	}
	return tx.Commit(ctx)
}
