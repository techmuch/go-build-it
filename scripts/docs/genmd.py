"""Regenerate docs/tracker/plan.md from docs/tracker/tracker-data.js. Usage: python3 scripts/docs/genmd.py"""
import json,os,re
ROOT=os.path.join(os.path.dirname(__file__),"..","..")
src=open(os.path.join(ROOT,"docs","tracker","tracker-data.js")).read()
D=json.loads(src[src.index("{"):src.rindex("}")+1])
W=D["sizes"]
out=[]
w=out.append
w("# go-build-it — Phased delivery plan\n")
w(f"Status: Draft · {D['updated']} · Based on [requirements rev 9](../requirements.md) and [Approach A mockups](../mockups/a-mission-control/desktop.html) · Live status: [tracker](index.html)\n")
w("""## 1. How this plan is built

- **Approach A is the UI baseline.** Every UI task names the Approach A view it implements (desktop `A-v01`…`A-v14` plus detail views `02b`, `03b`, `04b`, `05b`, `12b`; mobile `A-M1`…`A-M11`).
- **Every requirement maps to at least one task.** The tracker's *Requirement coverage* tab checks this.
- **Each phase ends at a review gate (◆),** the same mechanism the app itself provides (FR-115). A gate passes only when the owner approves it.
- **The MVP is the desktop app at the end of P5.** Mobile (P6), observability UI (P7), docs/UAT/auto-update (P8) and experimental features (P9) follow.
- **Simple path first.** Harness compaction is the MVP default; smart handoffs and attachable headed sessions are experimental (P9), as decided in rev 2.
- **Rev 3 (P10–P15): herdr and per-project teams.** P0–P9 delivered rev 2. Rev 3 makes [herdr](https://herdr.dev) the only agent backend ([ADR-009](../decisions/ADR-009-herdr-backend.md)) and replaces global teams with per-project team members made from global persona archetypes ([ADR-010](../decisions/ADR-010-project-team-members.md)). It starts with a spike and redesigned mockups (P10), then builds the runtime (P11), teams (P12), members on herdr (P13), the live terminal and UI (P14), and docs, UAT and the upstream watch (P15). Rev 3 phases are not MVP-flagged: the MVP shipped with P5.
- **Rev 4 (P16): change log and guided updates** ([ADR-011](../decisions/ADR-011-changelog-guided-updates.md)).
- **Rev 5 (P17–P18): doctor, AI-agent install and Learn** ([ADR-012](../decisions/ADR-012-doctor-and-learning.md)). P17 makes the doctor prove that gbi can start herdr and a harness, and explain failures to a person or an AI agent; it adds the install that works from the GitHub URL alone. P18 adds lessons with a coach panel and a practice project, driving gbi from your AI harness, and *How go-build-it thinks*.
- **Rev 6 (P19): folder trust** ([ADR-013](../decisions/ADR-013-folder-trust.md)). go-build-it answers a harness's \"trust this folder\" prompt for the folders it created, so members, helpers and the doctor start without a person. Other prompts stay with the person.
- **Rev 7 (P20): project documents** ([ADR-014](../decisions/ADR-014-project-documents.md)). Artifacts gets six sub-tabs: HTML entry points for requirements, mockups, plan and tracker, chosen explicitly; labels for every file, set through the CLI; a sandboxed viewer that follows links; Compare for mockups and the real app; traceability.
- **Rev 8 (P21): tabs from docs/gbi.toml** ([ADR-014](../decisions/ADR-014-project-documents.md), amended). Each project decides its Artifacts sub-tabs: its own roles such as vision, their order, their names, and leaving out roles it doesn't use.
- **Rev 9 (P22–P23): starter personas and standards; remote access** ([ADR-015](../decisions/ADR-015-remote-access.md)). P22 gives the six starter archetypes full role instructions, upgrades them in place when never edited, and preloads the starter standards on new installations. P23 proves go-build-it is served on the tailnet, is this go-build-it and is private, never takes another service's address, and shows a QR code for a phone.
- **Dogfooding from P4.** Once Courier can branch, PR, merge and release (end of P4), go-build-it manages its own remaining phases, and this tracker's data is imported into it.
- **Sizes, not dates.** S ≈ up to a day of agent work, M ≈ 2–4 days, L ≈ 1–2 weeks. Progress is weighted S=1, M=3, L=8. Dates are set once the P0 spike shows real throughput.

## 2. Phases at a glance

| Phase | Name | MVP | Tasks | Points | Gate |
|---|---|---|---|---|---|""")
for p in D["phases"]:
    ts=[t for t in D["tasks"] if t["phase"]==p["id"]]
    w(f"| {p['id']} | {p['name']} | {'✓' if p['mvp'] else ''} | {len(ts)} | {sum(W[t['size']] for t in ts)} | ◆ {p['gate']} |")
w("""
## 3. Critical path

- **Main chain:** P0 decisions and scaffold → P1 platform → P3 adapter interface → P3 queue and dispatcher → P4 Courier → P5 human loop → **MVP**.
- **In parallel:** P0-04 harness spike → P3 adapters; P1 → P2 projects and artifacts (feeds P3 context preparation and P5 editors).
- **After the MVP:** P5 → P6 mobile; P1 history store → P7 observability; P0-12 e2e harness + P5 rehearsals → P8 UAT and docs; P3 sessions view → P9 attachable sessions.

**Rev 3 chain:** P10 requirements and ADRs → P10-04 herdr spike → P11-01 herdr client → P11-02 session supervisor → P13-02 member pane lifecycle → P13-03 dispatch → P13-13 removal of tmux and headless runs → P14 live terminal → P15 UAT. P12 (teams) runs in parallel with P11 and must land before P13-02. Mockups (P10-07, P10-08) gate the UI tasks in P12 and P14.

The longest rev 2 chain ran through the **harness spike (P0-04, Q2)**, the adapter interface, the queue and dispatcher, Courier, and the work review. Start P0-04 first: its findings decide the scope of P3-06 and P3-07.

## 4. Phases in detail
""")
for p in D["phases"]:
    ts=[t for t in D["tasks"] if t["phase"]==p["id"]]
    w(f"### {p['id']} · {p['name']}{' (MVP)' if p['mvp'] else ''}\n")
    w(f"**Goal:** {p['goal']}\n")
    w(f"**◆ Gate: {p['gate']}.** Passes when:")
    for c in p["gateCriteria"]: w(f"- {c}")
    w("\n| Task | Size | Requirements | Approach A views | Depends on |\n|---|---|---|---|---|")
    for t in ts:
        st=" ✅" if t["status"]=="done" else (" 🔶" if t["status"]=="in-progress" else "")
        q=(" · waits on "+", ".join(t["q"])) if t["q"] else ""
        w(f"| **{t['id']}** {t['title']}{st}{q} | {t['size']} | {', '.join(t['reqs'])} | {', '.join(t['views']) or '—'} | {', '.join(t['deps']) or '—'} |")
    w("")
w("""## 5. Open questions schedule

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
""")
open(os.path.join(ROOT,"docs","tracker","plan.md"),"w").write("\n".join(out))
print("ok", len(out))
