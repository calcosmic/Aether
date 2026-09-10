---
phase: 201-queen-led-work-cycle
plan: "07"
subsystem: queen-orchestration
tags: [go, build-attempt, result-card, work-outcome, closeout, D-08, D-07, CAP-022, CAP-066]

# Dependency graph
requires:
  - phase: 201-queen-led-work-cycle (plan 04)
    provides: "pkg/colony/work_outcome.go: colony.WorkOutcome (six-verdict vocabulary), AllWorkOutcomes(), and cmd/lifecycle_closeout.go's equal-ceremony closeout this plan's recommended action and knowledge-delta evidence hang off"
  - phase: 201-queen-led-work-cycle (plan 05)
    provides: "the two-stage receipt trust boundary (admitCoherentJobTaskReceipts/finalizeCoherentJobTaskReceiptEvidence) this plan reads dispatch.CompletedTaskIDs/TaskClaims from -- never re-derives"
provides:
  - "cmd/build_attempt.go: buildAttemptRecord gains CreditedFiles, UncreditedFiles ([]buildAttemptUncreditedFile{Path,Location}), PlanReality (*buildPlanRealityReport), and KnowledgeDeltas ([]buildAttemptKnowledgeDelta), each written only by its own narrow setter (attachResultFilePrecision, attachBuildPlanRealityReport, attachBuildKnowledgeDeltas); deriveResultFilePrecision is the pure credited/uncredited split; renderResultFilePrecisionCard is pure, idempotent presentation"
  - "cmd/codex_build_finalize.go: buildPlanRealityForDispatches/planRealityBlocksCredit/blockedPlanRealityTasks (CAP-022's plan-versus-reality comparison and credit gate), wired into runCodexBuildFinalize right after the attempt is durably sealed"
  - "cmd/lifecycle_closeout.go: recommendedActionForWorkOutcome (D-07's total one-recommendation mapping over colony.AllWorkOutcomes()) and lifecycleCloseoutKnowledgeDeltaEvidence (CAP-066's read-only, attempt-bound rendering), both wired into buildLifecycleCloseout from one shared loadLatestBuildAttempt(phase) read; LifecycleCloseout gains a RecommendedAction field and a rendered \"Recommended:\" line"
affects: [201-08, 201-10, 201-12, 201-15]

# Actuals (#2632)
actuals:
  tokens: 12761
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Narrow setters, one per new evidence field: attachResultFilePrecision/attachBuildPlanRealityReport/attachBuildKnowledgeDeltas each touch exactly their own field and nothing else on buildAttemptRecord (not Status, not Dispatches, not Claims), mirroring attachBuildFreeCheckReport's and attachCheckFixAttempt's established discipline -- and each is non-fatal (warn, never fail an otherwise-complete build) when the write itself fails."
    - "Credit is read, never re-decided: deriveResultFilePrecision computes CreditedFiles/UncreditedFiles purely from the already-resolved dispatches the two-stage receipt boundary produced (dispatch.Outputs for a whole-success dispatch; dispatch.TaskClaims/CompletedTaskIDs for a grouped job) -- it never re-derives what counts as done, it only reports the existing decision's file-level consequence honestly, including naming an isolated workspace's exact path and branch for an uncredited file in worktree mode."
    - "Plan-reality blocking is scoped to the new result-card surface, not the phase-advancement engine: buildPlanRealityForDispatches/planRealityBlocksCredit compare a task's plan-declared artifacts (declaredPathsForTask) against the repository right now (rootBackedArtifactEvidence) and feed a blockedTasks set into deriveResultFilePrecision -- CAP-022's credit gate is enforced on the credited/uncredited file split this plan owns, deliberately not by mutating completedBuildTaskIDs or dispatch.Status, which many other phase-advancement paths depend on and were out of this plan's own file scope to re-verify."
    - "One shared attempt read serves two features: buildLifecycleCloseout loads the phase's attempt exactly once (loadLatestBuildAttempt) and reuses it for both recommendedActionForWorkOutcome (D-07) and lifecycleCloseoutKnowledgeDeltaEvidence (CAP-066) -- never two independent lookups, and the recommendation is a total switch over colony.AllWorkOutcomes() so an undeclared verdict is refused by name rather than falling through to a shared default."

key-files:
  created:
    - cmd/result_file_precision_test.go
  modified:
    - cmd/build_attempt.go
    - cmd/codex_build_finalize.go
    - cmd/lifecycle_closeout.go
    - cmd/spend_cost_line_test.go

