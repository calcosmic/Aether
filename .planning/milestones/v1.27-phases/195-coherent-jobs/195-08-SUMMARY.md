---
phase: 195-coherent-jobs
plan: 08
subsystem: build-orchestration
tags: [go, worktree, coherent-jobs, task-receipts, partial-credit, tdd]

requires:
  - phase: 195-coherent-jobs
    provides: coherent job planning ahead of worktree ownership (195-03), the shared two-stage receipt trust boundary (195-04), the external-lane receipt wiring (195-06), and the append-only unfinished-only retry (195-07)
provides:
  - One coherent job takes exactly one worktree, branch, worker session and merge-back, proven by the literal six-batch CalVault fixture
  - codex.WorkerDispatch carries ordered CoveredTaskIDs plus grouped job name/reason into the execution layer
  - Declared worktree ownership is the unique sorted union per execution owner, with refusals that name the job and every task it covers
  - resolveCoherentJobWorktreeReceipts - the shared admission -> receipt-scoped sync -> root finalization sequence for work that starts life inside a worktree
  - Uncredited worktree edits are neither synced nor destroyed - they stay in a preserved, tracked checkout with a plain-English diagnostic
  - External/wrapper worktree completions route through the same two stages instead of being skipped
affects: [195-09, 195-10]

actuals:
  tokens: 15500
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Sync-between-stages: stage 1 admission is lexical so it is safe to run while files exist only in a worker checkout; the sync step copies only what admission admitted; stage 2 is the sole source of completion credit"
    - "Per-build ledger (not package state) to carry a worktree-resolved verdict back to the layer that owns the dispatch list"

key-files:
  created: []
  modified:
    - pkg/codex/dispatch.go
    - cmd/codex_build.go
    - cmd/codex_build_worktree.go
    - cmd/codex_build_worktree_test.go
    - cmd/coherent_job_receipts.go
    - cmd/codex_build_finalize.go

key-decisions:
  - "The worktree lane inserts its sync step between the SAME two functions 195-04 built (admitCoherentJobTaskReceipts then finalizeCoherentJobTaskReceiptEvidence). No second validator was written, and finalization stayed the only producer of CompletedTaskIDs, so a file that exists solely inside a worktree can never become task credit."
  - "The unit of worktree ownership is the DISPATCH, not the task. Grouping already ran before this guard (195-03), so a job owns the unique sorted union of its tasks' declared paths and cannot conflict with itself, while two genuinely distinct jobs sharing one path in one wave are still refused atomically before any worker starts."
  - "A worktree-resolved verdict is authoritative and is never recomputed from root afterwards (codexBuildDispatch.ReceiptsResolved, in-process only, json:\"-\"). Re-running admission after the sync would re-derive credit from files this same build just copied in - exactly the circular reasoning the two-stage split exists to prevent."
  - "Sync happens one candidate path at a time. A single unwritable destination excludes exactly its own task, by name, with the concrete path and error, rather than poisoning the whole job or crediting the rest on a guess."
  - "Touched paths are now collected for every terminal result that produced one, not only a clean success - a worker that crashed after four of six steps still changed real files, and the proven/unproven partition cannot exist without knowing what it touched. applyObservedClaims stayed gated on a clean success so a failed worker's own claim fields are never re-derived."
  - "The completion-packet path-laundering guard was NOT widened to admit worktree-only claims. Weakening a security check to make a parity test pass was refused; the external-lane parity test drives the real crediting chain instead, and the boundary is recorded in deferred-items.md."

patterns-established:
  - "Admission -> receipt-scoped sync -> root finalization is now the single named sequence for any lane whose evidence starts outside root. Both worktree lanes call resolveCoherentJobWorktreeReceipts; neither owns a private validator."
  - "Preserve-don't-destroy for unproven work: anything a worker touched that no admitted receipt claimed is left in its checkout, the checkout is marked Orphaned rather than removed, and the user is told which files and which branch in plain English."

requirements-completed: [JOBS-02, JOBS-03, JOBS-04]

