"""Add rev 15 (the setup wizard, the CLI pointed at a server) to the tracker: new tasks in P28
(the Assistant) and P29 (sign-in), and Per user as the default in P31.

Idempotent: re-running replaces these tasks but keeps any status already set on them. Re-parses
docs/requirements.md and fails if a requirement is uncovered or a dependency is unknown.
Run after add_rev14_phases.py.

Usage: python3 scripts/docs/add_rev15_phases.py
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

T = []
def t(id, title, reqs_, views, size, deps, q=None, done="", status="todo"):
    T.append(dict(id=id, phase=id.split("-")[0], title=title, reqs=reqs_, views=views, size=size,
                  deps=deps, status=status, done=done, q=q or []))

# P28: the setup wizard sets up each person's Assistant in the browser.
t("P28-09", "Spike: sign Claude Code, Antigravity and Pi in through a web terminal on a server; record which need an API key", ["FR-220", "FR-221"], ["UM-W2"], "S", ["P27-11"])
t("P28-10", "Setup wizard: choose a harness, sign it in in a terminal on the page (link as a button, paste box), detect success by checking the harness; only its owner sees the terminal", ["FR-219", "FR-220", "FR-224"], ["UM-W1", "UM-W2"], "L", ["P28-09", "P28-02"])
t("P28-11", "Check step: a test prompt, a model that works on the person's plan; API key fallback", ["FR-219", "FR-221", "FR-205"], ["UM-W3"], "M", ["P28-10"])
t("P28-12", "Skip, the Set up chip and the Finish setting up checklist (home, profile, phone); Fix when a sign-in expires, with an inbox item", ["FR-222", "FR-223"], ["UM-W4"], "M", ["P28-10", "P28-03"])
t("P28-13", "Findable everywhere: first sign-in, top bar, home card, profile, Settings, phone home and More, ⌘K, Learn lesson 0, docs", ["FR-228"], ["UM-W4"], "S", ["P28-12"])
t("P28-14", "Shared mode set up by an admin through the same wizard in Settings → Harnesses", ["FR-225"], ["UM-D4"], "S", ["P28-11"])

# P29: the CLI pointed at a server.
t("P29-09", "CLI contexts: gbi context add/use, gbi login per server; every changing command names its server; gbi mcp uses the context", ["FR-226", "FR-166"], [], "M", ["P29-06"])
t("P29-10", "gbi sessions attach: watch and take over your agents' terminals from the CLI", ["FR-227", "FR-66"], [], "M", ["P29-09"])
t("P29-11", "A shared-server install starts in multi-user mode; gbi serve on a workstation stays solo", ["FR-163"], [], "S", ["P29-08"])

ids15 = {x["id"] for x in T}
old = {x["id"]: x for x in D["tasks"]}
for x in T:
    if x["id"] in old:
        x["status"] = old[x["id"]]["status"]
        x["done"] = old[x["id"]].get("done") or x["done"]
D["tasks"] = [x for x in D["tasks"] if x["id"] not in ids15]
pos = {"P28": "P28-08", "P29": "P29-08"}
for x in T:
    anchor = pos[x["phase"]]
    i = next(k for k, y in enumerate(D["tasks"]) if y["id"] == anchor)
    D["tasks"].insert(i, x)
for y in D["tasks"]:
    if y["id"] == "P28-08":
        y["deps"] = sorted(set(y["deps"]) | {"P28-11", "P28-12", "P28-13", "P28-14"})
    if y["id"] == "P29-08":
        y["deps"] = sorted(set(y["deps"]) | {"P29-09", "P29-10"})
    if y["id"] == "P31-01":
        y["title"] = "Shared or Per user setting (Per user is the default in multi-user mode); Settings → Harnesses shows the credential for each harness and model"
        y["reqs"] = sorted(set(y["reqs"]) | {"FR-200"})
for p in D["phases"]:
    if p["id"] == "P28":
        p["name"] = "The personal Assistant, the setup wizard and warm harnesses"
        p["goal"] = "Every user sets up their Assistant in the browser (a harness signed in through a terminal on the page, or an API key) and nothing on their own machine; it is made from the Assistant archetype, tuned in their profile, always running while they are signed in, shown on every screen, the one every side chat talks to, and kept warm."
        p["gateCriteria"] = [c for c in p["gateCriteria"] if not c.startswith("maria finishes")] + ["maria finishes the setup wizard in the browser with Claude Code (or an API key) and her Assistant answers; skipping it leaves a Set up chip and a Finish setting up card"]
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
print(f"ok: {len(D['tasks'])} tasks ({len(T)} rev 15), {len(reqs)} requirements covered, {len(questions)} questions, {done} done")
