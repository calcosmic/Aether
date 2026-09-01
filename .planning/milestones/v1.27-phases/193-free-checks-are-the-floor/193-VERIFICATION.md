---
phase: 193-free-checks-are-the-floor
verified: 2026-08-22T16:40:48Z
status: passed
score: 4/4 must-haves verified
behavior_unverified: 0
overrides_applied: 0
deferred:
  - truth: "No single AI helper type is sent to check the same phase twice unless the Queen explicitly asks — for every reviewer caste, not just the Watcher."
    addressed_in: "Phase 194"
    evidence: "WINDOWS.md id 1 (recorded by 193-02): 'Probe/auditor/gatekeeper still legitimately double-dispatch on build AND continue with no explicit Queen proposal ... scoped to Phase 194 (moves the required-caste floor).' Phase 194's goal is explicitly 'Which AI helpers get sent to do a job becomes the Queen's judgement call' and success criterion 1 explicitly retires the old always-required-caste rule."
---

# Phase 193: Free Checks Are the Floor Verification Report

**Phase Goal:** The program's own automatic checks — does the code build, do the types check, does the linter pass, do the tests pass, do the files a worker claims to have created actually exist, and does each requirement have real evidence — run on every phase and can never be skipped. These checks become the safety floor, so a phase can safely finish even when no AI reviewer ("caste") was sent to check it by hand.

**Verified:** 2026-08-22T16:40:48Z
**Status:** passed
**Re-verification:** No — initial verification

## Plain English Summary

I did not just read the summaries — I compiled the actual program, ran the four tests the roadmap names by name, and also ran the deeper black-box tests that start the real `aether` program and drive it through a fake project, because those are the strongest possible proof (they don't take any code's word for it — they run the finished binary like a user would). Everything the roadmap said had to be true, is true, and I read the code that makes it true, not just the notes about it.

One thing is honestly incomplete, and the project's own paper trail already says so: the "checked only once" rule is fully proven for the AI reviewer that does general work-checking (the "Watcher"), which was the actual source of the "one bug fix costs 8 helpers" problem this phase set out to fix. But three more specialized reviewers (security, quality, and test-coverage) can still, in rare cases, get sent twice with nobody explicitly asking for that a second time. This was flagged by the team itself, is tracked in the project's own defect list, and is explicitly assigned to the very next phase (194) to close. It is not a surprise I found — it's a known, disclosed gap.

## Goal Achievement

