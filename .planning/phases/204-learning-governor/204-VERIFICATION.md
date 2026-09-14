---
phase: 204-learning-governor
verified: 2026-09-14T16:20:22Z
status: gaps_found
score: 5/12 must-haves verified
behavior_unverified: 0
overrides_applied: 0
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
    status: failed
    reason: "The episode ledger's writer/reader pair is fully built and tested, but the boundary emitters for nine of its fields have zero production callers, so a real episode is recorded with only its outcome/terminal skeleton and every one of evidence_ids, hard_gate_results, changed_decision_ids, usage, reported_cost_usd, interventions, episode_revision, acceptance_digest, evaluator_digest stays empty (WINDOWS.md entry 38)."
    artifacts:
      - path: "cmd/live_events.go"
        issue: "emitColonyLiveOutcomeRecorded and emitColonyLiveInterventionRecorded are defined but have no non-test caller anywhere in cmd/ or pkg/"
    missing:
      - "Wire the build/continue/swarm/oracle/recovery boundary callers to pass real usage, gate results, evidence and intervention facts through the existing emitters."
  - truth: "SC3b: Every lifecycle lane opens and closes a durable episode record"
    status: failed
    reason: "TestEveryLifecycleLaneWritesADurableOutcome explicitly t.Skipf's the swarm and recovery lanes because neither calls emitColonyLiveEpisodeStarted/Ended in production; only build, continue, plan and oracle are proven (WINDOWS.md entry 41, reproduced live in this verification run)."
    artifacts:
      - path: "cmd/episode_ledger_test.go"
        issue: "Swarm and recovery subtests are skipped, not passing, so the phase's own test suite documents the gap rather than closing it"
    missing:
      - "Route cmd/swarm_cmd.go and the recovery entry point through the same episode-boundary calls the other four lanes already use."
  - truth: "SC3c: Confirmed incidents become a versioned regression-fixture bank actually guarded by real tests"
    status: failed
    reason: "Of the 47 fixtures in cmd/testdata/fixture-bank/v1/bank.json (verified by direct JSON read at this HEAD), only 7 carry a named guard test; 40 sit on the counted-unguarded list (WINDOWS.md entry 42). The bank's structure, provenance and dedup rules are real and tested, but most confirmed incidents are not yet actually regression-protected."
    artifacts:
      - path: "cmd/testdata/fixture-bank/v1/bank.json"
        issue: "40 of 47 fixtures have an empty guard field"
    missing:
      - "Name and confirm a real guard test for each remaining fixture's own confirmed incident, shrinking the recorded unguarded floor."
  - truth: "SC4b: A beneficial candidate is actually distinguishable from a harmful/overfit one by a real grading function, reachable in production"
    status: failed
    reason: "pkg/shadow's isolation and verdict-rule logic is real and fully tested (TestCompareVerdictRules, TestOverfitIsNeverReportedAsBeneficial, TestCandidateCannotAlterItsEvaluatorDigest all pass), but cmd/shadow_cmds.go's shadowEvaluator run function is a deterministic always-pass placeholder (WINDOWS.md entry 40, confirmed by reading cmd/shadow_cmds.go:183-201) -- every real comparison today reports a tied verdict regardless of the candidate. Separately, the CLI surface that would drive a comparison is not registered: `go run ./cmd/aether shadow-declare --help` returns 'unknown command shadow-declare for aether' (verified live in this run) -- shadowDeclareCmd/shadowCompareCmd are constructed in cmd/shadow_cmds.go's init() but never added to any parent command (WINDOWS.md entry 39, confirmed by grep across the whole tree finding zero AddCommand call for either variable)."
    artifacts:
      - path: "cmd/shadow_cmds.go"
        issue: "shadowEvaluator's run function always returns shadow.NewResult(true); shadowDeclareCmd/shadowCompareCmd are never registered on rootCmd or any subcommand tree"
    missing:
      - "A real per-fixture classifier wired through FrozenEvaluator.Run, and a registered CLI or programmatic caller that can actually invoke a comparison."
  - truth: "SC5b: A promotable, beneficial candidate can actually be promoted into a canary and atomically rolled back by the running program"
    status: failed
    reason: "admitCandidateToCanary (cmd/promotion_gate.go), startCanary, rollbackCanary and completeCanary (cmd/rollback.go) are all real, individually well-tested functions -- TestRetainedAuthorityCannotBecomeCanaryPromotable and TestRegressionRestoresAndQuarantinesAtomically both pass -- but a repo-wide grep for each function name outside _test.go files finds zero production call sites. Combined with the SC4b gap (nothing ever produces a real Comparison to feed this gate) and the unregistered shadow-declare/shadow-compare CLI, there is currently no path in the running program that can ever reach a canary promotion or a rollback. This is a stronger finding than WINDOWS.md entry 39 records: entry 39 names the CLI commands as uncalled; this verification additionally confirms the promotion gate and rollback functions those commands would eventually feed are themselves uncalled by anything in production."
    artifacts:
      - path: "cmd/promotion_gate.go"
        issue: "admitCandidateToCanary has no production caller"
      - path: "cmd/rollback.go"
        issue: "startCanary/rollbackCanary/completeCanary have no production caller"
    missing:
      - "A production entry point (CLI command, scheduled pass, or continue/build-lane hook) that actually drives a declared candidate through comparison, gate admission, and canary lifecycle."
  - truth: "SC5c: Verified-useful-task success and preventable interventions are reported as two honest figures, usable on a live colony"
    status: failed
    reason: "buildImprovementReport/renderImprovementReport are correctly implemented and tested against real ledger fixtures (TestTwoFiguresAreNeverCombined passes), and isVerifiedUsefulSuccess fails safe (treats an empty HardGateResults map as unclassified rather than falsely passing). But because SC3a's boundary emitters are unwired, every real production episode today has empty EvidenceIDs/HardGateResults/Interventions, so the report will classify every real episode as unclassified -- zero verified successes and zero preventable interventions reported on a live colony, by design rather than by bug, until SC3a closes. 204-10-SUMMARY.md's own Next Phase Readiness section states this explicitly."
    artifacts:
      - path: "cmd/improvement_report.go"
        issue: "Correct code, but downstream of the SC3a wiring gap -- currently produces no real signal on a live colony"
    missing:
      - "SC3a's production wiring, which this figure is entirely downstream of."
  - truth: "SC5d: A source-code improvement proposal can actually be created by the running program"
    status: failed
    reason: "proposeSourceImprovement (cmd/source_proposal.go) and its cannot-merge/publish/deploy refusal are real and tested (TestSourceProposalCannotMergePublishOrDeploy passes), but a repo-wide grep finds zero non-test, non-comment callers of proposeSourceImprovement -- 204-10-SUMMARY.md's own Next Phase Readiness section names this gap directly (WINDOWS.md entry 45). The structural boundary (a proposal can never self-merge) is real; the capability it bounds (the program can propose a source change at all) does not yet run in production."
    artifacts:
      - path: "cmd/source_proposal.go"
        issue: "proposeSourceImprovement has no production caller"
    missing:
      - "A CLI entry point or automatic trigger that actually calls proposeSourceImprovement, plus a declared closed intervention-kind vocabulary for its downstream classification (WINDOWS.md entry 45's second half)."
