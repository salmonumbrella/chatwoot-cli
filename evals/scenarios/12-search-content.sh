#!/usr/bin/env bash
# Eval: Search includes conversation content (not just metadata)

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/helpers.sh"
cd "$SCRIPT_DIR/../.."

start_eval "Search includes conversation message content"

# Search for a common word that's likely in messages
# Using "hello" or "thank" as they're common in customer support
output=$(run_cli search "thank" --type conversations --output json 2>/dev/null || true)

if ! echo "$output" | jq -e '.results' >/dev/null 2>&1; then
    log "  ✗ Expected JSON output with results array"
    end_eval "fail"
    exit 1
fi

count=$(echo "$output" | jq '.results | length')
log_verbose "  Found $count conversations matching 'thank'"

# Verify search summary
if assert_json_exists "$output" ".summary"; then
    log_verbose "  ✓ Has search summary"
fi

# Verify kind is correct
if ! assert_json_field "$output" ".kind" "search.conversations"; then
    end_eval "fail"
    exit 1
fi

log_verbose "  ✓ Search returns proper agent envelope"

end_eval "pass"
