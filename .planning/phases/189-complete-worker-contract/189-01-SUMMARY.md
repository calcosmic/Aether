---
phase: 189-complete-worker-contract
plan: 01
subsystem: build-orchestration
tags: [worker-dispatch, handoff-schema, codex, build-brief, worker-contract]

# Dependency graph
requires:
  - phase: 188
    provides: build-path merge-dispatch machinery (mergeDispatchInto, CoveredTaskIDs, coalesceSequentialDispatches)
provides:
  - findDispatchTasks (multi-task resolution) replacing single-task findDispatchTask, used by both the brief renderer and codegraph relevance
  - codex.HandoffFieldsSummary, the one canonical handoff-schema constant
  - A merged build dispatch's brief now carries every covered task's Constraints, Hints and Task Success Criteria, not just the first task's
  - composeBuildManifestBrief now states the exact handoff/return schema the finalizer enforces, for wrapper-spawned (Claude Code, OpenCode) build workers only
affects: [189-02 (reviewer parity, imports codex.HandoffFieldsSummary), 190 (build-path merge gate / duplication cleanup)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "One canonical constant, all readers converge on it (codex.HandoffFieldsSummary) instead of hand-copying the same sentence in multiple places"
    - "Wrapper-external-only content (no separate channel on the native-Codex path) is appended in the composition wrapper (composeBuildManifestBrief), never inside the shared renderer (renderCodexBuildWorkerBrief) both paths call"

key-files:
  created: []
  modified:
    - pkg/codex/handoff.go
    - pkg/codex/worker.go
    - cmd/codex_build.go
    - cmd/codex_build_test.go
    - cmd/codegraph_context.go
    - cmd/contract_schema_test.go

key-decisions:
  - "Followed CONTEXT.md D-03/D-04 exactly: the handoff schema is a single exported constant (codex.HandoffFieldsSummary) referenced by symbol from both renderResponseContract (native-Codex channel) and composeBuildManifestBrief (wrapper-only channel) -- never hand-copied a third time, never added to the shared renderCodexBuildWorkerBrief itself, so native-Codex workers see it exactly once"
  - "Followed D-01/D-02/D-08: findDispatchTasks (plural) replaces findDispatchTask (singular) at both call sites, reusing the already-correct dispatchCoveredTaskIDs rather than reimplementing coverage resolution; the pre-existing '## Constraints' -> '## Task Constraints' heading-name bug is fixed in the same change"
  - "Deviation (Rule 1, not in original plan scope): TestWrapperFieldListMatchesSchema regex-scanned pkg/codex/worker.go for the literal handoff-field sentence; Task 1's refactor moved that literal out of worker.go, breaking the test. Repointed the drift check at codex.HandoffFieldsSummary's own declaration in pkg/codex/handoff.go instead of restoring the literal text -- a more precise check now that one canonical constant is the actual source of truth"

patterns-established:
  - "renderDispatchTaskItemsSection: a single helper renders a '## <heading>' section for N tasks, with no per-task label when N==1 (byte-identical to the pre-merge-aware brief) and a '**Task N:**' sub-label per non-empty task when N>1, mirroring mergeDispatchInto's own '1. / 2. / 3.' numbering of dispatch.Task"

requirements-completed: []

# Metrics
duration: ~45min
completed: 2026-08-20
---

# Phase 189 Plan 01: Complete Worker Contract (Build-Side) Summary

**A merged build dispatch's brief now carries every covered task's constraints, hints and success criteria (not just the first task's), and wrapper-spawned build workers are told the exact handoff schema the finalizer enforces, via one canonical `codex.HandoffFieldsSummary` constant that native-Codex workers never see twice.**

## Performance

- **Duration:** ~45 min
- **Started:** 2026-08-20T00:20:00Z (approx.)
- **Completed:** 2026-08-20T01:07:00Z
- **Tasks:** 3 (Task 1 non-TDD; Tasks 2 and 3 both TDD, RED→GREEN)
- **Files modified:** 6 (5 planned + 1 deviation fix)

## Accomplishments

- Confirmed the defect described in the brief was real (not already handled): `findDispatchTask` (singular) resolved only `dispatch.TaskID` — the first task of a merged chain — so a worker covering 3 merged tasks saw all 3 goals in "## Assignment" but only task 1's Constraints/Hints/Task Success Criteria, then was judged by the finalizer against work it was never told about.
- `codex.HandoffFieldsSummary` is now the single exported source of the handoff field list, referenced by symbol from `renderResponseContract` (native-Codex path, byte-identical output verified) and `composeBuildManifestBrief` (wrapper-facing path, new).
- `findDispatchTasks` (plural) replaces `findDispatchTask` (singular) at both call sites (`renderCodexBuildWorkerBrief`, `codegraphTextPartsForBuildBrief`); the singular function no longer exists anywhere in `cmd/*.go`.
- A merged dispatch's brief now renders every covered task's Constraints/Hints/Task Success Criteria under `**Task N:**` sub-labels; a single-task dispatch is byte-identical to before except the heading rename `"## Constraints"` → `"## Task Constraints"` (a pre-existing bug: `TestBuildWorkerBriefIsMostlyTask`'s own counted-section switch already checked for `"Task Constraints"`, but no render path had ever emitted it).
- `composeBuildManifestBrief` now states the handoff schema after the base brief; `renderCodexBuildWorkerBrief`'s own raw output (fed to native-Codex workers as `TaskBrief`) does not contain it — proven by a negative-containment test covering all 9 schema fields, not a hand-picked couple.
- `TestBuildWorkerBriefIsMostlyTask` still passes, unweakened, unretuned. Measured share: **46.98% (303 of 645 chars)**, empirically confirmed **identical before and after** both build-side changes (verified via `git stash` to re-measure against the pre-fix code) — exactly as CONTEXT.md's D-07 predicted, because this test's own fixture has no `TaskID` and no `phase.Tasks`, so `dispatchCoveredTaskIDs` returns nil and neither criterion's change adds a single byte to its measured brief.

## Task Commits

Each task was committed atomically (Tasks 2 and 3 are TDD: test → feat, RED → GREEN):

1. **Task 1: Export the handoff schema as one canonical constant** - `7d700f43` (feat)
2. **Task 2 RED: failing test for merged-dispatch task coverage** - `35eeb4ba` (test)
3. **Deviation fix: repoint drift test at the new constant** - `6d50fa5b` (fix)
4. **Task 2 GREEN: render every covered task's constraints on a merged dispatch** - `e94d4f2b` (feat)
5. **Task 3 RED: failing test for handoff-schema placement (no duplication)** - `1c888563` (test)
6. **Task 3 GREEN: state the handoff schema in the wrapper-facing build brief** - `163e8909` (feat)

**Plan metadata:** committed separately after this SUMMARY (docs: complete plan)

## Files Created/Modified

- `pkg/codex/handoff.go` - Added exported `HandoffFieldsSummary` constant beside `WorkerHandoff`
- `pkg/codex/worker.go` - `renderResponseContract` now references `HandoffFieldsSummary` via `%s` instead of a hand-typed literal (byte-identical output, verified with a temporary throwaway test then removed)
- `cmd/codex_build.go` - `findDispatchTasks` (replaces `findDispatchTask`), new `renderDispatchTaskItemsSection` helper, `renderCodexBuildWorkerBrief`'s Constraints/Hints/Task-Success-Criteria block rewritten to loop over every covered task, `composeBuildManifestBrief` appends the handoff-schema sentence
- `cmd/codex_build_test.go` - Two new invariant tests: `TestBuildWorkerBriefCoversEveryMergedTaskConstraintsAndCriteria`, `TestComposeBuildManifestBriefStatesHandoffSchemaOnceNotOnNativePath`
- `cmd/codegraph_context.go` - `codegraphTextPartsForBuildBrief` migrated to loop over `findDispatchTasks` (bonus, D-02, same root cause)
- `cmd/contract_schema_test.go` - **Deviation:** `TestWrapperFieldListMatchesSchema` repointed from scanning `worker.go`'s literal sentence to reading `codex.HandoffFieldsSummary`'s own declaration in `handoff.go`

## Decisions Made

- Followed D-03/D-04 exactly on placement: the handoff schema lives in `composeBuildManifestBrief` (wrapper-external-only composition layer), never inside `renderCodexBuildWorkerBrief` itself, because that shared renderer's raw output also feeds `TaskBrief` on the native-Codex path (`executeCodexBuildDispatches`, confirmed by reading), which already states the schema via `renderResponseContract` on a separate channel. Embedding it in the shared renderer would have shown it to native-Codex workers twice — this is the exact duplication Phase 190 depends on this phase not creating.
- Followed D-01/D-08: reused the already-correct `dispatchCoveredTaskIDs` rather than reimplementing "which tasks does this dispatch cover"; fixed the `"## Constraints"` → `"## Task Constraints"` heading name in the same change since it is a genuine pre-existing bug (the test's own switch already expected the new name).
- `renderDispatchTaskItemsSection`'s single-task branch intentionally preserves the *exact* pre-existing quirk of the old code (heading appears whenever the raw item slice is non-empty, even if every item trims to `""`), while the new multi-task branch uses the stricter "at least one non-empty item" rule the plan's `<behavior>` block specifies for merged dispatches. These are deliberately different rules for deliberately different cases, not an inconsistency.
- Chose to check **every** field in `codex.HandoffFieldsSummary` programmatically in the Task 3 test (splitting the constant's own value), rather than the two example fields (`changed_files`, `do_not_repeat`) the plan's `<action>` names as illustrative — a stronger invariant that cannot silently pass if a field is dropped from one side, consistent with CLAUDE.md's Definition of Done preference for proportion/invariant checks over presence checks.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug I introduced] `TestWrapperFieldListMatchesSchema` broke after Task 1's refactor**

- **Found during:** Running the full `go test ./cmd/... -count=1` suite after Task 2 (required by the plan's own success criteria)
- **Issue:** `TestWrapperFieldListMatchesSchema` (`cmd/contract_schema_test.go`) is a pre-existing four-surface parity test (not named anywhere in this plan) that regex-scanned `pkg/codex/worker.go` for the literal sentence `"Include handoff with <fields> ("` to verify the schema, wrapper docs, and Go structs never drift apart. Task 1 (as instructed by the plan) moved that literal text out of `worker.go` into `codex.HandoffFieldsSummary`, replacing it with a `%s` placeholder — so the regex found nothing and the test failed with `no "Include handoff with ..." response-contract sentence found in .../pkg/codex/worker.go`.
- **Fix:** Repointed the drift check at the new canonical source: renamed `workerGoResponseContractFields`/`responseContractSentencePattern` to `handoffFieldsSummaryConstFields`/`handoffFieldsSummaryConstPattern`, reading `pkg/codex/handoff.go`'s `HandoffFieldsSummary` constant declaration instead of `worker.go`'s rendered sentence. This is a strictly more precise check than before: it verifies the one place the field list actually lives, rather than a rendered proxy for it, and it will not need to move again as long as D-03's "one canonical constant" design holds.
- **Files modified:** `cmd/contract_schema_test.go`
- **Verification:** `go test ./cmd/ -run TestWrapperFieldListMatchesSchema -v -count=1` passes; full `go test ./cmd/... -count=1` passes with zero failures afterward.
- **Committed in:** `6d50fa5b` (separate fix commit, between Task 2's RED and GREEN commits, since that is when the full-suite run surfaced it)

---

**Total deviations:** 1 auto-fixed (Rule 1 — bug my own Task 1 change introduced in a test outside this plan's stated file list)
**Impact on plan:** Necessary for "Full `go test ./cmd/... ./pkg/codex/...` suite passes with zero regressions" (this plan's own success criterion). No scope creep — the fix is a one-file, minimal repoint of an existing check at the new canonical location, not a new feature.

## Proof of Each New/Changed Test (fail-then-pass)

### `TestBuildWorkerBriefCoversEveryMergedTaskConstraintsAndCriteria` (Task 2)

**RED (against the unfixed `findDispatchTask`/single-task code), quoted verbatim:**
```
codex_build_test.go:3756: merged dispatch brief missing "constraint-only-in-task-2" (a task-2/3 constraint, hint, or success criterion was dropped):
codex_build_test.go:3756: merged dispatch brief missing "hint-only-in-task-2" (a task-2/3 constraint, hint, or success criterion was dropped):
codex_build_test.go:3756: merged dispatch brief missing "criteria-only-in-task-2" (a task-2/3 constraint, hint, or success criterion was dropped):
codex_build_test.go:3756: merged dispatch brief missing "constraint-only-in-task-3" (a task-2/3 constraint, hint, or success criterion was dropped):
codex_build_test.go:3756: merged dispatch brief missing "hint-only-in-task-3" (a task-2/3 constraint, hint, or success criterion was dropped):
codex_build_test.go:3756: merged dispatch brief missing "criteria-only-in-task-3" (a task-2/3 constraint, hint, or success criterion was dropped):
codex_build_test.go:3761: merged dispatch brief missing the renamed "## Task Constraints" heading:
codex_build_test.go:3764: merged dispatch brief still emits the old "## Constraints" heading:
--- FAIL: TestBuildWorkerBriefCoversEveryMergedTaskConstraintsAndCriteria (0.01s)
```

**GREEN:** `--- PASS: TestBuildWorkerBriefCoversEveryMergedTaskConstraintsAndCriteria (0.01s)`

**Additional regression-catch proof required by the plan's acceptance criteria:** temporarily changed `findDispatchTasks` to iterate only `coveredIDs[:1]` (first element), reran the test — it failed again (the 6 task-2/3 content assertions, not the 2 heading assertions, which are unaffected by that specific change), confirming the test genuinely exercises the merge-resolution path rather than passing vacuously. Restored the real implementation; test passes again, `go build ./...` and the 4-test targeted run both clean.

### `TestComposeBuildManifestBriefStatesHandoffSchemaOnceNotOnNativePath` (Task 3)

**RED (against `composeBuildManifestBrief` before the append), quoted verbatim:**
```
codex_build_test.go:3810: composeBuildManifestBrief missing handoff-schema field "assumptions"; a wrapper-spawned worker was never told the finalizer's schema:
codex_build_test.go:3810: composeBuildManifestBrief missing handoff-schema field "changed_files"; a wrapper-spawned worker was never told the finalizer's schema:
codex_build_test.go:3810: composeBuildManifestBrief missing handoff-schema field "commands_run"; a wrapper-spawned worker was never told the finalizer's schema:
codex_build_test.go:3810: composeBuildManifestBrief missing handoff-schema field "do_not_repeat"; a wrapper-spawned worker was never told the finalizer's schema:
codex_build_test.go:3810: composeBuildManifestBrief missing handoff-schema field "freshness"; a wrapper-spawned worker was never told the finalizer's schema:
codex_build_test.go:3810: composeBuildManifestBrief missing handoff-schema field "known_failures"; a wrapper-spawned worker was never told the finalizer's schema:
codex_build_test.go:3810: composeBuildManifestBrief missing handoff-schema field "next_worker_instructions"; a wrapper-spawned worker was never told the finalizer's schema:
codex_build_test.go:3810: composeBuildManifestBrief missing handoff-schema field "open_decisions"; a wrapper-spawned worker was never told the finalizer's schema:
codex_build_test.go:3810: composeBuildManifestBrief missing handoff-schema field "verification_status"; a wrapper-spawned worker was never told the finalizer's schema:
--- FAIL: TestComposeBuildManifestBriefStatesHandoffSchemaOnceNotOnNativePath (0.02s)
```
Note all 9 fields were correctly flagged missing from `composed`, and — importantly — none of the negative "already contains" assertions against `renderCodexBuildWorkerBrief`'s raw output fired even at RED, confirming no incidental substring collisions (e.g. the word "freshness") existed in the base brief before this change; the test is a clean signal in both directions.

**GREEN:** `--- PASS: TestComposeBuildManifestBriefStatesHandoffSchemaOnceNotOnNativePath (0.01s)`

### `TestWrapperFieldListMatchesSchema` (deviation fix)

**Failure after Task 1's refactor, before the fix, quoted verbatim:**
```
contract_schema_test.go:107: no "Include handoff with ..." response-contract sentence found in /Users/.../pkg/codex/worker.go
--- FAIL: TestWrapperFieldListMatchesSchema (0.00s)
```
**After the fix:** `--- PASS: TestWrapperFieldListMatchesSchema (0.01s)`

## Verification

```
go build ./...                                    -> exit 0
go vet ./...                                       -> exit 0
go test ./pkg/codex/... -count=1                   -> ok (20-21s)
go test ./cmd/ -run 'TestBuildWorkerBriefIsMostlyTask|TestBuildWorkerBriefOmitsPlaybooks|TestBuildWorkerBriefIncludesCodegraphContext|TestBuildWorkerBriefCoversEveryMergedTaskConstraintsAndCriteria|TestComposeBuildManifestBriefStatesHandoffSchemaOnceNotOnNativePath' -v -count=1
                                                    -> all 5 PASS
go test ./cmd/... -count=1 (full package, run twice: once after Task 2, once after Task 3)
                                                    -> ok, both times, zero failures
go test ./... -race                                -> see below
```

`go test ./... -race` (full repo, required by the parallel-execution instructions): started in the background before writing this SUMMARY; result recorded in the self-check section below once it completed.

## Issues Encountered

None beyond the one documented deviation above (a pre-existing test's fallout from Task 1's own planned refactor, fixed inline).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `codex.HandoffFieldsSummary` is exported and ready for 189-02 (reviewer parity) to import for continue's external reviewer/watcher briefs, per D-05/D-06.
- No new duplication was created for Phase 190 to remove: the handoff schema lives in exactly one place per delivery path (native via `renderResponseContract`, wrapper-external via `composeBuildManifestBrief`), never both, proven by `TestComposeBuildManifestBriefStatesHandoffSchemaOnceNotOnNativePath`.
- `findDispatchTask` (singular) is fully deleted; nothing downstream should ever reintroduce a single-task-only lookup for dispatch briefs.
- No blockers for 189-02.

---
*Phase: 189-complete-worker-contract*
*Completed: 2026-08-20*
