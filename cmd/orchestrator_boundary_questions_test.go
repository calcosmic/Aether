package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestOrchestratorBoundaryQuestionsCreatedForPlanOnlyWorkflows(t *testing.T) {
	t.Run("plan", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		setupRuntimeSkillAssignmentHub(t)

		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))
		withWorkingDir(t, root)
		withTestWorkspace(t, root)

		goal := "orchestrate boundary questions"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version:    "3.0",
			Goal:       &goal,
			State:      colony.StateREADY,
			ColonyMode: colony.ColonyModeOrchestrator,
			Plan:       colony.Plan{Phases: []colony.Phase{}},
		})

		result, err := runCodexPlanWithOptions(root, codexPlanOptions{PlanOnly: true})
		if err != nil {
			t.Fatalf("runCodexPlanWithOptions: %v", err)
		}
		assertBoundaryQuestionsCreated(t, dataDir, result, "orchestrator:plan:phase:1:planning-scope")
		manifest := result["plan_manifest"].(codexPlanManifest)
		if manifest.BoundaryQuestionCount != 1 {
			t.Fatalf("manifest BoundaryQuestionCount = %d, want 1", manifest.BoundaryQuestionCount)
		}
	})

	t.Run("build", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		setupRuntimeSkillAssignmentHub(t)

		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))
		withWorkingDir(t, root)
		withTestWorkspace(t, root)

		goal := "orchestrate build boundary"
		taskID := "1.1"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version:      "3.0",
			Goal:         &goal,
			State:        colony.StateREADY,
			CurrentPhase: 1,
			ColonyMode:   colony.ColonyModeOrchestrator,
			Plan: colony.Plan{Phases: []colony.Phase{{
				ID:          1,
				Name:        "Build boundary",
				Description: "prove the build boundary",
				Status:      colony.PhaseReady,
				Tasks: []colony.Task{{
					ID:     &taskID,
					Goal:   "Implement the build boundary",
					Status: colony.TaskPending,
				}},
			}}},
		})

		result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
		if err != nil {
			t.Fatalf("runCodexBuildPlanOnly: %v", err)
		}
		assertBoundaryQuestionsCreated(t, dataDir, result, "orchestrator:build:phase:1:build-scope:hard")
		manifest := result["dispatch_manifest"].(codexBuildManifest)
		if manifest.BoundaryQuestionCount != 1 {
			t.Fatalf("manifest BoundaryQuestionCount = %d, want 1", manifest.BoundaryQuestionCount)
		}
	})

	t.Run("continue", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		setupRuntimeSkillAssignmentHub(t)

		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))
		withWorkingDir(t, root)
		withTestWorkspace(t, root)

		goal := "orchestrate continue boundary"
		now := time.Now().UTC()
		taskID := "1.1"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version:        "3.0",
			Goal:           &goal,
			State:          colony.StateBUILT,
			CurrentPhase:   1,
			BuildStartedAt: &now,
			ColonyMode:     colony.ColonyModeOrchestrator,
			Plan: colony.Plan{Phases: []colony.Phase{{
				ID:          1,
				Name:        "Continue boundary",
				Description: "verify the boundary",
				Status:      colony.PhaseInProgress,
				Tasks: []colony.Task{{
					ID:     &taskID,
					Goal:   "Finish the boundary",
					Status: colony.TaskInProgress,
				}},
			}}},
		})
		seedContinueBuildPacket(t, dataDir, 1, "Continue boundary", goal, []codexBuildDispatch{
			{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-1", Task: "Finish the boundary", Status: "completed", TaskID: taskID},
			{Stage: "verification", Caste: "watcher", Name: "Keen-1", Task: "Build-time verification", Status: "completed"},
		})

		result, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{SkipWatchers: true, LightFlag: true})
		if err != nil {
			t.Fatalf("runCodexContinuePlanOnly: %v", err)
		}
		assertBoundaryQuestionsCreated(t, dataDir, result, "orchestrator:continue:phase:1:advance")
		manifest := result["continue_manifest"].(codexContinuePlanManifest)
		if manifest.BoundaryQuestionCount != 1 {
			t.Fatalf("manifest BoundaryQuestionCount = %d, want 1", manifest.BoundaryQuestionCount)
		}
	})

	t.Run("seal", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		setupRuntimeSkillAssignmentHub(t)

		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))
		withWorkingDir(t, root)
		withTestWorkspace(t, root)

		goal := "orchestrate seal boundary"
		taskID := "1.1"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version:      "3.0",
			Goal:         &goal,
			State:        colony.StateREADY,
			CurrentPhase: 1,
			ColonyMode:   colony.ColonyModeOrchestrator,
			Plan: colony.Plan{Phases: []colony.Phase{{
				ID:          1,
				Name:        "Seal boundary",
				Description: "ready to seal",
				Status:      colony.PhaseCompleted,
				Tasks: []colony.Task{{
					ID:     &taskID,
					Goal:   "Complete sealable work",
					Status: colony.TaskCompleted,
				}},
			}}},
		})

		result, err := runSealPlanOnly(root, false)
		if err != nil {
			t.Fatalf("runSealPlanOnly: %v", err)
		}
		assertBoundaryQuestionsCreated(t, dataDir, result, "orchestrator:seal:phase:1:release-boundary:hard")
		manifest := result["seal_manifest"].(sealPlanManifest)
		if manifest.BoundaryQuestionCount != 1 {
			t.Fatalf("manifest BoundaryQuestionCount = %d, want 1", manifest.BoundaryQuestionCount)
		}
	})
}

