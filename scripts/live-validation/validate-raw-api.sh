#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

ensure_tools
ensure_preflight
load_state

scratch_contact_id="$(require_state SCRATCH_CONTACT_ID)"

baseline_contact="$(contact_snapshot "${scratch_contact_id}")"
printf '%s' "${baseline_contact}" >"$(artifact_path raw-api-contact-baseline.json)"
baseline_name="$(printf '%s' "${baseline_contact}" | jq -r '.name')"
baseline_custom_attributes="$(printf '%s' "${baseline_contact}" | jq -c '.custom_attributes')"

object_body="$(jq -nc --arg name "${baseline_name}" '{name: $name, custom_attributes: {owner: "codex", cli_validation: "raw-api-object"}}')"
object_patch="$(cw ap "/contacts/${scratch_contact_id}" -X PATCH -d "${object_body}" --include -o json --compact-json)"
printf '%s' "${object_patch}" >"$(artifact_path raw-api-object-patch.json)"
printf '%s' "${object_patch}" | jq -e '.status == 200 and .body.payload.custom_attributes.cli_validation == "raw-api-object"' >/dev/null

merged_patch="$(cw ap "/contacts/${scratch_contact_id}" -X PATCH -d "{\"name\":\"${baseline_name}\"}" -f "name=${baseline_name}" -F 'custom_attributes={"owner":"codex","cli_validation":"raw-api-merged","mode":"merged"}' --include -o json --compact-json)"
printf '%s' "${merged_patch}" >"$(artifact_path raw-api-merged-patch.json)"
printf '%s' "${merged_patch}" | jq -e '.status == 200 and .body.payload.custom_attributes.cli_validation == "raw-api-merged" and .body.payload.custom_attributes.mode == "merged"' >/dev/null

oversize_file="$(mktemp "${LIVE_VALIDATION_TMP}/oversize.XXXXXX")"
{
  printf '{"payload":"'
  head -c 1048580 </dev/zero | tr '\0' 'a'
  printf '"}'
} >"${oversize_file}"
set +e
oversize_output="$(cw ap "/contacts/${scratch_contact_id}" -X PATCH -i "${oversize_file}" -o json --compact-json 2>&1)"
oversize_status=$?
set -e
printf '%s\n' "${oversize_output}" >"$(artifact_path raw-api-oversize.txt)"
[[ "${oversize_status}" -ne 0 ]] || die "oversized raw API input unexpectedly succeeded"
printf '%s' "${oversize_output}" | grep -q 'JSON payload exceeds maximum size'

set +e
invalid_merge_output="$(cw ap "/contacts/${scratch_contact_id}" -X PATCH -d '["bad"]' -f "name=${baseline_name}" -o json --compact-json 2>&1)"
invalid_merge_status=$?
set -e
printf '%s\n' "${invalid_merge_output}" >"$(artifact_path raw-api-invalid-merge.txt)"
[[ "${invalid_merge_status}" -ne 0 ]] || die "non-object merge unexpectedly succeeded"
printf '%s' "${invalid_merge_output}" | grep -q 'cannot use --field or --raw-field with a non-object JSON body'

restore_body="$(jq -nc --arg name "${baseline_name}" --argjson attrs "${baseline_custom_attributes}" '{name: $name, custom_attributes: $attrs}')"
restore_patch="$(cw ap "/contacts/${scratch_contact_id}" -X PATCH -d "${restore_body}" --include -o json --compact-json)"
printf '%s' "${restore_patch}" >"$(artifact_path raw-api-restore.json)"
printf '%s' "${restore_patch}" | jq -e --arg name "${baseline_name}" --argjson attrs "${baseline_custom_attributes}" '.status == 200 and .body.payload.name == $name and (.body.payload.custom_attributes == $attrs)' >/dev/null

printf 'raw_api_ok scratch_contact_id=%s\n' "${scratch_contact_id}"
