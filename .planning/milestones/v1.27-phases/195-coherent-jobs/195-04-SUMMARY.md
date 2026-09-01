---
phase: 195-coherent-jobs
plan: 04
subsystem: build-orchestration
tags: [go, completion-evidence, task-receipts, honest-partial-credit, tdd]

requires:
  - phase: 195-coherent-jobs
    provides: coherent job planning as the canonical grouping pass (195-01/195-03) and the additive codex.TaskReceipt wire contract (195-02)
provides:
  - Shared two-stage receipt trust boundary (admitCoherentJobTaskReceipts / finalizeCoherentJobTaskReceiptEvidence)
  - Native/internal worker transport that keeps task receipts alive across a failed or interrupted terminal result
  - Exact, evidence-backed partial task credit for a failed/blocked/timeout/interrupted grouped dispatch on the native build path
  - Backward-compatible whole-success crediting of every covered task, unchanged
affects: [195-06, 195-07, 195-08]

actuals:
  tokens: 13000
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns: [two-stage admission/finalization trust boundary, lexical-only stage-1 path normalization, root-evidence stage-2 credit]

key-files:
  created:
    - cmd/coherent_job_receipts.go
    - cmd/coherent_job_receipts_test.go
  modified:
    - cmd/codex_build.go
    - cmd/codex_build_finalize.go
    - cmd/internal_worker_adapter.go
    - cmd/build_worker_run.go
    - cmd/merged_dispatch_task_credit_test.go
    - .aether/schemas/completion-packet.schema.json

key-decisions:
  - "One shared two-stage contract owns every receipt-to-credit decision: admitCoherentJobTaskReceipts (structural, no root reads, no credit) then finalizeCoherentJobTaskReceiptEvidence (root-backed, the only source of CompletedTaskIDs)."
  - "completedBuildTaskIDs keeps its whole-success branch (completed/completed_no_change credits every covered task) byte-for-byte, and adds one new branch for every other terminal status that reads only dispatch.CompletedTaskIDs -- never touched files, never CoveredTaskIDs membership."
  - "The native/in-repo lane (executeCodexBuildDispatches) resolves both stages immediately after a worker's terminal result is known, since in-repo files already live in root. The external/wrapper lane is deferred to plan 195-06, which reuses the same two functions unchanged; mergeExternalBuildResults only threads task_receipts through untouched so that lane has something to resolve later."
  - "A receipt is bound to its own task's declared files (declaredPathsForTask) whenever the task declares any -- a receipt touching only unrelated files is refused as a requirement mismatch (D-09), never silently accepted because the files happened to be in the aggregate claim set."

patterns-established:
  - "Two-stage receipt trust boundary: stage 1 is pure/lexical (safe to run before any sync step), stage 2 is the only place completion credit or root-computed artifact hashes may originate. Plans 195-06 and 195-08 reuse both functions unchanged, inserting a sync step between them for the worktree lane."

requirements-completed: [JOBS-02]

coverage:
  - id: D1
    description: "A receipt is admitted only when it is uniquely task-scoped, covered by the dispatch, a successful status, non-empty summary, passing concrete verification, root-relative non-laundered paths that are a subset of the dispatch's own aggregate claims, and bound to the task's own declared files"
    requirement: JOBS-02
    verification:
      - kind: unit
        ref: "cmd/coherent_job_receipts_test.go#TestCoherentJobReceiptAdmission"
        status: pass
    human_judgment: false
  - id: D2
    description: "Root-evidence finalization is the only function that can populate CompletedTaskIDs, and only for candidates whose claimed files are present in the current checkout right now"
    requirement: JOBS-02
    verification:
      - kind: unit
        ref: "cmd/coherent_job_receipts_test.go#TestCoherentJobReceiptFinalization"
        status: pass
    human_judgment: false
  - id: D3
    description: "A failed/interrupted native worker's task-specific receipts survive the internal-worker-adapter transport and the durable build-attempt worker-run conversion into an external-shaped completion packet"
    requirement: JOBS-02
    verification:
      - kind: unit
        ref: "cmd/coherent_job_receipts_test.go#TestNativeWorkerTransportKeepsTaskReceiptsOnFailure"
        status: pass
    human_judgment: false
  - id: D4
    description: "Whole-success crediting of every covered task remains unchanged; a failed six-task chain with four valid receipts credits exactly those four and leaves two pending; the same shape with zero receipts credits zero, even with every file touched and every task named in the summary"
    requirement: JOBS-02
    verification:
      - kind: integration
        ref: "cmd/merged_dispatch_task_credit_test.go#TestMergedDispatchCreditsEveryCoveredTask, #TestFailedGroupedDispatchCreditsExactlyReceiptedTasks, #TestFailedGroupedDispatchWithoutReceiptsCreditsNone"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-08-27
status: complete
---

# Phase 195 Plan 04: Task-Receipt Trust Boundary and Native Partial Credit Summary

