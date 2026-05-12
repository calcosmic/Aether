package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestOrchestratorBoundaryGuidanceRoutesPendingQuestionsToDiscuss(t *testing.T) {
	saveGlobals(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	source := orchestratorBoundaryClarificationSource("build", 1, "build-scope", true)
	if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{Decisions: []PendingDecision{{
		ID:          "pd_build_boundary",
		Type:        clarificationDecisionType,
		Description: formatClarificationDescription("What boundary should builders protect?", []string{"selected tasks only", "pause"}),
		Source:      source,
		Resolved:    false,
		CreatedAt:   "2026-05-08T09:00:00Z",
	}}}); err != nil {
		t.Fatalf("seed pending decisions: %v", err)
	}

	result := map[string]interface{}{"next": "aether continue"}
	state := colony.ColonyState{ColonyMode: colony.ColonyModeOrchestrator}
	addOrchestratorBoundaryGuidance(result, "build", state, "aether continue", []discussQuestion{{
		ID:     "pd_build_boundary",
		Source: source,
	}})

	if got := result["next"]; got != "aether discuss" {
		t.Fatalf("next = %v, want aether discuss", got)
	}
	if got := result["after_discuss_next"]; got != "aether continue" {
		t.Fatalf("after_discuss_next = %v, want aether continue", got)
	}
	guidance, ok := result["orchestrator_boundary_guidance"].(orchestratorBoundaryGuidance)
	if !ok {
		t.Fatalf("orchestrator_boundary_guidance = %#v, want orchestratorBoundaryGuidance", result["orchestrator_boundary_guidance"])
	}
	if !guidance.Active || guidance.PendingCount != 1 {
		t.Fatalf("guidance active/pending = %v/%d, want true/1", guidance.Active, guidance.PendingCount)
	}
	if guidance.Next != "aether discuss" || guidance.AfterDiscussNext != "aether continue" {
		t.Fatalf("guidance next/after = %q/%q", guidance.Next, guidance.AfterDiscussNext)
	}
	if len(guidance.QuestionIDs) != 1 || guidance.QuestionIDs[0] != "pd_build_boundary" {
		t.Fatalf("question ids = %#v", guidance.QuestionIDs)
	}
	if len(guidance.QuestionSources) != 1 || guidance.QuestionSources[0] != source {
		t.Fatalf("question sources = %#v", guidance.QuestionSources)
	}
	if len(guidance.QuestionSummaries) != 1 || guidance.QuestionSummaries[0].Question != "What boundary should builders protect?" {
		t.Fatalf("question summaries = %#v", guidance.QuestionSummaries)
	}
	if !strings.Contains(guidance.Summary, "1 unresolved Orchestrator boundary question") {
		t.Fatalf("summary = %q", guidance.Summary)
	}
}

func TestOrchestratorBoundaryGuidanceKeepsNormalNextWhenResolved(t *testing.T) {
	saveGlobals(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	source := orchestratorBoundaryClarificationSource("plan", 1, "planning-scope", false)
	if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{Decisions: []PendingDecision{{
		ID:          "pd_plan_boundary",
		Type:        clarificationDecisionType,
		Description: formatClarificationDescription("What should the first generated plan optimize for?", []string{"smallest useful slice"}),
		Source:      source,
		Resolved:    true,
		Resolution:  "Smallest useful slice.",
		CreatedAt:   "2026-05-08T09:00:00Z",
		ResolvedAt:  "2026-05-08T09:01:00Z",
	}}}); err != nil {
		t.Fatalf("seed pending decisions: %v", err)
	}

	result := map[string]interface{}{"next": "aether build 1"}
	state := colony.ColonyState{ColonyMode: colony.ColonyModeOrchestrator}
	addOrchestratorBoundaryGuidance(result, "plan", state, "aether build 1", []discussQuestion{{
		ID:     "pd_plan_boundary",
		Source: source,
	}})

	if got := result["next"]; got != "aether build 1" {
		t.Fatalf("next = %v, want aether build 1", got)
	}
	if _, ok := result["after_discuss_next"]; ok {
		t.Fatalf("after_discuss_next should be omitted when no boundary questions are pending")
	}
	guidance, ok := result["orchestrator_boundary_guidance"].(orchestratorBoundaryGuidance)
	if !ok {
		t.Fatalf("orchestrator_boundary_guidance = %#v, want orchestratorBoundaryGuidance", result["orchestrator_boundary_guidance"])
	}
	if guidance.Active || guidance.PendingCount != 0 {
		t.Fatalf("guidance active/pending = %v/%d, want false/0", guidance.Active, guidance.PendingCount)
	}
}

