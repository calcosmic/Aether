---
phase: 157-ts-adapters-oracle
plan: 02
subsystem: oracle
tags: [oracle, typescript, control-plane, research, stubs]
dependency_graph:
  requires: [157-01]
  provides: [158-event-stream]
  affects: [control-ts/src/index.ts]
tech_stack:
  added: []
  patterns: [barrel-export, TDD-red-green, stub-simulation]
key_files:
  created:
    - control-ts/src/oracle/types.ts
    - control-ts/src/oracle/evaluateConfidence.ts
    - control-ts/src/oracle/buildResearchPlan.ts
    - control-ts/src/oracle/runOracleIteration.ts
    - control-ts/src/oracle/index.ts
    - control-ts/tests/oracle/oracle.test.ts
  modified:
    - control-ts/src/index.ts
decisions:
  - "Oracle stubs simulate worker responses with deterministic confidence growth (50 + iteration * 5) capped at target"
  - "All oracle types mirror Go oracle loop shapes for future bridge compatibility"
  - "Public API exports runOracleLoop, createOracleState, createOraclePlan plus key types"
  - "maxIterations defaults to 8 and targetConfidence to 75 (standard depth)"
  - "Placeholder question returned when all questions are answered to prevent nulls"
metrics:
  duration: "7 minutes"
  completed_date: "2026-05-24"
  tasks: 2
  files_created: 6
  files_modified: 1
  tests_added: 38
---

# Phase 157 Plan 02: Oracle Loop Stub Summary

**One-liner:** TypeScript Oracle research loop with confidence evaluation, rubric generation, plan building, iteration simulation, and full test coverage — mirroring Go oracle loop shapes as stubs.

## What Was Built

Created the Oracle module in the TypeScript control plane. The Oracle is the colony's research specialist. These stubs simulate the Go oracle loop behavior without making real worker invocations.

### Files Created

| File | Purpose | Exports |
|------|---------|---------|
| `control-ts/src/oracle/types.ts` | Oracle domain types | `OracleState`, `OraclePlan`, `OracleQuestion`, `OracleFinding`, `OracleSource`, `OracleWorkerResponse`, `OracleDepthConfig`, `OracleScopeProfile`, `OracleRubricEntry`, `OracleGapEntry`, `OracleEvidenceEntry` |
| `control-ts/src/oracle/evaluateConfidence.ts` | Confidence scoring and rubric generation | `evaluateConfidence`, `buildOracleRubric`, `clampConfidence`, `inferEvidenceType` |
| `control-ts/src/oracle/buildResearchPlan.ts` | Research plan creation and question management | `buildResearchPlan`, `selectNextQuestion`, `identifyGaps`, `collectEvidence`, `buildSynthesizedPrompt`, `resolveOracleDepth`, `resolveOracleScope`, `inferOracleTemplate` |
| `control-ts/src/oracle/runOracleIteration.ts` | Single iteration runner and loop controller | `runOracleIteration`, `runOracleLoop`, `createOracleState`, `createOraclePlan`, `oracleReadyForCompletion`, `snapshotOracleProgress`, `oracleProgressedSince` |
| `control-ts/src/oracle/index.ts` | Barrel export for the oracle module | Re-exports all types and functions above |
| `control-ts/tests/oracle/oracle.test.ts` | Comprehensive test suite | 38 tests covering confidence, rubric, plan building, iteration, loop, gaps, evidence, depth/scope resolution, and progress tracking |

### File Modified

| File | Change |
|------|--------|
| `control-ts/src/index.ts` | Added exports for `runOracleLoop`, `createOracleState`, `createOraclePlan`, and key oracle types |

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

These are intentional stubs per the plan. Real worker invocation and web search are deferred to the Go runtime bridge.

| Stub | File | Reason |
|------|------|--------|
| Simulated worker response | `runOracleIteration.ts` | No real worker invocations in stubs |
| Deterministic confidence growth | `runOracleIteration.ts` | Mock behavior for testing loop mechanics |
| Placeholder question when all answered | `buildResearchPlan.ts` | Prevents null returns; real logic will differ |

## Threat Flags

No new security-relevant surface introduced. The oracle stubs do not perform network calls, file writes, or execute external commands.

## Self-Check: PASSED

- [x] `control-ts/src/oracle/types.ts` exists
- [x] `control-ts/src/oracle/evaluateConfidence.ts` exists
- [x] `control-ts/src/oracle/buildResearchPlan.ts` exists
- [x] `control-ts/src/oracle/runOracleIteration.ts` exists
- [x] `control-ts/src/oracle/index.ts` exists
- [x] `control-ts/tests/oracle/oracle.test.ts` exists
- [x] All 38 oracle tests pass
- [x] `npm run typecheck` passes with zero errors
- [x] Commits verified: `675512e8`, `be6c6a88`, `caf9df3c`
