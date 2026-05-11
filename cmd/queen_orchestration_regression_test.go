package cmd

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestQueenAdaptiveCasteContractAcrossFlowHelpers(t *testing.T) {
	root := t.TempDir()
	taskID := "1.1"
	authPhase := colony.Phase{
		ID:          2,
		Name:        "Auth token rotation",
		Description: "Secure auth token rotation and session permissions",
		Mode:        colony.PhaseModeProduction,
		Tasks: []colony.Task{{
			ID:     &taskID,
			Goal:   "Implement secure token refresh and permission checks",
			Status: colony.TaskPending,
		}},
	}
	authState := colony.ColonyState{
		ColonyDepth:       "full",
		VerificationDepth: string(colony.VerificationDepthStandard),
	}

	buildDispatches := plannedBuildDispatchesForSelectionWithState(authPhase, authState, nil, colony.VerificationDepthStandard)
	regressionRequireBuildCaste(t, buildDispatches, "gatekeeper")

	continueSpecs := queenContinueReviewSpecs(authPhase, colony.VerificationDepthStandard)
	regressionRequireContinueSpec(t, continueSpecs, "gatekeeper")

	planningDispatches := plannedPlanningWorkersForGoal(root, "Plan secure auth token rotation and permission checks")
	regressionRequirePlanningCaste(t, planningDispatches, "gatekeeper")

	swarmCastes := queenSwarmSelectedCastes("Auth token regression in session permissions")
	if !swarmCastes["gatekeeper"] {
		t.Fatalf("auth swarm selection missing Queen-selected gatekeeper: %+v", swarmCastes)
	}
	swarmPlans := allSwarmPlans(root, "Auth token regression in session permissions")
	regressionRequireSwarmPlan(t, swarmPlans, "gatekeeper")
	regressionRequireSwarmAgent(t, swarmPlans, "gatekeeper", "aether-gatekeeper")

	surveyorSpecs := queenSurveyorSpecs()
	for _, caste := range []string{"surveyor-provisions", "surveyor-nest", "surveyor-disciplines", "surveyor-pathogens"} {
		regressionRequireSurveyorSpec(t, surveyorSpecs, caste)
	}
	regressionRejectSurveyorSpec(t, surveyorSpecs, "gatekeeper")

	routinePhase := colony.Phase{
		ID:          3,
		Name:        "Settings UI panel",
		Description: "Build a settings panel for user preferences",
		Mode:        colony.PhaseModePrototype,
		Tasks: []colony.Task{{
			ID:     &taskID,
			Goal:   "Implement SettingsPanel component with form controls",
			Status: colony.TaskPending,
		}},
	}
	routineState := colony.ColonyState{
		ColonyDepth:       "full",
		VerificationDepth: string(colony.VerificationDepthLight),
	}
	routineDispatches := plannedBuildDispatchesForSelectionWithState(routinePhase, routineState, nil, colony.VerificationDepthLight)
	if got, want := regressionBuildCastes(routineDispatches), []string{"builder", "probe", "watcher"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("routine UI build dispatches = %v, want lean Queen plan %v", got, want)
	}
	for _, caste := range []string{"gatekeeper", "oracle", "chaos", "measurer"} {
		regressionRejectBuildCaste(t, routineDispatches, caste)
	}

	discoveryPhase := colony.Phase{
		ID:     1,
		Name:   "Discovery spike",
		Mode:   colony.PhaseModeDiscovery,
		Status: colony.PhaseCompleted,
	}
	sealState := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthLight)}
	if specs := queenSealReviewSpecs(sealState, discoveryPhase, colony.VerificationDepthLight); len(specs) != 0 {
		t.Fatalf("light discovery seal specs = %+v, want no final review gates", specs)
	}
}

func regressionBuildCastes(dispatches []codexBuildDispatch) []string {
	castes := make([]string, 0, len(dispatches))
	for _, dispatch := range dispatches {
		castes = append(castes, dispatch.Caste)
	}
	return castes
}

func regressionRequireBuildCaste(t *testing.T, dispatches []codexBuildDispatch, caste string) {
	t.Helper()
	for _, dispatch := range dispatches {
		if dispatch.Caste == caste {
			return
		}
	}
	t.Fatalf("build dispatches missing %s: %v", caste, regressionBuildCastes(dispatches))
}

func regressionRejectBuildCaste(t *testing.T, dispatches []codexBuildDispatch, caste string) {
	t.Helper()
	for _, dispatch := range dispatches {
		if dispatch.Caste == caste {
			t.Fatalf("build dispatches should not include %s: %v", caste, regressionBuildCastes(dispatches))
		}
	}
}

func regressionRequireContinueSpec(t *testing.T, specs []codexContinueReviewSpec, caste string) {
	t.Helper()
	for _, spec := range specs {
		if spec.Caste == caste {
			return
		}
	}
	t.Fatalf("continue review specs missing %s: %+v", caste, specs)
}

func regressionRequirePlanningCaste(t *testing.T, dispatches []codexPlanningDispatch, caste string) {
	t.Helper()
	for _, dispatch := range dispatches {
		if dispatch.Caste == caste {
			return
		}
	}
	t.Fatalf("planning dispatches missing %s: %+v", caste, dispatches)
}

func regressionRequireSwarmPlan(t *testing.T, plans []swarmWorkerPlan, caste string) {
	t.Helper()
	for _, plan := range plans {
		if plan.Caste == caste {
			return
		}
	}
	t.Fatalf("swarm plans missing %s: %+v", caste, plans)
}

func regressionRequireSwarmAgent(t *testing.T, plans []swarmWorkerPlan, caste, agentName string) {
	t.Helper()
	for _, plan := range plans {
		if plan.Caste == caste && plan.AgentName == agentName {
			return
		}
	}
	t.Fatalf("swarm plans missing %s agent %s: %+v", caste, agentName, plans)
}

func regressionRequireSurveyorSpec(t *testing.T, specs []surveyorSpec, caste string) {
	t.Helper()
	for _, spec := range specs {
		if spec.Caste == caste {
			return
		}
	}
	t.Fatalf("surveyor specs missing %s: %+v", caste, specs)
}

func regressionRejectSurveyorSpec(t *testing.T, specs []surveyorSpec, caste string) {
	t.Helper()
	for _, spec := range specs {
		if spec.Caste == caste {
			t.Fatalf("surveyor specs should not include %s: %+v", caste, specs)
		}
	}
}
