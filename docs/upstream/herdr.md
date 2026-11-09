# herdr review log

## 2026-11-08 · `ee5bfb5` · 0.9.5 (baseline)

This is the baseline review done for ADR-009. go-build-it depends on the following:

| Area | herdr feature | go-build-it use |
|---|---|---|
| Sessions | `herdr --session gbi server`, socket `sessions/gbi/herdr.sock` | the only session go-build-it uses (FR-07) |
| Layout | `workspace.create`, `worktree.open`, `tab.create`, `layout` | one workspace per project, member worktrees grouped under it (FR-67) |
| Agents | `agent.start`, `agent.prompt` (with `wait`), `agent.wait`, `agent.send_keys`, `agent.get` (`agent_session`) | launch, dispatch, interrupt, conversation ID (FR-60, FR-61, FR-69) |
| State | `events.subscribe` (`pane.agent_status_changed`, `pane.exited`, `pane.created`), `session.snapshot`, `events_lost` recovery | the source of truth for member state (FR-68) |
| Labels | `pane.report_metadata`, `workspace.report_metadata`, `notification.show`, `agent.view.set` | what the user sees in herdr (FR-67) |
| Terminal | `terminal session observe` and `control` (stdio NDJSON: `terminal.frame`, `terminal.input`, `resize`, `release`) | live terminal in the browser (FR-66) |
| Integrations | `herdr integration status/install` | conversation identity and restore (FR-07) |
| Schema | `herdr api schema --json` | generated Go types (NFR-32) |

Findings that affect go-build-it:
- **Session isolation.** `herdr server` ignores `HERDR_SESSION`. Always pass `--session` or `HERDR_SOCKET_PATH`, and check the socket before writing. See the ADR-009 spike.
- **Long prompts on macOS.** Input lines over the terminal's canonical limit are rejected, so go-build-it sends prompts as a short "read `.gbi-task.md`" line.
- **Restart behaviour.** Pane IDs change across restarts and tokens are not restored, so go-build-it joins on the conversation ID.
- **Stalled observers.** Observers that make no progress for 30 s are disconnected, so the terminal bridge must drain promptly.
- **Windows.** Windows is in beta, and `terminal attach` is unsupported there (issue #4821). Check `terminal session observe` on Windows in CI (P11-07, P15-04).
