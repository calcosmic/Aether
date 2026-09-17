package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
	assertDependencyDispatchOrder(t, dispatches, "1.1", "1.2")
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
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")
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
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })
	var output bytes.Buffer
	stdout = &output
	if err := planCmd.ParseFlags([]string{"--repair-artifact"}); err != nil {
		t.Fatal(err)
	}
	if err := planCmd.RunE(planCmd, nil); err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Result map[string]interface{} `json:"result"`
	}
	if err := json.Unmarshal(output.Bytes(), &envelope); err != nil {
		t.Fatalf("invalid repair output: %v %s", err, output.String())
	}
	result := envelope.Result
	if result["validated"] != true || result["repaired"] != true || result["repair_scope"] != "legacy_phase_plan_artifact" {
		t.Fatalf("repair flag never reached repair: result=%+v", result)
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
	snapshot := planCandidateTestSnapshot(t, root)
	repair, err := runCodexPlanWithOptions(root, codexPlanOptions{RepairArtifact: true})
	if err != nil || repair["status"] != "accepted_plan_validated" || repair["repaired"] != false || repair["state_effect"] != "unchanged" || repair["revision_id"] != state.Plan.ActiveRevisionID || repair["candidate_id"] != candidate.ID {
		t.Fatalf("existing accepted plan did not receive honest validation-only recovery: %+v %v", repair, err)
	}
	planCandidateTestAssertSnapshot(t, root, snapshot)
	if visual := renderPlanVisual(repair); !strings.Contains(visual, "approval are unchanged") || strings.Contains(visual, "Repaired phase-plan") {
		t.Fatalf("misleading accepted repair output: %s", visual)
	}
	_, _, _, dispatches, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{LightFlag: true})
	if err != nil {
		t.Fatalf("accepted dependency rejected by actual build preparation: %v", err)
	}
	if len(dispatches) == 0 {
		t.Fatal("accepted plan produced no dispatch")
	}
	assertDependencyDispatchOrder(t, dispatches, "1.1", "1.2")
	after, _ := json.Marshal(mustReadSpecificationTestState(t, root).Plan)
	if string(before) != string(after) {
		t.Fatal("build changed accepted revision, candidate or approval hashes")
	}
}

func assertDependencyDispatchOrder(t *testing.T, dispatches []codexBuildDispatch, first, second string) {
	t.Helper()
	var predecessor, dependent *codexBuildDispatch
	firstIndex, secondIndex := -1, -1
	for i := range dispatches {
		ids := dispatches[i].CoveredTaskIDs
		if len(ids) == 0 {
			ids = []string{normalizedDispatchTaskID(dispatches[i])}
		}
		for position, id := range ids {
			if id == first {
				predecessor, firstIndex = &dispatches[i], position
			}
			if id == second {
				dependent, secondIndex = &dispatches[i], position
			}
		}
	}
	if predecessor == nil || dependent == nil {
		t.Fatal("dependency task missing from dispatch")
	}
	if predecessor == dependent {
		if firstIndex >= secondIndex {
			t.Fatal("coherent job runs dependent before its prerequisite")
		}
	} else if predecessor.ExecutionWave >= dependent.ExecutionWave || !containsString(dependent.DependsOn, predecessor.Name) {
		t.Fatalf("dependency edge missing from real dispatch schedule: %+v -> %+v", predecessor, dependent)
	}
}

func TestRepairArtifactRejectsConflictingOperations(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	setupBuildFlowTest(t)
	for _, flags := range [][]string{{"--plan-only"}, {"--preset", "fast"}, {"--print-brief"}, {"--refresh"}, {"--full"}} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			resetFlags(planCmd)
			if err := planCmd.ParseFlags(append([]string{"--repair-artifact"}, flags...)); err != nil {
				t.Fatal(err)
			}
			if err := planCmd.RunE(planCmd, nil); err == nil || !strings.Contains(err.Error(), "cannot be combined") {
				t.Fatalf("conflicting repair operation was ignored: %v", err)
			}
		})
	}
}
