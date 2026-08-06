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
	sort.Strings(required)
	return required
}

func queenBuildSafetyRequiredCaste(caste, flowType string, phase colony.Phase) bool {
	if normalizeQueenFlowType(flowType) != "build" {
		return false
	}
	return stringSet(queenBuildSafetyRequiredCastes(phase))[caste]
}

func queenBuildSafetyRequiredCastes(phase colony.Phase) []string {
	// Watcher is the only unconditional member: it is the check, and a build
	// with nothing verifying it is a build that reports success by assertion.
	required := []string{"watcher"}
	if effectiveQueenPhaseMode(phase) != colony.PhaseModeDiscovery {
		required = append(required, "builder")
	}
	// Probe used to sit alongside Watcher as unconditionally required, and
	// required castes bypass the worker budget entirely — so every build got a
	// test-coverage specialist no matter what the phase was. A documentation
	// phase, a research phase, and a `--light` build of a trivial change all
	// spawned a Probe with no code for it to cover. Combined with the standard
	// continue path, which also required Probe, a single phase paid for two.
	if queenPhaseProducesTestableCode(phase) {
		required = append(required, "probe")
	}

	riskLevel := phaseRiskLevel(phase)
	if riskLevel == "high" || queenBuildSafetyReviewRequired(phase) {
		// Auditor is the general quality gate, so production work earns one.
		required = append(required, "auditor")
	}
	// Gatekeeper is a security specialist, not a general reviewer. It used to
	// ride in beside Auditor on the same condition, which meant every
	// production-mode phase got one — and mode is inferred from wording, so
	// most real phases are production. "Add a dark mode toggle" summoned a
	// security auditor that could only report it had found no security
	// surface. It now needs an actual security signal: high risk, or security
	// or release wording.
	if riskLevel == "high" || queenPhaseHasSecuritySignal(phase) {
		required = append(required, "gatekeeper")
	}

	sort.Strings(required)
	return required
}

// queenPhaseHasSecuritySignal reports whether a phase names something a
// security specialist could actually review. Unlike
// queenBuildSafetyReviewRequired it does not treat production mode alone as a
// signal: production means "this is real work", not "this touches auth".
//
// The release-gate terms are here on purpose alongside the security surfaces.
// A final review or sign-off is the last point at which a security problem can
// be caught before it ships, which is exactly when the specialist is worth
// paying for. An earlier version of this list carried the security surfaces but
// dropped the gate terms, and a "Final signoff before handoff" phase silently
// lost its Gatekeeper — caught by TestQueenOrchestratePreservesSafetyCastes.
func queenPhaseHasSecuritySignal(phase colony.Phase) bool {
	return matchesAnyKeyword(collectPhaseText(phase), []string{
		// Security surfaces
		"security",
		"auth",
		"crypto",
		"secret",
		"token",
		"permission",
		"credential",
		"password",
		"compliance",
		"vulnerab",
		// Release gates — the last chance to catch something before it ships
		"release",
		"final review",
		"final-review",
		"final signoff",
		"signoff",
		"sign-off",
		"sign off",
	})
}

func queenBuildSafetyReviewRequired(phase colony.Phase) bool {
	if effectiveQueenPhaseMode(phase) == colony.PhaseModeProduction {
		return true
	}
	return matchesAnyKeyword(collectPhaseText(phase), []string{
		"security",
		"release",
		"production",
		"final review",
		"final-review",
		"final signoff",
		"signoff",
		"sign-off",
		"sign off",
	})
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
	return !queenPhaseIsDocumentationOnly(phase)
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
