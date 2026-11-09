# What `herdr --session gbi` shows

Requirements: FR-67, FR-12d, FR-64, FR-68 · Decision: [ADR-009](../decisions/ADR-009-herdr-backend.md) · Tracker: P10-09, built in P13-02 and P13-09

Opening `herdr --session gbi` in any terminal shows everything go-build-it is doing, organized the same way as the app.

## Session

- **Name:** `gbi`. go-build-it starts it with `herdr --session gbi server` and supervises it.
- **Socket:** `~/.config/herdr/sessions/gbi/herdr.sock`. go-build-it talks only to this socket. The user's default herdr session is never used.

## Workspaces

| herdr object | go-build-it object | Label | Notes |
|---|---|---|---|
| Workspace (primary) | Project | project name, e.g. `InboxQL` | `cwd` = the project's main checkout. Created on the first member launch. Holds a `gbi` tab (below). |
| Workspace (linked worktree, grouped under the project) | AI team member | member name, e.g. `forge` | Opened with `worktree.open` on the member's long-lived worktree `~/.go-build-it/worktrees/<project>/<member>`, so herdr groups it under the project. |
| Root pane of the member workspace | The member's harness | — | `agent start --kind <harness>`; agent name `<project>-<member>` (lowercase, `[a-z0-9_-]`, ≤ 32 characters). |

## Pane metadata (source `gbi`)

These values come from `pane.report_metadata`. herdr shows them in its sidebar and Agents list.

| Field | Value | Example |
|---|---|---|
| `display_agent` | member name and role | `Forge (developer)` |
| `title` | current task, or `idle` | `T-151 OpenCode usage reporting` |
| token `task` | task ID | `T-151` |
| token `ctx` | context used, with `~` when estimated | `41%`, `~18%` |
| token `conv` | tasks in the current conversation | `3` |
| `state_labels` | shown instead of herdr's default words | `working` → `on T-151`, `done` → `review ready` |

Tokens are refreshed whenever usage or the task changes. herdr does not keep tokens across its own restarts, so go-build-it re-reports them after reconnecting.

## Workspace metadata

On each project workspace, go-build-it sets token `gbi` to a summary such as `2 working · 1 needs you · 3 queued`. The summary refreshes on every member state change.

## Notifications

go-build-it raises `notification.show` (sound `request`) when:
- a blocking question is asked
- a review is ready
- a gate is waiting
- a member becomes blocked

The notification text names the project and the item, e.g. "InboxQL: review T-151 ready".

## The `gbi` tab

Each project workspace has a second tab, labelled `gbi`. It contains one pane running `gbi watch <project>`, a compact text view of the queue, members and anything that needs the human, so that the terminal view is useful on its own.

## Agents view

go-build-it sets `agent.view.set` with source `gbi`:
- **Filter:** agents that are blocked or done anywhere, plus everything in the current workspace.
- **Sort:** by attention, then by most recent state change.

## Reconnecting

After a herdr or machine restart, herdr restores workspaces and panes and resumes each agent's conversation. go-build-it then:
1. reads `session.snapshot`
2. matches each pane's `agent_session.value` to the member's stored conversation ID
3. re-reports metadata

A member with no matching pane is relaunched in its worktree, with the harness's resume arguments.
