---
phase: 195-coherent-jobs
plan: 06
subsystem: build-orchestration
tags: [go, completion-evidence, task-receipts, honest-partial-credit, external-lane, tdd]

requires:
  - phase: 195-coherent-jobs
    provides: the shared two-stage receipt trust boundary (admitCoherentJobTaskReceipts / finalizeCoherentJobTaskReceiptEvidence, cmd/coherent_job_receipts.go) and native partial credit (195-04)
provides:
  - The external/wrapper build-finalize lane wired onto the SAME shared receipt boundary the native lane uses (no second, external-only validator)
  - A phantom-build provenance carve-out for a failed/blocked/timeout worker carrying genuine, evidenced task_receipts
  - Colony-state gating (allSelectedBuildTasksCredited) so a partially-credited grouped external job never falsely reports BUILT
  - last-build-claims.json and result-collection.json diagnostics that include receipt-credited task claims and name credited/unfinished task IDs separately
affects: [195-07, 195-08]

actuals:
  tokens: 13000
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns: [external lane reuses the native lane's two-stage trust boundary unchanged, provenance carve-out for evidenced partial receipts, BUILT-transition gated on full task-set credit]

key-files:
  created: []
  modified:
    - cmd/provenance.go
    - cmd/codex_build_finalize.go
    - cmd/codex_build_finalize_test.go
    - cmd/wrapper_bundled_completion_test.go
    - cmd/merged_dispatch_task_credit_test.go
    - cmd/no_change_vocabulary_wiring_test.go

key-decisions:
  - "No second, external-only receipt validator was written. admitCoherentJobTaskReceipts and finalizeCoherentJobTaskReceiptEvidence (195-04) already operate on the lane-neutral codexBuildDispatch struct; the entire missing piece was wiring resolveCoherentJobDispatchReceipts into runCodexBuildFinalize itself, exactly as it was already wired into the native lane's executeCodexBuildDispatches."
  - "The provenance phantom-build guard (SAFE-01) is relaxed only for a worker whose task_receipts contain at least one structurally credible entry (successful receipt status, non-empty summary, passing handoff with a concrete commands_run) -- an entirely failed job with ZERO receipts is still rejected outright, unchanged, matching the existing TestValidateBuildProvenance_AllFailed invariant."
  - "BUILT-transition gating (allSelectedBuildTasksCredited) falls back to the union of every dispatch's own covered tasks when the caller passed no explicit selectedTaskIDs (the ordinary 'no --tasks filter' build) -- an early version that used selectedTaskIDs alone trivially returned true on an empty selection, silently defeating the gate for the most common build shape."
  - "A grouped job that isn't fully credited leaves updatedState.State at EXECUTING (never invents a new state) -- applyCodexBuildState already sets that value unconditionally; the fix was skipping the unconditional override to StateBUILT, in both the in-memory projection and the atomic commitBuildFinalizeState closure."
  - "The attempt journal's own final transition (buildAttemptBuilt) was deliberately left unconditional, out of scope for this plan -- gating it too would touch buildAttemptCompletionSealed/idempotent-replay semantics that plan 195-07 (recovery/retry) is the intended owner of. Replaying an identical partial packet today safely errors ('colony state has advanced') rather than duplicating credit, since state.State no longer equals BUILT for that case; a smoother idempotent replay path is deferred to 195-07."
  - "'An admitted candidate whose root artifact/evidence is absent is removed during finalization and remains pending' is not re-tested at the full build-finalize entrypoint: validateExternalWorkerResultClaimPaths (structural validation, runs before merge/receipt resolution) already refuses ANY completion packet whose own files_modified names a path that does not exist in root at submission time, on any lane -- so a 'vanishes between admission and finalization' packet cannot reach that entrypoint at all. The behavior itself is already locked, lane-neutrally, by TestCoherentJobReceiptFinalization's 'a candidate whose file is missing from root is not credited' subtest (cmd/coherent_job_receipts_test.go, 195-04), which this plan re-verified passing rather than duplicating."

patterns-established:
  - "External/wrapper build-finalize reuses the native lane's two-stage receipt boundary unchanged, only inserting a parallel-mode gate (skip for worktree, run for in-repo) -- plan 195-08 inserts its own sync step between the same two stages for the worktree lane, exactly as designed in 195-04."

requirements-completed: [JOBS-02]

coverage:
  - id: D1
    description: "The external/wrapper build-finalize lane resolves task_receipts through the same admitCoherentJobTaskReceipts stage the native lane uses, producing candidate claims and normalized sync paths with no completion credit exposed by mergeExternalBuildResults itself"
    requirement: JOBS-02
    verification:
      - kind: unit
        ref: "cmd/wrapper_bundled_completion_test.go#TestExternalTaskReceiptsUseSharedAdmission"
        status: pass
    human_judgment: false
  - id: D2
    description: "A four-of-six failed grouped external completion, run through the real aether build-finalize entrypoint, credits exactly the four receipted tasks, leaves the other two pending, and names both credited and unfinished task IDs in result-collection.json"
    requirement: JOBS-02
    verification:
      - kind: integration
        ref: "cmd/wrapper_bundled_completion_test.go#TestWrapperGroupedFailureCreditsFourOfSix, cmd/codex_build_finalize_test.go#TestExternalGroupedPartialPersistsExactTaskState"
        status: pass
    human_judgment: false
  - id: D3
    description: "A partially-credited grouped external job never reports the colony as BUILT (stays EXECUTING with the phase in_progress); a whole-success grouped job with no receipts still reaches BUILT with every covered task complete; an entirely failed job with zero receipts is rejected outright with no state change"
    requirement: JOBS-02
    verification:
      - kind: integration
        ref: "cmd/codex_build_finalize_test.go#TestExternalGroupedFullSuccessCreditsAll, #TestExternalGroupedFailureWithoutReceiptsCreditsNone, #TestExternalPartialClaimsMatchTaskState"
        status: pass
    human_judgment: false
  - id: D4
    description: "A duplicate/unknown receipt entry mixed into the same worker submission is refused by named rule without erasing the other valid receipts in that submission; a worktree-backed completion is left with no pre-sync CompletedTaskIDs, preserving TaskReceipts for plan 195-08"
    requirement: JOBS-02
    verification:
      - kind: unit
        ref: "cmd/wrapper_bundled_completion_test.go#TestWrapperTaskReceiptRefusalsDoNotEraseValidReceipts, cmd/codex_build_finalize_test.go#TestExternalPartialClaimsMatchTaskState/worktree-backed"
        status: pass
    human_judgment: false

duration: 100min
completed: 2026-08-27
status: complete
---

# Phase 195 Plan 06: External Lane Task-Receipt Trust Boundary Summary

**The external/wrapper `aether build-finalize` entrypoint now resolves task_receipts through the exact same admission/finalization boundary the native build path uses, so a real wrapper-submitted four-of-six failed grouped completion credits exactly four tasks, never falsely reports the colony as BUILT, and persists diagnostics naming both what finished and what didn't.**

## Performance

- **Duration:** ~100 min
- **Completed:** 2026-08-27
- **Tasks:** 2
- **Files modified:** 6 (0 created, 6 modified)

## Accomplishments

- Wired `resolveCoherentJobDispatchReceipts` (the shared `admitCoherentJobTaskReceipts` / `finalizeCoherentJobTaskReceiptEvidence` two-stage boundary from 195-04) into `runCodexBuildFinalize`, for in-repo completions only — a worktree-backed completion skips it, leaving `TaskReceipts` intact for plan 195-08's sync-first adapter.
- Extended `validateBuildProvenance`'s phantom-build guard (SAFE-01) with a carve-out: a failed/blocked/timeout/interrupted worker carrying at least one structurally credible task receipt is real provenance, not a phantom build — while a worker with zero receipts (or only hollow ones) is still rejected exactly as before.
- Added `allSelectedBuildTasksCredited` (plus `unfinishedBuildTaskIDs`/`creditedBuildTaskIDs` for diagnostics) so both the in-memory state projection and the atomic `commitBuildFinalizeState` commit only advance the colony to `BUILT` when every task the build was responsible for was actually credited — otherwise the colony stays at `EXECUTING` with the phase `in_progress`.
- Extended `claimsOrAggregate`'s fallback path to fold receipt-credited `dispatch.TaskClaims` into `last-build-claims.json`, not just whole-success dispatches, so persisted claims and actual task state can never silently disagree.
- Added `CreditedTaskIDs`/`UnfinishedTaskIDs` to `codexResultCollectionReport` (`result-collection.json`) so a partial build's diagnostics name both sets, separately, by task ID.

## Task Commits

1. **Task 1 + 2 RED: failing external-lane task-receipt trust boundary tests** - `1c333bac` (test)
2. **Task 1 + 2 GREEN: wire the shared task-receipt boundary into build-finalize** - `94ffb01c` (feat)

_Note: both plan tasks share the identical production wiring (the same functions in `codex_build_finalize.go`/`provenance.go`), so implementation was a single coherent change set rather than two independently separable diffs. RED was captured by reverting exactly the production hunks via a captured `git diff`/`git apply -R` round-trip (never `git stash`/`git checkout --`), running the new tests to confirm a genuine compile failure, then re-applying and confirming green — see "Deviations" for the exact RED output._

## Files Created/Modified

- `cmd/provenance.go` - `hasGenuineTaskReceiptEvidence` carve-out in `validateBuildProvenance` (SAFE-01) for a non-successful worker carrying genuine partial receipt evidence.
- `cmd/codex_build_finalize.go` - Wires `resolveCoherentJobDispatchReceipts` into `runCodexBuildFinalize` (in-repo only); adds `allSelectedBuildTasksCredited`/`buildFullBuildTaskIDSet`/`unfinishedBuildTaskIDs`/`creditedBuildTaskIDs`; gates the `StateBUILT` transition in both the in-memory projection and `commitBuildFinalizeState`; folds receipt-credited `TaskClaims` into `claimsOrAggregate`; adds `CreditedTaskIDs`/`UnfinishedTaskIDs` to `codexResultCollectionReport` and threads `selectedTaskIDs` into `buildExternalBuildResultCollectionReport`.
- `cmd/codex_build_finalize_test.go` - `TestExternalGroupedFullSuccessCreditsAll`, `TestExternalGroupedPartialPersistsExactTaskState`, `TestExternalGroupedFailureWithoutReceiptsCreditsNone`, `TestExternalPartialClaimsMatchTaskState` (in-repo + worktree-backed subtests), plus the shared `setupCoherentJobExternalFinalizeTest` fixture.
- `cmd/wrapper_bundled_completion_test.go` - `TestExternalTaskReceiptsUseSharedAdmission` (including the source-behavior assertion that `mergeExternalBuildResults` never calls `attachBuildArtifactEvidence` or assigns `CompletedTaskIDs`), `TestWrapperGroupedFailureCreditsFourOfSix`, `TestWrapperTaskReceiptRefusalsDoNotEraseValidReceipts`, plus the shared `setupCoherentJobWrapperTest` fixture.
- `cmd/merged_dispatch_task_credit_test.go` - `receiptForTask` now sets `FilesCreated`/`TestsWritten` to empty (not nil) slices so a receipt survives the completion-packet JSON-schema round-trip a full build-finalize call performs (native-lane callers were unaffected since they never round-trip a receipt through JSON).
- `cmd/no_change_vocabulary_wiring_test.go` - Updated two `buildExternalBuildResultCollectionReport` call sites for the new `selectedTaskIDs` parameter (passing `nil`, unaffected by the new diagnostics fields).

## Decisions Made

See `key-decisions` in the frontmatter. In short: no second external-only validator was needed (the 195-04 functions were already lane-neutral); the real gap was wiring plus the provenance carve-off plus BUILT-transition gating; the attempt-journal's own "built" status label was deliberately left ungated (195-07's territory); and the "vanishing root artifact between admission and finalization" acceptance criterion is satisfied by existing 195-04 coverage rather than a new, structurally-unreachable full-packet test.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Provenance's phantom-build guard rejected the four-of-six scenario outright**
- **Found during:** Task 2 (wiring the shared boundary into `runCodexBuildFinalize`)
- **Issue:** `validateBuildProvenanceForManifest` (SAFE-01) rejects a completion packet where no worker reached a successful status. A single merged dispatch covering six tasks that fails after finishing four is, by definition, a packet with zero successful workers — so the existing gate rejected the ENTIRE completion before receipt resolution ever ran, making the plan's core four-of-six deliverable unreachable.
- **Fix:** Added `hasGenuineTaskReceiptEvidence` and used it as a carve-out in `validateBuildProvenance`: a non-successful worker whose `task_receipts` contain at least one structurally credible entry (successful receipt status, non-empty summary, passing handoff with a concrete `commands_run`) is accepted as real provenance. A worker with zero receipts (or only hollow ones) still falls through to the unchanged rejection — verified against `TestValidateBuildProvenance_AllFailed` and the new `TestExternalGroupedFailureWithoutReceiptsCreditsNone`.
- **Files modified:** `cmd/provenance.go`
- **Verification:** `TestValidateBuildProvenance_AllFailed`/`_AllBlocked`/`_ZeroFilesModified`/etc. (existing, still green); `TestWrapperGroupedFailureCreditsFourOfSix`, `TestExternalGroupedPartialPersistsExactTaskState` (new, now reach dispatch processing).
- **Committed in:** `94ffb01c`

