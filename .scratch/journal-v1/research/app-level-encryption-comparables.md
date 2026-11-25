# Research: do well-known notes and journal apps encrypt at the application level?

Ticket: `issues/11-encryption-at-rest.md` (follow-up to [encryption-at-rest.md](encryption-at-rest.md), which already covers Railway, R2, Fly, Neon, Day One, Journey and Diarium; those findings are not repeated here).
Researched: 2026-11-25, from official security pages, help centres, privacy policies and platform-security guides (read via browser that day). Items marked **(unconfirmed)** are inference or general knowledge that no primary page fetched this session states outright.

## Terms used

- **(a) At-rest only.** Disk, volume or storage-layer encryption. The provider holds the keys, and decryption happens automatically for any authorised service. Protects stolen disks and discarded media.
- **(b) Server-side application-level (envelope) encryption.** The app or storage service encrypts records or chunks with data keys (DEKs). Those keys are wrapped by key-encryption keys (KEKs) in a KMS. The service still holds or can call the keys, so it decrypts to search, index and render. Adds a per-object key, a central audit point, and (with customer-managed keys) a kill switch.
- **(c) End-to-end / client-side encryption (E2EE).** Keys are derived or held on the user's (or the customer organisation's) side. The provider stores ciphertext it can't decrypt.

## Answer

- **Most big mainstream apps are (a) or (b) by default, and the provider can decrypt.** Notion [NO1][NO2], Google Docs/Drive [G1][G2], Evernote [E1], Microsoft OneNote / Microsoft 365 [M1] and Apple Notes under standard iCloud protection [AP1] all use provider-held keys. Each says so, directly or by implication. Their staff access is limited by policy, consent toggles and logging, not by cryptography. Examples: Notion support access needs a user/admin toggle and expires after 28 days [NO3]; Google offers "Access Transparency" logs of "actions taken by Google staff when accessing user content" [G5]; Evernote staff can review Content "with your express permission" [E3]; Apple "can decrypt your data on your behalf" [AP1].
- **The "big app" form of app-level encryption is (b), and it targets insiders and storage compromise, not the operator.** Google encrypts every storage chunk with its own DEK, wrapped by per-service KEKs in a central Keystore, with per-operation ACLs and logging [G2]. Notion's Enterprise Key Management (beta, select Enterprise customers) uses a customer AWS KMS key to wrap workspace keys, which wrap per-content DEKs. Revoking the key makes the data "completely inaccessible" [NO2]. Microsoft 365 Customer Key lets the customer control "root encryption keys at the application level", but the customer still "explicitly authorize[s] Microsoft 365 to use your encryption keys" for search indexing, eDiscovery and similar [M1]. Search and other features keep working because the service can still decrypt.
- **E2EE in mainstream apps is opt-in and narrow, or a paid/enterprise tier.** Apple Notes is E2EE only with Advanced Data Protection (ADP, opt-in) or for individually locked notes [AP1][AP2]. Google Workspace Client-side encryption covers only Enterprise Plus, Education Standard/Plus and Frontline Plus [G3]. Evernote encrypts *selected text* only, from desktop or web [E2]. OneNote password-protects *sections* [M2]. Notion has no E2EE at all; EKM is (b) [NO2].
- **The privacy-first notes apps make E2EE the default:** Standard Notes (always) [SN1][SN2], Obsidian Sync (default, with "standard encryption" as an option) [OB1], Day One (default for new journals, per the earlier file). Bear [BE1], Penzu [PZ1] and Joplin [JO1] offer it per note or journal, or as an opt-in.
- **What E2EE costs, per these vendors' own docs:**
  - **No password recovery:** Standard Notes [SN2], Obsidian [OB1], Evernote [E2b], OneNote [M2], Penzu [PZ1], Joplin [JO1], Bear [BE1], Apple ADP (recovery contact or key instead) [AP1].
  - **No server-side content search:** Google CSE files are searchable by "metadata, such as title and owner" but not content [G4]; OneNote's protected sections "are not included in notebook searches" [M2].
  - **Reduced web access:** with ADP, iCloud.com access is off by default. Turning it on gives Apple "temporary access to data-specific encryption keys" [AP1].
  - **Weaker sharing and collaboration:** Google CSE allows one editor at a time and no comments on Drive files [G3b]; Apple shared/collaborative features fall back to Apple-held keys [AP1].
  - **Metadata still leaks:** dates in Apple Notes [AP1][AP2], titles and tags in Bear [BE1], file metadata and deterministic hashes in Obsidian [OB1].
