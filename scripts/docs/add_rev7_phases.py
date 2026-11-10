"""Add the rev 7 phase (P20: project documents) to the tracker.

Idempotent: re-running replaces P20 tasks but keeps any status already set on them. Re-parses
docs/requirements.md and fails if a requirement is uncovered or a dependency is unknown.

Usage: python3 scripts/docs/add_rev7_phases.py
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
    dict(id="P20", name="Project documents", mvp=False,
         goal="Give every project the same documents: HTML entry points for requirements, mockups, plan and tracker, explicit labels for every file, a viewer that follows links, a comparison of mockups with the real app, and traceability.",
         gate="Documents UAT",
         gateCriteria=["The six Artifacts sub-tabs show the chosen HTML entry points in a sandbox; links between documents switch tabs and keep the anchor",
                       "Files are labelled only by gbi artifacts add/label or the UI; unlabelled files are listed; changes are committed in docs/gbi.toml",
                       "Compare shows each view with its screenshot side by side and overlaid; accept and file-task are stored with the checksum and come back on change",
                       "gbi artifacts init, check, snapshot and gbi trace list work on a demo project and on go-build-it's own docs"]),
]
T = []
def t(id, title, reqs_, views, size, deps, q=None, done="", status="todo"):
    T.append(dict(id=id, phase=id.split("-")[0], title=title, reqs=reqs_, views=views, size=size,
                  deps=deps, status=status, done=done, q=q or []))

t("P20-01", "Requirements rev 7 (FR-140–FR-150), ADR-014, mockups 21–26 and M14, plan and tracker", ["FR-140", "FR-141", "FR-142", "FR-143", "FR-144", "FR-145", "FR-146", "FR-147", "FR-148", "FR-149", "FR-150"], ["A-v21", "A-v22", "A-v23", "A-v24", "A-v25", "A-v26", "A-M14"], "S", [],
  done="docs/requirements.md rev 7; docs/decisions/ADR-014-project-documents.md; desktop views 21–26, mobile M14", status="done")
t("P20-02", "internal/artifacts: docs/gbi.toml entries and labels (folder labels, fixed label list, validation), scan of docs/, Courier commits; routes and CLI artifacts list/add/label/entry", ["FR-141", "FR-142", "FR-143", "FR-30"], ["A-v25"], "M", ["P20-01"])
t("P20-03", "Raw serving for the viewer: sandbox CSP, CORS for the page's own files, gbi-nav.js injected into HTML, Markdown opened rendered; gbi-md.js with requirement anchors", ["FR-144", "FR-145"], ["A-v21", "A-v22"], "M", ["P20-01"])
t("P20-04", "Templates and gbi artifacts init (never overwrites); gbi artifacts check (entries, unlabelled, stale labels, broken links, unknown IDs)", ["FR-149", "FR-150"], ["A-v25", "A-v26"], "M", ["P20-02"])
t("P20-05", "Traceability from IDs: requirements, data-view/data-req/data-route, task reqs and text; gaps; gbi trace list", ["FR-146"], ["A-v24"], "M", ["P20-02"])
t("P20-06", "Compare: views × screenshots, states, docs/compare.json with SHA-256, accept and file task; gbi compare list/accept/task", ["FR-147"], ["A-v23"], "M", ["P20-05"])
t("P20-07", "Tracker snapshot docs/tracker/tracker.json from milestones and tasks; gbi artifacts snapshot", ["FR-148"], ["A-v26"], "S", ["P20-02"])
t("P20-08", "Artifacts UI: six sub-tabs, entry viewer that follows links and keeps the address, Support files with labels and entry points, Compare side by side and overlay, Traceability, Tracker", ["FR-140", "FR-141", "FR-142", "FR-144", "FR-146", "FR-147", "FR-148"], ["A-v21", "A-v22", "A-v23", "A-v24", "A-v25", "A-v26"], "L", ["P20-03", "P20-04", "P20-06", "P20-07"])
t("P20-09", "Mobile: project Documents with the sub-tabs and Compare (M14)", ["FR-140", "FR-147", "FR-35"], ["A-M14"], "S", ["P20-08"])
t("P20-10", "Project documents standard, agent guide, user docs and CLI examples; go-build-it's own docs/gbi.toml", ["FR-150", "FR-130", "FR-139"], ["A-v14"], "S", ["P20-04"])
t("P20-11", "Demo and UAT e2e for the documents tabs (screenshots regenerated); CHANGELOG", ["FR-133", "FR-06a"], ["A-v21", "A-v23", "A-v25"], "M", ["P20-08", "P20-10"])
t("P20-12", "Release 2026.11.09.2: signed binaries, install check", ["FR-05", "FR-06"], [], "S", ["P20-11"])

old = {x["id"]: x for x in D["tasks"]}
for x in T:
    if x["id"] in old:
        x["status"] = old[x["id"]]["status"]
        x["done"] = old[x["id"]].get("done") or x["done"]
oldph = {p["id"]: p for p in D["phases"]}
for p in PHASES:
    for k in ("gateApproved", "gateApprovedBy", "gateEvidence"):
        if k in oldph.get(p["id"], {}):
            p[k] = oldph[p["id"]][k]
keep = lambda pid: int(pid[1:]) < 20
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
print(f"ok: {len(D['tasks'])} tasks ({len(T)} rev 7), {len(reqs)} requirements covered, {len(questions)} questions, {done} done")
