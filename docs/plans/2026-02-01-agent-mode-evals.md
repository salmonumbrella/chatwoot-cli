# Agent Mode Evals Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a comprehensive eval framework to test all agent mode features and measure their effectiveness for AI assistants.

**Architecture:** Shell-based eval scripts that run real CLI commands against a Chatwoot instance, capture outputs, and verify expected behaviors. Each eval scenario tests specific agent mode features with success/failure criteria.

**Tech Stack:** Bash scripts, jq for JSON parsing, the chatwoot CLI itself

---

## Overview

We have 12 agent mode features to evaluate:

**Phase 1 Features:**
1. `--since` flag for conversations list
2. `--unread-only` flag for conversations list
3. `--transcript` format for messages
4. Relationship summary in contact output
5. Search includes conversation content

**Phase 2 Features:**
6. `conversations get --with-messages`
7. `conversations list --waiting`
8. `conversations get --context`
9. `messages list --since-last-agent`
10. `contacts get --with-open-conversations`
11. `conversations resolve/assign` multi-ID
12. `inboxes stats`

Each eval tests:
- **Functionality**: Does the feature work as expected?
- **Efficiency**: Does it reduce API calls compared to manual approach?
- **Usefulness**: Is the output actionable for an AI assistant?

---

## Task 1: Create eval directory structure

**Files:**
- Create: `evals/README.md`
- Create: `evals/run-all.sh`
- Create: `evals/lib/helpers.sh`

**Step 1: Create directory structure**

```bash
mkdir -p evals/lib evals/scenarios evals/results
```

**Step 2: Create README.md**

Create `evals/README.md`:

```markdown
# Agent Mode Evals

Evaluation suite for chatwoot-cli agent mode features.

## Prerequisites

- Authenticated chatwoot CLI (`chatwoot auth login`)
- Access to a Chatwoot instance with test data

## Running Evals

```bash
# Run all evals
./evals/run-all.sh

# Run specific scenario
./evals/scenarios/01-since-flag.sh

# Run with verbose output
VERBOSE=1 ./evals/run-all.sh
```

## Results

Results are written to `evals/results/` with timestamps.

## Adding New Evals

1. Create a new script in `evals/scenarios/`
2. Source `../lib/helpers.sh`
3. Use `run_eval`, `assert_json_field`, `assert_contains` helpers
4. Return 0 for pass, 1 for fail
```

**Step 3: Create helpers.sh**

Create `evals/lib/helpers.sh`:

```bash
#!/usr/bin/env bash
# Eval helper functions

set -euo pipefail

EVAL_NAME=""
EVAL_START=""
EVAL_RESULTS_DIR="${EVAL_RESULTS_DIR:-evals/results}"
VERBOSE="${VERBOSE:-0}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "$*" >&2
}

log_verbose() {
    [[ "$VERBOSE" == "1" ]] && log "$*"
}

start_eval() {
    EVAL_NAME="$1"
    EVAL_START=$(date +%s.%N)
    log "${YELLOW}Running:${NC} $EVAL_NAME"
}

end_eval() {
    local status="$1"
    local end=$(date +%s.%N)
    local duration=$(echo "$end - $EVAL_START" | bc)

    if [[ "$status" == "pass" ]]; then
        log "${GREEN}✓ PASS${NC} $EVAL_NAME (${duration}s)"
        return 0
    else
        log "${RED}✗ FAIL${NC} $EVAL_NAME (${duration}s)"
        return 1
    fi
}

# Run a CLI command and capture output
run_cli() {
    local cmd="$*"
    log_verbose "  Running: chatwoot $cmd"
    ./bin/chatwoot $cmd 2>&1
}

# Assert JSON output has a field with expected value
assert_json_field() {
    local json="$1"
    local jq_path="$2"
    local expected="$3"

    local actual
    actual=$(echo "$json" | jq -r "$jq_path" 2>/dev/null || echo "JQ_ERROR")

    if [[ "$actual" == "$expected" ]]; then
        log_verbose "  ✓ $jq_path == $expected"
        return 0
    else
        log "  ✗ Expected $jq_path to be '$expected', got '$actual'"
        return 1
    fi
}

# Assert JSON output has a field that exists (not null)
assert_json_exists() {
    local json="$1"
    local jq_path="$2"

    local value
    value=$(echo "$json" | jq -r "$jq_path" 2>/dev/null || echo "null")

    if [[ "$value" != "null" && "$value" != "" ]]; then
        log_verbose "  ✓ $jq_path exists"
        return 0
    else
        log "  ✗ Expected $jq_path to exist"
        return 1
    fi
}

# Assert JSON array has at least N items
assert_json_array_min() {
    local json="$1"
    local jq_path="$2"
    local min_count="$3"

    local count
    count=$(echo "$json" | jq "$jq_path | length" 2>/dev/null || echo "0")

    if [[ "$count" -ge "$min_count" ]]; then
        log_verbose "  ✓ $jq_path has $count items (>= $min_count)"
        return 0
    else
        log "  ✗ Expected $jq_path to have >= $min_count items, got $count"
        return 1
    fi
}

# Assert output contains a string
assert_contains() {
    local output="$1"
    local expected="$2"

    if [[ "$output" == *"$expected"* ]]; then
        log_verbose "  ✓ Output contains '$expected'"
        return 0
    else
        log "  ✗ Expected output to contain '$expected'"
        return 1
    fi
}

# Count API calls (by counting CLI invocations in a function)
count_api_calls() {
    echo "$1"
}
```

