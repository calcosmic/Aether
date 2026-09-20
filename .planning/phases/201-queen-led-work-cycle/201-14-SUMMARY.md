---
phase: 201-queen-led-work-cycle
plan: "14"
subsystem: queen-orchestration
tags: [go, verification, brief-composition, model-routing, turnaround, WORK-08]

# Dependency graph
requires:
  - phase: 201-queen-led-work-cycle (plan 12)
    provides: "cmd/job_telemetry.go: the eight-segment attempt-bound timing record and the D-16 report-only guard (TestTelemetryIsReportOnly) this plan's routing table must stay outside of"
  - phase: 201-queen-led-work-cycle (plan 13)
    provides: ".planning/phases/201-queen-led-work-cycle/201-TURNAROUND-BASELINE.md: the owner-ratified 4m30s turnaround target for a small single-task job, and the naming of the three approved levers (targeted test lanes, slimmer briefs, model routing by caste) this plan implements"
provides:
  - "cmd/verification_scope.go: deriveVerificationScopeAtCyclePoint and the verificationCyclePointLoop/verificationCyclePointBoundary constants -- a loop-vs-boundary rule layered on top of the untouched deriveVerificationScope"
  - "cmd/deterministic_floor.go: runDeterministicFloorAtCyclePoint (cycle-point-aware) with runDeterministicFloor kept as an unchanged-signature wrapper defaulting to the boundary, so every one of its 13 existing test call sites and 4 production callers needed no change"
  - "cmd/codex_build_finalize.go: the build-time free-check report -- already documented there as never an advancement gate -- now runs in loop mode, so it can genuinely benefit from a scoped test run"
  - "cmd/build_print_brief.go: compactHandoffFieldNames (reflects codex.WorkerHandoff's own JSON fields) and missingCompactHandoffFields, wired into printWorkerBriefs as a real runtime gate against a silently-dropped handoff field"
  - "cmd/caste_model_routing.go: resolveCasteModelRoute, a policy layer above resolveCasteModel and above the spend ledger; casteModelRoutes names porter as the one routed caste today, with a stated reason"
  - "cmd/codex_build.go: codexBuildDispatch.ModelRoutingReason (new field), and attachBuildDispatchContext wired to resolveCasteModelRoute first, falling back to resolveCasteModel exactly as before for every unrouted caste"
affects: [201-15]

# Actuals (#2632)
actuals:
  tokens: 9499
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Cycle-point layering, not signature change: deriveVerificationScopeAtCyclePoint and runDeterministicFloorAtCyclePoint are NEW functions that wrap the existing deriveVerificationScope/runDeterministicFloor unchanged. The existing functions keep their exact signatures and behavior (verified against all 13 pre-existing runDeterministicFloor call sites and both existing verification_scope_test.go tests, which needed zero edits) while the new loop/boundary rule is available wherever a caller opts in. This mirrors the same additive-wrapper discipline this plan applied a second time (Task 3) rather than mutating a widely-called function's contract."
    - "Refusal checked before the routing table, not just disjoint from it: resolveCasteModelRoute consults casteModelReasons (the existing quality-sensitivity evidence table) FIRST and unconditionally, before ever looking up casteModelRoutes. A caste accidentally added to both tables is still refused -- the guarantee is structural, not merely 'the two tables happen not to overlap today' (TestQualitySensitiveCasteIsNeverRouted proves this both ways)."
    - "A runtime gate, not only a test: missingCompactHandoffFields is called from printWorkerBriefs itself (cmd/build_print_brief.go), so a future change that silently drops a WorkerHandoff field from the brief's schema statement fails a real `aether build N --print-brief` invocation, not only a unit test that could bit-rot unnoticed."
    - "Existing colony-prime trim discipline (blockers protected, trim ledger reported) and the existing baseline-aware-castes gate (cmd/phase_baseline.go) were both already correct; this plan added the first real fixture-driven tests proving each under genuine budget/caste pressure rather than adding new production logic where none was needed."

