---
phase: 204-learning-governor
verified: 2026-09-15T00:00:00Z
status: gaps_found
score: 9/11 sub-truths fully verified (2 narrowed, not fully closed)
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: "5/11 sub-truths fully verified (per previous body text; previous frontmatter said 5/12, which does not match its own 11-item truths list -- carried forward here only for continuity, not repeated)"
  gaps_closed:
    - "SC3b: all six lifecycle lanes (build, continue-native, continue-delegate, plan, oracle, swarm, recovery) now open and close a durable episode with zero skipped subtests in TestEveryLifecycleLaneWritesADurableOutcome (204-13); the code review additionally found and fixed a seventh gap in swarm's own delegate/finalize lane (CR-03) that 204-13/204-15's own sweep had missed"
    - "SC4b: shadowEvaluator is now a real per-fixture classifier (shadowClassifyAgainstBank) that provably distinguishes a beneficial candidate from a harmful/overfit one through the real production entrypoint (204-12), confirmed by an independent mutation test performed in this verification session"
    - "SC5b: runAutomaticImprovementPass drives a declared candidate through comparison, gate admission, canary start, and completion/rollback, reached from both check lanes through one call site inside runPhaseEndConsolidation (204-12/204-15), confirmed by an independent mutation test performed in this verification session"
    - "SC5c: buildImprovementReport now shows real, non-zero verified-success and preventable-intervention figures over episodes real production lanes actually wrote (204-15), no longer downstream-blocked by SC3a"
    - "SC5d: proposeSourceImprovement has a real, evidence-gated automatic caller (triggerRepeatedInterventionProposal, 3+ distinct episodes of the same intervention kind) reached from the same phase-end boundary both check lanes reach (204-16); the code review found and fixed a serious operational risk in this same new code (CR-04: unattended git checkout on the live working tree) by moving all git mutation into an isolated temporary worktree"
  gaps_remaining:
    - "SC3a (narrowed): 7 of the 9 originally writerless episode fields, plus interventions, now have real production writers on every lane that holds the fact -- but `usage`/`reported_cost_usd` remain absent specifically on both check lanes (native and delegate continue), a deliberate, documented scope boundary (204-15's own Deviations section), not a silent gap"
    - "SC3c (narrowed): the fixture-bank guard count nearly tripled (7 -> 21 of 51, was 40 unguarded, now 30 unguarded) and the floor is now a genuine two-sided ratchet that cannot be inflated -- but 30 of 51 (59%) of this project's own confirmed incidents still carry no guarding test"
  regressions: []
must_haves:
  truths:
    - "SC1: A cited mechanism study reconstructs Classic-era learning mechanics, audits every current store, and justifies the smallest provable modern architecture."
    - "SC2: Production writers/readers agree on versioned memory schemas; hypotheses/disproven entries never render as verified; promoted material retains provenance/outcome lineage."
    - "SC3a: Every started episode records immutable outcome, evidence, hard-gate, changed-decision, intervention, time, token, and cost."
    - "SC3b: Every lifecycle lane (build, continue, swarm, oracle, recovery, plan) opens and closes a durable episode record."
    - "SC3c: Confirmed incidents become a versioned regression-fixture bank actually guarded by real tests, with hidden holdouts and budgeted test tiers."
    - "SC4a: The shadow-comparison isolation mechanism (candidate cannot reach/alter its evaluator) is structurally proven."
    - "SC4b: A beneficial candidate is actually distinguishable from a harmful/overfit one by a real grading function, reachable in production."
    - "SC5a: The retained-authority boundary (nine categories) structurally cannot be bypassed by the canary-promotable path."
    - "SC5b: A promotable, beneficial candidate can actually be promoted into a canary and atomically rolled back by the running program."
    - "SC5c: Verified-useful-task success and preventable interventions are reported as two honest, never-blended figures, usable on a live colony."
    - "SC5d: A source-code improvement proposal can actually be created by the running program, and structurally cannot self-approve/merge/publish/deploy."
  artifacts: []
  key_links: []
