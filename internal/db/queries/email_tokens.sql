-- name: CreateEmailToken :exec
INSERT INTO email_tokens (token_hash, user_id, purpose, created_at, expires_at)
VALUES (@token_hash, @user_id, @purpose, @now, @expires_at);

-- name: UseEmailToken :one
-- Deleting it is what makes it work once. A Deactivated Account's tokens never work.
DELETE FROM email_tokens t
USING users u
WHERE t.token_hash = @token_hash AND t.purpose = @purpose AND t.expires_at > @now
  AND u.id = t.user_id AND u.deleted_at IS NULL
RETURNING t.user_id;

-- name: MarkEmailVerified :exec
UPDATE users SET email_verified_at = coalesce(email_verified_at, @now::timestamptz), updated_at = @now
WHERE id = @id;

-- name: SetPassword :exec
-- Adds a password identity, or replaces the hash on the one there is.
INSERT INTO auth_identities (user_id, provider, subject, password_hash, created_at)
SELECT u.id, 'password', u.email, @password_hash, @now FROM users u WHERE u.id = @user_id
ON CONFLICT (user_id, provider) DO UPDATE SET password_hash = excluded.password_hash;

-- name: DeleteExpiredEmailTokens :execrows
DELETE FROM email_tokens WHERE expires_at <= @now;

-- name: DeleteUserEmailTokens :exec
DELETE FROM email_tokens WHERE user_id = @user_id;
