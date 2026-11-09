"""Usage: setstatus.py STATUS ID [ID...]  — updates docs/tracker/tracker-data.js in place."""
import json, sys, datetime, re
import os
P = os.path.join(os.path.dirname(__file__), "..", "..", "docs", "tracker", "tracker-data.js")
src = open(P).read()
head, body = src[:src.index("window.TRACKER")], src[src.index("{"):src.rindex("}") + 1]
D = json.loads(body)
status, ids = sys.argv[1], sys.argv[2:]
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
