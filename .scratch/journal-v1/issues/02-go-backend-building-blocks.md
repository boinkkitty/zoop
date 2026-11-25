# Go backend building blocks

Type: research
Status: resolved
Blocked by:

## Question

What is the current (late 2026) well-supported choice for each of these in a Go + PostgreSQL web API?
- HTTP router (stdlib `net/http` 1.22+ patterns vs chi vs others)
- Postgres access (pgx + sqlc vs an ORM)
- Migrations tool
- Google OAuth 2.0 / OIDC login (golang.org/x/oauth2 + go-oidc, PKCE, state handling)
- Password hashing (argon2id vs bcrypt, and the recommended parameters)
- Postgres full-text search with prefix matching and snippet highlighting (`to_tsquery` with `:*`, `ts_headline`, GIN index), including any pitfalls

Recommend one option for each, with the reason.

Research: ../research/go-backend-building-blocks.md

## Answer

- **Router**: **chi v5** (user's decision, 2026-11-25, overriding the research pick of stdlib `ServeMux`). chi works with plain `net/http` handlers, so `http.CrossOriginProtection` (Go 1.25) still wraps it for CSRF protection on cookie sessions.
- **DB access**: pgx v5 (pgxpool) + sqlc (`sql_package: pgx/v5`). Plain SQL keeps the FTS queries first-class.
- **Migrations**: goose v3, embedded via `embed.FS`. sqlc reads goose files as its schema.
- **Google login**: `x/oauth2` + `coreos/go-oidc/v3`. Use the auth-code flow with PKCE S256, a single-use `state` in a short-lived cookie, and a `nonce` check. Key on `sub`, never email. Link to an existing account only when `email_verified` is true.
- **Passwords**: Argon2id (`x/crypto/argon2.IDKey`) with m=19 MiB, t=2, p=1 and a 16-byte salt (OWASP). Store a PHC string, compare in constant time, cap input length and limit concurrent hashing.
- **Search**: a `tsvector` generated column marked **STORED**, a GIN index, and a query built in Go: tokenise, quote each token, join with `&`, add `:*` to the last token, then pass to `to_tsquery`. Run `ts_headline` only on the LIMITed page of results.

Pitfalls for the Data model ticket:
- PG 18+ defaults generated columns to VIRTUAL, so write `STORED` explicitly.
- Pick one text-search config (`english` stems words; `simple` gives literal prefixes) and use it everywhere. Changing it later means a rewrite and reindex.
- Wrap source columns in `coalesce()` and filter search by `user_id`.
- A `tsvector` must be under 1 MB, so cap Page length.
- `ts_headline` output isn't XSS-safe. Use non-HTML markers and render them as React text.
- Store Google `sub` as a unique identity column. `password_hash` is nullable.

Full findings and sources: [research/go-backend-building-blocks.md](../research/go-backend-building-blocks.md)
