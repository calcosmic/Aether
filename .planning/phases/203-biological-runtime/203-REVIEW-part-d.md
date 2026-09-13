---
phase: 203-biological-runtime
part: d
reviewed: 2026-09-13T21:28:26Z
depth: standard
files_reviewed: 34
files_reviewed_list:
  - cmd/codex_build.go
  - cmd/codex_build_finalize.go
  - cmd/codex_continue.go
  - cmd/codex_continue_finalize.go
  - cmd/codex_dispatch_contract.go
  - cmd/codex_workflow_cmds.go
  - cmd/codex_plan.go
  - cmd/codex_plan_finalize.go
  - cmd/codex_plan_stage_resume.go
  - cmd/planning_state.go
  - cmd/planning_delta.go
  - cmd/planning_visuals.go
  - cmd/planning_acceptance_rehearsal.go
  - cmd/plan_candidate.go
  - cmd/plan_revision.go
  - cmd/plan_print_brief.go
  - cmd/discuss.go
  - cmd/spec_cmd.go
  - cmd/seal_outcome.go
  - cmd/swarm_cmd.go
  - cmd/security_cmds.go
  - cmd/session_flow_cmds.go
  - cmd/context.go
  - cmd/init_cmd.go
  - cmd/phase_commit.go
  - cmd/lifecycle_facts.go
  - cmd/update_availability.go
  - cmd/visuals_config.go
  - CLAUDE.md
  - AGENTS.md
  - README.md
  - .codex/CODEX.md
  - .opencode/OPENCODE.md
  - .claude/rules/aether-colony.md
  - .aether/rules/aether-colony.md
  - .planning/REQUIREMENTS.md
  - .aether/version.json
  - npm/package.json
findings:
  critical: 0
  warning: 3
  info: 2
  total: 5
status: issues
---

# Phase 203 Part D: Code Review Report

**Reviewed:** 2026-09-13T21:28:26Z
**Depth:** standard
**Files Reviewed:** 34 (28 requested `cmd/*.go` source files + doc set; the actual Phase 203 diff for this scope touches 38 files, 2017 insertions / 418 deletions)
**Status:** issues_found

## Summary

Part D covers the plan/build/continue lifecycle command surface Phase 203 touched, plus the shipped "Biological Runtime" documentation claims. I read the full diff for every file in the required-reading list (`git diff 8c6b92578bc1baa510e2e98b3e8ca1ef28aef179^..HEAD`, filtered to this scope), traced the new `codex_plan_stage_resume.go` mechanism end-to-end into its caller in `codex_plan.go`, verified every test named in CLAUDE.md's "Biological Runtime" section exists and read the load-bearing cost-claim tests (`TestNoRecruitmentPathCostsNothing`, `TestNoNewMandatoryStep`, `TestTuningPassIsFreeWithoutCredit`), and checked for `--dry-run` mutation across every file in scope.

**Good news first, because it matters for calibration:** this is one of the more carefully-wired phases I've reviewed in this repo. The 17 tests CLAUDE.md's Biological Runtime section cites all exist by name, several genuinely drive real production code paths rather than fixtures shaped to pass, and the cost/no-regression tests use real counters (call counts on wrapped seams, AST reachability scans, file-existence checks) rather than elapsed-time or self-referential assertions. The new `codex_plan_stage_resume.go` mechanism is real, is called unconditionally from `runCodexPlanPlanOnly` before a fresh run can start, and its own regression tests drive the real Scout→Route-Setter coordinator rather than hand-built fixtures.

That said, I found three real gaps worth fixing and two structural limitations worth knowing about — none of them severe enough to block, but two are documentation-accuracy issues of exactly the kind this project's "Definition of Done" warns about.

## Documentation claim audit

CLAUDE.md's "## Biological Runtime (v1.28, Phase 203)" section cites 17 distinct test names. I confirmed each exists as `func <Name>(t *testing.T)` in package `cmd`, and spot-checked the ones underpinning the two highest-value claims (admission gate, cost-free ordinary path) for whether they assert what the prose says rather than something weaker.

