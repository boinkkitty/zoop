# Research: encryption at rest for Zoop's journal data

Ticket: `issues/11-encryption-at-rest.md`
Researched: 2026-11-25, from official docs, trust and privacy pages (read via browser that day). Items marked **(unconfirmed)** are inference or general knowledge that no primary page fetched this session states outright.

## Answer

- **Railway already encrypts everything Zoop stores at rest with AES-256 and keys that Railway manages.** That covers volumes (the Postgres data directory), volume backups, point-in-time-recovery (PITR) archives, storage buckets and environment variables [R1]. Railway does not document customer-managed keys **(unconfirmed absence)**. Railway staff are not cryptographically prevented from reading the data. Railway's systems decrypt data to run the service. Access to variables is "restricted and logged" [R1], and only an Enterprise HIPAA BAA removes staff's "direct access" to workloads [R1][R2]. Disk encryption protects against stolen disks. It does not protect against the operator.
- **Cloudflare R2 encrypts every object and its metadata with AES-256-GCM, using keys Cloudflare manages** [C1]. The nightly `age`-encrypted dump is already encrypted on the client before upload, so R2 and anyone holding the R2 token see only ciphertext. Nothing more is needed for confidentiality. Two cheap additions are worth making: an **R2 bucket lock** (retention rule), so a leaked CI token can't delete or overwrite past dumps [C3], and keeping the `age` private key offline, which is already decided.
- **The fallback is the same model.** Fly volumes are encrypted at rest by default, with keys in Fly's secret storage [F1][F2], and Fly Managed Postgres advertises encryption at rest [F3]. Neon uses AES-256 on NVMe storage, keys in AWS KMS / Azure Key Vault that are accessible only to specific IAM roles and logged, and backups in S3/Azure Blob with server-side encryption kept 30 days [N1][N2]. All of these are provider-managed keys.
- **App-level encryption of `pages.doc` / `plain_text` costs a lot for v1 and gives little.** The STORED `search` tsvector lists every word of the Page *with its position* [P2][P3]. Encrypting `doc` and `plain_text` while keeping that column would leak nearly the full text, and a generated column can't be computed from ciphertext. The real choice is to give up server-side search, as Journey does when E2EE is on [J1], or to keep plaintext. Because the key would live in a Railway env var that Railway decrypts to run the service [R1], encryption in the Go API would not stop the host from reading the data either. It would only protect copies of the database taken without the app's key. It would also break plans already made: schema migrations as SQL `UPDATE`s on JSONB, deriving Titles and glimpses in SQL, the `plain_text <> ''` Written-Day index, and Postgres compression.
- **What comparable apps do:** Day One uses client-side E2EE by default for new journals. The key stays on the device (optionally backed up to iCloud), and Day One says its employees can't read E2EE journals. Its older "standard encryption" is plain at-rest encryption with keys Day One manages [D1][D2]. Journey offers *optional* E2EE that turns off search and leaves dates in plaintext [J1][J2]. Diarium has no server at all: entries stay on the device or sync through the user's own cloud storage [Di1].
- **Recommendation for v1:** rely on provider at-rest encryption plus the `age` dump. Add an R2 bucket lock and tighten the PITR bucket credential. Do **not** claim "end-to-end encrypted" or "we can't read your journal" in privacy copy (ticket 12). E2EE is a separate future project that would need client-side search, key recovery UX and client-side migrations.

## Findings

### 1. Railway (chosen host)

**What is encrypted.** Railway's security docs say: "All data stored on Railway is encrypted at rest with AES-256 in every region … This includes volumes, volume backups, point-in-time recovery archives, storage buckets, and environment variables. Railway manages the encryption keys." [R1] The same page adds: "Backups and archives are encrypted at rest." [R1] The Trust Center says: "All data at rest within Railway's systems is encrypted using AES 256" [R3]. The DPA's Exhibit C says: "Databases are encrypted at rest, with access controls and monitoring in place" [R4].

- **Postgres volume:** covered ("volumes") [R1].
- **Volume backups (daily/weekly/monthly):** covered ("volume backups") [R1].
- **PITR archives:** covered ("point-in-time recovery archives") [R1]. They are written to a Railway storage bucket [R5]. Railway buckets run on Tigris's servers and "are encrypted at rest" [R6].

