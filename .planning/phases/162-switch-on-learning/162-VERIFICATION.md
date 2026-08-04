---
phase: 162-switch-on-learning
verified: 2026-08-04T13:59:07Z
status: passed
score: 7/7 must-haves verified (1 gap closed post-verification)
overrides_applied: 0
gaps:
  - truth: "No document claims consolidation is unwired (162-06 must-have)"
    status: resolved
    reason: "CLAUDE.md line 37, inside the 'Definition of Done' historical bullet list, still reads: 'The learning pipeline has been declared restored in v1.10, v1.11, v1.13 and v1.23. `consolidation-phase-end` and `consolidation-seal` still have no caller.' This is now factually false — both subcommands have runtime callers (cmd/codex_continue.go:980, cmd/codex_continue_finalize.go:501, cmd/codex_workflow_cmds.go:425), proven by TestContinueAdvanceInvokesPhaseEndConsolidation and TestRunSealConsolidationRunsAllEightAnts, both passing. Plan 06's action text explicitly targeted three locations (CLAUDE.md:838, AGENTS.md:899, structural-learning-stack.md:15/214/221) and all three were corrected and verified clean. This fourth location, in the same file, was not identified by the plan and slipped through — the docs-truth test's banned-substring list ('no lifecycle command invokes', 'do not invoke either one yet', etc.) does not match this line's exact phrasing ('still have no caller'), so cmd/docs_truth_test.go does not catch it."
    artifacts:
      - path: "CLAUDE.md"
        issue: "Line 37 asserts, in present tense, that consolidation-phase-end and consolidation-seal have no caller — contradicted by this same phase's own implementation and tests"
    missing:
      - "Reword CLAUDE.md line 37 to past tense (e.g. '...consolidation-phase-end and consolidation-seal had no caller until v1.25 Phase 162.') or remove the now-inaccurate clause, consistent with how the other three CLAUDE.md/AGENTS.md/structural-learning-stack.md locations were corrected"
      - "Consider adding 'still have no caller' (or a normalized variant) to cmd/docs_truth_test.go's retiredLearningDocClaims so this class of staleness cannot recur"
---

# Phase 162: Switch On Learning — Verification Report

**Phase Goal:** `pkg/memory/pipeline.go` already wires Observe -> Promote -> Queen -> Consolidate. It is constructed in exactly two places (`consolidation-phase-end`, `consolidation-seal`) and neither is invoked by any wrapper, playbook, or Go call site — the colony described in CLAUDE.md has never learned anything. This phase invokes the two subcommands that already exist; it does not build a learning system. It also reconciles the fact that a second, different learning system (`pkg/learn`) already runs live on every continue, and makes the Hive Brain's off-by-default behaviour a deliberate, written decision instead of an accident.

**Verified:** 2026-08-04T13:59:07Z
**Status:** gaps_found (one minor documentation-staleness gap; all functional and decision-record must-haves verified)
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (merged: 5 ROADMAP Success Criteria + 1 plan-level truth not subsumed by them)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Running `/ant-continue` at the end of a phase visibly reports learning activity that did not appear before this phase shipped (LEARN-01) | ✓ VERIFIED | `runPhaseEndConsolidation` called at `cmd/codex_continue.go:980` (after the atomic `COLONY_STATE.json` write) and `cmd/codex_continue_finalize.go:501` (after `advanceExternalContinue` returns nil). `renderLearningBeat`/`continueLearningFlowStep` render a 🧠 Learning stage in 4 states. `TestContinueAdvanceInvokesPhaseEndConsolidation`, `TestExternalContinueAdvanceInvokesPhaseEndConsolidation`, and the negative test `TestContinueWithoutAdvanceDoesNotConsolidate` all pass (verified by direct execution, not SUMMARY claim) |
| 2 | Running `/ant-seal` visibly reports the full eight-ant consolidation pass, instinct decay, and an archive/report artifact being written (LEARN-02) | ✓ VERIFIED | `runSealConsolidation()` calls `curation.NewOrchestrator(store, bus).Run(ctx, false)` directly and writes `<.aether>/CURATION-REPORT.md`; called from `completeSealRuntime` at `cmd/codex_workflow_cmds.go:425`. `TestRunSealConsolidationRunsAllEightAnts`, `TestRunSealConsolidationWritesReportArtifact`, `TestSealRendersEightNamedAnts`, `TestSealRendersReportPath` all pass |
| 3 | A written decision states which of `pkg/learn` or `pkg/memory` is authoritative and what happens to the other (LEARN-03) | ✓ VERIFIED | `.aether/docs/learning-system-authority.md` Decision 1/2/3 name `pkg/memory` authoritative, `pkg/learn`'s Entry/Evidence layer explicitly subordinate, `pkg/learn/wrappers.go` an excluded shim, and the QUEEN.md double-promotion resolved by `TestSealDoesNotDoublePromoteInstincts` (passes) |
| 4 | A written decision states whether the Hive Brain default changes from off to on, with reasoning (LEARN-04) | ✓ VERIFIED | Same document, Decision 4, full `AETHER_HIVE_POLICY` resolution table. `cmd/hive_policy.go` implements exactly that table; `TestHiveRuntimePolicyDefault` (11 subtests) and `TestHiveRuntimePolicyUnrecognizedWarns` pass |
| 5 | The same worker task, run once with colony memory populated and once wiped, produces demonstrably different output, recorded as a before/after (LEARN-05) | ✓ VERIFIED | Layer 1: `TestWorkerBriefMemoryInjection` (sentinel + strict-length-comparison) and `TestResolveCodexWorkerContextCarriesMemory` pass. Layer 2: `162-MEMORY-PROOF.md` records a real `aether build 1 --print-brief` run against a scratch colony — Context Capsule measurably shrinks 958→615 chars, `## Active Instincts` and `## LOCAL QUEEN WISDOM` sections disappear, restoration reproduces 958 chars exactly |
| 6 | No document claims consolidation is unwired, and none claims phase-end runs three named ants (162-06 plan-level must-have) | ⚠️ PARTIAL | The false "three ants only: nurse → herald → janitor" claim is fully corrected in `structural-learning-stack.md` (grep confirms zero matches across all three target files). However `CLAUDE.md:37` — outside the plan's own list of three targeted locations — still states in present tense that `consolidation-phase-end` and `consolidation-seal` "still have no caller," which is now false. See Gaps Summary. |

