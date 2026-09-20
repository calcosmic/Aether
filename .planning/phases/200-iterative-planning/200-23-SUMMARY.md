---
phase: 200-iterative-planning
plan: 23
subsystem: testing
tags: [go, semantic-contract, integration, golden-tests, race-detector]

requires:
  - phase: 200-iterative-planning plans 17-22, 24, and 25
    provides: approved Specification lineage, staged Scout/Route planning, candidate authority, public wrappers, and lifecycle gates
provides:
  - closed executable corpus for SYN-200-01 through SYN-200-12 and all eight V-200 proof groups
  - provider-free real-repository proof from Discuss draft through exact Plan acceptance and execution eligibility
  - final targeted, migration, source-parity, full-suite, and race evidence with an honest inherited-blocker receipt
affects: [phase-205-product-acceptance, release-readiness, classic-contract, planning-lifecycle]

tech-stack:
  added: []
  patterns: [closed cross-referenced semantic corpus, real temporary-git integration fixture, authority-boundary golden tests]

key-files:
  created:
    - cmd/planning_public_paths_200_test.go
    - cmd/planning_real_repo_200_test.go
    - .planning/phases/200-iterative-planning/200-GATE-RECEIPT.md
  modified:
    - cmd/classic_contract_test.go
    - cmd/testdata/classic-contract/v1/schema.json
    - cmd/testdata/classic-contract/v1/mechanisms.json
    - cmd/testdata/classic-contract/v1/cases.json
    - .aether/schemas/completion-packet.schema.json
    - cmd/codex_plan_finalize_test.go
    - cmd/codex_visuals_test.go
    - cmd/lifecycle_card_startup_test.go
    - cmd/testdata/golden_plan.txt
    - cmd/next_action.go
    - cmd/codex_plan.go
    - cmd/spec_cmd.go
    - cmd/orchestrator_boundary_questions_test.go
    - cmd/orchestrator_boundary_guidance_test.go
    - cmd/command_audit_test.go
    - cmd/blackbox_harness_test.go
    - cmd/e2e_lifecycle_test.go
    - cmd/planning_visuals.go
    - cmd/planning_state.go
    - cmd/codex_plan_finalize.go
    - cmd/codex_build.go
    - cmd/codex_continue.go
    - cmd/advance_phase.go
    - cmd/testing_main_test.go
    - cmd/testdata/command_catalog.json
    - cmd/testdata/parity_snapshot.json
    - cmd/testdata/regression_snapshot.json

key-decisions:
  - "Keep the Phase 199 corpus byte-valid while making Phase 200 proof requirements conditional, additive, and closed under source/CAP/case references."
  - "Exercise production planning kernels in real temporary Git repositories without live providers or network access."
  - "Treat explicit Specification and preset authority boundaries as the current contract; migrate legacy immediate-dispatch fixtures instead of weakening those boundaries."
  - "Do not claim a green repository gate while the protected Phase 199 vocabulary inventory remains mismatched."
  - "All plan/spec command advice is selected from the shared next-action registry, including exact approval, repair, and preset boundaries."
  - "Fresh staged plans preserve Orchestrator questions before the one-Scout authorization and re-enter through the owner's exact preset after Discuss."
  - "Retire the obsolete plan-research-approve command instead of preserving an orphaned authority surface beside staged candidate acceptance."
  - "Mirror mutable lifecycle status into the active accepted revision while keeping its hashed plan definition immutable."

patterns-established:
  - "Causal proof cases carry before/after state hashes, artifact sets, dispatch counts, receipt chains, and structured-result assertions."
  - "Public planning projections are checked at width discontinuities and retain internal enums alongside exact human labels."
  - "Inherited baseline failures remain visible in the gate receipt and are not hidden by editing out-of-scope historical evidence."

requirements-completed: [SYNTH-02, CEC-03, PLAN-01, PLAN-02, PLAN-03, PLAN-04, PLAN-05, PLAN-06]

