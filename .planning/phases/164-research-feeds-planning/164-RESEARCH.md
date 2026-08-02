# Phase 164: Research Feeds Planning - Research

**Researched:** 2026-08-02
**Domain:** Go CLI orchestration (`cmd/`) + TypeScript dispatch host (`.aether/ts-host/`) — wiring an existing per-phase research dispatch into an existing-but-unused confidence loop, plus a plan-time proposal ceremony
**Confidence:** HIGH (all claims below verified by reading the current repo; no external library research was needed — this phase touches zero new dependencies)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**The Queen's research decision (RESEARCH-01, RESEARCH-02)**
- **D-01: One batched, tick-to-approve proposal.** After the route is drafted, the Queen presents a single list — "Phase 2: research (new external API); Phase 3: skip (pure refactor, codebase mapped)" — and the user approves or flips entries in one interaction. Same interaction pattern as `suggest-approve`. No per-phase interruptions, no decide-and-proceed.
- **D-02: Queen judgment grounded by runtime hints.** The Go runtime computes cheap signals per phase (external tech/API mentions, domain absent from the colonize survey map, phase mode) and passes them to the orchestrating Queen, who makes the call and writes the plain-English reason. Not pure keyword heuristics (Phase 167 is moving away from keyword inference) and not ungrounded LLM vibes.
- **D-03: Overrides are recorded as decisions.** A user flip (either direction) lands in the existing decision/assumption model (RESEARCH-06: no new planning store). Plan artifacts show "user overrode: skip research on phase 3". Replans see the record but still re-propose — overrides are not sticky.
- **D-04: Scout by default, Oracle on escalation.** The per-phase researcher stays a Scout (cheap, parallel, already wired). When the confidence loop stalls below target at deep/exhaustive depth, the Queen may escalate that one phase to an Oracle. Depth buys a heavier researcher only where earned.

**Re-research on replan (RESEARCH-04)**
- **D-05: Replans always re-research.** The current `hasWorkerAuthoredResearch` skip-if-exists behavior in `cmd/phase_research.go` is inverted for replans: a re-run of plan re-researches flagged phases from scratch. Within a single plan run's iterations, research still runs once per phase (unchanged).
- **D-06: The batched proposal is the cost control.** On replan the Queen re-proposes research per phase with reasons; everything approved re-researches fresh. The user sees the full list before workers spawn and can flip entries off. No silent 10-worker surprises, no staleness-threshold reuse (that quietly drifts back to the failure mode RESEARCH-04 targets).
- **D-07: Findings overwrite in place.** `phase-N-research.md` regenerates — one durable artifact per phase, no timestamped archive siblings.
- **D-08: Research failure warns loudly, planning proceeds.** Research is enrichment, not a gate (Phase 160 classification). If a research worker fails or produces nothing usable, the planner runs anyway and both the plan artifact and finalize output state "phase N planned WITHOUT its research — worker failed". Never blocks, never silent.

**Confidence readout & early accept (RESEARCH-07, RESEARCH-08)**
- **D-09: Full colony ceremony.** Each research Scout announces with caste emoji + ANSI color + deterministic ant name (🔍 Scout Antenna-42), and every loop iteration prints the ceremony line — confidence %, delta, budget remaining, stop reason at the end. `renderIterationCeremony()` in `confidence-loop.ts` already produces the line format. Consistent with Phase 168's living-colony direction.
- **D-10: Early-accept prompts only on stall or near-target.** The loop runs on its own; the user is asked only when diminishing returns are detected below target, or confidence is within ~5% of target with iterations left: "At 87%, gaining 2%/iteration — accept now or keep digging?" No per-iteration nagging, no upfront-only flag.
- **D-11: Confidence is evidence-scored with a self-check blend.** The runtime scores checkable evidence — required sections filled, patterns/gotchas cited with real paths or URLs, files-to-study verified to exist — blended with the researcher's own gap assessment. The number must be reproducible so a test can fail when scoring breaks (Definition of Done). A bare self-score from a cheap model is not trusted alone.
- **D-12: Depth binds target and iteration budget** per RESEARCH-08: fast 80%/4, balanced 90%/6, deep 95%/8, exhaustive 99%/12 — fed into the existing `ConfidenceLoopOptions` (`confidenceTarget`, `maxIterations`), not a new loop.

**Depth proposal ceremony (RESEARCH-09, RESEARCH-10)**
- **D-13: Multiple-choice proposal card, zero typing.** At plan start the Queen presents granularity, task decomposition depth, and verification depth as selectable options with her recommended pick pre-marked and a one-line plain-English reason each ("milestone granularity — goal spans several subsystems"). Accepting the recommendations is one tap; changing any knob is a tap on a different option. The user's words: typing everything is "a pain in the ass" — selection, not forms.
- **D-14: The plan flow has exactly two decision moments.** The depth proposal card and the research batch (D-01) — both tap-to-approve, both selection-based. No third ceremony.
- **D-15: Fast preset — Queen leans skip.** On a fast run the research proposal defaults every phase to "skip", but the batch still appears so the user can flip a phase on; a flipped-on phase runs at 80%/4. Speed stays fast's default contract, RESEARCH-08's numbers apply whenever research actually runs, nothing is silently impossible.
- **D-16: Autopilot auto-accepts, records, and logs.** Under `/ant-run` both decision moments auto-accept the Queen's recommendations, record them through the existing decision model, and print them in the run log ("auto-accepted: research phases 2,4; granularity milestone"). No smart pauses for proposals.

### Claude's Discretion
- How territory survey context (RESEARCH-05) is plumbed into the research brief and planner context — ride the manifest-level context path Phase 163 restored, mechanics up to the planner.
- The evidence-scoring formula internals (weights, section checks, citation verification mechanics) — bound by D-11's reproducibility rule.
- Runtime-hint computation details for D-02 (which signals, how surfaced to the Queen).
- Oracle escalation mechanics (threshold, brief shape) for D-04.
- Where depth→target/iteration binding lives (Go vs TS boundary), given the existing `--target`/`--max-iterations` flags in `cmd/codex_plan.go`.
- Exact ceremony wording, proposal card copy, and log-line formats.
- Whether `renderPhaseResearchBrief`'s six-section format changes to support evidence scoring.

