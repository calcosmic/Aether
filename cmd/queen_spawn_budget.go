package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

type queenSpawnBudget struct {
	FlowType       string
	MaxWorkers     int
	RequiredCastes []string
	RiskLevel      string
	Reason         string
}

type queenSpawnBudgetDecision struct {
	Caste     string
	Score     int
	Required  bool
	Selected  bool
	Rationale string
}

func queenSpawnBudgetForPhase(phase colony.Phase, flowType string, state colony.ColonyState) queenSpawnBudget {
	flowType = normalizeQueenFlowType(flowType)
	riskLevel := phaseRiskLevel(phase)
	requiredCastes := queenRequiredCastesForBudget(phase, flowType, state)
	maxWorkers, reason := queenMaxWorkersForBudget(phase, flowType, state, riskLevel)

	return queenSpawnBudget{
		FlowType:       flowType,
		MaxWorkers:     maxWorkers,
		RequiredCastes: requiredCastes,
		RiskLevel:      riskLevel,
		Reason:         reason,
	}
}

func applyQueenSpawnBudget(dispatches []CasteDispatch, phase colony.Phase, flowType string, state colony.ColonyState) []CasteDispatch {
	if len(dispatches) == 0 {
		return nil
	}

	budget := queenSpawnBudgetForPhase(phase, flowType, state)
	decisions := queenSpawnBudgetDecisions(dispatches, budget)
	selectedByCaste := make(map[string]queenSpawnBudgetDecision, len(decisions))
	for _, decision := range decisions {
		if decision.Selected {
			selectedByCaste[decision.Caste] = decision
		}
	}

	selected := make([]CasteDispatch, 0, len(decisions))
	for _, dispatch := range dispatches {
		decision, ok := selectedByCaste[dispatch.Caste]
		if !ok {
			continue
		}
		selected = append(selected, CasteDispatch{
			Caste:     decision.Caste,
			Score:     decision.Score,
			Rationale: decision.Rationale,
			FlowType:  budget.FlowType,
		})
	}
	return selected
}

func queenSpawnBudgetDecisions(dispatches []CasteDispatch, budget queenSpawnBudget) []queenSpawnBudgetDecision {
	required := stringSet(budget.RequiredCastes)
	ordered := append([]CasteDispatch(nil), dispatches...)
	sort.SliceStable(ordered, func(i, j int) bool {
		leftRequired := required[ordered[i].Caste]
		rightRequired := required[ordered[j].Caste]
		if leftRequired != rightRequired {
			return leftRequired
		}
		if ordered[i].Score != ordered[j].Score {
			return ordered[i].Score > ordered[j].Score
		}
		return ordered[i].Caste < ordered[j].Caste
	})

	requiredCount := 0
	for _, dispatch := range ordered {
		if required[dispatch.Caste] {
			requiredCount++
		}
	}
	selectionLimit := budget.MaxWorkers
	if requiredCount > selectionLimit {
		selectionLimit = requiredCount
	}

	decisions := make([]queenSpawnBudgetDecision, 0, len(ordered))
	for i, dispatch := range ordered {
		isRequired := required[dispatch.Caste]
		selected := isRequired || i < selectionLimit
		rationale := dispatch.Rationale
		if selected && !isRequired && budget.MaxWorkers > 0 && len(ordered) > budget.MaxWorkers {
			rationale = appendQueenBudgetRationale(rationale, budget)
		}
		if !selected && budget.MaxWorkers > 0 {
			rationale = appendQueenPrunedRationale(rationale, budget)
		}

		decisions = append(decisions, queenSpawnBudgetDecision{
			Caste:     dispatch.Caste,
			Score:     dispatch.Score,
			Required:  isRequired,
			Selected:  selected,
			Rationale: rationale,
		})
	}
	return decisions
}