func TestDefaultModeDoesNotAddOrchestratorBoundaryGuidance(t *testing.T) {
	saveGlobals(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	result := map[string]interface{}{"next": "aether continue"}
	addOrchestratorBoundaryGuidance(result, "build", colony.ColonyState{}, "aether continue", nil)

	if got := result["next"]; got != "aether continue" {
		t.Fatalf("next = %v, want aether continue", got)
	}
	if _, ok := result["orchestrator_boundary_guidance"]; ok {
		t.Fatalf("default mode should not add orchestrator boundary guidance: %#v", result)
	}
}

func TestOrchestratorBoundaryGuidanceMatchesWorkflowWithoutManifestQuestions(t *testing.T) {
	saveGlobals(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	continueSource := orchestratorBoundaryClarificationSource("continue", 2, "advance", false)
	buildSource := orchestratorBoundaryClarificationSource("build", 1, "build-scope", true)
	if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{Decisions: []PendingDecision{
		{
			ID:          "pd_continue_boundary",
			Type:        clarificationDecisionType,
			Description: formatClarificationDescription("What should continue preserve?", []string{"advance after reviews"}),
			Source:      continueSource,
			Resolved:    false,
			CreatedAt:   "2026-05-08T09:00:00Z",
		},
		{
			ID:          "pd_build_boundary",
			Type:        clarificationDecisionType,
			Description: formatClarificationDescription("What should build protect?", []string{"phase tasks only"}),
			Source:      buildSource,
			Resolved:    false,
			CreatedAt:   "2026-05-08T09:00:00Z",
		},
	}}); err != nil {
		t.Fatalf("seed pending decisions: %v", err)
	}

	result := map[string]interface{}{"next": "aether build 2"}
	state := colony.ColonyState{ColonyMode: colony.ColonyModeOrchestrator}
	addOrchestratorBoundaryGuidance(result, "continue", state, "aether build 2", nil)

	guidance := result["orchestrator_boundary_guidance"].(orchestratorBoundaryGuidance)
	if guidance.PendingCount != 1 {
		t.Fatalf("pending count = %d, want only the continue boundary", guidance.PendingCount)
	}
	if len(guidance.QuestionSources) != 1 || guidance.QuestionSources[0] != continueSource {
		t.Fatalf("question sources = %#v, want %q", guidance.QuestionSources, continueSource)
	}
}

func TestBuildFinalizeAddsOrchestratorBoundaryGuidance(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	withTestWorkspace(t, root)

	goal := "Build finalizer boundary guidance"
	taskID := "1.1"
	startedAt := time.Now().UTC()
	state := colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: &startedAt,
		ColonyMode:     colony.ColonyModeOrchestrator,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Boundary build",
			Status: colony.PhaseInProgress,
			Tasks:  []colony.Task{{ID: &taskID, Goal: "Implement boundary guidance", Status: colony.TaskInProgress}},
		}}},
	}
	createTestColonyState(t, dataDir, state)

	source := orchestratorBoundaryClarificationSource("build", 1, "build-scope", true)
	if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{Decisions: []PendingDecision{{
		ID:          "pd_build_finalize_boundary",
		Type:        clarificationDecisionType,
		Description: formatClarificationDescription("What boundary should builders protect for Phase 1?", []string{"phase tasks only", "pause"}),
		Source:      source,
		Resolved:    false,
		CreatedAt:   "2026-05-08T09:00:00Z",
	}}}); err != nil {
		t.Fatalf("seed pending decisions: %v", err)
	}

	manifest := codexBuildManifest{
		Phase:        1,
		PhaseName:    "Boundary build",
		Root:         root,
		ColonyMode:   string(colony.ColonyModeOrchestrator),
		PlanOnly:     true,
		DispatchMode: "plan-only",
		GeneratedAt:  startedAt.Format(time.RFC3339),
		State:        string(colony.StateEXECUTING),
		Dispatches: []codexBuildDispatch{{
			Stage:  "wave",
			Wave:   1,
			Caste:  "builder",
			Name:   "Mason-14",
			Task:   "Implement boundary guidance",
			TaskID: taskID,
			Status: "planned",
		}},
		SelectedTasks: []string{taskID},
		BoundaryQuestions: []discussQuestion{{
			ID:     "pd_build_finalize_boundary",
			Source: source,
		}},
	}
	completion := codexExternalBuildCompletion{
		DispatchManifest: &manifest,
		Results: []codexExternalBuildWorkerResult{{
			Stage:         "wave",
			Wave:          1,
			ExecutionWave: 1,
			Caste:         "builder",
			Name:          "Mason-14",
			TaskID:        taskID,
			Status:        "completed",
			Summary:       "Implemented boundary guidance",
			FilesModified: []string{"cmd/orchestrator_boundary_guidance.go"},
		}},
	}
	writeClaimFileForTest(t, root, "cmd/orchestrator_boundary_guidance.go")

	result, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("runCodexBuildFinalize: %v", err)
	}
	if got := result["next"]; got != "aether discuss" {
		t.Fatalf("next = %v, want aether discuss", got)
	}
	if got := result["after_discuss_next"]; got != "aether continue" {
		t.Fatalf("after_discuss_next = %v, want aether continue", got)
	}
	guidance := result["orchestrator_boundary_guidance"].(orchestratorBoundaryGuidance)
	if !guidance.Active || guidance.PendingCount != 1 {
		t.Fatalf("guidance active/pending = %v/%d, want true/1", guidance.Active, guidance.PendingCount)
	}
}

