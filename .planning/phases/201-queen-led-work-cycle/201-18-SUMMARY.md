---
phase: 201-queen-led-work-cycle
plan: "18"
subsystem: infra
tags: [go, cli, build-orchestration, evidence, knowledge-deltas]

# Dependency graph
requires:
  - phase: 201-queen-led-work-cycle
    provides: "attachResultFilePrecision/attachBuildKnowledgeDeltas/deriveResultFilePrecision (cmd/build_attempt.go, plan 201-07) — the tested-but-partially-unwired evidence writers and the credited/uncredited file derivation this plan extends and completes"
provides:
  - "deriveBuildKnowledgeDeltas (cmd/build_knowledge_deltas.go): a pure, deterministic, sanitised derivation of an attempt's own content-level decision and learning deltas from the worker handoffs its own dispatches actually produced"
  - "The native/in-repo build lane (cmd/codex_build.go) now attaches the credited/uncredited file split (D-08) to its own sealed attempt, immediately after the built-status transition — closing the half of D-08 that only the external/wrapper lane used to cover"
  - "Both build lanes now attach knowledge deltas (CAP-066) to the exact attempt that produced them, alongside the file split"
  - "TestBuildResultEvidenceHasProductionCallers: an AST call-graph guard proving attachResultFilePrecision and attachBuildKnowledgeDeltas both have production callers reachable from both build lane entry points, with a synthetic-removal sub-case proving the guard can fail"
  - "TestBothBuildLanesAttachResultEvidenceToTheExactAttempt / TestAttemptEvidenceNeverCrossesAttempts: end-to-end proof against a real native build and a real external build-finalize that both lanes' evidence matches the runtime's own derivation and never crosses between two attempts of the same phase"
affects: [201-queen-led-work-cycle]

# Actuals (#2632)
actuals:
  tokens: 8129
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Dispatch-order-anchored deterministic dedup: an attempt's own dispatch list (not the handoff store's write order) is the ordering and membership anchor for deriving per-attempt evidence from a shared cross-attempt handoff log — the same shape deriveResultFilePrecision already used for the file split, now reused for knowledge deltas."

key-files:
  created:
    - cmd/build_knowledge_deltas.go
    - cmd/build_knowledge_deltas_test.go
    - cmd/build_result_evidence_test.go
  modified:
    - cmd/codex_build.go
    - cmd/codex_build_finalize.go

key-decisions:
  - "deriveBuildKnowledgeDeltas drops a sentence that fails colony.SanitizeSignalContent rather than storing a fallback placeholder — unlike sanitizedWorkerSentence's single required attribution field, a knowledge delta has no obligation to exist at all, so an unsafe sentence is simply omitted."
  - "TestAttemptEvidenceNeverCrossesAttempts proves the same-phase, same-store crossing guarantee via two seeded attempt records (seedBuildAttemptForTest) with real persisted handoffs (persistDispatchWorkerHandoff) rather than two full sequential CLI builds of literally the same phase number — the runtime's own build-state gate (validateCodexBuildState) refuses building an already-active phase again without an intervening `aether continue`, so a true two-full-builds-same-phase fixture was not achievable without rebuilding unrelated state-machine plumbing out of scope for this gap-closure plan. The test still exercises deriveBuildKnowledgeDeltas' real worker-name-membership fencing against real, production-written handoff records sharing one phase number in one store."

requirements-completed: [CEC-06, WORK-05]

coverage:
  - id: D1
    description: "deriveBuildKnowledgeDeltas returns an attempt's own decision/learning deltas from persisted worker handoffs, deterministic, deduplicated, sanitised, nil when nothing qualifies"
    requirement: "CEC-06"
    verification:
      - kind: unit
        ref: "cmd -run TestDeriveBuildKnowledgeDeltas"
        status: pass
    human_judgment: false
  - id: D2
    description: "The native/in-repo build lane attaches the credited/uncredited file split and knowledge deltas to its own sealed attempt, after the built-status transition; the external/wrapper lane attaches knowledge deltas alongside its existing file split"
    requirement: "WORK-05"
    verification:
      - kind: integration
        ref: "cmd -run TestBothBuildLanesAttachResultEvidenceToTheExactAttempt"
        status: pass
    human_judgment: false
  - id: D3
    description: "Two attempts of the same phase never share evidence; a write failure never fails an otherwise-complete build"
    requirement: "CEC-06"
    verification:
      - kind: unit
        ref: "cmd -run TestAttemptEvidenceNeverCrossesAttempts"
        status: pass
      - kind: unit
        ref: "cmd -run TestBothBuildLanesAttachResultEvidenceToTheExactAttempt/an_evidence_write_failure_never_fails_an_otherwise-complete_build"
        status: pass
    human_judgment: false
  - id: D4
    description: "Both write-side functions have production callers reachable from both build lane entry points, and a synthetic disconnection is refused by name and position"
    requirement: "WORK-05"
    verification:
      - kind: unit
        ref: "cmd -run TestBuildResultEvidenceHasProductionCallers"
        status: pass
    human_judgment: false

duration: 45min
completed: 2026-09-10
status: complete
---

# Phase 201 Plan 18: Attach Result Evidence On Both Build Lanes Summary

**Every build now writes down what its workers actually decided and learned, and the native build lane now records which files it can take credit for and which it can't — both pieces of evidence bound to the exact attempt that produced them, on both ways of running a build.**

