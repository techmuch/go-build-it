# go-build-it — Phased delivery plan

Status: Draft · 2026-11-08 · Based on [requirements rev 2](../requirements.md) and [Approach A mockups](../mockups/a-mission-control/desktop.html) · Live status: [tracker](index.html)

## 1. How this plan is built

- **Approach A is the UI baseline.** Every UI task names the Approach A view it implements (desktop `A-v01`…`A-v14` plus detail views `02b`, `03b`, `04b`, `05b`, `12b`; mobile `A-M1`…`A-M11`).
- **Every requirement maps to at least one task.** The tracker's *Requirement coverage* tab checks this.
- **Each phase ends at a review gate (◆),** the same mechanism the app itself provides (FR-115). A gate passes only when the owner approves it.
- **The MVP is the desktop app at the end of P5.** Mobile (P6), observability UI (P7), docs/UAT/auto-update (P8) and experimental features (P9) follow.
- **Simple path first.** Harness compaction is the MVP default; smart handoffs and attachable headed sessions are experimental (P9), as decided in rev 2.
- **Dogfooding from P4.** Once Courier can branch, PR, merge and release (end of P4), go-build-it manages its own remaining phases, and this tracker's data is imported into it.
- **Sizes, not dates.** S ≈ up to a day of agent work, M ≈ 2–4 days, L ≈ 1–2 weeks. Progress is weighted S=1, M=3, L=8. Dates are set once the P0 spike shows real throughput.

## 2. Phases at a glance

| Phase | Name | MVP | Tasks | Points | Gate |
|---|---|---|---|---|---|
| P0 | Foundations & decisions | ✓ | 12 | 27 | ◆ Decisions review |
| P1 | Core platform (walking skeleton) | ✓ | 12 | 30 | ◆ Skeleton demo |
| P2 | Projects, catalog & artifacts | ✓ | 9 | 21 | ◆ Catalog review |
| P3 | Team, standards & harness engine | ✓ | 17 | 44 | ◆ Adapter demo |
| P4 | Courier: git & release | ✓ | 7 | 17 | ◆ Release rehearsal |
| P5 | Human-in-the-loop (MVP) | ✓ | 15 | 55 | ◆ MVP dress rehearsal |
| P6 | Mobile web UI |  | 9 | 17 | ◆ One-handed UAT |
| P7 | Observability & self-improvement |  | 5 | 13 | ◆ First retro |
| P8 | Docs, UAT, updates & integrations |  | 7 | 22 | ◆ v1 UAT |
| P9 | Experimental & expansion |  | 8 | 44 | ◆ Graduation reviews |

## 3. Critical path

- **Main chain:** P0 decisions and scaffold → P1 platform → P3 adapter interface → P3 queue and dispatcher → P4 Courier → P5 human loop → **MVP**.
- **In parallel:** P0-04 harness spike → P3 adapters; P1 → P2 projects and artifacts (feeds P3 context preparation and P5 editors).
- **After the MVP:** P5 → P6 mobile; P1 history store → P7 observability; P0-12 e2e harness + P5 rehearsals → P8 UAT and docs; P3 sessions view → P9 attachable sessions.

The longest chain runs through the **harness spike (P0-04, Q2)**, the adapter interface, the queue and dispatcher, Courier, and the work review. Start P0-04 first: its findings decide the scope of P3-06 and P3-07.

## 4. Phases in detail

### P0 · Foundations & decisions (MVP)

**Goal:** Lock the requirements and UI baseline, answer the questions that block building, and set up the repo, CI, standards and design kit.

**◆ Gate: Decisions review.** Passes when:
- Approach A confirmed as UI baseline
- Q2 spike results recorded; adapters scoped
- Q1, Q5, Q6 decided (Q4 can wait for P9)
- Repo builds a CalVer-stamped binary in CI

