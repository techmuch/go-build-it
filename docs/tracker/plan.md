# go-build-it — Phased delivery plan

Status: Draft · 2026-11-08 · Based on [requirements rev 3](../requirements.md) and [Approach A mockups](../mockups/a-mission-control/desktop.html) · Live status: [tracker](index.html)

## 1. How this plan is built

- **Approach A is the UI baseline.** Every UI task names the Approach A view it implements (desktop `A-v01`…`A-v14` plus detail views `02b`, `03b`, `04b`, `05b`, `12b`; mobile `A-M1`…`A-M11`).
- **Every requirement maps to at least one task.** The tracker's *Requirement coverage* tab checks this.
- **Each phase ends at a review gate (◆),** the same mechanism the app itself provides (FR-115). A gate passes only when the owner approves it.
- **The MVP is the desktop app at the end of P5.** Mobile (P6), observability UI (P7), docs/UAT/auto-update (P8) and experimental features (P9) follow.
- **Simple path first.** Harness compaction is the MVP default; smart handoffs and attachable headed sessions are experimental (P9), as decided in rev 2.
- **Rev 3 (P10–P15): herdr and per-project teams.** P0–P9 delivered rev 2. Rev 3 makes [herdr](https://herdr.dev) the only agent backend ([ADR-009](../decisions/ADR-009-herdr-backend.md)) and replaces global teams with per-project team members made from global persona archetypes ([ADR-010](../decisions/ADR-010-project-team-members.md)). It starts with a spike and redesigned mockups (P10), then builds the runtime (P11), teams (P12), members on herdr (P13), the live terminal and UI (P14), and docs, UAT and the upstream watch (P15). Rev 3 phases are not MVP-flagged: the MVP shipped with P5.
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
| P10 | herdr direction: decisions, spike & design |  | 11 | 24 | ◆ herdr direction review |
| P11 | herdr runtime foundation |  | 7 | 19 | ◆ Runtime demo |
| P12 | Per-project teams from archetypes |  | 9 | 24 | ◆ Team review |
| P13 | Team members on herdr |  | 13 | 39 | ◆ Member demo |
| P14 | Live terminal & UI |  | 8 | 21 | ◆ Watch & steer UAT |
| P15 | Rev 3 docs, UAT & upstream watch |  | 5 | 16 | ◆ Rev 3 UAT |

## 3. Critical path

- **Main chain:** P0 decisions and scaffold → P1 platform → P3 adapter interface → P3 queue and dispatcher → P4 Courier → P5 human loop → **MVP**.
- **In parallel:** P0-04 harness spike → P3 adapters; P1 → P2 projects and artifacts (feeds P3 context preparation and P5 editors).
- **After the MVP:** P5 → P6 mobile; P1 history store → P7 observability; P0-12 e2e harness + P5 rehearsals → P8 UAT and docs; P3 sessions view → P9 attachable sessions.

**Rev 3 chain:** P10 requirements and ADRs → P10-04 herdr spike → P11-01 herdr client → P11-02 session supervisor → P13-02 member pane lifecycle → P13-03 dispatch → P13-13 removal of tmux and headless runs → P14 live terminal → P15 UAT. P12 (teams) runs in parallel with P11 and must land before P13-02. Mockups (P10-07, P10-08) gate the UI tasks in P12 and P14.

The longest rev 2 chain ran through the **harness spike (P0-04, Q2)**, the adapter interface, the queue and dispatcher, Courier, and the work review. Start P0-04 first: its findings decide the scope of P3-06 and P3-07.

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
| **P4-06** Pages publishing (GitHub / GitLab) and stored URL ✅ | M | FR-44 | A-v01, A-v02 | P4-05 |
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
| **P9-05** Linux support (systemd) ✅ | M | NFR-01, FR-02 | — | P1-11 |
| **P9-06** Windows support (service, paths) ✅ | L | NFR-01, FR-02 | — | P1-11 |
| **P9-07** External interface challenge (if Q1 allows external access) ✅ · waits on Q1 | M | FR-18 | — | P1-06 |
| **P9-08** Multi-user roles and review assignment ✅ · waits on Q4 | L | FR-17, FR-117a | — | P5-09 |

### P10 · herdr direction: decisions, spike & design

**Goal:** Record rev 3 (herdr as the only backend, per-project team members from archetypes), prove the herdr integration with a spike, decide the open questions, and redesign the affected mockups before building.

**◆ Gate: herdr direction review.** Passes when:
- Requirements rev 3, ADR-009 and ADR-010 approved by the owner
- Spike: a Claude member in the gbi herdr session takes a prompt, finishes, reports its conversation ID and usage, and streams live to a browser
- Q8 and Q10 decided (Q9 can wait for P15)
- Rev 3 mockups (desktop, mobile, herdr layout) reviewed

| Task | Size | Requirements | Approach A views | Depends on |
|---|---|---|---|---|
| **P10-01** Requirements rev 3: herdr backend, per-project team members, upstream tracking ✅ | M | FR-01, FR-07 | — | — |
| **P10-02** ADR-009: herdr is the only agent backend; ADR-008 superseded ✅ | S | FR-60, FR-07 | — | P10-01 |
| **P10-03** ADR-010: per-project team members from global persona archetypes ✅ | S | FR-11, FR-12 | — | P10-01 |
| **P10-04** Spike: gbi herdr session, agent start + prompt --wait, agent_session → transcript usage, terminal observe → xterm.js ✅ | M | FR-60, FR-61, FR-66, FR-71 | — | P10-02 |
| **P10-05** Decide conversation rotation policy from spike cost data ✅ · waits on Q8 | S | FR-76 | — | P10-04 |
| **P10-06** Decide how archetype changes reach project members ✅ · waits on Q10 | S | FR-12b | — | P10-03 |
| **P10-07** Mockups rev 3, desktop: live members, live terminal, archetypes, project team, herdr settings and setup, card and overview team panel, queue assignment ✅ | L | FR-34, FR-12a, FR-66, FR-67 | A-v01, A-v02, A-v05, A-v05b, A-v06, A-v10, A-v10b, A-v13, A-v15 | P10-02, P10-03 |
| **P10-08** Mockups rev 3, mobile: members, terminal control and takeover, project team, blocked members in Inbox ✅ | M | FR-35, FR-66 | A-M2, A-M3, A-M8, A-M10 | P10-07 |
| **P10-09** herdr layout spec: what herdr --session gbi shows (workspaces, grouped worktrees, pane labels, tokens, summaries, notifications) ✅ | S | FR-67 | — | P10-04 |
| **P10-10** Upstream baseline: docs/upstream/ with herdr and Ovis revisions reviewed and findings ✅ | S | NFR-30 | — | P10-02 |
| **P10-11** Docs tooling in the repo: tracker, plan and HTML generators under scripts/docs ✅ | S |  | — | — |

### P11 · herdr runtime foundation

**Goal:** go-build-it owns and supervises the gbi herdr session: generated client, version and capability checks, onboarding and integrations, login service, event reconciliation, fake harness, CI with herdr on every platform.

**◆ Gate: Runtime demo.** Passes when:
- Login starts go-build-it and the gbi herdr session; restarting go-build-it leaves agents running
- Doctor shows herdr version, session and integration status, and fixes outdated integrations
- CI is green on macOS, Linux and Windows with a pinned herdr

| Task | Size | Requirements | Approach A views | Depends on |
|---|---|---|---|---|
| **P11-01** herdr client: socket/named-pipe NDJSON transport, types generated from herdr api schema, ping capability check, supported version range ✅ | M | NFR-32, FR-07 | — | P10-04 |
| **P11-02** Supervise herdr server --session gbi: start, health, restart, stop ✅ | M | FR-07 | — | P11-01 |
| **P11-03** Doctor and onboarding: herdr install and version, integration status, install or update integrations ✅ | M | FR-07 | A-v15, A-v13 | P11-01, P10-07 |
| **P11-04** Login service brings up go-build-it and the gbi session; go-build-it restarts and updates leave agents running ✅ | S | FR-08, FR-02, FR-06 | — | P11-02 |
| **P11-05** Event subscription and snapshot reconciler with events_lost recovery ✅ | M | NFR-11, FR-68 | — | P11-01 |
| **P11-06** Fake harness in herdr: reports state with pane.report_agent and writes a transcript ✅ | M | FR-61 | — | P11-01 |
| **P11-07** CI installs a pinned herdr on macOS, Linux and Windows; weekly job against latest herdr ✅ | M | NFR-32, NFR-01 | — | P11-06 |

### P12 · Per-project teams from archetypes

**Goal:** Replace global teams and personas with global persona archetypes and per-project team members that can be tuned, compared with their archetype, and promoted back.

**◆ Gate: Team review.** Passes when:
- Every imported project has a team made from starter archetypes
- A member's tuning, its difference from the archetype, and an archetype change applied per member are shown
- Each member's instruction file is generated in its worktree

| Task | Size | Requirements | Approach A views | Depends on |
|---|---|---|---|---|
| **P12-01** Schema: archetypes and project_members replace global teams and personas | M | FR-11, FR-12, FR-12d, FR-12e | — | P10-03 |
| **P12-02** Starter archetype per built-in role | S | FR-12, FR-25 | — | P12-01 |
| **P12-03** API, CLI and MCP: archetypes, project members (add, tune, remove), human roles per project | M | FR-30, FR-12a, FR-12e | — | P12-01 |
| **P12-04** Archetype change review: affected members, per-member apply keeping tuning · waits on Q10 | M | FR-12b | A-v10 | P12-03, P10-06 |
| **P12-05** Promote member tuning to its archetype through the artifact editor | S | FR-12c, FR-13 | A-v03b | P12-03 |
| **P12-06** Archetypes library and project Team tab with member detail and tuning | L | FR-13, FR-12a, FR-10 | A-v10, A-v10b | P12-03, P10-07 |
| **P12-07** Per-member instruction file generated in the member's worktree | S | FR-53 | — | P12-01 |
| **P12-08** Queue assignment to a role or a specific member; dispatcher picks a free member | M | FR-81 | A-v06 | P12-01 |
| **P12-09** Catalog card and overview show the project team and each member's herdr state | S | FR-41 | A-v01, A-v02, A-M2 | P12-06, P11-05 |

### P13 · Team members on herdr

**Goal:** Run every AI team member as a long-running harness in its herdr pane: profiles, pane lifecycle, dispatch, transcripts, status, limits, conversation rotation, reconnection and labels. Remove tmux and headless runs.

**◆ Gate: Member demo.** Passes when:
- Claude Code, Antigravity and Pi members each finish a task end to end through herdr, reviewed by Lens
- A blocked member shows as needing the human; herdr restart resumes members in their conversations
- herdr --session gbi shows every project and member labelled with role, task and context %
- No tmux or stream-json code remains

| Task | Size | Requirements | Approach A views | Depends on |
|---|---|---|---|---|
| **P13-01** Harness profiles: Claude Code, Antigravity, Pi, OpenCode (kind, launch and resume args, transcript reader, limit signals) | L | FR-61, FR-62, FR-63 | — | P11-01 |
| **P13-02** Member pane lifecycle: workspace per project, member worktree opened grouped, agent start, name, release | M | FR-12d, FR-64, FR-67 | — | P12-01, P11-02 |
| **P13-03** Dispatch: write .gbi-task.md, agent prompt --wait, hand completion to Courier and Lens | M | FR-60, FR-70 | — | P13-02, P12-08 |
| **P13-04** Transcript readers: structured events, messages and usage (measured or estimated) per harness | L | FR-66, FR-71, NFR-10 | — | P13-01 |
| **P13-05** Status mapping: herdr state drives session and task state; blocked becomes a needs-you item | M | FR-68 | — | P11-05, P13-02 |
| **P13-06** Rate and usage limit detection with pause and auto-resume | S | FR-65 | — | P13-04 |
| **P13-07** Conversation continuity and rotation per policy; smart handoff starts the new conversation · waits on Q8 | M | FR-69, FR-76, FR-72, FR-73, FR-74 | — | P13-04, P10-05 |
| **P13-08** Reconnect after herdr or machine restart by conversation ID; queue resumes | M | FR-69, FR-82 | — | P13-05 |
| **P13-09** herdr labels: pane metadata (name, role, task, context %), workspace summaries, notifications for gates and questions | S | FR-67 | — | P13-02, P10-09 |
| **P13-10** Permission prompts: permission modes, per-project allowlists, optional Claude permission hook as structured approval | M | FR-68 | — | P13-05 |
| **P13-11** Lens reviews run in a fresh conversation per review | S | FR-54, FR-112 | — | P13-03 |
| **P13-12** Interrupt, cancel and restart a member through herdr | S | FR-61 | — | P13-02 |
| **P13-13** Remove tmux mode, attachable_sessions flag and the headless stream-json path | S | FR-60 | — | P13-03, P13-04 |

### P14 · Live terminal & UI

**Goal:** Watch and steer any team member from the desktop and mobile UI through herdr's terminal streams, and bring herdr state into the live views, Inbox, settings and history.

**◆ Gate: Watch & steer UAT.** Passes when:
- Watch any member live on desktop and phone; take control and hand it back
- Blocked members appear in the Questions dashboard and mobile Inbox
- Every herdr state change, rotation and takeover is in the history

| Task | Size | Requirements | Approach A views | Depends on |
|---|---|---|---|---|
| **P14-01** Terminal bridge: herdr terminal session observe/control → WebSocket with ack-based flow control and takeover | L | FR-66 | — | P11-01, P10-04 |
| **P14-02** Desktop terminal view with take control and hand back | M | FR-66 | A-v05b | P14-01, P10-07 |
| **P14-03** Mobile terminal with key bar | M | FR-35, FR-66 | A-M10 | P14-01, P10-08 |
| **P14-04** Live members view: herdr state, task, context %, structured view | M | NFR-11, FR-66 | A-v05, A-M8 | P13-04, P13-05 |
| **P14-05** Blocked members in the Questions dashboard and mobile Inbox | S | FR-68, FR-104 | A-v04, A-M3 | P13-05 |
| **P14-06** Open in herdr: copy herdr --session gbi and focus the member's pane | S | FR-67 | A-v05 | P13-02 |
| **P14-07** Settings: herdr section (version, session, socket, integrations) | S | FR-63, FR-07 | A-v13 | P11-03 |
| **P14-08** History events for herdr state changes, rotations and takeovers | S | NFR-10, NFR-14 | A-v12, A-v12b | P13-05, P14-01 |

### P15 · Rev 3 docs, UAT & upstream watch

**Goal:** Update the embedded docs, UAT scripts and screenshots for the herdr flow, schedule the weekly herdr/Ovis upstream review, and settle Windows support.

**◆ Gate: Rev 3 UAT.** Passes when:
- UAT scripts for every role pass on the herdr flow and regenerate screenshots
- The first scheduled upstream review has run and recorded its findings
- Windows support level decided (Q9) and documented

| Task | Size | Requirements | Approach A views | Depends on |
|---|---|---|---|---|
| **P15-01** User docs rewrite: install, team, sessions, settings, mobile and CLI for herdr and archetypes | M | FR-130 | A-v14 | P14-04, P12-06 |
| **P15-02** UAT scripts for the herdr flow per role; regenerate screenshots | L | FR-132, FR-133, FR-131 | A-v14 | P15-01, P11-06 |
| **P15-03** Weekly upstream review job for herdr and Ovis that opens an issue or PR | S | NFR-31 | — | P10-10 |
| **P15-04** Windows at herdr's support level · waits on Q9 | M | NFR-01 | — | P11-07 |
| **P15-05** Rev 3 close-out: retire obsolete tests and text, mockup change log, coverage check | S | FR-01 | — | P15-02 |

## 5. Open questions schedule

| Question | Answer before | Why |
|---|---|---|
| Q2 harness headless + usage | P0 (spike P0-04) | Sets adapter scope for OpenCode and Antigravity |
| Q5 default git workflow | P0 (P0-05) | Courier's workflow engine reads the Git workflow standard |
| Q6 update channel | P0 (P0-06) | Release build and signing in P1-12 |
| Q1 external access | P0 (P0-07) | MVP default: refuse; full challenge only if needed (P9-07) |
| Q4 multi-user | before P5-09 | Rehearsals already show a second reviewer (FR-117a) |
| Q3 handoff threshold and graduation | P9 | Smart handoffs are experimental |
| Q7 attachable sessions | answered (rev 3) | Every session runs in herdr (ADR-009) |
| Q8 conversation rotation policy | P10 (P10-05) | Needs the spike's cost data; drives P13-07 |
| Q10 archetype change propagation | P10 (P10-06) | Shapes P12-04 |
| Q9 Windows support level | P15 (P15-04) | Depends on herdr's Windows beta |

## 6. Risks

| Risk | Effect | Mitigation |
|---|---|---|
| Antigravity has no usable headless mode (Q2) | One of three initial adapters slips | Spike first; ship Claude Code + OpenCode for MVP; keep the adapter interface harness-neutral |
| Context usage can only be estimated for some harnesses | Wrong compaction/handoff timing | Measured vs estimated flag (P3-11); compare estimates with real values (Q-34 answer) |
| Rich editor + agent inline proposals are complex (P5-12) | MVP slips | Use proven libraries (e.g. TipTap or CodeMirror); proposals as plain diffs first |
| Attachment viewer rendering untrusted HTML mockups | Script access to the app | Sandboxed iframe with no same-origin access |
| Subscription rate limits slow the team | Throughput drops | Pause/resume (P3-13), concurrency limits, schedule heavy work off-peak |
| Scope creep from experimental features | MVP never lands | Experimental work stays in P9 behind flags until a graduation review |
| herdr API changes (0.9.x, frequent releases) | go-build-it breaks after a herdr update | Types generated from `herdr api schema` (P11-01), capability check at connect, pinned herdr in CI plus a weekly latest-herdr job (P11-07), weekly upstream review (P15-03) |
| Transcript formats and screen-based state are not documented as stable | Wrong usage or status | Per-harness transcript readers with fixtures (P13-04); herdr state is authoritative; follow Ovis's readers (NFR-30) |
| Long-running conversations get expensive or degrade | Cost and quality drop | Memory kept in go-build-it, rotation policy (Q8, P13-07), fresh conversation per Lens review (P13-11) |
| herdr on Windows is beta | Windows lags | Support Windows at herdr's level (Q9, P15-04) |
| Permission prompts stall unattended members | Work waits on the human | Permission modes and allowlists, optional Claude permission hook (P13-10); blocked members surface in the Inbox (P14-05) |

## 7. Tracking

Status lives in [`tracker-data.js`](tracker-data.js) and is shown by [`index.html`](index.html). Set a task's `status` to `todo`, `in-progress`, `blocked` or `done` and commit. The task tables in this document are generated from the same data, so regenerate them when tasks are added or renamed:

- `python3 scripts/docs/add_rev3_phases.py`: re-sync requirements and rev 3 tasks into the data, and check coverage
- `python3 scripts/docs/setstatus.py done P10-04`: set task status
- `python3 scripts/docs/genmd.py`: regenerate this file
- `python3 scripts/docs/md2html.py docs/tracker/plan.md docs/tracker/plan.html` (and the same for `docs/requirements.md`): regenerate the HTML views