requirements_discrepancy:
  - "REQUIREMENTS.md marks LEARN-02, LEARN-06, LEARN-07 and LEARN-08 as [x] Satisfied. The structural/library-level work behind each is real and tested, but the production-reachability gaps above (episode-field wiring, shadow grading, promotion/rollback callers, source-proposal callers) mean the requirement's own wording ('Each started goal episode records... evidence, hard gates... interventions...'; 'run beside a frozen baseline... the candidate cannot select or edit its evaluator'; 'Only proven reversible low-risk project knowledge... may enter a bounded canary'; 'source improvements may be proposed') is not yet true of the running program. This mirrors the exact pattern CLAUDE.md's own Definition of Done section names as this project's repeated failure mode."
---

# Phase 204: Learning Governor Verification Report

**Phase Goal:** Make memory truthful and prove behavior changes through immutable outcomes, permanent evaluations, independent shadow comparison, bounded promotion, and rollback.
**Verified:** 2026-09-14T16:20:22Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Summary for the owner (plain English)

Most of this phase landed solidly: the project now has one honest place that decides what counts as "proven" versus "just a guess" (no worker is ever shown an unchecked idea labelled as verified), every remembered fact now carries a version number and a "where did this come from" tag, and there's a real permanent record when a piece of advice actually helped, did nothing, or made something worse — reached from both places the program checks its own work, and proven end-to-end with a real test.

