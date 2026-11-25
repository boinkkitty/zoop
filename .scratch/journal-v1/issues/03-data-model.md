# Data model

Type: grilling
Status: open
Blocked by: 02, 09

## Question

What are the Postgres tables and constraints for v1? Cover:
- users: plan flag, profile timezone, and how a timezone change "from the next Day" is stored
- linked auth identities: Google and password
- sessions: hashed opaque token, sliding expiry
- email verification and reset tokens
- Notebooks: fixed system rows vs per-user rows, given that more will come later
- Pages: unique per user + Notebook + Day, plain text, `updated_at`
- the search index column
- how revisions could be added later without migrating existing Pages
