---
phase: 200-iterative-planning
plan: 17
subsystem: planning-runtime
tags: [go, specification-impact, plan-revisions, immutable-candidates, seal-gates, owner-authority]

# Dependency graph
requires:
  - phase: 200-08
    provides: specification lineage, explicit-owner planning state, stable semantic IDs, and immutable revision contracts
  - phase: 200-16
    provides: exact candidate review/acceptance, accepted plan bindings, and completed-work preservation
provides:
  - Deterministic transitive affected-scope closure across specification items, tasks, dependencies, phases, and proof links
  - Exact candidate reconciliation that preserves compatible completed work and immutable predecessor revisions
  - Current-schema insert-phase proposals with specification coverage, stale-base, and active-attempt safeguards
  - Verified Seal enforcement for approved current specifications, exact accepted plans, and completed affected scope
affects: [200-19, 200-22, 200-23, iterative-planning, living-plan-reconciliation, seal-truth]

# Tech tracking
tech-stack:
  added: []
  patterns: [stable-ID impact closure, immutable candidate insertion, exact spec-plan binding, structural seal authority]

key-files:
  created: [cmd/plan_impact.go, cmd/plan_impact_test.go, cmd/spec_plan_seal_gate_200_test.go]
  modified: [cmd/plan_revision.go, cmd/planning_state.go, cmd/lifecycle_facts.go, cmd/state_extra.go, cmd/state_extra_test.go, cmd/seal_outcome.go, cmd/testdata/command_catalog.json]

key-decisions:
  - "Specification impact is a deterministic transitive closure over stable semantic IDs; ordinal phase/task numbers remain display and execution metadata only."
  - "Only exact candidate acceptance against the approved current specification clears live affected scope; recorded historical impact remains immutable evidence."
  - "Current-schema insert-phase is a proposal operation, while genuinely spec-less legacy plans retain their documented direct compatibility path."
  - "Seal enforces structural specification/plan truth without becoming Phase 205 product acceptance; forced-incomplete closure remains explicitly non-verified."

patterns-established:
  - "Living-plan reconciliation: immutable spec successor -> affected closure -> exact pending candidate -> explicit acceptance -> preserved compatible completion."
  - "Structural Seal gate: approved current spec + exact accepted plan + no affected IDs + completed work, with deterministic recovery commands on refusal."

requirements-completed: [CEC-03, PLAN-05, PLAN-06]

# Metrics
duration: 46m
completed: 2026-09-08
---

# Phase 200 Plan 17: Living Plan Reconciliation and Seal Gate Summary

**Material specification changes now invalidate exactly their requirement-to-task-to-proof closure, reconcile only through an immutable accepted candidate, and cannot be sealed as verified while changed intent remains unresolved.**

## Performance

- **Duration:** 46m
- **Started:** 2026-09-07T22:01:23Z
- **Completed:** 2026-09-07T22:46:58Z
- **Tasks:** 3
- **Files modified:** 10 implementation, test, and generated-contract files

## Accomplishments

- Added deterministic impact expansion for added, modified, or removed specification IDs through phase/task trace links, task dependencies, and positive, negative, recovery, and public-path proof categories.
- Preserved byte-identical predecessor revisions plus lifecycle completion for compatible unaffected work, while reopening only the affected phases and tasks in the accepted successor.
- Routed current-schema `insert-phase` through a content-addressed pending candidate bound to the approved specification and exact active base; missing coverage, stale bases, active attempts, and competing candidates fail without writes.
- Added structured Seal refusal data containing exact affected semantic IDs and deterministic `aether spec`, `aether plan`, or `aether plan --candidate` recovery commands.
- Preserved historical spec-less plan compatibility and the direct-owner forced-incomplete branch without fabricating modern approval or verified completion.

## Task Commits

Each TDD task used a separate failing-test gate before implementation:

1. **Task 1 RED: Affected closure contracts** - `2598e418` (test)
2. **Task 1 RED: Successor reconciliation contract** - `6080ac7d` (test)
3. **Task 1 GREEN: Exact impact and preservation engine** - `8d66f4ef` (feat)
4. **Task 2 RED: Immutable insert contracts** - `1327e131` (test)
5. **Task 2 GREEN: Candidate-backed phase insertion** - `3db382ad` (feat)
6. **Task 3 RED: Specification/plan Seal contracts** - `f57e4f3b` (test)
7. **Task 3 GREEN: Structural verified-Seal gate** - `73b08d64` (feat)
8. **Post-plan gate fix: Command catalog refresh** - `042643f9` (test)

## Files Created/Modified

