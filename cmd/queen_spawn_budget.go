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
	required := []string{"probe", "watcher"}
	if effectiveQueenPhaseMode(phase) != colony.PhaseModeDiscovery {
		required = append(required, "builder")
	}

	riskLevel := phaseRiskLevel(phase)
	if riskLevel == "high" || queenBuildSafetyReviewRequired(phase) {
		required = append(required, "auditor", "gatekeeper")
	}

	sort.Strings(required)
	return required
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

func queenMaxWorkersForBudget(phase colony.Phase, flowType string, state colony.ColonyState, riskLevel string) (int, string) {
	mode := effectiveQueenPhaseMode(phase)

	switch flowType {
	case "build":
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
	return colony.InferPhaseMode(phase.Name, phase.Description)
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
