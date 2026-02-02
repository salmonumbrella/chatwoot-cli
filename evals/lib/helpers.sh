#!/usr/bin/env bash
# Eval helper functions

set -euo pipefail

EVAL_NAME=""
EVAL_START=""
EVAL_RESULTS_DIR="${EVAL_RESULTS_DIR:-evals/results}"
VERBOSE="${VERBOSE:-0}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "$*" >&2
}

log_verbose() {
    [[ "$VERBOSE" == "1" ]] && log "$*"
}

start_eval() {
    EVAL_NAME="$1"
    EVAL_START=$(date +%s.%N)
    log "${YELLOW}Running:${NC} $EVAL_NAME"
}

end_eval() {
    local status="$1"
    local end=$(date +%s.%N)
    local duration=$(echo "$end - $EVAL_START" | bc)

    if [[ "$status" == "pass" ]]; then
        log "${GREEN}✓ PASS${NC} $EVAL_NAME (${duration}s)"
        return 0
    else
        log "${RED}✗ FAIL${NC} $EVAL_NAME (${duration}s)"
        return 1
    fi
}

# Run a CLI command and capture output
run_cli() {
    local cmd="$*"
    log_verbose "  Running: chatwoot $cmd"
    ./bin/chatwoot $cmd 2>&1
}

# Assert JSON output has a field with expected value
assert_json_field() {
    local json="$1"
    local jq_path="$2"
    local expected="$3"

    local actual
    actual=$(echo "$json" | jq -r "$jq_path" 2>/dev/null || echo "JQ_ERROR")

    if [[ "$actual" == "$expected" ]]; then
        log_verbose "  ✓ $jq_path == $expected"
        return 0
    else
        log "  ✗ Expected $jq_path to be '$expected', got '$actual'"
        return 1
    fi
}

# Assert JSON output has a field that exists (not null)
assert_json_exists() {
    local json="$1"
    local jq_path="$2"

    local value
    value=$(echo "$json" | jq -r "$jq_path" 2>/dev/null || echo "null")

    if [[ "$value" != "null" && "$value" != "" ]]; then
        log_verbose "  ✓ $jq_path exists"
        return 0
    else
        log "  ✗ Expected $jq_path to exist"
        return 1
    fi
}

# Assert JSON array has at least N items
assert_json_array_min() {
    local json="$1"
    local jq_path="$2"
    local min_count="$3"

    local count
    count=$(echo "$json" | jq "$jq_path | length" 2>/dev/null || echo "0")

    if [[ "$count" -ge "$min_count" ]]; then
        log_verbose "  ✓ $jq_path has $count items (>= $min_count)"
        return 0
    else
        log "  ✗ Expected $jq_path to have >= $min_count items, got $count"
        return 1
    fi
}

# Assert output contains a string
assert_contains() {
    local output="$1"
    local expected="$2"

    if [[ "$output" == *"$expected"* ]]; then
        log_verbose "  ✓ Output contains '$expected'"
        return 0
    else
        log "  ✗ Expected output to contain '$expected'"
        return 1
    fi
}

# Count API calls (by counting CLI invocations in a function)
count_api_calls() {
    echo "$1"
}
