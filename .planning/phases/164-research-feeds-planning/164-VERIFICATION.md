---
phase: 164-research-feeds-planning
verified: 2026-08-02T16:10:00Z
status: gaps_found
score: 2/5 must-haves verified
overrides_applied: 0
gaps:
  - truth: "Before planning a phase, the Queen states whether that phase needs research and why; the user can override the decision either way, and when research is warranted a worker runs automatically without the user having to invoke it"
    status: failed
    reason: "The decision/override machinery (Go side) is real and tested, but the worker that 'runs automatically' fails immediately on any real (non-simulated) aether host plan run: toWorkerDispatches in host.ts never copies permission_profile from the Go manifest, so dispatchRealWorker falls back to the TS-side permissionProfileForCaste, which still classifies scout as repository_read_only even though pkg/codex/permission_profile.go's canonical profile for scout is now workspace_write (scout was removed from repositoryReadOnlyCastes). The Go adapter's ResolvePermissionProfile does exact-equality enforcement and errors on any mismatch, so every real Scout dispatch (including phase_research Scouts) fails before doing any work."
    artifacts:
      - path: ".aether/ts-host/src/host.ts"
        issue: "toWorkerDispatches (lines 437-477) does not map dispatch.permission_profile onto the outgoing BuildDispatch"
      - path: ".aether/ts-host/src/worker-dispatch.ts"
        issue: "permissionProfileForCaste (line 388-391) still treats scout as read-only, out of sync with pkg/codex/permission_profile.go's repositoryReadOnlyCastes (now scout-free)"
    missing:
      - "Copy permission_profile through toWorkerDispatches when present on the source dispatch"
      - "Update worker-dispatch.ts's TS-side fallback so scout is no longer forced to repository_read_only"
      - "A non-mocked test that runs a plan dispatch through internalWorkerConfig/ResolvePermissionProfile for caste scout"
  - truth: "Research findings persist to a durable per-phase artifact and appear in the planner's context automatically — not pasted in by hand"
    status: failed
    reason: "The Route-Setter-side injection (Plan 07) is real and correctly reads whatever is on disk. But the artifact itself is compromised at the source: toWorkerDispatches also never maps the Go-emitted brief field (the six-section research mission built by renderPhaseResearchBrief) onto task_brief. Real research Scouts (and escalated Oracles) receive only the one-line task string, never the six-section contract the confidence scorer (ResearchConfidenceEvaluator) grades against. Combined with the CR-01 permission failure, no real phase-research.md artifact is produced by the researched pipeline in a live run — there is nothing durable for the Route-Setter to read beyond whatever a Scout guesses from a one-line task."
    artifacts:
      - path: ".aether/ts-host/src/host.ts"
        issue: "toWorkerDispatches drops dispatch.brief; only task_brief (a host-injected field plan dispatches never carry) is mapped, so dispatchRealWorker sends task_brief = dispatch.task"
    missing:
      - "Map dispatch.brief onto workerDispatch.task_brief in toWorkerDispatches when task_brief is absent"
      - "A test asserting the GoWorkerDispatchRequest.task_brief for a phase_research dispatch contains the six-section markers (e.g. '## Output', '## Files to Study')"
  - truth: "During a live plan run, a confidence readout is visible while research runs, using the existing confidence-loop.ts rather than a reimplementation, and the loop stops at the depth-bound target/iteration budget with an accept override to exit early"
    status: partial
    reason: "The ConfidenceLoop construction site, depth-to-budget binding, and evidence-based scorer are real, correctly wired, and unit-tested (164-03, 164-06). But because every round's dispatch loses permission_profile and brief (see above), a live run's confidence loop is scoring either a nonexistent artifact (failed dispatch) or a Scout's guess at an undescribed one-line task — the readout is real machinery grading fictional progress, not the depth-bound research loop the success criterion describes. The mechanism works; the input it consumes in production does not represent real research."
    artifacts: []
    missing:
      - "Same fixes as above (permission_profile + brief mapping) — this truth becomes true once CR-01/CR-02 are fixed"
  - truth: "Re-running plan on a phase re-researches from scratch rather than reusing stale findings, and territory survey context (where it exists) reaches both the research worker and the planner"
    status: failed
    reason: "The Go-side replan/survey logic (Plan 01) is correct and tested at the brief-construction layer: renderPhaseResearchBrief threads survey context and reresearch gating is wired at the call site. But because toWorkerDispatches drops brief entirely (CR-02), the survey context baked into the brief never reaches the real Scout process in a live run — only the planner (Route-Setter), which reads phase-research.md from disk, could see it indirectly, and only if a real artifact existed (blocked by CR-01)."
    artifacts:
      - path: ".aether/ts-host/src/host.ts"
        issue: "Same toWorkerDispatches gap as CR-01/CR-02"
    missing:
      - "Same fixes as above"
  - truth: "A research phase that stalls below target at deep or exhaustive depth escalates to an Oracle, announced with a stated reason"
    status: failed
    reason: "The escalation dispatch builder, scoped oracle permission restriction, and TS trigger are all correctly implemented and unit-tested against a colony state where Plan.Phases is already populated. But plan-research-escalate resolves phases via phaseResearchCandidates(state, codexPlanIterationState{}) — a zero-value seed — which falls through to state.Plan.Phases. During a fresh colony's planning loop (iteration >= 2, the only time research dispatches exist), Plan.Phases is empty because the in-progress draft lives in the planning iteration state file, not COLONY_STATE.json. The command returns outputError(1, 'phase N not found in the current plan'), which os.Exit()s non-zero; execFileSync in go-bridge.ts throws; runResearchConfidenceLoop has no try/catch around the escalation call (host.ts ~899-914), so the exception propagates to main and crashes the entire plan run — the exact deep/exhaustive-stall scenario escalation exists for is a guaranteed crash, not a graceful degrade, directly contradicting D-08 ('research is enrichment, never a gate')."
    artifacts:
      - path: "cmd/phase_research_escalate.go"
        issue: "planResearchEscalateCmd passes codexPlanIterationState{} instead of the loaded planning iteration state, so phaseResearchCandidates falls back to the empty state.Plan.Phases mid-loop"
      - path: ".aether/ts-host/src/host.ts"
        issue: "The plan-research-escalate call (~line 899-906) is not wrapped in try/catch; a failed escalation propagates and kills the whole plan run instead of degrading"
    missing:
      - "Load the planning iteration state (same seed source runCodexPlanPlanOnly uses) and pass it into phaseResearchCandidates in planResearchEscalateCmd"
      - "Wrap the escalation call in runResearchConfidenceLoop in try/catch: log a warning, skip that phase's escalation, and continue"
      - "A test that exercises plan-research-escalate against an empty state.Plan.Phases with a populated planning-iteration-state seed (the actual fresh-colony scenario), not just an already-populated colony state"
