---
phase: 199-front-door-and-classic-contract
plan: "04"
subsystem: lifecycle-projection
tags: [go, lifecycle, projection, read-only, provenance, next-up]

requires:
  - phase: 199-front-door-and-classic-contract
    plan: "03"
    provides: Typed lifecycle/v1 receipts, recovery provenance, seal outcomes, and archive evidence
provides:
  - One causally read-only LifecycleFacts aggregate with explicit source provenance
  - One deterministic LifecycleProjection shared across full, compact, focused, JSON, and visual views
  - Projection-owned Next Up policy with equal build/run choices, resume recovery, and optional entomb
affects: [status, phase, history, run, pause-resume, seal, entomb, lifecycle-renderers]

tech-stack:
  added: []
  patterns: [read-only fact aggregate, pure semantic projection, explicit unavailable provenance, projection-only lifecycle rendering]

key-files:
  created:
    - cmd/lifecycle_facts.go
    - cmd/lifecycle_facts_test.go
    - cmd/lifecycle_projection.go
    - cmd/lifecycle_projection_test.go
    - cmd/lifecycle_next_action_199_test.go
  modified:
    - cmd/state_load.go
    - cmd/next_action.go
    - cmd/next_action_input.go
    - cmd/next_action_card.go

key-decisions:
  - "Lifecycle orientation reads direct bytes and records missing, malformed, or unavailable provenance instead of repairing state."
  - "Accepted plans expose guided build and Autopilot as a coequal choice set with no preferred marker."
  - "Ambiguous recovery uses resume; sealed colonies use status with entomb retained only as an optional alternative."

patterns-established:
  - "Read once, project once: ordinary lifecycle screens consume LifecycleFacts through projectLifecycle rather than inferring state locally."
  - "Missing evidence stays unknown: projections never manufacture actors, costs, successful outcomes, or verified closure."
  - "Renderers translate runtime command spelling at the platform boundary without owning lifecycle branches."

requirements-completed: [CEC-01, CEC-02, LIFE-03]

duration: 31min
completed: 2026-09-03
---

# Phase 199 Plan 04: Read-Only Lifecycle Projection Summary

**A zero-write fact snapshot and pure semantic projection now give every lifecycle view one truthful answer, including equal guided/Autopilot choices and inspectable sealed state.**

## Performance

- **Duration:** 31 minutes
- **Started:** 2026-09-03T18:40:16Z
- **Completed:** 2026-09-03T19:11:43Z
- **Tasks:** 3/3
- **Files modified:** 9 production/test files

## Accomplishments

- Added one immutable `LifecycleFacts` load covering identity, progress, actors and lineage, signals, research, memory and findings, verification, elapsed time, recorded cost, history, blockers, session, and typed lifecycle evidence.
- Proved orientation reads do not change repository bytes, metadata, hub/registry/session fixtures, or Git refs across valid, missing, malformed, and legacy states.
- Added a deterministic `LifecycleProjection` with stable section order, shared result fields, explicit evidence provenance, and platform spelling applied only at the projection/render boundary.
- Replaced the independent Next Up lifecycle chooser with a projection adapter and renderer: no goal leads to init, an accepted goal leads to plan, an accepted plan presents equal build/run choices, uncertainty leads to resume, and sealed state leads to status with optional entomb.

## Task Commits

Each TDD task was committed as a RED test followed by its GREEN implementation:

1. **Task 1: Load authoritative lifecycle facts without writes** — `ecb23c37` (RED), `0089eaf7` (GREEN)
2. **Task 2: Project one semantic lifecycle result** — `19c4b8f8` (RED), `19c1929f` (GREEN)
3. **Task 3: Make Next Up a view of the shared projection** — `ca761d73` (RED), `5492fb7a` (GREEN)

**Plan metadata:** committed with this summary.

## Files Created/Modified