- `cmd/plan_impact.go` - Stable-ID graph construction, transitive affected/preserved closure, exact candidate coverage, and compatible completed-work restoration.
- `cmd/plan_impact_test.go` - Requirement, dependency, proof, deterministic add/remove, preservation, lifecycle, and successor-spec reconciliation contracts.
- `cmd/plan_revision.go` - Exact affected-candidate acceptance plus current-schema insertion candidate production and content-addressed artifact persistence.
- `cmd/planning_state.go` - Current-spec coexistence and validation rules for an accepted historical predecessor awaiting exact reconciliation.
- `cmd/lifecycle_facts.go` - Dynamically derived affected semantic IDs and accepted/affected binding status for lifecycle routing.
- `cmd/state_extra.go` - Public insert flags and modern candidate route while retaining the legacy-unbound compatibility branch.
- `cmd/state_extra_test.go` - Candidate creation, immutability, acceptance, missing-coverage, active-attempt, stale-base, and competing-candidate coverage.
- `cmd/seal_outcome.go` - Current specification approval, exact plan binding, affected-proof, recovery-command, legacy, and forced-incomplete Seal truth checks.
- `cmd/spec_plan_seal_gate_200_test.go` - Draft, unreconciled, wrong-spec, missing-proof, reconciled, legacy, and forced-incomplete Seal contracts.
- `cmd/testdata/command_catalog.json` - Generated catalog for the accepted Phase 200 specification, candidate, and immutable insert command surface.

## Decisions Made

- An affected task contributes its containing phase and every linked proof category to the closure; downstream tasks become affected through stable dependency aliases. An independent completed branch stays preserved only when its compatibility hash and upstream closure remain unchanged.
- A current approved specification successor may coexist with the historical accepted predecessor for inspection and recovery, but lifecycle facts mark the binding `affected`; build and verified Seal cannot rely on it until exact candidate acceptance.
- A candidate reconciles affected scope only when its proposal or explicit semantic removals cover every affected ID. Merely naming the newest specification revision is insufficient.
- Current-schema phase insertion requires a current approved spec item (or exact current spec revision), the exact active base revision, no active attempt, and no pending candidate. The accepted plan and predecessor revision remain untouched until the normal exact acceptance path runs.
- Seal consumes the immutable lifecycle snapshot and returns `AffectedSemanticIDs` plus `RecoveryCommands`. It does not infer final owner product approval, rewrite state, or backfill legacy plans.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Integrated successor impact into planning validation and lifecycle facts**

- **Found during:** Task 1 (successor-spec reconciliation)
- **Issue:** Computing the closure alone left two correctness gaps: current planning validation rejected the historically accepted predecessor as soon as a successor spec existed, and lifecycle routing had no dynamic affected marker to prevent work from relying on it.
- **Fix:** Allowed the immutable accepted predecessor to coexist as historical authority, derived live affected IDs from the current spec/active-plan mismatch, and classified the binding as `affected` until exact acceptance clears it.
- **Files modified:** `cmd/planning_state.go`, `cmd/lifecycle_facts.go`, `cmd/plan_impact.go`
- **Verification:** `TestPlanImpactPlanningStateAdmitsApprovedSuccessorWithoutMutatingAcceptedPredecessor` and the broader planning-state regression selection passed.
- **Committed in:** `8d66f4ef`

**2. [Rule 1 - Bug] Deep-cloned trace slices before preserving lifecycle status**

- **Found during:** Task 1 (predecessor immutability verification)
- **Issue:** Shallow phase/task copies could share proof-link backing arrays with retained revisions, allowing later candidate normalization to mutate historical evidence indirectly.
- **Fix:** Candidate preservation now works over deep-cloned phases/tasks and compares compatibility hashes that exclude only lifecycle and planning-binding metadata.
- **Files modified:** `cmd/plan_impact.go`, `cmd/plan_revision.go`
- **Verification:** Byte-identical predecessor and unaffected-completed-task tests passed.
- **Committed in:** `8d66f4ef`

**3. [Rule 3 - Blocking] Used canonical state approval for post-acceptance insertion**

- **Found during:** Task 2 (accepted-plan insert fixture)
- **Issue:** The generic approved-spec prerequisite also demanded a byte-current readable projection. Accepted plan traceability legitimately changes the regenerated projection after plan activation, so that check could block a valid immutable insert even though canonical state held the exact approval.
- **Fix:** The insert path validates the current specification lineage, status, approval receipt, and exact revision/hash directly from locked state; projection repair remains a separate presentation concern.
- **Files modified:** `cmd/plan_revision.go`
- **Verification:** Exact candidate creation and later acceptance pass while missing coverage, stale base, active attempt, and pending-candidate cases remain fail-before-write.
- **Committed in:** `3db382ad`

