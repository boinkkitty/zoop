// Package google is the adapter for signing in with Google over OpenID Connect.
// It implements auth.GoogleProvider with golang.org/x/oauth2 and go-oidc.
package google

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/boinkkitty/zoop/internal/auth"
	"github.com/boinkkitty/zoop/internal/clock"
)

// Issuer is Google's OpenID Connect issuer.
const Issuer = "https://accounts.google.com"

// Config is the OAuth client registered with Google.
type Config struct {
	Issuer       string // Issuer, or a stub's in tests
	ClientID     string
	ClientSecret string
	RedirectURL  string // where Google sends the browser back: .../api/v1/auth/google/callback
}

// Provider implements auth.GoogleProvider. It fetches the issuer's discovery
// document on first use, and again on later uses until that succeeds.
type Provider struct {
	cfg   Config
	clock clock.Clock

	mu       sync.Mutex
	oauth    *oauth2.Config
	verifier *oidc.IDTokenVerifier
}

var _ auth.GoogleProvider = (*Provider)(nil)

func New(cfg Config, c clock.Clock) *Provider {
	return &Provider{cfg: cfg, clock: c}
}

// AuthURL asks for an ID token with the user's email, bound to nonce, with a PKCE S256 challenge.
func (p *Provider) AuthURL(ctx context.Context, state, nonce, verifier string) (string, error) {
	o, _, err := p.discover(ctx)
	if err != nil {
		return "", err
	}
	return o.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)), nil
}

func (p *Provider) Exchange(ctx context.Context, code, verifier, nonce string) (auth.GoogleIdentity, error) {
	o, v, err := p.discover(ctx)
	if err != nil {
		return auth.GoogleIdentity{}, err
	}
	tok, err := o.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return auth.GoogleIdentity{}, fmt.Errorf("exchange code: %w", err)
	}
	raw, ok := tok.Extra("id_token").(string)
	if !ok {
		return auth.GoogleIdentity{}, errors.New("token response has no id_token")
	}
	idt, err := v.Verify(ctx, raw)
	if err != nil {
		return auth.GoogleIdentity{}, fmt.Errorf("verify id token: %w", err)
	}
	if idt.Nonce != nonce {
		return auth.GoogleIdentity{}, errors.New("id token nonce does not match")
	}
	var claims struct {
		Email         string `json:"email"`
		EmailVerified any    `json:"email_verified"` // a bool, though older tokens sent "true"
	}
	if err := idt.Claims(&claims); err != nil {
		return auth.GoogleIdentity{}, fmt.Errorf("read id token claims: %w", err)
	}
	if idt.Subject == "" || claims.Email == "" {
		return auth.GoogleIdentity{}, errors.New("id token has no subject or email")
	}
	return auth.GoogleIdentity{
		Subject:       idt.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified == true || claims.EmailVerified == "true",
	}, nil
}

func (p *Provider) discover(ctx context.Context) (*oauth2.Config, *oidc.IDTokenVerifier, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.oauth != nil {
		return p.oauth, p.verifier, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	op, err := oidc.NewProvider(ctx, p.cfg.Issuer)
	if err != nil {
		return nil, nil, fmt.Errorf("discover %s: %w", p.cfg.Issuer, err)
	}
	p.oauth = p.config(op.Endpoint())
	p.verifier = op.VerifierContext(context.WithoutCancel(ctx), &oidc.Config{ClientID: p.cfg.ClientID, Now: p.clock.Now})
	return p.oauth, p.verifier, nil
}

func (p *Provider) config(ep oauth2.Endpoint) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.cfg.ClientID,
		ClientSecret: p.cfg.ClientSecret,
		RedirectURL:  p.cfg.RedirectURL,
		Endpoint:     ep,
		Scopes:       []string{oidc.ScopeOpenID, "email"},
	}
}
