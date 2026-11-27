## Agent skills

### Issue tracker

GitHub Issues on `boinkkitty/zoop`, via the `gh` CLI. The `.scratch/` folder is an archive of the journal-v1 map. See `docs/agents/issue-tracker.md`.

### Triage labels

The five default roles (needs-triage, needs-info, ready-for-agent, ready-for-human, wontfix), applied as GitHub labels. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `GLOSSARY.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.

## Backend

- **Architecture:** hexagonal. Keep the rules in core packages, separate from the code that talks to HTTP, databases, queues and other external systems. See `docs/adr/0002-hexagonal-architecture.md`.
- **Migrations:** not run from `main` (`cmd/zoop`). The user runs them manually with `pnpm migrate` (`go run ./cmd/migrate`, needs `DATABASE_URL`).

## Pull Requests

### Title

Use a semantic title: `type(scope): short description`.

### Description

Keep the description brief and answer:

- What changed?
- Why was it needed?
- How was it tested?

Link relevant issues. For performance changes, include before-and-after measurements.
