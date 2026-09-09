---
phase: 200-iterative-planning
plan: 54
subsystem: lifecycle-projection-and-regression-fixtures
tags: [go, lifecycle, next-action, golden-tests, accepted-plan, static-ratchets]

# Dependency graph
requires:
  - phase: 200-51
    provides: typed partial-build recovery and exact redispatch authority
  - phase: 200-52
    provides: canonical lifecycle facts and accepted-plan authority
  - phase: 200-53
    provides: accepted-authority build presentation coverage
  - phase: 200-38
    provides: in-session plan delegate rename reconciled by the command-advice ratchet
provides:
  - one resolver-owned lifecycle action shared by cards and JSON envelopes
  - canonical approved-specification and accepted-plan build/continue golden fixtures
  - reviewed golden output with stable bound requirement evidence
  - exact command-advice and colony-state writer inventories with no added debt
affects: [lifecycle-projection, build, continue, static-ratchets, phase-200-final-gates]

# Tech tracking
tech-stack:
  added: []
  patterns: [resolver candidate as lifecycle command authority, semantic assertions before snapshot refresh, receipt-backed golden fixtures, shrinking-only static inventories]

key-files:
  created: []
  modified:
    - cmd/lifecycle_projection.go
    - cmd/lifecycle_card_startup_test.go
    - cmd/lifecycle_card_workloop_test.go
    - cmd/golden_workflow_test.go
    - cmd/testdata/golden_build.txt
    - cmd/testdata/golden_continue.txt
    - cmd/testdata/next_action_hardcode.json
    - cmd/testdata/next_action_hardcode_baseline.json
    - cmd/testdata/colony_state_write_allowlist.json

key-decisions:
  - "Lifecycle projections select from the resolver candidate catalogue; renderers may project compatibility strings but never author command advice."
  - "Partial-build cards and envelopes consume the exact typed partial_recovery next action instead of reconstructing authority from loose legacy fields."
  - "Golden snapshots may be refreshed only after approved-specification, accepted-plan, receipt, terminal-attempt, and semantic output assertions pass."
  - "Static debt may move one-for-one or shrink; the live and frozen command-advice inventories remain byte-identical."

patterns-established:
  - "Compare the exact typed next_action carried by the runtime envelope with the card projection; a second resolver call is not proof of shared authority."
  - "Golden lifecycle tests must prove canonical authority and durable transitions before comparing presentation text."
  - "Regenerate inventory files from scanners, inspect every delta, and reject additions that merely make a ratchet green."

requirements-completed: [SYNTH-02, CEC-03, PLAN-01, PLAN-02, PLAN-03, PLAN-04, PLAN-05, PLAN-06]

# Metrics
duration: 38min
completed: 2026-09-09
---

# Phase 200 Plan 54: Shared Lifecycle Projection and Reviewed Baselines Summary

**Lifecycle cards and JSON now share resolver-issued action objects, while build/continue goldens and static debt inventories are grounded in canonical authority and audited rather than blindly refreshed.**

## Performance

- **Duration:** 38 min
- **Started:** 2026-09-09T16:51:50Z
- **Completed:** 2026-09-09T17:29:40Z
- **Tasks:** 3
- **Files modified:** 9

## Accomplishments

- Removed the duplicate command catalogue from `lifecycleProjectionDecision`; each projected next action and alternative now comes from the canonical resolver candidates in `next_action.go`.
- Strengthened startup/work-loop tests to decode and compare the exact typed action object issued in JSON, including Plan 51's typed partial redispatch recovery, instead of re-resolving or rebuilding an answer independently.
- Rebuilt build, continue, and state-mutation golden fixtures on approved specifications, accepted plans, canonical build-start receipts, terminal attempts, real claims, and the production plan-only/finalize boundary.
- Reviewed both textual golden changes after semantic assertions passed, including accepted-plan objectives, canonical task IDs, team-policy explanations, and bound requirement evidence.
- Moved one unchanged command-advice identity one-for-one and removed two retired direct colony-state writer waivers; no static debt was added.

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: Require resolver-issued lifecycle actions** - `7c97e81b` (test)
2. **Task 1 GREEN: Issue lifecycle actions from resolver candidates** - `c82b841f` (fix)
3. **Task 2 RED: Require canonical golden workflow authority** - `5c370d6b` (test)
4. **Task 2 GREEN: Refresh reviewed workflow goldens** - `048944f1` (test)
5. **Task 3: Reconcile lifecycle debt inventories** - `3e6196a3` (test)

