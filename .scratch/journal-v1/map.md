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

## Not yet specified

- **Search results UI**: the search box sits at the top of the left sidebar, which is decided. How results display (a dropdown, replacing the rail, or over the sheet) is still open.
- **Privacy and data ownership**: journals are intimate. Open questions are encryption at rest or per-user encryption, account deletion, and data export. Export formats can start from the Markdown export in [Formatting on Pages](issues/09-page-formatting.md).
- **Sidebar nav items**: what Home / Filler 1 / Filler 2 really are (for example settings, account, a calendar view). The user expects 2–3 buttons. One of them must lead to the keyboard-shortcut sheet.
- **First-run experience**: what a brand-new user sees on Day 1 (empty state, hints about the two Notebooks).

## Out of scope

- **Phone apps (native or phone-specific design)**: a later effort. v1 is desktop-first web.
- **Billing / payments**: v1 only has the `plan` flag.
- **Premium features themselves** (edit history, editing past Pages): v1 only keeps the data model able to support them.
- **Elasticsearch**: ruled out. Postgres full-text search covers per-user journal volume.
- **More formatting** (strikethrough, divider, links, headings, photos, fonts, colours, sizes): not in v1, per [Formatting on Pages](issues/09-page-formatting.md). Divider, links and headings are parked to consider in a later effort.