duration: 3h 15m
completed: 2026-09-08
---

# Phase 200 Plan 23: Final Semantic Proof Summary

**A closed 12-mechanism causal corpus and real staged-candidate repository journey now prove Phase 200 planning authority through build, continue, and seal, while the final receipt isolates the protected Phase 199 baseline.**

## Performance

- **Duration:** 3h 15m
- **Started:** 2026-09-08T02:11:32Z
- **Completed:** 2026-09-08T05:26:48Z
- **Tasks:** 3 executed
- **Files modified:** 35 implementation, corpus, test, golden, and receipt files

## Accomplishments

- Added exactly `SYN-200-01` through `SYN-200-12`, 16 paired executable cases, and all eight required `V-200-*` proof groups without invalidating the Phase 199 corpus.
- Proved the real repository journey from settled Discuss intent through draft/approval, two fresh-evidence Scout/Route passes, pending-candidate refusal, exact acceptance, and build/run eligibility.
- Proved later material evidence reaches Route before owner pause, the full card persists first, equivalent answers resume directly, and contract-affecting answers create a blocked successor draft.
- Verified Codex, Claude, and OpenCode public contracts, widths 47/48/63/64/95/96, no-color output, public stop labels, internal enums, and managed-wrapper source parity.
- Reconciled every Phase 200-owned full/race failure and recorded the four protected Phase 199 test nodes without altering their artifacts.
- Corrected the first receipt after an independent gate exposed seven missed failures, then proved the command-advice ratchet and staged Orchestrator boundary clean in focused, normal, and race runs.
- Corrected the second receipt after another independent gate exposed stale parity/regression goldens, an orphan command, legacy delegate fixtures, lifecycle status/revision drift, and test-order-dependent child deadlines.
- Final uncapped gates reached 9,550 passed / 4 failed / 11 skipped normally and 8,526 passed / 4 failed / 8 skipped under `-race`; all four failures are the protected Phase 199 baseline.

## Task Commits

Each task was committed atomically; the two TDD tasks retain separate red and green gates:

1. **Task 1 RED: define the Phase 200 corpus contract** — `bdc01985` (test)
2. **Task 1 GREEN: add executable Phase 200 contract corpus** — `2f0acd50` (feat)
3. **Task 2 RED: add failing public planning journey proof** — `b52f4ec7` (test)
4. **Task 2 GREEN: prove public planning journey** — `01b61ab3` (feat)
5. **Task 3: reconcile and record final planning gates** — `cf98a927` (test)
6. **Task 3 follow-up: centralize planning command advice** — `0a671f86` (fix)
7. **Task 3 follow-up: preserve staged Orchestrator boundary guidance** — `1d981d8b` (fix)
8. **Task 3 follow-up: retire obsolete research approval surface** — `db590537` (fix)
9. **Task 3 follow-up: migrate delegate proofs to staged planning** — `6952546c` (test)
10. **Task 3 follow-up: close preset-required Plan guidance** — `5629597a` (fix)
11. **Task 3 follow-up: preserve staged authority through lifecycle** — `00377491` (fix)
12. **Task 3 follow-up: refresh preset planning golden** — `99164f77` (test)
13. **Task 3 follow-up: preserve isolated coverage under full load** — `c431f0d3` (test)

**Plan metadata:** initially recorded by `0d69ae85`, corrected after the first independent gate by `f7d6e0a8`, and finalized by the documentation commit following the second independent gate.

## Files Created/Modified