## Files Created/Modified

- `cmd/lifecycle_projection.go` - Selects next actions and alternatives from canonical resolver candidate objects rather than hand-typed command strings.
- `cmd/lifecycle_card_startup_test.go` - Compares the actual JSON-issued action with the rendered card and rejects renderer-owned command literals.
- `cmd/lifecycle_card_workloop_test.go` - Exercises typed partial recovery and proves card/envelope action parity through the real result object.
- `cmd/golden_workflow_test.go` - Uses accepted authority, receipt-backed attempts, production finalization, semantic-first assertions, and stable embedded Go timing normalization.
- `cmd/testdata/golden_build.txt` - Records reviewed accepted-plan build output with canonical task identities and team-policy rationale.
- `cmd/testdata/golden_continue.txt` - Records reviewed bound requirement evidence in successful continue output.
- `cmd/testdata/next_action_hardcode_baseline.json` - Moves the unchanged plan delegate site to its current in-session function identity.
- `cmd/testdata/next_action_hardcode.json` - Keeps the regenerable live inventory byte-identical to the frozen baseline.
- `cmd/testdata/colony_state_write_allowlist.json` - Removes the retired build and plan `SaveJSON` exceptions.

## Decisions Made

- Kept legacy `next` and recovery-command strings only as compatibility projections, and asserted they derive from the typed action object rather than acting as authority.
- Used the accepted candidate's creation time as the fixed point for canonical continue setup. A historic absolute timestamp predating the candidate correctly fails closed as stale authority.
- Used a runtime-issued plan-only manifest plus the public `build-finalize` boundary for state-mutation coverage. Synthetic completion is deliberately not sufficient evidence for phase advancement.
- Normalized only the nondeterministic Go package duration embedded in otherwise semantic requirement evidence; the evidence wording and result remain fully asserted and snapshotted.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Stabilized package timing embedded in requirement evidence**
- **Found during:** Task 2 (golden review)
- **Issue:** Canonical accepted-plan fixtures exposed bound requirement evidence containing Go's per-run package duration, so an inspected golden changed again on the next identical run.
- **Fix:** Added a narrowly scoped normalizer for the duration inside successful `go test` evidence summaries.
- **Files modified:** `cmd/golden_workflow_test.go`
- **Verification:** The continue golden update and an immediate non-update rerun both passed.
- **Committed in:** `048944f1`

**2. [Rule 3 - Blocking] Refreshed the omitted live command-advice companion**
- **Found during:** Task 3 (static inventory reconciliation)
- **Issue:** The plan declared only the frozen baseline, but `TestNextActionHardcodeBaselineMatchesLive` requires the regenerable live file to be byte-identical; changing one alone would break Plan 55's full-suite gate.
- **Fix:** With orchestrator approval, regenerated `cmd/testdata/next_action_hardcode.json` from the scanner and verified it contains the same one-for-one rename and remains byte-identical to the baseline.
- **Files modified:** `cmd/testdata/next_action_hardcode.json`
- **Verification:** The live/baseline equality test and both declared ratchets pass together.
- **Committed in:** `3e6196a3`

