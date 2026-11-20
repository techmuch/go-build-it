#!/usr/bin/env bash
# Harness check: proves a real AI harness works inside herdr the way go-build-it uses it.
# Uses a few prompts on your subscription (a minute or two of a small model's time).
#
#   scripts/harness_e2e.sh                 # Claude Code
#   HARNESS=pi scripts/harness_e2e.sh      # another harness
#   KEEP=1 scripts/harness_e2e.sh          # leave the server and herdr running to look around
#
# It runs in its own GBI_HOME and herdr session (gbi-hcheck); your installation, your
# projects and your default herdr session are never touched. Checks, in order:
#   1. doctor: herdr and the harness pass the static, live and harness tiers
#   2. a member starts, takes its task, writes a file and runs shell and gbi commands
#      without stopping for a permission, and the task reaches review
#   3. the reviewer helper answers (folder trust, prompt delivery, answer file)
#   4. a blocking question pauses the task; answering it resumes the task by itself
#   5. the same member takes a second task in the same conversation
set -euo pipefail
cd "$(dirname "$0")/.."
HARNESS=${HARNESS:-claude}
MODEL=${MODEL:-}
PORT=${PORT:-7442}
# A short path: herdr's socket lives under GBI_HOME, and macOS limits socket paths to 104 bytes.
export GBI_HOME=$(mktemp -d /tmp/gbihc.XXXX) GBI_URL="http://127.0.0.1:$PORT"
SESSION=gbi-hcheck
GBI="$(pwd)/bin/gbi"
[ -x "$GBI" ] || make build >/dev/null
gbi() { "$GBI" "$@"; }
pass=0
ok() { pass=$((pass + 1)); printf 'PASS  %s\n' "$*"; }
fail() {
	printf 'FAIL  %s\n' "$*"
	echo "--- last history:"; gbi history list --limit 12 --json 2>/dev/null | python3 -c "
import sys,json
for e in json.load(sys.stdin): print(' ',e.get('type'),'|',(e.get('summary') or '')[:90],'|',str(e.get('why') or '')[:120])" || true
	# What a stopped harness is showing: the reason it needs a person.
	XDG_CONFIG_HOME="$GBI_HOME/h" herdr --session "$SESSION" pane list 2>/dev/null | python3 -c "
import sys,json
for p in json.load(sys.stdin)['result']['panes']:
    if p.get('agent_status') == 'blocked': print(p['pane_id'], p.get('display_agent') or '')" | while read -r pane who; do
		echo "--- $who is waiting on this screen ($pane):"
		XDG_CONFIG_HOME="$GBI_HOME/h" herdr --session "$SESSION" pane read "$pane" 2>/dev/null | grep -v '^[[:space:]]*$' | tail -15
	done
	echo "--- data: $GBI_HOME"
	exit 1
}
stop() {
	[ -n "${KEEP:-}" ] && { echo "kept: $GBI_URL, herdr --session $SESSION (XDG_CONFIG_HOME=$GBI_HOME/h)"; return; }
	[ -f "$GBI_HOME/server.pid" ] && kill "$(cat "$GBI_HOME/server.pid")" 2>/dev/null || true
	XDG_CONFIG_HOME="$GBI_HOME/h" herdr session stop "$SESSION" >/dev/null 2>&1 || true
}
trap stop EXIT
# wait_for SECONDS DESCRIPTION COMMAND…: polls until the command succeeds.
wait_for() {
	local secs=$1 what=$2; shift 2
	for _ in $(seq 1 "$secs"); do "$@" >/dev/null 2>&1 && return 0; sleep 1; done
	fail "$what (waited ${secs}s)"
}
task_status() { gbi tasks get "$1" --json | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('task',d)['status'])"; }
status_is() { [ "$(task_status "$1")" = "$2" ]; }
events() { gbi history list --type "$1" --json | python3 -c "import sys,json; print(len(json.load(sys.stdin)))"; }

BIN=$(command -v "$HARNESS" || true)
[ -n "$BIN" ] || fail "$HARNESS is not on PATH"
cat > "$GBI_HOME/config.toml" <<EOF
listen = "127.0.0.1:$PORT"
default_user = "tester"
max_concurrent = 2
[harnesses.$HARNESS]
enabled = true
binary = "$BIN"
model = "$MODEL"
[herdr]
session = "$SESSION"
config_home = "$GBI_HOME/h"
[updates]
auto_install = false
EOF
for other in claude antigravity opencode pi; do
	[ "$other" = "$HARNESS" ] || printf '[harnesses.%s]\nenabled = false\n' "$other" >> "$GBI_HOME/config.toml"
done
"$GBI" serve > "$GBI_HOME/server.log" 2>&1 &
echo $! > "$GBI_HOME/server.pid"
wait_for 30 "the server starts" curl -sf "$GBI_URL/api/system"
echo "harness check: $HARNESS ($BIN) · data $GBI_HOME"

# 1. Doctor.
if gbi doctor --tier harness --harness "$HARNESS" --json > "$GBI_HOME/doctor.json" 2>&1; then
	ok "doctor: herdr and $HARNESS pass the static, live and harness tiers"
else
	python3 -c "
import json; d=json.load(open('$GBI_HOME/doctor.json'))
for c in d.get('checks',[]):
    if c.get('status') not in ('ok','skip'): print('   ',c.get('status'),c.get('id'),'|',str(c.get('detail'))[:160])" || cat "$GBI_HOME/doctor.json"
	fail "doctor"
