# Map: Zoop journal app — v1 build spec

Label: wayfinder:map

## Destination

A **v1 build spec for the Zoop web app**: product behaviour, data model, API contract, stack and hosting, decided well enough to hand straight to implementation. Done when nothing is left to decide before someone starts building v1.

## Notes

- **Domain**: a daily paper-journal web app. React frontend, Go backend, PostgreSQL. Vocabulary lives in [GLOSSARY.md](../../GLOSSARY.md). Use Notebook / Page / Day / Grace Window / Title / Island exactly as it defines them.
- **Skills to consult**: `grilling` + `domain-modeling` for grilling tickets; `prototype` for prototype tickets; `research` for research tickets. For web browsing, use the `ego-browser` skill (user's global instruction).
- **Feel**: it must look like a real, physical journal. Nothing that looks like a generic "AI/agentic" SaaS. Updated 2026-11-25 by [Journal page look and feel](issues/05-journal-page-prototype.md): minimal, modern, Apple-style. A white sheet with **no ruled lines**, Notebooks in an inset index column on the sheet's right edge, and a left sidebar (nav, search, then the Island rail) of Islands like the MacBook Dynamic Island.
- **Settled while charting** (from the destination grilling, 2026-11-25):
  - Target is web only and **desktop-first**. It stays usable in a narrow window (the rail folds into a drawer), with no phone-specific design.
  - Login opens **today's** Page in the user's timezone. Days start at signup, with every Day since then listed in the rail. Days with no writing appear faded but can still be opened.
  - Days are **lazy**: there is no daily job. Days are computed from signup → today, and a Page row is written on first keystroke.
  - **Grace Window**: from 00:00 to 02:00 local time, yesterday stays writable and its Island shows a "still open" marker. At 02:00 it locks for Free Users, and an in-flight autosave still lands.
  - **Timezone**: the profile timezone is detected from the browser. A change takes effect from the next Day, so a Day is never skipped or repeated.
  - **Title**: the first line of the *My Journal* Page, shown bold as the user types. Days with no writing show only the date on their Island.
  - **Island hover**: after a 150ms delay, the Island expands vertically, pushing the Islands below it down, with one row per non-empty Notebook (name plus ~80 characters). Clicking it opens that Day.
  - Pages autosave, and there is no save button. Pages are formatted, not plain text: see [Formatting on Pages](issues/09-page-formatting.md).
  - **Search** uses Postgres full-text search with prefix matching as you type. Results are snippets with the match highlighted plus Day and Notebook, and clicking one jumps to the Page. **No Elasticsearch.**
  - **Auth**: Google OAuth or email/password, with email verification and password reset. Accounts with the same email link into one. Sessions are opaque random tokens, stored hashed in Postgres, sent in an HttpOnly cookie, with a 30-day sliding expiry.
  - **Premium**: v1 has a `plan` flag enforced on the server (Free Users can't edit past Pages) and no billing. Revisions are not stored in v1, but the data model must leave room for them.

## Decisions so far

<!-- one line per closed ticket: [title](issues/NN-slug.md): gist -->

- [Email provider for verification and password reset](issues/01-email-provider.md): Resend (free 3k/mo, 100/day cap, official Go SDK) behind a `Mailer` interface. Upgrade to Pro before daily volume nears 100.
- [Journal page look and feel](issues/05-journal-page-prototype.md): Apple-minimal white sheet with no lines and a small date. Notebooks sit in an inset index column on the right. The left sidebar holds placeholder nav, search, then Islands that push down on hover and load lazily. The winner is T4 in prototypes/journal-page-v3.
- [Go backend building blocks](issues/02-go-backend-building-blocks.md): chi router (user's override), pgx + sqlc, goose, x/oauth2 + go-oidc (PKCE), Argon2id, and a STORED tsvector + GIN index queried with `to_tsquery` prefix. The ticket lists pitfalls for the data model.
- [Island rail and hover expansion](issues/06-island-rail-prototype.md): newest first with sticky month headers. A scrubber on the right edge never scrolls: months for this year and last, then one segment per older year with a hover month grid. Pulsing green dot for the Grace Window, Spring motion, slide-over drawer when narrow. Jumps load only the target month, so the API needs paging from any date plus a per-month summary.
- [Formatting on Pages](issues/09-page-formatting.md): bold, italic, underline, highlight, bullet, numbered and checklist lists (3 levels), and quote, applied by keyboard shortcuts only with a ⌘/ shortcut sheet. Stored as TipTap JSON (JSONB plus `schemaVersion`), checked by the server, with a derived plain-text column for search, glimpses and the Title. Markdown is for export only.
- [Hosting and deployment](issues/08-hosting.md): one Go binary serving the SPA and `/api` from one origin, on Railway Hobby in Singapore (~$5–8/mo) with Railway Postgres. Backups: point-in-time restore on from creation (~4 weeks), daily volume backups, and a nightly encrypted `pg_dump` to R2. Local and production only. GitHub Actions deploys on push to `main` with `goose up` first. The domain is bought at launch. The fallback is Fly.io plus Neon.
- [Data model](issues/03-data-model.md): six tables (users, auth_identities, sessions, email_tokens, a global seeded notebooks table, pages), UUIDv7 keys, and a `simple` STORED tsvector. Pages carry `version` for conflicts, and emptied Pages stay as rows. Title and glimpses are derived when queried. Users are soft-deleted with no cascades.
- [Frontend stack and editor](issues/07-frontend-stack.md): Vite + React + TS SPA embedded in the Go binary, TanStack Router + Query, CSS Modules + `tokens.css` (no component library, Radix only for the dialog and drawer), CSS-only animation, system fonts. TipTap v3 with an explicit extension list, a `paragraph block*` grammar for My Journal, and the bold Title done in CSS. Shared schema fixtures for TS and Go. pnpm/Biome/Vitest/Playwright in `web/`.
- [Operational basics](issues/10-operational-basics.md): Postgres-backed rate limits (per-email plus per-IP on auth, no lockout, 300 requests/min per user on the API). `slog` logs only allowlisted fields and never logs Page content, tokens or emails. Sentry free tier with heavy scrubbing. `/api/healthz` checked by UptimeRobot. The backup job has a minimum-size check and a healthchecks.io heartbeat. The `age` private key stays offline. Restore drill: before launch plus `RUNBOOK.md`, a monthly CI freshness check, a quarterly manual restore, and a point-in-time restore test twice a year.
- [API contract and autosave](issues/04-api-contract-and-autosave.md): JSON under `/api/v1` (`/api/healthz` unversioned), `YYYY-MM-DD` Days, coded errors. The server alone decides "today". A timezone change starts the next Day in the new zone after a "Switch / Keep" banner. Free Users' Grace Window saves get 60s of leeway after 02:00. Pages use GET/PUT with the full doc, `baseVersion`, and 409 on conflict ("Keep this version / Use the other" banner only when both sides have edits). Autosave after 1s idle or every 5s at most, mirrored to `localStorage`, silent unless failing. The rail pages by month with inline glimpses, plus a per-month written-Day summary. Search returns 20 results newest first, with snippets as segments.
- [Encryption at rest on Railway](issues/11-encryption-at-rest.md): Railway volumes, backups and point-in-time restore archives, R2, Fly and Neon all use AES-256 with provider-held keys. Staff access is limited by policy, not cryptography. The `age` dumps need nothing more, and an R2 bucket lock is advised. App-level encryption isn't worth it in v1, because it would break server-side search, JSONB migrations and Title queries. Privacy copy claims "encrypted at rest", never end-to-end.
- [Search results UI](issues/14-search-results-ui.md): results replace the Island rail (grouped by month, Day + Notebook + bold Title + 3-line snippet) and stay while you read. ⌘K, ↑↓, Enter, and Esc to clear back to the rail. The opened Page keeps blue matches lit until you type or press Esc. Dates are searched as text ("nov", "thursday"). This adds `title` to search results and the Day's date words to the tsvector.
- [Privacy and data ownership](issues/12-privacy-and-data-ownership.md): host encryption only (no app-level encryption, like Notion and Google Docs by default), plus an R2 bucket lock and a 90-day lifecycle on the dumps. No export in v1. "Deactivate account" means typing your email, with no way back, erasure within 1 year (so the purge must ship within a year) and a confirmation email. There is a short static privacy page that claims "encrypted at rest", never end-to-end.
- [Sidebar nav and Settings](issues/13-sidebar-nav-and-settings.md): two nav rows, Today and Settings. Settings is a Radix dialog (⌘,) with Keyboard shortcuts (⌘/), Email + verify, Sign-in (password via the reset email; Google display only), Timezone picker with a pending switch and Cancel, Privacy, Sign out, then Deactivate. `/me` gains `signInMethods`.
- [Sheet banners and first run](issues/15-sheet-banners-and-first-run.md): Notices live in one black notice pill at the top of the sheet area (Island style). It shows the most urgent Notice plus "+N" and expands on hover. Order: update, changed elsewhere, not saved, Day closed, timezone, verify email. Only the last two can be dismissed with ×. Day 1 shows placeholders plus three one-time coach marks (Page, Notebooks, rail), only while the user has no Written Day.

## Not yet specified

Nothing at the moment. Privacy, sidebar nav, search results UI, sheet banners and first run graduated into tickets on 2026-11-25.

## Out of scope

- **Phone apps (native or phone-specific design)**: a later effort. v1 is desktop-first web.
- **Billing / payments**: v1 only has the `plan` flag.
- **Premium features themselves** (edit history, editing past Pages): v1 only keeps the data model able to support them.
- **Elasticsearch**: ruled out. Postgres full-text search covers per-user journal volume.
- **More formatting** (strikethrough, divider, links, headings, photos, fonts, colours, sizes): not in v1, per [Formatting on Pages](issues/09-page-formatting.md). Divider, links and headings are parked to consider in a later effort.
- **User-created Notebooks**: a future feature, not planned in this effort. v1 has the two seeded Notebooks, per [Data model](issues/03-data-model.md).
- **Account re-activation and the 1-year purge of deleted accounts**: future work. v1 only soft-deletes, per [Data model](issues/03-data-model.md). The purge must ship within a year of the first deactivation, per [Privacy and data ownership](issues/12-privacy-and-data-ownership.md).
- **Export** (Markdown, JSON or any download of a journal): not in v1, per [Privacy and data ownership](issues/12-privacy-and-data-ownership.md).
- **End-to-end / app-level encryption**: not in v1, per [Privacy and data ownership](issues/12-privacy-and-data-ownership.md). It would move search, Titles and glimpses into the browser. Consider it later as an opt-in (Premium) mode.
