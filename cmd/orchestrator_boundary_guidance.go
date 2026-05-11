package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

const orchestratorBoundaryDiscussNext = "aether discuss"

type orchestratorBoundaryGuidance struct {
	Active            bool                                  `json:"active"`
	Workflow          string                                `json:"workflow"`
	ColonyMode        string                                `json:"colony_mode"`
	PendingCount      int                                   `json:"pending_count"`
	Next              string                                `json:"next"`
	AfterDiscussNext  string                                `json:"after_discuss_next,omitempty"`
	Summary           string                                `json:"summary,omitempty"`
	QuestionIDs       []string                              `json:"question_ids,omitempty"`
	QuestionSources   []string                              `json:"question_sources,omitempty"`
	QuestionSummaries []orchestratorBoundaryQuestionSummary `json:"question_summaries,omitempty"`
}

type orchestratorBoundaryQuestionSummary struct {
	ID             string   `json:"id"`
	Source         string   `json:"source"`
	Question       string   `json:"question"`
	Options        []string `json:"options,omitempty"`
	HardConstraint bool     `json:"hard_constraint,omitempty"`
}

func addOrchestratorBoundaryGuidance(result map[string]interface{}, workflow string, state colony.ColonyState, afterDiscussNext string, manifestQuestions []discussQuestion) (orchestratorBoundaryGuidance, bool) {
	if result == nil || state.EffectiveColonyMode() != colony.ColonyModeOrchestrator {
		return orchestratorBoundaryGuidance{}, false
	}
	workflow = normalizeOrchestratorBoundarySourcePart(workflow, orchestratorBoundaryFallbackWorkflow)
	afterDiscussNext = strings.TrimSpace(afterDiscussNext)
	if afterDiscussNext == "" {
		afterDiscussNext = strings.TrimSpace(stringValue(result["next"]))
	}

	guidance := orchestratorBoundaryGuidance{
		Active:       false,
		Workflow:     workflow,
		ColonyMode:   string(colony.ColonyModeOrchestrator),
		Next:         afterDiscussNext,
		PendingCount: 0,
		Summary:      fmt.Sprintf("No unresolved Orchestrator boundary questions for %s.", workflow),
	}

	pending := unresolvedOrchestratorBoundaryDecisions(workflow, manifestQuestions)
	if len(pending) > 0 {
		guidance.Active = true
		guidance.PendingCount = len(pending)
		guidance.Next = orchestratorBoundaryDiscussNext
		guidance.AfterDiscussNext = afterDiscussNext
		guidance.Summary = fmt.Sprintf("%d unresolved Orchestrator boundary question(s) should be answered with `aether discuss` before `%s`.", len(pending), afterDiscussNext)
		guidance.QuestionSummaries = make([]orchestratorBoundaryQuestionSummary, 0, len(pending))
		guidance.QuestionIDs = make([]string, 0, len(pending))
		guidance.QuestionSources = make([]string, 0, len(pending))
		for _, decision := range pending {
			question, options := parseClarificationDescription(decision.Description)
			guidance.QuestionIDs = append(guidance.QuestionIDs, decision.ID)
			guidance.QuestionSources = append(guidance.QuestionSources, decision.Source)
			guidance.QuestionSummaries = append(guidance.QuestionSummaries, orchestratorBoundaryQuestionSummary{
				ID:             decision.ID,
				Source:         decision.Source,
				Question:       question,
				Options:        options,
				HardConstraint: clarificationIsHardConstraint(decision),
			})
		}
		result["next"] = orchestratorBoundaryDiscussNext
		if afterDiscussNext != "" {
			result["after_discuss_next"] = afterDiscussNext
		}
	}

	result["orchestrator_boundary_guidance"] = guidance
	return guidance, true
}

func unresolvedOrchestratorBoundaryGuidanceError(workflow string, state colony.ColonyState, afterDiscussNext string, manifestQuestions []discussQuestion) error {
	if state.EffectiveColonyMode() != colony.ColonyModeOrchestrator {
		return nil
	}
	workflow = normalizeOrchestratorBoundarySourcePart(workflow, orchestratorBoundaryFallbackWorkflow)
	afterDiscussNext = strings.TrimSpace(afterDiscussNext)
	if afterDiscussNext == "" {
		afterDiscussNext = fmt.Sprintf("aether %s", workflow)
	}
	pending := unresolvedOrchestratorBoundaryDecisions(workflow, manifestQuestions)
	if len(pending) == 0 {
		return nil
	}
	return fmt.Errorf("%d unresolved Orchestrator boundary question(s) should be answered with `aether discuss` before `%s`", len(pending), afterDiscussNext)
}

func unresolvedOrchestratorBoundaryDecisions(workflow string, manifestQuestions []discussQuestion) []PendingDecision {
	pending := loadPendingDecisionFile()
	ids := map[string]bool{}
	sources := map[string]bool{}
	for _, question := range manifestQuestions {
		if id := strings.TrimSpace(question.ID); id != "" {
			ids[id] = true
		}
		if source := strings.TrimSpace(question.Source); source != "" {
			sources[source] = true
		}
	}

	workflow = normalizeOrchestratorBoundarySourcePart(workflow, orchestratorBoundaryFallbackWorkflow)
	sourcePrefix := fmt.Sprintf("%s:%s:", orchestratorBoundarySourcePrefix, workflow)
	allowAnyWorkflowBoundary := len(ids) == 0 && len(sources) == 0
	decisions := make([]PendingDecision, 0)
	seen := map[string]bool{}
	for _, decision := range pending.Decisions {
		if decision.Type != clarificationDecisionType || decision.Resolved {
			continue
		}
		source := strings.TrimSpace(decision.Source)
		id := strings.TrimSpace(decision.ID)
		if !allowAnyWorkflowBoundary && !ids[id] && !sources[source] {
			continue
		}
		if allowAnyWorkflowBoundary && !strings.HasPrefix(source, sourcePrefix) {
			continue
		}
		key := source
		if key == "" {
			key = id
		}
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		decisions = append(decisions, decision)
	}

	sort.SliceStable(decisions, func(i, j int) bool {
		if decisions[i].Source == decisions[j].Source {
			return decisions[i].ID < decisions[j].ID
		}
		return decisions[i].Source < decisions[j].Source
	})
	return decisions
}

func validateFinalizerManifestRoot(manifestName, manifestRoot, root string) error {
	if strings.TrimSpace(manifestRoot) == "" || sameCleanPath(manifestRoot, root) {
		return nil
	}
	return fmt.Errorf("%s root does not match current workspace (manifest=%s current=%s)", manifestName, manifestRoot, root)
}

func validateFinalizerManifestColonyMode(manifestName, manifestMode string, state colony.ColonyState) error {
	manifestMode = strings.ToLower(strings.TrimSpace(manifestMode))
	if manifestMode == "" {
		return nil
	}
	mode := colony.ColonyMode(manifestMode)
	if !mode.Valid() {
		return fmt.Errorf("%s colony_mode %q is invalid", manifestName, manifestMode)
	}
	if mode != state.EffectiveColonyMode() {
		return fmt.Errorf("%s colony_mode %q does not match active colony mode %q", manifestName, mode, state.EffectiveColonyMode())
	}
	return nil
}