gaps:
  - truth: "SC3a: Every started episode records immutable outcome, evidence, hard-gate, changed-decision, intervention, time, token, and cost"
    status: partial
    reason: "204-15 gave real production writers to 7 of the 9 originally writerless fields (evidence_ids, hard_gate_results, changed_decision_ids, episode_revision, acceptance_digest, evaluator_digest on the build and both check lanes; interventions already had writers from 204-13) plus registered the episode ledger as the seventh memory-schema census store, all confirmed passing in this session. But usage/reported_cost_usd remain honestly absent on BOTH check lanes (native runCodexContinue and delegate runCodexContinueFinalize) -- a watcher/reviewer worker's token usage is computed after the episode-close defer is already registered and is not surfaced to it. 204-15-SUMMARY.md documents this as a deliberate, time-boxed scope boundary, not a silent gap, and no acceptance criterion this plan declared required it. The literal SC3a wording ('every started episode records... time, token, and cost') is therefore still not true for check-lane episodes."
    artifacts:
      - path: "cmd/codex_continue.go"
        issue: "checkEpisodeCloseRecord never populates Usage/ReportedCostUSD; both stay nil on the native check lane's closed episode"
      - path: "cmd/codex_continue_finalize.go"
        issue: "same shared helper, same gap, on the delegate check lane"
    missing:
      - "Restructure runCodexContinue's ~500-line function to pre-declare a mutable usage accumulator the already-registered episode-close defer can read, surfacing the watcher/reviewer workers' own Usage figures the same way the build and swarm lanes already do."
  - truth: "SC3c: Confirmed incidents become a versioned regression-fixture bank actually guarded by real tests"
    status: partial
    reason: "204-14 guarded 10 more fixtures (7 -> 17 of 47 at the time) with real, individually-verified tests, refusing to guard any fixture whose incident is not genuinely fixed today (a documented, deliberate shortfall from the plan's own 'at least 12' target, held to CLAUDE.md's honesty bar). The floor (seedBankUnguardedFloor) is now a genuine two-sided ratchet -- TestSeedBankUnguardedFloorIsTheRealCount fails if the constant is ever raised above the real count, not only if the real count exceeds it. The bank subsequently grew to 51 fixtures (5 new ones seeded by the code-review fix pass's own now-fixed defects), landing at 21 guarded / 30 unguarded at this HEAD, confirmed by direct read in this session. 30 of 51 (59%) of this project's own confirmed incidents still carry no guarding test -- REQUIREMENTS.md itself leaves LEARN-05 honestly unticked for exactly this reason, and D-09 explicitly scoped this closure to shrink the floor honestly, not to reach zero."
    artifacts:
      - path: "cmd/testdata/fixture-bank/v1/bank.json"
        issue: "30 of 51 fixtures have a null guard field (confirmed by direct JSON read at this HEAD, not carried forward from an earlier document)"
    missing:
      - "Name and confirm a real guard test for more of the 30 remaining fixtures' own confirmed incidents, shrinking the recorded floor further -- the floor may only shrink, never widen."
requirements_discrepancy: []
documentation_staleness:
  - "REQUIREMENTS.md's LEARN-05 line and phase-204 traceability row, and WINDOWS.md entries 42/45, cite the fixture bank as '52 total / 22 guarded'. The real, current bank (confirmed by direct JSON read at this HEAD) is 51 total / 21 guarded / 30 unguarded. This is an off-by-one left over from the code-review fix pass: CR-01's fix regenerated the fixture bank via TestSeededBankUpdate, which removed the one fixture that had been seeded for WINDOWS entry 44 the moment that entry was honestly reopened (only status:fixed entries become regression fixtures) -- but the prose in REQUIREMENTS.md/WINDOWS.md describing the bank's size was not refreshed after that regeneration. Every test that reads the bank reads it live and is unaffected (TestEveryFixtureNamesItsGuardOrIsCountedUnguarded, TestSeedBankUnguardedFloorIsTheRealCount, TestSeedBankIndexTotalsAgree all pass against the real 51/21/30 numbers); this is purely a stale prose figure in two truth-telling documents whose entire purpose this closure was to make accurate. Non-blocking, but worth a one-line correction."
  - "WINDOWS.md's own frontmatter header (open_count: 32, fixed_count: 15, total_count: 47) does not match the table/JSON body it summarizes, which independently and consistently count to open:33, fixed:14, total:47 (verified by parsing both the markdown table and the JSON ledger block in this session -- the table and JSON agree perfectly with EACH OTHER, confirming 204-16's own claim that 'the markdown table and the JSON ledger agree'; only the separate summary header counts are off by one in each direction). Likely stale from entry 47 being appended (a pre-existing, phase-204-adjacent but NOT phase-204-caused swarm-worker-naming collision, found at the gap-closure wave-3 gate) without recomputing the header. Non-blocking, cosmetic."
---

# Phase 204: Learning Governor Verification Report

**Phase Goal:** Make memory truthful and prove behavior changes through immutable outcomes, permanent evaluations, independent shadow comparison, bounded promotion, and rollback.
**Verified:** 2026-09-15
**Status:** gaps_found
**Re-verification:** Yes — after gap-closure plans 204-12..204-16, plus a code-review-and-fix pass (204-REVIEW.md / 204-REVIEW-FIX.md) that found and fixed four additional Critical bugs in the closure's own new code.

## Summary for the owner (plain English)

Since the last check, five of the seven gaps I found are now genuinely fixed — not just "the code exists," but proven with a test that fails when the wiring is removed, which I independently re-ran and, for the two highest-stakes ones, personally broke and watched fail before restoring them myself.