func TestPlanFinalizeAddsOrchestratorBoundaryGuidance(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	setupRuntimeSkillAssignmentHub(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	withTestWorkspace(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-plan-guidance-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	goal := "Finalize planning with orchestrator guidance"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:    "3.0",
		Goal:       &goal,
		State:      colony.StateREADY,
		ColonyMode: colony.ColonyModeOrchestrator,
		Plan:       colony.Plan{Phases: []colony.Phase{}},
	})

	planResult, err := runCodexPlanWithOptions(root, codexPlanOptions{PlanOnly: true, Depth: "fast"})
	if err != nil {
		t.Fatalf("runCodexPlanWithOptions: %v", err)
	}
	manifest := planResult["plan_manifest"].(codexPlanManifest)
	if len(manifest.BoundaryQuestions) == 0 {
		t.Fatalf("expected plan boundary question in manifest")
	}

	scout := manifest.Dispatches[0]
	scout.Status = "completed"
	scout.Summary = "Scout mapped planning context."
	scout.ScoutReport = &codexScoutReport{
		Findings:   []codexScoutFinding{{Area: "Runtime", Discovery: "Plan finalizer owns planning state.", Source: "cmd/codex_plan_finalize.go"}},
		Confidence: 91,
		StudyFiles: []string{"cmd/codex_plan_finalize.go"},
	}
	routeSetter := manifest.Dispatches[1]
	routeSetter.Status = "completed"
	routeSetter.Summary = "Route-Setter shaped the first plan."
	routeSetter.PhasePlan = &codexWorkerPlanArtifact{
		Phases: []codexWorkerPlanPhase{{
			Name:        "Guided plan",
			Description: "Prove plan finalizer guidance.",
			Tasks: []codexWorkerPlanTask{{
				Goal:            "Route unresolved plan boundary questions through discuss",
				SuccessCriteria: []string{"plan finalizer guidance is active"},
			}},
			SuccessCriteria: []string{"Guidance is emitted"},
		}},
		Confidence: codexPlanConfidence{Knowledge: 90, Requirements: 90, Risks: 85, Dependencies: 85, Effort: 85, Overall: 87},
	}

	result, err := runCodexPlanFinalize(root, codexExternalPlanCompletion{
		PlanManifest: &manifest,
		Dispatches:   []codexPlanningDispatch{scout, routeSetter},
	})
	if err != nil {
		t.Fatalf("runCodexPlanFinalize: %v", err)
	}
	if got := result["next"]; got != "aether discuss" {
		t.Fatalf("next = %v, want aether discuss", got)
	}
	if got := result["after_discuss_next"]; got != "aether build 1" {
		t.Fatalf("after_discuss_next = %v, want aether build 1", got)
	}
	guidance := result["orchestrator_boundary_guidance"].(orchestratorBoundaryGuidance)
	if !guidance.Active || guidance.Workflow != "plan" || guidance.PendingCount != 1 {
		t.Fatalf("guidance = %#v, want active plan guidance with one pending question", guidance)
	}
}

