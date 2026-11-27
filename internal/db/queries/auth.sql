-- name: CreateUser :one
INSERT INTO users (email, first_day, timezone, created_at, updated_at)
VALUES (@email, @first_day, @timezone, @now, @now)
ON CONFLICT (email) DO NOTHING
RETURNING id;

-- name: CreatePasswordIdentity :exec
INSERT INTO auth_identities (user_id, provider, subject, password_hash, created_at)
VALUES (@user_id, 'password', @subject, @password_hash, @now);

-- name: CreateVerifiedUser :one
INSERT INTO users (email, email_verified_at, first_day, timezone, created_at, updated_at)
VALUES (@email, @now::timestamptz, @first_day, @timezone, @now, @now)
ON CONFLICT (email) DO NOTHING
RETURNING id;

-- name: CreateGoogleIdentity :exec
INSERT INTO auth_identities (user_id, provider, subject, created_at)
VALUES (@user_id, 'google', @subject, @now);

-- name: GetGoogleLogin :one
-- Deactivated Accounts too, so the sign-in can say the account is gone.
SELECT u.id, u.deleted_at
FROM users u
JOIN auth_identities i ON i.user_id = u.id AND i.provider = 'google'
WHERE i.subject = @subject;

-- name: GetUserByEmail :one
-- Deactivated Accounts too: their emails stay taken.
SELECT id, email_verified_at, deleted_at FROM users WHERE email = @email;

-- name: VerifyUnverifiedEmail :one
-- Only matches while the email is still unverified, so a takeover never removes a verified password.
UPDATE users SET email_verified_at = @now::timestamptz, updated_at = @now
WHERE id = @id AND email_verified_at IS NULL
RETURNING id;

-- name: DeletePasswordIdentity :exec
DELETE FROM auth_identities WHERE user_id = @user_id AND provider = 'password';

-- name: DeleteUserSessions :exec
DELETE FROM sessions WHERE user_id = @user_id;

-- name: GetPasswordLogin :one
-- Deactivated Accounts too, so the right password can be told the account is gone.
SELECT u.id, i.password_hash, u.deleted_at
FROM users u
JOIN auth_identities i ON i.user_id = u.id AND i.provider = 'password'
WHERE u.email = @email;

-- name: CreateSession :exec
INSERT INTO sessions (token_hash, user_id, created_at, last_seen_at, expires_at)
VALUES (@token_hash, @user_id, @now, @now, @expires_at);

-- name: GetLiveSession :one
SELECT s.id, s.user_id, s.last_seen_at, s.expires_at
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token_hash = @token_hash AND s.expires_at > @now AND u.deleted_at IS NULL;

-- name: RenewSession :exec
UPDATE sessions SET last_seen_at = @now, expires_at = @expires_at WHERE id = @id;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = @token_hash;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions WHERE expires_at <= @now;

-- name: GetUser :one
SELECT email, email_verified_at, plan, first_day, timezone, next_timezone, timezone_switch_day
FROM users
WHERE id = @id AND deleted_at IS NULL;

-- name: ListSignInMethods :many
SELECT provider FROM auth_identities WHERE user_id = @user_id ORDER BY provider;

-- name: SetTimezone :exec
UPDATE users
SET timezone = @timezone, next_timezone = @next_timezone, timezone_switch_day = @timezone_switch_day, updated_at = @now
WHERE id = @id AND deleted_at IS NULL;

-- name: DeactivateUser :one
UPDATE users SET deleted_at = @now::timestamptz, updated_at = @now
WHERE id = @id AND deleted_at IS NULL
RETURNING email;
