# Privacy and data ownership

Type: grilling
Status: resolved
Assignee: boinkkitty
Blocked by: 11

## Question

How does v1 protect and hand back a user's journal?
- Encryption: rely on host disk encryption only, or encrypt Page content in the app (per-user keys or one app key), given that it affects Postgres full-text search?
- Export: format (Markdown from [Formatting on Pages](09-page-formatting.md), JSON, a zip of both), scope (all Pages or a date range), and delivery (a direct download or emailed when ready).
- Account deletion as the user sees it: confirmation steps, whether export is offered first, and what is said about the soft delete and the later 1-year purge ([Data model](03-data-model.md)).
- What v1 promises in a privacy note (who can read Pages, where data lives, Sentry and log scrubbing per [Operational basics](10-operational-basics.md)).

Research: ../research/encryption-at-rest.md, ../research/app-level-encryption-comparables.md

## Answer

Grilled 2026-11-25.

- **Encryption: host encryption only.** Railway, R2 and the fallback hosts encrypt at rest with AES-256 and provider-held keys, plus the `age`-encrypted nightly dump. There is no app-level encryption, which matches the default of Notion, Google Docs, Evernote, OneNote and Apple Notes ([comparables research](../research/app-level-encryption-comparables.md)). Server-side envelope encryption was ruled out because it only protects against database-only leaks and would break the STORED tsvector, the Titles and glimpses built in SQL, and the JSONB migrations. Browser end-to-end encryption is out of scope (see the map). Add the **R2 bucket lock** from [Encryption at rest on Railway](11-encryption-at-rest.md), with a lock period shorter than the 90-day lifecycle below so expired dumps can still be removed.
- **Export: none in v1.** Out of scope: no Markdown or JSON download, no date range, no emailed link. This is a known gap: an EU user's GDPR request for their data would be answered by hand from SQL.
- **Account deactivation** (the user-facing word, replacing "delete account"):
  - Settings → "Deactivate account" opens a dialog that explains what happens, and the user **types their email** to confirm. There is no password or Google re-sign-in.
  - On confirm: `users.deleted_at` is set, sessions and email tokens are hard-deleted (as in [Data model](03-data-model.md)), the user is signed out, and Resend sends a "Your Zoop account was deactivated" email.
  - **No way back in v1.** Signing in to a deactivated account shows a plain "This account was deactivated" notice and creates no session. The email can't sign up again. Re-activation stays future work.
- **What deactivation promises** (the same wording in the dialog and on the privacy page): *"Your journal is hidden immediately and permanently erased within 1 year. Backups containing it expire within 90 days after that. This email can't be used to sign up again."*
  - Consequence: set an **R2 lifecycle rule of 90 days** on the nightly dumps. Railway point-in-time restore (~4 weeks) and volume backups (≤89 days) already fall within that.
  - Consequence: the 1-year purge is future work, but this promise means **it must ship within a year of the first deactivation**.
- **Privacy page: a short static page** (a public SPA route, no login), linked from the signup/login screens and Settings. Google's OAuth consent screen needs its URL, and PDPA and GDPR expect one. It contains:
  - where data lives (Railway, Singapore, with encrypted backups on Cloudflare R2),
  - subprocessors (Railway, Cloudflare, Resend, Sentry, and Google for sign-in only),
  - that Page content, emails and tokens are never logged or sent to Sentry ([Operational basics](10-operational-basics.md)),
  - the encryption claim, *"encrypted in transit and at rest by our hosting providers"*. Never "end-to-end", and never "we can't read your journal".
  - the deactivation wording above.

  It has no marketing promises and no export claim.
