#!/usr/bin/env bash
set -euo pipefail

# check-contract-compatibility.sh
# Fails when contract schemas/examples referenced by docs are missing.

ROOT="${1:-.}"

required=(
  "docs/ol-bridge-contracts.md"
  "docs/contracts/memrl-policy-integration.md"
  "docs/contracts/memrl-policy.schema.json"
  "docs/contracts/memrl-policy.profile.example.json"
)

missing=0
for f in "${required[@]}"; do
  if [[ ! -f "$ROOT/$f" ]]; then
    echo "Missing required contract artifact: $f"
    missing=1
  fi
done

if ! grep -q "memrl-policy.schema.json" "$ROOT/docs/INDEX.md"; then
  echo "docs/INDEX.md missing memrl-policy schema reference"
  missing=1
fi

if [[ "$missing" -ne 0 ]]; then
  echo "Contract compatibility check failed."
  exit 1
fi

echo "Contract compatibility check passed."
