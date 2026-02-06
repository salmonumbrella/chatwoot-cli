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

### Phase 5: Agent Workflow Power-Ups (Completed)
#### Phase 5.1: Piping-Friendly Bulk IDs (Completed)
- Make every `--ids` flag accept:
  - `@-` to read IDs from stdin
  - `@path` to read IDs from a file
- Accepted formats from stdin/file:
  - CSV: `1,2,3`
  - whitespace/newlines: `1 2 3` / `1\n2\n3`
  - JSON array: `[1,2,"#3","conv:4","https://.../conversations/5"]`
- Keep existing comma-separated behavior unchanged.

#### Phase 5.2: Non-Interactive “Best Match” Search (Completed)
- Add `chatwoot search --best` to auto-select the best result (no TTY required).
- Add `--emit id|url|json` to avoid `jq` for common chaining cases.

#### Phase 5.3: Tighten Agent-Mode Noise (Completed)
- Disable bulk progress indicators automatically in `--output agent`.
- Ensure helper printers (like `printAction`) never write to stdout in agent mode.

### Phase 6: Expand Flat ID/URL Access Beyond Core Resources (Completed)
Goal: apply the same “desire paths” (plain ID, `#id`, `type:id`, pasted UI URL) consistently across the rest of the CLI, so agents spend tokens on the task, not on navigation.

#### Phase 6.1: Teams + Campaigns + Agents (Completed)
- Make all team ID args accept: `123`, `#123`, `team:123`, `https://.../teams/123`.
- Make all campaign ID args accept: `123`, `#123`, `campaign:123`, `https://.../campaigns/123`.
- Make all agent ID args accept: `123`, `#123`, `agent:123`, `https://.../agents/123`.
- Add tests for `teams get` and `campaigns get` desire paths.

#### Phase 6.2: Webhooks + Custom Attributes/Filters (Completed)
- Apply the same ID acceptance to `webhooks`, `custom-attributes`, and `custom-filters`:
  - `123`, `#123`, and typed prefixes (`webhook:123`, `custom-attribute:123`, `custom-filter:123`).
- Note: Chatwoot UI URL parsing is only supported for resources recognized by `internal/urlparse` (conversations, contacts, inboxes, teams, agents, campaigns).
- Add targeted tests to prevent regression.

#### Phase 6.3: `open` Convenience Defaults (Completed)
- `chatwoot open <id>` defaults to opening a conversation (accepts `123`, `#123`, `conv:123`).
- `chatwoot open <typed-id>` infers the resource from the prefix (e.g. `contact:789`, `team:5`) without requiring `--type`.
- `chatwoot open <id> --type <resource>` (and `open <resource> <id>`) accept the same ID shorthands and pasted URLs.
- Tests.

### Phase 7: Typed Prefix Sweep For Remaining Commands (Completed)
Goal: remove remaining “numeric-only” cliffs in the CLI. Any command that takes a single resource ID should accept:
- plain: `123`
- hash: `#123`
- typed: `<resource>:123` (e.g. `label:5`, `hook:9`, `rule:7`, `article:11`)

Scope:
- `labels get|update|delete` accept `#id` and `label:id`
- `canned-responses get|update|delete` accept `#id` and `canned-response:id`
- `automation-rules get|update|delete|clone` accept `#id` and `rule:id`
- `agent-bots get|update|delete|delete-avatar|reset-token` accept `#id` and `bot:id`
- `integrations hook-update|hook-delete` accept `#id` and `hook:id`
- `portals articles get|update|delete` accept `#id` and `article:id`
- `platform` subcommands that take IDs (`account-id`, `user-id`, `bot-id`) accept `#id` and typed prefixes (`account:id`, `user:id`, `bot:id`)
- `campaigns --labels` parsing accepts `#id` and `label:id` (comma-separated list)

Non-goals (for now):
- Extending UI URL parsing beyond resources recognized by `internal/urlparse`.
- Adding new commands; this phase is strictly about input normalization for existing commands.

### Phase 8: Agent-Oriented Flat Access Resolver (Completed)
Goal: add a single-purpose command that turns “whatever the agent has” (URL, `#id`, typed `type:id`, or even a bare numeric ID) into a canonical, chainable reference.

Deliverable: `chatwoot ref <identifier>`
- Input forms:
  - `123`, `#123`
  - `type:123` (e.g. `contact:123`, `label:5`, `rule:7`)
  - supported UI URLs (via `internal/urlparse`)
