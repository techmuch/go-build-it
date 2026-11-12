#!/usr/bin/env bash
# Seed a demo go-build-it instance using only the CLI (FR-30 parity) and the scripted
# fake harness, so no AI subscription is used. Used for e2e tests and docs screenshots.
#
#   GBI_HOME=/tmp/gbi-demo PORT=7431 scripts/demo.sh
set -euo pipefail
: "${GBI_HOME:?set GBI_HOME to an empty directory}"
PORT="${PORT:-7431}"
GBI="${GBI:-$(pwd)/bin/gbi}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
export GBI_HOME GBI_URL="http://127.0.0.1:$PORT"
if [ -e "$GBI_HOME/gbi.db" ]; then echo "$GBI_HOME already has a database; use a fresh directory" >&2; exit 1; fi
mkdir -p "$GBI_HOME/repos"
gbi() { "$GBI" "$@"; }

# Config: fake harness, listen port.
cat > "$GBI_HOME/config.toml" <<EOF
listen = "127.0.0.1:$PORT"
default_user = "david"
max_concurrent = 2
[harnesses.fake]
enabled = true
binary = "$GBI"
# An isolated herdr: the demo never touches the user's own herdr sessions (ADR-009).
[herdr]
session = "gbi"
config_home = "$GBI_HOME/h"
${GBI_EXTRA_CONFIG:-}
EOF

# A scripted tailscale: the demo's tailnet is made up (devbox.tail3f2a.ts.net), another app
# already owns its root address, and nothing on this machine's real tailnet is read or shown.
mkdir -p "$GBI_HOME/fakebin"
cat > "$GBI_HOME/fakebin/tailscale" <<'TS'
#!/bin/sh
case "$*" in
  "status --json") echo '{"BackendState":"Running","Self":{"DNSName":"devbox.tail3f2a.ts.net."}}' ;;
  "serve status --json") echo '{"TCP":{"443":{"HTTPS":true},"5999":{"TCPForward":"127.0.0.1:5999"}},"Web":{"devbox.tail3f2a.ts.net:443":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:3999"}}}}}' ;;
  *) echo "Serve started and running in the background." ;;
esac
TS
chmod +x "$GBI_HOME/fakebin/tailscale"
PATH="$GBI_HOME/fakebin:$PATH" "$GBI" serve > "$GBI_HOME/server.log" 2>&1 &
echo $! > "$GBI_HOME/server.pid"
for _ in $(seq 1 50); do curl -sf "$GBI_URL/api/system" >/dev/null && break; sleep 0.2; done

mkrepo() { # name purpose
  local d="$GBI_HOME/repos/$1"
  mkdir -p "$d/docs/mockups" "$d/e2e-screenshots"
  git -C "$d" init -q -b main
  printf '# %s\n\n%s\n' "$1" "$2" > "$d/README.md"
  echo "$d"
}
commit() { git -C "$1" add -A && git -C "$1" -c user.name=demo -c user.email=demo@example.com commit -q -m "$2"; }

