#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readme="$repo_root/README.md"
root_go="$repo_root/internal/cmd/root.go"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

cd "$repo_root"

go run ./cmd/chatwoot --help-json >"$tmpdir/help.json"

# Expected top-level command aliases from live command metadata.
jq -r '.subcommands[] | [.name, ((.aliases // []) | join(", "))] | @tsv' "$tmpdir/help.json" \
  | awk -F'\t' '{ if ($2 == "") $2="-"; print $1 "\t" $2 }' \
  | sort >"$tmpdir/expected_cmd.tsv"

# README command alias table (between "## Command Aliases" and "Subcommands also have aliases").
awk '
  /^## Command Aliases$/ { in_section=1; in_table=0; next }
  in_section && /^Subcommands also have aliases/ { in_section=0; next }
  in_section && /^\|/ {
    in_table=1
    if ($0 ~ /^\| `[^`]+` \|/) {
      split($0, c, "|")
      cmd = c[2]
      aliases = c[3]
      gsub(/^ +| +$/, "", cmd)
      gsub(/^ +| +$/, "", aliases)
      gsub(/`/, "", cmd)
      gsub(/`/, "", aliases)
      if (aliases == "") aliases = "-"
      print cmd "\t" aliases
    }
    next
  }
  in_section && in_table && !/^\|/ { in_section=0 }
' "$readme" | sort >"$tmpdir/readme_cmd.tsv"

if ! diff -u "$tmpdir/expected_cmd.tsv" "$tmpdir/readme_cmd.tsv" >"$tmpdir/cmd_alias.diff"; then
  echo "README command alias table is out of sync with CLI aliases." >&2
  cat "$tmpdir/cmd_alias.diff" >&2
  exit 1
fi

# Expected root persistent flag aliases from source.
grep -E 'flagAlias\(root\.PersistentFlags\(\), "' "$root_go" \
  | sed -E 's/.*flagAlias\(root\.PersistentFlags\(\), "([^"]+)", "([^"]+)".*/--\1\t--\2/' \
  | sort -u >"$tmpdir/expected_global.tsv"

# Documented root global alias mappings from README "Global Flag Aliases" table.
awk '
  function extract_backticked(cell, canonical,    s, token) {
    s = cell
    while (match(s, /`[^`]+`/)) {
      token = substr(s, RSTART+1, RLENGTH-2)
      if (token ~ /^--/) {
        print canonical "\t" token
      }
      s = substr(s, RSTART + RLENGTH)
    }
  }
  /^### Global Flag Aliases$/ { in_section=1; in_table=0; next }
  /^## Command Aliases$/ { in_section=0; next }
  in_section && /^\|/ {
    in_table=1
    if ($0 ~ /^\| `--[^`]+` \|/) {
      split($0, c, "|")
      canonical_cell = c[2]
      alias_cell = c[3]
      canonical = ""
      if (match(canonical_cell, /`--[^`]+`/)) {
        canonical = substr(canonical_cell, RSTART+1, RLENGTH-2)
      }
      if (canonical != "") {
        extract_backticked(alias_cell, canonical)
      }
    }
    next
  }
  in_section && in_table && !/^\|/ { in_section=0 }
' "$readme" | sort -u >"$tmpdir/readme_global.tsv"

missing=0
while IFS=$'\t' read -r canonical alias; do
  if ! grep -Fqx -- "$canonical"$'\t'"$alias" "$tmpdir/readme_global.tsv"; then
    echo "README Global Flag Aliases is missing: $canonical => $alias" >&2
    missing=1
  fi
done <"$tmpdir/expected_global.tsv"

if [[ "$missing" -ne 0 ]]; then
  exit 1
fi

echo "README alias checks passed."
