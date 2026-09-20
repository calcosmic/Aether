---
phase: 203-biological-runtime
verified: 2026-09-14T00:30:00Z
status: passed
score: 10/10 requirements verified
behavior_unverified: 0
overrides_applied: 0
method: four-way parallel code review + full-suite run with execution-coverage assertion + pre-phase baseline diff
verifier_agent: none (workflow.verifier is false for this project; evidence is the review and the suite, both recorded below)
---

# Phase 203: Biological Runtime Verification Report

**Phase Goal:** Complete the real worker-discovery → governed recruitment → child result → knowledge transfer → downstream decision loop, then make pheromones operational decision inputs.
**Verified:** 2026-09-14
**Status:** passed, with two named limits disclosed below rather than hidden
**Re-verification:** No — initial verification

## How this was verified

This project runs with the verifier agent switched off (`workflow.verifier: false`),
so the evidence here is not a verifier's report. It is three independent things,
each reproducible:

1. **A four-way code review** of all 196 files the phase changed — recruitment
   core, pheromone bus and credit, live surfaces and wiring, lifecycle and
   documentation — partitioned so no reviewer's scope overlapped another's. Full
   per-finding evidence in `203-REVIEW.md` and its four part files.
2. **A full test-suite run whose coverage was itself asserted.** The headline
   `FULL-SUITE ... discovered=5305 executed=5305 lanes=59` was checked on every
   run reported here; no lane finished short. This matters more than usual: see
   "The gate that was not running" below.
3. **A pre-phase baseline diff.** Every failing test was re-run against a detached
   worktree at `1617ef3c`, the commit immediately before Phase 203 began, so
   "already broken" and "broken by this phase" are separated by evidence rather
   than by memory.

