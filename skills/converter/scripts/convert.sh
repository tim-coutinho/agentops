#!/usr/bin/env bash
# convert.sh — Cross-platform skill converter pipeline
# Usage: bash skills/converter/scripts/convert.sh <skill-dir> <target> [output-dir]
#        bash skills/converter/scripts/convert.sh --all <target> [output-dir]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# ─── Helpers ───────────────────────────────────────────────────────────

die() { echo "ERROR: $*" >&2; exit 1; }

usage() {
  cat <<'EOF'
Usage:
  bash skills/converter/scripts/convert.sh <skill-dir> <target> [output-dir]
  bash skills/converter/scripts/convert.sh --all <target> [output-dir]

Targets: codex, cursor, test

Examples:
  bash skills/converter/scripts/convert.sh skills/council codex
  bash skills/converter/scripts/convert.sh --all codex
  bash skills/converter/scripts/convert.sh skills/vibe test /tmp/out
EOF
  exit 1
}

# ─── Stage 1: Parse ───────────────────────────────────────────────────

# Parse SKILL.md frontmatter and body.
# Sets: BUNDLE_NAME, BUNDLE_DESC, BUNDLE_BODY, BUNDLE_FRONTMATTER
parse_skill_md() {
  local skill_md="$1"
  [[ -f "$skill_md" ]] || die "SKILL.md not found: $skill_md"

  local content
  content="$(<"$skill_md")"

  # Extract frontmatter (between first and second --- lines)
  local in_fm=0
  local fm_lines=()
  local body_lines=()
  local fm_ended=0
  local line_num=0

  while IFS= read -r line; do
    line_num=$((line_num + 1))
    if [[ $line_num -eq 1 && "$line" == "---" ]]; then
      in_fm=1
      continue
    fi
    if [[ $in_fm -eq 1 && "$line" == "---" ]]; then
      in_fm=0
      fm_ended=1
      continue
    fi
    if [[ $in_fm -eq 1 ]]; then
      fm_lines+=("$line")
    elif [[ $fm_ended -eq 1 ]]; then
      body_lines+=("$line")
    fi
  done <<< "$content"

  BUNDLE_FRONTMATTER="$(printf '%s\n' "${fm_lines[@]}")"

  # Extract name and description from frontmatter
  BUNDLE_NAME="$(echo "$BUNDLE_FRONTMATTER" | sed -n 's/^name: *//p' | tr -d "'" | tr -d '"')"
  BUNDLE_DESC="$(echo "$BUNDLE_FRONTMATTER" | sed -n 's/^description: *//p' | sed "s/^'//;s/'$//")"

  # Body: join with newlines
  BUNDLE_BODY="$(printf '%s\n' "${body_lines[@]}")"
}

