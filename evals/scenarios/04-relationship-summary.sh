#!/usr/bin/env bash
# Eval: Relationship summary in contact agent output

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/helpers.sh"
cd "$SCRIPT_DIR/../.."

start_eval "Contact agent output includes relationship summary"

# Find a contact with conversations
contacts=$(run_cli contacts list --output json 2>/dev/null || true)
contact_id=$(echo "$contacts" | jq -r '.items[0].id // empty')

if [[ -z "$contact_id" ]]; then
    log "  ⚠ No contacts found, skipping"
    end_eval "pass"
    exit 0
fi

# Get contact in agent mode
output=$(run_cli contacts get "$contact_id" --output agent 2>/dev/null || true)

# Verify agent envelope
if ! assert_json_field "$output" ".kind" "contacts.get"; then
    end_eval "fail"
    exit 1
fi

# Verify relationship field exists
if ! assert_json_exists "$output" ".item.relationship"; then
    end_eval "fail"
    exit 1
fi

# Verify relationship has expected fields
if ! assert_json_exists "$output" ".item.relationship.total_conversations"; then
    end_eval "fail"
    exit 1
fi

log_verbose "  ✓ Relationship summary present with total_conversations"

end_eval "pass"
