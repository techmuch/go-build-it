"""Add the rev 10 phase (P24: model discovery and the landing link) to the tracker.

Idempotent: re-running replaces P24 tasks but keeps any status already set on them. Re-parses
docs/requirements.md and fails if a requirement is uncovered or a dependency is unknown.

Usage: python3 scripts/docs/add_rev10_phases.py
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
    dict(id="P24", name="Model discovery & landing link", mvp=False,
         goal="Choose models from what each harness actually offers, in a searchable list that still accepts any name; the logo and title lead to the landing page.",
         gate="Model picker review",
         gateCriteria=["Archetype and member model fields list the harness's models (agy, pi, opencode asked; Claude Code built in); any name can be typed, with a warning when unknown",
                       "Lists are cached per harness, refresh on request or daily in the background, and a failed refresh keeps the last good list; gbi harnesses models returns the same",
                       "Clicking or pressing Enter on the logo or title opens the project catalog"]),
]
T = []
def t(id, title, reqs_, views, size, deps, q=None, done="", status="todo"):
    T.append(dict(id=id, phase=id.split("-")[0], title=title, reqs=reqs_, views=views, size=size,
                  deps=deps, status=status, done=done, q=q or []))

t("P24-01", "Requirements rev 10 (FR-63a, FR-34a); mockup 10", ["FR-63a", "FR-34a"], ["A-v10"], "S", [], done="docs/requirements.md rev 10", status="done")
t("P24-02", "Model lists per harness profile: agy models, pi --list-models, opencode models, Claude Code built in; cached route gbi harnesses models", ["FR-63a", "FR-61", "FR-30"], [], "M", ["P24-01"], done="internal/harness/models.go; internal/app/models.go", status="done")
t("P24-03", "ModelPicker for archetype default model and member override", ["FR-63a", "FR-12", "FR-12a"], ["A-v10", "A-v10b"], "S", ["P24-02"], done="Team.tsx ModelPicker", status="done")
t("P24-04", "Logo and title link to the landing page", ["FR-34a", "FR-34"], ["A-v01"], "S", [], done="shell.tsx; keyboard e2e", status="done")
t("P24-05", "Tests, UAT step, docs and CHANGELOG", ["FR-133", "FR-130", "FR-06a"], ["A-v10"], "S", ["P24-03", "P24-04"])
t("P24-06", "Release 2026.11.12.2: signed binaries, install check", ["FR-05", "FR-06"], [], "S", ["P24-05"])

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
keep = lambda pid: int(pid[1:]) < 24
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
print(f"ok: {len(D['tasks'])} tasks ({len(T)} rev 10), {len(reqs)} requirements covered, {len(questions)} questions, {done} done")
