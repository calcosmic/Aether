---
phase: 195-coherent-jobs
plan: 07
subsystem: build-orchestration
tags: [go, coherent-jobs, build-attempt, append-only, retry, tdd]

requires:
  - phase: 195-coherent-jobs
    provides: the shared two-stage receipt trust boundary and native/external partial credit (195-04, 195-06)
provides:
  - "planCoherentJobRetry: a pure, dependency-safe unfinished-only job planner reusing planCoherentJobs for ordering/waves"
  - "buildAttemptRecord.ParentAttemptID/ParentJobName plus attachBuildAttemptParentLink/beginChildBuildAttempt: append-only child-attempt provenance"
  - "reconcilePartialBuildRetry/findExistingBuildAttemptRetry: the shared, idempotent per-dispatch orchestration both build lanes call"
  - "Direct/native lane (codex_build.go): partial credit bypasses wholesale rollback, commits task credit via commitPartialBuildCredit, and creates a recovery attempt"
  - "External/wrapper lane (codex_build_finalize.go): the attempt-journal transition is now gated on buildFullyCredited (new buildAttemptPartial status, never a misleading built), and creates the same recovery attempt"
affects: [195-08, 195-09, 195-10]

actuals:
  tokens: 13200
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns: [pure unfinished-only job planner reusing the canonical grouping pass, append-only parent-linked attempt (begin+attach), idempotent retry-attempt creation keyed by ParentAttemptID]

key-files:
  created:
    - cmd/coherent_job_retry.go
    - cmd/coherent_job_retry_test.go
  modified:
    - cmd/build_attempt.go
    - cmd/build_attempt_test.go
    - cmd/build_attempt_external_test.go
    - cmd/codex_build.go
    - cmd/codex_build_finalize.go

key-decisions:
  - "planCoherentJobRetry routes a retry job through the exact same proposal-shaped validation every other coherent job goes through (planCoherentJobs with a single synthetic queen-shaped proposal over only the unfinished tasks) rather than a second hand-rolled graph walk. Credited tasks are simply excluded from the seed set, so their own already-satisfied dependencies never need special-casing -- coherentJobGraphPreflight (colony.DetectCycles over the whole real phase) already names a genuine cycle or missing dependency by ID."
  - "ParentAttemptID/ParentJobName follow the existing checkFixAttemptRecord precedent exactly: a narrow attachBuildAttemptParentLink setter (touches nothing else on the record) applied to a brand-new attempt beginBuildAttempt already created, never a write against the parent's own file. TestBuildAttemptChildLinksParentWithoutMutation proves the parent file is byte-for-byte unchanged."
  - "A new buildAttemptPartial status (distinct from both built and failed) marks an attempt that credited SOME but not all of its covered tasks. built keeps meaning 'every covered task proven done' (buildAttemptCompletionSealed is unchanged); failed keeps meaning 'nothing credited'; partial is neither, and is what the external lane now records instead of the pre-existing unconditional buildAttemptBuilt transition (flagged as a gap in 195-06's own SUMMARY)."
  - "Retry recovery only ever follows ACCEPTED credit, never precedes it (195-CONTEXT.md D-08/D-09/D-10 read together): reconcilePartialBuildRetry skips any dispatch with zero credited tasks entirely, leaving the existing whole-rollback (direct lane) / phantom-build rejection (external lane, SAFE-01) untouched for a total failure with no receipts. TestBuildDoesNotAdvanceWhenWorkersFailOrTimeout and TestExternalGroupedFailureWithoutReceiptsCreditsNone both still pass unchanged."
  - "Idempotency is scoped to reconcilePartialBuildRetry itself (findExistingBuildAttemptRetry scans the phase's attempt journal for a record already linked to the parent), proven directly against that function twice in a row. Making a REPEATED IDENTICAL completion packet reach that idempotent path a second time through the full runCodexBuildFinalize entrypoint is explicitly deferred -- see Deviations."
  - "The retry attempt's own Dispatches field is a lightweight 'planned' record for audit/journal purposes, not a live worker dispatch: since credited tasks are already marked colony.TaskCompleted by commitPartialBuildCredit/commitBuildFinalizeState, an ordinary `aether build <phase> --force` naturally replans and dispatches only the remaining pending tasks. The attempt journal's job is proof-of-linkage and an exact recovery command, not re-deriving live dispatch machinery."