**A two-stage receipt admission/finalization boundary now lets a failed or interrupted grouped build worker be credited for exactly the tasks it proved complete, never more and never inferred from touched files or named tasks.**

## Performance

- **Duration:** 25 min
- **Completed:** 2026-08-27
- **Tasks:** 2
- **Files modified:** 8 (2 created, 6 modified)

## Accomplishments

- Added `admitCoherentJobTaskReceipts` (stage 1): structural admission that proves a receipt is uniquely task-scoped, in-scope for its dispatch, a successful status, evidenced by a passing concrete verification, path-safe and a subset of the dispatch's own aggregate claims, and bound to the task's own declared files -- all without reading root or granting credit.
- Added `finalizeCoherentJobTaskReceiptEvidence` (stage 2): the only function permitted to populate `CompletedTaskIDs`, and only after confirming a candidate's claimed files are actually present in the current root checkout, attaching root-computed artifact hashes to a new per-claim `ArtifactEvidence` field.
- Made native/internal worker transport (`internal-worker-adapter`, durable build-attempt worker runs) carry `TaskReceipts` through unchanged regardless of terminal status, so a crashed worker's partial proof is not silently dropped alongside the rest of its result.
- Extended `codexBuildDispatch` with runtime-only `TaskReceipts`, `CompletedTaskIDs`, and `TaskClaims` fields, and gave `completedBuildTaskIDs` a second branch: whole-success dispatches still credit every covered task exactly as before; every other terminal status reads only the finalizer's own `CompletedTaskIDs`.
- Wired the native/in-repo build path (`executeCodexBuildDispatches`) to resolve both stages immediately once a worker's terminal result is known, and extended the merged-dispatch credit test with the four-of-six and zero-receipt counterexamples.

## Task Commits

1. **Task 1 RED: failing task-receipt trust boundary tests** - `3d377259` (test)
2. **Task 1 GREEN: shared receipt admission/finalization boundary** - `180e4ab1` (feat)
3. **Task 2 RED: failing four-of-six partial credit regressions** - `43fc1e6f` (test)
4. **Task 2 GREEN: honest partial credit wired into the native build path** - `670a3c27` (feat)

## Files Created/Modified

- `cmd/coherent_job_receipts.go` - The shared two-stage contract: `admitCoherentJobTaskReceipts`, `finalizeCoherentJobTaskReceiptEvidence`, `resolveCoherentJobDispatchReceipts` (native caller), and named violation rules.
- `cmd/coherent_job_receipts_test.go` - Adversarial admission table (duplicate, unknown, out-of-scope, failed-status, empty-summary, invalid-handoff, unevidenced, absolute/parent-path, aggregate-claim-subset, requirement-mismatch), finalization root-evidence tests, and the native-transport failure test.
- `cmd/codex_build.go` - New dispatch/task-claim fields; `completedBuildTaskIDs`'s partial-terminal branch; `executeCodexBuildDispatches` threads receipts through and calls the native resolver.
- `cmd/codex_build_finalize.go` - `mergeExternalBuildResults` threads `TaskReceipts` onto the merged dispatch unchanged (transport only; the external lane's own crediting wiring is 195-06's job).
- `cmd/internal_worker_adapter.go` / `cmd/build_worker_run.go` - `internalWorkerResult` gains `TaskReceipts`; both `mapInternalWorkerResult` and `buildCompletionFromWorkerRuns` carry it through regardless of status.
- `cmd/merged_dispatch_task_credit_test.go` - Added `TestFailedGroupedDispatchCreditsExactlyReceiptedTasks` and `TestFailedGroupedDispatchWithoutReceiptsCreditsNone`.
- `.aether/schemas/completion-packet.schema.json` - Regenerated (`aether contract-schema --write`) for the new additive struct fields; `--check` reports no drift.

## Decisions Made

