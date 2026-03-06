#!/usr/bin/env bash

set -euo pipefail

LIVE_VALIDATION_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
LIVE_VALIDATION_TMP="${TMPDIR:-/tmp}/chatwoot-cli-live-validation"
LIVE_VALIDATION_STATE="${LIVE_VALIDATION_TMP}/state.env"
LIVE_VALIDATION_ARTIFACTS="${LIVE_VALIDATION_TMP}/artifacts"

mkdir -p "${LIVE_VALIDATION_TMP}" "${LIVE_VALIDATION_ARTIFACTS}"

log() {
  printf '[live-validation] %s\n' "$*" >&2
}

die() {
  printf '[live-validation] ERROR: %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

ensure_tools() {
  need_cmd jq
  need_cmd curl
  need_cmd file
}

cw() {
  (
    cd "${LIVE_VALIDATION_ROOT}"
    go run ./cmd/chatwoot "$@"
  )
}

load_state() {
  if [[ -f "${LIVE_VALIDATION_STATE}" ]]; then
    # shellcheck disable=SC1090
    source "${LIVE_VALIDATION_STATE}"
  fi
}

set_state() {
  local key="$1"
  local value="$2"
  local tmp

  mkdir -p "${LIVE_VALIDATION_TMP}"
  touch "${LIVE_VALIDATION_STATE}"
  tmp="$(mktemp "${LIVE_VALIDATION_TMP}/state.XXXXXX")"
  grep -v -E "^${key}=" "${LIVE_VALIDATION_STATE}" >"${tmp}" || true
  printf '%s=%q\n' "${key}" "${value}" >>"${tmp}"
  mv "${tmp}" "${LIVE_VALIDATION_STATE}"
  export "${key}=${value}"
}

maybe_state() {
  load_state
  printf '%s' "${!1:-}"
}

require_state() {
  load_state
  [[ -n "${!1:-}" ]] || die "missing required state: ${1}"
  printf '%s' "${!1}"
}

artifact_path() {
  printf '%s/%s' "${LIVE_VALIDATION_ARTIFACTS}" "$1"
}

save_artifact() {
  local name="$1"
  cat >"$(artifact_path "${name}")"
}

ensure_preflight() {
  if [[ -z "$(maybe_state AUTH_PROFILE)" ]]; then
    "${LIVE_VALIDATION_ROOT}/scripts/live-validation/preflight.sh" >/dev/null
  fi
}

choose_api_inbox_id() {
  local inboxes_json
  inboxes_json="$(cw in ls -o json --compact-json)"
  printf '%s' "${inboxes_json}" >"$(artifact_path inboxes.json)"
  printf '%s' "${inboxes_json}" | jq -r 'first(.items[] | select(.channel_type == "Channel::Api") | .id) // empty'
}

choose_agent_ids() {
  local agents_json
  agents_json="$(cw a ls -o json --compact-json)"
  printf '%s' "${agents_json}" >"$(artifact_path agents.json)"
  printf '%s' "${agents_json}" | jq -r '[.items[] | select((.name // "") != "Bot") | .id] | @tsv'
}

choose_team_ids() {
  local teams_json
  teams_json="$(cw teams list -o json --compact-json)"
  printf '%s' "${teams_json}" >"$(artifact_path teams.json)"
  printf '%s' "${teams_json}" | jq -r '[.items[].id] | @tsv'
}

conversation_snapshot() {
  local conversation_id="$1"
  cw c g "${conversation_id}" -o json --compact-json | jq '{
    id,
    status,
    assignee_id: (.meta.assignee.id // null),
    assignee_name: (.meta.assignee.name // null),
    team_id: (.team_id // null),
    labels: (.labels // []),
    custom_attributes: (.custom_attributes // {})
  }'
}

contact_snapshot() {
  local contact_id="$1"
  cw co g "${contact_id}" -o json --compact-json | jq '{
    id,
    name,
    email,
    identifier,
    custom_attributes: (.custom_attributes // {})
  }'
}