key-decisions:
  - "CAP-022's absent-declared-artifact credit block is enforced at the result-card precision layer (deriveResultFilePrecision's blockedTasks parameter) rather than by rewriting completedBuildTaskIDs or dispatch.Status -- the core phase-advancement crediting engine used across many other build/continue paths this plan's own file scope did not include. A blocked task's files are excluded from CreditedFiles and named uncredited with the blocking reason; a flat whole-success dispatch that cannot be attributed to one task is conservatively withheld in full rather than guessed at."
  - "recommendedActionForWorkOutcome invents no new command vocabulary: partial reuses buildUnfinishedRetryRedispatchCommand (the exact recovery command already stored on the attempt record), blocker points at the already-locked `aether unblock --dispatch` Fixer intake, interrupted points at the sole `aether resume` recovery door Phase 199 locked, timeout follows the attempt's own recorded RecoveryCommand, and success/no_change point at `aether continue`."
  - "CAP-066's knowledge-delta storage/rendering (attachBuildKnowledgeDeltas, lifecycleCloseoutKnowledgeDeltaEvidence) is built and fully tested but has no live production writer yet in this plan -- there is no existing pipeline in this repository that currently produces a structured 'decision' or 'learning' delta payload to attach at attempt-finalize time. This mirrors 201-05's own precedent (buildUnverifiedCloseoutDetails/buildVerifiedCloseoutDetails: built and tested, wiring deferred) -- a future plan needs to name the real source of these deltas (e.g. the memory-capture pipeline) and call attachBuildKnowledgeDeltas from it."
  - "Two pre-existing/newly-introduced test fixture helpers (seedSpendElapsedAttemptForTest from plan 201-06, and this plan's own new seedBuildAttemptForTest) both wrote a raw attempt-record-plus-latest-pointer sequence, which is exactly the shape TestBuildStartLegacyHelpersRetired200 (cmd/build_attempt_external_test.go) refuses. Confirmed via a read-only git-archive export of the pre-plan commit that this test was already failing on main before this plan started, purely from the 201-06 helper -- fixed both (Rule 1) by extracting the pointer write into a new shared markLatestBuildAttemptForTest helper, so neither seeding function's own body both references latestBuildAttemptPointerPath and issues more than one store write."

requirements-completed: []
# WORK-05 and CEC-06 (this plan's declared requirements) are each also
# declared by sibling 201-* plans (WORK-05: 201-06 [complete]/10/15; CEC-06:
# 201-02 [complete]/04 [complete]/06 [complete]/12/15) and therefore stay
# open per the shared-ID gate (#2388) until every declaring plan has a
# SUMMARY.md -- confirmed via `gsd-tools query requirements.ready-ids`,
# correct and expected, not a gap in this plan's own work.

coverage:
  - id: D1
    description: "A result card names both the files credited to completed tasks and the files that were edited but credited to nothing, with where those uncredited files live -- and an uncredited file is never absorbed into a success, never deleted, and never hidden behind a drill-down"
    requirement: WORK-05
    verification:
      - kind: unit
        ref: "cmd/result_file_precision_test.go#TestResultCardNamesCreditedAndOrphanedFiles"
        status: pass
      - kind: unit
        ref: "cmd/result_file_precision_test.go#TestOrphanedEditsAreKeptAndLocated"
        status: pass
    human_judgment: false
  - id: D2
    description: "Rendering a result card twice for the same attempt produces the same card and writes nothing; two closeouts for two different attempts never merge their file lists"
    requirement: WORK-05
    verification:
      - kind: unit
        ref: "cmd/result_file_precision_test.go#TestResultCardRenderIsIdempotent"
        status: pass
      - kind: unit
        ref: "cmd/result_file_precision_test.go#TestTwoAttemptsNeverMergeFileLists"
        status: pass
    human_judgment: false
  - id: D3
    description: "After a non-success outcome the Queen recommends exactly one next action matched to that outcome, with a one-sentence reason and the alternatives listed beneath it, derived from the recorded verdict -- never from the rendered text -- and total across every declared verdict"
    requirement: CEC-06
    verification:
      - kind: unit
        ref: "cmd/result_file_precision_test.go#TestEveryVerdictHasExactlyOneRecommendedAction"
        status: pass
      - kind: unit
        ref: "cmd/result_file_precision_test.go#TestPartialRecommendationRetriesOnlyUnfinishedTasks"
        status: pass
    human_judgment: false
  - id: D4
    description: "Completion evidence records what the plan said should exist against what is actually present, and an absent declared artifact blocks credit rather than being noted and ignored"
    requirement: WORK-05
    verification:
      - kind: unit
        ref: "cmd/result_file_precision_test.go#TestAbsentDeclaredArtifactBlocksCredit"
        status: pass
    human_judgment: false
  - id: D5
    description: "Content-level decision and learning deltas are bound to the exact attempt that produced them and shown at the decision that consumes them, without ever writing during that rendering"
    requirement: CEC-06
    verification:
      - kind: unit
        ref: "cmd/result_file_precision_test.go#TestKnowledgeDeltasAreAttemptBound"
        status: pass
    human_judgment: true
    rationale: "The storage and read-only rendering mechanism (attachBuildKnowledgeDeltas, lifecycleCloseoutKnowledgeDeltaEvidence) is fully built and unit-proven, but no production call site in this repository yet produces a real decision/learning delta payload to attach at finalize time -- there is no existing pipeline this plan's own scope covers that generates that content. A human should confirm that gap is acceptable scope for this plan (matching 201-05's own precedent for a built-but-unwired capability) rather than a missed wiring step."

