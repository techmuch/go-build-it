#!/usr/bin/env bash
# Weekly upstream review (NFR-30, NFR-31; docs/upstream/README.md).
# Compares herdr and Ovis with the revisions last reviewed, checks herdr's latest API
# schema against the vendored one, and opens (or comments on) a GitHub issue when
# something affects go-build-it.  DRY_RUN=1 prints the report instead.
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
work="$(mktemp -d)"; trap 'rm -rf "$work"' EXIT
rev_of() { sed -n "s/^| $1 | \`\([0-9a-f]*\)\`.*/\1/p" "$root/docs/upstream/README.md" | head -1; }
herdr_rev="$(rev_of herdr)"; ovis_rev="$(rev_of Ovis)"
report="$work/report.md"; changed=0
echo "## Upstream review $(date -u +%Y-%m-%d)" > "$report"

review() { # name repo rev paths...
  local name="$1" repo="$2" rev="$3"; shift 3
  git clone -q --filter=blob:none "https://github.com/$repo" "$work/$name"
  local head; head="$(git -C "$work/$name" rev-parse --short HEAD)"
  local log; log="$(git -C "$work/$name" log --oneline "$rev..HEAD" -- "$@" 2>/dev/null | head -40 || true)"
  echo "" >> "$report"; echo "### $name: \`$rev\` → \`$head\`" >> "$report"
  if [ -n "$log" ]; then
    changed=1
    echo "Commits touching what go-build-it uses (\`$*\`):" >> "$report"
    echo '```' >> "$report"; echo "$log" >> "$report"; echo '```' >> "$report"
  else
    echo "No changes in the paths go-build-it relies on." >> "$report"
  fi
}
review herdr herdrdev/herdr "$herdr_rev" src/app/api src/protocol src/api docs/next/website/src/content/docs/socket-api.mdx docs/next/website/src/content/docs/cli-reference.mdx CHANGELOG.md
review Ovis devswha/ovis "$ovis_rev" server/herdr server/conversation.ts server/claude-transcript.ts server/pi.ts server/agy-store.ts server/mirror.ts

# herdr's latest API schema against the vendored one: methods go-build-it needs, and generated types.
if command -v herdr >/dev/null 2>&1; then
  herdr api schema --json > "$work/latest.schema.json"
  cp "$root/internal/herdr/schema/herdr-api.schema.json" "$work/vendored.json"
  cp "$work/latest.schema.json" "$root/internal/herdr/schema/herdr-api.schema.json"
  echo "" >> "$report"; echo "### herdr API schema ($(herdr --version))" >> "$report"
  if (cd "$root" && go run ./internal/herdr/gen -check) >/dev/null 2>"$work/gen.err"; then
    echo "Generated types unchanged." >> "$report"
  else
    changed=1
    echo "The generated types differ or a required method is gone; regenerate with \`go run ./internal/herdr/gen\` and review:" >> "$report"
    echo '```' >> "$report"; head -20 "$work/gen.err" >> "$report"; echo '```' >> "$report"
  fi
  cp "$work/vendored.json" "$root/internal/herdr/schema/herdr-api.schema.json"
fi
echo "" >> "$report"; echo "Follow docs/upstream/README.md: record findings in docs/upstream/herdr.md and ovis.md and update the reviewed revisions." >> "$report"

if [ "${DRY_RUN:-}" = 1 ]; then cat "$report"; exit 0; fi
[ "$changed" = 1 ] || { echo "no relevant upstream changes"; exit 0; }
existing="$(gh issue list --label upstream-review --state open --json number --jq '.[0].number' 2>/dev/null || true)"
if [ -n "$existing" ]; then
  gh issue comment "$existing" --body-file "$report"
else
  gh label create upstream-review --color 5319e7 --description "herdr / Ovis upstream changes to review" 2>/dev/null || true
  gh issue create --title "Upstream review: herdr / Ovis changes to check" --label upstream-review --body-file "$report"
fi
