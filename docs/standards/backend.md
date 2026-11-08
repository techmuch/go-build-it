# Backend stack

- Go (latest stable), standard library first: `net/http` with method patterns, `log/slog`, `embed`, `database/sql`.
- SQLite through a pure-Go driver (`modernc.org/sqlite`) so builds need no C toolchain and cross-compile cleanly.
- Packages live under `internal/`; one package per domain concept (projects, queue, harness, …).
- Every state change goes through a service that also writes a history event (NFR-10).
- Errors are wrapped with context (`fmt.Errorf("…: %w", err)`); no panics outside `main`.
- No global mutable state; dependencies are passed in through constructors.

## Why
A small dependency surface keeps the single binary portable and makes agents' work predictable.