duration: 40min
completed: 2026-09-10
status: complete
---

# Phase 201 Plan 07: Result-Card Precision, One Recommended Action, and Attempt-Bound Evidence Summary

**A build result card now names exactly which edits counted and which were credited to nothing (with where the uncredited ones live), the Queen recommends exactly one next action derived from the recorded work verdict, and CAP-022/CAP-066 evidence is provably bound to the one attempt that produced it.**

## Performance

- **Duration:** 40 min
- **Started:** 2026-09-10T14:24:00Z (approx., immediately following 201-06)
- **Completed:** 2026-09-10T15:04:00Z (approx.)
- **Tasks:** 3
- **Files modified:** 5 (1 created, 4 modified)

## Accomplishments

- `buildAttemptRecord` (`cmd/build_attempt.go`) gained four new fields, each written by exactly one narrow setter: `CreditedFiles`/`UncreditedFiles` (via `attachResultFilePrecision`), `PlanReality` (via `attachBuildPlanRealityReport`), and `KnowledgeDeltas` (via `attachBuildKnowledgeDeltas`). None of the setters touch `Status`, `Dispatches`, or `Claims`.
- `deriveResultFilePrecision` is the pure, read-only computation behind D-08: a whole-success dispatch's reported outputs are all credited; a grouped job's outputs are credited only for the tasks its receipts actually finalised (`dispatch.TaskClaims`/`CompletedTaskIDs`); everything else the attempt touched is uncredited, named with its location (`workspace <path> on branch <branch>` in isolated-workspace mode, `"the repository"` otherwise) -- never deleted, never folded into the credited set. `renderResultFilePrecisionCard` is pure presentation, proven byte-identical across two renders of the same attempt with an unchanged fixture-directory digest, and proven never to name another attempt's files.
- `buildPlanRealityForDispatches` (`cmd/codex_build_finalize.go`) implements CAP-022: for every task an attempt's dispatches cover, it compares the plan's declared artifacts (`declaredPathsForTask`) against the repository right now (`rootBackedArtifactEvidence`, the same root-backed checker the two-stage receipt boundary already uses). `planRealityBlocksCredit` names a task with a missing declared artifact, and `blockedPlanRealityTasks` feeds that set into `deriveResultFilePrecision` so the absence blocks credit at the result-card layer rather than being noted and ignored.
- `recommendedActionForWorkOutcome` (`cmd/lifecycle_closeout.go`) implements D-07: a total, one-recommendation mapping over `colony.AllWorkOutcomes()`, proven by iterating the runtime's own declared set rather than restating the six cases, with an undeclared verdict refused by name. Partial retries only the unfinished tasks (the attempt's own `buildUnfinishedRetryRedispatchCommand`), blocker names the recorded blocker and points at the already-locked `aether unblock --dispatch`, interrupted points at the sole `aether resume` recovery door Phase 199 locked, timeout follows the attempt's own recorded recovery command, and success/no_change point at `aether continue` -- no new command spelling is invented.
- `lifecycleCloseoutKnowledgeDeltaEvidence` implements CAP-066's read-only rendering: an attempt's `KnowledgeDeltas` become `colony.LifecycleEvidence` entries keyed to that attempt's own ID, proven never to write and never to leak between two attempts' cards. Both the recommendation and the knowledge-delta evidence are wired into `buildLifecycleCloseout` from one shared `loadLatestBuildAttempt(phase)` read; `LifecycleCloseout` gained a `RecommendedAction` field and a rendered "Recommended: ... — ..." line.
- Both the credited/uncredited file split and the plan-reality report are attached inside `runCodexBuildFinalize` right after the attempt is durably sealed (`buildAttemptBuilt`/`buildAttemptPartial`), from the exact same fully-resolved `dispatches` the two-stage receipt boundary just finalised -- this is a live production wiring point (the external/wrapper build-finalize lane), not built-but-unwired infrastructure.
- Added `cmd/result_file_precision_test.go` with all 8 named tests plus fixture helpers (`seedBuildAttemptForTest`, `markLatestBuildAttemptForTest`, `hashDirForTest`).