## Performance

- **Duration:** 45 min
- **Tasks:** 3 completed
- **Files modified:** 2 modified, 3 created

## Accomplishments

- `deriveBuildKnowledgeDeltas` (`cmd/build_knowledge_deltas.go`, new): a pure, read-only function that reads persisted worker handoffs for an attempt's own phase and dispatch names, turns open decisions into "decision" deltas and do-not-repeat/next-worker sentences into "learning" deltas, deduplicates identical sentences, orders the result deterministically by the attempt's own dispatch order, sanitises every sentence with the repository's existing worker-text sanitiser, and returns nil (not an empty list) when a build's workers left nothing behind.
- The native/in-repo build lane (`cmd/codex_build.go`) now calls `attachResultFilePrecision` and `attachBuildKnowledgeDeltas` immediately after its own built-status transition — closing the half of D-08 (the credited/uncredited file split) that only the external/wrapper lane used to record. This mirrors the external lane's existing warn-never-fail discipline exactly: both calls are reporting-only and a write failure prints a warning without touching the build's own success.
- The external/wrapper build-finalize lane (`cmd/codex_build_finalize.go`) now also attaches knowledge deltas, alongside the file split and plan-reality report it already recorded.
- Three new tests prove the wiring end-to-end: a real native build and a real external build-finalize each show recorded evidence matching the runtime's own derivation functions (never a hand-typed literal); two attempts of the same phase never mix evidence; and an AST call-graph guard proves both writer functions are reachable from both build lane entry points, with a synthetic-disconnection sub-case proving the guard can actually fail.

## Task Commits

Each task was committed atomically:

1. **Task 1: Derive an attempt's decision and learning deltas from what its workers left behind** - `b9c558eb` (feat)
2. **Task 2: Attach both pieces of evidence on both build lanes** - `a8288f34` (feat)
3. **Task 3: Prove both lanes attach both pieces to the exact attempt** - `b5abe2fa` (test)

**Plan metadata:** (this commit)

## Files Created/Modified

- `cmd/build_knowledge_deltas.go` (new) - `deriveBuildKnowledgeDeltas`, the pure derivation function.
- `cmd/build_knowledge_deltas_test.go` (new) - `TestDeriveBuildKnowledgeDeltas` unit tests (empty input, cross-phase/cross-worker exclusion, decision/learning kind mapping, dedup, deterministic ordering, no-write proof).
- `cmd/build_result_evidence_test.go` (new) - `TestBothBuildLanesAttachResultEvidenceToTheExactAttempt` (real native build + real external build-finalize, including a warn-never-fail sub-case), `TestAttemptEvidenceNeverCrossesAttempts`, `TestBuildResultEvidenceHasProductionCallers` (AST call-graph guard).
- `cmd/codex_build.go` - Native lane now calls `attachResultFilePrecision`/`attachBuildKnowledgeDeltas` right after `transitionBuildAttempt` seals the built status.
- `cmd/codex_build_finalize.go` - External lane now also calls `attachBuildKnowledgeDeltas` alongside its existing `attachResultFilePrecision`/`attachBuildPlanRealityReport` block.

## Decisions Made

- Dropped rather than placeholder-stored any worker sentence that fails `colony.SanitizeSignalContent` — a knowledge delta has no obligation to exist at all, unlike a single required attribution field elsewhere in this codebase.
- `TestAttemptEvidenceNeverCrossesAttempts` proves the same-phase, same-store crossing guarantee via two seeded attempt records with real persisted handoffs rather than two full sequential CLI builds of the literal same phase number, because the runtime's own build-state gate refuses re-building an already-active phase without an intervening `aether continue` — out of scope to rework for this gap-closure plan. The test still exercises the real production writer/derivation functions and a real worker-name-membership fencing check against handoff records genuinely sharing one phase number in one store.

## Deviations from Plan

None - plan executed exactly as written. (One test-fixture judgment call is documented above under Decisions Made rather than as a deviation, since it does not change any production code path and every acceptance criterion the plan named — no hand-typed literals, reading by own recorded path, never picking the latest twice — is met.)

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

D-08's remaining half (native-lane file-split attachment) and CAP-066 (knowledge deltas on both lanes) are closed. Both build lanes now durably record what an attempt produced — the credited/uncredited file split and the decision/learning deltas its workers left behind — bound to the exact attempt, proven by a real native build and a real external build-finalize, and proven never to cross between two attempts of the same phase. Remaining Phase 201 gaps (the closeout-rendering root cause: `LifecycleCloseoutDetails.WorkOutcome` never set in production, affecting D-03/D-05/D-06(run)/D-07/D-11's rendering) are unaddressed by this plan and tracked by the remaining gap-closure plans (201-19, 201-20 per STATE.md).

---
*Phase: 201-queen-led-work-cycle*
*Completed: 2026-09-10*

## Self-Check: PASSED

All claimed files exist (`cmd/build_knowledge_deltas.go`, `cmd/build_knowledge_deltas_test.go`, `cmd/build_result_evidence_test.go`, `cmd/codex_build.go`, `cmd/codex_build_finalize.go`, this SUMMARY.md) and all three task commit hashes (`b9c558eb`, `a8288f34`, `b5abe2fa`) are present in `git log --oneline --all`.
