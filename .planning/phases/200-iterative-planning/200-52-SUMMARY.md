---
phase: 200-iterative-planning
plan: 52
subsystem: iterative-planning-integrity
tags: [go, canonical-addressing, mutation-session, fallback-cleanup, command-errors, tdd]

# Dependency graph
requires:
  - phase: 200-28
    provides: repository-scoped read-through-commit planning mutation sessions
  - phase: 200-41
    provides: one-repository command-test binding and safe global cleanup
  - phase: 200-49
    provides: approved-specification and accepted-plan test authority
  - phase: 200-50
    provides: canonical downstream lifecycle fixtures and strict D-16 execution gates
provides:
  - canonical confidence, gap, semantic-delta, timeline, stage, and stop-policy fixtures
  - atomic force-replan cleanup with captured present-or-absent baselines and rollback proof
  - repository-bound discussion and exact rendered plan-acceptance failure coverage
  - deterministic bounded lineage for generated hard owner decisions
affects: [plan, discuss, force-replan, planning-confidence, planning-timeline, phase-200-final-gates]

# Tech tracking
tech-stack:
  added: []
  patterns: [production addressers in valid fixtures, mutate-then-readdress negative setup, declared cleanup baselines, domain-separated bounded lineage]

key-files:
  created: []
  modified:
    - cmd/codex_plan.go
    - cmd/codex_plan_finalize_test.go
    - cmd/codex_plan_test.go
    - cmd/composed_questions_test.go
    - cmd/discuss.go
    - cmd/planning_confidence_test.go
    - cmd/planning_state_test.go
    - cmd/planning_timeline_test.go
    - cmd/safety_invariant_test.go

key-decisions:
  - "Valid planning fixtures derive every nested identity through production Address* and canonical timeline helpers; a negative test re-addresses only setup it intentionally changed and leaves the field under test invalid."
  - "Force-replan captures each cleanup target's present-or-absent baseline before timestamp/source classification and commits all selected removals through the held planning mutation session."
  - "Generated decision lineage keeps established short identities byte-for-byte and uses a domain-separated digest only when the optional -hard companion would exceed the specification ID limit."
  - "Plan-only safety tests enter through approved specification and accepted plan authority while separately proving unchanged colony state and exact durable coordination journals."

patterns-established:
  - "Canonical fixture first: build valid nested planning evidence with the same addressers as production before applying a named conflict or tamper mutation."
  - "Cleanup observation is transaction evidence: capture the baseline before classifying whether a fallback artifact may be removed."

requirements-completed: [SYNTH-02, CEC-03, PLAN-01, PLAN-02, PLAN-03, PLAN-04, PLAN-05, PLAN-06]

# Metrics
duration: 83min
completed: 2026-09-09
---

# Phase 200 Plan 52: Canonical Planning Fixtures and Atomic Replan Cleanup Summary

**Force-replan now removes only stale fallback planning evidence in one rollback-safe transaction, while legacy confidence, timeline, stage, discussion, and plan-only tests exercise current canonical authority instead of copied hashes.**

## Performance

- **Duration:** 83 min
- **Started:** 2026-09-09T14:48:32Z
- **Completed:** 2026-09-09T16:11:44Z
- **Tasks:** 3
- **Files modified:** 9

## Accomplishments

- Made force-replan capture every fallback marker, plan, backup, Scout/Route-Setter projection, and stale phase-research baseline before classification, then remove the selected set through one planning mutation session with exact rollback coverage.
- Rebuilt stop-policy, confidence, gap, semantic-delta, iteration-card, timeline, and stage fixtures through production canonical addressers without weakening any validator or granting fallback output authority.
- Updated command tests to require a real nonzero rendered `plan --accept` error, exact `--accept-candidate` recovery guidance, zero state mutation, and a discussion round-trip bound to one physical repository.
- Bounded generated hard-decision specification lineage deterministically while preserving every established short ID and the REDIRECT/specification handoff behavior.
- Migrated the plan-only safety gate to approved specification and accepted-plan authority, retaining exact no-lifecycle-mutation checks and separately verifying the legitimate attempt/start-receipt journals.

## Task Commits

Each TDD gate and repair unit was committed atomically:

1. **Task 1: Rebuild canonical stop-policy history** - `5701deab` (test)
2. **Task 2 RED: Require complete transactional fallback cleanup** - `5a7525ff` (test)
3. **Task 2 GREEN: Capture fallback cleanup baselines** - `ffac1a1d` (feat)
4. **Task 3 RED: Align command failure and discussion fixtures** - `1a6b614b` (test)
5. **Task 3 GREEN / Rule 3: Bound generated hard-decision lineage** - `4f039565` (fix)
6. **Expanded gate RED: Expose stale plan-only authority and output reuse** - `348f9c0d` (test)
7. **Expanded gate GREEN: Migrate plan-only safety authority** - `2aec3215` (test)
8. **Expanded gate GREEN: Canonicalize shared timeline/stage fixtures** - `a04d7296` (test)
9. **Expanded gate GREEN: Canonicalize confidence fixtures** - `54723646` (test)

