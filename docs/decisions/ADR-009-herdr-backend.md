# ADR-009: herdr is the only agent backend

Status: Accepted (direction) · approved by the owner 2026-11-08 · confirmed at the P10 gate after the spike (P10-04) · Supersedes [ADR-008](ADR-008-attachable-sessions.md) · Requirements: FR-01, FR-07, FR-08, FR-60, FR-61, FR-64, FR-66–FR-69, NFR-30–NFR-32 · Tracker: P10–P15

## Context

Until rev 3, go-build-it ran harnesses headlessly (`claude -p --output-format stream-json`) and offered an experimental tmux mode for watching and taking control (ADR-008). The owner wants to replace their use of Ovis, a browser front end for herdr, with go-build-it. They also want to open herdr in a terminal and immediately see everything go-build-it is doing, organized by project and team member.

A review of herdr 0.9.5 and Ovis (2026-11-08) found that herdr's socket API already provides what go-build-it needs:

- **Layout:** workspaces, tabs and panes, with git worktrees grouped under their project workspace (`worktree.open`).
- **Agent control:** `agent start`, `agent prompt --wait`, `agent wait --until blocked|done`, and `agent send-keys`.
- **State:** `events.subscribe` delivers `pane.agent_status_changed` (working, blocked, done, idle), and `session.snapshot` returns the full current state.
- **Labels:** `pane.report_metadata` and `workspace.report_metadata` let go-build-it name panes and show status values in herdr's sidebar. `notification.show` raises toasts.
- **Conversation identity:** with herdr's integration installed, each pane reports the harness's native session ID (`agent_session`). That ID locates the transcript, which provides the structured view and context usage.
- **Live terminal:** `herdr terminal session observe` and `herdr terminal session control` stream a pane as newline-delimited JSON frames over stdio. go-build-it needs no PTY library or sidecar of its own.
- **Restart:** herdr restores the layout after a restart and resumes supported agents in their own conversations.

Ovis also showed what not to copy. It answers harness permission prompts by parsing the screen, with a separate dialog parser for each agent. That code runs to thousands of lines and breaks whenever an agent changes its UI.

## Decision

1. **One backend.**
   - Every harness runs interactively in a herdr pane. The tmux mode, the experimental `attachable_sessions` flag and the headless `stream-json` path are all removed.
   - Without herdr, go-build-it can show and plan work but cannot run agents.
2. **A dedicated session.**
   - go-build-it supervises `herdr server --session gbi` and talks to its socket.
   - The user opens that session with `herdr --session gbi`. Their own default herdr session stays separate.
3. **Layout.**
   - Each project is a herdr workspace.
   - Each AI team member ([ADR-010](ADR-010-project-team-members.md)) has a long-lived worktree, grouped under its project's workspace, and runs in that worktree's root pane.
   - Panes carry metadata: display name, role, current task, and context %.
   - Each workspace carries a status summary.
4. **Dispatch.**
   - go-build-it writes the task context to `.gbi-task.md` in the member's worktree.
   - It then sends `agent prompt --wait` with a short instruction to read that file. This keeps prompts short, because herdr rejects input lines longer than macOS's terminal input limit.
5. **Truth for status.**
   - herdr's agent state is authoritative. A `blocked` member becomes a "needs you" item linked to its live terminal.
   - go-build-it never parses harness dialogs. It avoids permission prompts through permission modes and per-project allowlists, and may use harness hooks for structured approvals.
6. **Transcripts for structure.**
   - Usage, messages and events come from the harness's transcript, located through `agent_session`.
   - Each harness profile declares its transcript reader: Claude Code, Antigravity, Pi and OpenCode.
7. **Reconnection.**
   - go-build-it finds its panes again by conversation ID, which survives a herdr restart. Pane IDs and metadata tokens do not.
   - If event history is lost (`events_lost`), go-build-it rebuilds its state from `session.snapshot`.
8. **Compatibility.**
   - go-build-it's herdr client types are generated from `herdr api schema --json`.
   - The supported version range is declared once, and CI installs herdr on every platform.
   - At connect time, go-build-it checks herdr's `ping` response for the API methods it requires.
9. **Upstream watch.**
   - `docs/upstream/` records the herdr and Ovis revisions last reviewed.
   - A weekly scheduled review proposes changes when either project moves.

## Consequences

- **Dependency.** go-build-it is no longer dependency-free. FR-01 now lists herdr as its one runtime dependency, and onboarding checks it.
- **Updates.** Agents keep running while go-build-it restarts or updates, because herdr owns them (FR-06, FR-08).
- **Unstructured parts.** Structured data now depends on transcript formats and screen-based state detection, neither of which is documented as stable. That risk is shared with herdr and Ovis, and NFR-30–NFR-32 keep it visible.
- **Windows (Q9, decided 2026-11-08).** Windows ships at herdr's beta level, with herdr in the Windows CI job. A herdr capability that is missing on Windows is shown as unavailable there rather than worked around.
- **Tests.** The fake harness keeps working: it runs in a herdr pane and reports its own state with `pane.report_agent`.

## Alternatives rejected

- **tmux plus Claude hooks.** No status model, no Windows support, and text-only control.
- **Keeping headless `stream-json` alongside herdr.** Two execution paths, two usage models and two test suites. It also leaves "open herdr and see everything" only half true.
- **Embedding a PTY in go-build-it (`creack/pty` and ConPTY).** Agents would die whenever go-build-it restarts, and go-build-it would re-implement what herdr already does.

## Spike findings (P10-04, 2026-11-08, herdr 0.9.5, Claude Code 2.1.319)

