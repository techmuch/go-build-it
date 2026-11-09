# Ovis review log

## 2026-11-09 · `d763e83` → `e1f57f7` (first run of the weekly review job)

| Commit | What changed | Effect on go-build-it |
|---|---|---|
| `77e95b8` | Over SSH, a remote PC now offers the herdr sessions already running there instead of quietly starting a second one. | None. Remote machines are out of scope (requirements §15). It confirms the pattern go-build-it already uses: adopt a running session before starting one (the supervisor's `Ensure`). |
| `c09b717` | Lists Letta Code's commands and skills in the chat's `/` menu. | None. go-build-it has no Letta profile. |

No follow-up is needed.

## 2026-11-08 · `d763e83` · 0.5.1 (baseline)

Ovis is a Bun and React web client for herdr. It's useful to go-build-it as a reference, and as a warning about what not to build.

**Patterns go-build-it follows:**
- **Transcript reading per agent** (`server/conversation.ts`, `server/claude-*.ts`, `server/codex*.ts`). It locates transcripts from the agent's session ID and process, and builds a chat view and usage from them. go-build-it does the same through each harness profile's transcript reader (FR-66, FR-71).
- **Status from herdr events** (`events.subscribe`), plus `pane.process_info` to identify the agent.
- **Browser flow control.** The browser acknowledges a chunk only after xterm.js has parsed it, with a transport budget of about 1 MiB. go-build-it's terminal bridge uses the same idea (P14-01).
- **Phone access through Tailscale Serve.** go-build-it already does this (FR-20).

**Patterns go-build-it does not follow:**
- **Answering agent dialogs by reading the screen.** `server/prompt.ts` is about 2.2k lines, `server/dialogs/*` holds a parser per agent, and the tests run to roughly 7.6k lines. Ovis has to change whenever an agent changes its UI. go-build-it shows a blocked member as "needs you" and links to the live terminal instead (FR-68, ADR-009).
- **A Rust or Node PTY sidecar.** Ovis needs one only because Bun can't load native PTY bindings and it attaches with `herdr terminal attach`. go-build-it uses `herdr terminal session observe/control` over stdio.
- **Remote machines over SSH port forwarding.** This is out of scope for go-build-it (requirements §15).

**Worth re-checking each review:**
- changes to how Ovis reads Claude, Antigravity, Pi and OpenCode transcripts, which go-build-it's transcript readers mirror
- how Ovis handles herdr protocol changes (`server/herdr/client.ts`, `methods.ts`)
- Windows mirror mode (`server/mirror.ts`): which herdr features are still missing on Windows