**Key management.** Railway manages the keys [R1]. Environment variables are also envelope encrypted, "under a key Railway holds separately" [R1]. No page describes customer-managed keys (BYOK), a KMS vendor, or rotation for volume data. **Unconfirmed:** whether any BYOK option exists, even on Enterprise. Railway buckets list "server-side encryption" (meaning S3 SSE headers, for example customer-provided keys) as "not yet supported" [R6].

**Can Railway staff read it?** Not stated outright for volumes. The evidence:
- Variables: "Railway's systems can decrypt any unsealed variable … That access is restricted and logged … To store a value Railway can't decrypt, encrypt it in your application first." Even sealed variables are decrypted to run the service [R1].
- Workloads: "Once the BAA is signed, Railway staff can no longer directly access your running workloads." This is Enterprise only, so by implication staff can access them without a BAA [R1][R2].
- Railway "at no point will a Railway team member modify your application without your expressed permission" [R2]. That is a policy promise about modifying, not a technical control on reading.
- Access to production needs MFA and encrypted connections, events are logged and reviewed, and staff sign NDAs and get background checks [R1][R4].
- **Conclusion (inference):** encryption with provider-held keys is transparent to Railway's own systems. Railway staff with production access could technically read Postgres data. Policy, logging and SOC 2 controls limit this, but cryptography does not.

**Infrastructure note.** The docs security page says Railway runs workloads "on hardware it owns and operates" (Railway Metal) [R1]. The Trust Center names Equinix data centres for physical security [R3]. The DPA still says "Railway infrastructure is run on Google Cloud Platform" [R4], and Google Cloud and AWS are listed as subprocessors [R3]. The newer docs page is likely the current state, but the documents are inconsistent **(unconfirmed which regions are still on GCP)**.

**Gotcha in Zoop's setup.** PITR restore variables reference the archive bucket's credential, which has read and write access with no read-only variant: "anyone who can see the restored service's variables can also write to the source's archive" [R5]. Treat that credential as sensitive. Seal it, and delete restored services after a drill.

**Compliance:** SOC 2 Type II, SOC 3, HIPAA (BAA on Enterprise), GDPR DPA signed self-service, EU-US DPF [R1][R2][R3].

### 2. Cloudflare R2 (off-host dump)

