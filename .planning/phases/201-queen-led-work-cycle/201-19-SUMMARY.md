---
phase: 201-queen-led-work-cycle
plan: "19"
subsystem: infra
tags: [go, cli, cobra, lifecycle, build-orchestration, closeout]

# Dependency graph
requires:
  - phase: 201-queen-led-work-cycle
    provides: "attachVerificationBoundary (plan 201-16) — the boundary decision now persisted on every attempt; attachResultFilePrecision/attachBuildKnowledgeDeltas on both lanes (plan 201-18) — the file split and knowledge deltas now attached on every attempt"
provides:
  - "buildWorkCloseoutDetails (cmd/work_closeout.go): one total resolver turning a phase's sealed build attempt into a verdict-carrying LifecycleCloseoutDetails, using only the attempt's own recorded status, free-check report, verification-boundary decision, and dispatch statuses"
  - "renderBuildResultFileSection: the credited/uncredited file card (D-08), rendered from the same sealed attempt, empty string when nothing was recorded"
  - "Two new honest closeout builders (buildFailingChecksCloseoutDetails, buildNoCheckReportCloseoutDetails) for the failed-checks and no-check-report cases the existing pair (buildUnverifiedCloseoutDetails/buildVerifiedCloseoutDetails) did not cover"
  - "Both build lanes (buildCmd's direct-dispatch RunE, cmd/codex_workflow_cmds.go; renderCeremonyCloseout, cmd/ceremony_cmd.go) now render the verdict, the recommended next action, and the file card on a finished build, falling back to today's exact rendering when no verdict resolves or the result carries no lifecycle projection"
  - "A dedup guard in buildLifecycleCloseout (cmd/lifecycle_closeout.go) preventing the same knowledge-delta evidence from rendering twice now that a real WorkOutcome reaches production for the first time"
  - "TestBuildWorkCloseoutDetailsCoversEveryTerminalStatus / TestBuildCloseoutCarriesTheRealVerdict / TestBuildCloseoutVerdictHasProductionCallers: unit, end-to-end, and AST call-graph proof that the wiring is real and cannot be silently disconnected"
affects: [201-queen-led-work-cycle]

# Actuals (#2632)
actuals:
  tokens: 12093
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Evidence-ID dedup at the closeout boundary: a caller-supplied LifecycleCloseoutDetails.Evidence entry and a downstream auto-derived evidence list are combined by excluding any ID already present in the caller-supplied list, rather than asserting the two producers can never overlap"
    - "AST cobra-literal RunE lookup: a call-graph guard covering an anonymous RunE closure (invisible to a named-FuncDecl indexer) by parsing the file directly for the &cobra.Command{} composite literal whose Use field matches, extracting its RunE *ast.FuncLit, and searching that subtree for the target call"

key-files:
  created:
    - cmd/work_closeout.go
    - cmd/work_closeout_test.go
  modified:
    - cmd/lifecycle_closeout.go
    - cmd/codex_workflow_cmds.go
    - cmd/ceremony_cmd.go