- `cmd/testdata/classic-contract/v1/schema.json` — additive schema for Phase 199 plus strict Phase 200 proof records.
- `cmd/testdata/classic-contract/v1/mechanisms.json` — twelve synthesis decisions with source anchors, CAP links, invariants, and executable cases.
- `cmd/testdata/classic-contract/v1/cases.json` — 16 Phase 200 success/refusal cases added to the 86 existing Phase 199 cases.
- `cmd/classic_contract_test.go` — cross-reference, mutation, and production-kernel execution harness.
- `cmd/planning_public_paths_200_test.go` — command registration, wrapper parity, renderer width, label, enum, and source-check proof.
- `cmd/planning_real_repo_200_test.go` — real temporary-Git multi-pass planning, decision, acceptance, revision, build/run, and Seal-gate proof.
- `.aether/schemas/completion-packet.schema.json` — regenerated schema matching current Go completion structures.
- `cmd/ceremony_emitter_test.go`, `cmd/codex_plan_finalize_test.go`, `cmd/codex_visuals_test.go`, `cmd/colony_mode_manifest_test.go`, `cmd/e2e_regression_test.go` — migrated Specification, preset, staged-Scout, and no-write expectations.
- `cmd/golden_workflow_test.go`, `cmd/testdata/golden_plan.txt` — deterministic golden for the unbiased preset boundary.
- `cmd/isolated_process_test.go` — deadline-aware nested-process safety under the mandated race suite.
- `cmd/lifecycle_card_startup_test.go` — current Discuss-to-Specification and Plan-to-preset authority cards.
- `.planning/phases/200-iterative-planning/200-GATE-RECEIPT.md` — reproducible commands, versions, digests, totals, retries, and blocker evidence.
- `cmd/next_action.go`, `cmd/codex_plan.go`, `cmd/spec_cmd.go` — central candidate/gating path for exact Specification actions and unbiased/selected Plan preset advice.
- `cmd/orchestrator_boundary_questions_test.go`, `cmd/orchestrator_boundary_guidance_test.go` — approved-Specification, explicit-preset proof for fresh staged Orchestrator questions and re-entry.
- `cmd/command_audit_test.go`, `cmd/testdata/command_catalog.json`, `cmd/testdata/parity_snapshot.json`, `cmd/testdata/regression_snapshot.json` — retired the orphan `plan-research-approve` surface and regenerated command/lifecycle inventories.
- `cmd/blackbox_harness_test.go`, `cmd/e2e_lifecycle_test.go` — real compiled and downstream journeys now review and exactly accept staged candidates before build/continue/seal.
- `cmd/planning_state.go`, `cmd/codex_build.go`, `cmd/codex_continue.go`, `cmd/advance_phase.go` — keep mutable execution status aligned with the active accepted revision across process boundaries.
- `cmd/codex_plan_finalize.go` — makes the first replacement phase after a preserved completed prefix immediately ready.
- `cmd/planning_visuals.go`, `cmd/testing_main_test.go` — canonical preset-required lifecycle closeout and sufficient default parent budget for the enlarged integration suite.

## Decisions Made

- Phase 200 schema requirements are conditional on Phase 200 records, preserving existing Phase 199 data rather than weakening either contract.
- Corpus cases count only when they execute real planning/specification kernels and assert causal state/artifact/receipt facts; expected strings alone are insufficient.
- Integration fixtures use actual Git repositories and production transactions but deterministic local worker results, so CI does not depend on provider credentials or network availability.
- Legacy tests now stop at required authority boundaries instead of silently selecting a preset, skipping exact Specification approval, or claiming immediate two-worker dispatch.
- The receipt marks the repository-wide implementation gate blocked because both required full commands return nonzero, even though every remaining node belongs to the protected Phase 199 vocabulary/receipt baseline.
- Fresh plan-only execution materializes Orchestrator questions before wrapper dispatch, carries them in the one-Scout manifest, and returns the exact selected-preset command after Discuss.
- Current planning has one acceptance route: staged candidate review plus exact owner acceptance; the obsolete research-approval command is removed from Cobra and every generated inventory.
- Runtime lifecycle status is mirrored into the active revision snapshot so a fresh CLI process validates the same accepted definition and execution facts its predecessor wrote.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Migrated remaining immediate-planning fixtures and schema**

