# Notes for AI agents

- **Installing go-build-it for someone?** Follow [README.md → Install with an AI agent](README.md#install-with-an-ai-agent). It is a numbered procedure with rules, and it ends with `gbi doctor --json --fix` repeated until the exit code is 0. Exit code 2 means a step needs the person.
- **Driving an installed go-build-it?** Run `gbi agent guide`. It prints the workflow, the commands and the rules (ask before starting paid work), and you can save it as a skill. Every UI action has a CLI command (`gbi help`). `gbi mcp` is an MCP server with the same operations.
- **Something broken?** `gbi doctor --report -` prints a Markdown report with the evidence for every check. Each check id is explained in [docs/user/20-troubleshooting.md](docs/user/20-troubleshooting.md).
- **Working on go-build-it's own code?** The requirements are in [docs/requirements.md](docs/requirements.md), the decisions in [docs/decisions/](docs/decisions/), and the plan in [docs/tracker/](docs/tracker/plan.md). `make test` runs vet, the type check and the Go tests.
