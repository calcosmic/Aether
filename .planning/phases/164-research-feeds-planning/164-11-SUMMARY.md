---
phase: 164-research-feeds-planning
plan: 11
subsystem: infra
tags: [go, typescript, worker-dispatch, resilience, gap-closure]

# Dependency graph
requires:
  - phase: 164-research-feeds-planning (plans 01-09)
    provides: phase-domain research dispatch machinery (renderPhaseResearchBrief, plannedPhaseResearchDispatches, phaseResearchCandidates, confidence loop)
  - phase: 164-research-feeds-planning (plan 10)
    provides: field-faithful toWorkerDispatches (permission_profile/brief pass-through) that this plan's escalation dispatches also rely on
provides:
  - plan-research-escalate resolving phases from the live planning iteration state (planning/iteration-state.json) during a fresh colony's mid-loop planning, with a preserved zero-seed fallback for the refresh/replan path
  - A non-fatal escalation round in runResearchConfidenceLoop: any plan-research-escalate failure (phase resolution, subprocess error, malformed envelope, or a failed escalation dispatch wave) degrades to a named ceremony warning and lets the plan run finish
  - Rebuilt .aether/ts-host/dist/host.js carrying the degrade fix, proven against the real compiled artifact and a real non-zero Go exit
affects: [165-core-lifecycle-commands, any future phase touching the plan-time research confidence loop or plan-research-escalate]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Fresh-colony CLI test fixture: seed planning/iteration-state.json directly via store.SaveJSON with an EMPTY colony Plan.Phases, rather than the previously-only-tested pre-populated-plan shape, so the test actually exercises the scenario the bug lived in"
    - "Real-binary degrade test: point GoBridgeOptions.goBinaryPath at AETHER_BINARY_PATH and cwd at a colony-less temp dir, call __restoreCallGoJSON(), and let execFileSync genuinely throw -- proves the try/catch against a real non-zero exit instead of a hand-mocked rejection"

key-files:
  created:
    - .aether/ts-host/test/research-escalation-degrade.test.ts
  modified:
    - cmd/phase_research_escalate.go
    - cmd/phase_research_escalate_test.go
    - .aether/ts-host/src/host.ts
    - .aether/ts-host/dist/host.js

key-decisions:
  - "planResearchEscalateCmd seeds phaseResearchCandidates from loadPlanningIterationState() when available, falling back to the zero-value seed (which reads state.Plan.Phases) only when no iteration state exists -- preserves the refresh/replan path exactly as before"
  - "The escalation round wraps two independent failure points separately: the per-candidate plan-research-escalate call (continue to the next candidate on failure) and the final escalation-dispatch wave (guarded by escalationDispatches.length > 0, wrapped in its own try/catch) -- so a provider-side dispatch failure degrades the same way a Go subcommand failure does"
  - "A phase id is only pushed onto the escalations array after its dispatch is successfully obtained, so ResearchLoopSummary.escalations reports what actually escalated, not what was attempted"
  - "The degrade test's real-binary cases duplicate research-confidence-loop.test.ts's private fixture helpers (makeFixtureRepo, installDispatchMock, etc.) locally rather than importing them, since the originals are unexported -- kept intentionally narrow instead of exporting test-only surface across files"

requirements-completed: [RESEARCH-01, RESEARCH-02, RESEARCH-07]

# Metrics
duration: 25min
completed: 2026-08-02
---

# Phase 164 Plan 11: Fresh-Colony Escalate Resolution + Non-Fatal Escalation Degrade (CR-03 Gap Closure) Summary

**Fixed `plan-research-escalate` resolving phases from an always-empty `Plan.Phases` during a fresh colony's mid-loop planning, and wrapped the TS host's escalation call so any future failure warns and continues instead of crashing the whole plan run.**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-08-02T18:03:00+02:00 (approx, first RED commit)
- **Completed:** 2026-08-02T18:12:01+02:00
- **Tasks:** 2 (both TDD RED/GREEN)
- **Files modified:** 5 (1 created, 4 modified)