- **Found during:** Task 3 full repository gate
- **Issue:** Ceremony, finalizer, visual, colony-mode, and stuck-plan fixtures still assumed implicit planning bounds or immediate dispatch; the committed completion-packet schema lagged current Go structures.
- **Fix:** Added exact preset/Specification setup, replaced invalid 99/4 bounds with typed stop-policy proof, asserted staged Scout authority, migrated no-write preset output, and regenerated the schema.
- **Files modified:** `.aether/schemas/completion-packet.schema.json`, `cmd/ceremony_emitter_test.go`, `cmd/codex_plan_finalize_test.go`, `cmd/codex_visuals_test.go`, `cmd/colony_mode_manifest_test.go`, `cmd/e2e_regression_test.go`
- **Verification:** focused remediation gate 10/10 passed; final full/race sweeps report no failure in these files.
- **Committed in:** `cf98a927`

**2. [Rule 3 - Blocking] Replaced the retired Plan golden and bounded nested test launches**

- **Found during:** Task 3 race gate
- **Issue:** Test ordering exposed an old immediate-dispatch golden, while a nested-process self-test could start after too much of the parent package deadline had elapsed.
- **Fix:** Made the golden consume an explicit typed preset-required result and skip nested child launch only when less than 30 seconds remains, preserving the required exit cushion.
- **Files modified:** `cmd/golden_workflow_test.go`, `cmd/testdata/golden_plan.txt`, `cmd/isolated_process_test.go`
- **Verification:** dedicated race gate 2/2 passed; final race sweep reports no recurrence.
- **Committed in:** `cf98a927`

**3. [Rule 3 - Blocking] Migrated startup lifecycle cards to Phase 200 authority**

- **Found during:** Task 3 race gate
- **Issue:** Phase 197 fixtures still required the generic resolver card after Discuss and Plan, conflicting with Phase 200's exact draft-Specification and unbiased-preset screens.
- **Fix:** Retained the generic resolver checks for unaffected commands and added explicit Discuss/Plan visual and machine-result assertions for the new authority boundaries.
- **Files modified:** `cmd/lifecycle_card_startup_test.go`
- **Verification:** dedicated race gate 16/16 passed; final full/race sweeps report no recurrence.
- **Committed in:** `cf98a927`

**4. [Rule 1 - Bug] Routed new Plan and Specification advice through the shared resolver registry**

- **Found during:** Independent Wave 13 full gate after the initial Plan 23 summary
- **Issue:** Four new command-advice sites hand-typed exact Plan/Specification commands outside the Phase 197 next-action authority, tripping `TestNextActionNeverHardcoded`.
- **Fix:** Added exact approval, projection repair, preset, and refresh-preset candidates; gated them against the live command tree; folded preset-required output through the shared resolver.
- **Files modified:** `cmd/next_action.go`, `cmd/codex_plan.go`, `cmd/spec_cmd.go`
- **Verification:** command/spec/golden selection 17/17 passed; final normal/race sweeps report no ratchet failure.
- **Committed in:** `0a671f86`

**5. [Rule 1 - Bug] Restored Orchestrator decisions on fresh staged planning**

- **Found during:** Independent Wave 13 full gate after the initial Plan 23 summary
- **Issue:** The Phase 200 staged refactor left boundary materialization only on the existing-plan branch. Fresh Orchestrator manifests had nil questions, default-mode results returned nil instead of an empty typed slice, guidance disappeared, and the clarification test panicked.
- **Fix:** Shared boundary materialization across existing and fresh paths, persisted typed fields on the one-Scout manifest/result, returned exact preset re-entry after Discuss, and migrated the old whole-chain fixture through approved Specification and explicit-preset authority.
- **Files modified:** `cmd/codex_plan.go`, `cmd/next_action.go`, `cmd/orchestrator_boundary_questions_test.go`, `cmd/orchestrator_boundary_guidance_test.go`
- **Verification:** exact independent failure set 13/13 passed; broader boundary/planning selection 66/66 passed; final normal/race sweeps report no Phase 200 recurrence.
- **Committed in:** `1d981d8b`