| Task | Size | Requirements | Approach A views | Depends on |
|---|---|---|---|---|
| **P0-01** Requirements draft (rev 2, incl. mockup feedback) ✅ | M | FR-01 | — | — |
| **P0-02** Three UI approaches mocked; Approach A reviewed ✅ | L | FR-34, FR-35 | A-all | P0-01 |
| **P0-03** Confirm Approach A as baseline; list aspects borrowed from B/C ✅ | S | FR-34, FR-35 | A-all | P0-02 |
| **P0-04** Harness spike: headless + usage reporting for Claude Code, OpenCode, Antigravity ✅ · waits on Q2 | M | FR-60, FR-61, FR-62, FR-71 | — | — |
| **P0-05** Decide default git workflow; write the Git workflow standard ✅ · waits on Q5 | S | FR-91 | A-v11 | — |
| **P0-06** Decide auto-update channel and signing ✅ · waits on Q6 | S | FR-06 | — | — |
| **P0-07** Decide external-interface access for MVP (default: refuse; Tailscale only) ✅ · waits on Q1 | S | FR-18, NFR-21 | — | — |
| **P0-08** Decide multi-user scope for v1 ✅ · waits on Q4 | S | FR-17, FR-117a | — | — |
| **P0-09** Repo scaffold: Go module, React + TS + Vite, embed, lint, CI, CalVer stamping ✅ | M | FR-01, FR-05 | — | — |
| **P0-10** Write go-build-it's own global standards (backend, frontend, tests, docs, versioning) ✅ | S | FR-50, FR-51 | A-v11 | P0-05 |
| **P0-11** Design tokens and component kit from Approach A (light + dark) ✅ | M | FR-34 | A-v01, A-v03b, A-v04b | P0-09 |
| **P0-12** Playwright e2e harness with desktop + phone screenshot capture ✅ | S | FR-133 | — | P0-09 |

### P1 · Core platform (walking skeleton) (MVP)

**Goal:** One Go binary with embedded React: config, SQLite, API, CLI parity, access classification, Tailscale Serve, live updates, history store, desktop shell.

**◆ Gate: Skeleton demo.** Passes when:
- Install, open on localhost and over Tailscale Serve with the right identity badge
- Every API call has a CLI command with --json
- History records every write with who/what/when/why

| Task | Size | Requirements | Approach A views | Depends on |
|---|---|---|---|---|
| **P1-01** State dir ~/.go-build-it and config loader (cross-platform paths) ✅ | S | FR-03, NFR-01 | — | P0-09 |
| **P1-02** SQLite storage layer and migrations ✅ | M | FR-04 | — | P1-01 |
| **P1-03** HTTP API skeleton, OpenAPI spec, generated TS client ✅ | M | FR-31 | — | P1-02 |
| **P1-04** CLI gbi as an API client with --json; parity test ✅ | M | FR-30, FR-32 | — | P1-03 |
| **P1-05** Request origin classifier and identity attribution ✅ | S | FR-15, FR-16, FR-17 | A-v13 | P1-03 |
| **P1-06** Loopback-only default; external refused unless configured ✅ | S | FR-19, FR-18, NFR-21 | — | P1-05, P0-07 |
| **P1-07** Tailscale Serve detection and guided setup ✅ | M | FR-20 | A-v13 | P1-05 |
| **P1-08** Live updates channel (SSE) and client store ✅ | M | FR-36 | — | P1-03 |
| **P1-09** Desktop shell: sidebar, topbar, identity badge, routing, ⌘K stub ✅ | M | FR-34 | A-v01 | P0-11, P1-03 |
| **P1-10** History event store used by every write (who/what/when/why/origin) ✅ | M | NFR-10 | — | P1-02 |
| **P1-11** Background service install (launchd) and gbi service commands ✅ | M | FR-02 | — | P1-04 |
| **P1-12** Release build: single binary, CalVer tag, checksums, signature ✅ | M | FR-01, FR-05 | — | P0-06, P0-09 |

### P2 · Projects, catalog & artifacts (MVP)

**Goal:** Manage real projects: catalog cards, project overview, roadmap and item detail, artifacts with git revisions, values and problem definition.

**◆ Gate: Catalog review.** Passes when:
- 6 real repos imported and shown correctly
- Release, website, CI and unpushed-work data on every card
- Roadmap items open their detail page

