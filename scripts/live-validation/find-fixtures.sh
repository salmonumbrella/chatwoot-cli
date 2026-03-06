#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

ensure_tools
ensure_preflight

pages="${1:-5}"

list_json="$(cw c ls -a -M "${pages}" -o json --compact-json)"
printf '%s' "${list_json}" >"$(artifact_path recent-conversations.json)"
: >"$(artifact_path fixture-scan.jsonl)"

mapfile -t conversation_ids < <(printf '%s' "${list_json}" | jq -r '.items[].id')

text_id=""
private_id=""
image_id=""
doc_id=""
image_fallback_id=""
doc_fallback_id=""
best_image_attachment_count=""
best_image_returned_messages=""
best_doc_attachment_count=""
best_doc_returned_messages=""

for conversation_id in "${conversation_ids[@]}"; do
  sender_name="$(printf '%s' "${list_json}" | jq -r --argjson id "${conversation_id}" 'first(.items[] | select(.id == $id) | .meta.sender.name) // ""')"
  inbox_id="$(printf '%s' "${list_json}" | jq -r --argjson id "${conversation_id}" 'first(.items[] | select(.id == $id) | .inbox_id) // 0')"
  messages_count="$(printf '%s' "${list_json}" | jq -r --argjson id "${conversation_id}" 'first(.items[] | select(.id == $id) | .messages_count) // 0')"

  transcript_json="$(cw c transcript "${conversation_id}" --limit 30 -M 10 -o json --compact-json 2>/dev/null || true)"
  [[ -n "${transcript_json}" ]] || continue

  returned_messages="$(printf '%s' "${transcript_json}" | jq '.meta.message_count // (.messages | length)')"
  private_count="$(printf '%s' "${transcript_json}" | jq '[.messages[]? | select(.private == true)] | length')"
  ctx_attachment_count="$(printf '%s' "${transcript_json}" | jq '[.messages[]?.attachments[]?] | length')"

  attachment_types=""
  attachment_count=0
  if [[ -z "${image_id}" || -z "${doc_id}" ]]; then
    attachments_json="$(cw c attachments "${conversation_id}" -o json --compact-json 2>/dev/null || true)"
    if [[ -n "${attachments_json}" ]]; then
      attachment_types="$(printf '%s' "${attachments_json}" | jq -r '[.items[]?.file_type] | unique | join(",")')"
      attachment_count="$(printf '%s' "${attachments_json}" | jq '.items | length')"
    fi
  fi

  jq -nc \
    --argjson id "${conversation_id}" \
    --arg sender_name "${sender_name}" \
    --argjson inbox_id "${inbox_id}" \
    --argjson messages_count "${messages_count}" \
    --argjson returned_messages "${returned_messages}" \
    --argjson private_count "${private_count}" \
    --argjson ctx_attachment_count "${ctx_attachment_count}" \
    --arg attachment_types "${attachment_types}" \
    --argjson attachment_count "${attachment_count}" \
    '{
      id: $id,
      sender_name: $sender_name,
      inbox_id: $inbox_id,
      messages_count: $messages_count,
      returned_messages: $returned_messages,
      private_count: $private_count,
      ctx_attachment_count: $ctx_attachment_count,
      attachment_types: ($attachment_types | select(. != "")),
      attachment_count: $attachment_count
    }' >>"$(artifact_path fixture-scan.jsonl)"

  if [[ -z "${text_id}" && "${private_count}" -eq 0 && "${ctx_attachment_count}" -eq 0 && "${returned_messages}" -gt 0 ]]; then
    text_id="${conversation_id}"
  fi

  if [[ -z "${private_id}" && "${private_count}" -gt 0 ]]; then
    private_id="${conversation_id}"
  fi

  if [[ ",${attachment_types}," == *",image,"* ]]; then
    if [[ -z "${image_fallback_id}" ]]; then
      image_fallback_id="${conversation_id}"
    fi
    if [[ -z "${image_id}" || "${attachment_count}" -lt "${best_image_attachment_count:-999999999}" || ( "${attachment_count}" -eq "${best_image_attachment_count:-999999999}" && "${returned_messages}" -lt "${best_image_returned_messages:-999999999}" ) ]]; then
      image_id="${conversation_id}"
      best_image_attachment_count="${attachment_count}"
      best_image_returned_messages="${returned_messages}"
    fi
  fi

  if [[ ",${attachment_types}," == *",file,"* ]]; then
    if [[ -z "${doc_fallback_id}" ]]; then
      doc_fallback_id="${conversation_id}"
    fi
    if [[ -z "${doc_id}" || "${attachment_count}" -lt "${best_doc_attachment_count:-999999999}" || ( "${attachment_count}" -eq "${best_doc_attachment_count:-999999999}" && "${returned_messages}" -lt "${best_doc_returned_messages:-999999999}" ) ]]; then
      doc_id="${conversation_id}"
      best_doc_attachment_count="${attachment_count}"
      best_doc_returned_messages="${returned_messages}"
    fi
  fi
done

if [[ -z "${image_id}" ]]; then
  image_id="${image_fallback_id}"
fi
if [[ -z "${doc_id}" ]]; then
  doc_id="${doc_fallback_id}"
fi

if [[ -n "${text_id}" ]]; then
  set_state TEXT_FIXTURE_ID "${text_id}"
fi
if [[ -n "${private_id}" ]]; then
  set_state PRIVATE_FIXTURE_ID "${private_id}"
  set_state PRIVATE_FIXTURE_SOURCE "live"
fi
if [[ -n "${image_id}" ]]; then
  set_state IMAGE_FIXTURE_ID "${image_id}"
fi
if [[ -n "${doc_id}" ]]; then
  set_state DOC_FIXTURE_ID "${doc_id}"
fi
set_state FIXTURE_SCAN_PAGES "${pages}"

printf 'category\tconversation_id\tstatus\n'
printf 'text\t%s\t%s\n' "${text_id:-not_found}" "$([[ -n "${text_id}" ]] && printf found || printf not_found)"
printf 'private\t%s\t%s\n' "${private_id:-not_found}" "$([[ -n "${private_id}" ]] && printf found || printf not_found)"
printf 'image\t%s\t%s\n' "${image_id:-not_found}" "$([[ -n "${image_id}" ]] && printf found || printf not_found)"
printf 'document\t%s\t%s\n' "${doc_id:-not_found}" "$([[ -n "${doc_id}" ]] && printf found || printf not_found)"

jq -nc \
  --arg text "${text_id}" \
  --arg private "${private_id}" \
  --arg image "${image_id}" \
  --arg document "${doc_id}" \
  '{text_fixture_id: ($text | select(. != "")), private_fixture_id: ($private | select(. != "")), image_fixture_id: ($image | select(. != "")), document_fixture_id: ($document | select(. != ""))}'
