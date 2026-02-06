# Chatwoot CLI Improvements Plan (Post Fuzzy Resolve + Cache)

This plan captures the 7 follow-up improvements discussed after landing fuzzy resolve + caching. Each item is independent, but they compose well.

## 1) Interactive Disambiguation (TTY-Only)

Goal: When a name query is ambiguous, let an agent pick a match interactively instead of failing.

User experience:
- If stdout is a TTY and output mode is text/agent, show a numbered shortlist and prompt.
- If non-interactive (piped output, `--output json/jsonl`, `--quiet`, `--silent`), keep current behavior: return an error listing candidates.

Implementation:
- Add a small helper `isInteractive(cmd)` using `golang.org/x/term` or existing IO context.
- Extend `internal/resolve` with an option to return top N candidates for display.
- Update `internal/cmd/helpers.go` matchers to:
  - Detect `*resolve.AmbiguousError`.
  - Prompt selection when interactive.

Tests:
- Unit: resolve ambiguity path returns candidates.
- Cmd: simulate TTY vs non-TTY (or inject an `io.Reader`/`isatty` function) and verify behavior.

## 2) `chatwoot cache refresh` (Prewarm)

Goal: One command to prefetch and cache the common resolver datasets so the first “name resolve” is instant.

User experience:
- `chatwoot cache refresh` fetches `inboxes`, `agents`, `teams` and writes caches.
- Support `--account-id/--server` overrides indirectly through existing auth/config, same as other commands.

Implementation:
- Add `internal/cmd/cache_cmd_refresh.go`:
  - Fetch inboxes/agents/teams.
  - Store via `internal/cache`.
  - Print summary (counts, cache paths) in text mode; structured JSON in JSON mode.

Tests:
- Command test with a route handler asserting each list endpoint hit once.
- Verifies cache files created.

## 3) Per-Command Cache Controls (`--no-cache`, optional `--cache-ttl`)

Goal: Give agents a quick escape hatch (freshness over speed) and advanced control for power users.

User experience:
- Global flag: `--no-cache` disables cache reads/writes for the current invocation.
- Optional: `--cache-ttl 30s` overrides TTL for reads (writes still store `cached_at`).

Implementation:
- Add root flag(s) in `internal/cmd/root.go`.
- Thread through via context or global flags:
  - `resolveCacheDir()` remains the same.
  - Cache `disabled()` also checks runtime flag (in addition to env).
  - TTL override passed to `cache.NewStoreWithTTL`.

Tests:
- Ensure `--no-cache` prevents file creation even when `CHATWOOT_CACHE_DIR` is set.
- TTL override test for “treat as stale”.

## 4) Better Matching Inputs (Agents + Inboxes)

Goal: Make name resolution feel “obvious” and forgiving, like kubectl resource selection.

User experience:
- Agents:
  - Match by name, full email, and email local-part.
  - Ambiguity list shows `Name <email>`.
- Inboxes:
  - Optionally include channel type or other hint text in match display.
  - Keep matching based on inbox name, but show better candidates.

Implementation:
- Update `fuzzyMatchAgents` to build candidates from:
  - `Name`
  - `Email`
  - `local-part` (already used as fallback; promote into candidates or preserve as fallback but improve ranking).
- Update ambiguity formatting helpers to show richer lines.

Tests:
- Agent match by email local-part.
- Ambiguity error contains `Name <email>`.

## 5) Cache Introspection (`cache ls`, targeted clears)

Goal: Let agents and developers see what is cached, and surgically clear a single resource type.

User experience:
- `chatwoot cache ls` shows:
  - resource key, account id, base url hash suffix, age, size
- `chatwoot cache clear` (existing) clears all.
- Add: `chatwoot cache clear inboxes` (resource-specific), and/or `--key inboxes`.

Implementation:
- Extend `internal/cache` with:
  - `List(dir) ([]EntryInfo, error)` returning filename, size, mtime/age, parsed key/account/suffix if possible.
  - `ClearKey(dir, key, baseURL, accountID)` or `ClearByPrefix(dir, "inboxes_")`.
- Implement `cache ls` + enhanced `cache clear`.

Tests:
- Creates a couple cache files, verifies `ls` output includes them.
- Resource-specific clear only deletes expected file(s).

## 6) Concurrency Hardening (Advisory Lock per Cache Entry)

Goal: Avoid multi-process races (two CLIs running concurrently) causing partial writes or “last writer wins” surprises.

Implementation options:
- Simple lock file: `inboxes_<suffix>_<acct>.lock` using `O_EXCL` create + stale lock TTL.
- Or use an existing small library, but prefer no deps.
- Keep atomic write via temp+rename, but wrap reads/writes with lock.

Tests:
- Concurrency test that runs `Put` in parallel and verifies file remains valid JSON.

## 7) Debug Visibility (`CHATWOOT_DEBUG_CACHE=1` or debug logger)

Goal: When agents/devs suspect the cache is “wrong”, make it explain itself.

User experience:
- When enabled, log cache actions to stderr:
  - hit/miss/stale
  - key + path
  - resolve outcome: exact/fuzzy/ambiguous

Implementation:
- Add logging hooks in `internal/cache` (or at call sites in resolvers) so `cache` remains quiet by default.
- Prefer existing `internal/debug` logger if it fits.

Tests:
- Not strictly required, but add a small test around “debug enabled emits something” if we want stability.

---

## (Separate Track) Gogcli-Inspired “Watch” for Agent Workflows

This is not one of the 7 items above, but it’s the next big workflow win for agents.

Inspiration to borrow from `gog gmail watch`:
- Clear CLI surface: `start/status/renew/stop/serve` (where it maps).
- Persistent state (what you’re watching, last-seen cursor, hook config).
- Dedupe and “stale cursor” recovery.
- Secure inbound events on the `serve` endpoint (OIDC/shared token), when push is used.

Chatwoot-specific implementation idea:
- Provide `chatwoot conversations follow <conversation>` to tail messages.
- Default to WebSocket streaming if available (Chatwoot `/cable` + `pubsub_token` from Profile API).
- Fallback to polling `GET /conversations/:id/messages` (use a “recent window” limit and filter by last-seen message ID).
- Optionally add `chatwoot watch serve` that receives Chatwoot webhook events (`message_created`) and prints/forwards them, mirroring gog’s push handler design.

