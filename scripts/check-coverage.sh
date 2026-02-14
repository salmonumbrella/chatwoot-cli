#!/usr/bin/env bash
set -euo pipefail

threshold="${COVERAGE_MIN:-79.0}"
profile="${COVERAGE_PROFILE:-/tmp/chatwoot-cover-ci.out}"

printf 'Running tests with coverage profile %s\n' "$profile"
go test ./... -coverprofile="$profile" >/dev/null

total_raw="$(go tool cover -func="$profile" | awk '/^total:/{print $3}')"
total="${total_raw%%%}"

printf 'Total coverage: %s%% (minimum %s%%)\n' "$total" "$threshold"

if awk -v got="$total" -v min="$threshold" 'BEGIN { exit !(got + 0 >= min + 0) }'; then
	printf 'Coverage gate passed.\n'
else
	printf 'Coverage gate failed: got %s%%, need at least %s%%.\n' "$total" "$threshold" >&2
	exit 1
fi