## Accomplishments

- `planResearchEscalateCmd` now calls `loadPlanningIterationState()` and seeds `phaseResearchCandidates` from the loaded planning draft when available -- resolving the exact scenario CR-03 documented: a fresh colony's mid-loop planning, where `COLONY_STATE.json`'s `Plan.Phases` is empty and the in-progress phase draft lives only in `planning/iteration-state.json`
- The zero-value seed fallback is preserved when no iteration state exists, so the refresh/replan path (where the active colony plan genuinely is the source) is unchanged
- Three new Go CLI subtests added to `TestOracleEscalationDispatchNamesTheStall`, starting from the real fresh-colony shape (empty `Plan.Phases`) rather than a pre-populated plan: `fresh_colony_mid_loop_resolves_phase_from_iteration_state`, `fresh_colony_unknown_phase_still_returns_clean_error`, `no_iteration_state_falls_back_to_colony_plan`
- `runResearchConfidenceLoop`'s escalation round in `.aether/ts-host/src/host.ts` now wraps the per-candidate `plan-research-escalate` call and the final escalation-dispatch wave each in their own try/catch, emitting a named ceremony warning ("Oracle escalation unavailable for phase N: ... Planning continues -- research is enrichment, never a gate (D-08)") and continuing rather than propagating
- A phase id is only recorded in `ResearchLoopSummary.escalations` after its dispatch is successfully obtained
- New `.aether/ts-host/test/research-escalation-degrade.test.ts` drives the REAL `aether plan-research-escalate` binary (via `AETHER_BINARY_PATH`, `__restoreCallGoJSON()`, and a colony-less temp `cwd`) into a genuine non-zero exit, proving the degrade against the actual failure class rather than a mocked rejection
- Rebuilt `.aether/ts-host/dist/host.js` (the artifact the runtime executes) and reproduced the plan's dist probe verbatim: pre-fix it crashed the node process with "failed to load colony state: no colony initialized"; post-fix it resolves with `stopReason: "diminishing_returns"` and an empty `escalations` array

## Task Commits

Each task followed TDD's RED/GREEN discipline:

1. **Task 1 RED: fresh-colony escalate CLI test** - `080b6654` (test) -- `fresh_colony_mid_loop_resolves_phase_from_iteration_state` failed with empty stdout (phase 2 not found against the zero-seeded empty `Plan.Phases`)
2. **Task 1 GREEN: seed from planning iteration state** - `11cf9801` (feat)
3. **Task 2 RED: real-binary escalation-degrade test** - `a9f49543` (test) -- both new real-binary cases threw an unhandled `Error: Go command failed: plan-research-escalate --phase N: failed to load colony state: no colony initialized`, crashing the test exactly as CR-03 described
4. **Task 2 GREEN: try/catch degrade + dist rebuild** - `1b075d53` (fix)

## Files Created/Modified

- `cmd/phase_research_escalate.go` - `planResearchEscalateCmd`'s RunE now seeds `phaseResearchCandidates` from `loadPlanningIterationState()` when available, falling back to the zero value; comment rewritten to explain why the zero seed was wrong
- `cmd/phase_research_escalate_test.go` - Three new subtests on `TestOracleEscalationDispatchNamesTheStall` plus a `writeFreshColonyIterationState` helper that seeds `planning/iteration-state.json` with an empty colony `Plan.Phases`, matching the real fresh-colony shape
- `.aether/ts-host/src/host.ts` - `runResearchConfidenceLoop`'s escalation round wraps the per-candidate `_callGoJSONRef` call and the final `_dispatchWorkersRef` escalation wave in try/catch; comment records the CR-03 crash history and D-08 rationale
- `.aether/ts-host/test/research-escalation-degrade.test.ts` (new) - Three cases: real-binary single-phase failure, real-binary two-phase failure (neither suppresses the other), and a mocked happy-path control proving the degrade did not disable successful escalation
- `.aether/ts-host/dist/host.js` - Rebuilt compiled artifact carrying the degrade fix

## Decisions Made

