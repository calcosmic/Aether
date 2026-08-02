---
phase: 164-research-feeds-planning
verified: 2026-08-02T18:35:00Z
status: passed
score: 5/5 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 2/5 (1 partial)
  gaps_closed:
    - "Before planning a phase, the Queen states whether that phase needs research and why; the user can override the decision either way, and when research is warranted a worker runs automatically without the user having to invoke it (CR-01: toWorkerDispatches dropped permission_profile)"
    - "Research findings persist to a durable per-phase artifact and appear in the planner's context automatically — not pasted in by hand (CR-02: toWorkerDispatches dropped brief)"
    - "During a live plan run, a confidence readout is visible while research runs, using the existing confidence-loop.ts rather than a reimplementation, and the loop stops at the depth-bound target/iteration budget with an accept override to exit early (downstream of CR-01/CR-02)"
    - "Re-running plan on a phase re-researches from scratch rather than reusing stale findings, and territory survey context (where it exists) reaches both the research worker and the planner (downstream of CR-02)"
    - "A research phase that stalls below target at deep or exhaustive depth escalates to an Oracle, announced with a stated reason, without crashing the plan run (CR-03: plan-research-escalate crashed fresh colonies; unwrapped escalation call crashed the whole plan run)"
  gaps_remaining: []
  regressions: []
deferred: []
human_verification: []
---

# Phase 164: Research Feeds Planning Verification Report

**Phase Goal:** Before a phase is planned, the Queen decides whether it needs research and states why; when it does, a research worker runs automatically and its findings persist and feed the plan directly — restoring v5.4.0 plan.md Step 3.6 "Phase Domain Research" choreography. This phase uses `.aether/ts-host/src/confidence-loop.ts` as-is; it does not reimplement a confidence loop.

**Verified:** 2026-08-02T18:35:00Z
**Status:** passed
**Re-verification:** Yes — after gap closure (plans 164-10, 164-11)

## Goal Achievement

### Observable Truths (mapped from ROADMAP.md Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Queen states research decision + reason, user can override, warranted research runs automatically without the user invoking it | ✓ VERIFIED | Decision/override machinery (`cmd/phase_research_decision.go`, `cmd/phase_research_decision_cmd.go`) unchanged and still passing. The dispatch-boundary defect that previously blocked "runs automatically" (CR-01) is fixed: `toWorkerDispatches` (`.aether/ts-host/src/host.ts:437-482`) now copies `permission_profile` through verbatim. Independently re-derived (not taken from SUMMARY): read the fixed source, ran `dispatch-field-fidelity.test.ts` directly (not through `npm test`) — 4/4 pass, including a test that mirrors Go's `repositoryReadOnlyCastes` map from source at test time. Ran the new Go-side proof `cmd/phase_research_permission_boundary_test.go` directly — `TestPlanResearchDispatchResolvesAtWorkerBoundary` builds a *real* `plannedPhaseResearchDispatches` output, round-trips it through JSON, and resolves it through the actual `internalWorkerConfig`/`ResolvePermissionProfile` chain; passes. The companion `TestScoutReadOnlyProfileIsRejectedAtWorkerBoundary` proves the old stale profile is still correctly rejected, confirming this is a real fix, not a widened acceptance. |
| 2 | Research findings persist to a durable per-phase artifact and appear in the planner's context automatically, not pasted by hand | ✓ VERIFIED | Route-Setter-side injection (`cmd/codex_plan.go` `routeSetterResearchBudgetChars`, `resolvePhaseResearchSection`) unchanged and confirmed present. CR-02 (the artifact-source defect) is fixed: `toWorkerDispatches` now promotes Go's `brief` field to `task_brief` when `task_brief` is not already host-injected (host.ts:405-411), so a real Scout receives its six-section mission instead of a bare one-line task. Confirmed via `TestPlanResearchDispatchCarriesSixSectionMission` (Go) and `dispatch-field-fidelity.test.ts`'s brief-promotion test (TS) — both pass, run directly. |
| 3 | Re-running plan re-researches from scratch instead of reusing stale findings; territory survey context reaches both the research worker and the planner | ✓ VERIFIED | Go-side survey injection and reresearch gating (`cmd/phase_research.go`, Plan 01) unchanged, still passing (`TestReplanReResearchesPhases`, `TestRenderPhaseResearchBriefIncludesSurvey`). The survey context baked into the brief now actually reaches the real worker process because CR-02's brief-drop is fixed (same evidence as truth 2). |
| 4 | Live plan run shows a confidence readout using the existing `confidence-loop.ts` (not reimplemented); loop stops at depth-bound target/iteration budget; accept override exits early | ✓ VERIFIED | `runResearchConfidenceLoop` still constructs one `ConfidenceLoop` per phase (host.ts), depth→target/iteration binding still matches RESEARCH-08 exactly (`researchLoopPreset`: 80/4, 90/6, 95/8, 99/12), `--accept` still honored — none of this needed to change. What was previously false ("scoring either a failed dispatch or a Scout's guess at a one-line task") is now genuinely fixed: the loop scores a real artifact produced by a real Scout dispatch carrying the correct permission profile and full brief. |
| 5 | Queen proposes plan granularity, task decomposition depth, and verification depth together, each with a plain-English reason; user accepts or changes before planning proceeds | ✓ VERIFIED | Unaffected by CR-01/CR-02/CR-03 (never touched the Go↔TS dispatch boundary). `cmd/plan_depth_proposal.go` (`computeDepthProposal`, `renderDepthProposalCard`) re-run: `TestPlanWrapperCardsParity`, `TestDepthProposalCardKnobs` pass. |

