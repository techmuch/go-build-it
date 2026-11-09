#!/usr/bin/env bash
# Seed a demo go-build-it instance using only the CLI (FR-30 parity) and the scripted
# fake harness, so no AI subscription is used. Used for e2e tests and docs screenshots.
#
#   GBI_HOME=/tmp/gbi-demo PORT=7431 scripts/demo.sh
set -euo pipefail
: "${GBI_HOME:?set GBI_HOME to an empty directory}"
PORT="${PORT:-7431}"
GBI="${GBI:-$(pwd)/bin/gbi}"
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
EOF

"$GBI" serve > "$GBI_HOME/server.log" 2>&1 &
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

gbi projects import --path "$TM" --purpose "Offline hiking maps you can use one-handed." --stage Design >/dev/null
gbi projects import --path "$LG" --purpose "Personal finance ledger with bank CSV import." --stage Implementation >/dev/null

# Team (scripted fake harness: no subscription use)
gbi teams save --id core --name Core >/dev/null
for p in "pilot Pilot pm" "atlas Atlas architect" "forge Forge developer" "probe Probe qa" "lens Lens reviewer"; do
  set -- $p
  gbi members save --id "$1" --name "$2" --kind ai --role "$3" --harness fake --teamId core --instructions "You are $2. Work one task at a time and ask when unsure." >/dev/null
done
gbi members save --id david --name david --kind human --role pm --teamId core --login david@github >/dev/null
gbi members save --id maria --name maria --kind human --role reviewer --teamId core --login maria@github >/dev/null
gbi projects update trailmap --teamId core >/dev/null
gbi projects update ledgerly --teamId core >/dev/null

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
GBI_PERSONA=probe gbi rehearsals finding "$G" --step 1 --text "Keyboard order skips the Date column." >/dev/null

# Work: one task through the full flow (fake agent → Courier → Lens review)
T1=$(gbi tasks add --project ledgerly --title "Column mapping screen" --description "Build the mapping screen from the approved mockup." --reqs FR-21,FR-22 --milestone "$M1" --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')
gbi tasks run "$T1" >/dev/null
for _ in $(seq 1 60); do st=$(gbi tasks get "$T1" --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["task"]["status"])'); [ "$st" = review ] && break; sleep 0.5; done

# A task that asks a blocking question (the fake agent asks through the CLI API)
T2=$(gbi tasks add --project ledgerly --title "Remember mapping per bank" --description "FAKE:ask" --reqs FR-23 --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')
gbi tasks run "$T2" >/dev/null
sleep 4
GBI_PERSONA=atlas gbi questions ask --project trailmap --title "Download maps per trail or per region?" --why "Trail packs are 10–40 MB, regions 300 MB+. This drives storage design." --blocking --data '{"options":[{"key":"A","title":"Per trail","detail":"Small, predictable downloads","recommended":true},{"key":"B","title":"Per region","detail":"One download covers everything"}],"attachments":[{"name":"map-layout-a.html","kind":"html","path":"docs/mockups/map-layout-a.html"},{"name":"sizes.json","kind":"json","body":"{\"trail\": \"10-40 MB\", \"region\": \"300 MB+\"}"}],"reqs":["FR-08"]}' >/dev/null
GBI_PERSONA=probe gbi questions ask --project ledgerly --title "UK bank CSV dates — DD/MM?" --assumption "DD/MM for GB banks" --why "Barclays exports use DD/MM/YYYY." >/dev/null

# More queued work
gbi tasks add --project ledgerly --title "Categorize transactions" --role developer --milestone "$(gbi milestones list ledgerly --json | python3 -c 'import json,sys; print(json.load(sys.stdin)[1]["id"])')" >/dev/null
gbi tasks add --project trailmap --title "Draft offline storage ADR" --role architect --priority 1 >/dev/null
GBI_PERSONA=lens gbi proposals create --title "Shorter developer briefs" --body "Trim task context notes to bullets." --evidence "Tasks with notes over 2k tokens had 18% more rework (demo data)." >/dev/null

echo "demo ready at $GBI_URL (server pid $(cat "$GBI_HOME/server.pid"))"
