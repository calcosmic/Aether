---
phase: 198-put-the-thrown-away-data-back-on-screen
plan: 08
subsystem: cli-visuals
tags: [go, resume, dashboard, plan-revision, SHOW-02]

# Dependency graph
requires:
  - phase: 198-06
    provides: renderContinueWorkerFlowValue dual-type rendering precedent (typed struct vs JSON-round-tripped map/slice), continueDetailCap "(+N more)" honest-overflow pattern
provides:
  - "buildResumeDashboardResult's result[\"phase_progress\"] -- a per-phase (id, name, status) list built straight from the plan's own recorded phase status, alongside the pre-existing overall fraction"
  - "renderResumePhaseProgress (cmd/codex_visuals.go) -- one line per phase in plain English (finished/in progress/not started), capped at 8 with an honest (+N more)"
  - "renderResumeRecentDecisions -- up to five recorded decisions, most recent first, each with its reason on a nested detail line, reading the resume dashboard's pre-existing result[\"recent\"][\"decisions\"]"
  - "renderResumeDriftNote -- one plain-English sentence on whether the plan has been revised since it was written, derived only from planRevisionSummary's own output (Phase 196 D-01 applied to a non-numeric drift signal): a real revision names the reason and how many phases it replaced, the legacy-import fallback says the plan has not been revised, and no plan says nothing has been recorded"
affects: [any-future-phase-touching-cmd/context.go-buildResumeDashboardResult-or-cmd/codex_visuals.go-renderResumeVisual]

# Actuals (#2632)
actuals:
  tokens: 6462
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Dual-type detail rendering applied to a genuinely new field: renderResumePhaseProgress type-switches between the in-process []resumePhaseProgressEntry and the JSON-round-tripped []interface{} shape (renderContinueWorkerFlowValue precedent, 198-PATTERNS.md), even though the resume dashboard never actually round-trips through JSON in production today (renderResumeVisual is always called with the live map) -- future-proofing for the house convention rather than a currently-exercised path."
    - "A signal the runtime cannot stand behind is never fabricated (Phase 196 D-01 extended to a non-numeric case): renderResumeDriftNote reads only result[\"plan_revision\"] (planRevisionSummary's own output) and computes no time-since or count-of-changes figure of its own -- it counts the length of an already-provided superseded_phase_ids slice, never a derived time delta. Locked by a go/ast assertion that the function body contains no time.* call."
    - "Honest overflow capping (198-06's continueDetailCap precedent) applied to phase progress: resumePhaseProgressCap caps the per-phase list at 8 named lines with an arithmetic '(+N more)' remainder, never a silent truncation."

key-files:
  created:
    - cmd/resume_detail_test.go
  modified:
    - cmd/context.go
    - cmd/codex_visuals.go

key-decisions:
  - "The drift note is derived from the plan's own recorded revision (planRevisionSummary), not a new drift-detection algorithm -- no such signal exists anywhere in the codebase and inventing one (e.g. a time-since-last-revision figure) would be a number the runtime cannot stand behind (per plan's own Claude's-discretion framing and CLAUDE.md's Definition of Done)."
  - "'How many phases it replaced' in the drift note is the count of the active revision's own SupersededPhaseIDs -- the old phases the revision tossed out and replaced -- not ReplacementPhaseIDs (the new phases that came in). This reads naturally as 'the plan was revised, and N old phases were replaced' rather than counting the new phase IDs the revision produced."
  - "Recent decisions are capped at the five extractRecentDecisions already returns; renderResumeRecentDecisions neither re-slices nor re-orders, matching the plan's explicit instruction and avoiding a second cap that could disagree with the first."
  - "resumePhaseStatusDisplay falls back to the raw status string for any status this repo has not named (rather than a generic 'unknown'), so a genuinely new phase-status constant added later is visible verbatim instead of silently swallowed -- while the three known constants (pending/in_progress/completed) always translate to plain English."
  - "TestResumeDetailIsPlainEnglish's internal-name check is scoped to multi-word snake_case constants (containing '_') -- 'in_progress', 'legacy_import', 'user_feedback', 'verification_failure', 'scope_change' -- and exempts single-word constants ('completed', 'manual', 'research', 'initial') because they are also ordinary English words that legitimately appear in plain-English prose. This mirrors 198-06's TestRestoredDetailIsPlainEnglish scoping its own internal-name scan to gate keys and not verification check keys, for the same reason."
  - "SHOW-02 is a requirement shared by six plans in this phase (198-04, 05, 06, 07, 08, 09). This plan's PLAN.md frontmatter still declares requirements-completed: [SHOW-02] per the summary template's instruction to copy the field verbatim, but REQUIREMENTS.md itself was NOT marked complete: 198-09 (a sibling plan in the same wave, running in a separate worktree) has no SUMMARY.md yet in this worktree's view of the phase directory, so the shared-ID gate (#2388) is not yet satisfied. Left REQUIREMENTS.md untouched (still 'Pending' for SHOW-02) for the orchestrator or a later plan's update_requirements step to mark once 198-09 finishes -- this matches the behavior already visible in REQUIREMENTS.md before this plan ran (198-04 through 198-07 also declared SHOW-02 and also correctly left it unmarked)."

