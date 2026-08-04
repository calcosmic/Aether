---
phase: 162-switch-on-learning
fixed_at: 2026-08-04T14:40:27Z
review_path: .planning/phases/162-switch-on-learning/162-REVIEW.md
iteration: 1
findings_in_scope: 4
fixed: 4
skipped: 0
status: all_fixed
---

# Phase 162: Code Review Fix Report

**Fixed at:** 2026-08-04T14:40:27Z
**Source review:** .planning/phases/162-switch-on-learning/162-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 4 (CR-01, WR-01, WR-02, WR-03 — Info findings excluded by scope)
- Fixed: 4
- Skipped: 0

Every fix is locked by a test proven to fail against the pre-fix code (verified
by temporarily reverting each fix and re-running before committing). Final
gate: `go build ./...` clean, `go vet ./cmd/` clean, `go test ./cmd/ -count=1`
ok (267s, full suite), `go test ./pkg/memory/ ./pkg/learn/ -count=1` ok.

## Fixed Issues

### CR-01: Curation failure no longer runs the mutating consolidation pipeline

**Files modified:** `cmd/consolidation_lifecycle.go`, `cmd/consolidation_lifecycle_test.go`, `cmd/seal_ceremony_test.go`
**Commit:** 94c36e5e
**Applied fix:** `runSealConsolidation` now short-circuits with `Ran:false`
immediately after a curation failure, BEFORE `pipeline.RunConsolidation` can
decay, archive, or promote against stores the sentinel just flagged corrupt.
On the remaining consolidation-failure branch (pipeline ran but reported
errors), `QueenPromotedIDs` is now populated from the pipeline result so the
D-09 skip-set reflects reality even when `Ran` is false. New tests:
`TestRunSealConsolidationSentinelAbortShortCircuitsPipeline` (asserts
instincts.json is byte-identical after a sentinel abort) and
`TestSealDoesNotDoublePromoteOnCurationFailure` (corrupt pheromones.json +
dual-eligible instinct; asserts the action text appears exactly once in
QUEEN.md — reproduced the 2-occurrence double-write against the old code).

### WR-01: `QueenPromotedIDs` now carries actually-promoted IDs, not eligible IDs

**Files modified:** `pkg/memory/consolidate.go`, `pkg/memory/pipeline.go`, `pkg/memory/pipeline_test.go`, `cmd/consolidation_lifecycle.go`, `cmd/consolidation_lifecycle_test.go`
**Commit:** 6c97cca0
**Applied fix:** Added `QueenPromoted []string` to `ConsolidationResult`,
appended in `Pipeline.RunConsolidation` only after `PromoteInstinct` returns
nil. `runSealConsolidation` builds `QueenPromotedIDs` (and therefore the seal's
skip-set and promoted report) from `QueenPromoted`, keeping `QueenEligible` as
the pure eligibility report. A failed QUEEN.md write now stays out of the
skip-set (so the subordinate fallback writer can recover it) and out of
CROWNED-ANTHILL.md. New tests:
`TestPipeline_RunConsolidation_QueenPromotedTracksActualWrites` (success +
forced-write-failure subtests, via QueenPath pointed at a directory) and
`TestRunSealConsolidationQueenPromotedIDsExcludesFailedWrites`.

### WR-02: The 30-second consolidation timeout is now a real bound

**Files modified:** `cmd/consolidation_lifecycle.go`, `cmd/consolidation_lifecycle_test.go`
**Commit:** baea5253
**Applied fix:** Added `runConsolidationStageBounded` (goroutine + select on
`ctx.Done()`); both the phase-end pipeline call and the seal's curation pass
and pipeline call now run through it, returning `Ran:false, Reason:
"consolidation timed out after 30s"` with the D-05 stderr warning on expiry.
The timed-out goroutine is deliberately abandoned (documented: it may be parked
in a deadline-less flock; its captured results are never read after timeout, so
no data race — verified under `-race`). `consolidationLifecycleTimeout` became
a var so tests can shorten it. The orchestrator is constructed before the
goroutine starts so an abandoned goroutine never reads the `store` global.
New tests: `TestRunPhaseEndConsolidationTimesOutOnStaleLock` and
`TestRunSealConsolidationTimesOutOnStaleLock` — each holds a real exclusive
flock on instincts.json (pinned against GC finalizer-driven fd close, which
silently released the lock mid-test in the first draft) and proves both paths
return within the bound instead of hanging; both fail with "hung on a stale
file lock" when the wrapper is neutered.

### WR-03: Seal report now includes pipeline promotions below the 0.8 snapshot bar

**Files modified:** `cmd/codex_workflow_cmds.go`, `cmd/seal_ceremony_test.go`
**Commit:** 2d9c723d
**Applied fix:** After the seal's own promotion loop, every ID in
`sealConsolidation.QueenPromotedIDs` not already counted is folded into
`promotedInstinctNames`, so `sealEnrichment.InstinctsPromoted` and
CROWNED-ANTHILL.md's "Promoted Instincts" section report the full set that
actually reached QUEEN.md — including instincts with snapshot confidence in
[0.75, 0.8). The one-seal hive-promotion lag for instincts newly created
during the same seal is now explicitly documented in a comment as intended
behavior. New test: `TestSealReportsPipelinePromotedInstinctsBelowLocalBar`
(instinct at snapshot confidence 0.78 with 3 applications must appear in both
the count and the Promoted Instincts list).

---

_Fixed: 2026-08-04T14:40:27Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