| Claim | Named test(s) | Exists? | Asserts the claim? | Verdict |
|---|---|---|---|---|
| One admission gate decides every recruitment request | `TestOneAdmissionAuthority`, `TestRecruitmentTracerEndToEnd` | Yes (`cmd/recruitment_admission_test.go:916`, `cmd/recruitment_test.go:34`) | Yes — drives the real `dispatchRecruitment`/`spawnCanSpawnDecision` path | PASS |
| Hard cap of two hops | `TestSpawnCanSpawnDeniesPastDepthCap` | Yes (`cmd/spawn_enforce_test.go:478`) | Yes — drives `spawn-can-spawn 99 --enforce` against the real `spawnMaxDelegationDepth` constant (=2) | PASS |
| Cost/budget check | `TestRecruitmentAdmissionCostAllowsWhenWithinRemaining` | Yes (`cmd/recruitment_admission_test.go:273`) | Yes — real `recruitmentCostReason` against a seeded spawn tree | PASS |
| Permission/path checks | `TestRecruitmentAdmissionPermissionDeniesReadOnlyCaste`, `TestRecruitmentAdmissionPathDeniesOutsideColonyRoot` | Yes (lines 148, 174) | Yes | PASS |
| Duplicate-job check | `TestRecruitmentAdmissionDuplicateDeniesPendingSameSubtree` | Yes (line 308) | Yes | PASS |
| A "no" never stops the work; command still reports success | `TestRecruitmentTracerEndToEnd` | Yes | Consistent with the tracer's own doc comment | PASS |
| Inline recruit/refusal lines | `TestInlineRecruitLine`, `TestInlineRefusalLine` | Yes (`cmd/recruitment_subtree_test.go:328,384`) | Yes | PASS |
| Family tree with cost | `TestRecruitmentFamilyTree`, `TestFamilyTreeAndCostBlockAgree` | Yes (lines 486, 685) | Yes — `TestRecruitmentFamilyTree` seeds a real refusal and a real recruit and asserts both render | PASS |
| Note strength tuning (helpful/neutral/harmful) | `TestNoteStrengthTuningHelpfulNeutralHarmfulMovements`, `TestNoteStrengthTuningQuarantinesOnHarmfulThreshold` | Yes (`cmd/pheromone_outcome_test.go:160,224`) | Yes | PASS |
| Pinned note never tuned | `TestPinnedNoteIsNeverTuned` | Yes (line 369) | Yes | PASS |
| Ordinary run pays nothing (3-part claim) | `TestNoRecruitmentPathCostsNothing`, `TestNoNewMandatoryStep`, `TestTuningPassIsFreeWithoutCredit` | Yes (`cmd/recruitment_latency_test.go:106,238,320`) | Yes — see "Cost claim" analysis below | PASS |

**Verdict: all 17 citations resolve to real, assertive tests.** No fabricated or misdescribed citation found in this section.

One structural caveat on the ratchet itself: `cmd/claudemd_biological_runtime_test.go`'s `TestCLAUDEMDBiologicalRuntimeClaimsCiteLiveTests` (the guard that is supposed to keep this honest going forward) only checks that a cited name resolves to `func <Name>(` somewhere in package `cmd` — it does not check that the test's body actually asserts the specific claim placed beside it in the prose. A future edit could rename a test to something the reviewer would call a coincidental match, or could weaken a test's assertions while leaving its name (and the doc's citation) untouched, and this ratchet would not catch it. This mirrors a known, accepted limitation of the equivalent `TestCLAUDEMDCitesLiveTestsForCoherentJobClaims` pattern elsewhere in the repo, so I am not flagging it as a new regression — see IN-01 below for the record.

**Rule-copy parity:** `.claude/rules/aether-colony.md` and `.aether/rules/aether-colony.md`'s "## Biological Runtime (v1.28, Phase 203)" sections are byte-identical to each other (`diff` confirms), and both correctly summarize CLAUDE.md's fuller section without contradicting it, deferring to CLAUDE.md by name for "the tests that lock every claim above." Parity between the two copies is enforced by existing tests (`cmd/session_start_doc_removal_test.go`, `cmd/current_vocabulary_199_test.go`/`current_vocabulary_docs_199_test.go`), so drift here would already be caught. `.codex/CODEX.md` and `.opencode/OPENCODE.md` carry no equivalent section at all — this appears to be consistent with the existing pattern (neither carries the "Coherent Jobs" or "Team Check-In" sections either) rather than a new omission specific to this phase.

## Cost / no-regression claim audit

I read all three tests in `cmd/recruitment_latency_test.go` in full.

