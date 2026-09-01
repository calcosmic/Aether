---
phase: 198-put-the-thrown-away-data-back-on-screen
plan: 01
subsystem: cli-visuals
tags: [go, ceremony, continue, closeout, D-12, SHOW-01]

# Dependency graph
requires:
  - phase: 197-one-answer-to-what-next
    provides: lifecycle next-action resolver, wrapper triplet parity test pattern
provides:
  - "closeoutDirectVisual / closeoutContinueRenderInputs bridge (cmd/closeout_direct_render.go)"
  - "continued_phase / continued_phase_name on both continue-finalize result maps"
  - "completion_raw on closeoutCompletionDetails"
  - "continueTypedResultMapValue dual-type rendering fix for verification/gates"
affects: [198-04-plan-and-seal-wiring, any-future-phase-touching-continue-closeout-rendering]

# Actuals (#2632)
actuals:
  tokens: 7912
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Dual-type rendering: a render helper that reads a completion-result field must handle both the in-process typed struct and the JSON-round-tripped map[string]interface{} shape (continueTypedResultMapValue, following renderContinueWorkerFlowValue's existing precedent)."
    - "Fixture construction for parity tests: build the typed result via real Go structs, then json.Marshal/json.Unmarshal into map[string]interface{} for the closeout side -- never hand-type a nested map literal."

key-files:
  created:
    - cmd/closeout_direct_render.go
    - cmd/wrapper_path_parity_test.go
  modified:
    - cmd/ceremony_cmd.go
    - cmd/closeout_cmd.go
    - cmd/codex_continue_finalize.go
    - cmd/codex_visuals.go
    - cmd/testdata/golden_continue.txt

key-decisions:
  - "closeoutDirectVisual is wired only for the continue workflow in this plan; plan and seal are deferred to 198-04 once this architecture is proven (per plan scope)."
  - "The one spend cost line is applied once in renderCeremonyCloseout around whichever body was produced (direct or generic), never inside closeoutDirectVisual itself -- preserves the Phase 196 exactly-one-cost-line-last rule."
  - "Fixed a real, previously-invisible bug as part of the tracer: mapValue silently returned nil for a typed struct (codexContinueVerificationReport / codexContinueGateReport), so the 'Verification: N passed' and 'Gates: N/N passed' lines never rendered on EITHER the direct or chat continue path. continueTypedResultMapValue fixes this for both paths identically; the golden_continue.txt fixture was regenerated via -update-golden to reflect the now-correct, previously-thrown-away data."
  - "closeoutContinueDirectVisual defensively unwraps one more {\"result\": {...}} envelope level when completion_raw (or its fallback) lacks continued_phase -- closeoutCompletionDetails' existing manifest/worker-map unwrap heuristic doesn't recognize a continue-finalize result's shape (no manifest key, worker_flow not dispatches/results/workers), so without this the real file-based wrapper flow would silently fall back to the generic renderer. Fixed locally in the bridge rather than widening the shared heuristic used by every other workflow."

requirements-completed: [SHOW-01]

coverage:
  - id: D1
    description: "The chat-path continue closeout (`aether ceremony closeout --workflow continue`) renders byte-identically to the direct continue-finalize screen for the same result, on both the advance and blocked outcomes."
    requirement: "SHOW-01"
    verification:
      - kind: unit
        ref: "cmd/wrapper_path_parity_test.go#TestWrapperPathRendersSameCeremonyAsDirectPath"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every other workflow (build, colonize, swarm, status) is untouched by this change and still uses the pre-existing generic renderer."
    verification:
      - kind: unit
        ref: "cmd/wrapper_path_parity_test.go#TestCloseoutUnhandledWorkflowsKeepTheGenericRenderer"
        status: pass
    human_judgment: false
  - id: D3
    description: "The single spend cost line still ends the continue closeout screen exactly once, in the last position, now that the body is produced by closeoutDirectVisual."
    verification:
      - kind: unit
        ref: "cmd/wrapper_path_parity_test.go#TestCloseoutContinueCostLineStaysLastAndSingle"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-08-29
status: complete
---

# Phase 198 Plan 1: One Renderer For The Continue Closing Screen Summary

**The chat-path continue closeout now calls the exact same renderer the direct `aether continue`/`continue-finalize` CLI uses, fed by the finalizer's whole saved result, and a byte-equality test locks the two paths together.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-08-29T00:00:00Z (approx, not separately timestamped by the executor harness)
- **Completed:** 2026-08-29
- **Tasks:** 2
- **Files modified:** 7 (2 created, 5 modified)

## Accomplishments

- `continued_phase` / `continued_phase_name` added to both `advanceExternalContinue` and `finalizeBlockedExternalContinue` result maps, so the completion file can name which phase was actually continued (the advance map's `current_phase` already holds the *next* phase, so this was otherwise unrecoverable).
- `closeoutCompletionDetails` now carries the whole unmodified completion payload under `completion_raw`.
- New `cmd/closeout_direct_render.go`: `closeoutDirectVisual` (workflow dispatch, continue-only in this plan) and `closeoutContinueRenderInputs` (reconstructs phase/nextPhase/housekeeping/final/reviewDepth from the saved result plus live state), wired into `renderCeremonyCloseout` ahead of the generic `renderCeremonyCloseoutVisual` fallback.
- Fixed a genuine, previously-invisible bug found while proving byte-equality: `mapValue` silently dropped the verification/gates summary lines for BOTH the direct and chat continue paths because it type-asserted only to `map[string]interface{}` and never handled the in-process typed structs. `continueTypedResultMapValue` fixes this; `golden_continue.txt` was regenerated to show the real (previously-hidden) "Verification: N passed" / "Gates: N/N passed" lines.
- `TestWrapperPathRendersSameCeremonyAsDirectPath`, `TestCloseoutUnhandledWorkflowsKeepTheGenericRenderer`, and `TestCloseoutContinueCostLineStaysLastAndSingle` lock the architecture end to end.

## Task Commits

Each task was committed atomically:

1. **Task 1: One renderer for the continue closing screen, end to end** - `1e64578c` (feat)
2. **Task 2: Every other workflow is untouched, and the cost line stays last and single** - `ade090ee` (test)

## Files Created/Modified

- `cmd/closeout_direct_render.go` - New bridge: `closeoutDirectVisual`, `closeoutContinueDirectVisual`, `closeoutContinueRenderInputs`, `colonyPhaseByID`, `decodeSignalHousekeepingResult`
- `cmd/ceremony_cmd.go` - `renderCeremonyCloseout` now tries `closeoutDirectVisual` before falling back to the generic renderer; applies `appendSpendCostLine` once around whichever body was produced
- `cmd/closeout_cmd.go` - `closeoutCompletionDetails` sets `completion_raw`
- `cmd/codex_continue_finalize.go` - both result maps gain `continued_phase` / `continued_phase_name`
- `cmd/codex_visuals.go` - new `continueTypedResultMapValue` dual-type helper; 4 call sites in `renderContinueVisual`/`renderContinueBlockedVisual` switched from `mapValue` to it
- `cmd/testdata/golden_continue.txt` - regenerated via `-update-golden` to include the now-correctly-rendered verification/gates summary lines
- `cmd/wrapper_path_parity_test.go` - new test file with the three tests above plus fixture helpers (`wrapperParityColonyState`, `wrapperParityAdvanceResult`, `wrapperParityBlockedResult`, `roundTripToMap`, `firstDiffLine`)

## Decisions Made

See `key-decisions` in frontmatter above (scope limited to `continue`; cost-line placement rule; the `mapValue` bug fix; the local double-envelope unwrap in the bridge).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed mapValue silently dropping verification/gates summary lines on the direct CLI path too**
- **Found during:** Task 1, while proving byte-equality between the direct and chat renders
- **Issue:** `mapValue(raw interface{})` type-asserts only to `map[string]interface{}`. `result["verification"]`/`result["gates"]` on the in-process (direct CLI) path are the typed structs `codexContinueVerificationReport`/`codexContinueGateReport`, so the assertion silently failed and returned nil -- `renderContinueVerificationSummaryMap`/`renderContinueGateSummaryMap` then rendered nothing at all, on BOTH the direct terminal and (pre-fix) the chat closeout. This was invisible because nothing asserted on that content before.
- **Fix:** Added `continueTypedResultMapValue`, a dual-type helper (map passthrough, or JSON round-trip for anything else) following `renderContinueWorkerFlowValue`'s existing precedent; swapped it in at the 4 call sites in `renderContinueVisual`/`renderContinueBlockedVisual` that read `result["verification"]`/`result["gates"]`.
- **Files modified:** cmd/codex_visuals.go, cmd/testdata/golden_continue.txt (regenerated)
- **Verification:** `TestGoldenContinueVisualOutput` regenerated and passing; full `go test ./cmd/...` green
- **Committed in:** 1e64578c (Task 1 commit)

**2. [Rule 1 - Bug] Local double-envelope unwrap in the continue bridge**
- **Found during:** Task 1, while writing `TestCloseoutContinueCostLineStaysLastAndSingle` against the real file-based `renderCeremonyCloseout` path
- **Issue:** `closeoutCompletionDetails`'s existing `{"result": {...}}` unwrap only triggers when the nested map already looks like a manifest/worker payload (`closeoutManifest`/`closeoutWorkerMaps`). A continue-finalize result has neither (no manifest key; `worker_flow`, not `dispatches`/`results`/`workers`), so a real completion file's `completion_raw` can still be the un-unwrapped `{"ok":true,"result":{...}}` envelope, and `closeoutDirectVisual` would silently fall back to the generic renderer for every real continue closeout.
- **Fix:** `closeoutContinueDirectVisual` unwraps one more `result` level locally when `continued_phase` is absent, rather than widening the shared heuristic (which is used by every other workflow and out of this plan's scope).
- **Files modified:** cmd/closeout_direct_render.go
- **Verification:** `TestCloseoutContinueCostLineStaysLastAndSingle` drives the full file-based path and passes
- **Committed in:** 1e64578c (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 bugs, both necessary for the tracer's own correctness claim to hold on the real production path, not just a synthetic unit test)
**Impact on plan:** Both fixes were required for `TestWrapperPathRendersSameCeremonyAsDirectPath`/`TestCloseoutContinueCostLineStaysLastAndSingle` to be genuinely meaningful rather than vacuously passing. No scope creep beyond `continue`.

## Issues Encountered

None beyond the two deviations documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The `closeoutDirectVisual` / `closeoutContinueRenderInputs` architecture is proven end to end for `continue`, ready for 198-04 to extend to `plan` and `seal`.
- `continueTypedResultMapValue` establishes the dual-type-rendering fix pattern any future field addition to continue's result map should follow.
- No blockers for subsequent plans in this phase.

## Self-Check: PASSED

- Verified all listed created/modified files exist on disk.
- Verified commits `1e64578c` and `ade090ee` exist in git log.
- Re-ran plan-level `<verification>` command set: `go build ./...`, `go vet ./cmd`, and the full named test list all exit 0.
- Re-ran full `go test ./cmd/...` (all packages under cmd): all green, no failures.

---
*Phase: 198-put-the-thrown-away-data-back-on-screen*
*Completed: 2026-08-29*