### Deferred Ideas (OUT OF SCOPE)
- **ts-host preflight timeout hardcoded** — matched by keyword only; it is Phase 163.2's executed scope, not research. Remains with 163.2.
- **continue-finalize does not count `--reconcile-task` for the implementation_evidence gate** — continue-finalize bug, unrelated to research-feeds-planning; already reviewed-and-deferred by 163.2. Remains pending for a future phase.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| RESEARCH-01 | Queen decides whether a phase needs research, states reason, user can override either way | `orchestrator_boundary_questions.go` + `pending_decision.go` give a ready decision/reason/override substrate; see Architecture Pattern 1 and Pitfall 1 for why the *existing* boundary-question mechanism is the wrong reuse target for this specific UX |
| RESEARCH-02 | Research runs automatically before the plan is drafted | Already true today — `plannedPhaseResearchDispatches` (`cmd/phase_research.go`) unconditionally emits wave-1 Scout dispatches; the gap is making it *conditional* on the Queen's decision, not building automation from scratch |
| RESEARCH-03 | Findings persist to a durable per-phase artifact, injected into planner context, not pasted by hand | Already true today — `.aether/data/phase-research/phase-N-research.md` + `resolvePhaseResearchSection()` (build-brief side). Gap: the Route-Setter's planning brief only gets a pointer sentence, not the content itself — see Pitfall 4 |
| RESEARCH-04 | Re-planning re-researches rather than reusing stale findings | `hasWorkerAuthoredResearch()` is the exact function to invert; `TestPhaseResearchDispatchedOncePerPhase` is the exact test that pins today's (wrong-for-replan) behavior and will need a companion/replacement test — see Code Examples |
| RESEARCH-05 | Territory survey context reaches both the research worker and the planner | Planner (Scout + Route-Setter) already gets survey via `loadCodexSurveyContext` → `renderPlanningWorkerBrief`. The research worker does not — `renderPhaseResearchBrief(goal, candidate)` receives no survey argument at all. This is the single cleanest, most concrete gap found — see Architecture Pattern 2 |
| RESEARCH-06 | Research output feeds the existing decision/assumption/plan-revision model, no new planning store | `PendingDecisionFile` / `pending-decisions.json` (`cmd/pending_decision.go`) is the existing store; `codexPlanRevisionContext` is the existing revision model — both already exist and are reused elsewhere in `cmd/codex_plan.go` |
| RESEARCH-07 | Use existing `confidence-loop.ts` rather than reimplementing; progress visible while it runs | `ConfidenceLoop` class exists and is fully unit-testable but is wired into **only** `runDispatchedBuildCommand` in `host.ts` today — `runDispatchedPlanCommand` never constructs one. See "The Three Loops" in Architecture Patterns — this is the load-bearing finding of this research pass |
| RESEARCH-08 | Depth binds target/iteration budget (fast 80/4, balanced 90/6, deep 95/8, exhaustive 99/12) with accept override | These exact numbers **already exist** in Go as `planningLoopPreset()` (`cmd/codex_plan.go:1691`), but that function feeds the *whole-plan* Go-native loop (`codexPlanningLoop`), not per-phase research and not the TS `ConfidenceLoop`. Do not assume this preset is already "the" binding this requirement asks for — see Pitfall 2 |
| RESEARCH-09 | Depth controls reachable/explained at plan time: granularity, task decomposition depth, verification depth | `pkg/colony/granularity.go` (`GranularityRange`) has the exact ranges already; `resolveSmartPlanningDepth` / `resolveSmartVerificationDepth` / `renderReviewDepthLineWithReason` already compute smart defaults + reasons for two of the three knobs — reusable, not build-from-scratch |
| RESEARCH-10 | Queen proposes granularity + both depths with a plain-English reason, user accepts/changes | `plan.md`'s existing "Depth Ceremony" section already asks the user to pick 1-of-4 for depth and light/standard/deep for task depth — but as an ungrounded static list with no computed reason and no verification-depth option in the same prompt. The gap is turning this into a *reasoned, three-knob, pre-marked-recommendation* card, not inventing selection UX from nothing |
</phase_requirements>

## Summary

This phase is pure wiring, not greenfield build, and the codebase confirms that framing precisely. Two of ten requirements (RESEARCH-02, RESEARCH-03) are already true in the current code and only need a conditional gate added. Two more (RESEARCH-06, RESEARCH-09) already have every data structure they need sitting unused nearby. The remaining six require real work, but every one of them has a close, reusable analog already in the codebase — nothing here needs a new subsystem.

The single most important finding is architectural: **the codebase already contains three functionally distinct "confidence loop" implementations**, and RESEARCH-07/RESEARCH-08/D-12 are unambiguous about which one this phase must use — the TypeScript `ConfidenceLoop` class in `.aether/ts-host/src/confidence-loop.ts` — which is currently wired into **zero** planning code. A Go-native loop with the identical-looking preset numbers (`planningLoopPreset`: 80/4, 90/6, 95/8, 99/12) already drives whole-plan-draft iteration end-to-end via `aether host plan` → `plan-finalize`, and it would be easy — and wrong — to mistake "the numbers already exist in Go" for "the requirement is already satisfied." It is not: that Go loop governs whether the *entire drafted plan* (Scout+Route-Setter confidence) needs another pass, not whether one phase's research is deep enough. Oracle adds a third, independent loop (`cmd/oracle_loop.go`, its own state file, default 95%/15) that already exists for the D-04 escalation path and should likely be invoked as-is rather than wrapped a second time in the TS class.

The second most load-bearing finding: survey context (RESEARCH-05) reaches the planner's Scout and Route-Setter today (`loadCodexSurveyContext` → `renderPlanningWorkerBrief`) but is completely absent from `renderPhaseResearchBrief`, which only receives `(goal, candidate)`. This is a small, mechanical, well-isolated fix.

**Primary recommendation:** Treat this phase as five connected changes to existing functions, not new subsystems: (1) gate `plannedPhaseResearchDispatches` behind a Queen decision recorded via `PendingDecisionFile`, using Go-computed hints (external tech mentions, survey-domain gaps, phase mode) rather than the `researchPhaseKeywords` pattern Phase 167 is retiring; (2) pass `codexSurveyContext` into `renderPhaseResearchBrief`; (3) invert `hasWorkerAuthoredResearch` for replans and extend/replace `TestPhaseResearchDispatchedOncePerPhase`; (4) add a research-specific iteration wrapper in `host.ts` that constructs a TS `ConfidenceLoop` around research-Scout dispatch+evaluate (mirroring `runDispatchedBuildCommand`'s pattern, not `runDispatchedPlanCommand`'s current single-shot pattern), fed by `planningLoopPreset`-shaped depth binding but landing in `ConfidenceLoopOptions`; (5) extend `plan.md`'s existing Depth Ceremony into a three-knob reasoned card using the already-built smart-default + reason-rendering helpers, and add the D-01 batched research-approval card using the `suggest-approve` interaction pattern rather than the orchestrator-mode-gated `discussQuestion`/`aether discuss` mechanism (which is mode-gated and routes through a heavier flow that violates D-14's "no third ceremony").

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Queen's research-needed decision + reason | API/Backend (Go, `cmd/`) | Frontend Server (wrapper `plan.md`) | Runtime computes hints and owns the decision record (RESEARCH-06: existing store); wrapper renders the tick-to-approve card and relays the user's flips back to the runtime |
| Research dispatch gating (skip vs. run) | API/Backend (Go, `plannedPhaseResearchDispatches`) | — | Dispatch emission is manifest construction — already Go-owned, unconditional today, needs a conditional flag |
| Per-phase research confidence iteration | API/Backend host process (TS host, `.aether/ts-host/`) | API/Backend (Go, dispatch + finalize) | The TS host is the dispatch loop driver for build already (`runDispatchedBuildCommand`); research iteration is the same shape of problem (dispatch → evaluate → maybe redispatch) and belongs in the same tier for the same reason `ConfidenceLoop` lives there |
| Confidence scoring (evidence + self-check blend) | API/Backend host process (TS, `confidence-evaluator.ts` is the existing analog) | Frontend Server (Go supplies verifiable facts: file existence, section presence) | Must be reproducible (D-11) — pure scoring logic belongs where `ConfidenceEvaluator` already lives; Go supplies the checkable facts (file-exists, path-cited) it cannot compute from prose alone |
| Findings persistence | Database/Storage (`.aether/data/phase-research/*.md`) | — | Unchanged — already a durable per-phase artifact on disk, gitignored, correctly local-only |
| Depth proposal card (granularity, task depth, verification depth) | Frontend Server (wrapper `plan.md`) | API/Backend (Go computes reasons via `resolveSmartPlanningDepth`/`resolveSmartVerificationDepth`/`renderReviewDepthLineWithReason`) | Selection/ceremony rendering is wrapper-owned per the UX Architecture ownership model in CLAUDE.md; the *reasoning* behind each recommendation must come from Go so it stays truthful and testable, not invented prose |
| Oracle escalation trigger | API/Backend (Go, likely in `plan-finalize` or the research dispatch gate) | — | Escalation is a dispatch-shape decision (swap caste scout→oracle) — same tier as the rest of dispatch construction; Oracle's own RALF loop (`oracle_loop.go`) then owns its internal iteration, unrelated to the TS `ConfidenceLoop` |
| Override recording (D-03) | Database/Storage (`pending-decisions.json`) | API/Backend (Go read/write via `pendingDecisionAddCmd`/`pendingDecisionResolveCmd`) | Existing store, existing CLI surface — RESEARCH-06 explicitly forbids a new one |

