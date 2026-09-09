package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codegraph"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanUsesSurveyAndRecordsPlanningDispatches(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir to test root: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n\nrequire github.com/spf13/cobra v1.9.0\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0755); err != nil {
		t.Fatalf("failed to create cmd dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "main_test.go"), []byte("package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {}\n"), 0644); err != nil {
		t.Fatalf("failed to write main_test.go: %v", err)
	}

	goal := "Bring Codex core colony commands to true ant-process parity"
	fixtureState := codexPlanSpecificationFixture(t, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	}, colony.SpecStatusApproved)
	createTestColonyState(t, dataDir, fixtureState)
	writeCodexPlanSpecificationProjection(t, root, fixtureState)

	rootCmd.SetArgs([]string{"colonize"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colonize returned error: %v", err)
	}

	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"plan", "--synthetic", "--preset", "balanced"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(stdout.(*bytes.Buffer).Bytes(), &envelope); err != nil {
		t.Fatalf("failed to parse plan output: %v\n%s", err, stdout.(*bytes.Buffer).String())
	}
	if envelope["ok"] != true {
		t.Fatalf("expected ok:true, got %v", envelope)
	}
	result := envelope["result"].(map[string]interface{})
	if existing, _ := result["existing_plan"].(bool); existing {
		t.Fatal("expected a fresh generated plan, not existing_plan:true")
	}
	if count := int(result["count"].(float64)); count < 4 {
		t.Fatalf("expected a grounded multi-phase plan, got %d phases", count)
	}
	dispatches := result["dispatches"].([]interface{})
	if len(dispatches) != 2 {
		t.Fatalf("expected 2 planning dispatches, got %d", len(dispatches))
	}
	planningFiles := result["planning_files"].([]interface{})
	if len(planningFiles) != 2 {
		t.Fatalf("expected 2 planning files, got %d", len(planningFiles))
	}
	phaseResearchFiles := result["phase_research_files"].([]interface{})
	if len(phaseResearchFiles) != int(result["count"].(float64)) {
		t.Fatalf("expected phase research files to match phase count, got %d", len(phaseResearchFiles))
	}

	for _, name := range []string{"SCOUT.md", "ROUTE-SETTER.md"} {
		if _, err := os.Stat(filepath.Join(dataDir, "planning", name)); err != nil {
			t.Fatalf("expected planning artifact %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dataDir, "phase-research", "phase-1-research.md")); err != nil {
		t.Fatalf("expected phase research file: %v", err)
	}

	spawnTreeData, err := os.ReadFile(filepath.Join(dataDir, "spawn-tree.txt"))
	if err != nil {
		t.Fatalf("expected spawn-tree.txt: %v", err)
	}
	if count := strings.Count(string(spawnTreeData), "|Queen|scout|"); count != 1 {
		t.Fatalf("expected 1 scout spawn entry, got %d\n%s", count, string(spawnTreeData))
	}
	if count := strings.Count(string(spawnTreeData), "|Queen|route_setter|"); count != 1 {
		t.Fatalf("expected 1 route_setter spawn entry, got %d\n%s", count, string(spawnTreeData))
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload colony state: %v", err)
	}
	if state.Plan.GeneratedAt == nil {
		t.Fatal("expected GeneratedAt to be set")
	}
	if state.Plan.Confidence == nil || *state.Plan.Confidence <= 0 {
		t.Fatal("expected plan confidence to be set")
	}
	if len(state.Plan.Phases) == 0 || state.Plan.Phases[0].Status != colony.PhaseReady {
		t.Fatalf("expected first phase to be ready, got %+v", state.Plan.Phases)
	}
	if len(state.Events) == 0 || !strings.Contains(state.Events[len(state.Events)-1], "plan_generated|plan") {
		t.Fatalf("expected plan_generated event, got %v", state.Events)
	}

	contextData, err := os.ReadFile(filepath.Join(root, ".aether", "CONTEXT.md"))
	if err != nil {
		t.Fatalf("expected CONTEXT.md: %v", err)
	}
	if !strings.Contains(string(contextData), "aether build 1") {
		t.Fatalf("expected CONTEXT.md to point at the first build, got:\n%s", string(contextData))
	}

	handoffData, err := os.ReadFile(filepath.Join(root, ".aether", "HANDOFF.md"))
	if err != nil {
		t.Fatalf("expected HANDOFF.md: %v", err)
	}
	if !strings.Contains(string(handoffData), goal) {
		t.Fatalf("expected HANDOFF.md to include the goal, got:\n%s", string(handoffData))
	}
}

func TestPlanRepairArtifactNormalizesCustomDependenciesWithoutWorkers(t *testing.T) {
	saveGlobals(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	planningDir := filepath.Join(dataDir, "planning")
	if err := os.MkdirAll(planningDir, 0755); err != nil {
		t.Fatalf("mkdir planning: %v", err)
	}
	artifact := codexWorkerPlanArtifact{
		Phases: []codexWorkerPlanPhase{{
			Name: "Repair dependency aliases",
			Tasks: []codexWorkerPlanTask{
				{Goal: "Create source task"},
				{Goal: "Run dependent task", DependsOn: []string{"P1-T1"}},
			},
		}},
		Confidence: codexPlanConfidence{Overall: 80},
	}
	if err := store.SaveJSON("planning/phase-plan.json", artifact); err != nil {
		t.Fatalf("save phase-plan: %v", err)
	}

	result, err := runCodexPlanRepairArtifact(root)
	if err != nil {
		t.Fatalf("runCodexPlanRepairArtifact: %v", err)
	}
	if repaired, _ := result["repaired"].(bool); !repaired {
		t.Fatalf("expected repaired true, got %#v", result)
	}

	var repaired codexWorkerPlanArtifact
	if err := store.LoadJSON("planning/phase-plan.json", &repaired); err != nil {
		t.Fatalf("load repaired phase-plan: %v", err)
	}
	got := repaired.Phases[0].Tasks[1].DependsOn
	if len(got) != 1 || got[0] != "1.1" {
		t.Fatalf("depends_on = %v, want [1.1]", got)
	}
}

func TestPlanRepairArtifactRejectsTextDependencies(t *testing.T) {
	saveGlobals(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	if err := os.MkdirAll(filepath.Join(dataDir, "planning"), 0755); err != nil {
		t.Fatalf("mkdir planning: %v", err)
	}
	artifact := codexWorkerPlanArtifact{
		Phases: []codexWorkerPlanPhase{{
			Name: "Reject prose dependencies",
			Tasks: []codexWorkerPlanTask{
				{Goal: "Create source task"},
				{Goal: "Run dependent task", DependsOn: []string{"Create source task"}},
			},
		}},
		Confidence: codexPlanConfidence{Overall: 80},
	}
	if err := store.SaveJSON("planning/phase-plan.json", artifact); err != nil {
		t.Fatalf("save phase-plan: %v", err)
	}

	_, err := runCodexPlanRepairArtifact(root)
	if err == nil || !strings.Contains(err.Error(), "not a valid task id") {
		t.Fatalf("expected actionable dependency error, got %v", err)
	}
}

func TestMDSRepoDetectionProducesMaxForLivePlanningSurface(t *testing.T) {
	saveGlobals(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	for _, rel := range []string{"devices", "m4l_builder", filepath.Join("scripts", "mds"), "MaxForLive_Vault"} {
		if err := os.MkdirAll(filepath.Join(root, rel), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("MDS Max for Live repo instructions\n"), 0644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	survey, err := loadCodexSurveyContext(root)
	if err != nil {
		t.Fatalf("load survey context: %v", err)
	}
	if !containsString(survey.Frameworks, "Max for Live") || !containsString(survey.Frameworks, "MDS") {
		t.Fatalf("frameworks = %v, want Max for Live and MDS", survey.Frameworks)
	}
	templates := planningTemplates("Build an AnalogWave MDS device", survey, codexScoutReport{Confidence: 80})
	blob, _ := json.Marshal(templates)
	if !strings.Contains(string(blob), "Max for Live") || !strings.Contains(string(blob), "MDS") {
		t.Fatalf("MDS planning templates missing domain-specific wording:\n%s", string(blob))
	}
	if strings.Contains(string(blob), "generic app") {
		t.Fatalf("MDS planning templates should not be generic app planning:\n%s", string(blob))
	}
}

func TestPlanAcceptWithExistingPlanFailsLoudly(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Reuse the current plan"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:     1,
				Name:   "Existing phase",
				Status: colony.PhaseReady,
				Tasks:  []colony.Task{{ID: &taskID, Goal: "Use the existing plan", Status: colony.TaskPending}},
			}},
		},
	})
	statePath := filepath.Join(dataDir, "COLONY_STATE.json")
	before, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"plan", "--accept"})
	commandErr := rootCmd.Execute()
	var renderedErr renderedCommandError
	if !errors.As(commandErr, &renderedErr) || renderedErr.code != 1 {
		t.Fatalf("plan --accept command error = %#v, want rendered exit 1", commandErr)
	}
	out := stdout.(*bytes.Buffer).String()
	errOut := stderr.(*bytes.Buffer).String()
	if strings.TrimSpace(out) != "" {
		t.Fatalf("plan --accept with an existing plan emitted success output:\n%s", out)
	}
	var envelope struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
		Code  int    `json:"code"`
	}
	if err := json.Unmarshal([]byte(errOut), &envelope); err != nil {
		t.Fatalf("parse plan --accept error envelope: %v\nstderr=%s", err, errOut)
	}
	if envelope.OK || envelope.Code != 1 {
		t.Fatalf("plan --accept envelope = %+v, want ok:false code:1", envelope)
	}
	if !strings.Contains(envelope.Error, "aether plan --accept-candidate <candidate-id>") {
		t.Fatalf("error must direct the user to exact candidate acceptance, got: %s", envelope.Error)
	}
	after, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("plan --accept refusal changed the active colony state")
	}
}

func TestPlanReturnsExistingPlanWithoutRefresh(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Reuse the current plan"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Existing phase",
					Description: "Already planned",
					Status:      colony.PhaseReady,
					Tasks: []colony.Task{
						{ID: &taskID, Goal: "Use the existing plan", Status: colony.TaskPending},
					},
				},
			},
		},
	})

	rootCmd.SetArgs([]string{"plan", "--preset", "balanced"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(stdout.(*bytes.Buffer).Bytes(), &envelope); err != nil {
		t.Fatalf("failed to parse plan output: %v\n%s", err, stdout.(*bytes.Buffer).String())
	}
	result := envelope["result"].(map[string]interface{})
	if existing, _ := result["existing_plan"].(bool); !existing {
		t.Fatalf("expected existing_plan:true, got %v", result)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "spawn-tree.txt")); err == nil {
		t.Fatal("expected no new planning spawns when reusing existing plan")
	}
}

func TestPlanOnlyExistingPlanDoesNotReturnFinalizerManifest(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Reuse the current plan without spawning planners"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:     1,
				Name:   "Existing phase",
				Status: colony.PhaseReady,
				Tasks:  []colony.Task{{ID: &taskID, Goal: "Use the existing plan", Status: colony.TaskPending}},
			}},
		},
	})

	rootCmd.SetArgs([]string{"plan", "--plan-only", "--preset", "balanced"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan --plan-only returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if existing, _ := result["existing_plan"].(bool); !existing {
		t.Fatalf("existing_plan = %v, want true", result["existing_plan"])
	}
	if requires, _ := result["requires_finalizer"].(bool); requires {
		t.Fatalf("requires_finalizer = true for existing plan no-op: %+v", result)
	}
	if _, ok := result["plan_manifest"]; ok {
		t.Fatalf("existing plan no-op returned plan_manifest: %+v", result["plan_manifest"])
	}
	if _, ok := result["planning_manifest"]; ok {
		t.Fatalf("existing plan no-op returned planning_manifest: %+v", result["planning_manifest"])
	}
	if dispatches, ok := result["dispatches"].([]interface{}); ok && len(dispatches) > 0 {
		t.Fatalf("existing plan no-op returned planning dispatches: %+v", dispatches)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "spawn-tree.txt")); err == nil {
		t.Fatal("plan --plan-only existing plan unexpectedly wrote spawn-tree.txt")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat spawn-tree.txt: %v", err)
	}
}

