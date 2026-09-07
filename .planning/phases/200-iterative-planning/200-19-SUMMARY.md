---
phase: 200-iterative-planning
plan: 19
subsystem: planning-presentation
tags: [go, terminal-ui, json, planning, specification, accessibility]

requires:
  - phase: 200-10
    provides: compiled specification command and typed specification result
  - phase: 200-11
    provides: Discuss-to-SPEC handoff and owner approval boundary
  - phase: 200-15
    provides: immutable iteration cards and semantic planning deltas
  - phase: 200-16
    provides: plan candidates, Queen recommendations, and acceptance receipts
  - phase: 200-17
    provides: living-plan impact and revision authority facts
  - phase: 200-18
    provides: shared build/run plan-authority policy
provides:
  - typed terminal and JSON projections for specification and iterative planning state
  - exact four-band terminal layouts with bounded append-only output
  - explicit machine/public stop-reason labels and receipt-gated execution actions
affects: [200-20, 200-21, 200-22, 200-23, 200-24, 200-25, plan, spec]

tech-stack:
  added: []
  patterns: [typed projection before rendering, semantic parity across output modes, width-aware append-only cards]

key-files:
  created: [cmd/planning_visuals.go, cmd/planning_visuals_test.go]
  modified: [cmd/codex_visuals.go, cmd/codex_visuals_test.go]

key-decisions:
  - "Terminal width changes layout only; typed planning facts and authority never change by width or output mode."
  - "JSON retains each machine stop enum and adds a sibling public label instead of replacing canonical vocabulary."
  - "Build and run actions appear together only after receipt-backed candidate acceptance; pending candidates expose only exact review/acceptance actions."
  - "Canonical Phase 200 result maps route before the legacy whole-plan renderer, while legacy and repair results retain their established path."

patterns-established:
  - "Planning projection: convert canonical structs into presentation structs, then render terminal or JSON without deriving policy."
  - "Terminal safety: strip cursor controls, retain safe SGR color, measure visible columns, and wrap prose without rewriting prior lines."

requirements-completed: [CEC-03, PLAN-01, PLAN-02, PLAN-03, PLAN-04, PLAN-05, PLAN-06]

duration: 22min
completed: 2026-09-08
---

# Phase 200 Plan 19: Planning Visual and JSON Projection Summary

**Canonical specification and planning facts now render as bounded terminal cards and typed JSON, with honest candidate authority and stable owner-facing stop labels.**

## Performance

- **Duration:** 22 min
- **Started:** 2026-09-07T23:35:03Z
- **Completed:** 2026-09-07T23:56:44Z
- **Tasks:** 2
- **Files modified:** 4 implementation/test files

## Accomplishments

- Added typed projections and terminal cards for specification, preset, Scout, Route-Setter, iteration, owner decision, stop, candidate, acceptance, living-plan impact, and four-beat refusal states.
- Preserved all nine specification body categories in approved order and every iteration card's five fixed planning-readiness dimensions, fresh evidence, weakest gap, semantic delta, authority impact, decision, and evidence-that-would-change facts.
- Added exact public stop labels beside retained machine enums in JSON: `target sufficiency`, `diminishing returns`, `stall detected`, and `iteration cap`.
- Enforced `<48`, `48-63`, `64-95`, and `>=96` terminal layout bands, visible-column bounds, NO_COLOR behavior, and append-only output with no cursor positioning.
- Routed canonical candidate and iteration result maps through the new renderer while preserving the legacy whole-plan renderer for non-Phase-200 results.

## Task Commits

Each task was committed atomically using TDD:

1. **Task 1 RED: Define semantic planning projection expectations** - `fc778bc6` (test)
2. **Task 1 GREEN: Render canonical specification and planning state** - `0c04928e` (feat)
3. **Task 2 RED: Lock width, no-color, identity, and routing contracts** - `335d08f3` (test)
4. **Task 2 GREEN: Enforce terminal safety and canonical result routing** - `2bf63e7a` (feat)

## Files Created/Modified