patterns-established:
  - "Unfinished-only retry planning: filter a dispatch's own covered task IDs down to the uncredited subset, seed the canonical planner with ONLY those tasks plus a single all-inclusive proposal, and let the existing dependency/cycle machinery do the rest -- no parallel graph implementation."
  - "Begin-then-attach for append-only provenance: create the new record through the unmodified shared constructor, then attach narrow, single-purpose provenance fields in a second call that touches nothing else -- the same shape checkFixAttemptRecord and outOfBandVerificationRecord already established."

requirements-completed: [JOBS-02]

coverage:
  - id: D1
    description: "A parent six-task job with four credited task IDs yields exactly one child job containing only the remaining two, in dependency-safe order; a credited-task dependency is treated as already satisfied while an unfinished-to-unfinished dependency remains ordered; a genuine cycle or missing dependency in the unfinished subgraph is a named hard error that creates no job."
    requirement: JOBS-02
    verification:
      - kind: unit
        ref: "cmd/coherent_job_retry_test.go#TestCoherentJobRetryContainsOnlyUnfinishedTasks, #TestCoherentJobRetryTreatsCreditedDependenciesAsSatisfied, #TestCoherentJobRetryFullSuccessReturnsNoJob, #TestCoherentJobRetryNamesRealCycle, #TestCoherentJobRetryNamesMissingDependency, #TestCoherentJobRetryRequiresCoveredTasks"
        status: pass
    human_judgment: false
  - id: D2
    description: "A new attempt links to its parent (ParentAttemptID/ParentJobName) through the existing SaveJSON/latest-pointer path, and the parent's own attempt file is byte-for-byte unchanged before and after child creation; legacy attempt JSON without these fields still decodes cleanly."
    requirement: JOBS-02
    verification:
      - kind: unit
        ref: "cmd/coherent_job_retry_test.go#TestBuildAttemptChildLinksParentWithoutMutation, #TestLegacyBuildAttemptReadsWithoutParentFields"
        status: pass
    human_judgment: false
  - id: D3
    description: "A real four-of-six failed grouped dispatch on the direct/native build lane, run through the actual runCodexBuild entrypoint with a scripted invoker returning genuine root-backed receipts, bypasses wholesale rollback: exactly four tasks are credited, the colony never reports BUILT, the parent attempt stays failed and unmutated, and a new append-only child attempt linked to the parent is created and becomes the phase's latest attempt."
    requirement: JOBS-02
    verification:
      - kind: integration
        ref: "cmd/build_attempt_test.go#TestGroupedJobPartialRetryIsAppendOnlyDirect"
        status: pass
    human_judgment: false
  - id: D4
    description: "The same four-of-six shape on the external/wrapper build-finalize lane, run through the real aether build-finalize entrypoint, credits exactly four tasks, seals the parent attempt as the new partial status (never built), and creates the identical shared retry attempt; calling the shared reconciler a second time for the same parent returns the SAME child rather than creating a duplicate."
    requirement: JOBS-02
    verification:
      - kind: integration
        ref: "cmd/build_attempt_external_test.go#TestGroupedJobPartialRetryIsAppendOnlyExternal, #TestGroupedJobPartialRetryIsIdempotent"
        status: pass
    human_judgment: false
  - id: D5
    description: "A retry attempt's own dispatch never lists any credited task ID among its covered tasks -- a fresh worker built from the retry attempt cannot be asked to redo proven work."
    requirement: JOBS-02
    verification:
      - kind: unit
        ref: "cmd/build_attempt_test.go#TestGroupedJobRetryNeverReassignsCreditedTasks"
        status: pass
    human_judgment: false

duration: 70min
completed: 2026-08-27
status: complete
---

# Phase 195 Plan 07: Append-Only Recovery for Partial Grouped Jobs Summary

**A pure `planCoherentJobRetry` planner and a shared `reconcilePartialBuildRetry` orchestrator now turn honest partial credit into an honest, append-only recovery job on both build lanes, without ever mutating the first worker's attempt or redoing proven work.**

