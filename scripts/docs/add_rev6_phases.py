"""Add the rev 6 phase (P19: folder trust answered for go-build-it's own folders) to the tracker.

Idempotent: re-running replaces P19 tasks but keeps any status already set on them. Re-parses
docs/requirements.md and fails if a requirement is uncovered or a dependency is unknown.

Usage: python3 scripts/docs/add_rev6_phases.py
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
    dict(id="P19", name="Folder trust", mvp=False,
         goal="Start members, helpers and the doctor without a person: go-build-it answers a harness's folder trust prompt for folders it created, records it, and leaves every other prompt to the person.",
         gate="Folder trust UAT",
         gateCriteria=["A member whose harness starts at a trust prompt in its worktree runs its task without becoming blocked; agent.trusted is in the history",
                       "The doctor's harness tier answers the trust prompt and passes; with the setting off it fails with a human fix naming the prompt",
                       "Real Claude Code and Antigravity trust prompts are recognised and answered; sign-in and permission screens are not",
                       "Folders outside ~/.go-build-it are never answered"]),
]
T = []
def t(id, title, reqs_, views, size, deps, q=None, done=""):
    T.append(dict(id=id, phase=id.split("-")[0], title=title, reqs=reqs_, views=views, size=size,
                  deps=deps, status="done", done=done, q=q or []))

t("P19-01", "Requirements rev 6 (FR-07f, FR-68 amended), ADR-013 (amends ADR-009), plan and tracker", ["FR-07f", "FR-68"], [], "S", [],
  done="docs/requirements.md rev 6; docs/decisions/ADR-013-folder-trust.md")
t("P19-02", "internal/sessions/trust.go: recognise the trust screen and the keys that choose Yes; only folders inside go-build-it's directory; agent.send_keys with a pane fallback", ["FR-07f"], [], "M", ["P19-01"],
  done="TrustKeys, TrustAllowed, AnswerTrust; unit tests hold the real Claude Code and Antigravity screens")
t("P19-03", "Answer it when a member starts, while it waits to be ready, and for helpers; agent.trusted history event", ["FR-07f", "FR-68"], [], "M", ["P19-02"],
  done="members.go, run.go, ask.go; internal/app TestMemberTrustScreenAnswered")
t("P19-04", "Doctor harness tier answers it; names it in the human fix when the setting is off; fake member --start-trust", ["FR-07f", "FR-07b"], ["A-v17"], "S", ["P19-02"],
  done="internal/doctor TestTrustScreenAnswered")
t("P19-05", "Setting [agents] trust_folders (on by default) and the Settings toggle; docs (sessions, settings, troubleshooting), CHANGELOG", ["FR-07f", "FR-07e"], ["A-v15"], "S", ["P19-03", "P19-04"],
  done="Settings → Agent harnesses → Answer folder trust prompts")

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
keep = lambda pid: int(pid[1:]) < 19
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
print(f"ok: {len(D['tasks'])} tasks ({len(T)} rev 6), {len(reqs)} requirements covered, {len(questions)} questions, {done} done")