key-decisions:
  - "Added a Rule-1 dedup guard (excludeLifecycleEvidenceByID) inside buildLifecycleCloseout (cmd/lifecycle_closeout.go), outside this plan's declared file list. buildWorkCloseoutDetails attaches the attempt's own knowledge-delta evidence to its returned details (an explicit Task 1 acceptance criterion), and buildLifecycleCloseout already auto-attaches the SAME evidence downstream from the same attempt once a real WorkOutcome is present — a pre-existing branch that was dead code until this plan supplied the first real verdict. Without the dedup, every knowledge delta would render twice on the finished screen for the first time this plan makes that path live. The guard is ID-based, order-preserving, and a no-op for every other existing caller of buildLifecycleCloseout (none of which populate details.Evidence with matching IDs)."
  - "Task 1's mapping is total in a fixed precedence: (1) every terminal dispatch reported no change -> the no-change verdict, (2) any dispatch's terminal status is a worker timeout -> the timeout verdict, checked BEFORE the attempt's own status switch, because neither carve-out is fully described by the four attempt-level status constants alone (buildAttemptBuilt/Partial/Failed/Interrupted) — a fully-credited all-no-change build and a build whose failure was a timeout both still carry an ordinary terminal status. Only after those two carve-outs does the attempt's own status decide: built (further resolved by check-report/boundary/reviewer evidence), partial, failed (-> blocker), interrupted."
  - "buildBuiltStatusCloseoutDetails checks in order: no check report at all (never claims checks passed) -> failing check report (blocker, naming the failed checks) -> build-end boundary with build-end reviewers actually passed (the only path to the success verdict) -> otherwise the check-step default (partial, honest 'not verified yet'). No branch falls through to success without a real passing reviewer dispatch."
  - "buildEndReviewerCastes is derived from queenBuildPostWavePlans (cmd/codex_build.go) at package-init time rather than a second, hand-typed caste list, so the closeout's notion of 'a build-end reviewer' can never silently drift from the Queen's own post-wave dispatch table."
  - "The direct lane (codex_workflow_cmds.go) sets result[\"current_phase\"] = phaseNum only inside the verdict-resolved branch, and deletes it again if applyLifecycleCloseout errors, so the no-verdict fallback path's result map (and therefore its JSON envelope) stays byte-identical to before this plan."
  - "renderCeremonyCloseoutVisual was split into renderCeremonyCloseoutVisualBody (everything through the closing block) and a thin wrapper appending the one cost line for build/continue, so the new build-workflow verdict branch in renderCeremonyCloseout can share the exact same body without either path producing two cost blocks."

requirements-completed: [CEC-06, WORK-04, WORK-05]

coverage:
  - id: D1
    description: "buildWorkCloseoutDetails resolves a sealed build attempt to one of the six declared verdicts, total over every terminal status, never falling through to success by default"
    requirement: "WORK-04"
    verification:
      - kind: unit
        ref: "cmd -run TestBuildWorkCloseoutDetailsCoversEveryTerminalStatus"
        status: pass
    human_judgment: false
  - id: D2
    description: "A finished build's screen on both lanes (direct RunE and the wrapper's ceremony closeout) carries the resolved verdict, the recommended next action, and the credited/uncredited file card, with exactly one cost block and one next-action card, falling back to today's exact rendering when no verdict resolves"
    requirement: "WORK-05"
    verification:
      - kind: integration
        ref: "cmd -run TestBuildCloseoutCarriesTheRealVerdict"
        status: pass
    human_judgment: false
  - id: D3
    description: "Both resolvers (buildWorkCloseoutDetails, renderBuildResultFileSection) have production callers and both build render paths reach them; a synthetic disconnection is refused by name and position"
    requirement: "WORK-04"
    verification:
      - kind: unit
        ref: "cmd -run TestBuildCloseoutVerdictHasProductionCallers"
        status: pass
    human_judgment: false
  - id: D4
    description: "The attempt's own recorded knowledge deltas reach the closeout as evidence, without duplication once a real WorkOutcome flows through buildLifecycleCloseout for the first time"
    requirement: "CEC-06"
    verification:
      - kind: unit
        ref: "cmd -run TestBuildWorkCloseoutDetailsCoversEveryTerminalStatus/knowledge_deltas_reach_the_resolved_details_as_evidence"
        status: pass
      - kind: integration
        ref: "cmd -run TestBuildCloseoutCarriesTheRealVerdict"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-09-11
status: complete
---

# Phase 201 Plan 19: Wire the Build Closeout's Verdict, Recommendation, and File Card into Production Summary

