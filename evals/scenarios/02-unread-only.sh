#!/usr/bin/env bash
# Eval: --unread-only flag for conversations list

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/helpers.sh"
cd "$SCRIPT_DIR/../.."

start_eval "--unread-only filters to unread conversations"

# Get conversations with --unread-only
output=$(run_cli conversations list --unread-only --output json 2>/dev/null || true)

if ! echo "$output" | jq -e '.items' >/dev/null 2>&1; then
    log "  ✗ Expected JSON output with items array"
    end_eval "fail"
    exit 1
fi

# Check that all returned conversations have unread_count > 0
zero_unread=$(echo "$output" | jq '[.items[] | select(.unread_count == 0)] | length')

if [[ "$zero_unread" -gt 0 ]]; then
    log "  ✗ Found $zero_unread conversations with unread_count=0"
    end_eval "fail"
    exit 1
fi

count=$(echo "$output" | jq '.items | length')
log_verbose "  ✓ All $count conversations have unread messages"

end_eval "pass"