- Kept the zero-value-seed fallback rather than always requiring iteration state, because the refresh/replan path's existing passing test (`no_iteration_state_falls_back_to_colony_plan`, formerly `cli_prints_dispatch_and_exits_zero`) depends on `state.Plan.Phases` remaining the source when no iteration state exists.
- Chose to guard the escalation-dispatch wave with `escalationDispatches.length > 0` before calling `_dispatchWorkersRef`, so a run where every escalation candidate already failed its Go call does not attempt an empty dispatch.
- Duplicated (rather than exported) `research-confidence-loop.test.ts`'s private fixture helpers into the new test file, since exporting test-only surface across production test files was judged higher cost than a small amount of local duplication.

## Deviations from Plan

None — plan executed exactly as written. `npm install` was required in `.aether/ts-host` before any test could run (this worktree also had no `node_modules/`, matching plan 164-10's noted first-run worktree setup, not a plan deviation).

## Issues Encountered

None beyond the expected first-run `npm install`.

## TDD Gate Compliance

- Task 1 (tdd="true"): RED commit `080b6654` (test, confirmed failing -- `fresh_colony_mid_loop_resolves_phase_from_iteration_state` produced empty stdout because phase 2 was not found against an empty `Plan.Phases`) followed by GREEN commit `11cf9801` (feat, confirmed all subtests passing, `go test ./cmd/...` clean at 287s).
- Task 2 (tdd="true"): RED commit `a9f49543` (test, confirmed failing -- both real-binary cases threw the unhandled CR-03 crash verbatim) followed by GREEN commit `1b075d53` (fix, confirmed 3/3 new tests passing, the pre-existing 13-test `research-confidence-loop.test.ts` suite still green, the dist probe passing against the rebuilt compiled artifact, and the full `npm test` suite at 555/555 passing). No REFACTOR commit needed for either task.

## Dist Escalation-Degrade Probe

Command (from the plan's `<verify><automated>` block, run against the rebuilt `.aether/ts-host/dist/host.js`):

```
node --input-type=module -e '... host.runResearchConfidenceLoop(...) ...'
```

Output:

```
Oracle escalation: phase 1 research stalled at 30% against a 95% target — escalating Scout to Oracle
Warning: Oracle escalation unavailable for phase 1: Go command failed: plan-research-escalate --phase 1: failed to load colony state: no colony initialized. Planning continues -- research is enrichment, never a gate (D-08).
dist escalation-degrade probe OK: {"phases":[{"phaseId":1,"iterations":3,"finalConfidence":30,"stopReason":"diminishing_returns"}],"escalations":[]}
```

`stopReason: "diminishing_returns"` proves the escalation path was genuinely entered (not vacuous); `escalations: []` proves the failed escalation was not recorded as successful.

## Known Stubs

None.

## Threat Flags

None — this plan's threat model (T-164-05 through T-164-07) was fully addressed as designed: both escalation failure paths (the per-candidate call and the dispatch wave) now degrade instead of propagating, the `previous_plan_draft` read is colony-owned scratch already trusted by the plan-only path, and the escalated Oracle dispatch's permission profile resolution is unchanged (`attachPlanningDispatchSkillAssignments` / `PermissionProfileForCaste("oracle")` still apply identically).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- CR-03 from `164-VERIFICATION.md` is closed: `plan-research-escalate` resolves phases from the in-progress planning draft during a fresh colony's mid-loop planning and exits zero, and a real non-zero escalate exit now degrades to a warning instead of crashing the plan run.
- RESEARCH-01, RESEARCH-02, and RESEARCH-07 are now fully satisfiable and marked Complete in REQUIREMENTS.md by the orchestrator (this plan closes the last of the three defects; RESEARCH-04/05 were already closed by plan 164-10).
- No blockers for Phase 165: this plan's fixes are self-contained and additive, touching only `cmd/phase_research_escalate.go`, its test file, and the TS host's escalation round.

---
*Phase: 164-research-feeds-planning*
*Completed: 2026-08-02*