1. **Every kind of colony activity now leaves a permanent record.** Build, both flavors of check, planning, research, bug-swarms, and recovery all open and close a durable record — I ran the test myself and watched all six pass with zero shortcuts.
2. **The "does a new idea actually work" checker now really checks.** It used to always say "fine" no matter what. Now it genuinely tells a good idea from a bad one — I broke it myself (put the old always-say-fine code back) and watched the test catch it, then restored it.
3. **The "try an idea safely, then keep or undo it automatically" system now actually runs**, at the end of every check, not just as unused code sitting on a shelf — again, I broke the connection myself and watched it fail, then restored it.
4. **The two honest scorecards — "how often did the colony's own suggestions actually help" and "how often did you have to step in" — now show real numbers** from real activity, instead of always reading zero.
5. **The colony can now write up a proposed code change by itself** when the same problem keeps recurring, and — this is the important part — while reviewing this work I found a real safety bug in that brand-new code: it could have accidentally left your whole project checked out on the wrong Git branch if something went wrong partway through. That bug is now fixed properly (the risky part now happens in an isolated, disposable copy, never on your actual working files), and I confirmed the fix with a test that proves it.

Two things are honestly still incomplete, and both are named plainly in the project's own paperwork rather than hidden:

- The permanent record for a **check** (as opposed to a build) is missing two pieces — how many tokens it used and what it cost — because reaching that number would have meant a much bigger rewrite than this round of work budgeted for. Not fabricated as a fake zero; genuinely left blank.
- Of this project's 51 confirmed past mistakes, only 21 now have a real automated test guaranteeing they can never silently happen again. That's up from 7, and the recorded target number can now only get stricter, never looser — but 30 of them are still unguarded.

I also want to flag something my own review of this closure's new code caught, because it's a good example of exactly the failure mode this whole project exists to fix: a piece of code meant to automatically promote a lesson from "unproven guess" to "proven true" was built, tested, and looked complete — but it can never actually promote anything in the real running program, because two separate parts of the system refer to the same kind of record using two different ID schemes that never connect. Rather than fake a connection, the team wrote the honest limitation directly into the project's own defect log and into this project's main documentation, and I independently confirmed that's the accurate state of things today.

## Goal Achievement