func queenRequiredCastesForBudget(phase colony.Phase, flowType string, state colony.ColonyState) []string {
	required := make([]string, 0, len(casteRelevanceRegistry))
	for _, profile := range casteRelevanceRegistry {
		if isAlwaysRequired(profile.Caste, flowType, phase, state) || queenBuildSafetyRequiredCaste(profile.Caste, flowType, phase) {
			required = append(required, profile.Caste)
		}
	}
	// The forced reviewer (D-01..D-05) is a CONTINUE-side requirement only —
	// the build announces it but never dispatches it (D-05). Folding it in
	// here, rather than only at the dispatch union in codex_continue.go, keeps
	// queenSpawnBudgetForPhase's RequiredCastes an honest answer to "what does
	// this phase actually require", which is what the check-in card and any
	// caller reading the budget struct directly (not just the final dispatch
	// list) sees.
	if normalizeQueenFlowType(flowType) == "continue" {
		for _, reviewer := range queenForcedReviewersForPhase(phase) {
			required = append(required, reviewer.Caste)
		}
	}
	return uniqueSortedStrings(required)
}

func queenBuildSafetyRequiredCaste(caste, flowType string, phase colony.Phase) bool {
	if normalizeQueenFlowType(flowType) != "build" {
		return false
	}
	return stringSet(queenBuildSafetyRequiredCastes(phase))[caste]
}

// queenBuildSafetyRequiredCastes used to force watcher unconditionally, probe
// wherever code was testable, and auditor/gatekeeper from inferred mode, a
// blast-radius keyword list, and phase position — a one-task bug fix measured
// at eight workers on 2026-08-22 because of this floor. Ruling D11
// (.planning/decisions/2026-08-22-queen-decides-program-checks.md) replaced
// all of that: the floor now returns only the caste that writes the code, and
// a reviewer is forced solely by a named risk signal
// (queenForcedReviewersForPhase, cmd/queen_risk_signals.go) at the checking
// step, never here. Watcher is no longer required at build at all — Phase 193
// (D-08) already stopped dispatching it there; this is the requirement
// following the dispatch. Discovery phases require nothing: they get their
// one researcher from the fallback path (194-CONTEXT.md D-12), not the floor.
func queenBuildSafetyRequiredCastes(phase colony.Phase) []string {
	if effectiveQueenPhaseMode(phase) == colony.PhaseModeDiscovery {
		return nil
	}
	return []string{"builder"}
}

// queenBuildBaseWorkerBudget is the phase's own answer — derived from mode and
// risk — before the operator's verification-depth choice is applied.
func queenBuildBaseWorkerBudget(phase colony.Phase, mode colony.PhaseMode, riskLevel string) (int, string) {
	if mode == colony.PhaseModeMaintenance && riskLevel == "low" && queenPhaseLooksDocumentationOrMaintenance(phase) {
		return 4, "low-risk documentation or maintenance build"
	}
	if riskLevel == "high" || mode == colony.PhaseModeProduction {
		return 8, "high-risk or production build"
	}
	if riskLevel == "medium" {
		return 6, "medium-risk build"
	}
	if mode == colony.PhaseModeDiscovery {
		return 5, "discovery build"
	}
	return 6, "standard build"
}

// applyBuildDepthToBudget binds the operator's verification-depth choice to the
// build worker budget.
//
// Until this existed, the build branch consulted only mode and risk, so
// `--verification-depth light` changed nothing about how many workers a build
// spawned. The numbers 5/6/8 that CLAUDE.md attributed to depth were real but
// keyed to risk instead, which inverted the promise: a light build of a
// production phase still got 8, a heavy build of a discovery phase got 5.
//
// Standard deliberately does NOT clamp. "Standard" means the Queen's ordinary
// judgement, which is exactly what mode and risk already express — imposing a
// flat ceiling there silently weakens the phases that need the most help. An
// earlier draft of this function capped standard at 6 and a DB-migration phase
// promptly lost its Architect, which is the regression this comment exists to
// stop someone reintroducing.
//
// Lowering the cap does not weaken safety. Castes returned by
// queenBuildSafetyRequiredCastes bypass the budget entirely (see the isRequired
// branch in applyQueenSpawnBudget), so a high-risk phase keeps its auditor and
// gatekeeper even at light; what a light cap removes is optional specialists.
func applyBuildDepthToBudget(base int, reason string, depth colony.VerificationDepth) (int, string) {
	switch depth {
	case colony.VerificationDepthLight:
		if base > buildWorkerCapLight {
			return buildWorkerCapLight, reason + ", capped by light verification depth"
		}
	case colony.VerificationDepthHeavy:
		if base < buildWorkerFloorHeavy {
			return buildWorkerFloorHeavy, reason + ", raised by heavy verification depth"
		}
	}
	return base, reason
}

