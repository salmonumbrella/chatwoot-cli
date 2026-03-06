#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

ensure_tools

status_json="$(cw st -o json --compact-json)"
printf '%s' "${status_json}" >"$(artifact_path preflight.json)"

authenticated="$(printf '%s' "${status_json}" | jq -r '.authenticated')"
[[ "${authenticated}" == "true" ]] || die "Chatwoot CLI is not authenticated"

profile="$(printf '%s' "${status_json}" | jq -r '.profile')"
base_url="$(printf '%s' "${status_json}" | jq -r '.base_url')"
account_id="$(printf '%s' "${status_json}" | jq -r '.account_id')"

set_state AUTH_PROFILE "${profile}"
set_state AUTH_BASE_URL "${base_url}"
set_state AUTH_ACCOUNT_ID "${account_id}"

printf 'authenticated=true profile=%s base_url=%s account_id=%s\n' "${profile}" "${base_url}" "${account_id}"
