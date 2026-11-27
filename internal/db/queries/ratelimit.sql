-- name: AddRateLimitHit :one
INSERT INTO rate_limits (key, window_start, count, expires_at)
VALUES (@key, @window_start, 1, @expires_at)
ON CONFLICT (key, window_start) DO UPDATE SET count = rate_limits.count + 1
RETURNING count;

-- name: DeleteExpiredRateLimits :execrows
DELETE FROM rate_limits WHERE expires_at <= @now;

-- name: GetRateLimitCount :one
SELECT coalesce(
    (SELECT count FROM rate_limits WHERE key = @key AND window_start = @window_start), 0
)::integer AS count;
