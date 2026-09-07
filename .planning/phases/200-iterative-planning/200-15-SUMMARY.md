---
phase: 200-iterative-planning
plan: 15
subsystem: planning-runtime
tags: [go, route-setter, iteration-cards, confidence, candidates, owner-authority]

# Dependency graph
requires:
  - phase: 200-09
    provides: typed planning stage machine and one-worker authority transitions
  - phase: 200-13
    provides: immutable planning run header and Scout-first dispatch boundary
  - phase: 200-14
    provides: strict Scout receipt, owner-decision routing, and exact Route-Setter authorization
provides:
  - Strict Route-Setter result decoding and exact Scout receipt validation
  - Go-derived confidence, semantic delta, stop policy, and one atomic iteration card per pass
  - Weakest-gap Scout continuation, post-card material-decision pause, and pending non-active candidate persistence
affects: [200-16, 200-17, 200-19, 200-22, iterative-planning, candidate-review]

# Tech tracking
tech-stack:
  added: []
  patterns: [proposal-only worker boundary, Go-owned confidence and stop policy, append-only iteration timeline, content-addressed pending candidate]

key-files:
  created: [cmd/planning_route_stage_200_test.go]
  modified: [cmd/codex_plan_finalize.go, cmd/codex_plan_finalize_test.go]

key-decisions:
  - "Route-Setter supplies only a complete evidence-linked semantic proposal and five dimension proposals; Go owns accepted scores, weighted overall, weakest gap, stop reason, and all authority transitions."
  - "A later material choice is bound to the completed Route card before owner review, so neither a worker nor an incomplete pass can pause or redirect the loop."
  - "A stopped candidate is a write-once run artifact in pending_review; it is not inserted into active plan state before the separate review and acceptance contract exists."

patterns-established:
  - "Route boundary: exact Scout receipt -> strict proposal validation -> Go-derived card -> one legal next authority."
  - "Iteration continuity: every next Scout carries the prior card, proposal, weakest gap, causal evidence request, and accumulated evidence frontier."

requirements-completed: [CEC-03, PLAN-01, PLAN-02, PLAN-03, PLAN-04, PLAN-05]

# Metrics
duration: 42m
completed: 2026-09-07
---

# Phase 200 Plan 15: Route Iteration and Candidate Finalization Summary

**Strict Route proposals now become one Go-derived evidence card per pass, then either target the next Scout at the weakest gap, pause for an owner decision, or persist a reviewable non-active candidate.**

## Performance

- **Duration:** 42m
- **Started:** 2026-09-07T20:20:20Z
- **Completed:** 2026-09-07T21:01:36Z
- **Tasks:** 3
- **Files modified:** 3 implementation/test files

## Accomplishments

- Added a strict Route-Setter result boundary that verifies the exact manifest, run/pass, specification, base plan, Scout receipt, prior card/frontier, candidate snapshot, semantic proposal, citations, gaps, and all five dimension proposals.
- Made Go validate confidence movement, recompute weighted overall, derive semantic and authority deltas, choose the weakest causal gap, apply stop policy, and atomically append the Route artifact, receipt, card, timeline, and stage transition.
- Completed every Route exit: one precisely bound next Scout, a post-card material owner-decision checkpoint, a successor DRAFT for contract-changing answers, or a content-addressed `pending_review` candidate that cannot activate the plan.
- Preserved replay safety across card, decision, continuation, and candidate paths, including a real two-pass late-material fixture and byte-for-byte protection of the active plan on stop.

## Task Commits

Each TDD task was committed with separate RED and GREEN gates:

1. **Task 1 RED: Route proposal contract tests** - `7aec78fc` (test)
2. **Task 1 GREEN: Exact Route proposal validation** - `22a3cd90` (feat)
3. **Task 2 RED: Route card persistence tests** - `73eed7cd` (test)
4. **Task 2 GREEN: Atomic causal iteration cards** - `8e0823a7` (feat)
5. **Task 3 RED: Route progression tests** - `d7ae328a` (test)
6. **Task 3 GREEN: Continue, decision, and candidate boundaries** - `4f89020b` (feat)

## Files Created/Modified

- `cmd/codex_plan_finalize.go` - Strict Route decoding, validation, confidence/card derivation, stage transactions, weakest-gap continuation, material-decision resume, successor-specification routing, and candidate persistence.
- `cmd/codex_plan_finalize_test.go` - Command-level Route finalizer shape and boundary coverage.
- `cmd/planning_route_stage_200_test.go` - Validation, normalization, card, confidence, atomicity, replay, multi-pass, material-decision, continuation, and candidate regression fixtures.

## Decisions Made

- Route-Setter remains proposal-only. It cannot submit weighted overall, stop policy, authority, acceptance, activation, candidate status, or a state patch; the Go runtime derives those values from durable evidence and policy.
- Later-pass material candidates are carried through the Route proposal so the pass can finish and its complete card can be persisted before owner authority begins.
- The completed-card hash, not merely the earlier Scout receipt, binds a Route-discovered owner decision. Equivalent answers can resume the authorized next Scout; contract-affecting answers must create and approve an immutable successor specification first.
- Pending candidates remain run-scoped artifacts until Plan 200-16 provides the explicit review/acceptance transition. This preserves the current state validator and guarantees `Plan.ActiveRevisionID` and `Plan.Phases` remain byte-identical at stop.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Preserved the cumulative evidence frontier across Route passes**

- **Found during:** Task 3 (multi-pass continuation verification)
- **Issue:** A next-Scout dispatch initially carried only the latest Route receipt's citations. Evidence first validated on an earlier pass could disappear from the next frontier and later be presented again as fresh.
- **Fix:** Load the prior Route-to-Scout dispatch and union its evidence frontier with the current Route evidence before writing the next Scout authorization.
- **Files modified:** `cmd/codex_plan_finalize.go`, `cmd/planning_route_stage_200_test.go`
- **Verification:** The real two-pass fixture proves prior evidence remains bound while newly discovered material evidence is handled only after the second card; all focused normal and race tests pass.
- **Committed in:** `4f89020b`

---

**Total deviations:** 1 auto-fixed (1 Rule 1 bug)
**Impact on plan:** The fix closes an evidence-replay hole without expanding the Route worker's authority or public surface.

## Issues Encountered

- `go test ./... -count=1` still reports the same 29 migration-era failures already recorded after Plan 200-14 (5,819 pass, 29 fail, 6 skip). They cover legacy whole-chain planning fixtures, visuals/goldens, lifecycle and command audits, Phase 199 vocabulary drift, and future-owned candidate/schema/catalog work; none exercises the new Route tests. The existing `deferred-items.md` entry assigns these migrations to later Phase 200 plans.
- The Plan 15 focused suite passes 28/28 normally and 28/28 under the race detector. `go vet ./cmd` also passes.

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 200-16 can discover the run-scoped `candidate.json`, verify its timeline/proposal/spec/base bindings, and implement explicit owner review and activation without trusting Route-Setter output.
- Plan 200-17 can reconcile any approved successor specification using the exact affected scope persisted by the existing specification engine.
- Plan 200-19 and Plan 200-22 can project the compact card/candidate contracts into visuals, schemas, and platform guidance.
- No Plan 15 implementation blocker remains; only the already-deferred broader migration ratchets are red.

## Self-Check: PASSED

- All three implementation/test files and this summary exist.
- All six RED/GREEN task commits are present in repository history.
- Focused normal and race verification pass 28/28; static analysis passes.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-07*
