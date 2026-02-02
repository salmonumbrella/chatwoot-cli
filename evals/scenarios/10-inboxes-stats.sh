#!/usr/bin/env bash
# Eval: inboxes stats command

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/helpers.sh"
cd "$SCRIPT_DIR/../.."

start_eval "inboxes stats provides health metrics"

# Find an inbox
inboxes=$(run_cli inboxes list --output json 2>/dev/null || true)
inbox_id=$(echo "$inboxes" | jq -r '.items[0].id // empty')

if [[ -z "$inbox_id" ]]; then
    log "  ⚠ No inboxes found, skipping"
    end_eval "pass"
    exit 0
fi

# Get stats
output=$(run_cli inboxes stats "$inbox_id" --output json 2>/dev/null || true)

# Verify required fields
if ! assert_json_exists "$output" ".inbox_id"; then
    end_eval "fail"
    exit 1
fi

if ! assert_json_exists "$output" ".inbox_name"; then
    end_eval "fail"
    exit 1
fi

if ! assert_json_exists "$output" ".open_count"; then
    end_eval "fail"
    exit 1
fi

if ! assert_json_exists "$output" ".pending_count"; then
    end_eval "fail"
    exit 1
fi

if ! assert_json_exists "$output" ".unread_count"; then
    end_eval "fail"
    exit 1
fi

log_verbose "  ✓ Has all expected metrics"

# Verify text output
text_output=$(run_cli inboxes stats "$inbox_id" 2>/dev/null || true)

if ! assert_contains "$text_output" "Open:"; then
    end_eval "fail"
    exit 1
fi

log_verbose "  ✓ Text output is readable"

end_eval "pass"
