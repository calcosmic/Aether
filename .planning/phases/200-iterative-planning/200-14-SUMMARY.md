---
phase: 200-iterative-planning
plan: 14
subsystem: planning-runtime
tags: [go, scout, route-setter, stage-receipts, owner-decisions, specification]

# Dependency graph
requires:
  - phase: 200-09
    provides: typed planning stage machine, one-worker manifests, and authority transitions
  - phase: 200-11
    provides: immutable specification revision and approval engine
  - phase: 200-12
    provides: material-decision batching, card projection, answer equivalence, and resume tokens
  - phase: 200-13
    provides: Scout-first planning manifest, run header, and persisted dispatch boundary
provides:
  - Strict Scout-only result decoding and evidence validation
  - Durable Scout artifact and chained stage receipt before Route-Setter authorization
  - One content-addressed first-pass owner decision batch with renderer-neutral cards
  - Exact direct-answer resume, late-material Route authorization, and successor-specification gating
affects: [200-15, 200-17, iterative-planning, route-setter, planning-ui]

# Tech tracking
tech-stack:
  added: []
  patterns: [strict typed worker result, content-addressed boundary token, write-once planning artifacts, replay-safe authority transition]

key-files:
  created: [cmd/planning_scout_stage_200_test.go]
  modified: [cmd/codex_plan_finalize.go, cmd/codex_plan_finalize_test.go]

key-decisions:
  - "Persist an answer-free boundary token with the first-pass batch; issue the completed resume token only after every exact card has an owner answer."
  - "Bind later-pass material candidates into the Route-Setter manifest through the candidate snapshot hash and expose the full candidates in a content-addressed Route authorization envelope."
  - "Translate contract-changing choices into minimal modifications of their affected stable specification items, then stop at exact specification approval and scoped reconciliation."

patterns-established:
  - "Scout boundary: strict result -> write-once artifact -> chained receipt -> one legal next authority."
  - "Owner decision: one evidence-first batch per first Scout pass, with no worker dispatch while authority is pending."

requirements-completed: [CEC-03, PLAN-01, PLAN-04]

# Metrics
duration: 34m
completed: 2026-09-07
---

# Phase 200 Plan 14: Scout Boundary and Decision Routing Summary

**Strict Scout receipts now lead to exactly one evidence-bound owner batch or one Route-Setter authorization, with contract-changing answers diverted through an immutable successor specification.**

## Performance

- **Duration:** 34m
- **Started:** 2026-09-07T19:38:57Z
- **Completed:** 2026-09-07T20:13:22Z
- **Tasks:** 2
- **Files modified:** 3 implementation/test files

## Accomplishments

- Added a strict Scout result contract that rejects scores, semantic plan changes, stop claims, candidates, activation, state patches, and unknown fields.
- Validated fresh evidence, finding/gap citations, decision candidates, exact stage authority, and immutable run-header scope before committing the Scout receipt.
- Persisted one content-addressed first-pass decision batch and renderer-neutral card set, while dispatching no Route-Setter until every answer is bound.
- Authorized exactly one Route-Setter for no-material or later-pass Scout outcomes and carried late material candidates through the exact hashed authorization.
- Routed equivalent answers directly to Route-Setter and contract-changing answers to a distinct successor DRAFT that cannot dispatch before approval and reconciliation.

## Task Commits

Each task was committed with TDD gates:

1. **Task 1 RED: Scout result boundary tests** - `12c406f2` (test)
2. **Task 1 GREEN: Strict Scout finalization** - `f949ab09` (feat)
3. **Task 2 RED: Decision and direct Route coordination tests** - `cf864ad1` (test)
4. **Task 2 RED: Late-material and successor-specification tests** - `c1161363` (test)
5. **Task 2 RED: Command finalizer boundary test** - `975ad58e` (test)
6. **Task 2 GREEN: Scout decision and Route coordination** - `e62f799a` (feat)

## Files Created/Modified

