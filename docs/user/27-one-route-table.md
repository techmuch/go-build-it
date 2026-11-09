section: How go-build-it thinks
reqs: FR-30, FR-31, FR-32, FR-33, FR-138
# One route table: UI, CLI, API and MCP

**In short:** go-build-it's API is defined once, in a single route table. The web UI, the CLI, the OpenAPI spec, the TypeScript client and the MCP server are all generated from it, so they cannot drift apart.

## Why

go-build-it is meant to be driven by AI agents as much as by people (it is AI-native). If the CLI lagged behind the UI, an agent could not do what you can. So:
- Every route has a method, a path, a CLI name (`tasks add`) and its flags.
- `gbi help` lists every command, and `gbi <group> <command> --json` returns what the UI sees.
- `gbi mcp` exposes the same operations as MCP tools, one per route.
- The web UI calls the same HTTP API through a generated client.

## What it means for you

Anything you can click, your AI harness can do. See *Drive gbi from your AI harness*. Commands that work without the server, such as `gbi doctor` and `gbi agent guide`, are the exceptions, because they must work when the server is down or not yet installed.

Decisions: [ADR-001](https://github.com/techmuch/go-build-it/blob/main/docs/decisions/ADR-001-ui-approach.md), [ADR-012](https://github.com/techmuch/go-build-it/blob/main/docs/decisions/ADR-012-doctor-and-learning.md).
