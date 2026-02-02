#!/usr/bin/env bash
# Eval: --with-open-conversations flag for contacts get

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/helpers.sh"
cd "$SCRIPT_DIR/../.."

start_eval "--with-open-conversations includes open conversations inline"

# Find a contact
contacts=$(run_cli contacts list --output json 2>/dev/null || true)
contact_id=$(echo "$contacts" | jq -r '.items[0].id // empty')

if [[ -z "$contact_id" ]]; then
    log "  ⚠ No contacts found, skipping"
    end_eval "pass"
    exit 0
fi

# Get contact WITH open conversations
output=$(run_cli contacts get "$contact_id" --with-open-conversations --output agent 2>/dev/null || true)

# Verify agent envelope
if ! assert_json_field "$output" ".kind" "contacts.get"; then
    end_eval "fail"
    exit 1
fi

# open_conversations should exist (may be empty array or have items)
if ! echo "$output" | jq -e '.item.open_conversations' >/dev/null 2>&1; then
    log "  ✗ Missing open_conversations field"
    end_eval "fail"
    exit 1
fi

open_count=$(echo "$output" | jq '.item.open_conversations | length')
log_verbose "  ✓ Has $open_count open conversations inline"

# If there are open conversations, verify they're all open or pending
if [[ "$open_count" -gt 0 ]]; then
    bad_status=$(echo "$output" | jq '[.item.open_conversations[] | select(.status != "open" and .status != "pending")] | length')
    if [[ "$bad_status" -gt 0 ]]; then
        log "  ✗ Found $bad_status conversations with wrong status"
        end_eval "fail"
        exit 1
    fi
    log_verbose "  ✓ All are open or pending status"
fi

end_eval "pass"
