#!/usr/bin/env bash
# check-product-freshness.sh
# Checks PRODUCT.md for:
#   1. Line count <= 200
#   2. last_reviewed date within 30 days of today (macOS BSD date)
#   3. No section (## heading) longer than 30 lines

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
PRODUCT_FILE="${REPO_ROOT}/PRODUCT.md"

FAILED=0

# ---------------------------------------------------------------------------
# Check: file exists
# ---------------------------------------------------------------------------
if [[ ! -f "${PRODUCT_FILE}" ]]; then
  echo "ERROR: PRODUCT.md not found at ${PRODUCT_FILE}" >&2
  exit 1
fi

# ---------------------------------------------------------------------------
# Check 1: Line count <= 200
# ---------------------------------------------------------------------------
LINE_COUNT=$(wc -l < "${PRODUCT_FILE}" | tr -d ' ')
if [[ "${LINE_COUNT}" -gt 200 ]]; then
  echo "FAIL [line-count]: PRODUCT.md has ${LINE_COUNT} lines (limit: 200)" >&2
  FAILED=1
fi

# ---------------------------------------------------------------------------
# Check 2: last_reviewed date within 30 days of today
# ---------------------------------------------------------------------------
# Extract frontmatter (between first pair of --- markers)
FRONTMATTER=$(awk '/^---$/{found++; if(found==2) exit; next} found==1{print}' "${PRODUCT_FILE}")

if [[ -z "${FRONTMATTER}" ]]; then
  echo "FAIL [freshness]: No YAML frontmatter found in PRODUCT.md" >&2
  FAILED=1
else
  LAST_REVIEWED=$(echo "${FRONTMATTER}" | grep -E '^last_reviewed:' | sed 's/last_reviewed:[[:space:]]*//' | tr -d '"'"'" | head -1)

  if [[ -z "${LAST_REVIEWED}" ]]; then
    echo "FAIL [freshness]: No 'last_reviewed' field found in frontmatter" >&2
    FAILED=1
  else
    # macOS BSD date: compute the threshold (30 days ago)
    THRESHOLD=$(date -v-30d +%Y-%m-%d)

    # Compare dates lexicographically (YYYY-MM-DD format supports this)
    if [[ "${LAST_REVIEWED}" < "${THRESHOLD}" ]]; then
      TODAY=$(date +%Y-%m-%d)
      echo "FAIL [freshness]: last_reviewed (${LAST_REVIEWED}) is more than 30 days old (today: ${TODAY}, threshold: ${THRESHOLD})" >&2
      FAILED=1
    fi
  fi
fi

# ---------------------------------------------------------------------------
# Check 3: No section longer than 30 lines
# ---------------------------------------------------------------------------
# Parse file: track current section heading and count lines per section
CURRENT_HEADING=""
SECTION_LINE_COUNT=0

check_section_length() {
  local heading="$1"
  local count="$2"
  if [[ "${count}" -gt 30 ]]; then
    echo "FAIL [section-length]: Section '${heading}' has ${count} lines (limit: 30)" >&2
    FAILED=1
  fi
}

while IFS= read -r line || [[ -n "${line}" ]]; do
  if [[ "${line}" =~ ^##[[:space:]] ]]; then
    # Flush previous section
    if [[ -n "${CURRENT_HEADING}" ]]; then
      check_section_length "${CURRENT_HEADING}" "${SECTION_LINE_COUNT}"
    fi
    CURRENT_HEADING="${line}"
    SECTION_LINE_COUNT=0
  elif [[ -n "${CURRENT_HEADING}" ]]; then
    (( SECTION_LINE_COUNT++ )) || true
  fi
done < "${PRODUCT_FILE}"

# Flush the final section
if [[ -n "${CURRENT_HEADING}" ]]; then
  check_section_length "${CURRENT_HEADING}" "${SECTION_LINE_COUNT}"
fi

# ---------------------------------------------------------------------------
# Exit
# ---------------------------------------------------------------------------
if [[ "${FAILED}" -eq 0 ]]; then
  exit 0
else
  exit 1
fi
