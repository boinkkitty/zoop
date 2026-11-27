package auth

import "log/slog"

// redacted is what every sensitive value logs as, so a careless slog call can't leak it.
var redacted = slog.StringValue("[redacted]")

// Email is an account's email address. Logs use the user id instead.
type Email string

func (Email) LogValue() slog.Value { return redacted }

// Token is a secret bearer token, such as a session cookie's value.
type Token string

func (Token) LogValue() slog.Value { return redacted }

// Password is a plaintext password as the visitor typed it.
type Password string

func (Password) LogValue() slog.Value { return redacted }