const (
	// buildWorkerCapLight is the ceiling when the operator asks for a light
	// build: enough for the required builder, watcher and probe plus headroom.
	buildWorkerCapLight = 5
	// buildWorkerFloorHeavy is the floor when the operator asks for a heavy
	// build, so choosing heavy on a cheap phase actually buys extra scrutiny.
	buildWorkerFloorHeavy = 8
)

func queenMaxWorkersForBudget(phase colony.Phase, flowType string, state colony.ColonyState, riskLevel string) (int, string) {
	mode := effectiveQueenPhaseMode(phase)

	switch flowType {
	case "build":
		base, reason := queenBuildBaseWorkerBudget(phase, mode, riskLevel)
		return applyBuildDepthToBudget(base, reason, stateVerificationDepth(state))
	case "continue":
		switch stateVerificationDepth(state) {
		case colony.VerificationDepthLight:
			return 3, "light verification"
		case colony.VerificationDepthHeavy:
			return 6, "heavy verification"
		default:
			return 4, "standard verification"
		}
	case "plan":
		if riskLevel == "high" {
			return 6, "high-risk planning"
		}
		return 4, "standard planning"
	case "colonize":
		if stateVerificationDepth(state) == colony.VerificationDepthLight {
			return 2, "light territory survey"
		}
		return 4, "territory survey"
	case "swarm":
		return 5, "focused swarm"
	case "seal":
		if stateVerificationDepth(state) == colony.VerificationDepthHeavy || riskLevel == "high" {
			return 5, "heavy seal review"
		}
		return 4, "standard seal review"
	default:
		return 5, fmt.Sprintf("standard %s flow", flowType)
	}
}

func effectiveQueenPhaseMode(phase colony.Phase) colony.PhaseMode {
	if phase.Mode.Valid() {
		return phase.Mode
	}
	// No keyword inference at runtime — this was the last site where prose
	// could steer dispatch. A phase whose description contained "research"
	// silently became a discovery phase here and got an Oracle instead of a
	// Builder. Modes are now written durably at authoring time
	// (resolveAuthoredPhaseMode) and backfilled by `aether migrate-state`; a
	// phase that still has none gets the neutral default, exactly as
	// InferPhaseMode returns when no keyword matches.
	return colony.PhaseModePrototype
}

// queenPhaseProducesTestableCode reports whether a phase is expected to leave
// behind code that a Probe could write tests against. It is the gate on
// spawning a Probe at all: a Probe on a phase with no new code cannot find a
// coverage gap, so it costs a worker run and returns noise.
func queenPhaseProducesTestableCode(phase colony.Phase) bool {
	// Discovery produces findings, not shipped code.
	if effectiveQueenPhaseMode(phase) == colony.PhaseModeDiscovery {
		return false
	}
	// A repository with no program code in it cannot have test coverage, so a
	// test-coverage specialist can only report that it found nothing.
	//
	// This check exists because the wording check below is not enough on its
	// own. queenPhaseIsDocumentationOnly requires one of eight documentation
	// words to be present, and a real phase -- "copy 110 markdown files into a
	// new folder tree" in an Obsidian vault -- contains none of them, so it was
	// classified as code work and a Probe was made mandatory. Asking what is in
	// the repository does not depend on the phase happening to use the right
	// vocabulary.
	if !workspaceContainsProgramCode() {
		return false
	}
	return !queenPhaseIsDocumentationOnly(phase)
}