## Performance

- **Duration:** ~70 min
- **Completed:** 2026-08-27
- **Tasks:** 2
- **Files modified:** 7 (2 created, 5 modified)

## Accomplishments

- Added `planCoherentJobRetry`, a pure function that derives a new dependency-safe coherent job containing only a parent dispatch's uncredited tasks, by seeding the existing canonical `planCoherentJobs` with a single all-inclusive proposal over the unfinished subset -- reusing the same cycle/order validation every other job goes through rather than a second hand-rolled graph.
- Added `ParentAttemptID`/`ParentJobName` (optional, `omitempty`) to `buildAttemptRecord`, plus `attachBuildAttemptParentLink` (a narrow setter mirroring `attachCheckFixAttempt`'s discipline) and `beginChildBuildAttempt` (begin via the unmodified `beginBuildAttempt`, then attach parent provenance) -- proven to leave the parent's own attempt file byte-for-byte unchanged.
- Added the shared, idempotent `reconcilePartialBuildRetry`/`findExistingBuildAttemptRetry`: for every dispatch that covered more tasks than it credited, plan a retry job and create (or, on a repeat call, find) exactly one append-only child attempt linked to the parent.
- Wired the direct/native build lane (`cmd/codex_build.go`): a dispatch error now first checks for validated partial proof before the existing wholesale `rollbackCodexBuildFailure`; genuine partial credit commits task state via the new `commitPartialBuildCredit` (same concurrency guard as the existing EXECUTING→BUILT commit, but never advances to BUILT) and returns the recovery info instead of a bare error.
- Wired the external/wrapper build-finalize lane (`cmd/codex_build_finalize.go`): the attempt-journal transition after `commitBuildFinalizeState` is now gated on `buildFullyCredited` -- a whole success still seals `built` exactly as before, a partial one is recorded as the new `buildAttemptPartial` status, and the same shared reconciler runs immediately after.

## Task Commits

1. **Task 1 RED: failing tests for coherent job retry and parent-linked attempts** - `2e5a2fd9` (test)
2. **Task 1 GREEN: unfinished-only job planner and parent-linked attempt provenance** - `ae52189b` (feat)
3. **Task 2 RED: failing tests for append-only partial-retry wiring** - `667bbfc5` (test)
4. **Task 2 GREEN: wire append-only partial-retry recovery into both build lanes** - `6d7e6d6b` (feat)

**Plan metadata:** (this commit)

_Note: RED for both tasks was necessarily a compile error against pre-existing code the first time (every referenced symbol -- `planCoherentJobRetry`, `beginChildBuildAttempt`, `ParentAttemptID`, `reconcilePartialBuildRetry`, `commitPartialBuildCredit`, `buildAttemptPartial` usage in the lanes -- is brand new). Per this repo's RED-quality guidance, genuine RED was additionally verified by mutation: the exact production hunks for each task were reverted via a captured `git diff` / `git apply -R` round-trip (never `git stash`/`git checkout --`), the named tests were re-run and shown to fail for the right reason (a real compile error for Task 1; real assertion failures -- "did not complete cleanly" / "did not report a D-10 recovery job" -- for Task 2, since Task 2's tests exercise pre-existing, already-compiling lane code paths), then the hunks were re-applied and confirmed green again. See exact RED output below._

## Files Created/Modified

