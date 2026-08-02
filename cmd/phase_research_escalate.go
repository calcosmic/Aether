package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// phaseResearchEscalationDispatch builds the Oracle dispatch that replaces a
// stalled Scout for one phase. It mirrors the dispatch literal built by
// plannedPhaseResearchDispatches, changing only the caste, agent name, name,
// and task ID -- the stage, wave, and output stay identical so the escalated
// Oracle overwrites the same phase-N-research.md artifact in place (D-07).
//
// The write promise carried by this dispatch is scoped by the "oracle" case
// added to behavioralRestrictionsForCaste (pkg/codex/permission_profile.go)
// in the same change, so escalating a Scout to an Oracle never broadens
// access beyond the Scout's own phase-research write scope plus Oracle's own
// artifact directory.
func phaseResearchEscalationDispatch(root, goal string, candidate phaseResearchCandidate, survey codexSurveyContext, stalledAt, target int) codexPlanningDispatch {
	fileName := fmt.Sprintf("phase-%d-research.md", candidate.ID)
	preamble := fmt.Sprintf(
		"## Escalation Notice\n\nThe Scout assigned to this phase's research stalled at %d%% confidence "+
			"against a %d%% target for Phase %d. This is a caste escalation from Scout to Oracle -- "+
			"close the specific gaps that remain rather than restating what the Scout already found.\n\n",
		stalledAt, target, candidate.ID,
	)
	dispatch := codexPlanningDispatch{
		Stage:     phaseResearchStage,
		Wave:      1,
		Caste:     "oracle",
		AgentName: "aether-oracle",
		Name:      deterministicAntName("oracle", fmt.Sprintf("%s|plan-research-escalate|phase-%d", root, candidate.ID)),
		Task:      fmt.Sprintf("Escalated Oracle research for phase %d: %s", candidate.ID, firstNonEmpty(candidate.Name, "unnamed phase")),
		TaskID:    fmt.Sprintf("plan-research-escalate-phase-%d", candidate.ID),
		Outputs:   []string{fileName},
		Status:    "planned",
		Brief:     preamble + renderPhaseResearchBrief(goal, candidate, survey),
	}
	dispatches := []codexPlanningDispatch{dispatch}
	attachPlanningDispatchSkillAssignments(dispatches)
	return dispatches[0]
}

// findPhaseResearchCandidate looks up a single candidate by phase ID out of
// the current plan's candidate set. Returns ok=false for any phase ID not
// present in the current plan -- the caller turns that into a clean
// outputError, never a panic.
func findPhaseResearchCandidate(candidates []phaseResearchCandidate, phaseID int) (phaseResearchCandidate, bool) {
	for _, candidate := range candidates {
		if candidate.ID == phaseID {
			return candidate, true
		}
	}
	return phaseResearchCandidate{}, false
}

// planResearchEscalateCmd is invoked by the TS host's confidence loop
// (runResearchConfidenceLoop) when a deep or exhaustive phase's research
// stalls below target. It builds and prints the Oracle dispatch; the TS host
// dispatches it exactly once and lets Oracle's own RALF loop drive the rest.
var planResearchEscalateCmd = &cobra.Command{
	Use:   "plan-research-escalate",
	Short: "Build an Oracle dispatch escalating a stalled phase research Scout",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		phase := mustGetInt(cmd, "phase")
		confidence, _ := cmd.Flags().GetInt("confidence")
		target, _ := cmd.Flags().GetInt("target")

		root := resolveAetherRoot()
		state, err := loadActiveColonyState()
		if err != nil {
			outputError(2, fmt.Sprintf("failed to load colony state: %v", err), nil)
			return nil
		}
		survey, err := loadCodexSurveyContext(root)
		if err != nil {
			outputError(2, fmt.Sprintf("failed to load survey context: %v", err), nil)
			return nil
		}

		candidates := phaseResearchCandidates(state, codexPlanIterationState{})
		candidate, ok := findPhaseResearchCandidate(candidates, phase)
		if !ok {
			outputError(1, fmt.Sprintf("phase %d not found in the current plan", phase), nil)
			return nil
		}

		goal := ""
		if state.Goal != nil {
			goal = *state.Goal
		}
		dispatch := phaseResearchEscalationDispatch(root, goal, candidate, survey, confidence, target)
		outputOK(map[string]interface{}{
			"dispatch": dispatch,
		})
		return nil
	},
}

func init() {
	planResearchEscalateCmd.Flags().Int("phase", 0, "Phase ID whose research stalled")
	_ = planResearchEscalateCmd.MarkFlagRequired("phase")
	planResearchEscalateCmd.Flags().Int("confidence", 0, "Confidence percentage the phase stalled at")
	planResearchEscalateCmd.Flags().Int("target", 0, "Confidence target percentage for the phase's depth")
	rootCmd.AddCommand(planResearchEscalateCmd)
}
