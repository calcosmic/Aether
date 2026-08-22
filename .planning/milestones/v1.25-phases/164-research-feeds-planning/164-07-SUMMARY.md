---
phase: 164-research-feeds-planning
plan: 07
subsystem: cmd (Go runtime — plan-time research injection and failure surfacing)
tags: [research, route-setter, planning-brief, budget-guard, fail-loudly]

requires:
  - phase: 164-05
    provides: "The Queen's research decision wired to dispatch: aether plan-research-approve, the four research_* manifest/result fields, and Scout dispatch gated on an approved map[int]bool"
provides:
  - "cmd/codex_plan.go: renderRouteSetterResearchContent + routeSetterResearchBudgetChars — appends each candidate phase's research excerpt into the Route-Setter's brief under a per-phase heading, bounded by a 12000-char aggregate budget, additive to the existing pointer sentence"
  - "cmd/codex_plan.go: writePhaseResearchArtifacts now returns failed phase IDs and writes a '**Research status:**' line into the fallback template for any approved-and-dispatched phase whose worker produced nothing"
  - "cmd/codex_plan_finalize.go: runCodexPlanFinalize's result carries research_failed_phases ([]int) and research_warning (string), computed via the new renderResearchFailedWarning helper — never blocks or errors the finalize"
affects: [164-09]

tech-stack:
  added: []
  patterns:
    - "Aggregate budget separate from per-phase budget: routeSetterResearchBudgetChars (12000) bounds the sum across every candidate phase in one planning run, layered on top of the existing per-phase phaseResearchBriefBudgetChars (3500) resolvePhaseResearchSection already enforces"
    - "Failure detection derived from the dispatch manifest, not a new stored flag: a phase counts as 'failed research' only when a phase_research Scout dispatch exists for it (same stage==phaseResearchStage && caste==scout filter validatePlanningWorkerChain already uses) AND it still falls through to the fallback-template branch — a phase never dispatched for research is never flagged"
    - "Failure is named in two independent places for durability (D-08): the artifact itself (**Research status:** line, template marker kept intact so it stays replaceable) and the finalize result (research_failed_phases + research_warning) — neither one alone would survive a lost completion packet"

key-files:
  created: []
  modified:
    - cmd/codex_plan.go
    - cmd/codex_plan_finalize.go
    - cmd/phase_research_dispatch_test.go
    - cmd/phase_research_preserve_test.go

key-decisions:
  - "renderRouteSetterResearchContent iterates researchCandidates (every candidate phase with a research file on disk), not researchDispatches (only phases dispatched this iteration) — a phase's research written in a prior iteration or a prior replan still surfaces to the Route-Setter even if this iteration's approval gate skipped re-dispatching it"
  - "Once the aggregate budget is exceeded, all remaining candidate phases (in ascending order) go straight to the closing line without re-checking whether a smaller one might still fit — predictable behaviour over marginal budget optimization, and it keeps the closing line's phase list monotonic with candidate order"
  - "The pointer sentence stays exactly as it was and is written first; content injection is purely additive after it, so iteration 1 (before any research Scout has run) renders byte-identical output to the pre-Plan-07 code"

requirements-completed: [RESEARCH-03]

duration: ~50min
completed: 2026-08-02
---

# Phase 164 Plan 07: Research Feeds the Route-Setter, Failures Stay Loud Summary

**The Route-Setter's brief now carries each phase's actual research excerpt (bounded by a 12000-char shared budget) instead of a bare pointer sentence, and a research worker that wrote nothing is named in both the fallback artifact and the finalize result rather than silently absorbed by the template.**

## Performance

- **Duration:** ~50 min
- **Tasks:** 2 completed
- **Files modified:** 4

## Accomplishments

- `renderRouteSetterResearchContent` appends every candidate phase's `resolvePhaseResearchSection` excerpt into the Route-Setter's brief, each under a `### Phase N: Name` heading, in ascending phase order, bounded by the new `routeSetterResearchBudgetChars = 12000` aggregate ceiling; phases that didn't fit are named in a closing line rather than silently dropped
- Iteration 1 (before any research Scout has written anything) renders the exact same pointer-only brief as before — content injection is additive, verified by a dedicated test asserting `HasSuffix` on the unchanged pointer sentence
- `writePhaseResearchArtifacts` now returns the phase IDs whose approved-and-dispatched research worker produced nothing, and writes a `**Research status:** phase N planned WITHOUT its research — worker failed` line directly into that phase's fallback template, right after the existing `**Research scope:**` line
- `runCodexPlanFinalize`'s result carries `research_failed_phases` and `research_warning` — the plan still finalizes with a nil error and all phases written when a research worker failed; research remains enrichment, not a gate (Phase 160 classification, D-08)
- A phase never dispatched for research (user skipped it via the Plan 05 approval flow) is correctly excluded from `research_failed_phases` — only a phase that was actually sent a Scout and got nothing back counts as failed

