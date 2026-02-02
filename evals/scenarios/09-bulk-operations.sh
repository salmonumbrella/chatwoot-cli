#!/usr/bin/env bash
# Eval: Bulk resolve/assign commands (parse only, don't actually resolve)

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/helpers.sh"
cd "$SCRIPT_DIR/../.."

start_eval "Bulk resolve/assign accept multiple IDs"

# Test 1: Verify resolve command accepts multiple IDs (help text)
help_output=$(run_cli conversations resolve --help 2>&1 || true)

if ! assert_contains "$help_output" "[id...]"; then
    log "  ✗ resolve command doesn't show multi-ID usage"
    end_eval "fail"
    exit 1
fi

log_verbose "  ✓ resolve command supports multiple IDs"

# Test 2: Verify assign command accepts multiple IDs
help_output=$(run_cli conversations assign --help 2>&1 || true)

if ! assert_contains "$help_output" "[id...]"; then
    log "  ✗ assign command doesn't show multi-ID usage"
    end_eval "fail"
    exit 1
fi

if ! assert_contains "$help_output" "--agent"; then
    log "  ✗ assign command missing --agent flag"
    end_eval "fail"
    exit 1
fi

log_verbose "  ✓ assign command supports multiple IDs and --agent flag"

# Test 3: Error handling for invalid IDs
error_output=$(run_cli conversations resolve abc 2>&1 || true)

if ! assert_contains "$error_output" "invalid"; then
    log "  ✗ No error for invalid ID 'abc'"
    end_eval "fail"
    exit 1
fi

log_verbose "  ✓ Proper error handling for invalid IDs"

# Test 4: Assign requires --agent or --team
error_output=$(run_cli conversations assign 123 2>&1 || true)

if ! assert_contains "$error_output" "required"; then
    log "  ✗ No error when --agent/--team missing"
    end_eval "fail"
    exit 1
fi

log_verbose "  ✓ Assign requires --agent or --team"

end_eval "pass"