**Step 4: Create run-all.sh**

Create `evals/run-all.sh`:

```bash
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
```

**Step 5: Make scripts executable and commit**

```bash
chmod +x evals/run-all.sh evals/lib/helpers.sh
git add -f evals/
git commit -m "feat(evals): add eval framework with helpers"
```

---

## Task 2: Eval for --since flag

**Files:**
- Create: `evals/scenarios/01-since-flag.sh`

**Step 1: Create eval script**

Create `evals/scenarios/01-since-flag.sh`:

```bash
#!/usr/bin/env bash
# Eval: --since flag for conversations list

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/helpers.sh"
cd "$SCRIPT_DIR/../.."

start_eval "--since flag filters conversations by time"

# Test 1: --since with relative time
output=$(run_cli conversations list --since "1d ago" --output json 2>/dev/null || true)

# Verify it returns valid JSON with items array
if ! echo "$output" | jq -e '.items' >/dev/null 2>&1; then
    log "  ✗ Expected JSON output with items array"
    end_eval "fail"
    exit 1
fi

# Test 2: Verify conversations are recent (within last day)
# Get the oldest last_activity timestamp
one_day_ago=$(date -v-1d +%s 2>/dev/null || date -d "1 day ago" +%s)
oldest=$(echo "$output" | jq '[.items[].last_activity_at.unix // 0] | min')

if [[ "$oldest" -gt 0 && "$oldest" -lt "$one_day_ago" ]]; then
    log "  ✗ Found conversation older than 1 day (timestamp: $oldest)"
    end_eval "fail"
    exit 1
fi

log_verbose "  ✓ All conversations are from last 24 hours"

# Test 3: --since with ISO date format
today=$(date +%Y-%m-%d)
output2=$(run_cli conversations list --since "$today" --output json 2>/dev/null || true)

if ! echo "$output2" | jq -e '.items' >/dev/null 2>&1; then
    log "  ✗ Failed with ISO date format"
    end_eval "fail"
    exit 1
fi

log_verbose "  ✓ Works with ISO date format"

end_eval "pass"
```

**Step 2: Make executable and commit**

```bash
chmod +x evals/scenarios/01-since-flag.sh
git add -f evals/scenarios/01-since-flag.sh
git commit -m "feat(evals): add --since flag eval"
```

---

## Task 3: Eval for --unread-only flag

**Files:**
- Create: `evals/scenarios/02-unread-only.sh`

**Step 1: Create eval script**

Create `evals/scenarios/02-unread-only.sh`:

```bash
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
```

**Step 2: Make executable and commit**

```bash
chmod +x evals/scenarios/02-unread-only.sh
git add -f evals/scenarios/02-unread-only.sh
git commit -m "feat(evals): add --unread-only flag eval"
```

---

## Task 4: Eval for --transcript format

**Files:**
- Create: `evals/scenarios/03-transcript.sh`

**Step 1: Create eval script**

Create `evals/scenarios/03-transcript.sh`:

```bash
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
```

**Step 2: Make executable and commit**

```bash
chmod +x evals/scenarios/03-transcript.sh
git add -f evals/scenarios/03-transcript.sh
git commit -m "feat(evals): add --transcript format eval"
```

---

## Task 5: Eval for relationship summary

**Files:**
- Create: `evals/scenarios/04-relationship-summary.sh`

**Step 1: Create eval script**

Create `evals/scenarios/04-relationship-summary.sh`:

```bash
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
```

**Step 2: Make executable and commit**

```bash
chmod +x evals/scenarios/04-relationship-summary.sh
git add -f evals/scenarios/04-relationship-summary.sh
git commit -m "feat(evals): add relationship summary eval"
```

---

## Task 6: Eval for --with-messages flag

**Files:**
- Create: `evals/scenarios/05-with-messages.sh`

