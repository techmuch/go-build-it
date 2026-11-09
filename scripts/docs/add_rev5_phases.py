"""Add the rev 5 phases (P17: doctor and AI-agent install; P18: Learn) to the tracker.

Idempotent: re-running replaces P17–P18 tasks but keeps any status already set on them. Re-parses
docs/requirements.md and fails if a requirement is uncovered or a dependency is unknown.

Usage: python3 scripts/docs/add_rev5_phases.py
"""
import datetime, json, os, re, sys

ROOT = os.path.join(os.path.dirname(__file__), "..", "..")
REQ_MD = os.path.join(ROOT, "docs", "requirements.md")
DATA = os.path.join(ROOT, "docs", "tracker", "tracker-data.js")

src = open(DATA).read()
head = src[: src.index("window.TRACKER")]
D = json.loads(src[src.index("{"): src.rindex("}") + 1])

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

PHASES = [
    dict(id="P17", name="Doctor & AI-agent install", mvp=False,
         goal="Prove that gbi can start herdr and each harness inside it, explain every failure to a person or an AI agent with evidence and a fix, and let an AI agent install gbi from the GitHub URL alone.",
         gate="Doctor UAT",
         gateCriteria=["gbi doctor runs with no server; static and live tiers open and close a real pane in the gbi session",
                       "The harness tier starts the scripted harness in herdr and reports a blocked or missing harness with evidence and a human fix",
                       "--json exit codes 0/1/2, --fix and --report behave as specified; every result id has a troubleshooting section",
                       "install.sh installs from a fake signed release, refuses a tampered one, and the installed gbi passes the doctor"]),
    dict(id="P18", name="Learn: lessons, practice project, harness control", mvp=False,
         goal="Teach every feature through short lessons with pro tips, run in a practice project that costs nothing, show how to drive gbi from your AI harness, and explain why gbi is designed the way it is.",
         gate="Learn UAT",
         gateCriteria=["Every lesson can be completed in the practice project and each step checks itself off",
                       "The coach panel's Show me goes to the right view and highlights the control; progress survives a reload",
                       "Every $ gbi example in the lessons and cookbook runs against a demo instance",
                       "How go-build-it thinks covers herdr, archetypes, worktrees, gates, questions, history and the route table, each linking its ADRs"]),
]
T = []
def t(id, title, reqs_, views, size, deps, q=None, done=""):
    T.append(dict(id=id, phase=id.split("-")[0], title=title, reqs=reqs_, views=views, size=size,
                  deps=deps, status="todo", done=done, q=q or []))

t("P17-01", "Requirements rev 5 (FR-02a–b, FR-07a–e, FR-134–139), ADR-012, mockups 17–20 and M13, plan and tracker", ["FR-02a", "FR-02b", "FR-07a", "FR-07b", "FR-07c", "FR-07d", "FR-07e", "FR-134", "FR-135", "FR-136", "FR-137", "FR-138", "FR-139"], ["A-v17", "A-v18", "A-v19", "A-v20", "A-M13"], "S", [], q=["Q11", "Q12"],
  done="docs/requirements.md rev 5; docs/decisions/ADR-012-doctor-and-learning.md; desktop views 17–20, mobile M13")
t("P17-02", "internal/doctor: result contract (stable ids, evidence, auto/human fixes, docs link) and the static tier, incl. harnesses on the login service PATH", ["FR-07a", "FR-07c", "FR-07"], ["A-v17"], "M", ["P17-01"])
t("P17-03", "Live tier: start the gbi session, open a throwaway doctor tab, echo round trip, harness binaries inside the pane, close", ["FR-07b", "FR-08"], ["A-v17"], "M", ["P17-02"])
t("P17-04", "Harness tier (start, wait for ready, blocked/exited/timeout with screen evidence) and opt-in prompt tier; tested with the scripted harness in a real herdr", ["FR-07b", "FR-60"], ["A-v17"], "M", ["P17-03"], q=["Q11"])
t("P17-05", "gbi doctor in-process: --json with exit codes 0/1/2, --fix, --report with secrets removed, --tier, --harness, --prompt", ["FR-07d", "FR-32"], [], "S", ["P17-04"])
t("P17-06", "API (GET /api/doctor, POST /api/doctor/fix) and Settings → herdr setup UI: tiers, evidence, human fixes, Test harnesses, test prompt, Copy report", ["FR-07a", "FR-07c", "FR-30"], ["A-v17", "A-v15"], "M", ["P17-05"])
t("P17-07", "Troubleshooting page with one section per doctor id, and a test that every id has one", ["FR-07e", "FR-130"], ["A-v14"], "S", ["P17-02"])
t("P17-08", "scripts/install.sh and install.ps1: version and directory, checksum and Ed25519 signature checks; AGENTS.md; test that the README, scripts and binary carry the same public key", ["FR-02b", "FR-06"], [], "M", ["P17-01"])
t("P17-09", "README: Install with an AI agent (numbered steps, rules, build from source while no release exists)", ["FR-02a"], [], "S", ["P17-08", "P17-05"])
t("P17-10", "Install e2e: install.sh against a fake signed release (and a tampered one), then the installed gbi passes the doctor against an isolated herdr; in CI", ["FR-02a", "FR-02b", "FR-07b"], [], "M", ["P17-08", "P17-05"])