func TestPlanIgnoresPriorGoalPlanningArtifactForFreshSession(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-stale-plan-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	stalePhaseName := "Prior goal stale phase"
	if err := store.SaveJSON("planning/phase-plan.json", codexWorkerPlanArtifact{
		Confidence: codexPlanConfidence{Overall: 97},
		Phases: []codexWorkerPlanPhase{
			{
				Name:        stalePhaseName,
				Description: "This plan belongs to the previous colony goal.",
				Tasks:       []codexWorkerPlanTask{{Goal: "Do prior-goal work"}},
			},
		},
	}); err != nil {
		t.Fatalf("save stale planning artifact: %v", err)
	}

	freshGoal := "Fresh session should get a fresh plan"
	sessionID := "fresh-session"
	sessionStarted := time.Now().UTC()
	if err := store.SaveJSON("session.json", colony.SessionFile{
		SessionID:   sessionID,
		StartedAt:   sessionStarted.Format(time.RFC3339),
		ColonyGoal:  freshGoal,
		LastCommand: "init",
	}); err != nil {
		t.Fatalf("save session: %v", err)
	}
	fixtureState := codexPlanSpecificationFixture(t, colony.ColonyState{
		Version:      "3.0",
		Goal:         &freshGoal,
		State:        colony.StateREADY,
		SessionID:    &sessionID,
		CurrentPhase: 0,
		Plan:         colony.Plan{Phases: []colony.Phase{}},
	}, colony.SpecStatusApproved)
	createTestColonyState(t, dataDir, fixtureState)
	writeCodexPlanSpecificationProjection(t, root, fixtureState)

	rootCmd.SetArgs([]string{"plan", "--synthetic", "--preset", "balanced"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if existing, _ := result["existing_plan"].(bool); existing {
		t.Fatalf("expected fresh plan generation, got existing_plan:true: %+v", result)
	}
	if result["goal"].(string) != freshGoal {
		t.Fatalf("goal = %q, want %q", result["goal"], freshGoal)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if len(state.Plan.Phases) == 0 {
		t.Fatal("expected plan to generate fresh phases")
	}
	for _, phase := range state.Plan.Phases {
		if strings.Contains(phase.Name, stalePhaseName) || strings.Contains(phase.Description, "previous colony goal") {
			t.Fatalf("stale planning artifact leaked into fresh plan: %+v", state.Plan.Phases)
		}
	}
	events := strings.Join(state.Events, "\n")
	if strings.Contains(events, "plan_recovered|state") {
		t.Fatalf("fresh session should not recover from stale planning artifact, got events: %v", state.Events)
	}
	if !strings.Contains(events, "plan_generated|plan") {
		t.Fatalf("expected fresh plan_generated event, got %v", state.Events)
	}
}

func TestPlanOnlyPrintsManifestWithoutMutatingState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	setupRuntimeSkillAssignmentHub(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	t.Setenv("AETHER_NARRATOR", "on")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-plan-only-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	goal := "Expose planning workers to wrappers"
	fixtureState := codexPlanSpecificationFixture(t, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	}, colony.SpecStatusApproved)
	createTestColonyState(t, dataDir, fixtureState)
	writeCodexPlanSpecificationProjection(t, root, fixtureState)

	rootCmd.SetArgs([]string{"plan", "--plan-only", "--depth", "deep"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan --plan-only returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if result["plan_only"] != true {
		t.Fatalf("plan_only = %v, want true", result["plan_only"])
	}
	if result["dispatch_mode"].(string) != "plan-only" {
		t.Fatalf("dispatch_mode = %q, want plan-only", result["dispatch_mode"])
	}
	if result["colony_mode"].(string) != "colony" {
		t.Fatalf("colony_mode = %q, want colony", result["colony_mode"])
	}
	manifest := result["plan_manifest"].(map[string]interface{})
	if _, ok := result["planning_manifest"].(map[string]interface{}); !ok {
		t.Fatalf("planning_manifest alias missing from result")
	}
	if manifest["dispatch_mode"].(string) != "plan-only" {
		t.Fatalf("manifest dispatch_mode = %q, want plan-only", manifest["dispatch_mode"])
	}
	if manifest["colony_mode"].(string) != "colony" {
		t.Fatalf("manifest colony_mode = %q, want colony", manifest["colony_mode"])
	}
	if manifest["depth"].(string) != "deep" || manifest["granularity"].(string) != "quarter" {
		t.Fatalf("manifest depth/granularity = %v/%v, want deep/quarter", manifest["depth"], manifest["granularity"])
	}
	if manifest["requires_finalizer"] != true {
		t.Fatalf("manifest requires_finalizer = %v, want true", manifest["requires_finalizer"])
	}
	dispatches := manifest["dispatches"].([]interface{})
	if len(dispatches) != 1 {
		t.Fatalf("expected one Scout dispatch, got %d", len(dispatches))
	}
	first := dispatches[0].(map[string]interface{})
	if first["caste"].(string) != "scout" || first["agent_name"].(string) != "aether-scout" || first["status"].(string) != "planned" {
		t.Fatalf("unexpected scout dispatch: %+v", first)
	}
	assertDispatchHasRuntimeSkillAssignment(t, first)
	if _, ok := result["stage_manifest"].(map[string]interface{}); !ok {
		t.Fatalf("plan-only result missing Scout stage_manifest")
	}
	if _, ok := result["planning_run_header"].(map[string]interface{}); !ok {
		t.Fatalf("plan-only result missing persisted planning_run_header")
	}

	for _, rel := range []string{"phase-research", "spawn-tree.txt", "session.json", "event-bus.jsonl", "spawn-runs.json"} {
		if _, err := os.Stat(filepath.Join(dataDir, rel)); err == nil {
			t.Fatalf("plan --plan-only unexpectedly wrote %s", rel)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", rel, err)
		}
	}
	for _, rel := range []string{filepath.Join(".aether", "CONTEXT.md"), filepath.Join(".aether", "HANDOFF.md")} {
		if _, err := os.Stat(filepath.Join(root, rel)); err == nil {
			t.Fatalf("plan --plan-only unexpectedly wrote %s", rel)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", rel, err)
		}
	}
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if len(state.Plan.Phases) != 0 || state.CurrentPhase != 0 || state.Plan.GeneratedAt != nil {
		t.Fatalf("plan --plan-only mutated state: %+v", state)
	}
}

func TestPlanRefreshUsesAgentDelegatePathInsideHostedAgent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	setupRuntimeSkillAssignmentHub(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	t.Setenv("AETHER_ACTIVE_PLATFORM", "opencode")
	t.Setenv("OPENCODE_AGENT", "1")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-plan-agent-delegate-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	goal := "Refresh planning without nested subprocess workers"
	taskID := "task-1"
	fixtureState := codexPlanSpecificationFixture(t, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Stale plan",
			Status: colony.PhaseInProgress,
			Tasks:  []colony.Task{{ID: &taskID, Goal: "Old task", Status: colony.TaskInProgress}},
		}}},
	}, colony.SpecStatusApproved)
	createTestColonyState(t, dataDir, fixtureState)
	writeCodexPlanSpecificationProjection(t, root, fixtureState)

	rootCmd.SetArgs([]string{"plan", "--refresh", "--depth", "fast"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan --refresh returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if result["dispatch_mode"].(string) != "agent-delegate" {
		t.Fatalf("dispatch_mode = %q, want agent-delegate", result["dispatch_mode"])
	}
	if result["agent_delegate"] != true {
		t.Fatalf("agent_delegate = %v, want true", result["agent_delegate"])
	}
	if result["requires_finalizer"] != true {
		t.Fatalf("requires_finalizer = %v, want true", result["requires_finalizer"])
	}
	manifest := result["plan_manifest"].(map[string]interface{})
	if manifest["dispatch_mode"].(string) != "agent-delegate" || manifest["requires_finalizer"] != true {
		t.Fatalf("unexpected manifest mode/finalizer: %+v", manifest)
	}
	if manifest["refresh"] != true {
		t.Fatalf("manifest refresh = %v, want true", manifest["refresh"])
	}
	dispatches := manifest["dispatches"].([]interface{})
	if len(dispatches) != 1 || dispatches[0].(map[string]interface{})["caste"] != "scout" {
		t.Fatalf("expected one Scout dispatch, got %+v", dispatches)
	}
	for _, rel := range []string{"phase-research", "spawn-tree.txt", "spawn-runs.json"} {
		if _, err := os.Stat(filepath.Join(dataDir, rel)); err == nil {
			t.Fatalf("agent-delegate plan unexpectedly wrote %s", rel)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", rel, err)
		}
	}
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.CurrentPhase != 1 || len(state.Plan.Phases) != 1 || state.Plan.Phases[0].Status != colony.PhaseInProgress {
		t.Fatalf("agent-delegate plan mutated stale phase state before finalize: %+v", state)
	}
}

func TestPlanDepthMapsToGranularityBounds(t *testing.T) {
	tests := []struct {
		depth       string
		wantDepth   string
		wantGran    colony.PlanGranularity
		wantMin     int
		wantMax     int
		expectError bool
	}{
		{depth: "fast", wantDepth: "fast", wantGran: colony.GranularitySprint, wantMin: 1, wantMax: 3},
		{depth: "balanced", wantDepth: "balanced", wantGran: colony.GranularityMilestone, wantMin: 4, wantMax: 7},
		{depth: "deep", wantDepth: "deep", wantGran: colony.GranularityQuarter, wantMin: 8, wantMax: 12},
		{depth: "exhaustive", wantDepth: "exhaustive", wantGran: colony.GranularityMajor, wantMin: 13, wantMax: 20},
		{depth: "quarter", wantDepth: "deep", wantGran: colony.GranularityQuarter, wantMin: 8, wantMax: 12},
		{depth: "bad", expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.depth, func(t *testing.T) {
			gotGran, gotDepth, err := resolvePlanGranularityDepth("", tt.depth)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotGran != tt.wantGran || gotDepth != tt.wantDepth {
				t.Fatalf("got %s/%s, want %s/%s", gotDepth, gotGran, tt.wantDepth, tt.wantGran)
			}
			if min, max := colony.GranularityRange(gotGran); min != tt.wantMin || max != tt.wantMax {
				t.Fatalf("range = %d-%d, want %d-%d", min, max, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestPlanStagesExternalScoutBeforeFinalization(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-plan-finalize-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	goal := "Finalize external planning workers"
	createApprovedCodexPlanTestColony(t, dataDir, root, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	rootCmd.SetArgs([]string{"plan", "--plan-only", "--depth", "fast"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan --plan-only returned error: %v", err)
	}
	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	manifest := env["result"].(map[string]interface{})["plan_manifest"].(map[string]interface{})
	dispatches := manifest["dispatches"].([]interface{})
	if len(dispatches) != 1 {
		t.Fatalf("expected one Scout dispatch, got %d", len(dispatches))
	}
	first := dispatches[0].(map[string]interface{})
	if first["caste"] != "scout" || first["stage"] != string(planningStageScoutRunning) {
		t.Fatalf("first staged dispatch = %+v, want a running Scout", first)
	}
	if _, leaked := manifest["route_setter"]; leaked {
		t.Fatalf("initial planning manifest leaked Route-Setter authority: %+v", manifest)
	}
	var stagedState colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &stagedState); err != nil {
		t.Fatalf("load state after Scout staging: %v", err)
	}
	if len(stagedState.Plan.Phases) != 0 {
		t.Fatalf("Scout staging activated a plan before finalization: %+v", stagedState.Plan.Phases)
	}
	if len(dispatches) == 1 {
		return
	}

	delete(manifest, "colony_mode")

	scout := dispatches[0].(map[string]interface{})
	routeSetter := dispatches[1].(map[string]interface{})
	completion := map[string]interface{}{
		"plan_manifest": manifest,
		"dispatches": []map[string]interface{}{
			{
				"name":    scout["name"],
				"caste":   scout["caste"],
				"stage":   scout["stage"],
				"wave":    scout["wave"],
				"task_id": scout["task_id"],
				"status":  "completed",
				"summary": "Scout mapped the current planning surface.",
				"scout_report": map[string]interface{}{
					"findings": []map[string]interface{}{
						{"area": "Runtime", "discovery": "Plan finalization belongs in Go", "source": "cmd/codex_plan.go"},
					},
					"gaps":        []string{},
					"confidence":  91,
					"study_files": []string{"cmd/codex_plan.go"},
				},
			},
			{
				"name":    routeSetter["name"],
				"caste":   routeSetter["caste"],
				"stage":   routeSetter["stage"],
				"wave":    routeSetter["wave"],
				"task_id": routeSetter["task_id"],
				"status":  "completed",
				"summary": "Route-Setter shaped the executable phases.",
				"phase_plan": map[string]interface{}{
					"phases": []map[string]interface{}{
						{
							"name":        "External plan foundation",
							"description": "Land external planning finalization.",
							"tasks": []map[string]interface{}{
								{
									"goal":                  "Record wrapper planning outputs through the runtime",
									"constraints":           []string{"Go owns state"},
									"hints":                 []string{"cmd/codex_plan_finalize.go"},
									"success_criteria":      []string{"Planning finalizer writes state"},
									"evidence_requirements": []map[string]interface{}{{"criterion": "Planning finalizer writes state", "checks": []string{"claims", "watcher"}}},
								},
							},
							"success_criteria":      []string{"Finalizer exists"},
							"evidence_requirements": []map[string]interface{}{{"criterion": "Finalizer exists", "checks": []string{"claims", "watcher"}}},
						},
						{
							"name":        "Wrapper plan ceremony",
							"description": "Wire Claude/OpenCode plan wrappers.",
							"tasks": []map[string]interface{}{
								{
									"goal":                  "Spawn Scout and Route-Setter from manifest",
									"constraints":           []string{"Use plan-finalize"},
									"hints":                 []string{".claude/commands/ant/plan.md"},
									"success_criteria":      []string{"Wrapper contract is tested"},
									"evidence_requirements": []map[string]interface{}{{"criterion": "Wrapper contract is tested", "checks": []string{"tests", "watcher"}}},
									"depends_on":            []string{"1.1"},
								},
							},
							"success_criteria":      []string{"Wrappers use real agents"},
							"evidence_requirements": []map[string]interface{}{{"criterion": "Wrappers use real agents", "checks": []string{"claims", "watcher"}}},
						},
					},
					"confidence": map[string]interface{}{"knowledge": 91, "requirements": 88, "risks": 82, "dependencies": 80, "effort": 84, "overall": 86},
					"gaps":       []string{"Manual platform smoke still needed."},
				},
			},
		},
	}
	completionPath := filepath.Join(root, "plan-completion.json")
	data, err := json.Marshal(completion)
	if err != nil {
		t.Fatalf("marshal completion: %v", err)
	}
	if err := os.WriteFile(completionPath, data, 0644); err != nil {
		t.Fatalf("write completion: %v", err)
	}

	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"plan-finalize", "--completion-file", completionPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan-finalize returned error: %v", err)
	}
	finalEnv := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := finalEnv["result"].(map[string]interface{})
	if result["dispatch_mode"].(string) != "external-task" {
		t.Fatalf("dispatch_mode = %q, want external-task", result["dispatch_mode"])
	}
	if result["next"].(string) != "aether build 1" {
		t.Fatalf("next = %q, want aether build 1", result["next"])
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.State != colony.StateREADY || state.CurrentPhase != 1 {
		t.Fatalf("state/current_phase = %s/%d, want READY/1", state.State, state.CurrentPhase)
	}
	if state.PlanGranularity != colony.GranularitySprint {
		t.Fatalf("plan_granularity = %q, want sprint", state.PlanGranularity)
	}
	if len(state.Plan.Phases) != 2 || state.Plan.Phases[0].Status != colony.PhaseReady {
		t.Fatalf("unexpected phases after finalizer: %+v", state.Plan.Phases)
	}
	if state.Plan.GeneratedAt == nil || state.Plan.Confidence == nil || *state.Plan.Confidence <= 0 {
		t.Fatalf("plan metadata missing after finalizer: %+v", state.Plan)
	}
	planningLoop := result["planning_loop"].(map[string]interface{})
	if planningLoop["stop_reason"].(string) == "" {
		t.Fatalf("planning_loop stop reason missing: %+v", planningLoop)
	}

	for _, rel := range []string{
		filepath.Join("planning", "SCOUT.md"),
		filepath.Join("planning", "ROUTE-SETTER.md"),
		filepath.Join("planning", "phase-plan.json"),
		filepath.Join("phase-research", "phase-1-research.md"),
		"spawn-tree.txt",
	} {
		if _, err := os.Stat(filepath.Join(dataDir, rel)); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}
	contextData, err := os.ReadFile(filepath.Join(root, ".aether", "CONTEXT.md"))
	if err != nil {
		t.Fatalf("expected CONTEXT.md: %v", err)
	}
	if !strings.Contains(string(contextData), "aether build 1") {
		t.Fatalf("CONTEXT.md missing next build guidance:\n%s", string(contextData))
	}
	planArtifactData, err := os.ReadFile(filepath.Join(dataDir, "planning", "phase-plan.json"))
	if err != nil {
		t.Fatalf("read phase-plan artifact: %v", err)
	}
	var planArtifact codexWorkerPlanArtifact
	if err := json.Unmarshal(planArtifactData, &planArtifact); err != nil {
		t.Fatalf("parse phase-plan artifact: %v", err)
	}
	if planArtifact.PlanningLoop == nil || planArtifact.PlanningLoop.StopReason == "" {
		t.Fatalf("phase-plan artifact missing planning_loop: %+v", planArtifact)
	}
}

func TestPlanFinalizeAcceptsAgentDelegateManifestMode(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	_, err := runCodexPlanFinalize(t.TempDir(), codexExternalPlanCompletion{
		PlanManifest: &codexPlanManifest{
			Goal:              "Agent delegate planning",
			DispatchMode:      "agent-delegate",
			RequiresFinalizer: true,
		},
	})
	if err == nil || !strings.Contains(err.Error(), "contains no dispatches") {
		t.Fatalf("expected agent-delegate manifest to pass mode validation and fail on dispatches, got %v", err)
	}
}

func TestPlanIncludesDispatchContract(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-plan-contract-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0755); err != nil {
		t.Fatalf("mkdir cmd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	goal := "Map plan dispatch contracts honestly"
	fixtureState := codexPlanSpecificationFixture(t, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	}, colony.SpecStatusApproved)
	createTestColonyState(t, dataDir, fixtureState)
	writeCodexPlanSpecificationProjection(t, root, fixtureState)

	rootCmd.SetArgs([]string{"plan", "--plan-only", "--preset", "balanced"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	contract, ok := result["dispatch_contract"].(map[string]interface{})
	if !ok {
		t.Fatalf("dispatch_contract missing or wrong type: %T", result["dispatch_contract"])
	}

	if got := stringValue(contract["execution_model"]); got != "1 staged planning worker: scout only" {
		t.Fatalf("execution_model = %q, want one staged Scout", got)
	}
	if got := int(contract["wave_count"].(float64)); got != 1 {
		t.Fatalf("wave_count = %d, want 1", got)
	}
	if got := int(contract["worker_count"].(float64)); got != 1 {
		t.Fatalf("worker_count = %d, want 1", got)
	}
	if got := int(contract["shared_timeout_seconds"].(float64)); got != 0 {
		t.Fatalf("shared_timeout_seconds = %d, want 0", got)
	}
	if got := int(contract["worker_timeout_seconds"].(float64)); got != int(maxDuration(planningScoutTimeout, planningRouteSetterTimeout)/time.Second) {
		t.Fatalf("worker_timeout_seconds = %d, want %d", got, int(maxDuration(planningScoutTimeout, planningRouteSetterTimeout)/time.Second))
	}
	if got := stringValue(contract["deadline_policy"]); !strings.Contains(got, "own timeout") || !strings.Contains(got, "dependency_blocked") {
		t.Fatalf("deadline_policy = %q, want per-worker timeout and dependency block language", got)
	}
	if got := stringValue(contract["dependency_behavior"]); !strings.Contains(got, "Route-Setter requires a later receipt-bound manifest") {
		t.Fatalf("dependency_behavior = %q, want future-stage authority guidance", got)
	}
	if got := stringValue(contract["fallback_behavior"]); !strings.Contains(got, "does not fall back to local synthesis") || !strings.Contains(got, "aether plan --synthetic") {
		t.Fatalf("fallback_behavior = %q, want fail-closed synthetic guidance", got)
	}
	if got := stringValue(contract["coordination_path"]); got != filepath.ToSlash(filepath.Join(".aether", "data", "spawn-tree.txt")) {
		t.Fatalf("coordination_path = %q", got)
	}

	visibility := stringSliceValue(contract["fallback_visibility"])
	for _, want := range []string{"dispatch_mode", "planning_warning", "synthetic", "synthetic_warning", "artifact_source", "plan_source"} {
		if !containsString(visibility, want) {
			t.Fatalf("fallback_visibility missing %q: %v", want, visibility)
		}
	}

	artifacts := stringSliceValue(contract["artifact_paths"])
	for _, want := range []string{
		filepath.ToSlash(filepath.Join(".aether", "data", "planning", "SCOUT.md")),
		filepath.ToSlash(filepath.Join(".aether", "data", "planning", "ROUTE-SETTER.md")),
		filepath.ToSlash(filepath.Join(".aether", "data", "planning", "phase-plan.json")),
		filepath.ToSlash(filepath.Join(".aether", "data", "phase-research")),
	} {
		if !containsString(artifacts, want) {
			t.Fatalf("artifact_paths missing %q: %v", want, artifacts)
		}
	}
}

func TestPlanningDispatchContractWithTimeoutOverride(t *testing.T) {
	contract := planningDispatchContractWithTimeout(7 * time.Minute)
	if got, ok := contract["worker_timeout_seconds"].(int); !ok || got != 420 {
		t.Fatalf("worker_timeout_seconds = %#v, want 420", contract["worker_timeout_seconds"])
	}
}

func TestPlanCommandExposesWorkerTimeoutFlag(t *testing.T) {
	if planCmd.Flags().Lookup("worker-timeout") == nil {
		t.Fatal("expected plan command to expose --worker-timeout")
	}
	for _, flag := range []string{"target", "max-iterations", "accept", "repair-artifact"} {
		if planCmd.Flags().Lookup(flag) == nil {
			t.Fatalf("expected plan command to expose --%s", flag)
		}
	}
}

func TestPlanCommandPresetFlags(t *testing.T) {
	for _, flag := range []string{"preset", "target", "max-iterations"} {
		if planCmd.Flags().Lookup(flag) == nil {
			t.Fatalf("expected plan command to expose --%s", flag)
		}
	}
}

func TestCodexPlanPresetResolverExactValues(t *testing.T) {
	tests := []struct {
		name    string
		target  int
		passCap int
	}{
		{name: "fast", target: 80, passCap: 4},
		{name: "balanced", target: 90, passCap: 6},
		{name: "deep", target: 95, passCap: 8},
		{name: "exhaustive", target: 99, passCap: 12},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			selection, err := resolvePlanningPreset(codexPlanOptions{Preset: tc.name, PresetSet: true})
			if err != nil {
				t.Fatal(err)
			}
			if selection.PresetRequired || string(selection.Policy.ID) != tc.name || selection.Policy.TargetConfidence != tc.target || selection.Policy.PassCap != tc.passCap {
				t.Fatalf("selection = %+v, want %s %d/%d", selection, tc.name, tc.target, tc.passCap)
			}
			if selection.SelectionSource != planningPresetSourceNamed {
				t.Fatalf("selection source = %q, want %q", selection.SelectionSource, planningPresetSourceNamed)
			}
		})
	}
}

func TestCodexPlanPresetResolverRequiresChoiceAndValidatesExplicitPair(t *testing.T) {
	selection, err := resolvePlanningPreset(codexPlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !selection.PresetRequired || len(selection.Options) != 4 || selection.Policy.ID != "" {
		t.Fatalf("unflagged selection = %+v, want four choices and no selected default", selection)
	}

	explicit, err := resolvePlanningPreset(codexPlanOptions{
		TargetConfidence:    95,
		TargetConfidenceSet: true,
		MaxIterations:       8,
		MaxIterationsSet:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if explicit.PresetRequired || explicit.Policy.ID != planningStagePresetDeep || explicit.SelectionSource != planningPresetSourceExplicitPair {
		t.Fatalf("explicit pair selection = %+v", explicit)
	}

	invalid := []codexPlanOptions{
		{Preset: "mystery", PresetSet: true},
		{TargetConfidence: 90, TargetConfidenceSet: true},
		{MaxIterations: 6, MaxIterationsSet: true},
		{TargetConfidence: 69, TargetConfidenceSet: true, MaxIterations: 6, MaxIterationsSet: true},
		{TargetConfidence: 90, TargetConfidenceSet: true, MaxIterations: 13, MaxIterationsSet: true},
		{Preset: "fast", PresetSet: true, TargetConfidence: 90, TargetConfidenceSet: true, MaxIterations: 6, MaxIterationsSet: true},
	}
	for i, opts := range invalid {
		if _, err := resolvePlanningPreset(opts); err == nil {
			t.Errorf("invalid selection %d unexpectedly succeeded: %+v", i, opts)
		}
	}
}

func TestPlanCommandLegacyAcceptCannotMutate(t *testing.T) {
	saveGlobals(t)
	_, root := setupPhaseResearchManifestTest(t, researchProposalTestPhases())
	before, err := os.ReadFile(filepath.Join(store.BasePath(), "COLONY_STATE.json"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = runCodexPlanWithOptions(root, codexPlanOptions{
		PlanOnly:  true,
		Preset:    "balanced",
		PresetSet: true,
		Accept:    true,
	})
	if err == nil || !strings.Contains(err.Error(), "aether plan --accept-candidate <candidate-id>") {
		t.Fatalf("legacy --accept error = %v, want exact candidate acceptance guidance", err)
	}
	after, readErr := os.ReadFile(filepath.Join(store.BasePath(), "COLONY_STATE.json"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("legacy --accept changed colony state")
	}
}

func TestCodexPlanApprovedSpecRequiredBeforeDispatch(t *testing.T) {
	tests := []struct {
		name   string
		status colony.SpecRevisionStatus
	}{
		{name: "missing"},
		{name: "draft", status: colony.SpecStatusDraft},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			saveGlobals(t)
			dataDir := setupBuildFlowTest(t)
			root := filepath.Dir(filepath.Dir(dataDir))
			goal := "Plan from an exact approved specification"
			state := colony.ColonyState{Version: "3.0", Goal: &goal, State: colony.StateREADY, Plan: colony.Plan{Phases: []colony.Phase{}}}
			if tc.status != "" {
				state = codexPlanSpecificationFixture(t, state, tc.status)
			}
			createTestColonyState(t, dataDir, state)
			if state.Specification != nil {
				writeCodexPlanSpecificationProjection(t, root, state)
			}

			_, err := runCodexPlanWithOptions(root, codexPlanOptions{
				PlanOnly: true, Preset: "balanced", PresetSet: true,
			})
			if err == nil || !strings.Contains(err.Error(), "aether spec") {
				t.Fatalf("plan error = %v, want exact aether spec recovery", err)
			}
			if _, statErr := os.Stat(filepath.Join(dataDir, "planning")); !os.IsNotExist(statErr) {
				t.Fatalf("planning directory exists before approved SPEC: %v", statErr)
			}
		})
	}
}

func TestCodexPlanApprovedSpecRejectsHashOrProjectionMismatch(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, root string, state *colony.ColonyState)
	}{
		{
			name: "approval_hash",
			mutate: func(t *testing.T, _ string, state *colony.ColonyState) {
				t.Helper()
				current, ok := currentSpecificationRevision(*state.Specification)
				if !ok {
					t.Fatal("fixture has no current specification revision")
				}
				current.Approval.RevisionContentHash = strings.Repeat("f", 64)
				state.Specification.Revisions[len(state.Specification.Revisions)-1] = current
			},
		},
		{
			name: "projection_drift",
			mutate: func(t *testing.T, root string, state *colony.ColonyState) {
				t.Helper()
				writeCodexPlanSpecificationProjection(t, root, *state)
				if err := os.WriteFile(filepath.Join(root, specificationProjectionRelativePath), []byte("tampered projection\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			saveGlobals(t)
			dataDir := setupBuildFlowTest(t)
			root := filepath.Dir(filepath.Dir(dataDir))
			goal := "Reject stale specification bindings"
			state := codexPlanSpecificationFixture(t, colony.ColonyState{
				Version: "3.0", Goal: &goal, State: colony.StateREADY, Plan: colony.Plan{Phases: []colony.Phase{}},
			}, colony.SpecStatusApproved)
			if tc.name != "projection_drift" {
				writeCodexPlanSpecificationProjection(t, root, state)
			}
			tc.mutate(t, root, &state)
			createTestColonyState(t, dataDir, state)

			_, err := runCodexPlanWithOptions(root, codexPlanOptions{
				PlanOnly: true, Preset: "balanced", PresetSet: true,
			})
			if err == nil || !strings.Contains(err.Error(), "aether spec") {
				t.Fatalf("plan error = %v, want exact aether spec recovery", err)
			}
		})
	}
}

func TestCodexPlanEvidencePrimesApprovedInputs(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	hub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hub)
	goal := "Prime every attributable planning input"
	state := codexPlanSpecificationFixture(t, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY, Plan: colony.Plan{Phases: []colony.Phase{}},
	}, colony.SpecStatusApproved)
	state.AcceptedCharter = &colony.AcceptedCharter{
		SchemaVersion: colony.AcceptedCharterSchemaVersion,
		EpisodeID:     "episode-plan-evidence",
		Goal:          goal,
		Provenance:    "owner",
		AcceptedAt:    time.Date(2026, time.September, 7, 17, 0, 0, 0, time.UTC),
		Charter:       &colony.Charter{Intent: goal, Constraints: "Keep authority in Go"},
	}
	state.Events = []string{"2026-09-07T16:00:00Z|phase_completed|continue|Prior verification passed"}
	researchPath := filepath.ToSlash(filepath.Join(".aether", "research", "planning-evidence.md"))
	if err := os.MkdirAll(filepath.Join(root, ".aether", "research"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(researchPath)), []byte("# Evidence\n\nThe staged loop must preserve exact authority.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	state.ResearchDocs = []string{researchPath}
	createTestColonyState(t, dataDir, state)
	writeCodexPlanSpecificationProjection(t, root, state)
	if err := os.MkdirAll(filepath.Join(dataDir, "survey"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "survey", "BLUEPRINT.md"), []byte("# Blueprint\n\ncmd/codex_plan.go owns planning.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{Decisions: []PendingDecision{{
		ID: "decision-plan-evidence", Type: "clarification", Description: "Keep stage authority in Go", Resolution: "confirmed", Resolved: true,
		SessionID: *state.SessionID, GoalHash: pendingDecisionGoalHash(goal), CreatedAt: "2026-09-07T16:10:00Z", ResolvedAt: "2026-09-07T16:11:00Z",
	}}}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(hub, "hive"), 0o755); err != nil {
		t.Fatal(err)
	}
	hiveBytes, err := json.Marshal(hiveWisdomData{Version: hiveWisdomSchemaVersion, Entries: []hiveWisdomEntry{{
		ID: "wisdom-plan-evidence", Text: "Bind every worker to one manifest.", Domain: "go", SourceRepo: "fixture", Confidence: 0.9,
		CreatedAt: "2026-09-07T15:00:00Z", AccessedAt: "2026-09-07T15:00:00Z", LastConfirmedAt: "2026-09-07T15:00:00Z",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hub, "hive", "wisdom.json"), hiveBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := runCodexPlanWithOptions(root, codexPlanOptions{PlanOnly: true, Refresh: true, Preset: "deep", PresetSet: true})
	if err != nil {
		t.Fatal(err)
	}
	decoded := decodeCodexPlanResult(t, result)
	header, ok := decoded["planning_run_header"].(map[string]interface{})
	if !ok {
		t.Fatalf("planning_run_header missing: %+v", decoded)
	}
	records, ok := header["evidence_catalogue"].([]interface{})
	if !ok {
		t.Fatalf("evidence_catalogue missing: %+v", header)
	}
	seen := map[string]bool{}
	for _, raw := range records {
		record := raw.(map[string]interface{})
		ref := record["reference"].(map[string]interface{})
		seen[ref["kind"].(string)] = true
	}
	for _, kind := range []colony.PlanningEvidenceKind{
		colony.PlanningEvidenceSpecification, colony.PlanningEvidenceSurvey, colony.PlanningEvidenceCharter,
		colony.PlanningEvidenceDecision, colony.PlanningEvidenceContext, colony.PlanningEvidenceResearch,
		colony.PlanningEvidenceHive, colony.PlanningEvidenceOutcome,
	} {
		if !seen[string(kind)] {
			t.Errorf("planning evidence catalogue missing %q: %v", kind, seen)
		}
	}
}

func TestCodexPlanScoutManifestAuthorizesOneStage(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	goal := "Authorize exactly one evidence-grounded Scout"
	state := codexPlanSpecificationFixture(t, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY, Plan: colony.Plan{Phases: []colony.Phase{}},
	}, colony.SpecStatusApproved)
	createTestColonyState(t, dataDir, state)
	writeCodexPlanSpecificationProjection(t, root, state)

	result, err := runCodexPlanWithOptions(root, codexPlanOptions{PlanOnly: true, Preset: "fast", PresetSet: true})
	if err != nil {
		t.Fatal(err)
	}
	decoded := decodeCodexPlanResult(t, result)
	manifest := decoded["plan_manifest"].(map[string]interface{})
	stage, ok := manifest["stage_manifest"].(map[string]interface{})
	if !ok {
		t.Fatalf("stage_manifest missing: %+v", manifest)
	}
	if stage["run_id"] != decoded["planning_run_id"] || int(stage["pass"].(float64)) != 1 {
		t.Fatalf("stage run/pass = %v/%v, want result run and pass 1", stage["run_id"], stage["pass"])
	}
	if stage["expected_caste"] != "scout" || stage["expected_result_type"] != string(planningStageResultScout) {
		t.Fatalf("stage worker contract = %v/%v", stage["expected_caste"], stage["expected_result_type"])
	}
	if stage["preset"] != "fast" || stage["base_plan_revision_id"] == "" || stage["base_plan_revision_hash"] == "" {
		t.Fatalf("stage preset/base binding incomplete: %+v", stage)
	}
	spec := stage["specification"].(map[string]interface{})
	current, _ := currentSpecificationRevision(*state.Specification)
	if spec["revision_id"] != current.ID || spec["content_hash"] != current.ContentHash || spec["status"] != "approved" {
		t.Fatalf("stage specification binding = %+v, want %s/%s", spec, current.ID, current.ContentHash)
	}
	if _, ok := stage["scout_receipt"]; ok {
		t.Fatal("Scout manifest leaked a Route-Setter receipt slot")
	}
	for _, forbidden := range []string{"candidate_snapshot_hash", "acceptance", "activation", "route_setter"} {
		if _, ok := stage[forbidden]; ok {
			t.Errorf("Scout manifest contains forbidden future authority %q", forbidden)
		}
	}
	dispatches := manifest["dispatches"].([]interface{})
	if len(dispatches) != 1 || dispatches[0].(map[string]interface{})["caste"] != "scout" {
		t.Fatalf("dispatch envelope = %+v, want one Scout", dispatches)
	}
	headerPath := filepath.Join(dataDir, "planning", decoded["planning_run_id"].(string), "run-header.json")
	if _, err := os.Stat(headerPath); err != nil {
		t.Fatalf("persisted run header missing: %v", err)
	}
}

func codexPlanSpecificationFixture(t *testing.T, state colony.ColonyState, status colony.SpecRevisionStatus) colony.ColonyState {
	t.Helper()
	currentState, _ := validCurrentPlanningState(t)
	specification := *currentState.Specification
	revision, ok := currentSpecificationRevision(specification)
	if !ok {
		t.Fatal("valid planning-state fixture has no current specification")
	}
	revision.Status = status
	if status == colony.SpecStatusApproved {
		if revision.Approval == nil {
			t.Fatal("approved fixture has no approval receipt")
		}
	} else {
		revision.Approval = nil
	}
	specification.Revisions[len(specification.Revisions)-1] = revision
	state.Specification = &specification
	sessionID := revision.Scope.SessionID
	state.SessionID = &sessionID
	return state
}

func writeCodexPlanSpecificationProjection(t *testing.T, root string, state colony.ColonyState) {
	t.Helper()
	projection, err := renderSpecificationProjection(*state.Specification, state.Plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".aether"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, specificationProjectionRelativePath), projection, 0o644); err != nil {
		t.Fatal(err)
	}
}

func createApprovedCodexPlanTestColony(t *testing.T, dataDir, root string, state colony.ColonyState) colony.ColonyState {
	t.Helper()
	state = codexPlanSpecificationFixture(t, state, colony.SpecStatusApproved)
	createTestColonyState(t, dataDir, state)
	writeCodexPlanSpecificationProjection(t, root, state)
	return state
}

func decodeCodexPlanResult(t *testing.T, result map[string]interface{}) map[string]interface{} {
	t.Helper()
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}

func TestPlanForceRecoversFromStaleInProgress(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	goal := "Force replan from stale state"
	taskID := "1.1"
	fixtureState := codexPlanSpecificationFixture(t, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Stale phase",
					Description: "In progress but no real artifacts",
					Status:      colony.PhaseInProgress,
					Tasks: []colony.Task{
						{ID: &taskID, Goal: "Do the work", Status: colony.TaskInProgress},
					},
				},
			},
		},
	}, colony.SpecStatusApproved)
	createTestColonyState(t, dataDir, fixtureState)
	writeCodexPlanSpecificationProjection(t, root, fixtureState)

	var errBuf bytes.Buffer
	stderr = &errBuf

	rootCmd.SetArgs([]string{"plan", "--force", "--synthetic", "--preset", "balanced"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan --force returned error: %v", err)
	}

	// Should succeed: no completed phases, so force-replan is allowed.
	// The command should not have rejected the request.
	output := errBuf.String()
	if strings.Contains(output, "cannot force-replan") {
		t.Fatalf("force-replan should have been accepted, but got rejection: %s", output)
	}
	// The plan was regenerated (state may have a new plan, but the command succeeded).
	// Verify the colony state is no longer stuck on the stale phase 1.
	var state colony.ColonyState
	stateData, err := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("failed to read state: %v", err)
	}
	if err := json.Unmarshal(stateData, &state); err != nil {
		t.Fatalf("failed to unmarshal state: %v", err)
	}
	if state.State == colony.StateEXECUTING && state.CurrentPhase == 1 {
		t.Fatal("expected state to not remain stuck at EXECUTING phase 1 after force-replan")
	}
}

func TestPlanForcePreservesCompletedPhasesAndRevisesFutureWork(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	goal := "Force replan after completion"
	taskID1 := "1.1"
	taskID2 := "2.1"
	fixtureState := codexPlanSpecificationFixture(t, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 2,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Done phase",
					Description: "Already completed",
					Status:      colony.PhaseCompleted,
					Tasks: []colony.Task{
						{ID: &taskID1, Goal: "Already done", Status: colony.TaskCompleted},
					},
				},
				{
					ID:          2,
					Name:        "Active phase",
					Description: "In progress",
					Status:      colony.PhaseInProgress,
					Tasks: []colony.Task{
						{ID: &taskID2, Goal: "Do the work", Status: colony.TaskInProgress},
					},
				},
			},
		},
	}, colony.SpecStatusApproved)
	createTestColonyState(t, dataDir, fixtureState)
	writeCodexPlanSpecificationProjection(t, root, fixtureState)

	var errBuf bytes.Buffer
	stderr = &errBuf

	rootCmd.SetArgs([]string{"plan", "--force", "--synthetic", "--preset", "balanced", "--revision-type", "user_feedback", "--revision-reason", "The remaining scope changed after phase one"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	if strings.Contains(errBuf.String(), `"ok":false`) {
		t.Fatalf("expected completed-prefix revision to succeed, got: %s", errBuf.String())
	}
	var revised colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &revised); err != nil {
		t.Fatalf("load revised state: %v", err)
	}
	if len(revised.Plan.Phases) < 2 {
		t.Fatalf("revised phase count = %d, want completed prefix plus replacement work", len(revised.Plan.Phases))
	}
	if revised.Plan.Phases[0].Name != "Done phase" || revised.Plan.Phases[0].Status != colony.PhaseCompleted || revised.Plan.Phases[0].Tasks[0].Status != colony.TaskCompleted {
		t.Fatalf("completed phase was not preserved: %+v", revised.Plan.Phases[0])
	}
	if revised.CurrentPhase != 2 || revised.Plan.Phases[1].ID != 2 {
		t.Fatalf("revision did not activate the replacement suffix at phase 2: current=%d phases=%+v", revised.CurrentPhase, revised.Plan.Phases)
	}
	if active, ok := activePlanRevision(revised.Plan); !ok || active.ReasonType != colony.PlanRevisionUserFeedback || len(revised.Plan.Revisions) != 2 {
		t.Fatalf("revision history was not recorded: active=%+v ok=%v history=%+v", active, ok, revised.Plan.Revisions)
	}
}

func TestPlanIncludesClarificationWarningWhenPendingClarificationsExist(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Reuse the current plan carefully"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Existing phase",
					Status: colony.PhaseReady,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Use the existing plan", Status: colony.TaskPending}},
				},
			},
		},
	})
	if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{
		Decisions: []PendingDecision{{
			ID:          "pd_clarify",
			Type:        clarificationDecisionType,
			Description: "Which verification bar do you want?",
			Source:      discussSource("verification", false),
			Resolved:    false,
			CreatedAt:   "2026-04-19T10:00:00Z",
		}},
	}); err != nil {
		t.Fatalf("seed pending decisions: %v", err)
	}

	rootCmd.SetArgs([]string{"plan", "--preset", "balanced"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(stdout.(*bytes.Buffer).Bytes(), &envelope); err != nil {
		t.Fatalf("failed to parse plan output: %v\n%s", err, stdout.(*bytes.Buffer).String())
	}
	result := envelope["result"].(map[string]interface{})
	if got := int(result["unresolved_clarifications"].(float64)); got != 1 {
		t.Fatalf("unresolved_clarifications = %d, want 1", got)
	}
	if warning := stringValue(result["clarification_warning"]); !strings.Contains(warning, "Unresolved clarifications exist") {
		t.Fatalf("expected clarification warning in plan result, got %q", warning)
	}
}

func TestPlanUsesWorkerWrittenArtifactsWhenProvided(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir to test root: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	goal := "Ground the plan in worker artifacts"
	createApprovedCodexPlanTestColony(t, dataDir, root, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &planningArtifactInvoker{} }
	defer func() { newCodexWorkerInvoker = originalInvoker }()

	rootCmd.SetArgs([]string{"plan", "--preset", "balanced"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := result["dispatch_mode"]; got != "real" {
		t.Fatalf("dispatch_mode = %v, want real", got)
	}
	if got := result["artifact_source"]; got != "worker-written" {
		t.Fatalf("artifact_source = %v, want worker-written", got)
	}
	if got := result["plan_source"]; got != "worker-artifact" {
		t.Fatalf("plan_source = %v, want worker-artifact", got)
	}

	phases := result["phases"].([]interface{})
	firstPhase := phases[0].(map[string]interface{})
	if firstPhase["name"] != "Worker planned phase" {
		t.Fatalf("first phase name = %v, want Worker planned phase", firstPhase["name"])
	}

	for _, check := range []struct {
		path string
		want string
	}{
		{filepath.Join(dataDir, "planning", "SCOUT.md"), "worker-authored scout"},
		{filepath.Join(dataDir, "planning", "ROUTE-SETTER.md"), "worker-authored route-setter"},
		{filepath.Join(dataDir, "phase-research", "phase-1-research.md"), "worker-authored phase research"},
	} {
		data, err := os.ReadFile(check.path)
		if err != nil {
			t.Fatalf("read %s: %v", filepath.Base(check.path), err)
		}
		if !strings.Contains(string(data), check.want) {
			t.Fatalf("expected %s to be preserved, got:\n%s", filepath.Base(check.path), string(data))
		}
	}
}

func TestPlanDirectRealBelowTargetPersistsOnlyIntermediateIteration(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-low-confidence-plan\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	goal := "Keep real planning iterative until confidence target"
	createApprovedCodexPlanTestColony(t, dataDir, root, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &planningLowConfidenceArtifactInvoker{} }
	defer func() { newCodexWorkerInvoker = originalInvoker }()

	rootCmd.SetArgs([]string{"plan", "--target", "90", "--max-iterations", "6"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if result["planned"] != false || result["requires_next_iteration"] != true {
		t.Fatalf("below-target real plan should require next iteration, got %+v", result)
	}
	if next := stringValue(result["next"]); !strings.Contains(next, "aether host plan") {
		t.Fatalf("next = %q, want host plan iteration command", next)
	}
	loop := result["planning_loop"].(map[string]interface{})
	if got := loop["stop_reason"].(string); got != planningLoopPendingStop {
		t.Fatalf("stop_reason = %q, want pending", got)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if len(state.Plan.Phases) != 0 {
		t.Fatalf("pending planning iteration must not commit final phases: %+v", state.Plan.Phases)
	}
	if _, err := os.Stat(filepath.Join(dataDir, planningIterationStateRel)); err != nil {
		t.Fatalf("expected planning iteration state: %v", err)
	}
}

func TestPlanUsesScoutReportReturnedByScoutWorker(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	goal := "Stabilize Aether multi-platform lifecycle reliability across Claude Code, OpenCode, and Codex CLI"
	createApprovedCodexPlanTestColony(t, dataDir, root, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	invoker := &planningScoutReportInvoker{}
	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	defer func() { newCodexWorkerInvoker = originalInvoker }()

	rootCmd.SetArgs([]string{"plan", "--preset", "balanced"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := result["dispatch_mode"]; got != "real" {
		t.Fatalf("dispatch_mode = %v, want real", got)
	}

	if len(invoker.briefs) != 2 {
		t.Fatalf("expected 2 planning briefs, got %d", len(invoker.briefs))
	}
	if !strings.Contains(invoker.briefs[1], "Scout preserved real direct-runtime finding") {
		t.Fatalf("route-setter brief did not receive scout guidance:\n%s", invoker.briefs[1])
	}

	scoutData, err := os.ReadFile(filepath.Join(dataDir, "planning", "SCOUT.md"))
	if err != nil {
		t.Fatalf("read SCOUT.md: %v", err)
	}
	if !strings.Contains(string(scoutData), "Scout preserved real direct-runtime finding") {
		t.Fatalf("SCOUT.md did not preserve worker scout report:\n%s", string(scoutData))
	}

	researchData, err := os.ReadFile(filepath.Join(dataDir, "phase-research", "phase-1-research.md"))
	if err != nil {
		t.Fatalf("read phase research: %v", err)
	}
	if !strings.Contains(string(researchData), "Scout preserved real direct-runtime finding") {
		t.Fatalf("phase research did not use worker scout report:\n%s", string(researchData))
	}
}

func TestScoutReportFromWorkerResultReadsArtifactsFallback(t *testing.T) {
	raw := json.RawMessage(`{
		"findings": [
			{"area": "planning", "discovery": "artifact fallback survives", "source": "test"}
		],
		"gaps": [],
		"confidence": 87,
		"study_files": ["cmd/codex_plan.go"]
	}`)
	report, ok := scoutReportFromWorkerResult(&codex.WorkerResult{
		Artifacts: map[string]json.RawMessage{"scout_report": raw},
	})
	if !ok {
		t.Fatal("expected scout report from artifacts fallback")
	}
	if got := report.Findings[0].Discovery; got != "artifact fallback survives" {
		t.Fatalf("discovery = %q, want artifact fallback survives", got)
	}
}

func TestPlanUsesFreshWorkerPlanArtifactWhenTimestampDoesNotAdvance(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir to test root: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	goal := "Stabilize Aether multi-platform lifecycle reliability across Claude Code, OpenCode, and Codex CLI"
	createApprovedCodexPlanTestColony(t, dataDir, root, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	stalePlanPath := filepath.Join(dataDir, "planning", "phase-plan.json")
	if err := os.MkdirAll(filepath.Dir(stalePlanPath), 0755); err != nil {
		t.Fatalf("create planning dir: %v", err)
	}
	stalePlan := []byte(`{"phases":[{"name":"Research charter and communication target","description":"Wrong stale language plan.","tasks":[{"goal":"Design a stale protocol"}]}],"confidence":{"overall":91}}`)
	if err := os.WriteFile(stalePlanPath, stalePlan, 0644); err != nil {
		t.Fatalf("write stale plan: %v", err)
	}
	staleTime := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(stalePlanPath, staleTime, staleTime); err != nil {
		t.Fatalf("set stale plan time: %v", err)
	}

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &planningSameTimestampInvoker{} }
	defer func() { newCodexWorkerInvoker = originalInvoker }()

	rootCmd.SetArgs([]string{"plan", "--preset", "balanced"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := result["plan_source"]; got != "worker-artifact" {
		t.Fatalf("plan_source = %v, want worker-artifact", got)
	}
	phases := result["phases"].([]interface{})
	firstPhase := phases[0].(map[string]interface{})
	if firstPhase["name"] != "Host dispatch fails closed" {
		t.Fatalf("first phase name = %v, want Host dispatch fails closed", firstPhase["name"])
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if len(state.Plan.Phases) == 0 {
		t.Fatal("expected phases")
	}
	if state.Plan.Phases[0].Name == "Research charter and communication target" {
		t.Fatalf("stale language plan leaked into committed state: %+v", state.Plan.Phases[0])
	}
}

func TestPlanFailsClosedWhenRealPlanningDispatchFails(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir to test root: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	goal := "Stop honestly when planner workers stall"
	createApprovedCodexPlanTestColony(t, dataDir, root, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &failingPlanningInvoker{} }
	defer func() { newCodexWorkerInvoker = originalInvoker }()

	rootCmd.SetArgs([]string{"plan", "--preset", "balanced"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	env := parseEnvelope(t, stderr.(*bytes.Buffer).String())
	if env["ok"] != false {
		t.Fatalf("expected error envelope, got %#v", env)
	}
	errText := stringValue(env["error"])
	for _, want := range []string{"real planning workers did not finish cleanly", "aether plan --synthetic"} {
		if !strings.Contains(errText, want) {
			t.Fatalf("error = %q, want %q", errText, want)
		}
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if len(state.Plan.Phases) != 0 {
		t.Fatalf("normal planning failure must not commit fallback phases: %+v", state.Plan.Phases)
	}
}

func TestPlanFailsClosedWhenPlanningWorkersUnavailableBeforeStateMutation(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Stop before local planning when provider is unavailable"
	createApprovedCodexPlanTestColony(t, dataDir, root, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &planTestUnavailableInvoker{} }
	defer func() { newCodexWorkerInvoker = originalInvoker }()

	rootCmd.SetArgs([]string{"plan", "--preset", "balanced"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	env := parseEnvelope(t, stderr.(*bytes.Buffer).String())
	if env["ok"] != false {
		t.Fatalf("expected error envelope, got %#v", env)
	}
	errText := stringValue(env["error"])
	for _, want := range []string{"real planning workers unavailable", "aether plan --synthetic"} {
		if !strings.Contains(errText, want) {
			t.Fatalf("error = %q, want %q", errText, want)
		}
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if len(state.Plan.Phases) != 0 {
		t.Fatalf("provider-unavailable planning must not commit phases: %+v", state.Plan.Phases)
	}
	if strings.TrimSpace(state.VerificationDepth) != "" {
		t.Fatalf("provider-unavailable planning should not persist verification depth, got %q", state.VerificationDepth)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "planning")); !os.IsNotExist(err) {
		t.Fatalf("provider-unavailable planning should not create planning dir, stat error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "spawn-tree.txt")); !os.IsNotExist(err) {
		t.Fatalf("provider-unavailable planning should not record spawn tree, stat error: %v", err)
	}
}

func TestPlanSyntheticForLanguageDesignGoalIsGoalAware(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Create Soliditas, a language for AI-to-AI communication with better token and context efficiency"
	createApprovedCodexPlanTestColony(t, dataDir, root, colony.ColonyState{
		Version:         "3.0",
		Goal:            &goal,
		State:           colony.StateREADY,
		PlanGranularity: colony.GranularityMilestone,
		Plan:            colony.Plan{Phases: []colony.Phase{}},
	})

	rootCmd.SetArgs([]string{"plan", "--synthetic", "--preset", "balanced"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := result["dispatch_mode"]; got != "synthetic" {
		t.Fatalf("dispatch_mode = %v, want synthetic", got)
	}
	if got := result["synthetic"]; got != true {
		t.Fatalf("synthetic = %v, want true", got)
	}
	if got := strings.TrimSpace(stringValue(result["synthetic_warning"])); got == "" {
		t.Fatal("expected synthetic_warning to be populated")
	}
	if got := int(result["count"].(float64)); got < 4 {
		t.Fatalf("synthetic count = %d, want at least 4 phases for milestone granularity", got)
	}

	phases := result["phases"].([]interface{})
	names := make([]string, 0, len(phases))
	blob := ""
	for _, raw := range phases {
		phase := raw.(map[string]interface{})
		name := phase["name"].(string)
		names = append(names, name)
		blob += name + "\n"
		tasks := phase["tasks"].([]interface{})
		for _, taskRaw := range tasks {
			task := taskRaw.(map[string]interface{})
			blob += task["goal"].(string) + "\n"
		}
	}

	for _, want := range []string{
		"Research charter and communication target",
		"Representation and grammar design",
		"Reference prototype and translation path",
		"Evaluation and next design loop",
		"communication problem",
		"grammar",
		"prototype",
	} {
		if !strings.Contains(blob, want) {
			t.Fatalf("goal-aware synthetic plan missing %q\nphase names: %v\n%s", want, names, blob)
		}
	}

	for _, unwanted := range []string{"Discovery and boundaries", "Implementation", "Verification and polish"} {
		if strings.Contains(blob, unwanted) {
			t.Fatalf("goal-aware synthetic plan should not collapse to generic template %q\nphase names: %v\n%s", unwanted, names, blob)
		}
	}
}

func TestPlanSyntheticDefaultMilestoneUsesArchitecturePhase(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/feature-fallback-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	goal := "Ship a safer project update flow"
	createApprovedCodexPlanTestColony(t, dataDir, root, colony.ColonyState{
		Version:         "3.0",
		Goal:            &goal,
		State:           colony.StateREADY,
		PlanGranularity: colony.GranularityMilestone,
		Plan:            colony.Plan{Phases: []colony.Phase{}},
	})

	rootCmd.SetArgs([]string{"plan", "--synthetic", "--preset", "balanced"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := int(result["count"].(float64)); got < 4 {
		t.Fatalf("synthetic count = %d, want at least 4 phases for milestone granularity", got)
	}

	phases := result["phases"].([]interface{})
	names := make([]string, 0, len(phases))
	for _, raw := range phases {
		phase := raw.(map[string]interface{})
		names = append(names, phase["name"].(string))
	}

	for _, want := range []string{"Discovery and boundaries", "Architecture and interfaces", "Implementation", "Verification and polish"} {
		if !containsString(names, want) {
			t.Fatalf("fallback phase list missing %q: %v", want, names)
		}
	}
}

func TestPlanSyntheticDoesNotTreatCommandCenterAsAetherCommandWork(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Redesign the dashboard as a premium personal command center"
	createApprovedCodexPlanTestColony(t, dataDir, root, colony.ColonyState{
		Version:         "3.0",
		Goal:            &goal,
		State:           colony.StateREADY,
		PlanGranularity: colony.GranularityMilestone,
		Plan:            colony.Plan{Phases: []colony.Phase{}},
	})

	rootCmd.SetArgs([]string{"plan", "--synthetic", "--preset", "balanced"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := result["dispatch_mode"]; got != "synthetic" {
		t.Fatalf("dispatch_mode = %v, want synthetic", got)
	}

	blob := ""
	for _, raw := range result["phases"].([]interface{}) {
		phase := raw.(map[string]interface{})
		blob += phase["name"].(string) + "\n"
		for _, taskRaw := range phase["tasks"].([]interface{}) {
			task := taskRaw.(map[string]interface{})
			blob += task["goal"].(string) + "\n"
		}
	}

	for _, unwanted := range []string{
		"Contract and gap mapping",
		"Colonize orchestration",
		"Planning orchestration",
		"Build orchestration",
		"Continue orchestration",
		"Codex command behavior",
		"ant workflow",
	} {
		if strings.Contains(blob, unwanted) {
			t.Fatalf("command-center synthetic plan should not produce Aether orchestration plan containing %q:\n%s", unwanted, blob)
		}
	}
	if strings.TrimSpace(blob) == "" {
		t.Fatal("expected command-center synthetic plan to produce non-empty non-Aether plan")
	}
}

func TestPlanFallbackStillSupportsExplicitAetherCommandWork(t *testing.T) {
	templates := planningTemplates("Fix Codex command orchestration parity for Aether workers", codexSurveyContext{}, codexScoutReport{})
	if len(templates) == 0 {
		t.Fatal("expected templates")
	}
	if templates[0].Name != "Contract and gap mapping" {
		t.Fatalf("first template = %q, want Contract and gap mapping", templates[0].Name)
	}
}

func TestPlanFallbackTreatsLifecycleReliabilityAsAetherWork(t *testing.T) {
	goal := "Stabilize Aether multi-platform lifecycle reliability across Claude Code, OpenCode, and Codex CLI"
	templates := planningTemplates(goal, codexSurveyContext{}, codexScoutReport{})
	if len(templates) == 0 {
		t.Fatal("expected templates")
	}
	if templates[0].Name != "Contract and gap mapping" {
		t.Fatalf("first template = %q, want Contract and gap mapping", templates[0].Name)
	}

	blob := ""
	for _, phase := range templates {
		blob += phase.Name + "\n" + phase.Description + "\n"
		for _, task := range phase.Tasks {
			blob += task.Goal + "\n"
		}
	}
	for _, unwanted := range []string{
		"Research charter and communication target",
		"Representation and grammar design",
		"communication problem",
		"semantic primitives",
	} {
		if strings.Contains(blob, unwanted) {
			t.Fatalf("lifecycle reliability fallback should not produce language-design plan containing %q:\n%s", unwanted, blob)
		}
	}
}

func TestPlannedPlanningWorkersUsesClassicScoutRouteSetterPair(t *testing.T) {
	saveGlobals(t)
	setupRuntimeSkillAssignmentHub(t)

	root := t.TempDir()
	dispatches := plannedPlanningWorkersForGoal(root, "Plan secure auth token rotation")

	for _, caste := range []string{"scout", "route_setter"} {
		if !planningDispatchHasCaste(dispatches, caste) {
			t.Fatalf("planned planning dispatches missing classic %s: %+v", caste, dispatches)
		}
	}
	if planningDispatchHasCaste(dispatches, "gatekeeper") {
		t.Fatalf("classic planning should not add Gatekeeper to the Scout/Route-Setter pair: %+v", dispatches)
	}
	if len(dispatches) != 2 {
		t.Fatalf("planning dispatch count = %d, want Scout + Route-Setter only: %+v", len(dispatches), dispatches)
	}
}

// --- dispatchRealPlanningWorkers tests ---

func TestDispatchRealPlanningWorkers_NilInvoker_ReturnsNil(t *testing.T) {
	result, err := dispatchRealPlanningWorkers(context.Background(), "/tmp/test-repo", nil)
	if err != nil {
		t.Fatalf("expected nil error for nil invoker, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result for nil invoker, got: %+v", result)
	}
}

func TestDispatchRealPlanningWorkers_UnavailableInvoker_ReturnsNil(t *testing.T) {
	// Use a custom invoker that reports unavailable (separate type to avoid redeclaration).
	unavailable := &planTestUnavailableInvoker{}
	result, err := dispatchRealPlanningWorkers(context.Background(), "/tmp/test-repo", unavailable)
	if err != nil {
		t.Fatalf("expected nil error for unavailable invoker, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result for unavailable invoker, got: %+v", result)
	}
}

func TestDispatchRealPlanningWorkers_AvailableInvoker_ReturnsDispatches(t *testing.T) {
	tmpDir := t.TempDir()
	codexAgentsDir := filepath.Join(tmpDir, ".codex", "agents")
	if err := os.MkdirAll(codexAgentsDir, 0755); err != nil {
		t.Fatalf("failed to create .codex/agents: %v", err)
	}
	for _, name := range []string{"aether-scout.toml", "aether-route-setter.toml"} {
		if err := os.WriteFile(filepath.Join(codexAgentsDir, name), []byte(`name = "test"
description = "test agent"
developer_instructions = "test instructions"`), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	invoker := &codex.FakeInvoker{}
	result, err := dispatchRealPlanningWorkers(context.Background(), tmpDir, invoker)
	if err != nil {
		t.Fatalf("expected nil error for available invoker, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for available invoker")
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 dispatches, got %d", len(result))
	}
	if result[0].Caste != "scout" {
		t.Fatalf("expected first dispatch caste 'scout', got %q", result[0].Caste)
	}
	if result[1].Caste != "route_setter" {
		t.Fatalf("expected second dispatch caste 'route_setter', got %q", result[1].Caste)
	}
	if result[0].Status != "completed" {
		t.Fatalf("expected first dispatch status 'completed', got %q", result[0].Status)
	}
	if result[1].Status != "completed" {
		t.Fatalf("expected second dispatch status 'completed', got %q", result[1].Status)
	}
}

func planningDispatchHasCaste(dispatches []codexPlanningDispatch, caste string) bool {
	for _, dispatch := range dispatches {
		if dispatch.Caste == caste {
			return true
		}
	}
	return false
}

func TestDispatchRealPlanningWorkers_UsesTimeoutOverrideAndSurveyFirstBrief(t *testing.T) {
	tmpDir := t.TempDir()
	codexAgentsDir := filepath.Join(tmpDir, ".codex", "agents")
	if err := os.MkdirAll(codexAgentsDir, 0755); err != nil {
		t.Fatalf("failed to create .codex/agents: %v", err)
	}
	for _, name := range []string{"aether-scout.toml", "aether-route-setter.toml"} {
		if err := os.WriteFile(filepath.Join(codexAgentsDir, name), []byte(`name = "test"
description = "test agent"
developer_instructions = "test instructions"`), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	invoker := &planningCaptureInvoker{}
	override := 7 * time.Minute
	survey := codexSurveyContext{
		SurveyDocs: []string{"PROVISIONS.md", "BLUEPRINT.md"},
	}

	result, err := dispatchRealPlanningWorkersWithTimeout(context.Background(), tmpDir, survey, invoker, override)
	if err != nil {
		t.Fatalf("expected nil error for available invoker, got: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 dispatches, got %d", len(result))
	}
	if len(invoker.timeouts) != 2 {
		t.Fatalf("expected 2 recorded timeouts, got %d", len(invoker.timeouts))
	}
	for i, got := range invoker.timeouts {
		if got != override {
			t.Fatalf("timeout[%d] = %s, want %s", i, got, override)
		}
	}
	if len(invoker.briefs) != 2 {
		t.Fatalf("expected 2 recorded briefs, got %d", len(invoker.briefs))
	}
	for _, want := range []string{
		".aether/data/survey/",
		".aether/backups/",
		".aether/chambers/",
		".aether/data/build/",
		".git/",
		"node_modules/",
		"Scout is read-only",
		"do not spawn subagents",
	} {
		if !strings.Contains(invoker.briefs[0], want) {
			t.Fatalf("scout brief missing %q:\n%s", want, invoker.briefs[0])
		}
	}
	if strings.Contains(invoker.briefs[0], "Write planning outputs directly into the repository.") {
		t.Fatalf("scout brief must not ask read-only Scout to write files:\n%s", invoker.briefs[0])
	}
	if !strings.Contains(invoker.briefs[1], ".aether/data/planning/SCOUT.md") {
		t.Fatalf("route-setter brief missing scout artifact guidance:\n%s", invoker.briefs[1])
	}
	if !strings.Contains(invoker.briefs[1], "if it is missing, proceed from the survey context") {
		t.Fatalf("route-setter brief should not hard-block on missing scout artifact:\n%s", invoker.briefs[1])
	}
}

// planTestUnavailableInvoker is a WorkerInvoker that always reports unavailable.
type planTestUnavailableInvoker struct{}

func (u *planTestUnavailableInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	return codex.WorkerResult{}, nil
}

func (u *planTestUnavailableInvoker) IsAvailable(ctx context.Context) bool {
	return false
}

func (u *planTestUnavailableInvoker) ValidateAgent(path string) error {
	return nil
}

type planningCaptureInvoker struct {
	timeouts []time.Duration
	briefs   []string
}

func (p *planningCaptureInvoker) Invoke(_ context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	p.timeouts = append(p.timeouts, config.Timeout)
	p.briefs = append(p.briefs, config.TaskBrief)
	return codex.WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "completed",
		Summary:    "captured planning dispatch",
	}, nil
}

func (p *planningCaptureInvoker) IsAvailable(_ context.Context) bool {
	return true
}

func (p *planningCaptureInvoker) ValidateAgent(_ string) error {
	return nil
}

type planningScoutReportInvoker struct {
	briefs []string
}

func (p *planningScoutReportInvoker) Invoke(_ context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	p.briefs = append(p.briefs, config.TaskBrief)
	result := codex.WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "completed",
		Summary:    "returned scout planning report",
	}
	if config.Caste == "scout" {
		result.ScoutReport = json.RawMessage(`{
			"findings": [
				{
					"area": "runtime planning",
					"discovery": "Scout preserved real direct-runtime finding",
					"source": "cmd/codex_plan.go"
				}
			],
			"gaps": ["Route-Setter needs the Scout result before drafting phases"],
			"confidence": 91,
			"study_files": ["cmd/codex_plan.go"]
		}`)
	}
	return result, nil
}

func (p *planningScoutReportInvoker) IsAvailable(_ context.Context) bool {
	return true
}

func (p *planningScoutReportInvoker) ValidateAgent(_ string) error {
	return nil
}

type planningArtifactInvoker struct{}

func (p *planningArtifactInvoker) Invoke(_ context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	result := codex.WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "completed",
		Summary:    "worker-authored planning artifact",
	}
	switch config.Caste {
	case "scout":
		target := filepath.Join(config.Root, ".aether", "data", "planning", "SCOUT.md")
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return codex.WorkerResult{}, err
		}
		if err := os.WriteFile(target, []byte("# worker-authored scout\n"), 0644); err != nil {
			return codex.WorkerResult{}, err
		}
		result.FilesCreated = []string{filepath.ToSlash(filepath.Join(".aether", "data", "planning", "SCOUT.md"))}
	case "route_setter":
		planningDir := filepath.Join(config.Root, ".aether", "data", "planning")
		if err := os.MkdirAll(planningDir, 0755); err != nil {
			return codex.WorkerResult{}, err
		}
		if err := os.WriteFile(filepath.Join(planningDir, "ROUTE-SETTER.md"), []byte("# worker-authored route-setter\n"), 0644); err != nil {
			return codex.WorkerResult{}, err
		}
		planArtifact := `{
  "phases": [
    {
      "name": "Worker planned phase",
      "description": "Phase loaded from route-setter artifact.",
      "tasks": [
        {
          "goal": "Land the worker-authored planning flow",
          "constraints": ["Keep plan artifact authoritative"],
          "hints": ["cmd/codex_plan.go"],
		  "success_criteria": ["The worker plan is applied"],
		  "evidence_requirements": [{"criterion":"The worker plan is applied","checks":["claims","watcher"]}],
		  "depends_on": []
        }
      ],
	      "success_criteria": ["Worker route-setter plan used"],
	      "evidence_requirements": [{"criterion":"Worker route-setter plan used","checks":["claims","watcher"]}]
    },
    {
      "name": "Verification",
      "description": "Verify the worker-authored plan.",
      "tasks": [
        {
          "goal": "Confirm the worker plan survives serialization",
          "constraints": [],
          "hints": ["cmd/codex_plan_test.go"],
		  "success_criteria": ["Regression coverage exists"],
		  "evidence_requirements": [{"criterion":"Regression coverage exists","checks":["tests","watcher"]}],
		  "depends_on": ["1.1"]
        }
      ],
	      "success_criteria": ["Plan verification ready"],
	      "evidence_requirements": [{"criterion":"Plan verification ready","checks":["claims","watcher"]}]
    }
  ],
  "confidence": {
    "knowledge": 92,
    "requirements": 91,
    "risks": 90,
    "dependencies": 90,
    "effort": 91,
    "overall": 91
  },
  "gaps": ["Worker identified one remaining follow-up."]
}`
		if err := os.WriteFile(filepath.Join(planningDir, "phase-plan.json"), []byte(planArtifact), 0644); err != nil {
			return codex.WorkerResult{}, err
		}

		researchDir := filepath.Join(config.Root, ".aether", "data", "phase-research")
		if err := os.MkdirAll(researchDir, 0755); err != nil {
			return codex.WorkerResult{}, err
		}
		if err := os.WriteFile(filepath.Join(researchDir, "phase-1-research.md"), []byte("# worker-authored phase research\n"), 0644); err != nil {
			return codex.WorkerResult{}, err
		}

		result.FilesCreated = []string{
			filepath.ToSlash(filepath.Join(".aether", "data", "planning", "ROUTE-SETTER.md")),
			filepath.ToSlash(filepath.Join(".aether", "data", "planning", "phase-plan.json")),
			filepath.ToSlash(filepath.Join(".aether", "data", "phase-research", "phase-1-research.md")),
		}
	}
	return result, nil
}

func (p *planningArtifactInvoker) IsAvailable(_ context.Context) bool {
	return true
}

func (p *planningArtifactInvoker) ValidateAgent(_ string) error {
	return nil
}

type planningLowConfidenceArtifactInvoker struct{}

func (p *planningLowConfidenceArtifactInvoker) Invoke(_ context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	result := codex.WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "completed",
		Summary:    "worker-authored low-confidence planning artifact",
	}
	switch config.Caste {
	case "scout":
		result.ScoutReport = json.RawMessage(`{
			"findings": [
				{
					"area": "planning loop",
					"discovery": "Scout found enough evidence for a draft but not enough to hit the target",
					"source": "cmd/codex_plan_test.go"
				}
			],
			"gaps": ["Route-Setter needs one more implementation-contract pass"],
			"confidence": 84,
			"study_files": ["cmd/codex_plan.go"]
		}`)
	case "route_setter":
		planningDir := filepath.Join(config.Root, ".aether", "data", "planning")
		if err := os.MkdirAll(planningDir, 0755); err != nil {
			return codex.WorkerResult{}, err
		}
		planArtifact := `{
  "phases": [
    {
      "name": "Pending real planning draft",
      "description": "Draft that must not become the colony plan before another iteration.",
      "tasks": [
        {
          "goal": "Gather the missing implementation-contract evidence",
          "constraints": ["Do not finalize below target without a stop condition"],
          "hints": ["cmd/codex_plan.go"],
          "success_criteria": ["The next iteration has targeted gap evidence"],
          "depends_on": []
        }
      ],
      "success_criteria": ["The draft remains intermediate"]
    }
  ],
  "confidence": {
    "knowledge": 84,
    "requirements": 84,
    "risks": 84,
    "dependencies": 84,
    "effort": 84,
    "overall": 84
  },
  "gaps": ["Route-Setter needs one more implementation-contract pass"]
}`
		if err := os.WriteFile(filepath.Join(planningDir, "phase-plan.json"), []byte(planArtifact), 0644); err != nil {
			return codex.WorkerResult{}, err
		}
		result.FilesCreated = []string{
			filepath.ToSlash(filepath.Join(".aether", "data", "planning", "phase-plan.json")),
		}
	}
	return result, nil
}

func (p *planningLowConfidenceArtifactInvoker) IsAvailable(_ context.Context) bool {
	return true
}

func (p *planningLowConfidenceArtifactInvoker) ValidateAgent(_ string) error {
	return nil
}

type planningSameTimestampInvoker struct{}

func (p *planningSameTimestampInvoker) Invoke(_ context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	result := codex.WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "completed",
		Summary:    "worker-authored planning artifact with unchanged timestamp",
	}
	if config.Caste != "route_setter" {
		return result, nil
	}

	planningDir := filepath.Join(config.Root, ".aether", "data", "planning")
	if err := os.MkdirAll(planningDir, 0755); err != nil {
		return codex.WorkerResult{}, err
	}
	target := filepath.Join(planningDir, "phase-plan.json")
	preserveTime := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	if info, err := os.Stat(target); err == nil {
		preserveTime = info.ModTime()
	}
	planArtifact := `{
  "phases": [
    {
      "name": "Host dispatch fails closed",
      "description": "Route host-specific spawning through explicit boundaries before lifecycle state changes.",
      "tasks": [
        {
          "goal": "Make planner artifacts from active workers authoritative even when timestamps do not advance",
          "constraints": ["Do not fall back to stale synthetic plans"],
          "hints": ["cmd/codex_plan.go", "cmd/codex_worker_artifacts.go"],
		  "success_criteria": ["Worker plan artifact is committed to colony state"],
		  "evidence_requirements": [{"criterion":"Worker plan artifact is committed to colony state","checks":["claims","watcher"]}],
		  "depends_on": []
        }
      ],
	      "success_criteria": ["Fresh worker-authored plan is selected"],
	      "evidence_requirements": [{"criterion":"Fresh worker-authored plan is selected","checks":["claims","watcher"]}]
    }
  ],
  "confidence": {
    "knowledge": 92,
    "requirements": 90,
    "risks": 90,
    "dependencies": 91,
    "effort": 90,
    "overall": 91
  },
  "gaps": []
}`
	if err := os.WriteFile(target, []byte(planArtifact), 0644); err != nil {
		return codex.WorkerResult{}, err
	}
	if err := os.Chtimes(target, preserveTime, preserveTime); err != nil {
		return codex.WorkerResult{}, err
	}
	return result, nil
}

func (p *planningSameTimestampInvoker) IsAvailable(_ context.Context) bool {
	return true
}

func (p *planningSameTimestampInvoker) ValidateAgent(_ string) error {
	return nil
}

type failingPlanningInvoker struct{}

func (f *failingPlanningInvoker) Invoke(_ context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	return codex.WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "timeout",
		Error:      context.DeadlineExceeded,
	}, nil
}

func (f *failingPlanningInvoker) IsAvailable(_ context.Context) bool {
	return true
}

func (f *failingPlanningInvoker) ValidateAgent(_ string) error {
	return nil
}

func TestDispatchRealPlanningWorkers_CancelledContext_ReturnsTimeoutError(t *testing.T) {
	tmpDir := t.TempDir()
	codexAgentsDir := filepath.Join(tmpDir, ".codex", "agents")
	if err := os.MkdirAll(codexAgentsDir, 0755); err != nil {
		t.Fatalf("failed to create .codex/agents: %v", err)
	}
	for _, name := range []string{"aether-scout.toml", "aether-route-setter.toml"} {
		if err := os.WriteFile(filepath.Join(codexAgentsDir, name), []byte(`name = "test"
description = "test agent"
developer_instructions = "test instructions"`), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	invoker := &codex.FakeInvoker{}
	result, err := dispatchRealPlanningWorkers(ctx, tmpDir, invoker)
	if err == nil {
		t.Fatal("expected timeout error for cancelled context")
	}
	if result == nil {
		t.Fatal("expected non-nil result for cancelled context")
	}
}

func TestClearFallbackPlanningArtifactsRemovesStaleFallbackArtifacts(t *testing.T) {
	root := t.TempDir()
	planningDir := filepath.Join(root, ".aether", "data", "planning")
	researchDir := filepath.Join(root, ".aether", "data", "phase-research")
	markerTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	writeTestFileAtTime(t, filepath.Join(planningDir, ".fallback-marker"), "fallback", markerTime)
	for _, rel := range []string{
		filepath.Join("planning", "SCOUT.md"),
		filepath.Join("planning", "ROUTE-SETTER.md"),
		filepath.Join("planning", "phase-plan.json"),
		filepath.Join("planning", "phase-plan.json.bak"),
		filepath.Join("phase-research", "phase-1-research.md"),
		filepath.Join("phase-research", "phase-2-research.md.bak"),
	} {
		writeTestFileAtTime(t, filepath.Join(root, ".aether", "data", rel), "stale fallback", markerTime.Add(-time.Minute))
	}

	if err := clearFallbackPlanningArtifacts(root); err != nil {
		t.Fatal(err)
	}

	for _, rel := range []string{
		filepath.Join("planning", ".fallback-marker"),
		filepath.Join("planning", "SCOUT.md"),
		filepath.Join("planning", "ROUTE-SETTER.md"),
		filepath.Join("planning", "phase-plan.json"),
		filepath.Join("planning", "phase-plan.json.bak"),
		filepath.Join("phase-research", "phase-1-research.md"),
		filepath.Join("phase-research", "phase-2-research.md.bak"),
	} {
		if _, err := os.Stat(filepath.Join(root, ".aether", "data", rel)); !os.IsNotExist(err) {
			t.Fatalf("expected stale fallback artifact %s to be removed, got %v", rel, err)
		}
	}
	if _, err := os.Stat(researchDir); err != nil {
		t.Fatalf("expected phase-research directory to remain: %v", err)
	}
}

func TestClearFallbackPlanningArtifactsPreservesNewerWorkerFiles(t *testing.T) {
	root := t.TempDir()
	planningDir := filepath.Join(root, ".aether", "data", "planning")
	markerTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	workerTime := markerTime.Add(time.Hour)

	writeTestFileAtTime(t, filepath.Join(planningDir, ".fallback-marker"), "fallback", markerTime)
	for _, rel := range []string{
		filepath.Join("planning", "SCOUT.md"),
		filepath.Join("planning", "ROUTE-SETTER.md"),
		filepath.Join("planning", "phase-plan.json"),
		filepath.Join("phase-research", "phase-1-research.md"),
	} {
		writeTestFileAtTime(t, filepath.Join(root, ".aether", "data", rel), "worker-authored", workerTime)
	}
	writeTestFileAtTime(t, filepath.Join(root, ".aether", "data", "phase-research", "phase-2-research.md"), "stale fallback", markerTime.Add(-time.Minute))

	if err := clearFallbackPlanningArtifacts(root); err != nil {
		t.Fatal(err)
	}

	for _, rel := range []string{
		filepath.Join("planning", "SCOUT.md"),
		filepath.Join("planning", "ROUTE-SETTER.md"),
		filepath.Join("planning", "phase-plan.json"),
		filepath.Join("phase-research", "phase-1-research.md"),
	} {
		data, err := os.ReadFile(filepath.Join(root, ".aether", "data", rel))
		if err != nil {
			t.Fatalf("expected newer worker artifact %s to be preserved: %v", rel, err)
		}
		if string(data) != "worker-authored" {
			t.Fatalf("expected newer worker artifact %s content to survive, got %q", rel, string(data))
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".aether", "data", "phase-research", "phase-2-research.md")); !os.IsNotExist(err) {
		t.Fatalf("expected stale phase research to be removed, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(planningDir, ".fallback-marker")); !os.IsNotExist(err) {
		t.Fatalf("expected fallback marker to be removed, got %v", err)
	}

	t.Run("transaction rollback restores every captured cleanup baseline", func(t *testing.T) {
		rollbackRoot := t.TempDir()
		dataRoot := filepath.Join(rollbackRoot, ".aether", "data")
		planningRoot := filepath.Join(dataRoot, "planning")
		observedAt := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)

		writeTestFileAtTime(t, filepath.Join(planningRoot, ".fallback-marker"), observedAt.Format(time.RFC3339), observedAt)
		for rel, content := range map[string]string{
			filepath.Join("planning", "ROUTE-SETTER.md"):           "stale route",
			filepath.Join("planning", "phase-plan.json"):           `{"stale":true}`,
			filepath.Join("planning", "phase-plan.json.bak"):       `{"backup":true}`,
			filepath.Join("phase-research", "phase-1-research.md"): "stale research",
		} {
			writeTestFileAtTime(t, filepath.Join(dataRoot, rel), content, observedAt.Add(-time.Minute))
		}
		newerResearch := filepath.Join(dataRoot, "phase-research", "phase-2-research.md")
		writeTestFileAtTime(t, newerResearch, "worker-authored research", observedAt.Add(time.Minute))

		baselines := make(map[string]lifecycleFileState)
		injected := errors.New("injected fallback cleanup failure")
		err := withPlanningMutationSession(rollbackRoot, "test-fallback-cleanup-rollback", func(session *planningMutationSession) error {
			targets, err := fallbackPlanningArtifactRemovalTargetsInSession(session)
			if err != nil {
				return err
			}
			expected := map[string]bool{
				"phase-research/phase-1-research.md": true,
				"planning/.fallback-marker":          true,
				"planning/ROUTE-SETTER.md":           true,
				"planning/SCOUT.md":                  true,
				"planning/phase-plan.json":           true,
				"planning/phase-plan.json.bak":       true,
			}
			for _, target := range targets {
				if !expected[target.Path] {
					return fmt.Errorf("unexpected fallback cleanup target %q", target.Path)
				}
				delete(expected, target.Path)
				baseline, err := session.capturedBaseline(target.Root, target.Path)
				if err != nil {
					return fmt.Errorf("cleanup target %s was not captured before classification: %w", target.Path, err)
				}
				baselines[target.Path] = baseline
			}
			if len(expected) != 0 {
				return fmt.Errorf("fallback cleanup omitted targets: %v", expected)
			}

			sort.Slice(targets, func(left, right int) bool {
				if targets[left].Root != targets[right].Root {
					return targets[left].Root < targets[right].Root
				}
				return targets[left].Path < targets[right].Path
			})
			tx, err := beginLifecycleTransaction(lifecycleTransactionConfig{
				TransactionID: "planning-refresh-cleanup-rollback-test",
				Command:       "planning-refresh-cleanup",
				Allowlist: lifecycleTransactionAllowlist{
					RepositoryRoot:    rollbackRoot,
					LifecycleDataRoot: dataRoot,
				},
				Session: session,
				Fault: func(point string) error {
					if point == "after_target_commit:target-0003" {
						return injected
					}
					return nil
				},
			})
			if err != nil {
				return err
			}
			for _, target := range targets {
				if target.Remove {
					err = tx.DeclareRemoval(target.Root, target.Path)
				} else {
					err = tx.DeclareWrite(target.Root, target.Path, target.Content)
				}
				if err != nil {
					return err
				}
			}
			if _, err := tx.Commit(); !errors.Is(err, injected) {
				return fmt.Errorf("cleanup transaction error = %v, want injected fault", err)
			}
			tx.config.Fault = nil
			return tx.Rollback()
		})
		if err != nil {
			t.Fatal(err)
		}
		for rel, before := range baselines {
			after, err := readLifecycleFileState(filepath.Join(dataRoot, filepath.FromSlash(rel)))
			if err != nil {
				t.Fatalf("read rolled-back target %s: %v", rel, err)
			}
			if before.Exists != after.Exists || before.Digest != after.Digest || before.Mode.Perm() != after.Mode.Perm() || !bytes.Equal(before.Bytes, after.Bytes) {
				t.Fatalf("cleanup target %s did not return to its exact baseline\nbefore=%+v\nafter=%+v", rel, before, after)
			}
		}
		if got, err := os.ReadFile(newerResearch); err != nil || string(got) != "worker-authored research" {
			t.Fatalf("rollback changed newer worker research: %q err=%v", got, err)
		}
	})
}

func writeTestFileAtTime(t *testing.T, path, content string, modTime time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("set time %s: %v", path, err)
	}
}

// TestE2EForceReplanRecovery proves the full recovery path:
// colony with fallback plan artifacts → plan --force → fallback artifacts cleared,
// new plan generated, phase status reset.
func TestE2EForceReplanRecovery(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	planningDir := dataDir + "/planning"
	if err := os.MkdirAll(planningDir, 0755); err != nil {
		t.Fatalf("failed to create planning dir: %v", err)
	}
	phaseResearchDir := dataDir + "/phase-research"
	if err := os.MkdirAll(phaseResearchDir, 0755); err != nil {
		t.Fatalf("failed to create phase-research dir: %v", err)
	}

	goal := "E2E force recovery"
	taskID := "1.1"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Fallback phase",
					Description: "Generated by fallback, not real workers",
					Status:      colony.PhaseInProgress,
					Tasks: []colony.Task{
						{ID: &taskID, Goal: "Fallback task", Status: colony.TaskInProgress},
					},
				},
			},
		},
	}
	createApprovedCodexPlanTestColony(t, dataDir, root, state)

	// Write fallback artifacts from one observed instant, with every fallback
	// projection older than its marker. This models the source classification
	// used by force-replan instead of depending on filesystem write order.
	observedAt := time.Now().UTC()
	fallbackMarker := filepath.Join(planningDir, ".fallback-marker")
	writeTestFileAtTime(t, fallbackMarker, observedAt.Format(time.RFC3339), observedAt)
	routeSetter := filepath.Join(planningDir, "ROUTE-SETTER.md")
	writeTestFileAtTime(t, routeSetter, "# Fallback route-setter\nThis was generated by fallback.", observedAt.Add(-time.Minute))
	phasePlan := filepath.Join(planningDir, "phase-plan.json")
	writeTestFileAtTime(t, phasePlan, `{"fallback": true}`, observedAt.Add(-time.Minute))
	phasePlanBackup := filepath.Join(planningDir, "phase-plan.json.bak")
	writeTestFileAtTime(t, phasePlanBackup, `{"stale_backup": true}`, observedAt.Add(-time.Minute))
	staleResearch := filepath.Join(phaseResearchDir, "phase-99-research.md")
	writeTestFileAtTime(t, staleResearch, "stale fallback research", observedAt.Add(-time.Minute))

	// Verify fallback artifacts exist before force-replan.
	if _, err := os.Stat(fallbackMarker); err != nil {
		t.Fatal("expected fallback marker to exist before force-replan")
	}

	// Run plan --force to recover.
	var errBuf bytes.Buffer
	stderr = &errBuf

	rootCmd.SetArgs([]string{"plan", "--force", "--synthetic", "--preset", "balanced"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan --force returned error: %v", err)
	}

	output := errBuf.String()
	if strings.Contains(output, "cannot force-replan") {
		t.Fatalf("force-replan should have been accepted, got: %s", output)
	}

	// Verify fallback marker was cleared.
	if _, err := os.Stat(fallbackMarker); !os.IsNotExist(err) {
		t.Fatalf("expected fallback marker to be removed after force-replan; stderr=%s", output)
	}

	// Verify the fallback route-setter was replaced (no longer contains "Fallback route-setter").
	if content, err := os.ReadFile(routeSetter); err == nil {
		if strings.Contains(string(content), "Fallback route-setter") {
			t.Fatal("expected fallback route-setter to be replaced, but it still contains fallback content")
		}
	}

	// Verify fallback phase-plan was replaced.
	if content, err := os.ReadFile(phasePlan); err == nil {
		if strings.Contains(string(content), `"fallback": true`) {
			t.Fatal("expected fallback phase-plan to be replaced, but it still contains fallback content")
		}
	}
	if _, err := os.Stat(phasePlanBackup); !os.IsNotExist(err) {
		t.Fatal("expected stale phase-plan backup to be removed after force-replan")
	}
	if _, err := os.Stat(staleResearch); !os.IsNotExist(err) {
		t.Fatal("expected stale fallback phase research to be removed after force-replan")
	}

	// Verify colony state is no longer stuck at EXECUTING phase 1.
	var newState colony.ColonyState
	stateData, err := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("failed to read state after force-replan: %v", err)
	}
	if err := json.Unmarshal(stateData, &newState); err != nil {
		t.Fatalf("failed to unmarshal state: %v", err)
	}
	if newState.State == colony.StateEXECUTING && newState.CurrentPhase == 1 && len(newState.Plan.Phases) == 1 && newState.Plan.Phases[0].Status == colony.PhaseInProgress {
		t.Fatal("expected colony state to be recovered from stale EXECUTING phase 1, but it's still stuck")
	}
}

// ---------------------------------------------------------------------------
// PlanningDepth tests (resolvePlanningDepth function)
// ---------------------------------------------------------------------------

func TestPlanningDepthDefault(t *testing.T) {
	got, err := resolvePlanningDepth("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "standard" {
		t.Fatalf("resolvePlanningDepth(\"\") = %q, want %q", got, "standard")
	}
}

func TestPlanningDepthValues(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"light", "light"},
		{"deep", "deep"},
		{"standard", "standard"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := resolvePlanningDepth(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolvePlanningDepth(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPlanningDepthAliases(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"minimal", "light"},
		{"thorough", "deep"},
		{"granular", "deep"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := resolvePlanningDepth(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolvePlanningDepth(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPlanningDepthInvalid(t *testing.T) {
	_, err := resolvePlanningDepth("bad")
	if err == nil {
		t.Fatal("expected error for invalid planning depth")
	}
	if !strings.Contains(err.Error(), "invalid planning depth") {
		t.Fatalf("error %q should contain 'invalid planning depth'", err.Error())
	}
}

func TestPlanningDepthIndependentOfGranularity(t *testing.T) {
	gotGran, _, err := resolvePlanGranularityDepth("", "balanced")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	min, max := colony.GranularityRange(gotGran)

	_, err = resolvePlanningDepth("light")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	gotGran2, _, err := resolvePlanGranularityDepth("", "balanced")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	min2, max2 := colony.GranularityRange(gotGran2)
	if min != min2 || max != max2 {
		t.Fatalf("granularity bounds changed: before=%d-%d after=%d-%d", min, max, min2, max2)
	}
}

func TestPlanningDepthInManifest(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-plan-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	goal := "Test planning depth in manifest"
	createApprovedCodexPlanTestColony(t, dataDir, root, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	rootCmd.SetArgs([]string{"plan", "--plan-only", "--depth", "fast", "--planning-depth", "light"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan --plan-only returned error: %v", err)
	}
	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	manifest := env["result"].(map[string]interface{})["plan_manifest"].(map[string]interface{})

	pd, ok := manifest["planning_depth"].(string)
	if !ok {
		t.Fatal("expected planning_depth string in manifest")
	}
	if pd != "light" {
		t.Fatalf("planning_depth = %q, want %q", pd, "light")
	}
}

func TestPlanningLoopOptionsClampAndEvaluateStopReasons(t *testing.T) {
	opts := codexPlanOptions{TargetConfidence: 1000, MaxIterations: 1000}
	loop := evaluatePlanningLoop(
		codexPlanConfidence{Overall: 82},
		[]string{"gap"},
		opts,
		"balanced",
	)
	if loop.TargetConfidence != 99 {
		t.Fatalf("target = %d, want clamped 99", loop.TargetConfidence)
	}
	if loop.MaxIterations != 12 {
		t.Fatalf("max iterations = %d, want clamped 12", loop.MaxIterations)
	}
	if loop.StopReason != planningLoopPendingStop {
		t.Fatalf("stop reason = %q, want pending", loop.StopReason)
	}
	if loop.Iterations != 1 {
		t.Fatalf("iterations = %d, want 1 evidence-backed pass", loop.Iterations)
	}
	if len(loop.History) != 1 {
		t.Fatalf("history length = %d, want 1", len(loop.History))
	}
	if loop.History[0].Delta != 0 {
		t.Fatalf("first delta = %d, want 0 without a prior evidence pass", loop.History[0].Delta)
	}
	if loop.History[0].StallCount != 0 {
		t.Fatalf("first stall count = %d, want 0 without a prior evidence pass", loop.History[0].StallCount)
	}
	if loop.History[0].EvidenceHash == "" {
		t.Fatal("expected evidence hash in planning loop sample")
	}
	if !containsString(loop.History[0].SelectedGaps, "gap") {
		t.Fatalf("selected gaps = %v, want gap", loop.History[0].SelectedGaps)
	}

	targetReached := evaluatePlanningLoop(codexPlanConfidence{Overall: 95}, nil, codexPlanOptions{}, "balanced")
	if targetReached.StopReason != planningLoopTargetReached {
		t.Fatalf("stop reason = %q, want target reached", targetReached.StopReason)
	}

	accepted := evaluatePlanningLoop(
		codexPlanConfidence{Overall: 72},
		nil,
		codexPlanOptions{TargetConfidence: 90, Accept: true},
		"balanced",
	)
	if accepted.StopReason != planningLoopAccepted || !accepted.AcceptedBelowTarget {
		t.Fatalf("accepted loop = %+v, want accepted below target", accepted)
	}
}

func TestPlanningEvidenceHashChangesWhenEvidenceChanges(t *testing.T) {
	base := codexPlanConfidence{Knowledge: 70, Requirements: 70, Risks: 60, Dependencies: 65, Effort: 75, Overall: 68}
	first := evaluatePlanningLoop(base, []string{"confirm provider dispatch"}, codexPlanOptions{}, "balanced")
	changed := evaluatePlanningLoop(codexPlanConfidence{Knowledge: 72, Requirements: 70, Risks: 60, Dependencies: 65, Effort: 75, Overall: 70}, []string{"confirm provider dispatch"}, codexPlanOptions{}, "balanced")
	gapChanged := evaluatePlanningLoop(base, []string{"confirm route-setter synthesis"}, codexPlanOptions{}, "balanced")

	if len(first.History) != 1 || len(changed.History) != 1 || len(gapChanged.History) != 1 {
		t.Fatalf("expected single-sample histories: %+v %+v %+v", first.History, changed.History, gapChanged.History)
	}
	if first.History[0].EvidenceHash == changed.History[0].EvidenceHash {
		t.Fatal("confidence input changes must change the evidence hash")
	}
	if first.History[0].EvidenceHash == gapChanged.History[0].EvidenceHash {
		t.Fatal("gap changes must change the evidence hash")
	}
	if !strings.Contains(first.History[0].Evidence, "single planning evidence pass") {
		t.Fatalf("evidence summary = %q", first.History[0].Evidence)
	}
}

func TestPlanOnlyManifestIncludesLockedPlanningPreset(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-plan-loop\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	goal := "Test classic planning loop controls"
	createApprovedCodexPlanTestColony(t, dataDir, root, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	rootCmd.SetArgs([]string{"plan", "--plan-only", "--preset", "deep"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan --plan-only returned error: %v", err)
	}
	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	manifest := result["plan_manifest"].(map[string]interface{})
	loop := manifest["planning_loop"].(map[string]interface{})
	if int(loop["target_confidence"].(float64)) != 95 {
		t.Fatalf("target_confidence = %v, want 95", loop["target_confidence"])
	}
	if int(loop["max_iterations"].(float64)) != 8 {
		t.Fatalf("max_iterations = %v, want 8", loop["max_iterations"])
	}
	if accepted, present := loop["accept"].(bool); present && accepted {
		t.Fatalf("accept = %v, want absent or false", loop["accept"])
	}
	if loop["stop_reason"].(string) != planningLoopPendingStop {
		t.Fatalf("stop_reason = %q, want pending", loop["stop_reason"])
	}
	if strings.TrimSpace(manifest["planning_run_id"].(string)) == "" {
		t.Fatal("expected planning_run_id in manifest")
	}
	if int(manifest["iteration"].(float64)) != 1 {
		t.Fatalf("iteration = %v, want 1", manifest["iteration"])
	}
	if int(manifest["target_confidence"].(float64)) != 95 {
		t.Fatalf("manifest target_confidence = %v, want 95", manifest["target_confidence"])
	}
	if int(manifest["max_iterations"].(float64)) != 8 {
		t.Fatalf("manifest max_iterations = %v, want 8", manifest["max_iterations"])
	}
	if got := len(manifest["expected_workers"].([]interface{})); got != 1 {
		t.Fatalf("expected_workers length = %d, want Scout only", got)
	}
}

func TestPlanningDepthInWrapperContract(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-plan-wc\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	goal := "Test planning depth in wrapper contract"
	createApprovedCodexPlanTestColony(t, dataDir, root, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	rootCmd.SetArgs([]string{"plan", "--plan-only", "--depth", "fast", "--planning-depth", "deep"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan --plan-only returned error: %v", err)
	}
	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	wrapperContract, ok := result["wrapper_contract"].(map[string]interface{})
	if !ok {
		t.Fatal("expected wrapper_contract in result")
	}
	pd, ok := wrapperContract["planning_depth"].(string)
	if !ok {
		t.Fatal("expected planning_depth string in wrapper_contract")
	}
	if pd != "deep" {
		t.Fatalf("wrapper_contract planning_depth = %q, want %q", pd, "deep")
	}
}

func TestPlanningWorkerBriefIncludesCodegraphContext(t *testing.T) {
	saveGlobals(t)

	root := t.TempDir()
	graphDir := filepath.Join(root, ".aether", "data")
	if err := os.MkdirAll(graphDir, 0755); err != nil {
		t.Fatalf("mkdir graph dir: %v", err)
	}
	graph := codegraph.CodeGraph{
		Files: []codegraph.FileNode{
			{Path: "src/app.ts", Language: "typescript"},
			{Path: "src/service.ts", Language: "typescript"},
		},
		Edges: []codegraph.DepEdge{{Source: "src/app.ts", Target: "src/service.ts", Type: "import"}},
	}
	if err := graph.Save(filepath.Join(graphDir, "codebase-graph.json")); err != nil {
		t.Fatalf("save graph: %v", err)
	}

	brief := renderPlanningWorkerBrief(root, codexSurveyContext{
		EntryPoints: []string{"src/app.ts"},
	}, planningWorkerSpecs[0])

	if !strings.Contains(brief, "## Codebase Graph Context") {
		t.Fatalf("planning brief missing codegraph context:\n%s", brief)
	}
	if !strings.Contains(brief, "src/service.ts") {
		t.Fatalf("planning brief missing related dependency file:\n%s", brief)
	}
}

func TestPlanningWorkerBriefIncludesLoopGuards(t *testing.T) {
	root := t.TempDir()
	survey := codexSurveyContext{
		SurveyDocs: []string{"BLUEPRINT.md", "PATHOGENS.md"},
	}

	scoutBrief := renderPlanningWorkerBrief(root, survey, planningWorkerSpecs[0])
	for _, want := range []string{
		"Loop guard: read each file at most once",
		"Scout read budget",
		"at most 8 targeted repository files",
		"record the gap instead of continuing to search",
		"Final result must include scout_report",
	} {
		if !strings.Contains(scoutBrief, want) {
			t.Fatalf("scout planning brief missing %q:\n%s", want, scoutBrief)
		}
	}

	routeBrief := renderPlanningWorkerBrief(root, survey, planningWorkerSpecs[1])
	for _, want := range []string{
		"Loop guard: read each file at most once",
		"Route-Setter read budget",
		"at most 6 targeted repository files",
		"Do not redo the Scout survey",
		"phase-plan.json",
	} {
		if !strings.Contains(routeBrief, want) {
			t.Fatalf("route-setter planning brief missing %q:\n%s", want, routeBrief)
		}
	}
}

func TestPlanningWorkerBriefIncludesSurveyFindingsAsBuildableGuidance(t *testing.T) {
	root := t.TempDir()
	survey := codexSurveyContext{
		SurveyDocs:   []string{"BLUEPRINT.md", "DISCIPLINES.md", "PATHOGENS.md"},
		Languages:    []string{"Go"},
		Frameworks:   []string{"cobra CLI"},
		Directories:  []string{"cmd", "pkg"},
		EntryPoints:  []string{"cmd/main.go"},
		Dependencies: []string{"github.com/spf13/cobra"},
		TestFiles:    []string{"cmd/codex_plan_test.go"},
		Issues:       []string{"Codex planning currently drops Scout results"},
	}

	brief := renderPlanningWorkerBrief(root, survey, planningWorkerSpecs[1])
	for _, want := range []string{
		"## Scout Planning Guidance",
		"Primary execution surfaces live around cmd/main.go",
		"The implementation surface spans Go",
		"Existing test coverage already exercises representative paths like cmd/codex_plan_test.go",
		"Codex planning currently drops Scout results",
	} {
		if !strings.Contains(brief, want) {
			t.Fatalf("route-setter planning guidance missing %q:\n%s", want, brief)
		}
	}
}

// --- Source anchors in planning worker brief ---

func TestRenderPlanningWorkerBrief_SourceAnchors(t *testing.T) {
	root := t.TempDir()

	t.Run("WithAnchors", func(t *testing.T) {
		survey := codexSurveyContext{
			SurveyDocs:    []string{"BLUEPRINT.md"},
			SourceAnchors: []string{"cmd/main.go", "pkg/storage/store.go", "cmd/codex_plan.go"},
		}
		brief := renderPlanningWorkerBrief(root, survey, planningWorkerSpecs[1])
		if !strings.Contains(brief, "Source anchors available: 3 repo-owned files from survey") {
			t.Fatalf("route-setter brief missing source anchor hint:\n%s", brief)
		}
	})

	t.Run("WithoutAnchors", func(t *testing.T) {
		survey := codexSurveyContext{
			SurveyDocs: []string{"BLUEPRINT.md"},
		}
		brief := renderPlanningWorkerBrief(root, survey, planningWorkerSpecs[1])
		if strings.Contains(brief, "Source anchors available") {
			t.Fatalf("route-setter brief should not mention source anchors when empty:\n%s", brief)
		}
	})
}

// --- Plan finalizer validation rejection tests (Task 4.1) ---

func TestMergeExternalPlanResults_RejectsMalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	malformedPath := filepath.Join(tmpDir, "completion.json")
	if err := os.WriteFile(malformedPath, []byte("{not valid json}"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := loadExternalPlanCompletion(malformedPath)
	if err == nil {
		t.Fatal("expected error for malformed JSON completion file")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Fatalf("expected parse error, got: %v", err)
	}
}

func TestMergeExternalPlanResults_RejectsMissingManifest(t *testing.T) {
	tmpDir := t.TempDir()
	noManifestPath := filepath.Join(tmpDir, "completion.json")
	validJSON := `{"dispatches": [{"name": "Scout-12", "caste": "scout", "status": "completed"}]}`
	if err := os.WriteFile(noManifestPath, []byte(validJSON), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := loadExternalPlanCompletion(noManifestPath)
	if err == nil {
		t.Fatal("expected error for completion file missing plan_manifest")
	}
	if !strings.Contains(err.Error(), "plan_manifest") {
		t.Fatalf("expected plan_manifest error, got: %v", err)
	}
}

func TestMergeExternalPlanResults_RejectsWrongCaste(t *testing.T) {
	manifest := codexPlanManifest{
		DispatchMode:      "plan-only",
		RequiresFinalizer: true,
		Dispatches: []codexPlanningDispatch{
			{Name: "Scout-12", Caste: "scout", Stage: "survey", TaskID: "plan-scout"},
		},
	}
	results := []codexPlanningDispatch{
		{Name: "Scout-12", Caste: "watcher", Status: "completed"},
	}
	_, err := mergeExternalPlanResults(manifest, results)
	if err == nil {
		t.Fatal("expected error for wrong caste in plan result")
	}
	if !strings.Contains(err.Error(), "caste") {
		t.Fatalf("expected caste mismatch error, got: %v", err)
	}
}

func TestMergeExternalPlanResults_RejectsWrongStage(t *testing.T) {
	manifest := codexPlanManifest{
		DispatchMode:      "plan-only",
		RequiresFinalizer: true,
		Dispatches: []codexPlanningDispatch{
			{Name: "Scout-12", Caste: "scout", Stage: "survey", TaskID: "plan-scout"},
		},
	}
	results := []codexPlanningDispatch{
		{Name: "Scout-12", Stage: "design", Status: "completed"},
	}
	_, err := mergeExternalPlanResults(manifest, results)
	if err == nil {
		t.Fatal("expected error for wrong stage in plan result")
	}
	if !strings.Contains(err.Error(), "stage") {
		t.Fatalf("expected stage mismatch error, got: %v", err)
	}
}

func TestMergeExternalPlanResults_RejectsWrongTaskID(t *testing.T) {
	manifest := codexPlanManifest{
		DispatchMode:      "plan-only",
		RequiresFinalizer: true,
		Dispatches: []codexPlanningDispatch{
			{Name: "Scout-12", Caste: "scout", Stage: "survey", TaskID: "plan-scout"},
		},
	}
	results := []codexPlanningDispatch{
		{Name: "Scout-12", TaskID: "wrong-task", Status: "completed"},
	}
	_, err := mergeExternalPlanResults(manifest, results)
	if err == nil {
		t.Fatal("expected error for wrong task_id in plan result")
	}
	if !strings.Contains(err.Error(), "task_id") {
		t.Fatalf("expected task_id mismatch error, got: %v", err)
	}
}

func TestMergeExternalPlanResults_RejectsWrongWave(t *testing.T) {
	manifest := codexPlanManifest{
		DispatchMode:      "plan-only",
		RequiresFinalizer: true,
		Dispatches: []codexPlanningDispatch{
			{Name: "Scout-12", Caste: "scout", Stage: "survey", Wave: 1, TaskID: "plan-scout"},
		},
	}
	results := []codexPlanningDispatch{
		{Name: "Scout-12", Wave: 2, Status: "completed"},
	}
	_, err := mergeExternalPlanResults(manifest, results)
	if err == nil {
		t.Fatal("expected error for wrong wave in plan result")
	}
	if !strings.Contains(err.Error(), "wave") {
		t.Fatalf("expected wave mismatch error, got: %v", err)
	}
}

func TestMergeExternalPlanResults_RejectsDuplicateWorkerResult(t *testing.T) {
	manifest := codexPlanManifest{
		DispatchMode:      "plan-only",
		RequiresFinalizer: true,
		Dispatches: []codexPlanningDispatch{
			{Name: "Scout-12", Caste: "scout", Stage: "survey", TaskID: "plan-scout"},
		},
	}
	results := []codexPlanningDispatch{
		{Name: "Scout-12", Status: "completed"},
		{Name: "Scout-12", Status: "completed"},
	}
	_, err := mergeExternalPlanResults(manifest, results)
	if err == nil {
		t.Fatal("expected error for duplicate plan worker result")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate error, got: %v", err)
	}
}

func TestMergeExternalPlanResults_RejectsNonTerminalStatus(t *testing.T) {
	manifest := codexPlanManifest{
		DispatchMode:      "plan-only",
		RequiresFinalizer: true,
		Dispatches: []codexPlanningDispatch{
			{Name: "Scout-12", Caste: "scout", Stage: "survey", TaskID: "plan-scout"},
		},
	}
	results := []codexPlanningDispatch{
		{Name: "Scout-12", Status: "running"},
	}
	_, err := mergeExternalPlanResults(manifest, results)
	if err == nil {
		t.Fatal("expected error for non-terminal status in plan result")
	}
	if !strings.Contains(err.Error(), "non-terminal") {
		t.Fatalf("expected non-terminal status error, got: %v", err)
	}
}

func TestMergeExternalPlanResults_RejectsNonCompletedNonReconciled(t *testing.T) {
	manifest := codexPlanManifest{
		DispatchMode:      "plan-only",
		RequiresFinalizer: true,
		Dispatches: []codexPlanningDispatch{
			{Name: "Scout-12", Caste: "scout", Stage: "survey", TaskID: "plan-scout"},
		},
	}
	results := []codexPlanningDispatch{
		{Name: "Scout-12", Status: "failed"},
	}
	_, err := mergeExternalPlanResults(manifest, results)
	if err == nil {
		t.Fatal("expected error for non-completed non-reconciled plan result")
	}
	if !strings.Contains(err.Error(), "did not complete cleanly") {
		t.Fatalf("expected 'did not complete cleanly' error, got: %v", err)
	}
}

func TestMergeExternalPlanResults_RejectsMissingResult(t *testing.T) {
	manifest := codexPlanManifest{
		DispatchMode:      "plan-only",
		RequiresFinalizer: true,
		Dispatches: []codexPlanningDispatch{
			{Name: "Scout-12", Caste: "scout", Stage: "survey", TaskID: "plan-scout"},
			{Name: "Mapper-45", Caste: "route_setter", Stage: "design", TaskID: "plan-route"},
		},
	}
	results := []codexPlanningDispatch{
		{Name: "Scout-12", Status: "completed"},
		// Mapper-45 result is missing
	}
	_, err := mergeExternalPlanResults(manifest, results)
	if err == nil {
		t.Fatal("expected error for missing plan worker result")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected missing result error, got: %v", err)
	}
}

func TestMergeExternalPlanResults_RejectsNamelessResult(t *testing.T) {
	manifest := codexPlanManifest{
		DispatchMode:      "plan-only",
		RequiresFinalizer: true,
		Dispatches: []codexPlanningDispatch{
			{Name: "Scout-12", Caste: "scout", Stage: "survey", TaskID: "plan-scout"},
		},
	}
	results := []codexPlanningDispatch{
		{Name: "", Status: "completed"},
	}
	_, err := mergeExternalPlanResults(manifest, results)
	if err == nil {
		t.Fatal("expected error for nameless plan result")
	}
	if !strings.Contains(err.Error(), "missing name") {
		t.Fatalf("expected missing name error, got: %v", err)
	}
}

func TestPlanningWorkerBriefGatekeeperNeverInstructsCLI(t *testing.T) {
	root := t.TempDir()
	survey := codexSurveyContext{}
	spec, ok := planningWorkerSpecForCaste("gatekeeper")
	if !ok {
		t.Fatal("planningWorkerSpecForCaste(gatekeeper) should return a spec")
	}

	brief := renderPlanningWorkerBrief(root, survey, spec)
	// Gatekeeper has no Bash tool by design; an instruction to run
	// `aether review-ledger-write` is unsatisfiable and made the caste
	// self-report blocked (Pocket-Chopper field report).
	if strings.Contains(brief, "review-ledger-write") {
		t.Fatalf("gatekeeper planning brief instructs a CLI command the caste cannot run:\n%s", brief)
	}
	if !strings.Contains(brief, "This is a review task") {
		t.Fatalf("gatekeeper planning brief missing review task indicator:\n%s", brief)
	}
	if !strings.Contains(brief, "Return status `blocked` if advancement is unsafe") {
		t.Fatalf("gatekeeper planning brief missing blocked status instruction:\n%s", brief)
	}
}
