# Changelog

All notable changes to go-build-it are recorded here. The app shows this file when you click its version number (FR-06a, FR-06b).
Versions use calendar versioning, `YYYY.MM.DD.N` (FR-05). The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
To release, run `gbi dev changelog release YYYY.MM.DD.N`, then `VERSION=YYYY.MM.DD.N make release` (see README).

## [Unreleased]

### Changed

- **Folder trust is answered for you** (FR-07f, ADR-013). Before, a harness's "Do you trust this folder?" prompt stopped every new member, helper and doctor run until you answered it in a terminal. Now go-build-it answers *Yes* for folders it created (member worktrees, helpers, the practice project), and records each answer in the history as `agent.trusted`. Sign-in and permission prompts still come to you, and folders outside `~/.go-build-it` are never answered. Turn it off in **Settings → Agent harnesses → Answer folder trust prompts**.
- The doctor names a trust prompt when the setting is off, and tells AI agents when a fix needs the person.

### Added

- Windows on ARM release binary (`gbi_windows_arm64.exe`), which `install.ps1` already expects.

## [2026.11.09.1] - 2026-11-09

### Added

- **Doctor tiers** (FR-07a–FR-07e). The doctor now proves the stack works:
  - **static:** herdr, its API, your harnesses and the login service's PATH
  - **live:** it opens a throwaway pane in the `gbi` session, runs a command, and finds each harness *inside* the pane
  - **harness:** it starts each harness in herdr
  - **prompt:** one test prompt, only when asked
  Every result has evidence and a fix, automatic or for a person. `gbi doctor` runs without the server; `--json` (exit codes 0/1/2), `--fix` and `--report` serve AI agents.
- **Install with an AI agent** (FR-02a, FR-02b): give your coding agent the GitHub URL. The README walks it through the install. `scripts/install.sh` and `install.ps1` check the checksum and signature before installing, and `AGENTS.md` points agents to the right place.
- **Learn** (FR-134–FR-136): eleven lessons with Pro tips, in a coach panel beside the app with **Show me**. Steps tick themselves off from the history. The lessons run in a practice project with scripted members, so they cost nothing. Desktop `g l`, and More → Learn on the phone.
- **Drive gbi from your AI harness** (FR-137): a lesson and a cookbook, with `gbi agent guide` to install as a skill.
- **How go-build-it thinks** (FR-138): seven pages on why it is designed this way, each linking its decisions.
- Troubleshooting page with a section for every doctor check. Doctor results link to it.

- Change log in the app: click the version number in the sidebar or on mobile's More screen, or press `g` `v` (FR-06a, FR-06b).
- Check for updates from the change log. It lists the change log entry of every release newer than yours (FR-06c).
- **Update now**, with eight named steps that show their progress across the restart: find, download, verify the signature, verify the checksum, test the new binary, install, restart, reconnect (FR-06d).
- go-build-it restarts itself in place after an update, and herdr keeps members working meanwhile (FR-06d, FR-08).
- `gbi updates job` and `gbi changelog` CLI commands; `gbi dev changelog release|notes` for releases.
- herdr is the only agent backend: every AI team member runs in a pane of the supervised `gbi` herdr session. Open it with `herdr --session gbi` (ADR-009).
- Per-project teams made from global persona archetypes, with tuning, drift review, promotion back to the archetype and a starter team on import (ADR-010).
- Live terminal for any member, read-only or with take control, on desktop and mobile.
- Conversation rotation at 75% context, with member notes carried into the next conversation.
- herdr setup and doctor, integration checks, and a weekly upstream review of herdr and Ovis.
- Viewer thumbnails, saved pins, and keyboard navigation on every screen.
- Projects, artifacts, questions, reviews, dress rehearsals, option sets, standards, history and insights, desktop and mobile UIs, CLI and MCP from one API.

### Changed

- The automatic updater uses the same steps as **Update now**, and every update is recorded in the history (FR-06e).
- A manual update may run while members are working, because herdr keeps them running (ADR-011).

### Removed

- tmux and headless (`-p`, `stream-json`) agent runs.

### Fixed

- Release tags written by other tools (e.g. `v1.2.3`) are recognised as stable releases.
- The login service gets the user's `PATH`, so it finds herdr and the harnesses.
