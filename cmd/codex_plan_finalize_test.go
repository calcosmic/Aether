package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestCodexPlanFinalizeScoutExposesRouteBoundary(t *testing.T) {
	root, stageManifest, scoutResult := planningScoutStageTestFixture(t)
	dispatch := codexPlanningDispatch{
		Stage:         "scouting",
		Caste:         string(planningStageCasteScout),
		Name:          "Scout-200",
		Task:          "Complete one exact Scout stage",
		Outputs:       []string{"scout-result.json"},
		StageManifest: &stageManifest,
	}
	manifest := codexPlanManifest{
		Root:              root,
		GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
		PlanningRunID:     stageManifest.RunID,
		Iteration:         stageManifest.Pass,
		SelectedPreset:    stageManifest.Preset,
		BaseRevisionID:    stageManifest.BasePlanRevisionID,
		BasePlanStateHash: stageManifest.BasePlanRevisionHash,
		ExpectedWorkers:   []codexPlanningDispatch{dispatch},
		Dispatches:        []codexPlanningDispatch{dispatch},
		DispatchMode:      "plan-only",
		RequiresFinalizer: true,
		StageManifest:     &stageManifest,
	}

	result, err := runCodexScoutStageFinalize(root, manifest, codexExternalPlanCompletion{
		PlanManifest: &manifest,
		ScoutResult:  planningScoutStageTestBytes(t, scoutResult),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result["status"] != string(planningStageRouteRunning) || result["next_boundary"] != string(planningStageRouteRunning) {
		t.Fatalf("Scout finalizer boundary = %#v, want visible route_running", result)
	}
	if result["route_stage_manifest"] == nil || result["stage_receipt"] == nil {
		t.Fatalf("Scout finalizer omitted exact receipt or Route-Setter manifest: %#v", result)
	}
	if result["iteration_card_created"] != false || result["scout_complete"] != true {
		t.Fatalf("Scout finalizer rendered an early card or hid Scout completion: %#v", result)
	}
}

func TestCodexPlanFinalizeScoutDecisionResumeExposesExactRouteBoundary(t *testing.T) {
	root, stageManifest, scoutResult := planningScoutStageTestFixture(t)
	scoutResult.DecisionCandidates = []planningDecisionCandidate{
		planningScoutStageMaterialCandidate(scoutResult.NewEvidence[0].Reference, "decision-finalizer-resume"),
	}
	dispatch := codexPlanningDispatch{Caste: string(planningStageCasteScout), Name: "Scout-200", Task: "Complete Scout", Outputs: []string{"scout-result.json"}, StageManifest: &stageManifest}
	manifest := codexPlanManifest{
		Root:              root,
		GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
		PlanningRunID:     stageManifest.RunID,
		Iteration:         stageManifest.Pass,
		SelectedPreset:    stageManifest.Preset,
		BaseRevisionID:    stageManifest.BasePlanRevisionID,
		BasePlanStateHash: stageManifest.BasePlanRevisionHash,
		ExpectedWorkers:   []codexPlanningDispatch{dispatch},
		Dispatches:        []codexPlanningDispatch{dispatch},
		DispatchMode:      "plan-only",
		RequiresFinalizer: true,
		StageManifest:     &stageManifest,
	}
	completion := codexExternalPlanCompletion{PlanManifest: &manifest, ScoutResult: planningScoutStageTestBytes(t, scoutResult)}
	first, err := runCodexScoutStageFinalize(root, manifest, completion)
	if err != nil {
		t.Fatal(err)
	}
	checkpoint, ok := first["decision_checkpoint"].(*planningScoutDecisionCheckpoint)
	if !ok || checkpoint == nil || first["status"] != string(planningStageOwnerDecision) || first["route_stage_manifest"] != nil {
		t.Fatalf("first-pass decision finalizer = %#v, want one owner checkpoint and no route", first)
	}
	card := checkpoint.Cards[0]
	resume, err := buildPlanningScoutDecisionResumeToken(*checkpoint, []planningScoutDecisionAnswer{{DecisionID: card.DecisionID, ChoiceID: card.Choices[0].ID, Answer: card.Choices[0].Label}})
	if err != nil {
		t.Fatal(err)
	}
	completion.DecisionResume = &resume
	completion.DecisionResolvedAt = time.Date(2026, time.September, 7, 19, 15, 0, 0, time.UTC)
	second, err := runCodexScoutStageFinalize(root, manifest, completion)
	if err != nil {
		t.Fatal(err)
	}
	if second["status"] != string(planningStageRouteRunning) || second["route_stage_manifest"] == nil || second["completed_decision_resume_token"] == nil {
		t.Fatalf("completed decision finalizer = %#v, want exact Route-Setter resume", second)
	}
}

func TestCodexPlanFinalizeRouteExposesNextScoutBoundary(t *testing.T) {
	root, stageManifest, routeResult := planningRouteStageTestFixture(t)
	dispatch := codexPlanningDispatch{
		Stage: "routing", Caste: string(planningStageCasteRouteSetter), Name: "Route-Setter-200",
		Task: "Complete one exact Route-Setter stage", Outputs: []string{"route-result.json"}, StageManifest: &stageManifest,
	}
	manifest := codexPlanManifest{
		Root: root, GeneratedAt: time.Now().UTC().Format(time.RFC3339), PlanningRunID: stageManifest.RunID,
		Iteration: stageManifest.Pass, SelectedPreset: stageManifest.Preset,
		BaseRevisionID: stageManifest.BasePlanRevisionID, BasePlanStateHash: stageManifest.BasePlanRevisionHash,
		ExpectedWorkers: []codexPlanningDispatch{dispatch}, Dispatches: []codexPlanningDispatch{dispatch},
		DispatchMode: "plan-only", RequiresFinalizer: true, StageManifest: &stageManifest,
	}
	result, err := runCodexRouteStageFinalize(root, manifest, codexExternalPlanCompletion{
		PlanManifest: &manifest, RouteResult: planningRouteStageTestBytes(t, routeResult),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result["status"] != string(planningStageScoutRunning) || result["next_boundary"] != string(planningStageScoutRunning) || result["scout_stage_manifest"] == nil {
		t.Fatalf("Route finalizer boundary = %#v, want visible next Scout dispatch", result)
	}
	if result["iteration_card"] == nil || result["route_stage_receipt"] == nil || result["proposal_hash"] == "" {
		t.Fatalf("Route finalizer omitted its card, receipt, or proposal hash: %#v", result)
	}
}

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
	bindTestPlanManifestToCurrentState(manifest)

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
	bindTestPlanManifestToCurrentState(manifest)

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
	stateBefore := readPlanFinalizeStateBytes(t)
	completion := codexExternalPlanCompletion{
		PlanManifest: testPlanManifest(root, goal, time.Now().UTC().Add(-25*time.Hour), survey, dispatches),
		Dispatches:   testCompletedPlanningResults(dispatches),
	}

	_, err = runCodexPlanFinalize(root, completion)
	if err == nil || !strings.Contains(err.Error(), "stale plan_manifest") {
		t.Fatalf("expected stale manifest error, got %v", err)
	}
	assertPlanFinalizeStateBytesUnchanged(t, stateBefore)
	assertPlanFinalizeStateUnchanged(t, 0)
}

func TestPlanRevisionFinalizeRejectsChangedBaseWithoutStateMutation(t *testing.T) {
	saveGlobals(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/stale-revision\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	evidencePath := filepath.Join(root, ".aether", "oracle", "synthesis.md")
	if err := os.MkdirAll(filepath.Dir(evidencePath), 0755); err != nil {
		t.Fatalf("create Oracle evidence directory: %v", err)
	}
	if err := os.WriteFile(evidencePath, []byte("The original assumption is false.\n"), 0644); err != nil {
		t.Fatalf("write Oracle evidence: %v", err)
	}

	goal := "Reject a plan revision after its base changes"
	doneTaskID := "1.1"
	futureTaskID := "2.1"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 2,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Accepted work", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &doneTaskID, Goal: "Keep accepted work", Status: colony.TaskCompleted}}},
			{ID: 2, Name: "Old future", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: &futureTaskID, Goal: "Replace this work", Status: colony.TaskPending}}},
		}},
	}
	createTestColonyState(t, dataDir, state)
	survey, err := loadCodexSurveyContext(root)
	if err != nil {
		t.Fatalf("load survey context: %v", err)
	}
	dispatches := testPlanningDispatches()
	manifest := testPlanManifest(root, goal, time.Now().UTC(), survey, dispatches)
	manifest.ExistingPlan = true
	manifest.Refresh = true
	var baseState colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &baseState); err != nil {
		t.Fatalf("load revision base state: %v", err)
	}
	baseState = normalizeLegacyColonyState(baseState)
	manifest.Revision, err = buildPlanRevisionContext(root, baseState, codexPlanOptions{
		Refresh:          true,
		RevisionType:     string(colony.PlanRevisionResearch),
		RevisionReason:   "Oracle disproved the future approach",
		RevisionEvidence: []string{".aether/oracle/synthesis.md"},
	})
	if err != nil {
		t.Fatalf("build revision context: %v", err)
	}
	bindTestPlanManifestToCurrentState(manifest)

	state.Plan.Phases[1].Description = "A newer session already changed this plan"
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save changed plan state: %v", err)
	}
	stateBefore := readPlanFinalizeStateBytes(t)

	_, err = runCodexPlanFinalize(root, codexExternalPlanCompletion{
		PlanManifest: manifest,
		Dispatches:   testCompletedPlanningResults(dispatches),
	})
	assertPlanFinalizeErrorContains(t, err, "plan state changed after this planning packet")
	assertPlanFinalizeStateBytesUnchanged(t, stateBefore)
	assertPlanFinalizeNoSpawnRuns(t)
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