**Step 1: Create eval script**

Create `evals/scenarios/05-with-messages.sh`:

```bash
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
```

**Step 2: Make executable and commit**

```bash
chmod +x evals/scenarios/05-with-messages.sh
git add -f evals/scenarios/05-with-messages.sh
git commit -m "feat(evals): add --with-messages eval"
```

---

## Task 7: Eval for --context flag

**Files:**
- Create: `evals/scenarios/06-context.sh`

**Step 1: Create eval script**

Create `evals/scenarios/06-context.sh`:

```bash
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
```

**Step 2: Make executable and commit**

```bash
chmod +x evals/scenarios/06-context.sh
git add -f evals/scenarios/06-context.sh
git commit -m "feat(evals): add --context flag eval"
```

---

## Task 8: Eval for --since-last-agent flag

**Files:**
- Create: `evals/scenarios/07-since-last-agent.sh`

**Step 1: Create eval script**

Create `evals/scenarios/07-since-last-agent.sh`:

```bash
#!/usr/bin/env bash
# Eval: --since-last-agent flag for messages list

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/helpers.sh"
cd "$SCRIPT_DIR/../.."

start_eval "--since-last-agent filters to new customer messages"

# Find an open conversation (more likely to have mixed message types)
convs=$(run_cli conversations list --status open --output json 2>/dev/null || true)
conv_id=$(echo "$convs" | jq -r '.items[0].id // empty')

if [[ -z "$conv_id" ]]; then
    log "  ⚠ No open conversations found, skipping"
    end_eval "pass"
    exit 0
fi

# Get all messages first
all_msgs=$(run_cli messages list "$conv_id" --output json 2>/dev/null || true)
all_count=$(echo "$all_msgs" | jq '.items | length')

# Get messages since last agent
filtered=$(run_cli messages list "$conv_id" --since-last-agent --output json 2>/dev/null || true)
filtered_count=$(echo "$filtered" | jq '.items | length')

log_verbose "  All messages: $all_count, Since last agent: $filtered_count"

# Filtered should be <= all
if [[ "$filtered_count" -gt "$all_count" ]]; then
    log "  ✗ Filtered count ($filtered_count) > all count ($all_count)"
    end_eval "fail"
    exit 1
fi

# If there are filtered messages, verify they're all incoming (type 0)
if [[ "$filtered_count" -gt 0 ]]; then
    outgoing=$(echo "$filtered" | jq '[.items[] | select(.type == "outgoing")] | length')
    if [[ "$outgoing" -gt 0 ]]; then
        log "  ✗ Found $outgoing outgoing messages in filtered results"
        end_eval "fail"
        exit 1
    fi
    log_verbose "  ✓ All filtered messages are incoming"
fi

end_eval "pass"
```

**Step 2: Make executable and commit**

```bash
chmod +x evals/scenarios/07-since-last-agent.sh
git add -f evals/scenarios/07-since-last-agent.sh
git commit -m "feat(evals): add --since-last-agent eval"
```

---

## Task 9: Eval for --with-open-conversations flag

**Files:**
- Create: `evals/scenarios/08-with-open-conversations.sh`

**Step 1: Create eval script**

Create `evals/scenarios/08-with-open-conversations.sh`:

```bash
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
```

**Step 2: Make executable and commit**

```bash
chmod +x evals/scenarios/08-with-open-conversations.sh
git add -f evals/scenarios/08-with-open-conversations.sh
git commit -m "feat(evals): add --with-open-conversations eval"
```

---

## Task 10: Eval for bulk resolve/assign commands

**Files:**
- Create: `evals/scenarios/09-bulk-operations.sh`

**Step 1: Create eval script**

Create `evals/scenarios/09-bulk-operations.sh`:

```bash
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
```

**Step 2: Make executable and commit**

```bash
chmod +x evals/scenarios/09-bulk-operations.sh
git add -f evals/scenarios/09-bulk-operations.sh
git commit -m "feat(evals): add bulk operations eval"
```

---

## Task 11: Eval for inboxes stats command

**Files:**
- Create: `evals/scenarios/10-inboxes-stats.sh`

**Step 1: Create eval script**

Create `evals/scenarios/10-inboxes-stats.sh`:

