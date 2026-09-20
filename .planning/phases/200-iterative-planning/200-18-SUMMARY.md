---
phase: 200-iterative-planning
plan: 18
subsystem: planning-authority
tags: [go, planning, acceptance, build, autopilot, lifecycle]

requires:
  - phase: 200-16
    provides: exact candidate acceptance receipts and immutable accepted plan revisions
provides:
  - one pure current/legacy plan execution-authority policy
  - identical build and autopilot refusal codes and recovery commands
  - accepted plan lineage attribution in build manifests
affects: [200-19, 200-23, 201-execution, build, run, autopilot]

tech-stack:
  added: []
  patterns: [pure policy over lifecycle facts, read-only artifact verification, structured zero-effect refusal]

key-files:
  created: [cmd/plan_authority.go, cmd/plan_authority_test.go, cmd/plan_acceptance_gate_200_test.go]
  modified: [cmd/codex_build.go, cmd/autopilot_policy.go, cmd/autopilot_policy_test.go]

key-decisions:
  - "Build and run consume the same structured planAuthorityDecision rather than independently inferring acceptance."
  - "Current authority requires exact approved SPEC, accepted candidate, immutable proposal, timeline, and receipt bindings; legacy authority requires the explicit legacy_unbound marker."
  - "A newer pending candidate remains inactive but does not revoke an already accepted plan."

patterns-established:
  - "Authority gate: verify repository artifacts read-only, then apply one pure validator before any attempt, manifest, dispatch, or phase mutation."
  - "Authority attribution: carry revision, SPEC, candidate, timeline, receipt, and compatibility classification into build output."

requirements-completed: [PLAN-05, PLAN-06]

duration: 32min
completed: 2026-09-08
---

# Phase 200 Plan 18: Accepted Plan Authority Summary

**Exact receipt-bound plan authority now gates both manual build and autopilot, with honest legacy compatibility and zero-effect recovery.**

## Performance

- **Duration:** 32 min
- **Started:** 2026-09-07T22:52:27Z
- **Completed:** 2026-09-07T23:24:05Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Added one pure validator that accepts only an exact current owner-accepted lineage or an explicitly migrated `legacy_unbound` plan.
- Placed the same authority gate before build preparation and before the autopilot loop, returning identical structured refusal codes and exact recovery commands without state effects.
- Attached accepted revision, specification, candidate, timeline, and receipt identity to build manifests and results for downstream attribution.
- Proved candidate-ready, draft, stale, affected, accepted, legacy, and newer-pending-over-accepted behavior across both execution surfaces.

## Task Commits

Each task was committed atomically using TDD:

1. **Task 1 RED: Define accepted-plan authority contract** - `3d8aa067` (test)
2. **Task 1 GREEN: Enforce exact current and legacy authority** - `6fbc0f26` (feat)
3. **Task 2 RED: Pin build/autopilot parity and zero-effect refusal** - `5b12b3de` (test)
4. **Task 2 GREEN: Gate both execution surfaces and carry attribution** - `7851e690` (feat)
5. **Task 2 regression: Prove the real plan-only command creates no effects on refusal** - `5f4df0f2` (test)

## Files Created/Modified

- `cmd/plan_authority.go` - Shared authority decision types, pure validator, and read-only verified artifact loader.
- `cmd/plan_authority_test.go` - Exact current, draft, pending/rejected, stale, affected, broken-timeline, and legacy policy tests.
- `cmd/codex_build.go` - Pre-preparation authority gate, structured error plumbing, and manifest/result attribution.
- `cmd/plan_acceptance_gate_200_test.go` - Cross-surface parity, disk-backed acceptance/candidate-ready, and zero-mutation tests.
- `cmd/autopilot_policy.go` - Artifact-backed authority check before autopilot eligibility and loop entry.
- `cmd/autopilot_policy_test.go` - Autopilot affected-scope refusal and state immutability coverage.

## Decisions Made