### Observable Truths (Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Turning off every optional reviewer still leaves build/types/lint/tests/files-exist/evidence checks running; a test walks every skip path (`TestDeterministicChecksCannotBeSkipped`) | ✓ VERIFIED | Test exists at `cmd/floor_unskippable_test.go:329`. Ran it directly: all 10 named skip-path subtests pass (`skip-watchers`, light, heavy, explicit depth x3, caste proposal naming no reviewer, unavailable provider, host-boundary, consecutive-failure). Companion tests `TestExecutedCheckSetIsInvariantAcrossDepthAndProposal` and `TestFailingCheckStillBlocksOnEverySkipPath` also pass, proving the executed-check set doesn't merely run but stays identical and still blocks on a real failure across every path. |
| 2 | On a test project with zero reviewer helpers sent, a phase advances when the automatic checks pass and is stopped when they fail — proven in both directions through `continue` on a fixture project | ✓ VERIFIED | Ran `TestZeroReviewerPhaseAdvancesWhenFreeChecksPass` and `TestZeroReviewerPhaseIsBlockedWhenFreeChecksFail` (`cmd/blackbox_zero_reviewer_test.go`). Both build and run the real compiled `aether` binary (`go build -o ... ./cmd/aether`, then `exec.Command(h.binary, ...)`) against a fixture repo, run `build 1 --light` then `continue --skip-watchers`, and assert the actual on-disk colony state and verification report. Green-checks run: phase completes, watcher shows "skipped", all four steps ran and passed. Red-checks run (tests command fails): phase does not complete, blocking issue names "tests failed". Both PASS. |
| 3 | A requirement that today silently expects a Watcher to have run is satisfied by the automatic checks alone when no Watcher was sent, and `continue-finalize --reconcile-task` counts as real proof (`TestGateAcceptsDeterministicEvidenceWithoutReviewer`) | ✓ VERIFIED | Test exists at `cmd/floor_reviewer_free_gate_test.go:65`, ran it directly: both subtests pass ("all checks green: gate list passes, phase advances, no reviewer dispatched, both lanes" and "a failing check still blocks, both lanes"). Read `evaluateCriterionCheck`'s `case "watcher"` (`cmd/criterion_evidence.go:706-723`): when no reviewer was dispatched (or it was skipped), the check falls back to `deterministicFloorSatisfies(steps, claims)` rather than failing outright; a dispatched-and-failed watcher still returns a hard issue. The folded 2026-08-01 todo is closed: `continueTasksSupportAdvancement`'s H-04 branch (`cmd/codex_continue.go:2162-2193`) now requires only `task.Verified` for a reconciled task, and `codex_continue_finalize.go:52-64` merges `--reconcile-task` into the manifest on the wrapper lane too — confirmed passing via `TestFinalizeCountsReconcileTaskAsEvidence` and `TestReconcileIsNotABypass`, both green. |
| 4 | A real build followed by a real `continue` proves no single AI helper type is sent to check the same phase twice unless the Queen explicitly asks (`TestPhaseVerifiedOnce`) | ✓ VERIFIED (scope: Watcher caste only — see Deferred Items) | Test exists at `cmd/phase_verified_once_test.go:225`, ran it directly: all 5 subtests pass, including the row proving the test isn't measuring mere emptiness (an explicit watcher proposal correctly produces overlap). Read the dispatch gate directly: `cmd/codex_build.go:1230` gates the build-side verification-stage watcher dispatch on `queenAskedFor["watcher"]` (the Queen's literal proposal, `cmd/codex_build.go:1141`), not on the required-caste floor set. The test's own doc comment (lines 244-259) explicitly names and scopes what it does *not* cover: Probe/Auditor/Gatekeeper can still legitimately double-dispatch with no explicit proposal, a gap recorded in `.planning/WINDOWS.md` id 1 and explicitly assigned to Phase 194. This is the criterion exactly as the roadmap names it (citing this specific test), and that test is real and passing — but the plain-English wording of the criterion ("no single AI helper type") is broader than the one caste (Watcher) the shipped code and its own named test actually cover. See Deferred Items below. |

**Score:** 4/4 truths verified (0 present-but-behavior-unverified)

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | The full "verified once" guarantee for every reviewer caste (Probe, Auditor, Gatekeeper), not just the Watcher | Phase 194 | `.planning/WINDOWS.md` id 1, recorded by 193-02: "Probe/auditor/gatekeeper still legitimately double-dispatch on build AND continue with no explicit Queen proposal ... scoped to Phase 194 (moves the required-caste floor)." Phase 194's ROADMAP goal is explicitly "Which AI helpers get sent to do a job becomes the Queen's judgement call, not a fixed rule," and its success criterion 1 explicitly retires the old always-required-caste rule (`TestWatcherIsAlwaysRequiredOnBuild`). |

This was not discovered by this verification — it was disclosed by the executing team in the phase's own SUMMARY and the project's defect ledger, with an explicit next-phase owner. Because it is disclosed, matched to a specific concrete follow-up phase, and does not affect the mechanism this phase actually claims to deliver (the Watcher double-dispatch that produced the measured "8 workers for a 1-task fix" baseline), it does not block this phase.

