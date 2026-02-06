# Agent-Friendly CLI Surface Refactor Plan

Date started: 2026-02-06

This file tracks the incremental “surface level” refactor to make `chatwoot` maximally agent-friendly: desire paths, flat access by ID/URL, low-friction discovery, and predictable agent output.

## Phases

### Phase 1: Desire-Path Shortcuts (Completed)
- Add top-level shortcuts: `comment`, `note`, `close`, `reopen`, `ctx`.
- Add `get`/`show` aliases for `open`.
- Tests for new commands.

### Phase 2: ID Normalization (Completed)
- Extend `parseIDOrURL` to accept `#123` and `conv:123` / `conversation:123`.
- Update `conversations context` to accept pasted URLs.
- Tests.

### Phase 3: Name Resolution For Assignment (Completed)
- Allow `--agent` and `--team` to accept IDs or names/emails for:
  - `chatwoot assign`
  - `chatwoot conversations assign`
- Tests.

### Phase 4: Remaining Gaps (Completed)
#### Phase 4.1: Universal ID/URL Acceptance (Completed)
- Replace remaining direct numeric parsing so conversation/contact/inbox IDs accept:
  - plain IDs: `123`
  - hash IDs: `#123`
  - prefixed IDs: `conv:123`, `conversation:123`, `contact:456`
  - pasted UI URLs: `https://app.chatwoot.com/app/accounts/1/conversations/123`
- Ensure multi-ID commands accept the same formats:
  - space-separated args: `chatwoot close 1 2 3`
  - comma-separated args: `chatwoot close 1,2,3`
  - `--ids` flags: `--ids 1,2,3`
- Add shorthand support for secondary IDs (message/note) via `#456` and `message:456`.

#### Phase 4.2: Bulk Assign Name Resolution (Completed)
- `chatwoot conversations bulk assign` should accept agent/team by:
  - numeric ID: `--agent 5`
  - name: `--agent "Jane Doe"`
  - email: `--agent jane@example.com`
- Keep backwards-compat flags (`--agent-id`, `--team-id`) but prefer `--agent`, `--team`.

#### Phase 4.3: Agent Error Envelope (Completed)
- Standardize agent-mode errors to:
  - `{ "kind": "...", "error": { "code": "...", "message": "...", ... } }`
- Keep existing JSON error payload unchanged for `--output json`.

#### Phase 4.4: Agent Output URL Metadata (Completed)
- Add `url` metadata more broadly to agent outputs (items + meta) for easy follow-up operations.
- Prefer returning a fully-qualified Chatwoot UI URL when possible.

## Work Log

### 2026-02-06
- Phase 1–3 implemented (see `docs/work/agent-surface-refactor-2026-02-06.md` for details).
- Phase 4 implemented:
  - Multi-ID parsing (`close`, `reopen`, `resolve`, `conversations resolve`) now accepts `#123`, `conv:123`, and UI URLs.
  - Bulk conversation operations (`conversations bulk resolve|assign|add-label`) now accept URL/prefix/hash IDs in `--ids`.
  - Error messages from `parseIDOrURL` include the resource label (e.g. "invalid inbox ID") to preserve UX and test expectations.
  - `conversations bulk assign` supports `--agent/--team` name/email/ID resolution (and keeps `--agent-id/--team-id` as hidden, deprecated flags).
  - Agent output now includes fully-qualified Chatwoot UI URLs on common items (conversation/contact) when configured.
  - Agent-mode failures are standardized into an error envelope `{kind, error}` for reliable agent handling.
