# Research: Go backend building blocks for the journal API

Ticket: ../issues/02-go-backend-building-blocks.md
Date: 2026-11-25. Sources are official docs, specs, OWASP cheat sheets, repo READMEs and the GitHub releases API, all read on that date. Release dates come from `api.github.com/repos/<repo>/releases`.

Context: the current Go is 1.27 (1.27.0 came out 2026-08-19, and 1.27.4 on 2026-11-10). Go supports a major release until two newer ones exist, so 1.26 and 1.27 are supported now [GO1]. The newest PostgreSQL is 19 (2026-11-19). Versions 15–19 are supported, and 14 reached end of life on 2026-11-12 [PG0].

## Recommendations

| Concern | Pick | One-line reason |
|---|---|---|
| HTTP router | stdlib `net/http.ServeMux` (Go 1.22+ patterns) | Method matching, `{wildcards}`, automatic 405s and order-independent precedence are built in, with zero dependencies. A journal API doesn't need more. |
| Postgres access | `pgx/v5` (native interface + `pgxpool`) with `sqlc` (`sql_package: "pgx/v5"`) | Type-safe generated Go from plain SQL. Postgres-specific SQL (tsvector, `ts_headline`, `:*`) stays first-class instead of being fought through an ORM. |
| Migrations | `pressly/goose` v3 (SQL files, embedded with `embed.FS`) | Actively released (v3.30.1, 2026-11-12). Each migration runs in a transaction by default, with a per-file `NO TRANSACTION` opt-out. It has a Postgres locker for concurrent deploys, and sqlc reads its files natively. |
| Google login | `golang.org/x/oauth2` + `github.com/coreos/go-oidc/v3` with state, nonce and PKCE S256 | x/oauth2 ships PKCE helpers. go-oidc does discovery, JWKS and ID-token verification. RFC 9700 recommends PKCE even for confidential clients. |
| Password hashing | Argon2id from `golang.org/x/crypto/argon2` at m=19456 KiB (19 MiB), t=2, p=1, 16-byte salt, 32-byte key, stored as a PHC string | This is OWASP's first choice and minimum config. bcrypt is "legacy only" and truncates or rejects input over 72 bytes. |
| Full-text search | `STORED` generated `tsvector` column (explicit `'english'` config) + GIN index. Build the prefix query yourself as `to_tsquery('english', 'tok1:* & tok2:*')` from sanitised tokens. Run `ts_headline` only on the final LIMITed page. | `websearch_to_tsquery` never errors but ignores `:*`, so it can't do search-as-you-type. `ts_headline` re-parses the original text and is slow. GIN is the preferred FTS index. |

---

## 1. HTTP router: stdlib `net/http` ServeMux

What Go 1.22+ gives you [GO2][GO3]:
- Patterns like `"GET /entries/{id}"` and `"/files/{path...}"`. Read wildcards with `r.PathValue("id")`. `GET` also matches `HEAD`.
- A request whose path matches but whose method doesn't gets **405 Method Not Allowed** with an `Allow` header, automatically.
- Precedence is "most specific wins" and doesn't depend on registration order. If two patterns overlap and neither is more specific (e.g. `/posts/{id}` and `/{resource}/latest`), **registration panics**. You find out at startup, not in production.
- `{$}` anchors the end of the path: `"/{$}"` matches only `/`, while `"/"` matches everything.
- `GODEBUG=httpmuxgo121=1` restores the old behaviour. Don't set it.

What it lacks: route groups, sub-router mounting and a middleware stack. Middleware is just `func(http.Handler) http.Handler`, and you wrap the mux or individual handlers. For a dozen journal endpoints that's a few lines.

Alternative: **chi v5** is healthy. v5.5.0 shipped 2026-10-14, the last commit was 2026-11-07, and it has 23k stars [CHI1][API]. It is "100% compatible with net/http", has no external dependencies, and adds `Route`/`Group`/`Mount` plus a middleware package (RequestID, Logger, Recoverer, Timeout) [CHI2]. Its recent releases bumped the minimum Go to 1.24, use `request.Pattern`, and implement `PathValue` [CHI1]. Handlers written for the stdlib mux therefore move to chi unchanged if you later want groups. That makes stdlib-first a low-regret choice.

