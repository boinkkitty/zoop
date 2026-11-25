# Data model

Type: grilling
Status: resolved
Assignee: boinkkitty
Blocked by: 02

## Question

What are the Postgres tables and constraints for v1? Cover:
- users: plan flag, profile timezone, and how a timezone change "from the next Day" is stored
- linked auth identities: Google and password
- sessions: hashed opaque token, sliding expiry
- email verification and reset tokens
- Notebooks: fixed system rows vs per-user rows, given that more will come later
- Pages: unique per user + Notebook + Day, TipTap JSON in a JSONB column plus `schemaVersion`, a derived plain-text column, and `updated_at` (per [Formatting on Pages](09-page-formatting.md))
- the search index column, built from the plain-text column
- how revisions could be added later without migrating existing Pages (Formatting on Pages suggests whole JSON copies in their own table)

## Context (2026-11-25)

- **Page column type: JSONB**, decided by the user after [Formatting on Pages](09-page-formatting.md). Postgres can look inside it, so a TipTap schema migration can be a single SQL `UPDATE`. The write cost is negligible at Page size. sqlc and pgx return it as `json.RawMessage`. There is **no compression in the app** (no Go-side gzip into a `bytea` column), because Postgres already compresses values over ~2 KB automatically.
- **To consider, measure in production first:** set the column to LZ4 compression (`doc jsonb COMPRESSION lz4`) instead of Postgres's default. Not decided for v1, so add it only if production numbers show a benefit.

## Answer

Settled with the user on 2026-11-25 over three grilling rounds. Builds on the pitfalls in [Go backend building blocks](02-go-backend-building-blocks.md) and the storage decision in [Formatting on Pages](09-page-formatting.md).

```
users           id uuidv7 PK, email (lowercased) UNIQUE, email_verified_at,
                plan text CHECK('free','premium') DEFAULT 'free',
                first_day date, timezone, next_timezone NULL, timezone_switch_day date NULL,
                created_at, updated_at, deleted_at NULL

auth_identities id uuidv7 PK, user_id → users, provider CHECK('google','password'),
                subject, password_hash NULL, created_at
                UNIQUE(provider, subject), UNIQUE(user_id, provider),
                CHECK((provider='password') = (password_hash IS NOT NULL))

sessions        id uuidv7 PK, token_hash bytea UNIQUE, user_id → users,
                created_at, last_seen_at, expires_at

email_tokens    token_hash bytea PK, user_id → users, purpose CHECK('verify','reset'),
                expires_at, used_at NULL

notebooks       id smallint PK, slug UNIQUE, name, position
                seeded: 1 my-journal, 2 i-learned

pages           id uuidv7 PK, user_id → users, notebook_id → notebooks, day date,
                doc jsonb, schema_version smallint, plain_text text,
                search tsvector GENERATED ALWAYS AS (to_tsvector('simple', coalesce(plain_text,''))) STORED,
                version int, created_at, updated_at
                UNIQUE(user_id, notebook_id, day)
                GIN(search) · partial index (user_id, day) WHERE plain_text <> ''
```

- **Keys**: UUIDv7 (Postgres 18 `uuidv7()`) for users, identities, sessions and Pages. Notebooks use a small fixed id.
- **Sign-in identities** live in their own table, chosen so more providers can be added later. A Google identity's `subject` is Google's `sub`, never the email. A password identity's `subject` is the user's id and its row holds the Argon2id hash. Email login looks up `users.email`, then the user's password identity. Linking an account adds an identity row.
- **Users**: `plan` is a checked text column. `first_day` is stored at signup in the browser's timezone. A timezone change is stored as the pending pair `next_timezone` + `timezone_switch_day`, which takes over at the next Day. "Today" never moves backwards. The exact rule for "today" belongs to [API contract and autosave](04-api-contract-and-autosave.md).
- **Notebooks**: one global table seeded with two rows, and code refers to them by `slug`. User-created Notebooks are out of scope.
- **Pages**: a surrogate `id` gives future revisions one key to point at. `version` goes up by one on every save, for the API ticket's conflict handling. Emptied Pages stay as rows with empty content, and a [Written Day](../../../GLOSSARY.md) is one with non-empty `plain_text`. The Title (first line of the *My Journal* Page) and hover glimpses (~80 characters) are **derived when queried**, not stored. Caps: request bodies up to 512 KB and `plain_text` up to 100,000 characters, checked on save.
- **Search**: the `simple` text-search setting is used everywhere, giving literal word-start matching as you type (how Notes, Notion and Obsidian behave). The tsvector column is generated and explicitly `STORED`, with a GIN index, and every query filters by `user_id`.
- **Sessions**: the token is stored hashed. The 30-day expiry slides, with `expires_at` pushed forward at most once per hour. Resetting a password deletes all of that user's sessions.
- **Email tokens**: one table for both purposes. Verification links last 24 hours and reset links last 1 hour, and each can be used once.
- **Deletes are soft**: `users.deleted_at` only, and foreign keys use `NO ACTION` with no cascades. Soft-deleting a user hard-deletes their sessions and email tokens. The email index covers deleted accounts too, so a deleted account's email can't sign up again. Account re-activation and the purge are future work. The agreed policy for that later purge is a hard delete after 1 year.
- **Cleanup**: a ticker inside the app clears expired sessions and tokens, with no separate cron.
- **Revisions later**: a `page_revisions(id, page_id → pages.id, doc, schema_version, created_at)` table, written when autosave settles. Nothing is built in v1, and existing Pages need no migration.
- **Deferred**: LZ4 compression on `doc`, decided only after production numbers show a benefit.

- **From [Operational basics](10-operational-basics.md) (2026-11-25):** v1 needs a rate-limit counter table, a fixed window keyed by (key, window start) with a count and an expiry. Expired rows are deleted as new rows are written, or by the existing cleanup ticker.
- **From [Search results UI](14-search-results-ui.md) (2026-11-25):** the Day's date is searchable as text. The `search` column becomes `to_tsvector('simple', zoop_day_words(day) || ' ' || coalesce(plain_text,''))`, where `zoop_day_words(date)` is a SQL function declared `IMMUTABLE` that returns `to_char(d, 'FMDay FMDD FMMonth YYYY')`. That is always English, because there is no `TM` prefix.