But five of the eleven building blocks in this phase were built and thoroughly tested in isolation, then never actually connected to anything that runs. Concretely:

1. **The "did this run cost too much / did it fail / who stepped in" record** — the pipe exists and the two places that would fill it in are written, but nothing in the actual build, swarm, or recovery process calls them yet, so those fields stay empty on every real run.
2. **The "try a safer idea safely" system (shadow comparison)** — the safety mechanism that stops a proposed change from cheating on its own test is real and proven. But the actual grading step inside it is a placeholder that always says "pass," and the command you'd type to try it (`aether shadow-declare`) doesn't exist yet — I checked by running it.
3. **The "promote a good idea automatically, undo a bad one automatically" system** — also fully built and unit-tested, but nothing in the running program ever calls it. There is currently no way for this feature to actually happen during real use.
4. **The "confirmed bug becomes a permanent regression test" bank** — has 47 entries, but only 7 of them actually have a real test locking them in yet.
5. **The "suggest an improvement to Aether's own code" feature** — the safety fence (it can never merge or publish itself) is real and proven, but nothing calls the feature that would actually propose a change, so it can't happen yet either.

None of this is hidden — the team that built it wrote every one of these gaps down honestly in the project's own defect log before I even started checking (see WINDOWS.md entries 38–45), and I independently confirmed each one by reading the code and, for the shadow command, by actually trying to run it. The parts that are done are done well. The parts that aren't are inert, not broken — they fail safe (report "not yet classified" rather than lying), but the phase goal ("prove behavior changes... bounded promotion, and rollback") isn't actually happening in the running program yet.

## Goal Achievement

