# ADR-008: Attachable (headed) sessions use tmux (Q7)

Status: Accepted (experimental) · approved by the owner 2026-11-08 · Tracker: P9-03

When `experimental.attachable_sessions` is on and tmux is installed, a session runs the harness's interactive TUI inside a tmux session named `gbi-<session id>`, in the task's worktree.

- **Prompt:** the prepared context is written to `.gbi-task.md`, and the agent is told to read it, so newlines don't submit early.
- **Terminal view:** go-build-it captures the pane every second into the terminal view (FR-66).
- **Input:** input from the web or mobile UI is sent with `tmux send-keys`.
- **Taking control:** a human runs `tmux attach -t gbi-<id>` from any terminal (locally or over SSH via Tailscale) and detaches with Ctrl-B d.
- **Finishing:** the session ends when the harness exits or someone presses *Finish*. Courier then commits, pushes and opens the review as usual.

Tradeoff: interactive TUIs don't stream structured events, so context usage is not reported in this mode. That is why it stays experimental, and headless sessions remain the default. A PTY-based in-browser terminal (xterm.js) could replace tmux later without changing the API.