**4. [Rule 3 - Blocking] Refreshed the generated command catalog**

- **Found during:** Post-task command-package verification
- **Issue:** The accepted new insert-phase flags made `TestAuditCatalogGolden` stale; the same generated refresh also captured the already-landed Phase 200 spec and candidate flags.
- **Fix:** Regenerated `cmd/testdata/command_catalog.json` through its repository-native golden update path and verified it normally afterward.
- **Files modified:** `cmd/testdata/command_catalog.json`
- **Verification:** `go test ./cmd -run '^TestAuditCatalogGolden$' -count=1` passed.
- **Committed in:** `042643f9`

---

**Total deviations:** 4 auto-fixed (1 Rule 1 bug, 1 Rule 2 missing-critical integration, 2 Rule 3 blocking issues)
**Impact on plan:** Every change closes a direct correctness or verification gap in the planned immutable reconciliation path. No unrelated feature scope or external dependency was added.

## Issues Encountered

- The Go build cache had grown to roughly 76 GiB and exhausted the filesystem during verification. `go clean -cache` was stopped after it safely freed about 32 GiB; no repository file was removed or rewritten.
- The complete `go test ./cmd -count=1` run still fails five older Plan-only fixture areas because they invoke Plan without first establishing the newly required approved specification. Focused reruns reproduce the issue before Plan 17 code is exercised. The exact tests and follow-up are recorded in `deferred-items.md`; weakening D-10/D-12 was intentionally avoided.
- `requirements.mark-complete` does not parse this milestone's legacy requirement format, but `CEC-03`, `PLAN-05`, and `PLAN-06` were already checked complete. `state.update-progress` likewise found no Markdown-body progress field; `state.advance-plan` updated the structured completed-plan count and the roadmap advanced to 17/25.
- No package, authentication, or external-service gate was encountered.

## Known Stubs

None. The modified command catalog still lists two older commands whose existing descriptions say `placeholder`; neither entry was introduced or changed by Plan 17.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: owner_authority_transition | `cmd/state_extra.go`, `cmd/plan_revision.go` | `insert-phase` now creates persisted candidate/stage/timeline artifacts. Exact approved-spec/base checks, active-attempt and competing-candidate refusal, content addressing, allowlisted repository paths, and normal explicit acceptance constrain the new state-changing path. |

## User Setup Required

None - no external service configuration required.

## Verification

- `go test ./cmd -run 'TestPlanImpact|TestPlanRevision.*Preserve|TestPlanRevision.*Affected' -count=1` - passed after Task 1.
- `go test ./cmd -run 'TestInsertPhase.*Candidate|TestInsertPhase.*Immutable|TestInsertPhase.*Refuse' -count=1` - passed after Task 2.
- `go test ./cmd -run 'TestSpecPlanSealGate200|TestSealOutcome.*Legacy|TestSealOutcome.*Forced' -count=1` - passed after Task 3.
- `go test ./cmd -run 'TestPlanImpact|TestInsertPhase|TestSpecPlanSealGate200|TestSealOutcome' -count=1` - passed in 20.394s after all commits.
- `go test -race ./cmd -run 'TestPlanImpact|TestInsertPhase|TestSpecPlanSealGate200|TestSealOutcome' -count=1` - passed in 24.156s.
- `go test ./cmd -run '^TestAuditCatalogGolden$' -count=1` - passed after the generated catalog refresh.
- `go test ./cmd -count=1` - broader suite ran for 243.261s and exposed the five deferred pre-approved-SPEC fixture failures described above; Plan 17's required and race gates remain green.

## TDD Gate Compliance

- Task 1: RED commits `2598e418` and `6080ac7d` precede GREEN commit `8d66f4ef`.
- Task 2: RED commit `1327e131` precedes GREEN commit `3db382ad`.
- Task 3: RED commit `f57e4f3b` precedes GREEN commit `73b08d64`.

## Next Phase Readiness

- Plan 200-19 can render affected-scope and exact recovery information without recomputing specification or plan authority.
- Plans 200-22/23 can document and prove the real revise -> inspect candidate -> exact accept -> build/seal journey.
- The remaining old Plan-only fixtures need approved-spec setup in their later-owned public-path migration; this does not block Plan 17's living-plan or Seal contracts.

## Self-Check: PASSED

- All ten implementation, test, and generated-contract files plus this summary exist.
- All eight RED/GREEN/generated-gate commits are present in repository history.
- No tracked file was deleted, the required combined suite passes, and the same selection passes under the race detector.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-08*