- **A notable reversal:** Reflect, the daily-notes app, built E2EE into its web and desktop app (XChaCha20-Poly1305, key from the user's password [RF1]). In July 2026 it rebuilt as "Reflect Open". It cited E2EE and its own sync engine as "a great deal of complexity". The new app stores local Markdown and uses iCloud for sync and encryption [RF2].
- **For Zoop on Railway:** (i) at-rest is what Zoop has. (ii) Envelope encryption with a key outside the DB protects against DB-only leaks (dumps, PITR archives, SQL injection reading rows). It doesn't protect against the host or the app process, and it collides with the STORED tsvector. (iii) Browser E2EE gives the honest "we can't read it" claim. It moves search, Titles, glimpses and migrations into the browser and makes password reset destroy data unless there is a recovery key. Reflect's own docs point out a web-specific weakness: the server serves the JavaScript that does the encrypting [RF1]. See §5.

## Findings

### 1. Notion

- **Default:** "encrypted in transit (TLS 1.2 or higher) and at rest (AES-256) with keys rotated regularly." [NO1] The Security Exhibit: "Customer Data stored in production systems is encrypted at rest using industry‑standard algorithms (currently including AES‑256 for primary data stores and backups)" [NO4]. Neither page says whether this is disk-level or app-level; it is provider-held keys either way. Model **(a)**, possibly (b) internally **(unconfirmed)**.
- **Enterprise Key Management (beta, select Enterprise customers):** customers grant Notion access to an AWS KMS Customer Managed Key. "Notion uses industry-standard envelope encryption: the customer's CMK is used to encrypt a set of Notion-managed Workspace Encryption Keys (WEKs), which in turn are used to encrypt Data Encryption Keys (DEKs) protecting each piece of content." Customers can "unilaterally revoke CMK access", rotate keys, and audit use in CloudTrail [NO2]. This is textbook model **(b)**. Notion still decrypts to serve, search and run AI features while the key is granted.
- **Staff access:** "Notion support can only access your account or workspace data to troubleshoot an issue or recover content after you contact support and grant access". Access lasts 28 days or until revoked [NO3]. This is a permission control, so Notion can technically read the data. Enterprise owners also get "Content search" across members' private pages [NO2].
- **No E2EE offering found** on the pages read.

### 2. Google Docs / Drive / Workspace

- **Default:** "All files uploaded to Drive or created in Docs, Sheets, and Slides are encrypted in transit and at rest with AES256 bit encryption." [G1]
- **How (model (b) inside Google's storage layer):** "Data is broken into logical chunks … Each chunk is encrypted at the storage level with an individual data encryption key (DEK): two chunks won't have the same DEK". An updated chunk gets a new key. "DEKs are encrypted with (wrapped by) a key encryption key (KEK), using a technique known as envelope encryption. These KEKs are not specific to customers; instead, one or more KEKs exist for each service." KEKs live in Keystore, aren't exportable, and every use is "authenticated and logged" [G2]. There is also a separate device-level AES-256 layer [G2]. The stated threat model: "An attacker … would need to know and be able to access two things: all of the storage chunks … and all of the encryption keys that correspond to the chunks" [G2]. It mitigates storage-media theft, cross-system blast radius and unaudited insider access to raw storage. It doesn't stop Google services that are authorised to decrypt.
- **Staff access:** Workspace offers "Access Transparency … logs of actions taken by Google staff when accessing user content", plus "Access Approvals" and "Access Management" to limit it [G5]. Staff access is possible and logged.
- **Client-side encryption (CSE), model (c):** "end-to-end encryption that Google servers and third parties can't decrypt". Editions: Frontline Plus, Enterprise Plus, Education Standard and Education Plus [G3]. The admin enables it and the keys live in the organisation's external key service [G3][G4]. These are org-held keys, not per-user passphrases.
  - **Costs:** "Only one person can edit an encrypted file at a time" [G3b]. Not available: commenting on Drive files, mobile editing of Docs/Sheets/Slides, grammar check, translate, voice typing, add-ons, downloading Docs/Slides, Apps Script [G3b]. Spellcheck runs on-device [G4]. A 100 MB file cap [G3b].
  - **Search:** in Vault, CSE files can be searched "by their metadata, such as title and owner. However, you can't search their content" [G4].

### 3. Apple Notes

- **Standard data protection (default):** "Your iCloud data is encrypted in transit and stored in an encrypted format at rest. The encryption keys from your trusted devices are secured in Apple data centers, so Apple can decrypt your data on your behalf". Notes is listed with key holder "Apple" [AP1]. Model **(a)/(b)** with Apple-held keys.
- **Advanced Data Protection (opt-in, iOS 16.2+):** Notes becomes end-to-end encrypted, with keys on trusted devices only [AP1]. Costs:
  - Apple can't help with recovery; the user must set up a recovery contact or recovery key first [AP1].
  - iCloud.com access is "disabled by default". Turning it on lets "the web browser that you're using and Apple to have temporary access to data-specific encryption keys" [AP1].
  - Collaboration and "anyone with the link" features upload keys to Apple [AP1].
  - Some metadata (modification dates, checksums) stays under standard protection [AP1].
- **Locked notes (per note, any tier):** "end-to-end encrypted using a user-provided passphrase". PBKDF2-SHA256 derives a 16-byte key, and the note and attachments are encrypted with AES-GCM [AP2]. Resetting a forgotten passphrase lets the user create new locked notes but "won't allow them to see previously secured notes" [AP2]. Passphrase-locked notes can't be shared [AP2] (by implication: "Notes that aren't end-to-end encrypted with a passphrase can be shared"). Search inside locked notes was not covered on the page **(unconfirmed)**.

### 4. Evernote

- **Default:** hosted on Google Cloud, using "Google's server-side encryption feature with Google-managed encryption keys to encrypt all data at rest using AES-256" [E1]. Model **(a)**, inheriting Google's (b) storage layer.
- **Staff access:** the privacy policy covers customer-support access "when you give customer support permission to access your Content". Content review for product improvement happens "only with your express permission and is subject to strict confidentiality rules and data access controls". Systems "may analyze … the notes you share" for spam and malware [E3]. Evernote can technically read the data.
- **Opt-in (c), narrow:** "While you can encrypt the text content within a note, you cannot encrypt an entire note or notebook". This works only in Evernote for Mac, Windows or Web [E2]. AES-128-CBC, key from PBKDF2 (HMAC-SHA256, 50,000 rounds). "Evernote never receives a copy of this key or your passphrase … if you forget your passphrase, your data cannot be recovered." [E2b]

### 5. Microsoft OneNote (Microsoft 365)

- **Default:** "Your Microsoft 365 data is always encrypted at rest using BitLocker and Distributed Key Manager (DKM)" [M1]. Exchange mailboxes are "encrypted using Microsoft-managed keys" by default [M1]. Model **(a)** plus service-level encryption with Microsoft-held keys. OneNote notebooks live in SharePoint/OneDrive, so the SharePoint/OneDrive coverage applies **(unconfirmed for OneNote specifically)**.
- **Customer Key (enterprise, licensed):** an added layer for Exchange, SharePoint, OneDrive and Teams. It lets the customer "control the root encryption keys at the application level". The customer "explicitly authorize[s] Microsoft 365 to use your encryption keys to deliver value-added services like eDiscovery, anti-malware, anti-spam, and search indexing" [M1]. Model **(b)**, the same shape as Notion EKM.
- **Password-protected sections, model (c)-like, per section:** "If you forget your password, you will not be able to unlock your notes. Not even Microsoft Technical Support could unlock your notes." "Password-protected sections are not included in notebook searches." They can't be used in live sharing, and audio/video can't be protected [M2].

### 6. Standard Notes

- **Always E2EE (every tier, including free):** "Only you have access to the keys required to decrypt your data" [SN3]. The design treats the server as "a non-trustworthy entity", with a versioned, audited encryption specification (004) [SN1].
- **Recovery:** "we, or anyone else, cannot read the contents of your notes, nor do we know what your encryption key is. For this reason, password resets are simply not possible." The only option is to delete the account and start over [SN2].
- **Web access** is kept: there is a "Go to web app" link, so decryption happens in the browser [SN1].
- **Search:** title-only and in-note search options exist [SN4]. They must run on the client over decrypted notes, since the server holds only ciphertext **(unconfirmed; not stated outright)**.

### 7. Obsidian Sync

- **E2EE is the default**, with "Standard encryption", which "uses an encryption key managed by Obsidian", as the alternative [OB1]. scrypt key derivation and AES-256-GCM. "No one — not even the Obsidian team — can access your notes." [OB1]
- Obsidian describes standard encryption as "the same method of encryption used by … Google Docs, Dropbox, and iCloud (without Advanced Data Protection)". With it, "the encryption key is stored on company servers, it can be used to decrypt your data, e.g. in a case where the company is subject to a search warrant, or in a case where you want to access your data via a web browser." [OB1] This is a vendor stating the (a)/(b) vs (c) distinction plainly.
- **Costs:** a lost password means the remote vault is "encrypted and unusable forever" [OB1]. File hashes are encrypted deterministically, for deduplication. Uploader device, timestamps and the path-to-content mapping are not E2EE [OB1]. The local vault isn't encrypted, and search is local because Obsidian is a local-files app **(search location unconfirmed on the page read)**.

### 8. Penzu (online journal)

- **Two lock types:**
  - "Basic Locking" is a journal password that can be reset by email, so it is an access control, not encryption.
  - "Encryption Lock is available with a paid Penzu plan. It encrypts the journal's entries using the journal password. If you lose an Encryption Locking password, the journal cannot be unlocked and its contents cannot be recovered." [PZ1]
- Marketing says "double password protection and military strength encryption" for Pro [PZ2]. The default (unlocked or Basic-locked) storage model and staff access aren't documented on the pages read **(unconfirmed; presumably (a))**.

### 9. Reflect (daily notes)

- **Original Reflect:** "We encrypt all your notes on the client with a cypher called XChaCha20-Poly1305 before sending them to our servers. We use the password you supplied us as the key … The data stored on our servers is an encrypted blob which we can't read." [RF1] The app had a web version. Recovery only works while still logged in on some device: change password, or generate a recovery kit on iOS [RF1]. AI and Whisper features send plaintext to OpenAI [RF1]. Reflect also admitted the web threat model: "is this encryption really secure? You could just edit the JavaScript anytime … this is where we've chosen to draw the line." [RF1]
- **Reflect Open (July 2026):** "we built end-to-end encryption into the app, and created our own note format and sync engine … they also created a great deal of complexity and made Reflect increasingly difficult to evolve". The new app "stores your notes as ordinary Markdown files on your computer, uses iCloud for sync and encryption" [RF2]. The old app is staying for now [RF2]. Encryption for Reflect Open therefore follows the user's iCloud setting, standard or ADP **(inference)**.

### 10. Bear

- **Sync is through Apple CloudKit:** "Every piece of information stored within CloudKit is encrypted with Apple's private keys, and we don't have access to users credentials or any sensitive data." With ADP on, "only you … can decrypt your data". Without ADP it is Apple-held keys [BE2][AP1]. Bear itself runs no content server.
- **Bear Pro note encryption (per note):** "your Note Password is known only to you—we cannot see or reset it". Encrypted notes show only titles in the list [BE1]. Bear says note titles and tag names stay unencrypted under ADP so Pro features work [BE2].
- **Bear Web:** decrypts in the browser from iCloud. On trusted devices it keeps "an encrypted copy of your notes in that browser" [BE3].

### 11. Joplin (extra comparable, open-source notes)

- **Opt-in E2EE** across all apps. "Even the developers of Joplin" can't access the data. A Master Key is protected by the user's password, and the password "cannot be recovered" [JO1]. It works over arbitrary sync targets (OneDrive, Nextcloud, Joplin Server) [JO1]. Search is local, because each client keeps a full local copy **(unconfirmed on the page read)**.

### 12. Comparison table

| App | Default model | Stronger option (tier) | Provider can technically read default? | What the E2EE option gives up |
|---|---|---|---|---|
| **Notion** | (a) AES-256 at rest, Notion-held keys [NO1][NO4] | (b) EKM: customer AWS KMS key wrapping per-content DEKs; Enterprise beta [NO2] | Yes. Support access is gated by user/admin consent, 28 days [NO3] | No E2EE offered |
| **Google Docs/Drive** | (b) per-chunk DEKs + per-service KEKs in Keystore, Google-held [G1][G2] | (c) Workspace CSE with org key service; Enterprise Plus, Edu Std/Plus, Frontline Plus [G3] | Yes. Staff access logged via Access Transparency [G5] | Content search (metadata only) [G4]; one editor at a time, no Drive comments, no mobile editing, no add-ons [G3b] |
| **Apple Notes** | (a)/(b) Apple-held keys; "Apple can decrypt" [AP1] | (c) ADP (opt-in, all accounts) [AP1]; locked notes with passphrase (any account) [AP2] | Yes under standard [AP1] | ADP: no Apple recovery, iCloud.com off by default, shared features revert to Apple keys [AP1]. Locked notes: reset loses old notes, can't share [AP2] |
| **Evernote** | (a) Google-managed keys, AES-256 [E1] | (c) encrypt *selected text*, desktop/web, all tiers [E2] | Yes, with "express permission" policy [E3] | No recovery [E2b]; whole notes can't be encrypted [E2] |
| **OneNote / M365** | (a) BitLocker/DKM + service encryption, Microsoft-managed keys [M1] | (b) Customer Key, enterprise [M1]; (c)-like password-protected sections [M2] | Yes (inference from Microsoft-managed keys) | Protected sections excluded from search, no live sharing, no recovery [M2] |
| **Standard Notes** | (c) always, all tiers [SN1][SN3] | — | No [SN2] | No password reset [SN2]; search on client (unconfirmed) |
| **Obsidian Sync** | (c) E2EE default; "standard" (Obsidian-held key) optional [OB1] | — | No with E2EE; yes with standard [OB1] | No recovery; some metadata and deterministic hashes visible [OB1] |
| **Penzu** | Not documented (unconfirmed, likely (a)) | (c)-like Encryption Lock per journal, paid [PZ1] | Not documented | No recovery; no auto-lock [PZ1] |
| **Reflect (original)** | (c) client-side XChaCha20-Poly1305, password key [RF1] | — | No (but the server ships the JS) [RF1] | Recovery only from a logged-in device [RF1]; later abandoned for local files + iCloud [RF2] |
| **Bear** | Apple CloudKit, Apple-held keys unless ADP [BE2] | (c) per-note password, Bear Pro [BE1]; ADP [BE2] | Bear: no server. Apple: yes without ADP [AP1] | Titles/tags stay plaintext [BE1][BE2]; no reset [BE1] |
| **Joplin** | Plain on user's sync target | (c) opt-in E2EE, free [JO1] | Depends on sync target | No recovery [JO1] |
| **Day One** (earlier file) | (c) E2EE default for new journals | — | No (per Day One) | See [encryption-at-rest.md](encryption-at-rest.md) §5 |

### 13. Envelope encryption (b) vs E2EE (c)

- **(b)** answers "what if someone gets the bytes without going through our service?" Examples: a stolen disk, a backup copy, a storage bug, or an engineer reading raw storage outside the logged key path. Google's per-chunk DEKs limit "a potential data encryption key compromise to only that data chunk", and every KEK use is logged [G2]. A customer-managed root key (Notion EKM, M365 Customer Key) adds a revocation switch and customer-side audit [NO2][M1]. The service still decrypts on every request, so search, previews, AI and support access all keep working. The provider can read the content whenever its own code path is authorised.
- **(c)** answers "what if the provider itself, a subpoena or a full server breach is the threat?" Obsidian frames it exactly this way [OB1]. It costs server-side features (search, indexing, collaboration) and recovery.
- None of the mainstream apps above (Notion, Google, Microsoft, Evernote) makes (c) the default. Apple comes closest with opt-in ADP. Default E2EE is found in privacy-branded or small apps (Standard Notes, Obsidian Sync, Day One, original Reflect).

## 5. Realistic options for Zoop on Railway

**(i) At-rest only (current).** Railway AES-256 with Railway-held keys; details in [encryption-at-rest.md](encryption-at-rest.md) [R1 there]. This is the same posture as Notion's or Evernote's default. Honest copy: "encrypted in transit and at rest"; never "we can't read it".

**(ii) Server-side envelope encryption in the Go API.**
- *Shape:* a per-user or per-Page DEK encrypts `doc` / `plain_text`. The DEK is wrapped by a KEK held either in a Railway env var or in an external KMS (AWS/GCP KMS), with the wrapped DEK stored next to the row. This mirrors Google [G2] and Notion EKM [NO2] at small scale.
- *Protects against:* DB-only exposure. That covers a leaked Postgres credential, SQL injection reading rows, an exfiltrated PITR archive or volume backup, and a leaked unencrypted dump. An external KMS adds CloudTrail-style audit and a revoke switch, and keeps the KEK out of Railway's variable store **(inference)**.
- *Doesn't protect against:* Railway insiders or the control plane (with an env-var KEK, Railway decrypts variables to run the service, per the earlier file [R1 there]), a compromised Go process, or anyone with deploy access. An external KMS doesn't change this much, because the running app must be able to call it **(inference)**. Postgres itself warns that server-side keys "can be intercepted by someone with complete access to the database server" [P1 there].
- *Collides with:*
  - The STORED tsvector, which is a near-copy of the text, unless search moves to a blind index or is dropped (see the earlier file §4).
  - Titles and glimpses derived in SQL, which would have to move to Go.
  - JSONB migrations as SQL `UPDATE`s, which would become Go batch jobs.
  - The `plain_text <> ''` partial index, which would need a plaintext flag.
  - Value is real but narrow, and only meaningful if backups are treated as a likely leak path **(inference)**.

**(iii) Browser-side E2EE with a passphrase.**
- *Search:* Postgres full-text search can't see content. The browser must download and decrypt Pages and search locally, as in Google CSE's "can't search their content" [G4] and OneNote's protected sections [M2]. A blind index leaks word frequency (earlier file §4) **(unconfirmed technique)**.
- *Titles and glimpses:* must be derived in the browser. They are either stored as extra ciphertext or computed on load. Day and date metadata likely stays plaintext, as Apple [AP1], Obsidian [OB1] and Journey (earlier file) leave dates visible.
- *Password reset:* account-password reset can't recover content. That needs a recovery key or kit (Apple ADP [AP1], Reflect [RF1]), or reset means deleting the data (Standard Notes [SN2]). The login password and the encryption passphrase must be decoupled, or a login reset has to warn that it wipes the journal **(inference)**.
- *Web-specific caveat:* the server delivers the encrypting JavaScript, so a compromised or malicious deploy could exfiltrate the passphrase. Reflect acknowledges this [RF1]. It is weaker than native-app E2EE **(inference beyond Reflect's statement)**.
- *Market signal:* Reflect dropped its own E2EE because of the complexity it caused [RF2]. Mainstream apps keep E2EE opt-in [AP1][G3].

## Sources

- [NO1] Notion, Security — https://www.notion.com/security
- [NO2] Notion Help, Notion Enterprise security provisions (incl. Enterprise Key Management, Content search) — https://www.notion.com/help/guides/notion-enterprise-security-provisions
- [NO3] Notion Help, Allow or remove Notion support access — https://www.notion.com/help/data-access-consent
- [NO4] Notion Security Exhibit — https://notion.notion.site/Security-Exhibit-306efdeead0580b1b316ff746169a11f (redirected from https://www.notion.com/help/security-and-privacy)
- [G1] Google Drive Help, Get started with encrypted files in Drive, Docs, Sheets & Slides — https://support.google.com/drive/answer/10519333
- [G2] Google Cloud, Default encryption at rest — https://docs.cloud.google.com/docs/security/encryption/default-encryption
- [G3] Google Workspace Admin, About client-side encryption — https://knowledge.workspace.google.com/admin/security/about-client-side-encryption
- [G3b] Same as [G1], sections "Find what's different" / "Find what's not available for encrypted files"
- [G4] Google Workspace Admin, Client-side encryption FAQ — https://knowledge.workspace.google.com/admin/security/client-side-encryption-faq
- [G5] Google Workspace Admin, Access Transparency: View logs on Google access to user content — https://knowledge.workspace.google.com/admin/security/access-transparency-view-logs-on-google-access-to-user-content
- [AP1] Apple Support, iCloud data security overview — https://support.apple.com/en-us/102651
- [AP2] Apple Platform Security, Secure features in the Notes app — https://support.apple.com/guide/security/secure-features-in-the-notes-app-sec1782bcab1/web
- [E1] Evernote, Security — https://evernote.com/security
- [E2] Evernote Help, Encrypt text in a note — https://help.evernote.com/hc/en-us/articles/209005547-Encrypt-text-in-a-note
- [E2b] Evernote Help, What type of encryption does Evernote use? — https://help.evernote.com/hc/en-us/articles/208314128-What-type-of-encryption-does-Evernote-use
- [E3] Evernote, Privacy Policy — https://evernote.com/privacy/policy
- [M1] Microsoft Learn, Overview of Service encryption with Microsoft Purview Customer Key — https://learn.microsoft.com/en-us/purview/customer-key-overview
- [M2] Microsoft Support, Protect your notes with a password (OneNote) — https://support.microsoft.com/en-us/onenote/onenote-help-and-learning/protect-your-notes-with-a-password
- [SN1] Standard Notes Help, Security (encryption versions, strict sign-in) — https://standardnotes.com/help/security
- [SN2] Standard Notes Help, I've forgotten my password — https://standardnotes.com/help/6/i-ve-forgotten-my-password-what-should-i-do
- [SN3] Standard Notes, home page — https://standardnotes.com/
- [SN4] Standard Notes Blog, Announcing smarter search, tags, and more — https://standardnotes.com/blog/search-improvements
- [OB1] Obsidian Help, Sync security and privacy — https://obsidian.md/help/sync/security
- [PZ1] Penzu Support, How to lock your journal, and locking settings explained — https://support.penzu.com/articles/75753-how-to-lock-your-journal-and-locking-settings-explained
- [PZ2] Penzu PRO — https://penzu.com/pro
- [RF1] Reflect Academy, Security and encryption — https://reflect.academy/security-and-encryption
- [RF2] Reflect Blog, Reflect Open – a new chapter for Reflect (2026-07-14) — https://reflect.app/blog/reflect-open
- [BE1] Bear FAQ, How to encrypt & lock notes with Bear — https://bear.app/faq/how-to-encrypt-lock-notes-with-bear/
- [BE2] Bear FAQ, Information about Syncing & Privacy — https://bear.app/faq/syncing-privacy/
- [BE3] Bear FAQ, Bear Web – Sync and Trusted Devices — https://bear.app/faq/bear-web-sync-and-trusted-devices/
- [JO1] Joplin Help, End-To-End Encryption (E2EE) — https://joplinapp.org/help/apps/sync/e2ee/
- Railway [R1], PostgreSQL [P1] and Day One/Journey sources: see [encryption-at-rest.md](encryption-at-rest.md)
