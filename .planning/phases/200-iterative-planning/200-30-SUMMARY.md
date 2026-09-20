---
phase: 200-iterative-planning
plan: 30
subsystem: candidate-review-authority
tags: [go, canonical-hashing, tamper-resistance, planning, tdd]

requires:
  - phase: 200-28
    provides: "Repository mutation sessions and serialized planning persistence"
  - phase: 200-29
    provides: "Canonical specification authority at planning and execution boundaries"
provides:
  - "One complete immutable candidate review-payload digest covering proposal, review evidence, semantic delta, authority impact, recommendation, and time boundary"
  - "Canonical nested identities for planning gaps, assessments, semantic changes, authority impacts, semantic deltas, and Queen recommendations"
  - "Strict standalone-revision and accepted build/run validation through one shared stripped-proposal preimage"
  - "Mutation-matrix proof that copied hashes and stale acceptance tokens cannot bless edited review meaning"
affects: [phase-200-verification, plan-candidates, candidate-acceptance, planning-state, build-authority, run-authority]

tech-stack:
  added: []
  patterns:
    - "Address immutable candidate content before populating derived candidate back-references."
    - "Tag sequence-bearing semantic changes with their plan section while normalizing only set-like evidence."
    - "Persisted hashes are claims; review and authority boundaries independently reproduce the canonical preimage."

key-files:
  created:
    - cmd/plan_candidate_binding_200_test.go
  modified:
    - pkg/colony/planning.go
    - pkg/colony/planning_test.go
    - cmd/planning_delta.go
    - cmd/codex_plan_finalize.go
    - cmd/plan_candidate.go
    - cmd/plan_revision.go
    - cmd/planning_state.go
    - cmd/plan_authority.go

key-decisions:
  - "Candidate identity excludes only mutable status/acceptance and clears only proposal-, phase-, and task-level candidate back-references plus the recommendation candidate ID while deriving the immutable digest."
  - "Retained in-memory state permits structural legacy compatibility, while artifact review and build/run authority require loader-attested canonical candidate, timeline, and acceptance-receipt validation."
  - "PlanningGap.Validate remains the structural pre-addressing seam; canonical candidate validation recomputes every addressed gap body and identity."

patterns-established:
  - "Construction, acceptance, standalone revision validation, and accepted authority all call canonicalPlanCandidateProposalHash."
  - "Candidate loaders expose acceptance fields only after nested review records, timeline binding, derived back-references, candidate identity, and exact acceptance token have all been recomputed."

requirements-completed: [PLAN-02, PLAN-03, PLAN-06]

coverage:
  - id: D1
    description: "Every immutable review field and nested planning record is canonically addressed and recomputed before candidate review or authority is granted."
    requirement: PLAN-02
    verification:
      - kind: integration
        ref: "cmd/plan_candidate_binding_200_test.go#TestPlanCandidateSemanticIntegrity200"
        status: pass
      - kind: unit
        ref: "pkg/colony/planning_test.go#TestPlanning"
        status: pass
    human_judgment: false
  - id: D2
    description: "Adjacent-section changes remain distinct, evidence sets normalize deterministically, proposal sequences stay order-sensitive, and empty or degenerate material deltas are refused."
    requirement: PLAN-03
    verification:
      - kind: integration
        ref: "cmd/plan_candidate_binding_200_test.go#TestPlanCandidateSemanticIntegrity200"
        status: pass
      - kind: other
        ref: "go test ./cmd -run '^(TestPlanCandidateSemanticIntegrity200|TestPlanningSemanticDelta.*|TestPlanning.*Assessment.*)' -count=1"
        status: pass
    human_judgment: false
  - id: D3
    description: "Valid and mutated candidates traverse standalone revision and accepted build/run authority validation, with stale hashes, derived back-references, or tokens refused without repository mutation."
    requirement: PLAN-06
    verification:
      - kind: integration
        ref: "cmd/plan_candidate_binding_200_test.go#TestPlanCandidateSemanticIntegrity200"
        status: pass
      - kind: other
        ref: "go test ./cmd -run '^(TestPlanCandidateSemanticIntegrity200|TestPlanCandidate.*|TestPlanningRoute.*Candidate.*|TestPlanningState.*|TestPlanAuthority.*)' -count=1"
        status: pass
    human_judgment: false

