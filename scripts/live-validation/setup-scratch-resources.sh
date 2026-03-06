#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

ensure_tools
ensure_preflight
load_state

scratch_label="cli-live-validation"
scratch_name="CLI Live Validation"
scratch_email="cli-live-validation@example.com"
scratch_identifier="cli-live-validation"

inbox_id="$(maybe_state SCRATCH_INBOX_ID)"
if [[ -z "${inbox_id}" ]]; then
  inbox_id="$(choose_api_inbox_id)"
fi
[[ -n "${inbox_id}" ]] || die "no Channel::Api inbox available for scratch validation"

contact_search_json="$(cw co search "${scratch_identifier}" -o json --compact-json)"
printf '%s' "${contact_search_json}" >"$(artifact_path scratch-contact-search.json)"
contact_id="$(printf '%s' "${contact_search_json}" | jq -r --arg identifier "${scratch_identifier}" --arg email "${scratch_email}" 'first(.items[] | select((.identifier // "") == $identifier or (.email // "") == $email) | .id) // empty')"

if [[ -z "${contact_id}" ]]; then
  create_payload="$(jq -nc --arg name "${scratch_name}" --arg email "${scratch_email}" --arg identifier "${scratch_identifier}" '{name: $name, email: $email, identifier: $identifier, custom_attributes: {cli_validation: true, owner: "codex"}}')"
  contact_json="$(printf '%s' "${create_payload}" | cw co create --json -o json --compact-json)"
else
  contact_json="$(cw co g "${contact_id}" -o json --compact-json)"
fi

contact_id="$(printf '%s' "${contact_json}" | jq -r '.id')"
contact_baseline_body="$(jq -nc --arg name "${scratch_name}" --arg email "${scratch_email}" '{name: $name, email: $email, custom_attributes: {cli_validation: true, owner: "codex"}}')"
cw ap "/contacts/${contact_id}" -X PATCH -d "${contact_baseline_body}" -o json --compact-json >/dev/null
contact_json="$(cw co g "${contact_id}" -o json --compact-json)"
printf '%s' "${contact_json}" >"$(artifact_path scratch-contact.json)"

recent_json="$(cw c ls -a -M 10 -o json --compact-json)"
printf '%s' "${recent_json}" >"$(artifact_path scratch-recent-conversations.json)"
conversation_id="$(printf '%s' "${recent_json}" | jq -r --arg identifier "${scratch_identifier}" --arg email "${scratch_email}" 'first(.items[] | select((.meta.sender.identifier // "") == $identifier or (.meta.sender.email // "") == $email) | .id) // empty')"

if [[ -z "${conversation_id}" ]]; then
  create_conversation_json="$(cw c create -I "${inbox_id}" -C "${contact_id}" -m "CLI live validation scratch conversation" -o json --compact-json)"
  printf '%s' "${create_conversation_json}" >"$(artifact_path scratch-conversation-create.json)"
  conversation_id="$(printf '%s' "${create_conversation_json}" | jq -r '.id')"
fi

agent_ids="$(choose_agent_ids)"
baseline_agent_id="$(printf '%s\n' "${agent_ids}" | awk '{print $1}')"
alternate_agent_id="$(printf '%s\n' "${agent_ids}" | awk '{print $2}')"
[[ -n "${baseline_agent_id}" ]] || die "no agent IDs available for scratch validation"

team_ids="$(choose_team_ids)"
baseline_team_id="$(printf '%s\n' "${team_ids}" | awk '{print $1}')"
alternate_team_id="$(printf '%s\n' "${team_ids}" | awk '{print $2}')"

cw assign "${conversation_id}" --agent "${baseline_agent_id}" --light -o json --compact-json >/dev/null
if [[ -n "${baseline_team_id}" ]]; then
  cw assign "${conversation_id}" --agent "${baseline_agent_id}" --team "${baseline_team_id}" --light -o json --compact-json >/dev/null || true
fi

cw c labels-add "${conversation_id}" --labels "${scratch_label}" -o json --compact-json >/dev/null

scratch_ctx_json="$(cw ct "${conversation_id}" --tail 30 -o json --compact-json)"
printf '%s' "${scratch_ctx_json}" >"$(artifact_path scratch-context.json)"
private_note_count="$(printf '%s' "${scratch_ctx_json}" | jq '[.messages[]? | select(.private == true)] | length')"
if [[ "${private_note_count}" -eq 0 ]]; then
  cw note "${conversation_id}" --content "CLI live validation scratch private note" --light -o json --compact-json >/dev/null
fi

scratch_snapshot="$(conversation_snapshot "${conversation_id}")"
printf '%s' "${scratch_snapshot}" >"$(artifact_path scratch-snapshot.json)"

set_state SCRATCH_LABEL "${scratch_label}"
set_state SCRATCH_CONTACT_ID "${contact_id}"
set_state SCRATCH_CONVERSATION_ID "${conversation_id}"
set_state SCRATCH_INBOX_ID "${inbox_id}"
set_state SCRATCH_IDENTIFIER "${scratch_identifier}"
set_state SCRATCH_EMAIL "${scratch_email}"
set_state SCRATCH_BASELINE_AGENT_ID "${baseline_agent_id}"
set_state SCRATCH_ALTERNATE_AGENT_ID "${alternate_agent_id}"
set_state SCRATCH_BASELINE_TEAM_ID "${baseline_team_id}"
set_state SCRATCH_ALTERNATE_TEAM_ID "${alternate_team_id}"

if [[ -z "${PRIVATE_FIXTURE_ID:-}" ]]; then
  set_state PRIVATE_FIXTURE_ID "${conversation_id}"
  set_state PRIVATE_FIXTURE_SOURCE "scratch"
fi

printf 'scratch_contact_id=%s scratch_conversation_id=%s scratch_inbox_id=%s baseline_agent_id=%s baseline_team_id=%s\n' \
  "${contact_id}" \
  "${conversation_id}" \
  "${inbox_id}" \
  "${baseline_agent_id}" \
  "${baseline_team_id:-none}"

printf 'rollback: restore baseline assignee/team, keep status open, labels [%s], and clear scratch-only custom attributes if added during validation\n' "${scratch_label}"
