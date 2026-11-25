# Operational basics

Type: grilling
Status: resolved
Assignee: boinkkitty
Blocked by:

## Question

With hosting settled on Railway (see [Hosting and deployment](08-hosting.md)), what does v1 need to run safely day to day? Decide:
- **Rate limiting** on the auth endpoints (login, signup, password reset, verification resend): the limits, whether they key on IP, email or both, and where the counters live (in-process memory, Postgres, or Railway Redis). This depends on how many Railway replicas run.
- **Logging**: structured logs from Go's `slog` to Railway's logs (Hobby keeps 7 days), and what must **never** be logged. Page content, session tokens and passwords are the obvious ones; journal text is intimate.
- **Error tracking**: Sentry or something similar on the free tier, for the Go API and the React app, with Page content scrubbed from reports.
- **Uptime and alerting**: an external uptime check on a health endpoint, plus alerts when the nightly R2 backup fails.
- **Restore drill**: how often to test restoring from point-in-time recovery and from the R2 dump, and who runs it.

## Answer

Settled with the user on 2026-11-25 over two grilling rounds. Railway runs a single service with no replicas (per [Hosting and deployment](08-hosting.md)).

- **Rate-limit storage: Postgres.** A fixed-window counter table keyed by (key, window), with expired rows deleted as new rows are written, behind a `RateLimiter` interface so it can move to Redis later. No Redis. Counts survive deploys and keep working if a second replica is ever added. In-process memory was ruled out because every deploy would reset the counts.
- **Auth limits** (a hit returns 429 with `Retry-After`, and **no account lockout**, because lockout lets anyone lock a real user out):
  - Login: 5 *failed* attempts per email per 15 min, plus 20 per IP per 15 min. Successful logins don't count.
  - Signup: 5 per IP per hour.
  - Password reset request and verification resend: 3 per email per hour plus 10 per IP per hour. The response is always the same generic "if that account exists, we've sent an email", so attackers can't learn which emails have accounts. This also protects the Resend cap of 100 emails per day.
  - The client IP comes from Railway's forwarded header, trusted one hop only.
- **General API limit:** 300 requests/min per user on authenticated `/api/*`, using the same limiter. It catches runaway autosave loops and scripted clients and sits well above normal search-as-you-type and autosave traffic.
- **Logging:** `slog` JSON to stdout, so it lands in Railway's logs (7 days on Hobby). Only an **allowlist** of fields is logged: method, route *pattern* (never the raw URL), status, latency, user id, request id. **Never logged:** Page content (JSON or plain text), Titles, search queries, session tokens or cookies, verification and reset tokens, passwords or hashes, OAuth codes and tokens, any request or response body, and email addresses (log the user id instead). Redaction is enforced in code: sensitive types implement `LogValue()` and return `[redacted]`.
- **Error tracking: Sentry free (Developer) tier** for both the Go API and the React app. `SendDefaultPII: false`, no request bodies, no editor or console breadcrumbs, Session Replay off, and a `beforeSend` hook that strips request data and any `extra` fields not on an allowlist. Frontend source maps are uploaded in CI.
- **Health and uptime:** `GET /api/healthz` returns a bare 200 when the process is up and `SELECT 1` succeeds within ~1s. It returns no details and needs no auth. UptimeRobot (free) checks it every 5 min and alerts by email.
- **Backup alerts:** the nightly `pg_dump` job (1) fails, and GitHub emails, if the dump is below a minimum byte size, and (2) pings a **healthchecks.io** heartbeat only after a successful upload. A missing ping alerts after ~26h. This catches a schedule that never runs, for example when GitHub disables cron on a repo after 60 days of inactivity.
- **Dump encryption key custody:** dumps are encrypted with `age` to a recipient (public) key held in GitHub Secrets. The **private key lives only in the user's password manager**, never in CI.
- **Restore drill** (run by the user, the only operator):
  - **Before launch:** restore an R2 dump into local Docker Postgres, do a point-in-time restore on Railway, and write both up in `RUNBOOK.md`.
  - **Monthly, automated in GitHub Actions:** check that the newest R2 object exists, is less than ~26h old and is above the minimum size. It does **not** decrypt, because CI holds no private key.
  - **Quarterly, manual:** decrypt the latest R2 dump, restore it locally, and check row counts and a known Page.
  - **Twice a year, manual:** test Railway's point-in-time restore.
