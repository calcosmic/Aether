---
phase: 200-iterative-planning
plan: 46
subsystem: memory
tags: [go, consolidation, queen, repository-containment, mutation-session, tdd]

# Dependency graph
requires:
  - phase: 200-27
    provides: physical repository authority and strict no-follow storage containment
  - phase: 200-28
    provides: repository mutation sessions and journaled lifecycle transactions
  - phase: 200-42
    provides: command lifecycle compatibility baseline
provides:
  - typed Queen-promotion callback separated from the lifecycle data store
  - contained repository-local `.aether/QUEEN.md` promotion writes
  - truthful failed-write accounting with mutation-free dry runs
affects: [colony-prime, phase-end-consolidation, seal-consolidation, final-gate]

# Tech tracking
tech-stack:
  added: []
  patterns: [typed filesystem-authority callback, repository-relative transaction target, captured-store callback state]

key-files:
  created: []
  modified:
    - pkg/memory/pipeline.go
    - pkg/memory/pipeline_test.go
    - pkg/learn/wrappers.go
    - cmd/graph_consolidation_cmds.go
    - cmd/consolidation_lifecycle.go
    - cmd/consolidation_promotion_target_test.go

key-decisions:
  - "Memory retains eligibility policy while cmd owns the repository authority and persists only the typed, already-approved instinct."
  - "The callback captures the lifecycle data root when the pipeline is assembled so a timed-out goroutine never consults mutable package-global store state."
  - "A refused Queen write stays absent from QueenPromoted and visible in pipeline errors without turning best-effort consolidation into a lifecycle gate."

patterns-established:
  - "Sibling-file boundary: never widen the `.aether/data` Store; commit `.aether/QUEEN.md` as a repository-relative lifecycle transaction target."
  - "Compatibility callback: nil preserves package-local relative-store behavior, while command callers inject repository authority."

requirements-completed: [CEC-03, PLAN-06]

# Metrics
duration: 27min
completed: 2026-09-09
---

# Phase 200 Plan 46: Contained Local-Queen Promotion Summary

**Eligible instincts now reach the repository-local `.aether/QUEEN.md` through a journaled repository transaction, while linked boundaries and failed writes are refused and never reported as successful promotions.**

## Performance

- **Duration:** 27 min
- **Started:** 2026-09-09T10:25:38Z
- **Completed:** 2026-09-09T10:52:28Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Added a typed Queen instinct promotion callback to the memory pipeline while retaining the existing relative-store path whenever no callback is injected.
- Routed command-level promotion into `.aether/QUEEN.md` through the existing repository mutation session and lifecycle transaction rather than widening `.aether/data` store authority.
- Moved legacy `## Instincts` self-healing into the same in-memory derivation and repository transaction, leaving no raw-write fallback after containment refusal.
- Kept dry-run services on a harmless relative path and proved phase-end dry run performs no mutation.
- Proved the promoted entry is read back from local Queen wisdom into the colony-prime worker prompt, independently of `instincts.json` injection.
- Proved linked `.aether` boundaries and non-file Queen targets fail closed, remain visible as errors, and stay out of `QueenPromoted`.

## Task Commits

Each TDD gate and implementation unit was committed atomically:

1. **Task 1 RED: Define typed writer and failure-accounting behavior** - `845b7a51` (test)
2. **Task 1 GREEN: Inject the typed Queen promotion writer** - `7343c686` (feat)
3. **Task 2 RED: Define contained local-Queen integration behavior** - `4ad318ce` (test)
4. **Rule 3 integration glue: Forward the callback through pkg/learn** - `91c04c20` (fix)
5. **Task 2 GREEN: Commit local-Queen promotions through repository authority** - `612a7eb6` (fix)
6. **Task 2 REFACTOR: Make the Queen-promotion key link explicit** - `5cd84344` (refactor)

## Files Created/Modified

- `pkg/memory/pipeline.go` - Adds the typed callback and records injected writer failures without false promotion success.
- `pkg/memory/pipeline_test.go` - Proves exact-once callback delivery, nil-callback compatibility, and truthful failure accounting.
- `pkg/learn/wrappers.go` - Forwards the callback type and field through the cmd-facing facade without adding policy.
- `cmd/graph_consolidation_cmds.go` - Injects the contained writer and keeps all store-facing Queen paths relative.
- `cmd/consolidation_lifecycle.go` - Derives local Queen content in memory and commits it through a repository mutation session.
- `cmd/consolidation_promotion_target_test.go` - Proves local target/readback, dry-run immutability, linked-boundary refusal, and failed-write accounting.

## Decisions Made

- Kept confidence, eligibility, and promotion selection entirely inside `pkg/memory`; the command callback only formats the existing entry shape and persists it.
- Captured `store.BasePath()` in the callback closure at pipeline construction time so bounded lifecycle goroutines do not dereference mutable global state after a timeout.
- Preserved consolidation's enrichment-not-gate behavior: callback errors are logged and retained in `ConsolidationResult.Errors`, but lifecycle summaries continue to report unrelated decay/archive work honestly.
- Kept `consolidationQueenPath()` as a read-only inspection resolver for compatibility tests; no data-store operation receives its absolute sibling path.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Forwarded the new callback through `pkg/learn`**

