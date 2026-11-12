# Go backend

How a project's Go backend is built: one statically linked binary whose HTTP API, CLI and background work all call the same services.

- Every rule has an ID. Review findings and project overrides cite it (`GO-33`).
- **SHALL** rules block a merge. **SHOULD** rules may be skipped with a reason in the pull request.
- A project that needs something different overrides the rule, with a reason. It does not quietly ignore it.

## Stack and dependencies

- **GO-01** SHALL: The latest stable Go, pinned in `go.mod`. CI builds with the same version.
- **GO-02** SHALL: Standard library first: `net/http` with method patterns, `log/slog`, `embed`, `database/sql`, `testing`. No web framework, ORM, dependency-injection container or assertion library.
- **GO-03** SHALL: Pure Go only. Every target platform builds with `CGO_ENABLED=0`, so SQLite is a pure-Go driver (`modernc.org/sqlite`).
- **GO-04** SHALL: A new direct dependency is recorded in an ADR: what it does, why the standard library is not enough, and its licence.
- **GO-05** SHALL: `go mod tidy` changes nothing, and `govulncheck ./...` reports no vulnerability the code can reach.

## Layout and wiring

- **GO-10** SHALL: Code lives under `internal/`, one package per domain concept, named after the concept (`queue`, `reviews`). No `util`, `common`, `helpers` or `models` packages.
- **GO-11** SHALL: `main` only calls `run(args) error` and turns the error into an exit code. Everything else is in packages that tests can call.
- **GO-12** SHALL: Every package has a package comment that says what it is for. Every exported identifier has a doc comment.
- **GO-13** SHALL: Dependencies are passed in through constructors (`New(db, hist, …)`). One function wires every service together, and tests call that same function.
- **GO-14** SHALL: No package-level variable is written after start-up, and no `init()` has side effects. Fixed tables (a list of stages, a compiled regexp) are fine.
- **GO-15** SHALL: A service that must react to another service does it through a hook: a documented function field, set where services are wired, safe when nil. No import cycles, and no global event bus for control flow.
- **GO-16** SHALL: An interface is declared by the package that uses it, where a test needs to replace something real (an external command, the clock, the network). No interface that has one implementation and no test double.
- **GO-17** SHOULD: A file stays under 600 lines and a function under 80. Split by concept, not by size alone.

## State and history

