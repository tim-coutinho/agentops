# shellcheck shell=bash
# Shared color definitions and helper functions for test scripts
# Usage: source "${SCRIPT_DIR}/../lib/colors.sh"
# (adjust path based on file location relative to tests/lib/)

# Guard against multiple sourcing
[[ -n "${COLORS_SH_LOADED:-}" ]] && return 0
COLORS_SH_LOADED=1

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Initialize pass/fail counters only if not already set
: "${PASS_COUNT:=0}"
: "${FAIL_COUNT:=0}"

# Helper functions
log() { echo -e "${BLUE}[TEST]${NC} $1"; }
pass() { echo -e "${GREEN}  ✓${NC} $1"; }
fail() { echo -e "${RED}  ✗${NC} $1"; }
warn() { echo -e "${YELLOW}  ⚠${NC} $1"; }

# Additional color helper functions (printf style)
red() { printf '\033[0;31m%s\033[0m\n' "$1"; }
green() { printf '\033[0;32m%s\033[0m\n' "$1"; }
yellow() { printf '\033[0;33m%s\033[0m\n' "$1"; }
blue() { printf '\033[0;34m%s\033[0m\n' "$1"; }