- The two-stage split is load-bearing, not stylistic: stage 1 never touches the filesystem so it is safe to run before a worktree sync step exists (195-08); stage 2 is the sole source of completion credit so no other code path can ever "accidentally" populate `CompletedTaskIDs`.
- `completedBuildTaskIDs`'s existing whole-success branch was left untouched rather than refactored to share code with the new branch -- the two crediting rules are deliberately different (assignment-based vs. evidence-based) and conflating them risked quietly weakening the whole-success guarantee.
- Requirement-mismatch binding (`declaredPathsForTask`) only fires when a task declares at least one file hint/evidence artifact; a task with no declared paths skips the check rather than refusing every receipt for that task, matching the research note that ownership must never be inferred from prose when no declared path exists.
- `mergeExternalBuildResults` (external/wrapper lane) was touched only to thread `task_receipts` through untouched -- no credit-granting change was made there, keeping 195-06's future diff clean and avoiding assumptions about how the external contract will validate a submitted packet's receipts.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Extended file scope beyond the plan's `files_modified` list to keep receipts flowing end to end**
- **Found during:** Task 2 (wiring native partial credit)
- **Issue:** The plan's frontmatter lists `cmd/codex_build.go`, `cmd/merged_dispatch_task_credit_test.go`, `cmd/coherent_job_receipts_test.go`, `cmd/coherent_job_receipts.go`, `cmd/internal_worker_adapter.go`, and `cmd/build_worker_run.go`, but `codexExternalBuildWorkerResult.TaskReceipts` (added in 195-02) was never copied onto the merged `codexBuildDispatch` inside `mergeExternalBuildResults` (`cmd/codex_build_finalize.go`) -- without that one line, no receipt could ever reach `completedBuildTaskIDs`, on either lane, through the codebase's actual, shared merge boundary.
- **Fix:** Added a single additive line (`dispatch.TaskReceipts = append([]codex.TaskReceipt{}, result.TaskReceipts...)`) immediately after the existing `dispatch.Outputs` assignment. No other behavior in that function changed.
- **Files modified:** `cmd/codex_build_finalize.go`
- **Verification:** `TestMergedDispatchCreditsEveryCoveredTask` (existing whole-success path) and the two new failed-dispatch tests all pass through this exact function; the drift check (`aether contract-schema --check`) reports clean.
- **Committed in:** `180e4ab1` (Task 1 GREEN commit, since this is transport/data-fidelity, not credit-granting logic)

---

**Total deviations:** 1 auto-fixed (1 blocking wiring gap)
**Impact on plan:** Necessary for the plan's own acceptance criteria to be reachable at all; no scope creep beyond one additive line in a shared merge boundary both lanes already depend on.

## Issues Encountered

- TDD RED verification for both tasks required temporarily reverting the relevant implementation hunks via a captured `patch -R`/`patch` round-trip (never `git stash`, `git checkout --`, or any other rollback command, per this execution's constraints) to confirm a genuine compile/test failure before restoring and committing GREEN. Both tasks showed real RED: Task 1's removal produced an `undefined: resolveCoherentJobDispatchReceipts` compile error, and Task 2's removal produced a concrete task-status assertion failure in `TestFailedGroupedDispatchCreditsExactlyReceiptedTasks`.
- `runCodexBuildWithOptions`'s existing all-or-nothing rollback (`validateRuntimeBuildDispatchResults` returns an error on any non-successful dispatch, and the caller rolls back colony state entirely on that error) means a real production build whose sole worker fails today still rolls back before `reconcileCompletedBuildTasks` ever runs, even though `resolveCoherentJobDispatchReceipts` now correctly computes partial credit onto the dispatch. Relaxing that all-or-nothing behavior is explicitly out of scope for this plan (the plan's own action text: "Do not implement retry or alter the append-only attempt yet; plan 195-07 owns recovery persistence") and is recorded here so it is not mistaken for a completed capability.

## Known Stubs

None — every code path added is exercised by a real, passing test; no placeholder or unwired data path was introduced.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 195-06 (external lane) can call `admitCoherentJobTaskReceipts`/`finalizeCoherentJobTaskReceiptEvidence` unchanged once it decides how a wrapper-submitted completion packet reaches `resolveCoherentJobDispatchReceipts` (or an equivalent call) for the finalize entry points in `cmd/codex_build_finalize.go`.
- Plan 195-07 (recovery/retry) can build the append-only unfinished-only retry attempt on top of `dispatch.CompletedTaskIDs`/`.TaskClaims`, and separately needs to decide how a real production failure reaches partial-credit persistence given the existing all-or-nothing rollback noted above.
- Plan 195-08 (worktree wiring) can insert its sync step between the same two stages exactly as designed: call `admitCoherentJobTaskReceipts` for candidate `SyncPaths`, sync only those paths to root, then call `finalizeCoherentJobTaskReceiptEvidence`.

## Self-Check: PASSED

- All declared new/modified files exist on disk and match this summary.
- All four RED/GREEN task commits are present in git history (`3d377259`, `180e4ab1`, `43fc1e6f`, `670a3c27`).
- `go build ./...`, `go vet ./...`, and `go build ./cmd/aether` are clean.
- `go run ./cmd/aether contract-schema --check` reports no drift.
- The plan's full verification command (`go test ./cmd -run 'Test(CoherentJobReceipt(Admission|Finalization)|NativeWorkerTransport|MergedDispatchCredits|FailedGroupedDispatch)' -count=1'`) passes, along with a broader regression sweep across coalescing, CalVault, build-finalize, wrapper-bundled-completion, build-attempt, worktree, and grouped-job tests.

## Self-Check: PASSED (verified)

- All 9 declared files confirmed present on disk via direct filesystem check.
- All 4 commit hashes confirmed present in `git log --oneline --all`.

---
*Phase: 195-coherent-jobs*
*Completed: 2026-08-27*
