package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"

	"github.com/boinkkitty/zoop/internal/auth/password"
	"github.com/boinkkitty/zoop/internal/mail"
	"github.com/boinkkitty/zoop/internal/ratelimit"
)

const (
	// VerifyLifetime is how long a verify link works.
	VerifyLifetime = 24 * time.Hour
	// ResetLifetime is how long a reset link works.
	ResetLifetime = time.Hour
)

// ErrBadToken means an emailed link's token is unknown, expired or already used.
var ErrBadToken = errors.New("auth: link expired or already used")

// ErrMailNotSent means the account change happened but its email couldn't be sent.
// The caller should report it and carry on: the user can ask for the email again.
var ErrMailNotSent = errors.New("auth: email not sent")

// link is an app page carrying token, for an email.
func (s *Service) link(page string, token Token) string {
	return s.appURL + page + "?" + url.Values{"token": {string(token)}}.Encode()
}

// VerifyEmail marks the email verified for the account a verify link was sent to.
// The token works once, for VerifyLifetime. Whoever holds it needn't be signed in.
func (s *Service) VerifyEmail(ctx context.Context, token Token) error {
	if token == "" {
		return ErrBadToken
	}
	err := s.store.VerifyEmail(ctx, hashToken(token), s.clock.Now(ctx))
	if errors.Is(err, ErrNotFound) {
		return ErrBadToken
	}
	return err
}

// ResendVerify emails a fresh verify link to rawEmail if it's an account's unverified email.
// It answers the same either way, so it doesn't reveal which emails have accounts.
// Requests count toward EmailRequestsPerEmail and EmailRequestsPerIP.
func (s *Service) ResendVerify(ctx context.Context, rawEmail Email, clientIP string) error {
	email := normalizeEmail(rawEmail)
	if err := s.takeEmailRequest(ctx, email, clientIP); err != nil {
		return err
	}
	owner, err := s.store.UserByEmail(ctx, string(email))
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil || owner.EmailVerified {
		return err
	}
	return s.sendVerify(ctx, owner.UserID, string(email))
}

// ForgotPassword emails a reset link to rawEmail if it's a live account's. A Google-only
// account sets its first password through the same link. It answers the same either way,
// so it doesn't reveal which emails have accounts. Requests count like ResendVerify's.
func (s *Service) ForgotPassword(ctx context.Context, rawEmail Email, clientIP string) error {
	email := normalizeEmail(rawEmail)
	if err := s.takeEmailRequest(ctx, email, clientIP); err != nil {
		return err
	}
	owner, err := s.store.UserByEmail(ctx, string(email))
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	token, err := s.emailToken(ctx, owner.UserID, PurposeReset, ResetLifetime)
	if err != nil {
		return err
	}
	err = s.mailer.Send(ctx, mail.Message{
		To:      string(email),
		Subject: "Reset your Zoop password",
		Text: "Choose a new password for " + string(email) + " by opening this link:\n\n" +
			s.link("/reset", token) + "\n\nIt works once, within an hour. If you didn't ask for this, ignore this email.\n",
	})
	if err != nil {
		return fmt.Errorf("%w: reset: %v", ErrMailNotSent, err)
	}
	return nil
}

// ResetPassword sets a new password for the account a reset link was sent to, and signs
// out every one of its devices. The token works once, for ResetLifetime.
func (s *Service) ResetPassword(ctx context.Context, token Token, plain Password) error {
	if err := validateNewPassword(plain); err != nil {
		return err
	}
	if token == "" {
		return ErrBadToken
	}
	hash, err := password.Hash(string(plain))
	if err != nil {
		return err
	}
	err = s.store.ResetPassword(ctx, hashToken(token), hash, s.clock.Now(ctx))
	if errors.Is(err, ErrNotFound) {
		return ErrBadToken
	}
	return err
}

// takeEmailRequest counts a request for an email to email, so nobody can flood an inbox.
func (s *Service) takeEmailRequest(ctx context.Context, email Email, clientIP string) error {
	if err := s.limits.Take(ctx, ratelimit.EmailRequestsPerIP, clientIP); err != nil {
		return err
	}
	return s.limits.Take(ctx, ratelimit.EmailRequestsPerEmail, string(email))
}

// emailToken makes and stores a new token for userID, returning it to put in a link.
func (s *Service) emailToken(ctx context.Context, userID uuid.UUID, p Purpose, lifetime time.Duration) (Token, error) {
	token := Token(randomString())
	now := s.clock.Now(ctx)
	err := s.store.CreateEmailToken(ctx, userID, NewEmailToken{
		TokenHash: hashToken(token), Purpose: p, CreatedAt: now, ExpiresAt: now.Add(lifetime),
	})
	return token, err
}

func (s *Service) sendVerify(ctx context.Context, userID uuid.UUID, email string) error {
	token, err := s.emailToken(ctx, userID, PurposeVerify, VerifyLifetime)
	if err != nil {
		return err
	}
	err = s.mailer.Send(ctx, mail.Message{
		To:      email,
		Subject: "Verify your email for Zoop",
		Text: "Confirm that " + email + " is yours by opening this link:\n\n" +
			s.link("/verify", token) + "\n\nIt works once, within 24 hours.\n",
	})
	if err != nil {
		return fmt.Errorf("%w: verify: %v", ErrMailNotSent, err)
	}
	return nil
}