**2. [Rule 1 - Bug] BUILT-transition gate trivially passed for the ordinary "no explicit task selection" build**
- **Found during:** Task 2, first test run after wiring the gate
- **Issue:** `runCodexBuildFinalize`'s `selectedTaskIDs` is `uniqueSortedStrings(manifest.SelectedTasks)`, which is empty whenever the caller (or, in these tests, `runCodexBuildPlanOnly(root, 1, nil)`) does not pass an explicit task filter — the common case. My first `allSelectedBuildTasksCredited` implementation returned `true` immediately for an empty `selectedTaskIDs`, so the gate silently no-opped for every partial-credit test and the colony still reached `StateBUILT` with only 4 of 6 tasks done.
- **Fix:** Added `buildFullBuildTaskIDSet`: when `selectedTaskIDs` is empty, fall back to the union of every dispatch's own `dispatchCoveredTaskIDs` — the actual set of tasks this build session is responsible for, regardless of whether an explicit filter was passed.
- **Files modified:** `cmd/codex_build_finalize.go`
- **Verification:** `TestExternalGroupedPartialPersistsExactTaskState`, `TestWrapperGroupedFailureCreditsFourOfSix` (now correctly assert non-BUILT); `TestExternalGroupedFullSuccessCreditsAll` (still correctly reaches BUILT); the broader regression sweep listed below stayed green throughout.
- **Committed in:** `94ffb01c`