t("P18-01", "Lesson format in docs/learn (front matter, steps with check/show lines, Pro tips), parser and tests; API and CLI gbi learn list|get|start|done", ["FR-134", "FR-135", "FR-30"], [], "M", ["P17-01"])
t("P18-02", "Progress per user (learn_progress): steps checked from history events in the practice project, visits and ticks", ["FR-135", "NFR-10"], [], "M", ["P18-01"])
t("P18-03", "Practice project: gbi learn practice [--reset|--remove], seeded repo under ~/.go-build-it/practice, starter team on the scripted harness, practice label", ["FR-136"], ["A-v18"], "M", ["P18-01"])
t("P18-04", "Learn page (g l): lessons with progress, practice project card, How go-build-it thinks", ["FR-134", "FR-138"], ["A-v18"], "M", ["P18-02", "P18-03"])
t("P18-05", "Coach panel docked in the shell: current step, live check-off, Show me navigation and highlight, Pro tips, Esc to hide", ["FR-134", "FR-135"], ["A-v19"], "M", ["P18-04"])
t("P18-06", "Mobile Learn screen (M13) and Learn on More", ["FR-134", "FR-35"], ["A-M13", "A-M9"], "S", ["P18-04"])
t("P18-07", "Lessons 0–10 covering every feature, with Pro tips", ["FR-134", "FR-132"], ["A-v18", "A-v19"], "L", ["P18-05"])
t("P18-08", "Drive gbi from your AI harness: lesson, cookbook page (Claude Code first; Antigravity, Pi, OpenCode notes) and gbi agent guide", ["FR-137", "FR-33"], ["A-v20"], "M", ["P18-01"])
t("P18-09", "How go-build-it thinks: explanation pages linking the ADRs", ["FR-138"], ["A-v18"], "S", ["P17-01"])
t("P18-10", "scripts/docs_examples.sh runs every $ gbi example in the lessons and cookbook against a demo instance", ["FR-139"], [], "S", ["P18-07", "P18-08"])
t("P18-11", "web/e2e/learn.spec.ts completes every lesson in the practice project and regenerates the lesson screenshots", ["FR-139", "FR-133", "FR-131"], ["A-v18", "A-v19", "A-M13"], "M", ["P18-07"])
t("P18-12", "Docs and release notes: keyboard (g l), CLI page, README tests, CHANGELOG Unreleased; keyboard test", ["FR-130", "FR-06a"], ["A-v14"], "S", ["P18-11", "P17-10"])

old = {x["id"]: x for x in D["tasks"]}
for x in T:
    if x["id"] in old:
        x["status"] = old[x["id"]]["status"]
        x["done"] = old[x["id"]].get("done") or x["done"]
    elif x["id"] == "P17-01":
        x["status"] = "done"
oldph = {p["id"]: p for p in D["phases"]}
for p in PHASES:
    for k in ("gateApproved", "gateApprovedBy", "gateEvidence"):
        if k in oldph.get(p["id"], {}):
            p[k] = oldph[p["id"]][k]
keep = lambda pid: int(pid[1:]) < 17
D["phases"] = [p for p in D["phases"] if keep(p["id"])] + PHASES
D["tasks"] = [x for x in D["tasks"] if keep(x["phase"])] + T
D["requirements"], D["questions"] = reqs, questions
D["updated"] = datetime.date.today().isoformat()

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
print(f"ok: {len(D['tasks'])} tasks ({len(T)} rev 5), {len(reqs)} requirements covered, {len(questions)} questions, {done} done")
