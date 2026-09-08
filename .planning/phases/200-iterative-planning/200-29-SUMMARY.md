---
phase: 200-iterative-planning
plan: 29
subsystem: specification-authority
tags: [go, canonical-hashing, tamper-resistance, owner-authority, tdd]

requires:
  - phase: 200-28
    provides: "Repository mutation sessions and serialized planning persistence"
provides:
  - "One production canonicalizer for all nine typed specification sections, revision metadata, predecessor deltas, and approval receipts"
  - "Body-derived specification validation at persisted-state, planning, candidate-acceptance, build, and run authority boundaries"
  - "Zero-mutation refusal coverage for copied-hash, projection, receipt, and downstream authority tampering"
affects: [phase-200-verification, specification-lifecycle, planning-authority, build-authority, autopilot]

tech-stack:
  added: []
  patterns:
    - "Persisted hashes are treated as claims and compared against production-derived canonical preimages."
    - "Immutable revision material includes canonical UTC creation time; status and approval remain separately receipt-bound lifecycle authority."
    - "Repository-backed execution gates carry specification-integrity failures as explicit read-only authority refusals."

key-files:
  created:
    - cmd/specification_integrity_200_test.go
  modified:
    - pkg/colony/specification.go
    - pkg/colony/specification_test.go
    - cmd/specification.go
    - cmd/planning_state.go
    - cmd/plan_authority.go

key-decisions:
  - "Derive stable item IDs from their closed typed section and visible canonical semantic lineage, then include the exact ID in the item-content preimage."
  - "Bind created_at inside immutable revision content while keeping status and approval outside it; bind approval actor, time, token hash, revision ID, and revision hash inside the receipt ID."
  - "Keep structural validation for explicit legacy compatibility, but require full canonical recomputation for goal-addressed production lineages and every persisted specification load."

patterns-established:
  - "Canonical construction and validation call the same pkg/colony functions; tests do not carry a parallel hashing implementation."
  - "Build/run artifact loading reports a specification-integrity error before candidate or execution authority can become eligible."

requirements-completed: [PLAN-05, PLAN-06]

coverage:
  - id: D1
    description: "Every typed body value, stable item ID, item hash, scope, predecessor delta, immutable timestamp, revision hash, and revision ID is recomputed from canonical material."
    requirement: PLAN-05
    verification:
      - kind: integration
        ref: "cmd/specification_integrity_200_test.go#TestSpecificationIntegrity200"
        status: pass
      - kind: unit
        ref: "pkg/colony/specification_test.go#TestSpecWholeGoalRevisionRoundTripPreservesTypedBody"
        status: pass
    human_judgment: false
  - id: D2
    description: "Draft, scoped successor, supersession, explicit specification approval, and separate plan acceptance preserve D-09 through D-12 and D-16 authority semantics."
    requirement: PLAN-05
    verification:
      - kind: integration
        ref: "cmd/specification_integrity_200_test.go#TestSpecificationIntegrity200/draft_successor_and_approval_retain_separate_owner_authorities"
        status: pass
      - kind: unit
        ref: "pkg/colony/specification_test.go#TestSpecApprovalReceiptBindsExactRevisionAndHash"
        status: pass
    human_judgment: false
  - id: D3
    description: "Planning, candidate acceptance, build, and run reject typed-body or projection tampering before modifying state, stage artifacts, plan authority, build journals, or repository files."
    requirement: PLAN-06
    verification:
      - kind: integration
        ref: "cmd/specification_integrity_200_test.go#TestSpecificationIntegrity200"
        status: pass
      - kind: other
        ref: "go test ./cmd -run '^(TestSpecificationIntegrity200|TestSpecification.*|TestSpecProjection.*|TestSpecCommand.*|Test.*PlanAuthority.*|Test.*Specification.*Authority.*)' -count=1"
        status: pass
    human_judgment: false

duration: 21 min
completed: 2026-09-08
status: complete
---

# Phase 200 Plan 29: Canonical Specification Authority Summary

**Typed owner intent now earns authority only when production code can reproduce every item, revision, projection, token, and approval identity from canonical content.**

## Performance

- **Duration:** 21 min
- **Started:** 2026-09-08T21:14:58Z
- **Completed:** 2026-09-08T21:36:25Z
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments

- Centralized canonical normalization and hashing in `pkg/colony` for the nine typed specification sections, stable IDs, sorted evidence, scopes, predecessor deltas, UTC revision timestamps, revision IDs, owner tokens, and approval receipts.
- Rebuilt package fixtures with those production builders so positive tests begin from genuinely canonical objects and negative tests mutate one authority field at a time.
- Made persisted specification loads, planning validation, revision/approval writes, candidate acceptance, build, and Autopilot run fail closed on recomputation mismatch while retaining explicit specification-less legacy compatibility.
- Added an end-to-end tamper matrix proving copied hashes and a matching forged `SPEC.md` cannot bless edited owner intent, with identical before/after repository snapshots at every refusal boundary.

