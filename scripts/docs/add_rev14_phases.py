"""Add the rev 14 phase (P34: per-user workspaces on a Linux server) to the tracker.

Idempotent: re-running replaces P34 tasks but keeps any status already set on them. Re-parses
docs/requirements.md and fails if a requirement is uncovered or a dependency is unknown.
Run after add_rev13_phases.py.

Usage: python3 scripts/docs/add_rev14_phases.py
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
    dict(id="P34", name="Per-user workspaces on a Linux server", mvp=False,
         goal="On a multi-user Linux server every user works in their own isolated workspace: images pulled with go-containerregistry and run by gVisor with no daemon; their Assistant, agent work, worktrees, credentials and herdr inside; code leaves only through review.",
         gate="Two users, two workspaces",
         gateCriteria=["On a stock Linux server, david and maria each get a workspace; from inside maria's, nothing of david's (files, processes, credentials) or the server's data can be seen",
                       "An agent task runs in its starter's workspace, is watched and taken over from the UI, and its change leaves only through Courier and review",
                       "A workspace starts in under 3 s, and a real task (npm ci, tests, git) runs within 10% of the host or with a reasoned exception (NFR-25)",
                       "The doctor explains what a host is missing for the runtime; solo mode, macOS and Windows behave as before"]),
]
T = []
def t(id, title, reqs_, views, size, deps, q=None, done="", status="todo"):
    T.append(dict(id=id, phase=id.split("-")[0], title=title, reqs=reqs_, views=views, size=size,
                  deps=deps, status=status, done=done, q=q or []))

t("P34-01", "Spike: gVisor runs a workspace image built from go-containerregistry, rootless or least privilege; herdr and harnesses work inside; measure start and task cost against runc and the host", ["FR-208", "FR-209", "NFR-25"], [], "M", ["P28-08"], q=["Q18", "Q20"])
t("P34-02", "Decide ADR-017 from the spike (accept, or fall back to runc by default or containerd)", ["FR-208"], [], "S", ["P34-01"], q=["Q18", "Q19", "Q20"])
t("P34-03", "Runtime interface (create, start, run, attach, stop, remove, use) with gVisor and runc implementations", ["FR-208"], [], "L", ["P34-02"])
t("P34-04", "Images: pull by digest with go-containerregistry, unpack per user, persistent home; default workspace image (git, toolchains, herdr, harnesses); project images from a registry", ["FR-209"], [], "L", ["P34-03"], q=["Q19"])
t("P34-05", "Limits (CPU, memory, disk, processes) per everyone, group or user; use shown in Settings, profile and the dev bar", ["FR-210"], [], "M", ["P34-03"])
t("P34-06", "Network allow-list per workspace; go-build-it's API reachable only through the user's token", ["FR-211"], [], "M", ["P34-03"])
t("P34-07", "herdr inside each workspace; terminal view, output and blocked detection across the boundary; watch by project access, control only your own", ["FR-214", "FR-66", "FR-68"], [], "L", ["P34-03"])
t("P34-08", "Worktrees and clones inside workspaces; code leaves only as commits through Courier and review; agent work runs in its starter's workspace", ["FR-207", "FR-212", "FR-213"], [], "L", ["P34-07"])
t("P34-09", "Lifecycle: create, start, stop when idle, keep, reset or remove by an admin, all recorded; image updates staged and reversible", ["FR-215", "FR-218"], [], "M", ["P34-04", "P34-08"])
t("P34-10", "Workspace shell from the UI (desktop and phone) and gbi workspace shell; image, limits, use and network on the profile", ["FR-217"], [], "M", ["P34-07"])
t("P34-11", "Doctor: user namespaces, subordinate IDs, cgroups v2 delegation, KVM; solo mode, macOS and Windows unchanged; NFR-20 in multi-user mode", ["FR-216", "NFR-20"], [], "M", ["P34-03"])
t("P34-12", "Tests (isolation checks between two users, harness check inside a workspace), docs, CHANGELOG, release", ["FR-133", "NFR-23", "NFR-25"], [], "M", ["P34-09", "P34-10", "P34-11"])

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
keep = lambda pid: int(pid[1:]) < 34
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
print(f"ok: {len(D['tasks'])} tasks ({len(T)} rev 14), {len(reqs)} requirements covered, {len(questions)} questions, {done} done")