requirements-completed: [SHOW-02]

coverage:
  - id: D1
    description: "The resume view shows progress phase by phase, not only an overall fraction."
    requirement: "SHOW-02"
    verification:
      - kind: unit
        ref: "cmd/resume_detail_test.go#TestResumeShowsProgressPhaseByPhase"
        status: pass
    human_judgment: false
  - id: D2
    description: "The resume view shows the most recent decisions that were recorded, with what was decided and why."
    requirement: "SHOW-02"
    verification:
      - kind: unit
        ref: "cmd/resume_detail_test.go#TestResumeShowsRecentDecisions"
        status: pass
    human_judgment: false
  - id: D3
    description: "The resume view shows a drift note describing whether the plan has been revised since it was written, derived from a real recorded revision -- and says so plainly, never inventing a signal, when no revision has ever been recorded."
    requirement: "SHOW-02"
    verification:
      - kind: unit
        ref: "cmd/resume_detail_test.go#TestDriftNoteIsDerivedNotInvented"
        status: pass
    human_judgment: false
  - id: D4
    description: "The richer resume view is still provably read-only and reads in ordinary words, not internal status codes."
    verification:
      - kind: unit
        ref: "cmd/resume_detail_test.go#TestResumeViewDoesNotMutate"
        status: pass
      - kind: unit
        ref: "cmd/resume_detail_test.go#TestResumeDetailIsPlainEnglish"
        status: pass
      - kind: unit
        ref: "cmd/display_house_style_test.go#TestHumanDisplaysUseHeadedSectionsNotMachineTables"
        status: pass
    human_judgment: false

duration: ~15min
completed: 2026-08-29
status: complete
---

# Phase 198 Plan 8: Show Resume Progress, Recent Decisions, and a Drift Note Summary

**The resume dashboard now shows per-phase progress in plain English, up to five recent decisions with their reasons, and an honest plan-revision drift note -- three things `buildResumeDashboardResult` already computed or was extended to compute, that `renderResumeVisual` had never printed.**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-08-29T20:36:21+02:00 (previous phase-tracking commit)
- **Completed:** 2026-08-29T20:51:22+02:00
- **Tasks:** 3 completed
- **Files modified:** 3 (1 created, 2 modified)

## Accomplishments

- `buildResumeDashboardResult` (`cmd/context.go`) now builds `result["phase_progress"]` -- a `[]resumePhaseProgressEntry{Phase, Name, Status}` list derived by iterating `state.Plan.Phases`, carrying each phase's own recorded status constant unchanged. The pre-existing overall fraction (`Phase: N/M`) is untouched.
- `renderResumePhaseProgress` (`cmd/codex_visuals.go`) renders one line per phase ("Phase 2 — Core Features: in progress"), translating the internal `pending`/`in_progress`/`completed` constants to "not started"/"in progress"/"finished", capped at 8 with an honest `(+N more)` remainder. A colony with no plan renders nothing and does not error.
- `renderResumeRecentDecisions` renders up to five recorded decisions (already capped and ordered most-recent-first by the pre-existing `extractRecentDecisions`), each naming the claim with its reason on a nested `└──` detail line. A colony with no recorded decisions renders nothing.
- `renderResumeDriftNote` renders one plain-English sentence on whether the plan has been revised since it was written, reading only `result["plan_revision"]` (the pre-existing `planRevisionSummary` output): a real recorded revision names its reason and how many phases it replaced; the legacy-import fallback says the plan has not been revised since it was written; a colony with no plan at all says plainly that nothing has been recorded. A go/ast assertion (`TestDriftNoteIsDerivedNotInvented`) proves the function contains no `time.*` call, so it can never compute a fabricated time-since figure.
- All three renderers are wired into `renderResumeVisual`, reached by both the `resume-dashboard` command (fully read-only) and `resume-colony`/`resume` (the fuller recovery view), from a single source of truth.
- `TestResumeViewDoesNotMutate` snapshots the entire saved project data tree before and after two `resume-dashboard` runs and asserts byte-identical contents, extending the same CLAUDE.md "an inspection command must not mutate state" proof this repo has needed twice before (the `--dry-run` consolidation bugs).
- `TestResumeDetailIsPlainEnglish` proves none of the three new blocks leak an internal multi-word status or revision-reason constant (`in_progress`, `legacy_import`, `user_feedback`, `verification_failure`, `scope_change`) verbatim, deriving the checked name list from the actual `colony` package constants so the check cannot go stale as new ones are added.