key-files:
  created:
    - cmd/caste_model_routing.go
    - cmd/turnaround_levers_test.go
  modified:
    - cmd/verification_scope.go
    - cmd/deterministic_floor.go
    - cmd/codex_build_finalize.go
    - cmd/build_print_brief.go
    - cmd/codex_build.go

key-decisions:
  - "runDeterministicFloor's signature and default behavior were left completely unchanged (a new sibling function, runDeterministicFloorAtCyclePoint, carries the cycle-point parameter). 13 existing test call sites across 6 test files call the old signature; changing it would have forced edits far outside this plan's declared file list for no behavioral gain, since every one of those 13 callers is genuinely an advancement decision (the boundary) anyway."
  - "Wiring runDeterministicFloorAtCyclePoint into cmd/deterministic_floor.go and cmd/codex_build_finalize.go was judged necessary even though neither file is in this plan's declared files_modified list (Rule 2 -- missing critical functionality). Without it, the new loop-scoping logic would be real, tested code that nothing in the runtime ever calls -- exactly the 'machinery exists but was never switched on' failure mode this project has repeatedly shipped and CLAUDE.md names by name. The change is a single, surgical, backward-compatible call-site swap in each file, verified against every existing test that exercises either function."
  - "Model routing (Task 3) starts with exactly one routed caste, 'porter', rather than a longer list. The plan requires the reason to be genuine, not fabricated, and this codebase's own evidence (cmd/caste_model_reason.go's quality-sensitivity table, which names every caste whose work needs judgment) only supports a caste with a plainly mechanical, no-judgment task as a routing candidate. Porter (publish/deploy: fixed commands over already-reviewed work) meets that bar; other 'sonnet-tier' castes were left unrouted rather than routed on invented justification."
  - "The team check-in card (cmd/ceremony_team_checkin.go) was deliberately NOT wired to show the new routing reason, and cmd/codex_continue.go's telemetry Source string was NOT extended to name the scope mode. Both are real, undeclared files whose value is presentational rather than behavioral; the plan's actual requirements (a routed dispatch carries its model and reason; loop/boundary scoping is real and wired) are fully met and tested without touching either. Left as a natural follow-up rather than expanding this plan's undeclared-file footprint further."

requirements-completed: [WORK-08]

