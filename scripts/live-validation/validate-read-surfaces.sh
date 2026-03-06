#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

ensure_tools
ensure_preflight
load_state

text_fixture_id="${TEXT_FIXTURE_ID:-${SCRATCH_CONVERSATION_ID:-}}"
private_fixture_id="${PRIVATE_FIXTURE_ID:-${SCRATCH_CONVERSATION_ID:-}}"
image_fixture_id="${IMAGE_FIXTURE_ID:-}"

[[ -n "${text_fixture_id}" ]] || die "no text fixture available"
[[ -n "${private_fixture_id}" ]] || die "no private-note fixture available"
[[ -n "${image_fixture_id}" ]] || die "no image fixture available"

root_help="$(cw --help-json)"
ctx_help="$(cw ctx --help-json)"
assign_help="$(cw assign --help-json)"
conv_assign_help="$(cw c assign --help-json)"
msg_create_help="$(cw m create --help-json)"
list_help="$(cw c ls --help-json)"
schema_list="$(cw sc ls -o json --compact-json)"
skill_output="$(cw auth skill 2>&1)"
text_ctx_json="$(cw ct "${text_fixture_id}" -o json --compact-json)"
private_ctx_json="$(cw ct "${private_fixture_id}" --tail 5 --public-only -o json --compact-json)"
private_ctx_agent="$(cw ct "${private_fixture_id}" --tail 5 --public-only -o agent)"
image_ctx_json="$(cw ct "${image_fixture_id}" --tail 5 --exclude-attachments -o json --compact-json)"

printf '%s' "${root_help}" >"$(artifact_path help-root.json)"
printf '%s' "${ctx_help}" >"$(artifact_path help-ctx.json)"
printf '%s' "${assign_help}" >"$(artifact_path help-assign.json)"
printf '%s' "${conv_assign_help}" >"$(artifact_path help-conversations-assign.json)"
printf '%s' "${msg_create_help}" >"$(artifact_path help-messages-create.json)"
printf '%s' "${list_help}" >"$(artifact_path help-conversations-list.json)"
printf '%s' "${schema_list}" >"$(artifact_path schema-list.json)"
printf '%s\n' "${skill_output}" >"$(artifact_path auth-skill.txt)"
printf '%s' "${text_ctx_json}" >"$(artifact_path text-context.json)"
printf '%s' "${private_ctx_json}" >"$(artifact_path private-context-public-only.json)"
printf '%s\n' "${private_ctx_agent}" >"$(artifact_path private-context-public-only.agent.txt)"
printf '%s' "${image_ctx_json}" >"$(artifact_path image-context-exclude-attachments.json)"

printf '%s' "${root_help}" | jq -e '.subcommands | any(.name == "assign") and any(.name == "ctx") and any(.name == "conversations")' >/dev/null
printf '%s' "${ctx_help}" | jq -e '
  (.args | length) == 1 and
  .args[0].name == "conversation-id|url" and
  (.flags | any(.name == "tail")) and
  (.flags | any(.name == "public-only")) and
  (.flags | any(.name == "exclude-attachments")) and
  (.flags | any(.name == "embed-images")) and
  (.flags | any(.name == "light"))
' >/dev/null
printf '%s' "${assign_help}" | jq -e '.mutates == true and .supports_dry_run == true and (.args | length) == 1 and .args[0].name == "conversation-id"' >/dev/null
printf '%s' "${conv_assign_help}" | jq -e '.mutates == true and .supports_dry_run == true and (.args | length) == 1 and .args[0].name == "id" and .args[0].variadic == true' >/dev/null
printf '%s' "${msg_create_help}" | jq -e '.mutates == true and .supports_dry_run == true' >/dev/null
printf '%s' "${list_help}" | jq -e '.field_schema == "conversation" and (.field_presets.minimal | length) > 0' >/dev/null
printf '%s' "${schema_list}" | jq -e '(.items | length) > 0' >/dev/null
printf '%s' "${skill_output}" | grep -q 'Generated '
skill_path="$(printf '%s' "${skill_output}" | sed -n 's/^Generated //p' | tail -n 1)"
[[ -n "${skill_path}" && -f "${skill_path}" ]] || die "auth skill did not report a generated skill path"
printf '%s' "${text_ctx_json}" | jq -e '.meta.total_messages >= .meta.returned_messages' >/dev/null
printf '%s' "${private_ctx_json}" | jq -e '.meta.public_only == true and ([.messages[]? | select(.private == true)] | length) == 0' >/dev/null
printf '%s' "${image_ctx_json}" | jq -e '.meta.exclude_attachments == true and ([.messages[]?.attachments[]?] | length) == 0' >/dev/null

printf 'read_surfaces_ok text_fixture_id=%s private_fixture_id=%s private_source=%s image_fixture_id=%s skill_path=%s\n' \
  "${text_fixture_id}" \
  "${private_fixture_id}" \
  "${PRIVATE_FIXTURE_SOURCE:-unknown}" \
  "${image_fixture_id}" \
  "${skill_path}"