- `cmd/coherent_job_retry.go` - `planCoherentJobRetry`, `partialBuildRetryOutcome`, `reconcilePartialBuildRetry`, `findExistingBuildAttemptRetry`, `buildAttemptPathForID`, `commitPartialBuildCredit`.
- `cmd/coherent_job_retry_test.go` - Pure-planner fixtures (contains-only-unfinished, credited-dependency-satisfied, full-success-no-job, named cycle, named missing dependency, empty-input refusal) and the append-only attempt-journal proof (`TestBuildAttemptChildLinksParentWithoutMutation`, `TestLegacyBuildAttemptReadsWithoutParentFields`).
- `cmd/build_attempt.go` - `ParentAttemptID`/`ParentJobName` fields, new `buildAttemptPartial` status (added to `buildAttemptStatusTerminal`), `attachBuildAttemptParentLink`, `beginChildBuildAttempt`.
- `cmd/build_attempt_test.go` - `coherentJobPartialFailInvoker` (scripted direct-lane failing invoker carrying real task receipts), `TestGroupedJobPartialRetryIsAppendOnlyDirect`, `TestGroupedJobRetryNeverReassignsCreditedTasks`.
- `cmd/build_attempt_external_test.go` - `TestGroupedJobPartialRetryIsAppendOnlyExternal`, `TestGroupedJobPartialRetryIsIdempotent`.
- `cmd/codex_build.go` - Direct-lane partial-terminal path: `reconcilePartialBuildRetry` check before rollback, `commitPartialBuildCredit` call, structured recovery result.
- `cmd/codex_build_finalize.go` - External-lane `buildFullyCredited`-gated attempt status, `reconcilePartialBuildRetry` call, recovery fields merged into `result`.

## Decisions Made