## Goal Achievement

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A worker can emit one versioned recruitment intent with authenticated parent/attempt identity, bounded objective, permissions, urgency, scope and cost limits (BIO-01) | ✓ VERIFIED | `cmd/recruitment_intent.go`; `validateRecruitmentIntent` refuses by name; intent recorded durably *before* any decision is taken. Plans 203-02, 203-03. Reviewed in `203-REVIEW-part-a.md` — the validate/record/decide ordering is real, not documented-only |
| 2 | Go validates depth, tree budget, cycles, duplicates, permissions, paths, cost and parent authority atomically before dispatch (BIO-02) | ✓ VERIFIED, one exemption disclosed | `cmd/recruitment_admission.go`; `TestOneAdmissionAuthority`, `TestRecruitmentTracerEndToEnd`, `TestSpawnCanSpawnDeniesPastDepthCap`, and the four dimension checks. **Both lanes now reach the same verdict** — the autopilot lane previously skipped permission, path, cost and duplicate entirely; fixed in `2e15417c` and locked by the rewritten `TestBothLanesUseOneReasonVocabulary`, proven able to fail by reverting the origin. Exemption: see Limit 1 |
| 3 | Each platform binds a native child or uses a bounded, process-tree-safe, repository-isolated fallback; unsupported nesting is reported, never faked (BIO-03) | ✓ VERIFIED | `cmd/recruitment_probe.go`, `cmd/recruitment_dispatch.go`, `pkg/codex/platform_dispatch.go`. Workspace containment and process-group-safe timeout confirmed correctly wired by review. Plans 203-02, 203-04, 203-15 |
| 4 | A child result binds child, intent, dispatch, execution generation, status, evidence, artifacts, handoff and usage exactly once; missing, duplicated, altered, timed-out or replayed results drive explicit recovery (BIO-04) | ✓ VERIFIED | `cmd/recruitment_result.go`, `cmd/recruitment_recovery.go`. Review confirmed the replay/conflict/generation logic is sound and its content-snapshot fields match its diff-check fields 1:1. Plans 203-02, 203-07 |
| 5 | A scoped packet carries the useful child result to an acknowledged parent continuation or one named follow-on consumer, which records the decision it made from it (BIO-05) | ✓ VERIFIED | `cmd/trophallaxis.go`. Plan 203-10 |
| 6 | Status, watch, recovery and cost/depth accounting include every governed descendant and follow-on edge (BIO-06) | ✓ VERIFIED | `cmd/recruitment_subtree.go`; `TestInlineRecruitLine`, `TestInlineRefusalLine`, `TestRecruitmentFamilyTree`, `TestFamilyTreeAndCostBlockAgree`. A paused project no longer shows a finished run's workers as live — fixed in `91dcb741` after the closing suite caught it. Plans 203-14, 203-15 |
| 7 | User, runtime, import and learning paths share one sanitizer, normalizer, deduplicating writer, provenance model and scope/expiry/strength resolver; cross-project input starts quarantined (BIO-07) | ✓ VERIFIED | `cmd/pheromone_resolver.go`, `cmd/exchange.go`; quarantine cannot be cleared by any ungoverned path (`TestNoUngovernedQuarantineClear`). Plans 203-05, 203-08 |
| 8 | Suggested notes support eight influence actions on an append-only history with a recorded actor; outcome-weighted strengthening, weakening and quarantine (BIO-08) | ✓ VERIFIED | **This was false for accept, edit and reject when the phase closed** — all three wrote no history entry and recorded no actor, and the test only checked the function and flag existed. Fixed in `d9331eb6`; `TestEveryDeclaredActionAppendsHistoryWithAnActor` now performs every declared action and reads the history back, and `TestPendingNoteActionsAppendRatherThanOverwriteHistory` proves the history is genuinely append-only. Tuning: `TestNoteStrengthTuningHelpfulNeutralHarmfulMovements`, `TestNoteStrengthTuningQuarantinesOnHarmfulThreshold`, `TestPinnedNoteIsNeverTuned`. Plans 203-13, 203-08, 203-11 |
| 9 | A contribution earns credit only when the runtime records the decision it changed and a later verified outcome, including neutral or harmful results (CEC-07) | ✓ VERIFIED | `cmd/agency_contract.go`, `cmd/recruitment_credit.go`. Plans 203-12, 203-13 |
| 10 | The biological mechanism study reconstructing prior emergence, help-seeking, delegation, return, trophallaxis and pheromone propagation, reconciled with modern Go authorization (SYNTH-05) | ✓ VERIFIED | `203-CLASSIC-SYNTHESIS.md`, registered into the versioned contract corpus. Plan 203-01. Confidence self-capped at medium in the document itself, because one source brief was a local-only file absent from the worktree — disclosed there, not silently substituted |
| 11 | Every helper is told the ability exists | ✓ VERIFIED | **This was true on one dispatch lane only when the phase closed.** The invitation reached the wrapper build lane and nothing else — not the direct lane autopilot uses, not the Codex platform, not the check lane's reviewers. Fixed in `fb25c625`; locked by `TestNativeBuildLaneWorkerIsToldHowToAskForHelp`, `TestContinueLaneWorkersAreToldHowToAskForHelp`, `TestCodexAgentDefinitionsCarryTheRecruitInvitation`, with `TestTheRecruitInstructionHasOneSource` now enumerating callers by name so a new lane cannot be added silently |
| 12 | An ordinary run that never asks for backup pays nothing | ✓ VERIFIED | `TestNoRecruitmentPathCostsNothing`, `TestNoNewMandatoryStep`, `TestTuningPassIsFreeWithoutCredit`. Independently read by the reviewer and judged real, not tautological: wrapped call counters, AST reachability over the actual build-dispatch files, and `os.Stat` file-existence checks — never elapsed time, never an assertion on a value the same code just set |
| 13 | Every claim written into CLAUDE.md about this phase names a test that exists and asserts it | ✓ VERIFIED | All 17 citations resolve to real, assertive test functions; spot-checked in depth for the admission-gate, depth-cap and cost-free claims. Both rule-file copies are byte-identical to each other and consistent with CLAUDE.md |

## The gate that was not running

The most consequential finding of this verification is not about Phase 203's code.

An unqualified `go test ./...` on this repository **reported a result after running
1635 of 5299 tests**, with 30 lanes never started. The `cmd` suite needs about 21
minutes; Go's default per-package timeout is 10. On hitting it, the suite's own
controller stops early and prints a complete-looking per-lane summary of the
fraction it finished — so a truncated run is indistinguishable from a clean one
unless the reader checks the `discovered=`/`executed=` headline.

This is the root cause of what `.planning/WINDOWS.md` had recorded three separate
times (entries 12, 16, 27) as "a test silently did not run". Every unqualified
gate run in this repository's history has been checking roughly a third of the
suite.

Closed: `make test` and CLAUDE.md's Verification Commands now both carry
`-timeout 90m`, with the discovered/executed check written down beside them.

## Test evidence

Final run, clean tree, no second checkout registered, nothing else touching the repo:

- `FULL-SUITE discovered=5305 executed=5305 lanes=59` — every lane ran everything.
- 17 of 18 packages pass. Only `cmd` fails.
- **Zero regressions against the pre-phase baseline.** The failing set is exactly
  the 16 tests already red at `1617ef3c`, unchanged.