func TestContinueFinalizeAddsOrchestratorBoundaryGuidance(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	setupRuntimeSkillAssignmentHub(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	withTestWorkspace(t, root)

	goal := "Finalize continue with orchestrator guidance"
	now := time.Now().UTC()
	taskID := "1.1"
	nextTaskID := "2.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		ColonyMode:     colony.ColonyModeOrchestrator,
		Plan: colony.Plan{Phases: []colony.Phase{
			{
				ID:     1,
				Name:   "Continue boundary",
				Status: colony.PhaseInProgress,
				Tasks:  []colony.Task{{ID: &taskID, Goal: "Verify current phase", Status: colony.TaskInProgress}},
			},
			{
				ID:     2,
				Name:   "Next phase",
				Status: colony.PhasePending,
				Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Continue after guidance", Status: colony.TaskPending}},
			},
		}},
	})
	seedContinueBuildPacket(t, dataDir, 1, "Continue boundary", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Mason-continue", Task: "Verify current phase", Status: "completed", TaskID: taskID, Outputs: []string{"cmd/orchestrator_boundary_guidance.go"}},
		{Stage: "verification", Caste: "watcher", Name: "Keen-continue", Task: "Verify current phase", Status: "completed", Outputs: []string{"cmd/orchestrator_boundary_guidance_test.go"}},
	})

	planResult, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{HeavyFlag: true})
	if err != nil {
		t.Fatalf("runCodexContinuePlanOnly: %v", err)
	}
	plan := planResult["continue_manifest"].(codexContinuePlanManifest)
	if len(plan.BoundaryQuestions) == 0 {
		t.Fatalf("expected continue boundary question in manifest")
	}
	results := make([]codexContinueExternalDispatch, 0, len(plan.Dispatches))
	for _, dispatch := range plan.Dispatches {
		result := dispatch
		result.Status = "completed"
		result.Summary = dispatch.Name + " cleared continue review"
		results = append(results, result)
	}

	result, _, _, _, _, _, err := runCodexContinueFinalize(root, codexExternalContinueCompletion{
		ContinueManifest: &plan,
		Dispatches:       results,
	}, false, 0, false)
	if err != nil {
		t.Fatalf("runCodexContinueFinalize: %v", err)
	}
	if got := result["next"]; got != "aether discuss" {
		t.Fatalf("next = %v, want aether discuss", got)
	}
	if got := result["after_discuss_next"]; got != "aether build 2" {
		t.Fatalf("after_discuss_next = %v, want aether build 2", got)
	}
	guidance := result["orchestrator_boundary_guidance"].(orchestratorBoundaryGuidance)
	if !guidance.Active || guidance.Workflow != "continue" || guidance.PendingCount != 1 {
		t.Fatalf("guidance = %#v, want active continue guidance with one pending question", guidance)
	}
}

func TestSealFinalizeBlocksUnresolvedOrchestratorBoundaryGuidance(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	setupRuntimeSkillAssignmentHub(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	withTestWorkspace(t, root)

	goal := "Seal with unresolved orchestrator guidance"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateCOMPLETED,
		CurrentPhase: 1,
		ColonyMode:   colony.ColonyModeOrchestrator,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Complete work",
			Status: colony.PhaseCompleted,
			Mode:   colony.PhaseModeProduction,
		}}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	planResult, err := runSealPlanOnly(root, false)
	if err != nil {
		t.Fatalf("runSealPlanOnly: %v", err)
	}
	manifest := planResult["seal_manifest"].(sealPlanManifest)
	if len(manifest.BoundaryQuestions) == 0 {
		t.Fatalf("expected seal boundary question in manifest")
	}
	results := append([]codexContinueExternalDispatch{}, manifest.Dispatches...)
	for i := range results {
		results[i].Status = "completed"
		results[i].Summary = results[i].Name + " cleared final review"
		results[i].Report = "# Final review\n\nNo blockers."
	}

	err = runSealFinalize(root, externalSealCompletion{SealManifest: &manifest, Dispatches: results})
	if err == nil || !strings.Contains(err.Error(), "aether discuss") || !strings.Contains(err.Error(), "aether seal") {
		t.Fatalf("runSealFinalize error = %v, want guidance to run aether discuss before aether seal", err)
	}
	var after colony.ColonyState
	if loadErr := store.LoadJSON("COLONY_STATE.json", &after); loadErr != nil {
		t.Fatalf("load state: %v", loadErr)
	}
	if after.Milestone == "Crowned Anthill" {
		t.Fatalf("seal-finalize sealed despite unresolved Orchestrator boundary guidance: %+v", after)
	}
	if _, statErr := os.Stat(filepath.Join(root, ".aether", "CROWNED-ANTHILL.md")); !os.IsNotExist(statErr) {
		t.Fatalf("seal-finalize should not write CROWNED-ANTHILL.md, stat err=%v", statErr)
	}
}