deferred: []
human_verification: []
---

# Phase 164: Research Feeds Planning Verification Report

**Phase Goal:** Before a phase is planned, the Queen decides whether it needs research and states why; when it does, a research worker runs automatically and its findings persist and feed the plan directly — restoring v5.4.0 plan.md Step 3.6 "Phase Domain Research" choreography. This phase uses `.aether/ts-host/src/confidence-loop.ts` as-is; it does not reimplement a confidence loop.

**Verified:** 2026-08-02T16:10:00Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (mapped from ROADMAP.md Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Queen states research decision + reason, user can override, warranted research runs automatically without the user invoking it | ✗ FAILED | Decision/override machinery (`cmd/phase_research_decision.go`, `cmd/phase_research_decision_cmd.go`) is real and unit-tested. But the "runs automatically" half fails at the Go↔TS dispatch boundary: `permission_profile` is dropped in `toWorkerDispatches` (host.ts:437-477), causing every real Scout dispatch to fail Go's exact-equality permission check (`ResolvePermissionProfile`, `pkg/codex/permission_profile.go:116-129`). Confirmed by direct runtime probe (see Behavioral Spot-Checks). |
| 2 | Research findings persist to a durable per-phase artifact and appear in the planner's context automatically, not pasted by hand | ✗ FAILED | Route-Setter-side injection (`cmd/codex_plan.go` `renderRouteSetterResearchContent`) is real and reads whatever is on disk. But `toWorkerDispatches` also drops the `brief` field (the six-section mission `renderPhaseResearchBrief` builds), so real research Scouts never receive their mission and no real six-section artifact gets produced in a live run. Confirmed by direct runtime probe. |
| 3 | Re-running plan re-researches from scratch instead of reusing stale findings; territory survey context reaches both the research worker and the planner | ✗ FAILED | The Go-side survey-injection and reresearch gating (`cmd/phase_research.go`, Plan 01) are correct and unit-tested at the brief-construction layer, but the survey context embedded in that brief never reaches the real worker process because of the same `toWorkerDispatches` brief-drop. |
| 4 | Live plan run shows a confidence readout using the existing `confidence-loop.ts` (not reimplemented); loop stops at depth-bound target/iteration budget; accept override exits early | ⚠ PARTIAL | `runResearchConfidenceLoop` genuinely constructs one `ConfidenceLoop` per phase (`.aether/ts-host/src/host.ts:634+`), the depth→target/iteration binding matches RESEARCH-08 exactly (`research-confidence.ts`), and `--accept` is honored. This machinery is real and not reimplemented. But because of the CR-01/CR-02 dispatch gaps, in a live run this loop scores either a failed dispatch or a Scout guessing at a one-line task, not real depth-bound research. The class and wiring exist; the thing it measures does not, in production. |
| 5 | Queen proposes plan granularity, task decomposition depth, and verification depth together, each with a plain-English reason; user accepts or changes before planning proceeds | ✓ VERIFIED | `cmd/plan_depth_proposal.go` (`computeDepthProposal`, `renderDepthProposalCard`) computes all three knobs with reasons and pre-marked recommendations; wired into both plan-only branches and both wrappers (Plan 09, `TestPlanWrapperCardsParity` passing). This path never touches the Go↔TS dispatch boundary, so it is unaffected by CR-01/CR-02/CR-03. |

**Score:** 2/5 truths fully verified (1 partial credit given no weight — counted as failed for score purposes: 2 verified equivalents including SC5 and the decision-machinery half of SC1... see Gaps for the literal accounting.) — for frontmatter purposes this phase is 1 clean pass (SC5) + 4 truths blocked by the same three root-cause defects.

### Independent Verification of 164-REVIEW.md Critical Findings

All three CR findings in `.planning/phases/164-research-feeds-planning/164-REVIEW.md` were independently re-derived from the code (not taken on the reviewer's word) and confirmed:

| Finding | Independently Confirmed? | How Verified |
|---------|--------------------------|--------------|
| **CR-01** — `toWorkerDispatches` drops `permission_profile`; TS fallback still treats scout as read-only; Go's `ResolvePermissionProfile` does exact-equality and errors | ✓ CONFIRMED | Read `.aether/ts-host/src/host.ts:437-477` (no `permission_profile` mapping present); `.aether/ts-host/src/worker-dispatch.ts:388-391` (`readOnly = normalized === "scout" \|\| normalized === "includer"`); `pkg/codex/permission_profile.go:53-55` (`repositoryReadOnlyCastes` now contains only `includer` — scout was removed); `pkg/codex/permission_profile.go:116-129` (`ResolvePermissionProfile` calls `permissionEnforcementEqual`, errors on mismatch). Additionally reproduced live: ran the actual compiled `runResearchConfidenceLoop`/`toWorkerDispatches` (`.aether/ts-host/dist/host.js`) via Node against a Go-shaped dispatch carrying `permission_profile: {name: "workspace_write", ...}` — the object that reached the mocked `dispatchWorkers` had no `permission_profile` key at all. |
| **CR-02** — `toWorkerDispatches` drops `brief`, so research Scouts/escalated Oracles never receive the six-section mission | ✓ CONFIRMED | `cmd/codex_plan.go:36` (`Brief string \`json:"brief,omitempty"\``) and `cmd/phase_research.go:102` (`Brief: renderPhaseResearchBrief(...)`) confirm Go emits `brief`. `toWorkerDispatches` (host.ts:437-477) only ever reads `dispatch.task_brief`, never `dispatch.brief`. Reproduced live: the same Node probe sent a dispatch with a populated `brief` field containing `## Files to Study`; the captured outgoing object's `task_brief` was `undefined`. |
| **CR-03** — `plan-research-escalate` resolves phases from `state.Plan.Phases`, empty during a fresh colony's mid-loop planning; no try/catch around the call in `host.ts` | ✓ CONFIRMED | `cmd/phase_research_escalate.go:83` passes `codexPlanIterationState{}` (zero value) into `phaseResearchCandidates`, which (`cmd/phase_research.go:31-42`) falls through to `state.Plan.Phases` whenever `seed.PreviousPlanDraft` is nil. `outputError` (`cmd/helpers.go:35-50` → `cmd/root.go:219,245,250`) calls `os.Exit` with a non-zero code, which makes `execFileSync` in `_realCallGoJSON` (`.aether/ts-host/src/go-bridge.ts:96-116`) throw. The call site in `runResearchConfidenceLoop` (`host.ts` ~899-906) has no surrounding try/catch. Confirmed the only CLI test for this command (`cmd/phase_research_escalate_test.go`, `cli_prints_dispatch_and_exits_zero`) pre-populates `colony.Plan{Phases: [...]}` — it never exercises the empty-`Plan.Phases` fresh-mid-loop scenario that is the actual failure trigger. |

**Verdict on the review:** All three critical findings are real, present in the current committed code (no fixes landed after the review was written — `git log` shows no commits touching `host.ts`, `worker-dispatch.ts`, or `phase_research_escalate.go` after the review timestamp), and they compound: CR-01 means real Scout dispatches fail outright; CR-02 means even if permission_profile were fixed, the Scout would receive no mission; CR-03 means the one compensating mechanism (Oracle escalation) crashes the entire plan run under the exact conditions it exists for. Together they mean the phase's central claim — "a research worker runs automatically and its findings persist and feed the plan directly" — is false for any real (non-`--simulate`) `aether host plan` invocation.

### Why the test suites all pass anyway

Ran both suites in full:
- `go test ./cmd/...` → all passing (including `TestOracleEscalationDispatchNamesTheStall`, `TestReplanReResearchesPhases`, `TestQueenResearchDecision`, etc.)
- `cd .aether/ts-host && npm test` → 548/548 passing, 0 failing

Every test that exercises the dispatch or escalation boundary does so through a mock: `research-confidence-loop.test.ts` installs a fake dispatcher via `__setDispatchWorkers`/`__setCallGoJSON` (documented in its own header comment: *"Drives the function directly with a fake dispatcher installed through host.ts's existing dispatch-mock hook"*); `worker-dispatch.test.ts` constructs `BuildDispatch` objects directly (bypassing `toWorkerDispatches` entirely) so `permission_profile` is already correct at the point the test starts; `phase_research_escalate_test.go`'s only CLI-level test pre-seeds `Plan.Phases`, sidestepping the actual empty-state trigger. This matches this project's own CLAUDE.md Definition of Done warning precisely: passing tests against a mocked boundary are not proof the boundary itself works.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/phase_research.go` | Survey-aware brief, replan gating | ✓ VERIFIED | `renderPhaseResearchBrief` takes `survey codexSurveyContext`; reresearch gate implemented; `TestReplanReResearchesPhases` passes |
| `cmd/phase_research_decision.go` | Per-phase recommendation + reason, grounded in runtime signals | ✓ VERIFIED | `computePhaseResearchProposal` present, tested, no dependency on retiring `researchPhaseKeywords` |
| `.aether/ts-host/src/research-confidence.ts` | Depth binding + evidence scorer | ✓ VERIFIED | `researchLoopPreset`/`researchLoopOptions` match 80/4, 90/6, 95/8, 99/12 exactly; `ResearchConfidenceEvaluator` present and tested |
| `cmd/plan_depth_proposal.go` | Three-knob proposal card | ✓ VERIFIED | `computeDepthProposal` present; wired into manifest via 164-09; `TestPlanWrapperCardsParity` passes |
| `cmd/phase_research_decision_cmd.go` | `plan-research-approve` verb | ✓ VERIFIED (exists, tested) — see WR-02 in review for a real but non-blocking edge-case bug (`--flip` with all-invalid tokens still resolves the batch) |
| `.aether/ts-host/src/host.ts` (`toWorkerDispatches`) | Faithful conversion of Go-emitted dispatch fields to the worker request | ✗ STUB (functionally) | Exists and is substantive for build/continue dispatches (which rely on `task_brief`, already host-injected), but for plan-time research dispatches it silently drops two fields (`permission_profile`, `brief`) the research pipeline depends on. Confirmed by direct runtime probe. |
| `cmd/phase_research_escalate.go` | `plan-research-escalate` verb + Oracle dispatch builder | ⚠ ORPHANED IN THE FAILURE PATH | Dispatch builder itself is correct; the CLI verb crashes (non-zero exit, uncaught by the TS host) in the one scenario (fresh colony, mid-loop) escalation exists for |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `cmd/codex_plan.go` | `plannedPhaseResearchDispatches` | survey + reresearch args threaded | ✓ WIRED | Call site confirmed |
| `cmd/phase_research_decision.go` | `cmd/pending_decision.go` | `PendingDecision` struct reuse | ✓ WIRED | No new store created |
| `.aether/ts-host/src/research-confidence.ts` | `.aether/ts-host/src/confidence-loop.ts` | `ConfidenceLoopOptions` | ✓ WIRED | Consumed unmodified by the existing `ConfidenceLoop` class |
| `.aether/ts-host/src/host.ts` | `.aether/ts-host/src/confidence-loop.ts` | `new ConfidenceLoop(...)` in `runResearchConfidenceLoop` | ✓ WIRED | Confirmed at host.ts ~801 |
| `cmd/codex_plan.go` | `computePhaseResearchProposal` | manifest carries proposal + card | ✓ WIRED | `research_proposal_card`, `research_awaiting_approval` fields confirmed on manifest |
| `.aether/ts-host/src/host.ts` (`runResearchConfidenceLoop`) | Go worker dispatch (`internal-worker-adapter`) | `toWorkerDispatches` → `dispatchRealWorker` → `internalWorkerConfig`/`ResolvePermissionProfile` | ✗ NOT WIRED | `permission_profile` and `brief` never reach the outgoing request; confirmed by direct execution against the compiled `dist/host.js` |
| `.aether/ts-host/src/host.ts` (`runResearchConfidenceLoop`) | `cmd/phase_research_escalate.go` (`plan-research-escalate`) | `_callGoJSONRef` | ✗ NOT WIRED (fails in the fresh-colony case) | No try/catch around the call; Go command resolves phases from an empty `state.Plan.Phases` mid-loop and exits non-zero, which propagates and kills the whole plan run |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `toWorkerDispatches` preserves `permission_profile` for a Go-shaped scout dispatch | Direct Node invocation of the compiled `runResearchConfidenceLoop`/`toWorkerDispatches` (`.aether/ts-host/dist/host.js`) against a synthetic dispatch carrying `permission_profile: {name: "workspace_write", ...}`, captured via the module's own `__setDispatchWorkers` test hook | Captured outgoing dispatch had **no `permission_profile` key** | ✗ FAIL |
| `toWorkerDispatches` preserves `brief` (six-section mission) as `task_brief` | Same probe, dispatch carried `brief: "## Mission...## Files to Study\n- cmd/foo.go\n"` | Captured outgoing dispatch's `task_brief` was **`undefined`** | ✗ FAIL |
| Go and TS test suites pass | `go test ./cmd/...`, `cd .aether/ts-host && npm test` | Go: pass. TS: 548/548 pass, 0 fail | ✓ PASS (but does not exercise the real boundary — see above) |
| `go build ./...` | `go build ./...` | Clean build, no errors | ✓ PASS |
| Stale doc claim ("Scout's repository_read_only profile must remain host-enforced") still present after the canonical profile changed to workspace_write | `grep` across `.aether/commands/plan.yaml`, `.claude/commands/ant/plan.md`, `.claude/commands/ant-plan.md`, `.opencode/commands/ant/plan.md`, `cmd/command_guide.go` | 4/5 files still contain the stale claim verbatim (WR-06) | ✗ FAIL (non-blocking; documented as WARNING) |

### Probe Execution

No `scripts/*/tests/probe-*.sh` files exist for this phase's scope (`find scripts -path '*/tests/probe-*.sh'` returned nothing, and neither PLAN nor SUMMARY files reference a probe script). Step 7c: SKIPPED — no declared or conventional probes for this phase.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| RESEARCH-01 | 164-02, 164-05, 164-09 | Queen decides + states reason + user override | ✗ BLOCKED | Decision/override built and tested; "warranted research runs automatically" fails at the dispatch boundary (CR-01) |
| RESEARCH-02 | 164-02, 164-05 | Research runs automatically when warranted | ✗ BLOCKED | Same CR-01 root cause |
| RESEARCH-03 | 164-07 | Findings persist + injected into planner's context | ✗ BLOCKED (REQUIREMENTS.md marks Complete — contradicted) | Route-Setter injection logic is real, but the underlying artifact is never durably produced by a real dispatch (CR-01 + CR-02); REQUIREMENTS.md's `[x]` for this item is not supported by an end-to-end runnable proof |
| RESEARCH-04 | 164-01 | Replan re-researches from scratch | ⚠ PARTIAL | Go-side gating correct and tested; real re-research doesn't happen because the dispatch itself fails (CR-01) |
| RESEARCH-05 | 164-01 | Survey context reaches worker + planner | ⚠ PARTIAL | Brief construction correct; never reaches the real worker (CR-02) |
| RESEARCH-06 | 164-05 | No new planning store | ✓ SATISFIED | Confirmed — `PendingDecision` reused, no new JSON file |
| RESEARCH-07 | 164-03, 164-06, 164-08 | `confidence-loop.ts` used as-is, progress visible | ⚠ PARTIAL | Class reused correctly (not reimplemented) and progress line renders; but it's grading dispatches that fail or arrive briefless in a live run |
| RESEARCH-08 | 164-03 | Depth binds target/iteration budget exactly | ✓ SATISFIED | `researchLoopPreset` binds 80/4, 90/6, 95/8, 99/12 exactly, tested |
| RESEARCH-09 | 164-04, 164-09 | Three depth controls reachable + explained | ✓ SATISFIED | `computeDepthProposal`, wired into manifest and both wrappers |
| RESEARCH-10 | 164-04, 164-09 | Queen proposes with reason, pre-marked | ✓ SATISFIED | Same evidence as RESEARCH-09 |

**Orphaned requirements:** None — `.planning/REQUIREMENTS.md` line 441 lists exactly RESEARCH-01 through RESEARCH-10 for Phase 164, and all ten appear across the nine plans' `requirements:` frontmatter.

**Note on REQUIREMENTS.md checkbox state:** As of this verification, `.planning/REQUIREMENTS.md` marks RESEARCH-03, RESEARCH-09, and RESEARCH-10 as `[x]` Complete and the remaining seven as `[ ]` Pending — evidently not yet updated to reflect the SUMMARY.md claims for 164-02/05/06/08 (which claim RESEARCH-01, 02, 06, 07 complete). Independent of that staleness, this verification finds RESEARCH-03's `[x]` mark is not actually supportable end-to-end (see above), so no action is needed to "catch up" that checkbox to Complete — it should remain unmarked until CR-01/CR-02 are fixed.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `.aether/ts-host/src/host.ts` | 437-477 | Silent field-drop in `toWorkerDispatches` (no error, no warning, dispatch proceeds with missing fields) | 🛑 Blocker | Root cause of CR-01/CR-02; a broken dispatch appears to succeed at every layer except the final Go permission check |
| `cmd/phase_research_escalate.go` | 83 | Zero-value seed passed where a real planning-iteration-state seed is required (`codexPlanIterationState{}`) | 🛑 Blocker | Root cause of CR-03 |
| `.aether/ts-host/src/host.ts` | ~899-906 | Unwrapped `_callGoJSONRef` call for an operation the code's own design intent (D-08: "research is enrichment, never a gate") requires to degrade gracefully | 🛑 Blocker | Turns a single-phase escalation failure into a whole-plan-run crash |
| `.aether/commands/plan.yaml`, `.claude/commands/ant/plan.md`, `.claude/commands/ant-plan.md`, `.opencode/commands/ant/plan.md`, `cmd/command_guide.go` | (per WR-06 in review) | Stale documentation claim contradicting the runtime's own canonical permission change made in this same phase | ⚠ Warning | Wrapper instructions actively mislead about the sandbox guarantee for research Scouts |
| `.aether/ts-host/src/research-confidence.ts` | 189-205 (per WR-04) | `isCited` counts a citation as verified without checking the cited path exists | ⚠ Warning | Undercuts D-11's "checkable evidence dominates" claim; self-reportable score component |
| `cmd/codex_plan.go` (per WR-01) | `computePhaseResearchProposalFields` fresh-plan call site | `--dry-run` mutates `pending-decisions.json` | ⚠ Warning | Violates this project's own CLAUDE.md corollary: "An inspection or --dry-run command must not mutate state" |

No `TBD`/`FIXME`/`XXX` debt markers found in any file modified by this phase.

### Human Verification Required

None. All findings in this report are derived from static code inspection, unit test execution, and a direct runtime probe against the actual compiled dispatch-conversion code — no visual, real-time, or external-service behavior remains ambiguous. The core defects (CR-01, CR-02, CR-03) are deterministic and were reproduced without any human judgment call.

### Gaps Summary

Nine of nine plans landed working, individually well-tested Go and TypeScript modules — the decision/recommendation layer, the depth-proposal card, the evidence-based confidence scorer, and the escalation dispatch builder are all real, not stubs. Success Criterion 5 (three-knob depth proposal) is fully achieved and does not touch the affected code path.

But the phase's central promise — "a research worker runs automatically and its findings persist and feed the plan directly" — is false for any real, non-simulated `aether host plan` run, because of one shared root cause: `toWorkerDispatches` in `.aether/ts-host/src/host.ts` silently drops two fields the Go manifest emits (`permission_profile`, `brief`) when converting plan-time dispatches into the worker request the Go adapter actually executes. This was independently reproduced by running the compiled conversion code directly against a Go-shaped dispatch object. A third, compounding defect (`cmd/phase_research_escalate.go` resolving phases from an empty `state.Plan.Phases` during the exact fresh-colony mid-loop scenario escalation exists for, uncaught by the TS host) turns the intended safety-net path into a whole-plan-run crash instead.

All three defects are invisible to the phase's own test suites because every test that exercises this boundary mocks it (`__setDispatchWorkers`/`__setCallGoJSON` in the TS suite; pre-populated `colony.Plan.Phases` in the Go CLI test) — exactly the failure class this project's CLAUDE.md documents as its recurring root cause ("the machinery exists but has never been proven by execution").

These three defects live in a small, well-isolated area (`toWorkerDispatches` in host.ts, plus the escalate command's seed argument and a missing try/catch) and do not require re-planning the phase — a focused closure plan addressing CR-01, CR-02, and CR-03 together should close all four blocked/partial truths at once.

---

_Verified: 2026-08-02T16:10:00Z_
_Verifier: Claude (gsd-verifier)_
