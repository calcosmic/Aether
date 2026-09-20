---
phase: 199-front-door-and-classic-contract
plan: "10"
subsystem: lifecycle-orientation
tags: [go, phase, history, lifecycle-projection, read-only, deterministic-ordering]

requires:
  - phase: 199-front-door-and-classic-contract
    plan: "04"
    provides: Immutable lifecycle facts, pure lifecycle projection, and shared Next Up policy
  - phase: 199-front-door-and-classic-contract
    plan: "09"
    provides: Authoritative full and compact status views over the shared projection
provides:
  - Focused phase detail plus deterministic list/all views from one lifecycle fact load
  - Typed reverse-chronological history over events, actors, and lifecycle receipts
  - Cross-view agreement ratchet for status, phase, and history across seven lifecycle states
affects: [lifecycle-closeouts, command-wrappers, compatibility-tests, phase-199-plan-26]

tech-stack:
  added: []
  patterns: [focused projection adapters, typed human history rows, deterministic evidence ordering, zero-write orientation]

key-files:
  created:
    - cmd/lifecycle_phase_199_test.go
    - cmd/lifecycle_history_199_test.go
    - cmd/orientation_agreement_199_test.go
  modified:
    - cmd/phase.go
    - cmd/history.go

key-decisions:
  - "History treats event source commands as evidence sources, never as inferred actors; only recorded spawn-tree identities populate actors."
  - "Receipts without an authoritative timestamp sort after timestamped evidence instead of borrowing or inventing a time."
  - "Phase exposes the shared current_phase fact separately from the selected detail so detail, list, and all retain status agreement."

patterns-established:
  - "Focused view adapter: copy identity, standing, blockers, decisions, and Next Up from LifecycleProjection while adding only command-specific detail."
  - "Honest timeline: sort authoritative timestamps newest-first, use a stable semantic tie-break, and retain malformed or unknown evidence explicitly."

requirements-completed: [CEC-01, CEC-02, LIFE-03]

duration: 27min
completed: 2026-09-03
---

# Phase 199 Plan 10: Focused Phase and Human History Summary

**Phase and history now present focused, causally read-only views of the same lifecycle answer as status, with deterministic phase selectors and a typed human evidence timeline.**

## Performance

- **Duration:** 27 minutes
- **Started:** 2026-09-03T22:26:12Z
- **Completed:** 2026-09-03T22:52:50Z
- **Tasks:** 3/3
- **Files modified:** 5 production/test files

## Accomplishments

- Replaced phase-local state inference with one immutable fact load and focused projection, while adding current, numbered, deterministic `--list`, and complete `--all` selections.
- Added accepted-plan dependencies, tasks, success criteria, attempt/actor evidence, verification, blockers, and the unchanged shared Next Up answer to focused phase output.
- Rebuilt history as typed event/actor/receipt rows ordered by authoritative timestamps, with explicit unknown evidence and machine detail excluded from default prose.
- Proved status, phase detail/list/all, and history agree on identity, goal, current phase, standing, blockers, decisions, Next Up, and projection revision across seven lifecycle fixtures.

## Task Commits

Each TDD task was committed as a RED test followed by its GREEN implementation:

1. **Task 1: Project the focused phase view** — `08d6d3d6` (RED), `4b9c1866` (GREEN)
2. **Task 2: Project human-readable reverse-chronological history** — `8c876a2d` (RED), `c279762d` (GREEN)
3. **Task 3: Ratchet orientation agreement** — `da47ab1f` (RED), `de0a99aa` (GREEN)

**Plan metadata:** committed with this summary.

## Files Created/Modified

