# The backend is hexagonal: the rules never touch HTTP or the database

The backend keeps the application's rules apart from the code that talks to the outside world: HTTP, Postgres, email, the wall clock, and anything added later (queues, Google, Resend). The rules live in **core** packages. Each one declares the **ports** (Go interfaces) it needs. **Adapters** sit at the edge and implement or drive those ports. Dependencies point inward only: an adapter imports the core, and the core never imports an adapter.

- **Core:** `internal/auth`, `internal/auth/password`, `internal/days`, and later Pages, Search and the rest. The core holds the vocabulary from `GLOSSARY.md`, the rules (who may write which Day, session expiry, validation) and its ports, for example `auth.Store`. It must not import `net/http`, chi, pgx, `database/sql`, `internal/db` or any adapter.
- **Driven adapters** (the core calls out through them): `internal/postgres` implements the store ports with sqlc (`internal/db/dbgen`) and owns transactions. The Mailer (Resend, and the in-memory fake), the Clock (`clock.System`, `clock.Fake`) and, later, the rate limiter are adapters too.
- **Driving adapters** (they call the core): `internal/server` turns HTTP requests into core calls and core results and errors into JSON, cookies and status codes. It never queries the database itself.
- **Composition root:** `internal/app` builds the adapters, hands them to the core, and returns the HTTP handler. `cmd/zoop` and the tests both build the app through it.

`internal/archtest` fails the build if a core package imports anything on the forbidden list. Add each new core package to its list.

A port is shaped by what the core needs, in domain terms ("create this account and its first session together"), not by the tables. That keeps transactions inside the adapter. The main tests stay at the HTTP seam against a real Postgres, as the spec says. Ports exist to keep the rules separate, not so that the database can be mocked, so don't write a fake store for tests the real one can run. One exception is allowed: `/api/healthz` exists to check infrastructure, so it gets a plain ping function from the composition root rather than a core port.