coverage:
  - id: D1
    description: "The literal six-batch CalVault fixture runs as one grouped job in worktree mode: one worker session, one worktree, one branch, one merge-back, six covered task IDs in order, and all six tasks credited"
    requirement: JOBS-04
    verification:
      - kind: integration
        ref: "cmd/codex_build_worktree_test.go#TestCalVaultSixBatchesBecomeOneWorktreeJob"
        status: pass
    human_judgment: false
  - id: D2
    description: "One grouped job reaches codex.WorkerDispatch carrying every covered task ID in order and the unique sorted union of its tasks' declared paths, so its intentional internal overlap has exactly one owner"
    requirement: JOBS-04
    verification:
      - kind: unit
        ref: "cmd/codex_build_worktree_test.go#TestGroupedWorktreeOwnsUnionedPaths"
        status: pass
    human_judgment: false
  - id: D3
    description: "Two genuinely distinct jobs declaring the same path in one wave are still refused atomically before any worker starts, and the refusal names the contested path and both jobs"
    requirement: JOBS-04
    verification:
      - kind: integration
        ref: "cmd/codex_build_worktree_test.go#TestDistinctWorktreeJobsStillRejectOverlap, #TestGroupedWorktreeOwnsUnionedPaths"
        status: pass
    human_judgment: false
  - id: D4
    description: "Structural admission yields four candidates and four sync paths with zero CompletedTaskIDs while the files exist only in the worktree; after the receipt-scoped sync, exactly four tasks are credited and their files are present in root, with the other two left pending"
    requirement: JOBS-02
    verification:
      - kind: integration
        ref: "cmd/codex_build_worktree_test.go#TestGroupedWorktreePartialReceiptsSyncBeforeCredit"
        status: pass
    human_judgment: false
  - id: D5
    description: "Touched paths outside the admission are neither synced into root nor destroyed - they remain inside a preserved, colony-tracked orphaned worktree"
    requirement: JOBS-03
    verification:
      - kind: integration
        ref: "cmd/codex_build_worktree_test.go#TestGroupedWorktreeUncreditedEditsRemainOrphaned"
        status: pass
    human_judgment: false
  - id: D6
    description: "When every candidate path fails to reach root, nothing is credited, no task advances, and the build takes the ordinary failure path"
    requirement: JOBS-02
    verification:
      - kind: integration
        ref: "cmd/codex_build_worktree_test.go#TestGroupedWorktreeSyncFailureCreditsNone"
        status: pass
    human_judgment: false
  - id: D7
    description: "The D-10 recovery job created after worktree partial credit contains only the two unproven tasks, links to its parent, and never rewrites the first worker's attempt"
    requirement: JOBS-02
    verification:
      - kind: integration
        ref: "cmd/codex_build_worktree_test.go#TestGroupedWorktreeRetryContainsOnlyUnfinishedTasks"
        status: pass
    human_judgment: false
  - id: D8
    description: "The external/wrapper worktree lane routes through the same admission/sync/finalization functions and produces the same four-task credit set as the native lane, syncing no unproven edit"
    requirement: JOBS-04
    verification:
      - kind: integration
        ref: "cmd/codex_build_worktree_test.go#TestGroupedWorktreePartialReceiptsExternalLaneMatchesNative"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-08-27
status: complete
---

# Phase 195 Plan 08: Coherent Jobs and Honest Partial Credit in Worktree Mode Summary

**One grouped job now takes exactly one isolated worktree, and a worker that finishes only part of that job is credited strictly for the tasks whose proof was copied back into the project — everything it touched but never proved stays recoverable on its own branch instead of becoming false completion.**

## Performance

- **Duration:** ~55 min
- **Completed:** 2026-08-27
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Reproduced the literal six-batch CalVault field fixture in worktree mode and locked it: one worker session, one worktree, one branch, one merge-back, six covered task IDs in order. Phase 184/H5 deliberately excluded worktree mode from coalescing, so the exact failure this project measured — six fresh agents re-reading the same source list — had only ever been fixed for in-repo users.
- Extended `codex.WorkerDispatch` additively with `CoveredTaskIDs`, `JobName` and `JobReason`, and extracted the previously inline conversion into `buildCodexWorkerDispatches` — the single place a planned coherent job becomes an execution owner, and therefore the correct thing for worktree ownership to read.
- Made declared worktree ownership the unique sorted union per execution owner, so a grouped job's intentional internal overlap has one owner, while two genuinely distinct jobs sharing a path in one wave are still refused before any worker starts — now with a refusal that names the job and every task it covers.
- Added `resolveCoherentJobWorktreeReceipts`: the shared admission → receipt-scoped sync → root-evidence finalization sequence, reusing 195-04's exact two functions with the sync step inserted between them. No second validator exists, and `finalizeCoherentJobTaskReceiptEvidence` remains the only producer of `CompletedTaskIDs`.
- Wired that sequence into the native worktree lane (`reconcileWorktreeWave`) via a per-build ledger, and into the external/wrapper lane (`resolveWorktreeExternalDispatchReceipts`), replacing 195-06's explicit "skipped, left for 195-08" gap.
- Made unproven work recoverable rather than lost or credited: touched paths that no admitted receipt claimed are never synced, the checkout is preserved as Orphaned, and the user is told which files and which branch in plain English.