### Observable Truths (mapped to ROADMAP success criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| SC1 | Cited mechanism study reconstructs Classic learning mechanics, audits every store, justifies smallest architecture | ✓ VERIFIED | `204-CLASSIC-SYNTHESIS.md`/`204-CLASSIC-HISTORICAL-EVIDENCE.md` exist; unchanged since the last check. Not re-touched by this closure. |
| SC2 | Versioned memory schemas; hypothesis never rendered as verified; provenance/lineage on promoted material | ✓ VERIFIED | Unchanged since the last check; independently re-confirmed passing this session. See also the CR-01 note below — a *new* mechanism this closure added (hypothesis-to-validated promotion) is honestly disclosed as unable to fire in production, but this does not reduce SC2's original scope (hypotheses genuinely never render as verified, provenance is genuinely retained). |
| SC3a | Every started episode records outcome, evidence, hard-gate, changed-decision, intervention, time, token, cost | ⚠ PARTIAL | 7 of 9 fields now write on every lane that holds the fact, confirmed by 5 independent FAILS-WHEN-UNWIRED mutations (build, native check, delegate check, census, live-report) all re-run in this session. `usage`/`reported_cost_usd` remain absent on BOTH check lanes, an honestly-documented scope boundary, not a fabricated zero. Not fully true of the running program yet. |
| SC3b | Every lifecycle lane writes a durable episode | ✓ VERIFIED | `TestEveryLifecycleLaneWritesADurableOutcome` re-run live this session: 0 skips across all 6 lanes (swarm, build, continue, plan, recovery, oracle) — PASS. The code review additionally found and fixed a 7th gap (swarm's own delegate/finalize lane, CR-03), confirmed present in this session's `git log` and its own new test passing. |
| SC3c | Confirmed incidents become a guarded, versioned regression-fixture bank with hidden holdouts and budgeted gates | ⚠ PARTIAL | Gate/holdout architecture unchanged, still VERIFIED. Bank guard count nearly tripled (7→21 of 51, confirmed by direct JSON read this session); floor is now a genuine two-sided ratchet (`TestSeedBankUnguardedFloorIsTheRealCount`, re-run this session, PASS). 30 of 51 (59%) still unguarded. |
| SC4a | Candidate structurally cannot reach/alter its evaluator | ✓ VERIFIED | Unchanged; `pkg/shadow` full isolation suite re-run this session — PASS. |
| SC4b | Beneficial candidate actually distinguishable from harmful/overfit one, reachable in production | ✓ VERIFIED | `shadowEvaluator()` now built from `shadowClassifyAgainstBank`, a real per-fixture classifier (confirmed by direct read of `cmd/shadow_cmds.go`). I independently mutated the run function back to `shadow.NewResult(true)`, rebuilt, and confirmed `TestShadowGraderDistinguishesBeneficialFromHarmful` fails with 3 distinct assertion failures naming the exact fixtures it should have distinguished; reverted cleanly (`git diff` empty afterward). |
| SC5a | Retained-authority boundary structurally unbypassable by the canary path | ✓ VERIFIED | Unchanged; re-run this session — PASS. |
| SC5b | A beneficial candidate can actually be promoted and rolled back by the running program | ✓ VERIFIED | `runAutomaticImprovementPass` drives comparison → `admitCandidateToCanary` → `startCanary` → `completeCanary`/`rollbackCanary`, called once from `runPhaseEndConsolidation`. I independently commented out that one call site, rebuilt, and confirmed `TestAutomaticImprovementPassIsReachedFromBothCheckLanes` fails naming both `runCodexContinue` and `runCodexContinueFinalize`; reverted cleanly. Note: `processRunningImprovementCanary`'s complete-vs-rollback decision uses the current phase's own whole-check pass/fail as a proxy for the specific canary's health (WR-02) — a documented design limitation (now explained in-code), not a wiring gap; the mechanism itself functions and is reachable. |
| SC5c | Verified-success and preventable-intervention figures are honest and usable on a live colony | ✓ VERIFIED | `TestTwoFiguresAreRealOnALiveColony` (re-run this session, PASS) drives real build/check/intervention episodes through real production writers and confirms non-zero verified-success and non-zero preventable-intervention figures, no hand-built ledger records. No longer downstream-blocked by SC3a. |
| SC5d | A source-code improvement can actually be proposed by the running program, and cannot self-merge/publish/deploy | ✓ VERIFIED | `triggerRepeatedInterventionProposal` gives `proposeSourceImprovement` its first real caller (3+ distinct episodes of the same declared intervention kind), reached from the same phase-end boundary. The code review found a serious issue in this exact new code (CR-04: unattended `git checkout` on the live working tree, with a documented "leaves the repo on the wrong branch" failure mode) and it was fixed by moving all git mutation into an isolated temporary worktree — confirmed by grep (`"checkout"` as a literal string no longer appears anywhere in `cmd/source_proposal.go`) and by re-running `TestSourceProposalFailureNeverTouchesTheLiveCheckout`/`TestSourceProposalNeverChecksOutInTheLiveRepository`/`TestSourceProposalCannotMergePublishOrDeploy`, all PASS. |

**Score:** 9/11 sub-truths fully verified; 2 narrowed but not fully closed (SC3a, SC3c), both honestly documented in REQUIREMENTS.md/WINDOWS.md with the exact remaining number.

**Roadmap-level rollup** (the 5 official ROADMAP.md success criteria, each of which several sub-truths above compose): SC1 ✓, SC2 ✓, SC3 ⚠ (episode-field and fixture-guard completeness both partial), SC4 ✓, SC5 ✓ (all four SC5 sub-truths now closed). 4 of 5 roadmap-level criteria fully met.

### Additional finding beyond the original 11 sub-truths: the automatic hypothesis-to-validated promoter (WINDOWS #44)

204-16 built `promoteHelpfulHypotheses`, intended to promote a learned lesson from "unproven guess" to "proven true" once the program's own records show it genuinely helped. This is not one of the original 11 sub-truths above (it is a new mechanism this closure added, tied to WINDOWS entry 44, not to SC2's original wording about rendering/provenance). The phase's own code review (204-REVIEW.md, CR-01) found it can **never promote anything in production**: the learning-propose/-validate pipeline's `learn.Entry.ID` and the guidance-application ledger's `GuidanceID` are two disjoint identifier spaces with no production bridge between them, and independently, no production code anywhere writes the terminal "helpful/neutral/harmful" states the promoter reads at all. The original test suite passed only because it hand-typed an ID into a shape the real runtime cannot produce — exactly the "false certificate" pattern this repo's own CLAUDE.md calls out by name.

This was **not swept under the rug**: WINDOWS.md entry 44 was reopened from `fixed` back to `open` with the full root cause named in both the table and the JSON ledger; CLAUDE.md's and `.claude/rules/aether-colony.md`'s prose were corrected to say the gating rule is real and tested but the promotion itself "cannot happen in the running program yet"; the false-certificate test was replaced with `TestHypothesisPromotionNeverCrossesTheIdentifierGap`, which asserts NO promotion occurs when driven through the real writer chain. I independently confirmed this test passes, confirmed the corrected CLAUDE.md prose is accurate against the code, and confirmed WINDOWS #44's table/JSON both say `open`.

