# Data model

Type: grilling
Status: open
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
