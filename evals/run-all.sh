#!/usr/bin/env bash
# Run all agent mode evals

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# Ensure CLI is built
make build >/dev/null 2>&1

# Check auth
if ! ./bin/chatwoot auth status >/dev/null 2>&1; then
    echo "Error: Not authenticated. Run 'chatwoot auth login' first."
    exit 1
fi

PASS=0
FAIL=0
SKIP=0

echo "========================================"
echo "Agent Mode Evals"
echo "========================================"
echo ""

for scenario in evals/scenarios/*.sh; do
    if [[ -x "$scenario" ]]; then
        if bash "$scenario"; then
            ((PASS++))
        else
            ((FAIL++))
        fi
    fi
done

echo ""
echo "========================================"
echo "Results: $PASS passed, $FAIL failed, $SKIP skipped"
echo "========================================"

[[ "$FAIL" -eq 0 ]]