## Standard Stack

No new external dependencies. This phase modifies existing Go (`cmd/`) and TypeScript (`.aether/ts-host/src/`) code only.

### Core (existing, reused)
| Component | Location | Purpose | Why it's the standard for this phase |
|---|---|---|---|
| `ConfidenceLoop` | `.aether/ts-host/src/confidence-loop.ts` | Target/iteration-bound iteration driver | RESEARCH-07 explicitly requires reuse, not reimplementation |
| `ConfidenceEvaluator` | `.aether/ts-host/src/confidence-evaluator.ts` | Converts raw signals into a 0-100 score | Natural home for D-11's evidence-scoring blend (bridges to `ConfidenceMetric`, same shape `ConfidenceLoop.evaluate()` consumes) |
| `plannedPhaseResearchDispatches` / `hasWorkerAuthoredResearch` / `renderPhaseResearchBrief` / `resolvePhaseResearchSection` | `cmd/phase_research.go` | Dispatch emission, staleness check, brief construction, build-brief injection | The whole file is the extension point per CONTEXT.md canonical refs |
| `PendingDecisionFile` / `pendingDecisionAddCmd` / `pendingDecisionResolveCmd` | `cmd/pending_decision.go` | Decision/override storage | RESEARCH-06's "no new planning store" — this is the store |
| `planningLoopPreset` / `codexPlanningLoop` / `resolvePlanningLoopOptions` | `cmd/codex_plan.go` | Existing depth→target/iteration mapping (whole-plan loop) | Numeric source of truth for D-12's four presets; **do not confuse with the TS loop this requirement targets** — see Pitfall 2 |
| `GranularityRange` | `pkg/colony/granularity.go` | Sprint/milestone/quarter/major phase-count ranges | RESEARCH-09's exact numbers, already correct |
| `resolveSmartPlanningDepth` / `resolveSmartVerificationDepth` / `renderReviewDepthLineWithReason` | `cmd/codex_plan.go`, `cmd/review_depth.go`, `cmd/codex_visuals.go` | Smart-default computation + plain-English reason rendering | Reusable machinery for D-13's per-knob reason line |
| `casteIdentity()` / `casteEmoji()` / `casteColorMap` / `deterministicAntName()` | `cmd/codex_visuals.go` | Caste emoji + color + deterministic name | D-09's ceremony identity — already correct for the `scout` caste used by research dispatches |
| `codex.PermissionProfileForCaste` | `cmd/codex_plan.go` (via `attachPlanningDispatchSkillAssignments`) | Assigns `repository_read_only` etc. per caste | Already applied to research dispatches today — confirms plan.md's "Scout's repository_read_only profile must remain host-enforced" claim is already true and must stay true for any Oracle-escalation path too |
| `oracle_loop.go` (`runOracleLoop`, `oracleStateFile`) | `cmd/oracle_loop.go` | Oracle's own RALF-style iteration | The pre-existing "heavier researcher" D-04 escalates to — has its own target/iteration model (default 95%/15), independent of both other loops |

**Installation:** None required — no `npm install` / `go get` needed. If any TS file under `.aether/ts-host/src/` changes, rebuild with `npm --prefix .aether/ts-host run build` (dist is `//go:embed`-ed and tracked in git — confirmed via `git ls-files .aether/ts-host/dist`).

**Version verification:** N/A — no package versions change in this phase.

## Architecture Patterns

### The Three Loops (read this before touching any "confidence" code)

The single biggest risk in planning this phase is loop conflation. Three independent iteration mechanisms already exist:

| Loop | Where | Governs | Target/iteration model | Wired into research today? |
|---|---|---|---|---|
| **A. Whole-plan Go loop** | `cmd/codex_plan.go` (`codexPlanningLoop`, `planningLoopPreset`) + `cmd/codex_plan_finalize.go` (`evaluatePlanningLoopIteration`) | Whether the *entire drafted plan* (Scout+Route-Setter confidence) needs another `aether host plan` → `plan-finalize` round-trip. Driven by the wrapper (`plan.md`) re-invoking the CLI when `requires_next_iteration: true` | fast 80/4, balanced 90/6, deep 95/8, exhaustive 99/12 (`planningLoopPreset`) — **identical numbers to RESEARCH-08** | No — this loop has no concept of "phase" or "research," only overall plan confidence |
| **B. TS `ConfidenceLoop`** | `.aether/ts-host/src/confidence-loop.ts` (`ConfidenceLoop` class) | Build wave iteration only, inside `runDispatchedBuildCommand` in `host.ts`. Never constructed in `runDispatchedPlanCommand` or `runDispatchedContinueCommand` | Configurable via `ConfidenceLoopOptions` (`confidenceTarget`, `maxIterations`, defaults 80/3) | **No — zero current construction sites in the plan path.** This is the loop RESEARCH-07/D-12 mandate be reused |
| **C. Oracle's RALF loop** | `cmd/oracle_loop.go` (`runOracleLoop`, `oracleStateFile`, `runOracleIterationAttempt`) | Oracle's own deep-research iteration, entirely self-contained (own state file, own worker-dispatch attempts, own stop conditions) | Default 95% / 15 iterations (`defaultOracleTargetConfidence`, `defaultOracleMaxIterations`), independently configurable | Not directly — but this is very likely the mechanism D-04's "escalate to Oracle" should invoke as-is, rather than wrapping Oracle in Loop B a second time |

**Implication for planning:** RESEARCH-07/RESEARCH-08 need a **new construction site** for Loop B (the TS class) inside the planning path — most naturally as a research-specific sub-loop in `host.ts`'s plan-command handling, or a new dedicated function that mirrors the `runDispatchedBuildCommand` shape (dispatch → evaluate via a research-flavored `ConfidenceEvaluator`-style scorer → `renderIterationCeremony` → loop). Loop A's existing numbers can be treated as a validated reference for what the four depth tiers *should* resolve to, but Loop A itself is not the mechanism being wired — do not "satisfy" RESEARCH-08 by pointing at `planningLoopPreset` alone. When D-04 escalates a phase to Oracle, prefer invoking Loop C as-is (it already knows how to iterate) rather than nesting Loop B around Oracle's own internal iteration — that would be two loops driving one worker.

### System Architecture Diagram

```
 /ant-plan (wrapper: plan.md)
    │
    ├─▶ [1] Depth proposal card (NEW, D-13/D-14)
    │     Go computes: granularity reason, planning-depth reason,
    │     verification-depth reason (reuse resolveSmart*/renderReviewDepthLineWithReason)
    │     Wrapper renders: pre-marked multi-choice card, one tap to accept/change
    │     └─▶ user selection flows into --depth/--planning-depth/--verification-depth
    │
    ├─▶ aether host plan --depth <x> ...   (TS host: runDispatchedPlanCommand)
    │     │
    │     ├─▶ [2] Go: plan manifest fetch (existing)
    │     │     - loadCodexSurveyContext(root)              [existing]
    │     │     - phaseResearchCandidates(state, seed)       [existing]
    │     │     - Queen research decision (NEW, D-01/D-02)
    │     │         Go computes per-phase hints:
    │     │           external tech/API mention in description
    │     │           domain absent from survey.Languages/Frameworks/Dependencies
    │     │           phase.Mode (best-effort; not required pre-Phase-167)
    │     │         → recommendation + reason per phase
    │     │     - [3] Research batch proposal card (NEW, D-01)
    │     │         suggest-approve-style tick list, NOT the orchestrator-mode
    │     │         discussQuestion/aether-discuss gate (see Pitfall 1)
    │     │         user flips → recorded via PendingDecisionFile (D-03, RESEARCH-06)
    │     │
    │     ├─▶ [4] plannedPhaseResearchDispatches(root, planDepth, goal, candidates)
    │     │     NOW gated by the approved batch, not unconditional
    │     │     NOW: hasWorkerAuthoredResearch inverted on replan (D-05)
    │     │     renderPhaseResearchBrief(goal, candidate, survey)  [survey arg NEW, RESEARCH-05]
    │     │
    │     └─▶ dispatches[] returned to TS host (wave 1: base Scout + N research Scouts + Route-Setter wave 2)
    │
    ├─▶ [5] TS host: research confidence sub-loop (NEW — Loop B applied to research)
    │     For each approved research phase (or as a batched wave, mirroring
    │     runDispatchedBuildCommand's shape):
    │       dispatch research Scout(s) → evaluate (D-11 evidence + self-check blend)
    │       → ConfidenceLoop.evaluate(score, workersUsed)
    │       → renderIterationCeremony() to stderr (D-09, visible while running)
    │       → if stalled/near-target: early-accept prompt (D-10)
    │       → if still below target at deep/exhaustive and stalled: escalate one
    │         phase to Oracle (D-04) — invokes Loop C, not a second Loop B wrap
    │     Depth binds target/iterations per D-12 (fast 80/4 … exhaustive 99/12)
    │
    ├─▶ [6] Route-Setter dispatch (wave 2) — brief now carries research CONTENT,
    │     not just a pointer sentence (RESEARCH-03 gap — see Pitfall 4)
    │
    └─▶ plan-finalize → plan artifact states "phase N planned WITHOUT its
          research — worker failed" when applicable (D-08, never blocks)
```