- Output:
  - defaults to emitting a typed ID in text mode (easy chaining)
  - supports `--emit json|id|url`
  - in `--output agent`, returns an agent envelope with the resolved reference
- Probing (for bare IDs):
  - default probes `conversation` + `contact` to avoid “numeric-only” ambiguity without excessive API calls
  - configurable via `--try <type>` (repeatable)
  - errors clearly on ambiguity (agent-mode structured error with match list)

### Phase 9: `ref` Next-Step Suggestions (Completed)
Goal: reduce agent “what should I do next?” tokens by returning a small, correct set of follow-up actions for a resolved reference, without extra API calls.

Deliverable: add `actions[]` to `chatwoot ref --emit json` and `--output agent` output.
- Actions are:
  - structured objects (`id`, `title`, `argv`, `destructive` + optional `notes`)
  - pre-filled with the canonical `type:id` where possible
  - may include placeholders (e.g. `<text>`, `<portal-slug>`) when required by the underlying command

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

- Phase 5 implemented:
  - `--ids` flags (bulk contacts/conversations operations) accept `@-` (stdin) and `@path` (file) and can parse CSV/whitespace/JSON arrays.
  - `chatwoot search --best` added for non-interactive best-match selection, plus `--emit id|url|json`.
  - Agent-mode noise tightened (progress/action printers won’t interfere with `--output agent`).

- Phase 6.2 implemented:
  - `webhooks`, `custom-attributes`, and `custom-filters` accept `#id` and typed prefixes (`webhook:id`, `custom-attribute:id`, `custom-filter:id`).
  - Added tests for prefixed/hash IDs on these commands.

- Phase 6.3 implemented:
  - `open` accepts typed ID prefixes (`contact:789`, `team:5`, etc.) without requiring `--type`.
  - Added tests for the typed prefix behavior.

- Phase 7 started:
  - Plan: sweep remaining numeric-only ID args and make them accept `#id` and typed prefixes (`resource:id`) for consistent agent “desire paths”.

- Phase 7 completed:
  - Updated remaining numeric-only commands to use `parseIDOrURL`, enabling `#id` and typed prefixes:
    - `labels`, `canned-responses`, `automation-rules`, `agent-bots`, `integrations hook-update|hook-delete`, `portals articles`, `platform`, and `campaigns --labels`.
  - Added tests ensuring `#id` and `<resource>:id` desire paths work for:
    - `labels get`, `automation-rules get`, `agent-bots get`, `integrations hook-update`, `portals articles get`, `platform accounts get`, and `campaigns create --labels`.
  - Verified: `go test ./...` is green.

- Phase 8 started:
  - Added plan + started implementing `chatwoot ref` as an agent-friendly “flat access resolver”.

- Phase 8 completed:
  - Added `chatwoot ref` command:
    - accepts `123`, `#123`, `type:123`, and supported UI URLs
    - emits `--emit id|url|json` (defaults: `id` in text mode, `json` in json/agent modes)
    - probes bare IDs by default across `conversation` + `contact` (configurable via `--try`)
    - returns structured agent errors on ambiguity (includes match list)
  - Added tests for typed IDs, URL parsing, probe resolution, and ambiguity.
  - Verified: `go test ./...` is green.

- Phase 9 started:
  - Plan: enrich `chatwoot ref` JSON/agent output with follow-up `actions[]` for common next steps.

- Phase 9 completed:
  - Added `actions[]` to `chatwoot ref` JSON/agent output. Each action includes `id`, `title`, `argv`, and optional `destructive`/`notes`.
  - Actions are pre-filled with canonical `type:id` (and placeholders when required).
  - Added tests asserting actions are present for common types.
  - Verified: `go test ./...` is green.

### Phase 10: `ref` Actions Polish (Completed)
Goal: make `chatwoot ref` action suggestions more correct and easier for agents to execute.

Scope:
- Fix action safety metadata (don’t label ordinary `update` operations as destructive).
- Add structured `inputs` metadata to actions so agents can fill required parameters without parsing placeholder strings.
  - Example: `comment` exposes an input named `text`; `assign` exposes `agent` and/or `team`.

Status:
- Implemented:
  - `actions[]` placeholders standardized to `$name` tokens.
  - Added `inputs[]` to relevant actions (`comment`, `note`, `assign`, and update-like actions).
  - Fixed incorrect `destructive` flags on update operations.
  - Added tests verifying `inputs[]` presence and placeholder tokenization.
  - Verified: `go test ./...` is green.