**Score:** 6/7 (5 ROADMAP criteria fully verified; the plan-level "no unwired claims" truth is 90% closed with one residual line missed by the plan's own scope)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/hive_policy.go` | Single hive control surface, promote-by-default, explicit off, fail-safe on typo | ✓ VERIFIED | Read in full; `case "off":`, `case "":`, `default:` (warn + off) all present exactly as specified |
| `cmd/hive_policy_test.go` | Table test pinning every env-var value | ✓ VERIFIED | `TestHiveRuntimePolicyDefault` runs 11 subtests, all pass |
| `cmd/hive.go` | Consent-file mechanism removed | ✓ VERIFIED | `grep -rn 'hiveRetrievalOptedIn\|writeHiveRetrievalConsent\|hiveRetrievalConsentPath\|hiveRetrievalConsentFile\|enableHiveForTest\|hive-opt-in\|hive-opt-out' cmd/` returns zero matches (only the banned-substring literal inside `docs_truth_test.go`) |
| `cmd/graph_consolidation_cmds.go` | `consolidationQueenPath()` resolving to the read file | ✓ VERIFIED | Function exists at line 46, used at all 3 `QueenPath`/`NewDryRunConsolidationService` call sites; zero bare `"QUEEN.md"` literals remain |
| `cmd/queen.go` | `ensureQueenInstinctsSection()` + `## Instincts` in default template | ✓ VERIFIED | Function present at line 693; `## Instincts` header present at line 40 |
| `cmd/context.go` | `readQUEENMd` ingests `Instincts` section | ✓ VERIFIED | `sectionName == "Instincts"` and prefix variant both present at lines 1530/1534 |
| `cmd/consolidation_lifecycle.go` | `runPhaseEndConsolidation`, `runSealConsolidation`, D-05 warning wording | ✓ VERIFIED | Both functions present; `"phase advanced WITHOUT consolidation — "` literal present |
| `cmd/codex_visuals.go` | 9 curation-ant caste identities, `renderLearningBeat`, `renderSealConsolidationBeats` | ✓ VERIFIED | All 9 keys (sentinel/nurse/critic/herald/janitor/archivist/librarian/scribe/curator) present ×3 in emoji/color/label maps; both render functions present |
| `.aether/docs/learning-system-authority.md` | ADR covering authority, duplicated stages, LearningValidator, Hive default | ✓ VERIFIED | Read in full; all 4 decisions present, Consequences table maps every claim to a named passing test |
| `cmd/docs_truth_test.go` | Regression guard against retired claims re-appearing | ✓ VERIFIED (but scope-incomplete — see gap) | Test exists and passes; its banned-substring list does not include the exact phrase in CLAUDE.md:37, so that specific stale claim is not caught |
| `cmd/memory_injection_test.go` | Layer 1 deterministic populated-vs-wiped proof | ✓ VERIFIED | Both subtests pass; length-invariant assertion present, not just header presence |
| `.planning/phases/162-switch-on-learning/162-MEMORY-PROOF.md` | Layer 2 recorded exhibit | ✓ VERIFIED | Non-empty, names exact commands, shows two distinguishable real outputs with a stated diff, confirms restoration |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `cmd/context_weighting.go` | hive `wisdom.json` read | `automaticHiveReadEnabled()` only | ✓ WIRED | No second gate found in the file |
| `cmd/colony_prime_context.go` | `result.Warnings` | unconditional fallback surfacing | ✓ WIRED | Confirmed by 162-01 SUMMARY and passing `TestColonyPrimeGracefulWithMissingData` |
| `pkg/memory QueenService.PromoteInstinct` | `.aether/QUEEN.md ## Instincts` | `consolidationQueenPath()` in `PipelineConfig.QueenPath` | ✓ WIRED | `TestConsolidationPromotesIntoLocalQueen` passes |
| `.aether/QUEEN.md ## Instincts` | `buildColonyPrimeOutput().PromptSection` | `readQUEENMd` ingesting Instincts | ✓ WIRED | `TestPromotedInstinctReachesWorkerPrompt` passes |
| `cmd/codex_continue.go` | `runPhaseEndConsolidation` | call beside `captureContinueLearning`, after atomic write | ✓ WIRED | Line 980, positioned correctly (confirmed by direct code read) |
| `cmd/codex_continue_finalize.go` | `runPhaseEndConsolidation` | call after `advanceExternalContinue` returns nil | ✓ WIRED | Line 501, positioned correctly |
| `phaseEndConsolidationSummary` | `renderContinueVisual` stdout | `renderStageMarker("Learning")` | ✓ WIRED | Line 1567/1595 |
| `completeSealRuntime` | `runSealConsolidation` | direct call before promotion loop | ✓ WIRED | Line 425, before the `queenAlreadyPromoted` skip-set loop at 438+ |
| `curation.CurationResult.Steps` | seal stdout | per-ant caste-styled beats | ✓ WIRED | `renderSealConsolidationBeats` at line 1624 |
| scribe `StepResult` report | `.aether/CURATION-REPORT.md` | runtime writes artifact | ✓ WIRED | Line 282 in `consolidation_lifecycle.go`; `TestSealRendersReportPath` passes |
| `cmd/docs_truth_test.go` | `.aether/docs/learning-system-authority.md` | existence + content assertion | ✓ WIRED | `TestLearningDecisionRecordExists` passes |

### Data-Flow Trace (Level 4)

Not applicable in the traditional UI-data sense — this phase's "data flow" is the prompt-injection pipeline verified directly by `162-MEMORY-PROOF.md` and `TestWorkerBriefMemoryInjection`. Both are real-data traces (real instinct → real QUEEN.md write → real worker prompt text), not mocked or stubbed at any point in the chain checked.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full `cmd` package test suite (all phase-162 tests plus full regression) | `go test ./cmd/... -count=1` | `ok github.com/calcosmic/Aether/cmd 285.685s`, zero failures | ✓ PASS |
| Build integrity | `go build ./...` | clean, no output | ✓ PASS |
| Static analysis | `go vet ./...` | clean, no output | ✓ PASS |
| Hive policy resolution table | `go test ./cmd/ -run TestHiveRuntimePolicy -count=1 -v` | all 11+2 subtests pass, warning fires exactly once | ✓ PASS |
| Consolidation reachability | `go test ./cmd/ -run 'TestConsolidationQueenPath\|TestPromotedInstinctReachesWorkerPrompt\|TestConsolidation.*DryRun' -count=1 -v` | all pass | ✓ PASS |
| Continue-path wiring (positive + negative) | `go test ./cmd/ -run 'ContinueAdvanceInvokesPhaseEndConsolidation\|ContinueWithoutAdvanceDoesNotConsolidate' -count=1 -v` | all pass, including the negative "no advance → no consolidation" case | ✓ PASS |
| Seal-path wiring, double-promotion invariant | `go test ./cmd/ -run 'TestRunSealConsolidation\|TestSeal' -count=1 -v` | all pass (29 seal-related tests) | ✓ PASS |
| Memory-injection proof | `go test ./cmd/ -run 'TestWorkerBriefMemoryInjection\|TestResolveCodexWorkerContextCarriesMemory' -count=1 -v` | all pass | ✓ PASS |
| Docs-truth regression guard | `go test ./cmd/ -run 'TestLearningDocsDoNotClaimUnwiredConsolidation\|TestLearningDecisionRecordExists' -count=1 -v` | both pass | ✓ PASS |
| Caste-map parity | `go test ./cmd/ -run 'TestCasteEmojiMapCompleteness\|TestCodexVisualParity\|TestCasteIdentityAllCastes' -count=1 -v` | all pass | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes declared by this phase's PLAN/SUMMARY files, and none found under `scripts/` matching that convention. Step 7c: SKIPPED (no declared or conventional probes for this phase).

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|-----------------|--------------|--------|----------|
| LEARN-01 | 162-02, 162-03 | `consolidation-phase-end` runs at end of every phase | ✓ SATISFIED | Wired at 2 call sites, 3 wiring tests including a negative case, all passing |
| LEARN-02 | 162-04 | `consolidation-seal` runs at seal — full 8-ant pass, decay, archive, scribe report | ✓ SATISFIED | `runSealConsolidation` wired, report artifact written, 4 dedicated tests passing |
| LEARN-03 | 162-02, 162-04, 162-06 | Two competing learning systems reconciled, one authoritative | ✓ SATISFIED | ADR + no-double-promotion invariant test passing |
| LEARN-04 | 162-01, 162-06 | Hive Brain default decided deliberately | ✓ SATISFIED | Policy flip implemented and tested; decision recorded in ADR |
| LEARN-05 | 162-05 | Worker output demonstrably differs, memory populated vs wiped | ✓ SATISFIED | Two-layer proof (deterministic test + recorded real exhibit) |

No orphaned requirements: `grep -n "Phase 162" .planning/REQUIREMENTS.md` returns exactly LEARN-01..05, all marked Complete, matching the five requirement IDs declared across the six plans' frontmatter.

### Anti-Patterns Found

Scanned every file this phase modified (17 `cmd/` files, `pkg/events/ceremony.go`, and the 3 corrected docs plus the new ADR) for `TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER` and placeholder-style prose. Zero matches in any file. No debt markers, no empty stub implementations (`return null`/`return {}`/`=> {}`), no hardcoded-empty data flowing to rendering.

### Human Verification Required

None. Every observable truth in this phase is either a Go-level runtime behavior (verified by direct test execution against the current codebase, not by trusting SUMMARY.md) or a documentation claim (verified by direct file reads and grep against the actual committed text). No UI, visual, or subjective-quality judgment is involved in this phase's deliverables.

### Gaps Summary

One gap, minor and narrowly scoped:

**CLAUDE.md line 37** — inside the "Definition of Done" section's historical bullet list (the exact section that names this pipeline as its own canonical example of the failure being avoided) — still reads: *"The learning pipeline has been declared restored in v1.10, v1.11, v1.13 and v1.23. `consolidation-phase-end` and `consolidation-seal` still have no caller."* The word "still" makes this a present-tense claim, and it is now false: both subcommands have runtime callers, proven by this same phase's own passing tests (`TestContinueAdvanceInvokesPhaseEndConsolidation`, `TestRunSealConsolidationRunsAllEightAnts`).

This was not a fabricated or hallucinated SUMMARY claim — the engineering work is real and thoroughly tested. It is a genuine documentation-scope miss: Plan 06's `<interfaces>` section explicitly enumerated three locations to fix (`CLAUDE.md:838`, `AGENTS.md:899`, `structural-learning-stack.md:15/214/221`), and all three were corrected and are verified clean by direct grep. Line 37, in the same file, was not on that list and was not touched. `cmd/docs_truth_test.go`'s banned-substring list (`"no lifecycle command invokes"`, `"do not invoke either one yet"`, etc.) does not match this line's different phrasing (`"still have no caller"`), so the phase's own regression guard does not catch it either.

Everything else — all 5 ROADMAP success criteria, all 5 requirement IDs, every artifact, every key link, the full `cmd` package test suite (285.7s, zero failures), and both layers of the LEARN-05 memory-injection proof — is genuinely and rigorously implemented, wired, and verified against the live codebase.

**This looks like an oversight, not an intentional deviation** — no override is appropriate. The fix is a one-line text correction (reword to past tense or remove the now-inaccurate clause) plus, optionally, extending `cmd/docs_truth_test.go`'s banned-substring list to also catch `"still have no caller"` so this exact class of staleness cannot recur, closing the loop the same way the other three locations already are.

---

*Verified: 2026-08-04T13:59:07Z*
*Verifier: Claude (gsd-verifier)*

## Post-Verification Gap Closure (2026-08-04)

The single gap (stale CLAUDE.md line 37 "still have no caller" claim) was closed
by the orchestrator immediately after verification:

- CLAUDE.md:37 reworded to the past-tense historical record ("had no caller
  until v1.25 (Phase 162) wired them into the continue and seal paths").
- `cmd/doc_consolidation_claims_test.go` guard updated to pin the corrected
  historical sentence instead of the now-false one (its own trip-wire message
  anticipated exactly this Phase 162 update).
- "still have no caller" added to `retiredLearningDocClaims` in
  `cmd/docs_truth_test.go` per this report's hardening suggestion.

Full `go test ./...` green after all three changes.
