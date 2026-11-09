"""Add the rev 3 phases (P10-P15: herdr backend, per-project team members) to the tracker.

Idempotent: re-running replaces P10+ phases and tasks but keeps any status already set on them.
Also re-parses docs/requirements.md so requirement and question text stays in sync, and fails
if any requirement is not covered by a task, a dependency is unknown, or a question is unknown.

Usage: python3 scripts/docs/add_rev3_phases.py
"""
import datetime, json, os, re, sys

ROOT = os.path.join(os.path.dirname(__file__), "..", "..")
REQ_MD = os.path.join(ROOT, "docs", "requirements.md")
DATA = os.path.join(ROOT, "docs", "tracker", "tracker-data.js")

src = open(DATA).read()
head = src[: src.index("window.TRACKER")]
D = json.loads(src[src.index("{"): src.rindex("}") + 1])

# ---- requirements and questions, parsed from the source ----
md = open(REQ_MD).read()
reqs, questions, section = {}, {}, ""
for line in md.splitlines():
    h = re.match(r"^#{2,3} (.+)", line)
    if h:
        section = h.group(1)
    m = re.match(r"^- \*\*((?:FR|NFR)-\d+[a-z]?)\*\*: (.*)", line)
    if m:
        text = re.sub(r"\*\*|\*|`", "", m.group(2))
        first = re.split(r"(?<=[.:])\s", text, maxsplit=1)[0]
        reqs[m.group(1)] = {"text": first[:180], "section": section}
    q = re.match(r"^\| (Q\d+) \| (.*) \|$", line)
    if q:
        questions[q.group(1)] = re.sub(r"\*\*|`|~~", "", q.group(2))

# ---- rev 3 phases ----
PHASES = [
    dict(id="P10", name="herdr direction: decisions, spike & design", mvp=False,
         goal="Record rev 3 (herdr as the only backend, per-project team members from archetypes), prove the herdr integration with a spike, decide the open questions, and redesign the affected mockups before building.",
         gate="herdr direction review",
         gateCriteria=["Requirements rev 3, ADR-009 and ADR-010 approved by the owner",
                       "Spike: a Claude member in the gbi herdr session takes a prompt, finishes, reports its conversation ID and usage, and streams live to a browser",
                       "Q8 and Q10 decided (Q9 can wait for P15)",
                       "Rev 3 mockups (desktop, mobile, herdr layout) reviewed"]),
    dict(id="P11", name="herdr runtime foundation", mvp=False,
         goal="go-build-it owns and supervises the gbi herdr session: generated client, version and capability checks, onboarding and integrations, login service, event reconciliation, fake harness, CI with herdr on every platform.",
         gate="Runtime demo",
         gateCriteria=["Login starts go-build-it and the gbi herdr session; restarting go-build-it leaves agents running",
                       "Doctor shows herdr version, session and integration status, and fixes outdated integrations",
                       "CI is green on macOS, Linux and Windows with a pinned herdr"]),
    dict(id="P12", name="Per-project teams from archetypes", mvp=False,
         goal="Replace global teams and personas with global persona archetypes and per-project team members that can be tuned, compared with their archetype, and promoted back.",
         gate="Team review",
         gateCriteria=["Every imported project has a team made from starter archetypes",
                       "A member's tuning, its difference from the archetype, and an archetype change applied per member are shown",
                       "Each member's instruction file is generated in its worktree"]),
    dict(id="P13", name="Team members on herdr", mvp=False,
         goal="Run every AI team member as a long-running harness in its herdr pane: profiles, pane lifecycle, dispatch, transcripts, status, limits, conversation rotation, reconnection and labels. Remove tmux and headless runs.",
         gate="Member demo",
         gateCriteria=["Claude Code, Antigravity and Pi members each finish a task end to end through herdr, reviewed by Lens",
                       "A blocked member shows as needing the human; herdr restart resumes members in their conversations",
                       "herdr --session gbi shows every project and member labelled with role, task and context %",
                       "No tmux or stream-json code remains"]),
    dict(id="P14", name="Live terminal & UI", mvp=False,
         goal="Watch and steer any team member from the desktop and mobile UI through herdr's terminal streams, and bring herdr state into the live views, Inbox, settings and history.",
         gate="Watch & steer UAT",
         gateCriteria=["Watch any member live on desktop and phone; take control and hand it back",
                       "Blocked members appear in the Questions dashboard and mobile Inbox",
                       "Every herdr state change, rotation and takeover is in the history"]),
    dict(id="P15", name="Rev 3 docs, UAT & upstream watch", mvp=False,
         goal="Update the embedded docs, UAT scripts and screenshots for the herdr flow, schedule the weekly herdr/Ovis upstream review, and settle Windows support.",
         gate="Rev 3 UAT",
         gateCriteria=["UAT scripts for every role pass on the herdr flow and regenerate screenshots",
                       "The first scheduled upstream review has run and recorded its findings",
                       "Windows support level decided (Q9) and documented"]),
]

