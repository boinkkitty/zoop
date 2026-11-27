package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/boinkkitty/zoop/internal/mail"
)

// Deactivate closes userID's account for good: its journal is hidden at once, every device is
// signed out, and the user is emailed to say so. typed must be the account's email, as confirmation.
// If only the email fails, it returns ErrMailNotSent.
func (s *Service) Deactivate(ctx context.Context, userID uuid.UUID, typed Email) error {
	u, err := s.store.User(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return ErrNoSession
	}
	if err != nil {
		return err
	}
	if string(normalizeEmail(typed)) != u.Email {
		return &ValidationError{"Type your account's email to confirm."}
	}
	email, err := s.store.Deactivate(ctx, userID, s.clock.Now(ctx))
	if errors.Is(err, ErrNotFound) {
		return ErrNoSession
	}
	if err != nil {
		return err
	}
	err = s.mailer.Send(ctx, mail.Message{
		To:      email,
		Subject: "Your Zoop account was deactivated",
		Text: "The Zoop account for " + email + " was deactivated, and its journal is hidden.\n\n" +
			"It will be erased within a year, and gone from backups 90 days after that.\n",
	})
	if err != nil {
		return fmt.Errorf("%w: deactivation: %v", ErrMailNotSent, err)
	}
	return nil
}
