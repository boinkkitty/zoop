# Operational basics

Type: grilling
Status: open
Blocked by:

## Question

With hosting settled on Railway (see [Hosting and deployment](08-hosting.md)), what does v1 need to run safely day to day? Decide:
- **Rate limiting** on the auth endpoints (login, signup, password reset, verification resend): the limits, whether they key on IP, email or both, and where the counters live (in-process memory, Postgres, or Railway Redis). This depends on how many Railway replicas run.
- **Logging**: structured logs from Go's `slog` to Railway's logs (Hobby keeps 7 days), and what must **never** be logged. Page content, session tokens and passwords are the obvious ones; journal text is intimate.
- **Error tracking**: Sentry or something similar on the free tier, for the Go API and the React app, with Page content scrubbed from reports.
- **Uptime and alerting**: an external uptime check on a health endpoint, plus alerts when the nightly R2 backup fails.
- **Restore drill**: how often to test restoring from point-in-time recovery and from the R2 dump, and who runs it.