### Observable Truths (mapped to ROADMAP success criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| SC1 | Cited mechanism study reconstructs Classic learning mechanics, audits every store, justifies smallest architecture | ✓ VERIFIED | `.planning/phases/204-learning-governor/204-CLASSIC-SYNTHESIS.md` and `204-CLASSIC-HISTORICAL-EVIDENCE.md` exist; `TestClassicContractPhase204MechanismRegistry`, `TestClassicContractPhase204Cases`, `TestClassicContractRegistryHasNoRuntimeWriter` all pass (verified live, this run) |
| SC2 | Versioned memory schemas; hypothesis never rendered as verified; provenance/lineage on promoted material | ✓ VERIFIED | `cmd/memory_schema.go`; `TestEveryMemoryStoreFieldHasALiveWriter`, `TestLegacyRecordsReadAsLegacy`, `TestNewRecordsCarryVersionAndLineage`, `TestHypothesisIsNeverRenderedAsVerified`, `TestOneLearningStatusVocabulary`, `TestRetiredFieldWithoutOwnerAgreementIsRefused` all pass (verified live, this run) |
| SC3a | Every started episode records outcome, evidence, hard-gate, changed-decision, intervention, time, token, cost | ✗ FAILED | 9 of the episode record's fields (`evidence_ids`, `hard_gate_results`, `changed_decision_ids`, `usage`, `reported_cost_usd`, `interventions`, `episode_revision`, `acceptance_digest`, `evaluator_digest`) have zero production writers — confirmed by grep across `cmd/` and `pkg/` (WINDOWS #38) |
| SC3b | Every lifecycle lane writes a durable episode | ✗ FAILED | `TestEveryLifecycleLaneWritesADurableOutcome` skips the swarm and recovery subtests by name at this HEAD (verified live, this run — WINDOWS #41) |
| SC3c | Confirmed incidents become a guarded, versioned regression-fixture bank with hidden holdouts and budgeted gates | ⚠️ PARTIAL / FAILED on the guard claim | Gate/holdout architecture VERIFIED (`TestEvalGateVocabularyMatchesTheManifest`, `TestTruncatedGateRunFails`, `TestEverySentinelStillExists` all pass); bank has 47 fixtures, only 7 carry a guard (verified by direct JSON read, this run — WINDOWS #42) |
| SC4a | Candidate structurally cannot reach/alter its evaluator | ✓ VERIFIED | `pkg/shadow` full suite passes: `TestCandidateCannotAlterItsEvaluatorDigest`, `TestEvaluatorHasNoMutator`, `TestNoExportedFunctionReturnsAMutableEvaluator`, `TestCandidateHoldsNoEvaluator`, `TestPackageImportsNothingFromCmd` (verified live, this run) |
| SC4b | Beneficial candidate actually distinguishable from harmful/overfit one, reachable in production | ✗ FAILED | `shadowEvaluator`'s run function is a hardcoded always-pass placeholder (`cmd/shadow_cmds.go:197-199`, confirmed by direct read); `aether shadow-declare` is not a registered command (`go run ./cmd/aether shadow-declare --help` → "unknown command", verified live, this run) |
| SC5a | Retained-authority boundary structurally unbypassable by the canary path | ✓ VERIFIED | `TestOnlyTwoScopesAreCanaryPromotable`, `TestRetainedAuthorityCannotBecomeCanaryPromotable`, `TestEachRetainedScopeRefusalNamesItsAuthority` all pass (verified live, this run) |
| SC5b | A beneficial candidate can actually be promoted and rolled back by the running program | ✗ FAILED | `admitCandidateToCanary`, `startCanary`, `rollbackCanary`, `completeCanary` have zero non-test callers anywhere in the tree (confirmed by grep, this run — goes beyond what WINDOWS #39 records) |
| SC5c | Verified-success and preventable-intervention figures are honest and usable on a live colony | ⚠️ PARTIAL | Code correct and tested (`TestTwoFiguresAreNeverCombined` passes) but entirely downstream of SC3a; reports "unclassified" for every real episode today, by design (204-10-SUMMARY.md's own admission) |
| SC5d | A source-code improvement can actually be proposed by the running program, and cannot self-merge/publish/deploy | ⚠️ PARTIAL | Refusal boundary VERIFIED (`TestSourceProposalCannotMergePublishOrDeploy` passes); `proposeSourceImprovement` has zero production caller (confirmed by grep, this run — WINDOWS #45) |

**Score:** 5/11 sub-truths fully verified; 6 failed or partial, all traced to specific missing production wiring, all independently confirmed against live code/tests in this session.

### Deferred Items

None. Phase 205's success criteria (research-to-execution reconciliation, real UAT, owner acceptance) do not name closing these production-wiring gaps, so none of the above gaps are deferred to a later phase — they remain open against Phase 204's own success criteria.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `.planning/phases/204-learning-governor/204-CLASSIC-SYNTHESIS.md` | Mechanism study | ✓ VERIFIED | Exists, all 12 SYN-204 rows present, cited by 12 contract cases |
| `cmd/application_evidence.go` | Real production writer into credit ledger | ✓ VERIFIED | `recordPhaseApplicationCredit` called once from `runPhaseEndConsolidation`, reached from both check lanes (tested) |
| `cmd/learning_status_vocabulary.go` | One shared verified/unverified vocabulary | ✓ VERIFIED | `learningVerifiedEntries`/`learningUnverifiedEntries`, single-predicate structural test passes |
| `cmd/memory_schema.go` | Versioned schema/provenance contract | ✓ VERIFIED | Live-writer census, legacy-read compatibility, exception-list ratchet all tested and passing |
| `cmd/episode_ledger.go` | Durable, immutable episode/outcome ledger | ⚠️ PARTIAL | Store/reader/idempotency structurally sound and tested; 9 of its fields have no production writer (SC3a) |
| `cmd/testdata/fixture-bank/v1/bank.json` | Versioned regression-fixture bank | ⚠️ PARTIAL | 47 fixtures with real provenance/dedup; only 7 guarded by a real test |
| `pkg/shadow/*.go` | Isolated evaluator/comparison package | ✓ VERIFIED | Full isolation and verdict-rule suite passes |
| `cmd/shadow_cmds.go` | Command surface for declare/compare | ⚠️ ORPHANED | Builds and tests pass; commands not registered on any cobra parent; grading function is a placeholder |
| `cmd/promotion_gate.go` | Structural authority-refusal admission gate | ⚠️ ORPHANED | Gate logic correct and tested; zero production caller |
| `cmd/rollback.go` | Atomic canary checkpoint/rollback/quarantine | ⚠️ ORPHANED | Correct and tested; zero production caller |
| `cmd/improvement_report.go` | Two-figure honest reporting | ⚠️ PARTIAL | Correct and tested; downstream of SC3a, reports unclassified in production today |
| `cmd/source_proposal.go` | Propose-only source-change boundary | ⚠️ ORPHANED | Refusal boundary correct and tested; `proposeSourceImprovement` has zero caller |
| `CLAUDE.md` §Learning Governor | Test-cited claims, plain English | ⚠️ PARTIAL | Every cited test genuinely exists and passes (`TestCLAUDEMDLearningGovernorClaimsCiteLiveTests`), but the prose itself describes the shadow-comparison and canary mechanisms in terms ("now be tried side by side," "run automatically on a small, watched, reversible trial basis") that overstate what a real, unattended run of the program can currently do — see gaps SC4b/SC5b above |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `cmd/consolidation_lifecycle.go` | `cmd/application_evidence.go` | `recordPhaseApplicationCredit` | ✓ WIRED | Exactly one call site, both check lanes reach it (tested) |
| `cmd/colony_prime_context.go` / `cmd/autopilot_lessons.go` | `cmd/learning_status_vocabulary.go` | `learningVerifiedEntries` | ✓ WIRED | Confirmed by passing structural + render tests |
| `pkg/colony/instincts.go` etc. | `cmd/memory_schema.go` | shared schema/provenance contract | ✓ WIRED | Confirmed by passing census test |
| lifecycle lanes (build/continue/plan/oracle) | `cmd/episode_ledger.go` | `emitColonyLiveEpisodeStarted/Ended` | ✓ WIRED (4 of 6 lanes) | swarm/recovery NOT_WIRED (test skips them by name) |
| boundary callers (build/continue/swarm/oracle/recovery) | `cmd/live_events.go` | `emitColonyLiveOutcomeRecorded`/`emitColonyLiveInterventionRecorded` | ✗ NOT_WIRED | Zero production callers found anywhere |
| `cmd/shadow_cmds.go` | `pkg/shadow/comparison.go` | `shadow.Compare` | ✓ WIRED (internally) | The Go call exists; but nothing external reaches `cmd/shadow_cmds.go` itself — no cobra registration |
| `cmd/shadow_cmds.go` | `cmd/episode_ledger.go` | `recordEpisodeOutcome` | ✓ WIRED (internally) | Same caveat — the command surface itself is unreachable |
| (nothing) | `cmd/promotion_gate.go` | `admitCandidateToCanary` | ✗ NOT_WIRED | No caller anywhere in the tree |
| (nothing) | `cmd/rollback.go` | `startCanary`/`rollbackCanary` | ✗ NOT_WIRED | No caller anywhere in the tree |
| (nothing) | `cmd/source_proposal.go` | `proposeSourceImprovement` | ✗ NOT_WIRED | No caller anywhere in the tree |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Credit-ledger tracer, all 5 outcome branches | `go test ./cmd -run '^TestPhaseApplicationCreditTracerEndToEnd$'` | PASS (5/5 subtests) | ✓ PASS |
| Hypothesis never rendered verified | `go test ./cmd -run '^TestHypothesisIsNeverRenderedAsVerified$'` | PASS (3/3 subtests) | ✓ PASS |
| Memory schema census / legacy compatibility | `go test ./cmd -run '^(TestEveryMemoryStoreFieldHasALiveWriter\|TestLegacyRecordsReadAsLegacy\|TestNewRecordsCarryVersionAndLineage)$'` | PASS | ✓ PASS |
| Episode ledger lifecycle-lane coverage | `go test ./cmd -run '^TestEveryLifecycleLaneWritesADurableOutcome$' -v` | PASS overall, but swarm + recovery subtests SKIP by name | ✗ FAIL (on the "every lane" claim) |
| Fixture bank / eval gates | `go test ./cmd -run '^(TestSeededBankFixturesAllCiteRealProvenance\|TestEveryFixtureNamesItsGuardOrIsCountedUnguarded\|TestEvalGateVocabularyMatchesTheManifest\|TestTruncatedGateRunFails\|TestEverySentinelStillExists)$'` | PASS | ✓ PASS |
| `pkg/shadow` full isolation suite | `go test ./pkg/shadow/...` | PASS (23/23 tests) | ✓ PASS |
| Promotion gate / rollback / quarantine | `go test ./cmd -run '^(TestOnlyTwoScopesAreCanaryPromotable\|TestRetainedAuthorityCannotBecomeCanaryPromotable\|TestEachRetainedScopeRefusalNamesItsAuthority\|TestRegressionRestoresAndQuarantinesAtomically\|TestQuarantineThresholdBoundaries)$'` | PASS | ✓ PASS |
| Honest reporting / source boundary | `go test ./cmd -run '^(TestTwoFiguresAreNeverCombined\|TestSourceProposalCannotMergePublishOrDeploy)$'` | PASS | ✓ PASS |
| Classic contract / CLAUDE.md removal-proof guard | `go test ./cmd -run '^(TestClassicContractPhase204MechanismRegistry\|TestClassicContractRegistryHasNoRuntimeWriter\|TestClassicContractPhase204Cases\|TestClassicContractSchema\|TestClassicMechanismCoverage\|TestCLAUDEMDLearningGovernorClaimsCiteLiveTests\|TestCLAUDEMDLearningGovernorSectionIsPlainEnglish)$'` | PASS | ✓ PASS |
| Shadow CLI reachability (real binary) | `go run ./cmd/aether shadow-declare --help` | `Error: unknown command "shadow-declare" for "aether"` | ✗ FAIL — confirms the command is not registered |
| Fixture-bank guard ratio (real data) | `python3 -c "..." bank.json` | `total 47 guarded 7 unguarded 40` | ✗ FAIL — confirms WINDOWS #42's ratio |

### Probe Execution

Not applicable — no `scripts/*/tests/probe-*.sh` conventional probes are declared by this phase's PLAN/SUMMARY files.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| SYNTH-06 | 204-01 | Learning mechanism study | ✓ SATISFIED | 204-CLASSIC-SYNTHESIS.md + historical evidence file, all tests pass |
| LEARN-01 | 204-01/02/03/11 | Memory truth and lineage | ✓ SATISFIED | Schema/provenance contract, hypothesis-never-verified rule, both fully wired and tested |
| LEARN-02 | 204-04 | Outcome and intervention ledger | ⚠️ PARTIALLY SATISFIED (marked [x] in REQUIREMENTS.md, evidence contradicts full satisfaction) | Ledger structure/idempotency real; 9 of its named fields (evidence, hard gates, changed decisions, interventions, tokens, cost, digests) have no production writer (WINDOWS #38) |
| LEARN-03 | 204-02 | Real application evidence | ✓ SATISFIED | `recordPhaseApplicationCredit` reached from both check lanes, all 4 outcome branches + no-record case proven end to end |
| LEARN-04 | 204-05 | Failure-to-evaluation conversion | ✓ SATISFIED (fixture data itself; see LEARN-05 for guard-test caveat) | 47 fixtures with real provenance, dedup and retirement rules |
| LEARN-05 | 204-07 | Permanent evaluation/test architecture | ⚠️ PARTIALLY SATISFIED | Seven named gates, sentinels, holdouts, truncation detection all real and tested; but only 7/47 fixtures are actually guarded by a named test today |
| LEARN-06 | 204-08 | Independent shadow comparison | ⚠️ PARTIALLY SATISFIED (marked [x] in REQUIREMENTS.md, evidence contradicts full satisfaction) | Structural isolation fully real; grading function is an always-pass placeholder and the CLI surface is unregistered — no real comparison can happen today |
| LEARN-07 | 204-09 | Tiered promotion and rollback | ⚠️ PARTIALLY SATISFIED (marked [x] in REQUIREMENTS.md, evidence contradicts full satisfaction) | Authority-refusal boundary fully real and tested; the admission/rollback functions themselves have zero production caller |
| LEARN-08 | 204-10/11 | Honest improvement/source boundary | ⚠️ PARTIALLY SATISFIED (marked [x] in REQUIREMENTS.md, evidence contradicts full satisfaction) | Refusal boundary and two-figure reporting both real and tested; `proposeSourceImprovement` has zero caller, and the report is downstream of the LEARN-02 gap |

No orphaned requirements found — all nine phase requirements are claimed by at least one plan.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `cmd/shadow_cmds.go` | 197-199 | Hardcoded always-true return (`return shadow.NewResult(true)`) reachable from a real (if unregistered) command path | ⚠️ Warning | Honestly documented in-code as a deliberate placeholder (not hidden), but is a real "return true" stub per the stub-detection patterns in this workflow |
| `cmd/shadow_cmds.go` | `init()` | Cobra commands constructed but never added to a parent command | ⚠️ Warning | Command is fully built, tested, and inert — classic orphan pattern this project's own audits repeatedly flag |
| `cmd/promotion_gate.go`, `cmd/rollback.go`, `cmd/source_proposal.go` | whole files | No production caller for any exported entry point | ⚠️ Warning (elevated to gap above) | Same orphan pattern — confirmed independently by this verification beyond what WINDOWS.md records |

No `TBD`/`FIXME`/`XXX` debt markers found in phase-modified files. No `TODO`/`HACK`/`PLACEHOLDER` markers found. All gaps above are explicitly documented in code comments, SUMMARY.md deviation sections, and WINDOWS.md — this phase's own honesty about its gaps is itself a positive finding, not a violation.

### Human Verification Required

None. Every gap above is independently confirmed by direct code inspection, live test execution, and a live CLI invocation in this session — none require subjective/visual/UX judgment.

### Gaps Summary

Phase 204 built substantial, well-tested infrastructure for every one of its five success criteria. The mechanism study (SC1) and the memory-truth/schema work (SC2) are genuinely complete and wired into production. The credit ledger's tracer path (a slice of SC3) is genuinely wired end to end.

But the phase's second half — the parts of the goal statement that say "prove behavior changes... through immutable outcomes... independent shadow comparison, bounded promotion, and rollback" — is built as a library of correct, individually-tested functions with **no path connecting them to anything the running program actually does**:

- The episode ledger's evidence/hard-gate/intervention/cost fields have emitters, but zero callers (SC3a).
- Two of six lifecycle lanes never open an episode at all (SC3b).
- 40 of 47 regression fixtures have no guarding test yet (SC3c).
- The shadow-comparison grader always says "pass," and its CLI command isn't even registered in the shipped binary — confirmed by literally running it (SC4b).
- The promotion gate and atomic rollback have zero production callers anywhere (SC5b) — a finding this verification makes explicit beyond what the project's own defect register (WINDOWS.md) already recorded for the CLI layer.
- The honest two-figure report is correct but will show "unclassified" for every real episode today, because it depends on SC3a (SC5c).
- The source-improvement proposal function has zero caller (SC5d).

All of these gaps were already honestly recorded by the team in `.planning/WINDOWS.md` entries 38–45 before this verification began, and each was independently reproduced here (via grep, direct JSON inspection, and one live CLI invocation) rather than taken on the SUMMARY's word. REQUIREMENTS.md currently marks LEARN-02, LEARN-06, LEARN-07 and LEARN-08 as fully satisfied; this verification's evidence does not support that for the running program, only for the isolated library/test layer.

Recommended next step: a closure plan (or a small set of them) that wires the existing boundary callers — build/continue/swarm/oracle/recovery lanes into the episode-ledger emitters, a registered CLI or automatic trigger into shadow-compare and the promotion gate, and a caller into proposeSourceImprovement — rather than any redesign. Every piece needed already exists and is tested; it needs a caller.

---

_Verified: 2026-09-14T16:20:22Z_
_Verifier: Claude (gsd-verifier)_