T = []
def t(id, title, reqs_, views, size, deps, q=None, done=""):
    T.append(dict(id=id, phase=id.split("-")[0], title=title, reqs=reqs_, views=views, size=size,
                  deps=deps, status="todo", done=done, q=q or []))

# P10
t("P10-01", "Requirements rev 3: herdr backend, per-project team members, upstream tracking", ["FR-01", "FR-07"], [], "M", [],
  done="docs/requirements.md rev 3; Q7 answered, Q8–Q10 added.")
t("P10-02", "ADR-009: herdr is the only agent backend; ADR-008 superseded", ["FR-60", "FR-07"], [], "S", ["P10-01"],
  done="docs/decisions/ADR-009-herdr-backend.md")
t("P10-03", "ADR-010: per-project team members from global persona archetypes", ["FR-11", "FR-12"], [], "S", ["P10-01"],
  done="docs/decisions/ADR-010-project-team-members.md")
t("P10-04", "Spike: gbi herdr session, agent start + prompt --wait, agent_session → transcript usage, terminal observe → xterm.js", ["FR-60", "FR-61", "FR-66", "FR-71"], [], "M", ["P10-02"],
  done="Spike notes appended to ADR-009; throwaway code not merged.")
t("P10-05", "Decide conversation rotation policy from spike cost data", ["FR-76"], [], "S", ["P10-04"], q=["Q8"])
t("P10-06", "Decide how archetype changes reach project members", ["FR-12b"], [], "S", ["P10-03"], q=["Q10"])
t("P10-07", "Mockups rev 3, desktop: live members, live terminal, archetypes, project team, herdr settings and setup, card and overview team panel, queue assignment", ["FR-34", "FR-12a", "FR-66", "FR-67"],
  ["A-v01", "A-v02", "A-v05", "A-v05b", "A-v06", "A-v10", "A-v10b", "A-v13", "A-v15"], "L", ["P10-02", "P10-03"])
t("P10-08", "Mockups rev 3, mobile: members, terminal control and takeover, project team, blocked members in Inbox", ["FR-35", "FR-66"],
  ["A-M2", "A-M3", "A-M8", "A-M10"], "M", ["P10-07"])
t("P10-09", "herdr layout spec: what herdr --session gbi shows (workspaces, grouped worktrees, pane labels, tokens, summaries, notifications)", ["FR-67"], [], "S", ["P10-04"])
t("P10-10", "Upstream baseline: docs/upstream/ with herdr and Ovis revisions reviewed and findings", ["NFR-30"], [], "S", ["P10-02"])
t("P10-11", "Docs tooling in the repo: tracker, plan and HTML generators under scripts/docs", [], [], "S", [],
  done="scripts/docs/{add_rev3_phases,genmd,md2html,setstatus}.py")

# P11
t("P11-01", "herdr client: socket/named-pipe NDJSON transport, types generated from herdr api schema, ping capability check, supported version range", ["NFR-32", "FR-07"], [], "M", ["P10-04"])
t("P11-02", "Supervise herdr server --session gbi: start, health, restart, stop", ["FR-07"], [], "M", ["P11-01"])
t("P11-03", "Doctor and onboarding: herdr install and version, integration status, install or update integrations", ["FR-07"], ["A-v15", "A-v13"], "M", ["P11-01", "P10-07"])
t("P11-04", "Login service brings up go-build-it and the gbi session; go-build-it restarts and updates leave agents running", ["FR-08", "FR-02", "FR-06"], [], "S", ["P11-02"])
t("P11-05", "Event subscription and snapshot reconciler with events_lost recovery", ["NFR-11", "FR-68"], [], "M", ["P11-01"])
t("P11-06", "Fake harness in herdr: reports state with pane.report_agent and writes a transcript", ["FR-61"], [], "M", ["P11-01"])
t("P11-07", "CI installs a pinned herdr on macOS, Linux and Windows; weekly job against latest herdr", ["NFR-32", "NFR-01"], [], "M", ["P11-06"])

