---
phase: 117-oracle-enhancement
plan: 01
type: execute
subsystem: oracle
wave: 1
tags: [oracle, ceremony, novelty, phase-directives, typescript]
dependency_graph:
  requires: []
  provides: [ORA-01, ORA-02]
  affects: [cmd/oracle_loop.go, cmd/ceremony_emitter.go, pkg/events/ceremony.go, .aether/ts-host]
tech_stack:
  added: []
  patterns: [phase-aware-prompts, jaccard-novelty, ceremony-events]
key_files:
  created:
    - .aether/ts-host/test/oracle-events.test.ts
  modified:
    - cmd/oracle_loop.go
    - cmd/ceremony_emitter.go
    - pkg/events/ceremony.go
    - .aether/ts-host/src/types.ts
    - .aether/ts-host/src/narrator.ts
decisions: []
metrics:
  duration_seconds: 426
  completed_date: 2026-05-13
  tasks_completed: 5
  files_created: 1
  files_modified: 5
---

# Phase 117 Plan 01: Oracle Enhancement Wave 1 Summary

## One-liner

Enhanced the Go Oracle RALF loop with phase-aware prompts (survey, verify, investigate, synthesize), diminishing-returns detection via Jaccard-distance novelty tracking, and ceremony event emission for TS host visibility.

## What Changed

### Task 1: Phase-aware prompt directives
- `nextOraclePhase` now returns `"synthesize"` as the final phase after `"verify"` completes.
- `buildOraclePhaseDirective` helper injects explicit phase-specific instructions into each Oracle worker brief:
  - **survey**: "Identify key concepts, existing solutions, and open questions. Do not form conclusions yet."
  - **verify**: "Test assumptions, look for contradictions, and assess confidence levels."
  - **investigate**: "Deep-dive into the most promising areas identified in the survey."
  - **synthesize**: "Connect dots, resolve contradictions, and formulate recommendations."

### Task 2: Diminishing-returns detection
- Added `noveltyTracker` struct to `oracleState` with `LastKeywords`, `ConsecutiveLow`, and `Threshold`.
- `extractKeywordsSet` filters stop words and returns a keyword set.
- `computeNoveltyDelta` calculates Jaccard distance: `1 - |intersection| / |union|`.
- `oracleProgressedSince` returns `false` when `ConsecutiveLow >= 3` (novelty < 15% for 3 consecutive iterations), causing the loop to stop.
- `applyOracleWorkerResponse` updates the tracker after each worker response.

### Task 3: Oracle ceremony event emitters
- Added `emitOraclePhaseTransition` and `emitOracleIteration` to `cmd/ceremony_emitter.go`.
- Added `ceremony.oracle.phase_transition` and `ceremony.oracle.iteration` topics to `pkg/events/ceremony.go`.
- `runOracleLoop` emits a phase-transition event when the phase changes, and an iteration event at the start of each loop iteration.

### Task 4: TS host narrator updates
- Added new ceremony topics to `CEREMONY_TOPICS` in `.aether/ts-host/src/types.ts`.
- Narrator renders `ceremony.oracle.phase_transition` as a stage separator: `Oracle: {from} → {to}`.
- Narrator renders `ceremony.oracle.iteration` as a spawn frame with caste `oracle`, name `Oracle-{iteration}`, and task `Researching: {question}`.

### Task 5: Oracle ceremony event tests
- Created `.aether/ts-host/test/oracle-events.test.ts` with 2 tests:
  1. Phase transition rendering
  2. Iteration frame rendering

## Deviations from Plan

### Minor deviation: Test payload shape
- **Found during:** Task 5
- **Issue:** The plan's test specification used `phase_from`/`phase_to`/`iteration` field names, but the Go `CeremonyPayload` struct maps the transition to `status` and `message` fields (no `phase_from`/`phase_to` keys).
- **Fix:** Updated the test payloads to match the actual Go emitter output: `status: "survey → verify"`, `phase_name: "survey"`, `wave: 5`, `task: "..."`.
- **Files modified:** `.aether/ts-host/test/oracle-events.test.ts`
- **Commit:** `5b007a92`

## Threat Flags

None. No new network endpoints, auth paths, file access patterns, or schema changes at trust boundaries were introduced.

## Known Stubs

None. All plan goals achieved; no placeholder data or un-wired components remain.

## Self-Check: PASSED

- [x] `cmd/oracle_loop.go` modified
- [x] `cmd/ceremony_emitter.go` modified
- [x] `pkg/events/ceremony.go` modified
- [x] `.aether/ts-host/src/types.ts` modified
- [x] `.aether/ts-host/src/narrator.ts` modified
- [x] `.aether/ts-host/test/oracle-events.test.ts` created
- [x] `go test ./cmd/... -run "Oracle"` passes
- [x] `go test ./pkg/events/...` passes
- [x] `npx tsx --test test/narrator.test.ts test/oracle-events.test.ts` passes
- [x] `npx tsc --noEmit -p tsconfig.build.json` compiles cleanly
- [x] All 5 commits verified in git log
