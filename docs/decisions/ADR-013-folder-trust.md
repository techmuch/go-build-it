# ADR-013: go-build-it answers the folder trust prompt for its own folders

Status: Accepted · decided by the owner 2026-11-09 · Tracker: P19-01 · Requirements: FR-07f, FR-68 · Amends [ADR-009](ADR-009-herdr-backend.md)

## Context

Interactive harnesses ask once per folder whether to trust it before they do anything:
- **Claude Code:** "Quick safety check: Is this a project you created or one you trust?", with *No, exit* selected.
- **Antigravity CLI:** "Do you trust the contents of this project?", with *Yes, I trust this folder* selected.

ADR-009 decided that go-build-it would never answer this itself and would show it as a one-time "needs you" item. In practice that means:
- every new member, in every project, stops at the prompt the first time it runs, because member worktrees live in `~/.go-build-it/worktrees/…`, outside the folders the person has already trusted
- every helper harness stops once too, and so does the doctor's harness tier

The owner decided this works against the purpose of go-build-it, which is to automate the use of AI agents. They asked for gbi to say yes for them, or to use a command-line option.

**No command-line option exists:**
- Claude Code skips the trust dialog only in non-interactive mode (`-p`). Members run interactively in herdr.
- Antigravity's `--dangerously-skip-permissions` covers tool permissions, not folder trust.

## Decision

1. **Answer the prompt with keystrokes.** When herdr reports an agent `blocked`, go-build-it reads the pane's screen.
   - If the screen is a folder trust prompt, it presses the keys a person would: down or up to reach *Yes, I trust this folder*, then enter.
   - It recognises the screen by its question ("trust this folder", "one you trust", "trust the contents/files") and a *Yes, I trust* or *Yes, proceed* option, and it finds the selected option by its cursor mark.
   - It uses herdr's `agent.send_keys`. For an agent herdr did not start by name, it types the key codes into the pane.
   - Tested by hand against Claude Code 2.1.321 and Antigravity CLI 1.5.3; both reached `idle` within a few seconds.
2. **Only go-build-it's own folders.** The pane's folder, as herdr reports it, must be inside go-build-it's directory (`~/.go-build-it`): member worktrees, helpers and the practice project. go-build-it created these folders from repositories the person imported.
3. **On by default, can be turned off:** `[agents] trust_folders`, or **Settings → Agent harnesses → Answer folder trust prompts**.
4. **Recorded.** Each answer is an `agent.trusted` history event with the member, harness and folder.
5. **Everything else is still the person's.** Sign-in screens, permission dialogs and any screen that isn't recognised as a trust prompt stay a "needs you" item (FR-68). The doctor's harness tier answers trust prompts the same way. When the setting is off, its human fix names the trust prompt and the setting.

## Consequences

- A new member, helper or doctor run starts without a person. The first run in a worktree no longer appears as blocked.
- **It depends on how the screen is worded.** If a harness rewords its trust prompt, go-build-it stops recognising it and falls back to "needs you", which is safe. The unit tests hold the real screens.
- **Trust is a safety prompt, and go-build-it now answers it.** The folders are go-build-it's own checkouts of repositories the person chose to import, the answer is recorded, and the setting turns it off. Folders outside `~/.go-build-it` are never answered.