### Required Artifacts (from PLAN frontmatter, all 5 plans)

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/deterministic_floor.go` | Shared floor body both continue lanes call | ✓ VERIFIED | 123 lines, real logic (build/types/lint/tests loop, claims, criterion evidence, D-07 scoping, D-01 warning text). Called from both `codex_continue.go:1585` (in-process) and `codex_continue_plan.go:268` (wrapper snapshot, itself called by `codex_continue_finalize.go:227`). No stub patterns found. |
| `cmd/deterministic_floor_test.go` | Floor tests | ✓ VERIFIED | `TestBothContinueLanesApplyTheSameFloor`, `TestDeterministicFloorIsTheOnlySourceOfAPass`, `TestDeterministicFloorStepOrderIsFixed` — all pass. |
| `cmd/blackbox_zero_reviewer_test.go` | End-to-end binary proof | ✓ VERIFIED | 4 tests, all run the real compiled binary, all pass (see Truth 2 above). |
| `cmd/floor_unskippable_test.go` | FLOOR-01 as a runnable command | ✓ VERIFIED | 4 tests, all pass. |
| `cmd/phase_verified_once_test.go` | FLOOR-04 no-double-dispatch invariant | ✓ VERIFIED | 5 tests, all pass (scoped to watcher caste as documented). |
| `cmd/floor_reviewer_free_gate_test.go` | FLOOR-03 gate + reconciliation + owner-confirmation | ✓ VERIFIED | 9 tests, all pass. |
| `cmd/criterion_owner_confirmation.go` | D-05 owner-confirmation state | ✓ VERIFIED | 118 lines. `criterionStateNeedsOwnerConfirmation` const, `ownerConfirmationSealBlockers` wired into `checkSealBlockers` (`cmd/codex_workflow_cmds.go:1097`), which is called from both `runSeal` (line 409) and `seal_final_review.go:372`. |
| `cmd/verification_scope.go` | D-07 targeted/full scoping | ✓ VERIFIED | Real scoping logic with plain-English `Reason` field; only the tests command is rewritten (build/type/lint untouched, documented rationale). |
| `cmd/verification_scope_test.go` | Scoping tests | ✓ VERIFIED | 4 tests, all pass. |
| `cmd/check_fix_attempt.go` | D-02/D-03 one bounded fix attempt | ✓ VERIFIED | 347 lines. Failure-index construction, single-attempt gating, append-only journal entry. |
| `cmd/floor_fix_attempt_test.go` | Fix-attempt tests | ✓ VERIFIED | 8 tests, all pass, including `TestNoSecondAutomaticFixAttempt` and `TestFixAttemptNeverOverwritesTheFirstResult`. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `syntheticCriterionRequirements` | `evaluateCriterionCheck` | criterion evidence pipeline | ✓ WIRED | `checks := []string{"claims", "watcher"}` in `cmd/criterion_evidence.go`; `watcher` no longer requires a live dispatch to pass — see Truth 3. |
| `runDeterministicFloor` | both continue lanes | `runCodexContinueVerification`, `runCodexContinueVerificationSnapshot` | ✓ WIRED | Confirmed by direct grep and read; `codex_continue_finalize.go` reaches the same body through `runCodexContinueVerificationSnapshot`, not a separate implementation. |
| `plannedBuildDispatchesWithJudgement` → `queenJudgement.Proposed` | build-side watcher dispatch | `queenAskedFor["watcher"]` gate | ✓ WIRED | `cmd/codex_build.go:1141,1230` — confirmed the dispatch reads the explicit proposal set, not the effective/required-caste set. |
| `runCodexBuildFinalize` | attempt journal report field | `attachBuildFreeCheckReport` | ✓ WIRED | `TestBuildFinalizeRecordsFreeChecksAsAReport` and `TestBuildFinalizeFreeChecksDoNotAdvanceThePhase` both pass — report differs between pass/fail runs, phase/task status does not. |
| `continue-finalize --reconcile-task` | `implementation_evidence` gate | `mergeReconcileTaskIDs` → H-04 branch | ✓ WIRED | Confirmed in code and by passing tests (see Truth 3). |
| worker `commands_run`/`changed_files` | criterion evidence | `reRunBuilderReportedEvidence` | ✓ WIRED | `TestBuilderReportedCommandIsReRunByTheProgram` and `TestEvidenceReRunNeverFabricatesAWorkerReceipt` both pass. |
| `needs_owner_confirmation` state | `aether seal` | `checkSealBlockers` → `ownerConfirmationSealBlockers` | ✓ WIRED | Confirmed in code; `TestSealBlocksOnUnconfirmedCriterion` and `TestSealForceStillOverridesOwnerConfirmation` both pass. |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| FLOOR-01 | 193-01, 193-03, 193-05 | Free checks run on every phase, unskippable | ✓ SATISFIED | `TestDeterministicChecksCannotBeSkipped` passes; 10-row skip-path table walked and confirmed real (see above). |
| FLOOR-02 | 193-01 | Zero-reviewer phase advances/blocks on free checks alone | ✓ SATISFIED | Both black-box directions proven through the real binary. |
| FLOOR-03 | 193-01, 193-04 | No implicit checker-worker requirement; reconciliation counts as evidence | ✓ SATISFIED | `TestGateAcceptsDeterministicEvidenceWithoutReviewer` passes; folded todo closed and tested. |
| FLOOR-04 | 193-02 | Phase verified once; no double-dispatch without explicit ask | ✓ SATISFIED (scoped) | `TestPhaseVerifiedOnce` passes for the Watcher caste, which is the caste the measured baseline (8 workers, 3 verification) was actually built on. Wider scope (Probe/Auditor/Gatekeeper) explicitly deferred to Phase 194, disclosed in WINDOWS.md. |

No orphaned requirements found — REQUIREMENTS.md lists exactly FLOOR-01..04 against Phase 193, and all four are claimed across the five plans.

### Anti-Patterns Found

None. Scanned all 13 files this phase created or modified in `cmd/` for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` and for "placeholder/coming soon/not yet implemented" language. The only hits were pre-existing, unrelated uses of the phrase "timeout placeholder" describing an established data-model concept (a worker result record awaiting a real outcome), not stub code. No empty-implementation or hardcoded-empty-data patterns found in the new files (`deterministic_floor.go`, `criterion_owner_confirmation.go`, `verification_scope.go`, `check_fix_attempt.go`).

