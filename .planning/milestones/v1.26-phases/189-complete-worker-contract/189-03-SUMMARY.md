---
phase: 189-complete-worker-contract
plan: 03
subsystem: infra
tags: [go, codex, worker-handoff, finalize, code-review-remediation]

# Dependency graph
requires:
  - phase: 189-complete-worker-contract (plans 01, 02)
    provides: merged-dispatch task coverage, the handoff schema statement in wrapper-external briefs, continue's capsule/pheromone delivery
provides:
  - Continue's external finalize rejects a "completed" result whose handoff carries no relay content, matching build's long-standing enforcement (CR-01)
  - Merged-dispatch brief "Task N:" labels survive an unresolvable covered task ID without shifting later tasks' labels; unresolved IDs are visibly marked (WR-01)
  - A merged-dispatch brief is covered by the same task-content-dominates invariant as the single-task brief (WR-02)
  - One canonical freshness-inclusive empty-handoff check shared by both packages instead of three near-duplicates (IN-01)
affects: [190-print-brief-dedup, any future continue/seal finalize work, any future merged-dispatch brief rendering work]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Terminal-rejection empty-handoff check placed in the merge/validation function (mergeExternalContinueResults), mirroring build's placement in persistExternalBuildHandoffs, so both finalize chains fail at the equivalent point with the equivalent error class"
    - "Position-stable covered-task resolution: findDispatchTasks keeps every covered ID's slot (Task nil when unresolved) instead of filtering it out, so 1-based Position survives gaps"

key-files:
  created: []
  modified:
    - cmd/codex_continue_finalize.go
    - cmd/codex_continue_finalize_test.go
    - cmd/codex_continue_test.go
    - cmd/codex_workflow_cmds_test.go
    - cmd/consolidation_lifecycle_test.go
    - cmd/finality_parity_test.go
    - cmd/orchestrator_boundary_guidance_test.go
    - cmd/finalizer_completion_contract_test.go
    - cmd/codex_build.go
    - cmd/codegraph_context.go
    - cmd/build_print_brief.go
    - cmd/codex_build_test.go
    - pkg/codex/handoff.go
    - pkg/codex/worker.go
    - cmd/codex_dispatch_contract.go

key-decisions:
  - "Empty-handoff rejection placed inside mergeExternalContinueResults (not persistExternalContinueHandoffs), matching where build's equivalent early-validation logic lives and guaranteeing the check runs before any persistence occurs"
  - "WARNING 1 and WARNING 2 committed together (one commit, not two): both touch the identical merged-dispatch brief-rendering path in the same test file, and WR-02 is explicitly an extension of WR-01's own code path's test coverage -- splitting them would have required a risky hand-rolled patch split of a single contiguous test-file diff for no safety benefit"
  - "IN-01 consolidated into ONE exported function (IsEmptyWorkerHandoffIncludingFreshness) rather than merging into IsEmptyWorkerHandoff, because the two are not interchangeable: terminal-rejection checks must not count a bare Freshness timestamp as content, but pre-synthesis fallback checks (which run before NormalizeWorkerHandoff stamps a blank Freshness) must not let a synthesized replacement clobber an explicit worker-supplied Freshness value"

patterns-established:
  - "A promise stated in a worker brief (\"an empty handoff is rejected\") must be traced to an actual enforcing check on every finalize path that states it, not assumed from a similarly-named function existing somewhere in the same file"

requirements-completed: []  # No PLAN.md requirements frontmatter for this remediation pass; see 189-REVIEW.md for the findings this closes (CR-01, WR-01, WR-02, IN-01)

# Metrics
duration: ~50min
completed: 2026-08-20
---

# Phase 189 Plan 3: Code Review Remediation Summary

**Continue's finalize now rejects empty worker handoffs like build's always has; merged-dispatch briefs keep correct task numbering across an unresolvable task ID and are covered by the same scaffolding-ratio test as single-task briefs; three duplicate empty-handoff checks collapsed to one.**

## Performance