**I am not counting this as a 12th failed must-have** (it was never one of the ROADMAP's decomposed sub-truths verified last time), but it is worth surfacing prominently: it is the exact "machinery exists but was never switched on" failure mode this whole phase exists to fix, caught by this session's own review rather than by an external audit — a healthy sign for the review process, but a reminder that "SUMMARY says fixed" is never sufficient evidence on its own, including for this closure's own summaries.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/shadow_cmds.go` | Real per-fixture classifier behind `shadowEvaluator` | ✓ VERIFIED | `shadowClassifyAgainstBank`; confirmed by direct read and by my own revert-and-fail mutation test |
| `cmd/improvement_pass.go` | Automatic pass driving comparison→gate→canary, and the source-proposal trigger | ✓ VERIFIED | `runAutomaticImprovementPass`, `triggerRepeatedInterventionProposal`; confirmed by direct read and my own mutation test |
| `cmd/improvement_cmds.go` | Registered `aether improve`/`shadow-declare`/`shadow-compare` | ✓ VERIFIED | All three exit 0 against the real built binary, confirmed live in this session (previously all three failed or did not exist) |
| `.aether/commands/improve.yaml`, `.claude/commands/ant/improve.md`, `.opencode/commands/ant/improve.md` | `/ant-improve` wrapper triplet | ✓ VERIFIED | All three exist; Claude/OpenCode copies confirmed byte-identical by `diff` in this session |
| `cmd/episode_ledger.go` | Durable, immutable episode/outcome ledger with 9 previously-writerless fields | ⚠ PARTIAL | 7 of 9 fields real and written on every lane holding the fact, confirmed by mutation tests; usage/cost absent on both check lanes (documented) |
| `cmd/testdata/fixture-bank/v1/bank.json` | Versioned, guarded regression-fixture bank | ⚠ PARTIAL | 51 fixtures, 21 guarded (confirmed by direct JSON read), 30 unguarded; two-sided ratchet confirmed passing |
| `cmd/promotion_gate.go`, `cmd/rollback.go` | Structural gate, atomic canary lifecycle, now with a real caller | ✓ VERIFIED | Reached from both check lanes via the automatic pass, confirmed by my own mutation test |
| `cmd/improvement_report.go` | Two-figure honest reporting, now real on live data | ✓ VERIFIED | `TestTwoFiguresAreRealOnALiveColony` re-run, PASS |
| `cmd/source_proposal.go` | Propose-only source-change boundary, with a real caller, git-isolated | ✓ VERIFIED | No `checkout` call site against the live root remains (grep-confirmed); isolated-worktree pattern confirmed by direct read |
| `cmd/learning_validator.go` | Automatic hypothesis-to-validated promoter | ⚠ ORPHANED (honestly documented) | Wiring is real and reachable from both check lanes, but cannot promote anything today due to the ID-space mismatch (CR-01); see the dedicated section above |
| `.planning/REQUIREMENTS.md` | Truthful satisfaction state | ✓ VERIFIED, with one stale figure | SYNTH-06/LEARN-01/02/03/04/06/07/08 ticked on named passing tests, all independently re-confirmed passing this session; LEARN-05 correctly left unticked. One stale number: cites the fixture bank as 52/22 where the real current bank is 51/21 (see `documentation_staleness` in frontmatter) |
| `.planning/WINDOWS.md` | Truthful defect ledger, table/JSON in agreement | ✓ VERIFIED, with one stale header | Table and JSON ledger agree with EACH OTHER on every entry, confirmed by parsing both in this session (33 open / 14 fixed / 47 total, matching exactly). The separate summary header block itself is stale by one in each direction (see `documentation_staleness`) — cosmetic, not a table/JSON disagreement |
| `CLAUDE.md` §Learning Governor | Test-cited claims, plain English, honest about limitations | ✓ VERIFIED | `TestCLAUDEMDLearningGovernorClaimsCiteLiveTests` re-run, PASS; manually read the hypothesis-promotion paragraph and confirmed it accurately states the promotion "cannot happen in the running program yet" |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `cmd/consolidation_lifecycle.go` (`runPhaseEndConsolidation`) | `cmd/improvement_pass.go` (`runAutomaticImprovementPass`) | one call site, both check lanes | ✓ WIRED | I independently commented out this call, rebuilt, confirmed `TestAutomaticImprovementPassIsReachedFromBothCheckLanes` fails naming both lanes, reverted (`git diff` clean afterward) |
| `cmd/improvement_pass.go` | `cmd/shadow_cmds.go` (`runShadowCompare`) → `cmd/promotion_gate.go` (`admitCandidateToCanary`) → `cmd/rollback.go` (`startCanary`/`completeCanary`/`rollbackCanary`) | automatic pass | ✓ WIRED | Confirmed via `TestAutomaticImprovementPassTracerEndToEnd` (re-run, PASS) |
| `cmd/swarm_cmd.go` (`runSwarmDestroy`) | `cmd/live_events.go` (episode boundary) | direct call | ✓ WIRED | Re-run `TestSwarmLaneOpensAndClosesADurableEpisode`, PASS |
| `cmd/swarm_cmd.go` (`runSwarmFinalize`) | `cmd/live_events.go` (episode boundary) | direct call, added by CR-03 fix | ✓ WIRED | `TestSwarmFinalizeLaneOpensAndClosesADurableEpisode` re-run, PASS — this lane had NO episode boundary at all before the review fix |
| `cmd/recovery_orchestrator.go` (`orchestrateRecovery`) | `cmd/live_events.go` (episode boundary, only when it owns the episode) | direct call, disambiguated by CR-02 fix | ✓ WIRED | `TestBuildFinalize_MultipleFailedDispatches_EachGetsADistinctDurableEpisode` re-run, PASS — before the fix, two failed dispatches in one phase silently dropped the second recovery record |
| `cmd/codex_build.go`/`cmd/codex_continue.go`/`cmd/codex_continue_finalize.go` | `cmd/live_events.go` (`emitColonyLiveOutcomeRecorded`) | deferred close | ✓ WIRED | Confirmed via 3 separate FAILS-WHEN-UNWIRED tests, all re-run PASS |
| `cmd/memory_schema.go` (census) | `cmd/episode_ledger.go` (`episodeLedgerRecord`) | 21 writer entries | ✓ WIRED | `TestEveryMemoryStoreFieldHasALiveWriter` re-run, PASS |
| `cmd/improvement_pass.go` (`triggerRepeatedInterventionProposal`) | `cmd/source_proposal.go` (`proposeSourceImprovement`) | direct call, isolated worktree | ✓ WIRED | `TestRepeatedInterventionProposesExactlyOneSourceChange` re-run, PASS; `sourceProposalReachabilityEntryPoints` includes the new trigger, confirmed by `TestSourceProposalCannotMergePublishOrDeploy` |
| `cmd/consolidation_lifecycle.go` | `cmd/learning_validator.go` (`promoteHelpfulHypotheses`) | one call site, both check lanes | ✓ WIRED (call site real; capability inert) | `TestHypothesisPromoterIsReachedFromBothCheckLanes` re-run, PASS — the call site genuinely reaches both lanes, but see the dedicated finding above: nothing in production ever feeds it a promotable record |

### Behavioral Spot-Checks (independently re-run in this session, not taken from any SUMMARY's word)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Real grader distinguishes beneficial/harmful/overfit | `go test ./cmd -run '^(TestShadowGraderDistinguishesBeneficialFromHarmful\|TestOverfitCandidateIsRefusedAtTheGate\|TestEvaluatorDigestIsStableAndChanged)$'` | PASS | ✓ PASS |
| Automatic pass reached from both check lanes; no bypass parameter; retained scopes refused | `go test ./cmd -run '^(TestAutomaticImprovementPassTracerEndToEnd\|TestAutomaticImprovementPassIsReachedFromBothCheckLanes\|TestAutomaticPassNeverPromotesOutsideTheTwoScopes\|TestAutomaticPassCarriesNoBypassParameter\|TestUnrecognizedScopeIsRefusedAndWritesNothing)$'` | PASS | ✓ PASS |
| Every lifecycle lane writes a durable outcome, zero skips | `go test ./cmd -run '^TestEveryLifecycleLaneWritesADurableOutcome$' -v` | PASS, 6/6 subtests, no SKIP lines | ✓ PASS |
| Swarm's delegate/finalize lane (CR-03 fix) | `go test ./cmd -run '^TestSwarmFinalizeLaneOpensAndClosesADurableEpisode$'` | PASS | ✓ PASS |
| Recovery's per-decision episode disambiguation (CR-02 fix) | `go test ./cmd -run '^TestBuildFinalize_MultipleFailedDispatches_EachGetsADistinctDurableEpisode$'` | PASS | ✓ PASS |
| Episode fields on build/native-check/delegate-check | `go test ./cmd -run '^(TestBuildEpisodeRecordsItsOwnFacts\|TestCheckEpisodeRecordsItsOwnFacts\|TestDelegateCheckEpisodeRecordsItsOwnFacts\|TestEpisodeOutcomeIsRecordedFromBothCheckLanes)$'` | PASS | ✓ PASS |
| Two honest figures real on a live colony | `go test ./cmd -run '^(TestTwoFiguresAreRealOnALiveColony\|TestReportBoundariesOnRealRecords\|TestEmptyWindowOverRealLedgerIsZeroNotPerfect\|TestRenderedLiveReportKeepsTheTwoFiguresApart)$'` | PASS | ✓ PASS |
| Source proposal never touches the live checkout (CR-04 fix) | `go test ./cmd -run '^(TestSourceProposalFailureNeverTouchesTheLiveCheckout\|TestSourceProposalNeverChecksOutInTheLiveRepository\|TestSourceProposalCannotMergePublishOrDeploy\|TestRepeatedInterventionProposesExactlyOneSourceChange\|TestAutomaticProposalReplayCreatesNoSecondBranch)$'` | PASS | ✓ PASS |
| No `"checkout"` literal remains in source_proposal.go | `grep -n '"checkout"' cmd/source_proposal.go` | no output (0 matches) | ✓ PASS |
| Hypothesis promotion never crosses the identifier gap (CR-01 fix) | `go test ./cmd -run '^(TestHypothesisPromotionNeverCrossesTheIdentifierGap\|TestActedOnIsNotEnoughToValidate\|TestNeutralOrHarmfulIsNeverValidated\|TestUncorroboratedClaimIsNeverValidated\|TestHypothesisWithNoRecordStaysAHypothesis\|TestPromotionPassIsIdempotentAndLeavesAlreadyValidatedEntriesAlone\|TestDisprovenEntryIsNeverPromoted\|TestHypothesisPromoterIsReachedFromBothCheckLanes)$'` | PASS | ✓ PASS |
| WR-01/WR-03 warning fixes | `go test ./cmd -run '^(TestRefusedCandidateIsNotReCompareOrReReportedOnANextCheck\|TestTerseFixtureStillHasAQualifyingSubjectWord\|TestEveryBankFixtureHasAQualifyingSubjectWord)$'` | PASS | ✓ PASS |
| Fixture bank / eval gates | `go test ./cmd -run '^(TestEveryFixtureNamesItsGuardOrIsCountedUnguarded\|TestSeedBankUnguardedFloorIsTheRealCount\|TestSeedBankIndexTotalsAgree\|TestSeededBankFixturesAllCiteRealProvenance)$'` | PASS | ✓ PASS |
| Retained authority / isolation unbroken | `go test ./cmd -run '^(TestOnlyTwoScopesAreCanaryPromotable\|TestRetainedAuthorityCannotBecomeCanaryPromotable\|TestEachRetainedScopeRefusalNamesItsAuthority\|TestNeitherCoordinatorNorAutopilotCanWaiveARetainedRefusal)$'` + `go test ./pkg/shadow/...` | PASS | ✓ PASS |
| Wrapper/reachability parity did not widen | `go test ./cmd -run '^(TestPlatformParityGolden\|TestNoRegisteredSubcommandIsUnreferenced)$'`; `python3 -c "..." orphan_allowlist.json` → 255 (unchanged) | PASS | ✓ PASS |
| CLAUDE.md claims cite live tests | `go test ./cmd -run '^TestCLAUDEMDLearningGovernorClaimsCiteLiveTests$'` | PASS | ✓ PASS |
| **My own independent mutation #1** (never taken from a SUMMARY's word): commented out `runAutomaticImprovementPass(phaseID)` call in `runPhaseEndConsolidation`, rebuilt | `go test ./cmd -run '^TestAutomaticImprovementPassIsReachedFromBothCheckLanes$'` | FAIL, naming both `runCodexContinue` and `runCodexContinueFinalize` | ✓ Confirms real wiring (reverted, `git diff` clean afterward) |
| **My own independent mutation #2**: restored `shadowEvaluator`'s run function to `shadow.NewResult(true)`, rebuilt | `go test ./cmd -run '^TestShadowGraderDistinguishesBeneficialFromHarmful$'` | FAIL, 3 named assertion failures | ✓ Confirms real grading (reverted, `git diff` clean afterward) |
| `aether shadow-declare --help` / `shadow-compare --help` / `improve --help` (real built binary) | `go run ./cmd/aether ...` | all exit 0 | ✓ PASS (previously non-zero/nonexistent) |
| `go build ./cmd/aether`, `go vet ./cmd/... ./pkg/...`, `gofmt -l` | | build/vet clean; 3 gofmt-flagged files pre-exist and are untouched by this phase's diff (confirmed by `git diff --name-only`) | ✓ PASS |

### Probe Execution

Not applicable — no `scripts/*/tests/probe-*.sh` conventional probes are declared by this phase's PLAN/SUMMARY files, confirmed by `find`/`grep` in this session (same as the previous verification).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| SYNTH-06 | 204-01 | Learning mechanism study | ✓ SATISFIED | Unchanged, re-confirmed |
| LEARN-01 | 204-01/02/03/11 | Memory truth and lineage | ✓ SATISFIED | Unchanged, re-confirmed |
| LEARN-02 | 204-04/13/15 | Outcome and intervention ledger | ✓ SATISFIED, with a named remaining limit | All six lanes now write durable episodes with zero skips (204-13); build and both check lanes write 7 of 9 fields + interventions (204-15). Remaining limit: usage/reported_cost_usd absent on both check lanes — named in REQUIREMENTS.md itself, matching this verification's own SC3a finding |
| LEARN-03 | 204-02 | Real application evidence | ✓ SATISFIED | Unchanged, re-confirmed |
| LEARN-04 | 204-05 | Failure-to-evaluation conversion | ✓ SATISFIED | Unchanged, re-confirmed (bank structure/provenance, independent of guard completeness) |
| LEARN-05 | 204-07/14 | Permanent evaluation/test architecture | ✗ NOT SATISFIED (correctly left unticked) | Gate architecture real and tested; 30 of 51 fixtures still carry no guard. REQUIREMENTS.md's own honest assessment matches this verification's SC3c finding, modulo the stale 52/22 prose figure noted above |
| LEARN-06 | 204-08/12 | Independent shadow comparison | ✓ SATISFIED, with a named (deliberate) scope limit | Real grader confirmed by my own mutation test; comparison covers this project's own settings/routing fixtures only, never a code-valued candidate — this is LEARN-06's own declared scope, not an incomplete-work gap |
| LEARN-07 | 204-09/12 | Tiered promotion and rollback | ✓ SATISFIED | Automatic pass reaches admission/canary/rollback from both check lanes, confirmed by my own mutation test |
| LEARN-08 | 204-10/11/15/16 | Honest improvement and source boundary | ✓ SATISFIED | Two honest figures real on live data (204-15); source proposal has a real, git-safe automatic caller (204-16 + CR-04 fix) |

No orphaned requirements found — all nine phase requirements are claimed by at least one plan, confirmed by cross-referencing every plan's frontmatter `requirements:` field against REQUIREMENTS.md's Phase 204 traceability row.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | — | No `TBD`/`FIXME`/`XXX` debt markers found in any file this phase's gap-closure/review-fix diff touched (confirmed by grepping every file in `git diff --name-only 4eb7a9ed HEAD` in this session) | — | — |
| `cmd/improvement_pass.go` | `processRunningImprovementCanary` | WR-02: the canary complete-vs-rollback decision uses the current phase's own whole-check pass/fail as a proxy for the specific canary's own health — almost always reads "passed" regardless of the canary's actual scope | ℹ️ Info (documented design choice, not fixed functionally) | Now has an explicit doc comment naming this limitation (per the review's own accepted resolution); the mechanism still functions, just with an imprecise heuristic |
| `cmd/learning_validator.go` | `promoteHelpfulHypotheses`/`learningEntryHasHelpfulApplication` | The wired call site is real, but the capability is currently inert due to an ID-space mismatch (CR-01) | ⚠️ Warning (honestly documented, not hidden) | See the dedicated finding above; WINDOWS #44 correctly reopened, CLAUDE.md corrected |
| N/A | N/A | No `TODO`/`HACK`/`PLACEHOLDER` markers found in the touched files | — | — |