func TestFinalizerManifestValidationAcceptsLegacyAndMatchingValues(t *testing.T) {
	root := t.TempDir()
	if err := validateFinalizerManifestRoot("dispatch_manifest", "", root); err != nil {
		t.Fatalf("empty root should be accepted: %v", err)
	}
	if err := validateFinalizerManifestRoot("dispatch_manifest", root, root); err != nil {
		t.Fatalf("matching root should be accepted: %v", err)
	}

	state := colony.ColonyState{ColonyMode: colony.ColonyModeOrchestrator}
	if err := validateFinalizerManifestColonyMode("dispatch_manifest", "", state); err != nil {
		t.Fatalf("empty colony_mode should be accepted for legacy manifests: %v", err)
	}
	if err := validateFinalizerManifestColonyMode("dispatch_manifest", "orchestrator", state); err != nil {
		t.Fatalf("matching colony_mode should be accepted: %v", err)
	}
}

func TestFinalizerManifestValidationRejectsMismatches(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	if err := validateFinalizerManifestRoot("dispatch_manifest", other, root); err == nil || !strings.Contains(err.Error(), "root does not match") {
		t.Fatalf("root mismatch error = %v", err)
	}

	state := colony.ColonyState{ColonyMode: colony.ColonyModeOrchestrator}
	if err := validateFinalizerManifestColonyMode("dispatch_manifest", "colony", state); err == nil || !strings.Contains(err.Error(), "does not match active colony mode") {
		t.Fatalf("mode mismatch error = %v", err)
	}
	if err := validateFinalizerManifestColonyMode("dispatch_manifest", "sideways", state); err == nil || !strings.Contains(err.Error(), "is invalid") {
		t.Fatalf("invalid mode error = %v", err)
	}
}

func TestBuildFinalizeRejectsMismatchedOrchestratorManifestBeforeWrites(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	withTestWorkspace(t, root)

	goal := "Reject stale orchestrator manifest"
	taskID := "1.1"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Default build",
			Status: colony.PhaseReady,
			Tasks:  []colony.Task{{ID: &taskID, Goal: "Keep default mode unchanged", Status: colony.TaskPending}},
		}}},
	}
	createTestColonyState(t, dataDir, state)

	source := orchestratorBoundaryClarificationSource("build", 1, "build-scope", true)
	manifest := codexBuildManifest{
		Phase:        1,
		PhaseName:    "Default build",
		Root:         root,
		ColonyMode:   string(colony.ColonyModeOrchestrator),
		PlanOnly:     true,
		DispatchMode: "plan-only",
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		State:        string(colony.StateREADY),
		Dispatches: []codexBuildDispatch{{
			Stage:  "wave",
			Wave:   1,
			Caste:  "builder",
			Name:   "Mason-stale",
			Task:   "Keep default mode unchanged",
			TaskID: taskID,
			Status: "planned",
		}},
		SelectedTasks: []string{taskID},
		BoundaryQuestions: []discussQuestion{{
			ID:     "pd_stale_boundary",
			Source: source,
		}},
	}

	_, _, _, _, err := runCodexBuildFinalize(root, 1, codexExternalBuildCompletion{DispatchManifest: &manifest}, false)
	if err == nil || !strings.Contains(err.Error(), "does not match active colony mode") {
		t.Fatalf("runCodexBuildFinalize error = %v, want colony mode mismatch", err)
	}

	var loaded colony.ColonyState
	if loadErr := store.LoadJSON("COLONY_STATE.json", &loaded); loadErr != nil {
		t.Fatalf("load state: %v", loadErr)
	}
	if loaded.State != colony.StateREADY || loaded.Plan.Phases[0].Status != colony.PhaseReady {
		t.Fatalf("mismatched manifest mutated state: state=%s phase=%s", loaded.State, loaded.Plan.Phases[0].Status)
	}
	if _, statErr := os.Stat(filepath.Join(dataDir, "build", "phase-1", "manifest.json")); !os.IsNotExist(statErr) {
		t.Fatalf("mismatched manifest should not write build manifest, stat err=%v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(dataDir, pendingDecisionsFile)); !os.IsNotExist(statErr) {
		t.Fatalf("build finalizer should not write boundary questions, stat err=%v", statErr)
	}
}
