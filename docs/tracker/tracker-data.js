// go-build-it delivery tracker data. Edit task "status" values here
// (todo | in-progress | blocked | done) and update "updated". The page reads this file.
window.TRACKER = {
 "project": "go-build-it",
 "updated": "2026-11-09",
 "sizes": {
  "S": 1,
  "M": 3,
  "L": 8
 },
 "phases": [
  {
   "id": "P0",
   "name": "Foundations & decisions",
   "mvp": true,
   "goal": "Lock the requirements and UI baseline, answer the questions that block building, and set up the repo, CI, standards and design kit.",
   "gate": "Decisions review",
   "gateCriteria": [
    "Approach A confirmed as UI baseline",
    "Q2 spike results recorded; adapters scoped",
    "Q1, Q5, Q6 decided (Q4 can wait for P9)",
    "Repo builds a CalVer-stamped binary in CI"
   ],
   "gateApproved": "2026-11-08",
   "gateApprovedBy": "david (owner)"
  },
  {
   "id": "P1",
   "name": "Core platform (walking skeleton)",
   "mvp": true,
   "goal": "One Go binary with embedded React: config, SQLite, API, CLI parity, access classification, Tailscale Serve, live updates, history store, desktop shell.",
   "gate": "Skeleton demo",
   "gateCriteria": [
    "Install, open on localhost and over Tailscale Serve with the right identity badge",
    "Every API call has a CLI command with --json",
    "History records every write with who/what/when/why"
   ],
   "gateApproved": "2026-11-08",
   "gateApprovedBy": "david (owner)"
  },
  {
   "id": "P2",
   "name": "Projects, catalog & artifacts",
   "mvp": true,
   "goal": "Manage real projects: catalog cards, project overview, roadmap and item detail, artifacts with git revisions, values and problem definition.",
   "gate": "Catalog review",
   "gateCriteria": [
    "6 real repos imported and shown correctly",
    "Release, website, CI and unpushed-work data on every card",
    "Roadmap items open their detail page"
   ],
   "gateApproved": "2026-11-08",
   "gateApprovedBy": "david (owner)"
  },
  {
   "id": "P3",
   "name": "Team, standards & harness engine",
   "mvp": true,
   "goal": "Personas and standards feed headless harness sessions; the queue dispatches tasks; context is monitored; compaction and rate limits are handled.",
   "gate": "Adapter demo",
   "gateCriteria": [
    "Claude Code and OpenCode each finish a task headlessly (Antigravity if Q2 allows)",
    "Context % visible live, marked measured or estimated",
    "Compaction logged; rate-limit pause and auto-resume shown",
    "Queue survives a restart"
   ],
   "gateApproved": "2026-11-08",
   "gateApprovedBy": "david (owner)"
  },
  {
   "id": "P4",
   "name": "Courier: git & release",
   "mvp": true,
   "goal": "The Git / Release Manager runs the workflow from the Git workflow standard: branches, PRs, CI, merges, CalVer releases and Pages, with no LLM unless judgment is needed.",
   "gate": "Release rehearsal",
   "gateCriteria": [
    "Task → branch → PR → merge → tag → Pages, end to end on a sample project",
    "Nothing completed is left unpushed",
    "From here go-build-it can manage its own remaining phases (dogfooding)"
   ],
   "gateApproved": "2026-11-08",
   "gateApprovedBy": "david (owner)"
  },
  {
   "id": "P5",
   "name": "Human-in-the-loop (MVP)",
   "mvp": true,
   "goal": "Questions, attachment viewer, dictation, assessment-style work review, review gates and dress rehearsals, option sets, and agent-assisted editing of artifacts, personas and standards.",
   "gate": "MVP dress rehearsal",
   "gateCriteria": [
    "Full desktop UAT walkthrough passes for the Owner role",
    "A real feature goes option set → rehearsal → build → assessment review → release",
    "MVP release tagged"
   ],
   "gateApproved": "2026-11-08",
   "gateApprovedBy": "david (owner)"
  },
  {
   "id": "P6",
   "name": "Mobile web UI",
   "mvp": false,
   "goal": "The separate phone UI from Approach A mobile: projects, inbox, answer, review, rehearsal, options, sessions with terminal, viewer, more.",
   "gate": "One-handed UAT",
   "gateCriteria": [
    "Answer a blocking question, approve a review and give rehearsal feedback one-handed in under a minute each",
    "Phone e2e screenshots in docs"
   ],
   "gateApproved": "2026-11-08",
   "gateApprovedBy": "david (owner)"
  },
  {
   "id": "P7",
   "name": "Observability & self-improvement",
   "mvp": false,
   "goal": "History UI and event detail pages, metrics agents can analyze, and improvement proposals that go through normal review.",
   "gate": "First retro",
   "gateCriteria": [
    "Rebuild any task's story from history alone",
    "An agent proposes an improvement from real metrics and it is reviewed"
   ],
   "gateApproved": "2026-11-08",
   "gateApprovedBy": "david (owner)"
  },
  {
   "id": "P8",
   "name": "Docs, UAT, updates & integrations",
   "mvp": false,
   "goal": "Embedded searchable docs with auto-captured screenshots, the per-role UAT walkthrough, finished settings, auto-update with rollback, MCP server.",
   "gate": "v1 UAT",
   "gateCriteria": [
    "UAT scripts for every role pass and regenerate docs screenshots",
    "Auto-update and rollback demonstrated",
    "External agent drives the app through MCP"
   ],
   "gateApproved": "2026-11-08",
   "gateApprovedBy": "david (owner)"
  },
  {
   "id": "P9",
   "name": "Experimental & expansion",
   "mvp": false,
   "goal": "Features behind flags or after v1: smart handoffs, attachable headed sessions, Pi-agent, Linux and Windows, external access challenge, multi-user.",
   "gate": "Graduation reviews",
   "gateCriteria": [
    "Each experimental feature has evidence (NFR-12 metrics) before it becomes default",
    "Linux and Windows pass the UAT suite"
   ],
   "gateApproved": "2026-11-08",
   "gateApprovedBy": "david (owner)"
  },
  {
   "id": "P10",
   "name": "herdr direction: decisions, spike & design",
   "mvp": false,
   "goal": "Record rev 3 (herdr as the only backend, per-project team members from archetypes), prove the herdr integration with a spike, decide the open questions, and redesign the affected mockups before building.",
   "gate": "herdr direction review",
   "gateCriteria": [
    "Requirements rev 3, ADR-009 and ADR-010 approved by the owner",
    "Spike: a Claude member in the gbi herdr session takes a prompt, finishes, reports its conversation ID and usage, and streams live to a browser",
    "Q8 and Q10 decided (Q9 can wait for P15)",
    "Rev 3 mockups (desktop, mobile, herdr layout) reviewed"
   ],
   "gateApproved": "2026-11-08",
   "gateApprovedBy": "david (owner)",
   "gateEvidence": "Requirements rev 3 and ADR-009/010 merged (#4); spike passed end to end (ADR-009 findings); Q8–Q10 decided by owner; rev 3 mockups desktop 05/05b/10/10b/13/15 + 01/02/06 and mobile M2/M3/M8/M10; layout spec and upstream baseline written."
  },
  {
   "id": "P11",
   "name": "herdr runtime foundation",
   "mvp": false,
   "goal": "go-build-it owns and supervises the gbi herdr session: generated client, version and capability checks, onboarding and integrations, login service, event reconciliation, fake harness, CI with herdr on every platform.",
   "gate": "Runtime demo",
   "gateCriteria": [
    "Login starts go-build-it and the gbi herdr session; restarting go-build-it leaves agents running",
    "Doctor shows herdr version, session and integration status, and fixes outdated integrations",
    "CI is green on macOS, Linux and Windows with a pinned herdr"
   ],
   "gateApproved": "2026-11-08",
   "gateApprovedBy": "david (owner)",
   "gateEvidence": "herdr client generated from the vendored schema (gen -check in make vet); supervisor adopts/starts a detached gbi session and restarts it; doctor + fixes (API, CLI, Settings → herdr setup); watcher with snapshot reconciliation; fake member runs in herdr; live tests pass against an isolated herdr; CI installs pinned herdr on Linux and macOS (required) and Windows (informational, beta), weekly latest-herdr job."
  },
  {
   "id": "P12",
   "name": "Per-project teams from archetypes",
   "mvp": false,
   "goal": "Replace global teams and personas with global persona archetypes and per-project team members that can be tuned, compared with their archetype, and promoted back.",
   "gate": "Team review",
   "gateCriteria": [
    "Every imported project has a team made from starter archetypes",
    "A member's tuning, its difference from the archetype, and an archetype change applied per member are shown",
    "Each member's instruction file is generated in its worktree"
   ],
   "gateApproved": "2026-11-09",
   "gateApprovedBy": "david (owner)",
   "gateEvidence": "Archetypes with revisions and starter archetypes; per-project members (forge@project) made from archetypes; tuning, effective persona, diff, change review with per-member and bulk apply (Q10), promote; humans per project; starter team on import; Archetypes library and project Team tab (A-v10, A-v10b), team on card/overview/mobile; task assignment to a specific member. Team unit tests, app suite and 28 e2e tests pass."
  },
  {
   "id": "P13",
   "name": "Team members on herdr",
   "mvp": false,
   "goal": "Run every AI team member as a long-running harness in its herdr pane: profiles, pane lifecycle, dispatch, transcripts, status, limits, conversation rotation, reconnection and labels. Remove tmux and headless runs.",
   "gate": "Member demo",
   "gateCriteria": [
    "Claude Code, Antigravity and Pi members each finish a task end to end through herdr, reviewed by Lens",
    "A blocked member shows as needing the human; herdr restart resumes members in their conversations",
    "herdr --session gbi shows every project and member labelled with role, task and context %",
    "No tmux or stream-json code remains"
   ],
   "gateApproved": "2026-11-09",
   "gateApprovedBy": "david (owner)",
   "gateEvidence": "Members run as long-lived harnesses in herdr panes, one worktree each: profiles for Claude Code, Antigravity, Pi, OpenCode; transcript readers (Claude, Antigravity, Pi) with measured usage; dispatch through .gbi-task.md and Submit/WaitSettled; blocked → needs-you (inbox, card); rate limits; conversation continuity and rotation at 75% with member notes (Q8); reattach after restart; herdr labels and workspace summary; helper conversations for chat/Lens in herdr; tmux, attachable flag and headless stream-json removed. Real run: Claude Code and Antigravity finished end to end; Pi ran mechanically end to end but its configured local model returned empty output (paid OpenRouter models not used without asking). Go suite, 3 new member tests and 28 e2e tests pass."
  },
  {
   "id": "P14",
   "name": "Live terminal & UI",
   "mvp": false,
   "goal": "Watch and steer any team member from the desktop and mobile UI through herdr's terminal streams, and bring herdr state into the live views, Inbox, settings and history.",
   "gate": "Watch & steer UAT",
   "gateCriteria": [
    "Watch any member live on desktop and phone; take control and hand it back",
    "Blocked members appear in the Questions dashboard and mobile Inbox",
    "Every herdr state change, rotation and takeover is in the history"
   ]
  },
  {
   "id": "P15",
   "name": "Rev 3 docs, UAT & upstream watch",
   "mvp": false,
   "goal": "Update the embedded docs, UAT scripts and screenshots for the herdr flow, schedule the weekly herdr/Ovis upstream review, and settle Windows support.",
   "gate": "Rev 3 UAT",
   "gateCriteria": [
    "UAT scripts for every role pass on the herdr flow and regenerate screenshots",
    "The first scheduled upstream review has run and recorded its findings",
    "Windows support level decided (Q9) and documented"
   ]
  }
 ],
 "tasks": [
  {
   "id": "P0-01",
   "phase": "P0",
   "title": "Requirements draft (rev 2, incl. mockup feedback)",
   "reqs": [
    "FR-01"
   ],
   "views": [],
   "size": "M",
   "deps": [],
   "status": "done",
   "done": "docs/requirements.md rev 2 committed; IDs stable.",
   "q": []
  },
  {
   "id": "P0-02",
   "phase": "P0",
   "title": "Three UI approaches mocked; Approach A reviewed",
   "reqs": [
    "FR-34",
    "FR-35"
   ],
   "views": [
    "A-all"
   ],
   "size": "L",
   "deps": [
    "P0-01"
   ],
   "status": "done",
   "done": "docs/mockups rev 2 with overview and change log.",
   "q": []
  },
  {
   "id": "P0-03",
   "phase": "P0",
   "title": "Confirm Approach A as baseline; list aspects borrowed from B/C",
   "reqs": [
    "FR-34",
    "FR-35"
   ],
   "views": [
    "A-all"
   ],
   "size": "S",
   "deps": [
    "P0-02"
   ],
   "status": "done",
   "done": "Decision recorded in docs/decisions/ADR-001-ui-approach.md.",
   "q": []
  },
  {
   "id": "P0-04",
   "phase": "P0",
   "title": "Harness spike: headless + usage reporting for Claude Code, OpenCode, Antigravity",
   "reqs": [
    "FR-60",
    "FR-61",
    "FR-62",
    "FR-71"
   ],
   "views": [],
   "size": "M",
   "deps": [],
   "status": "done",
   "done": "Each harness: start headless, stream events, read context usage (measured or estimated), detect rate limits. Findings recorded against Q2.",
   "q": [
    "Q2"
   ]
  },
  {
   "id": "P0-05",
   "phase": "P0",
   "title": "Decide default git workflow; write the Git workflow standard",
   "reqs": [
    "FR-91"
   ],
   "views": [
    "A-v11"
   ],
   "size": "S",
   "deps": [],
   "status": "done",
   "done": "Branch naming, PR/MR rules, merge strategy, tagging written as a global standard.",
   "q": [
    "Q5"
   ]
  },
  {
   "id": "P0-06",
   "phase": "P0",
   "title": "Decide auto-update channel and signing",
   "reqs": [
    "FR-06"
   ],
   "views": [],
   "size": "S",
   "deps": [],
   "status": "done",
   "done": "Channel (GitHub Releases, Homebrew tap…) and signature scheme chosen.",
   "q": [
    "Q6"
   ]
  },
  {
   "id": "P0-07",
   "phase": "P0",
   "title": "Decide external-interface access for MVP (default: refuse; Tailscale only)",
   "reqs": [
    "FR-18",
    "NFR-21"
   ],
   "views": [],
   "size": "S",
   "deps": [],
   "status": "done",
   "done": "Decision recorded; full challenge deferred to P9-07 if needed.",
   "q": [
    "Q1"
   ]
  },
  {
   "id": "P0-08",
   "phase": "P0",
   "title": "Decide multi-user scope for v1",
   "reqs": [
    "FR-17",
    "FR-117a"
   ],
   "views": [],
   "size": "S",
   "deps": [],
   "status": "done",
   "done": "Single owner + attributed reviewers, or full roles. Affects rehearsal and review assignment.",
   "q": [
    "Q4"
   ]
  },
  {
   "id": "P0-09",
   "phase": "P0",
   "title": "Repo scaffold: Go module, React + TS + Vite, embed, lint, CI, CalVer stamping",
   "reqs": [
    "FR-01",
    "FR-05"
   ],
   "views": [],
   "size": "M",
   "deps": [],
   "status": "done",
   "done": "make build produces one binary with the frontend embedded; CI runs lint, unit and e2e.",
   "q": []
  },
  {
   "id": "P0-10",
   "phase": "P0",
   "title": "Write go-build-it's own global standards (backend, frontend, tests, docs, versioning)",
   "reqs": [
    "FR-50",
    "FR-51"
   ],
   "views": [
    "A-v11"
   ],
   "size": "S",
   "deps": [
    "P0-05"
   ],
   "status": "done",
   "done": "Markdown standards in repo; later imported into the app (dogfooding).",
   "q": []
  },
  {
   "id": "P0-11",
   "phase": "P0",
   "title": "Design tokens and component kit from Approach A (light + dark)",
   "reqs": [
    "FR-34"
   ],
   "views": [
    "A-v01",
    "A-v03b",
    "A-v04b"
   ],
   "size": "M",
   "deps": [
    "P0-09"
   ],
   "status": "done",
   "done": "Sidebar, topbar, table, chip, card, segmented control, editor shell, chat panel, viewer overlay; dark theme.",
   "q": []
  },
  {
   "id": "P0-12",
   "phase": "P0",
   "title": "Playwright e2e harness with desktop + phone screenshot capture",
   "reqs": [
    "FR-133"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P0-09"
   ],
   "status": "done",
   "done": "Every e2e test can save named screenshots at both sizes.",
   "q": []
  },
  {
   "id": "P1-01",
   "phase": "P1",
   "title": "State dir ~/.go-build-it and config loader (cross-platform paths)",
   "reqs": [
    "FR-03",
    "NFR-01"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P0-09"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P1-02",
   "phase": "P1",
   "title": "SQLite storage layer and migrations",
   "reqs": [
    "FR-04"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P1-01"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P1-03",
   "phase": "P1",
   "title": "HTTP API skeleton, OpenAPI spec, generated TS client",
   "reqs": [
    "FR-31"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P1-02"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P1-04",
   "phase": "P1",
   "title": "CLI gbi as an API client with --json; parity test",
   "reqs": [
    "FR-30",
    "FR-32"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P1-03"
   ],
   "status": "done",
   "done": "A test fails if any API operation lacks a CLI command.",
   "q": []
  },
  {
   "id": "P1-05",
   "phase": "P1",
   "title": "Request origin classifier and identity attribution",
   "reqs": [
    "FR-15",
    "FR-16",
    "FR-17"
   ],
   "views": [
    "A-v13"
   ],
   "size": "S",
   "deps": [
    "P1-03"
   ],
   "status": "done",
   "done": "Local, Tailscale Serve (with login) and external classified; identity badge in UI.",
   "q": []
  },
  {
   "id": "P1-06",
   "phase": "P1",
   "title": "Loopback-only default; external refused unless configured",
   "reqs": [
    "FR-19",
    "FR-18",
    "NFR-21"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P1-05",
    "P0-07"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P1-07",
   "phase": "P1",
   "title": "Tailscale Serve detection and guided setup",
   "reqs": [
    "FR-20"
   ],
   "views": [
    "A-v13"
   ],
   "size": "M",
   "deps": [
    "P1-05"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P1-08",
   "phase": "P1",
   "title": "Live updates channel (SSE) and client store",
   "reqs": [
    "FR-36"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P1-03"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P1-09",
   "phase": "P1",
   "title": "Desktop shell: sidebar, topbar, identity badge, routing, ⌘K stub",
   "reqs": [
    "FR-34"
   ],
   "views": [
    "A-v01"
   ],
   "size": "M",
   "deps": [
    "P0-11",
    "P1-03"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P1-10",
   "phase": "P1",
   "title": "History event store used by every write (who/what/when/why/origin)",
   "reqs": [
    "NFR-10"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P1-02"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P1-11",
   "phase": "P1",
   "title": "Background service install (launchd) and gbi service commands",
   "reqs": [
    "FR-02"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P1-04"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P1-12",
   "phase": "P1",
   "title": "Release build: single binary, CalVer tag, checksums, signature",
   "reqs": [
    "FR-01",
    "FR-05"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P0-06",
    "P0-09"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P2-01",
   "phase": "P2",
   "title": "Project model, import existing repo, lifecycle states",
   "reqs": [
    "FR-43"
   ],
   "views": [
    "A-v01"
   ],
   "size": "M",
   "deps": [
    "P1-02"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P2-02",
   "phase": "P2",
   "title": "Forge detection and gh / glab auth check",
   "reqs": [
    "FR-90"
   ],
   "views": [
    "A-v13"
   ],
   "size": "S",
   "deps": [
    "P2-01"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P2-03",
   "phase": "P2",
   "title": "Catalog view: cards, needs-you first, search, filter, sort",
   "reqs": [
    "FR-40",
    "FR-41",
    "FR-42"
   ],
   "views": [
    "A-v01",
    "A-M1"
   ],
   "size": "M",
   "deps": [
    "P1-09",
    "P2-01"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P2-04",
   "phase": "P2",
   "title": "Card data: release, commits since, website, CI, PRs/issues, placeholders",
   "reqs": [
    "FR-41",
    "FR-41a",
    "FR-44"
   ],
   "views": [
    "A-v01"
   ],
   "size": "M",
   "deps": [
    "P2-02"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P2-05",
   "phase": "P2",
   "title": "Unpushed-work detector and card warning",
   "reqs": [
    "FR-94"
   ],
   "views": [
    "A-v01"
   ],
   "size": "S",
   "deps": [
    "P2-01"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P2-06",
   "phase": "P2",
   "title": "Project overview page",
   "reqs": [
    "FR-41",
    "NFR-11"
   ],
   "views": [
    "A-v02",
    "A-M2"
   ],
   "size": "M",
   "deps": [
    "P2-03"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P2-07",
   "phase": "P2",
   "title": "Roadmap model (milestones, gates) and roadmap item detail page",
   "reqs": [
    "FR-45b",
    "FR-115"
   ],
   "views": [
    "A-v02",
    "A-v02b"
   ],
   "size": "M",
   "deps": [
    "P2-06"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P2-08",
   "phase": "P2",
   "title": "Artifacts browser: repo docs, git revisions, rendered Markdown and HTML mockups",
   "reqs": [
    "FR-45",
    "FR-46"
   ],
   "views": [
    "A-v03"
   ],
   "size": "M",
   "deps": [
    "P2-01"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P2-09",
   "phase": "P2",
   "title": "Problem definition and values with provenance",
   "reqs": [
    "FR-123"
   ],
   "views": [
    "A-v03"
   ],
   "size": "S",
   "deps": [
    "P2-08"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P3-01",
   "phase": "P3",
   "title": "Teams, personas, humans and roles; assign to projects",
   "reqs": [
    "FR-10",
    "FR-11",
    "FR-12",
    "FR-25",
    "FR-26"
   ],
   "views": [
    "A-v10"
   ],
   "size": "M",
   "deps": [
    "P1-03"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P3-02",
   "phase": "P3",
   "title": "Global standards store with project overrides and reasons",
   "reqs": [
    "FR-50",
    "FR-51",
    "FR-52"
   ],
   "views": [
    "A-v11"
   ],
   "size": "M",
   "deps": [
    "P1-03",
    "P0-10"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P3-03",
   "phase": "P3",
   "title": "Generate AGENTS.md / CLAUDE.md / OpenCode instructions per project and persona",
   "reqs": [
    "FR-53"
   ],
   "views": [
    "A-v11"
   ],
   "size": "S",
   "deps": [
    "P3-02",
    "P3-01"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P3-04",
   "phase": "P3",
   "title": "Harness adapter interface (start, events, input, usage, cancel, limits)",
   "reqs": [
    "FR-61"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P0-04"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P3-05",
   "phase": "P3",
   "title": "Claude Code adapter (stream-json)",
   "reqs": [
    "FR-60",
    "FR-62"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P3-04"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P3-06",
   "phase": "P3",
   "title": "OpenCode adapter (run/serve; usage estimate per Q-34)",
   "reqs": [
    "FR-62",
    "FR-71"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P3-04"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P3-07",
   "phase": "P3",
   "title": "Antigravity adapter (scope from Q2 spike)",
   "reqs": [
    "FR-62"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P3-04"
   ],
   "status": "done",
   "done": "",
   "q": [
    "Q2"
   ]
  },
  {
   "id": "P3-08",
   "phase": "P3",
   "title": "Harness detection and settings (UI + CLI)",
   "reqs": [
    "FR-63"
   ],
   "views": [
    "A-v13"
   ],
   "size": "S",
   "deps": [
    "P3-04"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P3-09",
   "phase": "P3",
   "title": "Git worktree per concurrent session",
   "reqs": [
    "FR-64"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P3-04"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P3-10",
   "phase": "P3",
   "title": "Session context preparation (task, reqs, standards, values, handoff)",
   "reqs": [
    "FR-70"
   ],
   "views": [
    "A-v05"
   ],
   "size": "M",
   "deps": [
    "P3-03",
    "P2-09"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P3-11",
   "phase": "P3",
   "title": "Context monitor with measured / estimated flag",
   "reqs": [
    "FR-71"
   ],
   "views": [
    "A-v05"
   ],
   "size": "S",
   "deps": [
    "P3-05",
    "P3-06"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P3-12",
   "phase": "P3",
   "title": "Compaction detection and history logging (MVP default)",
   "reqs": [
    "FR-72"
   ],
   "views": [
    "A-v05",
    "A-v12"
   ],
   "size": "S",
   "deps": [
    "P3-11",
    "P1-10"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P3-13",
   "phase": "P3",
   "title": "Rate-limit detection, pause and auto-resume",
   "reqs": [
    "FR-65"
   ],
   "views": [
    "A-v05"
   ],
   "size": "M",
   "deps": [
    "P3-05"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P3-14",
   "phase": "P3",
   "title": "Task queue: priorities, deps, role routing, concurrency, retries, persistence",
   "reqs": [
    "FR-80",
    "FR-81",
    "FR-82"
   ],
   "views": [
    "A-v06"
   ],
   "size": "L",
   "deps": [
    "P1-02"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P3-15",
   "phase": "P3",
   "title": "Dispatcher: pick task, start session as persona, resume after restart",
   "reqs": [
    "FR-60",
    "FR-81",
    "FR-82"
   ],
   "views": [
    "A-v06"
   ],
   "size": "M",
   "deps": [
    "P3-14",
    "P3-10"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P3-16",
   "phase": "P3",
   "title": "Sessions view: table, structured live output, terminal (raw) view",
   "reqs": [
    "FR-66",
    "NFR-11"
   ],
   "views": [
    "A-v05",
    "A-v05b"
   ],
   "size": "M",
   "deps": [
    "P3-15",
    "P1-08"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P3-17",
   "phase": "P3",
   "title": "PM sizing: flag tasks with repeated compactions; propose split",
   "reqs": [
    "FR-75"
   ],
   "views": [
    "A-v06"
   ],
   "size": "S",
   "deps": [
    "P3-12",
    "P3-14"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P4-01",
   "phase": "P4",
   "title": "Workflow engine from the Git workflow standard (with overrides)",
   "reqs": [
    "FR-91"
   ],
   "views": [
    "A-v11"
   ],
   "size": "M",
   "deps": [
    "P3-02",
    "P0-05"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P4-02",
   "phase": "P4",
   "title": "Deterministic git ops: branch, commit, push, PR/MR, CI status",
   "reqs": [
    "FR-92",
    "FR-93"
   ],
   "views": [
    "A-v09"
   ],
   "size": "M",
   "deps": [
    "P4-01",
    "P2-02"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P4-03",
   "phase": "P4",
   "title": "Agent-assisted ops: merge conflicts, release notes",
   "reqs": [
    "FR-92",
    "FR-93"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P4-02",
    "P3-15"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P4-04",
   "phase": "P4",
   "title": "Push guarantee and reconciliation loop",
   "reqs": [
    "FR-94"
   ],
   "views": [
    "A-v01"
   ],
   "size": "S",
   "deps": [
    "P4-02"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P4-05",
   "phase": "P4",
   "title": "Managed-project releases: CalVer tag and forge release",
   "reqs": [
    "FR-92",
    "FR-44"
   ],
   "views": [
    "A-v02"
   ],
   "size": "M",
   "deps": [
    "P4-02"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P4-06",
   "phase": "P4",
   "title": "Pages publishing (GitHub / GitLab) and stored URL",
   "reqs": [
    "FR-44"
   ],
   "views": [
    "A-v01",
    "A-v02"
   ],
   "size": "M",
   "deps": [
    "P4-05"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P4-07",
   "phase": "P4",
   "title": "Artifact revisions committed by Courier",
   "reqs": [
    "FR-46",
    "FR-45a"
   ],
   "views": [
    "A-v03b"
   ],
   "size": "S",
   "deps": [
    "P4-02"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P5-01",
   "phase": "P5",
   "title": "Questions: blocking / non-blocking, assumptions, answers routed to sessions",
   "reqs": [
    "FR-100",
    "FR-101",
    "FR-105"
   ],
   "views": [
    "A-v04"
   ],
   "size": "M",
   "deps": [
    "P3-15"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P5-02",
   "phase": "P5",
   "title": "Questions dashboard: triage list, detail, structured answers, propose value",
   "reqs": [
    "FR-102",
    "FR-103",
    "FR-104"
   ],
   "views": [
    "A-v04",
    "A-M3",
    "A-M4"
   ],
   "size": "M",
   "deps": [
    "P5-01"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P5-03",
   "phase": "P5",
   "title": "Attachment and mockup viewer: live HTML with viewports, images, diffs, logs, pins",
   "reqs": [
    "FR-102a",
    "FR-116a"
   ],
   "views": [
    "A-v04b",
    "A-M11"
   ],
   "size": "L",
   "deps": [
    "P0-11"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P5-04",
   "phase": "P5",
   "title": "Voice dictation component with editable transcript",
   "reqs": [
    "FR-107"
   ],
   "views": [
    "A-v04",
    "A-v09"
   ],
   "size": "S",
   "deps": [
    "P0-11"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P5-05",
   "phase": "P5",
   "title": "Lens assessment: standards, requirements, feedback addressed",
   "reqs": [
    "FR-54",
    "FR-112"
   ],
   "views": [
    "A-v09"
   ],
   "size": "M",
   "deps": [
    "P3-15",
    "P3-02"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P5-06",
   "phase": "P5",
   "title": "Mockup vs built comparison from e2e screenshots (side by side, overlay, slider)",
   "reqs": [
    "FR-112"
   ],
   "views": [
    "A-v09"
   ],
   "size": "M",
   "deps": [
    "P0-12",
    "P5-03"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P5-07",
   "phase": "P5",
   "title": "Work review page; verdicts feed follow-up task context",
   "reqs": [
    "FR-110",
    "FR-111",
    "FR-112"
   ],
   "views": [
    "A-v09",
    "A-M5"
   ],
   "size": "M",
   "deps": [
    "P5-05",
    "P5-06",
    "P4-02"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P5-08",
   "phase": "P5",
   "title": "Review gates block downstream work until approved",
   "reqs": [
    "FR-115",
    "FR-118"
   ],
   "views": [
    "A-v02b"
   ],
   "size": "S",
   "deps": [
    "P2-07",
    "P3-14"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P5-09",
   "phase": "P5",
   "title": "Dress rehearsal runner: steps, findings by several reviewers, live agent changes, gate decision",
   "reqs": [
    "FR-116",
    "FR-117",
    "FR-117a",
    "FR-118"
   ],
   "views": [
    "A-v08",
    "A-M6"
   ],
   "size": "L",
   "deps": [
    "P5-08",
    "P5-03"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P5-10",
   "phase": "P5",
   "title": "Option sets: generate against values, large preview, compare, combination tray",
   "reqs": [
    "FR-120",
    "FR-121",
    "FR-122",
    "FR-124"
   ],
   "views": [
    "A-v07",
    "A-M7"
   ],
   "size": "L",
   "deps": [
    "P5-03",
    "P2-09"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P5-11",
   "phase": "P5",
   "title": "Values and problem definition refined from answers, choices, feedback",
   "reqs": [
    "FR-123",
    "FR-105"
   ],
   "views": [
    "A-v03"
   ],
   "size": "S",
   "deps": [
    "P5-02",
    "P5-10"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P5-12",
   "phase": "P5",
   "title": "Artifact editor: WYSIWYG + Markdown, agent chat with inline proposals, save → revision",
   "reqs": [
    "FR-45a"
   ],
   "views": [
    "A-v03b"
   ],
   "size": "L",
   "deps": [
    "P4-07",
    "P3-15"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P5-13",
   "phase": "P5",
   "title": "Persona coach and rich persona editor",
   "reqs": [
    "FR-13"
   ],
   "views": [
    "A-v10"
   ],
   "size": "M",
   "deps": [
    "P5-12",
    "P3-01"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P5-14",
   "phase": "P5",
   "title": "Standards editing through the artifact editor with agent chat",
   "reqs": [
    "FR-51"
   ],
   "views": [
    "A-v11"
   ],
   "size": "S",
   "deps": [
    "P5-12",
    "P3-02"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P5-15",
   "phase": "P5",
   "title": "Needs-you aggregation: badges and overview card across questions, reviews, gates, options",
   "reqs": [
    "FR-104"
   ],
   "views": [
    "A-v02",
    "A-v01"
   ],
   "size": "S",
   "deps": [
    "P5-02",
    "P5-07",
    "P5-08",
    "P5-10"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P6-01",
   "phase": "P6",
   "title": "Mobile shell: routing and 4 bottom tabs",
   "reqs": [
    "FR-35"
   ],
   "views": [
    "A-M1"
   ],
   "size": "M",
   "deps": [
    "P1-09"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P6-02",
   "phase": "P6",
   "title": "Projects and project status screens",
   "reqs": [
    "FR-35",
    "FR-40",
    "FR-41"
   ],
   "views": [
    "A-M1",
    "A-M2"
   ],
   "size": "S",
   "deps": [
    "P6-01",
    "P2-06"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P6-03",
   "phase": "P6",
   "title": "Inbox and answer screen with dictation",
   "reqs": [
    "FR-104",
    "FR-107"
   ],
   "views": [
    "A-M3",
    "A-M4"
   ],
   "size": "M",
   "deps": [
    "P6-01",
    "P5-02",
    "P5-04"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P6-04",
   "phase": "P6",
   "title": "Review screen (assessment + verdict)",
   "reqs": [
    "FR-110",
    "FR-112"
   ],
   "views": [
    "A-M5"
   ],
   "size": "S",
   "deps": [
    "P6-01",
    "P5-07"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P6-05",
   "phase": "P6",
   "title": "Rehearsal step and option set screens",
   "reqs": [
    "FR-117",
    "FR-122"
   ],
   "views": [
    "A-M6",
    "A-M7"
   ],
   "size": "M",
   "deps": [
    "P5-09",
    "P5-10"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P6-06",
   "phase": "P6",
   "title": "Sessions and terminal view with key bar",
   "reqs": [
    "FR-66"
   ],
   "views": [
    "A-M8",
    "A-M10"
   ],
   "size": "M",
   "deps": [
    "P3-16"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P6-07",
   "phase": "P6",
   "title": "Full-screen viewer on phone",
   "reqs": [
    "FR-102a",
    "FR-116a"
   ],
   "views": [
    "A-M11"
   ],
   "size": "S",
   "deps": [
    "P5-03"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P6-08",
   "phase": "P6",
   "title": "More: read-only areas and chat-based edits",
   "reqs": [
    "FR-35"
   ],
   "views": [
    "A-M9"
   ],
   "size": "S",
   "deps": [
    "P6-01"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P6-09",
   "phase": "P6",
   "title": "PWA install; spike web push (out-of-app notifications stay out of scope)",
   "reqs": [
    "FR-106"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P6-01"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P7-01",
   "phase": "P7",
   "title": "History UI: search, filters, metric tiles",
   "reqs": [
    "NFR-10",
    "NFR-11"
   ],
   "views": [
    "A-v12"
   ],
   "size": "M",
   "deps": [
    "P1-10"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P7-02",
   "phase": "P7",
   "title": "Event detail views per type, human actions included",
   "reqs": [
    "NFR-14"
   ],
   "views": [
    "A-v12b"
   ],
   "size": "M",
   "deps": [
    "P7-01"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P7-03",
   "phase": "P7",
   "title": "Metrics pipeline: rework, answer time, time per task, compactions, limits",
   "reqs": [
    "NFR-12"
   ],
   "views": [
    "A-v12"
   ],
   "size": "M",
   "deps": [
    "P1-10"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P7-04",
   "phase": "P7",
   "title": "Agent improvement proposals through normal review",
   "reqs": [
    "NFR-13"
   ],
   "views": [
    "A-v12"
   ],
   "size": "M",
   "deps": [
    "P7-03",
    "P5-07"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P7-05",
   "phase": "P7",
   "title": "Agent-readable history export / analysis API",
   "reqs": [
    "NFR-12"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P7-03"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P8-01",
   "phase": "P8",
   "title": "Embedded docs: Markdown, search, contextual ? Docs links",
   "reqs": [
    "FR-130"
   ],
   "views": [
    "A-v14"
   ],
   "size": "M",
   "deps": [
    "P1-09"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P8-02",
   "phase": "P8",
   "title": "UAT walkthrough per role as Playwright scripts",
   "reqs": [
    "FR-132",
    "FR-133"
   ],
   "views": [
    "A-v14"
   ],
   "size": "L",
   "deps": [
    "P0-12",
    "P5-09"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P8-03",
   "phase": "P8",
   "title": "Screenshot regeneration into docs",
   "reqs": [
    "FR-131",
    "FR-133"
   ],
   "views": [
    "A-v14"
   ],
   "size": "S",
   "deps": [
    "P8-02",
    "P8-01"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P8-04",
   "phase": "P8",
   "title": "Settings page complete: harnesses, access, forges, experimental, updates",
   "reqs": [
    "FR-63",
    "FR-19",
    "FR-20"
   ],
   "views": [
    "A-v13"
   ],
   "size": "M",
   "deps": [
    "P3-08",
    "P1-07"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P8-05",
   "phase": "P8",
   "title": "Auto-update: verify, install when idle, roll back",
   "reqs": [
    "FR-06"
   ],
   "views": [
    "A-v13"
   ],
   "size": "M",
   "deps": [
    "P1-12"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P8-06",
   "phase": "P8",
   "title": "MCP server over the API",
   "reqs": [
    "FR-33"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P1-04"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P8-07",
   "phase": "P8",
   "title": "Security posture check: exposure tests; no sandbox by design",
   "reqs": [
    "NFR-20",
    "NFR-21"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P1-06"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P9-01",
   "phase": "P9",
   "title": "Smart handoffs behind experimental flag: threshold, briefing, Lens check, next session",
   "reqs": [
    "FR-72",
    "FR-73",
    "FR-74"
   ],
   "views": [
    "A-v05"
   ],
   "size": "L",
   "deps": [
    "P3-12",
    "P5-05"
   ],
   "status": "done",
   "done": "",
   "q": [
    "Q3"
   ]
  },
  {
   "id": "P9-02",
   "phase": "P9",
   "title": "Handoff vs compaction evaluation and graduation criteria",
   "reqs": [
    "FR-72",
    "NFR-12"
   ],
   "views": [
    "A-v12"
   ],
   "size": "M",
   "deps": [
    "P9-01",
    "P7-03"
   ],
   "status": "done",
   "done": "",
   "q": [
    "Q3"
   ]
  },
  {
   "id": "P9-03",
   "phase": "P9",
   "title": "Attachable headed sessions: PTY/multiplexer, attach, detach, control handback",
   "reqs": [
    "FR-67"
   ],
   "views": [
    "A-v05b",
    "A-M10"
   ],
   "size": "L",
   "deps": [
    "P3-16"
   ],
   "status": "done",
   "done": "",
   "q": [
    "Q7"
   ]
  },
  {
   "id": "P9-04",
   "phase": "P9",
   "title": "Pi-agent adapter",
   "reqs": [
    "FR-62"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P3-04"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P9-05",
   "phase": "P9",
   "title": "Linux support (systemd)",
   "reqs": [
    "NFR-01",
    "FR-02"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P1-11"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P9-06",
   "phase": "P9",
   "title": "Windows support (service, paths)",
   "reqs": [
    "NFR-01",
    "FR-02"
   ],
   "views": [],
   "size": "L",
   "deps": [
    "P1-11"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P9-07",
   "phase": "P9",
   "title": "External interface challenge (if Q1 allows external access)",
   "reqs": [
    "FR-18"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P1-06"
   ],
   "status": "done",
   "done": "",
   "q": [
    "Q1"
   ]
  },
  {
   "id": "P9-08",
   "phase": "P9",
   "title": "Multi-user roles and review assignment",
   "reqs": [
    "FR-17",
    "FR-117a"
   ],
   "views": [],
   "size": "L",
   "deps": [
    "P5-09"
   ],
   "status": "done",
   "done": "",
   "q": [
    "Q4"
   ]
  },
  {
   "id": "P10-01",
   "phase": "P10",
   "title": "Requirements rev 3: herdr backend, per-project team members, upstream tracking",
   "reqs": [
    "FR-01",
    "FR-07"
   ],
   "views": [],
   "size": "M",
   "deps": [],
   "status": "done",
   "done": "docs/requirements.md rev 3; Q7 answered, Q8–Q10 added.",
   "q": []
  },
  {
   "id": "P10-02",
   "phase": "P10",
   "title": "ADR-009: herdr is the only agent backend; ADR-008 superseded",
   "reqs": [
    "FR-60",
    "FR-07"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P10-01"
   ],
   "status": "done",
   "done": "docs/decisions/ADR-009-herdr-backend.md",
   "q": []
  },
  {
   "id": "P10-03",
   "phase": "P10",
   "title": "ADR-010: per-project team members from global persona archetypes",
   "reqs": [
    "FR-11",
    "FR-12"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P10-01"
   ],
   "status": "done",
   "done": "docs/decisions/ADR-010-project-team-members.md",
   "q": []
  },
  {
   "id": "P10-04",
   "phase": "P10",
   "title": "Spike: gbi herdr session, agent start + prompt --wait, agent_session → transcript usage, terminal observe → xterm.js",
   "reqs": [
    "FR-60",
    "FR-61",
    "FR-66",
    "FR-71"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P10-02"
   ],
   "status": "done",
   "done": "Spike notes appended to ADR-009; throwaway code not merged.",
   "q": []
  },
  {
   "id": "P10-05",
   "phase": "P10",
   "title": "Decide conversation rotation policy from spike cost data",
   "reqs": [
    "FR-76"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P10-04"
   ],
   "status": "done",
   "done": "",
   "q": [
    "Q8"
   ]
  },
  {
   "id": "P10-06",
   "phase": "P10",
   "title": "Decide how archetype changes reach project members",
   "reqs": [
    "FR-12b"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P10-03"
   ],
   "status": "done",
   "done": "",
   "q": [
    "Q10"
   ]
  },
  {
   "id": "P10-07",
   "phase": "P10",
   "title": "Mockups rev 3, desktop: live members, live terminal, archetypes, project team, herdr settings and setup, card and overview team panel, queue assignment",
   "reqs": [
    "FR-34",
    "FR-12a",
    "FR-66",
    "FR-67"
   ],
   "views": [
    "A-v01",
    "A-v02",
    "A-v05",
    "A-v05b",
    "A-v06",
    "A-v10",
    "A-v10b",
    "A-v13",
    "A-v15"
   ],
   "size": "L",
   "deps": [
    "P10-02",
    "P10-03"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P10-08",
   "phase": "P10",
   "title": "Mockups rev 3, mobile: members, terminal control and takeover, project team, blocked members in Inbox",
   "reqs": [
    "FR-35",
    "FR-66"
   ],
   "views": [
    "A-M2",
    "A-M3",
    "A-M8",
    "A-M10"
   ],
   "size": "M",
   "deps": [
    "P10-07"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P10-09",
   "phase": "P10",
   "title": "herdr layout spec: what herdr --session gbi shows (workspaces, grouped worktrees, pane labels, tokens, summaries, notifications)",
   "reqs": [
    "FR-67"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P10-04"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P10-10",
   "phase": "P10",
   "title": "Upstream baseline: docs/upstream/ with herdr and Ovis revisions reviewed and findings",
   "reqs": [
    "NFR-30"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P10-02"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P10-11",
   "phase": "P10",
   "title": "Docs tooling in the repo: tracker, plan and HTML generators under scripts/docs",
   "reqs": [],
   "views": [],
   "size": "S",
   "deps": [],
   "status": "done",
   "done": "scripts/docs/{add_rev3_phases,genmd,md2html,setstatus}.py",
   "q": []
  },
  {
   "id": "P11-01",
   "phase": "P11",
   "title": "herdr client: socket/named-pipe NDJSON transport, types generated from herdr api schema, ping capability check, supported version range",
   "reqs": [
    "NFR-32",
    "FR-07"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P10-04"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P11-02",
   "phase": "P11",
   "title": "Supervise herdr server --session gbi: start, health, restart, stop",
   "reqs": [
    "FR-07"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P11-01"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P11-03",
   "phase": "P11",
   "title": "Doctor and onboarding: herdr install and version, integration status, install or update integrations",
   "reqs": [
    "FR-07"
   ],
   "views": [
    "A-v15",
    "A-v13"
   ],
   "size": "M",
   "deps": [
    "P11-01",
    "P10-07"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P11-04",
   "phase": "P11",
   "title": "Login service brings up go-build-it and the gbi session; go-build-it restarts and updates leave agents running",
   "reqs": [
    "FR-08",
    "FR-02",
    "FR-06"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P11-02"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P11-05",
   "phase": "P11",
   "title": "Event subscription and snapshot reconciler with events_lost recovery",
   "reqs": [
    "NFR-11",
    "FR-68"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P11-01"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P11-06",
   "phase": "P11",
   "title": "Fake harness in herdr: reports state with pane.report_agent and writes a transcript",
   "reqs": [
    "FR-61"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P11-01"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P11-07",
   "phase": "P11",
   "title": "CI installs a pinned herdr on macOS, Linux and Windows; weekly job against latest herdr",
   "reqs": [
    "NFR-32",
    "NFR-01"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P11-06"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P12-01",
   "phase": "P12",
   "title": "Schema: archetypes and project_members replace global teams and personas",
   "reqs": [
    "FR-11",
    "FR-12",
    "FR-12d",
    "FR-12e"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P10-03"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P12-02",
   "phase": "P12",
   "title": "Starter archetype per built-in role",
   "reqs": [
    "FR-12",
    "FR-25"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P12-01"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P12-03",
   "phase": "P12",
   "title": "API, CLI and MCP: archetypes, project members (add, tune, remove), human roles per project",
   "reqs": [
    "FR-30",
    "FR-12a",
    "FR-12e"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P12-01"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P12-04",
   "phase": "P12",
   "title": "Archetype change review: affected members, per-member apply keeping tuning",
   "reqs": [
    "FR-12b"
   ],
   "views": [
    "A-v10"
   ],
   "size": "M",
   "deps": [
    "P12-03",
    "P10-06"
   ],
   "status": "done",
   "done": "",
   "q": [
    "Q10"
   ]
  },
  {
   "id": "P12-05",
   "phase": "P12",
   "title": "Promote member tuning to its archetype through the artifact editor",
   "reqs": [
    "FR-12c",
    "FR-13"
   ],
   "views": [
    "A-v03b"
   ],
   "size": "S",
   "deps": [
    "P12-03"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P12-06",
   "phase": "P12",
   "title": "Archetypes library and project Team tab with member detail and tuning",
   "reqs": [
    "FR-13",
    "FR-12a",
    "FR-10"
   ],
   "views": [
    "A-v10",
    "A-v10b"
   ],
   "size": "L",
   "deps": [
    "P12-03",
    "P10-07"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P12-07",
   "phase": "P12",
   "title": "Per-member instruction file generated in the member's worktree",
   "reqs": [
    "FR-53"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P12-01"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P12-08",
   "phase": "P12",
   "title": "Queue assignment to a role or a specific member; dispatcher picks a free member",
   "reqs": [
    "FR-81"
   ],
   "views": [
    "A-v06"
   ],
   "size": "M",
   "deps": [
    "P12-01"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P12-09",
   "phase": "P12",
   "title": "Catalog card and overview show the project team and each member's herdr state",
   "reqs": [
    "FR-41"
   ],
   "views": [
    "A-v01",
    "A-v02",
    "A-M2"
   ],
   "size": "S",
   "deps": [
    "P12-06",
    "P11-05"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P13-01",
   "phase": "P13",
   "title": "Harness profiles: Claude Code, Antigravity, Pi, OpenCode (kind, launch and resume args, transcript reader, limit signals)",
   "reqs": [
    "FR-61",
    "FR-62",
    "FR-63"
   ],
   "views": [],
   "size": "L",
   "deps": [
    "P11-01"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P13-02",
   "phase": "P13",
   "title": "Member pane lifecycle: workspace per project, member worktree opened grouped, agent start, name, release",
   "reqs": [
    "FR-12d",
    "FR-64",
    "FR-67"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P12-01",
    "P11-02"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P13-03",
   "phase": "P13",
   "title": "Dispatch: write .gbi-task.md, agent prompt --wait, hand completion to Courier and Lens",
   "reqs": [
    "FR-60",
    "FR-70"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P13-02",
    "P12-08"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P13-04",
   "phase": "P13",
   "title": "Transcript readers: structured events, messages and usage (measured or estimated) per harness",
   "reqs": [
    "FR-66",
    "FR-71",
    "NFR-10"
   ],
   "views": [],
   "size": "L",
   "deps": [
    "P13-01"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P13-05",
   "phase": "P13",
   "title": "Status mapping: herdr state drives session and task state; blocked becomes a needs-you item",
   "reqs": [
    "FR-68"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P11-05",
    "P13-02"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P13-06",
   "phase": "P13",
   "title": "Rate and usage limit detection with pause and auto-resume",
   "reqs": [
    "FR-65"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P13-04"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P13-07",
   "phase": "P13",
   "title": "Conversation continuity and rotation per policy; smart handoff starts the new conversation",
   "reqs": [
    "FR-69",
    "FR-76",
    "FR-72",
    "FR-73",
    "FR-74"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P13-04",
    "P10-05"
   ],
   "status": "done",
   "done": "",
   "q": [
    "Q8"
   ]
  },
  {
   "id": "P13-08",
   "phase": "P13",
   "title": "Reconnect after herdr or machine restart by conversation ID; queue resumes",
   "reqs": [
    "FR-69",
    "FR-82"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P13-05"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P13-09",
   "phase": "P13",
   "title": "herdr labels: pane metadata (name, role, task, context %), workspace summaries, notifications for gates and questions",
   "reqs": [
    "FR-67"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P13-02",
    "P10-09"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P13-10",
   "phase": "P13",
   "title": "Permission prompts: permission modes, per-project allowlists, optional Claude permission hook as structured approval",
   "reqs": [
    "FR-68"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P13-05"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P13-11",
   "phase": "P13",
   "title": "Lens reviews run in a fresh conversation per review",
   "reqs": [
    "FR-54",
    "FR-112"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P13-03"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P13-12",
   "phase": "P13",
   "title": "Interrupt, cancel and restart a member through herdr",
   "reqs": [
    "FR-61"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P13-02"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P13-13",
   "phase": "P13",
   "title": "Remove tmux mode, attachable_sessions flag and the headless stream-json path",
   "reqs": [
    "FR-60"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P13-03",
    "P13-04"
   ],
   "status": "done",
   "done": "",
   "q": []
  },
  {
   "id": "P14-01",
   "phase": "P14",
   "title": "Terminal bridge: herdr terminal session observe/control → WebSocket with ack-based flow control and takeover",
   "reqs": [
    "FR-66"
   ],
   "views": [],
   "size": "L",
   "deps": [
    "P11-01",
    "P10-04"
   ],
   "status": "todo",
   "done": "",
   "q": []
  },
  {
   "id": "P14-02",
   "phase": "P14",
   "title": "Desktop terminal view with take control and hand back",
   "reqs": [
    "FR-66"
   ],
   "views": [
    "A-v05b"
   ],
   "size": "M",
   "deps": [
    "P14-01",
    "P10-07"
   ],
   "status": "todo",
   "done": "",
   "q": []
  },
  {
   "id": "P14-03",
   "phase": "P14",
   "title": "Mobile terminal with key bar",
   "reqs": [
    "FR-35",
    "FR-66"
   ],
   "views": [
    "A-M10"
   ],
   "size": "M",
   "deps": [
    "P14-01",
    "P10-08"
   ],
   "status": "todo",
   "done": "",
   "q": []
  },
  {
   "id": "P14-04",
   "phase": "P14",
   "title": "Live members view: herdr state, task, context %, structured view",
   "reqs": [
    "NFR-11",
    "FR-66"
   ],
   "views": [
    "A-v05",
    "A-M8"
   ],
   "size": "M",
   "deps": [
    "P13-04",
    "P13-05"
   ],
   "status": "todo",
   "done": "",
   "q": []
  },
  {
   "id": "P14-05",
   "phase": "P14",
   "title": "Blocked members in the Questions dashboard and mobile Inbox",
   "reqs": [
    "FR-68",
    "FR-104"
   ],
   "views": [
    "A-v04",
    "A-M3"
   ],
   "size": "S",
   "deps": [
    "P13-05"
   ],
   "status": "todo",
   "done": "",
   "q": []
  },
  {
   "id": "P14-06",
   "phase": "P14",
   "title": "Open in herdr: copy herdr --session gbi and focus the member's pane",
   "reqs": [
    "FR-67"
   ],
   "views": [
    "A-v05"
   ],
   "size": "S",
   "deps": [
    "P13-02"
   ],
   "status": "todo",
   "done": "",
   "q": []
  },
  {
   "id": "P14-07",
   "phase": "P14",
   "title": "Settings: herdr section (version, session, socket, integrations)",
   "reqs": [
    "FR-63",
    "FR-07"
   ],
   "views": [
    "A-v13"
   ],
   "size": "S",
   "deps": [
    "P11-03"
   ],
   "status": "todo",
   "done": "",
   "q": []
  },
  {
   "id": "P14-08",
   "phase": "P14",
   "title": "History events for herdr state changes, rotations and takeovers",
   "reqs": [
    "NFR-10",
    "NFR-14"
   ],
   "views": [
    "A-v12",
    "A-v12b"
   ],
   "size": "S",
   "deps": [
    "P13-05",
    "P14-01"
   ],
   "status": "todo",
   "done": "",
   "q": []
  },
  {
   "id": "P15-01",
   "phase": "P15",
   "title": "User docs rewrite: install, team, sessions, settings, mobile and CLI for herdr and archetypes",
   "reqs": [
    "FR-130"
   ],
   "views": [
    "A-v14"
   ],
   "size": "M",
   "deps": [
    "P14-04",
    "P12-06"
   ],
   "status": "todo",
   "done": "",
   "q": []
  },
  {
   "id": "P15-02",
   "phase": "P15",
   "title": "UAT scripts for the herdr flow per role; regenerate screenshots",
   "reqs": [
    "FR-132",
    "FR-133",
    "FR-131"
   ],
   "views": [
    "A-v14"
   ],
   "size": "L",
   "deps": [
    "P15-01",
    "P11-06"
   ],
   "status": "todo",
   "done": "",
   "q": []
  },
  {
   "id": "P15-03",
   "phase": "P15",
   "title": "Weekly upstream review job for herdr and Ovis that opens an issue or PR",
   "reqs": [
    "NFR-31"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P10-10"
   ],
   "status": "todo",
   "done": "",
   "q": []
  },
  {
   "id": "P15-04",
   "phase": "P15",
   "title": "Windows at herdr's support level",
   "reqs": [
    "NFR-01"
   ],
   "views": [],
   "size": "M",
   "deps": [
    "P11-07"
   ],
   "status": "todo",
   "done": "",
   "q": [
    "Q9"
   ]
  },
  {
   "id": "P15-05",
   "phase": "P15",
   "title": "Rev 3 close-out: retire obsolete tests and text, mockup change log, coverage check",
   "reqs": [
    "FR-01"
   ],
   "views": [],
   "size": "S",
   "deps": [
    "P15-02"
   ],
   "status": "todo",
   "done": "",
   "q": []
  }
 ],
 "requirements": {
  "FR-01": {
   "text": "The application SHALL be a single Go binary with the React frontend embedded.",
   "section": "3. Platform, installation and runtime"
  },
  "FR-07": {
   "text": "go-build-it SHALL require herdr and SHALL use it as its only agent backend:",
   "section": "3. Platform, installation and runtime"
  },
  "FR-08": {
   "text": "The background service (FR-02) SHALL ensure both go-build-it and its herdr session are running at login.",
   "section": "3. Platform, installation and runtime"
  },
  "FR-02": {
   "text": "The application SHALL be installable globally on a machine and SHOULD run as a background service (launchd on macOS, systemd on Linux, a Windows service on Windows).",
   "section": "3. Platform, installation and runtime"
  },
  "NFR-01": {
   "text": "macOS SHALL be the initial target platform.",
   "section": "3. Platform, installation and runtime"
  },
  "FR-03": {
   "text": "All application state SHALL live under ~/.go-build-it/ (%USERPROFILE%\\.go-build-it\\ on Windows).",
   "section": "3. Platform, installation and runtime"
  },
  "FR-04": {
   "text": "Persistence SHALL use SQLite.",
   "section": "3. Platform, installation and runtime"
  },
  "FR-05": {
   "text": "The application SHALL use calendar versioning in the format YYYY.MM.DD.N, where N is the release number for that day, starting at 1.",
   "section": "3. Platform, installation and runtime"
  },
  "FR-06": {
   "text": "The application SHALL be able to install new releases automatically:",
   "section": "3. Platform, installation and runtime"
  },
  "FR-10": {
   "text": "A team member SHALL be either an AI team member or a human user.",
   "section": "4.1 Team members"
  },
  "FR-11": {
   "text": "Each project SHALL have its own team.",
   "section": "4.1 Team members"
  },
  "FR-12": {
   "text": "The application SHALL maintain global persona archetypes.",
   "section": "4.1 Team members"
  },
  "FR-12a": {
   "text": "An AI team member SHALL be created in a project from an archetype.",
   "section": "4.1 Team members"
  },
  "FR-12b": {
   "text": "When an archetype changes, the application SHALL show which project members it affects and SHALL let the user apply the change to each of them, keeping their project tuning.",
   "section": "4.1 Team members"
  },
  "FR-12c": {
   "text": "Tuning made in a project MAY be promoted back to the archetype, through the artifact editor's normal accept or reject flow.",
   "section": "4.1 Team members"
  },
  "FR-12d": {
   "text": "Each AI team member SHALL be bound to exactly one harness and model, one current harness conversation, one long-lived git worktree, and one herdr pane in the gbi herdr session (§8.",
   "section": "4.1 Team members"
  },
  "FR-12e": {
   "text": "Human users SHALL remain global identities (§4.2).",
   "section": "4.1 Team members"
  },
  "FR-13": {
   "text": "Archetypes and project members' tuning SHALL be edited in the artifact editor (FR-45a).",
   "section": "4.1 Team members"
  },
  "FR-15": {
   "text": "The application SHALL detect how each request reaches it and classify it as one of:",
   "section": "4.2 Access and identity"
  },
  "FR-16": {
   "text": "Local requests SHALL NOT be challenged.",
   "section": "4.2 Access and identity"
  },
  "FR-17": {
   "text": "Tailscale Serve requests SHALL be treated as trusted, the same as local requests, because the user has already authenticated with Tailscale.",
   "section": "4.2 Access and identity"
  },
  "FR-18": {
   "text": "External interface requests SHALL be challenged before access is granted.",
   "section": "4.2 Access and identity"
  },
  "FR-19": {
   "text": "By default the application SHALL listen only on loopback.",
   "section": "4.2 Access and identity"
  },
  "FR-20": {
   "text": "The application SHALL help the user set up Tailscale Serve, for example by detecting whether Tailscale is installed and logged in and then offering to configure tailscale serve.",
   "section": "4.2 Access and identity"
  },
  "FR-25": {
   "text": "The application SHALL provide these built-in roles.",
   "section": "4.3 Roles"
  },
  "FR-26": {
   "text": "Users SHOULD be able to define additional custom roles.",
   "section": "4.3 Roles"
  },
  "FR-30": {
   "text": "Everything that can be done in the web UI SHALL also be possible through the CLI, so AI agents can operate the application.",
   "section": "5. User interfaces"
  },
  "FR-31": {
   "text": "The application SHALL be API-first.",
   "section": "5. User interfaces"
  },
  "FR-32": {
   "text": "CLI commands SHALL support machine-readable output (--json).",
   "section": "5. User interfaces"
  },
  "FR-33": {
   "text": "The application SHOULD expose an MCP server so external agents can drive it.",
   "section": "5. User interfaces"
  },
  "FR-34": {
   "text": "The application SHALL provide a desktop web UI designed for laptop and desktop screens.",
   "section": "5. User interfaces"
  },
  "FR-35": {
   "text": "The application SHALL provide a separate mobile web UI designed for phones.",
   "section": "5. User interfaces"
  },
  "FR-36": {
   "text": "User interactions SHALL feel interactive and dynamic, like working with a team.",
   "section": "5. User interfaces"
  },
  "FR-40": {
   "text": "The landing view SHALL be the project catalog, showing all managed projects as quick-view cards.",
   "section": "6.1 Project catalog"
  },
  "FR-41": {
   "text": "At a glance, each card SHALL show:",
   "section": "6.1 Project catalog"
  },
  "FR-41a": {
   "text": "Clicking a card's release or website link SHALL open the target directly.",
   "section": "6.1 Project catalog"
  },
  "FR-42": {
   "text": "The catalog SHALL support search, filtering (by state, team, or needs-attention) and sorting, because many projects will be managed at once.",
   "section": "6.1 Project catalog"
  },
  "FR-43": {
   "text": "Projects SHALL have a defined lifecycle.",
   "section": "6.1 Project catalog"
  },
  "FR-44": {
   "text": "Each project SHALL have a website hosted on GitHub Pages.",
   "section": "6.1 Project catalog"
  },
  "FR-45": {
   "text": "The team SHALL be able to develop the following with the user for each project:",
   "section": "6.2 Project artifacts"
  },
  "FR-45a": {
   "text": "Each artifact SHALL have an Edit action that opens the artifact editor.",
   "section": "6.2 Project artifacts"
  },
  "FR-45b": {
   "text": "Roadmap items (milestones and review gates) SHALL open a detail page showing:",
   "section": "6.2 Project artifacts"
  },
  "FR-46": {
   "text": "Project artifacts SHOULD be stored in the project's git repository (e.g.",
   "section": "6.2 Project artifacts"
  },
  "FR-50": {
   "text": "The application SHALL maintain a set of global standards that apply to every project.",
   "section": "7. Global standards"
  },
  "FR-51": {
   "text": "Standards SHALL be authored as Markdown and managed in the UI and the CLI.",
   "section": "7. Global standards"
  },
  "FR-52": {
   "text": "A project MAY override or extend a global standard.",
   "section": "7. Global standards"
  },
  "FR-53": {
   "text": "Standards SHALL be supplied to every agent session automatically, for example by generating the harness's instruction file (CLAUDE.md, AGENTS.md or equivalent), or by injecting the",
   "section": "7. Global standards"
  },
  "FR-54": {
   "text": "The Reviewer role SHALL check completed work against the applicable standards.",
   "section": "7. Global standards"
  },
  "FR-60": {
   "text": "The application SHALL run every agent harness interactively in a herdr pane of the gbi herdr session (FR-07), at the direction of the development team.",
   "section": "8.1 Harness integration"
  },
  "FR-61": {
   "text": "Harness support SHALL be implemented through a harness profile that describes, for each harness:",
   "section": "8.1 Harness integration"
  },
  "FR-62": {
   "text": "Initial adapters:",
   "section": "8.1 Harness integration"
  },
  "FR-63": {
   "text": "Harnesses and their settings (binary path, model, flags, permission mode) SHALL be configurable in the UI and the CLI.",
   "section": "8.1 Harness integration"
  },
  "FR-66": {
   "text": "The live view of an AI team member SHALL offer two modes:",
   "section": "8.1 Harness integration"
  },
  "FR-67": {
   "text": "The gbi herdr session SHALL be organized so that opening it shows everything go-build-it is doing:",
   "section": "8.1 Harness integration"
  },
  "FR-64": {
   "text": "Each AI team member SHALL work in its own long-lived git worktree, so concurrent members don't interfere.",
   "section": "8.1 Harness integration"
  },
  "FR-68": {
   "text": "The application SHALL treat herdr's agent state as the source of truth for whether a member is working, blocked, done or idle.",
   "section": "8.1 Harness integration"
  },
  "FR-69": {
   "text": "Each AI team member's conversation SHALL continue across tasks until it is rotated (§8.2).",
   "section": "8.1 Harness integration"
  },
  "FR-65": {
   "text": "Cost is limited by the user's AI subscriptions, so no budget controls are needed.",
   "section": "8.1 Harness integration"
  },
  "FR-70": {
   "text": "The application SHALL prepare each session's starting context deliberately:",
   "section": "8.2 Context management and smart handoffs"
  },
  "FR-71": {
   "text": "The application SHALL monitor each AI team member's context-window usage, read from its harness transcript, and SHALL show whether the value is measured or estimated.",
   "section": "8.2 Context management and smart handoffs"
  },
  "FR-72": {
   "text": "(Experimental.) When smart handoffs are enabled and usage reaches a configurable threshold (e.g.",
   "section": "8.2 Context management and smart handoffs"
  },
  "FR-73": {
   "text": "A handoff SHALL be a curated briefing, written as if handing responsibility to another person.",
   "section": "8.2 Context management and smart handoffs"
  },
  "FR-74": {
   "text": "The next session SHALL start from the handoff and FR-70's prepared context.",
   "section": "8.2 Context management and smart handoffs"
  },
  "FR-76": {
   "text": "A member's lasting knowledge SHALL live in go-build-it (persona, project values, decisions, notes from finished tasks), not only in its conversation.",
   "section": "8.2 Context management and smart handoffs"
  },
  "FR-75": {
   "text": "Tasks SHOULD be sized so most of them fit in one session.",
   "section": "8.2 Context management and smart handoffs"
  },
  "FR-80": {
   "text": "Each project SHALL have a persistent task queue that can manage large numbers of tasks.",
   "section": "8.3 Task queues"
  },
  "FR-81": {
   "text": "Queues SHALL support:",
   "section": "8.3 Task queues"
  },
  "FR-82": {
   "text": "Queue state SHALL survive application and machine restarts.",
   "section": "8.3 Task queues"
  },
  "NFR-30": {
   "text": "Because go-build-it depends on herdr, it SHALL keep a record of the herdr and Ovis (a herdr web client) source revisions last reviewed, what changed, and what go-build-it did about",
   "section": "8.4 Upstream tracking"
  },
  "NFR-31": {
   "text": "A scheduled review SHOULD run at least weekly.",
   "section": "8.4 Upstream tracking"
  },
  "NFR-32": {
   "text": "go-build-it's herdr client types SHALL be generated from herdr's published API schema, so an incompatible change fails the build.",
   "section": "8.4 Upstream tracking"
  },
  "FR-90": {
   "text": "The application SHALL work with both GitHub (gh) and GitLab (glab), and SHALL detect which one each project uses and whether its CLI is authenticated.",
   "section": "9. Git, GitHub and GitLab workflow"
  },
  "FR-91": {
   "text": "The application SHALL enforce a defined development workflow.",
   "section": "9. Git, GitHub and GitLab workflow"
  },
  "FR-92": {
   "text": "All git and forge operations SHALL be carried out by the Git / Release Manager role, including:",
   "section": "9. Git, GitHub and GitLab workflow"
  },
  "FR-93": {
   "text": "Deterministic git operations (e.g.",
   "section": "9. Git, GitHub and GitLab workflow"
  },
  "FR-94": {
   "text": "The application SHALL make sure completed work is pushed to the remote.",
   "section": "9. Git, GitHub and GitLab workflow"
  },
  "FR-100": {
   "text": "Team members SHALL be able to send questions to human team members through a dedicated channel.",
   "section": "10.1 Questions"
  },
  "FR-101": {
   "text": "Each question SHALL be marked blocking (the asking work pauses until it is answered) or non-blocking (work continues on a stated assumption, which the human can later confirm or co",
   "section": "10.1 Questions"
  },
  "FR-102": {
   "text": "Each question SHALL carry everything needed to answer it without digging:",
   "section": "10.1 Questions"
  },
  "FR-102a": {
   "text": "Attachments SHALL open in an in-app viewer suited to their type:",
   "section": "10.1 Questions"
  },
  "FR-103": {
   "text": "Questions MAY offer structured answers (choices, yes/no, pick-an-option) as well as free text.",
   "section": "10.1 Questions"
  },
  "FR-104": {
   "text": "A Questions dashboard (on desktop and mobile) SHALL list open questions across all projects.",
   "section": "10.1 Questions"
  },
  "FR-105": {
   "text": "Answers SHALL be sent back to the asking session or task automatically.",
   "section": "10.1 Questions"
  },
  "FR-106": {
   "text": "Email and other out-of-app notifications are out of scope for now.",
   "section": "10.1 Questions"
  },
  "FR-107": {
   "text": "Wherever the user writes an answer, feedback or chat message, they SHALL be able to dictate it by voice (speech-to-text) as well as type it.",
   "section": "10.1 Questions"
  },
  "FR-110": {
   "text": "The application SHALL provide an easy way to give feedback on completed work.",
   "section": "10.2 Feedback on completed work"
  },
  "FR-111": {
   "text": "Feedback SHALL go into the context of the follow-up task automatically, and SHALL be recorded in the history.",
   "section": "10.2 Feedback on completed work"
  },
  "FR-112": {
   "text": "The review of completed work SHALL be an assessment, not primarily a code diff.",
   "section": "10.2 Feedback on completed work"
  },
  "FR-115": {
   "text": "Roadmaps SHALL support review gates:",
   "section": "10.3 Review gates and dress rehearsals"
  },
  "FR-116": {
   "text": "A review gate MAY be a dress rehearsal.",
   "section": "10.3 Review gates and dress rehearsals"
  },
  "FR-116a": {
   "text": "Mockups and screens shown in option sets, rehearsals and reviews SHALL open in a large viewer (full screen, with desktop and mobile viewport sizes), because a desktop-sized mockup ",
   "section": "10.3 Review gates and dress rehearsals"
  },
  "FR-117": {
   "text": "During a dress rehearsal, the human SHALL be able to give feedback on each step, and SHALL be able to talk with the agents live to request changes or improvements as the rehearsal ",
   "section": "10.3 Review gates and dress rehearsals"
  },
  "FR-117a": {
   "text": "More than one reviewer, human or AI persona, SHALL be able to record findings on the same rehearsal.",
   "section": "10.3 Review gates and dress rehearsals"
  },
  "FR-118": {
   "text": "Rehearsal outcomes (feedback per step, changes requested, approval) SHALL be recorded.",
   "section": "10.3 Review gates and dress rehearsals"
  },
  "FR-120": {
   "text": "For significant design or implementation decisions, the team SHALL present an option set of 2–3 alternative approaches, not a single answer.",
   "section": "10.4 Exploratory design and values"
  },
  "FR-121": {
   "text": "Each option SHALL include a summary and tradeoffs, and SHALL include a visual artifact where relevant (an HTML mockup or an image).",
   "section": "10.4 Exploratory design and values"
  },
  "FR-122": {
   "text": "The human SHALL be able to choose one option, or combine parts of several options and explain why.",
   "section": "10.4 Exploratory design and values"
  },
  "FR-123": {
   "text": "The application SHALL keep a living problem definition and a set of values (guiding principles) for each project.",
   "section": "10.4 Exploratory design and values"
  },
  "FR-124": {
   "text": "Future option sets SHALL be generated with the recorded values in mind, so options get closer to what the human wants as understanding grows.",
   "section": "10.4 Exploratory design and values"
  },
  "NFR-10": {
   "text": "The application SHALL keep a complete, searchable history of what happened, when, why, and who or which persona did it.",
   "section": "11. Observability and self-improvement"
  },
  "NFR-11": {
   "text": "It SHALL always be possible to determine what the system is currently doing and why.",
   "section": "11. Observability and self-improvement"
  },
  "NFR-12": {
   "text": "Records SHALL be structured so AI agents can analyze the application's own performance.",
   "section": "11. Observability and self-improvement"
  },
  "NFR-14": {
   "text": "Every history event SHALL open a detail view suited to its type:",
   "section": "11. Observability and self-improvement"
  },
  "NFR-13": {
   "text": "Agents SHALL be able to use that analysis to propose and make improvements to go-build-it itself, such as prompts, persona definitions, handoff templates or code.",
   "section": "11. Observability and self-improvement"
  },
  "FR-130": {
   "text": "Documentation SHALL be embedded in the application, searchable, and linked from the relevant places in the UI.",
   "section": "12. Built-in documentation"
  },
  "FR-131": {
   "text": "Documentation SHALL be fully illustrated with screenshots.",
   "section": "12. Built-in documentation"
  },
  "FR-132": {
   "text": "Documentation SHALL include a dress-rehearsal / UAT walkthrough that covers every feature, step by step, for every role.",
   "section": "12. Built-in documentation"
  },
  "FR-133": {
   "text": "The UAT walkthrough SHOULD be written as automated end-to-end scripts (e.g.",
   "section": "12. Built-in documentation"
  },
  "NFR-20": {
   "text": "Agents MAY run arbitrary code on the development machine.",
   "section": "13. Security posture"
  },
  "NFR-21": {
   "text": "Access control relies on network exposure and Tailscale (§4.2).",
   "section": "13. Security posture"
  }
 },
 "questions": {
  "Q1": "For access through an external network interface (FR-18), what should the challenge be: a local password or token, an OAuth/OIDC provider, or refusing external access completely and requiring Tailscale Serve?",
  "Q2": "Check that each target harness supports headless operation and reports context-window usage. Claude Code reports usage in its streaming JSON output. OpenCode offers a non-interactive mode (opencode run) and a server mode; how it reports context usage needs to be confirmed. Antigravity's headless and CLI support needs to be confirmed.",
  "Q3": "When smart handoffs graduate from experimental, what is the default context threshold, and should it be configurable per harness or model? What evidence (rework rate, handoff quality vs. compaction) decides that they are ready?",
  "Q4": "Can multiple humans collaborate on the same projects (e.g. different Tailscale users with different roles), or is it single-user with optional reviewers?",
  "Q5": "What is the exact default git workflow (branch naming, PR/MR requirements, merge strategy)?",
  "Q6": "What is the mechanism and channel for auto-updates (GitHub Releases, Homebrew tap, etc.)?",
  "Q7": "How should attachable (headed) sessions work? Answered (rev 3): every session runs in herdr; see ADR-009.",
  "Q8": "Conversation rotation policy (FR-76)? Answered 2026-11-08: keep one conversation per member across tasks until context reaches the handoff threshold (75%), then rotate, starting from a handoff plus the member's notes. Lens starts a fresh conversation for every review. See ADR-010.",
  "Q9": "Windows support level? Answered 2026-11-08: ship Windows at herdr's beta level, with herdr in the Windows CI job. A herdr capability missing on Windows is shown as unavailable there. See ADR-009.",
  "Q10": "Archetype change propagation (FR-12b)? Answered 2026-11-08: a proposal per affected member, plus one action to apply the change to every member that has not overridden the changed fields. See ADR-010."
 }
};