**Recommendation:** use the stdlib ServeMux. Only add chi if the route tree grows enough that per-group middleware (e.g. an "authenticated" group) gets awkward.

## 2. Postgres access: pgx v5 + sqlc

- **pgx** (`github.com/jackc/pgx/v5`): v5.11.0 came out 2026-09-07, the last commit was 2026-11-22, and it has 14.5k stars [API]. Its README says the native interface is faster and exposes Postgres-specific features. It **recommends the native pgx interface when the app targets only PostgreSQL and no other library needs `database/sql`** [PGX1]. Use `pgxpool` for the concurrency-safe pool. pgx supports the Go and PostgreSQL versions their own teams still support, which today means Go 1.25+ and PostgreSQL 14+. It also tests against PG 14–18 [PGX1].
- **sqlc**: you write SQL queries and it generates "fully type-safe idiomatic Go code" [SQLC1]. Set `sql_package: "pgx/v5"` in `sqlc.yaml`; pgx/v5 is supported from sqlc v1.18.0 [SQLC2]. The latest release is v1.31.1 (2026-04-22) and the last commit was 2026-10-23, so releases are slower than pgx but the project is maintained [API].
- **ORM alternative:** GORM is active (v1.31.2, 2026-06-25) [API]. It still doesn't fit here, because the search feature is raw Postgres SQL (`@@`, `to_tsquery`, `ts_headline`, `ts_rank`). With sqlc those are just queries in a `.sql` file, checked against the schema at generate time.

Pitfalls:
- sqlc **does not run migrations**. It parses your migration files to learn the schema and ignores down migrations [SQLC3].
- sqlc reads migration files in **lexicographic** order. goose's default timestamp names (`20260506082420_x.sql`) are safe. With `goose -s` sequential names, zero-pad them [SQLC3][GOOSE1].

## 3. Migrations: goose v3

- Activity: v3.30.1 (2026-11-12) and v3.30.0 (2026-11-10). The last commit was 2026-11-25 and it has 11.8k stars [API].
- It works as a CLI and as a library. Migrations can be SQL files or Go functions, and it supports **embedded migrations** via `goose.SetBaseFS(embed.FS)` and out-of-order migrations [GOOSE1].
- "By default, all migrations are run within a transaction". Add `-- +goose NO TRANSACTION` to a file for statements that can't run in one, e.g. `CREATE INDEX CONCURRENTLY` [GOOSE1].
- Wrap PL/pgSQL bodies with `-- +goose StatementBegin` / `StatementEnd`. Every statement must end with `;` [GOOSE1].
- A recent addition is a `Locker` interface with a **Postgres table-based lock**, which keeps two app instances from migrating at the same time [GOOSE2].
- sqlc natively understands goose's `-- +goose Up/Down` annotations [SQLC3].

Alternatives:
- **golang-migrate** is active (v4.20.1, 2026-09-09; 19k stars) and has pgx v5 and `io/fs` sources [API][MIG1]. But a failed migration leaves a **"dirty"** flag that you must fix by hand and then `force` [MIG2]. Its library API is also oriented around `database/sql`.
- **Atlas** (v1.4.0, 2026-11-17) is declarative, schema-as-code tooling [API]. It's more machinery than a solo journal app needs.

**Recommendation:** goose, with SQL-only migrations embedded in the binary and run at startup (or as a deploy step) through the Provider API.

## 4. Google OAuth 2.0 / OIDC login