- `cmd/lifecycle_facts.go` — read-only aggregate loader, source diagnostics, recorded-cost facts, recovery report reader, and in-memory-state adapter.
- `cmd/lifecycle_facts_test.go` — causal before/after fingerprints and provenance coverage for valid, legacy, absent, malformed, and unavailable sources.
- `cmd/state_load.go` — reusable non-mutating colony-state compatibility load.
- `cmd/lifecycle_projection.go` — pure shared result object, ordered view sections, lifecycle action policy, and platform command translation.
- `cmd/lifecycle_projection_test.go` — structured policy, determinism, section-order, evidence, closure, and cross-platform tests.
- `cmd/lifecycle_next_action_199_test.go` — locked Next Up cases and source ratchets against command-local lifecycle policy.
- `cmd/next_action.go` — compatibility adapter that delegates lifecycle decisions to `projectLifecycle`.
- `cmd/next_action_input.go` — single lifecycle fact load for Next Up without write-backed storage reads.
- `cmd/next_action_card.go` — projection renderer for choices, reasons, evidence, alternatives, and machine-readable output.

## Decisions Made

- Used direct file reads at the orientation boundary because the existing storage reader creates lock artifacts; read-only commands must be causally unable to mutate state.
- Kept legacy state normalization in memory and attached provenance to every fact domain so malformed or absent evidence cannot silently become a successful claim.
- Preserved non-lifecycle detail overrides and the live Cobra availability gate while making `projectLifecycle` the only owner of init/plan/build/run/resume/seal/entomb policy.
- Represented accepted-plan execution as an ordered, equal-rank choice set instead of populating a single `next_command`, so wrappers cannot accidentally promote build over Autopilot.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Stopped deriving a token total that was never reported**

- **Found during:** Task 3 integration verification
- **Issue:** The new reported-cost reader initially added input and output token fields when a ledger row omitted `total_tokens`, violating the repository rule that read-only cost views display only provider-recorded totals.
- **Fix:** Count only an explicit recorded `total_tokens` value and update the valid fixture to contain the provider total it expects.
- **Files modified:** `cmd/lifecycle_facts.go`, `cmd/lifecycle_facts_test.go`
- **Verification:** `TestBothAccountingPathsAgreeOnTheTotal` and all lifecycle fact/projection tests pass.
- **Committed in:** `5492fb7a`

---

**Total deviations:** 1 auto-fixed bug.
**Impact on plan:** The correction strengthens the plan's “reported cost only when recorded” contract; no feature scope was added.

## Issues Encountered

- The exact plan-level suite passes 36 tests.
- A broader `go test ./cmd -count=1` run exposed legacy Next Up tests and golden files that still assert the deliberately superseded discuss/build-force/recover/mandatory-entomb policy. Those command-specific migrations belong to later Phase 199 plans and are recorded in `deferred-items.md`; no out-of-scope snapshots were rewritten here.
- The pre-existing archived branch-disposition fixture and suite-order-sensitive colony-prime test remain separately documented in `deferred-items.md`.

## Verification

- `go test ./cmd -run '^(TestLifecycleFacts|TestLifecycleProjection|TestLifecycleNextAction|TestNextActionCardUsesProjection)' -count=1` — 36 passed.
- `go test ./cmd -run '^(TestLoadNextActionInputDoesNotMutate|TestLoadNextActionInputGathersSavedState|TestLoaderContainsNoCommandDecision|TestBothAccountingPathsAgreeOnTheTotal)$' -count=1` — 6 passed.
- `git diff --check` — clean for all implementation commits.
- Stub scan found no TODO, FIXME, placeholder, coming-soon, or unwired UI data path in the created/modified files.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Later status, phase, history, run, pause/resume, and closure plans can render the same projection instead of rebuilding lifecycle policy.
- The typed transaction work in Plan 199-05 can use the same evidence/provenance vocabulary without changing read-only orientation.
- No Plan 199-04 implementation blocker remains.

## Self-Check: PASSED

- All nine planned production/test files and this summary exist.
- All six RED/GREEN task commits are present in Git history.
- The exact 36-test plan verification and the six existing guard tests pass.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-03*