**3. [Rule 3 - Blocking] Task receipts failed JSON-schema round-trip validation inside a full build-finalize call**
- **Found during:** Task 2, first full-E2E test run
- **Issue:** `codex.TaskReceipt.FilesCreated`/`TestsWritten` are non-`omitempty` — the completion-packet schema requires them present as arrays, never JSON `null`. The shared `receiptForTask` test fixture (195-04) only set `FilesModified`, leaving the other two `nil`; native-lane callers never JSON-round-trip a receipt, so this was invisible until a full `runCodexBuildFinalize` call (which does, via `validateCompletionPacketSemantics` → `completionPacketAsRaw`) rejected every new test with 8+ schema violations.
- **Fix:** `receiptForTask` now sets `FilesCreated: []string{}` and `TestsWritten: []string{}` explicitly. Purely additive; verified the existing native-lane tests that use this fixture (`TestFailedGroupedDispatchCreditsExactlyReceiptedTasks`, etc.) stayed green.
- **Files modified:** `cmd/merged_dispatch_task_credit_test.go`
- **Verification:** Full regression sweep (listed below) stayed green.
- **Committed in:** `1c333bac`

---

**Total deviations:** 3 auto-fixed (2 blocking, 1 bug). No scope creep — all three were necessary for the plan's own stated acceptance criteria to be reachable at all.

