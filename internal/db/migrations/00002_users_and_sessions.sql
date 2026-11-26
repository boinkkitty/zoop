-- +goose Up
CREATE TABLE users (
    id                  uuid        PRIMARY KEY DEFAULT uuidv7(),
    -- Unique including Deactivated Accounts, so a closed email can't sign up again.
    email               text        NOT NULL UNIQUE CHECK (email = lower(email)),
    email_verified_at   timestamptz,
    plan                text        NOT NULL DEFAULT 'free' CHECK (plan IN ('free', 'premium')),
    first_day           date        NOT NULL,
    timezone            text        NOT NULL,
    next_timezone       text,
    timezone_switch_day date,
    created_at          timestamptz NOT NULL,
    updated_at          timestamptz NOT NULL,
    deleted_at          timestamptz
);

CREATE TABLE auth_identities (
    id            uuid        PRIMARY KEY DEFAULT uuidv7(),
    user_id       uuid        NOT NULL REFERENCES users,
    provider      text        NOT NULL CHECK (provider IN ('google', 'password')),
    subject       text        NOT NULL,
    password_hash text,
    created_at    timestamptz NOT NULL,
    UNIQUE (provider, subject),
    UNIQUE (user_id, provider),
    CHECK ((provider = 'password') = (password_hash IS NOT NULL))
);

CREATE TABLE sessions (
    id           uuid        PRIMARY KEY DEFAULT uuidv7(),
    token_hash   bytea       NOT NULL UNIQUE,
    user_id      uuid        NOT NULL REFERENCES users,
    created_at   timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL,
    expires_at   timestamptz NOT NULL
);
CREATE INDEX sessions_user_id ON sessions (user_id);

-- +goose Down
DROP TABLE sessions;
DROP TABLE auth_identities;
DROP TABLE users;