func TestDefaultColonyModeDoesNotCreateBoundaryQuestions(t *testing.T) {
	t.Run("plan", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		setupRuntimeSkillAssignmentHub(t)

		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))
		withWorkingDir(t, root)
		withTestWorkspace(t, root)

		goal := "default mode plan boundary"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version: "3.0",
			Goal:    &goal,
			State:   colony.StateREADY,
			Plan:    colony.Plan{Phases: []colony.Phase{}},
		})

		result, err := runCodexPlanWithOptions(root, codexPlanOptions{PlanOnly: true})
		if err != nil {
			t.Fatalf("runCodexPlanWithOptions: %v", err)
		}
		assertNoBoundaryQuestions(t, dataDir, result)
	})

	t.Run("build", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		setupRuntimeSkillAssignmentHub(t)

		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))
		withWorkingDir(t, root)
		withTestWorkspace(t, root)

		goal := "default mode build boundary"
		taskID := "1.1"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version:      "3.0",
			Goal:         &goal,
			State:        colony.StateREADY,
			CurrentPhase: 1,
			Plan: colony.Plan{Phases: []colony.Phase{{
				ID:          1,
				Name:        "Default build boundary",
				Description: "default mode should not ask",
				Status:      colony.PhaseReady,
				Tasks: []colony.Task{{
					ID:     &taskID,
					Goal:   "Implement default build",
					Status: colony.TaskPending,
				}},
			}}},
		})

		result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
		if err != nil {
			t.Fatalf("runCodexBuildPlanOnly: %v", err)
		}
		assertNoBoundaryQuestions(t, dataDir, result)
	})

	t.Run("continue", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		setupRuntimeSkillAssignmentHub(t)

		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))
		withWorkingDir(t, root)
		withTestWorkspace(t, root)

		goal := "default mode continue boundary"
		now := time.Now().UTC()
		taskID := "1.1"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version:        "3.0",
			Goal:           &goal,
			State:          colony.StateBUILT,
			CurrentPhase:   1,
			BuildStartedAt: &now,
			Plan: colony.Plan{Phases: []colony.Phase{{
				ID:          1,
				Name:        "Default continue boundary",
				Description: "default mode should not ask",
				Status:      colony.PhaseInProgress,
				Tasks: []colony.Task{{
					ID:     &taskID,
					Goal:   "Finish default work",
					Status: colony.TaskInProgress,
				}},
			}}},
		})
		seedContinueBuildPacket(t, dataDir, 1, "Default continue boundary", goal, []codexBuildDispatch{
			{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-default", Task: "Finish default work", Status: "completed", TaskID: taskID},
			{Stage: "verification", Caste: "watcher", Name: "Keen-default", Task: "Build-time verification", Status: "completed"},
		})

		result, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{SkipWatchers: true, LightFlag: true})
		if err != nil {
			t.Fatalf("runCodexContinuePlanOnly: %v", err)
		}
		assertNoBoundaryQuestions(t, dataDir, result)
	})

	t.Run("seal", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		setupRuntimeSkillAssignmentHub(t)

		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))
		withWorkingDir(t, root)
		withTestWorkspace(t, root)

		goal := "default mode seal boundary"
		taskID := "1.1"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version:      "3.0",
			Goal:         &goal,
			State:        colony.StateREADY,
			CurrentPhase: 1,
			Plan: colony.Plan{Phases: []colony.Phase{{
				ID:          1,
				Name:        "Default seal boundary",
				Description: "default mode should not ask",
				Status:      colony.PhaseCompleted,
				Tasks: []colony.Task{{
					ID:     &taskID,
					Goal:   "Complete default work",
					Status: colony.TaskCompleted,
				}},
			}}},
		})

		result, err := runSealPlanOnly(root, false)
		if err != nil {
			t.Fatalf("runSealPlanOnly: %v", err)
		}
		assertNoBoundaryQuestions(t, dataDir, result)
	})
}