## Files Created/Modified

- `cmd/codex_plan.go` - Captures every fallback cleanup target through the held mutation session before source/timestamp classification.
- `cmd/codex_plan_finalize_test.go` - Uses canonical addressed stop histories and explicit preserved semantic changes.
- `cmd/codex_plan_test.go` - Proves exact cleanup membership, rollback, one-clock expiry classification, rendered acceptance failure, and zero mutation.
- `cmd/composed_questions_test.go` - Binds discussion to one repository and preserves hard-question, REDIRECT, and specification-decision assertions.
- `cmd/discuss.go` - Derives deterministic bounded lineage only for generated decision identities that cannot fit with their `-hard` companion.
- `cmd/planning_confidence_test.go` - Replaces forged assessment, gap, and delta hashes with production-derived identities while retaining each named invalid-input assertion.
- `cmd/planning_state_test.go` - Provides canonical shared iteration-card, assessment, gap, semantic-delta, authority-impact, and stop fixtures.
- `cmd/planning_timeline_test.go` - Re-addresses intentionally changed valid setup before round-trip and typed replay-conflict checks.
- `cmd/safety_invariant_test.go` - Rebinds nested commands, isolates output, uses accepted build authority, and distinguishes durable coordination from lifecycle mutation.

## Decisions Made

- Kept validators strict. Test fixtures now follow production addressing instead of copying plausible-looking hashes or assigning arbitrary IDs.
- Represented a semantically unchanged planning pass with an explicit `preserved` semantic change whose before/after hashes match; an empty delta is no longer treated as valid evidence.
- Kept repeated-gap identity honest: the underlying evidence and materiality remain stable across passes when a test needs to exercise stall detection.
- Preserved fallback planning as non-authoritative. Cleanup classification may retain newer worker-authored artifacts byte-for-byte, but it cannot elevate them into accepted planning authority.
- Preserved the existing generated lineage for short decision IDs. Hash compaction applies only when the base plus `-hard` cannot satisfy the 40-character specification lineage limit.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking Correctness] Bounded generated hard-decision lineage**

- **Found during:** Task 3 (discussion round-trip)
- **Issue:** A real hard composed question produced `owner-decision-<generated-id>-hard`, which exceeded the canonical 40-character specification lineage limit and prevented the intended REDIRECT/specification handoff.
- **Fix:** Added a narrow helper that preserves valid short lineage and otherwise derives a stable 16-hex-character suffix from domain-separated input. Both the binding-decision item and `-hard` companion remain deterministic and at most 40 characters.
- **Files modified:** `cmd/discuss.go`, `cmd/composed_questions_test.go`
- **Verification:** `TestDiscussSpecificationDecisionLineageBoundsGeneratedHardIDs`, `TestDiscussAddQuestionRoundTrip`, and `TestDiscussResolveHardConstraintEmitsRedirect` pass.
- **Committed in:** `1a6b614b`, `4f039565`

**2. [Rule 3 - Omitted Census] Repaired legacy planning fixture families named by the Plan 52 truth**

- **Found during:** Final plan-scoped planning selector after Tasks 1-3
- **Issue:** The plan's truth requires canonical assessment, gap, timeline, candidate, and acceptance hashes, but its file census omitted the shared helpers used by planning-confidence, timeline, stage-receipt, and Scout-stage tests. No later Plan 53-55 task owned those failures.
- **Fix:** Expanded the approved scope to `planning_state_test.go`, `planning_confidence_test.go`, and `planning_timeline_test.go`; built valid bases through production `Address*` functions and re-addressed only deliberate valid setup mutations before preserving each later conflict/tamper assertion.
- **Files modified:** `cmd/planning_state_test.go`, `cmd/planning_confidence_test.go`, `cmd/planning_timeline_test.go`, `cmd/codex_plan_finalize_test.go`
- **Verification:** All standalone `TestPlanningConfidence`, `TestPlanningTimeline`, `TestPlanningStage`, and `TestPlanningScoutStage` families pass.
- **Committed in:** `a04d7296`, `54723646`

**3. [Rule 1 - Legacy Fixture Bug] Restored the Plan 41 repository contract in the plan-only safety gate**