duration: 45 min
completed: 2026-09-09
status: complete
---

# Phase 200 Plan 30: Complete Candidate Review Payload Addressing Summary

**A plan candidate now identifies the exact proposal, semantic change, authority impact, evidence, recommendation, and review window that the owner accepts, and every later authority gate independently reproduces that meaning.**

## Performance

- **Duration:** 45 min
- **Started:** 2026-09-08T21:51:23Z
- **Completed:** 2026-09-08T22:36:11Z
- **Tasks:** 3
- **Files modified:** 9

## Accomplishments

- Added canonical production builders and validators for every nested planning review record, including section-aware semantic changes, distinct authority impacts, set-normalized evidence, whole-number confidence, exact before/after identities, and non-empty material deltas.
- Centralized the non-self-referential proposal preimage in `canonicalPlanCandidateProposalHash` and used it for candidate construction, acceptance, standalone revision validation, and accepted build/run authority.
- Bound the complete immutable review payload, creation/expiry boundary, timeline/card authority, stop decision, five assessments, residual gaps, evidence-that-would-change, semantic delta, and Queen recommendation into candidate identity while leaving only status and acceptance mutable.
- Added a stopped two-pass production fixture and exhaustive mutation matrix covering every nested review field, exact acceptance tokens, derived back-references, adjacency, empty/degenerate input, ordering, vocabulary, no-change behavior, and zero-mutation refusal.

## Task Commits

1. **Task 1: Specify complete candidate and nested-delta addressing** — `dc2ad274` (test)
2. **Task 2: Canonically recompute semantic changes and authority impacts** — `65f4034d` (feat)
3. **Task 3: Address and load the complete immutable review payload** — `5d40e6e5` (feat)

## Files Created/Modified

- `pkg/colony/planning.go` — Owns canonical normalization, addressing, and strict body-derived validation for nested review records.
- `pkg/colony/planning_test.go` — Builds named strict fixtures through production addressing functions.
- `cmd/planning_delta.go` — Produces section-tagged semantic changes, distinct authority impacts, and canonical semantic deltas.
- `cmd/codex_plan_finalize.go` — Constructs the complete candidate review payload and populates derived back-references only after addressing.
- `cmd/plan_candidate.go` — Defines the shared stripped-proposal preimage, complete candidate digest, review-payload addressing, and exact acceptance token.
- `cmd/plan_revision.go` — Uses the shared proposal preimage for acceptance and phase-insert candidate construction.
- `cmd/planning_state.go` — Strictly validates standalone canonical revisions while retaining a structural seam for legacy in-memory fixtures.
- `cmd/plan_authority.go` — Requires loader-attested canonical candidate, timeline, receipt, and token validation before build/run authority.
- `cmd/plan_candidate_binding_200_test.go` — Exercises the full immutable-payload mutation, ordering, vocabulary, acceptance, and authority matrix.

## Decisions Made