Libraries:
- `golang.org/x/oauth2` (last commit 2026-11-06 [API]). It includes `GenerateVerifier()` ("32 octets of randomness … per RFC 7636"; make a fresh one per authorization), `S256ChallengeOption(verifier)` for `AuthCodeURL`, and `VerifierOption(verifier)` for `Exchange` [OA1].
- `github.com/coreos/go-oidc/v3` (v3.21.0, 2026-09-01) [API]. `oidc.NewProvider(ctx, "https://accounts.google.com")` handles discovery, and `provider.Verifier(&oidc.Config{ClientID: ...})` returns an ID-token verifier. Read the token with `oauth2Token.Extra("id_token")` and pass it to `Verify`, then call `idToken.Claims(&struct{Email; EmailVerified})` [OIDC1].

The flow, with what each source requires:
1. **Login start.** Generate `state` (≥30 random chars per Google [GG1]; use 32 bytes from `crypto/rand`), `nonce`, and a PKCE `verifier`. Put all three in a short-lived, `HttpOnly`, `Secure`, `SameSite=Lax` cookie, or in a server-side pre-auth session. Then redirect to `cfg.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier))`.
2. **Callback.** Check that `state` matches the cookie. Google says "you must confirm that the state received from Google matches the session token you created" [GG1]. Exchange with `cfg.Exchange(ctx, code, oauth2.VerifierOption(verifier))`, then verify the ID token.
3. **Check the nonce yourself.** go-oidc's `Verify` "does NOT do nonce validation, which is the caller's responsibility" [OIDC1]. Compare `idToken.Nonce` to the cookie value.
4. **Identify the user by `sub`, never by `email`.** `sub` is "unique among all Google Accounts and never reused … Use sub … as the unique-identifier key for the user". Email "may not be unique … and could change over time" [GG1]. Store `(provider='google', subject=sub)`.
5. If you auto-link a Google login to an existing email+password account, require `email_verified == true` [GG1]. Only check `hd` if you restrict access to a Workspace domain. The request-side `hd` parameter is only a UI hint, so verify the claim [GG1].

Why PKCE for a server-side (confidential) client: RFC 9700, the OAuth 2.0 Security BCP, says "For confidential clients, the use of PKCE is RECOMMENDED". It protects against authorization-code injection, and clients should use S256 [RFC9700]. Google's discovery document lists `code_challenge_methods_supported` [GG1]. Keep `state` anyway, as Google's docs require it.

## 5. Password hashing: Argon2id

OWASP Password Storage Cheat Sheet [OW1]:
- "Use Argon2id with a minimum configuration of 19 MiB of memory, an iteration count of 2, and 1 degree of parallelism."
- Equivalent minimums trade CPU against RAM: m=47104 (46 MiB) t=1 p=1; **m=19456 (19 MiB) t=2 p=1**; m=12288 t=3; m=9216 t=4; m=7168 t=5 (all p=1).
- scrypt is the fallback: N=2^17, r=8, p=1. **bcrypt "should only be used … in legacy systems where Argon2 and scrypt are not available"**, with cost ≥10 and a 72-byte input limit. PBKDF2-HMAC-SHA256 with ≥600,000 iterations is only for when FIPS-140 is required.
- Hashing should take under about 1 s. Benchmark on the target host, re-hash with new parameters on the next successful login, and treat a pepper as optional defence in depth, stored outside the DB.

Go implementation, `golang.org/x/crypto/argon2` [AR1]:
- `argon2.IDKey(password, salt, time, memory /*KiB*/, threads, keyLen)`.
- The package docs quote the RFC 9106 parameters: t=1, 2 GiB, p=4, or t=3, 64 MiB, p=4. Those are stronger than the OWASP floor, and memory is multiplied by concurrent logins. On a small VPS, start with the OWASP setting (t=2, m=19456, p=1) and raise it if benchmarks allow.
- The package returns raw bytes only. You generate the salt (16 bytes from `crypto/rand`) and encode a PHC string yourself, e.g. `$argon2id$v=19$m=19456,t=2,p=1$<b64salt>$<b64hash>`, so parameters can be upgraded later. Compare with `crypto/subtle.ConstantTimeCompare`.
- Cap the password length (e.g. 1024 bytes) and limit concurrent hash operations (a semaphore) so 19 MiB × many parallel login attempts can't exhaust memory. OWASP warns about high work factors enabling DoS [OW1].

