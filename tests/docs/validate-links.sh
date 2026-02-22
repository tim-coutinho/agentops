#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ALLOWLIST="$REPO_ROOT/tests/docs/broken-links-allowlist.txt"

total=0
broken=0
allowlisted=0

# Load allowlist into associative array
declare -A allowed
if [[ -f "$ALLOWLIST" ]]; then
  while IFS= read -r line; do
    [[ -z "$line" || "$line" == \#* ]] && continue
    allowed["$line"]=1
  done < "$ALLOWLIST"
fi

# Find all markdown files in specified directories
md_files=()
while IFS= read -r f; do
  md_files+=("$f")
done < <(find "$REPO_ROOT" -maxdepth 1 -name '*.md' -type f)

for dir in docs skills cli; do
  if [[ -d "$REPO_ROOT/$dir" ]]; then
    while IFS= read -r f; do
      md_files+=("$f")
    done < <(find "$REPO_ROOT/$dir" -name '*.md' -type f)
  fi
done

for file in "${md_files[@]}"; do
  rel_file="${file#"$REPO_ROOT"/}"
  file_dir="$(dirname "$file")"

  # Extract all links with line numbers in one pass using grep -n
  while IFS= read -r match; do
    [[ -z "$match" ]] && continue
    line_num="${match%%:*}"
    target="${match#*:}"

    # Skip external URLs
    [[ "$target" == http://* || "$target" == https://* ]] && continue
    # Skip anchor-only links
    [[ "$target" == \#* ]] && continue
    # Skip mailto links
    [[ "$target" == mailto:* ]] && continue
    # Skip empty
    [[ -z "$target" ]] && continue

    # Strip anchor fragment
    target_path="${target%%#*}"
    [[ -z "$target_path" ]] && continue

    # Strip trailing whitespace and quotes (image title syntax)
    target_path="${target_path%% *}"

    total=$((total + 1))

    # Resolve path relative to the linking file's directory
    if [[ "$target_path" == /* ]]; then
      resolved="$target_path"
    else
      resolved="$file_dir/$target_path"
    fi

    if [[ ! -e "$resolved" ]]; then
      allowlist_key="$rel_file:$target_path"
      if [[ -n "${allowed[$allowlist_key]+x}" ]]; then
        allowlisted=$((allowlisted + 1))
      else
        broken=$((broken + 1))
        echo "BROKEN: $rel_file:$line_num -> $target_path"
      fi
    fi
  done < <(grep -noE '\]\([^)]+\)' "$file" 2>/dev/null | sed 's/:\]/:/;s/)$//' | sed 's/^\([0-9]*\):\(.*\)$/\1:\2/' | sed 's/^\([0-9]*\):(/\1:/')
done

echo ""
echo "$total links checked, $broken broken ($allowlisted allowlisted)"

if [[ "$broken" -gt 0 ]]; then
  exit 1
fi

exit 0