- **Found during:** Task 2 RED compilation
- **Issue:** Command code imports the `pkg/learn` facade, whose mirrored `PipelineConfig` did not expose the new memory callback; the planned command wiring could not compile without passthrough glue.
- **Fix:** Added a type alias and field forwarding only. No promotion, confidence, eligibility, or filesystem policy was added to the wrapper.
- **Files modified:** `pkg/learn/wrappers.go`
- **Commit:** `91c04c20`

**2. [Rule 1 - Bug] Preserved non-blocking lifecycle semantics for visible Queen failures**

- **Found during:** Task 2 compatibility verification
- **Issue:** Surfacing callback failures in `ConsolidationResult.Errors` caused seal consolidation to classify otherwise-completed decay/archive work as a failed run, contradicting its established enrichment-not-gate contract.
- **Fix:** Kept Queen errors visible at the pipeline boundary and excluded failed IDs from success, while filtering that already-reported error only when computing phase/seal lifecycle completion.
- **Files modified:** `cmd/consolidation_lifecycle.go`
- **Commit:** `612a7eb6`

## Issues Encountered

- The original focused integration tests failed exactly at the regression: the sentinel never appeared in local Queen wisdom because the strict `.aether/data` store refused the absolute sibling destination.
- A voluntary repository-wide `go test ./...` run was stopped on orchestrator direction because later Plans 47-54 own known standalone failures; the scoped Plan 46 gates and adjacent phase/seal compatibility suite are the meaningful verification for this repair.
- The installed state-advance handler dropped the Plan 45 `current_phase`, `current_phase_name`, and `state_head` frontmatter fields while serializing STATE.md. They were restored before commit, and `state_head` is anchored to this plan's metadata commit in the follow-up provenance commit.

## TDD Gate Compliance

- **Task 1 RED:** `845b7a51` failed to compile because `PipelineConfig` had no typed writer callback.
- **Task 1 GREEN:** `7343c686` passed both new tests and four existing nil-callback compatibility tests.
- **Task 2 RED:** `4ad318ce` first failed to compile at the missing facade field, then failed at the nil command callback after the passthrough glue landed.
- **Task 2 GREEN:** `612a7eb6` passed the three exact plan tests, both hostile-boundary tests, the combined plan regex, and the adjacent phase/seal lifecycle suite.
- **REFACTOR:** `5cd84344` made the verifier-facing Queen-promotion boundary explicit without changing behavior; its two focused integration tests remained green.

## Verification

- `go test ./pkg/memory -run '^TestPipeline(InjectedQueenPromotion|QueenPromotionFailureAccounting)200$' -count=1` - PASS (2 tests).
- Existing nil-callback pipeline compatibility selection - PASS (4 tests).
- `go test ./cmd -run '^Test(ConsolidationPromotesIntoLocalQueen|PromotedInstinctReachesWorkerPrompt|ConsolidationPhaseEndDryRunDoesNotMutate)$' -count=1` - PASS (3 tests).
- `go test ./cmd -run '^Test(ConsolidationRefusesLinkedLocalQueenBoundary200|ConsolidationQueenWriteFailureIsNotPromoted200)$' -count=1` - PASS (2 tests).
- `go test ./pkg/memory ./cmd -run 'Test(Consolidation|PromotedInstinct)' -count=1` - PASS (11 tests).
- `go test ./cmd -run '^TestRun(Seal|PhaseEnd)Consolidation' -count=1` - PASS (11 tests).
- `test ! -e .aether/data/QUEEN.md` - PASS.
- `git diff --check` - PASS.

## Known Stubs

None.

## Threat Flags

None. The only filesystem trust-boundary change is the repository-local Queen target explicitly modeled as T-200-46-01; it uses the existing no-follow mutation authority and introduces no new endpoint, schema, credential path, or broader store permission.

## User Setup Required

None - no dependency, credential, service, or configuration change is required.

## Next Phase Readiness

- Local Queen promotions are reachable by worker context again without weakening repository containment.
- Plans 47-54 can continue closing their owned standalone and full-gate failures; Plan 46 introduces no Phase 204 learning-governance policy.

## Self-Check: PASSED

- All six modified implementation/test files and this summary exist.
- Task commits `845b7a51`, `7343c686`, `4ad318ce`, `91c04c20`, `612a7eb6`, and `5cd84344` exist in repository history.
- Both required key-link patterns resolve, all scoped verification gates pass, and `.aether/data/QUEEN.md` is absent.
- `.planning/config.json`, `.gsd/`, and untracked Phase 199 `199-PATTERNS.md` were neither staged nor changed by Plan 46.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-09*
---