### Recommended Project Structure (files this phase touches)
```
cmd/
├── phase_research.go              # extend: survey param, replan inversion, Queen-gate hook
├── phase_research_dispatch_test.go # extend: replan-reresearches test, survey-in-brief test
├── codex_plan.go                  # extend: depth-proposal reason wiring, research-decision hints
├── pending_decision.go            # reuse as-is: override recording (D-03)
├── orchestrator_boundary_questions.go  # DO NOT extend for D-01 — mode-gated, wrong shape (Pitfall 1)
├── codex_visuals.go                # extend: ceremony rendering for proposal cards if Go-side text needed
└── oracle_loop.go                  # reuse as-is: D-04 escalation target

.aether/ts-host/src/
├── host.ts                        # extend: new research iteration path using ConfidenceLoop (mirror runDispatchedBuildCommand)
├── confidence-loop.ts             # reuse as-is per RESEARCH-07 — do not modify the class, only construct it in a new place
├── confidence-evaluator.ts        # extend or add a research-flavored sibling for D-11's evidence blend
└── (dist/ rebuilt via `npm --prefix .aether/ts-host run build` after any src change)

.claude/commands/ant/plan.md       # extend: Depth Ceremony → 3-knob proposal card; add research batch card
.opencode/commands/ant/plan.md     # mirror the same changes (OpenCode parity)
.aether/commands/plan.yaml         # source-of-truth YAML if the wrapper spec changes structurally
```

### Pattern 1: Batched tick-to-approve, not a blocking clarification gate
**What:** D-01 wants a single-screen list the user approves/flips in one interaction, with no per-phase interruption and no routing through a separate command.
**When to use:** The Queen's research decision (D-01) and arguably the depth proposal (D-13), though D-13 is framed as a pre-marked multi-choice card rather than a tick list.
**Example (existing analog to follow, not to directly call):**
```go
// Source: cmd/ceremony_cmd.go:622 (suggest-approve rendering pattern)
fmt.Fprintf(&b, "  Approve: aether suggest-approve --approve %s\n", s.ID)
fmt.Fprintf(&b, "  Dismiss: aether suggest-approve --dismiss %s\n", s.ID)
```
The research batch needs the same shape: one Go-rendered list, one wrapper interaction, one CLI call back (something like `aether plan-research-approve` or reuse of `pending-decision-resolve` per flipped phase) — not a redirect into `/ant-discuss`.

### Pattern 2: Survey context injection into a worker brief
**What:** `renderPlanningWorkerBrief(root, survey, spec, ...)` already takes a `codexSurveyContext` and writes survey doc paths + source anchors into the brief text.
**When to use:** Directly transferable to `renderPhaseResearchBrief` for RESEARCH-05.
**Example:**
```go
// Source: cmd/codex_plan.go:1919-1957 (renderPlanningWorkerBrief) — the exact
// pattern renderPhaseResearchBrief needs, just applied to a narrower brief.
surveyDocs := make([]string, 0, len(survey.SurveyDocs))
for _, name := range survey.SurveyDocs {
    surveyDocs = append(surveyDocs, filepath.ToSlash(filepath.Join(surveyDir, name)))
}
if len(surveyDocs) > 0 {
    b.WriteString("- Survey docs to read first: ")
    b.WriteString(strings.Join(surveyDocs, ", "))
}
```
Contrast with today's `renderPhaseResearchBrief(goal, candidate)` — no survey parameter at all (`cmd/phase_research.go:105`).

### Pattern 3: TS host confidence loop around a dispatch wave (the pattern to replicate for research)
**What:** `runDispatchedBuildCommand` constructs one `ConfidenceLoop`, then loops: dispatch wave → evaluate via `ConfidenceEvaluator` → `confidenceLoop.evaluate()` → `renderIterationCeremony()` → check `shouldContinue`.
**When to use:** This is the shape RESEARCH-07/08 need reproduced for research, scoped to one phase's research dispatch(es) rather than a whole build wave.
**Example:**
```typescript
// Source: .aether/ts-host/src/host.ts:842-916 (runDispatchedBuildCommand) —
// the loop shape to mirror, not call directly, for research.
const loopOpts: ConfidenceLoopOptions = { totalBudget: spawnBudget };
if (parsed.maxIterations) loopOpts.maxIterations = parseInt(parsed.maxIterations, 10);
if (parsed.targetConfidence) loopOpts.confidenceTarget = parseInt(parsed.targetConfidence, 10);
const confidenceLoop = new ConfidenceLoop(loopOpts);
const confidenceEvaluator = new ConfidenceEvaluator();

while (true) {
  lastWaveResult = await dispatchBuildWave(/* ... */);
  const evaluated = confidenceEvaluator.evaluate({ workerClaims: lastWaveResult.workerClaims });
  const loopResult = confidenceLoop.evaluate(evaluated.score, lastWaveResult.workerCount);
  renderIterationCeremony(loopResult);   // -> emitCeremonyOutput -> stderr, visible live
  if (!loopResult.shouldContinue) { renderIterationComplete(loopResult.stopReason); break; }
}
```
For research, `workerClaims`-shaped evaluation (test pass rate, files touched, blockers) does not map cleanly — D-11 wants evidence-scoring (sections filled, citations verified, files-to-study exist) blended with self-assessment, which argues for a **research-specific evaluator**, not literal reuse of `ConfidenceEvaluator`, while still feeding the same `ConfidenceLoop` class.

### Pattern 4: Depth ceremony with computed reasons (extend, don't replace)
**What:** `plan.md`'s existing "Depth Ceremony" section is a hardcoded 4-option list with no reasoning; `renderReviewDepthLineWithReason` already exists and computes a reason for verification depth elsewhere in the codebase (post-hoc phase rendering, not the pre-plan proposal).
**When to use:** RESEARCH-10's "plain-English reason each."
**Example:**
```go
// Source: cmd/codex_visuals.go:1336-1352 (renderReviewDepthLineWithReason) —
// reusable reason-computation pattern; currently invoked for a single phase's
// verification depth post-hoc, not as a three-knob pre-plan proposal.
func renderReviewDepthLineWithReason(depth colony.VerificationDepth, phaseNum, totalPhases int, phase colony.Phase, smartDefault bool) string {
    reason := renderSmartDepthReason(phase, totalPhases)
    // ...
    return fmt.Sprintf("Review depth: standard (%s)", reason)
}
```
A parallel function is needed for granularity's reason (goal-scope based, likely using `codexSurveyContext`/goal text length or phase-count heuristics — no existing helper found; this is new, small Go logic) alongside reuse of the planning-depth and verification-depth reason logic that already exists.