## Task Commits

1. **Task 1 RED: failing CalVault worktree grouping tests** — `8bf35a9e` (test)
2. **Task 1 GREEN: one coherent job, one worktree owner** — `f093de72` (feat)
3. **Task 2 RED: failing worktree partial-receipt tests** — `9c8095a4` (test)
4. **Task 2 GREEN: sync accepted worktree receipts before crediting them** — `aae13d62` (feat)

## Files Created/Modified

- `pkg/codex/dispatch.go` — `WorkerDispatch.CoveredTaskIDs` / `.JobName` / `.JobReason` (additive).
- `cmd/codex_build.go` — extracted `buildCodexWorkerDispatches`; populates covered IDs, job identity and the unique sorted declared-path union; per-build `worktreeReceiptLedger`; applies the worktree verdict onto dispatches; new in-process `codexBuildDispatch.ReceiptsResolved` (`json:"-"`, no contract change).
- `cmd/codex_build_worktree.go` — `worktreeReceiptLedger`; `worktreeOwnershipIdentity` and the job-naming ownership refusal; touched-path collection for every terminal result; `resolveWorktreePartialReceipts` in the wave reconciliation.
- `cmd/coherent_job_receipts.go` — `resolveCoherentJobWorktreeReceipts` (admission → per-path sync → finalization), `violationRuleTaskReceiptSyncFailed`, `resolveWorktreeExternalDispatchReceipts`, `worktreeTouchedPathsForExternalResult`; `resolveCoherentJobDispatchReceipts` now skips an already-resolved dispatch.
- `cmd/codex_build_finalize.go` — the external lane's worktree branch now resolves through the shared boundary instead of being skipped.
- `cmd/codex_build_worktree_test.go` — the CalVault worktree fixture, the ownership/union test, the distinct-job overlap guard, the four partial-receipt regressions, and the external-lane parity test.

## Verification Commands and Results

Every command below was run and its real output is reported.

- Task 1: `go test ./cmd -run 'Test(CalVaultSixBatchesBecomeOneWorktreeJob|GroupedWorktreeOwnsUnionedPaths|DistinctWorktreeJobsStillRejectOverlap)' -count=1` → `ok github.com/calcosmic/Aether/cmd 1.717s`
- Task 2: `go test ./cmd -run 'TestGroupedWorktree(PartialReceipts|UncreditedEdits|SyncFailure|Retry)' -count=1` → `ok github.com/calcosmic/Aether/cmd 4.283s`
- Plan verification: `go test ./cmd -run 'Test(CalVaultSixBatchesBecomeOneWorktreeJob|GroupedWorktree|DistinctWorktreeJobs)' -count=1` → `ok github.com/calcosmic/Aether/cmd 8.126s`
- In-repo vs worktree CalVault parity: `go test ./cmd -run 'TestCalVaultSixBatchesBecomeOne(InRepo|Worktree)Job' -count=1 -v` → `--- PASS: TestCalVaultSixBatchesBecomeOneWorktreeJob (1.09s)` and `--- PASS: TestCalVaultSixBatchesBecomeOneInRepoJob (0.00s)`. Both assert one job with the same six covered IDs in order and one execution owner.
- Targeted regression sweep: `go test ./cmd -run 'Test(CoherentJob|MergedDispatch|FailedGroupedDispatch|GroupedJob|BuildWorktreeMode|ExternalGrouped|BuildAttempt|LegacyBuildAttempt|BuildDoesNotAdvance|AuditCatalogGolden|Coalesce)' -count=1` → `ok github.com/calcosmic/Aether/cmd 7.837s`
- `go test ./pkg/codex/ -count=1` → `ok github.com/calcosmic/Aether/pkg/codex 26.874s`
- `go build ./...` → clean. `go vet ./cmd/ ./pkg/codex/` → clean.
- `go run ./cmd/aether contract-schema --check` → `{"ok":true,"result":{"drift":false,...}}` (no completion-packet contract field changed; `ReceiptsResolved` is `json:"-"`).
- `go test ./cmd -run TestAuditCatalogGolden` → passes; no CLI flag or subcommand changed, so no golden update was needed.

## TDD Evidence: Genuine RED

### Task 1 — compile RED, then a real assertion RED

The RED commit (`8bf35a9e`) failed to compile, because `WorkerDispatch.CoveredTaskIDs` and `buildCodexWorkerDispatches` are both new:

```
cmd/codex_build_worktree_test.go:1214:27: undefined: buildCodexWorkerDispatches
FAIL	github.com/calcosmic/Aether/cmd [build failed]
```

Per this execution's RED-quality rule, a compile error alone proves little, so after the pure extraction refactor (new symbol present, fields still unpopulated) the test produced a genuine assertion failure:

```
--- FAIL: TestGroupedWorktreeOwnsUnionedPaths (0.12s)
    codex_build_worktree_test.go:1224: WorkerDispatch.CoveredTaskIDs = [], want the six batches in order [3.1 3.2 3.3 3.4 3.5 3.6]
```

**Honest finding: the two other Task 1 tests passed on first run.** `TestCalVaultSixBatchesBecomeOneWorktreeJob` and `TestDistinctWorktreeJobsStillRejectOverlap` were green against pre-existing code, because plan 195-03 had already moved grouping ahead of `validateDeclaredWorktreeOwnership`. They are therefore new executable regression guards for behaviour that already held, not proof of new behaviour. Both were verified load-bearing by mutation:

- Reverting the grouping loop to pre-195-03 one-worker-per-task behaviour:
  ```
  --- FAIL: TestCalVaultSixBatchesBecomeOneWorktreeJob (1.45s)
      CalVault worktree build returned error: build dispatch did not complete cleanly: Mason-6=failed
      (worktree wave reconciliation conflict: templates/batch-1.md produced by multiple workers
      (Mason-6, Weld-95, Hammer-85, Brick-75, Bolt-65, Forge-55) but declared by task 3.1, ...)
  ```
  This is exactly the "multiple worktrees / conflict" failure the plan predicted.
- Removing the `validateDeclaredWorktreeOwnership` call:
  ```
  --- FAIL: TestDistinctWorktreeJobsStillRejectOverlap (0.83s)
      two unrelated jobs declaring one path were accepted; want an atomic pre-dispatch refusal
  ```

