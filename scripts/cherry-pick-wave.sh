#!/usr/bin/env bash
set -euo pipefail

# cherry-pick-wave.sh
# Automates cherry-picking commits from worktree agent branches onto the current branch.
#
# Usage:
#   cherry-pick-wave.sh [--dry-run] <worktree-path> [worktree-path...]
#   cherry-pick-wave.sh --help
#
# Options:
#   --dry-run   Show what would be cherry-picked without making any changes
#   --help      Show this help message
#
# Exit codes:
#   0 = all cherry-picks succeeded (or dry-run completed)
#   1 = one or more cherry-picks failed or conflicts encountered

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# ── Globals ──────────────────────────────────────────────────────────────────
DRY_RUN=false
WORKTREE_PATHS=()

TOTAL_COMMITS=0
TOTAL_WORKTREES=0
FAILED_WORKTREES=()

# ── Helpers ───────────────────────────────────────────────────────────────────
usage() {
  cat <<'EOF'
Usage: cherry-pick-wave.sh [--dry-run] <worktree-path> [worktree-path...]

Cherry-pick commits from one or more worktree agent branches onto the current branch.

Options:
  --dry-run   Show what would be cherry-picked without making any changes
  --help      Show this help message

Arguments:
  worktree-path   Path to a git worktree directory (one or more)

Examples:
  # Cherry-pick from a single worktree
  cherry-pick-wave.sh .claude/worktrees/agent-abc123

  # Cherry-pick from multiple worktrees
  cherry-pick-wave.sh .claude/worktrees/agent-abc123 .claude/worktrees/agent-def456

  # Preview without applying
  cherry-pick-wave.sh --dry-run .claude/worktrees/agent-abc123

Exit codes:
  0 = all cherry-picks succeeded (or dry-run completed)
  1 = one or more cherry-picks failed or conflicts encountered
EOF
}

log_info()  { echo "[INFO]  $*"; }
log_warn()  { echo "[WARN]  $*" >&2; }
log_error() { echo "[ERROR] $*" >&2; }

# Resolve an absolute path, following symlinks where possible
resolve_path() {
  local p="$1"
  if command -v realpath &>/dev/null; then
    realpath -m "$p" 2>/dev/null || echo "$p"
  else
    cd "$p" 2>/dev/null && pwd || echo "$p"
  fi
}

# ── Argument Parsing ──────────────────────────────────────────────────────────
if [[ $# -eq 0 ]]; then
  usage
  exit 1
fi

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run)
      DRY_RUN=true
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    -*)
      log_error "Unknown option: $1"
      usage
      exit 1
      ;;
    *)
      WORKTREE_PATHS+=("$1")
      shift
      ;;
  esac
done