- `cmd/codex_plan_finalize.go` - Strict Scout decoder, evidence validator, receipt coordination, decision persistence, exact answer resume, Route authorization, and successor-specification handoff.
- `cmd/planning_scout_stage_200_test.go` - Boundary, rejection, replay, batch, late-material, answer, and contract-change regression fixtures.
- `cmd/codex_plan_finalize_test.go` - Command-level checks for visible Scout completion and exact next-boundary projection.

## Decisions Made

- The first response persists a content-addressed boundary token without fabricating answers. The existing completed resume-token invariant remains intact: a completed token exists only after all cards are answered.
- The typed stage manifest remains narrow. Full late-material candidates live in the content-addressed Route authorization envelope, while the manifest binds their exact canonical set through `candidate_snapshot_hash`.
- Contract-changing answers preserve each affected specification item's stable ID and section-specific verification/path fields; the selected answer and consequence become explicit decision evidence in the successor revision.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Bound decision candidates into the candidate snapshot**

- **Found during:** Task 2 (Route authorization implementation)
- **Issue:** The initial Scout receipt snapshot hash covered the run frontier but not the normalized decision candidates, so a Route manifest could not prove the exact candidate set it was given.
- **Fix:** Added the canonical candidate list to `planningScoutCandidateSnapshotHash` and carried the same candidates in the content-addressed Route authorization envelope.
- **Files modified:** `cmd/codex_plan_finalize.go`
- **Verification:** Late-material test proves the exact evidence-bearing candidate reaches Route-Setter; all focused normal/race tests pass.
- **Committed in:** `e62f799a`

**2. [Rule 1 - Bug] Made successor-specification resume recoverable across split durable boundaries**

- **Found during:** Task 2 (contract-answer replay verification)
- **Issue:** The specification engine and planning-stage store use separate lifecycle transactions; a retry after the specification commit but before the stage-state commit could otherwise see the predecessor as superseded and fail permanently.
- **Fix:** Resolve the exact predecessor from immutable lineage, accept only its approved-or-superseded identity, replay the same successor, and return the existing approval checkpoint without issuing Route authority.
- **Files modified:** `cmd/codex_plan_finalize.go`, `cmd/planning_scout_stage_200_test.go`
- **Verification:** Exact contract-answer replay returns the same successor DRAFT and preserves `spec_approval_required`.
- **Committed in:** `e62f799a`

---

**Total deviations:** 2 auto-fixed (2 Rule 1 bugs)
**Impact on plan:** Both fixes strengthen exact candidate identity and crash-safe replay without expanding the public planning authority model.

## Issues Encountered

- The repository-wide `go test ./cmd -count=1` gate still reports 29 migration-era failures in legacy Plan visuals/fixtures, lifecycle next-action ratchets, Phase 199 vocabulary tracking, and future-owned schema/catalog artifacts. These are outside Plan 14 and are recorded in `deferred-items.md`.
- Plan 14's focused gate passes 29/29 normally and 29/29 under the race detector; `go vet ./cmd` also passes.

## Known Stubs

None.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: planning-artifact-integrity | `cmd/codex_plan_finalize.go` | Adds persistent owner-decision batches, boundary/resume tokens, and Route authorization envelopes under `.aether/data/planning`; every artifact is strictly decoded, content-addressed, path-validated, write-once, and committed through lifecycle transactions. |
| threat_flag: specification-authority | `cmd/codex_plan_finalize.go` | Converts exact contract-changing owner choices into successor DRAFT revisions; no answer grants approval or Route authority, and the stage stops at the existing approval/reconciliation gates. |

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 200-15 can consume the exact Route-Setter manifest, Scout receipt, candidate snapshot, and late-material candidate envelope to build the complete iteration card.
- Plan 200-17 can complete approved successor reconciliation using the pending stable-ID scope recorded by this plan.
- No Plan 14 blocker remains; only already-deferred broader migration ratchets are red.

## Self-Check: PASSED

- All three implementation/test files and this summary exist.
- All six TDD/task commits are present in repository history.
- Focused normal and race verification both pass 29/29 tests.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-07*