If you used bcrypt instead: `bcrypt.GenerateFromPassword` returns `ErrPasswordTooLong` for >72 bytes, and DefaultCost is 10 [BC1].

## 6. Postgres full-text search: prefix-as-you-type + highlighted snippets

Schema (PG docs pattern [PG1]):
```sql
ALTER TABLE entries ADD COLUMN search tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(body,  '')), 'B')
  ) STORED;
CREATE INDEX entries_search_idx ON entries USING GIN (search);
```
- **Write `STORED` explicitly.** Since PG 18 a generated column is **virtual by default** [PG2], and the docs' FTS pattern is a stored column [PG1].
- Use the **two-argument** `to_tsvector('english', …)`. Only functions with an explicit config can appear in index or generated expressions, because the result must not depend on `default_text_search_config` [PG1]. Generated expressions must be immutable [PG2].
- `coalesce` keeps one NULL field from nulling the whole vector [PG1].
- A separate column is faster than an expression index, because matches don't re-run `to_tsvector` [PG1]. **GIN is the preferred FTS index type**; GiST is lossy [PG3]. GIN supports "partial match", which is what makes `:*` prefix queries indexable [PG4].

Query construction:
- `to_tsquery` supports prefix matching with `word:*` [PG5][PG6]. But it is strict: bad input raises a syntax error [PG5].
- `websearch_to_tsquery` "will never raise syntax errors", so it is safe for raw input. But like `plainto_tsquery` it **does not recognise prefix-match labels** [PG5], so it can't do search-as-you-type.
- So build the query in Go:
  1. Split the input on anything that isn't a Unicode letter or digit. That strips every tsquery operator (`& | ! <-> : * ( ) '`), so the result can't be a syntax error.
  2. Drop empty tokens and cap the count (e.g. 8) and the length of each.
  3. Join them as `t1:* & t2:* …`, or put `:*` only on the last, still-being-typed token.
  4. Pass the string as a bind parameter to `to_tsquery('english', $1)`.
  5. Skip the query when the result is empty or under 2 characters. A single-letter prefix matches nearly everything.

```sql
-- name: SearchEntries :many
SELECT id, created_at,
       ts_headline('english', body, q,
         E'StartSel=, StopSel=, MaxFragments=2, MaxWords=20, MinWords=5') AS snippet