| Task | Size | Requirements | Approach A views | Depends on |
|---|---|---|---|---|
| **P2-01** Project model, import existing repo, lifecycle states ✅ | M | FR-43 | A-v01 | P1-02 |
| **P2-02** Forge detection and gh / glab auth check ✅ | S | FR-90 | A-v13 | P2-01 |
| **P2-03** Catalog view: cards, needs-you first, search, filter, sort ✅ | M | FR-40, FR-41, FR-42 | A-v01, A-M1 | P1-09, P2-01 |
| **P2-04** Card data: release, commits since, website, CI, PRs/issues, placeholders ✅ | M | FR-41, FR-41a, FR-44 | A-v01 | P2-02 |
| **P2-05** Unpushed-work detector and card warning ✅ | S | FR-94 | A-v01 | P2-01 |
| **P2-06** Project overview page ✅ | M | FR-41, NFR-11 | A-v02, A-M2 | P2-03 |
| **P2-07** Roadmap model (milestones, gates) and roadmap item detail page ✅ | M | FR-45b, FR-115 | A-v02, A-v02b | P2-06 |
| **P2-08** Artifacts browser: repo docs, git revisions, rendered Markdown and HTML mockups ✅ | M | FR-45, FR-46 | A-v03 | P2-01 |
| **P2-09** Problem definition and values with provenance ✅ | S | FR-123 | A-v03 | P2-08 |

### P3 · Team, standards & harness engine (MVP)

**Goal:** Personas and standards feed headless harness sessions; the queue dispatches tasks; context is monitored; compaction and rate limits are handled.

**◆ Gate: Adapter demo.** Passes when:
- Claude Code and OpenCode each finish a task headlessly (Antigravity if Q2 allows)
- Context % visible live, marked measured or estimated
- Compaction logged; rate-limit pause and auto-resume shown
- Queue survives a restart

| Task | Size | Requirements | Approach A views | Depends on |
|---|---|---|---|---|
| **P3-01** Teams, personas, humans and roles; assign to projects ✅ | M | FR-10, FR-11, FR-12, FR-25, FR-26 | A-v10 | P1-03 |
| **P3-02** Global standards store with project overrides and reasons ✅ | M | FR-50, FR-51, FR-52 | A-v11 | P1-03, P0-10 |
| **P3-03** Generate AGENTS.md / CLAUDE.md / OpenCode instructions per project and persona ✅ | S | FR-53 | A-v11 | P3-02, P3-01 |
| **P3-04** Harness adapter interface (start, events, input, usage, cancel, limits) ✅ | M | FR-61 | — | P0-04 |
| **P3-05** Claude Code adapter (stream-json) ✅ | M | FR-60, FR-62 | — | P3-04 |
| **P3-06** OpenCode adapter (run/serve; usage estimate per Q-34) ✅ | M | FR-62, FR-71 | — | P3-04 |
| **P3-07** Antigravity adapter (scope from Q2 spike) ✅ · waits on Q2 | M | FR-62 | — | P3-04 |
| **P3-08** Harness detection and settings (UI + CLI) ✅ | S | FR-63 | A-v13 | P3-04 |
| **P3-09** Git worktree per concurrent session ✅ | S | FR-64 | — | P3-04 |
| **P3-10** Session context preparation (task, reqs, standards, values, handoff) ✅ | M | FR-70 | A-v05 | P3-03, P2-09 |
| **P3-11** Context monitor with measured / estimated flag ✅ | S | FR-71 | A-v05 | P3-05, P3-06 |
| **P3-12** Compaction detection and history logging (MVP default) ✅ | S | FR-72 | A-v05, A-v12 | P3-11, P1-10 |
| **P3-13** Rate-limit detection, pause and auto-resume ✅ | M | FR-65 | A-v05 | P3-05 |
| **P3-14** Task queue: priorities, deps, role routing, concurrency, retries, persistence ✅ | L | FR-80, FR-81, FR-82 | A-v06 | P1-02 |
| **P3-15** Dispatcher: pick task, start session as persona, resume after restart ✅ | M | FR-60, FR-81, FR-82 | A-v06 | P3-14, P3-10 |
| **P3-16** Sessions view: table, structured live output, terminal (raw) view ✅ | M | FR-66, NFR-11 | A-v05, A-v05b | P3-15, P1-08 |
| **P3-17** PM sizing: flag tasks with repeated compactions; propose split ✅ | S | FR-75 | A-v06 | P3-12, P3-14 |