- "All objects stored in R2, including their metadata, are encrypted at rest. Encryption and decryption are automatic … Encryption keys are managed by Cloudflare … Objects are encrypted using AES-256 … R2 uses GCM … as its preferred mode." [C1] Data in transit uses TLS [C1].
- R2 also supports SSE-C (customer-provided keys per request), and Cloudflare can't recover objects if the key is lost [C2]. **Not needed for Zoop:** the dump is already `age`-encrypted on the GitHub runner before upload, so SSE-C would only add a second key to manage.
- `age` is "a simple, modern and secure file encryption tool, format, and Go library" [A1]. It encrypts to a public recipient key (X25519, now with post-quantum hybrid keys) [A1]. Only the holder of the private key (the user's password manager, per ticket 10) can decrypt. Cloudflare, GitHub and a thief of the R2 token see only ciphertext.
- **What's left to do:**
  1. **Bucket lock.** "Bucket locks prevent the deletion and overwriting of objects in an R2 bucket for a specified period — or indefinitely." [C3] A rule of, say, 35–90 days means a leaked R2 token can't wipe the backup history. Old dumps still contain deleted users' journals, so the retention period should match the deletion promise in ticket 12.
  2. Scope the R2 API token to the one bucket. **(unconfirmed whether R2 offers a put-only token; R2 token permissions were not checked this session)**
  3. Unchanged from ticket 10: the private key never goes into CI, and the quarterly decrypt drill proves the key still works.

### 3. Fallback: Fly.io + Neon

- **Fly volumes:** "Volumes are, by default, created with encryption-at-rest enabled … Use `--no-encryption` to instead create an unencrypted volume." [F1] "Fly Volumes are encrypted with keys stored in redundant industry-proven secret storage systems." [F2] Daily block-level snapshots are kept 1–60 days (5 by default) [F1]. **Unconfirmed:** whether snapshots are encrypted (not stated on the pages read; likely, since they are block-level copies of encrypted volumes). Fly doesn't document whether staff can read data.
- **Fly Managed Postgres:** "Automatic encryption of data at rest and in transit." [F3]
- **Neon:** NVMe storage "encrypted using an AES-256 block cipher implemented in a hardware module on the instance". Keys are in AWS KMS / Azure Key Vault "with key rotation policies"; "Only services and users with specific IAM roles can access the decryption keys, and all access is logged via AWS CloudTrail". Backups are in S3/Azure Blob "encrypted using server-side encryption (SSE) and … retained for 30 days." [N1][N2] Neon is SOC 2 and HIPAA compliant, and employees sign NDAs [N2].
- **Conclusion:** the same model as Railway, with provider-managed keys and insiders limited by IAM and logging, not by cryptography. Neon documents its key management in the most detail. Moving to the fallback changes nothing about the need for (or the uselessness of) app-level encryption.

### 4. What app-level encryption would cost

**What Postgres itself says** [P1]:
- Full-disk/filesystem encryption (what the providers do) "does not protect against attacks while the file system is mounted".
- `pgcrypto` column encryption puts "the decrypted data and the decryption key … on the server for a brief time … [where they] can be intercepted by someone with complete access to the database server".
- "If the system administrator for the server's machine cannot be trusted, it is necessary for the client to encrypt the data". For Zoop, Postgres's client is the Go API, which also runs on Railway.

**Threat model if Go encrypts with a key in a Railway env var:**
- **Protects against:** a leaked database credential or SQL injection reading raw rows, PITR archives or volume backups being exfiltrated (for example through the read-write PITR credential [R5]), and a leaked logical dump that isn't separately encrypted.
- **Does not protect against:** Railway insiders or a compromised Railway control plane. The key is an env var that Railway decrypts to run the container [R1]. It also doesn't protect against a compromised Go process or anyone with deploy access.
- So it is defence in depth against database-only leaks. It doesn't give "the host can't read it". Only E2EE (keys derived on the user's device) does that.

**Collision with the STORED tsvector ([Data model](../issues/03-data-model.md)):**
- `to_tsvector` "returns a tsvector which lists the lexemes together with their positions in the document" [P3], and tsvectors store lexemes with optional integer positions [P2]. Zoop uses the `simple` configuration, which keeps essentially every word lower-cased and drops no stop words **(general knowledge, not re-checked this session)**. So `search` is a near-copy of the Page text, ordered by position. Encrypting `doc` and `plain_text` but leaving `search` hides almost nothing.
- A generated column "can only use immutable functions" and references only the current row [P4]. It can't be computed from ciphertext unless the key is put inside the database, which defeats the purpose [P1].
- The options, then:
  1. **Drop server-side full-text search.** The client downloads and decrypts Pages and searches locally. Journey does this: turning on E2EE disables search "as this prevents Journey from indexing your journal entries" [J1]. Day One's E2EE journals work the same way, with every client holding the decrypted journal **(Day One's local search is inferred, not stated on the FAQ page)**. For Zoop's web app this means downloading the whole journal into the browser.
  2. **Blind index:** store HMAC(key, lexeme) tokens in an indexed array column. Exact-word search survives, but search-as-you-type prefix matching only works by also indexing every prefix. That grows the index and leaks word frequency and co-occurrence. **(general technique, no primary source fetched)**
  3. Keep `plain_text` and `search` in plaintext and encrypt only `doc`. This is pointless, because `plain_text` *is* the content.

**Other v1 decisions that app-level encryption would break:**
- The TipTap schema migration as one SQL `UPDATE` on JSONB ([Data model](../issues/03-data-model.md) context) becomes a Go batch job that decrypts and re-encrypts every Page.
- Titles and ~80-character glimpses are "derived when queried" in SQL. They would have to be derived in Go after decryption, or stored as separate ciphertext.
- The partial index `WHERE plain_text <> ''` for Written Days needs a separate plaintext `is_written` flag, which leaks *which* days were written.
- Postgres compression of large values (and the deferred LZ4 option) does nothing on ciphertext.
- Key rotation, plus key loss meaning permanent data loss, become operational risks. Today the only such key is the dump key.

### 5. What comparable journal apps claim