func TestPlanFinalizeRejectsRootMismatchBeforeStateMutation(t *testing.T) {
	saveGlobals(t)

	root, survey, dispatches := setupPlanFinalizeFailureFixture(t, "Reject mismatched planning roots")
	manifest := testPlanManifest(root, "Reject mismatched planning roots", time.Now().UTC(), survey, dispatches)
	manifest.Root = filepath.Join(root, "other-workspace")
	stateBefore := readPlanFinalizeStateBytes(t)

	_, err := runCodexPlanFinalize(root, codexExternalPlanCompletion{
		PlanManifest: manifest,
		Dispatches:   testCompletedPlanningResults(dispatches),
	})
	assertPlanFinalizeErrorContains(t, err, "root does not match")
	assertPlanFinalizeStateBytesUnchanged(t, stateBefore)
	assertPlanFinalizeStateUnchanged(t, 0)
}

func TestPlanFinalizeRejectsMissingPhasePlanBeforeStateMutation(t *testing.T) {
	saveGlobals(t)

	goal := "Reject missing route-setter phase plans"
	root, survey, dispatches := setupPlanFinalizeFailureFixture(t, goal)
	results := testCompletedPlanningResults(dispatches)
	for i := range results {
		if results[i].Caste == "route_setter" {
			results[i].PhasePlan = nil
		}
	}
	stateBefore := readPlanFinalizeStateBytes(t)

	_, err := runCodexPlanFinalize(root, codexExternalPlanCompletion{
		PlanManifest: testPlanManifest(root, goal, time.Now().UTC(), survey, dispatches),
		Dispatches:   results,
	})
	assertPlanFinalizeErrorContains(t, err, "route-setter phase_plan")
	assertPlanFinalizeStateBytesUnchanged(t, stateBefore)
	assertPlanFinalizeStateUnchanged(t, 0)
}

