# ADR-012: A doctor that proves the stack works, AI-agent install, and lessons

Status: Accepted · approved by the owner 2026-11-09 · Tracker: P17-01, P18-01 · Requirements: FR-02a–FR-02b, FR-07a–FR-07e, FR-134–FR-139

## Context

The herdr doctor (FR-07) only inspected things: herdr on PATH and its version, a ping to the `gbi` session, the API methods, the integrations and the login service. It never opened a pane or started a harness, so the most common real failures went unnoticed:
- the harness binary missing from the PATH that herdr panes get
- a harness stuck on a sign-in or first-run trust screen
- a session that answers a ping but cannot create panes

It also ran only through `gbi serve`, which is often the broken part during an install, and its results were written for a person. An AI agent could not tell which fixes it was allowed to apply.

The owner asked for three things:
- a doctor that tests that gbi can start herdr and a harness inside it, and that helps a person and an AI agent alike find and fix the problem
- an install that works by giving an AI agent the GitHub URL
- lessons with pro tips, a guide to driving gbi from the user's harness, and an explanation of why gbi is designed this way

## Decision

### Doctor (FR-07a–FR-07e)

1. **It runs in the binary.** `gbi doctor` runs inside the process, using the config, the herdr supervisor and the harness profiles. It does not call the server. `GET /api/doctor` and the Settings page call the same package (`internal/doctor`).
2. **Checks run in tiers, and each tier stops at the first failure:**
   - **static**: today's checks, plus the harness binaries on the login service's PATH, which is read from the installed launchd plist or systemd unit.
   - **live**: start the gbi session if it is not running, open a `doctor` tab in the `gbi-doctor` workspace, and type `echo` with a random marker. Read the pane until the marker comes back. Then run `command -v` (`Get-Command` on Windows) for each enabled harness *inside the pane*, and close the tab.
   - **harness**: in a throwaway pane, start the harness with herdr's `agent.start` (or the scripted fake member) and wait until herdr reports it idle. A harness that is blocked, exits or times out fails the check, with the last 20 lines of the screen as evidence. No prompt is sent.
   - **prompt**: submit "Reply with the single word READY." and wait for the agent to settle. This is opt-in only (Q11), because it uses tokens.
3. **Every result is a contract.** Each result has a stable `id`, `status`, `title`, `detail`, `evidence{command, exit, output}`, `fix{kind: auto|human, action, instructions}` and `docs`. The ids are listed in one Go table, which the troubleshooting page and its test share.
4. **Exit codes:** 0 when nothing failed, 1 when every failure has an automatic fix, 2 when a failure needs a human. `--fix` applies the automatic fixes and runs the checks again. `--report` writes Markdown with the configuration (`external_token` removed) and every result.

### AI-agent install (FR-02a–FR-02b)

5. **The README is the install program.**
   - Its *Install with an AI agent* section is numbered, deterministic steps. It ends in `gbi doctor --json --fix` repeated until the exit code is 0 or 2.
   - The rules are part of that section: no default herdr session, no prompts without asking, no sudo without asking.
   - `AGENTS.md` points agents to it.
6. **The install scripts verify before installing.** `scripts/install.sh` and `scripts/install.ps1` take a version (default: latest) and a directory (default `~/.local/bin` or `%LOCALAPPDATA%\Programs\gbi`).
   - The SHA-256 must match `SHA256SUMS`.
   - The signature of `SHA256SUMS` is checked with OpenSSL (Ed25519) when it supports it; otherwise the script says that it was not checked. The public key is printed in the README and the script.
   - `GBI_RELEASE_API` points the script at a fake release server, so `scripts/install_e2e.sh` tests the real script without publishing anything.
7. **No release yet.** Until the first GitHub release is published (an outward-facing step the owner takes), the README gives the build-from-source steps, and the install script says that there is no release yet.

### Learning (FR-134–FR-139)

8. **Lessons, not an overlay tour.** Lessons are Markdown in `docs/learn/`, embedded like the user docs.
   - Front matter gives the title, minutes and summary.
   - Each `## Step:` has optional lines:
     - `check: event <type>` (a history event in the practice project since the lesson started)
     - `check: visit <path>`
     - `show: <path> <selector>`
   - `> **Pro tip:**` blocks are styled as tips.
   - A coach panel in the shell shows the active lesson's current step on every page, and **Show me** navigates and pulses the highlighted control.
9. **Progress comes from the history.** A step is done when its event exists in the practice project after the lesson started, when its view was visited, or when the user ticked it. Lesson starts and ticks are stored per user (`learn_progress`). The history is the source of truth for everything else, so no lesson-specific events are added to the product.
10. **Practice project.** `gbi learn practice` creates a `practice` project from a small seeded repo under `~/.go-build-it/practice/`, with the starter team overridden to the fake harness. It is labeled *practice* in the UI, and `--reset` / `--remove` recreate or remove it. No subscription is used (Q12).
11. **Harness control.**
    - A lesson and a cookbook page (`docs/user/19-ai-harness.md`) show request → commands → result.
    - Claude Code is the main example (`claude mcp add gbi -- gbi mcp`, or Bash with `gbi … --json`).
    - `gbi agent guide` prints the agent guide from the same embedded file, so a harness skill can be installed with `gbi agent guide > ~/.claude/skills/gbi/SKILL.md`.
12. **How go-build-it thinks.** Explanation pages are in `docs/user/` under section *How go-build-it thinks*, and each links its ADRs.
13. **Tested docs.**
    - `scripts/docs_examples.sh` runs every `$ gbi …` line from the lessons and the cookbook against a demo instance.
    - `web/e2e/learn.spec.ts` completes every lesson in the practice project, asserts each step is checked off, and regenerates the screenshots.

## Consequences

- The doctor opens and closes panes in the `gbi` session. They are labeled `doctor` and live in their own workspace, so they are easy to recognize. They never appear in the user's default session.
- The harness tier can leave a harness's first-run trust prompt answered in its helper folder. That is the same folder rule the `Ask` helpers use.
- The lessons depend on history event names. Renaming an event fails the lesson e2e test, which is intended.
- The README now carries the release public key. Rotating the key means updating the README, `install.sh`, `install.ps1` and `update.PublicKey` together. A test checks that all four match.
