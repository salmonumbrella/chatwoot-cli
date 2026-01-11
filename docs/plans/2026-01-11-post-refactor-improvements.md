# Post-Refactor Improvement Plan

Goal: After the refactor, polish the CLI for a more consistent UX, stronger safety, and higher confidence in correctness.

Non-goals:
- No breaking changes to existing flags/commands without explicit migration messaging.
- No large API surface expansion unless already tracked in the missing-endpoints plan.

## Phase 1: UX polish + consistency
- Standardize command help/Examples across all resources (same ordering, same verbs).
- Add shell completion docs and ensure `chatwoot completion` is discoverable.
- Make success/error text uniform (verbs, IDs, punctuation), including multi-resource operations.
- Review `--output json` text-mode messages to ensure no extra stdout noise.

Acceptance:
- Commands have consistent usage blocks and examples.
- `--output json` produces clean JSON on stdout for all commands.

## Phase 2: Config/profile ergonomics
- Add `chatwoot auth status --json` (or similar) for machine-readable info.
- Improve profile selection feedback (current profile name and base URL).
- Add a `--profile` default resolution order doc section in README.

Acceptance:
- Profiles are discoverable and debuggable without reading config files.

## Phase 3: Reliability + safety
- Introduce per-command timeouts (global default + override flag).
- Add idempotency keys for safe retries on mutation (where supported by API).
- Surface request IDs in error output when available (for support/debugging).

Acceptance:
- Users can set a timeout and see it reflected in failures.
- Retried writes are safe when server supports idempotency.

## Phase 4: Output & formatting improvements
- Extend table output to show key fields consistently (IDs, names, status, timestamps).
- Add `--fields` presets per resource (e.g., `minimal`, `default`, `debug`).
- Improve `--template` errors (show template line/column where possible).

Acceptance:
- Default outputs are predictable and aligned per resource.
- Template errors are actionable.

## Phase 5: Test + fixture hardening
- Add golden output tests for key commands (text + json).
- Add integration-style tests with httptest for list/paginated commands.
- Add regression tests for edge cases (empty lists, single-item, error surfaces).

Acceptance:
- Core commands have snapshot tests guarding output format.
- Pagination logic has coverage beyond unit tests.

## Phase 6: Docs + examples
- Add quick-start examples per resource in README.
- Document common workflows (inboxes -> conversations -> messages).
- Add troubleshooting section (auth failures, 401/404, URL validation errors).

Acceptance:
- README contains at least one runnable example per resource group.
- Troubleshooting addresses top 5 error classes.

## Order of execution
1) Phase 1
2) Phase 2
3) Phase 3
4) Phase 4
5) Phase 5
6) Phase 6

## Open questions
- Which shell completions are highest priority (bash/zsh/fish)?
- Do we want to add a `--timeout` global flag or per-command flags?
- Should `auth status` include token age or only validity check?