## Issues Encountered

- The plan's Task 1 action text directed adding admission logic literally *inside* `mergeExternalBuildResults`, with a new signature (`root`, `phase` parameters) that would have required updating ~30 existing call sites across the test suite. Investigation showed this was unnecessary: `admitCoherentJobTaskReceipts`/`finalizeCoherentJobTaskReceiptEvidence`/`resolveCoherentJobDispatchReceipts` (195-04) already operate on the lane-neutral `codexBuildDispatch` struct and don't care which lane produced it. The actual missing piece was calling `resolveCoherentJobDispatchReceipts` from `runCodexBuildFinalize` itself — exactly mirroring how the native lane already calls it from `executeCodexBuildDispatches`. This is a smaller, less invasive change that achieves the same acceptance criteria (proven by `TestExternalTaskReceiptsUseSharedAdmission`'s direct calls to `admitCoherentJobTaskReceipts` against a post-merge external dispatch, and the source-behavior assertion that `mergeExternalBuildResults` grants no credit).
- Initial test design for `TestExternalGroupedPartialPersistsExactTaskState` attempted to prove "an admitted candidate whose root artifact vanishes before finalization is not credited" by deleting a claimed file between receipt construction and `runCodexBuildFinalize`. This cannot be reached at the full entrypoint: `validateExternalWorkerResultClaimPaths` (structural validation, runs before merge/receipt resolution on any lane) already refuses a completion packet whose own `files_modified` names a path absent from root at submission time. Simplified the test to the straightforward four-of-six scenario and confirmed the "vanished artifact" behavior is already covered, lane-neutrally, by 195-04's `TestCoherentJobReceiptFinalization` (re-verified passing, not duplicated).
- TDD RED verification required reverting exactly the two production files (`cmd/provenance.go`, `cmd/codex_build_finalize.go`) via a captured `git diff` / `git apply -R` round-trip — never `git stash`, `git checkout --`, or any other rollback command, per this execution's constraints — confirming a genuine compile failure against the new tests, then re-applying and confirming green.

## Known Stubs

None — every code path added is exercised by a real, passing test; no placeholder or unwired data path was introduced.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 195-07 (recovery/retry) can build the append-only unfinished-only retry attempt on top of `dispatch.CompletedTaskIDs`/`.TaskClaims` for the external lane exactly as it will for the native lane; it should also decide whether a partial build's attempt-journal status (currently left at `buildAttemptBuilt` regardless of `allSelectedBuildTasksCredited`) needs its own gating so a replay of an identical partial packet gets a smoother idempotent no-op instead of today's safe-but-blunt "colony state has advanced" error.
- Plan 195-08 (worktree wiring) can insert its sync step between the same two stages exactly as designed: `mergeExternalBuildResults` already threads `TaskReceipts` through unchanged for a worktree-backed completion, and `runCodexBuildFinalize`'s new gate (`effectiveParallelMode(state) != colony.ModeWorktree`) already skips calling `resolveCoherentJobDispatchReceipts` for that lane, leaving root-untouched receipts ready for 195-08's sync-then-resolve adapter.

## Self-Check: PASSED

- All 6 declared modified files confirmed present on disk with the expected changes.
- Both commit hashes (`1c333bac`, `94ffb01c`) confirmed present in `git log --oneline`.
- `go build ./...`, `go vet ./cmd/`, and `go run ./cmd/aether contract-schema --check` (no drift) all clean.
- `go test ./cmd -run TestAuditCatalogGolden` green (no CLI surface changed, no golden update needed).
- The plan's full verification command (`go test ./cmd -run 'Test(ExternalTaskReceipt|ExternalGrouped|ExternalPartial|WrapperBundled)' -count=1'`) passes.
- A targeted regression sweep across every test in `codex_build_finalize_test.go`, `build_attempt_external_test.go`, `provenance_test.go`, `no_change_vocabulary_wiring_test.go`, `no_change_results_test.go`, `wrapper_bundled_completion_test.go`, and `finalizer_completion_contract_test.go` (by name, run together) passes.

---
*Phase: 195-coherent-jobs*
*Completed: 2026-08-27*
