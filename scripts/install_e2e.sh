#!/usr/bin/env bash
# Install e2e (FR-02a, FR-02b, ADR-012): scripts/install.sh against a fake signed release,
# a tampered one and a missing one. Then the installed gbi passes the doctor against an
# isolated herdr with the scripted harness. Nothing is published.
#
#   scripts/install_e2e.sh
set -euo pipefail
cd "$(dirname "$0")/.."
W=$(mktemp -d)
PORT=${PORT:-7463}
cleanup() {
	[ -n "${FR:-}" ] && kill "$FR" 2>/dev/null || true
	[ -n "${HOMEH:-}" ] && XDG_CONFIG_HOME="$HOMEH" herdr session stop gbi >/dev/null 2>&1 || true
	rm -rf "$W"
}
trap cleanup EXIT
fail() { echo "FAIL: $*" >&2; exit 1; }

TAG=2026.12.31.1
read -r PUB PRIV < <(go run ./scripts/e2e/fakerelease keygen)
os=$(uname -s | tr A-Z a-z); arch=$(uname -m); [ "$arch" = x86_64 ] && arch=amd64; [ "$arch" = aarch64 ] && arch=arm64
ASSET="gbi_${os}_${arch}"

mkdir -p "$W/good" "$W/bad"
echo "building $TAG"
CGO_ENABLED=0 go build -ldflags "-X main.version=$TAG" -o "$W/good/$ASSET" ./cmd/gbi
GBI_SIGNING_KEY="$PRIV" "$W/good/$ASSET" dev sign "$W/good/$ASSET" || fail "sign"
cp "$W/good/"* "$W/bad/"
printf 'tampered' >> "$W/bad/$ASSET"

serve() { # dir
	[ -n "${FR:-}" ] && kill "$FR" 2>/dev/null && wait "$FR" 2>/dev/null || true
	go build -o "$W/fakerelease" ./scripts/e2e/fakerelease
	"$W/fakerelease" serve "127.0.0.1:$PORT" "$1" "$TAG" > "$W/fr.log" 2>&1 &
	FR=$!
	for _ in $(seq 1 50); do curl -sf "http://127.0.0.1:$PORT/techmuch/go-build-it/releases/latest/download/SHA256SUMS" >/dev/null && return; sleep 0.1; done
	fail "fake release server did not start"
}
export GBI_RELEASE_BASE="http://127.0.0.1:$PORT" GBI_PUBLIC_KEY="$PUB"

echo "1. a tampered release is refused"
serve "$W/bad"
set +e; sh scripts/install.sh --dir "$W/bin-bad" > "$W/bad.out" 2>&1; code=$?; set -e
[ $code -eq 2 ] || { cat "$W/bad.out"; fail "tampered: exit $code, want 2"; }
grep -q 'checksum mismatch' "$W/bad.out" || fail "tampered: no checksum message"
[ ! -e "$W/bin-bad/gbi" ] || fail "tampered: gbi was installed"

echo "2. a release signed by another key is refused (when OpenSSL can check)"
serve "$W/good"
if openssl pkeyutl -help 2>&1 | grep -q -- '-rawin'; then
	read -r OTHER _ < <(go run ./scripts/e2e/fakerelease keygen)
	set +e; GBI_PUBLIC_KEY="$OTHER" sh scripts/install.sh --dir "$W/bin-key" > "$W/key.out" 2>&1; code=$?; set -e
	[ $code -eq 2 ] || { cat "$W/key.out"; fail "wrong key: exit $code, want 2"; }
	grep -q 'bad signature' "$W/key.out" || fail "wrong key: no signature message"
	[ ! -e "$W/bin-key/gbi" ] || fail "wrong key: gbi was installed"
else
	echo "   (skipped: this OpenSSL cannot verify Ed25519)"
fi

echo "3. no release yet"
set +e; GBI_REPO=nobody/nothing sh scripts/install.sh --dir "$W/bin-none" > "$W/none.out" 2>&1; code=$?; set -e
[ $code -eq 3 ] || { cat "$W/none.out"; fail "no release: exit $code, want 3"; }
grep -q 'build it from source' "$W/none.out" || fail "no release: no build-from-source hint"

echo "4. a good release installs (latest, then a pinned version)"
sh scripts/install.sh --dir "$W/bin" > "$W/good.out" 2>&1 || { cat "$W/good.out"; fail "install"; }
tail -1 "$W/good.out" | grep -q "^gbi installed: $W/bin/gbi $TAG" || { cat "$W/good.out"; fail "last line"; }
[ "$("$W/bin/gbi" version)" = "$TAG" ] || fail "installed version"
sh scripts/install.sh --version "$TAG" --dir "$W/bin2" > "$W/pin.out" 2>&1 || { cat "$W/pin.out"; fail "pinned install"; }
set +e; sh scripts/install.sh --version 2026.01.01.9 --dir "$W/bin3" > "$W/wrong.out" 2>&1; code=$?; set -e
[ $code -eq 3 ] || fail "unknown version: exit $code, want 3"

echo "5. the installed gbi passes the doctor (isolated herdr, scripted harness, no server)"
export GBI_HOME="$W/home"
HOMEH="$W/h"
mkdir -p "$GBI_HOME"
cat > "$GBI_HOME/config.toml" <<EOF
listen = "127.0.0.1:7464"
[harnesses.fake]
enabled = true
binary = "$W/bin/gbi"
[harnesses.claude]
enabled = false
[harnesses.opencode]
enabled = false
[harnesses.antigravity]
enabled = false
[harnesses.pi]
enabled = false
[herdr]
session = "gbi"
config_home = "$HOMEH"
EOF
set +e; "$W/bin/gbi" doctor --json --tier harness --prompt > "$W/doctor.json"; code=$?; set -e
python3 - "$W/doctor.json" "$code" <<'PY' || fail "doctor"
import json, sys
r = json.load(open(sys.argv[1])); code = int(sys.argv[2])
bad = [x for x in r["results"] if x["status"] == "fail"]
for x in bad: print("  fail:", x["id"], x["detail"])
ran = {t["tier"]: t["ran"] for t in r["tiers"]}
assert code == r["exit"] == 0, ("exit", code, r["exit"], r["summary"])
assert all(ran.values()), ran
ids = {x["id"] for x in r["results"]}
for want in ("pane.roundtrip", "harness.on_pane_path", "harness.starts", "harness.answers"):
    assert want in ids, want
print("   doctor:", r["summary"], "·", ", ".join(f'{t["tier"]} {t["ok"]}/{t["total"]}' for t in r["tiers"]))
PY
echo "PASS: install.sh verifies checksum$(openssl pkeyutl -help 2>&1 | grep -q -- -rawin && echo ' and signature'), refuses tampering, reports a missing release, and the installed gbi passes every doctor tier"
