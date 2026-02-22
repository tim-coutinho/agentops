#!/usr/bin/env bash
set -euo pipefail

# validate-go-fast.sh
# Lightweight local Go gate: run race-enabled tests only for changed packages.
#
# Exit codes:
#   0 - pass or no Go changes
#   1 - failures / setup issues

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

if ! command -v go >/dev/null 2>&1; then
    echo "SKIP: go not installed"
    exit 0
fi

collect_target_files() {
    # Best pre-push scope: commits ahead of upstream.
    if git rev-parse --git-dir >/dev/null 2>&1; then
        if git rev-parse --abbrev-ref --symbolic-full-name '@{upstream}' >/dev/null 2>&1; then
            ahead_files="$(git diff --name-only '@{upstream}...HEAD' 2>/dev/null || true)"
            if [[ -n "$ahead_files" ]]; then
                printf '%s\n' "$ahead_files"
                return 0
            fi
        fi
        # Fallbacks for detached or no upstream.
        git diff --name-only --cached 2>/dev/null || true
        git diff --name-only 2>/dev/null || true
        git show --name-only --pretty=format: HEAD 2>/dev/null || true
    fi
}

find_module_root() {
    local path="$1"
    local dir="$path"
    if [[ ! -d "$dir" ]]; then
        dir="$(dirname "$dir")"
    fi
    dir="$(cd "$dir" 2>/dev/null && pwd -P 2>/dev/null || true)"
    while [[ -n "$dir" && "$dir" != "/" ]]; do
        if [[ -f "$dir/go.mod" ]]; then
            printf '%s\n' "$dir"
            return 0
        fi
        dir="$(dirname "$dir")"
    done
    return 1
}

tmp_files="$(mktemp)"
tmp_pairs="$(mktemp)"
trap 'rm -f "$tmp_files" "$tmp_pairs"' EXIT

collect_target_files | sed '/^[[:space:]]*$/d' | sort -u > "$tmp_files"

if [[ ! -s "$tmp_files" ]]; then
    echo "SKIP: no changed files detected"
    exit 0
fi

go_changed=0

while IFS= read -r file; do
    [[ -z "$file" ]] && continue
    abs_path="$REPO_ROOT/$file"

    case "$file" in
        *.go|go.mod|go.sum)
            go_changed=1
            ;;
        *)
            continue
            ;;
    esac

    module_root="$(find_module_root "$abs_path" || true)"
    [[ -z "$module_root" ]] && continue

    # Dependency changes are module-wide.
    if [[ "$file" == "go.mod" || "$file" == "go.sum" || "$file" == */go.mod || "$file" == */go.sum ]]; then
        printf '%s\t%s\n' "$module_root" "./..." >> "$tmp_pairs"
        continue
    fi

    # For changed go files, test only their package directory.
    dir_path="$(dirname "$abs_path")"
    dir_path="$(cd "$dir_path" 2>/dev/null && pwd -P 2>/dev/null || true)"
    [[ -z "$dir_path" ]] && continue

    rel="${dir_path#"$module_root"/}"
    if [[ "$dir_path" == "$module_root" ]]; then
        rel="."
    fi

    if [[ "$rel" == "." ]]; then
        printf '%s\t%s\n' "$module_root" "." >> "$tmp_pairs"
    else
        printf '%s\t%s\n' "$module_root" "./$rel" >> "$tmp_pairs"
    fi
done < "$tmp_files"

if [[ "$go_changed" -eq 0 ]]; then
    echo "SKIP: no Go changes in push scope"
    exit 0
fi

if [[ ! -s "$tmp_pairs" ]]; then
    echo "SKIP: Go changes detected but no resolvable module/package paths"
    exit 0
fi

tmp_unique="$(mktemp)"
trap 'rm -f "$tmp_files" "$tmp_pairs" "$tmp_unique"' EXIT
sort -u "$tmp_pairs" > "$tmp_unique"

echo "Running lightweight Go race checks on changed scope..."

while IFS=$'\t' read -r module_root _; do
    [[ -z "$module_root" ]] && continue

    patterns=()
    while IFS= read -r pattern; do
        [[ -z "$pattern" ]] && continue
        patterns+=("$pattern")
    done < <(awk -F '\t' -v m="$module_root" '$1 == m {print $2}' "$tmp_unique" | sort -u)
    if [[ "${#patterns[@]}" -eq 0 ]]; then
        continue
    fi

    echo ""
    echo "module: ${module_root#"$REPO_ROOT"/}"
    echo "packages: ${patterns[*]}"
    (
        cd "$module_root"
        go test -race -count=1 "${patterns[@]}"
    )
done < <(cut -f1 "$tmp_unique" | sort -u)

echo ""
echo "PASS: lightweight Go race checks succeeded"
