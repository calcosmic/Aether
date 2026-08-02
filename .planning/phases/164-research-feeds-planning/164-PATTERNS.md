# Phase 164: Research Feeds Planning - Pattern Map

**Mapped:** 2026-08-02
**Files analyzed:** 12 (5 modified Go, 2 new-ish Go test surfaces, 3 modified/new TS, 2 modified wrapper docs)
**Analogs found:** 12 / 12 (all files extend existing code; this phase is wiring, not greenfield — every target file already exists except one small TS test file and possibly one small Go helper file)

This phase is pure extension of existing functions. Nearly every "new" file
is actually an existing file gaining new logic, so "closest analog" for most
rows below is **the same file's neighboring function**, not a different file.
Where a genuinely new construction site is needed (the research confidence
sub-loop in `host.ts`, the Queen-decision hint computation in Go), the analog
is the nearest sibling function performing the same *shape* of work.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `cmd/phase_research.go` (extend: survey param, Queen-gate, replan inversion) | service (Go dispatch builder) | request-response (manifest construction) | itself — `plannedPhaseResearchDispatches` / `renderPhaseResearchBrief` | exact (self-extension) |
| `cmd/phase_research_decision.go` (NEW — D-02 hint computation + D-01 batch struct) | service (Go decision/hint builder) | transform (signals → recommendation) | `cmd/plan_grounding.go` (`checkPlanGrounding`, phase-scanning shape) + `cmd/review_depth.go` (`phaseRiskLevel`, `renderSmartDepthReason`, reason-string shape) | role-match |
| `cmd/phase_research_dispatch_test.go` (extend) | test | CRUD/unit | itself — `TestPhaseResearchDispatchedOncePerPhase`, `TestBuildBriefCarriesPhaseResearch` | exact |
| `cmd/codex_plan.go` (extend: thread survey into research call, wire Queen decision + depth-proposal reasons into manifest) | controller/orchestrator (Go manifest assembly) | request-response | itself — the `researchDispatches := plannedPhaseResearchDispatches(...)` call site (line ~900) and `renderPlanningWorkerBrief` (survey-injection pattern, line 1919) | exact (self-extension) |
| `cmd/codex_plan_finalize.go` (extend: D-08 "planned WITHOUT its research" state) | controller (Go finalize) | request-response | `cmd/codex_plan_finalize.go` — `evaluatePlanningLoopIteration` (existing loop-A finalize logic) | role-match |
| `cmd/pending_decision.go` (reuse as-is — D-03/D-06 overrides) | service (Go decision store) | CRUD | itself — `pendingDecisionAddCmd` / `pendingDecisionResolveCmd` | exact (no changes needed) |
| `cmd/oracle_loop.go` (reuse as-is — D-04 escalation target) | service (Go RALF loop) | event-driven / batch | itself — `runOracleLoop`, `oracleStateFile` | exact (no changes needed) |
| `cmd/codex_visuals.go` (extend: ceremony/reason text for proposal cards, D-13) | utility (Go text rendering) | transform | itself — `renderReviewDepthLineWithReason` (lines 1336-1353), `casteIdentity`/`casteEmoji`/`deterministicAntName` (lines 3509-3593) | exact (self-extension) |
| `.aether/ts-host/src/host.ts` (extend: new research confidence sub-loop construction site) | service (TS dispatch/iteration driver) | event-driven (dispatch → evaluate → maybe redispatch) | itself — `runDispatchedBuildCommand` (lines 783-966, the loop shape to mirror) vs. `runDispatchedPlanCommand` (lines 979-1052, today's single-shot plan path with NO `ConfidenceLoop`) | exact (shape to replicate, wrong function today) |
| `.aether/ts-host/src/confidence-loop.ts` (reuse as-is per RESEARCH-07 — construct, don't modify) | service (TS iteration primitive) | transform | itself — `ConfidenceLoop` class | exact (no changes needed) |
| `.aether/ts-host/src/confidence-evaluator.ts` (extend or add research-flavored sibling for D-11) | service (TS scoring) | transform | itself — `ConfidenceEvaluator.evaluate()` (worker-claims scoring shape to mirror for evidence scoring) | role-match (new evaluator, same shape) |
| `.aether/ts-host/test/research-confidence-loop.test.ts` (NEW, or extend `confidence-loop.test.ts`) | test | unit | `.aether/ts-host/test/confidence-loop.test.ts` (existing, exercises `ConfidenceLoop` in isolation) | role-match |
| `.claude/commands/ant/plan.md` + `.opencode/commands/ant/plan.md` (extend: 3-knob depth card, research batch card) | provider/wrapper (markdown orchestration spec) | request-response (ceremony rendering) | itself — existing "Depth Ceremony" section (lines 11-33) and research choreography note (lines 74-93) | exact (self-extension) |
| `.aether/commands/plan.yaml` (extend if wrapper spec structurally changes) | config (YAML source of truth) | transform | itself — source YAML for `plan.md` | exact (self-extension, touch only if structure changes) |

## Pattern Assignments

### `cmd/phase_research.go` (service, request-response)

**Analog:** itself — `plannedPhaseResearchDispatches` / `hasWorkerAuthoredResearch` / `renderPhaseResearchBrief`

**Imports pattern** (lines 1-10):
```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)
```

**Core dispatch-gating pattern — the exact function D-01/D-02/D-05 converge on** (lines 61-89):
```go
func plannedPhaseResearchDispatches(root, planDepth, goal string, candidates []phaseResearchCandidate) []codexPlanningDispatch {
	if planDepth == "fast" || len(candidates) == 0 {
		return nil
	}
	dispatches := make([]codexPlanningDispatch, 0, len(candidates))
	for _, candidate := range candidates {
		fileName := fmt.Sprintf("phase-%d-research.md", candidate.ID)
		existingPath := filepath.Join(root, ".aether", "data", "phase-research", fileName)
		if hasWorkerAuthoredResearch(existingPath) {
			continue // <- D-05 must invert this specifically for the replan case
		}
		dispatches = append(dispatches, codexPlanningDispatch{
			Stage:     phaseResearchStage,
			Wave:      1,
			Caste:     "scout",
			AgentName: "aether-scout",
			Name:      deterministicAntName("scout", fmt.Sprintf("%s|plan-research|phase-%d", root, candidate.ID)),
			Task:      fmt.Sprintf("Research domain knowledge for phase %d: %s", candidate.ID, firstNonEmpty(candidate.Name, "unnamed phase")),
			TaskID:    fmt.Sprintf("plan-research-phase-%d", candidate.ID),
			Outputs:   []string{fileName},
			Status:    "planned",
			Brief:     renderPhaseResearchBrief(goal, candidate),
		})
	}
	attachPlanningDispatchSkillAssignments(dispatches)
	return dispatches
}
```
To satisfy D-01/D-02/D-05 this function needs two new parameters (or a wrapping call): (a) a per-candidate approval flag from the Queen's approved batch, replacing the unconditional loop; (b) a replan-aware override of the `hasWorkerAuthoredResearch` skip (e.g. `opts.Refresh` threaded in, inverting the `continue`).

**Staleness-check pattern (D-05's exact inversion target)** (lines 91-100):
```go
func hasWorkerAuthoredResearch(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return !strings.Contains(string(data), phaseResearchTemplateMarker)
}
```

**Brief-rendering pattern to extend with survey (RESEARCH-05)** (lines 105-136, no survey param today):
```go
func renderPhaseResearchBrief(goal string, candidate phaseResearchCandidate) string {
	var b strings.Builder
	b.WriteString("You are a Scout performing Phase Domain Research.\n\n")
	b.WriteString("## Mission\n")
	b.WriteString(fmt.Sprintf("Investigate the domain knowledge needed for Phase %d: %s\n", candidate.ID, firstNonEmpty(candidate.Name, "unnamed phase")))
	// ... six-section output contract unchanged; add a
	// "Survey docs to read first" block here, mirroring renderPlanningWorkerBrief below
}
```

**Build-brief injection pattern (already correct, unchanged)** (lines 143-170): `resolvePhaseResearchSection` — budget-truncated at `phaseResearchBriefBudgetChars = 3500`, truncates at a paragraph boundary and appends a pointer to the full file. This is the pattern D-08's "planned WITHOUT its research" fallback text and any Route-Setter content-injection (Pitfall 4) should mirror for consistent truncation behavior.

---

### `cmd/phase_research_decision.go` (NEW — service, transform)

**Analog:** `cmd/plan_grounding.go` (phase-scanning shape) + `cmd/review_depth.go` (reason-string shape) + `cmd/orchestrator_boundary_questions.go` (`discussQuestion` struct shape, for JSON contract inspiration only — do NOT reuse the delivery mechanism, see Anti-Pattern below)

**Phase-mode guard pattern to copy for D-02's "phase mode" hint (Pitfall 3)** — `cmd/plan_grounding.go:46` area:
```go
// researchPhaseKeywords identifies phases that legitimately lack file targets.
var researchPhaseKeywords = []string{"research", "survey", "architecture", "design", "planning", "discovery"}
```
**Do not build D-02's hints on top of this list** — it is a Phase-167 removal target (see Anti-Patterns below). It is shown here only as the `if phase.Mode.Valid() { ... }`-style defensive-guard precedent for a currently-unreliable field.

**Reason-string computation pattern to copy** (`cmd/codex_visuals.go:1308-1330`):
```go
func renderSmartDepthReason(phase colony.Phase, totalPhases int) string {
	risk := phaseRiskLevel(phase)
	position := phasePositionLevel(phase.ID, totalPhases)

	if risk == "high" {
		return getSmartDefaultReason("high_risk")
	}
	if position == "final" {
		return getSmartDefaultReason("final_phase")
	}
	// ...
	return getSmartDefaultReason("standard")
}
```

**Discussion-question struct shape to borrow (JSON contract only, not the delivery path)** (`cmd/orchestrator_boundary_questions.go:43-58`):
```go
func planBoundaryQuestionCandidates(state colony.ColonyState, granularity colony.PlanGranularity, planDepth, planningDepth, verificationDepth string) []discussQuestion {
	goal := strings.TrimSpace(derefGoal(state.Goal))
	reasoning := fmt.Sprintf("Planning is about to choose boundaries for %s with %s granularity, %s planning depth, and %s verification depth.", goal, granularity, emptyFallback(planDepth, "default"), emptyFallback(verificationDepth, "default"))
	return []discussQuestion{{
		Category:  "planning-scope",
		Question:  "What should the first generated plan optimize for?",
		Options:   []string{"smallest useful slice", "balanced milestone plan", "surface risky dependencies first"},
		Reasoning: reasoning,
	}}
}
```
`discussQuestion{Category, Question, Options []string, Reasoning}` is the right *field shape* for D-01/D-13's proposal cards. The delivery mechanism (`materializeOrchestratorBoundaryQuestions`, gated at `cmd/orchestrator_boundary_questions.go:23` by `state.EffectiveColonyMode() != colony.ColonyModeOrchestrator`) must NOT be reused — see Anti-Patterns.

---

### `cmd/codex_plan.go` (controller, request-response)

**Analog:** itself — the research-dispatch call site and `renderPlanningWorkerBrief`

**Survey-in-scope-but-not-threaded pattern (the exact gap for RESEARCH-05)** (lines 874-900):
```go
survey, err := loadCodexSurveyContext(root)
if err != nil {
	return nil, err
}
// ... survey is in scope here ...
researchDispatches := plannedPhaseResearchDispatches(root, planDepth, *state.Goal, phaseResearchCandidates(state, iterationSeed))
// ^ survey available but not passed — add survey param to
//   plannedPhaseResearchDispatches / renderPhaseResearchBrief
if len(researchDispatches) > 0 {
	for i := range dispatches {
		if dispatches[i].Caste == "route_setter" {
			dispatches[i].Brief += "\n\n## Phase Research Available\n\nParallel research Scouts are writing per-phase findings to `.aether/data/phase-research/phase-N-research.md` during wave 1. Read each phase's research before finalizing the route, and fold its Recommended Approach and Gotchas into task constraints and hints.\n"
		}
	}
	dispatches = append(dispatches, researchDispatches...)
}
```
This is Pitfall 4's exact location: the Route-Setter brief gets a *pointer sentence*, not content. If RESEARCH-03 requires actual content injection, extend this append block using `resolvePhaseResearchSection`'s truncation pattern (`cmd/phase_research.go:143-170`) rather than inventing new truncation logic.

**Survey-injection pattern to copy verbatim for `renderPhaseResearchBrief`** (lines 1919-1930, `renderPlanningWorkerBrief`):
```go
func renderPlanningWorkerBrief(root string, survey codexSurveyContext, spec planningWorkerSpec, scoutGuidanceOpt ...string) string {
	planningDir := filepath.ToSlash(filepath.Join(".aether", "data", "planning"))
	surveyDir := filepath.ToSlash(filepath.Join(".aether", "data", "survey"))
	surveyDocs := make([]string, 0, len(survey.SurveyDocs))
	for _, name := range survey.SurveyDocs {
		surveyDocs = append(surveyDocs, filepath.ToSlash(filepath.Join(surveyDir, name)))
	}
	var b strings.Builder
	b.WriteString("Use the existing survey artifacts first before scanning the wider repository.\n")
	b.WriteString("- Primary survey source: ")
	b.WriteString(surveyDir)
	b.WriteString("\n")
	if len(surveyDocs) > 0 {
		b.WriteString("- Survey docs to read first: ")
		b.WriteString(strings.Join(surveyDocs, ", "))
		b.WriteString("\n")
	}
	// ...
}
```

**Depth-preset numeric source of truth (D-12's numbers — do NOT bind research to this loop, only borrow the numbers)** (lines 1689-1700):
```go
func planningLoopPreset(planDepth string) (int, int) {
	switch strings.ToLower(strings.TrimSpace(planDepth)) {
	case "fast":
		return 80, 4
	case "deep":
		return 95, 8
	case "exhaustive":
		return 99, 12
	default:
		return 90, 6
	}
}
```

---

### `cmd/pending_decision.go` (service, CRUD — reuse as-is)

**Analog:** itself. No changes required — D-03/D-06 write overrides through this existing store.

**Imports pattern** (lines 1-8):
```go
package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)
```

**CRUD pattern to call into (add)** (lines 32-84, `pendingDecisionAddCmd` — note the JSON shape and scope stamping):
```go
decision := PendingDecision{
	ID:          fmt.Sprintf("pd_%d", time.Now().UnixNano()),
	Type:        decisionType,
	Description: description,
	Source:      source,
	Resolved:    false,
	CreatedAt:   time.Now().UTC().Format(time.RFC3339),
}
if phase > 0 {
	decision.Phase = &phase
}
stampPendingDecisionScope(&decision, loadCurrentPendingDecisionScope())

var file PendingDecisionFile
if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
	file = PendingDecisionFile{Decisions: []PendingDecision{}}
}
file.Decisions = append(file.Decisions, decision)
if err := store.SaveJSON(pendingDecisionsFile, file); err != nil {
	outputError(2, fmt.Sprintf("failed to save decisions: %v", err), nil)
	return nil
}
```
D-03's override ("user overrode: skip research on phase 3") should call `pendingDecisionAddCmd`'s underlying logic with `Type: "research-override"`, `Phase: &candidateID`, and `Description` carrying the flip direction — no new struct, no new file.

**Error handling pattern** (consistent across all three commands in this file):
```go
if store == nil {
	outputErrorMessage("no store initialized")
	return nil
}
// ...
if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
	outputError(1, "pending-decisions.json not found", nil)
	return nil
}
```

---

### `.aether/ts-host/src/host.ts` (service, event-driven dispatch/iterate)

**Analog:** `runDispatchedBuildCommand` (the shape to mirror) vs. `runDispatchedPlanCommand` (today's plan path — confirmed to construct zero `ConfidenceLoop` instances)

**Imports/setup pattern already present at top of file** — `ConfidenceLoop`, `ConfidenceLoopOptions`, `ConfidenceEvaluator`, `EvaluatedConfidence` are already imported and used by the build path; the research path reuses the same imports, no new dependency.

**Loop-B construction + iterate-until-stop pattern to replicate for research** (lines 842-920):
```typescript
const loopOpts: ConfidenceLoopOptions = {
  totalBudget: spawnBudget,
};
if (parsed.maxIterations) {
  loopOpts.maxIterations = parseInt(parsed.maxIterations, 10);
}
if (parsed.targetConfidence) {
  loopOpts.confidenceTarget = parseInt(parsed.targetConfidence, 10);
}
const confidenceLoop = new ConfidenceLoop(loopOpts);
const confidenceEvaluator = new ConfidenceEvaluator();

let lastWaveResult: WaveResult | undefined;
while (true) {
  lastWaveResult = await dispatchBuildWave(bridge, parsed, ceremony, buildManifest, dispatches, spawnOrchestrator, iterationFeedback);
  const evaluated: EvaluatedConfidence = confidenceEvaluator.evaluate({ workerClaims: lastWaveResult.workerClaims });
  const loopResult = confidenceLoop.evaluate(evaluated.score, lastWaveResult.workerCount);
  renderIterationCeremony(loopResult);
  if (!loopResult.shouldContinue) {
    renderIterationComplete(loopResult.stopReason);
    break;
  }
}
```
For research (D-12), depth binds `loopOpts.confidenceTarget`/`loopOpts.maxIterations` to the fast/balanced/deep/exhaustive presets (80/4, 90/6, 95/8, 99/12) instead of parsed CLI flags directly, and the evaluator is a research-flavored sibling of `ConfidenceEvaluator` (see below) instead of the worker-claims one, because research findings don't have `test_results`/`files_created` in the same shape.

**Ceremony-rendering pattern (D-09 — reuse, do not fork a third copy, Pitfall 5)** (lines 640-657):
```typescript
function renderIterationCeremony(result: ConfidenceResult): void {
  const parts: string[] = [
    `Iteration ${result.iterationCount}:`,
    `confidence ${result.currentConfidence}%`,
    `(delta ${result.delta >= 0 ? "+" : ""}${result.delta}%,`,
    `budget ${result.budgetRemaining} workers remaining)`,
  ];
  if (result.stopReason) {
    parts.push(`[${result.stopReason}]`);
  }
  emitCeremonyOutput(`── ${parts.join(" ")} ──`);
}
function renderIterationComplete(stopReason: string): void {
  emitCeremonyOutput(`── Iteration complete: ${stopReason} ──`);
}
```
Note: this free function duplicates `ConfidenceLoop.renderIterationCeremony()` (the instance method inside `confidence-loop.ts:158-171`). Extend this duplication consistently for research rather than adding a third copy — call the same free function or the instance method, not both.

**Confirmed absence — today's single-shot plan path (do NOT copy this shape for research)** (lines 979-1052, `runDispatchedPlanCommand`): no `ConfidenceLoop` construction anywhere in this function; it dispatches once, writes one completion file, and calls `plan-finalize` once. This is the function proving RESEARCH-07 is unmet today.

---

### `.aether/ts-host/src/confidence-loop.ts` (service, transform — reuse as-is)

**Analog:** itself. RESEARCH-07 explicitly forbids modifying this class — only construct `new ConfidenceLoop(...)` from a new call site.

**Constructor options this phase binds via D-12** (lines 19-30):
```typescript
export interface ConfidenceLoopOptions {
  maxIterations?: number;        // Default: 3 -- research: 4/6/8/12 by depth
  confidenceTarget?: number;     // Default: 80 -- research: 80/90/95/99 by depth
  diminishingReturnsThreshold?: number; // Default: 5
  diminishingReturnsWindow?: number;    // Default: 2
  totalBudget?: number;          // Default: 20
}
```

**Stop-condition priority order (governs D-10's early-accept trigger and D-04's escalation trigger)** (lines 201-227):
```typescript
private checkStopConditions(currentConfidence: number, iterationCount: number, budgetRemaining: number): string {
  if (iterationCount >= this.maxIterations) return "max_iterations_met";
  if (currentConfidence >= this.confidenceTarget) return "confidence_target_met";
  if (this.totalBudget > 0 && budgetRemaining <= 0) return "budget_exhausted";
  if (this.isDiminishingReturns()) return "diminishing_returns";
  return "";
}
```
D-04's "stalls below target at deep/exhaustive" reads directly off `stopReason === "diminishing_returns"` combined with a depth-tier check and `currentConfidence < confidenceTarget` — this is the exact signal to key Oracle escalation off (per RESEARCH.md Open Question 3's recommendation).

---

### `.aether/ts-host/src/confidence-evaluator.ts` (service, transform — extend with a sibling)

**Analog:** itself — `ConfidenceEvaluator.evaluate()`, whose scoring shape (base + bonuses − penalties, clamped 0-100) is the pattern a research-flavored evaluator should mirror, substituting evidence checks for worker-claims checks.

**Scoring pattern to mirror** (lines 76-176):
```typescript
export class ConfidenceEvaluator {
  evaluate(input: ConfidenceInput): EvaluatedConfidence {
    let score = BASE_SCORE; // 50
    if (totalTests > 0) score += TEST_BONUS_MAX * (totalPassed / totalTests); // +30 max
    if (filesCovered > 0) score += FILE_BONUS; // +10
    if (totalWorkers > 0 && completedWorkers === totalWorkers) score += ALL_COMPLETED_BONUS; // +10
    const blockerPenalty = Math.min(blockers.length * BLOCKER_PENALTY, BLOCKER_PENALTY_CAP); // -10 each, cap -30
    score -= blockerPenalty;
    score = Math.max(SCORE_MIN, Math.min(SCORE_MAX, Math.round(score)));
    return { score, test_pass_rate: testPassRate, files_covered: filesCovered, blockers, source };
  }
}
```
For D-11's evidence-scoring blend, a `ResearchConfidenceEvaluator` (or an extended `evaluate` overload) should replace the worker-claims-shaped bonuses with: required sections filled (bonus per section present out of the six), citations with real paths/URLs verified (bonus, requires a Go-supplied fact since Node can check file existence directly against the repo root too), and files-to-study existing on disk (bonus). Blend with the researcher's own self-reported gap count (penalty per unresolved gap), same clamp-to-[0,100] pattern. Must be reproducible per D-11/Definition of Done — no randomness, no un-testable LLM self-score alone.

---

### `.claude/commands/ant/plan.md` (provider/wrapper, request-response)

**Analog:** itself — existing "Depth Ceremony" section

**Current static, ungrounded list to extend into a reasoned 3-knob card (RESEARCH-10/D-13)** (lines 11-30):
```markdown
## Depth Ceremony

Before requesting a planning manifest, choose the planning depth.

If `$ARGUMENTS` already contains one of `fast`, `balanced`, `deep`, or `exhaustive`, use that value and state the selection. Otherwise ask the user once:

1. Fast — sprint granularity, 1-3 phases
2. Balanced — milestone granularity, 4-7 phases. Recommended default
3. Deep — quarter granularity, 8-12 phases
4. Exhaustive — major granularity, 13-20 phases

Do not continue until a depth is selected.

## Planning Depth

After selecting planning depth, choose task decomposition depth. If `$ARGUMENTS` already contains `light`, `standard`, or `deep` as planning-depth, use it. Otherwise default to `standard`:

1. Light — coarse tasks, 1-3 per plan
2. Standard — normal task breakdown. Default
3. Deep — granular subtasks with edge cases and test coverage
```
This is two separate ceremonies today (granularity, then planning depth) with no verification-depth knob and no computed reason — D-13/D-14 require one combined 3-knob card with a pre-marked recommendation and a plain-English reason per knob, sourced from Go (`renderReviewDepthLineWithReason`-style output), not invented wrapper prose.

**Research choreography note to keep accurate (already correct today, do not regress)** (lines 74-79, 93):
```markdown
Dispatch every worker in `plan_manifest.dispatches`, using manifest names, castes, task IDs, briefs, `permission_profile`, and `agent_name` as `subagent_type`. The set is: one Scout in wave 1, zero or more `phase_research` Scouts also in wave 1 (parallel with the base Scout — one per drafted phase, researching its domain), then exactly one Route-Setter in wave 2. Scout's `repository_read_only` profile must remain host-enforced...
...
All wave-1 workers (base Scout and any research Scouts) must complete before the wave-2 Route-Setter starts — the route is set with research in hand.
```
Add the D-01 batched research-approval card as a new step before this dispatch block (after the manifest fetch returns per-phase recommendations+reasons), using the `suggest-approve` tick-to-approve rendering pattern below, not a `discussQuestion`/`aether discuss` redirect.

**`suggest-approve`-style rendering pattern to copy for D-01's batch card** (`cmd/ceremony_cmd.go:611-626`, `renderPendingSuggestionsBlock`):
```go
func renderPendingSuggestionsBlock(suggestions []colony.PendingSuggestion) string {
	var b strings.Builder
	for i, s := range suggestions {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "[%s] %s\n", s.Type, s.Content)
		if reason := strings.TrimSpace(s.Reason); reason != "" {
			fmt.Fprintf(&b, "  Reason: %s\n", reason)
		}
		fmt.Fprintf(&b, "  ID: %s\n", s.ID)
		fmt.Fprintf(&b, "  Approve: aether suggest-approve --approve %s\n", s.ID)
		fmt.Fprintf(&b, "  Dismiss: aether suggest-approve --dismiss %s\n", s.ID)
	}
	return b.String()
}
```
The research batch needs the same shape: one Go-rendered list (phase, recommendation, reason, ID), one wrapper-rendered interaction, one CLI call back per flipped entry (reusing `pendingDecisionResolveCmd` or a thin new verb) — never a redirect into `/ant-discuss`.

---

## Shared Patterns

### Caste identity / ceremony rendering (D-09)
**Source:** `cmd/codex_visuals.go:3509-3603` (`deterministicAntName`, `casteEmoji`, `casteLabel`, `casteIdentity`, `colorizeCaste`) and `cmd/codex_visuals.go:65-84` (`casteColorMap`)
**Apply to:** Every research Scout dispatch (already applied at `cmd/phase_research.go:79`) and any Oracle-escalation dispatch (D-04) — `casteIdentity("oracle")` resolves correctly today since the lookup is caste-keyed.
```go
Name: deterministicAntName("scout", fmt.Sprintf("%s|plan-research|phase-%d", root, candidate.ID)),
```
```go
func casteIdentity(caste string) string {
	return casteEmoji(caste) + " " + colorizeCaste(caste, casteLabel(caste))
}
```

### Smart-default reason computation (RESEARCH-10, D-13)
**Source:** `cmd/review_depth.go:100-132` (`getSmartDefaultReason`), `cmd/review_depth.go:243-330` (`phasePositionLevel`, `phaseRiskLevel`, `resolveSmartPlanningDepth`, `resolveSmartVerificationDepth`), `cmd/codex_visuals.go:1308-1353` (`renderSmartDepthReason`, `renderReviewDepthLineWithReason`)
**Apply to:** All three D-13 knobs (granularity, planning depth, verification depth) — planning-depth and verification-depth already have working reason-computation; granularity does not (confirmed via research grep — no existing `renderGranularity*Reason` function) and needs small new Go logic following the same `getSmartDefaultReason(key)` lookup-table shape.
```go
func renderReviewDepthLineWithReason(depth colony.VerificationDepth, phaseNum, totalPhases int, phase colony.Phase, smartDefault bool) string {
	base := renderReviewDepthLine(depth, phaseNum, totalPhases)
	if !smartDefault {
		return base
	}
	reason := renderSmartDepthReason(phase, totalPhases)
	switch depth {
	case colony.VerificationDepthHeavy:
		return fmt.Sprintf("Review depth: heavy (%s)", reason)
	// ...
	}
}
```

### Granularity range lookup (RESEARCH-09)
**Source:** `pkg/colony/granularity.go` (full file, 18 lines)
**Apply to:** The granularity knob's proposal-card option list.
```go
func GranularityRange(g PlanGranularity) (min int, max int) {
	switch g {
	case GranularitySprint:
		return 1, 3
	case GranularityMilestone:
		return 4, 7
	case GranularityQuarter:
		return 8, 12
	case GranularityMajor:
		return 13, 20
	default:
		return 1, 3
	}
}
```

### Decision/override persistence (D-03, D-06, RESEARCH-06)
**Source:** `cmd/pending_decision.go` (full file) — `PendingDecisionFile`, `pendingDecisionAddCmd`, `pendingDecisionResolveCmd`, scope-matching (`loadCurrentPendingDecisionScope`, `pendingDecisionMatchesScope`)
**Apply to:** Every user flip in the D-01 research batch and every accept/change in the D-13 depth card — no new JSON store, reuse `pending-decisions.json` via the existing three CLI verbs.

### Depth → target/iteration numeric binding (D-12)
**Source:** `cmd/codex_plan.go:1689-1700` (`planningLoopPreset`) — numeric source of truth only, NOT the loop to wire into.
```go
func planningLoopPreset(planDepth string) (int, int) {
	switch strings.ToLower(strings.TrimSpace(planDepth)) {
	case "fast": return 80, 4
	case "deep": return 95, 8
	case "exhaustive": return 99, 12
	default: return 90, 6
	}
}
```
**Apply to:** The research confidence sub-loop's `ConfidenceLoopOptions` construction in `host.ts` (Loop B), fed by these same four numbers — but constructed at a NEW site (see Anti-Patterns), not by rerouting through `resolvePlanningLoopOptions`/`codexPlanningLoop` (Loop A).

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| Research-specific `ConfidenceEvaluator` sibling (evidence + self-check blend, D-11) | service | transform | No existing evaluator scores markdown-section-presence/citation-verification; `ConfidenceEvaluator.evaluate()` is worker-claims-shaped (test pass rate, files touched, blockers) and is the nearest structural analog but requires new scoring logic, not reuse |
| Granularity reason-computation helper (RESEARCH-09/10's third knob) | utility | transform | Confirmed via research pass: no `renderGranularity*Reason` or equivalent exists; `renderSmartDepthReason`/`getSmartDefaultReason` are the pattern to replicate, not something to call directly |
| D-01 batch-approval CLI verb (if `pendingDecisionResolveCmd` proves too generic for "phase N flip") | controller | request-response | `pendingDecisionResolveCmd` resolves by opaque `id` + free-text `resolution`; if the research batch needs phase-indexed bulk resolution in one call, no existing bulk-resolve verb exists — extend `pending_decision.go` with a thin new verb only if per-ID resolution proves too chatty for one card |

## Anti-Patterns to Avoid (carried from RESEARCH.md, load-bearing for planning)

- **Do not construct a fourth confidence loop.** Three already exist (Go `codexPlanningLoop`/`planningLoopPreset`, TS `ConfidenceLoop`, Go `oracle_loop.go`'s RALF loop). New code must construct `new ConfidenceLoop(...)` from `.aether/ts-host/src/confidence-loop.ts`, or delegate to `oracle_loop.go`'s existing loop for D-04 escalation — never a new counter/threshold/stop-reason structure.
- **Do not route D-01/D-13 through `materializeOrchestratorBoundaryQuestions`/`aether discuss`** (`cmd/orchestrator_boundary_questions.go:22-34`) — it is hard-gated to `state.EffectiveColonyMode() == colony.ColonyModeOrchestrator` and its resolution path is a separate blocking command/session, violating D-14's "exactly two decision moments, no third ceremony." The `discussQuestion` struct shape (`Category`, `Question`, `Options []string`, `Reasoning`) can inspire the JSON contract; the delivery mechanism must not be reused wholesale.
- **Do not build D-02's hints on `researchPhaseKeywords`/`isResearchPhase`** (`cmd/plan_grounding.go:16-92`) — explicitly named in `.planning/REQUIREMENTS.md` (TYPED-05) as a Phase-167 removal target.
- **Do not treat `planningLoopPreset`'s matching numbers as proof RESEARCH-08 is satisfied** — that function feeds Loop A (whole-plan-draft confidence), which has no per-phase concept at all. Only a `new ConfidenceLoop(` construction reachable from the research dispatch/evaluate path satisfies RESEARCH-07/08.
- **Do not fork a third copy of `renderIterationCeremony`** — it already exists twice (`host.ts:641` free function, `confidence-loop.ts:158` instance method) with the same output format; extend the existing duplication consistently, don't add a third.

## Metadata

**Analog search scope:** `cmd/` (phase_research*.go, codex_plan*.go, pending_decision.go, oracle_loop.go, codex_visuals.go, review_depth.go, orchestrator_boundary_questions.go, ceremony_cmd.go, plan_grounding.go), `pkg/colony/granularity.go`, `.aether/ts-host/src/` (host.ts, confidence-loop.ts, confidence-evaluator.ts), `.claude/commands/ant/plan.md`
**Files scanned:** 15 read in full or targeted, cross-verified against RESEARCH.md's independently-researched line citations (all matched on direct re-read; no discrepancies found)
**Pattern extraction date:** 2026-08-02