# Collect files from a subdirectory into parallel arrays.
# Args: <dir> <array-name-names> <array-name-contents>
collect_files() {
  local dir="$1"
  local -n names_arr="$2"
  local -n contents_arr="$3"
  names_arr=()
  contents_arr=()

  if [[ -d "$dir" ]]; then
    local f
    for f in "$dir"/*; do
      [[ -f "$f" ]] || continue
      names_arr+=("$(basename "$f")")
      contents_arr+=("$(<"$f")")
    done
  fi
}

# Full parse: populate all BUNDLE_* variables and REF/SCRIPT arrays
parse_bundle() {
  local skill_dir="$1"
  parse_skill_md "$skill_dir/SKILL.md"
  collect_files "$skill_dir/references" REF_NAMES REF_CONTENTS
  collect_files "$skill_dir/scripts" SCRIPT_NAMES SCRIPT_CONTENTS
}

# ─── Stage 2: Convert ─────────────────────────────────────────────────

# Test target: emit SkillBundle as structured markdown
convert_test() {
  local out=""
  out+="# SkillBundle: ${BUNDLE_NAME}"$'\n\n'
  out+="## Name"$'\n\n'
  out+="${BUNDLE_NAME}"$'\n\n'
  out+="## Description"$'\n\n'
  out+="${BUNDLE_DESC}"$'\n\n'
  out+="## Frontmatter"$'\n\n'
  out+='```yaml'$'\n'
  out+="${BUNDLE_FRONTMATTER}"$'\n'
  out+='```'$'\n\n'
  out+="## Body"$'\n\n'
  out+="${BUNDLE_BODY}"$'\n\n'

  out+="## References (${#REF_NAMES[@]})"$'\n\n'
  local i
  for i in "${!REF_NAMES[@]}"; do
    out+="### ${REF_NAMES[$i]}"$'\n\n'
    out+='```'$'\n'
    out+="${REF_CONTENTS[$i]}"$'\n'
    out+='```'$'\n\n'
  done

  out+="## Scripts (${#SCRIPT_NAMES[@]})"$'\n\n'
  for i in "${!SCRIPT_NAMES[@]}"; do
    out+="### ${SCRIPT_NAMES[$i]}"$'\n\n'
    out+='```'$'\n'
    out+="${SCRIPT_CONTENTS[$i]}"$'\n'
    out+='```'$'\n\n'
  done

  CONVERTED_OUTPUT="$out"
  CONVERTED_FILENAME="bundle.md"
}

# Codex target: SKILL.md + prompt.md
# Codex skills live at ~/.codex/skills/<name>/SKILL.md
# Codex prompts live at ~/.codex/prompts/<name>.md
# Description max 1024 chars, no hooks support, tool names pass through
convert_codex() {
  local desc="$BUNDLE_DESC"

  # Truncate description to 1024 chars at word boundary
  if [[ ${#desc} -gt 1024 ]]; then
    desc="${desc:0:1021}"
    # Trim to last word boundary (space)
    desc="${desc% *}..."
  fi

  # ── Build SKILL.md ──
  local skill_md=""
  skill_md+="# ${BUNDLE_NAME}"$'\n\n'
  skill_md+="${desc}"$'\n\n'
  skill_md+="${BUNDLE_BODY}"$'\n'

  # Inline references as appended sections
  if [[ ${#REF_NAMES[@]} -gt 0 ]]; then
    skill_md+=$'\n'"---"$'\n\n'
    skill_md+="## References"$'\n\n'
    local i
    for i in "${!REF_NAMES[@]}"; do
      skill_md+="### ${REF_NAMES[$i]}"$'\n\n'
      skill_md+="${REF_CONTENTS[$i]}"$'\n\n'
    done
  fi

  # Inline scripts as code blocks
  if [[ ${#SCRIPT_NAMES[@]} -gt 0 ]]; then
    skill_md+=$'\n'"---"$'\n\n'
    skill_md+="## Scripts"$'\n\n'
    local i
    for i in "${!SCRIPT_NAMES[@]}"; do
      # Detect language from extension
      local ext="${SCRIPT_NAMES[$i]##*.}"
      local lang=""
      case "$ext" in
        sh|bash) lang="bash" ;;
        py)      lang="python" ;;
        js)      lang="javascript" ;;
        ts)      lang="typescript" ;;
        *)       lang="$ext" ;;
      esac
      skill_md+="### ${SCRIPT_NAMES[$i]}"$'\n\n'
      skill_md+="\`\`\`${lang}"$'\n'
      skill_md+="${SCRIPT_CONTENTS[$i]}"$'\n'
      skill_md+="\`\`\`"$'\n\n'
    done
  fi

  # ── Build prompt.md ──
  local prompt_md=""
  prompt_md+="# ${BUNDLE_NAME}"$'\n\n'
  prompt_md+="${desc}"$'\n\n'
  prompt_md+="## Instructions"$'\n\n'
  prompt_md+="Load and follow the skill instructions from \`~/.codex/skills/${BUNDLE_NAME}/SKILL.md\`."$'\n'

  # Set primary output (SKILL.md)
  CONVERTED_OUTPUT="$skill_md"
  CONVERTED_FILENAME="SKILL.md"

  # Set secondary output (prompt.md)
  CONVERTED_OUTPUT_2="$prompt_md"
  CONVERTED_FILENAME_2="prompt.md"
}

# Cursor target: .mdc rule file with YAML frontmatter + optional mcp.json
# Cursor rules format: .cursor/rules/<name>.mdc (Cursor 0.40+)
# Max output size: 100KB (102400 bytes). References are budget-fitted.
CURSOR_MAX_BYTES=102400

convert_cursor() {
  local out=""

  # ── YAML frontmatter ──
  out+="---"$'\n'
  out+="description: ${BUNDLE_DESC}"$'\n'
  out+="globs: "$'\n'
  out+="alwaysApply: false"$'\n'
  out+="---"$'\n\n'

  # ── Body content ──
  out+="${BUNDLE_BODY}"$'\n'

  # ── Scripts as code blocks (included before references — smaller, higher value) ──
  if [[ ${#SCRIPT_NAMES[@]} -gt 0 ]]; then
    out+=$'\n'"## Scripts"$'\n\n'
    local i
    for i in "${!SCRIPT_NAMES[@]}"; do
      local ext="${SCRIPT_NAMES[$i]##*.}"
      local lang=""
      case "$ext" in
        sh|bash) lang="bash" ;;
        py)      lang="python" ;;
        js)      lang="javascript" ;;
        ts)      lang="typescript" ;;
        *)       lang="$ext" ;;
      esac
      out+="### ${SCRIPT_NAMES[$i]}"$'\n\n'
      out+="\`\`\`${lang}"$'\n'
      out+="${SCRIPT_CONTENTS[$i]}"$'\n'
      out+="\`\`\`"$'\n\n'
    done
  fi

  # ── Inline references (budget-fitted to stay under CURSOR_MAX_BYTES) ──
  if [[ ${#REF_NAMES[@]} -gt 0 ]]; then
    local current_size=${#out}
    local budget=$(( CURSOR_MAX_BYTES - current_size - 200 ))  # 200 byte margin for section header + omission note
    local ref_section=""
    local omitted=0
    local i

    ref_section+=$'\n'"## References"$'\n\n'
    for i in "${!REF_NAMES[@]}"; do
      local entry=""
      entry+="### ${REF_NAMES[$i]}"$'\n\n'
      entry+="${REF_CONTENTS[$i]}"$'\n\n'
      local entry_size=${#entry}

      if [[ $budget -ge $entry_size ]]; then
        ref_section+="$entry"
        budget=$(( budget - entry_size ))
      else
        omitted=$(( omitted + 1 ))
      fi
    done

    if [[ $omitted -gt 0 ]]; then
      ref_section+="*${omitted} reference(s) omitted to stay under 100KB size limit.*"$'\n\n'
      echo "WARN: ${BUNDLE_NAME}: omitted $omitted reference(s) to stay under 100KB" >&2
    fi

    out+="$ref_section"
  fi

  CONVERTED_OUTPUT="$out"
  CONVERTED_FILENAME="${BUNDLE_NAME}.mdc"

  # ── MCP detection: scan body + references for MCP server references ──
  # If skill content references MCP servers, generate a stub mcp.json
  local all_content="${BUNDLE_BODY}"
  local i
  for i in "${!REF_CONTENTS[@]}"; do
    all_content+=$'\n'"${REF_CONTENTS[$i]}"
  done

  if echo "$all_content" | grep -qiE '(mcpServers|mcp_server|"mcp"|mcp\.json)'; then
    CONVERTED_OUTPUT_2='{
  "mcpServers": {}
}'
    CONVERTED_FILENAME_2="mcp.json"
  fi
}

run_convert() {
  local target="$1"
  case "$target" in
    test)   convert_test ;;
    codex)  convert_codex ;;
    cursor) convert_cursor ;;
    *)      die "Unknown target: $target. Supported: codex, cursor, test" ;;
  esac
}

# ─── Stage 3: Write ───────────────────────────────────────────────────

write_output() {
  local output_dir="$1"

  # Clean-write: delete target dir before writing
  if [[ -d "$output_dir" ]]; then
    rm -rf "$output_dir"
  fi
  mkdir -p "$output_dir"

  echo "$CONVERTED_OUTPUT" > "$output_dir/$CONVERTED_FILENAME"
  echo "OK: $output_dir/$CONVERTED_FILENAME"

  # Write secondary output if present (e.g., codex prompt.md)
  if [[ -n "${CONVERTED_OUTPUT_2:-}" && -n "${CONVERTED_FILENAME_2:-}" ]]; then
    echo "$CONVERTED_OUTPUT_2" > "$output_dir/$CONVERTED_FILENAME_2"
    echo "OK: $output_dir/$CONVERTED_FILENAME_2"
  fi
}

# ─── Main ─────────────────────────────────────────────────────────────

convert_one_skill() {
  local skill_dir="$1"
  local target="$2"
  local output_dir="$3"

  # Resolve skill_dir to absolute if relative
  if [[ "$skill_dir" != /* ]]; then
    skill_dir="$REPO_ROOT/$skill_dir"
  fi

  [[ -d "$skill_dir" ]] || die "Skill directory not found: $skill_dir"
  [[ -f "$skill_dir/SKILL.md" ]] || die "No SKILL.md in: $skill_dir"

  parse_bundle "$skill_dir"

  [[ -n "$BUNDLE_NAME" ]] || die "Failed to parse name from $skill_dir/SKILL.md"

  # Default output dir
  if [[ -z "$output_dir" ]]; then
    output_dir="$REPO_ROOT/.agents/converter/$target/$BUNDLE_NAME"
  elif [[ "$output_dir" != /* ]]; then
    output_dir="$REPO_ROOT/$output_dir"
  fi

  # Reset output variables
  CONVERTED_OUTPUT=""
  CONVERTED_FILENAME=""
  CONVERTED_OUTPUT_2=""
  CONVERTED_FILENAME_2=""

  run_convert "$target"
  write_output "$output_dir"
}

main() {
  [[ $# -ge 2 ]] || usage

  local skill_dir_or_flag="$1"
  local target="$2"
  local output_dir="${3:-}"

  if [[ "$skill_dir_or_flag" == "--all" ]]; then
    local skills_root="$REPO_ROOT/skills"
    local count=0
    for d in "$skills_root"/*/; do
      [[ -f "$d/SKILL.md" ]] || continue
      local sname
      sname="$(basename "$d")"
      local out="$output_dir"
      if [[ -n "$out" ]]; then
        # Per-skill subdir under the provided output dir
        if [[ "$out" != /* ]]; then
          out="$REPO_ROOT/$out/$sname"
        else
          out="$out/$sname"
        fi
      fi
      convert_one_skill "$d" "$target" "$out"
      count=$((count + 1))
    done
    echo "Converted $count skills to target '$target'"
  else
    convert_one_skill "$skill_dir_or_flag" "$target" "$output_dir"
  fi
}

main "$@"
