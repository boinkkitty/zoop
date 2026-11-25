# Hosting and deployment

Type: grilling
Status: resolved
Assignee: boinkkitty
Blocked by:

## Question

Where do the Go API, PostgreSQL and the React frontend run for v1, and how are they deployed? Options include Fly.io, Railway, Render, or a VPS, with managed Postgres such as Neon or Supabase-Postgres. Constraints:
- the frontend and API must share a site, so the HttpOnly session cookie works
- low cost at launch
- backups

Research: ../research/hosting.md

## Answer

Settled with the user on 2026-11-25 over four grilling rounds, based on [research/hosting.md](../research/hosting.md).

- **Shape: one Go binary, one Docker image.** It serves the embedded React build and `/api/*` from one origin, behind Railway's edge, which handles TLS and routing. There is no CORS, and the HttpOnly session cookie needs no `Domain=` setting. A separate frontend host with an `api.` subdomain was considered and deferred. It brings CORS, cookie scoping and two deploys, and before a domain exists the two hosts would be different sites, so the browser wouldn't send the session cookie between them. The frontend can be split out later without changing the API.
- **Platform: Railway Hobby, Singapore region**, about $5–8/month. It has a Go service and Railway Postgres in one project. Singapore was chosen because the user is moving back there, and the extra latency from France is acceptable because autosave runs in the background. Build on the trial credit, and **move to Hobby before any real journal is stored**. Free tiers were ruled out because none keeps restorable backups.
- **Backups, three layers:**
  1. Railway point-in-time restore (pgBackRest, about 4 weeks), **enabled when the database is created**, because the window only starts once it is on.
  2. Railway daily volume backups.
  3. A **nightly encrypted `pg_dump` to Cloudflare R2**, run on a GitHub Actions schedule. This costs effectively nothing: R2 is free up to 10 GB and the job uses about a minute a night.
- **Environments: local (Postgres via Docker Compose) and production only.** There is no staging or preview environment.
- **Deploys: GitHub Actions on push to `main`.** The workflow runs tests, builds the image, runs `goose up` before the new version takes traffic, then deploys. Migrations stay backward-compatible with the running version.
- **Domain: deferred until launch.** Until then the app runs on the free `*.up.railway.app` address. At launch, buy one at Cloudflare Registrar (about $10–15/year), serve the app on the bare domain (for example `zoop.app`), add the Resend DNS records, and put Cloudflare's proxy in front to get CDN caching of the JS/CSS files.
- **Fallback if Railway disappoints: a Fly.io app plus Neon Launch.** Moving takes about a day because it is one container and plain Postgres.
- **Known trade-offs:** Railway is a smaller company with past platform-wide outages, its point-in-time restore is new, and its volume backups are labelled as still under development. The R2 dump is the hedge. Compared with a high-end setup, v1 gives up a global CDN, separate frontend deploys, replicas and failover, and staging, none of which matter at v1 scale.

