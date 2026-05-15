---
plan: 134-01
phase: 134
status: complete
completed: "2026-05-15"
---

# Plan 134-01: Oracle Storage Pattern

## What Was Built

Moved Oracle iteration state from plain file I/O to locked/atomic Go-owned storage, matching the colony state storage pattern. Added interrupt recovery so Oracle sessions can resume after process kills.

## Changes

### Go Runtime

**cmd/oracle_iterate_cmd.go**
- `loadOracleState()` now uses `store.ReadFile("oracle/state.json")` with automatic fallback to `os.ReadFile` when `store == nil` (test path)
- `saveOracleState()` now uses `store.SaveJSON("oracle/state.json", state)` with same nil-store fallback
- Added interrupt recovery fields to `oracleState`: `PendingIteration`, `PendingStartTime`, `LastWorkerStatus`
- Added `Resuming` flag to `iterationManifest` so the TS host receives resume signals
- `--plan-only` writes a pending marker before returning the manifest
- `oracle-iterate-finalize` clears the pending marker on success
- Added `isPendingStale()` helper with 1-hour timeout auto-clear

**cmd/state_extra.go**
- `validate-oracle-state` now validates that the oracle state path is within `.aether/data/oracle/`

### Tests

**cmd/oracle_iterate_cmd_test.go**
- `TestOracleStateConcurrentWrites` — 10 goroutines write concurrently, verify no corruption
- `TestOracleStateReadDuringWrite` — 50-cycle writer/reader loop, verify no malformed reads
- `TestOracleStateStoreInitCreatesDirectories` — verify `SaveJSON` auto-creates `.aether/data/oracle/`
- `TestOracleStateMissingReturnsError` — verify clean error (not panic) for missing state
- `TestOracleStateInterruptRecovery` — verify `resuming=true` when pending marker matches current iteration, and finalize clears it
- `TestOracleStateStalePendingAutoClears` — verify stale pending (>1h) is replaced with fresh marker
- Fixed `setupOracleTestDir` to reset global `store` and isolate `AETHER_ROOT` to prevent cross-test leakage

**cmd/state_extra_test.go**
- `TestValidateOracleStatePathValidation` — verify command passes when state written via `store.SaveJSON`

**TypeScript Host**

**.aether/ts-host/src/types.ts**
- Added `resuming?: boolean` to `OracleIterationState` interface

**.aether/ts-host/test/oracle-lifecycle.test.ts**
- Added test verifying TS host continues loop when Go manifest indicates `resuming: true`

## Verification

- `go test ./cmd/ -run TestOracle -race` — PASS
- `go test ./pkg/storage/ -race` — PASS
- `npm test` in `.aether/ts-host/` — 217/217 pass
- `go vet ./...` — clean
- All Oracle-related tests pass in isolation and in suite

## Issues Encountered

1. **Store path mismatch**: `store.ReadFile` resolves relative to `.aether/data/`, but `oracleStatePath()` returns `.aether/data/oracle/state.json`. Fixed by using `"oracle/state.json"` with the store and keeping the full path for plain-file fallback.
2. **Test store leakage**: Global `store` and `AETHER_ROOT` env var leaked between tests in full-suite runs. Fixed `setupOracleTestDir` to snapshot and restore both.
3. **Subagent stuck in reading loop**: The spawned executor agent got stuck reading files repeatedly. Switched to inline execution.

## Pre-existing Failures (not caused by this work)

The full `go test ./cmd/ -race` suite shows 4 pre-existing failures unrelated to Oracle state:
- `TestAuditCatalogGolden`
- `TestContinueWrapperCeremonyContract`
- `TestPlanWrapperCeremonyContract`
- `TestRegressionSnapshot`