## Task Commits

1. **Tests for Tasks 1-3 (RED)** - `a09be79d` (test) -- all 8 named tests, referencing symbols that did not exist yet; does not compile on its own, proving the tests are real before the implementation.
2. **Implementation for Tasks 1-3 (GREEN)** - `89552387` (feat) -- `cmd/build_attempt.go`, `cmd/codex_build_finalize.go`, `cmd/lifecycle_closeout.go`.
3. **Ratchet fix (Rule 1)** - `68443bd3` (fix) -- `cmd/result_file_precision_test.go`, `cmd/spend_cost_line_test.go`.

**Plan metadata:** committed alongside this summary.

_Note on granularity: Tasks 1-3 could not be cleanly split into three separate RED/GREEN pairs by git hunk -- all three edit overlapping regions of the same three production files (the struct definition, the finalize wiring point, and the closeout builder), the same pattern already documented in `193-04-SUMMARY.md` and `195-05-SUMMARY.md`. Instead the whole plan followed one RED-then-GREEN sequence (commits 1-2), plus a third commit fixing a ratchet violation discovered while proving the implementation green (commit 3)._

## Files Created/Modified

- `cmd/build_attempt.go` - `buildAttemptUncreditedFile`, `buildAttemptRecord.CreditedFiles/UncreditedFiles/PlanReality/KnowledgeDeltas`, `attachResultFilePrecision`, `attachBuildPlanRealityReport`, `attachBuildKnowledgeDeltas`, `buildAttemptWorktreeLocation`, `deriveResultFilePrecision`, `renderResultFilePrecisionCard`
- `cmd/codex_build_finalize.go` - `buildTaskPlanRealityEntry`, `buildPlanRealityReport`, `buildAttemptKnowledgeDelta`, `buildPlanRealityForDispatches`, `planRealityBlocksCredit`, `blockedPlanRealityTasks`, and the `runCodexBuildFinalize` wiring that attaches both after the attempt is sealed
- `cmd/lifecycle_closeout.go` - `LifecycleCloseoutRecommendedAction`, `LifecycleCloseout.RecommendedAction`, `recommendedActionForWorkOutcome`, `lifecycleCloseoutKnowledgeDeltaEvidence`, the shared attempt-read in `buildLifecycleCloseout`, and the rendered "Recommended:" line in `renderLifecycleCloseout`
- `cmd/result_file_precision_test.go` (new) - all 8 named tests plus `seedBuildAttemptForTest`, `markLatestBuildAttemptForTest`, `hashDirForTest`
- `cmd/spend_cost_line_test.go` - `seedSpendElapsedAttemptForTest` refactored to delegate its pointer write to the new shared `markLatestBuildAttemptForTest` (ratchet fix, see Deviations)

## Decisions Made

See `key-decisions` in the frontmatter for full reasoning; in short: CAP-022's credit block is enforced at the new result-card layer rather than by rewriting the core phase-advancement crediting engine; the recommended-action vocabulary reuses only already-locked commands; CAP-066's storage/rendering is built and tested but has no live production writer yet (no existing pipeline in this repository currently produces a structured delta payload); and a pre-existing ratchet violation from plan 201-06, plus this plan's own new instance of the same shape, were both fixed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Two test fixture helpers tripped `TestBuildStartLegacyHelpersRetired200`**
- **Found during:** Task 1, while running the plan's own targeted `<verify>` commands alongside a broader `go test ./cmd -run 'TestBuild'` sweep
- **Issue:** `TestBuildStartLegacyHelpersRetired200` (`cmd/build_attempt_external_test.go`) refuses any function whose body both references `latestBuildAttemptPointerPath` and issues more than one `store.SaveJSON`/`AtomicWrite` call -- the old multi-write build-start adapter shape. This plan's own new `seedBuildAttemptForTest` fixture helper matched that shape. Confirmed (via a read-only `git archive` export of the commit immediately before this plan's first commit, so no working-tree mutation) that the pre-existing `seedSpendElapsedAttemptForTest` in `cmd/spend_cost_line_test.go` (plan 201-06) already matched the identical shape and was already failing this test before this plan started.
- **Fix:** Extracted the pointer write into a new shared `markLatestBuildAttemptForTest` helper (`cmd/result_file_precision_test.go`) and had both seeding functions delegate to it, so neither function's own body carries both traits at once.
- **Files modified:** `cmd/result_file_precision_test.go`, `cmd/spend_cost_line_test.go`
- **Verification:** `go test ./cmd -run '^TestBuildStartLegacyHelpersRetired200$'` passes; the full `TestBuild`-matched sweep (215s) that surfaced this was the only failure in that sweep, and re-running every directly related targeted suite (the 8 new tests, `Spend`, `Closeout|WorkOutcome`, `BuildFinalize|ExternalBuild|PartialBuildRetry|MergedDispatch`, full `go test ./pkg/...`) passed clean both before and after this fix.
- **Committed in:** `68443bd3`