**3. [Rule 1 - Bug] Preserved milestone progress and current operator handoff**
- **Found during:** Plan metadata closeout
- **Issue:** The state progress handler temporarily recalculated the already completed Phase 199 as incomplete, and the session handler does not rewrite the narrative operator-next-step line.
- **Fix:** Restored the proven one-of-seven completed-phase count and 14% milestone progress, then pointed the operator handoff at Plan 55 while retaining the SDK-updated plan count, timestamps, and state-head provenance.
- **Files modified:** `.planning/STATE.md`
- **Verification:** The final state diff preserves prior milestone facts, advances only Plan 54 to Plan 55, and names the correct next wave.
- **Committed in:** final plan metadata commit

---

**Total deviations:** 3 auto-fixed (2 Rule 1, 1 Rule 3)
**Impact on plan:** These fixes preserve deterministic tests, the existing inventory contract, and accurate planning provenance; none adds production surface or baseline debt.

## Issues Encountered

- The original Task 1 selector passed because the tests independently re-resolved next actions and partial recovery still reconstructed authority from loose fields. The RED commit changed the proof to compare the exact object carried across the boundary.
- The first canonical continue fixture used a historic timestamp earlier than its newly created accepted candidate. The runtime correctly rejected it as `clock_before_candidate_creation`; anchoring the attempt after candidate creation preserved deterministic valid authority.
- A synthetic build cannot advance through continue because simulated claims intentionally carry no implementation evidence. The state test now supplies a real external completion against the runtime-issued manifest and crosses the production finalizer instead.
- The new canonical golden output included bound requirement evidence absent from the legacy fixture. Each textual delta was inspected individually before the checked-in snapshots were accepted.

## TDD Gate Compliance

- **Task 1:** RED commit `7c97e81b` made the exact selector fail on the renderer-owned command catalogue; GREEN commit `c82b841f` routed every projection through resolver candidates.
- **Task 2:** RED commit `5c370d6b` established canonical authority and semantic checks while leaving both old snapshots failing; GREEN commit `048944f1` accepted only the two reviewed diffs after semantics passed.
- **Task 3:** The existing ratchets were RED with one new/stale identity pair and two stale writer entries. Commit `3e6196a3` moved the identity one-for-one, shrank the writer list, and made the declared plus live/baseline ratchets green.

## Verification

- Task 1 lifecycle selector - PASS (`30.444s` under concurrent verification load).
- Task 2 golden workflow selector - PASS (`29.833s` under concurrent verification load).
- Task 3 command-advice/live-baseline/writer ratchets - PASS (`2.309s`).
- Combined Plan 54 selector - PASS (`46.784s`).
- `go vet ./cmd` - PASS.
- `gsd-sdk query verify.key-links .../200-54-PLAN.md --raw` - PASS (`valid`).
- Plan commit-range `git diff --check` - PASS.
- Changed-file inventory - PASS: nine task files, including the approved live companion, and no unrelated source.

The repository-wide suite was intentionally not run. Plan 55 owns the bounded normal/race full-suite harness and depends on this plan.

## Known Stubs

None. No production placeholder, TODO, FIXME, unwired empty value, or mock lifecycle path was introduced. Empty collections in the golden harness are deliberate valid zero-state fixtures.

## User Setup Required

None - no dependency, credential, external service, or configuration change is required.

## Next Phase Readiness

- Plan 55 can exercise the full normal/race suite with lifecycle projection, canonical golden fixtures, and byte-identical static inventories already green.
- No scoped blocker or deferred production work remains.
- Protected `.planning/config.json`, `.gsd/`, and Phase 199 `199-PATTERNS.md` remain outside this plan and unstaged.

## Self-Check: PASSED

- All nine modified task files and this summary exist.
- Task commits `7c97e81b`, `c82b841f`, `5c370d6b`, `048944f1`, and `3e6196a3` exist in repository history.
- Exact task gates, the combined selector, live/baseline equality, vet, key links, and whitespace checks pass.
- No tracked file was deleted, and the approved inventory changes contain one identity move plus two waiver removals with no additions.
- STATE points to Plan 55, ROADMAP marks Plan 54 complete at 53/55, and shared requirements remain checked while Plan 55 is still outstanding.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-09*