func TestPlanFinalizeRejectsMissingScoutEvidenceBeforeStateMutation(t *testing.T) {
	saveGlobals(t)

	goal := "Reject missing Scout evidence"
	root, survey, dispatches := setupPlanFinalizeFailureFixture(t, goal)
	results := testCompletedPlanningResults(dispatches)
	for i := range results {
		if results[i].Caste == "scout" {
			results[i].ScoutReport = nil
		}
	}
	stateBefore := readPlanFinalizeStateBytes(t)

	_, err := runCodexPlanFinalize(root, codexExternalPlanCompletion{
		PlanManifest: testPlanManifest(root, goal, time.Now().UTC(), survey, dispatches),
		Dispatches:   results,
	})
	assertPlanFinalizeErrorContains(t, err, "Scout evidence")
	assertPlanFinalizeStateBytesUnchanged(t, stateBefore)
	assertPlanFinalizeStateUnchanged(t, 0)
}

func TestPlanFinalizeRejectsEmptyPhasePlanBeforeStateMutation(t *testing.T) {
	saveGlobals(t)

	goal := "Reject empty route-setter phase plans"
	root, survey, dispatches := setupPlanFinalizeFailureFixture(t, goal)
	results := testCompletedPlanningResults(dispatches)
	for i := range results {
		if results[i].Caste == "route_setter" {
			results[i].PhasePlan = &codexWorkerPlanArtifact{
				Phases:     []codexWorkerPlanPhase{},
				Confidence: testWorkerPlanArtifact().Confidence,
			}
		}
	}
	stateBefore := readPlanFinalizeStateBytes(t)

	_, err := runCodexPlanFinalize(root, codexExternalPlanCompletion{
		PlanManifest: testPlanManifest(root, goal, time.Now().UTC(), survey, dispatches),
		Dispatches:   results,
	})
	assertPlanFinalizeErrorContains(t, err, "contains no phases")
	assertPlanFinalizeStateBytesUnchanged(t, stateBefore)
	assertPlanFinalizeStateUnchanged(t, 0)
}

