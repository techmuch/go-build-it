#!/usr/bin/env bash
# End-to-end test of the change log and a guided update through a real restart, while
# demo members are live in herdr
# (FR-06a–FR-06e, ADR-011). Nothing is published: a fake release server serves a new
# build signed with a throwaway key that only these test binaries trust, and an
# isolated herdr is used (never your own herdr sessions).
#
#   scripts/update_e2e.sh            (needs herdr, Go, Node with web/ deps, Chrome)
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
PORT="${PORT:-7433}"
FAKE="127.0.0.1:${FAKE_PORT:-7434}"
OLD="2026.01.01.1"; MID="2026.06.01.1"; NEW="2026.12.31.1"
W="$(mktemp -d /tmp/gbiu.XXXX)" # short: herdr's socket path must fit the Unix limit
export GBI_HOME="$W/home"
SERVER_PID=""; FAKE_PID=""
cleanup() {
  [ -n "$SERVER_PID" ] && kill "$SERVER_PID" 2>/dev/null || true
  [ -n "$FAKE_PID" ] && kill "$FAKE_PID" 2>/dev/null || true
  XDG_CONFIG_HOME="$GBI_HOME/h" herdr session stop gbi >/dev/null 2>&1 || true
  if [ "${KEEP:-}" = 1 ]; then echo "kept $W"; else rm -rf "$W"; fi
}
trap cleanup EXIT

[ -f web/dist/index.html ] || make web
read -r PUB PRIV < <(go run ./scripts/e2e/fakerelease keygen)
KEYFLAG="-X github.com/techmuch/go-build-it/internal/update.PublicKey=$PUB"
ASSET="gbi_$(go env GOOS)_$(go env GOARCH)"
mkdir -p "$W/bin" "$W/rel" "$GBI_HOME"
echo "building $OLD (running) and $NEW (release)"
go build -ldflags "-s -w -X main.version=$OLD $KEYFLAG" -o "$W/bin/gbi" ./cmd/gbi
go build -ldflags "-s -w -X main.version=$NEW $KEYFLAG" -o "$W/rel/$ASSET" ./cmd/gbi
GBI_SIGNING_KEY="$PRIV" "$W/bin/gbi" dev sign "$W/rel/$ASSET"

go build -o "$W/fakerelease" ./scripts/e2e/fakerelease
"$W/fakerelease" serve "$FAKE" "$W/rel" "$NEW" "$MID" > "$W/fake.log" 2>&1 &
FAKE_PID=$!

# Seed the demo (projects, a finished task, and members still live: one blocked waiting
# for the human), so the update runs while members work (FR-08). The demo starts the
# server; the extra config points its updater at the fake release server.
export GBI="$W/bin/gbi" PORT GBI_EXTRA_CONFIG="[updates]
channel = \"stable\"
repo = \"techmuch/go-build-it\"
auto_install = false
api = \"http://$FAKE\""
echo "seeding the demo"
scripts/demo.sh > "$W/demo.log" 2>&1 || { cat "$W/demo.log"; exit 1; }
SERVER_PID="$(cat "$GBI_HOME/server.pid")"
export GBI_URL="http://127.0.0.1:$PORT"
curl -sf "$GBI_URL/api/system" | grep -q "\"version\": \"$OLD\"" || { echo "server did not start"; cat "$GBI_HOME/server.log"; exit 1; }
live() { "$W/bin/gbi" sessions list --live --json | python3 -c 'import json,sys; print(" ".join(sorted(s["id"]+":"+s["status"] for s in json.load(sys.stdin))))'; }
BEFORE="$(live)"
[ -n "$BEFORE" ] || { echo "the demo left no live member"; exit 1; }
echo "live before the update: $BEFORE"

echo "running the browser test"
(cd web && GBI_E2E_URL="http://127.0.0.1:$PORT" GBI_UPDATE_LIVE="$(echo "$BEFORE" | wc -w | tr -d ' ')" GBI_UPDATE_FROM="$OLD" GBI_UPDATE_MID="$MID" GBI_UPDATE_TO="$NEW" npx playwright test e2e/update.spec.ts)

echo "checking the result"
fail() { echo "FAIL: $*"; tail -30 "$GBI_HOME/server.log"; exit 1; }
[ "$("$W/bin/gbi" version)" = "$NEW" ] || fail "installed binary is not $NEW"
[ "$(cat "$W/bin/gbi.prev.version")" = "$OLD" ] || fail "previous version not kept"
[ "$("$W/bin/gbi.prev" version)" = "$OLD" ] || fail "previous binary is not $OLD"
kill -0 "$SERVER_PID" 2>/dev/null || fail "server process is gone (expected an in-place restart)"
curl -sf "http://127.0.0.1:$PORT/api/system" | grep -q "\"version\": \"$NEW\"" || fail "server is not running $NEW"
AFTER="$(live)"
[ "$AFTER" = "$BEFORE" ] || fail "live members changed across the update: before [$BEFORE] after [$AFTER]"
H="$("$W/bin/gbi" history list --type update.completed --json)"
echo "$H" | grep -q "Updated from $OLD to $NEW" || fail "history has no update.completed: $H"
grep -q "restarting into the updated binary" "$GBI_HOME/server.log" || fail "no restart in the log"
echo "PASS: $OLD → $NEW through a real restart (same PID $SERVER_PID); members kept working [$AFTER]; previous kept for rollback; history recorded"