- **Duration:** ~50 min (from 189-REVIEW.md's 06:51:36Z timestamp to final commit)
- **Completed:** 2026-08-20
- **Commits:** 3 (one per finding group)
- **Files modified:** 15

## Accomplishments

- Closed the BLOCKER (CR-01): continue's external finalize chain now enforces the "an empty handoff is rejected" promise its own wrapper brief states, mirroring build's `persistExternalBuildHandoffs` guard exactly, including the "no handoff at all" case.
- Closed WARNING 1 (WR-01): a merged dispatch's brief no longer mislabels later tasks when an earlier covered task ID fails to resolve — every resolvable task keeps its true `Task N:` position, and unresolved IDs are now visibly flagged in a new "Task Resolution Notice" section instead of vanishing silently.
- Closed WARNING 2 (WR-02): `TestBuildWorkerBriefIsMostlyTask`'s scaffolding-ratio regression lock is now also exercised against a real merged (multi-task) brief, closing the gap where the code this phase added sat entirely outside that test's protection.
- Closed the INFO item (IN-01): the three near-identical "is this handoff empty" functions are down to two genuinely-distinct ones, both exported from `pkg/codex`, with the duplication's cause (and the reason it isn't a full merge) documented inline.
- Fixed 13 pre-existing test fixtures across the continue/build/seal finalize test suite whose "completed" worker results carried no handoff content — previously silently accepted, now correctly rejected, exposing a real coverage gap in how realistically those fixtures modeled a completed worker.

## Task Commits

1. **BLOCKER (CR-01):** `133f4840` — `fix(189-03): continue's finalize now rejects empty handoffs (CR-01)`
2. **WARNING 1 + WARNING 2 (WR-01, WR-02):** `df128044` — `fix(189-03): merged-brief numbering survives unresolved task IDs (WR-01, WR-02)`
3. **INFO (IN-01):** `a589fc2c` — `refactor(189-03): consolidate the three empty-handoff checks (IN-01)`

## Files Created/Modified

- `cmd/codex_continue_finalize.go` — `mergeExternalContinueResults` now rejects a `"completed"` result whose `Handoff` is empty, mirroring build's finalize enforcement
- `cmd/codex_continue_finalize_test.go` — new `TestContinueFinalizeRejectsCompletedWorkerWithoutHandoff`, driving the real `runCodexContinueFinalize` entry point for both the explicit-empty and no-handoff-key-at-all cases
- `cmd/codex_continue_test.go`, `cmd/codex_workflow_cmds_test.go`, `cmd/consolidation_lifecycle_test.go`, `cmd/finality_parity_test.go`, `cmd/orchestrator_boundary_guidance_test.go`, `cmd/finalizer_completion_contract_test.go` — fixture fixes: 13 pre-existing "completed" result fixtures across these files now carry realistic non-empty handoffs
- `cmd/codex_build.go` — `findDispatchTasks` returns `[]coveredDispatchTask{Position, ID, Task}` instead of `[]*colony.Task`, preserving each covered ID's original chain position even when unresolved; `renderDispatchTaskItemsSection` labels by `.Position`, not filtered-slice index; new `renderUnresolvedDispatchTaskNotice` visibly flags any unresolved covered task ID
- `cmd/codegraph_context.go` — `codegraphTextPartsForBuildBrief` updated for the new `coveredDispatchTask` return type, skipping unresolved entries
- `cmd/build_print_brief.go` — registered `"Task Resolution Notice"` in `briefOwnedSections` so `splitBriefSections` attributes it as its own section (counted as scaffolding, not task content — the conservative choice)
- `cmd/codex_build_test.go` — new `TestBuildWorkerBriefMergedDispatchKeepsNumberingAcrossUnresolvedTask` (WR-01 proof) and `TestBuildWorkerBriefIsMostlyTaskForMergedDispatch` (WR-02 proof, sibling to the existing `TestBuildWorkerBriefIsMostlyTask`, unweakened)
- `pkg/codex/handoff.go` — `workerHandoffIsEmpty` renamed and exported to `IsEmptyWorkerHandoffIncludingFreshness`, with a doc comment explaining why it stays distinct from `IsEmptyWorkerHandoff`
- `pkg/codex/worker.go` — `normalizeWorkerClaims` calls the renamed exported function
- `cmd/codex_dispatch_contract.go` — deleted the cmd-local `workerHandoffEmpty` duplicate; `buildWorkerHandoffRecord` now calls `codex.IsEmptyWorkerHandoffIncludingFreshness`