### P4 · Courier: git & release (MVP)

**Goal:** The Git / Release Manager runs the workflow from the Git workflow standard: branches, PRs, CI, merges, CalVer releases and Pages, with no LLM unless judgment is needed.

**◆ Gate: Release rehearsal.** Passes when:
- Task → branch → PR → merge → tag → Pages, end to end on a sample project
- Nothing completed is left unpushed
- From here go-build-it can manage its own remaining phases (dogfooding)

| Task | Size | Requirements | Approach A views | Depends on |
|---|---|---|---|---|
| **P4-01** Workflow engine from the Git workflow standard (with overrides) ✅ | M | FR-91 | A-v11 | P3-02, P0-05 |
| **P4-02** Deterministic git ops: branch, commit, push, PR/MR, CI status ✅ | M | FR-92, FR-93 | A-v09 | P4-01, P2-02 |
| **P4-03** Agent-assisted ops: merge conflicts, release notes ✅ | M | FR-92, FR-93 | — | P4-02, P3-15 |
| **P4-04** Push guarantee and reconciliation loop ✅ | S | FR-94 | A-v01 | P4-02 |
| **P4-05** Managed-project releases: CalVer tag and forge release ✅ | M | FR-92, FR-44 | A-v02 | P4-02 |
| **P4-06** Pages publishing (GitHub / GitLab) and stored URL | M | FR-44 | A-v01, A-v02 | P4-05 |
| **P4-07** Artifact revisions committed by Courier ✅ | S | FR-46, FR-45a | A-v03b | P4-02 |

### P5 · Human-in-the-loop (MVP) (MVP)

**Goal:** Questions, attachment viewer, dictation, assessment-style work review, review gates and dress rehearsals, option sets, and agent-assisted editing of artifacts, personas and standards.

**◆ Gate: MVP dress rehearsal.** Passes when:
- Full desktop UAT walkthrough passes for the Owner role
- A real feature goes option set → rehearsal → build → assessment review → release
- MVP release tagged

| Task | Size | Requirements | Approach A views | Depends on |
|---|---|---|---|---|
| **P5-01** Questions: blocking / non-blocking, assumptions, answers routed to sessions ✅ | M | FR-100, FR-101, FR-105 | A-v04 | P3-15 |
| **P5-02** Questions dashboard: triage list, detail, structured answers, propose value ✅ | M | FR-102, FR-103, FR-104 | A-v04, A-M3, A-M4 | P5-01 |
| **P5-03** Attachment and mockup viewer: live HTML with viewports, images, diffs, logs, pins ✅ | L | FR-102a, FR-116a | A-v04b, A-M11 | P0-11 |
| **P5-04** Voice dictation component with editable transcript ✅ | S | FR-107 | A-v04, A-v09 | P0-11 |
| **P5-05** Lens assessment: standards, requirements, feedback addressed ✅ | M | FR-54, FR-112 | A-v09 | P3-15, P3-02 |
| **P5-06** Mockup vs built comparison from e2e screenshots (side by side, overlay, slider) ✅ | M | FR-112 | A-v09 | P0-12, P5-03 |
| **P5-07** Work review page; verdicts feed follow-up task context ✅ | M | FR-110, FR-111, FR-112 | A-v09, A-M5 | P5-05, P5-06, P4-02 |
| **P5-08** Review gates block downstream work until approved ✅ | S | FR-115, FR-118 | A-v02b | P2-07, P3-14 |
| **P5-09** Dress rehearsal runner: steps, findings by several reviewers, live agent changes, gate decision ✅ | L | FR-116, FR-117, FR-117a, FR-118 | A-v08, A-M6 | P5-08, P5-03 |
| **P5-10** Option sets: generate against values, large preview, compare, combination tray ✅ | L | FR-120, FR-121, FR-122, FR-124 | A-v07, A-M7 | P5-03, P2-09 |
| **P5-11** Values and problem definition refined from answers, choices, feedback ✅ | S | FR-123, FR-105 | A-v03 | P5-02, P5-10 |
| **P5-12** Artifact editor: WYSIWYG + Markdown, agent chat with inline proposals, save → revision ✅ | L | FR-45a | A-v03b | P4-07, P3-15 |
| **P5-13** Persona coach and rich persona editor ✅ | M | FR-13 | A-v10 | P5-12, P3-01 |
| **P5-14** Standards editing through the artifact editor with agent chat ✅ | S | FR-51 | A-v11 | P5-12, P3-02 |
| **P5-15** Needs-you aggregation: badges and overview card across questions, reviews, gates, options ✅ | S | FR-104 | A-v02, A-v01 | P5-02, P5-07, P5-08, P5-10 |