| App | Model | Search | Claim |
|---|---|---|---|
| **Day One** | E2EE by default for new journals. The key is on the device, optionally backed up to iCloud (Day One encrypts that copy before upload, but it isn't E2EE). Older "Standard encryption" = encrypted at rest on Day One's servers with keys Day One manages [D1] | Not stated on the FAQ page | "your journal data cannot be read by our servers or by Day One employees" [D1]. Privacy policy: with E2EE "it is impossible for our employees to access your journal data" [D2]. An external audit (nVisium) found 4 medium-severity and 3 low-severity issues [D1]. Exports and text backups are not encrypted [D1] |
| **Journey** | Optional E2EE with a user passphrase, RSA + AES. Media is uploaded unencrypted first and encrypted on the server. Dates and drive display names stay unencrypted [J1] | **Disabled when E2EE is on** [J1] | "If that option is enabled, it is impossible for our employees to access your journal data" [J2]. Without E2EE, entries live in the user's Google Drive, which Google encrypts at rest [J1] |
| **Diarium** | No account. The database stays on the device or syncs to the user's own OneDrive/Google Drive/Dropbox/iCloud/WebDAV [Di1][Di2] | Local | "Diarium does not send us your journal entries or attachments. We do not store or retain them on our servers." [Di2] |

What the market shows: apps that keep journals on their own servers either offer E2EE and accept the costs (key loss, no server search) or say only "encrypted at rest". Zoop v1 is in the second group. The honest claim is close to Day One's "standard encryption": encrypted in transit and at rest by our hosting providers, with access limited to the operator. It must not imply E2EE.

## Suggested follow-ups (for other tickets)

- **Ticket 12 (privacy):** word the claim as at-rest encryption, not E2EE. State how long deleted content survives in backups: Railway PITR about 4 weeks, monthly volume backups 89 days [R1], plus the R2 retention you choose.
- **Runbook:** seal the PITR bucket credential and delete restored services after drills [R5]. Add an R2 bucket-lock rule [C3].
- **Later, if E2EE becomes a goal:** design client-side search (in the browser, over decrypted Pages), key derivation and recovery (Day One's lost-key path [D1]), and migrations run on the client before committing.

## Sources

- [R1] Railway docs, Security — https://docs.railway.com/enterprise/security
- [R2] Railway docs, Compliance — https://docs.railway.com/enterprise/compliance (redirected from /maturity/compliance)
- [R3] Railway Trust Center (Data Security, Endpoint Security, Subprocessors) — https://trust.railway.com
- [R4] Railway DPA, Exhibit C — https://railway.com/legal/dpa
- [R5] Railway docs, Point-in-Time Recovery — https://docs.railway.com/volumes/point-in-time-recovery
- [R6] Railway docs, Storage buckets (S3 compatibility, FAQ) — https://docs.railway.com/storage-buckets
- [C1] Cloudflare R2, Data security — https://developers.cloudflare.com/r2/reference/data-security/
- [C2] Cloudflare R2, SSE-C — https://developers.cloudflare.com/r2/examples/ssec/
- [C3] Cloudflare R2, Bucket locks — https://developers.cloudflare.com/r2/buckets/bucket-locks/
- [A1] age — https://github.com/FiloSottile/age
- [F1] Fly.io Volumes overview — https://fly.io/docs/volumes/overview/
- [F2] Fly.io Security — https://fly.io/security/
- [F3] Fly.io Managed Postgres — https://fly.io/docs/mpg/
- [N1] Neon Security overview — https://neon.com/docs/security/security-overview
- [N2] Neon Security (trust page) — https://neon.com/security
- [P1] PostgreSQL docs, 18.8 Encryption Options — https://www.postgresql.org/docs/current/encryption-options.html
- [P2] PostgreSQL docs, Text Search Types — https://www.postgresql.org/docs/current/datatype-textsearch.html
- [P3] PostgreSQL docs, Controlling Text Search — https://www.postgresql.org/docs/current/textsearch-controls.html
- [P4] PostgreSQL docs, Generated Columns — https://www.postgresql.org/docs/current/ddl-generated-columns.html
- [D1] Day One, End-to-End Encryption FAQ — https://dayoneapp.com/guides/day-one-sync/end-to-end-encryption-faq/
- [D2] Day One, Privacy Policy — https://dayoneapp.com/privacy-policy/
- [J1] Journey, How does Encryption Work in Journey — https://support.journey.cloud/en/categories/synchronization-cloud/articles/how-does-encryption-work-in-journey
- [J2] Journey, Privacy policy — https://journey.cloud/policy
- [Di1] Diarium home — https://diariumapp.com/en
- [Di2] Diarium Privacy Policy — https://diariumapp.com/privacypolicy