## Decisions Made

- **Empty-handoff check placement:** inside `mergeExternalContinueResults`, immediately after the existing `codex.ValidateWorkerHandoff` call, gated on `ok && status == buildWorkerCompleted`. This runs before `persistExternalContinueHandoffs` (the next step in `runCodexContinueFinalize`), so the rejection happens at the same relative point build's `persistExternalBuildHandoffs` check does, with an error that reaches the caller identically (a plain `error` return, not a swallowed violation).
- **"No handoff at all" is the same code path as "empty handoff," not a separate branch.** `codexContinueExternalDispatch.Handoff` is a value type (`codex.WorkerHandoff`, not `*codex.WorkerHandoff`), so JSON that omits the `"handoff"` key entirely and JSON that supplies `"handoff":{}` decode to the identical zero value. The new test proves this by decoding real JSON with the key omitted and asserting it hits the same rejection.
- **WARNING 1 and WARNING 2 combined into one commit.** Both touch `renderCodexBuildWorkerBrief`'s merged-dispatch path and the same test file; WR-02 is explicitly a coverage extension of the exact code WR-01 fixes (189-CONTEXT.md's own D-07 already treats the two as one design question). Splitting a single contiguous test-file addition into two commits would have required constructing a hand-rolled `git apply` patch against auto-generated hunk line-count headers — real risk of silently corrupting carefully fail-then-pass-verified test code for a purely organizational goal. Judged not worth the risk; the commit message names both findings explicitly.
- **`"Task Resolution Notice"` counted as scaffolding, not task content**, in both `TestBuildWorkerBriefIsMostlyTask`'s switch (untouched) and `renderBriefChecklist`'s switch (untouched) — the conservative choice, since this section only ever appears in the (today, latent) stale-ID edge case and treating it as scaffolding can only make the 40% floor *stricter* to pass, never looser.
- **IN-01: two functions kept, not one.** `IsEmptyWorkerHandoff` (freshness-excluded, terminal rejection) and `IsEmptyWorkerHandoffIncludingFreshness` (freshness-included, pre-synthesis fallback decision) test genuinely different questions at genuinely different points in the pipeline — merging them would either let a bare timestamp pass as "real content" at the rejection gate, or let a synthesized handoff silently overwrite an explicit worker-supplied Freshness value earlier in the pipeline. Consolidated the *duplication* (3 → 2 definitions, one shared exported symbol per distinct question) without touching either behavior.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed 13 pre-existing test fixtures broken by the correct CR-01 enforcement**
- **Found during:** Verifying the BLOCKER fix and (later) the final full-suite race run
- **Issue:** `go test ./cmd/...` (targeted, then full) surfaced 13 pre-existing tests whose fixtures constructed a `"completed"` continue/seal worker result with no `Handoff` field set at all — behavior the old, permissive code silently accepted. With CR-01 fixed, these are correctly rejected, which broke the fixtures, not the fix.
- **Fix:** Added a realistic, non-empty `Handoff` (`VerificationStatus: "pass"` plus `NextWorkerInstructions`, or a failing-verification handoff where the test's own scenario calls for one) to each fixture's `"completed"` result(s).
- **Files modified:** `cmd/codex_continue_test.go`, `cmd/codex_workflow_cmds_test.go`, `cmd/consolidation_lifecycle_test.go`, `cmd/finality_parity_test.go` (plus a new `parityResultWithHandoff` helper), `cmd/orchestrator_boundary_guidance_test.go`, `cmd/finalizer_completion_contract_test.go`
- **Verification:** Each fixed test re-run individually (pass); full `go test ./... -race` re-run afterward found no further fixture gaps (see Proof section).
- **Committed in:** `133f4840` (BLOCKER commit)

**2. [Rule 3 - Blocking] `full-race-test-2.log` reads raced against concurrent Edit calls twice**
- **Found during:** Verification, before the INFO fix
- **Issue:** Two background `go test`/`go test -race` runs read source files mid-edit (once during the INFO consolidation edit sequence, once earlier during a plain `go test ./cmd/...`), producing a spurious `undefined: workerHandoffEmpty` build-failure artifact that was not a real test failure.
- **Fix:** Recognized the timing race (confirmed via immediate `go build`/`go vet`/targeted `go test` all passing cleanly on the stable file state), then re-ran a clean, `pipefail`-protected full race suite with no concurrent edits in flight, which produced the genuine result used for the Proof section below.
- **Files modified:** None (process discipline fix, not a code fix)
- **Committed in:** N/A (no code change)

---

**Total deviations:** 2 (1 required test-fixture repair across 6 files touching 13 fixtures, 1 process-discipline correction with no code impact)
**Impact on plan:** The fixture repair is a direct, necessary consequence of correctly closing CR-01 — every one of those 13 fixtures was modeling an unrealistic "completed worker with nothing to relay" scenario that the fix correctly stopped accepting. No scope creep beyond what CR-01's own fix required to keep the suite green.

## Proof (fail-then-pass, quoted per CLAUDE.md's Definition of Done)

### BLOCKER (CR-01)

Before the fix, `TestContinueFinalizeRejectsCompletedWorkerWithoutHandoff` (driving the real `runCodexContinueFinalize` entry point):
```
=== RUN   TestContinueFinalizeRejectsCompletedWorkerWithoutHandoff
phase advanced WITHOUT consolidation — load instincts: ...
    codex_continue_finalize_test.go:367: completed continue worker with an explicitly empty handoff was accepted; the finalizer must reject content-free relays, matching build's own enforcement
--- FAIL: TestContinueFinalizeRejectsCompletedWorkerWithoutHandoff (0.98s)
```
(Note the empty handoff was not just "accepted" — the phase actually advanced.)

After the fix:
```
--- PASS: TestContinueFinalizeRejectsCompletedWorkerWithoutHandoff (0.73s)
```
The same test also proves the no-handoff-key-at-all case via real JSON decoding (`json.Unmarshal` on a payload with no `"handoff"` key), asserting it hits the identical rejection.

### WARNING 1 (WR-01)

Before the fix, `TestBuildWorkerBriefMergedDispatchKeepsNumberingAcrossUnresolvedTask`:
```
codex_build_test.go:3827: task 3's constraint did not render under its correct label "**Task 3:**" (numbering desync across the unresolved task):
...
## Task Constraints

**Task 1:**
- constraint-only-in-task-1
**Task 2:**
- constraint-only-in-task-3
...
codex_build_test.go:3847: brief does not visibly mark task 2 as unresolved -- a worker has no way to tell "task 2 has no constraints" from "task 2's constraints could not be found"
--- FAIL: TestBuildWorkerBriefMergedDispatchKeepsNumberingAcrossUnresolvedTask (0.01s)
```
Task 3's content was misattributed to the label "Task 2", and task 2's absence carried no marker at all.

After the fix: `--- PASS`, task 1 renders under "Task 1:", task 3 renders under "Task 3:" (never "Task 2:"), and the brief contains `Task 2 (id "2") could not be resolved against this phase's task list...`.

### WARNING 2 (WR-02)

`TestBuildWorkerBriefIsMostlyTaskForMergedDispatch` measures and logs the real number: **67.2% (735 of 1094 chars)** task-relevant content for a real 3-task merged dispatch — well clear of the 40% floor, and improving (not degrading) relative to the single-task case, consistent with 189-CONTEXT.md's D-07 prediction.

Proof the invariant actually binds (temporarily removing the merged-task rendering and injecting a large scaffolding block, then reverting byte-for-byte, confirmed via `git diff --exit-code -- cmd/codex_build.go` before committing):
```
codex_build_test.go:3926: merged-dispatch task-relevant share: 9.3% (339 of 3661 chars)
codex_build_test.go:3928: task-relevant content is 9.3% of the merged-dispatch worker brief (339 of 3661 chars); framework scaffolding now outweighs the task
--- FAIL: TestBuildWorkerBriefIsMostlyTaskForMergedDispatch (0.01s)
```
`TestBuildWorkerBriefIsMostlyTask` itself was never modified — its threshold, counted-section list, and fixture are byte-identical to before this plan.

### Full-suite verification

- `go build ./...` — clean.
- `go vet ./...` — clean.
- `go test ./... -race` — ran to genuine completion (verified via `pipefail`-protected exit code plus direct process inspection, after an earlier attempt raced against concurrent edits and had to be discarded). Result: every package `ok` except one real, pre-existing-fixture failure in `cmd` (`TestLifecycleResultMergersPreferCompletedResultOverTimeoutPlaceholder/continue`, `finalizer_completion_contract_test.go:282` — same root cause as the 13 fixtures above, found by the full run rather than the targeted ones). Fixed (added a realistic handoff to that fixture too, included in the BLOCKER commit) and re-verified individually:
  ```
  --- PASS: TestLifecycleResultMergersPreferCompletedResultOverTimeoutPlaceholder (0.00s)
      --- PASS: TestLifecycleResultMergersPreferCompletedResultOverTimeoutPlaceholder/continue (0.00s)
  ```
  No `WARNING: DATA RACE` output anywhere in the full log.
- A final foreground `go test ./cmd/ -count=1` was started per the coordinator's instruction but exceeded the tool's synchronous timeout and moved to the background; not waited on further per the coordinator's explicit "do not wait again" instruction. The evidence above (the full, genuinely-completed `-race` run, plus every individual affected test re-verified synchronously after each fix) is the basis for this SUMMARY's pass claim, not that backgrounded run.

## Known Stubs

None.

## Threat Flags

None — this plan only tightens an existing validation gate (rejecting previously-silently-accepted empty handoffs) and fixes a brief-rendering labeling bug; no new network endpoints, auth paths, file-access patterns, or schema changes at a trust boundary were introduced.

## Issues Encountered

- Two background test runs raced against my own concurrent file edits (see Deviations #2), producing misleading build-failure artifacts that were not real. Resolved by recognizing the timing pattern and re-running cleanly with no concurrent edits in flight.
- A background full-suite `go test ./... -race` run that had genuinely completed was, on my end, mistaken for still-running due to a stale `ps aux` snapshot catching a different, unrelated process tree; the coordinator's notification was correct. Resolved per the coordinator's explicit redirection to stop waiting and commit immediately.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- All four review findings (1 blocker, 2 warnings, 1 info) from `189-REVIEW.md` are closed, each with fail-then-pass proof.
- Continue's and build's finalize chains now enforce the identical "empty handoff is rejected" contract their briefs both state — the exact symmetry this phase's own framing named as its goal.
- Phase 190 (`depends_on: 189`) can proceed on a clean base: no new duplication was introduced by this remediation pass (IN-01 reduced duplication; the BLOCKER and WARNING fixes touch enforcement/labeling logic only, not the brief-composition-layer placement decisions 189-01/189-02 already made).
- No blockers for the orchestrator's own final race-suite re-run after merge.

---
*Phase: 189-complete-worker-contract*
*Completed: 2026-08-20*

## Self-Check: PASSED

- Commits found: `133f4840`, `df128044`, `a589fc2c` (all confirmed via `git log --oneline --all`)
- Files found: `cmd/codex_continue_finalize.go`, `cmd/codex_continue_finalize_test.go`, `cmd/codex_build.go`, `cmd/codex_build_test.go`, `pkg/codex/handoff.go`, `pkg/codex/worker.go`, `cmd/codex_dispatch_contract.go`, `cmd/finalizer_completion_contract_test.go` (all confirmed present via `ls -la`)
