package cmd

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanOnlyEmitsExplicitOrchestratorColonyMode(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	setupRuntimeSkillAssignmentHub(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	withTestWorkspace(t, root)

	goal := "orchestrator plan mode"
	state := colony.ColonyState{
		Version:    "3.0",
		Goal:       &goal,
		State:      colony.StateREADY,
		ColonyMode: colony.ColonyModeOrchestrator,
		Plan:       colony.Plan{Phases: []colony.Phase{}},
	}
	createTestColonyState(t, dataDir, state)

	result, err := runCodexPlanWithOptions(root, codexPlanOptions{PlanOnly: true})
	if err != nil {
		t.Fatalf("runCodexPlanWithOptions: %v", err)
	}
	if result["colony_mode"] != string(colony.ColonyModeOrchestrator) {
		t.Fatalf("top-level colony_mode = %v, want orchestrator", result["colony_mode"])
	}
	manifest, ok := result["plan_manifest"].(codexPlanManifest)
	if !ok {
		t.Fatalf("plan_manifest type = %T, want codexPlanManifest", result["plan_manifest"])
	}
	if manifest.ColonyMode != string(colony.ColonyModeOrchestrator) {
		t.Fatalf("manifest colony_mode = %q, want orchestrator", manifest.ColonyMode)
	}
	assertPlanOnlyGuidanceActive(t, result, manifest.OrchestratorGuidance, "plan", "aether plan")
}

func TestBuildPlanOnlyEmitsExplicitOrchestratorColonyMode(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	setupRuntimeSkillAssignmentHub(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	withTestWorkspace(t, root)

	goal := "orchestrator build mode"
	id := "1.1"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		ColonyMode:   colony.ColonyModeOrchestrator,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:          1,
			Name:        "Phase 1",
			Description: "test phase",
			Status:      colony.PhaseReady,
			Tasks: []colony.Task{{
				ID:     &id,
				Goal:   "test task",
				Status: colony.TaskPending,
			}},
		}}},
	}
	createTestColonyState(t, dataDir, state)

	result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnly: %v", err)
	}
	if result["colony_mode"] != string(colony.ColonyModeOrchestrator) {
		t.Fatalf("top-level colony_mode = %v, want orchestrator", result["colony_mode"])
	}
	manifest, ok := result["dispatch_manifest"].(codexBuildManifest)
	if !ok {
		t.Fatalf("dispatch_manifest type = %T, want codexBuildManifest", result["dispatch_manifest"])
	}
	if manifest.ColonyMode != string(colony.ColonyModeOrchestrator) {
		t.Fatalf("manifest colony_mode = %q, want orchestrator", manifest.ColonyMode)
	}
	assertPlanOnlyGuidanceActive(t, result, manifest.OrchestratorGuidance, "build", "aether build 1")
}

func TestContinuePlanOnlyEmitsExplicitOrchestratorColonyMode(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	setupRuntimeSkillAssignmentHub(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	withTestWorkspace(t, root)

	goal := "orchestrator continue mode"
	now := time.Now()
	id := "1.1"
	state := colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		ColonyMode:     colony.ColonyModeOrchestrator,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:          1,
			Name:        "Phase 1",
			Description: "test phase",
			Status:      colony.PhaseInProgress,
			Tasks: []colony.Task{{
				ID:     &id,
				Goal:   "test task",
				Status: colony.TaskInProgress,
			}},
		}}},
	}
	createTestColonyState(t, dataDir, state)
	seedContinueBuildPacket(t, dataDir, 1, "Phase 1", goal, []codexBuildDispatch{
		{
			Stage:   "wave",
			Caste:   "builder",
			Name:    "Forge-1",
			Task:    "test task",
			Status:  "completed",
			Outputs: []string{"main.go"},
		},
		{
			Stage:   "verification",
			Caste:   "watcher",
			Name:    "Keen-1",
			Task:    "verify test task",
			Status:  "completed",
			Outputs: []string{"main.go"},
		},
	})

	result, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{SkipWatchers: true, LightFlag: true})
	if err != nil {
		t.Fatalf("runCodexContinuePlanOnly: %v", err)
	}
	if result["colony_mode"] != string(colony.ColonyModeOrchestrator) {
		t.Fatalf("top-level colony_mode = %v, want orchestrator", result["colony_mode"])
	}
	manifest, ok := result["continue_manifest"].(codexContinuePlanManifest)
	if !ok {
		t.Fatalf("continue_manifest type = %T, want codexContinuePlanManifest", result["continue_manifest"])
	}
	if manifest.ColonyMode != string(colony.ColonyModeOrchestrator) {
		t.Fatalf("manifest colony_mode = %q, want orchestrator", manifest.ColonyMode)
	}
	assertPlanOnlyGuidanceActive(t, result, manifest.OrchestratorGuidance, "continue", "aether continue")
}