- **Found during:** Final plan-scoped planning selector
- **Issue:** Nested plan-only subtests reused output and did not rebind the same repository after global cleanup. Once that masking bug was exposed, the build fixture stopped at D-16 because it manufactured raw state without approved specification and accepted-plan authority.
- **Fix:** Rebound each command to its physical temporary repository, reset stdout/stderr independently, migrated build setup through the Plan 49 accepted-authority helper, and retained exact `COLONY_STATE.json` no-mutation plus durable journal checks.
- **Files modified:** `cmd/safety_invariant_test.go`
- **Verification:** `TestPlanOnlyUnchanged` passes independently and proves no dispatch-start event was recorded.
- **Committed in:** `348f9c0d`, `2aec3215`

---

**Total deviations:** 3 auto-fixed (2 blocking correctness/integration, 1 legacy fixture bug).

**Impact on plan:** The expanded test-only scope was required to satisfy the plan's declared canonical-fixture truth and left all production validators, accepted-plan gates, and newer worker-authored evidence protections intact.

## Issues Encountered

- The first broad planning selector took about seven minutes and exposed the omitted legacy fixture families. It was used once as baseline evidence; final verification used exact standalone families and did not repeat that broad selector.
- Canonically changing a gap's materiality also changes its content-addressed ID. The stalled-gap owner-decision fixture now marks the same underlying gap material across the complete repeated history instead of forging the old ID on only the final pass.
- Strict semantic-delta validation rejects an empty no-change receipt. Stop-history fixtures now include one explicit preserved phase node, which remains non-material to the stop policy.

## TDD Gate Compliance

- **Task 1 RED/GREEN:** The pre-existing exact gate failed on stale nested assessment hashes; `5701deab` rebuilt the test-only fixture through production addressers and reached the intended two-repeat stall assertion.
- **Task 2 RED (`5a7525ff`):** exact target-membership and injected-fault checks exposed uncaptured cleanup baselines. **GREEN (`ffac1a1d`):** every selected removal is read through the session before classification and the rollback test restores exact bytes, presence, mode, and digest.
- **Task 3 RED (`1a6b614b`):** modern command assertions reached a real generated hard-ID overflow. **GREEN (`4f039565`):** bounded lineage preserves both specification items and the original hard constraint flow.
- **Expanded gates:** `348f9c0d` deliberately exposed the plan-only root/authority failure; `2aec3215`, `a04d7296`, and `54723646` repaired the approved legacy test scope without changing validation policy.

## Verification

- Exact eight-test Plan 52 selector - PASS (`6.807s`).
- All `TestPlanningConfidence*` tests - PASS (`3.273s`).
- All `TestPlanningTimeline*` tests - PASS (`20.446s`).
- All `TestPlanningStage*` tests - PASS (`28.438s`).
- All `TestPlanningScoutStage*` tests - PASS (`34.680s`).
- `TestPlanOnlyUnchanged` - PASS (`12.610s`).
- `go vet ./cmd` - PASS.
- `gsd-sdk query verify.key-links .../200-52-PLAN.md --raw` - PASS (`2/2 valid`).
- `git diff --check` - PASS.

The repository-wide suite was intentionally not run. Plan 55 owns the single-run normal/race harness, and the parent orchestrator explicitly limited this plan's closeout to exact Plan 52 and formerly failing standalone families.

## Known Stubs

None. No placeholder, TODO, FIXME, hardcoded user-facing empty value, or mock runtime path was introduced.

## User Setup Required

None - no dependency, credential, external service, or configuration change is required.

## Next Phase Readiness

- Plan 53 can restore output-mode and visual parity against canonical planning evidence without inheriting any known unmapped confidence, timeline, stage, discussion, or plan-only failure from Plan 52.
- Force-replan now has exact target and rollback proof while preserving newer worker-authored Scout, Route-Setter, plan, and phase-research artifacts.
- No Plan 52 blocker remains; the unchanged full normal/race gate stays reserved for Plan 55 and the final Plan 40 receipt.

## Self-Check: PASSED

- All nine modified source/test files and this summary exist.
- TDD and repair commits `5701deab`, `5a7525ff`, `ffac1a1d`, `1a6b614b`, `4f039565`, `348f9c0d`, `2aec3215`, `a04d7296`, and `54723646` exist in repository history.
- Exact Plan 52 gates, every formerly failing standalone family, both declared key links, vet, and whitespace checks pass.
- STATE points to Plan 53, ROADMAP marks Plan 52 complete at 51/55, and all eight plan requirements remain checked complete.
- Protected `.planning/config.json`, `.gsd/`, and Phase 199 `199-PATTERNS.md` remain unstaged and untouched by Plan 52.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-09*