`go build ./cmd/...` and `go vet ./cmd/...` both exit clean.

### Behavioral Spot-Checks / Probe Execution

Not applicable as a separate step — the phase's own must-haves are proven by full behavioral tests (the black-box binary tests are themselves the strongest form of spot-check), all of which were run directly rather than trusted from SUMMARY prose. No standalone `scripts/*/tests/probe-*.sh` files apply to this phase.

### Human Verification Required

None. Every roadmap success criterion resolves to a named, run test with real assertions (not a stubbed decision function), and the one place where the shipped scope is narrower than the criterion's plain-English wording (Truth 4) is already disclosed, evidenced, and assigned to a specific next phase — it does not need a fresh human judgment call, only awareness that Phase 194 is where it closes.

### Gaps Summary

No blocking gaps. One narrower-than-worded but disclosed and scheduled item (see Deferred Items) — the "verified once" guarantee is fully proven for the Watcher caste (the actual source of the measured 8-worker baseline this phase exists to fix) and is explicitly not yet extended to Probe/Auditor/Gatekeeper, which Phase 194 is scoped to close.

---

_Verified: 2026-08-22T16:40:48Z_
_Verifier: Claude (gsd-verifier)_

## Post-Verification Addendum (execute-phase orchestrator, 2026-08-22)

After this report was written, the phase's code review found and fixed six items in two fix/re-review iterations (commits `6c4e0af1`, `23063526`, `2933a215`, `bb7aeaaa`, `bb231a39`, `6c57f5da`): builder-reported commands are now allowlisted and run without a shell; the owner-confirmation command is shell-quoted; the fix-attempt planner reads the real manifest; owner-facing gate text is plain English; owner-confirmation seal blockers carry their own working recovery command; a root-level file change is labelled a full run. The final re-review reports `status: clean`. The full gate was re-run on the fixed tree: `go build ./...`, `go vet ./...` and `go test ./...` all exit 0 (18 packages ok). None of the four verified truths above changed; their named tests are part of that suite.