### Phase 11: Flag/List Normalization Sweep (Completed)
Goal: make “list-shaped” flags consistently agent-friendly across the CLI.

Deliverables:
- Add `ParseStringListFlag` to support `@-`/`@path`, CSV/whitespace/newlines, and JSON arrays for string list flags.
- Update list flags to accept:
  - `@-` / `@path`
  - CSV: `a,b,c`
  - whitespace/newlines: `a b c` / `a\nb\nc`
  - JSON array: `["a","b"]` (or `[1,2]` where appropriate)
- Targets:
  - `teams members-add|members-remove --user-ids`
  - `inbox-members add|remove|update --user-ids`
  - `portals articles reorder --article-ids`
  - label list flags (`--labels`) on contacts/conversations bulk + per-resource commands
  - `campaigns --labels` should accept `@-`/`@path` and JSON arrays (label IDs)

Status:
- Implemented:
  - Added `ParseStringListFlag` for string list flags with `@-`/`@path`, CSV/whitespace/newlines, and JSON arrays.
  - Updated `--user-ids` (teams + inbox-members) and `--article-ids` (portals reorder) to accept `@-`/`@path` and JSON arrays.
  - Updated label list flags (`--labels`) on contacts/conversations commands and bulk commands to accept `@-`/`@path` and JSON arrays.
  - Updated `campaigns --labels` parsing to accept `@-`/`@path` and JSON arrays (label IDs).
  - Added tests for stdin-driven list flags (`--user-ids @-`, `--article-ids @-`, `--labels @-`).
  - Verified: `go test ./...` is green.

### Phase 12: Parsing Consistency Cleanup (Completed)
Goal: remove remaining “typed prefix surprises” and reduce the risk of agents hitting sharp edges due to mismatched resource names.

Scope:
- Fix `user:<id>` typed prefix handling so it works for platform user IDs (`chatwoot platform users ...`) and `chatwoot ref user:<id>`.
- Add regression tests to prevent future prefix/type mismatches.

Status:
- Implemented:
  - `parseIDOrURL` now treats `user:<id>` as:
    - platform user IDs when `expectedResource=="user"` (platform commands)
    - agent IDs otherwise (backwards compatible for account agent workflows)
  - `chatwoot ref user:<id>` now resolves to `type: "user"` (so it suggests `chatwoot platform users ...` actions).
  - Fixed a flaky async wait test by normalizing wrapped timeout/cancel errors to the canonical `context.DeadlineExceeded` / `context.Canceled`.
  - Added regression tests for platform user typed IDs and `ref user:<id>`.
  - Verified: `go test ./...` is green.

### Phase 13: List Flags + `--emit` Convenience (Completed)
Goal: reduce “string parsing cliffs” and cut down agent tokens for common chaining patterns.

Scope:
- List-shaped flags:
  - Make remaining comma-separated list flags accept `@-`/`@path`, whitespace/newlines, and JSON arrays.
  - Targets:
    - `webhooks --subscriptions`
    - `agents bulk-create --emails`
    - global `--fields` (output selection shorthand)
- `--emit` convenience:
  - Add `--emit id|url|json` to more single-item create/update/get commands, so agents can chain without `jq`.
  - Start with core resources where an ID and UI URL are well-defined (conversations, contacts, inboxes, teams, agents, campaigns), then extend to other single-resource create/update commands.

Status:
- Implemented list-flag upgrades:
  - `webhooks --subscriptions` now accepts repeatable values, CSV/whitespace/newlines, JSON arrays, and `@-`/`@path`.
  - `agents bulk-create --emails` now accepts CSV/whitespace/newlines, JSON arrays, and `@-`/`@path`.
  - global `--fields` now accepts CSV/whitespace/newlines, JSON arrays, and `@-`/`@path`.
- Implemented `--emit id|url|json` on core single-resource commands:
  - conversations: `get|create|update`
  - contacts: `get|show|create|update`
  - inboxes: `get|create|update`
  - teams: `get|create|update`
  - agents: `get|create|update`
  - campaigns: `get|create|update`
  - webhooks: `create|update`
- Added regression tests for:
  - webhooks subscriptions from stdin and JSON arrays
  - agents bulk-create emails from stdin
  - `--fields @-` parsing
  - `--emit` short-circuit behavior (skip API calls when emitting `id`/`url`)
- Verified: `go test ./...` is green.