**Score:** 5/5 truths verified.

### CR-01 / CR-02 / CR-03 Independent Re-Verification

The previous verification's three blocking findings were independently re-derived from the current code and by executing tests directly (not accepting `npm test`/`go test` aggregate pass counts, and not accepting SUMMARY.md's copy-pasted probe output as evidence):

| Finding | Status | How independently confirmed this run |
|---------|--------|----------------------------------------|
| **CR-01** — `toWorkerDispatches` dropped `permission_profile` | ✓ FIXED | Read `.aether/ts-host/src/host.ts:477-482` — `permission_profile` is copied through unmodified when present. Read `.aether/ts-host/src/worker-dispatch.ts:388-402` — `permissionProfileForCaste`'s read-only predicate now matches `pkg/codex/permission_profile.go:53-55`'s `repositoryReadOnlyCastes` (only `includer`; scout resolves to `workspace_write`). Ran `dispatch-field-fidelity.test.ts` directly via `node --import tsx --test` (bypassing `npm test`'s aggregation) — passes, including the cross-language test that derives Go's read-only set from source text at test time. Ran `cmd/phase_research_permission_boundary_test.go` directly — passes, using a dispatch built by the real `plannedPhaseResearchDispatches`, not a hand-written profile literal (the exact gap that let CR-01 hide previously). Confirmed the fix is present in the compiled artifact: `grep` on `.aether/ts-host/dist/host.js` shows the same `permission_profile` copy-through logic at the corresponding lines. |
| **CR-02** — `toWorkerDispatches` dropped `brief` | ✓ FIXED | Same `host.ts` read confirms `brief` is promoted to `task_brief` (host-injected `task_brief` still wins for build/continue precedence). `TestPlanResearchDispatchCarriesSixSectionMission` (Go, run directly) confirms a real dispatch's `Brief` field contains all six section headings and that they survive the `internalWorkerConfig` resolution step. `dist/host.js` grep confirms the promotion logic is present in the compiled artifact. |
| **CR-03** — `plan-research-escalate` crashed fresh colonies mid-loop; unwrapped escalation call crashed the whole plan run | ✓ FIXED | Read `cmd/phase_research_escalate.go:83-98` — `planResearchEscalateCmd` now seeds `phaseResearchCandidates` from `loadPlanningIterationState()` when available, falling back to the zero-value seed only when no iteration state exists (preserving the refresh/replan path). Ran the three new Go subtests directly (`go test ./cmd/... -run OracleEscalation -v`): `fresh_colony_mid_loop_resolves_phase_from_iteration_state` reproduces the exact fresh-colony shape (empty `state.Plan.Phases` + populated `planning/iteration-state.json`) and resolves correctly; `fresh_colony_unknown_phase_still_returns_clean_error` and `no_iteration_state_falls_back_to_colony_plan` both pass. Read `host.ts:905-960` — the per-candidate `plan-research-escalate` call and the final escalation-dispatch wave are each wrapped in their own try/catch, emitting a named warning and continuing (D-08). Ran `research-escalation-degrade.test.ts` directly — 3/3 pass, and the live run's own stderr output (captured directly, not copied from a SUMMARY) shows the real degrade path firing: `Oracle escalation: phase 7 research stalled at 35%... ` followed by `Warning: Oracle escalation unavailable for phase 7: Go subprocess failed for plan-research-escalate --phase 7: exit status 1;... Planning continues -- research is enrichment, never a gate (D-08).` — the plan run did not crash. |

**Verdict:** All three previously-blocking defects are genuinely fixed, not just claimed fixed. Each was re-derived from source (not taken on SUMMARY's word) and re-executed against real, non-mocked test paths that specifically target the previously-hidden boundary.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/phase_research.go` | Survey-aware brief, replan gating | ✓ VERIFIED | Unchanged from prior verification, still passing |
| `cmd/phase_research_decision.go` | Per-phase recommendation + reason, grounded in runtime signals | ✓ VERIFIED | Unchanged, `researchPhaseKeywords` dependency confirmed absent |
| `.aether/ts-host/src/research-confidence.ts` | Depth binding + evidence scorer | ✓ VERIFIED | Unchanged, `researchLoopPreset` binds 80/4, 90/6, 95/8, 99/12 exactly |
| `cmd/plan_depth_proposal.go` | Three-knob proposal card | ✓ VERIFIED | Unchanged, `TestPlanWrapperCardsParity` passes |
| `cmd/phase_research_decision_cmd.go` | `plan-research-approve` verb | ✓ VERIFIED | Unchanged (WR-07's `--flip`-all-invalid edge case remains open, non-blocking) |
| `.aether/ts-host/src/host.ts` (`toWorkerDispatches`) | Faithful conversion of Go-emitted dispatch fields to the worker request | ✓ VERIFIED (fixed) | `permission_profile` and `brief` now copied through; guarded by a non-mocked field-fidelity invariant test (`dispatch-field-fidelity.test.ts`) that fails on any future silently-dropped field |
| `cmd/phase_research_escalate.go` | `plan-research-escalate` verb + Oracle dispatch builder | ✓ VERIFIED (fixed) | Now resolves phases from the live planning iteration state during fresh-colony mid-loop planning; zero-seed fallback preserved for refresh/replan |
| `.aether/ts-host/test/dispatch-field-fidelity.test.ts` | Non-mocked invariant guarding the Go→worker boundary | ✓ VERIFIED | Ran directly, 4/4 pass; classifies every field as mapped/renamed/intentionally-unmapped |
| `cmd/phase_research_permission_boundary_test.go` | Go-side proof of a real dispatch resolving at the permission boundary | ✓ VERIFIED | Ran directly, 3/3 pass, starts from real `plannedPhaseResearchDispatches` output |
| `.aether/ts-host/test/research-escalation-degrade.test.ts` | Non-mocked proof that a real non-zero escalate exit degrades instead of crashing | ✓ VERIFIED | Ran directly, 3/3 pass; drives the real `aether` binary into a genuine subprocess failure |
| `.aether/ts-host/dist/host.js` | Compiled artifact carrying both fixes | ✓ VERIFIED | `grep` confirms `permission_profile` copy-through and `brief`→`task_brief` promotion present in the compiled output; `tsc --noEmit` clean |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `cmd/codex_plan.go` | `plannedPhaseResearchDispatches` | survey + reresearch args threaded | ✓ WIRED | Unchanged from prior verification |
| `cmd/phase_research_decision.go` | `cmd/pending_decision.go` | `PendingDecision` struct reuse | ✓ WIRED | No new store created (confirmed: no new JSON file in `cmd/phase_research_decision*.go`) |
| `.aether/ts-host/src/research-confidence.ts` | `.aether/ts-host/src/confidence-loop.ts` | `ConfidenceLoopOptions` | ✓ WIRED | Consumed unmodified by the existing `ConfidenceLoop` class |
| `.aether/ts-host/src/host.ts` | `.aether/ts-host/src/confidence-loop.ts` | `new ConfidenceLoop(...)` in `runResearchConfidenceLoop` | ✓ WIRED | Confirmed unchanged |
| `cmd/codex_plan.go` | `computePhaseResearchProposal` | manifest carries proposal + card | ✓ WIRED | Confirmed unchanged |
| `.aether/ts-host/src/host.ts` (`runResearchConfidenceLoop`) | Go worker dispatch (`internal-worker-adapter`) | `toWorkerDispatches` → `dispatchRealWorker` → `internalWorkerConfig`/`ResolvePermissionProfile` | ✓ WIRED (fixed) | `permission_profile` and `brief` now reach the outgoing request; confirmed by direct test execution against both `src/` and `dist/` |
| `.aether/ts-host/src/host.ts` (`runResearchConfidenceLoop`) | `cmd/phase_research_escalate.go` (`plan-research-escalate`) | `_callGoJSONRef` inside try/catch | ✓ WIRED (fixed) | Both the per-candidate call and the final dispatch wave are individually try/catch-guarded; confirmed by direct execution reproducing a real Go subprocess failure and observing graceful degrade, not a crash |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| Scout worker dispatch request | `permission_profile`, `task_brief` | `plannedPhaseResearchDispatches` (Go) → JSON → `toWorkerDispatches` (TS) | Yes — real dispatch builder, no static/mocked fallback in the fixed path | ✓ FLOWING |
| Confidence loop score | `ResearchConfidenceEvaluator` output | Reads the actual research artifact a real Scout dispatch would now be able to produce (previously: dispatch failed or was briefless) | Yes, now that the upstream dispatch delivers real inputs | ✓ FLOWING |
| Route-Setter research context | `resolvePhaseResearchSection` content | `phase-N-research.md` on disk, now producible by a real Scout dispatch | Yes | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `toWorkerDispatches` preserves `permission_profile` for a Go-shaped scout dispatch | `node --import tsx --import ./test/ensure-aether-binary.ts --test test/dispatch-field-fidelity.test.ts` (run directly, not via `npm test`) | 4/4 pass, including cross-language mirror test | ✓ PASS |
| `toWorkerDispatches` preserves `brief` as `task_brief` | Same file, same run | Confirmed heading-preserving promotion, host-injected precedence preserved | ✓ PASS |
| A real fresh-colony `plan-research-escalate` invocation resolves the phase from the in-progress draft | `go test ./cmd/... -run OracleEscalationDispatchNamesTheStall -v` | `fresh_colony_mid_loop_resolves_phase_from_iteration_state` passes; brief names "Wire exporter" resolved from the draft, not the empty colony plan | ✓ PASS |
| A real Go subprocess escalation failure degrades to a warning instead of crashing the plan run | `node --import tsx --import ./test/ensure-aether-binary.ts --test test/research-escalation-degrade.test.ts` (run directly) | 3/3 pass; captured live stderr shows `Warning: Oracle escalation unavailable for phase 7: Go subprocess failed... Planning continues -- research is enrichment, never a gate (D-08).` — no crash | ✓ PASS |
| A real plan-time research dispatch resolves at the Go permission boundary for caste scout | `go test ./cmd/... -run PermissionBoundary -v` | `TestPlanResearchDispatchResolvesAtWorkerBoundary` and `TestScoutReadOnlyProfileIsRejectedAtWorkerBoundary` both pass | ✓ PASS |
| Full ts-host test suite | `npm test` in `.aether/ts-host` | 555/555 pass, 0 failing | ✓ PASS |
| Full Go test suite | `go test ./...` | All packages `ok`, exit code 0 | ✓ PASS |
| `go build ./...` | `go build ./...` | Clean build, no errors | ✓ PASS |
| `go vet ./cmd/... ./pkg/codex/...` | `go vet` | Clean, no findings | ✓ PASS |
| `tsc --noEmit` | `cd .aether/ts-host && npx tsc --noEmit` | Clean, no errors | ✓ PASS |
| Stale doc claim ("Scout's `repository_read_only` profile must remain host-enforced") | `grep` across `.aether/commands/plan.yaml`, `.claude/commands/ant/plan.md`, `.claude/commands/ant-plan.md`, `.opencode/commands/ant/plan.md`, `cmd/command_guide.go` | 0/5 files contain the stale claim; all now state `workspace_write` scoped behaviorally | ✓ PASS (previously FAIL, now fixed) |

### Probe Execution

No `scripts/*/tests/probe-*.sh` files exist for this phase's scope (`find scripts -path '*/tests/probe-*.sh'` returned nothing; neither PLAN nor SUMMARY reference a shell probe script). Step 7c: SKIPPED — no declared or conventional probes for this phase. (Plans 164-10/164-11's "dist probe" verify blocks are inline Node one-liners executed directly against `dist/host.js`, not `scripts/*/tests/probe-*.sh` files — these were independently re-run above under Behavioral Spot-Checks rather than accepted from SUMMARY.md's copy-pasted output.)

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| RESEARCH-01 | 164-02, 164-05, 164-09, 164-11 | Queen decides + states reason + user override; runs automatically | ✓ SATISFIED | Decision/override machinery + fixed dispatch boundary (CR-01) + fixed escalation crash (CR-03) |
| RESEARCH-02 | 164-02, 164-05, 164-11 | Research runs automatically when warranted | ✓ SATISFIED | Same evidence as RESEARCH-01 |
| RESEARCH-03 | 164-07 | Findings persist + injected into planner's context | ✓ SATISFIED | Route-Setter injection + fixed artifact production (CR-01/CR-02) |
| RESEARCH-04 | 164-01, 164-10 | Replan re-researches from scratch | ✓ SATISFIED | Go-side gating correct and tested; real re-research now happens because the dispatch itself succeeds |
| RESEARCH-05 | 164-01, 164-10 | Survey context reaches worker + planner | ✓ SATISFIED | Brief construction correct and now reaches the real worker (CR-02 fixed) |
| RESEARCH-06 | 164-05 | No new planning store | ✓ SATISFIED | `PendingDecision` reused, no new JSON file. **Note:** REQUIREMENTS.md still marks this `[ ]` Pending — a documentation staleness gap, not a functional gap (see Anti-Patterns) |
| RESEARCH-07 | 164-03, 164-06, 164-08, 164-11 | `confidence-loop.ts` used as-is, progress visible | ✓ SATISFIED | Class reused correctly; now grading real dispatches instead of failed/briefless ones |
| RESEARCH-08 | 164-03 | Depth binds target/iteration budget exactly | ✓ SATISFIED | `researchLoopPreset` binds 80/4, 90/6, 95/8, 99/12 exactly, tested. **Note:** REQUIREMENTS.md still marks this `[ ]` Pending — documentation staleness, not a functional gap |
| RESEARCH-09 | 164-04, 164-09 | Three depth controls reachable + explained | ✓ SATISFIED | `computeDepthProposal`, wired into manifest and both wrappers |
| RESEARCH-10 | 164-04, 164-09 | Queen proposes with reason, pre-marked | ✓ SATISFIED | Same evidence as RESEARCH-09 |

**Orphaned requirements:** None — `.planning/REQUIREMENTS.md` lists exactly RESEARCH-01 through RESEARCH-10 for Phase 164, and all ten appear across the eleven plans' `requirements:`/`requirements-completed:` frontmatter.

**Documentation staleness (non-blocking):** `.planning/REQUIREMENTS.md` (lines 84-92, 250-259) marks RESEARCH-06 and RESEARCH-08 as `[ ]` Pending / "Pending" in the traceability table, even though this verification (and the 164-05/164-06 SUMMARY.md `requirements-completed:` frontmatter) independently confirms both are functionally satisfied in the current code. This appears to be an oversight in the gap-closure orchestration (164-10/164-11's tracking commits updated RESEARCH-01/02/04/05/07 but not 06/08, which were never blocked by CR-01/02/03 in the first place). Recommend updating REQUIREMENTS.md's checkboxes for RESEARCH-06 and RESEARCH-08 to Complete as a follow-up documentation fix — this is a paperwork gap, not a code gap, and does not block this phase's goal achievement.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `.planning/REQUIREMENTS.md` | 91, 258 | RESEARCH-06 and RESEARCH-08 checkboxes not updated to Complete despite verified satisfaction | ℹ️ Info | Documentation bookkeeping gap only; does not affect runtime behavior |
| `.aether/ts-host/src/host.ts` | 1342-1358 | Research worker `WorkerResult` always reports `status: "completed"` regardless of the loop's actual per-iteration dispatch outcome (WR-01, carried from post-fix review, confirmed still present) | ⚠️ Warning | The durable completion packet can claim a research worker succeeded even when every dispatch failed; the confidence score staying low is the only signal of trouble. Non-blocking per D-08 ("research is enrichment, never a gate"), but reduces auditability |
| `.aether/ts-host/src/host.ts` | 923-960 | `escalations.push(state.phaseId)` happens immediately after the Go call succeeds, before the dispatch wave is attempted; if `_dispatchWorkersRef` subsequently throws, `escalations` still reports the phase as escalated (WR-02, confirmed still present by direct code read) | ⚠️ Warning | `ResearchLoopSummary.escalations` can over-report successful escalations when the dispatch wave (not the Go call) fails; narrow window, does not crash the run |
| `cmd/phase_research_escalate.go` | 94-98 | `loadPlanningIterationState()` result is trusted without the goal/root staleness check `planningManifestIterationSeed` applies elsewhere (WR-04, carried from review, not addressed by gap closure — correctly out of scope for CR-03's crash fix) | ⚠️ Warning | An abandoned planning loop's leftover iteration-state file could be trusted on a later refresh/replan for the same colony, resolving against a stale draft. Narrow edge case, not the fresh-colony crash CR-03 fixed |
| `.aether/ts-host/src/research-confidence.ts` | 189-205 | `isCited` counts a citation as verified without checking the cited path exists (WR-06, carried from first review) | ⚠️ Warning | Undercuts the "checkable evidence dominates" scoring claim; self-reportable score component |
| `cmd/phase_research_decision_cmd.go` | 153-189 | An entirely-invalid `--flip` still resolves the whole research batch (WR-07, carried from first review) | ⚠️ Warning | User's intended override is silently discarded and cannot be retried |

No `TBD`/`FIXME`/`XXX` debt markers found in any file touched by plans 164-01 through 164-11.

None of the warnings above are must-have truths for this phase's ROADMAP success criteria; all are carried-over robustness/correctness gaps in the surrounding orchestration that the code review (164-REVIEW.md) already surfaced and classified as non-critical. They do not block phase completion but are worth tracking as follow-up work (likely Phase 165+ or a dedicated hardening pass).

### Human Verification Required

None. Every truth in this report was verified via direct source reading, direct (non-aggregated) test execution against non-mocked test files, and direct compiled-artifact inspection. No visual, real-time, or external-service behavior remains ambiguous.

### Gaps Summary

All three critical findings from the previous verification (CR-01, CR-02, CR-03) are now genuinely fixed and independently re-confirmed by this verification — not accepted on SUMMARY.md's word. The fixes were re-derived from the current source, and the specific tests that were added to catch a regression of each defect (`dispatch-field-fidelity.test.ts`, `cmd/phase_research_permission_boundary_test.go`, `research-escalation-degrade.test.ts`, and three new fresh-colony Go subtests) were run directly by this verifier — outside of `npm test`/`go test ./...` aggregation — and all pass, several while directly reproducing the exact failure scenario that used to break in production (a real Go subprocess exit failure was captured live during this run's execution of `research-escalation-degrade.test.ts`, and the loop demonstrably continued instead of crashing).

The phase's central promise — "a research worker runs automatically and its findings persist and feed the plan directly" — now holds for a real, non-simulated `aether host plan` invocation. All 5 ROADMAP success criteria are verified. All 10 RESEARCH requirements are functionally satisfied in code (two — RESEARCH-06 and RESEARCH-08 — have a stale, unrelated checkbox in REQUIREMENTS.md that should be fixed as a documentation follow-up, not a code gap).

Six non-blocking warnings remain (WR-01, WR-02, WR-04, WR-06, WR-07 from the code review, plus the REQUIREMENTS.md checkbox staleness) — all are robustness/correctness/documentation gaps in the surrounding orchestration, not failures of this phase's must-have truths. They are appropriate candidates for a follow-up hardening plan but do not block phase completion.

---

_Verified: 2026-08-02T18:35:00Z_
_Verifier: Claude (gsd-verifier)_
