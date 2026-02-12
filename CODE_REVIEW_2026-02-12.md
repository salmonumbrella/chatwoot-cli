# Chatwoot CLI Code Review Follow-up (2026-02-12)

## Scope

This document records the three issues called out in the review and the concrete fixes implemented in this change set.

## 1) `CHATWOOT_RESOLVE_NAMES` default was not applied at runtime

### Problem

`Execute()` initialized `flags.ResolveNames` from `CHATWOOT_RESOLVE_NAMES`, but the Cobra flag registration used a hardcoded default (`false`).  
That meant the env-driven default was overwritten during command setup.

### Fix

- Updated root flag binding to use `flags.ResolveNames` as the default value.
- Added an integration-style command test that verifies env-only behavior:
  - `internal/cmd/conversations_cmd_test.go`
  - `TestConversationsListCommand_AgentResolveNamesFromEnv`

## 2) `--allow-private` state could leak across `Execute()` calls

### Problem

`validation.SetAllowPrivate(true)` was only called when `--allow-private` was present.  
Because validation state is global, a previous run could leave it enabled for subsequent runs in the same process.

### Fix

- Made allow-private policy deterministic per execution:
  - `allowPrivate := CHATWOOT_ALLOW_PRIVATE || --allow-private`
  - Always call `validation.SetAllowPrivate(allowPrivate)` in `PersistentPreRunE`.
- Added regression test:
  - `internal/cmd/root_test.go`
  - `TestExecute_AllowPrivateDoesNotLeakAcrossRuns`

## 3) Large command file maintainability

### Problem

`internal/cmd/conversations.go` mixed command registration and large list/table logic, increasing cognitive load and edit risk.

### Fix

- Extracted list-specific logic into `internal/cmd/conversations_list.go`:
  - `newConversationsListCmd`
  - `printConversationsTable`
  - `conversationRow`
  - `conversationsListSummary`
- Left behavior unchanged while narrowing responsibilities in the main `conversations.go`.

## Files updated

- `internal/cmd/root.go`
- `internal/cmd/root_test.go`
- `internal/cmd/conversations.go`
- `internal/cmd/conversations_list.go`
- `internal/cmd/conversations_cmd_test.go`
- `CODE_REVIEW_2026-02-12.md`