---

**Total deviations:** 1 auto-fixed (1 bug -- pre-existing ratchet violation, made visible and fixed alongside this plan's own new instance of the same pattern). **Impact on plan:** Necessary for `go test ./cmd`'s own build-start ratchet to stay green; no unrelated behavior was touched.

## Issues Encountered

- The unfiltered, broad `go test ./cmd -run 'TestBuild'` sweep (215s) is exactly the kind of full-suite run the owner's stated verification policy asks the executor to avoid by default (targeted `-run` filters preferred; this repository's `go test ./cmd` hits a documented pre-existing environment ceiling under parallel load). It was run once here specifically because it is the ratchet's own enclosing sweep, and it earned its cost by finding the real, fixable issue above -- confirmed fixed by a second full `TestBuild` sweep plus every directly relevant narrower suite.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The credited/uncredited file split (D-08) and CAP-022's plan-reality report are wired into the external/wrapper build-finalize lane (`cmd/codex_build_finalize.go`'s `runCodexBuildFinalize`) -- the live production path for `aether build-finalize`. **Open gap for a later plan:** the native/in-repo build lane's own attempt-sealing point (`cmd/codex_build.go` around its `transitionBuildAttempt(attemptRel, buildAttemptBuilt, ...)` call) does not yet call `attachResultFilePrecision`/`attachBuildPlanRealityReport`, so a same-checkout (non-external) build does not yet get these fields populated.
- **Open gap for a later plan (CAP-066):** `attachBuildKnowledgeDeltas`/`lifecycleCloseoutKnowledgeDeltaEvidence` are fully built and tested but have no live production writer -- a future plan needs to identify the real source of a "content-level decision or learning delta" (most likely this repository's existing memory-capture/consolidation pipeline) and call `attachBuildKnowledgeDeltas` from it at attempt-finalize time.
- `recommendedActionForWorkOutcome`/`lifecycleCloseoutKnowledgeDeltaEvidence` render through `buildLifecycleCloseout`, which -- per `201-05-SUMMARY.md` and `201-06-SUMMARY.md`'s own documented gap -- still has no production call site anywhere in the tree that sets `LifecycleCloseoutDetails.WorkOutcome` yet; this plan inherits, and does not close, that pre-existing wiring gap.
- `go build ./...` and `go vet ./...` are clean. Every task-level `<verify>` command from `201-07-PLAN.md` passes individually. Targeted regression sweeps (`Spend`, `Closeout|WorkOutcome`, `BuildFinalize|ExternalBuild|PartialBuildRetry|MergedDispatch`, full `go test ./pkg/...`) all pass. The one broad `TestBuild`-matched sweep failure found (pre-existing, see Deviations) is fixed and reconfirmed.
- Ready for `201-08-PLAN.md`.

---
*Phase: 201-queen-led-work-cycle*
*Completed: 2026-09-10*

## Self-Check: PASSED

- Created file verified present on disk: `cmd/result_file_precision_test.go`.
- Task commit hashes verified present in `git log`: `a09be79d`, `89552387`, `68443bd3`.
- Re-ran plan-level `<verify>` commands:
  - `go test ./cmd -run '^(TestResultCardNamesCreditedAndOrphanedFiles|TestOrphanedEditsAreKeptAndLocated|TestResultCardRenderIsIdempotent|TestTwoAttemptsNeverMergeFileLists)$' -count=1` -- PASS
  - `go test ./cmd -run '^TestCoherentJobReceipt' -count=1` -- PASS
  - `go test ./cmd -run '^(TestEveryVerdictHasExactlyOneRecommendedAction|TestPartialRecommendationRetriesOnlyUnfinishedTasks)$' -count=1` -- PASS
  - `go test ./cmd -run '^(TestAbsentDeclaredArtifactBlocksCredit|TestKnowledgeDeltasAreAttemptBound)$' -count=1` -- PASS
  - `go vet ./cmd` -- clean
- `go build ./...` -- clean. `go vet ./...` -- clean.
