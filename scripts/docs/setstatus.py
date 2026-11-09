"""Usage: setstatus.py STATUS ID [ID...]   set task status (todo | in-progress | blocked | done)
       setstatus.py gate PHASE "evidence"  record an owner-approved gate
Updates docs/tracker/tracker-data.js in place."""
import json, sys, datetime, re
import os
P = os.path.join(os.path.dirname(__file__), "..", "..", "docs", "tracker", "tracker-data.js")
src = open(P).read()
head, body = src[:src.index("window.TRACKER")], src[src.index("{"):src.rindex("}") + 1]
D = json.loads(body)
status, ids = sys.argv[1], sys.argv[2:]
if status == "gate":
    ph = {p["id"]: p for p in D["phases"]}[ids[0]]
    open_tasks = [t["id"] for t in D["tasks"] if t["phase"] == ids[0] and t["status"] != "done"]
    if open_tasks:
        sys.exit("gate refused, open tasks: " + ", ".join(open_tasks))
    ph["gateApproved"] = datetime.date.today().isoformat()
    ph["gateApprovedBy"] = "david (owner)"
    if len(ids) > 1:
        ph["gateEvidence"] = ids[1]
    D["updated"] = datetime.date.today().isoformat()
    open(P, "w").write(head + "window.TRACKER = " + json.dumps(D, indent=1, ensure_ascii=False) + ";\n")
    print(f"gate {ids[0]} approved")
    sys.exit(0)
assert status in ("todo", "in-progress", "blocked", "done"), status
known = {t["id"]: t for t in D["tasks"]}
for i in ids:
    if i not in known:
        sys.exit("unknown task " + i)
    known[i]["status"] = status
D["updated"] = datetime.date.today().isoformat()
open(P, "w").write(head + "window.TRACKER = " + json.dumps(D, indent=1, ensure_ascii=False) + ";\n")
done = sum(1 for t in D["tasks"] if t["status"] == "done")
print(f"{len(ids)} → {status}; {done}/{len(D['tasks'])} done")