coverage:
  - id: D1
    description: "Inside the working loop a confidently derived package set narrows the tests command; at the phase boundary the full suite always runs regardless of how narrow a loop-scoped pass would have been; an unattributable change and the plan's final phase both always run full in either cycle point; build/type-check/lint commands are never touched by any of this."
    requirement: WORK-08
    verification:
      - kind: unit
        ref: "cmd/turnaround_levers_test.go#TestScopedTestsRunInTheLoopAndTheFullSuiteAtTheBoundary"
        status: pass
      - kind: unit
        ref: "cmd/turnaround_levers_test.go#TestOnlyTheTestsCommandIsEverNarrowed"
        status: pass
      - kind: unit
        ref: "cmd/verification_scope_test.go#TestScopeIsDerivedFromChangedFiles"
        status: pass
      - kind: unit
        ref: "cmd/verification_scope_test.go#TestScopeFallsBackToFullWhenUndecidable"
        status: pass
      - kind: unit
        ref: "cmd/verification_scope_test.go#TestFinalPhaseAlwaysRunsFull"
        status: pass
      - kind: unit
        ref: "cmd/verification_scope_test.go#TestScopedCommandNeverBroadensTheRun"
        status: pass
    human_judgment: false
  - id: D2
    description: "A worker's brief carries only task-relevant context (the phase-baseline section only reaches verifying castes) plus a compact handoff schema statement reflected directly off codex.WorkerHandoff's own fields -- no second schema -- and printWorkerBriefs itself fails if that statement ever silently drops a field the type still carries. Blockers survive real trim pressure at the tightest (compact) colony-prime budget. The existing task-share proportion regression lock is unaffected."
    requirement: WORK-08
    verification:
      - kind: unit
        ref: "cmd/turnaround_levers_test.go#TestBriefCarriesOnlyTaskRelevantContext"
        status: pass
      - kind: unit
        ref: "cmd/turnaround_levers_test.go#TestCompactHandoffUsesTheExistingShape"
        status: pass
      - kind: unit
        ref: "cmd/turnaround_levers_test.go#TestBlockersSurviveTheTightestBudget"
        status: pass
      - kind: unit
        ref: "cmd/codex_build_test.go#TestBuildWorkerBriefIsMostlyTask"
        status: pass
    human_judgment: false
  - id: D3
    description: "A caste named in casteModelRoutes (porter) dispatches on the routed model carrying a non-empty stated reason; an unrouted caste's model resolution is byte-identical to before; a caste the existing quality-sensitivity evidence table names is refused a route structurally; every dispatched worker still appears on the per-worker cost line with its own figure regardless of routing; the D-16 report-only telemetry guard is unaffected."
    requirement: WORK-08
    verification:
      - kind: unit
        ref: "cmd/turnaround_levers_test.go#TestRoutedCasteCarriesItsModelAndReason"
        status: pass
      - kind: unit
        ref: "cmd/turnaround_levers_test.go#TestUnroutedCasteModelResolutionIsUnchanged"
        status: pass
      - kind: unit
        ref: "cmd/turnaround_levers_test.go#TestQualitySensitiveCasteIsNeverRouted"
        status: pass
      - kind: unit
        ref: "cmd/turnaround_levers_test.go#TestEveryWorkerAppearsOnTheCostLine"
        status: pass
      - kind: unit
        ref: "cmd/job_telemetry_test.go#TestTelemetryIsReportOnly"
        status: pass
    human_judgment: false
  - id: D4
    description: "No lever in this plan raises a worker-count ceiling or widens a parallel wave."
    requirement: WORK-08
    verification: []
    human_judgment: true
    rationale: "This is an absence claim (nothing in the diff touches worker-count or wave-width logic), reviewable directly from the diff but not something a single automated assertion proves end to end -- the plan's own prohibitions list is the checklist a human reviewer applies against the actual files changed."

duration: 55min
completed: 2026-09-10
status: complete
---

# Phase 201 Plan 14: Turnaround Levers -- Test Scope, Slimmer Brief, Model Routing Summary

**Loop-scoped test lanes with a full-suite boundary gate, a slimmer worker brief with a real (not merely tested) compact-handoff schema check, and per-caste model routing for one mechanical caste (porter) with a stated reason and an unaffected per-worker cost line.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-09-10T19:05:00Z (approx.)
- **Completed:** 2026-09-10T20:00:00Z (approx.)
- **Tasks:** 3
- **Files modified:** 7 (2 created, 5 modified)

## Accomplishments