### P6 · Mobile web UI

**Goal:** The separate phone UI from Approach A mobile: projects, inbox, answer, review, rehearsal, options, sessions with terminal, viewer, more.

**◆ Gate: One-handed UAT.** Passes when:
- Answer a blocking question, approve a review and give rehearsal feedback one-handed in under a minute each
- Phone e2e screenshots in docs

| Task | Size | Requirements | Approach A views | Depends on |
|---|---|---|---|---|
| **P6-01** Mobile shell: routing and 4 bottom tabs ✅ | M | FR-35 | A-M1 | P1-09 |
| **P6-02** Projects and project status screens ✅ | S | FR-35, FR-40, FR-41 | A-M1, A-M2 | P6-01, P2-06 |
| **P6-03** Inbox and answer screen with dictation ✅ | M | FR-104, FR-107 | A-M3, A-M4 | P6-01, P5-02, P5-04 |
| **P6-04** Review screen (assessment + verdict) ✅ | S | FR-110, FR-112 | A-M5 | P6-01, P5-07 |
| **P6-05** Rehearsal step and option set screens ✅ | M | FR-117, FR-122 | A-M6, A-M7 | P5-09, P5-10 |
| **P6-06** Sessions and terminal view with key bar ✅ | M | FR-66 | A-M8, A-M10 | P3-16 |
| **P6-07** Full-screen viewer on phone ✅ | S | FR-102a, FR-116a | A-M11 | P5-03 |
| **P6-08** More: read-only areas and chat-based edits ✅ | S | FR-35 | A-M9 | P6-01 |
| **P6-09** PWA install; spike web push (out-of-app notifications stay out of scope) ✅ | S | FR-106 | — | P6-01 |

### P7 · Observability & self-improvement

**Goal:** History UI and event detail pages, metrics agents can analyze, and improvement proposals that go through normal review.

**◆ Gate: First retro.** Passes when:
- Rebuild any task's story from history alone
- An agent proposes an improvement from real metrics and it is reviewed

| Task | Size | Requirements | Approach A views | Depends on |
|---|---|---|---|---|
| **P7-01** History UI: search, filters, metric tiles ✅ | M | NFR-10, NFR-11 | A-v12 | P1-10 |
| **P7-02** Event detail views per type, human actions included ✅ | M | NFR-14 | A-v12b | P7-01 |
| **P7-03** Metrics pipeline: rework, answer time, time per task, compactions, limits ✅ | M | NFR-12 | A-v12 | P1-10 |
| **P7-04** Agent improvement proposals through normal review ✅ | M | NFR-13 | A-v12 | P7-03, P5-07 |
| **P7-05** Agent-readable history export / analysis API ✅ | S | NFR-12 | — | P7-03 |

### P8 · Docs, UAT, updates & integrations

**Goal:** Embedded searchable docs with auto-captured screenshots, the per-role UAT walkthrough, finished settings, auto-update with rollback, MCP server.

**◆ Gate: v1 UAT.** Passes when:
- UAT scripts for every role pass and regenerate docs screenshots
- Auto-update and rollback demonstrated
- External agent drives the app through MCP