Nine tests failed that were not red at the baseline. All nine were fixed, and each
fix was proved by watching the test fail without it and pass with it:

| Fixed | Cause | Commit |
|---|---|---|
| `TestStatusPausedColonyIgnoresStaleSpawnTreeWorkers` | 203-14's family tree was added to the status screen without consulting the liveness rule the Active Workers list already used, so a paused colony showed a previous session's worker as live | `91dcb741` |
| `TestAetherCorpusCatchesAnUnregisteredFlag` | 203-15 replaced `workers.md`'s `spawn-can-spawn` invocation with `recruit`, leaving the negative corpus test hunting a call the live corpus no longer holds — it refused to pass vacuously, which is what it was built to do | `5d870684` |
| `TestDeliberatelyDroppedDisplayChoicesStayDropped` | 203-14 put a retired phrase into a comment that quoted it in order to forbid it; the ratchet is a substring scan and cannot tell. Comment reworded, ratchet untouched | `da98f9a8` |
| `TestSkillIndexReadEmpty`, `TestSkillIsUserCreated`, `TestSkillIsUserCreatedShipped`, `TestVisualsDumpExportsCasteIdentityContract` (four failures, one cause) | `setupSanitizationTest` used `os.Setenv("AETHER_ROOT", …)`, which outlives `t.TempDir`'s cleanup, so every later test in the lane resolved the repository root to a deleted path | `d045056b` |
| `TestCLIFlagAudit` | Cobra registers `help` lazily, so whether the audit saw the whole command tree depended on lane composition. Pre-existing — fails identically at `702aee63` | `505309e1` |
| `TestOracleStatusFollowStreamsExistingRoundsAndExitsOnRunEnd` | `currentStreamingCommand` is set on every command execution and nothing reset it, so a test that ran a quiet command silenced `emitVisualLine` for the rest of the lane. Now restored by `saveGlobals`, and the test derives its own context from the real command rather than inheriting whatever ran before it | `f6dc3911`, `b0c966e9` |

## Known limits, disclosed not hidden

**Limit 1 — nothing proves who is calling.** Two authorization decisions are taken
on an unauthenticated, caller-supplied string: the delegation depth cap trusts a
`--parent` name matched against a fixed sentinel list (`cmd/spawn.go:22`), and the
owner-only note actions trust an `--actor` flag that defaults to `"owner"`
(`cmd/pheromone_mgmt.go:417`). The first pre-dates this phase (inherited from Phase
173) and CLAUDE.md has disclosed it openly since the phase was written; the second
was found by this verification's own review.

The owner decided on 2026-09-13 that these get their own design work rather than a
patch, because both close the same way and a narrow mechanism invented separately
at each call site would give two different answers to one question. Scheduled as
`.planning/todos/pending/2026-09-13-nothing-proves-who-is-calling.md`, with the
open questions and the acceptance proof written down. Tracked as WINDOWS entries
18 and 30.

**Limit 2 — two review castes cannot use the mechanism.** The security reviewer and
the quality reviewer are granted no shell, so they are deliberately never told to
run any command (`TestReviewSpecsDoNotInstructBashlessCastes`, after an earlier
incident where instructing a bashless caste to run a command blocked phase
advancement). CLAUDE.md now says so explicitly rather than claiming every helper
can ask. One inconsistency remains recorded, not resolved: the scout is granted no
shell either, yet does receive the invitation — one rule should decide all three
(WINDOWS entry 35).

## Outstanding from the review, not fixed

`203-REVIEW.md` carries 13 Warnings and 6 Info findings beyond the five Critical
ones. Three Criticals were fixed here, two are Limit 1 above. The Warnings are
recorded and none blocks the phase goal; the two most substantial are that
`aether status`'s Classic-voice guarantee still measures a renderer the real
command never calls, and that `renderPlanningStopVisual` remains a fully orphaned
renderer — both confirmed still true, both pre-dating Phase 203 (WINDOWS entries
22, 23).

One latent bug found at this gate is recorded rather than patched because the fix
turns on a behaviour question the owner should answer: a live-view snapshot resumed
from a checkpoint can carry a stale elapsed figure a full replay would not have,
because elapsed is only recomputed while an episode is open (WINDOWS entry 36). It
surfaces roughly one run in several, when seeded events straddle a one-second
boundary.

---

_Verified 2026-09-14. Method and evidence above are reproducible; no claim here
rests on a test that was not confirmed to have executed._