func TestResolvedBoundaryQuestionFlowsThroughClarifiedIntent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	setupRuntimeSkillAssignmentHub(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	withTestWorkspace(t, root)

	goal := "resolve orchestrator boundary"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:    "3.0",
		Goal:       &goal,
		State:      colony.StateREADY,
		ColonyMode: colony.ColonyModeOrchestrator,
		Plan:       colony.Plan{Phases: []colony.Phase{}},
	})

	result, err := runCodexPlanWithOptions(root, codexPlanOptions{PlanOnly: true})
	if err != nil {
		t.Fatalf("runCodexPlanWithOptions: %v", err)
	}
	questions := result["boundary_questions"].([]discussQuestion)
	if len(questions) != 1 || strings.TrimSpace(questions[0].ID) == "" {
		t.Fatalf("boundary questions = %#v, want one persisted question", questions)
	}

	if _, err := resolveDiscussQuestion(questions[0].ID, "Keep the first plan to the smallest useful slice."); err != nil {
		t.Fatalf("resolveDiscussQuestion: %v", err)
	}

	lines := clarifiedIntentPromptEntries()
	rendered := strings.Join(lines, "\n")
	if !strings.Contains(rendered, questions[0].Question) {
		t.Fatalf("clarified intent missing boundary question %q in:\n%s", questions[0].Question, rendered)
	}
	if !strings.Contains(rendered, "Keep the first plan to the smallest useful slice.") {
		t.Fatalf("clarified intent missing boundary answer in:\n%s", rendered)
	}
}

func TestBoundaryQuestionCandidateVariants(t *testing.T) {
	selected := buildBoundaryQuestionCandidates(colony.Phase{ID: 2}, []string{"2.1"})
	if got := selected[0].Options[0]; got != "selected tasks only" {
		t.Fatalf("selected build option = %q, want selected tasks only", got)
	}
	if !strings.Contains(selected[0].Question, "Phase 2") {
		t.Fatalf("selected build question should use fallback phase label, got %q", selected[0].Question)
	}

	recovery := continueBoundaryQuestionCandidates(
		colony.Phase{ID: 3, Name: "Recovery"},
		codexContinueVerificationReport{Passed: false},
		codexContinueAssessment{Passed: false, Summary: "blocked", BlockingIssues: []string{"missing evidence"}},
	)
	if recovery[0].Category != "recovery" || !recovery[0].HardConstraint {
		t.Fatalf("continue recovery candidate = %#v, want hard recovery question", recovery[0])
	}

	seal := sealBoundaryQuestionCandidates(colony.ColonyState{}, colony.Phase{ID: 4}, colony.VerificationDepthHeavy)
	if !strings.Contains(seal[0].Reasoning, "the colony") {
		t.Fatalf("seal fallback reasoning = %q, want fallback colony goal", seal[0].Reasoning)
	}
}

