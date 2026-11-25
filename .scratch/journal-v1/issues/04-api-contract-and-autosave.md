# API contract and autosave

Type: grilling
Status: resolved
Assignee: boinkkitty
Blocked by: 03

## Question

What are the v1 HTTP endpoints, their request/response shapes, and how does autosave behave? Cover:
- auth flows
- fetching a range of Days for the rail, including Titles and hover glimpses. Per [Island rail and hover expansion](06-island-rail-prototype.md), a page must be fetchable from **any anchor date in either direction** (a jump loads only the target month, then newer Days above and older Days below), and the rail's scrubber needs a **per-month written-Day summary across the whole history**
- fetching and saving one Page
- search
- autosave debounce
- conflicts when the same Page is open in two browser tabs or on two devices
- how the server decides "today" and enforces the Grace Window lock for Free Users at 02:00

## Context (2026-11-25)

- **From [Data model](03-data-model.md):** `pages.version` goes up by one on every save, for detecting conflicting saves. A timezone change is stored as `next_timezone` + `timezone_switch_day`, so this ticket defines the rule for "today" (it never moves backwards). Title and glimpses are derived from `plain_text` when queried. Caps are request bodies up to 512 KB and `plain_text` up to 100,000 characters. Soft-deleted users can't sign in.
- **To consider, measure in production first:** gzip or brotli compression of HTTP responses (chi `middleware.Compress`, or at the host or proxy). It would help most when loading long Pages and months of rail data. Autosave uploads would stay uncompressed. Not decided for v1, so turn it on only if production numbers show a benefit.
- From [Frontend stack and editor](07-frontend-stack.md): the client uses TanStack Query, and the rail is a `useInfiniteQuery` paging both ways, so rail responses should carry cursors for the next and previous pages. The SPA's own URLs are `/day/<YYYY-MM-DD>/<notebook-slug>`, and every non-`/api` path falls back to `index.html`.
- **From [Operational basics](10-operational-basics.md) (2026-11-25):** the auth endpoints are rate-limited (login 5 failures per email per 15 min plus 20 per IP; signup 5 per IP per hour; reset and resend 3 per email per hour plus 10 per IP), and authenticated `/api/*` is limited to 300 requests/min per user. All limits return 429 with `Retry-After`. Reset and resend always give the same generic response. `GET /api/healthz` needs no auth and returns a bare 200 after a database check. Autosave debounce and search-as-you-type must stay well under 300 requests/min.

## Answer

Settled with the user on 2026-11-25 over three grilling rounds.