**`buildWorkCloseoutDetails` now resolves a phase's real sealed build attempt to one of six honest verdicts, and both build lanes' ending screens actually show it — closing the root cause behind D-03, D-05, D-07, D-08's rendering, and CAP-066: the closeout layer was fully built and unit-tested, but no production call site had ever set `LifecycleCloseoutDetails.WorkOutcome`, so every real build kept showing the pre-Phase-201 screen regardless of what actually happened.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-09-10 (approx., continuing from 201-18)
- **Completed:** 2026-09-11
- **Tasks:** 3 completed
- **Files modified:** 3 modified, 2 created

## Accomplishments

- `buildWorkCloseoutDetails` (`cmd/work_closeout.go`, new): the one resolver mapping a sealed build attempt to a declared work verdict, derived only from the attempt's own status, free-check report, recorded verification-boundary decision, and dispatch statuses — never from rendered text. Two carve-outs (all-dispatches-no-change, any-dispatch-timeout) are checked before the attempt's own status decides between built/partial/failed/interrupted, and the built case is further resolved through two new honest builders (failing checks -> blocker naming the failures; no check report at all -> partial, never claiming checks passed) plus the two existing tested builders (`buildUnverifiedCloseoutDetails`, `buildVerifiedCloseoutDetails`), reused rather than reimplemented.
- `renderBuildResultFileSection`: the D-08 credited/uncredited file card, rendered from the same sealed attempt, empty for an attempt with no recorded files.
- Both build lanes wired: `buildCmd`'s direct-dispatch `RunE` (`cmd/codex_workflow_cmds.go`) and the wrapper's `renderCeremonyCloseout` (`cmd/ceremony_cmd.go`) now resolve the verdict, apply it through the existing `applyLifecycleCloseout`, and render with `appendLifecycleCloseoutVisual` over the existing body plus the file card — falling back to exactly today's rendering when no verdict resolves or the result carries no lifecycle projection.
- `renderCeremonyCloseoutVisual` split into a body function and a thin cost-line wrapper so the new verdict-carrying path and the generic renderer share one body without ever producing two cost blocks.
- A dedup guard (`excludeLifecycleEvidenceByID`, `cmd/lifecycle_closeout.go`) stops the same knowledge-delta evidence from rendering twice now that a real verdict reaches `buildLifecycleCloseout`'s previously-dead knowledge-delta auto-attachment branch for the first time.
- `cmd/work_closeout_test.go` (new): a unit test proving every terminal status and every declared verdict is reachable from a genuinely-constructed attempt (real writers, never hand-typed JSON); an end-to-end test driving one real native build to completion and proving both lanes' finished screens carry the resolved verdict, the recommended action, the file card, and the knowledge deltas, each with exactly one cost block and one next-action card; and an AST call-graph guard (including a cobra-RunE-closure lookup for the anonymous direct lane, plus a synthetic-disconnection sub-case) proving both resolvers have production callers on both render paths.

## Task Commits

Each task was committed atomically:

1. **Task 1: One resolver from a sealed attempt to a verdict-carrying closeout** - `3862e737` (feat)
2. **Task 2: Render the verdict, the recommendation and the file card on both build lanes** - `dce8ee20` (feat)
3. **Task 3: Prove a real build closeout carries the verdict and the honest wording** - `1282f163` (test)

**Plan metadata:** (this commit)

## Files Created/Modified