- Acceptance is not a boolean shortcut: every current authority field must match the current approved specification, accepted candidate, active immutable revision, complete timeline, and separately persisted receipt.
- Recovery is machine-readable and deterministic: pending/rejected candidate uses `aether plan --candidate`, draft specification uses `aether spec`, and stale, affected, or structurally invalid planning authority uses `aether plan`.
- Pending candidate discovery is consulted only when no accepted current lineage is active, preserving the existing rule that a new proposal cannot silently revoke prior accepted work.
- Normal task and phase lifecycle status changes are checked through the immutable plan-definition hash and node authority bindings, rather than mistaken for a changed accepted plan.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Recognized the synthetic genesis base for a first accepted plan**

- **Found during:** Task 2 disk-backed current-acceptance verification
- **Issue:** The initial validator expected every base ID to name a retained revision, but the first legitimate candidate binds the synthetic `plan-unbound` base.
- **Fix:** Allow `plan-unbound` only for revision 1 with no parent while preserving exact base receipt checks.
- **Files modified:** `cmd/plan_authority.go`
- **Verification:** Current accepted disk fixture and all focused authority tests pass.
- **Committed in:** `7851e690`

**2. [Rule 1 - Bug] Decoupled immutable acceptance from mutable execution statuses**

- **Found during:** Task 2 production artifact-loader integration
- **Issue:** The existing candidate loader validates full active phase snapshots, including phase/task statuses that legitimately change during execution, which could make a valid accepted plan appear stale after a build.
- **Fix:** Added an authority-specific read-only artifact path that validates canonical hashes, exact stage/receipt bindings, revision chains, and per-node authority while comparing the mutable active view through its definition hash.
- **Files modified:** `cmd/plan_authority.go`
- **Verification:** Current accepted disk authority, existing candidate tests, and the broader build/autopilot group pass.
- **Committed in:** `7851e690`

**3. [Rule 2 - Missing Critical] Added safe candidate-only refusal discovery**

- **Found during:** Task 2 candidate-ready disk verification
- **Issue:** A reviewable candidate is persisted before it becomes active state, so state alone could only report a generic missing plan instead of the required exact candidate-review recovery.
- **Fix:** Read candidate-ready artifacts under the canonical planning root with segment validation, symlink rejection, hash/schema/stage checks, and ambiguity refusal. An already accepted lineage takes precedence so a newer pending proposal does not revoke it.
- **Files modified:** `cmd/plan_authority.go`, `cmd/plan_acceptance_gate_200_test.go`
- **Verification:** Candidate-ready build/run both refuse with `plan_authority_candidate_not_accepted`, `aether plan --candidate`, and an unchanged disk hash.
- **Committed in:** `7851e690`

---

**Total deviations:** 3 auto-fixed (2 bugs, 1 missing critical functionality)
**Impact on plan:** All fixes are required to enforce acceptance against real Phase 200 artifacts without breaking legitimate legacy or already accepted execution.

## Issues Encountered

- Repository-wide verification reported 5,823 passing, 27 failing, and 6 skipped tests in legacy/future-owned planning visuals, lifecycle cards, schemas, inventories, and orchestrator fixtures. None touches the Plan 18 authority gate; details are recorded in `deferred-items.md` and the scoped suites pass.

## Verification

- `go test ./cmd -run 'TestPlanAuthority|TestPlanAcceptanceGate200|TestAutopilotPolicy.*Authority|TestCodexBuild.*Authority' -count=1` — 23 passed.
- `go test ./cmd -race -run 'TestPlanAuthority|TestPlanAcceptanceGate200|TestAutopilotPolicy.*Authority|TestCodexBuild.*Authority' -count=1` — 23 passed.
- Broader build/run/planning regression selection — 86 passed.
- `go test ./... -count=1` — 5,823 passed, 27 pre-existing/future-owned failures, 6 skipped; deferred as out of scope.

## Known Stubs

None.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: planning-artifact-discovery | `cmd/plan_authority.go` | Build/run now perform a read-only candidate directory scan when no accepted lineage exists; access is constrained to the canonical planning root with validated path segments, symlink rejection, content hashes, schema validation, and exact stage binding. |

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Subsequent Phase 200 public-path and end-to-end plans can consume one typed authority decision from both build and run.
- The 27 unrelated repository-wide failures remain assigned to later planning migration work and do not block the Plan 18 contract.

## Self-Check: PASSED

- All six implementation/test files and this summary exist.
- All five task commit hashes are present in repository history.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-08*
