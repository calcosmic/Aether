package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestValidateExternalPlanStateSuggestsStaleCleanupForFreshManifest(t *testing.T) {
	saveGlobals(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "Build a fresh dashboard"
	taskID := "1.1"
	createTestColonyState(t, s.BasePath(), colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:     1,
				Name:   "Old stale phase",
				Status: colony.PhaseReady,
				Tasks: []colony.Task{{
					ID:     &taskID,
					Goal:   "Old task",
					Status: colony.TaskPending,
				}},
			}},
		},
	})

	manifest := &codexPlanManifest{
		Goal:         goal,
		Root:         tmpDir,
		Granularity:  "milestone",
		ExistingPlan: false,
		Refresh:      false,
	}

	_, _, err := validateExternalPlanState(manifest)
	if err == nil {
		t.Fatal("expected error when colony has stale phases but manifest says fresh plan")
	}
	errMsg := err.Error()
	// The error should mention stale state to help the user understand
	// that existing phases from a prior session are blocking finalization.
	if !strings.Contains(errMsg, "stale") {
		t.Fatalf("error should mention stale state for fresh manifest with existing phases, got: %s", errMsg)
	}
}

// TestValidateExternalPlanStateAllowsExistingPlanWhenManifestAcknowledges verifies
// that plan-finalize succeeds when manifest.ExistingPlan is true and phases exist.
// FIX C regression: the guard was over-rejecting by not checking manifest.ExistingPlan.
func TestValidateExternalPlanStateAllowsExistingPlanWhenManifestAcknowledges(t *testing.T) {
	saveGlobals(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "Build a dashboard"
	taskID := "1.1"
	createTestColonyState(t, s.BasePath(), colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:     1,
				Name:   "Foundation phase",
				Status: colony.PhaseReady,
				Tasks: []colony.Task{{
					ID:     &taskID,
					Goal:   "Set up foundation",
					Status: colony.TaskPending,
				}},
			}},
		},
	})

	manifest := &codexPlanManifest{
		Goal:         goal,
		Root:         tmpDir,
		Granularity:  "milestone",
		ExistingPlan: true,
		Refresh:      false,
	}

	_, _, err := validateExternalPlanState(manifest)
	if err != nil {
		t.Fatalf("expected no error when manifest.ExistingPlan is true, got: %s", err)
	}
}

// TestEffectiveContinueReviewTimeoutDefaultsTo10Minutes verifies that the
// default review timeout for continue workers is 10 minutes (increased from 5m).
// FIX B regression: review worker timeout was too short for complex verification.
func TestEffectiveContinueReviewTimeoutDefaultsTo10Minutes(t *testing.T) {
	got := effectiveContinueReviewTimeout(0)
	want := 10 * time.Minute
	if got != want {
		t.Fatalf("effectiveContinueReviewTimeout(0) = %v, want %v", got, want)
	}
}

// TestEffectiveContinueReviewTimeoutHonorsOverride verifies that an explicit
// override is respected even when the default is increased.
func TestEffectiveContinueReviewTimeoutHonorsOverride(t *testing.T) {
	override := 3 * time.Minute
	got := effectiveContinueReviewTimeout(override)
	if got != override {
		t.Fatalf("effectiveContinueReviewTimeout(%v) = %v, want %v", override, got, override)
	}
}