# P12
t("P12-01", "Schema: archetypes and project_members replace global teams and personas", ["FR-11", "FR-12", "FR-12d", "FR-12e"], [], "M", ["P10-03"])
t("P12-02", "Starter archetype per built-in role", ["FR-12", "FR-25"], [], "S", ["P12-01"])
t("P12-03", "API, CLI and MCP: archetypes, project members (add, tune, remove), human roles per project", ["FR-30", "FR-12a", "FR-12e"], [], "M", ["P12-01"])
t("P12-04", "Archetype change review: affected members, per-member apply keeping tuning", ["FR-12b"], ["A-v10"], "M", ["P12-03", "P10-06"], q=["Q10"])
t("P12-05", "Promote member tuning to its archetype through the artifact editor", ["FR-12c", "FR-13"], ["A-v03b"], "S", ["P12-03"])
t("P12-06", "Archetypes library and project Team tab with member detail and tuning", ["FR-13", "FR-12a", "FR-10"], ["A-v10", "A-v10b"], "L", ["P12-03", "P10-07"])
t("P12-07", "Per-member instruction file generated in the member's worktree", ["FR-53"], [], "S", ["P12-01"])
t("P12-08", "Queue assignment to a role or a specific member; dispatcher picks a free member", ["FR-81"], ["A-v06"], "M", ["P12-01"])
t("P12-09", "Catalog card and overview show the project team and each member's herdr state", ["FR-41"], ["A-v01", "A-v02", "A-M2"], "S", ["P12-06", "P11-05"])

# P13
t("P13-01", "Harness profiles: Claude Code, Antigravity, Pi, OpenCode (kind, launch and resume args, transcript reader, limit signals)", ["FR-61", "FR-62", "FR-63"], [], "L", ["P11-01"])
t("P13-02", "Member pane lifecycle: workspace per project, member worktree opened grouped, agent start, name, release", ["FR-12d", "FR-64", "FR-67"], [], "M", ["P12-01", "P11-02"])
t("P13-03", "Dispatch: write .gbi-task.md, agent prompt --wait, hand completion to Courier and Lens", ["FR-60", "FR-70"], [], "M", ["P13-02", "P12-08"])
t("P13-04", "Transcript readers: structured events, messages and usage (measured or estimated) per harness", ["FR-66", "FR-71", "NFR-10"], [], "L", ["P13-01"])
t("P13-05", "Status mapping: herdr state drives session and task state; blocked becomes a needs-you item", ["FR-68"], [], "M", ["P11-05", "P13-02"])
t("P13-06", "Rate and usage limit detection with pause and auto-resume", ["FR-65"], [], "S", ["P13-04"])
t("P13-07", "Conversation continuity and rotation per policy; smart handoff starts the new conversation", ["FR-69", "FR-76", "FR-72", "FR-73", "FR-74"], [], "M", ["P13-04", "P10-05"], q=["Q8"])
t("P13-08", "Reconnect after herdr or machine restart by conversation ID; queue resumes", ["FR-69", "FR-82"], [], "M", ["P13-05"])
t("P13-09", "herdr labels: pane metadata (name, role, task, context %), workspace summaries, notifications for gates and questions", ["FR-67"], [], "S", ["P13-02", "P10-09"])
t("P13-10", "Permission prompts: permission modes, per-project allowlists, optional Claude permission hook as structured approval", ["FR-68"], [], "M", ["P13-05"])
t("P13-11", "Lens reviews run in a fresh conversation per review", ["FR-54", "FR-112"], [], "S", ["P13-03"])
t("P13-12", "Interrupt, cancel and restart a member through herdr", ["FR-61"], [], "S", ["P13-02"])
t("P13-13", "Remove tmux mode, attachable_sessions flag and the headless stream-json path", ["FR-60"], [], "S", ["P13-03", "P13-04"])

