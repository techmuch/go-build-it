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
