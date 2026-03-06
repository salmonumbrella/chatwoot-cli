#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

ensure_tools
ensure_preflight
load_state

doc_fixture_id="$(require_state DOC_FIXTURE_ID)"
attachments_json="$(cw c attachments "${doc_fixture_id}" -o json --compact-json)"
printf '%s' "${attachments_json}" >"$(artifact_path doc-attachments.json)"

mapfile -t file_entries < <(printf '%s' "${attachments_json}" | jq -r '.items[] | select(.file_type == "file") | @base64' | head -n 3)
[[ "${#file_entries[@]}" -gt 0 ]] || die "no file attachments found for document fixture conversation ${doc_fixture_id}"

summary_jsonl="$(artifact_path doc-characterization.jsonl)"
: >"${summary_jsonl}"

for index in "${!file_entries[@]}"; do
  entry="$(printf '%s' "${file_entries[${index}]}" | base64 --decode)"
  url="$(printf '%s' "${entry}" | jq -r '.data_url')"
  reported_file_size="$(printf '%s' "${entry}" | jq -r '.file_size')"
  download_path="${LIVE_VALIDATION_TMP}/doc-attachment-$((index + 1)).bin"
  header_path="${LIVE_VALIDATION_TMP}/doc-attachment-$((index + 1)).headers"
  meta_path="${LIVE_VALIDATION_TMP}/doc-attachment-$((index + 1)).meta"

  curl -fsSL -D "${header_path}" -o "${download_path}" -w '%{url_effective}\n%{content_type}\n' "${url}" >"${meta_path}"

  effective_url="$(sed -n '1p' "${meta_path}")"
  content_type="$(sed -n '2p' "${meta_path}")"
  url_host="$(printf '%s' "${effective_url}" | sed -E 's#^[A-Za-z]+://([^/]+)/?.*$#\1#')"
  content_disposition="$(awk 'BEGIN{IGNORECASE=1} /^content-disposition:/ {sub(/\r$/, ""); sub(/^[^:]+:[[:space:]]*/, ""); print; exit}' "${header_path}")"
  content_length="$(awk 'BEGIN{IGNORECASE=1} /^content-length:/ {sub(/\r$/, ""); sub(/^[^:]+:[[:space:]]*/, ""); print; exit}' "${header_path}")"
  mime_type="$(file -b --mime-type "${download_path}")"
  file_description="$(file -b "${download_path}")"
  sha256="$(shasum -a 256 "${download_path}" | awk '{print $1}')"

  doc_kind="unsupported"
  extraction_status="not_attempted"
  excerpt=""

  if [[ "${mime_type}" == "application/pdf" ]]; then
    doc_kind="pdf"
    if command -v pdftotext >/dev/null 2>&1; then
      text_path="${LIVE_VALIDATION_TMP}/doc-attachment-$((index + 1)).txt"
      if pdftotext "${download_path}" "${text_path}" >/dev/null 2>&1; then
        extraction_status="pdftotext_ok"
        excerpt="$(tr '\n' ' ' <"${text_path}" | sed 's/[[:space:]]\+/ /g' | cut -c1-200)"
      else
        extraction_status="pdftotext_failed"
      fi
    fi
  elif [[ "${mime_type}" == text/* ]]; then
    doc_kind="text"
    extraction_status="native_text"
    excerpt="$(tr '\n' ' ' <"${download_path}" | sed 's/[[:space:]]\+/ /g' | cut -c1-200)"
  elif [[ "${mime_type}" == "application/zip" || "${mime_type}" == "application/vnd.openxmlformats-officedocument.wordprocessingml.document" ]]; then
    if command -v unzip >/dev/null 2>&1 && unzip -l "${download_path}" 2>/dev/null | grep -q 'word/document.xml'; then
      doc_kind="docx"
    else
      doc_kind="zip"
    fi
  fi

  jq -nc \
    --argjson index "$((index + 1))" \
    --arg url_host "${url_host}" \
    --arg content_type "${content_type}" \
    --arg content_disposition "${content_disposition}" \
    --arg content_length "${content_length}" \
    --arg mime_type "${mime_type}" \
    --arg file_description "${file_description}" \
    --arg doc_kind "${doc_kind}" \
    --arg extraction_status "${extraction_status}" \
    --arg excerpt "${excerpt}" \
    --arg sha256 "${sha256}" \
    --argjson reported_file_size "${reported_file_size}" \
    '{
      sample_index: $index,
      url_host: $url_host,
      content_type: $content_type,
      content_disposition: $content_disposition,
      content_length: $content_length,
      reported_file_size: $reported_file_size,
      detected_mime_type: $mime_type,
      file_description: $file_description,
      detected_kind: $doc_kind,
      extraction_status: $extraction_status,
      excerpt: ($excerpt | select(. != "")),
      sha256: $sha256
    }' >>"${summary_jsonl}"
done

jq -s '{items: .}' "${summary_jsonl}"
