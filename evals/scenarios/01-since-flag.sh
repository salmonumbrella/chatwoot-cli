#!/usr/bin/env bash
# Eval: --since flag for conversations list

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/helpers.sh"
cd "$SCRIPT_DIR/../.."

start_eval "--since flag filters conversations by time"

# Test 1: --since with relative time
output=$(run_cli conversations list --since "1d ago" --output json 2>/dev/null || true)

# Verify it returns valid JSON with items array
if ! echo "$output" | jq -e '.items' >/dev/null 2>&1; then
    log "  ✗ Expected JSON output with items array"
    end_eval "fail"
    exit 1
fi

# Test 2: Verify conversations are recent (within last day)
# Get the oldest last_activity timestamp
one_day_ago=$(date -v-1d +%s 2>/dev/null || date -d "1 day ago" +%s)
oldest=$(echo "$output" | jq '[.items[].last_activity_at.unix // 0] | min')

if [[ "$oldest" -gt 0 && "$oldest" -lt "$one_day_ago" ]]; then
    log "  ✗ Found conversation older than 1 day (timestamp: $oldest)"
    end_eval "fail"
    exit 1
fi

log_verbose "  ✓ All conversations are from last 24 hours"

# Test 3: --since with ISO date format
today=$(date +%Y-%m-%d)
output2=$(run_cli conversations list --since "$today" --output json 2>/dev/null || true)

if ! echo "$output2" | jq -e '.items' >/dev/null 2>&1; then
    log "  ✗ Failed with ISO date format"
    end_eval "fail"
    exit 1
fi

log_verbose "  ✓ Works with ISO date format"

end_eval "pass"
