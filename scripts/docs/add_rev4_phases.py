"""Add the rev 4 phase (P16: in-app change log and guided updates) to the tracker.

Idempotent: re-running replaces P16 tasks but keeps any status already set on them. Re-parses
docs/requirements.md and fails if a requirement is uncovered or a dependency is unknown.

Usage: python3 scripts/docs/add_rev4_phases.py
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
    dict(id="P16", name="Change log & guided updates", mvp=False,
         goal="Keep go-build-it's change log in the repo and in the app, open it from the version number, check for updates there, and update with one button that shows each step completing, across the restart.",
         gate="Update UAT",
         gateCriteria=["Clicking the version (desktop and mobile) opens the change log with the installed version marked",
                       "Check for updates lists every newer release's change log entry",
                       "Update now runs all eight steps through a real restart against a signed test release, and the restarted server finishes the job",
                       "A failed verification leaves the running version untouched and offers Try again; history records every update"]),
]
T = []
def t(id, title, reqs_, views, size, deps, q=None, done=""):
    T.append(dict(id=id, phase=id.split("-")[0], title=title, reqs=reqs_, views=views, size=size,
                  deps=deps, status="todo", done=done, q=q or []))

t("P16-01", "Requirements rev 4 (FR-06a–e), ADR-011, mockups 16 and M12, plan and tracker", ["FR-06a", "FR-06b", "FR-06c", "FR-06d", "FR-06e"], ["A-v16", "A-M12", "A-v13", "A-v01"], "S", [],
  done="docs/requirements.md rev 4; docs/decisions/ADR-011-changelog-guided-updates.md; desktop view 16, mobile M12")
t("P16-02", "CHANGELOG.md embedded in the binary, parser and validity test; release tooling (gbi dev changelog release|notes) and make release check", ["FR-06a", "FR-05"], [], "S", ["P16-01"])
t("P16-03", "Update check lists every newer stable release with its notes and when it ran", ["FR-06c"], ["A-v16"], "S", ["P16-01"])
t("P16-04", "Update job: eight named steps, download progress, binary self-test, saved to disk; Try again and Roll back", ["FR-06d", "FR-06"], ["A-v16"], "M", ["P16-03"])
t("P16-05", "Restart in place (exec on macOS/Linux, spawn on Windows), port retry, restarted server finishes restart and reconnect steps", ["FR-06d", "FR-08", "NFR-01"], [], "M", ["P16-04"])
t("P16-06", "Automatic updater uses the same job; history events for start, completion, failure and rollback", ["FR-06e", "NFR-10"], [], "S", ["P16-04"])
t("P16-07", "API and CLI: changelog, updates check/install/job/rollback; regenerate the TS client", ["FR-30", "FR-31", "FR-06d"], [], "S", ["P16-04"])
t("P16-08", "Desktop change log page from the sidebar version (g v), check, Update now with live steps through the restart", ["FR-06b", "FR-06c", "FR-06d"], ["A-v16", "A-v01", "A-v13"], "M", ["P16-07"])
t("P16-09", "Mobile change log screen from the version on More", ["FR-06b", "FR-35"], ["A-M12", "A-M9"], "S", ["P16-08"])
t("P16-10", "End-to-end update test against a fake signed release with a real restart (scripts/update_e2e.sh, web/e2e/update.spec.ts)", ["FR-06d", "FR-132"], ["A-v16"], "M", ["P16-05", "P16-08"])
t("P16-11", "Docs: settings and updates page, keyboard (g v), CLI, README release steps; keyboard test", ["FR-130", "FR-06a"], ["A-v14"], "S", ["P16-08"])

old = {x["id"]: x for x in D["tasks"]}
for x in T:
    if x["id"] in old:
        x["status"] = old[x["id"]]["status"]
        x["done"] = old[x["id"]].get("done") or x["done"]
    elif x["id"] == "P16-01":
        x["status"] = "done"
oldph = {p["id"]: p for p in D["phases"]}
for p in PHASES:
    for k in ("gateApproved", "gateApprovedBy", "gateEvidence"):
        if k in oldph.get(p["id"], {}):
            p[k] = oldph[p["id"]][k]
keep = lambda pid: int(pid[1:]) < 16
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
print(f"ok: {len(D['tasks'])} tasks ({len(T)} rev 4), {len(reqs)} requirements covered, {len(questions)} questions, {done} done")