**6. [Rule 3 - Blocking] Retired orphan approval authority and migrated delegate fixtures**

- **Found during:** Second independent Wave 13 full gate
- **Issue:** `plan-research-approve` remained registered without a supported caller, command/parity/regression goldens were stale, and plan-only/delegate fixtures still modeled the retired whole-chain authority.
- **Fix:** Removed the command through its Cobra source, regenerated all affected inventories, and moved delegate/capsule/wrapper assertions to the approved-Specification, explicit-preset, one-Scout staged contract.
- **Files modified:** `cmd/plan_revision.go`, `cmd/main.go`, `cmd/command_audit_test.go`, `cmd/safety_invariant_test.go`, `cmd/wire_capsule_198_2_test.go`, `cmd/territory_wrapper_contract_199_test.go`, `cmd/visual_wrapper_contract_test.go`, and three command snapshot JSON files.
- **Verification:** Exact second-independent selection passed 36/36.
- **Committed in:** `db590537`, `6952546c`

**7. [Rule 1 - Bug] Preserved accepted authority across build/continue process boundaries**

- **Found during:** Uncapped full-gate migration of compiled and downstream lifecycle journeys
- **Issue:** Candidate finalization left the first replacement after a completed prefix `pending`, while build/continue mutated live phase/task status without mirroring it into the active accepted revision. A fresh CLI process therefore rejected state written by the previous process.
- **Fix:** Started readiness after the preserved prefix and introduced a narrow execution-fact synchronizer for phase status, task status, and watcher failure counts; immutable plan-definition hashes remain unchanged.
- **Files modified:** `cmd/codex_plan_finalize.go`, `cmd/planning_state.go`, `cmd/codex_build.go`, `cmd/codex_continue.go`, `cmd/advance_phase.go`.
- **Verification:** Compiled install-to-seal and provider-backed revision journeys passed 2/2; the broader latent lifecycle set passed 16/16.
- **Committed in:** `00377491`

**8. [Rule 3 - Blocking] Migrated end-to-end fixtures to exact candidate acceptance**

- **Found during:** Uncapped full-gate migration after direct synthetic planning produced `legacy_invalid` authority
- **Issue:** The versioned-revision, provider, compiled-install, and downstream lifecycle fixtures activated plans through retired direct acceptance, which could not prove current Specification/candidate authority.
- **Fix:** Added deterministic staged Scout/Route helpers, exact candidate review/acceptance through the compiled CLI, semantic preservation/removal declarations, and real temporary-repository build/continue/seal coverage.
- **Files modified:** `cmd/planning_real_repo_200_test.go`, `cmd/blackbox_harness_test.go`, `cmd/e2e_lifecycle_test.go`.
- **Verification:** Named lifecycle set passed 16/16 and final full/race gates report no Phase 200 failure.
- **Committed in:** `00377491`

**9. [Rule 3 - Blocking] Kept the enlarged integration suite executable under full load**

- **Found during:** First uncapped post-repair full gate
- **Issue:** Phase 200 coverage pushed the sequential `cmd` package close to Go's stock ten-minute deadline. Tests intentionally deferred with `t.Parallel` then refused to launch isolated children because no safe exit cushion remained; all passed focused, proving a suite-order budget defect.
- **Fix:** Extended only the stock ten-minute parent-package timeout to thirty minutes in `TestMain`; explicit non-default and isolated-child budgets remain unchanged. Refreshed the Plan preset golden with its required shared lifecycle closing card.
- **Files modified:** `cmd/testing_main_test.go`, `cmd/testdata/golden_plan.txt`.
- **Verification:** Isolated-helper/golden set passed 15/15; final uncapped full and race gates execute the isolated tests and reach only the protected Phase 199 baseline.
- **Committed in:** `99164f77`, `c431f0d3`