## Task Commits

Each task was committed atomically:

1. **Task 1: Route-Setter brief carries research content, not a pointer** - `46207d20` (feat)
2. **Task 2: A failed research worker is named, and planning proceeds** - `d942cdaa` (feat)

## Files Created/Modified

- `cmd/codex_plan.go` - `routeSetterResearchBudgetChars` constant, `renderRouteSetterResearchContent` (per-phase heading + budget accumulation + over-budget closing line), the Route-Setter brief-append block extended to call it, `writePhaseResearchArtifacts` widened to `([]string, int, []int, error)` with the failed-phase-ID collection and `**Research status:**` line, `renderResearchFailedWarning` helper
- `cmd/codex_plan_finalize.go` - the `writePhaseResearchArtifacts` call site updated for the new 4-value signature, `research_failed_phases` and `research_warning` added to `runCodexPlanFinalize`'s result map
- `cmd/phase_research_dispatch_test.go` - `TestRouteSetterBriefIncludesResearchContent` (3 sub-behaviours: no-research byte-identical, content+pointer both present with only-route_setter-modified check, three-phase excerpts each headed by number), `TestRouteSetterBriefResearchIsBounded`, `TestResearchWorkerFailureWarnsLoudlyWithoutBlocking` (failed/preserved/skipped three-phase fixture through the real `runCodexPlanFinalize` path)
- `cmd/phase_research_preserve_test.go` - `writePhaseResearchArtifacts` call site updated for the new signature

## Decisions Made

See `key-decisions` in frontmatter — summarized: content injection reads from all candidates (not just this iteration's dispatches) so previously-written research keeps surfacing; over-budget phases are cut off monotonically rather than backfilled; the pointer sentence's exact byte content is preserved as a hard backward-compatibility contract for iteration 1.

## Deviations from Plan

None — plan executed exactly as written. Two test-construction details were necessary but not explicitly spelled out in the plan's action text, both within Rule 3 (blocking test-setup issues caused directly by this plan's own fixtures):

**1. [Rule 3 - Blocking] `--approve-all` accepts the Queen's recommendation, not a blanket approval**
- **Found during:** Task 1 (writing `TestRouteSetterBriefIncludesResearchContent`'s three-phase sub-test)
- **Issue:** Generic fixture phase names/descriptions carry no external-tech signal, so `computePhaseResearchProposal` recommends "skip" for all three, and `plan-research-approve --approve-all` accepts that recommendation — meaning no research dispatches were created and the Route-Setter brief never got the pointer sentence at all
- **Fix:** Used `plan-research-approve --flip 1,2,3` to force all three phases into research regardless of the default recommendation
- **Files modified:** cmd/phase_research_dispatch_test.go (test only, no production code change)
- **Commit:** 46207d20 (Task 1 commit)

**2. [Rule 3 - Blocking] Research dispatches are gated on approval (Plan 05); an unapproved batch renders no pointer sentence at all**
- **Found during:** Task 1 (writing the "no research on disk" sub-test)
- **Issue:** The Route-Setter brief only gets the "Phase Research Available" pointer sentence when `len(researchDispatches) > 0` — with no prior approval, the whole append block never runs, so a naive test asserting "byte-identical to the pointer-only brief" would actually be asserting against a brief with no research section at all, not the actual pointer-only state Task 1's behaviour describes
- **Fix:** Each sub-test now runs `plan --plan-only --refresh` once, calls `plan-research-approve` to resolve the batch, then runs `plan --plan-only --refresh` again to reach the dispatched-but-not-yet-researched state the behaviour actually describes
- **Files modified:** cmd/phase_research_dispatch_test.go (test only)
- **Commit:** 46207d20 (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 3, test fixture construction only — no production code changes)
**Impact on plan:** Both fixes were necessary to make the tests actually exercise the behaviours described; no scope creep into production code.

## Issues Encountered

None beyond the test-fixture adjustments documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `routeSetterResearchBudgetChars`, `renderRouteSetterResearchContent`, and the widened `writePhaseResearchArtifacts` signature are live and tested; `go test ./cmd/... -count=1`, `go build ./cmd/aether`, and `go vet ./cmd/...` all pass
- RESEARCH-03 is now fully satisfied: research findings are injected into both the build brief (pre-existing) and the planner's brief (this plan), not pasted by hand
- No blockers for Plan 09 (surfacing the proposal card and warnings to the user-facing wrapper)

---
*Phase: 164-research-feeds-planning*
*Completed: 2026-08-02*

## Self-Check: PASSED

- FOUND: .planning/phases/164-research-feeds-planning/164-07-SUMMARY.md
- FOUND commit 46207d20: feat(164-07): Route-Setter brief carries research content, not a pointer
- FOUND commit d942cdaa: feat(164-07): name a failed research worker instead of papering over it