See `key-decisions` in the frontmatter. In short: retry planning reuses the canonical grouping pass rather than a second graph implementation; parent-child linkage follows the existing checkFixAttemptRecord begin-then-attach precedent exactly; a new `partial` attempt status is honest where the prior unconditional `built` transition (195-06's own noted gap) was not; recovery strictly follows accepted credit, never a total failure; and idempotency is proven at the shared reconciler layer, not claimed at the full end-to-end resubmission layer (see Deviations).

## Deviations from Plan

### Auto-fixed Issues

None -- both tasks' `<action>`/`<behavior>` requirements were implemented as specified.

### Scoped decision (documented, not a Rule 1-4 fix)

**Full end-to-end idempotent resubmission through `runCodexBuildFinalize` is deferred, not implemented.** 195-06's own SUMMARY flagged that a partial attempt's "built" transition was left unconditional and asked this plan to "decide whether a partial build's attempt-journal status ... needs its own gating so a replay of an identical partial packet gets a smoother idempotent no-op instead of today's safe-but-blunt 'colony state has advanced' error." This plan's decision: gate the status (now `buildAttemptPartial`, not `built`) so the attempt journal is honest, and make the underlying recovery primitive (`reconcilePartialBuildRetry`) itself fully idempotent (proven directly, twice in a row, in `TestGroupedJobPartialRetryIsAppendOnlyExternal` and `TestGroupedJobPartialRetryIsIdempotent`). Making a literal second `aether build-finalize` CLI call with the byte-identical completion packet reach that idempotent path automatically is a separate, larger change: today it still hits `validateCodexBuildState`'s existing "phase %d is already active; run `aether continue`" refusal before ever reaching the reconciler a second time, because that validation was not part of this plan's `files_modified` scope and predates D-10 entirely. This is recorded here rather than silently narrowing the idempotency claim; the D-10 guarantee this plan is responsible for -- the retry mechanism itself never creates a duplicate child -- holds and is tested.

---

**Total deviations:** 0 auto-fixed. 1 documented scoping decision (idempotency proven at the shared-reconciler layer, full-CLI-resubmission smoothing explicitly out of scope), no functional gap against this plan's own acceptance criteria.
**Impact on plan:** None on correctness — every `<acceptance_criteria>` and the plan-level `<verification>` command passed exactly as specified.

## Issues Encountered

- TDD RED verification for both tasks required a captured `git diff`/`git apply -R` round-trip on the exact production files each task touches (never `git stash`, `git checkout --`, or any other rollback command, per this execution's constraints), to confirm a genuine failure before restoring and committing GREEN. Task 1's revert produced real compile errors (`undefined: planCoherentJobRetry`, `undefined: beginChildBuildAttempt`, `buildAttemptRecord has no field ParentAttemptID`) -- expected, since every referenced symbol is brand new. Task 2's revert produced real assertion failures against pre-existing, already-compiling code: the direct-lane test failed with "build dispatch did not complete cleanly" (today's unconditional rollback) and the external-lane test failed with "result did not report a D-10 recovery job" (today's unconditional `built` transition) -- exact output quoted below.
- `TestGroupedJobPartialRetryIsIdempotent` continued to pass even with Task 2's lane wiring reverted, because it exercises `reconcilePartialBuildRetry` directly (a Task 1 deliverable) rather than the lane wiring Task 2 adds. This is noted rather than silently presented as new Task-2-sensitive coverage; `TestGroupedJobPartialRetryIsAppendOnlyExternal` (which DOES go through the real `runCodexBuildFinalize` entrypoint and DID fail under the revert) is the test that actually proves Task 2's external-lane wiring.

### Genuine RED output (Task 1, compile error)

```
cmd/coherent_job_retry_test.go:22:14: undefined: planCoherentJobRetry
...
cmd/coherent_job_retry_test.go:181:19: undefined: beginChildBuildAttempt
cmd/coherent_job_retry_test.go:201:11: child.ParentAttemptID undefined (type buildAttemptRecord has no field or method ParentAttemptID)
cmd/coherent_job_retry_test.go:204:11: child.ParentJobName undefined (type buildAttemptRecord has no field or method ParentJobName)
```

### Genuine RED output (Task 2, real assertion failures)

```
--- FAIL: TestGroupedJobPartialRetryIsAppendOnlyExternal
    build_attempt_external_test.go:608: result did not report a D-10 recovery job: map[... state:EXECUTING ...]
--- FAIL: TestGroupedJobPartialRetryIsAppendOnlyDirect
    build_attempt_test.go:510: runCodexBuild should accept a validated partial-credit failure, got error: build dispatch did not complete cleanly: Anvil-40=failed (crashed after finishing four of six steps)
--- FAIL: TestGroupedJobRetryNeverReassignsCreditedTasks
    build_attempt_test.go:623: runCodexBuild: build dispatch did not complete cleanly: Anvil-40=failed (crashed after finishing four of six steps)
```

## Known Stubs

None -- every code path added is exercised by a real, passing test; no placeholder or unwired data path was introduced.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 195-08 (worktree wiring) inserts its own sync step between `admitCoherentJobTaskReceipts`/`finalizeCoherentJobTaskReceiptEvidence` exactly as already designed in 195-04/195-06; this plan's `reconcilePartialBuildRetry` is lane-neutral and reads only `dispatch.CompletedTaskIDs`/`CoveredTaskIDs`, so it needs no changes once worktree-mode receipts flow through the same two stages.
- A future plan may choose to smooth the full-CLI idempotent-resubmission path (see Deviations) by widening `runCodexBuildFinalize`'s early routing checks (currently gated on `buildAttemptBuilt` specifically) to also short-circuit on `buildAttemptPartial` before `validateCodexBuildState` runs -- deliberately left for a dedicated decision rather than folded in here.
- Both lanes now expose `recovery_job`, `parent_attempt_id`, `retry_attempt_id`, `retry_attempt_path`, `unfinished_task_ids`, and `recovery_command` on their result maps; a future wrapper/CLI-surfacing plan can render these directly without any further Go-side plumbing.

## Self-Check: PASSED

- All 7 declared files confirmed present on disk with the expected changes (`git status --short`, `git diff --stat`).
- All 4 commit hashes confirmed present in `git log --oneline` (`2e5a2fd9`, `ae52189b`, `667bbfc5`, `6d7e6d6b`).
- `go build ./...`, `go build ./cmd/aether`, and `go vet ./cmd/` are clean.
- `go run ./cmd/aether contract-schema --check` reports no drift (no completion-packet contract field changed).
- `go test ./cmd -run TestAuditCatalogGolden` passes (no CLI surface changed, no golden update needed).
- The plan's own verification command, `go test ./cmd -run 'Test(CoherentJobRetry|BuildAttemptChild|GroupedJobPartialRetry|GroupedJobRetryNever|LegacyBuildAttempt)' -count=1`, passes (12/12 subtests).
- A broader regression sweep across dispatch-coalesce/CalVault, grouped-job, coherent-job, build-attempt, external, merged-dispatch, wrapper-bundled-completion, build-does-not-advance, build-finalize, provenance, phase-verified-once, deterministic-floor, worktree, check-fix-attempt, and no-change tests (run together by name) all pass.

---
*Phase: 195-coherent-jobs*
*Completed: 2026-08-27*