func TestPlanFinalizeRejectsPhasePlanWithoutBuildableTasksBeforeStateMutation(t *testing.T) {
	saveGlobals(t)

	goal := "Reject plans without buildable tasks"
	root, survey, dispatches := setupPlanFinalizeFailureFixture(t, goal)
	results := testCompletedPlanningResults(dispatches)
	for i := range results {
		if results[i].Caste == "route_setter" {
			results[i].PhasePlan = &codexWorkerPlanArtifact{
				Phases: []codexWorkerPlanPhase{{
					Name:        "No buildable work",
					Description: "A phase without concrete tasks must not advance state.",
				}},
				Confidence: testWorkerPlanArtifact().Confidence,
			}
		}
	}
	stateBefore := readPlanFinalizeStateBytes(t)

	_, err := runCodexPlanFinalize(root, codexExternalPlanCompletion{
		PlanManifest: testPlanManifest(root, goal, time.Now().UTC(), survey, dispatches),
		Dispatches:   results,
	})
	assertPlanFinalizeErrorContains(t, err, "no buildable")
	assertPlanFinalizeStateBytesUnchanged(t, stateBefore)
	assertPlanFinalizeStateUnchanged(t, 0)
	assertPlanFinalizeNoSpawnRuns(t)
}

func TestPlanFinalizeRejectsTextDependsOnBeforeStateMutation(t *testing.T) {
	saveGlobals(t)

	goal := "Reject prose dependency references"
	root, survey, dispatches := setupPlanFinalizeFailureFixture(t, goal)
	results := testCompletedPlanningResults(dispatches)
	for i := range results {
		if results[i].Caste == "route_setter" {
			results[i].PhasePlan = &codexWorkerPlanArtifact{
				Phases: []codexWorkerPlanPhase{{
					Name: "Dependency contract",
					Tasks: []codexWorkerPlanTask{
						{Goal: "Create the source task"},
						{Goal: "Run the dependent task", DependsOn: []string{"Create the source task"}},
					},
				}},
				Confidence: testWorkerPlanArtifact().Confidence,
			}
		}
	}
	stateBefore := readPlanFinalizeStateBytes(t)

	_, err := runCodexPlanFinalize(root, codexExternalPlanCompletion{
		PlanManifest: testPlanManifest(root, goal, time.Now().UTC(), survey, dispatches),
		Dispatches:   results,
	})
	assertPlanFinalizeErrorContains(t, err, "not a valid task id")
	assertPlanFinalizeErrorContains(t, err, "Create the source task")
	assertPlanFinalizeStateBytesUnchanged(t, stateBefore)
	assertPlanFinalizeNoSpawnRuns(t)
}

func TestPlanFinalizeNormalizesStableCustomDependencyReferences(t *testing.T) {
	saveGlobals(t)

	goal := "Normalize custom dependency ids"
	root, survey, dispatches := setupPlanFinalizeFailureFixture(t, goal)
	results := testCompletedPlanningResults(dispatches)
	for i := range results {
		if results[i].Caste == "route_setter" {
			results[i].PhasePlan = &codexWorkerPlanArtifact{
				Phases: []codexWorkerPlanPhase{{
					Name:            "Dependency contract",
					SuccessCriteria: []string{"Dependency ids are normalized"},
					EvidenceRequirements: []colony.CriterionEvidenceRequirement{{
						Criterion: "Dependency ids are normalized",
						Checks:    []string{"claims", "watcher"},
					}},
					Tasks: []codexWorkerPlanTask{
						{
							Goal:            "Create the source task",
							SuccessCriteria: []string{"Source task exists"},
							EvidenceRequirements: []colony.CriterionEvidenceRequirement{{
								Criterion: "Source task exists", Checks: []string{"claims", "watcher"},
							}},
						},
						{
							Goal:            "Run the dependent task",
							DependsOn:       []string{"P1-T1"},
							SuccessCriteria: []string{"Dependent task runs after source"},
							EvidenceRequirements: []colony.CriterionEvidenceRequirement{{
								Criterion: "Dependent task runs after source", Checks: []string{"claims", "watcher"},
							}},
						},
					},
				}},
				Confidence: testWorkerPlanArtifact().Confidence,
			}
		}
	}

	result, err := runCodexPlanFinalize(root, codexExternalPlanCompletion{
		PlanManifest: testPlanManifest(root, goal, time.Now().UTC(), survey, dispatches),
		Dispatches:   results,
	})
	if err != nil {
		t.Fatalf("runCodexPlanFinalize: %v", err)
	}
	phases := result["phases"].([]colony.Phase)
	got := phases[0].Tasks[1].DependsOn
	if len(got) != 1 || got[0] != "1.1" {
		t.Fatalf("depends_on = %v, want [1.1]", got)
	}
	if warning := stringValue(result["planning_warning"]); !strings.Contains(warning, "P1-T1") || !strings.Contains(warning, "1.1") {
		t.Fatalf("planning_warning = %q, want normalized dependency note", warning)
	}
}