- `cmd/planning_visuals.go` - Typed terminal/JSON projections, canonical result routing, width bands, control sanitization, and semantic card renderers.
- `cmd/planning_visuals_test.go` - Semantic order, authority, JSON-label, six-boundary width, no-color, append-only, and safe-identifier tests.
- `cmd/codex_visuals.go` - `spec: 📜` identity, canonical Plan renderer dispatch, and planning JSON projection hook.
- `cmd/codex_visuals_test.go` - Compiled candidate/iteration routing and full Specification identity tests.

## Decisions Made

- Renderer inputs remain canonical typed structs. Presentation may reshape or wrap those facts but does not calculate stop policy, candidate eligibility, or owner authority.
- A pending/draft candidate never emits build or run. An accepted candidate emits both as coequal actions only when the accepted status is backed by its acceptance receipt.
- Human stop copy is not a lossy replacement for machine state: terminal output uses the public label, while JSON carries both the original enum and `stop_reason_public_label`.
- Long terminal content wraps at measured display width. Only a deliberately labelled indivisible `Identifier: sha256:...` line is permitted to exceed a width, and that exception has a dedicated test.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Removed a renderer-owned next-action literal caught by the repository ratchet**

- **Found during:** Task 2 broader command-package verification
- **Issue:** The first implementation hand-wrote the combined accepted-candidate build/run line, creating one new site outside the shared action data.
- **Fix:** Render the line from the typed `ExecutionActions` projection, preserving equal visual treatment without adding a second action authority.
- **Files modified:** `cmd/planning_visuals.go`
- **Verification:** `TestNextActionNeverHardcoded` now reports only the four pre-existing Phase 200 sites in `cmd/codex_plan.go` and `cmd/spec_cmd.go`; no Plan 19 site remains.
- **Committed in:** `2bf63e7a`

---

**Total deviations:** 1 auto-fixed bug
**Impact on plan:** The fix strengthens the required no-rewrite/single-authority contract without expanding scope.

## Issues Encountered

- Full `go test ./cmd -count=1` completed with 4,281 passing, 30 failing, and 5 skipped tests. The failures are pre-existing/future-owned migration expectations around legacy Plan/preset/SPEC fixtures, visual goldens, lifecycle cards, schemas, Phase 199 vocabulary, and orchestrator boundaries; details are appended to `deferred-items.md`.
- The installed requirement updater does not parse this milestone's bold-ID checkbox format; all seven named requirements were already checked complete, so `REQUIREMENTS.md` needed no manual edit. `state.update-progress` likewise found no Markdown-body progress field, while `state.advance-plan` updated the structured plan count and the roadmap advanced normally.

## Verification

- `go test ./cmd -run 'TestPlanningVisuals.*Semantic|TestPlanningVisuals.*JSON|TestPlanningVisuals.*Authority' -count=1` — 5 passed.
- `go test ./cmd -run 'TestPlanningVisuals.*Width|TestPlanningVisuals.*NoColor|TestCodexVisuals.*Planning|TestCodexVisuals.*Spec' -count=1` — 12 passed.
- `go test ./cmd -run 'TestPlanningVisuals|TestCodexVisuals.*Planning|TestCodexVisuals.*Spec' -count=1` — 17 passed.
- `go test ./cmd -race -run 'TestPlanningVisuals|TestCodexVisuals.*Planning|TestCodexVisuals.*Spec' -count=1` — 17 passed.
- `go vet ./cmd` — passed.
- `go test ./cmd -count=1` — 4,281 passed, 30 future-owned migration failures, 5 skipped; deferred as out of scope.

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plans 200-20 and 200-21 can reuse the compiled presentation vocabulary when migrating Claude Code and OpenCode public wrappers.
- Plans 200-22 through 200-25 can now test schema, parity, inventory, and end-to-end journeys against one typed terminal/JSON projection boundary.
- The unrelated legacy command-suite failures remain tracked for those later migration plans and do not block the Plan 19 contract.

## Self-Check: PASSED

- All four implementation/test files and this summary exist.
- All four RED/GREEN task commit hashes are present in repository history.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-08*
