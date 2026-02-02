#!/usr/bin/env bash
# Eval: --with-messages flag for conversations get

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/helpers.sh"
cd "$SCRIPT_DIR/../.."

start_eval "--with-messages includes messages inline"

# Find a conversation
convs=$(run_cli conversations list --output json 2>/dev/null || true)
conv_id=$(echo "$convs" | jq -r '.items[0].id // empty')

if [[ -z "$conv_id" ]]; then
    log "  ⚠ No conversations found, skipping"
    end_eval "pass"
    exit 0
fi

# Test 1: Get conversation WITH messages in agent mode
output=$(run_cli conversations get "$conv_id" --with-messages --output agent 2>/dev/null || true)

# Verify messages array exists
if ! assert_json_exists "$output" ".item.messages"; then
    end_eval "fail"
    exit 1
fi

msg_count=$(echo "$output" | jq '.item.messages | length')
log_verbose "  ✓ Got $msg_count messages inline"

# Test 2: Without flag, should NOT have messages
output_no_flag=$(run_cli conversations get "$conv_id" --output agent 2>/dev/null || true)

if echo "$output_no_flag" | jq -e '.item.messages' >/dev/null 2>&1; then
    log "  ✗ Messages present without --with-messages flag"
    end_eval "fail"
    exit 1
fi

log_verbose "  ✓ No messages without flag"

# Test 3: Message limit works
output_limited=$(run_cli conversations get "$conv_id" --with-messages --message-limit 5 --output agent 2>/dev/null || true)
limited_count=$(echo "$output_limited" | jq '.item.messages | length')

if [[ "$limited_count" -gt 5 ]]; then
    log "  ✗ Message limit not respected: got $limited_count"
    end_eval "fail"
    exit 1
fi

log_verbose "  ✓ Message limit works ($limited_count <= 5)"

end_eval "pass"