func TestPlanFinalizeRejectsAmbiguousTopLevelPhasePlanBeforeStateMutation(t *testing.T) {
	saveGlobals(t)

	goal := "Reject ambiguous synthetic planning claims"
	root, survey, dispatches := setupPlanFinalizeFailureFixture(t, goal)
	results := testCompletedPlanningResults(dispatches)
	for i := range results {
		if results[i].Caste == "route_setter" {
			results[i].PhasePlan = nil
		}
	}
	stateBefore := readPlanFinalizeStateBytes(t)

	_, err := runCodexPlanFinalize(root, codexExternalPlanCompletion{
		PlanManifest: testPlanManifest(root, goal, time.Now().UTC(), survey, dispatches),
		Dispatches:   results,
		PhasePlan:    testWorkerPlanArtifact(),
	})
	assertPlanFinalizeErrorContains(t, err, "explicit synthesis")
	assertPlanFinalizeStateBytesUnchanged(t, stateBefore)
	assertPlanFinalizeStateUnchanged(t, 0)
	assertPlanFinalizeNoSpawnRuns(t)
}

func TestPlanFinalizeRejectsSynthesisCompletionBeforeStateMutation(t *testing.T) {
	saveGlobals(t)

	goal := "Reject host synthesis planning claims"
	root, survey, dispatches := setupPlanFinalizeFailureFixture(t, goal)
	stateBefore := readPlanFinalizeStateBytes(t)

	_, err := runCodexPlanFinalize(root, codexExternalPlanCompletion{
		PlanManifest: testPlanManifest(root, goal, time.Now().UTC(), survey, dispatches),
		Synthesis: &codexPlanSynthesis{
			Source:    "ts-host",
			Reason:    "legacy lifecycle smoke harness",
			PhasePlan: testWorkerPlanArtifact(),
		},
	})
	assertPlanFinalizeErrorContains(t, err, "synthesis planning packets cannot satisfy")
	assertPlanFinalizeStateBytesUnchanged(t, stateBefore)
	assertPlanFinalizeStateUnchanged(t, 0)
	assertPlanFinalizeNoSpawnRuns(t)
}

func TestPlanFinalizeRejectsPlannedOrSpawnedResultsBeforeStateMutation(t *testing.T) {
	for _, status := range []string{"planned", "spawned"} {
		t.Run(status, func(t *testing.T) {
			saveGlobals(t)

			dataDir := setupBuildFlowTest(t)
			root := filepath.Dir(filepath.Dir(dataDir))
			withWorkingDir(t, root)
			if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/nonterminal-plan\n\ngo 1.24\n"), 0644); err != nil {
				t.Fatalf("write go.mod: %v", err)
			}

			goal := "Reject non-terminal planning worker results"
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
			results := testCompletedPlanningResults(dispatches)
			for i := range results {
				results[i].Status = status
			}

			_, err = runCodexPlanFinalize(root, codexExternalPlanCompletion{
				PlanManifest: testPlanManifest(root, goal, time.Now().UTC(), survey, dispatches),
				Dispatches:   results,
			})
			if err == nil {
				t.Fatalf("expected plan-finalize to reject %q planning results", status)
			}
			if !strings.Contains(err.Error(), "non-terminal") && !strings.Contains(err.Error(), "did not complete cleanly") {
				t.Fatalf("expected non-terminal completion error for %q, got %v", status, err)
			}
			assertPlanFinalizeStateUnchanged(t, 0)
		})
	}
}

