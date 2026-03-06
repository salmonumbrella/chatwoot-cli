#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

ensure_tools
ensure_preflight
load_state

scratch_conversation_id="$(require_state SCRATCH_CONVERSATION_ID)"
scratch_label="$(require_state SCRATCH_LABEL)"
baseline_agent_id="${SCRATCH_BASELINE_AGENT_ID:-}"
alternate_agent_id="${SCRATCH_ALTERNATE_AGENT_ID:-}"

baseline_snapshot="$(conversation_snapshot "${scratch_conversation_id}")"
printf '%s' "${baseline_snapshot}" >"$(artifact_path mutation-baseline.json)"

baseline_agent_id="${baseline_agent_id:-$(printf '%s' "${baseline_snapshot}" | jq -r '.assignee_id // empty')}"
[[ -n "${baseline_agent_id}" ]] || die "scratch baseline assignee is missing"

if [[ -z "${alternate_agent_id}" || "${alternate_agent_id}" == "${baseline_agent_id}" ]]; then
  alternate_agent_id="$(printf '%s\n' "$(choose_agent_ids)" | awk -v current="${baseline_agent_id}" '{for (i = 1; i <= NF; i++) if ($i != current) {print $i; exit}}')"
fi
[[ -n "${alternate_agent_id}" ]] || die "no alternate agent available for scratch mutation validation"

baseline_attrs_json="$(printf '%s' "${baseline_snapshot}" | jq -c '.custom_attributes')"
baseline_labels_json="$(printf '%s' "${baseline_snapshot}" | jq -c '.labels')"
baseline_labels_csv="$(printf '%s' "${baseline_labels_json}" | jq -r 'join(",")')"

cw assign "${scratch_conversation_id}" --agent "${alternate_agent_id}" --light -o json --compact-json >"$(artifact_path mutation-assign.json)"
assign_snapshot="$(conversation_snapshot "${scratch_conversation_id}")"
printf '%s' "${assign_snapshot}" >"$(artifact_path mutation-after-assign.json)"
printf '%s' "${assign_snapshot}" | jq -e --argjson agent "${alternate_agent_id}" '.assignee_id == $agent' >/dev/null

cw assign "${scratch_conversation_id}" --agent "${baseline_agent_id}" --light -o json --compact-json >/dev/null
restore_assign_snapshot="$(conversation_snapshot "${scratch_conversation_id}")"
printf '%s' "${restore_assign_snapshot}" >"$(artifact_path mutation-after-assign-restore.json)"
printf '%s' "${restore_assign_snapshot}" | jq -e --argjson agent "${baseline_agent_id}" '.assignee_id == $agent' >/dev/null

cw close "${scratch_conversation_id}" --light -o json --compact-json >"$(artifact_path mutation-close.json)"
closed_snapshot="$(conversation_snapshot "${scratch_conversation_id}")"
printf '%s' "${closed_snapshot}" >"$(artifact_path mutation-after-close.json)"
printf '%s' "${closed_snapshot}" | jq -e '.status == "resolved"' >/dev/null

cw reopen "${scratch_conversation_id}" --light -o json --compact-json >"$(artifact_path mutation-reopen.json)"
reopened_snapshot="$(conversation_snapshot "${scratch_conversation_id}")"
printf '%s' "${reopened_snapshot}" >"$(artifact_path mutation-after-reopen.json)"
printf '%s' "${reopened_snapshot}" | jq -e '.status == "open"' >/dev/null

cw c ca "${scratch_conversation_id}" --set cli_validation=active -o json --compact-json >"$(artifact_path mutation-custom-attributes.json)"
attrs_snapshot="$(conversation_snapshot "${scratch_conversation_id}")"
printf '%s' "${attrs_snapshot}" >"$(artifact_path mutation-after-custom-attributes.json)"
printf '%s' "${attrs_snapshot}" | jq -e '.custom_attributes.cli_validation == "active"' >/dev/null

restore_attrs_body="$(jq -nc --argjson attrs "${baseline_attrs_json}" '{custom_attributes: $attrs}')"
cw ap "/conversations/${scratch_conversation_id}/custom_attributes" -X POST -d "${restore_attrs_body}" -o json --compact-json >/dev/null
attrs_restored_snapshot="$(conversation_snapshot "${scratch_conversation_id}")"
printf '%s' "${attrs_restored_snapshot}" >"$(artifact_path mutation-after-custom-attributes-restore.json)"
[[ "$(printf '%s' "${attrs_restored_snapshot}" | jq -c -S '.custom_attributes')" == "$(printf '%s' "${baseline_attrs_json}" | jq -c -S '.')" ]] || die "custom attributes were not restored"

temp_label="${scratch_label}-mutation"
cw c labels-add "${scratch_conversation_id}" --labels "${temp_label}" -o json --compact-json >"$(artifact_path mutation-labels-add.json)"
labels_added_snapshot="$(conversation_snapshot "${scratch_conversation_id}")"
printf '%s' "${labels_added_snapshot}" >"$(artifact_path mutation-after-labels-add.json)"
printf '%s' "${labels_added_snapshot}" | jq -e --arg label "${temp_label}" '.labels | index($label) != null' >/dev/null

if [[ -n "${baseline_labels_csv}" ]]; then
  cw c labels-add "${scratch_conversation_id}" --labels "${baseline_labels_csv}" -o json --compact-json >"$(artifact_path mutation-labels-remove.json)"
else
  cw c labels-remove "${scratch_conversation_id}" --labels "${temp_label}" -o json --compact-json >"$(artifact_path mutation-labels-remove.json)"
fi
cw assign "${scratch_conversation_id}" --agent "${baseline_agent_id}" --light -o json --compact-json >/dev/null
labels_restored_snapshot="$(conversation_snapshot "${scratch_conversation_id}")"
printf '%s' "${labels_restored_snapshot}" >"$(artifact_path mutation-after-labels-restore.json)"
[[ "$(printf '%s' "${labels_restored_snapshot}" | jq -c -S '.labels')" == "$(printf '%s' "${baseline_labels_json}" | jq -c -S '.')" ]] || die "labels were not restored"

final_snapshot="$(conversation_snapshot "${scratch_conversation_id}")"
printf '%s' "${final_snapshot}" >"$(artifact_path mutation-final.json)"
[[ "$(printf '%s' "${final_snapshot}" | jq -c -S '.')" == "$(printf '%s' "${baseline_snapshot}" | jq -c -S '.')" ]] || die "scratch conversation did not return to baseline state"

printf 'scratch_mutations_ok scratch_conversation_id=%s baseline_agent_id=%s alternate_agent_id=%s\n' \
  "${scratch_conversation_id}" \
  "${baseline_agent_id}" \
  "${alternate_agent_id}"