---

**Total deviations:** 9 auto-fixed groups (six Rule 3 blockers and three Rule 1 bugs).
**Impact on plan:** All changes were required to execute the plan's full/race gates against the current Phase 200 contract. No runtime authority was widened and no external dependency was added.

## Issues Encountered

- The real-repository test initially encountered macOS `/var` versus `/private/var` path aliases and non-SHA fixture hashes. Canonical path resolution and valid deterministic hashes fixed both before the Task 2 green commit.
- The dispatch baseline was 5,840 passed, 28 failed, and 6 skipped before Plan 23 reconciliation began.
- The independent Wave 13 gate after the first summary reported 5,889 passed, 9 failed, and 6 skipped. Seven Phase 200 nodes exposed the command-advice and fresh-boundary defects fixed in `0a671f86` and `1d981d8b`.
- The earlier RTK evidence log was capped at 1,048,576 payload bytes, and the nil assertion panic terminated the `cmd` test binary; treating its compressed two-failure summary as exhaustive was incorrect. The independent named set now provides the missing explicit proof.
- The second independent gate reported 8,950 passed, 26 failed, and 8 skipped. It exposed stale parity/regression snapshots, an orphan command, legacy delegate fixtures, and a nil staged-manifest panic. The exact named selection passed 36/36 after `db590537` and `6952546c`.
- The first uncapped follow-up reported 9,523 passed, 15 failed, and 11 skipped. Eight isolated failures passed immediately when focused; three deterministic lifecycle failures led to the staged acceptance/status fixes in `00377491`.
- A later uncapped run reported 9,455 passed, 15 failed, and 11 skipped: the Plan closing golden was stale and ten isolated tests reached the parent package's stock deadline before launch. Their focused set passed 16/16, and `99164f77`/`c431f0d3` removed that order dependence without skipping coverage.
- Final normal gate: 9,550 passed, 4 failed, 11 skipped across 20 packages.
- Final race gate: 8,526 passed, 4 failed, 8 skipped across 20 packages, with no race detector diagnostics.
- The remaining protected Phase 199 baseline is exactly four nodes: `TestCurrentVocabulary199/tracked-occurrences-are-exhaustively-classified`, its parent `TestCurrentVocabulary199`, `TestPhase199GateReceiptSchema`, and `TestPhase199GateReceipt`. The first pair reports the two unclassified `legacy_pause`/`legacy_resume` UAT tokens; that stale Phase 199 evidence also invalidates its checked-in receipt. Plan 23 left all protected Phase 199 artifacts untouched.

## TDD Gate Compliance

- Task 1 RED commit `bdc01985` precedes GREEN commit `2f0acd50`.
- Task 2 RED commit `b52f4ec7` precedes GREEN commit `01b61ab3`.
- Final targeted results: 45 focused Phase 200 tests, 134 classic-contract tests, and 217 named migration tests passed.
- Independent correction results: 13/13 exact failing tests and 66/66 broader boundary/planning tests passed.
- Second independent correction results: 36/36 delegate/parity tests, 16/16 latent lifecycle tests, and 15/15 golden/isolated-helper tests passed.

## User Setup Required

None — tests are deterministic, local, provider-free, and network-free.

## Next Phase Readiness

- The Phase 200 semantic proof and real-repository lifecycle journey are ready for Phase 205 owner product acceptance.
- Before claiming a repository-wide green implementation gate, the owner of Phase 199 historical vocabulary evidence must reconcile the two UAT occurrences with its exhaustive inventory and rerun the full normal/race commands.
- Publication, deployment, and release remain separate owner-authorized workflows and are not claimed here.

## Self-Check: PASSED

- All created proof, receipt, and summary files exist.
- All thirteen Task 1 through Task 3 and follow-up commit hashes resolve in repository history.
- Summary formatting passes `git diff --check`.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-08*
