-- One-use tokens sent in emails: verify an email (#33), reset a password (#34). Up only; safe to apply twice.
-- Only the token's sha256 is stored, so a leaked table can't be replayed.
CREATE TABLE IF NOT EXISTS email_tokens (
    id         uuid        PRIMARY KEY DEFAULT uuidv7(),
    token_hash bytea       NOT NULL UNIQUE,
    user_id    uuid        NOT NULL REFERENCES users,
    purpose    text        NOT NULL CHECK (purpose IN ('verify', 'reset')),
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS email_tokens_user_id ON email_tokens (user_id);
