#!/usr/bin/env bash
#
# Copy "Claude-style" skill folders (directories containing SKILL.md)
# into a Codex skills directory (default: ~/.codex/skills).
#
# Safe by default:
# - Creates a timestamped backup of any destination skill it overwrites
# - Supports --dry-run to preview changes
#
# Usage:
#   ./scripts/export-claude-skills-to-codex.sh \
#     --src ./skills \
#     --dst "$HOME/.codex/skills" \
#     --dry-run
#
set -euo pipefail

usage() {
  cat <<'EOF'
export-claude-skills-to-codex.sh

Copies skill directories (each containing SKILL.md) from --src into --dst.

Options:
  --src <dir>         Source directory containing skill folders (default: ./skills if present, else ./.agents/skills)
  --dst <dir>         Destination Codex skills directory (default: ~/.codex/skills)
  --backup <dir>      Backup directory (default: ~/.codex/skills.backup.<timestamp>)
  --dry-run           Show what would change (no writes)
  --only <a,b,c>      Only copy these skill folder names (comma-separated)
  --help              Show this help

Examples:
  ./scripts/export-claude-skills-to-codex.sh --dry-run
  ./scripts/export-claude-skills-to-codex.sh --src ./skills --dst ~/.codex/skills
  ./scripts/export-claude-skills-to-codex.sh --only research,vibe --dry-run
EOF
}

SRC=""
DST=""
BACKUP=""
DRY_RUN="false"
ONLY_CSV=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --src)
      SRC="${2:-}"
      shift 2
      ;;
    --dst)
      DST="${2:-}"
      shift 2
      ;;
    --backup)
      BACKUP="${2:-}"
      shift 2
      ;;
    --dry-run)
      DRY_RUN="true"
      shift 1
      ;;
    --only)
      ONLY_CSV="${2:-}"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "Unknown arg: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if ! command -v rsync >/dev/null 2>&1; then
  echo "Error: rsync not found. Install rsync and re-run." >&2
  exit 1
fi

if [[ -z "$SRC" ]]; then
  if [[ -d "skills" ]]; then
    SRC="skills"
  elif [[ -d ".agents/skills" ]]; then
    SRC=".agents/skills"
  else
    echo "Error: cannot infer --src (no ./skills or ./.agents/skills)." >&2
    exit 1
  fi
fi

if [[ -z "$DST" ]]; then
  DST="$HOME/.codex/skills"
fi

timestamp="$(date +%Y%m%d-%H%M%S)"
if [[ -z "$BACKUP" ]]; then
  BACKUP="$HOME/.codex/skills.backup.${timestamp}"
fi

if [[ ! -d "$SRC" ]]; then
  echo "Error: --src does not exist: $SRC" >&2
  exit 1
fi

mkdir -p "$DST"
if [[ "$DRY_RUN" != "true" ]]; then
  mkdir -p "$BACKUP"
fi

declare -A ONLY
if [[ -n "$ONLY_CSV" ]]; then
  IFS=',' read -r -a only_arr <<<"$ONLY_CSV"
  for name in "${only_arr[@]}"; do
    name="$(echo "$name" | xargs)"
    [[ -n "$name" ]] && ONLY["$name"]=1
  done
fi

copied=0
skipped=0

echo "Source: $SRC"
echo "Dest:   $DST"
echo "Backup: $BACKUP"
echo "DryRun: $DRY_RUN"
echo ""

shopt -s nullglob
for skill_dir in "$SRC"/*/; do
  skill_name="$(basename "$skill_dir")"

  if [[ -n "$ONLY_CSV" ]] && [[ -z "${ONLY[$skill_name]:-}" ]]; then
    skipped=$((skipped + 1))
    continue
  fi

  if [[ ! -f "${skill_dir}SKILL.md" ]]; then
    skipped=$((skipped + 1))
    continue
  fi

  dst_skill="${DST%/}/${skill_name}"

  # Backup existing dest skill before overwriting
  if [[ -d "$dst_skill" ]] && [[ "$DRY_RUN" != "true" ]]; then
    rsync -a --delete "${dst_skill%/}/" "${BACKUP%/}/${skill_name%/}/"
  fi

  # Copy skill (mirror, no symlinks)
  rsync_args=(-a --delete --copy-links)
  if [[ "$DRY_RUN" == "true" ]]; then
    rsync_args+=(--dry-run)
  fi

  rsync "${rsync_args[@]}" "${skill_dir%/}/" "${dst_skill%/}/" >/dev/null
  copied=$((copied + 1))
done

echo "Skills copied: $copied"
echo "Skills skipped: $skipped"
if [[ "$DRY_RUN" != "true" ]]; then
  echo "Backups written to: $BACKUP"
fi