fi

# A tiny project with one AI member and a reviewer, both on this harness.
P="$GBI_HOME/repos/hcheck"; mkdir -p "$P"
git -C "$P" init -q -b main
printf '# hcheck\n' > "$P/README.md"
git -C "$P" add -A && git -C "$P" -c user.name=t -c user.email=t@t commit -qm init
gbi projects import --path "$P" --id hcheck --purpose "Harness check" >/dev/null
# Importing adds the starter team; point its developer and reviewer archetypes at this harness.
for a in developer reviewer; do
	gbi archetypes get "$a" --json | python3 -c "
import sys,json; d=json.load(sys.stdin); d=d.get('archetype',d); print(d['name']); print(d['role'])" > "$GBI_HOME/arch"
	gbi archetypes save --id "$a" --name "$(sed -n 1p "$GBI_HOME/arch")" --role "$(sed -n 2p "$GBI_HOME/arch")" --harness "$HARNESS" >/dev/null || fail "set the $a archetype to $HARNESS"
done
gbi members list hcheck --json | python3 -c "
import sys,json
bad=[m['id']+':'+str(m.get('harness')) for m in json.load(sys.stdin) if m.get('kind')!='human' and m.get('role') in ('developer','reviewer') and m.get('harness')!='$HARNESS']
sys.exit('members not on $HARNESS: '+', '.join(bad) if bad else 0)" || fail "the developer and reviewer members use $HARNESS"
DEV=$(gbi members list hcheck --json | python3 -c "import sys,json; print([m['id'] for m in json.load(sys.stdin) if m.get('role')=='developer'][0])")

# 2. A member works without stopping.
T1=$(gbi tasks add --project hcheck --role developer --title "Write hello.txt" --description "Create the file hello.txt containing the single word ok. Then run these two commands and nothing else: git status, and gbi tasks list --project hcheck. Do not ask questions. Say done when finished." --json | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
gbi tasks run "$T1" --member "$DEV" >/dev/null
wait_for 420 "$T1 reaches review" status_is "$T1" review
[ "$(events session.blocked)" = 0 ] || fail "the member stopped for a permission or prompt (session.blocked)"
[ "$(events session.failed)" = 0 ] || fail "a session failed"
WT=$(gbi sessions list --json | python3 -c "import sys,json; print(json.load(sys.stdin)[0]['workdir'])")
git -C "$WT" show HEAD:hello.txt 2>/dev/null | grep -qi ok || fail "hello.txt was not committed on the task branch"
ok "member: took the task, wrote a file, ran git and gbi without a permission stop; $T1 in review ($(events session.resubmitted) resends)"

# 3. The reviewer helper answered.
R1=$(gbi reviews list --json | python3 -c "import sys,json; print([r['id'] for r in json.load(sys.stdin) if r['taskId']=='$T1'][0])")
SUM=$(gbi reviews get "$R1" --json | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('review',d)['assessment']['summary'])")
case "$SUM" in "" | "Lens could not assess"*) fail "reviewer helper: ${SUM:-no summary}" ;; esac
ok "reviewer helper answered ($(events helper.resubmitted) resends, $(events agent.trusted) trust screens answered)"
gbi reviews verdict "$R1" --verdict approve --comment "harness check" >/dev/null

# 4. A blocking question pauses the task, and the answer resumes it.
T2=$(gbi tasks add --project hcheck --role developer --title "Ask, then write answer.txt" --description "First ask the person one blocking question with exactly this command: gbi questions ask --project hcheck --task TASK_ID --title 'Pick a colour' --why 'harness check' --blocking --options '[{\"key\":\"A\",\"title\":\"red\"},{\"key\":\"B\",\"title\":\"blue\"}]' (TASK_ID is your task's ID, named at the top of .gbi-task.md), then stop and wait. When you get the answer, write the chosen colour (one word) to answer.txt and say done." --json | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
gbi tasks run "$T2" --member "$DEV" >/dev/null
open_q() { gbi questions list --json | python3 -c "import sys,json; qs=[q for q in json.load(sys.stdin) if q.get('taskId')=='$T2' and q.get('status')=='open']; print(qs[0]['id']) if qs else sys.exit(1)"; }
wait_for 300 "$T2 asks a question" open_q
Q=$(open_q)
wait_for 120 "$T2 pauses for the question" status_is "$T2" blocked
gbi questions answer "$Q" --choice B >/dev/null
wait_for 420 "$T2 resumes and reaches review" status_is "$T2" review
WT2=$(gbi sessions list --json | python3 -c "import sys,json; print([s['workdir'] for s in json.load(sys.stdin) if s['taskId']=='$T2'][0])")
git -C "$WT2" show HEAD:answer.txt 2>/dev/null | grep -qi blue || fail "answer.txt does not hold the answer (blue)"
ok "blocking question: $T2 paused, the answer resumed it without a manual run, and the work used the answer"

# 5. The same member takes another task in its conversation.
T3=$(gbi tasks add --project hcheck --role developer --title "Write bye.txt" --description "Create bye.txt containing the single word bye. Do not ask questions. Say done." --json | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
gbi tasks run "$T3" --member "$DEV" >/dev/null
wait_for 300 "$T3 reaches review" status_is "$T3" review
ok "second task in the same member's conversation reached review"

[ "$(events session.blocked)" = 0 ] || fail "a member stopped for a permission or prompt during the run"
echo "PASS: $pass checks; $HARNESS works in herdr as go-build-it uses it"
