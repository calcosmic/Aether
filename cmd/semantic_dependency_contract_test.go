package cmd

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestSemanticDependenciesReachCoherentDispatch(t *testing.T) {
	first, _ := coherentJobTestTask("1.1", "builder")
	first.SemanticID = "preserve-restoration-baseline"
	second, _ := coherentJobTestTask("1.2", "builder", first.SemanticID)
	second.SemanticID = "rehearse-wallpaper-writing-and-daily"
	phase := coherentJobTestPhase(1, first, second)
	before, _ := json.Marshal(phase)
	state := colony.ColonyState{Plan: colony.Plan{Phases: []colony.Phase{phase}}}
	dispatches, _, err := plannedBuildDispatchesWithJobProposals(phase, state, nil, colony.VerificationDepthLight, nil, "", nil, nil)
	if err != nil {
		t.Fatalf("semantic target exists but dispatch refused it: %v", err)
	}
	if len(dispatches) == 0 {
		t.Fatal("no dispatch assignments")
	}
	after, _ := json.Marshal(phase)
	if string(before) != string(after) {
		t.Fatal("dispatch rewrote the accepted dependency")
	}
	plans := codexBuildTaskPlans(phase)
	if plans[0].Wave >= plans[1].Wave {
		t.Fatalf("semantic ordering disappeared from task manifest: %+v", plans)
	}
}

func TestCrossPhaseDependenciesReachCoherentDispatch(t *testing.T) {
	for _, reference := range []string{"1.1", "preserve-restoration-baseline"} {
		t.Run(reference, func(t *testing.T) {
			first, _ := coherentJobTestTask("1.1", "builder")
			first.SemanticID, first.Status = "preserve-restoration-baseline", colony.TaskCompleted
			prior := coherentJobTestPhase(1, first)
			prior.Status = colony.PhaseCompleted
			second, _ := coherentJobTestTask("2.1", "builder", reference)
			phase := coherentJobTestPhase(2, second)
			state := colony.ColonyState{Plan: colony.Plan{Phases: []colony.Phase{prior, phase}}}
			if dispatches, _, err := plannedBuildDispatchesWithJobProposals(phase, state, nil, colony.VerificationDepthLight, nil, "", nil, nil); err != nil || len(dispatches) == 0 {
				t.Fatalf("completed cross-phase target rejected: %v", err)
			}
			state.Plan.Phases[0].Tasks[0].Status = colony.TaskPending
			state.Plan.Phases[0].Status = colony.PhasePending
			if _, _, err := plannedBuildDispatchesWithJobProposals(phase, state, nil, colony.VerificationDepthLight, nil, "", nil, nil); err == nil {
				t.Fatal("unfinished cross-phase dependency was ignored")
			}
		})
	}
}

func TestRepairArtifactRouteBypassesPreset(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	goal := "Repair the saved staging dependency references"
	state := codexPlanSpecificationFixture(t, colony.ColonyState{Version: "3.0", Goal: &goal, State: colony.StateREADY}, colony.SpecStatusApproved)
	createTestColonyState(t, dataDir, state)
	writeCodexPlanSpecificationProjection(t, root, state)
	artifact := codexWorkerPlanArtifact{Phases: []codexWorkerPlanPhase{{Name: "Legacy dependency repair", Tasks: []codexWorkerPlanTask{{Goal: "Baseline"}, {Goal: "Daily", DependsOn: []string{"P1-T1"}}}}}}
	if err := store.SaveJSON("planning/phase-plan.json", artifact); err != nil {
		t.Fatal(err)
	}
	result, err := runCodexPlanWithOptions(root, codexPlanOptions{RepairArtifact: true})
	if err != nil || result["validated"] != true || result["repaired"] != true {
		t.Fatalf("repair flag never reached repair: result=%+v err=%v", result, err)
	}
	var saved codexWorkerPlanArtifact
	if err := store.LoadJSON("planning/phase-plan.json", &saved); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved.Phases[0].Tasks[1].DependsOn, []string{"1.1"}) {
		t.Fatal("reported repair did not change the staging artifact")
	}
}

func TestAcceptedSemanticDependenciesPreserveApprovalAtBuild(t *testing.T) {
	saveGlobals(t)
	root, manifest, result := planningRouteStageTestFixture(t)
	planningRouteStageSetPolicy(t, root, manifest.RunID, 70, 6)
	planningParityAddHousekeepingTask(&result)
	result.Proposal.Phases[0].Tasks[1].DependsOn = []string{result.Proposal.Phases[0].Tasks[0].SemanticID}
	candidate, err := planningParityDraft(t, root, manifest, result)
	if err != nil {
		t.Fatal(err)
	}
	if err := planningParityAccept(root, candidate); err != nil {
		t.Fatal(err)
	}
	planningParityAssertBuildable(t, root)
	state := mustReadSpecificationTestState(t, root)
	before, _ := json.Marshal(state.Plan)
	if state.Plan.Phases[0].Tasks[1].DependsOn[0] != result.Proposal.Phases[0].Tasks[0].SemanticID {
		t.Fatal("fixture no longer represents an already-accepted semantic dependency")
	}
	_, _, _, dispatches, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{LightFlag: true})
	if err != nil {
		t.Fatalf("accepted dependency rejected by actual build preparation: %v", err)
	}
	if len(dispatches) == 0 {
		t.Fatal("accepted plan produced no dispatch")
	}
	after, _ := json.Marshal(mustReadSpecificationTestState(t, root).Plan)
	if string(before) != string(after) {
		t.Fatal("build changed accepted revision, candidate or approval hashes")
	}
}