func TestSealPlanOnlyEmitsExplicitOrchestratorColonyMode(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	setupRuntimeSkillAssignmentHub(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	withTestWorkspace(t, root)

	goal := "orchestrator seal mode"
	id := "1.1"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		ColonyMode:   colony.ColonyModeOrchestrator,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:          1,
			Name:        "Phase 1",
			Description: "test phase",
			Status:      colony.PhaseCompleted,
			Tasks: []colony.Task{{
				ID:     &id,
				Goal:   "test task",
				Status: colony.TaskCompleted,
			}},
		}}},
	}
	createTestColonyState(t, dataDir, state)

	result, err := runSealPlanOnly(root, false)
	if err != nil {
		t.Fatalf("runSealPlanOnly: %v", err)
	}
	if result["colony_mode"] != string(colony.ColonyModeOrchestrator) {
		t.Fatalf("top-level colony_mode = %v, want orchestrator", result["colony_mode"])
	}
	manifest, ok := result["seal_manifest"].(sealPlanManifest)
	if !ok {
		t.Fatalf("seal_manifest type = %T, want sealPlanManifest", result["seal_manifest"])
	}
	if manifest.ColonyMode != string(colony.ColonyModeOrchestrator) {
		t.Fatalf("manifest colony_mode = %q, want orchestrator", manifest.ColonyMode)
	}
	assertPlanOnlyGuidanceActive(t, result, manifest.OrchestratorGuidance, "seal", "aether seal")
}

func assertPlanOnlyGuidanceActive(t *testing.T, result map[string]interface{}, manifestGuidance *orchestratorBoundaryGuidance, workflow, afterDiscussNext string) {
	t.Helper()
	if got := result["next"]; got != "aether discuss" {
		t.Fatalf("next = %v, want aether discuss", got)
	}
	if got := result["after_discuss_next"]; got != afterDiscussNext {
		t.Fatalf("after_discuss_next = %v, want %s", got, afterDiscussNext)
	}
	guidance, ok := result["orchestrator_boundary_guidance"].(orchestratorBoundaryGuidance)
	if !ok {
		t.Fatalf("orchestrator_boundary_guidance = %#v, want orchestratorBoundaryGuidance", result["orchestrator_boundary_guidance"])
	}
	if !guidance.Active || guidance.Workflow != workflow || guidance.PendingCount != 1 {
		t.Fatalf("guidance active/workflow/pending = %v/%q/%d, want true/%q/1", guidance.Active, guidance.Workflow, guidance.PendingCount, workflow)
	}
	if guidance.Next != "aether discuss" || guidance.AfterDiscussNext != afterDiscussNext {
		t.Fatalf("guidance next/after = %q/%q, want aether discuss/%q", guidance.Next, guidance.AfterDiscussNext, afterDiscussNext)
	}
	if manifestGuidance == nil {
		t.Fatalf("manifest orchestrator_boundary_guidance missing")
	}
	if !manifestGuidance.Active || manifestGuidance.Workflow != workflow || manifestGuidance.PendingCount != guidance.PendingCount {
		t.Fatalf("manifest guidance = %#v, want active workflow %q pending %d", manifestGuidance, workflow, guidance.PendingCount)
	}
	if manifestGuidance.Next != guidance.Next || manifestGuidance.AfterDiscussNext != guidance.AfterDiscussNext {
		t.Fatalf("manifest guidance next/after = %q/%q, want %q/%q", manifestGuidance.Next, manifestGuidance.AfterDiscussNext, guidance.Next, guidance.AfterDiscussNext)
	}
}
