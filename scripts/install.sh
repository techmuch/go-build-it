#!/bin/sh
# Install gbi (go-build-it) on macOS or Linux from a GitHub release (FR-02b, ADR-012).
#
#   sh install.sh [--version YYYY.MM.DD.N] [--dir ~/.local/bin]
#
# It downloads the release binary for this machine with SHA256SUMS and SHA256SUMS.sig. Then:
#   - It refuses a download whose SHA-256 does not match SHA256SUMS.
#   - It checks the Ed25519 signature of SHA256SUMS with OpenSSL when OpenSSL supports it,
#     and refuses on a bad signature. When OpenSSL can't check it, it says so.
#   - It installs gbi into the folder without sudo, and checks that the installed binary runs.
# The last line is "gbi installed: <path> <version>", so an AI agent can read the result.
#
# Exit codes: 0 installed, 1 error, 2 verification failed (nothing installed), 3 no release yet.
#
# For tests: GBI_RELEASE_BASE replaces https://github.com (scripts/install_e2e.sh), and
# GBI_PUBLIC_KEY replaces the release key below.
set -eu

REPO="${GBI_REPO:-techmuch/go-build-it}"
BASE="${GBI_RELEASE_BASE:-https://github.com}"
# The release signing public key. It must match update.PublicKey, README.md and install.ps1;
# a test checks that all four agree.
PUBLIC_KEY="${GBI_PUBLIC_KEY:-eJ3eKk81oKaKZlNaLjafw1j1GO+Uueyt51jNMCchih4=}"
VERSION="latest"
DIR="${HOME}/.local/bin"

while [ $# -gt 0 ]; do
	case "$1" in
	--version) VERSION="$2"; shift 2 ;;
	--dir) DIR="$2"; shift 2 ;;
	-h|--help) sed -n '2,16p' "$0"; exit 0 ;;
	*) echo "unknown option: $1" >&2; exit 1 ;;
	esac
done

say() { printf '%s\n' "$*" >&2; }
die() { say "error: $1"; exit "${2:-1}"; }

case "$(uname -s)" in
Darwin) os=darwin ;;
Linux) os=linux ;;
*) die "unsupported system $(uname -s); on Windows use scripts/install.ps1" ;;
esac
case "$(uname -m)" in
x86_64|amd64) arch=amd64 ;;
arm64|aarch64) arch=arm64 ;;
*) die "unsupported CPU $(uname -m)" ;;
esac
asset="gbi_${os}_${arch}"

if [ "$VERSION" = latest ]; then
	url="$BASE/$REPO/releases/latest/download"
else
	url="$BASE/$REPO/releases/download/$VERSION"
fi

tmp=$(mktemp -d 2>/dev/null || mktemp -d -t gbi)
trap 'rm -rf "$tmp"' EXIT INT TERM

fetch() { # url file → 0, or 22 when the server says not found
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL --retry 2 -o "$2" "$1"
	elif command -v wget >/dev/null 2>&1; then
		wget -q -O "$2" "$1"
	else
		die "neither curl nor wget is installed"
	fi
}

say "Downloading $asset ($VERSION) from $BASE/$REPO"
if ! fetch "$url/SHA256SUMS" "$tmp/SHA256SUMS"; then
	say "No release found at $url."
	say "If go-build-it has not published a release yet, build it from source instead:"
	say "  git clone https://github.com/$REPO && cd go-build-it && make build && ./bin/gbi version"
	exit 3
fi
fetch "$url/SHA256SUMS.sig" "$tmp/SHA256SUMS.sig" || die "the release has no SHA256SUMS.sig" 2
fetch "$url/$asset" "$tmp/$asset" || die "the release has no $asset for this machine"

# 1. Checksum.
want=$(awk -v n="$asset" '{ f=$2; sub(/^\*/, "", f); if (f == n) print $1 }' "$tmp/SHA256SUMS")
[ -n "$want" ] || die "$asset is not listed in SHA256SUMS" 2
if command -v sha256sum >/dev/null 2>&1; then
	got=$(sha256sum "$tmp/$asset" | awk '{print $1}')
else
	got=$(shasum -a 256 "$tmp/$asset" | awk '{print $1}')
fi
[ "$got" = "$want" ] || die "checksum mismatch: the download does not match SHA256SUMS; nothing was installed" 2
say "Checksum OK (SHA-256 $got)"

# 2. Signature: Ed25519 over SHA256SUMS. The key is wrapped in its DER prefix for OpenSSL.
sig_checked=no
if command -v openssl >/dev/null 2>&1 && openssl pkeyutl -help 2>&1 | grep -q -- '-rawin'; then
	{ printf '\060\052\060\005\006\003\053\145\160\003\041\000'; printf '%s' "$PUBLIC_KEY" | openssl base64 -d -A; } > "$tmp/key.der"
	tr -d '\n\r ' < "$tmp/SHA256SUMS.sig" | openssl base64 -d -A > "$tmp/sig.bin"
	if openssl pkey -pubin -inform DER -in "$tmp/key.der" -out "$tmp/key.pem" 2>/dev/null &&
		openssl pkeyutl -verify -pubin -inkey "$tmp/key.pem" -rawin -in "$tmp/SHA256SUMS" -sigfile "$tmp/sig.bin" >/dev/null 2>&1; then
		sig_checked=yes
		say "Signature OK (Ed25519, release key)"
	else
		die "bad signature: SHA256SUMS was not signed by the go-build-it release key; nothing was installed" 2
	fi
else
	say "Signature NOT checked: this OpenSSL cannot verify Ed25519 (install OpenSSL 3 to check it). The checksum was checked; gbi checks signatures itself for every later update."
fi

# 3. Install and check that it runs.
mkdir -p "$DIR"
cp "$tmp/$asset" "$DIR/gbi.new"
chmod 755 "$DIR/gbi.new"
ver=$("$DIR/gbi.new" version 2>/dev/null) || { rm -f "$DIR/gbi.new"; die "the downloaded gbi does not run on this machine"; }
if [ "$VERSION" != latest ] && [ "${ver#v}" != "${VERSION#v}" ]; then
	rm -f "$DIR/gbi.new"
	die "the downloaded gbi reports version $ver, expected $VERSION" 2
fi
mv -f "$DIR/gbi.new" "$DIR/gbi"

case ":$PATH:" in
*":$DIR:"*) ;;
*) say "Note: $DIR is not on your PATH. Add it, for example: echo 'export PATH=\"$DIR:\$PATH\"' >> ~/.profile" ;;
esac
say "Next: gbi doctor --json --fix   (see README: Install with an AI agent)"
echo "gbi installed: $DIR/gbi $ver (signature checked: $sig_checked)"
