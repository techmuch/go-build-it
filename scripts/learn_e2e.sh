#!/usr/bin/env bash
# Learn e2e (FR-134–FR-139, ADR-012): a demo whose only harness is the scripted one; every
# `$ gbi` example in the lessons and cookbook runs, then web/e2e/learn.spec.ts completes every
# lesson in the practice project and regenerates the Learn and doctor screenshots.
#
#   scripts/learn_e2e.sh
set -euo pipefail
cd "$(dirname "$0")/.."
export GBI_HOME=$(mktemp -d) PORT=${PORT:-7441}
export GBI_URL="http://127.0.0.1:$PORT"
stop() {
	[ -f "$GBI_HOME/server.pid" ] && kill "$(cat "$GBI_HOME/server.pid")" 2>/dev/null || true
	XDG_CONFIG_HOME="$GBI_HOME/h" herdr session stop gbi >/dev/null 2>&1 || true
}
trap stop EXIT
[ -x bin/gbi ] || make build
# Only the scripted harness: Learn and the doctor never start a real one here.
export GBI_EXTRA_CONFIG=$'[harnesses.claude]\nenabled = false\n[harnesses.opencode]\nenabled = false\n[harnesses.antigravity]\nenabled = false\n[harnesses.pi]\nenabled = false'
scripts/demo.sh > "$GBI_HOME/demo.log" 2>&1 || { tail -30 "$GBI_HOME/demo.log"; exit 1; }
scripts/docs_examples.sh
# docs_examples created the practice project; Learn starts from none.
GBI=bin/gbi; bin/gbi learn practice --remove >/dev/null
(cd web && GBI_LEARN=1 GBI_E2E_URL="$GBI_URL" npx playwright test e2e/learn.spec.ts)
echo "PASS: every lesson completed in the practice project; examples and screenshots current"