| Task | Size | Requirements | Approach A views | Depends on |
|---|---|---|---|---|
| **P8-01** Embedded docs: Markdown, search, contextual ? Docs links ✅ | M | FR-130 | A-v14 | P1-09 |
| **P8-02** UAT walkthrough per role as Playwright scripts ✅ | L | FR-132, FR-133 | A-v14 | P0-12, P5-09 |
| **P8-03** Screenshot regeneration into docs ✅ | S | FR-131, FR-133 | A-v14 | P8-02, P8-01 |
| **P8-04** Settings page complete: harnesses, access, forges, experimental, updates ✅ | M | FR-63, FR-19, FR-20 | A-v13 | P3-08, P1-07 |
| **P8-05** Auto-update: verify, install when idle, roll back ✅ | M | FR-06 | A-v13 | P1-12 |
| **P8-06** MCP server over the API ✅ | M | FR-33 | — | P1-04 |
| **P8-07** Security posture check: exposure tests; no sandbox by design ✅ | S | NFR-20, NFR-21 | — | P1-06 |

### P9 · Experimental & expansion

**Goal:** Features behind flags or after v1: smart handoffs, attachable headed sessions, Pi-agent, Linux and Windows, external access challenge, multi-user.

**◆ Gate: Graduation reviews.** Passes when:
- Each experimental feature has evidence (NFR-12 metrics) before it becomes default
- Linux and Windows pass the UAT suite

| Task | Size | Requirements | Approach A views | Depends on |
|---|---|---|---|---|
| **P9-01** Smart handoffs behind experimental flag: threshold, briefing, Lens check, next session ✅ · waits on Q3 | L | FR-72, FR-73, FR-74 | A-v05 | P3-12, P5-05 |
| **P9-02** Handoff vs compaction evaluation and graduation criteria ✅ · waits on Q3 | M | FR-72, NFR-12 | A-v12 | P9-01, P7-03 |
| **P9-03** Attachable headed sessions: PTY/multiplexer, attach, detach, control handback ✅ · waits on Q7 | L | FR-67 | A-v05b, A-M10 | P3-16 |
| **P9-04** Pi-agent adapter ✅ | M | FR-62 | — | P3-04 |
| **P9-05** Linux support (systemd) 🔶 | M | NFR-01, FR-02 | — | P1-11 |
| **P9-06** Windows support (service, paths) 🔶 | L | NFR-01, FR-02 | — | P1-11 |
| **P9-07** External interface challenge (if Q1 allows external access) ✅ · waits on Q1 | M | FR-18 | — | P1-06 |
| **P9-08** Multi-user roles and review assignment ✅ · waits on Q4 | L | FR-17, FR-117a | — | P5-09 |

## 5. Open questions schedule

| Question | Answer before | Why |
|---|---|---|
| Q2 harness headless + usage | P0 (spike P0-04) | Sets adapter scope for OpenCode and Antigravity |
| Q5 default git workflow | P0 (P0-05) | Courier's workflow engine reads the Git workflow standard |
| Q6 update channel | P0 (P0-06) | Release build and signing in P1-12 |
| Q1 external access | P0 (P0-07) | MVP default: refuse; full challenge only if needed (P9-07) |
| Q4 multi-user | before P5-09 | Rehearsals already show a second reviewer (FR-117a) |
| Q3 handoff threshold and graduation | P9 | Smart handoffs are experimental |
| Q7 attachable sessions | P9 | Headed sessions are experimental |

## 6. Risks

| Risk | Effect | Mitigation |
|---|---|---|
| Antigravity has no usable headless mode (Q2) | One of three initial adapters slips | Spike first; ship Claude Code + OpenCode for MVP; keep the adapter interface harness-neutral |
| Context usage can only be estimated for some harnesses | Wrong compaction/handoff timing | Measured vs estimated flag (P3-11); compare estimates with real values (Q-34 answer) |
| Rich editor + agent inline proposals are complex (P5-12) | MVP slips | Use proven libraries (e.g. TipTap or CodeMirror); proposals as plain diffs first |
| Attachment viewer rendering untrusted HTML mockups | Script access to the app | Sandboxed iframe with no same-origin access |
| Subscription rate limits slow the team | Throughput drops | Pause/resume (P3-13), concurrency limits, schedule heavy work off-peak |
| Scope creep from experimental features | MVP never lands | Experimental work stays in P9 behind flags until a graduation review |

## 7. Tracking

Status lives in [`tracker-data.js`](tracker-data.js) and is shown by [`index.html`](index.html). Set a task's `status` to `todo`, `in-progress`, `blocked` or `done` and commit. The task tables in this document are generated from the same data, so regenerate them when tasks are added or renamed.
