#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

ensure_tools
ensure_preflight
load_state

need_cmd pdftotext

doc_fixture_id="$(require_state DOC_FIXTURE_ID)"

attachments_light_json="$(cw c attachments "${doc_fixture_id}" --li --compact-json)"
pdf_index="$(printf '%s' "${attachments_light_json}" | jq -r '.items[] | select(.t == "file" and (.n | endswith(".pdf"))) | .i' | head -n 1)"
xlsx_index="$(printf '%s' "${attachments_light_json}" | jq -r '.items[] | select(.t == "file" and (.n | endswith(".xlsx"))) | .i' | head -n 1)"

[[ -n "${pdf_index}" ]] || die "no PDF attachment found in document fixture conversation ${doc_fixture_id}"
[[ -n "${xlsx_index}" ]] || die "no XLSX attachment found in document fixture conversation ${doc_fixture_id}"

extract_all_light_json="$(cw c attachments extract "${doc_fixture_id}" --limit 0 --max-chars 800 --li --compact-json)"
extract_pdf_light_json="$(cw c attachments extract "${doc_fixture_id}" --index "${pdf_index}" --max-chars 800 --li --compact-json)"
extract_xlsx_light_json="$(cw c attachments extract "${doc_fixture_id}" --index "${xlsx_index}" --max-chars 800 --li --compact-json)"
extract_agent_json="$(cw c attachments extract "${doc_fixture_id}" --index "${pdf_index}" --max-chars 800 -o agent --compact-json)"

printf '%s' "${attachments_light_json}" >"$(artifact_path doc-attachments-light.json)"
printf '%s' "${extract_all_light_json}" >"$(artifact_path doc-extraction-all-light.json)"
printf '%s' "${extract_pdf_light_json}" >"$(artifact_path doc-extraction-pdf-light.json)"
printf '%s' "${extract_xlsx_light_json}" >"$(artifact_path doc-extraction-xlsx-light.json)"
printf '%s' "${extract_agent_json}" >"$(artifact_path doc-extraction-agent.json)"

printf '%s' "${attachments_light_json}" | jq -e --argjson id "${doc_fixture_id}" '
  .id == $id and
  (.items | length) > 0 and
  all(.items[]; (has("u") | not) and (has("url") | not) and (has("data_url") | not))
' >/dev/null
printf '%s' "${extract_all_light_json}" | jq -e --argjson id "${doc_fixture_id}" '
  .id == $id and
  (.meta.ok // 0) >= 2 and
  ([.items[].x] | unique | index("pdftotext")) != null and
  ([.items[].x] | unique | index("xlsx-xml")) != null
' >/dev/null
printf '%s' "${extract_pdf_light_json}" | jq -e --argjson id "${doc_fixture_id}" '
  .id == $id and
  (.items | length) == 1 and
  .items[0].i == '"${pdf_index}"' and
  .items[0].x == "pdftotext" and
  (.items[0].txt | length) > 0
' >/dev/null
printf '%s' "${extract_xlsx_light_json}" | jq -e --argjson id "${doc_fixture_id}" '
  .id == $id and
  (.items | length) == 1 and
  .items[0].i == '"${xlsx_index}"' and
  .items[0].x == "xlsx-xml" and
  (.items[0].txt | contains("[sheet1.xml]"))
' >/dev/null
printf '%s' "${extract_agent_json}" | jq -e --argjson id "${doc_fixture_id}" '
  .kind == "conversations.attachments.extract" and
  .item.conversation_id == $id and
  (.item.attachments | length) > 0
' >/dev/null

extracted_count="$(printf '%s' "${extract_all_light_json}" | jq -r '.items | length')"
downloaded_bytes="$(printf '%s' "${extract_all_light_json}" | jq -r '.meta.db')"

printf 'doc_extraction_ok doc_fixture_id=%s pdf_index=%s xlsx_index=%s extracted_attachments=%s downloaded_bytes=%s\n' \
  "${doc_fixture_id}" \
  "${pdf_index}" \
  "${xlsx_index}" \
  "${extracted_count}" \
  "${downloaded_bytes}"
