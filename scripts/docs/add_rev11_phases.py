"""Add the rev 11 phase (P25: phones get the mobile UI) to the tracker.

Idempotent: re-running replaces P25 tasks but keeps any status already set on them. Re-parses
docs/requirements.md and fails if a requirement is uncovered or a dependency is unknown.

Usage: python3 scripts/docs/add_rev11_phases.py
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
    dict(id="P25", name="Phones get the mobile UI", mvp=False,
         goal="A phone always reaches a usable screen: the QR code and any desktop link open the mobile UI; the desktop site stays usable on a narrow screen.",
         gate="Phone review",
         gateCriteria=["Scanning the Remote access QR code opens /m/; the address is written under the code and gbi tailscale qr prints it",
                       "On a phone, a desktop link (question, project, documents, session) opens the matching mobile screen; Desktop site sticks per device until Mobile UI",
                       "On a narrow screen the desktop UI has a ☰ menu (closes on a choice, a tap outside, Esc) and no page is wider than the screen"]),
]
T = []
def t(id, title, reqs_, views, size, deps, q=None, done="", status="todo"):
    T.append(dict(id=id, phase=id.split("-")[0], title=title, reqs=reqs_, views=views, size=size,
                  deps=deps, status=status, done=done, q=q or []))

t("P25-01", "Requirements rev 11 (FR-35a, FR-20c); mockups M9, M15, 27", ["FR-35a", "FR-20c"], ["A-M9", "A-M15", "A-v27"], "S", [], done="docs/requirements.md rev 11")
t("P25-02", "QR code opens the mobile UI: PhoneURL in the status, written under the code, gbi tailscale qr", ["FR-20c"], ["A-v27"], "S", ["P25-01"], done="internal/tailscale; Org.tsx RemoteAccessCard")
t("P25-03", "Phone redirect to the matching mobile screen; Desktop site per device; Mobile UI clears it", ["FR-35a", "FR-35"], ["A-M9", "A-M15"], "S", ["P25-01"], done="web/src/phone.ts; App.tsx; Mobile.tsx More")
t("P25-04", "☰ drawer for the desktop UI on narrow screens; desktop pages fit the width", ["FR-35a", "FR-34"], ["A-M15"], "S", ["P25-01"], done="shell.tsx; styles.css")
t("P25-05", "Lessons started on the phone tick their visit steps", ["FR-134", "FR-35"], ["A-M13"], "S", ["P25-03"], done="Mobile.tsx LearnScreen")
t("P25-06", "Tests, UAT and keyboard steps, docs and CHANGELOG", ["FR-133", "FR-130", "FR-06a"], ["A-M15"], "S", ["P25-02", "P25-03", "P25-04", "P25-05"])
t("P25-07", "Release 2026.11.12.3: signed binaries, install check", ["FR-05", "FR-06"], [], "S", ["P25-06"])

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
keep = lambda pid: int(pid[1:]) < 25
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
print(f"ok: {len(D['tasks'])} tasks ({len(T)} rev 11), {len(reqs)} requirements covered, {len(questions)} questions, {done} done")