func TestPlanFinalizeRejectsStaleManifestBeforeStateMutation(t *testing.T) {
	saveGlobals(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/stale-plan\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	goal := "Finalize only fresh planning workers"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	survey, err := loadCodexSurveyContext(root)
	if err != nil {
		t.Fatalf("load survey context: %v", err)
	}
	dispatches := testPlanningDispatches()
	completion := codexExternalPlanCompletion{
		PlanManifest: testPlanManifest(root, goal, time.Now().UTC().Add(-25*time.Hour), survey, dispatches),
		Dispatches:   testCompletedPlanningResults(dispatches),
	}

	_, err = runCodexPlanFinalize(root, completion)
	if err == nil || !strings.Contains(err.Error(), "stale plan_manifest") {
		t.Fatalf("expected stale manifest error, got %v", err)
	}
	assertPlanFinalizeStateUnchanged(t, 0)
}

func TestPlanFinalizeRejectsFutureManifestBeforeStateMutation(t *testing.T) {
	saveGlobals(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/future-plan\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	goal := "Reject future-dated planning workers"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	survey, err := loadCodexSurveyContext(root)
	if err != nil {
		t.Fatalf("load survey context: %v", err)
	}
	dispatches := testPlanningDispatches()
	completion := codexExternalPlanCompletion{
		PlanManifest: testPlanManifest(root, goal, time.Now().UTC().Add(10*time.Minute), survey, dispatches),
		Dispatches:   testCompletedPlanningResults(dispatches),
	}

	_, err = runCodexPlanFinalize(root, completion)
	if err == nil || !strings.Contains(err.Error(), "too far in the future") {
		t.Fatalf("expected future manifest error, got %v", err)
	}
	assertPlanFinalizeStateUnchanged(t, 0)
}

func TestPlanFinalizeRejectsWorkspaceDriftBeforeStateMutation(t *testing.T) {
	saveGlobals(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/plan-drift\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	goal := "Reject stale planning workspace context"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	survey, err := loadCodexSurveyContext(root)
	if err != nil {
		t.Fatalf("load survey context: %v", err)
	}
	dispatches := testPlanningDispatches()
	completion := codexExternalPlanCompletion{
		PlanManifest: testPlanManifest(root, goal, time.Now().UTC(), survey, dispatches),
		Dispatches:   testCompletedPlanningResults(dispatches),
	}
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"scripts":{"test":"node --test"}}`+"\n"), 0644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	_, err = runCodexPlanFinalize(root, completion)
	if err == nil || !strings.Contains(err.Error(), "plan_manifest workspace changed") {
		t.Fatalf("expected workspace drift error, got %v", err)
	}
	assertPlanFinalizeStateUnchanged(t, 0)
}

func TestPlanFinalizeUsesRouteSetterEvidenceWithDynamicWorkers(t *testing.T) {
	saveGlobals(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/dynamic-plan\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	goal := "Preserve route-setter task evidence"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})
	survey, err := loadCodexSurveyContext(root)
	if err != nil {
		t.Fatalf("load survey context: %v", err)
	}

	dispatches := []codexPlanningDispatch{
		{Stage: "scouting", Wave: 1, Caste: "scout", Name: "Scout-1", Task: "Summarize planning context", TaskID: "plan-scout", Outputs: []string{"SCOUT.md"}},
		{Stage: "architecture", Wave: 2, Caste: "architect", Name: "Arch-1", Task: "Identify structural risks", TaskID: "plan-architect", Outputs: []string{"ARCHITECT.md"}},
		{Stage: "routing", Wave: 3, Caste: "route_setter", Name: "Route-1", Task: "Create constrained phase plan", TaskID: "plan-route-setter", Outputs: []string{"ROUTE-SETTER.md", "phase-plan.json"}},
	}
	results := testCompletedPlanningResults(dispatches)
	results[2].PhasePlan = testWorkerPlanArtifact()

	result, err := runCodexPlanFinalize(root, codexExternalPlanCompletion{
		PlanManifest: testPlanManifest(root, goal, time.Now().UTC(), survey, dispatches),
		Dispatches:   results,
	})
	if err != nil {
		t.Fatalf("runCodexPlanFinalize: %v", err)
	}
	if got := len(result["dispatches"].([]map[string]interface{})); got != 3 {
		t.Fatalf("dispatch count = %d, want 3", got)
	}

	routeSetterData, err := os.ReadFile(filepath.Join(dataDir, "planning", "ROUTE-SETTER.md"))
	if err != nil {
		t.Fatalf("read route-setter artifact: %v", err)
	}
	if !strings.Contains(string(routeSetterData), "Route-Setter: Route-1") {
		t.Fatalf("route-setter artifact did not use route-setter dispatch:\n%s", string(routeSetterData))
	}
	if strings.Contains(string(routeSetterData), "Route-Setter: Arch-1") {
		t.Fatalf("route-setter artifact used non-route-setter dispatch:\n%s", string(routeSetterData))
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if len(state.Plan.Phases) != 1 || len(state.Plan.Phases[0].Tasks) != 1 {
		t.Fatalf("unexpected saved phases: %+v", state.Plan.Phases)
	}
	task := state.Plan.Phases[0].Tasks[0]
	for _, want := range []string{"Keep route-setter evidence authoritative", "cmd/codex_plan_finalize.go", "Constraints survive finalization"} {
		if !stringSliceContains(append(append([]string{}, task.Constraints...), append(task.Hints, task.SuccessCriteria...)...), want) {
			t.Fatalf("saved task evidence missing %q: %+v", want, task)
		}
	}
}

func TestPlanFinalizeRejectsStaleClaimedPhasePlanArtifact(t *testing.T) {
	saveGlobals(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/stale-claimed-plan\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	goal := "Reject stale claimed route-setter artifacts"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	planningDir := filepath.Join(root, ".aether", "data", "planning")
	if err := os.MkdirAll(planningDir, 0755); err != nil {
		t.Fatalf("create planning dir: %v", err)
	}
	stalePlanPath := filepath.Join(planningDir, "phase-plan.json")
	stalePlan := `{"phases":[{"name":"Old route","description":"Old plan","tasks":[{"goal":"Old task"}]}],"confidence":{"overall":80}}` + "\n"
	if err := os.WriteFile(stalePlanPath, []byte(stalePlan), 0644); err != nil {
		t.Fatalf("write stale phase plan: %v", err)
	}

	survey, err := loadCodexSurveyContext(root)
	if err != nil {
		t.Fatalf("load survey context: %v", err)
	}
	dispatches := testPlanningDispatches()
	manifest := testPlanManifest(root, goal, time.Now().UTC(), survey, dispatches)
	manifest.Snapshots = snapshotRelativeFiles(root, filepath.ToSlash(filepath.Join(".aether", "data", "planning")))
	if err := os.Chtimes(stalePlanPath, time.Now().UTC().Add(time.Minute), time.Now().UTC().Add(time.Minute)); err != nil {
		t.Fatalf("touch stale phase plan: %v", err)
	}
	results := testCompletedPlanningResults(dispatches)
	for i := range results {
		if results[i].Caste == "route_setter" {
			results[i].PhasePlan = nil
			results[i].FilesCreated = []string{filepath.ToSlash(filepath.Join(".aether", "data", "planning", "phase-plan.json"))}
		}
	}

	_, err = runCodexPlanFinalize(root, codexExternalPlanCompletion{
		PlanManifest: manifest,
		Dispatches:   results,
	})
	if err == nil || !strings.Contains(err.Error(), "already present when plan_manifest was generated") {
		t.Fatalf("expected stale claimed artifact error, got %v", err)
	}
	assertPlanFinalizeStateUnchanged(t, 0)
}

func TestPlanFinalizeRejectsInvalidClaimedPhasePlanFiles(t *testing.T) {
	tests := []struct {
		name      string
		setupFile func(t *testing.T, path string)
		want      string
	}{
		{
			name: "missing",
			setupFile: func(t *testing.T, path string) {
				t.Helper()
			},
			want: "is missing",
		},
		{
			name: "directory",
			setupFile: func(t *testing.T, path string) {
				t.Helper()
				if err := os.MkdirAll(path, 0755); err != nil {
					t.Fatalf("create claimed phase-plan directory: %v", err)
				}
			},
			want: "is not a regular file",
		},
		{
			name: "symlink",
			setupFile: func(t *testing.T, path string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatalf("create planning dir: %v", err)
				}
				target := filepath.Join(filepath.Dir(path), "target-phase-plan.json")
				if err := os.WriteFile(target, []byte(`{"phases":[{"name":"target","tasks":[{"goal":"target"}]}]}`+"\n"), 0644); err != nil {
					t.Fatalf("write symlink target: %v", err)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Skipf("symlink unsupported in this environment: %v", err)
				}
			},
			want: "is not a regular file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saveGlobals(t)

			dataDir := setupBuildFlowTest(t)
			root := filepath.Dir(filepath.Dir(dataDir))
			withWorkingDir(t, root)
			if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/invalid-claimed-plan\n\ngo 1.24\n"), 0644); err != nil {
				t.Fatalf("write go.mod: %v", err)
			}

			goal := "Reject invalid claimed phase plan files"
			createTestColonyState(t, dataDir, colony.ColonyState{
				Version: "3.0",
				Goal:    &goal,
				State:   colony.StateREADY,
				Plan:    colony.Plan{Phases: []colony.Phase{}},
			})

			relPath := filepath.ToSlash(filepath.Join(".aether", "data", "planning", "phase-plan.json"))
			tt.setupFile(t, filepath.Join(root, filepath.FromSlash(relPath)))
			survey, err := loadCodexSurveyContext(root)
			if err != nil {
				t.Fatalf("load survey context: %v", err)
			}
			dispatches := testPlanningDispatches()
			results := testCompletedPlanningResults(dispatches)
			for i := range results {
				if results[i].Caste == "route_setter" {
					results[i].PhasePlan = nil
					results[i].FilesCreated = []string{relPath}
				}
			}

			_, err = runCodexPlanFinalize(root, codexExternalPlanCompletion{
				PlanManifest: testPlanManifest(root, goal, time.Now().UTC(), survey, dispatches),
				Dispatches:   results,
			})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected claimed artifact error containing %q, got %v", tt.want, err)
			}
			assertPlanFinalizeStateUnchanged(t, 0)
		})
	}
}

func TestPlanOnlyDoesNotPersistVerificationDepth(t *testing.T) {
	saveGlobals(t)
	setupRuntimeSkillAssignmentHub(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/plan-only-depth\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	goal := "Plan without mutating verification settings"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	result, err := runCodexPlanWithOptions(root, codexPlanOptions{PlanOnly: true, VerificationDepth: "heavy"})
	if err != nil {
		t.Fatalf("runCodexPlanWithOptions: %v", err)
	}
	if got := result["verification_depth"]; got != string(colony.VerificationDepthHeavy) {
		t.Fatalf("verification_depth result = %v, want heavy", got)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.VerificationDepth != "" {
		t.Fatalf("plan-only persisted verification_depth = %q, want empty", state.VerificationDepth)
	}
}

func TestSyntheticPlanAddsActiveRedirectsToTaskConstraints(t *testing.T) {
	saveGlobals(t)

	s, _ := newTestStore(t)
	store = s
	strength := 1.0
	signal := colony.PheromoneFile{Signals: []colony.PheromoneSignal{{
		ID:        "sig-redirect-plan",
		Type:      "REDIRECT",
		Priority:  "high",
		Source:    "user",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Active:    true,
		Strength:  &strength,
		Content:   json.RawMessage(`{"text":"Do not touch legacy wrappers"}`),
	}}}
	if err := store.SaveJSON("pheromones.json", signal); err != nil {
		t.Fatalf("save pheromones: %v", err)
	}

	redirect := "Do not touch legacy wrappers"
	phases, _, _ := synthesizeRouteSetterPlan("Stabilize multi-platform lifecycle reliability", colony.GranularitySprint, codexSurveyContext{}, codexScoutReport{})
	for _, phase := range phases {
		for _, task := range phase.Tasks {
			if !stringSliceContains(task.Constraints, redirect) {
				t.Fatalf("active REDIRECT signal missing from task %q constraints: %+v", task.Goal, task.Constraints)
			}
		}
	}
}

func testPlanningDispatches() []codexPlanningDispatch {
	return []codexPlanningDispatch{
		{Stage: "scouting", Wave: 1, Caste: "scout", Name: "Scout-1", Task: "Summarize planning context", TaskID: "plan-scout", Outputs: []string{"SCOUT.md"}},
		{Stage: "routing", Wave: 2, Caste: "route_setter", Name: "Route-1", Task: "Create constrained phase plan", TaskID: "plan-route-setter", Outputs: []string{"ROUTE-SETTER.md", "phase-plan.json"}},
	}
}

func testPlanManifest(root, goal string, generatedAt time.Time, survey codexSurveyContext, dispatches []codexPlanningDispatch) *codexPlanManifest {
	return &codexPlanManifest{
		Goal:              goal,
		Root:              root,
		GeneratedAt:       generatedAt.Format(time.RFC3339),
		ColonyMode:        "colony",
		Depth:             "fast",
		Granularity:       string(colony.GranularitySprint),
		GranularityMin:    1,
		GranularityMax:    2,
		PlanningDepth:     "standard",
		VerificationDepth: "standard",
		Survey:            survey,
		Dispatches:        dispatches,
		DispatchMode:      "plan-only",
		DispatchContract:  planningDispatchContract(),
		FinalizeSurface:   "pending",
		RequiresFinalizer: true,
	}
}

func testCompletedPlanningResults(dispatches []codexPlanningDispatch) []codexPlanningDispatch {
	results := make([]codexPlanningDispatch, 0, len(dispatches))
	for _, dispatch := range dispatches {
		dispatch.Status = "completed"
		dispatch.Summary = dispatch.Name + " completed"
		if dispatch.Caste == "route_setter" {
			dispatch.PhasePlan = testWorkerPlanArtifact()
		}
		results = append(results, dispatch)
	}
	return results
}

func testWorkerPlanArtifact() *codexWorkerPlanArtifact {
	return &codexWorkerPlanArtifact{
		Phases: []codexWorkerPlanPhase{{
			Name:        "Route-setter evidence phase",
			Description: "Preserve route-setter task fields.",
			Tasks: []codexWorkerPlanTask{{
				Goal:            "Save constrained task evidence",
				Constraints:     []string{"Keep route-setter evidence authoritative"},
				Hints:           []string{"cmd/codex_plan_finalize.go"},
				SuccessCriteria: []string{"Constraints survive finalization"},
			}},
			SuccessCriteria: []string{"Route-setter plan is committed"},
		}},
		Confidence: codexPlanConfidence{Knowledge: 90, Requirements: 90, Risks: 80, Dependencies: 80, Effort: 80, Overall: 84},
	}
}

func assertPlanFinalizeStateUnchanged(t *testing.T, wantPhases int) {
	t.Helper()
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if len(state.Plan.Phases) != wantPhases || state.CurrentPhase != 0 || state.Plan.GeneratedAt != nil {
		t.Fatalf("plan-finalize mutated state before validation: %+v", state)
	}
}
