# Phase 121: Go Runtime Test Restoration - Research

**Gathered:** 2026-05-14
**Status:** Ready for planning

## Test Failures

### 1. Resume dashboard signal injection (GOT-01)
**File:** `cmd/context_test.go:435`
**Test:** `TestResumeDashboardIncludesSignalsBlockersSurveyAndRecoverySource`

```
signals.items = [], want injected focus signal
```

The resume dashboard context builder fails to inject the active pheromone signals into the output. The test expects `signals.items` to contain a focus signal, but the array is empty.

**Likely cause:** The `buildResumeContext` function (or equivalent) reads pheromones but doesn't surface them in the dashboard context structure, or the signal injection path is broken after a refactor.

### 2-5. Data flow audit tests (GOT-02)
**File:** `cmd/data_flow_audit_test.go`
**Tests:** `TestDataFlowSnapshot`, `TestDataFlowDeadEnds`, `TestDataFlowColonyPrimeWiring`, `TestDataFlowReportAccuracy`

All 4 tests fail with:
```
read DATA-FLOW.md: open .planning/phases/103-data-flow-artifact-wiring/DATA-FLOW.md: no such file or directory
```

The referenced phase directory and its `DATA-FLOW.md` were deleted/archived during milestone cleanup. The tests are hardcoded to read this specific file path.

### 6-7. Worker economy tests (GOT-03)
**File:** `cmd/worker_economy_test.go`
**Tests:** `TestDispatchedCastesDocumented`, `TestVisualOutputTracesToState`

Both fail with:
```
read WORKER-ECONOMY.md: open .planning/phases/102-worker-economy-visual-ceremony-audit/WORKER-ECONOMY.md: no such file or directory
```

Same issue — the referenced phase artifacts were archived.

## Root Cause

During v1.17 milestone archival, SUMMARY.md and PLAN.md files from old phases were deleted. Some Go tests depend on those specific file paths for audit/documentation verification. The tests were not updated to handle missing archived artifacts.

## Files to Modify
- `cmd/context_test.go` — fix signal injection assertion or the code under test
- `cmd/data_flow_audit_test.go` — skip or stub when DATA-FLOW.md missing
- `cmd/worker_economy_test.go` — skip or stub when WORKER-ECONOMY.md missing

## Verification Target
- `go test ./cmd` passes with zero failures
- No state files or runtime behavior changed
