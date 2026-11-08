# Testing

- Unit tests with the Go `testing` package; table-driven where it helps. Coverage target ≥ 80% for `internal/`.
- Adapter parsers are tested against recorded fixtures in `testdata/`.
- API handlers are tested with `httptest` against a temporary SQLite database.
- End-to-end tests use Playwright with the installed Chrome (`channel: "chrome"`); each spec saves named screenshots.
- `make test` runs everything; CI runs it on every push.
