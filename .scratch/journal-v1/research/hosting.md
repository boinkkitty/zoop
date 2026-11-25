# Research: hosting for Zoop's Go app + PostgreSQL

Ticket: `issues/08-hosting.md`
Researched: 2026-11-25, from official pricing and docs pages (read via browser that day). Items marked **(unverified)** come from general knowledge or secondary sources and were not confirmed on a primary page this session.

## Question

Where should the single Go binary (API + built React SPA, same origin, HttpOnly session cookie) and PostgreSQL (STORED tsvector + GIN full-text search) run for v1? Compare Fly.io (+ Managed Postgres or Neon), Railway, Render, a Hetzner VPS with self-run Postgres, and Neon/Supabase as the managed DB. Constraints: very low cost at launch, and journals are intimate data, so backups and point-in-time recovery (PITR) matter a lot.

## Recommendation

**Railway Hobby (Go service + Railway Postgres with PITR enabled), about $5-8/month.** It is the only option where a cheap, always-on app *and* real PITR (WAL archived continuously by pgBackRest, ~4-week window) cost single-digit dollars a month [R2]. Add the daily volume-backup schedule as a second layer, plus your own weekly `pg_dump` to R2/B2 so one copy lives outside the host.

**Runner-up: Fly.io app (shared-cpu-1x, $2.19-3.69) + Neon Launch, about $5-25/month.** Neon gives instant restore to any point in a history window of up to 7 days on Launch, plus scheduled snapshots, with no monthly minimum [N1][N3]. It is cheap while the DB scales to zero, but scale-to-zero means a few-hundred-ms cold start on the first query, and keeping it always-on costs about $19/month in compute [N1]. Pin both to the same area (Fly `fra` + Neon `aws-eu-central-1`, or Fly `iad` + Neon `aws-us-east-1`).

Avoid for v1: **Fly Managed Postgres Developer** ($4, but only one daily backup kept 1 day, no PITR, auto-pauses with a 15-25 s wake) [F2], **Fly MPG Basic** (good backups but $38+/month) [F1], **Supabase** (PITR is a ~$100/month add-on on top of $25 Pro; free tier pauses and has no backups) [S1][S2], and **free tiers in general**: none gives restorable backups.

## Comparison

Launch cost = smallest always-on app + smallest durable Postgres with automatic backups, ~1 GB of data, negligible traffic.

| Option | ~Monthly at launch | App sleeps? | Postgres backups / PITR | Regions (EU / US) |
|---|---|---|---|---|
| **Railway Hobby** (app + Railway PG) | **~$5-8** ($5 minimum includes $5 usage) [R1] | No on Hobby; Free plan sleeps [R4] | Volume backups daily/weekly/monthly (kept 6 d / 27 d / 89 d) [R3]; optional **PITR via pgBackRest, ~4 weeks**, billed as bucket storage + egress only [R2] | Amsterdam / Virginia, California [R5] |
| **Fly app + Neon Launch** | **~$5-25** ($2.19-3.69 app [F1] + Neon usage; ~$19 compute if always-on [N1]) | Fly: no if auto-stop off. Neon: compute scales to zero after 5 min (can disable) [N1] | Instant restore (PITR) **up to 7 days**, snapshots + daily/weekly/monthly scheduled backups [N1] | Neon: Frankfurt, London / N. Virginia, Ohio, Oregon [N4]. Fly MPG/app regions below |
| **Fly app + Fly MPG Developer** | ~$6-8 ($2.19-3.69 + $4) [F1][F2] | App no; **DB auto-pauses after 15 min idle, wakes in ~15-25 s** [F2] | One full backup/day, **kept 1 day**, no PITR [F2] | MPG: ams, fra, lhr / iad, ord, dfw, lax, sjc, yyz [F3] |
| **Fly app + Fly MPG Basic** | ~$42 ($3.69 + $38 + $0.28/GB) [F1] | No | Backups kept 10 days, restore from backup **or point in time** [F2] | as above |
| **Render** (Starter web + Basic-256mb PG) | ~$13-14 ($7 + $6 + $0.30/GB) [D1] | Paid: no. Free web spins down after 15 min, ~1 min wake [D3] | PITR **3 days on Hobby workspace, 7 days on Pro ($25/mo)**; logical exports kept 7 days [D2] | Frankfurt / Oregon, Ohio, Virginia [D4] |
| **Supabase Pro** (DB only) + any app host | ~$27-30 ($25 incl. Micro compute [S1] + app) | Free projects pause after 1 week inactivity [S1] | Daily backups, 7 days. **PITR add-on ~$100/mo (7 d)** [S1][S2] | (unverified: many AWS regions incl. EU and US) |
| **Hetzner VPS** + Docker + Caddy + self-run PG, pgBackRest/WAL-G to R2/B2 | ~€6-7 (CX23 ~€5.49 after 2026 increases, secondary source) + R2 free ≤10 GB [H1][H2][O1] | No | Whatever you build: continuous WAL archiving = PITR with any retention, but **you** own monitoring, restore drills, upgrades, security patches | EU: Falkenstein, Nuremberg, Helsinki / US: Ashburn, Hillsboro [H3] |

