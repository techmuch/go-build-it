"""Add the rev 8 phase (P21: tabs from docs/gbi.toml) to the tracker.

Idempotent: re-running replaces P21 tasks but keeps any status already set on them. Re-parses
docs/requirements.md and fails if a requirement is uncovered or a dependency is unknown.

Usage: python3 scripts/docs/add_rev8_phases.py
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
    dict(id="P21", name="Tabs from docs/gbi.toml", mvp=False,
         goal="Let each project decide its Artifacts sub-tabs in docs/gbi.toml: its own roles (e.g. vision), their order, their names, and leaving out roles such as mockups.",
         gate="Tabs UAT",
         gateCriteria=["A vision entry listed first in [entries] is the first sub-tab; tabs = [...] sets order and presence, titles rename",
                       "A project without a mockups entry has no Mockups or Compare tab, and check does not report it",
                       "gbi keeps order, tabs and titles when it rewrites the file; init creates only the roles used"]),
]
T = []
def t(id, title, reqs_, views, size, deps, q=None, done="", status="todo"):
    T.append(dict(id=id, phase=id.split("-")[0], title=title, reqs=reqs_, views=views, size=size,
                  deps=deps, status=status, done=done, q=q or []))

t("P21-01", "Requirements rev 8 (FR-151; FR-140 amended), ADR-014 amendment, mockup 25", ["FR-151", "FR-140"], ["A-v25"], "S", [], done="docs/requirements.md rev 8; ADR-014 amendment", status="done")
t("P21-02", "Manifest: any role, [entries] order, tabs, titles; kept on rewrite; check and init follow the file", ["FR-151", "FR-141", "FR-149", "FR-150"], ["A-v25"], "M", ["P21-01"])
t("P21-03", "Sub-tabs on desktop and phone from the manifest; docs, standard, changelog", ["FR-151", "FR-140", "FR-130"], ["A-v21", "A-M14"], "S", ["P21-02"])

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
keep = lambda pid: int(pid[1:]) < 21
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
print(f"ok: {len(D['tasks'])} tasks ({len(T)} rev 8), {len(reqs)} requirements covered, {len(questions)} questions, {done} done")
