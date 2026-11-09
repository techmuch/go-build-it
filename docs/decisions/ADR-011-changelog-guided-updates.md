# ADR-011: In-app change log and guided updates

Status: Accepted · approved by the owner 2026-11-09 · Tracker: P16-01 · Requirements: FR-06a–FR-06e

## Context

go-build-it could already check for, verify, install and roll back releases (ADR-004). That only showed up as one line in Settings: there was no record of what each version changed, and an update ended with "restart to apply" and no view of what had happened. The owner asked for:
- a change log inside the app, opened by clicking the version number
- an update check run from the change log
- a button that starts the update and shows each step as it completes

## Decision

1. **One change log, kept in the repo.** It lives in `CHANGELOG.md` at the repository root, in the [Keep a Changelog](https://keepachangelog.com) layout:
   - Headings are `## [YYYY.MM.DD.N] - YYYY-MM-DD`, plus `## [Unreleased]`.
   - Changes are grouped under `### Added`, `### Changed`, `### Fixed` and `### Removed`.
   - The root package `gobuildit` embeds the file, and `internal/update` parses it.
   - A test fails the build when the file does not parse, or when a version is not CalVer or is out of order.
2. **The release notes are the change log.**
   - `gbi dev changelog release VERSION` turns Unreleased into the dated entry.
   - `gbi dev changelog notes VERSION` prints an entry for `gh release create --notes-file`.
   - `make release` refuses a version that has no entry.
   - Because of this, the in-app view of a *newer* release (taken from the GitHub release body) matches what that release's own change log will say once it is installed.
3. **Clicking the version opens the change log.** On desktop it is the sidebar version, which opens `/changelog`; on mobile it is the version on More, which opens `/m/changelog`. Both are pages, not dialogs, so they can be linked to and are reachable by keyboard (`g` `v`).
4. **The check lists every newer release.** It uses GitHub's list-releases API (not only `latest`): stable, non-draft releases newer than the installed version, newest first. The result shows when the check ran.
5. **An update is a job with named steps.**
   - The job is saved to `~/.go-build-it/update/job.json` after every change.
   - The steps are: find release → download (with bytes and %) → verify signature → verify checksum → test the new binary (`<new> version` must print the expected tag) → install (keep the previous binary and version for rollback) → restart → reconnect.
   - The UI polls `GET /api/updates/job` twice a second. While the server restarts, the UI shows "Waiting for go-build-it to come back" and keeps polling.
6. **Restart replaces the process in place.**
   - On macOS and Linux the server shuts down HTTP cleanly, then `exec`s the new binary with the same arguments and environment. The PID stays the same, so launchd and systemd see no exit.
   - On Windows (beta, NFR-01) it starts the new binary detached and exits. The new process retries binding the port for up to 15 s.
7. **The new process finishes the job.**
   - On start, if the saved job is `restarting` and the running version is its target, the restart step is marked done.
   - Reconnect is marked done once the gbi herdr session answers and running sessions have been recovered. The job then becomes `done`, and `update.completed` is written to the history.
   - If the running version is not the target, the job fails at the restart step and Roll back is offered.
8. **Members keep working.** herdr owns the harnesses (FR-08), so a manual update is allowed while members work, and the view says they keep running. The automatic updater still waits until no session is running, so it never restarts on its own in the middle of someone's work.
9. **Automatic updates use the same job.** Their trigger is recorded as `auto` and they are shown in the same view (FR-06e).
10. **Testing.**
    - `update.PublicKey` is a variable that can be set at build time (`-ldflags -X`). A test build trusts a test key, but a release binary cannot be told to trust another key without rebuilding it.
    - `updates.api` in the config points the updater at a fake release server.
    - `scripts/update_e2e.sh` builds an old and a new binary, signs the new one with a throwaway key, serves a fake GitHub API, and runs `web/e2e/update.spec.ts` through a real restart.

## Consequences

- Each release needs a change log entry. That is a small cost, and the release command enforces it.
- ADR-004's "wait until no sessions run" now applies only to the automatic updater. A manual update is the user's call.
- A crash-looping new binary would not be caught by this process. The "test the new binary" step, which runs it before the swap, makes that much less likely, and `gbi updates rollback` can always restore the previous version.
