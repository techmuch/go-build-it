"""Add the rev 9 phases (P22: world-class starter archetypes and standards; P23: remote access) to the tracker.

Idempotent: re-running replaces P22-P23 tasks but keeps any status already set on them. Re-parses
docs/requirements.md and fails if a requirement is uncovered or a dependency is unknown.

Usage: python3 scripts/docs/add_rev9_phases.py
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
    dict(id="P22", name="Starter archetypes & standards", mvp=False,
         goal="Give every new installation strong starter personas and standards: full role instructions for the six archetypes, upgraded in place when never edited, and the owner's standards preloaded.",
         gate="Personas review",
         gateCriteria=["Each starter archetype states its mission, what good looks like, how it works with gbi commands, when it hands back and what it never does",
                       "An unedited starter is upgraded as a new revision, keeping the owner's harness and model; edited ones are untouched; members move through the change review",
                       "A fresh installation has the six archetypes and the starter standards; an existing one is not reseeded"]),
    dict(id="P23", name="Remote access", mvp=False,
         goal="Prove go-build-it is served on the tailnet, is this go-build-it and is private; show what else the machine serves; never take another service's address; a QR code for a phone.",
         gate="Remote access UAT",
         gateCriteria=["The checklist matches the Serve rule exactly, proves the round trip reaches this go-build-it, and fails when Funnel is on",
                       "With another service on 443, setup names it, proposes go-build-it's own port and runs only after confirmation; stop removes only go-build-it's rule",
                       "A QR code of the working URL appears in Settings and in gbi tailscale qr; gbi doctor reports tailscale.serve"]),
]
T = []
def t(id, title, reqs_, views, size, deps, q=None, done="", status="todo"):
    T.append(dict(id=id, phase=id.split("-")[0], title=title, reqs=reqs_, views=views, size=size,
                  deps=deps, status=status, done=done, q=q or []))

t("P22-01", "Requirements rev 9 (FR-12f, FR-50a); mockup 10 note", ["FR-12f", "FR-50a"], ["A-v10"], "S", [], done="docs/requirements.md rev 9", status="done")
t("P22-02", "Starter archetype instructions for PM, architect, developer, QA, reviewer and release manager; upgrade unedited starters; role descriptions; release notes follow the release manager", ["FR-12f", "FR-12", "FR-12b", "FR-93"], ["A-v10"], "M", ["P22-01"], done="internal/team/starters.go; TestStarterInstructionsUpgrade", status="done")
t("P22-03", "New installations start with the starter standards from docs/standards; never reseed", ["FR-50a", "FR-50"], [], "S", ["P22-01"], done="SeedStandards; TestNewInstallationDefaults", status="done")
t("P22-04", "User docs (starter archetypes table) and CHANGELOG", ["FR-12f", "FR-130"], [], "S", ["P22-02", "P22-03"], done="docs/user/22-archetypes-and-members.md", status="done")
t("P23-01", "Requirements rev 9 (FR-20a-FR-20d), ADR-015, mockup 27", ["FR-20a", "FR-20b", "FR-20c", "FR-20d"], ["A-v27"], "S", [], done="ADR-015; mockup v27", status="done")
t("P23-02", "Serve detection: exact rule, round trip to this go-build-it, Funnel, other services and their programs; setup never replaces an address; stop removes only its own rule; owner only; history", ["FR-20", "FR-20a", "FR-20b"], ["A-v27"], "M", ["P23-01"], done="internal/tailscale; tailscale_test.go", status="done")
t("P23-03", "Settings: Remote access card with the checklist, other services, proposal and confirm, stop, and the QR code; gbi tailscale qr", ["FR-20a", "FR-20b", "FR-20c"], ["A-v27", "A-v13"], "M", ["P23-02"], done="Org.tsx RemoteAccessCard; gbi tailscale qr", status="done")
t("P23-04", "Doctor check tailscale.serve and its troubleshooting section", ["FR-20d", "FR-07c"], [], "S", ["P23-02"], done="doctor tailscale.serve; troubleshooting", status="done")
t("P23-05", "Scripted tailscale in the demo; UAT steps and screenshots; user docs; CHANGELOG", ["FR-133", "FR-130", "FR-06a"], ["A-v27"], "S", ["P23-03", "P23-04"], done="demo.sh scripted tailscale; UAT 13; 31+14+3 e2e passed", status="done")
t("P23-06", "Release 2026.11.12.1: signed binaries, install check", ["FR-05", "FR-06"], [], "S", ["P22-04", "P23-05"])

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
keep = lambda pid: int(pid[1:]) < 22
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
print(f"ok: {len(D['tasks'])} tasks ({len(T)} rev 9), {len(reqs)} requirements covered, {len(questions)} questions, {done} done")
