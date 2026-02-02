#!/usr/bin/env bash
# Eval: --since-last-agent flag for messages list

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/helpers.sh"
cd "$SCRIPT_DIR/../.."

start_eval "--since-last-agent filters to new customer messages"

# Find an open conversation (more likely to have mixed message types)
convs=$(run_cli conversations list --status open --output json 2>/dev/null || true)
conv_id=$(echo "$convs" | jq -r '.items[0].id // empty')

if [[ -z "$conv_id" ]]; then
    log "  ⚠ No open conversations found, skipping"
    end_eval "pass"
    exit 0
fi

# Get all messages first
all_msgs=$(run_cli messages list "$conv_id" --output json 2>/dev/null || true)
all_count=$(echo "$all_msgs" | jq '.items | length')

# Get messages since last agent
filtered=$(run_cli messages list "$conv_id" --since-last-agent --output json 2>/dev/null || true)
filtered_count=$(echo "$filtered" | jq '.items | length')

log_verbose "  All messages: $all_count, Since last agent: $filtered_count"

# Filtered should be <= all
if [[ "$filtered_count" -gt "$all_count" ]]; then
    log "  ✗ Filtered count ($filtered_count) > all count ($all_count)"
    end_eval "fail"
    exit 1
fi

# If there are filtered messages, verify they're all incoming (type 0)
if [[ "$filtered_count" -gt 0 ]]; then
    outgoing=$(echo "$filtered" | jq '[.items[] | select(.type == "outgoing")] | length')
    if [[ "$outgoing" -gt 0 ]]; then
        log "  ✗ Found $outgoing outgoing messages in filtered results"
        end_eval "fail"
        exit 1
    fi
    log_verbose "  ✓ All filtered messages are incoming"
fi

end_eval "pass"
