-- name: CreateUser :one
INSERT INTO users (email, first_day, timezone, created_at, updated_at)
VALUES (@email, @first_day, @timezone, @now, @now)
ON CONFLICT (email) DO NOTHING
RETURNING id;

-- name: CreatePasswordIdentity :exec
INSERT INTO auth_identities (user_id, provider, subject, password_hash, created_at)
VALUES (@user_id, 'password', @subject, @password_hash, @now);

-- name: GetPasswordLogin :one
SELECT u.id, i.password_hash
FROM users u
JOIN auth_identities i ON i.user_id = u.id AND i.provider = 'password'
WHERE u.email = @email AND u.deleted_at IS NULL;

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

-- name: GetUser :one
SELECT email, email_verified_at, plan, first_day, timezone, next_timezone
FROM users
WHERE id = @id AND deleted_at IS NULL;

-- name: ListSignInMethods :many
SELECT provider FROM auth_identities WHERE user_id = @user_id ORDER BY provider;
