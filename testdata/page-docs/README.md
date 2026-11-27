# Page document fixtures

Shared by the Go validator tests (`internal/pages`) and the client schema tests (`web/src/page`), so the server and the editor agree on what a Page document is.

- `schema-version.json`: the current `schemaVersion`. Both sides' constants must equal it.
- `valid/*.json`: `{"notebook", "doc", "plainText"}`. The doc must be accepted, and `plainText` is what the server derives from it.
- `invalid/*.json`: `{"notebook", "doc", "why"}`. The doc must be refused.

When the schema changes, bump `schema-version.json` and both constants together.