- `cmd/phase.go` — focused phase result, selector validation, deterministic list/all views, accepted-plan evidence, visual closeout, and shared current-phase fact.
- `cmd/lifecycle_phase_199_test.go` — focused semantics, selector ordering/errors, missing evidence, and success/error zero-write proofs.
- `cmd/history.go` — typed history rows from lifecycle facts, stable reverse chronology, explicit actor provenance, machine-only raw detail, and focused rendering.
- `cmd/lifecycle_history_199_test.go` — mixed event/actor/receipt ordering, unknown evidence, focused closeout, legacy compatibility, and zero-write proofs.
- `cmd/orientation_agreement_199_test.go` — seven-state semantic comparison across status, phase detail/list/all, and history, including one-load source ratchets.

## Decisions Made

- Kept event `source` separate from `actor`. Existing durable events name the command that emitted them, not necessarily a person or worker, so the human row says `Unknown` unless spawn-tree evidence records an actor.
- Kept untimestamped receipts visible at the end of the timeline. Assigning the read time or a nearby event time would turn missing evidence into a false chronology.
- Preserved raw event strings and full receipt detail in JSON fields while the visual renderer consumes only the human event, actor, result, and timestamp fields.
- Added `current_phase` beside phase-specific `phase`/`phases` selections so list and all modes expose the same phase standing as status without conflating the current phase with the requested detail.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Preserved an empty JSON array for history events**

- **Found during:** Task 2 compatibility verification
- **Issue:** Copying an empty typed row slice through a nil destination serialized `events` as `null`, breaking the established empty-history array contract.
- **Fix:** Copy rows through a non-nil empty slice so structured output consistently emits `"events": []`.
- **Files modified:** `cmd/history.go`
- **Verification:** `TestHistoryJSONEmpty` and the combined 15-case legacy/new history suite pass.
- **Committed in:** `c279762d`

**2. [Rule 2 - Missing Critical] Retained shared phase standing in every phase selector**

- **Found during:** Task 3 cross-view agreement RED gate
- **Issue:** Detail exposed a requested phase and list/all exposed phase collections, but none retained the projection's shared current-phase fact, so those shapes could not prove agreement with status and history.
- **Fix:** Added the projection-owned `current_phase` fact to `LifecyclePhaseResult` for detail, list, and all without recomputing it.
- **Files modified:** `cmd/phase.go`
- **Verification:** `TestOrientationViewsAgree199` passes all eight test cases, including seven lifecycle fixtures.
- **Committed in:** `de0a99aa`

---

**Total deviations:** 2 auto-fixed (1 bug, 1 missing critical functionality).
**Impact on plan:** Both corrections are required to preserve the established structured contract and the plan's explicit cross-view agreement; no unrelated runtime surface was changed.

## Issues Encountered

- The initial Task 2 test fixture encoded a terminal spawn transition as one eight-field row, while the authoritative spawn-tree format records a seven-field spawn followed by a four-field status transition. The RED fixture was corrected before implementation and remained failing for the intended missing history behavior.
- The complete command-package sweep reported 7,201 passing tests, 108 failures, and 10 skips. These are the staged Phase 199 migration set already assigned to later plans: legacy status/Next Up expectations, shared goldens, the archived Phase 196 fixture, and unfinished command-surface migrations. Exact evidence and follow-up are recorded in `deferred-items.md`.

## Known Stubs

None. The pattern scan found only typed empty-slice initialization and ordinary control-flow checks; no focused view renders placeholder or mock data.

## Verification

- Exact focused phase contract — 14 passed.
- Exact focused history contract — 7 passed.
- Exact cross-view agreement contract — 8 passed.
- Combined race-enabled Plan 199-10 contract — 29 passed.
- Existing and new history compatibility suite — 15 passed.
- `git diff --check` — clean for implementation changes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Status, phase, and history now share one authoritative lifecycle answer while retaining distinct presentation scopes.
- Plan 199-26 can extend the agreement ratchet to init, colonize, plan, build, run, pause, resume, seal, and entomb after those implementations settle.
- The broader legacy/golden cleanup remains explicitly deferred to its owning Phase 199 plans and final verification gate.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-03*

## Self-Check: PASSED

- All five implementation/test artifacts and this summary exist.
- All six RED/GREEN task commits are present in Git history.