# P14
t("P14-01", "Terminal bridge: herdr terminal session observe/control → WebSocket with ack-based flow control and takeover", ["FR-66"], [], "L", ["P11-01", "P10-04"])
t("P14-02", "Desktop terminal view with take control and hand back", ["FR-66"], ["A-v05b"], "M", ["P14-01", "P10-07"])
t("P14-03", "Mobile terminal with key bar", ["FR-35", "FR-66"], ["A-M10"], "M", ["P14-01", "P10-08"])
t("P14-04", "Live members view: herdr state, task, context %, structured view", ["NFR-11", "FR-66"], ["A-v05", "A-M8"], "M", ["P13-04", "P13-05"])
t("P14-05", "Blocked members in the Questions dashboard and mobile Inbox", ["FR-68", "FR-104"], ["A-v04", "A-M3"], "S", ["P13-05"])
t("P14-06", "Open in herdr: copy herdr --session gbi and focus the member's pane", ["FR-67"], ["A-v05"], "S", ["P13-02"])
t("P14-07", "Settings: herdr section (version, session, socket, integrations)", ["FR-63", "FR-07"], ["A-v13"], "S", ["P11-03"])
t("P14-08", "History events for herdr state changes, rotations and takeovers", ["NFR-10", "NFR-14"], ["A-v12", "A-v12b"], "S", ["P13-05", "P14-01"])

# P15
t("P15-01", "User docs rewrite: install, team, sessions, settings, mobile and CLI for herdr and archetypes", ["FR-130"], ["A-v14"], "M", ["P14-04", "P12-06"])
t("P15-02", "UAT scripts for the herdr flow per role; regenerate screenshots", ["FR-132", "FR-133", "FR-131"], ["A-v14"], "L", ["P15-01", "P11-06"])
t("P15-03", "Weekly upstream review job for herdr and Ovis that opens an issue or PR", ["NFR-31"], [], "S", ["P10-10"])
t("P15-04", "Windows at herdr's support level", ["NFR-01"], [], "M", ["P11-07"], q=["Q9"])
t("P15-05", "Rev 3 close-out: retire obsolete tests and text, mockup change log, coverage check", ["FR-01"], [], "S", ["P15-02"])

# ---- merge, keeping any status already recorded ----
old = {x["id"]: x for x in D["tasks"]}
for x in T:
    if x["id"] in old:
        x["status"] = old[x["id"]]["status"]
        x["done"] = old[x["id"]].get("done") or x["done"]
for x in T:
    if x["id"] in ("P10-01", "P10-02", "P10-03", "P10-11") and x["id"] not in old:
        x["status"] = "done"
oldph = {p["id"]: p for p in D["phases"]}
for p in PHASES:
    for k in ("gateApproved", "gateApprovedBy"):
        if k in oldph.get(p["id"], {}):
            p[k] = oldph[p["id"]][k]
keep = lambda pid: int(pid[1:]) < 10
D["phases"] = [p for p in D["phases"] if keep(p["id"])] + PHASES
D["tasks"] = [x for x in D["tasks"] if keep(x["phase"])] + T
D["requirements"], D["questions"] = reqs, questions
D["updated"] = datetime.date.today().isoformat()

# ---- checks ----
ids = {x["id"] for x in D["tasks"]}
errs = []
for x in D["tasks"]:
    errs += [f"{x['id']} bad dep {d}" for d in x["deps"] if d not in ids]
    errs += [f"{x['id']} unknown req {r}" for r in x["reqs"] if r not in reqs]
    errs += [f"{x['id']} unknown question {q}" for q in x["q"] if q not in questions]
missing = sorted(set(reqs) - {r for x in D["tasks"] for r in x["reqs"]})
if missing:
    errs.append("uncovered: " + ", ".join(missing))
if errs:
    print("\n".join(errs)); sys.exit(1)

open(DATA, "w").write(head + "window.TRACKER = " + json.dumps(D, indent=1, ensure_ascii=False) + ";\n")
done = sum(1 for x in D["tasks"] if x["status"] == "done")
print(f"ok: {len(D['tasks'])} tasks ({len(T)} rev 3), {len(reqs)} requirements covered, {len(questions)} questions, {done} done")