func TestPlanFinalizeRejectsDynamicPlanningWorkerSets(t *testing.T) {
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

	_, err = runCodexPlanFinalize(root, codexExternalPlanCompletion{
		PlanManifest: testPlanManifest(root, goal, time.Now().UTC(), survey, dispatches),
		Dispatches:   results,
	})
	if err == nil || !strings.Contains(err.Error(), "second worker must be Route-Setter") {
		t.Fatalf("expected dynamic worker-set rejection, got %v", err)
	}
	assertPlanFinalizeStateUnchanged(t, 0)

	// Trailing workers beyond the core pair are allowed ONLY as phase_research
	// Scouts; any other dynamic addition stays rejected.
	trailing := []codexPlanningDispatch{
		{Stage: "scouting", Wave: 1, Caste: "scout", Name: "Scout-1", Task: "Summarize planning context", TaskID: "plan-scout", Outputs: []string{"SCOUT.md"}},
		{Stage: "routing", Wave: 2, Caste: "route_setter", Name: "Route-1", Task: "Create constrained phase plan", TaskID: "plan-route-setter", Outputs: []string{"ROUTE-SETTER.md", "phase-plan.json"}},
		{Stage: "architecture", Wave: 3, Caste: "architect", Name: "Arch-1", Task: "Identify structural risks", TaskID: "plan-architect", Outputs: []string{"ARCHITECT.md"}},
	}
	trailingResults := testCompletedPlanningResults(trailing)
	trailingResults[1].PhasePlan = testWorkerPlanArtifact()
	_, err = runCodexPlanFinalize(root, codexExternalPlanCompletion{
		PlanManifest: testPlanManifest(root, goal, time.Now().UTC(), survey, trailing),
		Dispatches:   trailingResults,
	})
	if err == nil || !strings.Contains(err.Error(), "must be a phase_research Scout") {
		t.Fatalf("expected trailing non-research worker rejection, got %v", err)
	}
	assertPlanFinalizeStateUnchanged(t, 0)
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

func TestPlanFinalizePendingIterationDoesNotWriteFinalPlanAndDrivesNextManifest(t *testing.T) {
	saveGlobals(t)

	goal := "Iterate planning until confidence target"
	root, survey, dispatches := setupPlanFinalizeFailureFixture(t, goal)
	var colonyState colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &colonyState); err != nil {
		t.Fatal(err)
	}
	colonyState = codexPlanSpecificationFixture(t, colonyState, colony.SpecStatusApproved)
	if err := store.SaveJSON("COLONY_STATE.json", colonyState); err != nil {
		t.Fatal(err)
	}
	writeCodexPlanSpecificationProjection(t, root, colonyState)
	manifest := testPlanManifest(root, goal, time.Now().UTC(), survey, dispatches)
	manifest.TargetConfidence = 99
	manifest.MaxIterations = 12
	manifest.Depth = "exhaustive"
	manifest.PlanningLoop.TargetConfidence = 99
	manifest.PlanningLoop.MaxIterations = 12
	results := testCompletedPlanningResults(dispatches)
	for i := range results {
		if results[i].Caste == "route_setter" {
			results[i].PhasePlan = testWorkerPlanArtifactWithConfidence(84, []string{"Route-Setter needs API boundary evidence"})
		}
	}

	result, err := runCodexPlanFinalize(root, codexExternalPlanCompletion{
		PlanManifest: manifest,
		Dispatches:   results,
	})
	if err != nil {
		t.Fatalf("runCodexPlanFinalize pending iteration: %v", err)
	}
	if result["planned"] != false || result["requires_next_iteration"] != true {
		t.Fatalf("pending result should require next iteration, got %+v", result)
	}
	assertPlanFinalizeStateUnchanged(t, 0)
	if _, err := os.Stat(filepath.Join(store.BasePath(), planningIterationStateRel)); err != nil {
		t.Fatalf("expected planning iteration state: %v", err)
	}

	next, err := runCodexPlanWithOptions(root, codexPlanOptions{
		PlanOnly:      true,
		Preset:        "exhaustive",
		PresetSet:     true,
		PlanningDepth: "standard",
	})
	if err != nil {
		t.Fatalf("next plan-only manifest: %v", err)
	}
	nextManifest := next["plan_manifest"].(codexPlanManifest)
	if nextManifest.Iteration != 2 {
		t.Fatalf("next iteration = %d, want 2", nextManifest.Iteration)
	}
	if nextManifest.PreviousConfidence != 84 {
		t.Fatalf("previous confidence = %d, want 84", nextManifest.PreviousConfidence)
	}
	if !containsString(nextManifest.SelectedGaps, "Route-Setter needs API boundary evidence") {
		t.Fatalf("selected gaps = %v, want route-setter gap", nextManifest.SelectedGaps)
	}
	if nextManifest.PreviousPlanDraft == nil || len(nextManifest.PreviousPlanDraft.Phases) == 0 {
		t.Fatalf("expected previous_plan_draft in next manifest: %+v", nextManifest.PreviousPlanDraft)
	}
	if len(nextManifest.Dispatches) != 1 || nextManifest.Dispatches[0].Caste != string(planningStageCasteScout) || !strings.Contains(nextManifest.Dispatches[0].Brief, "Route-Setter needs API boundary evidence") {
		t.Fatalf("next manifest did not authorize exactly one Scout with the selected gap context: %+v", nextManifest.Dispatches)
	}
}

