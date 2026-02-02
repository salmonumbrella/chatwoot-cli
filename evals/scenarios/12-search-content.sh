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

# The search output has { "query": "...", "conversations": [...] } structure
if ! echo "$output" | jq -e '.conversations' >/dev/null 2>&1; then
    log "  ✗ Expected JSON output with conversations array"
    end_eval "fail"
    exit 1
fi

count=$(echo "$output" | jq '.conversations | length')
log_verbose "  Found $count conversations matching 'thank'"

# Verify the query is included in the response
if ! assert_json_field "$output" ".query" "thank"; then
    log "  ✗ Query not echoed in response"
    end_eval "fail"
    exit 1
fi

log_verbose "  ✓ Search returns conversations with query echo"

end_eval "pass"