if [[ ${#WORKTREE_PATHS[@]} -eq 0 ]]; then
  log_error "No worktree paths provided."
  usage
  exit 1
fi

# ── Validate we are inside a git repo ─────────────────────────────────────────
if ! git -C "$REPO_ROOT" rev-parse --is-inside-work-tree &>/dev/null; then
  log_error "Not inside a git repository: $REPO_ROOT"
  exit 1
fi

CURRENT_BRANCH="$(git -C "$REPO_ROOT" rev-parse --abbrev-ref HEAD)"
log_info "Current branch: $CURRENT_BRANCH"

if [[ "$DRY_RUN" == "true" ]]; then
  log_info "Dry-run mode — no commits will be applied."
fi

# ── Per-worktree processing ───────────────────────────────────────────────────
process_worktree() {
  local wt_path="$1"
  local wt_abs

  wt_abs="$(resolve_path "$wt_path")"

  if [[ ! -d "$wt_abs" ]]; then
    log_error "Worktree path does not exist: $wt_abs"
    FAILED_WORKTREES+=("$wt_abs (path not found)")
    return 1
  fi

  # Verify it is a registered git worktree
  local wt_branch
  wt_branch="$(git -C "$REPO_ROOT" worktree list --porcelain \
    | awk -v wp="$wt_abs" '
        /^worktree / { cur=$2 }
        /^branch /   { if (cur == wp) { sub(/^refs\/heads\//, "", $2); print $2; exit } }
      ')"

  if [[ -z "$wt_branch" ]]; then
    # Fallback: ask the worktree itself
    if git -C "$wt_abs" rev-parse --is-inside-work-tree &>/dev/null 2>&1; then
      wt_branch="$(git -C "$wt_abs" rev-parse --abbrev-ref HEAD 2>/dev/null || true)"
    fi
  fi

  if [[ -z "$wt_branch" ]]; then
    log_error "Could not determine branch for worktree: $wt_abs"
    FAILED_WORKTREES+=("$wt_abs (branch unknown)")
    return 1
  fi

  log_info "Worktree: $wt_abs  ->  branch: $wt_branch"

  # Find the merge-base between the worktree branch and the current branch
  local merge_base
  if ! merge_base="$(git -C "$REPO_ROOT" merge-base "$CURRENT_BRANCH" "$wt_branch" 2>/dev/null)"; then
    log_error "Cannot find merge-base between '$CURRENT_BRANCH' and '$wt_branch' for worktree: $wt_abs"
    FAILED_WORKTREES+=("$wt_abs (no merge-base)")
    return 1
  fi

  # Commits on the worktree branch that are NOT on the current branch
  local commits
  commits="$(git -C "$REPO_ROOT" log --oneline --reverse \
    "${merge_base}..${wt_branch}" 2>/dev/null || true)"

  if [[ -z "$commits" ]]; then
    log_info "  No new commits on $wt_branch since branch point — skipping."
    return 0
  fi

  local commit_count
  commit_count="$(echo "$commits" | wc -l | tr -d ' ')"
  log_info "  Found $commit_count commit(s) to cherry-pick:"

  while IFS= read -r line; do
    log_info "    $line"
  done <<< "$commits"

  if [[ "$DRY_RUN" == "true" ]]; then
    TOTAL_COMMITS=$(( TOTAL_COMMITS + commit_count ))
    TOTAL_WORKTREES=$(( TOTAL_WORKTREES + 1 ))
    return 0
  fi

  # Apply commits one by one for precise conflict reporting
  local sha subject
  while IFS= read -r line; do
    sha="${line%% *}"
    subject="${line#* }"

    log_info "  Applying: $sha $subject"

    if ! git -C "$REPO_ROOT" cherry-pick "$sha" 2>&1; then
      log_error "  Conflict on commit $sha ('$subject') from worktree: $wt_abs"
      log_info  "  Aborting cherry-pick for this worktree."
      git -C "$REPO_ROOT" cherry-pick --abort 2>/dev/null || true
      FAILED_WORKTREES+=("$wt_abs (conflict at $sha)")
      return 1
    fi
  done <<< "$commits"

  log_info "  Successfully applied $commit_count commit(s) from $wt_branch."
  TOTAL_COMMITS=$(( TOTAL_COMMITS + commit_count ))
  TOTAL_WORKTREES=$(( TOTAL_WORKTREES + 1 ))
}

# ── Main Loop ─────────────────────────────────────────────────────────────────
for wt in "${WORKTREE_PATHS[@]}"; do
  process_worktree "$wt" || true  # collect failures, keep going
done

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo "─────────────────────────────────────────────"
if [[ "$DRY_RUN" == "true" ]]; then
  echo "DRY RUN complete: would cherry-pick $TOTAL_COMMITS commits from $TOTAL_WORKTREES worktree(s)."
else
  echo "Cherry-picked $TOTAL_COMMITS commits from $TOTAL_WORKTREES worktree(s)."
fi

if [[ ${#FAILED_WORKTREES[@]} -gt 0 ]]; then
  echo "Failed worktrees (${#FAILED_WORKTREES[@]}):"
  for fw in "${FAILED_WORKTREES[@]}"; do
    echo "  - $fw"
  done
  echo "─────────────────────────────────────────────"
  exit 1
fi

echo "─────────────────────────────────────────────"
exit 0
