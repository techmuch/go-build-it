# Changelog

All notable changes to go-build-it are recorded here. The app shows this file when you click its version number (FR-06a, FR-06b).
Versions use calendar versioning, `YYYY.MM.DD.N` (FR-05). The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
To release, run `gbi dev changelog release YYYY.MM.DD.N`, then `VERSION=YYYY.MM.DD.N make release` (see README).

## [Unreleased]

### Added

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