- The candidate digest clears only proposal-, phase-, and task-level candidate IDs/content hashes and `QueenPlanRecommendation.CandidateID` to break derived fixed points. All other immutable review and authority inputs remain bound.
- Mutable status and acceptance fields do not rewrite immutable candidate identity. The acceptance receipt and exact token separately bind the approved transition.
- Sequence-bearing proposal sections remain order-sensitive; only contractually set-like evidence is sorted and deduplicated.
- Legacy retained-state validation remains structural because existing callers can hold pre-addressed shapes. Persisted artifact review and repository-backed authority remain strict and require internal proof that canonical loading succeeded.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Preserved retained-state compatibility without weakening artifact authority**
- **Found during:** Task 3 focused `TestPlanningState.*` and `TestPlanAuthority.*` verification
- **Issue:** Enabling strict recomputation directly inside the retained-state structural seam rejected established pre-addressed legacy fixtures before they reached the canonical persisted-artifact boundary.
- **Fix:** Kept retained-state shape validation compatible, added canonical verification markers to the production loader, and made accepted build/run authority require those markers together with independently recomputed candidate, timeline, receipt, and token identities.
- **Files modified:** `cmd/planning_state.go`, `cmd/plan_authority.go`
- **Verification:** The complete `TestPlanningState.*`, `TestPlanAuthority.*`, candidate-route, standalone-revision, and accepted-authority suites pass.
- **Committed in:** `5d40e6e5`

---

**Total deviations:** 1 auto-fixed bug
**Impact on plan:** Compatibility remains confined to non-authoritative retained shapes; persisted review and execution authority are still fail-closed and canonically recomputed.

## Issues Encountered

- The Task 1 RED suite failed only on the deliberately absent canonical addressing APIs, establishing the required fail-first gate.
- Strict Task 3 validation exposed legacy structural fixtures in existing state/authority tests; the authority-marker split above resolved the regression inside the declared file scope.
- The known unrelated repository-wide `go test ./...` command-package timeout was not rerun. Plan 30's exact focused suites, repeated combined gate, and vet checks are green.

## TDD Gate Compliance

- **RED:** `dc2ad274` added the complete review-payload mutation matrix and failed on the intended missing production APIs.
- **GREEN:** `65f4034d` added canonical nested-record addressing and migrated the strict package fixtures; the Task 2 focused suites passed.
- **INTEGRATION GREEN:** `5d40e6e5` bound the complete candidate payload and propagated the shared proposal preimage through acceptance, standalone validation, and build/run authority; the full Task 3 suite passed.
- **REFACTOR:** No separate behavior-neutral refactor commit was needed.

## Verification

- Exact Plan 30 package and command gate passed twice on the committed implementation.
- The expanded `TestPlanCandidateSemanticIntegrity200`, candidate, candidate-route, planning-state, and plan-authority matrix passed, including valid and mutated standalone/accepted authority cases.
- `TestInsertPhase(Candidate|Immutable|Refuses)` and all `TestPlanningRoute` cases passed.
- `go vet ./pkg/colony ./cmd` and `git diff --check 5f8a7031..HEAD` passed.
- The implementation range contains exactly the nine declared source/test paths.
- Protected `.planning/config.json`, all `.gsd` content, Phase 199 patterns/evidence, and `200-VERIFICATION.md` remain untouched and byte-identical.

## Known Stubs

None. The added-line scan found no TODO, FIXME, placeholder, unavailable, or hardcoded empty-rendering stubs.

## Threat Model Outcome

- Route-provided semantic content is canonicalized and fully addressed before it becomes a candidate.
- Stored nested hashes, enclosing candidate hashes, derived back-references, and acceptance tokens are all treated as untrusted claims and recomputed before review or authority.
- No dependency, endpoint, authentication path, schema, or new file-access trust boundary was introduced.

## User Setup Required

None - no dependency, credential, service, or local configuration change is required.

## Next Phase Readiness

- The exact review payload is ready for Plan 31's independent acceptance-time impact re-derivation.
- Plan 30 has no scoped blocker. The known broad command-package timeout remains a pre-existing integration-runner condition.

## Self-Check: PASSED

- The summary and all nine declared source/test files exist.
- Task commits `dc2ad274`, `65f4034d`, and `5d40e6e5` are present in repository history.
- The implementation range contains exactly the nine owned paths and passes whitespace validation.
- Exact repeated suites, vet, stub, ownership, and protected-file checks all pass.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-09*