FROM (
  SELECT e.id, e.created_at, e.body, q
  FROM entries e, to_tsquery('english', sqlc.arg(query)) q
  WHERE e.user_id = sqlc.arg(user_id) AND e.search @@ q
  ORDER BY ts_rank(e.search, q) DESC, e.created_at DESC
  LIMIT 20
) page;
```
(StartSel/StopSel are private-use sentinel characters U+E000/U+E001. The Go side HTML-escapes the snippet, then swaps them for `<mark>`. See pitfall 4.)

Pitfalls (with sources):
1. **Prefixes are stemmed too.** `to_tsvector('postgraduate') @@ to_tsquery('postgres:*')` is true because `postgres` stems to `postgr` [PG6]. Expect some over-matching on partial words. If that bothers users, add a second `'simple'`-config tsvector for the prefix term.
2. **A stop-word-only query (e.g. "the") becomes an empty tsquery and matches nothing** (stop words are dropped [PG7]). Handle the empty case in the app instead of showing "no results".
3. **`ts_headline` is slow.** It "uses the original document, not a tsvector summary … should be used with care" [PG5]. Only run it on the LIMITed page (the subquery above), never in the `WHERE` or on the full match set.
4. **`ts_headline` output is not XSS-safe.** It strips only some simple tags and "this does not provide an effective defense against … XSS". Sanitise it, or strip HTML from the input [PG5]. The default `StartSel`/`StopSel` is `<b>`/`</b>` [PG5]. Safer: use sentinel characters, HTML-escape the whole snippet, then replace the sentinels with `<mark>`/`</mark>`.
5. **Ranking cost.** `ts_rank` has to read each match's tsvector, which "can be I/O bound and therefore slow" [PG5]. Journals are per-user and small, so filtering by `user_id` keeps this cheap. A GIN index can't combine `user_id =` with `@@` unless you add `btree_gin`, but a btree on `user_id` plus the GIN is fine at this scale.
6. **Limits.** A tsvector must be < 1 MB, with ≤256 positions per lexeme and positions ≤16,383 [PG8]. Very long entries lose positions past that. Words still match, but phrase and proximity get less accurate.
7. `ts_headline` highlights every word that appears in the query, "even if those word positions do not match the query's restrictions" [PG5]. That's fine for an AND query.

---

## Sources

- [GO1] Go release history: https://go.dev/doc/devel/release
- [GO2] Go blog, "Routing Enhancements for Go 1.22": https://go.dev/blog/routing-enhancements
- [GO3] net/http ServeMux docs: https://pkg.go.dev/net/http#ServeMux
- [CHI1] chi releases: https://github.com/go-chi/chi/releases
- [CHI2] chi README: https://github.com/go-chi/chi
- [PGX1] pgx README: https://github.com/jackc/pgx
- [SQLC1] sqlc docs: https://docs.sqlc.dev/en/latest/
- [SQLC2] sqlc, Using Go and pgx: https://docs.sqlc.dev/en/latest/guides/using-go-and-pgx.html
- [SQLC3] sqlc, Modifying the database schema: https://docs.sqlc.dev/en/latest/howto/ddl.html
- [GOOSE1] goose README: https://github.com/pressly/goose
- [GOOSE2] goose releases: https://github.com/pressly/goose/releases
- [MIG1] golang-migrate README: https://github.com/golang-migrate/migrate
- [MIG2] golang-migrate FAQ (dirty state): https://github.com/golang-migrate/migrate/blob/master/FAQ.md
- [OA1] golang.org/x/oauth2: https://pkg.go.dev/golang.org/x/oauth2
- [OIDC1] go-oidc v3 package docs: https://pkg.go.dev/github.com/coreos/go-oidc/v3/oidc (repo: https://github.com/coreos/go-oidc)
- [GG1] Google, OpenID Connect: https://developers.google.com/identity/openid-connect/openid-connect
- [RFC9700] OAuth 2.0 Security Best Current Practice: https://www.rfc-editor.org/rfc/rfc9700.html
- [OW1] OWASP Password Storage Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
- [AR1] golang.org/x/crypto/argon2: https://pkg.go.dev/golang.org/x/crypto/argon2
- [BC1] golang.org/x/crypto/bcrypt: https://pkg.go.dev/golang.org/x/crypto/bcrypt
- [PG0] PostgreSQL versioning policy: https://www.postgresql.org/support/versioning/
- [PG1] PG docs 12.2, Tables and Indexes: https://www.postgresql.org/docs/current/textsearch-tables.html
- [PG2] PG docs 5.4, Generated Columns: https://www.postgresql.org/docs/current/ddl-generated-columns.html
- [PG3] PG docs 12.9, Preferred Index Types: https://www.postgresql.org/docs/current/textsearch-indexes.html
- [PG4] PG docs, GIN (partial match): https://www.postgresql.org/docs/current/gin.html
- [PG5] PG docs 12.3, Controlling Text Search: https://www.postgresql.org/docs/current/textsearch-controls.html
- [PG6] PG docs 8.11, Text Search Types: https://www.postgresql.org/docs/current/datatype-textsearch.html
- [PG7] PG docs 12.6, Dictionaries (stop words): https://www.postgresql.org/docs/current/textsearch-dictionaries.html
- [PG8] PG docs 12.11, Limitations: https://www.postgresql.org/docs/current/textsearch-limitations.html
- [API] GitHub REST API, `GET /repos/{owner}/{repo}/releases` and `/commits`, for go-chi/chi, jackc/pgx, sqlc-dev/sqlc, pressly/goose, golang-migrate/migrate, ariga/atlas, coreos/go-oidc, golang/oauth2, go-gorm/gorm (queried 2026-11-25)
