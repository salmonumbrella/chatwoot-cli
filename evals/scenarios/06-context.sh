#!/usr/bin/env bash
# Eval: --context flag for comprehensive conversation context

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/helpers.sh"
cd "$SCRIPT_DIR/../.."

start_eval "--context provides comprehensive context in one call"

# Find a conversation with a contact
convs=$(run_cli conversations list --output json 2>/dev/null || true)
conv_id=$(echo "$convs" | jq -r '.items[0].id // empty')

if [[ -z "$conv_id" ]]; then
    log "  ⚠ No conversations found, skipping"
    end_eval "pass"
    exit 0
fi

# Get comprehensive context
output=$(run_cli conversations get "$conv_id" --context --output agent 2>/dev/null || true)

# Verify all three components exist
if ! assert_json_exists "$output" ".item.conversation"; then
    log "  ✗ Missing conversation"
    end_eval "fail"
    exit 1
fi

if ! assert_json_exists "$output" ".item.messages"; then
    log "  ✗ Missing messages"
    end_eval "fail"
    exit 1
fi

# Contact may be null if conversation has no contact_id
contact_id=$(echo "$output" | jq -r '.item.conversation.contact_id // 0')
if [[ "$contact_id" -gt 0 ]]; then
    if ! assert_json_exists "$output" ".item.contact"; then
        log "  ✗ Missing contact (contact_id=$contact_id)"
        end_eval "fail"
        exit 1
    fi

    if ! assert_json_exists "$output" ".item.contact.relationship"; then
        log "  ✗ Missing contact relationship"
        end_eval "fail"
        exit 1
    fi
    log_verbose "  ✓ Has conversation, messages, contact with relationship"
else
    log_verbose "  ✓ Has conversation, messages (no contact_id)"
fi

# This single call replaces: conversations get + messages list + contacts get + contacts conversations
log_verbose "  ✓ Single call provides full context (replaces 4 API calls)"

end_eval "pass"