No stub patterns (hardcoded empty returns, always-true placeholders reachable from production) remain: the one previously-flagged stub (`shadowEvaluator`'s always-pass placeholder) is now a real classifier, confirmed by my own revert-and-fail test.

### Human Verification Required

None. Every truth above was independently confirmed by direct code inspection, live scoped test execution, and — for the two highest-stakes claims (the real grader and the automatic cross-lane pass) — a mutation I personally performed, watched fail, and reverted myself in this session, leaving the tree byte-identical to its pre-mutation state (confirmed by `git diff`/`git status --porcelain`).

### Documentation staleness (non-blocking, see frontmatter for full detail)

Two small bookkeeping figures in the project's own truth-telling documents are one number off from the codebase's real current state, both traced to the code-review fix pass's own bank regeneration (which quietly shrank the fixture bank by one entry when WINDOWS #44 was honestly reopened): REQUIREMENTS.md/WINDOWS.md cite the fixture bank as 52 total/22 guarded where it is actually 51/21, and WINDOWS.md's own summary header (32 open/15 fixed) is off by one from its own table+JSON body (33 open/14 fixed — which agree perfectly with each other). Neither affects any test, ratchet, or gate, since those all read the bank/ledger live. Worth a one-line correction the next time either file is touched, but not a phase-blocking gap.

### Gaps Summary

Phase 204's gap-closure round (204-12 through 204-16) closed 5 of the 7 gaps this verification previously found — and its own code review caught 4 additional Critical bugs the gap-closure plans' own SUMMARYs did not surface (a recovery-episode collision that silently dropped records on the primary interactive path, a swarm delegate lane that recorded no episode at all, an automatic pass that could leave the real repository on the wrong git branch, and an automatic promoter whose ID-space mismatch made it permanently unable to promote anything) — all of which are now genuinely fixed and independently re-confirmed in this session, except the last, which is honestly disclosed as an open limitation rather than falsely marked fixed.

Two gaps remain, both substantially narrowed rather than newly discovered, and both are named accurately in this project's own REQUIREMENTS.md and WINDOWS.md rather than hidden under a tick:

- **SC3a:** a watcher/reviewer worker's token usage and cost are not yet surfaced to the check-lane episode close, so those two fields stay honestly absent (never fabricated) on continue episodes specifically.
- **SC3c:** 30 of this project's 51 confirmed incidents in the regression-fixture bank still have no guarding test, though the guarded count nearly tripled and the recorded floor can now only tighten, never loosen.

Recommended next step: a further plan (or a small set of them) restructuring `runCodexContinue`'s call chain to surface watcher usage to its already-registered episode-close boundary, and continuing to guard more of the 30 remaining fixture-bank incidents as their underlying fixes are individually confirmed — following the same honesty bar 204-14 already held itself to (never attach a guard to an incident that is not genuinely fixed). Separately, WINDOWS entry 44 (the inert hypothesis promoter) needs either a real identifier bridge between `learn.Entry` and the guidance-application ledger, or a real production writer of the Helpful/Neutral/Harmful terminal states — neither of which this closure attempted, correctly, since no honest bridge existed within its own reach.

---

_Verified: 2026-09-15_
_Verifier: Claude (gsd-verifier)_
