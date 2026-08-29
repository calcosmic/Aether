package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

const orchestratorBoundaryQuestionLimit = 1

type orchestratorBoundaryQuestionsResult struct {
	Questions []discussQuestion
	Created   int
	Existing  int
}

func emptyOrchestratorBoundaryQuestionsResult() orchestratorBoundaryQuestionsResult {
	return orchestratorBoundaryQuestionsResult{Questions: []discussQuestion{}}
}

func materializeOrchestratorBoundaryQuestions(workflow string, state colony.ColonyState, phase colony.Phase, candidates []discussQuestion) (orchestratorBoundaryQuestionsResult, error) {
	if state.EffectiveColonyMode() != colony.ColonyModeOrchestrator {
		return emptyOrchestratorBoundaryQuestionsResult(), nil
	}
	questions, created, existing, err := materializeOrchestratorBoundaryClarifications(workflow, phase.ID, loadPendingDecisionFile(), candidates, orchestratorBoundaryQuestionLimit, false)
	if err != nil {
		return emptyOrchestratorBoundaryQuestionsResult(), err
	}
	if questions == nil {
		questions = []discussQuestion{}
	}
	return orchestratorBoundaryQuestionsResult{Questions: questions, Created: created, Existing: existing}, nil
}

// checkOrchestratorBoundaryQuestions is materializeOrchestratorBoundaryQuestions's
// read-only sibling: it reports whether an unresolved boundary question
// exists (already pending, or one the same candidates would raise) without
// ever creating a new pending decision. Used by the direct build lane
// (`aether build <phase>`, no `--plan-only`), which has already dispatched
// real work by the time its advisory heads-up runs -- persisting a brand-new
// question at that point would be too late to act on and would duplicate
// whatever the plan-only lane already recorded for the same phase. It still
// reads state so both lanes agree on BoundaryQuestionCount for the same
// phase (CLAUDE.md: "if the direct path and the chat path can disagree,
// both get proved").
func checkOrchestratorBoundaryQuestions(workflow string, state colony.ColonyState, phase colony.Phase, candidates []discussQuestion) (orchestratorBoundaryQuestionsResult, error) {
	if state.EffectiveColonyMode() != colony.ColonyModeOrchestrator {
		return emptyOrchestratorBoundaryQuestionsResult(), nil
	}
	questions, created, existing, err := materializeOrchestratorBoundaryClarifications(workflow, phase.ID, loadPendingDecisionFile(), candidates, orchestratorBoundaryQuestionLimit, true)
	if err != nil {
		return emptyOrchestratorBoundaryQuestionsResult(), err
	}
	if questions == nil {
		questions = []discussQuestion{}
	}
	return orchestratorBoundaryQuestionsResult{Questions: questions, Created: created, Existing: existing}, nil
}

func addBoundaryQuestionResultFields(result map[string]interface{}, boundary orchestratorBoundaryQuestionsResult) {
	result["boundary_questions"] = boundary.Questions
	result["boundary_question_count"] = len(boundary.Questions)
	result["boundary_questions_created"] = boundary.Created
	result["boundary_questions_existing"] = boundary.Existing
}

func planBoundaryQuestionCandidates(state colony.ColonyState, granularity colony.PlanGranularity, planDepth, planningDepth, verificationDepth string) []discussQuestion {
	goal := strings.TrimSpace(derefGoal(state.Goal))
	if goal == "" {
		goal = "the colony goal"
	}
	reasoning := fmt.Sprintf("Planning is about to choose boundaries for %s with %s granularity, %s planning depth, and %s verification depth.", goal, granularity, emptyFallback(planDepth, "default"), emptyFallback(verificationDepth, "default"))
	if strings.TrimSpace(planningDepth) != "" {
		reasoning = fmt.Sprintf("%s The wrapper should not infer the first planning tradeoff.", reasoning)
	}
	return []discussQuestion{{
		Category:  "planning-scope",
		Question:  "What should the first generated plan optimize for?",
		Options:   []string{"smallest useful slice", "balanced milestone plan", "surface risky dependencies first"},
		Reasoning: reasoning,
	}}
}

func buildBoundaryQuestionCandidates(phase colony.Phase, selectedTaskIDs []string) []discussQuestion {
	scopeOption := "phase tasks only"
	if len(selectedTaskIDs) > 0 {
		scopeOption = "selected tasks only"
	}
	phaseLabel := boundaryPhaseLabel(phase)
	return []discussQuestion{{
		Category:       "build-scope",
		Question:       fmt.Sprintf("What boundary should builders protect for %s?", phaseLabel),
		Options:        []string{scopeOption, "include direct prerequisites", "pause before expanding scope"},
		Reasoning:      fmt.Sprintf("The build manifest has %d task(s) and %d selected task filter(s); workers need a compact rule for scope pressure.", len(phase.Tasks), len(selectedTaskIDs)),
		HardConstraint: true,
	}}
}

func continueBoundaryQuestionCandidates(phase colony.Phase, verification codexContinueVerificationReport, assessment codexContinueAssessment) []discussQuestion {
	if !assessment.Passed || len(assessment.BlockingIssues) > 0 || !verification.Passed {
		return []discussQuestion{{
			Category:       "recovery",
			Question:       fmt.Sprintf("How should %s handle blocked continue evidence?", boundaryPhaseLabel(phase)),
			Options:        []string{"redispatch recovery tasks", "reconcile with evidence and reverify", "pause for manual review"},
			Reasoning:      fmt.Sprintf("Continue assessment says %q with %d blocking issue(s).", emptyFallback(assessment.Summary, "blocked"), len(assessment.BlockingIssues)),
			HardConstraint: true,
		}}
	}
	return []discussQuestion{{
		Category:  "advance",
		Question:  fmt.Sprintf("What should continue preserve before advancing past %s?", boundaryPhaseLabel(phase)),
		Options:   []string{"advance after planned reviews", "run heavier review first", "pause before next phase"},
		Reasoning: fmt.Sprintf("Verification passed for %d task assessment(s); the wrapper can now lock the advancement posture.", len(assessment.Tasks)),
	}}
}

func sealBoundaryQuestionCandidates(state colony.ColonyState, phase colony.Phase, reviewDepth colony.VerificationDepth) []discussQuestion {
	goal := strings.TrimSpace(derefGoal(state.Goal))
	if goal == "" {
		goal = "the colony"
	}
	return []discussQuestion{{
		Category:       "release-boundary",
		Question:       "What boundary should final seal reviewers enforce?",
		Options:        []string{"block on security or quality issues", "allow nonblocking follow-ups", "pause for manual release review"},
		Reasoning:      fmt.Sprintf("Seal review is preparing %s from %s with %s review depth.", goal, boundaryPhaseLabel(phase), reviewDepth),
		HardConstraint: true,
	}}
}

func boundaryPhaseLabel(phase colony.Phase) string {
	name := strings.TrimSpace(phase.Name)
	if name == "" {
		return fmt.Sprintf("Phase %d", phase.ID)
	}
	return fmt.Sprintf("Phase %d (%s)", phase.ID, name)
}
