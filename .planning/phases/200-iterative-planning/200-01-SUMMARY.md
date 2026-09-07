---
phase: 200-iterative-planning
plan: 01
subsystem: planning-domain
tags: [go, specification, planning, evidence, immutable-revisions]

# Dependency graph
requires:
  - phase: 199-front-door-and-classic-contract
    provides: Go-owned ColonyState, Plan, Phase, and Task contracts extended by this plan
provides:
  - Immutable typed specification lineage with exact revision approval receipts
  - Evidence-backed iterative planning, stopping, candidate, recommendation, and acceptance contracts
  - Backward-compatible semantic and proof bindings for accepted plans, phases, and tasks
affects: [phase-200-planning-engine, lifecycle-finalizers, status-projections]

# Tech tracking
tech-stack:
  added: []
  patterns: [immutable state snapshots, closed wire enums, separate authority receipts]

key-files:
  created:
    - pkg/colony/specification.go
    - pkg/colony/specification_test.go
    - pkg/colony/planning.go
    - pkg/colony/planning_test.go
  modified:
    - pkg/colony/colony.go

key-decisions:
  - "Keep canonical specification snapshots in optional ColonyState.Specification while treating .aether/SPEC.md as a projection."
  - "Represent specification approval, candidate stopping, and active-plan acceptance with separate exact records and status transitions."
  - "Retain ordinal IDs for build and display compatibility while adding optional immutable semantic IDs and proof bindings."

patterns-established:
  - "Closed wire enums: custom JSON marshal and unmarshal methods reject unknown domain values."
  - "Causal validation: current-schema gaps, iterations, stops, and candidates require evidence_that_would_change while additive fields remain absent in legacy JSON."

requirements-completed: [PLAN-05, PLAN-06]

# Metrics
duration: 25 min
completed: 2026-09-07
---

# Phase 200 Plan 01: Planning Authority Contracts Summary

**Immutable specification lineage and evidence-backed planning candidates now have separate, exact Go authority contracts without breaking legacy colony state.**

## Performance

- **Duration:** 25 min
- **Started:** 2026-09-07T11:05:14Z
- **Completed:** 2026-09-07T11:30:22Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Added immutable whole-goal and feature-scoped specification revisions with typed stable-ID body sections, explicit category deltas, and exact approval receipts.
- Added the five-dimension evidence and iteration model, semantic deltas, stop decisions, plan candidates, advisory Queen recommendations, and exact acceptance receipts.
- Extended `ColonyState`, `Plan`, `PlanRevision`, `Phase`, and `Task` additively so legacy JSON still decodes without fabricating specification, candidate, or acceptance authority.

## Task Commits

Each TDD task was committed atomically through a failing-test commit followed by its implementation:

1. **Task 1: Define immutable specification contracts** — `094623a1` (test/RED), `4d500a2d` (feat/GREEN)
2. **Task 2: Define evidence, iteration, candidate, and revision bindings** — `7b941bcd` (test/RED), `1b4b7d67` (feat/GREEN)

**Plan metadata:** committed separately after state synchronization.

## Files Created/Modified

- `pkg/colony/specification.go` — Versioned specification scopes, typed content, deltas, immutable revisions, exact approvals, and validation.
- `pkg/colony/specification_test.go` — Lineage, scope, body round-trip, delta, exact-approval, and legacy-state tests.
- `pkg/colony/planning.go` — Closed planning vocabulary, evidence and confidence cards, semantic deltas, stop/candidate/recommendation/acceptance contracts, and validation.
- `pkg/colony/planning_test.go` — Five-dimension, causal-evidence, advisory-authority, exact-binding, semantic-proof, and legacy JSON tests.
- `pkg/colony/colony.go` — Optional specification/candidate authority and additive semantic/proof fields on existing state types.

## Decisions Made

- The authoritative specification is the immutable snapshot aggregate in `ColonyState`; `.aether/SPEC.md` remains a deterministic, human-readable projection rather than an authority source.
- Specification approval, planning stop/candidate state, Queen advice, and plan acceptance cannot share a shortcut: each has a distinct type, status, producer, and exact binding.
- Existing ordinal identifiers remain available to current builders and renderers; immutable semantic IDs and proof links are additive so compatible revisions preserve traceability.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Reconciled requirement tracking with the legacy heading format**

- **Found during:** Plan metadata synchronization
- **Issue:** `requirements.mark-complete` safely returned `not_found` because this milestone keeps each requirement ID and title inside one bold span, while the installed handler recognizes only an ID whose bold span closes immediately.
- **Fix:** Updated only the exact `PLAN-05` and `PLAN-06` checkboxes named in this plan after the prescribed query reported that it made no changes.
- **Files modified:** `.planning/REQUIREMENTS.md`
- **Verification:** Both exact requirement headings are checked and no other requirement status changed.
- **Committed in:** Plan metadata commit

---

**Total deviations:** 1 auto-fixed (1 blocking closeout-tool compatibility issue)
**Impact on plan:** Tracking-only fallback; implementation scope and runtime behavior are unchanged.

## Issues Encountered

- The additional repository-wide `go test ./...` run found a pre-existing Phase 199 vocabulary-inventory mismatch in `TestCurrentVocabulary199`: two terms in `199-UAT.md` are absent from its exhaustive inventory. It does not touch the files or behavior in this plan; the reproducible details are recorded in `deferred-items.md` for separate cleanup.

## TDD Gate Compliance

- Task 1 RED (`094623a1`) preceded GREEN (`4d500a2d`).
- Task 2 RED (`7b941bcd`) preceded GREEN (`1b4b7d67`).
- `go test ./pkg/colony -run 'TestPlanning|TestSpec' -count=1`, `go test ./pkg/colony -count=1`, and `go vet ./pkg/colony` all pass.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The durable contracts required by later Phase 200 commands and finalizers are ready for `200-02-PLAN.md`.
- Phase 200 work is unblocked; the unrelated Phase 199 vocabulary-inventory failure remains deferred.

## Self-Check: PASSED

- All four created Go files and the plan summary exist.
- All four TDD task commits are present in order.
- Scoped tests, the full `pkg/colony` suite, vet, and whitespace checks pass.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-07*