func TestMaterializeOrchestratorBoundaryQuestionsReturnsStoreError(t *testing.T) {
	saveGlobals(t)
	store = nil

	state := colony.ColonyState{ColonyMode: colony.ColonyModeOrchestrator}
	_, err := materializeOrchestratorBoundaryQuestions("build", state, colony.Phase{ID: 1}, []discussQuestion{{
		Category: "scope",
		Question: "Which boundary should be protected?",
		Options:  []string{"scope"},
	}})
	if err == nil {
		t.Fatalf("materializeOrchestratorBoundaryQuestions returned nil error with no store")
	}
	if !strings.Contains(err.Error(), "no store initialized") {
		t.Fatalf("error = %v, want no store initialized", err)
	}
}

func assertBoundaryQuestionsCreated(t *testing.T, dataDir string, result map[string]interface{}, wantSource string) {
	t.Helper()
	if got := intValue(result["boundary_question_count"]); got != 1 {
		t.Fatalf("boundary_question_count = %d, want 1", got)
	}
	if got := intValue(result["boundary_questions_created"]); got != 1 {
		t.Fatalf("boundary_questions_created = %d, want 1", got)
	}
	if got := intValue(result["boundary_questions_existing"]); got != 0 {
		t.Fatalf("boundary_questions_existing = %d, want 0", got)
	}
	questions, ok := result["boundary_questions"].([]discussQuestion)
	if !ok || len(questions) != 1 {
		t.Fatalf("boundary_questions = %#v, want one []discussQuestion", result["boundary_questions"])
	}
	if questions[0].Source != wantSource {
		t.Fatalf("boundary question source = %q, want %q", questions[0].Source, wantSource)
	}

	var file PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
		t.Fatalf("load pending decisions: %v", err)
	}
	if len(file.Decisions) != 1 {
		t.Fatalf("pending decisions = %d, want 1", len(file.Decisions))
	}
	if file.Decisions[0].Source != wantSource {
		t.Fatalf("pending source = %q, want %q", file.Decisions[0].Source, wantSource)
	}
	if file.Decisions[0].Type != clarificationDecisionType {
		t.Fatalf("pending type = %q, want %q", file.Decisions[0].Type, clarificationDecisionType)
	}
	if _, err := os.Stat(filepath.Join(dataDir, pendingDecisionsFile)); err != nil {
		t.Fatalf("expected pending decisions file: %v", err)
	}
}

func assertNoBoundaryQuestions(t *testing.T, dataDir string, result map[string]interface{}) {
	t.Helper()
	if got := intValue(result["boundary_question_count"]); got != 0 {
		t.Fatalf("boundary_question_count = %d, want 0", got)
	}
	if got := intValue(result["boundary_questions_created"]); got != 0 {
		t.Fatalf("boundary_questions_created = %d, want 0", got)
	}
	if got := intValue(result["boundary_questions_existing"]); got != 0 {
		t.Fatalf("boundary_questions_existing = %d, want 0", got)
	}
	if questions, ok := result["boundary_questions"].([]discussQuestion); !ok || len(questions) != 0 {
		t.Fatalf("boundary_questions = %#v, want empty []discussQuestion", result["boundary_questions"])
	}
	if _, err := os.Stat(filepath.Join(dataDir, pendingDecisionsFile)); !os.IsNotExist(err) {
		t.Fatalf("default mode should not create %s, stat err=%v", pendingDecisionsFile, err)
	}
}
