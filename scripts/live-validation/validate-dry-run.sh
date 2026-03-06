#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

ensure_tools
ensure_preflight
load_state

scratch_conversation_id="$(require_state SCRATCH_CONVERSATION_ID)"
baseline_agent_id="${SCRATCH_BASELINE_AGENT_ID:-}"
if [[ -z "${baseline_agent_id}" ]]; then
  baseline_agent_id="$(printf '%s\n' "$(choose_agent_ids)" | awk '{print $1}')"
fi

before_snapshot="$(conversation_snapshot "${scratch_conversation_id}")"
assign_preview="$(cw assign "${scratch_conversation_id}" --agent "${baseline_agent_id}" --dry-run -o json --compact-json)"
close_preview="$(cw close "${scratch_conversation_id}" --dry-run -o json --compact-json)"
reopen_preview="$(cw reopen "${scratch_conversation_id}" --dry-run -o json --compact-json)"
after_snapshot="$(conversation_snapshot "${scratch_conversation_id}")"

printf '%s' "${before_snapshot}" >"$(artifact_path dry-run-before.json)"
printf '%s' "${assign_preview}" >"$(artifact_path dry-run-assign.json)"
printf '%s' "${close_preview}" >"$(artifact_path dry-run-close.json)"
printf '%s' "${reopen_preview}" >"$(artifact_path dry-run-reopen.json)"
printf '%s' "${after_snapshot}" >"$(artifact_path dry-run-after.json)"

printf '%s' "${assign_preview}" | jq -e '.dry_run == true and .operation == "assign"' >/dev/null
printf '%s' "${close_preview}" | jq -e '.dry_run == true and .operation == "close"' >/dev/null
printf '%s' "${reopen_preview}" | jq -e '.dry_run == true and .operation == "reopen"' >/dev/null

before_compact="$(printf '%s' "${before_snapshot}" | jq -c -S '.')"
after_compact="$(printf '%s' "${after_snapshot}" | jq -c -S '.')"
[[ "${before_compact}" == "${after_compact}" ]] || die "scratch conversation changed after dry-run commands"

printf 'dry_run_ok scratch_conversation_id=%s baseline_agent_id=%s\n' "${scratch_conversation_id}" "${baseline_agent_id}"
