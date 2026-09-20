package cmd

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func TestSwarmScope199(t *testing.T) {
	saveGlobals(t)
	root := t.TempDir()
	dataDir := filepath.Join(root, ".aether", "data")
	var err error
	store, err = storage.NewStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	targetID, dependencyID, dependentID, independentID := "job-target", "job-base", "job-dependent", "job-docs"
	state := colony.ColonyState{
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID: 1, Status: colony.PhaseInProgress,
			Tasks: []colony.Task{
				{ID: &dependencyID, Goal: "Prepare the dependency", Status: colony.TaskCompleted},
				{ID: &targetID, Goal: "Repair the affected path", Status: colony.TaskInProgress, DependsOn: []string{dependencyID}},
				{ID: &dependentID, Goal: "Wait for the affected repair", Status: colony.TaskInProgress, DependsOn: []string{targetID}},
				{ID: &independentID, Goal: "Continue independent documentation", Status: colony.TaskInProgress},
			},
		}}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	result, err := runSwarmPlanOnly(root, targetID)
	if err != nil {
		t.Fatalf("swarm plan-only: %v", err)
	}
	contract, ok := result["intervention_contract"].(SwarmInterventionContract)
	if !ok {
		t.Fatalf("intervention_contract = %T, want SwarmInterventionContract", result["intervention_contract"])
	}
	if contract.AffectedJobID != targetID {
		t.Fatalf("affected job = %q, want %q", contract.AffectedJobID, targetID)
	}
	if got, want := strings.Join(contract.DependencyPath, ","), dependencyID+","+dependentID; got != want {
		t.Fatalf("dependency path = %q, want %q", got, want)
	}
	if got := strings.Join(contract.IndependentJobIDs, ","); got != independentID {
		t.Fatalf("independent jobs = %q, want %q", got, independentID)
	}
	// CEC-07 (203-12-PLAN.md, Task 2): the limitation sentence no longer
	// names a phase number -- it states in present tense what evidence this
	// preflight is missing, since a phase-numbered claim silently becomes
	// false the day that phase ships and nothing detects it.
	if contract.CurrentCapability != "read_only_localization_preflight" || !strings.Contains(contract.Limitation, "checkpoint evidence") {
		t.Fatalf("capability boundary is not explicit: %+v", contract)
	}
	if strings.Contains(contract.Limitation, "Phase") {
		t.Fatalf("limitation sentence still names a phase number: %q", contract.Limitation)
	}
	if contract.AffectedCheckpoint != "" || contract.VerifiedResultEvidenceID != "" {
		t.Fatalf("preflight invented live checkpoint/result evidence: %+v", contract)
	}
	visual := renderSwarmCompatibilityVisual(result)
	for _, want := range []string{"Affected job: job-target", "Current capability: read_only_localization_preflight", "checkpoint evidence"} {
		if !strings.Contains(visual, want) {
			t.Fatalf("swarm visual lacks %q:\n%s", want, visual)
		}
	}

	facts, err := loadLifecycleFacts(root, store, time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	projection := projectLifecycle(facts, LifecycleViewFocused, "runtime")
	_, err = BuildSwarmInterventionContract(projection, SwarmInterventionEvidence{
		AffectedJobID:            targetID,
		VerifiedResultEvidenceID: "unverified-direct-result",
	})
	if err == nil || !strings.Contains(err.Error(), "checkpoint evidence") {
		t.Fatalf("unverified result integration error = %v", err)
	}
}
