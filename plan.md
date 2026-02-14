# Coverage Plan (Toward 100%)

## Goal
Reach `100.0%` statement coverage for `go test ./...` while keeping behavior stable and tests maintainable.

## Baseline (2026-02-14)
- Total statement coverage: `74.8%`
- High-gap packages:
- `cmd/chatwoot` (`0.0%`)
- `internal/skill` (`7.7%`)
- `internal/agentfmt` (`49.0%`)
- `internal/cmd` (`72.1%`)

## Current Snapshot (after this pass)
- Total statement coverage: `79.3%` (up from `74.8%`)
- `cmd/chatwoot`: `100.0%` (was `0.0%`)
- `internal/agentfmt`: `100.0%` (was `49.0%`)
- `internal/api`: `92.1%` (was `87.7%`)
- `internal/cache`: `84.9%` (was `72.6%`)
- `internal/resolve`: `91.7%` (was `72.2%`)
- `internal/skill`: `89.7%` (was `7.7%`)
- `internal/cmd`: `75.8%` (was `72.1%`)

## Principles
- Prefer small, behavior-focused tests over broad mocks.
- Add test seams only when needed for determinism.
- Enforce alias/flag invariants with explicit guardrail tests.
- Keep tests deterministic (no network dependence beyond `httptest`).

## Phase 1 (Implemented in this pass)
- [x] Add tests for `cmd/chatwoot/main.go` startup wiring and argument passthrough.
- [x] Add focused tests for `internal/skill/generator.go` paths and error branches.
- [x] Add more command-path/edge-case tests in `internal/cmd` to push overall coverage higher.

## Phase 2 (Highest ROI)
- [x] Raise `internal/agentfmt` coverage:
- Added table tests for `Transform`, `TransformListItems`, and projection helpers.
- Covered type-conversion helpers (`toInt`, `mapFromAny`, sender/contact meta extraction).
- [x] Raise `internal/cmd` coverage:
- Added tests for `conversations follow` helper/event/cursor branches, transcript helpers, and cache command branches.
- Expanded alias and nested-command regression coverage.
- [x] Raise `internal/api` uncovered branch coverage:
- Added tests for `DoRaw`, `SetRetryConfig`, rate-limit parsing/metadata branches, `CreateFromMap`, `Mute/Unmute`, `ListWithLimit`, and context service wrappers.

## Phase 3
- [x] Cover remaining branch-only gaps per package using `go tool cover -func` drilldowns.
- [x] Add CI gate for strict coverage target:
- Added `scripts/check-coverage.sh`.
- Added workflow gate in `.github/workflows/ci.yml` with baseline ratchet target `79.0`.
- [x] Final sweep for this pass:
- Closed high-impact residual gaps and locked them with regression tests.

## Verification Commands
```bash
go test ./... -cover
go test ./... -coverprofile=/tmp/chatwoot-cover.out
go tool cover -func=/tmp/chatwoot-cover.out
./scripts/check-coverage.sh
```