## Task Commits

1. **Task 1: Lock the complete specification tamper matrix** — `dfbcb169` (test)
2. **Task 2: Recompute typed revision and approval identities canonically** — `74a34cfb` (feat)
3. **Task 3: Require recomputation at every authority boundary** — `01cfaafe` (fix)

## Files Created/Modified

- `pkg/colony/specification.go` — Owns canonical item, scope, delta, revision, token, receipt, and full-lineage recomputation.
- `pkg/colony/specification_test.go` — Builds canonical positive fixtures through production identities and mutates only the field under test.
- `cmd/specification.go` — Delegates specification construction to shared canonical functions, preserves timestamp-bound exact replay, and rejects invalid loads/writes.
- `cmd/planning_state.go` — Adds the canonical validation seam while retaining structural legacy compatibility.
- `cmd/plan_authority.go` — Carries specification-integrity failures into repository-backed build/run refusals.
- `cmd/specification_integrity_200_test.go` — Covers every section, revision/receipt field, projection, lifecycle transition, downstream authority gate, and zero-mutation invariant.

## Decisions Made

- Canonical item IDs keep the normalized semantic lineage visible and add a section-bound digest; item hashes also bind that exact stable ID, closing ID substitution as well as body substitution.
- `created_at` is immutable revision content. Exact replay replaces a retry's new wall-clock observation with the already-persisted timestamp before comparing successor identity.
- Revision status and approval remain outside the immutable body hash so draft → approved → superseded transitions do not masquerade as content edits; the separate receipt cryptographically binds approval actor, time, deterministic token, revision ID, and revision hash.
- Old specification-less plans remain explicitly `legacy_unbound`. Structural compatibility for older synthetic/in-memory shapes does not relax the strict persisted-state loader or repository-backed planning/execution gates.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Preserved exact revision replay after timestamp binding**
- **Found during:** Task 2 focused command verification
- **Issue:** Adding `created_at` to immutable revision material made a harmless CLI retry derive a new revision because the command observed a later wall-clock time.
- **Fix:** When the named predecessor already has one current successor, replay construction uses that persisted successor timestamp before comparing the complete canonical identity; materially different operations still refuse as divergent.
- **Files modified:** `cmd/specification.go`
- **Verification:** `TestSpecCommandAddModifyRemoveAndExactReplay` and the complete Task 2/3 command suites pass.
- **Committed in:** `74a34cfb`

---

**Total deviations:** 1 auto-fixed bug
**Impact on plan:** The fix preserves both timestamp tamper detection and D-12 exact-replay semantics without widening file scope or weakening canonical validation.

## Issues Encountered

- The Task 1 RED commit failed to compile on the intentionally absent `validateCanonicalSpecificationState`, establishing the required fail-first gate.
- The first Task 2 integration run exposed timestamp-bound replay divergence and the intentionally unwired downstream authority checks; both were resolved within the declared source files.
- The known unrelated repository-wide `go test ./...` command-package timeout was not rerun. Plan 29's exact focused gates, three-run tamper repetition, and vet checks completed without assertion failures.

## TDD Gate Compliance

- **RED:** `dfbcb169` added the complete tamper/authority matrix; its focused gate failed because the canonical validator did not yet exist.
- **GREEN:** `74a34cfb` added shared production preimages, canonical validation, migrated fixtures, and exact timestamp-bound replay; the package and command specification suites passed.
- **INTEGRATION GREEN:** `01cfaafe` propagated strict validation into repository-backed build/run authority and pre-write approval/revision checks; the complete authority suite passed.
- **REFACTOR:** No separate behavior-neutral refactor commit was needed.

## Verification

- Exact Plan 29 package and command gate — passed twice on the committed implementation.
- `TestSpecificationIntegrity200` plus all package `TestSpec*` tests — passed three consecutive runs.
- `go vet ./pkg/colony ./cmd` and `git diff --check 9c994af5..01cfaafe` — passed.
- The implementation range contains exactly the six declared source/test paths.
- Protected `.planning/config.json`, `.gsd` content, Phase 199 patterns, and `200-VERIFICATION.md` remain untouched and byte-identical.

## Known Stubs

None. Empty assignments found by the scan are canonical hash-field clearing, empty delta initialization, or optional-state comparisons rather than rendered placeholders.

## User Setup Required

None - no dependency, credential, service, or local configuration change is required.

## Next Phase Readiness

- Canonical specification authority is ready for Plan 30 to bind the full semantic delta and review payload into candidate identity.
- The known broad command-package timeout remains an integration-runner condition; Plan 29's scoped authority evidence is green.

## Self-Check: PASSED

- The summary and all six declared source/test files exist.
- Task commits `dfbcb169`, `74a34cfb`, and `01cfaafe` are present in repository history.
- The implementation range contains exactly the six owned paths and passes whitespace validation.
- Exact, repeated, vet, stub, ownership, and protected-file checks all pass.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-08*