## Findings

### Fly.io

- **Pricing**: no platform fee; shared-cpu-1x 256 MB is $2.19/30 days always-on, 512 MB $3.69, 1 GB $6.70 (Ashburn; Frankfurt is ~15% higher, e.g. 1 GB $7.73). Stopped machines pay only rootfs at $0.15/GB-month. Egress $0.02/GB in NA/EU. Shared IPv4 and IPv6 free; dedicated IPv4 $2/month. First 10 single-hostname TLS certs free (Let's Encrypt), then $0.10/month each [F1].
- **Free plan**: new accounts get $1 credit without card, $5/month usage once a card is added, max 3 machines; the $4 MPG Developer plan fits inside it [F1][F2].
- **Managed Postgres (MPG)**: Postgres 16 or 17 chosen at creation; no in-place major upgrade (dump/restore to a new cluster). Bundled extensions plus pgvector, PostGIS, pg_repack (v2). HA (primary + replica) on every plan, encryption at rest/in transit. Storage up to 1 TB (5 GB on Developer) [F4]. Plans: Developer $4 (shared-1x, 256 MB, storage included), Basic $38 (1 GB), Starter $72; storage $0.28/GB-month on non-Developer plans, metered on usage [F4].
- **Developer plan gotchas** [F2]: one full backup a day **kept for 1 day**, restore from backup only (no PITR); max 20 direct connections, no PgBouncer; auto-pauses after 15 min with no connections and takes ~15 s (up to 25 s) to wake, so set connect timeout ≥30 s; maintenance can happen any time; full 5 GB disk crashes Postgres. Fly itself says "keep your own copy of anything you can't lose, for example with pg_dump". Basic and up: backups kept 10 days, restore from backup or point in time, weekly maintenance window.
- **Recent changes**: from February 2026, inter-region private-network traffic for MPG is charged like Machines (same-region free) [F4]. The old unmanaged "Fly Postgres" is now labelled unsupported [F1].
- **Volumes** (if you ever self-run on Fly): $0.15/GB-month, daily snapshots kept 5 days by default, first 10 GB of snapshots free [F1].
- **Deploy / ops**: `fly deploy` builds from a Dockerfile (builders and registry currently free [F1]); GitHub Actions via `flyctl` + deploy token; `fly secrets set` for secrets (`fly mpg attach` writes `DATABASE_URL` as a secret [F2]); custom domains via `fly certs add` with automatic Let's Encrypt. No built-in preview environments; create a second app for staging. **(deploy/secrets/preview details unverified this session beyond the MPG attach step)**

### Railway

- **Plans** [R1]: Free $0 ($1/month usage, services sleep when idle, 0.5 GB volume); **Hobby $5 minimum usage, includes $5 credit**, up to 5 GB volume storage, single developer workspace, 7-day logs; Pro $20 minimum.
- **Usage rates** [R1]: RAM $0.00000386/GB-s (≈$10/GB-month), CPU $0.00000772/vCPU-s (≈$20/vCPU-month), volumes $0.00000006/GB-s (≈$0.16/GB-month), egress $0.05/GB, object storage (buckets) $0.015/GB-month with free egress. Billing is on *used* CPU/RAM, not allocated, so an idle Go binary (~30-60 MB) plus a small Postgres (~100-250 MB) plus a 1 GB volume should land inside or just above the $5 credit. (Estimate, not a quote.)
- **Free plan gotchas** [R4]: services and databases sleep after 5-10 min without *outbound* traffic; no custom domains, cron jobs or region choice; when the $1 is spent services stop until next cycle. Trial: $5 once for 30 days.
- **Backups** [R3]: volume backups (incremental, copy-on-write, billed at volume rates) on daily (kept 6 days), weekly (27 days) or monthly (89 days) schedules, plus manual. Caveats: "newer feature still under development", wiping a volume deletes all its backups, restore only into the same project + environment.
- **PITR** [R2]: one-click "Enable PITR" for Postgres. pgBackRest archives every WAL segment to a Railway bucket (`archive_timeout=60`), weekly full + daily differential base backups, last 4 fulls kept: **restore window ≈ 4 weeks**. Restore creates a new service at the chosen timestamp; source untouched. No separate fee: billed as bucket storage + upload egress (compressed). Window starts at the first base backup after enabling, so enable it on day one. Under a long S3 outage WAL is dropped after a 5 GiB queue (window truncates, DB stays up). CLI: `railway postgres pitr ...`. The doc does not state a plan restriction.
- **Regions** [R5]: US West (California), US East (Virginia), EU West (Amsterdam), Southeast Asia (Singapore). Moving a service with a volume means migration downtime.
- **Deploy / ops**: git push from a connected GitHub repo or `railway up`; Dockerfile or Railpack auto-build; variables per environment; custom domains with automatic TLS (Hobby+); environments (incl. PR environments) for staging. Postgres image version and extensions: **(unverified; Railway's template is the standard Postgres image, so built-in FTS/GIN works)**.

### Render

- **Workspace plans** [D1]: Hobby $0 + compute (5 GB bandwidth, single-service previews, custom domains); Pro $25/month + compute (25 GB bandwidth, full-stack previews, isolated environments).
- **Compute** [D1]: web service Free (512 MB, limits apply) or Starter $7 (<1 CPU, 512 MB). **Postgres** Free, Basic-256mb $6, Basic-1gb $19; storage $0.30/GB; PITR and logical exports on paid instances.
- **Free tier gotchas** [D3]: free web services spin down after 15 min without inbound traffic and take ~1 min to spin up; free Postgres **expires 30 days after creation** (14-day grace, then deleted) and has **no backups of any kind**; 750 free instance hours/month.
- **Backups** [D2]: continuous backups for PITR on paid databases. **Recovery window 3 days on Hobby workspace, 7 days on Pro or higher**; restore creates a new instance. Logical exports kept 7 days; Render's docs suggest a cron job pushing `pg_dump` to S3 for longer retention. Backups are stored in the same region as the DB.
- **Regions** [D4]: Oregon, Ohio, Virginia, Frankfurt, Singapore. Region cannot be changed after creation.
- **Deploy / ops**: git push auto-deploy, Dockerfile or native Go runtime, `render.yaml` blueprints, environment groups for secrets, managed TLS on custom domains [D1]. **(Postgres major versions offered: unverified)**

### Neon

- **Plans** [N1] (page "last updated November 4, 2026"): Free $0 (100 projects, 100 CU-hours/month per project, 1 GB storage per project, 5 GB egress per project, instant-restore history up to 6 hours or 1 GB of changes, 1 snapshot). **Launch: usage-based, no monthly minimum**: $0.106/CU-hour, $0.35/GB-month storage, $0.20/GB-month history storage, instant restore **up to 7 days**, 100 snapshots, scheduled backups daily/weekly/monthly at $0.09/GB-month, 500 GB egress per project then $0.10/GB. Scale: $0.222/CU-hour, history up to 30 days.
- **Cost shape** [N1]: scale-to-zero after 5 min (can be disabled on Launch); restart "within a few hundred milliseconds". Always-on minimum is 0.25 CU × 730 h ≈ 182.5 CU-hours ≈ **$19.35/month** compute. A journaling app used a few hours a day with scale-to-zero would be a few dollars.
- **Free gotchas** [N1]: hitting 100 CU-hours or 5 GB transfer **suspends compute until next billing period**. The regions page notes Free plan projects inactive 90+ days are subject to deletion as of 2026-10-05 [N4].
- **Postgres** [N2]: versions 14-19 supported, 18 is the default for new projects; many extensions (pgvector, PostGIS, TimescaleDB...). Built-in FTS works.
- **Restore** [N3]: instant restore of a branch to any point in its history window, down to the millisecond, with Time Travel Assist to query the past before restoring. Branching gives cheap preview/staging DBs ($0.002/branch-hour beyond included) [N1].
- **Regions** [N4]: AWS N. Virginia, Ohio, Oregon, Frankfurt, London, Singapore, Sydney, São Paulo. Azure regions deprecated (no new projects).
- **Recent change**: Neon now brands its Postgres as "Lakebase Postgres, by Databricks" and bundles Auth, Object Storage, Functions [N1].

### Supabase

- **Plans** [S1]: Free (2 active projects, **paused after 1 week of inactivity**, 500 MB DB, 5 GB egress, **no backups**: Supabase recommends regular `supabase db dump` exports [S2]). **Pro from $25/month**, includes $10 compute credit covering one Micro (1 GB, shared) instance; 8 GB disk then $0.125/GB; 250 GB egress; **daily backups kept 7 days**. Spend cap on by default.
- **PITR** [S2][S3]: add-on on Pro and up, available on all compute sizes (replaces daily backups when on): **7 days ≈ $100/month**, 14 days ≈ $200, 28 days ≈ $400. Restoring makes the project unavailable during the restore (in place, not a fork).
- **Fit for Zoop**: you'd use only its Postgres (FTS fine). For the backup level wanted here it costs ~$125/month, the most expensive option. Supabase Postgres version and region list: **(unverified this session)**.

### Hetzner VPS (self-run)

- **Prices**: Hetzner raised cloud prices for new orders twice in 2026 (1 April, and 15 June affecting "all dedicated servers and cloud plans at all locations"; existing contracts keep their terms; volumes, snapshots, IPs, object storage not affected) [H1]. The pricing pages did not render prices in the browser on 2026-11-25 and listed CX/CAX types as "currently unavailable" in the EU locations shown [H2]. Secondary trackers report CX23 (2 vCPU, 4 GB, 40 GB NVMe, 20 TB traffic) at about **€5.49/month** after the increases **(secondary source, unverified)**.
- **Locations** [H3]: Falkenstein, Nuremberg, Helsinki; Ashburn, Hillsboro (US); Singapore. US locations include less traffic than EU.
- **Backup target**: Cloudflare R2 Standard $0.015/GB-month, **first 10 GB-month free**, 1 M Class A / 10 M Class B ops free, **no egress fees** [O1]. Backblaze B2 $6.95/TB-month, free egress up to 3× storage [O2]. A small journal DB plus a few weeks of compressed WAL fits in R2's free tier.
- **Trade-off**: cheapest and most flexible (any Postgres version, any extension, PITR with whatever retention you configure via pgBackRest/WAL-G), but the backup is only as good as your monitoring and restore drills. Single VM = no HA; disk loss or a bad `apt upgrade` is on you. Deploy via GitHub Actions → registry → `docker compose pull && up` over SSH (or Kamal); Caddy gets Let's Encrypt certs automatically; secrets live in env files on the box. No preview environments.

## Gotchas that matter for Zoop

- **Enable PITR on day one** wherever it is opt-in (Railway): the window starts at the first base backup after enabling, not retroactively [R2].
- **Keep an off-host logical backup anyway** (nightly `pg_dump` → R2, encrypted). Every provider stores its backups in the same account/region, and Railway notes wiping a volume deletes its backups [R3]; Fly says to keep your own copy on Developer [F2].
- **Free tiers are not an option for real journals**: Render free DB is deleted after 30 days + 14 grace and has no backups [D3]; Supabase free pauses after a week and has no backups [S1][S2]; Railway free sleeps and stops when $1 runs out [R4]; Fly MPG Developer keeps one backup for one day [F2].
- **Cold starts**: Fly MPG Developer wakes in 15-25 s [F2]; Neon scale-to-zero wakes in a few hundred ms [N1]; Render free web ~1 min [D3]. A pgx pool with `MinConns > 0` keeps Fly Developer / Neon awake (and billing) all the time.
- **Same origin**: every option serves the Go binary + SPA from one host, so the HttpOnly cookie works anywhere; no platform-specific constraint here.

## Sources

- [F1] Fly.io Resource Pricing — https://fly.io/docs/about/pricing/
- [F2] Fly.io Managed Postgres Developer plan — https://docs.fly.io/postgres/developer-plan.md
- [F3] Fly.io regions (MPG column) — https://docs.fly.io/reference/regions.md
- [F4] Fly.io Managed Postgres overview and pricing — https://docs.fly.io/postgres
- [R1] Railway pricing — https://railway.com/pricing
- [R2] Railway Point-in-Time Recovery — https://docs.railway.com/volumes/point-in-time-recovery
- [R3] Railway volume backups — https://docs.railway.com/volumes/backups
- [R4] Railway Free plan — https://docs.railway.com/pricing/free-plan
- [R5] Railway regions — https://docs.railway.com/deployments/regions
- [D1] Render pricing — https://render.com/pricing
- [D2] Render Postgres recovery and backups — https://render.com/docs/postgresql-backups
- [D3] Render free instances — https://render.com/docs/free
- [D4] Render regions — https://render.com/docs/regions
- [N1] Neon pricing — https://neon.com/pricing
- [N2] Neon Postgres version support — https://neon.com/docs/postgresql/postgres-version-support
- [N3] Neon instant restore — https://neon.com/docs/postgres/backup-restore/branch-restore
- [N4] Neon regions — https://neon.com/docs/introduction/regions
- [S1] Supabase pricing — https://supabase.com/pricing
- [S2] Supabase backups — https://supabase.com/docs/guides/platform/backups
- [S3] Supabase PITR usage billing — https://supabase.com/docs/guides/platform/manage-your-usage/point-in-time-recovery
- [H1] Hetzner press release, price adjustment effective 15 June 2026 — https://www.hetzner.com/pressroom/standardization-and-price-adjustment-of-our-server-products/
- [H2] Hetzner Cloud cost-optimized / regular-performance plans — https://www.hetzner.com/cloud/cost-optimized/ , https://www.hetzner.com/cloud/regular-performance/
- [H3] Hetzner Cloud overview (locations) — https://www.hetzner.com/cloud/
- [O1] Cloudflare R2 pricing — https://developers.cloudflare.com/r2/pricing/
- [O2] Backblaze B2 pricing — https://www.backblaze.com/cloud-storage/pricing