func TestPlanFinalizeStallsOnlyAfterTwoLowImprovements(t *testing.T) {
	history := planningConfidenceStopHistoryFixture([]int{60, 60}, []string{"same-gap", "same-gap"})
	first, err := evaluatePlanningStopPolicy(planningStopPolicyInput{Target: 90, PassCap: 12, History: history})
	if err != nil {
		t.Fatal(err)
	}
	if first.Decision.Reason != colony.PlanningStopContinue {
		t.Fatalf("one repeated unimproved gap stopped planning: %+v", first)
	}
	history = planningConfidenceStopHistoryFixture([]int{60, 60, 60}, []string{"same-gap", "same-gap", "same-gap"})
	second, err := evaluatePlanningStopPolicy(planningStopPolicyInput{Target: 90, PassCap: 12, History: history})
	if err != nil {
		t.Fatal(err)
	}
	if second.Decision.Reason != colony.PlanningStopStalledGap || second.Trigger != colony.PlanningStopStalledGap {
		t.Fatalf("two repeated unimproved gaps did not produce the typed stall stop: %+v", second)
	}
}

func TestPlanFinalizeRejectsReusedIterationCompletionPacket(t *testing.T) {
	saveGlobals(t)

	goal := "Reject reused planning completion"
	root, survey, dispatches := setupPlanFinalizeFailureFixture(t, goal)
	manifest := testPlanManifest(root, goal, time.Now().UTC(), survey, dispatches)
	manifest.TargetConfidence = 99
	manifest.PlanningLoop.TargetConfidence = 99
	results := testCompletedPlanningResults(dispatches)
	for i := range results {
		if results[i].Caste == "route_setter" {
			results[i].PhasePlan = testWorkerPlanArtifactWithConfidence(84, []string{"Need second evidence pass"})
		}
	}
	completion := codexExternalPlanCompletion{PlanManifest: manifest, Dispatches: results}
	if _, err := runCodexPlanFinalize(root, completion); err != nil {
		t.Fatalf("first pending finalize: %v", err)
	}
	_, err := runCodexPlanFinalize(root, completion)
	if err == nil || !strings.Contains(err.Error(), "reused planning completion packet") {
		t.Fatalf("expected reused packet rejection, got %v", err)
	}
	assertPlanFinalizeStateUnchanged(t, 0)
}

func TestPlanFinalizeRejectsConfidenceIncreaseWithoutNewEvidence(t *testing.T) {
	saveGlobals(t)

	goal := "Reject confidence-only planning bump"
	root, survey, dispatches := setupPlanFinalizeFailureFixture(t, goal)
	results := testCompletedPlanningResults(dispatches)
	for i := range results {
		if results[i].Caste == "route_setter" {
			results[i].PhasePlan = testWorkerPlanArtifactWithConfidence(80, []string{"Need stronger dependency evidence"})
		}
	}
	scout := *results[0].ScoutReport
	hash := planningCompletionEvidenceHash(scout, *results[1].PhasePlan)
	manifest := testPlanManifest(root, goal, time.Now().UTC(), survey, dispatches)
	manifest.Iteration = 2
	manifest.PreviousConfidence = 70
	manifest.PreviousEvidenceHash = hash
	manifest.PlanningLoop.History = []codexPlanningLoopSample{{
		Iteration:    1,
		Confidence:   70,
		EvidenceHash: hash,
	}}

	_, err := runCodexPlanFinalize(root, codexExternalPlanCompletion{
		PlanManifest: manifest,
		Dispatches:   results,
	})
	if err == nil || !strings.Contains(err.Error(), "without new Scout or Route-Setter evidence") {
		t.Fatalf("expected confidence-only rejection, got %v", err)
	}
	assertPlanFinalizeStateUnchanged(t, 0)
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
	state := codexPlanSpecificationFixture(t, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	}, colony.SpecStatusApproved)
	createTestColonyState(t, dataDir, state)
	writeCodexPlanSpecificationProjection(t, root, state)

	result, err := runCodexPlanWithOptions(root, codexPlanOptions{PlanOnly: true, Preset: "balanced", PresetSet: true, VerificationDepth: "heavy"})
	if err != nil {
		t.Fatalf("runCodexPlanWithOptions: %v", err)
	}
	if got := result["verification_depth"]; got != string(colony.VerificationDepthHeavy) {
		t.Fatalf("verification_depth result = %v, want heavy", got)
	}

	var persisted colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &persisted); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if persisted.VerificationDepth != "" {
		t.Fatalf("plan-only persisted verification_depth = %q, want empty", persisted.VerificationDepth)
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
	loop := resolvePlanningLoopOptions("fast", codexPlanOptions{})
	manifest := &codexPlanManifest{
		Goal:              goal,
		Root:              root,
		GeneratedAt:       generatedAt.Format(time.RFC3339),
		ColonyMode:        "colony",
		PlanningRunID:     planningRunID(goal, root, generatedAt),
		Iteration:         1,
		TargetConfidence:  loop.TargetConfidence,
		MaxIterations:     loop.MaxIterations,
		ExpectedWorkers:   append([]codexPlanningDispatch{}, dispatches...),
		Depth:             "fast",
		Granularity:       string(colony.GranularitySprint),
		GranularityMin:    1,
		GranularityMax:    2,
		PlanningDepth:     "standard",
		PlanningLoop:      loop,
		VerificationDepth: "standard",
		Survey:            survey,
		Dispatches:        dispatches,
		DispatchMode:      "plan-only",
		DispatchContract:  planningDispatchContract(),
		FinalizeSurface:   "pending",
		RequiresFinalizer: true,
	}
	bindTestPlanManifestToCurrentState(manifest)
	return manifest
}

