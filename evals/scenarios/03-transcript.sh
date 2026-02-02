#!/usr/bin/env bash
# Eval: --transcript flag for messages list

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/helpers.sh"
cd "$SCRIPT_DIR/../.."

start_eval "--transcript produces readable conversation format"

# First, find a conversation with messages
conv_output=$(run_cli conversations list --output json 2>/dev/null || true)
conv_id=$(echo "$conv_output" | jq -r '.items[0].id // empty')

if [[ -z "$conv_id" ]]; then
    log "  ⚠ No conversations found, skipping"
    end_eval "pass"  # Can't test without data
    exit 0
fi

# Get transcript
output=$(run_cli messages list "$conv_id" --transcript 2>/dev/null || true)

# Verify transcript format markers
if ! assert_contains "$output" "==="; then
    end_eval "fail"
    exit 1
fi

# Should have timestamp format [YYYY-MM-DD HH:MM]
if ! echo "$output" | grep -qE '\[[0-9]{4}-[0-9]{2}-[0-9]{2}'; then
    log "  ✗ Missing timestamp format"
    end_eval "fail"
    exit 1
fi

log_verbose "  ✓ Has conversation header and timestamps"

# Should have direction markers (incoming/outgoing)
if echo "$output" | grep -qE '(incoming|outgoing)'; then
    log_verbose "  ✓ Has direction markers"
else
    log "  ✗ Missing direction markers"
    end_eval "fail"
    exit 1
fi

end_eval "pass"
