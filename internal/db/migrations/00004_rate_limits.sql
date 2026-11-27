-- Fixed-window rate-limit counters (#35). Up only; safe to apply twice.
CREATE TABLE IF NOT EXISTS rate_limits (
    key          text        NOT NULL,
    window_start timestamptz NOT NULL,
    count        integer     NOT NULL,
    expires_at   timestamptz NOT NULL,
    PRIMARY KEY (key, window_start)
);
CREATE INDEX IF NOT EXISTS rate_limits_expires_at ON rate_limits (expires_at);