### Anti-Patterns to Avoid
- **Building a fourth confidence loop.** Three already exist (A/B/C above). If a new counter/threshold/stop-reason structure starts taking shape anywhere in this phase's diff, stop — it should be configuration into Loop B, or delegation to Loop C, never a new implementation.
- **Routing D-01's research batch through `materializeOrchestratorBoundaryQuestions` / `aether discuss`.** That mechanism is gated behind `state.EffectiveColonyMode() == colony.ColonyModeOrchestrator` (most colonies run in `"colony"` mode and would never see it) and its resolution path is a full clarification session, not a one-tap card. Using it would violate D-14 ("exactly two decision moments... no third ceremony") by introducing a mode-dependent detour.
- **Reusing `researchPhaseKeywords`/`isResearchPhase` (`cmd/plan_grounding.go`) as the basis for D-02's hints.** That keyword list exists for a different gate (grounding exemption) and is explicitly named in `.planning/REQUIREMENTS.md` (TYPED-05) as a keyword-inference site Phase 167 is removing. Building D-02 on top of it means immediately un-wiring it again next phase.
- **Treating `planningLoopPreset`'s matching numbers as proof RESEARCH-08 is done.** See "The Three Loops" — the numbers are right, the wiring target is wrong.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Target/iteration-bound loop with diminishing-returns detection | A new loop class or hand-rolled while-loop with counters | `ConfidenceLoop` (`.aether/ts-host/src/confidence-loop.ts`) | RESEARCH-07 is explicit; the class already handles max-iterations, confidence-target, diminishing-returns-window, and budget tracking correctly and has existing unit tests (`confidence-loop.test.ts`) to extend |
| Deterministic caste-colored worker identity | New naming/coloring logic | `deterministicAntName()`, `casteIdentity()`, `casteEmoji()`, `casteColorMap` (`cmd/codex_visuals.go`) | Already used for the `scout` caste on today's research dispatches; Oracle escalation just needs `casteIdentity("oracle")`, which already resolves correctly |
| Decision/override persistence | A new JSON file under `.aether/data/` | `PendingDecisionFile` / `pending-decisions.json` (`cmd/pending_decision.go`) | RESEARCH-06 explicitly forbids a new store, and this one already has add/list/resolve CLI verbs and scope-matching logic (`pendingDecisionMatchesScope`) |
| Granularity → phase-count range lookup | A new range table | `GranularityRange()` (`pkg/colony/granularity.go`) | One function, already exactly right (1-3/4-7/8-12/13-20) |
| Smart depth defaulting + reason text | New heuristics for "why standard, why heavy" | `resolveSmartPlanningDepth`, `resolveSmartVerificationDepth`, `renderReviewDepthLineWithReason` | Already computes and renders reasons for two of RESEARCH-10's three knobs |
| Permission scoping per caste | New per-dispatch permission logic for research/Oracle workers | `codex.PermissionProfileForCaste(caste)` via `attachPlanningDispatchSkillAssignments` | Already invoked inside `plannedPhaseResearchDispatches`; confirmed correct for `scout` today, will resolve correctly for `oracle` too since the lookup is caste-keyed, not call-site-keyed |

**Key insight:** Every "don't hand-roll" item here is not a generic library recommendation — it is a specific existing function in this repository that already solves the exact sub-problem. The risk profile of this phase is entirely about correctly locating and extending these functions, not about picking the right abstraction from scratch.

## Common Pitfalls

