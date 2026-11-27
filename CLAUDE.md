## Agent skills

### Issue tracker

GitHub Issues on `boinkkitty/zoop`, via the `gh` CLI. The `.scratch/` folder is an archive of the journal-v1 map. See `docs/agents/issue-tracker.md`.

### Triage labels

The five default roles (needs-triage, needs-info, ready-for-agent, ready-for-human, wontfix), applied as GitHub labels. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `GLOSSARY.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.

## Backend

- **Architecture:** hexagonal. Keep the rules in core packages, separate from the code that talks to HTTP, databases, queues and other external systems. See `docs/adr/0002-hexagonal-architecture.md`.
- **Migrations:** plain SQL files in `internal/db/migrations/` (`NNNNN_name.sql`, up only, no tool). The user runs them by hand with `psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -1 -f <file>`. Write them so applying twice is harmless (`CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`, never `DROP`). Nothing in the app or scripts applies them. The only exceptions are throwaway test databases: `testenv` applies them all for `go test`, and CI's e2e job applies them with the same `psql` command.

## Pull Requests

### Title

Use a semantic title: `type(scope): short description`.

### Description

Keep the description brief and answer:

- What changed?
- Why was it needed?
- How was it tested?

Link relevant issues. For performance changes, include before-and-after measurements.
