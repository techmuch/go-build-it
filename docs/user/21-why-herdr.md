section: How go-build-it thinks
reqs: FR-07, FR-08, FR-60, FR-138
# Why herdr runs every agent

**In short:** go-build-it never runs an AI harness itself. Every member runs interactively in a pane of a herdr session that go-build-it owns, called `gbi`.

## The problem

AI harnesses like Claude Code are built to run interactively, for hours, in a terminal. Earlier designs ran them headless (`-p`, streaming JSON) or in tmux. Both caused trouble:
- **Headless runs** lose the harness's own interactive features (permission prompts, compaction, the conversation you can resume), and every harness behaves differently.
- **tmux** gave a terminal, but go-build-it had to scrape the screen to know whether an agent was working, waiting or done.

## The decision

[herdr](https://herdr.dev) is a terminal workspace manager made for coding agents. It runs each harness in a real terminal, reports its state (working, blocked, done, idle) through an API and events, and keeps running when go-build-it stops. So:
- **Members survive restarts and updates.** Restarting or updating go-build-it does not stop them; it reconnects to the `gbi` session.
- **You can always look.** `herdr --session gbi` shows every member's real terminal, grouped by project, from any terminal. The browser's live terminal is the same pane.
- **One process host.** There is no tmux and no headless path to maintain, and harness support follows herdr's.

## What it costs

go-build-it depends on herdr: it must be installed and in a supported version range. The **doctor** checks that, and a weekly upstream review keeps go-build-it in step with new herdr versions.

Decisions: [ADR-009](https://github.com/techmuch/go-build-it/blob/main/docs/decisions/ADR-009-herdr-backend.md) (herdr as the only backend), [ADR-008](https://github.com/techmuch/go-build-it/blob/main/docs/decisions/ADR-008-attachable-sessions.md) (the tmux design it replaced), [ADR-012](https://github.com/techmuch/go-build-it/blob/main/docs/decisions/ADR-012-doctor-and-learning.md) (the doctor).
