# Phase 122: Classic Parity Coverage - Context

**Gathered:** 2026-05-14
**Status:** Ready for planning

## Phase Boundary

Verify that restored behavior matches Classic v5.4 baseline. All tests and documentation already exist — this is a verification-only phase with no code changes required.

## Implementation Decisions

- No code changes needed — all PAR requirements are already satisfied
- Verification consists of running the full test suite and confirming all golden/parity tests pass
- If any test fails, investigate whether it's a regression or a stale test

## Threat Model

| Threat | Mitigation |
|--------|-----------|
| False positive — tests pass but coverage is incomplete | Research document maps each PAR requirement to specific test file and line |
| Stale golden files | Golden files were updated during Phase 108; tests verify exact output |
| Missing documentation | classic-baseline.md exists and is comprehensive |

## Test Strategy
- `go test ./...` must pass completely
- Individual PAR requirement verification via targeted test runs
