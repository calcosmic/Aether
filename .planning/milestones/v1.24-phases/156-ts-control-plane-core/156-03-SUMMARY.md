# Phase 156-03 Summary: Plan Executor (`executePlan`)

## Deliverables Status

| Deliverable | Status | Notes |
|-------------|--------|-------|
| `src/orchestrator/executePlan.ts` | **Complete** | Plan executor with failure policy handling, colony state updates, NDJSON event emission |
| `tests/orchestrator/executePlan.test.ts` | **Complete** | 8 tests, all passing |
| Public API exports (`src/index.ts`, `src/types/index.ts`) | **Complete** | `executePlan` and `PlanResult` exported |
| Typecheck | **Passing** | `tsc --noEmit` clean |
| Full test suite | **Passing** | 68/68 tests across 11 files |
| Test isolation | **Fixed** | `AETHER_EVENTS_FILE` env var isolates NDJSON events per test file |

## What Was Built

### `executePlan` Function
- Accepts a phase ID sequence and optional per-phase inputs / max retries
- Runs phases in order via `runPhase`
- Handles four failure policies:
  - `block` (default): stop execution
  - `skip`: continue to next phase
  - `retry`: retry up to `maxRetries` (default 1), then treat as block
  - `escalate`: stop execution
- Emits `plan:start` and `plan:complete` events to `.aether/events/current.ndjson`
- Updates colony state (`state`, `current_phase`) via `updateColonyState`
- Returns `PlanResult` with overall status (`completed` | `failed` | `partial`)

### Key Design Decisions
- **Status semantics**: If the last result is a failed block/escalate/retry-exhausted, the overall status is forced to `failed`, even if some earlier phases completed. This prevents a mixed completed/failed sequence from being misreported as `partial` when the final outcome is a hard stop.
- **ESM mocking strategy**: Used `vi.doMock` + dynamic `import()` with query-string cache busting (`?block=1`, `?skip=1`, etc.) because `vi.spyOn` cannot redefine ESM namespace exports.
- **Module cache isolation**: Each mocked test uses a unique query string on the dynamic import to prevent mock leakage between tests.
- **Test isolation via env var**: Both `runPhase` and `executePlan` read `process.env.AETHER_EVENTS_FILE` to determine the NDJSON events path. Tests set this to a temp directory (`mkdtempSync`) in `beforeEach` and clean it up in `afterEach`, preventing cross-test file pollution when Vitest runs tests in parallel.

## Test Coverage

8 tests covering:
1. Full sequence execution in order
2. `plan:start` and `plan:complete` NDJSON events
3. `block` failure policy stops execution
4. `skip` failure policy continues to next phase
5. `retry` failure policy retries once then blocks
6. Colony state updates (`current_phase`, `state` transitions)
7. Empty sequence returns `completed` immediately
8. Per-phase inputs passed through `options.inputs`

## Build / Test Results

```
> tsc --noEmit
(clean)

> vitest run
Test Files  11 passed (11)
Tests       68 passed (68)
```

## Files Created / Modified

- **Created**: `control-ts/src/orchestrator/executePlan.ts`
- **Created**: `control-ts/tests/orchestrator/executePlan.test.ts`
- **Modified**: `control-ts/src/types/index.ts` (added `PlanResult` export)
- **Modified**: `control-ts/src/index.ts` (added `executePlan` and `PlanResult` exports)
- **Modified**: `control-ts/src/orchestrator/runPhase.ts` (added `AETHER_EVENTS_FILE` env var support)
- **Modified**: `control-ts/tests/orchestrator/runPhase.test.ts` (switched to temp-dir isolation via env var)
- **Modified**: `control-ts/package.json` (reverted accidental `--no-isolate` change)
- **Created**: `.planning/phases/156-ts-control-plane-core/156-03-SUMMARY.md` (this file)