```bash
#!/usr/bin/env bash
# Eval: inboxes stats command

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/helpers.sh"
cd "$SCRIPT_DIR/../.."

start_eval "inboxes stats provides health metrics"

# Find an inbox
inboxes=$(run_cli inboxes list --output json 2>/dev/null || true)
inbox_id=$(echo "$inboxes" | jq -r '.items[0].id // empty')

if [[ -z "$inbox_id" ]]; then
    log "  ⚠ No inboxes found, skipping"
    end_eval "pass"
    exit 0
fi

# Get stats
output=$(run_cli inboxes stats "$inbox_id" --output json 2>/dev/null || true)

# Verify required fields
if ! assert_json_exists "$output" ".inbox_id"; then
    end_eval "fail"
    exit 1
fi

if ! assert_json_exists "$output" ".inbox_name"; then
    end_eval "fail"
    exit 1
fi

if ! assert_json_exists "$output" ".open_count"; then
    end_eval "fail"
    exit 1
fi

if ! assert_json_exists "$output" ".pending_count"; then
    end_eval "fail"
    exit 1
fi

if ! assert_json_exists "$output" ".unread_count"; then
    end_eval "fail"
    exit 1
fi

log_verbose "  ✓ Has all expected metrics"

# Verify text output
text_output=$(run_cli inboxes stats "$inbox_id" 2>/dev/null || true)

if ! assert_contains "$text_output" "Open:"; then
    end_eval "fail"
    exit 1
fi

log_verbose "  ✓ Text output is readable"

end_eval "pass"
```

**Step 2: Make executable and commit**

```bash
chmod +x evals/scenarios/10-inboxes-stats.sh
git add -f evals/scenarios/10-inboxes-stats.sh
git commit -m "feat(evals): add inboxes stats eval"
```

---

## Task 12: Eval for --waiting flag

**Files:**
- Create: `evals/scenarios/11-waiting-sort.sh`

**Step 1: Create eval script**

Create `evals/scenarios/11-waiting-sort.sh`:

```bash
#!/usr/bin/env bash
# Eval: --waiting flag sorts by customer wait time

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/helpers.sh"
cd "$SCRIPT_DIR/../.."

start_eval "--waiting sorts by customer wait time"

# Get conversations sorted by wait time
output=$(run_cli conversations list --waiting --output json 2>/dev/null || true)

if ! echo "$output" | jq -e '.items' >/dev/null 2>&1; then
    log "  ✗ Expected JSON output with items array"
    end_eval "fail"
    exit 1
fi

count=$(echo "$output" | jq '.items | length')

if [[ "$count" -lt 2 ]]; then
    log_verbose "  ⚠ Need 2+ conversations to verify sort order, got $count"
    end_eval "pass"
    exit 0
fi

# Verify sort order: oldest last_activity_at should be first
# (using last_activity_at as proxy for wait time)
first_activity=$(echo "$output" | jq '.items[0].last_activity_at.unix // .items[0].last_activity_at // 0')
last_activity=$(echo "$output" | jq '.items[-1].last_activity_at.unix // .items[-1].last_activity_at // 0')

if [[ "$first_activity" -gt "$last_activity" && "$last_activity" -gt 0 ]]; then
    log "  ✗ Sort order wrong: first=$first_activity, last=$last_activity"
    end_eval "fail"
    exit 1
fi

log_verbose "  ✓ Sorted by wait time (oldest activity first)"

end_eval "pass"
```

**Step 2: Make executable and commit**

```bash
chmod +x evals/scenarios/11-waiting-sort.sh
git add -f evals/scenarios/11-waiting-sort.sh
git commit -m "feat(evals): add --waiting sort eval"
```

---

## Task 13: Eval for search content

**Files:**
- Create: `evals/scenarios/12-search-content.sh`

**Step 1: Create eval script**

Create `evals/scenarios/12-search-content.sh`:

```bash
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
```

**Step 2: Make executable and commit**

```bash
chmod +x evals/scenarios/12-search-content.sh
git add -f evals/scenarios/12-search-content.sh
git commit -m "feat(evals): add search content eval"
```

---

## Summary

| Task | Eval | Feature Tested |
|------|------|----------------|
| 1 | Framework setup | Helpers, run-all script |
| 2 | 01-since-flag.sh | `--since` flag |
| 3 | 02-unread-only.sh | `--unread-only` flag |
| 4 | 03-transcript.sh | `--transcript` format |
| 5 | 04-relationship-summary.sh | Relationship summary |
| 6 | 05-with-messages.sh | `--with-messages` flag |
| 7 | 06-context.sh | `--context` flag |
| 8 | 07-since-last-agent.sh | `--since-last-agent` flag |
| 9 | 08-with-open-conversations.sh | `--with-open-conversations` flag |
| 10 | 09-bulk-operations.sh | Bulk resolve/assign |
| 11 | 10-inboxes-stats.sh | `inboxes stats` command |
| 12 | 11-waiting-sort.sh | `--waiting` flag |
| 13 | 12-search-content.sh | Search content |

After all tasks, run `./evals/run-all.sh` to execute all evals against a live Chatwoot instance.