func bindTestPlanManifestToCurrentState(manifest *codexPlanManifest) {
	var state colony.ColonyState
	if store == nil {
		panic("test store is not initialized")
	}
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		panic(err)
	}
	state = normalizeLegacyColonyState(state)
	hash, err := planStateHash(state.Plan)
	if err != nil {
		panic(err)
	}
	manifest.BaseRevisionID = activePlanRevisionID(state.Plan)
	manifest.BasePlanStateHash = hash
}

func testCompletedPlanningResults(dispatches []codexPlanningDispatch) []codexPlanningDispatch {
	results := make([]codexPlanningDispatch, 0, len(dispatches))
	for _, dispatch := range dispatches {
		dispatch.Status = "completed"
		dispatch.Summary = dispatch.Name + " completed"
		if dispatch.Caste == "scout" {
			dispatch.ScoutReport = &codexScoutReport{
				Findings: []codexScoutFinding{{
					Area:      "Planning fixture",
					Discovery: "Fixture Scout gathered repository evidence",
					Source:    "cmd/codex_plan_finalize_test.go",
				}},
				Gaps:       []string{},
				Confidence: 85,
				StudyFiles: []string{
					"cmd/codex_plan_finalize.go",
				},
			}
		}
		if dispatch.Caste == "route_setter" {
			dispatch.PhasePlan = testWorkerPlanArtifact()
		}
		results = append(results, dispatch)
	}
	return results
}

func testWorkerPlanArtifact() *codexWorkerPlanArtifact {
	artifact := &codexWorkerPlanArtifact{
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
	phases := bindSyntheticPlanEvidence(buildWorkerPlanPhases(*artifact))
	bound := workerPlanArtifactFromPhases(artifact.Confidence, artifact.Gaps, phases, codexPlanningLoop{})
	return &bound
}

func testWorkerPlanArtifactWithConfidence(overall planScore, gaps []string) *codexWorkerPlanArtifact {
	artifact := testWorkerPlanArtifact()
	artifact.Confidence = codexPlanConfidence{
		Knowledge:    overall,
		Requirements: overall,
		Risks:        overall,
		Dependencies: overall,
		Effort:       overall,
		Overall:      overall,
	}
	artifact.Gaps = append([]string{}, gaps...)
	return artifact
}

func setupPlanFinalizeFailureFixture(t *testing.T, goal string) (string, codexSurveyContext, []codexPlanningDispatch) {
	t.Helper()
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/plan-finalize-failure\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
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
	return root, survey, testPlanningDispatches()
}

func readPlanFinalizeStateBytes(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(store.BasePath(), "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("read colony state: %v", err)
	}
	return data
}

func assertPlanFinalizeStateBytesUnchanged(t *testing.T, before []byte) {
	t.Helper()
	after := readPlanFinalizeStateBytes(t)
	if !bytes.Equal(after, before) {
		t.Fatalf("plan-finalize changed COLONY_STATE.json before validation:\nbefore: %s\nafter: %s", string(before), string(after))
	}
}

func assertPlanFinalizeErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected plan-finalize error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected plan-finalize error containing %q, got %v", want, err)
	}
}

func assertPlanFinalizeNoSpawnRuns(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(store.BasePath(), "spawn-runs.json")); err == nil {
		t.Fatal("plan-finalize wrote spawn-runs.json before validation failure")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat spawn-runs.json: %v", err)
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