- `cmd/work_closeout.go` (new) - `buildWorkCloseoutDetails`, `renderBuildResultFileSection`, `buildBuiltStatusCloseoutDetails`, `buildNoCheckReportCloseoutDetails`, `buildFailingChecksCloseoutDetails`, `buildPartialStatusCloseoutDetails`, `buildBlockerStatusCloseoutDetails`, `buildInterruptedStatusCloseoutDetails`, `buildNoChangeCloseoutDetails`, `buildTimeoutCloseoutDetails`, dispatch-status carve-out helpers, and `buildEndReviewerCastes`/`buildEndReviewerDispatches`/`buildEndReviewersPassed`/`buildEndReviewerNames`.
- `cmd/lifecycle_closeout.go` - Added `excludeLifecycleEvidenceByID` and wired it into `buildLifecycleCloseout`'s existing knowledge-delta attachment to prevent double-rendering once a real verdict flows through it.
- `cmd/codex_workflow_cmds.go` - `buildCmd`'s direct-dispatch ending screen now resolves the verdict, applies the closeout, and renders it with the file card appended, falling back to the original rendering otherwise.
- `cmd/ceremony_cmd.go` - Split `renderCeremonyCloseoutVisual` into a body function plus a thin cost-line wrapper; `renderCeremonyCloseout` now resolves and applies the build workflow's verdict right after `applyLifecycleNextAction`.
- `cmd/work_closeout_test.go` (new) - `TestBuildWorkCloseoutDetailsCoversEveryTerminalStatus`, `TestBuildCloseoutCarriesTheRealVerdict`, `TestBuildCloseoutVerdictHasProductionCallers`.

## Decisions Made

See `key-decisions` in frontmatter for the full list. The most consequential: adding the knowledge-delta dedup guard to `cmd/lifecycle_closeout.go` — a file outside this plan's declared `files_modified` list — under deviation Rule 1 (auto-fix bugs directly caused by the current task's changes). Without it, this plan would have shipped a visible duplicate-evidence bug the moment it became the first production caller to ever populate `WorkOutcome`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Knowledge-delta evidence would have rendered twice**
- **Found during:** Task 1
- **Issue:** `buildWorkCloseoutDetails` is instructed to attach the attempt's own knowledge deltas to its returned `details.Evidence` (an explicit acceptance criterion). `buildLifecycleCloseout` (cmd/lifecycle_closeout.go) already auto-attaches the SAME evidence downstream, from the same attempt, whenever `details.WorkOutcome.Valid()` — a branch that existed but was dead code, since no production caller had ever supplied a real `WorkOutcome` before this plan. Combining both without a guard would render every knowledge delta twice on the finished screen.
- **Fix:** Added `excludeLifecycleEvidenceByID`, filtering the downstream auto-derived list by any evidence ID already present in `details.Evidence` before combining.
- **Files modified:** `cmd/lifecycle_closeout.go`
- **Commit:** `3862e737`

Or otherwise: no further deviations — the plan's Tasks 1-3 executed as written, including the file-split, the carve-out precedence for no-change/timeout, and the AST cobra-RunE-closure lookup needed because `buildCmd`'s direct-dispatch entry point is an anonymous `RunE` closure invisible to the existing named-`FuncDecl` call-graph indexer.

## Issues Encountered

None beyond the dedup fix documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

D-03, D-05 (build-lane rendering), D-07, D-08's rendering, and CAP-066 are closed for the build lanes: a real `aether build` on either the direct or wrapper lane now shows the honest verdict, the recommended next action, the credited/uncredited file card, and the recorded knowledge deltas. Per `201-VERIFICATION.md`'s root-cause analysis, the SAME closeout mechanism also covers D-06 (the `aether run` autopilot lane's cost/time block) and D-11 (failed-repair handback) — those call sites are outside this plan's declared scope (`cmd/work_closeout.go`, `cmd/codex_workflow_cmds.go`, `cmd/ceremony_cmd.go`, `cmd/work_closeout_test.go`) and remain tracked as `201-20` per STATE.md, since the check/continue lane and autopilot lane are separate call sites this plan did not touch.

---
*Phase: 201-queen-led-work-cycle*
*Completed: 2026-09-11*

## Self-Check: PASSED

All claimed files exist (`cmd/work_closeout.go`, `cmd/work_closeout_test.go`, `cmd/lifecycle_closeout.go`, `cmd/codex_workflow_cmds.go`, `cmd/ceremony_cmd.go`, this SUMMARY.md) and all three task commit hashes (`3862e737`, `dce8ee20`, `1282f163`) are present in `git log --oneline --all`.