### Pitfall 1: Reusing the orchestrator-boundary-question mechanism for D-01/D-13
**What goes wrong:** `materializeOrchestratorBoundaryQuestions` / `discussQuestion` / `orchestratorBoundaryGuidance` (`cmd/orchestrator_boundary_questions.go`, `cmd/orchestrator_boundary_guidance.go`) looks like a ready-made "options + reasoning" proposal mechanism — it has `Options []string` and `Reasoning string` fields and is already wired into `runCodexPlanPlanOnly`. It is tempting to extend `planBoundaryQuestionCandidates` for the depth/research proposals.
**Why it happens:** The struct shape genuinely matches what D-13 describes (options + a reasoning string), and the canonical_refs in CONTEXT.md point at `planBoundaryQuestionCandidates` as a "natural home."
**How to avoid:** Check `addOrchestratorBoundaryGuidance` before reusing it — it only activates when `state.EffectiveColonyMode() == colony.ColonyModeOrchestrator`, and unresolved questions redirect the *entire plan run* to `aether discuss`, a separate command/session, not an inline tap-to-approve card. That is a third ceremony and a mode-gated one. If the depth/research proposals must appear for every colony mode (which D-13/D-01's phrasing implies — "at plan start," no mode qualifier), a lighter, always-active mechanism is needed. The `discussQuestion` struct shape can still inspire the JSON contract; the delivery mechanism (`materializeOrchestratorBoundaryQuestions` → `aether discuss`) should not be reused wholesale.
**Warning signs:** A test that only exercises `colony_mode: "orchestrator"`, or a plan run in default `"colony"` mode that never shows the proposal cards.

### Pitfall 2: Conflating the Go `codexPlanningLoop` with the TS `ConfidenceLoop`
**What goes wrong:** Because `planningLoopPreset()` already returns exactly 80/4, 90/6, 95/8, 99/12, it is easy to mark RESEARCH-08 "done" by pointing at that function, or to bind research depth into `resolvePlanningLoopOptions` instead of constructing a TS `ConfidenceLoop`.
**Why it happens:** Same numbers, adjacent code, both called "planning loop" / "confidence" in comments.
**How to avoid:** Loop A (`codexPlanningLoop`) has no per-phase concept at all — it operates on `codexPlanConfidence` (Scout/Route-Setter confidence for the *whole draft*). Verify any new code that claims to satisfy RESEARCH-07/08 actually constructs `new ConfidenceLoop(...)` from `.aether/ts-host/src/confidence-loop.ts` somewhere in the call path, and that the construction site is reachable from the research dispatch/evaluate cycle, not the whole-plan iteration cycle.
**Warning signs:** A grep for `new ConfidenceLoop(` still returning only the one hit in `runDispatchedBuildCommand`.

### Pitfall 3: `phase.Mode` is not populated yet for D-02's hints
**What goes wrong:** D-02 lists "phase mode" as one of three cheap Go-computed signals. `colony.Phase.Mode` exists (`PhaseModeDiscovery`/`Prototype`/`Production`/`Maintenance`) but is `omitempty` and not backfilled — `TYPED-01`/`TYPED-02` (Phase 167, not yet run) are what will backfill it. Most phases today have an empty/zero `Mode`.
**Why it happens:** The requirement text lists "phase mode" as a hint source without noting it's currently unreliable.
**How to avoid:** Treat `phase.Mode` as a best-effort, frequently-empty signal (`if phase.Mode.Valid() { ... }` guard, same pattern `checkPlanGrounding` already uses at `cmd/plan_grounding.go:46`), not a primary signal. Lean on the two signals that work regardless of typed-mode adoption: external tech/API mention in the phase description, and domain absent from `survey.Languages`/`survey.Frameworks`/`survey.Dependencies`.
**Warning signs:** A hint-computation function that returns "unknown"/empty for the mode signal on most or all of Aether's own current phases when tested against real `.planning/` data.

### Pitfall 4: RESEARCH-03's "injected into planner's context" is only half-true today
**What goes wrong:** Assuming RESEARCH-03 is already fully satisfied because `resolvePhaseResearchSection()` exists and injects research into build briefs.
**Why it happens:** `resolvePhaseResearchSection` genuinely does this correctly for the *build* brief (verified: budget-truncated at 3500 chars, called from `codex_build.go`).
**How to avoid:** Check what the Route-Setter (the *planner*, not the builder) actually receives during the same plan run the research was generated in: `cmd/codex_plan.go`'s route-setter-brief-append logic (around the `researchDispatches` append block) only adds one sentence — *"Parallel research Scouts are writing per-phase findings to `.aether/data/phase-research/phase-N-research.md`... Read each phase's research before finalizing the route"* — a pointer, not the content. This relies on the Route-Setter agent independently reading the files mid-session. RESEARCH-03 says findings should be "injected into the planner's context... not pasted by hand" — a file-read instruction is closer to "the worker has to go get it" than "injected." Decide explicitly whether that's sufficient (it may be, since the Route-Setter is itself a worker following instructions, not a human) or whether the brief should be extended to include truncated content directly, mirroring `resolvePhaseResearchSection`'s pattern but for the Route-Setter's own brief construction.
**Warning signs:** A plan whose phase descriptions don't visibly reflect research findings even though `phase-N-research.md` exists and is fresh.

### Pitfall 5: `renderIterationCeremony` exists in two places with the same name
**What goes wrong:** `host.ts` defines its own top-level `function renderIterationCeremony(result: ConfidenceResult)` (line 641) that duplicates the *instance method* `ConfidenceLoop.renderIterationCeremony()` defined inside the class in `confidence-loop.ts`. They produce the same output format today, but a change to one without the other will silently desync the ceremony line.
**Why it happens:** The class method was presumably added, then a free function was independently written in `host.ts` for the build-loop call site rather than calling `confidenceLoop.renderIterationCeremony(result)` on the instance already in scope.
**How to avoid:** When adding the research iteration path, either call the existing instance method or explicitly note in the plan that this duplication is being extended (not created) — do not add a *third* copy for research.
**Warning signs:** Ceremony output format drifting between build and research iterations.

### Pitfall 6: `.aether/data/phase-research/` is gitignored — findings never survive a fresh clone
**What goes wrong:** Assuming "durable per-phase artifact" (RESEARCH-03/D-07) means committed/durable across machines or CI.
**Why it happens:** "Durable" is ambiguous; the artifact is durable *within a working copy* (overwritten in place, not deleted between plan runs) but `.aether/.gitignore` excludes `data/` entirely, confirmed via `git check-ignore -v`.
**How to avoid:** Confirm this is the intended scope (local-durability, not cross-clone durability) before treating "not committed to git" as a bug — it matches every other `.aether/data/` artifact and the Protected Paths table in `.claude/rules/aether-colony.md`. No action needed unless the plan explicitly wants research findings to survive `git clone`, which nothing in CONTEXT.md asks for.
**Warning signs:** A task that tries to `git add` files under `.aether/data/phase-research/`.

## Code Examples

### Today's unconditional research dispatch (the exact gate point for D-01/D-02)
```go
// Source: cmd/phase_research.go:61-89 (plannedPhaseResearchDispatches)
func plannedPhaseResearchDispatches(root, planDepth, goal string, candidates []phaseResearchCandidate) []codexPlanningDispatch {
    if planDepth == "fast" || len(candidates) == 0 {
        return nil
    }
    dispatches := make([]codexPlanningDispatch, 0, len(candidates))
    for _, candidate := range candidates {
        fileName := fmt.Sprintf("phase-%d-research.md", candidate.ID)
        existingPath := filepath.Join(root, ".aether", "data", "phase-research", fileName)
        if hasWorkerAuthoredResearch(existingPath) {
            continue   // <- D-05 must invert this specifically for the replan case
        }
        dispatches = append(dispatches, codexPlanningDispatch{ /* ... */ })
    }
    attachPlanningDispatchSkillAssignments(dispatches)
    return dispatches
}
```
This is the function D-01/D-02/D-05 all converge on: it needs (a) a per-candidate boolean from the Queen's approved batch, and (b) `opts.Refresh`-aware inversion of the `hasWorkerAuthoredResearch` skip.

### Today's staleness test that pins the behavior D-05 must change
```go
// Source: cmd/phase_research_dispatch_test.go:66-85 (TestPhaseResearchDispatchedOncePerPhase)
// This test currently asserts that worker-authored research SUPPRESSES a
// re-dispatch. D-05 requires the opposite behavior specifically when the
// plan run is a replan (opts.Refresh == true). This test will need either
// a companion test gated on replan, or a parameter added to
// plannedPhaseResearchDispatches that this test must be updated to pass.
```

### Existing survey-loading + brief-injection pattern to copy for the research brief
```go
// Source: cmd/codex_plan.go:1461 (loadCodexSurveyContext) — already called
// once per plan-only run at cmd/codex_plan.go:874/:419, so the survey value
// is already in scope at the call site of plannedPhaseResearchDispatches
// (cmd/codex_plan.go:895 area) — it just isn't threaded through today.
survey, err := loadCodexSurveyContext(root)
// ...
researchDispatches := plannedPhaseResearchDispatches(root, planDepth, *state.Goal, phaseResearchCandidates(state, iterationSeed))
// ^ survey is available in this scope but not passed — the fix is adding a
//   survey parameter to plannedPhaseResearchDispatches / renderPhaseResearchBrief.
```

### Existing TS host flags already accepting target/iteration overrides
```typescript
// Source: .aether/ts-host/src/host.ts:225-232 (arg parsing) — --target and
// --max-iterations are ALREADY parsed host-side; RESEARCH-08's "accept
// override to exit early" maps directly onto the existing --accept flag
// (line ~233), which cmd/codex_plan.go already threads into
// codexPlanningLoop.Accept for the whole-plan loop. The same flags can very
// plausibly be repurposed/extended for a research-scoped invocation.
} else if (arg === "--target") {
  targetConfidence = readValue(arg);
} else if (arg === "--max-iterations") {
  maxIterations = readValue(arg);
} else if (arg === "--accept") {
  accept = true;
}
```

## State of the Art

| Old Approach (v5.4.0, per REQUIREMENTS.md/ROADMAP framing) | Current Approach (verified in code, 2026-08-02) | When Changed | Impact |
|---|---|---|---|
| Research is a standalone command the user must remember to run and paste findings into the plan | Research already dispatches automatically in wave 1 alongside the base Scout, before the Route-Setter (`plannedPhaseResearchDispatches`, `plan.md` lines ~74-93) | Commits `281dd34a` / `a2c8288e`, 2026-07-26 (per `.planning/REQUIREMENTS.md` "Corrections carried forward") | The roadmap's stated premise for this phase is already partially stale — confirmed independently by re-reading the current `phase_research.go` and `plan.md`, matching what CONTEXT.md's `<domain>` section already flags |
| `v5.4.0` replans reused stale research (RESEARCH-04's named failure mode) | Current code *also* reuses — `hasWorkerAuthoredResearch` skips re-dispatch whenever any non-template file exists on disk, replan or not | Not yet fixed | This is a genuine, unaddressed gap — D-05 correctly targets it |
| No confidence loop for research at all | A general-purpose confidence loop exists (`ConfidenceLoop`) but is scoped to build only | `confidence-loop.ts` added at some point before this phase (per REQUIREMENTS.md's RESEARCH section framing: "keeping the TS host makes it easier... a working implementation") | Confirms the roadmap's own claim that keeping (not deleting) the TS host was the right call for this phase specifically |

**Deprecated/outdated:** Nothing in this phase's scope is being deprecated — all three loops, the survey-loading pattern, the pending-decision store, and the caste-identity system remain in active use and should be extended, not replaced.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The most natural TS-side construction point for a research-scoped `ConfidenceLoop` is a new function in `host.ts` mirroring `runDispatchedBuildCommand`'s shape, rather than modifying `runDispatchedPlanCommand` in place or adding a wholly separate `aether host research` command | Architecture Pattern 3, Summary | If wrong, the planner may scope tasks around the wrong entry point; low risk since CONTEXT.md explicitly leaves "where depth→target/iteration binding lives (Go vs TS boundary)" to Claude's Discretion, so this is a starting hypothesis, not a locked design |
| A2 | D-04's Oracle escalation should invoke `cmd/oracle_loop.go`'s existing RALF loop as-is rather than wrapping Oracle in the TS `ConfidenceLoop` a second time | "The Three Loops", Anti-Patterns | If wrong (e.g., if the intent is genuinely to have the TS loop drive a single Oracle dispatch attempt, not Oracle's multi-iteration internal loop), double-loop nesting could occur; CONTEXT.md explicitly defers "Oracle escalation mechanics" to discretion, so this needs confirmation during planning, not blind execution |
| A3 | The granularity-proposal reason (no existing helper found, unlike planning-depth/verification-depth which have `resolveSmart*`/`renderReviewDepthLineWithReason`) will need small new Go logic, likely based on goal text length/keyword density or existing phase count, rather than reusing an undiscovered existing function | Pattern 4 | If an existing granularity-reasoning function exists that this research pass missed, this could result in duplicate logic; searched exhaustively for `Granularity.*[Rr]eason` and `renderGranularity` with no hits, so confidence is high this genuinely doesn't exist yet |
| A4 | D-01's research batch card should use a `suggest-approve`-style mechanism (new or adapted) rather than the `discussQuestion`/`orchestrator_boundary_questions.go` mechanism | Pattern 1, Pitfall 1 | Medium risk — CONTEXT.md's canonical_refs literally names `planBoundaryQuestionCandidates` as "natural home for D-13," which could mean the user/prior discussion intended it despite the mode-gating issue found here. This should be explicitly surfaced to the planner as a decision point, not silently overridden |

**If this table is empty:** N/A — see above; all four entries stem from genuine design-space gaps in CONTEXT.md's "Claude's Discretion" section, not unverified factual claims.

## Open Questions (RESOLVED)

All three questions below were resolved during planning — see Plans 164-09, 164-06, and 164-08 respectively.

1. **RESOLVED (Plan 164-09):** Reuse the `discussQuestion`-shaped JSON contract for consistency, but deliver it through a new, always-active, single-interaction card — not `materializeOrchestratorBoundaryQuestions`/`aether discuss`, whose mode gate stays untouched. — **Does D-13's canonical-ref pointer to `planBoundaryQuestionCandidates` mean the orchestrator-boundary mechanism should be extended despite its mode-gating, with the mode-gate itself being loosened/removed as part of this phase?**
   - What we know: The mechanism has the right field shape (Options + Reasoning) and CONTEXT.md's canonical_refs explicitly names it as the "natural home." It is also currently hard-gated to `colony_mode: "orchestrator"` and its resolution path is `aether discuss`, a separate command/session.
   - What's unclear: Whether "natural home" meant "reuse the struct shape" or "reuse the entire mechanism including the mode gate and discuss-routing," and whether removing/loosening the mode gate is in scope for this phase or would be scope creep into colony-mode semantics.
   - Recommendation: Planner should decide explicitly and record the decision (likely: reuse the `discussQuestion`-shaped JSON contract for consistency, but deliver it through a new, always-active, single-interaction card rather than `aether discuss`, per D-14's "no third ceremony" and the fact that D-01's batch is explicitly described as happening inline "after the route is drafted," not as a separate blocking session).

2. **RESOLVED (Plan 164-06):** Per-phase instantiation — one `ConfidenceLoop` per approved research phase, managed as `Map<phaseID, ConfidenceLoop>` in the TS host. — **Should the research confidence sub-loop run once per approved phase (N separate `ConfidenceLoop` instances, N separate research-Scout-iterate cycles) or once for the whole batch (one loop, one Scout wave, evaluated in aggregate)?**
   - What we know: `runDispatchedBuildCommand`'s Loop B usage evaluates an entire wave's aggregate confidence, not per-worker. D-12's depth-bound target/iteration numbers ("fast 80%/4...") read naturally as per-phase (a single research topic reaching 80% confidence in 4 iterations), and D-10's early-accept prompt ("At 87%, gaining 2%/iteration") reads as a single research thread's progress, not a batch average.
   - What's unclear: Whether iterating N phases' research in parallel each with their own loop is intended, or whether depth just parameterizes a shared loop applied per-phase sequentially/in parallel with per-phase state.
   - Recommendation: Per-phase instantiation (one `ConfidenceLoop` per approved research phase) matches D-10's phrasing and D-04's "escalate *that one phase*" framing most closely; the planner should confirm this reading and design the TS-side loop management (likely `Map<phaseID, ConfidenceLoop>`) accordingly.

3. **RESOLVED (Plan 164-08):** Escalation triggers on `stopReason === "diminishing_returns"` combined with a deep/exhaustive depth tier and confidence below target; one escalation per phase per run, no nested loop. — **What exactly triggers Oracle escalation (D-04): a fixed stall-detection rule reused from `ConfidenceLoop.isDiminishingReturns()`, or a new deep/exhaustive-only threshold check?**
   - What we know: D-04 says "when the confidence loop stalls below target at deep/exhaustive depth." `ConfidenceLoop` already has private `isDiminishingReturns()` logic (window + threshold) that produces a `"diminishing_returns"` stop reason — this is likely the exact signal to key escalation off, combined with a depth check (`planDepth in ["deep","exhaustive"]`) and a target-not-met check (`currentConfidence < confidenceTarget`).
   - What's unclear: Whether escalation happens automatically the moment `stopReason === "diminishing_returns"` at deep/exhaustive, or requires an additional Queen judgment step (grounded reasoning, not just a stop-reason match) — CONTEXT.md's D-02 emphasizes "Queen judgment grounded by runtime hints," which could extend to this decision too.
   - Recommendation: Treat the stop-reason + depth-tier combination as the *trigger* for offering escalation, but keep the actual escalation decision itself explainable/Queen-stated (consistent with D-02's philosophy applied to this sub-decision), rather than fully automatic.

## Environment Availability

Skipped — this phase modifies existing Go and TypeScript source in an already-configured repository. No new external tools, services, or runtimes are introduced. `node`, `go`, and the existing `.aether/ts-host/` toolchain (`npm --prefix .aether/ts-host run build`) are already required by the rest of the codebase and confirmed present via the existing `dist/` build artifacts checked into git.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Go framework | Go standard `testing` package |
| Go config file | none — `go test ./...` at repo root |
| TS framework | Node's built-in `node:test` runner via `tsx` |
| TS config file | `.aether/ts-host/package.json` (`test` / `test:all` scripts) |
| Quick run command (Go) | `go test ./cmd/... -run TestPhaseResearch` |
| Quick run command (TS) | `npm --prefix .aether/ts-host run test -- confidence-loop` (or `test:all` filtered by filename glob) |
| Full suite command (Go) | `go test ./... -race` |
| Full suite command (TS) | `npm --prefix .aether/ts-host run test:all` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| RESEARCH-01 | Queen decision + reason recorded, user override both directions | unit (Go) | `go test ./cmd/... -run TestQueenResearchDecision` | ❌ Wave 0 |
| RESEARCH-02 | Research runs automatically when approved, skipped when declined | unit (Go) | `go test ./cmd/... -run TestPhaseResearchDispatchedOncePerPhase` (extend existing) | ✅ (extend) |
| RESEARCH-03 | Findings present in planner's actual context, not just a file pointer | unit (Go) | `go test ./cmd/... -run TestRouteSetterBriefIncludesResearchContent` | ❌ Wave 0 |
| RESEARCH-04 | Replan re-researches; single-iteration research still runs once | unit (Go) | `go test ./cmd/... -run TestReplanReResearchesPhases` | ❌ Wave 0 (companion to existing `TestPhaseResearchDispatchedOncePerPhase`) |
| RESEARCH-05 | Survey context reaches the research worker's brief | unit (Go) | `go test ./cmd/... -run TestRenderPhaseResearchBriefIncludesSurvey` | ❌ Wave 0 |
| RESEARCH-06 | Overrides land in `pending-decisions.json`, no new store created | unit (Go) | `go test ./cmd/... -run TestResearchOverrideUsesPendingDecisionStore` | ❌ Wave 0 |
| RESEARCH-07 | `new ConfidenceLoop(` reachable from the research dispatch path | unit (TS) | `npm --prefix .aether/ts-host run test -- research-confidence-loop` | ❌ Wave 0 |
| RESEARCH-08 | Depth binds correct target/iterations for research specifically | unit (TS) | extend `confidence-loop.test.ts` or new `research-confidence-loop.test.ts` with 4 depth-tier cases | ❌ Wave 0 (extend existing file) |
| RESEARCH-09 | Granularity/planning-depth/verification-depth all surfaced with correct ranges | unit (Go) | `go test ./cmd/... -run TestDepthProposalCardKnobs` | ❌ Wave 0 |
| RESEARCH-10 | Each proposed knob carries a plain-English reason, pre-marked recommendation | unit (Go) | `go test ./cmd/... -run TestDepthProposalReasons` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** targeted `go test ./cmd/... -run <TestName>` and/or `npm --prefix .aether/ts-host run test -- <pattern>`
- **Per wave merge:** `go test ./cmd/... -race` (package-scoped, faster than full repo) plus `npm --prefix .aether/ts-host run test:all`
- **Phase gate:** `go test ./... -race` full suite green, plus `npm --prefix .aether/ts-host run test:all`, before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/phase_research_dispatch_test.go` needs new/extended cases for: Queen-gated dispatch (RESEARCH-01/02), replan re-research (RESEARCH-04), survey-in-brief (RESEARCH-05)
- [ ] A new or extended Go test file for the depth-proposal card (RESEARCH-09/10) — no existing test file targets `plan.md`'s Depth Ceremony content directly since it's currently static wrapper prose; once Go computes the reasons, a Go-side test becomes possible
- [ ] `.aether/ts-host/test/research-confidence-loop.test.ts` (or extension of `confidence-loop.test.ts`) for RESEARCH-07/08 — must assert `new ConfidenceLoop(...)` is actually constructed and driven from the research dispatch path, not merely that the class works in isolation (the class is already tested; the gap is the *construction site*)
- [ ] A regression test pinning D-08 (research-worker-failure warning text appears in both the plan artifact and finalize output) — no existing test covers research-worker failure specifically today (only `hasWorkerAuthoredResearch`'s success-path behavior is tested)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | No auth surface touched — internal CLI/wrapper orchestration only |
| V3 Session Management | No | No session/token handling in this phase |
| V4 Access Control | Yes (narrow) | `codex.PermissionProfileForCaste(caste)` — already correctly scopes `scout` to `repository_read_only`; verify the same holds for `oracle` when D-04 escalation is implemented (Oracle's existing profile should already be defined for its existing standalone command, but confirm it stays read-appropriate when invoked from the plan-time escalation path, not just from `/ant-oracle`) |
| V5 Input Validation | Yes | User's tick-to-approve flips and depth-card selections are structured (index/id selections per D-13's "selection, not forms"), not free text — validate flip/selection payloads against the known candidate set (phase IDs, depth enum values) the same way existing `--depth`/`--planning-depth` flag validation already rejects unknown values (`resolvePlanningDepth` error path) |
| V6 Cryptography | No | Not applicable |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Prompt injection via phase description/goal text reaching the Queen's hint computation or a research Scout's brief | Tampering | Existing pheromone-content sanitization pattern (`.claude/rules/aether-colony.md`'s "Prompt Injection Sanitization" — XML tag rejection, angle-bracket escaping, instruction-override phrase rejection, 500-char cap) is the established precedent in this codebase; if D-02's hints or the research brief surface user-controlled phase text into a worker prompt, apply the same sanitization discipline already used for pheromone signals rather than passing raw text through |
| A malicious/malformed phase description tricking the "domain absent from survey map" hint into over- or under-triggering research | Tampering (low severity) | Low risk — worst case is an extra (cost) or missing (quality) research dispatch, not a security boundary violation; no mitigation beyond normal input validation needed |
| Escalating to Oracle without preserving the read-only boundary Scout has today | Elevation of Privilege | Confirm `codex.PermissionProfileForCaste("oracle")` does not silently grant broader write access than intended for the plan-time escalation use case specifically (Oracle's standalone `/ant-oracle` command may have different write needs — e.g. writing `ORACLE.md` — than a plan-time research escalation should have) |

## Sources

### Primary (HIGH confidence — all verified by direct file inspection in this repository, 2026-08-02)
- `cmd/phase_research.go` — full file read; dispatch emission, staleness check, brief rendering, build-brief injection
- `cmd/phase_research_dispatch_test.go` — existing test coverage and pinned behaviors
- `.aether/ts-host/src/confidence-loop.ts` — full file read; `ConfidenceLoop` class, `ConfidenceLoopOptions`, stop-condition logic
- `.aether/ts-host/src/confidence-evaluator.ts` — full file read; scoring algorithm and `ConfidenceMetric` bridge
- `.aether/ts-host/src/host.ts` — read `runDispatchedBuildCommand`, `runDispatchedPlanCommand`, `runDispatchedContinueCommand`, arg parsing, ceremony helpers
- `cmd/codex_plan.go` — read `codexPlanningLoop`, `planningLoopPreset`, `resolvePlanningLoopOptions`, `runCodexPlanPlanOnly`, `renderPlanningWorkerBrief`, `planningWorkerSpecForCaste`
- `cmd/codex_plan_finalize.go` — read `evaluatePlanningLoopIteration`
- `cmd/orchestrator_boundary_questions.go`, `cmd/orchestrator_boundary_guidance.go` — read in full; confirmed mode-gating
- `cmd/pending_decision.go` — read in full; existing decision store
- `cmd/plan_grounding.go` — read in full; confirmed `researchPhaseKeywords`/`isResearchPhase` is a Phase-167 removal target, not a D-02 building block
- `cmd/oracle_loop.go` — grepped for loop/target/iteration structure; confirmed independent third loop
- `cmd/codex_visuals.go` — read `renderReviewDepthLineWithReason`, caste identity helpers, planning-loop ceremony rendering
- `pkg/colony/granularity.go`, `pkg/colony/colony.go` (PhaseMode) — read in full
- `.claude/commands/ant/plan.md` — read in full; current wrapper choreography
- `.planning/phases/163-context-reaches-workers/163-CONTEXT.md`, `.planning/phases/160-fail-loudly/160-CONTEXT.md` — read for D-04/D-07/D-08/manifest-path precedent
- `.planning/REQUIREMENTS.md`, `.planning/STATE.md`, `.planning/phases/164-research-feeds-planning/164-CONTEXT.md` — full read
- `git ls-files .aether/ts-host/dist`, `git check-ignore -v .aether/data/phase-research/` — verified dist is tracked/embedded and phase-research data is gitignored

### Secondary (MEDIUM confidence)
- None — no WebSearch or external documentation was needed; this phase's entire surface is internal repository code

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero new dependencies, every referenced function/file confirmed to exist by direct reading
- Architecture (Three Loops finding): HIGH — confirmed by grepping every `ConfidenceLoop`/`codexPlanningLoop`/`oracle_loop` construction site in the repo; no construction of the TS class exists outside `runDispatchedBuildCommand`
- Pitfalls: HIGH — each pitfall is grounded in a specific line-referenced code fact, not speculation
- Design-space items left to discretion (Oracle escalation mechanics, evidence-scoring formula, exact ceremony wording): appropriately unresolved per CONTEXT.md's own "Claude's Discretion" list — documented as Open Questions rather than asserted

**Research date:** 2026-08-02
**Valid until:** Should be re-checked if Phase 160-163 land materially different code than what's on `main` today (this research was performed against the current working tree at commit range ending `1f5283cf`); otherwise valid for the duration of Phase 164 planning and execution (internal-only surface, no external API drift risk)