// programCodeFilePatterns is deliberately a list of source extensions rather
// than "anything that is not markdown": a repository of notes, prose or data
// should read as having no code, and adding a language here is a one-line
// change with an obvious meaning.
var programCodeFilePatterns = []string{
	"*.go", "*.ts", "*.tsx", "*.js", "*.jsx", "*.py", "*.rb", "*.java",
	"*.rs", "*.c", "*.h", "*.cc", "*.cpp", "*.cs", "*.php", "*.swift",
	"*.kt", "*.scala", "*.ex", "*.exs", "*.dart", "*.m", "*.sh",
}

func workspaceContainsProgramCode() bool {
	root := strings.TrimSpace(skillWorkspaceRoot())
	if root == "" {
		// Unknown workspace: assume code. Failing open here costs one worker;
		// failing closed would silently remove test coverage from real code.
		return true
	}
	for _, pattern := range programCodeFilePatterns {
		if repoMatchesFilePattern(root, pattern) {
			return true
		}
	}
	return false
}

// containsDocumentationWord matches a documentation noun as a whole word,
// allowing only a plural "s". containsKeyword's open-ended suffix tolerance is
// right for concepts ("test" should match "tests", "structure" should match
// "structured") but wrong for these: it read "manually inspect any failures" as
// the word "manual", classified a production safety-verification phase as
// documentation-only, and dropped its Probe. A verb that merely starts with a
// document's name is not a document.
func containsDocumentationWord(text, keyword string) bool {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return false
	}
	for offset := 0; offset < len(text); {
		idx := strings.Index(text[offset:], keyword)
		if idx < 0 {
			return false
		}
		start := offset + idx
		end := start + len(keyword)
		if end < len(text) && text[end] == 's' {
			end++
		}
		startOK := start == 0 || !isWordByte(text[start-1])
		endOK := end >= len(text) || !isWordByte(text[end])
		if startOK && endOK {
			return true
		}
		offset = start + 1
	}
	return false
}

// queenPhaseIsDocumentationOnly is deliberately stricter than
// queenPhaseLooksDocumentationOrMaintenance, which exists for budget sizing and
// matches loose substrings ("standard" hits "standardise", "doc" hits
// "docker"). Dropping a caste needs a higher bar than trimming a budget: this
// requires an anchored documentation word AND the absence of any
// implementation word, so "Document the API and add the /health endpoint"
// still counts as producing code.
func queenPhaseIsDocumentationOnly(phase colony.Phase) bool {
	text := collectPhaseText(phase)

	documentation := false
	for _, keyword := range []string{
		"documentation", "readme", "changelog", "guide", "manual", "doc", "docs", "tutorial",
	} {
		if containsDocumentationWord(text, keyword) {
			documentation = true
			break
		}
	}
	if !documentation {
		return false
	}

	for _, keyword := range []string{
		"implement", "build", "refactor", "migrate", "endpoint", "api",
		"function", "component", "module", "schema", "test", "fix", "bug",
	} {
		if containsKeyword(text, keyword) {
			return false
		}
	}
	return true
}

func queenPhaseLooksDocumentationOrMaintenance(phase colony.Phase) bool {
	text := collectPhaseText(phase)
	for _, keyword := range []string{
		"doc", "documentation", "readme", "guide", "manual", "changelog",
		"maintenance", "cleanup", "standard", "standards",
	} {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		set[value] = true
	}
	return set
}

func appendQueenBudgetRationale(rationale string, budget queenSpawnBudget) string {
	suffix := fmt.Sprintf("selected within Queen spawn budget %d (%s)", budget.MaxWorkers, budget.Reason)
	if strings.TrimSpace(rationale) == "" {
		return suffix
	}
	return rationale + "; " + suffix
}

func appendQueenPrunedRationale(rationale string, budget queenSpawnBudget) string {
	suffix := fmt.Sprintf("not spawned; outside Queen spawn budget %d (%s)", budget.MaxWorkers, budget.Reason)
	if strings.TrimSpace(rationale) == "" {
		return suffix
	}
	return rationale + "; " + suffix
}
