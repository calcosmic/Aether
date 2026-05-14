# Phase 121: Go Runtime Test Restoration - Context

**Gathered:** 2026-05-14
**Status:** Ready for planning

## Phase Boundary

Fix `go test ./cmd` failures caused by archived phase artifacts and a broken resume dashboard signal injection. No new features — test stabilization only.

## Implementation Decisions

- For missing DATA-FLOW.md and WORKER-ECONOMY.md: tests should skip gracefully when files don't exist, rather than fail. These are documentation audit tests, not runtime logic tests.
- For resume dashboard: investigate whether the test expectation is wrong or the code is broken, then fix the appropriate side.

## Threat Model

| Threat | Mitigation |
|--------|-----------|
| Skip logic hides real regressions | Only skip when file explicitly missing; assert all other behavior |
| Resume dashboard fix breaks runtime | Test-only change or verified with existing integration tests |
| Archived artifacts restored by accident | Do not recreate deleted files; update tests instead |

## Test Strategy
- `go test ./cmd` must pass as final verification
- Individual test files verified in isolation before full suite