**Conventions**
- JSON under **`/api/v1`**. Only `GET /api/healthz` stays unversioned (fixed by [Operational basics](10-operational-basics.md)).
- camelCase fields. Days are ISO 8601 `YYYY-MM-DD` (a Day is a date in the user's zone, never a moment). Moments (`updatedAt`, `locksAt`) are RFC 3339 UTC. Notebooks are addressed by slug (`my-journal`, `i-learned`).
- Errors: `{ "error": { "code": "page_locked", "message": "…" } }` from a fixed list of codes, including `page_locked`, `version_conflict`, `client_outdated`, `not_found`, `rate_limited`, `validation`, `unauthenticated`.
- The session cookie is HttpOnly + `SameSite=Lax`, and `http.CrossOriginProtection` wraps the router for CSRF.

**Auth** (all under `/api/v1`)

| Endpoint | Body / notes |
|---|---|
| `POST /auth/signup` | `{email, password, timezone}`. Creates the user, sends the verify email and **signs in immediately**. |
| `POST /auth/login` | `{email, password}` |
| `POST /auth/logout` | Deletes the current session. |
| `GET /auth/google/start` → `GET /auth/google/callback` | PKCE + state + nonce, per [Go backend building blocks](02-go-backend-building-blocks.md). |
| `POST /auth/verify` | `{token}` |
| `POST /auth/verify/resend` | Generic response. |
| `POST /auth/password/forgot` | `{email}`, generic response. Also the way a Google-only user adds a password. |
| `POST /auth/password/reset` | `{token, password}`. Deletes all of the user's sessions. |
| `GET /me` | `{email, emailVerified, plan, timezone, pendingTimezone, firstDay, today, graceUntil}` |
| `PATCH /me` | `{timezone}` records a pending switch. |
| `DELETE /me` | Soft delete. |

- Email links open SPA routes (`/verify?token=…`, `/reset?token=…`), and the page **POSTs** the token, so link-scanners that send a GET can't use up single-use tokens.
- Unverified users can use everything, with a quiet "verify your email" banner.
- **Pre-hijack guard**: when Google signs in to an account whose email is unverified, the server deletes the unverified password identity, marks the email verified, then links Google.

**"Today" and timezone**
- The **server alone** decides "today". The client never sends it.
- Effective zone: `next_timezone` once that zone's local date is ≥ `timezone_switch_day` (the server then copies it into `timezone` and clears the pending pair). Otherwise `timezone`. `timezone_switch_day` = tomorrow in the old zone when the change is recorded. A second change before the switch replaces the pending zone.
- Result: flying west gives one long Day, flying east one short Day. No Day is skipped or repeated, and today never moves backwards. The Grace Window always uses the effective zone, with no special case on switch days.
- **Detection**: on load the client compares `Intl.DateTimeFormat().resolvedOptions().timeZone` with the profile. If they differ, a quiet banner on the sheet (not a modal) asks *"Looks like you're on Tokyo time. Start your Days on Tokyo time from tomorrow? — Switch / Keep London"*. Switch → `PATCH /me`. Keep is remembered per detected zone in that browser's `localStorage`. Signup sets the zone silently.

**Who can write which Day** (checked by the server on every save)
- Future Days: nobody. Days before `first_day`: 404 for everyone.
- Free Users: today, plus yesterday until 02:00 local, with a **60-second server-side leeway** so an in-flight autosave still lands. The client force-flushes at `locksAt`, then makes the editor read-only. The client's clock is never trusted.
- Premium Users: any Day from `first_day` to today. The `plan` flag is enforced in v1 even though nobody can buy Premium yet.
- A locked save returns 403 `page_locked`.

**Pages**
- `GET /pages/{day}/{notebook}` → `{day, notebook, doc, schemaVersion, version, updatedAt, editable, locksAt}`. A Page with no row returns an empty doc with `version: 0` (no 404, no row written).
- `PUT /pages/{day}/{notebook}` with `{doc, schemaVersion, baseVersion}`. It sends the **full document** every time (≤512 KB) and upserts. The **server derives `plain_text`** from the doc in Go and validates the doc against the schema.
- A `schemaVersion` older than the server's → 409 `client_outdated`. The tab keeps its `localStorage` copy and shows "Zoop was updated — reload". The reloaded client sends that copy, upgraded to the new format.

**Autosave**
- Save after **1s idle**, or every **5s at most** while typing continues. Also flush on blur, `visibilitychange` → hidden and route change, plus `fetch(…, {keepalive:true})` on unload.
- One save in flight per Page. A newer edit replaces any queued one, so continuous typing is at most ~12 requests/min.
- Until the server acks it, the latest unsaved doc is mirrored to `localStorage` per Page.
- **No "Saving…" indicator.** Nothing shows while saves succeed. "Not saved — retrying" appears only after failures, with exponential backoff.

**Conflicts** (two tabs or two devices)
- A mismatched `baseVersion` → 409 `version_conflict` with the current Page in the body.
- No unsaved local edits: the client quietly takes the newer version. TanStack Query's refetch-on-focus covers the common "switched laptops" case.
- Edits on both sides: a quiet banner on the sheet, *"This Page changed on another device — Keep this version / Use the other"*. Keep re-sends with the new `baseVersion`.
- No automatic merge and no live sync between tabs in v1.

**Rail**
- `GET /days?month=YYYY-MM` → `{month, days: [{day, written, title, glimpses: {"my-journal": "…", "i-learned": "…"}}], newerMonth, olderMonth}`. It covers every Day of the month within `first_day`..today, newest first. Cursors are null at either end. Glimpses come inline, so hover needs no request. A month lines up with the sticky month headers and with scrubber jumps, and it is the `useInfiniteQuery` page unit in both directions.
- `GET /days/summary` → `{firstDay, today, months: [{month, writtenDays}]}` over the whole history, from one `GROUP BY` on the partial index. The client invalidates it when a save changes a Day between written and unwritten. Tick width uses written Days, not character counts.

**Search**
- `GET /search?q=…&cursor=…` → 20 results, **newest Day first** (not ranked by relevance), each `{day, notebook, snippet: [{text, match}]}`. The server splits `ts_headline` output on private-use-character markers, so the client renders plain text segments and never parses HTML.
- The client searches from 1 character, 200ms after typing stops, and cancels the previous request with `AbortController`. That stays well under the 300 requests/min limit alongside autosave.

- **From [Search results UI](14-search-results-ui.md) (2026-11-25):** each result also carries `title` (the Title derived from `plain_text`, empty for I Learned), so results are `{day, notebook, title, snippet}`. Date words (weekday, day, month, year) match like text through the tsvector, so the query needs no date parsing. If only the date matches, the snippet is the start of the Page with nothing marked.

**Deferred**: HTTP response compression, turned on only if production numbers show a benefit (unchanged from the Context above).

No ADR: each choice is easy to reverse or follows the conventions. No new glossary terms.
