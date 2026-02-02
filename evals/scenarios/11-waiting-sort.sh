#!/usr/bin/env bash
# Eval: --waiting flag sorts by customer wait time

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/helpers.sh"
cd "$SCRIPT_DIR/../.."

start_eval "--waiting sorts by customer wait time"

# Get conversations sorted by wait time
output=$(run_cli conversations list --waiting --output json 2>/dev/null || true)

if ! echo "$output" | jq -e '.items' >/dev/null 2>&1; then
    log "  ✗ Expected JSON output with items array"
    end_eval "fail"
    exit 1
fi

count=$(echo "$output" | jq '.items | length')

if [[ "$count" -lt 2 ]]; then
    log_verbose "  ⚠ Need 2+ conversations to verify sort order, got $count"
    end_eval "pass"
    exit 0
fi

# Verify sort order: oldest last_activity_at should be first
# (using last_activity_at as proxy for wait time)
first_activity=$(echo "$output" | jq '.items[0].last_activity_at.unix // .items[0].last_activity_at // 0')
last_activity=$(echo "$output" | jq '.items[-1].last_activity_at.unix // .items[-1].last_activity_at // 0')

if [[ "$first_activity" -gt "$last_activity" && "$last_activity" -gt 0 ]]; then
    log "  ✗ Sort order wrong: first=$first_activity, last=$last_activity"
    end_eval "fail"
    exit 1
fi

log_verbose "  ✓ Sorted by wait time (oldest activity first)"

end_eval "pass"
