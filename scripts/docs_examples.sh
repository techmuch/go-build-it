#!/usr/bin/env bash
# Run every `$ gbi …` example in the lessons and the AI-harness cookbook against a running
# demo instance (FR-139, ADR-012). Fails on the first example that fails.
#
#   GBI_HOME=… GBI_URL=http://127.0.0.1:7431 scripts/docs_examples.sh
set -euo pipefail
cd "$(dirname "$0")/.."
: "${GBI_URL:?set GBI_URL to a running demo (scripts/demo.sh)}"
GBI="${GBI:-$(pwd)/bin/gbi}"
BIN=$(mktemp -d); trap 'rm -rf "$BIN"' EXIT
ln -s "$GBI" "$BIN/gbi"
export PATH="$BIN:$PATH"
gbi learn practice >/dev/null   # the examples use the practice project
n=0
while IFS= read -r line; do
	cmd="${line#\$ }"
	n=$((n + 1))
	if ! out=$(bash -c "$cmd" 2>&1 </dev/null); then
		code=$?
		# gbi doctor exits 1 or 2 when something needs fixing; the example still worked.
		if [[ "$cmd" == "gbi doctor"* ]] && [[ $code -le 2 ]] && grep -q '"results"' <<<"$out"; then continue; fi
		echo "FAIL ($code): $cmd"; echo "$out" | tail -20; exit 1
	fi
done < <(grep -h '^\$ gbi ' docs/learn/*.md docs/user/04-artifacts.md docs/user/19-ai-harness.md)
[ "$n" -gt 5 ] || { echo "FAIL: only $n examples found"; exit 1; }
echo "PASS: $n gbi examples from the lessons and the cookbook ran against $GBI_URL"