Both mutations were applied from a captured file copy and reverted (never `git stash`, `git checkout --`, or any other rollback command, per this execution's constraints); `grep -c MUTATION` confirmed 0 afterwards and the tests were re-run green.

### Task 2 — real assertion RED, no compile error

The Task 2 RED commit (`9c8095a4`) references only pre-existing symbols, so it compiled and failed on real assertions:

```
--- FAIL: TestGroupedWorktreePartialReceiptsSyncBeforeCredit (0.50s)
    worktree partial credit should be accepted, got error: build dispatch did not complete cleanly: Anvil-40=failed (crashed after finishing four of six steps)
--- FAIL: TestGroupedWorktreeUncreditedEditsRemainOrphaned (0.49s)
    worktree partial credit should be accepted, got error: build dispatch did not complete cleanly: Anvil-40=failed (crashed after finishing four of six steps)
--- FAIL: TestGroupedWorktreeRetryContainsOnlyUnfinishedTasks (0.55s)
    worktree partial credit should be accepted, got error: build dispatch did not complete cleanly: Anvil-40=failed (crashed after finishing four of six steps)
--- FAIL: TestGroupedWorktreePartialReceiptsExternalLaneMatchesNative (0.37s)
    external worktree finalize should accept validated partial credit, got error: colony state changed after build attempt ... was prepared
```

`TestGroupedWorktreeSyncFailureCreditsNone` was green at RED (it asserts the failure path, which was the pre-existing behaviour) and is a guard for the new code, so it was verified load-bearing by mutation:

- Crediting every admitted candidate without root evidence:
  ```
  --- FAIL: TestGroupedWorktreeSyncFailureCreditsNone (0.78s)
      a build whose every receipt failed to sync was accepted; want the ordinary failure path
  ```
- Syncing everything the worker touched rather than only the admitted candidate paths:
  ```
  --- FAIL: TestGroupedWorktreeUncreditedEditsRemainOrphaned (1.22s)
      unproven edit scratch/unproven-notes.md was synced into root anyway: err=<nil>
  ```

Both mutations were reverted from a captured copy and re-verified green.

## Decisions Made

See `key-decisions` in the frontmatter. The load-bearing one: the sync step goes *between* the two existing stages and nowhere else, so structural admission never reads root and root-backed finalization is still the only thing that can hand out task credit. A worktree-only file therefore cannot be credited, by construction rather than by convention.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Extended file scope to `cmd/codex_build.go` and `cmd/codex_build_finalize.go`**
- **Found during:** Tasks 1 and 2
- **Issue:** The plan's `files_modified` lists five files, but the conversion from planned dispatch to executable `codex.WorkerDispatch` lives in `cmd/codex_build.go`, and the external lane's worktree skip that this plan exists to close lives in `cmd/codex_build_finalize.go`. Without touching both, the plan's own acceptance criteria (`WorkerDispatch carries all six ordered IDs`; `Native and external worktree fixtures call the same admission/finalization functions`) are unreachable.
- **Fix:** In `codex_build.go`, extracted the inline conversion into `buildCodexWorkerDispatches`, populated the new fields, added the per-build receipt ledger and the in-process `ReceiptsResolved` marker. In `codex_build_finalize.go`, replaced the eleven-line "skipped, left for 195-08" branch with a call to `resolveWorktreeExternalDispatchReceipts`. Both changes are additive; no existing credit rule was altered.
- **Files modified:** `cmd/codex_build.go`, `cmd/codex_build_finalize.go`
- **Verification:** Full targeted regression sweep above, plus `contract-schema --check` clean.
- **Committed in:** `f093de72` and `aae13d62`

### Scoped decision (documented, not a Rule 1-4 fix)

**The completion-packet path-laundering guard was deliberately not widened.** The external-lane worktree parity test initially failed inside `validateCompletionPacketSemantics` → `validateAndNormalizeClaimPathToRoot` with `claim_path.escapes_root`: that validator refuses any file claim that does not resolve inside the repository root, which a worktree-only claim by definition does not. Making it accept such a claim is a trust-boundary change, not a wiring change, and this plan declined to weaken a security check to make a test pass. In practice no shipped flow reaches the refusal — only the native dispatch path allocates worktrees, so a wrapper-submitted completion never has worktree-only claims today. The parity test therefore drives the external lane's real crediting chain (`mergeExternalBuildResults` → `resolveWorktreeExternalDispatchReceipts` → `reconcileCompletedBuildTasks`) rather than the packet validator, and the boundary plus its follow-up decision are recorded in `deferred-items.md`.

---

**Total deviations:** 1 auto-fixed (1 blocking file-scope extension). 1 documented scoping decision (security validator left intact).
**Impact on plan:** No correctness impact. Every `<acceptance_criteria>` and the plan-level `<verification>` command passed, except that the external-lane parity is asserted one layer below the packet validator, which is stated above rather than papered over.

## Issues Encountered

- The external-lane fixture initially failed with `colony state changed after build attempt ... was prepared`, because allocating a real worktree mutates `COLONY_STATE.json` after the plan-only build has already recorded its state digest. Resolved by asserting on the external lane's crediting chain directly rather than replaying a stale completion packet through `runCodexBuildFinalize`.
- `gofmt -l cmd/ pkg/` reports `cmd/criterion_owner_confirmation.go` as unformatted. `git diff HEAD` on that file is empty, so the drift predates this plan; logged to `deferred-items.md` rather than fixed inside a receipt-boundary commit (scope boundary).

## Known Stubs

None — every code path added is exercised by a real, passing test, and each of the two tests that were already green was independently proven load-bearing by mutation.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- JOBS-04 now holds for both full and partial grouped jobs in worktree mode: grouping precedes ownership and allocation, and root-backed evidence always precedes task credit.
- Plans 195-09/195-10 can rely on `dispatch.CompletedTaskIDs` being lane-neutral: native in-repo, native worktree, external in-repo and external worktree all populate it through the same two functions.
- One open decision is recorded for a future plan (not a gap in this one): whether `validateAndNormalizeClaimPathToRoot` should accept a path that resolves inside a colony-tracked worktree under `.aether/worktrees/`, should a wrapper lane ever gain worktree allocation.

## Self-Check: PASSED

- All 6 declared modified files confirmed present with the expected changes (`git diff --stat HEAD~3 HEAD`: 962 insertions, 60 deletions across `cmd/codex_build.go`, `cmd/codex_build_finalize.go`, `cmd/codex_build_worktree.go`, `cmd/codex_build_worktree_test.go`, `cmd/coherent_job_receipts.go`, `pkg/codex/dispatch.go`).
- All 4 task commits confirmed in `git log --oneline`: `8bf35a9e`, `f093de72`, `9c8095a4`, `aae13d62`.
- `go build ./...`, `go vet ./cmd/ ./pkg/codex/` clean; `contract-schema --check` reports no drift; `TestAuditCatalogGolden` passes with no golden update.
- Every `<acceptance_criteria>` command and the plan-level `<verification>` command were run, with the real output quoted above.

---
*Phase: 195-coherent-jobs*
*Completed: 2026-08-27*