- `TestNoRecruitmentPathCostsNothing` (line 106) wraps the two real chokepoints (`recruitmentProbeRunner`, `spawnCanSpawnDecision`) with counting doubles that still call through to the original implementation, drives a real `aether build 1` against a single non-recruiting task, and asserts three real things: zero probe calls, zero admission-gate calls, and zero files written under the `recruitment`/`credit` store prefixes. It then runs a static AST reachability scan (`assertNoRecruitmentOrCreditSymbolsInBuildDispatch`) over the actual ordinary build-dispatch files (`codex_build.go`, `codex_build_worktree.go`, `codex_dispatch_contract.go`, `coherent_jobs.go`) to prove no code path in the plain build lane references a recruitment/credit symbol at all. This is a real, breakable measurement — not a mock returning a canned "0".
- `TestNoNewMandatoryStep` (line 238) proves three separate things structurally: (1) `decideBuildCheckin` has exactly one call site outside its own definition (a hard-coded literal count, so a second call site fails it), (2) `buildCheckinDecisionInput`'s field set is byte-identical to a recorded baseline (so a new field — the actual mechanism a future change would use to thread in a forced pause — fails it), and (3) the real, unmodified `decideBuildCheckin` still resolves the one-worker/no-recruitment scenario to the pre-existing fast-path constant. All three are genuinely falsifiable.
- `TestTuningPassIsFreeWithoutCredit` (line 320) runs the real `runPheromoneOutcomeTuning()` against a genuinely empty credit store and asserts by `os.Stat` (not by trusting the function's own report) that no bookkeeping file, pheromone file, or influence-history file was written.

**Verdict: these three tests are real and not tautological.** They measure call counts, AST reachability, and file existence — never elapsed time or an assertion on a variable the same function set moments earlier. I traced the wiring point in `cmd/codex_continue.go`'s `runCodexContinue` (`outcomeTuning := runPheromoneOutcomeTuning()`, called unconditionally right after hive promotion on both continue lanes) and confirmed it is non-blocking (its result is only attached to the summary, never returned as an error) and, per the test above, writes nothing when there is nothing to tune.

## Findings

### WR-01: Only build-dispatched workers are ever told about `aether recruit` — continue/check workers are not

**File:** `cmd/codex_build.go:4667-4733` (recruitment invitation appended in `composeBuildManifestBrief`); confirmed absent from `cmd/codex_continue.go` and every other continue-dispatch brief composer (no hits for `renderRecruitmentInvitation` or `aether recruit` outside `codex_build.go`, `live_events.go` comments, `next_action.go`, and `recruitment_*.go`).

**Issue:** CLAUDE.md's Biological Runtime section opens with "A helper working on a piece of the job can now ask the program for backup" — framed generically, with no build/continue distinction — and the underlying admission mechanism (`recruitmentAdmissionChecks`, `spawnCanSpawnDecision`) does not discriminate by workflow either. But `renderRecruitmentInvitation()` (the text that actually tells a worker the command exists and that a refusal is a normal, non-error outcome) is only ever emitted from `composeBuildManifestBrief`, confirmed as the invitation's single production caller by `cmd/recruitment_wiring_test.go`'s `TestTheRecruitInstructionHasOneSource`. A Watcher, Gatekeeper, Auditor, or Probe dispatched during `aether continue` never receives this text in its prompt, so in practice such a worker cannot discover the mechanism exists and will never invoke it — even though nothing in the admission gate itself would refuse it.

**Failure scenario:** A quality reviewer (Auditor) dispatched at `/ant-continue` gets stuck needing a second pair of eyes on a large diff. Per CLAUDE.md's framing, "a helper... can now ask the program for backup" — but this Auditor's prompt never mentions `aether recruit` at all, so it has no way to know the option exists, and silently does its best alone. This is the same "orphan" failure mode this exact phase spent five plans fixing for the build lane (commits `b729108c`, `b5067aeb`, `73dc4a3b`) — just not yet closed for the continue lane.

**Fix:** Either (a) call `renderRecruitmentInvitation()` from whatever composes the continue-dispatch brief (mirroring the build-lane wiring), and extend `TestEveryDispatchedWorkerIsToldHowToAskForHelp` to cover a continue-dispatched caste, or (b) if this scoping to build-only is intentional, narrow CLAUDE.md's claim to say so explicitly ("a helper dispatched during a build...") so the doc doesn't overstate scope the mechanism doesn't cover.

### WR-02: Retiring a plan candidate is recorded with the same failure reason as an expired one

**File:** `cmd/plan_revision.go:1182` (`retirePlanCandidateInSession`, `To: planningStageFailed, FailureReason: planningStageFailureCandidateExpired`); constant declared `cmd/planning_stage.go:19` as `"candidate_expired"`.

**Issue:** `retirePlanCandidateInSession` is new code (this phase) added specifically so an owner can deliberately reject a stale/undesired plan candidate (`cd19fe7a fix: let an owner retire a waiting plan candidate`). It deliberately mirrors `expirePlanCandidateInSession`'s transaction shape, but it also reuses that function's `FailureReason` value verbatim — there is no `planningStageFailureCandidateRejected` (or similar) constant in the codebase; `planningStageFailureCandidateExpired` is the only one that exists, used by both paths. The doc comment on the new function explicitly distinguishes the two cases ("retirement is the owner's own deliberate action," contrasted with expiry being "a side effect of an acceptance attempt") but the persisted data does not distinguish them at all — a candidate the owner explicitly retired via `aether plan --retire-candidate` is written to disk with the identical `"candidate_expired"` failure-reason string as one that simply timed out unattended.

**Failure scenario:** Any future feature that reads `stage-state.json`'s `failure_reason` field to explain to the owner why a candidate is gone (a status screen, an audit tool, a `/ant-history` entry) will report "expired" for candidates the owner actually chose to reject — the opposite of what happened. `cmd/plan_candidate_retire_test.go` contains no assertion on `FailureReason` at all, so this mislabeling is currently untested and would not be caught by a future refactor either.

**Fix:** Add a distinct `planningStageFailureCandidateRejected = "candidate_rejected"` constant and use it in `retirePlanCandidateInSession`, and add an assertion in `cmd/plan_candidate_retire_test.go` that the persisted stage state actually reports a rejection, not an expiry.

### WR-03: `Reason`-string collision between build-dispatch symbols and recruitment/credit symbols is trusted, not derived

**File:** `cmd/recruitment_latency_test.go:37-47` (`creditAndTuningSymbolsForLatencyScan`) and `cmd/recruitment_test.go:432-450` (`recruitmentSymbols`).

**Issue:** Both maps are hand-maintained literal lists of symbol names the AST scan treats as "recruitment/credit" identifiers. This is a reasonable pattern (the project uses it elsewhere per CLAUDE.md's reachability-ratchet precedent), but it means the guarantee `TestNoRecruitmentPathCostsNothing` provides is only as strong as these two lists staying exhaustive. A newly-added recruitment or credit helper function that isn't added to one of these two maps would silently escape the reachability scan even if it were wired into the plain build-dispatch path — the test would keep passing while the claim it backs quietly stopped being true. This is a lower-severity version of the same "citation exists but doesn't prove the full claim" pattern in IN-01 below, and is worth flagging precisely because this file's own comment (line 6-8) states "none of these tests can become flaky" as if completeness were structurally guaranteed — it is not; it is maintained by hand.

**Fix:** Not urgent, but consider deriving the symbol list from a shared prefix/package convention (e.g., every exported/unexported identifier in `recruitment_*.go`, `pheromone_outcome.go`, `phase_end_signals.go` via `go/ast` package scanning) rather than a hand-typed literal, so a new symbol is picked up automatically rather than requiring the author to remember to add it to two separate lists.

### IN-01: The doc-truthfulness ratchet only proves a cited test exists, not that it asserts the specific claim

**File:** `cmd/claudemd_biological_runtime_test.go:55-89` (`TestCLAUDEMDBiologicalRuntimeClaimsCiteLiveTests`).

**Issue:** As noted in the Documentation claim audit above, this test's check is `strings.Contains(pkgSources, "func "+name+"(")` — it proves the cited identifier is a real Go test function, but nothing ties the *content* of that function to the sentence beside its citation in CLAUDE.md. In every case I spot-checked in this review the cited test did assert the specific claim, so there's no live defect today — but the guard as written cannot detect a future edit that renames a test to something coincidentally matching, or waters down an existing test's assertions while its name (and the doc's citation) stay put.

**Fix:** No action required for this phase specifically (this mirrors an existing, accepted limitation of the sibling `TestCLAUDEMDCitesLiveTestsForCoherentJobClaims` pattern). Worth tracking as a shared, cross-cutting improvement — e.g., requiring each cited test's doc comment to literally quote (or hash-pin) the sentence it locks — rather than fixing per-phase.

### IN-02: `planningStageResumeWorkerSpec` silently falls through two lookup mechanisms with different filtering semantics

**File:** `cmd/codex_plan_stage_resume.go:294-303`.

**Issue:** `planningStageResumeWorkerSpec` first scans `planningWorkerSpecsForGoal("")` (goal argument intentionally ignored — confirmed at `cmd/codex_plan.go:2485`, `func planningWorkerSpecsForGoal(_ string) []planningWorkerSpec`), then falls back to `planningWorkerSpecForCaste(caste)`. Passing a literal empty string into a function whose signature still accepts (but ignores) a goal argument is a slightly awkward call shape — it works correctly today only because the goal parameter happens to be unused, and a future change that re-enables goal-based filtering in `planningWorkerSpecsForGoal` (e.g., to support goal-conditional castes) would silently start omitting valid resume castes for the empty-goal call from this file, with no test in `cmd/planning_stage_resume_203_test.go` covering that scenario directly (the existing tests only exercise the two current castes, Scout and Route-Setter, both of which are also reachable via the `planningWorkerSpecForCaste` fallback).

**Fix:** Low priority — either pass the real, resumed run's goal (available via `state.Goal` at the `buildPlanningStageResumeManifest` call site, one layer up) instead of a literal `""`, or add a short comment/test pinning the assumption that `planningWorkerSpecsForGoal` is goal-independent for the two staged-planning castes, so a future goal-filtering change to that function fails a Phase-203 test by name instead of silently breaking stage resume.

---

_Reviewed: 2026-09-13T21:28:26Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