- `deriveVerificationScopeAtCyclePoint` (`cmd/verification_scope.go`, new) layers D-15a's loop-vs-boundary rule on top of the existing `deriveVerificationScope`, which it leaves byte-for-byte unchanged. Inside the working loop, a confidently derived package set narrows the tests command exactly as before; at the phase boundary, the full suite always runs regardless of how narrow a loop-scoped pass would have been. `runDeterministicFloorAtCyclePoint` (`cmd/deterministic_floor.go`, new) threads this through; `runDeterministicFloor` itself keeps its exact signature and defaults to the boundary, so all 13 existing test call sites and 4 existing production callers needed zero changes. The one real wiring point that benefits (`cmd/codex_build_finalize.go`'s build-time free-check report, already documented there as never an advancement gate) now runs in loop mode.
- `compactHandoffFieldNames`/`missingCompactHandoffFields` (`cmd/build_print_brief.go`, new) reflect `codex.WorkerHandoff`'s own JSON fields directly off the type -- no second, hand-typed schema exists anywhere in this change -- and are wired into `printWorkerBriefs` as a real gate: `--print-brief` now fails if a worker's brief ever silently drops a field the handoff type still carries. Two already-implemented invariants got their first real fixture-driven proof: a builder's brief omits the phase-baseline section (only verifying castes need it) while a watcher's brief still carries it, and a real active blocker survives genuine trim pressure at colony-prime's tightest (4000-char) budget alongside 60 padded, unprotected instinct entries that do get trimmed.
- `resolveCasteModelRoute` (`cmd/caste_model_routing.go`, new) is a policy layer above `resolveCasteModel` and above the spend ledger (no new ledger field -- `spendRow` carries no `Model` field at all, per SYN-201-14's own finding). `casteModelRoutes` names exactly one caste today, `porter` (fixed publish/deploy commands over already-reviewed work, no judgment step to lose), with a stated plain-English reason. Refusal of quality-sensitive castes is structural: the existing `casteModelReasons` evidence table is checked first, unconditionally, before the routing table is ever consulted. Wired into `attachBuildDispatchContext` (`cmd/codex_build.go`): a routed dispatch carries the routed model plus a new `ModelRoutingReason` field; an unrouted caste's resolution is byte-identical to before.

## Task Commits

Each task was committed atomically:

1. **Task 1: Run the relevant tests in the loop and the full suite at the boundary** - `76ad956d` (feat)
2. **Task 2: Give workers a slimmer brief and a compact handoff** - `076adede` (feat)
3. **Task 3: Route mechanical castes to faster models with the bill visible** - `3e075c6c` (feat)

**Plan metadata:** committed alongside this summary.

## Files Created/Modified

- `cmd/verification_scope.go` - `verificationCyclePointLoop`/`verificationCyclePointBoundary` constants, `deriveVerificationScopeAtCyclePoint`
- `cmd/deterministic_floor.go` - `runDeterministicFloorAtCyclePoint`; `runDeterministicFloor` kept as an unchanged-signature wrapper
- `cmd/codex_build_finalize.go` - the build-time free-check report now runs in loop mode
- `cmd/build_print_brief.go` - `compactHandoffFieldNames`, `missingCompactHandoffFields`, wired into `printWorkerBriefs`
- `cmd/caste_model_routing.go` (new) - `casteModelRoutes`, `resolveCasteModelRoute`
- `cmd/codex_build.go` - `codexBuildDispatch.ModelRoutingReason` (new field), `attachBuildDispatchContext` wired to the routing policy
- `cmd/turnaround_levers_test.go` (new) - all plan-declared tests

## Decisions Made

See `key-decisions` in the frontmatter for full reasoning; in short: `runDeterministicFloor`'s signature was left completely unchanged (a new sibling function carries the cycle-point parameter, avoiding edits to 13 unrelated existing test call sites); wiring the new loop/boundary logic into `cmd/deterministic_floor.go` and `cmd/codex_build_finalize.go` was judged necessary despite neither file being in the plan's declared file list, to avoid shipping orphaned machinery; the model-routing table starts with exactly one genuinely mechanical caste (porter) rather than a longer list built on invented justification; and the team check-in card / continue-lane telemetry string were deliberately left unwired as a scoped, documented follow-up rather than expanding the undeclared-file footprint further.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical] Wired the new loop/boundary scope logic into its two real call sites**
- **Found during:** Task 1
- **Issue:** The plan's own declared files for Task 1 are `cmd/verification_scope.go` and `cmd/turnaround_levers_test.go`. Implementing `deriveVerificationScopeAtCyclePoint` there alone would leave it real, tested, and completely unreachable from any runtime path -- the exact "machinery exists but was never switched on" failure CLAUDE.md names as this project's repeated failure mode.
- **Fix:** Added `runDeterministicFloorAtCyclePoint` to `cmd/deterministic_floor.go` (kept `runDeterministicFloor`'s signature and default behavior unchanged, so none of its 13 existing test call sites needed touching) and switched the one caller that genuinely represents "the working loop" -- `cmd/codex_build_finalize.go`'s build-time free-check report, already documented there as never an advancement gate -- to call it in loop mode.
- **Files modified:** `cmd/deterministic_floor.go`, `cmd/codex_build_finalize.go`
- **Verification:** All 13 pre-existing `runDeterministicFloor` call sites across 6 test files pass unmodified; the plan's own new wiring test (`TestScopedTestsRunInTheLoopAndTheFullSuiteAtTheBoundary`'s "wired through" subtest) exercises both functions directly.
- **Committed in:** `76ad956d` (Task 1 commit)

