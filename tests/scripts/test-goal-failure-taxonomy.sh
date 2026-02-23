#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCRIPT="$ROOT/scripts/goal-failure-taxonomy.sh"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

INPUT="$TMP_DIR/fitness.json"
cat > "$INPUT" <<'JSON'
{
  "goals": [
    {"goal_id":"release-security-gate","result":"fail","description":"release gate missing"},
    {"goal_id":"go-cli-builds","result":"fail","description":"build fails"},
    {"goal_id":"go-coverage-floor","result":"fail","description":"coverage low"},
    {"goal_id":"goal-quality","result":"fail","description":"bad goal quality"},
    {"goal_id":"custom-unknown-check","result":"fail","description":"unknown class"},
    {"goal_id":"go-vet-clean","result":"pass","description":"passes"}
  ]
}
JSON

OUT="$TMP_DIR/out.json"
bash "$SCRIPT" "$INPUT" > "$OUT"
jq -e . "$OUT" >/dev/null

[[ "$(jq -r '.summary.total_failing' "$OUT")" == "5" ]]
[[ "$(jq -r '.summary.by_category.security' "$OUT")" == "1" ]]
[[ "$(jq -r '.summary.by_category.reliability' "$OUT")" == "1" ]]
[[ "$(jq -r '.summary.by_category.quality' "$OUT")" == "1" ]]
[[ "$(jq -r '.summary.by_category.governance' "$OUT")" == "1" ]]
[[ "$(jq -r '.summary.by_category.other' "$OUT")" == "1" ]]

echo "PASS: goal failure taxonomy classification"
