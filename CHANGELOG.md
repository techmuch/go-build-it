# Changelog

All notable changes to go-build-it are recorded here. The app shows this file when you click its version number (FR-06a, FR-06b).
Versions use calendar versioning, `YYYY.MM.DD.N` (FR-05). The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
To release, run `gbi dev changelog release YYYY.MM.DD.N`, then `VERSION=YYYY.MM.DD.N make release` (see README).

## [Unreleased]

## [2026.11.12.2] - 2026-11-12

### Added

- **Choose models from a list** (FR-63a). An archetype's default model and a member's model override offer the models the harness actually has: from `agy models`, `pi --list-models` or `opencode models`, and built in for Claude Code (aliases and current IDs). It's searchable, any name can still be typed (with a warning when the harness doesn't list it), and ↻ asks the harness again. Lists are cached for a day. CLI: `gbi harnesses models <harness>`.

### Changed

- **The logo and title lead home** (FR-34a). Clicking the logo or *go-build-it* in the sidebar, or pressing Enter on it, opens the project catalog.

## [2026.11.12.1] - 2026-11-12

### Added

- **Remote access in Settings** (FR-20a–FR-20d, ADR-015). A checklist proves go-build-it works on your tailnet:
  - **Tailscale:** installed, running and logged in.
  - **The Serve rule:** a rule points at go-build-it's own port, matched exactly. Before, a text search counted any mention of the port.
  - **The round trip:** the tailnet URL answers, and the answer comes from *this* go-build-it.
  - **Private:** Funnel is off.
  - **Other services:** everything else the machine serves is listed, with the program behind each address.
- **QR code** of the working tailnet URL, for your phone, in Settings and in the terminal (`gbi tailscale qr`).
- **Doctor check `tailscale.serve`**, skipped when you don't use Tailscale.

### Changed

- **Setting up Tailscale Serve never takes another service's address.** If another app owns `https://<machine>.ts.net`, go-build-it names it, proposes its own port, shows the command, and serves there only when you confirm (`gbi tailscale setup --confirm`). Before, setup silently replaced whatever was there.
- **Stop serving** (`gbi tailscale stop`) removes only go-build-it's rule. Only owners can set up or stop serving, and each change is recorded in the history.
- **New installations start with the starter standards:** access control, AI layer security, code design, development loop, the four DoD standards, Go backend, operational resilience, planning and alignment, React frontend and software hardening, plus Project documents. Existing installations are not reseeded.
- **Stronger starter personas.** The six starter archetypes now carry full role instructions: what good looks like, how the role works with gbi's commands, when to hand back, and what it never does.
  - **Pilot (PM):** small vertical slices tied to requirement IDs, ordered by value and risk, with gates decided by people.
  - **Atlas (architect):** measurable quality-attribute scenarios, option sets before costly decisions, ADRs after.
  - **Forge (developer):** test-first, small diffs, the full suite green, and never weakens a test.
  - **Probe (QA):** risk-based end-to-end tests, exploratory testing (SFDPOT), and reproducible findings.
  - **Lens (reviewer):** approves changes that improve the system, in a fixed order of checks, with evidence for each finding.
  - **Courier (release manager):** small revertible batches, a releasable main branch, and release notes written for users. The agent that writes release notes now follows these rules.
- **Upgrades:** starter instructions you never edited are upgraded to a new revision. Your harness and model choices are kept. Members move to the new revision through the archetype change review (`gbi archetypes apply <id> --all`). Instructions you edited are never touched.

## [2026.11.11.1] - 2026-11-11

### Changed

- **`docs/gbi.toml` controls the Artifacts sub-tabs** (FR-151):
  - **Your own roles:** add any role (for example `vision = "docs/vision.html"`), and it gets a tab.
  - **Order:** tabs follow the order of `[entries]`, or an explicit `tabs = [...]`.
  - **Only what you use:** a role you leave out has no tab, so a project without mockups has no Mockups or Compare tab.
  - **Names:** `[titles]` renames tabs.
  - **Kept:** gbi keeps your order, `tabs` and titles when it rewrites the file. Before, it dropped roles it didn't know.
  - **Check and init:** `gbi artifacts check` no longer reports standard roles you left out, and `gbi artifacts init` only creates the roles you use.

## [2026.11.09.2] - 2026-11-09

### Changed

- **Folder trust is answered for you** (FR-07f, ADR-013). Before, a harness's "Do you trust this folder?" prompt stopped every new member, helper and doctor run until you answered it in a terminal. Now go-build-it answers *Yes* for folders it created (member worktrees, helpers, the practice project), and records each answer in the history as `agent.trusted`. Sign-in and permission prompts still come to you, and folders outside `~/.go-build-it` are never answered. Turn it off in **Settings → Agent harnesses → Answer folder trust prompts**.
- The doctor names a trust prompt when the setting is off, and tells AI agents when a fix needs the person.

### Added

- **Project documents** (FR-140–FR-150, ADR-014). A project's **Artifacts** tab has six sub-tabs: **Requirements, Mockups, Plan, Tracker, Compare and Support files**.
  - **Entry points:** each of the first four opens an HTML entry point you choose (`gbi artifacts entry`). It's shown in a sandbox, and links between documents switch tabs and keep the `#anchor`.
  - **Markdown inside HTML:** `<gbi-md src="requirements.md">` includes Markdown in an HTML page, and turns `**FR-07**` lines into anchors.
  - **Labels:** every file under `docs/` has labels you give it (`gbi artifacts add|label`). Nothing is guessed, and unlabelled files are listed. Entry points and labels live in `docs/gbi.toml`, and Courier commits each change.
  - **Compare:** each mockup view (`data-view="v02"`) next to its screenshot (`v02.png`), side by side or as an overlay. Accept the real screen or file a task; decisions are kept in `docs/compare.json`, and a view comes back when its screenshot changes.
  - **Traceability:** requirement → views → tasks → real screens, with the gaps (`gbi trace list --gaps`).
  - **Tracker snapshot:** `gbi artifacts snapshot` writes `docs/tracker/tracker.json`, which the tracker HTML reads.
  - **Getting started:** `gbi artifacts init` creates the standard set without overwriting anything, and `gbi artifacts check` lists what needs attention. The new **Project documents** standard describes it all for people and AI members.
  - **Phone:** a project's **Documents** screen, with Compare.
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
