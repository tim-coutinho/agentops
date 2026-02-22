#!/usr/bin/env bash
# assertions.sh — Content assertion framework for OpenCode headless tests
#
# Source this file to get assertion functions for validating test output.
# Each function returns 0 on pass, 1 on fail, and prints a description.

# Assert that a file contains a pattern (case-insensitive)
assert_contains() {
    local logfile="$1"
    local pattern="$2"
    local description="${3:-contains '$pattern'}"

    if grep -qi "$pattern" "$logfile" 2>/dev/null; then
        return 0
    fi
    echo "  ASSERT FAIL: $description (pattern not found: $pattern)" >&2
    return 1
}

# Assert that a file was created during the test
assert_file_created() {
    local filepath="$1"
    local description="${2:-file exists: $filepath}"

    if [[ -f "$filepath" ]]; then
        return 0
    fi
    echo "  ASSERT FAIL: $description (file not found: $filepath)" >&2
    return 1
}

# Assert output exceeds minimum byte count
assert_output_gt() {
    local logfile="$1"
    local min_bytes="$2"
    local description="${3:-output > ${min_bytes} bytes}"

    local actual_bytes
    actual_bytes=$(wc -c < "$logfile" | tr -d ' ')
    if [[ $actual_bytes -gt $min_bytes ]]; then
        return 0
    fi
    echo "  ASSERT FAIL: $description (got ${actual_bytes} bytes, need > ${min_bytes})" >&2
    return 1
}

# Assert no common error patterns in output
assert_no_error() {
    local logfile="$1"
    local description="${2:-no error patterns}"

    # Check for common error indicators
    if grep -qiE '(Traceback|panic:|FATAL|Segmentation fault|core dumped)' "$logfile" 2>/dev/null; then
        local error_line
        error_line=$(grep -iE '(Traceback|panic:|FATAL|Segmentation fault|core dumped)' "$logfile" | head -1)
        echo "  ASSERT FAIL: $description (found: $error_line)" >&2
        return 1
    fi
    return 0
}

# Assert output contains ANY of the given patterns (case-insensitive)
assert_contains_any() {
    local logfile="$1"
    shift
    local description="${1:-contains any pattern}"
    shift
    local patterns=("$@")

    for pattern in "${patterns[@]}"; do
        if grep -qi "$pattern" "$logfile" 2>/dev/null; then
            return 0
        fi
    done
    echo "  ASSERT FAIL: $description (none of: ${patterns[*]})" >&2
    return 1
}