Run in an isolated session (`herdr --session gbispike server`), then deleted.

- **Session isolation needs an explicit socket.**
  - `herdr server` ignores `HERDR_SESSION`. With the default server already running, a `workspace create` sent while `HERDR_SESSION` was set landed in the user's **default** session. It was closed immediately.
  - go-build-it must therefore start the server with `herdr --session gbi server`, and talk only to that session's socket (`~/.config/herdr/sessions/gbi/herdr.sock`, via `HERDR_SOCKET_PATH`).
  - Before any write, it must check that `status server` answers on that socket. It must never fall back to the default session.
- **Starting an agent.**
  - `agent start --kind claude --pane … -- --permission-mode acceptEdits` returned `agent_not_ready`, because Claude was `blocked` on its first-run "trust this folder" dialog.
  - That's correct herdr behaviour (FR-68). Each member worktree needs this trust once. go-build-it shows it as a one-time "needs you" item linked to the live terminal, and does not answer it itself. *(Superseded by [ADR-013](ADR-013-folder-trust.md): go-build-it now answers the trust prompt for its own folders.)*
  - After the trust was given, the agent reached `idle` within about 3 s.
- **Session ID.** `agent get` exposed `agent_session = {source: herdr:claude, kind: id, value: <uuid>}` even with the outdated v7 integration. go-build-it still asks for the current version.
- **Dispatch.**
  - go-build-it wrote `.gbi-task.md`, then sent `agent prompt atlas "Read .gbi-task.md in this folder and do what it says." --wait --until idle --until done --until blocked`.
  - The call returned after 8 s with `idle` and `completion_seq: 4`, and the file the task asked for existed.
- **Transcript and usage.**
  - The transcript is at `~/.claude/projects/<cwd with every non-alphanumeric character replaced by "-">/<session id>.jsonl`.
  - `assistant` rows carry `message.usage`. Context used is `input_tokens + cache_creation_input_tokens + cache_read_input_tokens` of the latest assistant row: 42,545 tokens here, a measured value.
  - Tool calls appear as `tool_use` content items (`Read`, `Write`), which give the structured view.
- **Terminal stream.**
  - `terminal session observe <pane> --cols --rows` prints NDJSON `terminal.frame` records `{seq, full, width, height, encoding, bytes(base64 ANSI)}`. The first frame is a full repaint of about 4.3 KB.
  - `terminal session control` reads `terminal.input {text|bytes}`, `terminal.resize {cols, rows}`, `terminal.scroll`, `terminal.mouse` and `terminal.release`. Source: `src/client/terminal_sessions.rs`.
  - There is no acknowledgement-based flow control, and herdr drops an observer that makes no progress for 30 s. go-build-it's WebSocket bridge must drain promptly and apply its own browser-side flow control.
- **Labels.**
  - `pane report-metadata --source gbi --title … --display-agent … --token k=v` showed up in `pane get` as `title`, `display_agent` and `tokens`.
  - `workspace report-metadata --token` takes the same form.

Result: the direction holds. No change to the decision.

### Further findings while building P11 (2026-11-08)

- **Session IDs from custom agents aren't kept.** herdr stores a session reference only from its official integrations (`src/agent_resume.rs`, `is_official_agent_source`). go-build-it therefore **chooses each conversation ID itself at launch**: Claude Code takes `--session-id <uuid>` for a new conversation and `--resume <uuid>` to continue it. herdr's `agent_session`, when present, is only a cross-check.
- **Agent commands work only for agent kinds herdr detects.** `agent.prompt` and `agent.send_keys` require one of herdr's known kinds (`effective_known_agent`), so custom agents such as the fake member cannot use them. go-build-it uses one dispatch path for every harness:
  1. Record the agent's `state_change_seq`.
  2. Submit the prompt: `agent.prompt` for known kinds, or `pane.send_text` plus Enter for custom agents.
  3. Poll `agent.get` until `state_change_seq` has advanced and the status is idle, done or blocked.

  This is race-free, and a stall timeout covers prompts that never start.
- **Status subscriptions are per pane.** `pane.agent_status_changed` must name a `pane_id`. The watcher subscribes for each member pane plus the global lifecycle events, and resubscribes whenever members change.
- **Isolated herdr for tests.** herdr honours `XDG_CONFIG_HOME`, so tests and demos run an entirely separate herdr (`[herdr] config_home`) and never touch the user's sessions.

### P13 real-harness run (2026-11-09: Claude Code 2.1.319, Antigravity 1.5.3, Pi 0.85.1, herdr 0.9.5)

A scratch project with one member per harness ran one small task each through herdr. Each member had its own worktree and pane.

- **Claude Code and Antigravity** completed end to end:
  - launch, folder trust, prompt, edit, settle, transcript usage (43k and 19k tokens, both measured), Courier commit, review.
  - Both showed a one-time "trust this folder" prompt for the new member worktree. go-build-it showed it as `blocked` / needs you and waited. Antigravity showed its prompt only after `agent start` had returned, so go-build-it checks for a blocked status after a successful start as well.
- **Pi** worked mechanically: launch with `--session-id`, prompt, settle, and the transcript found through the chosen ID. However, the local model configured in this Pi install returned empty replies with zero tokens, so nothing changed in the worktree. Pi's other configured models go through OpenRouter, which bills per token, so they were not used without asking the owner.
- **Timing fixes** found by this run:
  - A freshly opened pane may still be starting its shell (`agent_pane_busy`), so `agent start` is retried briefly.
  - A just-started agent can be refused by `agent prompt` (`agent_not_ready`) for a moment, so go-build-it waits for `interactive_ready` and retries.