mockup() { # file title body-html
  cat > "$1" <<EOF
<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>$2</title>
<style>body{margin:0;font:14px -apple-system,sans-serif;background:#f6f7f5}header{background:#264d3b;color:#fff;padding:14px 18px;font-weight:700}main{padding:18px}.card{background:#fff;border-radius:10px;padding:14px;margin:10px 0;box-shadow:0 1px 3px rgba(0,0,0,.1)}.map{height:260px;border-radius:12px;background:linear-gradient(135deg,#cfe3d4,#a9c9b3)}.sheet{background:#fff;border-radius:14px 14px 0 0;padding:14px;margin-top:-40px;position:relative}</style></head>
<body><header>$2</header><main>$3</main></body></html>
EOF
}

# trailmap: design stage, option set + tabletop rehearsal
TM=$(mkrepo trailmap "Offline hiking maps you can use one-handed.")
cat > "$TM/docs/requirements.md" <<'EOF'
# trailmap — Requirements

## 3. Offline maps
- **FR-07**: The map SHALL remain fully usable with no network connection for any downloaded region.
- **FR-08**: The user SHALL be able to download maps per trail or per region.
- **FR-09**: Battery usage while recording a track SHOULD stay under 5% per hour.
- **FR-10**: All primary actions SHALL be reachable with one thumb on a 6.1" phone.
EOF
mockup "$TM/docs/mockups/map-layout-a.html" "A · Full-bleed map" '<div class="map"></div><div class="sheet"><b>Eagle Ridge Loop</b><br>6.2 mi · 1,180 ft · offline ✓</div>'
mockup "$TM/docs/mockups/map-layout-b.html" "B · Split map / list" '<div class="map" style="height:160px"></div><div class="card">Eagle Ridge Loop · 6.2 mi</div><div class="card">Cedar Falls · 3.1 mi</div>'
mockup "$TM/docs/mockups/map-layout-c.html" "C · Trail-first cards" '<div class="card"><div class="map" style="height:90px"></div><b>Eagle Ridge Loop</b></div><div class="card"><div class="map" style="height:90px"></div><b>Cedar Falls</b></div>'
commit "$TM" "requirements and map layout mockups"
# Project documents (rev 7): phone mockups with marked views, screenshots of the built app,
# and one research file added outside gbi (it stays unlabelled until someone labels it).
cat > "$TM/docs/mockups/index.html" <<'EOF'
<!doctype html><html><head><meta charset="utf-8"><title>trailmap · Mockups</title>
<style>body{margin:0;font:15px/1.6 -apple-system,sans-serif;color:#1c2329}nav{display:flex;gap:16px;padding:10px 24px;border-bottom:1px solid #e1e5e8;font-size:13px}nav a{color:#2f6fd1;text-decoration:none}main{padding:8px 24px}</style></head>
<body><nav><b>trailmap</b><a href="../requirements.html">Requirements</a><a href="index.html"><b>Mockups</b></a><a href="../plan.html">Plan</a><a href="../tracker/index.html">Tracker</a></nav>
<main><h1>trailmap · Mockups</h1><p>Two sets:</p><ul><li><a href="phone.html">Phone</a>: the app, view by view (v01–v04)</li><li>Map layout options from OS-0: <a href="map-layout-a.html">A</a> · <a href="map-layout-b.html">B</a> · <a href="map-layout-c.html">C</a></li></ul></main></body></html>
EOF
cat > "$TM/docs/mockups/phone.html" <<'EOF'
<!doctype html><html><head><meta charset="utf-8"><title>trailmap · Phone</title>
<style>body{margin:0;font:15px/1.6 -apple-system,sans-serif;color:#1c2329}nav{padding:10px 24px;border-bottom:1px solid #e1e5e8;font-size:13px}nav a{color:#2f6fd1}main{padding:8px 24px}section{display:flex;gap:24px;align-items:flex-start;padding:18px 0;border-bottom:1px solid #eef1f3}
.ph{width:360px;height:660px;border:8px solid #1d1f22;border-radius:30px;overflow:hidden;position:relative;background:linear-gradient(135deg,#d6e8d9,#b5d1bd)}.tb{height:56px;background:#fff;display:flex;align-items:center;padding:0 16px;font-weight:700}
.btn{position:absolute;left:16px;right:16px;bottom:18px;height:54px;border-radius:14px;background:#0f9d8a;color:#fff;display:grid;place-items:center;font-weight:700}.badge{position:absolute;right:14px;top:68px;background:#1f9d55;color:#fff;border-radius:999px;padding:2px 12px;font-size:13px}
.card{background:#fff;border-radius:12px;margin:12px;padding:16px}.loop{position:absolute;left:80px;top:150px;width:200px;height:320px;border:4px dotted #c0582f;border-radius:50%}</style></head>
<body><nav><a href="index.html">Mockups</a> › Phone</nav><main>
<section id="v01" data-view="v01" data-route="/" data-req="FR-10"><div class="ph"><div class="tb">Trails nearby</div><div class="card"><b>Eagle Ridge Loop</b><br>6.2 mi</div><div class="card"><b>Cedar Falls</b><br>3.1 mi</div><div class="card"><b>Bear Creek</b><br>2.4 mi</div></div>
 <div><h2>v01 · Trail list</h2><p>Nearby trails as big cards, reachable with one thumb (<a href="../requirements.html#FR-10">FR-10</a>).</p></div></section>
<section id="v02" data-view="v02" data-route="/trail/:id" data-req="FR-07 FR-08 FR-10"><div class="ph"><div class="tb">Bear Creek Loop</div><span class="badge">offline ✓</span><div class="loop"></div><div class="btn">⬇ Download this trail · 18 MB</div></div>
 <div><h2>v02 · Trail map</h2><p>The download button sits at the bottom in thumb reach, and shows the size first (<a href="../requirements.html#FR-08">FR-08</a>). The <i>offline ✓</i> badge shows when the trail is on the phone (<a href="../requirements.html#FR-07">FR-07</a>).</p><p>Notes: <a href="notes/v02.md">notes/v02.md</a></p></div></section>
<section id="v03" data-view="v03" data-route="/downloads" data-req="FR-08"><div class="ph"><div class="tb">Downloads</div><div class="card">Bear Creek Loop · 18 MB ✓</div><div class="card">Eagle Ridge Loop · 34 MB ✓</div></div>
 <div><h2>v03 · Downloads</h2><p>Everything on the phone, with sizes (<a href="../requirements.html#FR-08">FR-08</a>).</p></div></section>
<section id="v04" data-view="v04" data-route="/settings/night" data-req="FR-22"><div class="ph" style="background:#10161b"><div class="tb" style="background:#1b252e;color:#fff">Night mode</div></div>
 <div><h2>v04 · Night mode</h2><p>Dim red map for night hikes. Proposed; no requirement yet.</p></div></section>
</main></body></html>
EOF
mkdir -p "$TM/docs/mockups/notes" "$TM/docs/screenshots" "$TM/docs/research"
printf '# v02 notes\n\n- The size comes from the trail pack manifest.\n- Answer to Q-1 decides per trail vs per region.\n' > "$TM/docs/mockups/notes/v02.md"
python3 "$ROOT/scripts/e2e/phone_png.py" "$TM/docs/screenshots/v01.png" list
python3 "$ROOT/scripts/e2e/phone_png.py" "$TM/docs/screenshots/v02.png" map-v2
python3 "$ROOT/scripts/e2e/phone_png.py" "$TM/docs/screenshots/v03.png" downloads
printf '# Trail survey\n\n12 hikers, 3 trails. Most lose signal within 20 minutes.\n' > "$TM/docs/research/trail-survey.md"
printf '# Interviews\n\nRaw notes from five interviews.\n' > "$TM/docs/research/interviews.md"
commit "$TM" "phone mockups, screenshots and research"

# ledgerly: implementation, task flow + review
LG=$(mkrepo ledgerly "Personal finance ledger with bank CSV import.")
cat > "$LG/docs/requirements.md" <<'EOF'
# ledgerly — Requirements

- **FR-21**: ledgerly SHALL guess the column mapping of an uploaded bank CSV.
- **FR-22**: The user SHALL be able to fix the guessed mapping before importing.
- **FR-23**: ledgerly SHALL remember the mapping per bank and reuse it on the next import.
EOF
mockup "$LG/docs/mockups/column-mapping.html" "Map CSV columns" '<div class="card">Date → <b>Date</b></div><div class="card">Amount → <b>Amount</b> (debit/credit?)</div><div class="card">Payee → <b>Description</b></div>'
commit "$LG" "requirements and mockup"

# Archetypes use the scripted fake harness (no subscription use). Saved before importing,
# so the starter team each import creates is made from these revisions (ADR-010).
for p in "pm Pilot" "architect Atlas" "developer Forge" "qa Probe" "reviewer Lens"; do
  set -- $p
  gbi archetypes save --id "$1" --name "$2" --role "$1" --harness fake --instructions "You are $2. Work one task at a time and ask when unsure." >/dev/null
done

gbi projects import --path "$TM" --purpose "Offline hiking maps you can use one-handed." --stage Design >/dev/null
gbi projects import --path "$LG" --purpose "Personal finance ledger with bank CSV import." --stage Implementation >/dev/null

# Teams are per project: the starter team came with the import; add QA and the humans.
for proj in trailmap ledgerly; do
  gbi members add "$proj" --archetype qa >/dev/null
  gbi members add "$proj" --kind human --name david --login david@github --role pm >/dev/null
  gbi members add "$proj" --kind human --name maria --login maria@github --role reviewer >/dev/null
done
gbi members tune forge@ledgerly --tuning $'- Bank CSVs vary: add a fixture for every new bank.\n- Ask before adding a dependency.' >/dev/null

# Standards
gbi standards save git-workflow --title "Git workflow" --body $'# Git workflow\n\n1. One branch per task.\n2. Courier pushes and opens the PR.\n3. Lens review, then a human approves.\n4. Squash-merge.\n5. Releases are tagged YYYY.MM.DD.N and Pages is published.' >/dev/null
gbi standards save frontend --title "Frontend stack" --body $'# Frontend stack\n\n- React + TypeScript (strict), built with Vite\n- Every view has a Playwright test that captures desktop and phone screenshots\n\n## Why\nOne stack means personas reuse patterns.' >/dev/null
gbi standards override frontend --project trailmap --body "Native-feeling mobile web only." --reason "trailmap ships as a PWA" >/dev/null

# Roadmaps
gbi milestones save ledgerly --name "M1 · CSV import" --scope "Import bank CSVs with a guessed, fixable column mapping." --status active --gateName "Dress rehearsal" --gateKind uat --gateCriteria '["Import a real bank CSV","Fix a wrong guess","Mapping remembered"]' >/dev/null
gbi milestones save ledgerly --name "M2 · Categories" --scope "Categorize transactions." --gateName "Category review" --gateKind review >/dev/null
gbi milestones save trailmap --name "M0 · Map screen design" --status active --gateName "Mockup review" --gateKind tabletop --gateCriteria '["Layout chosen","One-handed check"]' >/dev/null

# Values and problem
gbi values add trailmap --text "Offline first" --source "option choice OS-0" >/dev/null
gbi values add trailmap --text "One-handed on trail" --source "feedback on mockup B" >/dev/null
gbi values add trailmap --text "Glanceable in sunlight" --source "rehearsal step 2" --status proposed >/dev/null
gbi problem refine trailmap --text "Hikers lose signal on most trails and need a map they can trust offline, operated one-handed, without draining the phone they rely on for safety." --source "answer to Q-1" >/dev/null

# Option set with mockups
gbi options create --project trailmap --title "Map screen layout" --data '{"options":[
 {"key":"A","title":"Full-bleed map + sheet","summary":"Map fills the screen; trail info slides up.","pros":["Most map area","Sheet in thumb reach"],"cons":["Expanded sheet hides the map"],"mockup":"docs/mockups/map-layout-a.html","aspects":["Full-bleed map","Bottom sheet","Floating zoom"]},
 {"key":"B","title":"Split map / list","summary":"Half map, half nearby trails.","pros":["Compare at a glance"],"cons":["Small map on trail"],"mockup":"docs/mockups/map-layout-b.html","aspects":["Split layout","Elevation sparkline"]},
 {"key":"C","title":"Trail-first cards","summary":"Cards with mini maps; tap for the full map.","pros":["Great for planning","Low battery"],"cons":["Extra tap to navigate"],"mockup":"docs/mockups/map-layout-c.html","aspects":["Static thumbnails","Card feed"]}]}' >/dev/null

# Rehearsal for the ledgerly gate
M1=$(gbi milestones list ledgerly --json | python3 -c 'import json,sys; print(json.load(sys.stdin)[0]["id"])')
gbi rehearsals create --project ledgerly --milestone "$M1" --title "CSV import walkthrough" --kind tabletop --data '{"steps":[
 {"title":"Upload a CSV","scenario":"You upload barclays-oct.csv from the dashboard.","reqs":["FR-21"],"screen":"docs/mockups/column-mapping.html"},
 {"title":"Map CSV columns","scenario":"ledgerly guesses the mapping; confirm or fix it before importing 214 rows.","reqs":["FR-21","FR-22"],"screen":"docs/mockups/column-mapping.html"},
 {"title":"Import again","scenario":"Upload next month. The mapping is remembered.","reqs":["FR-23"]}]}' >/dev/null
G=$(gbi rehearsals list --json | python3 -c 'import json,sys; print(json.load(sys.stdin)[0]["id"])')
gbi rehearsals finding "$G" --step 1 --text "Amount is right but I cannot tell debit vs credit. Show 3 preview rows?" >/dev/null
GBI_PERSONA=probe@ledgerly gbi rehearsals finding "$G" --step 1 --text "Keyboard order skips the Date column." >/dev/null

# Work: one task through the full flow (fake agent → Courier → Lens review)
T1=$(gbi tasks add --project ledgerly --title "Column mapping screen" --description "Build the mapping screen from the approved mockup." --reqs FR-21,FR-22 --milestone "$M1" --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')
gbi tasks run "$T1" >/dev/null
for _ in $(seq 1 60); do st=$(gbi tasks get "$T1" --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["task"]["status"])'); [ "$st" = review ] && break; sleep 0.5; done

# A task that asks a blocking question (the fake agent asks through the CLI API)
T2=$(gbi tasks add --project ledgerly --title "Remember mapping per bank" --description "FAKE:ask" --reqs FR-23 --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')
gbi tasks run "$T2" >/dev/null
sleep 4
GBI_PERSONA=atlas@trailmap gbi questions ask --project trailmap --title "Download maps per trail or per region?" --why "Trail packs are 10–40 MB, regions 300 MB+. This drives storage design." --blocking --data '{"options":[{"key":"A","title":"Per trail","detail":"Small, predictable downloads","recommended":true},{"key":"B","title":"Per region","detail":"One download covers everything"}],"attachments":[{"name":"map-layout-a.html","kind":"html","path":"docs/mockups/map-layout-a.html"},{"name":"sizes.json","kind":"json","body":"{\"trail\": \"10-40 MB\", \"region\": \"300 MB+\"}"}],"reqs":["FR-08"]}' >/dev/null
GBI_PERSONA=probe@ledgerly gbi questions ask --project ledgerly --title "UK bank CSV dates — DD/MM?" --assumption "DD/MM for GB banks" --why "Barclays exports use DD/MM/YYYY." >/dev/null

# More queued work
gbi tasks add --project ledgerly --title "Categorize transactions" --role developer --milestone "$(gbi milestones list ledgerly --json | python3 -c 'import json,sys; print(json.load(sys.stdin)[1]["id"])')" >/dev/null
gbi tasks add --project trailmap --title "Draft offline storage ADR" --role architect --priority 1 >/dev/null
GBI_PERSONA=lens@ledgerly gbi proposals create --title "Shorter developer briefs" --body "Trim task context notes to bullets." --evidence "Tasks with notes over 2k tokens had 18% more rework (demo data)." >/dev/null

# A member waiting for you in its terminal (FR-68): it asks permission before an edit.
T3=$(gbi tasks add --project trailmap --title "Guarded storage migration" --description "FAKE:block" --role developer --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')
gbi tasks run "$T3" >/dev/null
for _ in $(seq 1 100); do
  gbi sessions list --live --json | python3 -c 'import json,sys; sys.exit(0 if any(s["status"]=="blocked" for s in json.load(sys.stdin)) else 1)' && break
  sleep 0.2
done

# Project documents: the standard set (keeps requirements.md and the mockups), labels,
# tasks that cite requirements, Compare decisions and a tracker snapshot.
gbi artifacts init trailmap >/dev/null
gbi artifacts add trailmap docs/research/trail-survey.md --label research >/dev/null
gbi tasks add --project trailmap --title "Trail map screen (v02)" --role developer --reqs FR-07,FR-10 >/dev/null
gbi tasks add --project trailmap --title "Download per trail" --role developer --reqs FR-08 >/dev/null
gbi compare accept trailmap v01 --note "Matches the mockup." >/dev/null
gbi compare accept trailmap v03 --note "Sizes shown; good." >/dev/null
gbi artifacts snapshot trailmap >/dev/null
python3 "$ROOT/scripts/e2e/phone_png.py" "$TM/docs/screenshots/v03.png" list   # a later UAT run changed it: v03 drifts

echo "demo ready at $GBI_URL (server pid $(cat "$GBI_HOME/server.pid"))"
