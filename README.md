# go-build-it

go-build-it manages long-running software projects with a team of AI personas and humans. It is one Go binary with an embedded React UI (desktop and mobile), a CLI with the same operations, and an MCP server. Agent harnesses (Claude Code, OpenCode, Antigravity, Pi) run headlessly in isolated git worktrees.

- Requirements: [docs/requirements.md](docs/requirements.md)
- Delivery plan and tracker: [docs/tracker/](docs/tracker/index.html)
- Decisions: [docs/decisions/](docs/decisions/)
- User docs (also inside the app): [docs/user/](docs/user/)

## Build and run

```
make build                  # web UI + binary → bin/gbi
./bin/gbi serve             # http://127.0.0.1:7420, state in ~/.go-build-it
./bin/gbi help              # every UI action has a CLI command
```

## Test

```
make test                                                   # go vet, tsc, Go tests (incl. end-to-end workflow with a scripted agent)
GBI_HOME=$(mktemp -d) PORT=7431 scripts/demo.sh             # seed a demo via the CLI (no AI subscription used)
cd web && GBI_E2E_URL=http://127.0.0.1:7431 npx playwright test   # UAT walkthrough, regenerates docs screenshots
```

## Release

```
VERSION=2026.11.08.1 make release   # cross-compiled, SHA256SUMS signed with Ed25519 (ADR-004)
```