## Task Commits

Each task was committed atomically:

1. **Task 1: Progress, phase by phase** - `d321cea1` (feat)
2. **Task 2: Recent decisions and the drift note** - `b093371a` (feat)
3. **Task 3: The resume view still writes nothing and still reads as plain English** - `2123c0cf` (test)

## Files Created/Modified

- `cmd/context.go` - Added `resumePhaseProgressEntry` struct and `buildResumePhaseProgress` helper; wired `result["phase_progress"]` into both branches of `buildResumeDashboardResult` (the no-COLONY_STATE.json default branch and the main branch).
- `cmd/codex_visuals.go` - Added `resumePhaseProgressCap`, `resumePhaseStatusDisplay`, `renderResumePhaseProgress`, `renderResumeRecentDecisions`, `intSliceLen`, `renderResumeDriftNote`; wired all three render calls into `renderResumeVisual` immediately after the existing "Next phase" block.
- `cmd/resume_detail_test.go` (new) - All five named tests: `TestResumeShowsProgressPhaseByPhase`, `TestResumeShowsRecentDecisions`, `TestDriftNoteIsDerivedNotInvented`, `TestResumeViewDoesNotMutate`, `TestResumeDetailIsPlainEnglish`.

## Decisions Made

See `key-decisions` in frontmatter: drift note derived from `planRevisionSummary`, not a new signal; "phases replaced" counts `SupersededPhaseIDs`; recent decisions neither re-sliced nor re-ordered; unknown phase statuses fall back to the raw value rather than a generic placeholder; the plain-English check is scoped to multi-word snake_case constants only; and SHOW-02 left unmarked in REQUIREMENTS.md pending sibling plan 198-09.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. One implementation note: `renderResumePhaseProgress`'s dual-type switch (typed `[]resumePhaseProgressEntry` vs. JSON-round-tripped `[]interface{}`) is not currently exercised by any production code path -- `renderResumeVisual` is always called directly with the in-process result map, never after a JSON round-trip (unlike `continue`'s ceremony-closeout path). Implemented anyway per the plan's explicit "type-switch over the typed and round-tripped shapes as the house pattern requires" instruction and covered by a round-trip equality test, so the renderer is already correct if a future resume closeout path is added.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `buildResumeDashboardResult`/`renderResumeVisual` now surface all three items this plan targeted; the resume dashboard's remaining carried-but-unrendered fields (if any) are out of this plan's scope.
- SHOW-02 remains `Pending` in `.planning/REQUIREMENTS.md` -- six plans in this phase (198-04, 05, 06, 07, 08, 09) declare it, and 198-09 (a sibling plan in the same wave 4, running in its own worktree) has no SUMMARY.md yet from this worktree's vantage point. The orchestrator or a later plan's `update_requirements` step should re-run `requirements.ready-ids`/`requirements.mark-complete` once 198-09 completes and both worktrees are merged.
- No blockers for subsequent plans in phase 198.

## Self-Check: PASSED

- Verified `cmd/context.go`, `cmd/codex_visuals.go`, `cmd/resume_detail_test.go` all exist on disk with the expected content.
- Verified commits `d321cea1`, `b093371a`, `2123c0cf` exist in `git log`.
- Re-ran the plan's full `<verification>` command: `go build ./...` and `go vet ./cmd` exit 0; `go test ./cmd -run 'TestResumeShowsProgressPhaseByPhase|TestResumeShowsRecentDecisions|TestDriftNoteIsDerivedNotInvented|TestResumeViewDoesNotMutate|TestResumeDetailIsPlainEnglish|TestResumeDashboard|TestHumanDisplaysUseHeadedSectionsNotMachineTables' -count=1` -- all pass.
- Ran the full `go test ./cmd/...` suite once at the end after all three commits: green, 230s, zero failures.
- Confirmed `.planning/REQUIREMENTS.md` line 121 (`SHOW-02 | Phase 198 | Pending`) is unchanged by this plan, consistent with the shared-ID gate not yet being satisfied.

---
*Phase: 198-put-the-thrown-away-data-back-on-screen*
*Completed: 2026-08-29*
