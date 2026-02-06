# Chatwoot CLI: Agent-Friendliness Plan

Date: 2026-02-06

Goal: make `chatwoot` the lowest-friction, most predictable CLI possible for LLM agents and automation.

This plan is intentionally biased toward:
- desire paths (commands/flags agents naturally try)
- flat access (act by ID/URL immediately, avoid hierarchy traversal)
- a strict IO + error contract (stdout is data, errors are structured in machine modes)

Related tracking docs:
- `plan.md` (repo-level incremental tracking)
- `docs/work/agent-surface-refactor-2026-02-06.md` (work log + acceptance criteria)
- `docs/AGENT_INTEGRATION.md` (how agents should use the CLI)

## Principles (What We Will Optimize For)

- One-ID operations: for common tasks, the agent should be able to do the thing with a single identifier.
  Examples: `chatwoot comment 123 "..."`, `chatwoot close 123`.
- Accept the identifiers agents have: numeric IDs, `#123`, `conv:123`, and pasted Chatwoot UI URLs.
- Predictable machine output:
  - `--output json|jsonl|agent` never prints progress/noise to stdout.
  - errors in machine output are structured and include `code`, `suggestion`, and request context when available.
- Desire-path aliases: implement the obvious synonyms and short forms (pluralization, `get/show`, etc).

## Work Items (Next)

1. `--yes/-y` desire path (global)
  - Add a global `--yes/-y` flag as an alias for “assume yes / skip confirmations”.
  - In JSON mode, `--yes` should satisfy “require force for JSON” checks (equivalent to `--force`).
  - `--yes` should also imply non-interactive mode (same effect as `--no-input`).

2. Universal ID shorthand support
  - Make `validation.ParsePositiveInt` accept `#123` (many commands still use this parser).
  - Ensure resolver helpers (`resolveInboxID`, `resolveTeamID`, `resolveAgentID`, `resolveContactID`) accept:
    - numeric IDs, `#123`, `resource:123`
    - pasted UI URLs when applicable (`/app/accounts/...`)

3. Close remaining “hierarchy taxes”
  - Prefer flags that accept names where agents naturally have names (`--inbox-id Support` already works in some places).
  - For mutation commands that still require multiple IDs, add an alternate “flat” entrypoint when it can be done safely/reliably.

## Acceptance Checks

- `go test ./...` passes.
- In machine modes (`--output json|agent`):
  - stdout is valid JSON for successful commands
  - stdout is valid JSON for errors (via `RunE` structured errors)
  - no prompts appear
- Desire paths:
  - `chatwoot --yes ...` works for commands that otherwise require `--force` in JSON mode
  - `chatwoot inboxes get #123` works
  - `chatwoot conversations bulk assign --ids conv:1,https://.../conversations/2 --agent "Agent"` works