- **GO-20** SHALL: Every state change goes through a service method. Handlers, CLI commands and background loops never write to the database themselves.
- **GO-21** SHALL: Every state change records a history event: who did it, what changed (a dotted `noun.verb` type and the subject's ID), when, and **why**.
- **GO-22** SHALL: The event is written in the same transaction as the change. Where that is not possible, a failed event write is logged at `Error`. It is never dropped silently.
- **GO-23** SHALL: The actor travels in `context.Context`. Middleware sets it for requests. Background work sets it explicitly, as the persona or as the system. Humans and AI members are recorded the same way.
- **GO-24** SHALL: Event type names are a contract, because docs, metrics and tests read them. Renaming one is a breaking change with a change log entry.

## Data

- **GO-30** SHALL: Connection settings (`busy_timeout`, WAL, `foreign_keys`) go in the DSN, so every pooled connection has them. `Open` checks that they took effect.
- **GO-31** SHALL: Schema changes are numbered, forward-only SQL files, embedded in the binary and applied in a transaction at start-up. A merged migration is never edited.
- **GO-32** SHALL: Every query uses `?` placeholders and the `…Context` methods with the caller's context. Input values are never concatenated into SQL.
- **GO-33** SHALL: A change to more than one row or table runs in one transaction: `BeginTx`, then `defer tx.Rollback()`.
- **GO-34** SHALL: Every `Query` has `defer rows.Close()`, and `rows.Err()` is checked after the loop.
- **GO-35** SHALL: Timestamps are UTC RFC 3339 strings from one helper. IDs come from one helper and are never reused or renumbered.
- **GO-36** SHALL: Columns are `NOT NULL` with a default. A column with a fixed set of values lists them in a comment in the migration and as Go constants.
- **GO-37** SHALL: "Not found" is a sentinel error (`ErrNotFound`) wrapped with the ID, and callers test it with `errors.Is`.

## API and CLI

- **GO-40** SHALL: Every operation is declared once, in a route table: method, path, dotted name, CLI words, summary and flags. The HTTP routes, the CLI, the MCP tools, the OpenAPI document and the typed client all come from that table.
- **GO-41** SHALL: Nothing can be done from the UI that cannot be done from the CLI. A test fails when a route has no CLI words.
- **GO-42** SHALL: A handler decodes the request, calls one service method and returns `(value, error)`. One function writes JSON and maps errors to statuses.
- **GO-43** SHALL: Errors carry their status: 400 for bad input, 403 for not allowed, 404 for an unknown ID, 409 for a conflict with the current state. 500 means a defect. The body is `{"error": "…"}`.
- **GO-44** SHALL: JSON fields are camelCase. An empty list is `[]`, never `null`. Secret fields are tagged `json:"-"`.
- **GO-45** SHALL: Request bodies have a size limit (`http.MaxBytesReader`) and the server sets `ReadHeaderTimeout`.
- **GO-46** SHALL: A stream (SSE, WebSocket) ends when the request context ends, and sends keep-alives while it is idle.
- **GO-47** SHALL: Generated files start with `// Code generated … DO NOT EDIT.` and the generator has a check mode that CI runs, so a stale file fails the build.
- **GO-48** SHALL: Every command has `--json` output. A command an agent is expected to loop on documents its exit codes: `0` done, `1` it can retry or fix this itself, `2` a person is needed.
- **GO-49** SHOULD: Routes declare their response type, so the generated client is typed and does not return `any`.

## Errors

- **GO-50** SHALL: An error is wrapped with what was being attempted: `fmt.Errorf("open db: %w", err)`. Lower case, no full stop, no "failed to".
- **GO-51** SHALL: No `panic`, `log.Fatal` or `os.Exit` outside `main` and tests.
- **GO-52** SHALL: No error is discarded silently. `_ =` is allowed for best-effort cleanup, or with a comment that says why ignoring it is safe.
- **GO-53** SHALL: The code tests its own errors with `errors.Is` and `errors.As`, never by matching message text.
- **GO-54** SHALL: A message a person sees says what happened and what to do next.
- **GO-55** SHALL: A problem that an AI agent may be asked to fix is reported as a contract: a stable `id`, the evidence (command, exit code, output), and a fix marked `auto` or `human`. An agent applies `auto` fixes and passes `human` ones to the person word for word.

## Concurrency and lifecycle

- **GO-60** SHALL: `ctx context.Context` is the first parameter of anything that does I/O. A context is never stored in a struct.
- **GO-61** SHALL: `context.Background()` appears only in `main`, in tests, and for work that must outlive the request that started it, with a comment that says so.
- **GO-62** SHALL: Every goroutine has an owner and a way to stop. Constructors start none: a `Start(ctx)` method does, and they end when `ctx` is cancelled. Tickers and timers are stopped.
- **GO-63** SHALL: Shared state is guarded by a mutex declared next to the fields it guards, or is atomic. A value handed out from under a lock is a copy.
- **GO-64** SHALL: A slow subscriber never blocks a publisher: channels are buffered, and a full one drops the message.
- **GO-65** SHALL: SIGINT and SIGTERM cancel the root context. The server shuts down with a deadline, then the database closes.
- **GO-66** SHALL: Work that takes longer than a request survives a restart. Its state is saved as it goes, and start-up resumes or recovers it. State held only in memory is a cache.

## External tools and services

- **GO-70** SHALL: Each external tool or service (git, a forge CLI, another daemon) has one adapter package. Nothing else calls `exec.Command` or that service's API.
- **GO-71** SHALL: Commands run with `exec.CommandContext`, arguments as a slice (never a shell string), an explicit working directory and a timeout. `stderr` goes into the returned error.
- **GO-72** SHALL: The adapter has a seam that tests replace (a `Runner` function type). Its parsers are tested against recorded output in `testdata/`.
- **GO-73** SHALL: For an upstream with a versioned API: types are generated from its published schema, the supported version range is declared in one place, CI pins the same version, and a scheduled CI job runs against the latest release.
- **GO-74** SHALL: An adapter that has not been checked against a real run is marked *unverified* in the code and wherever a user sees it.

## Platforms

- **GO-80** SHALL: The build and the tests pass on macOS, Linux and Windows. Paths use `path/filepath`, the home directory comes from `os.UserHomeDir`, and no shell is assumed.
- **GO-81** SHALL: OS-specific code sits in `_unix.go` and `_windows.go` files with build tags, behind one function with the same signature on every platform.
- **GO-82** SHALL: One function returns the state directory, and one environment variable overrides it. Tests and demos use that variable.

## Security

- **GO-90** SHALL: The server listens on loopback by default. Middleware classifies every request by where it came from. A request that is not local is refused unless the configuration allows it *and* it passes a challenge.
- **GO-91** SHALL: Secrets are compared with `crypto/subtle`. They never appear in logs, API responses, reports, history events or the repository.
- **GO-92** SHALL: A file path taken from a request is cleaned and checked to be inside its root before it is used.
- **GO-93** SHALL: Anything downloaded to be run is verified first: a checksum, and a signature checked against a key compiled into the binary. The previous version is kept for rollback.
- **GO-94** SHALL: An action that costs money or is visible outside the machine (a push, a release, a paid model call) is its own operation and is recorded in the history. It is never a side effect of a read.

## Logging

- **GO-100** SHALL: `log/slog` only, with the logger passed in. The message is a constant and the details are key/value attributes.
- **GO-101** SHALL: `Info` for lifecycle, `Warn` for degraded but running, `Error` for work that failed. Library packages never print: stdout belongs to command output.
- **GO-102** SHALL: The history says what happened in the product. Logs are for whoever operates the binary. Neither stands in for the other.

## Tests

- **GO-110** SHALL: Every package that has logic has tests. Coverage of `internal/` is at least 80%, measured in CI across packages (`-coverpkg`), and a pull request does not lower it.
- **GO-111** SHALL: The standard `testing` package, with table-driven tests where there are several cases, and `t.Helper`, `t.TempDir`, `t.Cleanup` and `t.Setenv`.
- **GO-112** SHALL: The API is tested with `httptest` through the real route table, on a temporary SQLite database with the real migrations. The database is never mocked.
- **GO-113** SHALL: At least one end-to-end test drives the main workflow through the public API. Anything paid or remote is replaced by a scripted fake (a fake agent, a fake release server), so the test needs no credentials and costs nothing.
- **GO-114** SHALL: Tests never touch the user's real state. They use their own home directory, ports and sessions.
- **GO-115** SHALL: A test that needs an optional tool skips with a reason on a developer's machine and **fails** in CI (`…_REQUIRE_<TOOL>=1`). A test that calls a real remote service runs only when an environment variable opts in.
- **GO-116** SHALL: One source of truth, and a test for every copy. When the same value or list is in two places (a key in the README and the code, a version in CI and the code, IDs in the code and the docs), a test fails when they differ.
- **GO-117** SHALL: A bug fix starts with a test that fails without the fix.
- **GO-118** SHALL: A test waits for a condition by polling with a deadline, not with a fixed sleep. Polling intervals are fields that tests shorten.
- **GO-119** SHALL: `go test -race ./...` passes.

## Traceability

- **GO-120** SHALL: Code that implements a requirement names its exact ID in a comment (`FR-71`). Code that follows a decision names the ADR.
- **GO-121** SHALL: Comments say why: the constraint, the decision or the trap that the code cannot show. They do not repeat the code.
- **GO-122** SHALL: A choice between real alternatives is written as an ADR before the code merges.
- **GO-123** SHALL: A change a user can see adds an entry under *Unreleased* in the change log. A new command, setting or check ID updates the user docs in the same pull request.

## Before asking for review

All of these pass, and `make test` and CI run them:

```
gofmt -l .                                   # prints nothing
go vet ./...
go test -race -cover ./...
go mod tidy && git diff --exit-code go.mod go.sum
govulncheck ./...
CGO_ENABLED=0 GOOS=<os> GOARCH=<arch> go build ./cmd/...   # every target platform
```

## Why

- **Predictable for agents.** One route table, one way to wire services and one error style mean an AI member can add an operation by copying the one next to it, and a reviewer can check it rule by rule.
- **Accountable.** Because every change is an attributed event with a reason, a person can find out what an agent did and why without reading its transcript.
- **Cheap to verify.** Scripted fakes and isolated state let people and agents run the whole workflow as often as they like, on any machine, without spending tokens or touching real work.
- **Portable.** A small, pure-Go dependency surface keeps the single binary easy to cross-compile, sign and update.
