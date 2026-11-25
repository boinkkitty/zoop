# Encryption at rest on Railway

Type: research
Status: resolved
Assignee: boinkkitty
Blocked by:

## Question

What protects Zoop's journal data at rest under the hosting decided in [Hosting and deployment](08-hosting.md), and what is left to do?
- Does Railway encrypt Postgres volumes, volume backups and point-in-time restore storage at rest, and with what key management? Can Railway staff read them?
- Does Cloudflare R2 encrypt stored objects, and does the nightly `age`-encrypted `pg_dump` (per [Operational basics](10-operational-basics.md)) need anything more?
- Same questions for the fallback (Fly.io volumes + Neon).
- What would app-level encryption of `pages.doc` / `plain_text` cost? Specifically, how does it interact with the STORED tsvector search column ([Data model](03-data-model.md)), and what do comparable journal apps (Day One, Journey, Diarium) do and claim?

Research: ../research/encryption-at-rest.md

## Answer

Researched 2026-11-25 from primary sources. Railway encrypts volumes, volume backups, PITR archives, buckets and env vars at rest with AES-256, using keys Railway manages. R2 encrypts every object with AES-256-GCM, using keys Cloudflare manages. Fly volumes and Neon follow the same model. In every case the provider holds the keys, so this protects against stolen disks, not against the operator: Railway staff are limited by policy and logging, not by cryptography (only an Enterprise BAA removes their direct access). The `age`-encrypted R2 dump needs nothing more for confidentiality. Add an R2 bucket lock so a leaked token can't delete history, and seal the read-write PITR bucket credential. App-level encryption isn't worth it for v1. The STORED `simple` tsvector holds every word with its position, so encryption would mean giving up server-side search (as Journey does under E2EE). It would also break the SQL-only JSONB migrations, the Title and glimpse queries and the Written-Day index, and with the key in a Railway env var it still wouldn't hide data from the host. Day One defaults to true E2EE, Journey makes E2EE optional and loses search, and Diarium keeps no server copy at all. Zoop's privacy copy should claim at-rest encryption only, never E2EE.

Full findings: [research/encryption-at-rest.md](../research/encryption-at-rest.md)