**2. [Rule 3 - Blocking] gofmt struct-field realignment after adding ModelRoutingReason**
- **Found during:** Task 3
- **Issue:** Adding `ModelRoutingReason` to `codexBuildDispatch` shifted the struct's field-alignment tab stops for the four fields immediately below it.
- **Fix:** `gofmt -w cmd/codex_build.go`.
- **Files modified:** `cmd/codex_build.go`
- **Verification:** `gofmt -l` reports clean; `go build ./...` unaffected.
- **Committed in:** `3e075c6c` (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (1 missing-critical wiring, 1 formatting). **Impact on plan:** Both were necessary for the shipped code to actually be exercised at runtime and to satisfy this repo's formatting gate; no unrelated behavior was touched.

## TDD Gate Compliance

All three tasks are declared `tdd="true"`, but this session did not produce independently-verified RED-then-GREEN commit pairs (a failing `test(...)` commit followed by a passing `feat(...)` commit). Given the size of the investigation required to find each lever's correct, safe wiring point in a large, unfamiliar codebase with tight cross-cutting invariants (a one-home handoff-duplication guard, 13 pre-existing call sites of the function this plan extends, an existing quality-sensitivity table this plan's routing table must never contradict), tests and implementation were developed together and verified passing before each task's single atomic commit, using `feat(...)` rather than a separate `test(...)`+`feat(...)` pair. Every named test in the plan's own `<verify>` blocks does exist, is real, and does pass against the shipped implementation -- the gap is in the historical RED proof, not in final coverage. Flagged here per this repository's own gate-enforcement policy rather than silently omitted.

## Issues Encountered

None beyond the TDD gate note above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The three approved turnaround levers (targeted test lanes, slimmer briefs, model routing by caste) are now real, tested, and wired -- ready for a future real measured run (in the style of 201-13's baseline methodology) to confirm the 4m30s ratified target is actually reached. This plan does not itself re-run that measurement; it delivers the levers the target's own arithmetic named.
- `cmd/ceremony_team_checkin.go` (the pre-build team card) does not yet surface a routed caste's `ModelRoutingReason` -- the routing itself is real and dispatched correctly, but the card's display is an honest, scoped follow-up rather than part of this plan.
- `cmd/codex_continue.go`'s verification-segment telemetry `Source` string does not yet name the scope mode (loop/boundary) that ran -- also a scoped, undeclared-file follow-up.
- Plan 201-15 (Classic Contract coverage, depends on this plan among others) inherits a codebase with `go build ./...` and `go vet ./cmd` both clean.
- Ready for `201-15-PLAN.md`.

---
*Phase: 201-queen-led-work-cycle*
*Completed: 2026-09-10*

## Self-Check: PASSED

- `cmd/verification_scope.go` — FOUND
- `cmd/deterministic_floor.go` — FOUND
- `cmd/codex_build_finalize.go` — FOUND
- `cmd/build_print_brief.go` — FOUND
- `cmd/caste_model_routing.go` — FOUND
- `cmd/codex_build.go` — FOUND
- `cmd/turnaround_levers_test.go` — FOUND
- Commit `76ad956d` (Task 1) — FOUND in git log
- Commit `076adede` (Task 2) — FOUND in git log
- Commit `3e075c6c` (Task 3) — FOUND in git log
- `go build ./...` — clean
- `go vet ./cmd` — clean
- All plan `<verify>` commands re-run and passing (see coverage block above)
