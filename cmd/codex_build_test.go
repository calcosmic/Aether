package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codegraph"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func forceBuildJSONOutput(t *testing.T) {
	t.Helper()
	t.Setenv("AETHER_OUTPUT_MODE", "json")
}

func TestBuildWritesDispatchArtifactsAndUpdatesState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceBuildJSONOutput(t)

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

	goal := "Bring Codex build parity to the ant process"
	researchID := "1.1"
	implementID := "1.2"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Build parity",
					Description: "Replace fake build dispatch with real artifacts and spawn records",
					Status:      colony.PhaseReady,
					Tasks: []colony.Task{
						{ID: &researchID, Goal: "Research the missing build orchestration gaps", Status: colony.TaskPending},
						{ID: &implementID, Goal: "Implement the Go-native build packet", Status: colony.TaskPending, DependsOn: []string{researchID}},
					},
					SuccessCriteria: []string{"Build artifacts exist", "Spawn tree reflects the worker packet"},
				},
			},
		},
	})

	rootCmd.SetArgs([]string{"build", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(stdout.(*bytes.Buffer).Bytes(), &envelope); err != nil {
		t.Fatalf("failed to parse build output: %v\n%s", err, stdout.(*bytes.Buffer).String())
	}
	if envelope["ok"] != true {
		t.Fatalf("expected ok:true, got %v", envelope)
	}

	result := envelope["result"].(map[string]interface{})
	if got := int(result["dispatch_count"].(float64)); got != 6 {
		// Modeless phase resolves to prototype: prose "Research" in a task no
		// longer spawns an Oracle (typed phase mode).
		t.Fatalf("dispatch_count = %d, want 6", got)
	}
	if got := int(result["wave_count"].(float64)); got != 2 {
		t.Fatalf("wave_count = %d, want 2 task waves", got)
	}
	if got := int(result["parallel_waves"].(float64)); got != 0 {
		t.Fatalf("parallel_waves = %d, want 0", got)
	}
	if got := int(result["execution_wave_count"].(float64)); got != 6 {
		t.Fatalf("execution_wave_count = %d, want 6 execution waves", got)
	}
	if next := result["next"].(string); next != "aether continue" {
		t.Fatalf("next = %q, want aether continue", next)
	}
	if waveExecution, ok := result["wave_execution"].([]interface{}); !ok || len(waveExecution) != 2 {
		t.Fatalf("wave_execution = %#v, want 2 wave plans", result["wave_execution"])
	}
	if executionPlan, ok := result["execution_plan"].([]interface{}); !ok || len(executionPlan) != 6 {
		t.Fatalf("execution_plan = %#v, want 6 execution stages", result["execution_plan"])
	}

	for _, rel := range []string{
		"checkpoints/pre-build-phase-1.json",
		"build/phase-1/manifest.json",
		"last-build-claims.json",
	} {
		if _, err := os.Stat(filepath.Join(dataDir, rel)); err != nil {
			t.Fatalf("expected artifact %s: %v", rel, err)
		}
	}

	var manifest codexBuildManifest
	if err := store.LoadJSON("build/phase-1/manifest.json", &manifest); err != nil {
		t.Fatalf("failed to load build manifest: %v", err)
	}
	if manifest.Phase != 1 || manifest.PhaseName != "Build parity" {
		t.Fatalf("unexpected manifest header: %+v", manifest)
	}
	if manifest.DispatchMode != "simulated" {
		t.Fatalf("dispatch mode = %q, want simulated", manifest.DispatchMode)
	}
	if len(manifest.Dispatches) != 6 {
		t.Fatalf("expected 6 manifest dispatches, got %d", len(manifest.Dispatches))
	}
	if len(manifest.WorkerBriefs) != 6 {
		t.Fatalf("expected 6 worker briefs in manifest, got %d", len(manifest.WorkerBriefs))
	}
	if len(manifest.Tasks) != 2 {
		t.Fatalf("expected 2 planned tasks, got %d", len(manifest.Tasks))
	}
	if len(manifest.WaveExecution) != 2 {
		t.Fatalf("expected 2 manifest wave execution plans, got %d", len(manifest.WaveExecution))
	}
	for _, plan := range manifest.WaveExecution {
		if plan.Strategy != "serial" {
			t.Fatalf("manifest wave %d strategy = %q, want serial", plan.Wave, plan.Strategy)
		}
	}
	for _, brief := range manifest.WorkerBriefs {
		rel := strings.TrimPrefix(brief, ".aether/data/")
		if _, err := os.Stat(filepath.Join(dataDir, rel)); err != nil {
			t.Fatalf("expected worker brief %s: %v", brief, err)
		}
	}

	var claims codexBuildClaims
	if err := store.LoadJSON("last-build-claims.json", &claims); err != nil {
		t.Fatalf("failed to load last-build-claims.json: %v", err)
	}
	if claims.BuildPhase != 1 {
		t.Fatalf("claims build phase = %d, want 1", claims.BuildPhase)
	}
	if len(claims.FilesCreated) != 0 || len(claims.FilesModified) != 0 {
		t.Fatalf("expected empty claims for pre-execution packet, got %+v", claims)
	}

	spawnTreeData, err := os.ReadFile(filepath.Join(dataDir, "spawn-tree.txt"))
	if err != nil {
		t.Fatalf("expected spawn-tree.txt: %v", err)
	}
	for _, want := range []string{"|Queen|builder|", "|Queen|watcher|", "|Queen|probe|"} {
		if !strings.Contains(string(spawnTreeData), want) {
			t.Fatalf("spawn tree missing %q\n%s", want, string(spawnTreeData))
		}
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload colony state: %v", err)
	}
	if state.State != colony.StateBUILT {
		t.Fatalf("state = %s, want BUILT", state.State)
	}
	if state.CurrentPhase != 1 {
		t.Fatalf("current_phase = %d, want 1", state.CurrentPhase)
	}
	if state.BuildStartedAt == nil {
		t.Fatal("expected BuildStartedAt to be set")
	}
	if state.Plan.Phases[0].Status != colony.PhaseInProgress {
		t.Fatalf("phase status = %s, want in_progress", state.Plan.Phases[0].Status)
	}
	if state.Plan.Phases[0].Tasks[0].Status != colony.TaskCompleted {
		t.Fatalf("task 1 status = %s, want completed", state.Plan.Phases[0].Tasks[0].Status)
	}
	if state.Plan.Phases[0].Tasks[1].Status != colony.TaskCompleted {
		t.Fatalf("task 2 status = %s, want completed", state.Plan.Phases[0].Tasks[1].Status)
	}
	if manifest.Tasks[0].Status != colony.TaskCompleted || manifest.Tasks[1].Status != colony.TaskCompleted {
		t.Fatalf("manifest task statuses = %s/%s, want completed/completed", manifest.Tasks[0].Status, manifest.Tasks[1].Status)
	}
	for _, dispatch := range manifest.Dispatches {
		if dispatch.TaskID == "" || dispatch.Status != "completed" {
			continue
		}
		if len(dispatch.Outputs) == 0 {
			t.Fatalf("completed task dispatch %s has no output evidence", dispatch.Name)
		}
		outputRel := strings.TrimPrefix(dispatch.Outputs[0], ".aether/data/")
		outputData, err := os.ReadFile(filepath.Join(dataDir, outputRel))
		if err != nil {
			t.Fatalf("read output evidence for %s: %v", dispatch.Name, err)
		}
		if !strings.Contains(string(outputData), "## Recorded Outcome") || !strings.Contains(string(outputData), "Status: completed") {
			t.Fatalf("output evidence for %s is not a final outcome report:\n%s", dispatch.Name, string(outputData))
		}
		if strings.Contains(dispatch.Outputs[0], "/worker-briefs/") && !strings.Contains(string(outputData), "## Recorded Outcome") {
			t.Fatalf("output evidence for %s points to assignment-only worker brief %s", dispatch.Name, dispatch.Outputs[0])
		}
	}
	if len(state.Events) < 2 || !strings.Contains(strings.Join(state.Events[len(state.Events)-2:], "\n"), "build_dispatched|build") {
		t.Fatalf("expected build_dispatched event, got %v", state.Events)
	}

	contextData, err := os.ReadFile(filepath.Join(root, ".aether", "CONTEXT.md"))
	if err != nil {
		t.Fatalf("expected CONTEXT.md: %v", err)
	}
	if !strings.Contains(string(contextData), "aether continue") {
		t.Fatalf("expected CONTEXT.md to point at continue, got:\n%s", string(contextData))
	}

	handoffData, err := os.ReadFile(filepath.Join(root, ".aether", "HANDOFF.md"))
	if err != nil {
		t.Fatalf("expected HANDOFF.md: %v", err)
	}
	if !strings.Contains(string(handoffData), "Phase 1 dispatched") {
		t.Fatalf("expected HANDOFF.md to summarize build progress, got:\n%s", string(handoffData))
	}
}

// TestWorkerBriefFileHoldsComposedBrief proves the D-12 fix: the file at
// worker-briefs/{name}.md is byte-identical to the dispatch's manifest brief
// field (the composed brief -- base + pheromone signals + prior handoffs),
// not the base-only render writeCodexBuildArtifacts wrote before this change.
func TestWorkerBriefFileHoldsComposedBrief(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceBuildJSONOutput(t)

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

	// Seed an active pheromone signal so the composed brief diverges from the
	// base-only render -- proving the file changed, not just that a heading
	// exists somewhere.
	recent := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{Type: "FOCUS", Content: json.RawMessage(`{"text":"security"}`), Active: true, Strength: floatPtr(0.8), CreatedAt: recent},
		},
	}
	if err := store.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("failed to save pheromones: %v", err)
	}

	goal := "Prove worker briefs carry the composed prompt"
	researchID := "1.1"
	implementID := "1.2"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Composed brief parity",
					Description: "Prove the worker-briefs file matches the manifest brief field byte for byte",
					Status:      colony.PhaseReady,
					Tasks: []colony.Task{
						{ID: &researchID, Goal: "Research the missing build orchestration gaps", Status: colony.TaskPending},
						{ID: &implementID, Goal: "Implement the Go-native build packet", Status: colony.TaskPending, DependsOn: []string{researchID}},
					},
					SuccessCriteria: []string{"Build artifacts exist"},
				},
			},
		},
	})

	rootCmd.SetArgs([]string{"build", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	var manifest codexBuildManifest
	if err := store.LoadJSON("build/phase-1/manifest.json", &manifest); err != nil {
		t.Fatalf("failed to load build manifest: %v", err)
	}
	if len(manifest.Dispatches) == 0 {
		t.Fatal("expected at least one dispatch")
	}

	sawPheromoneSection := false
	for _, dispatch := range manifest.Dispatches {
		if strings.TrimSpace(dispatch.Brief) == "" {
			t.Fatalf("dispatch %s has no composed brief in the manifest", dispatch.Name)
		}
		if strings.TrimSpace(dispatch.BriefPath) == "" {
			t.Fatalf("dispatch %s has no brief_path in the manifest", dispatch.Name)
		}
		briefRel := strings.TrimPrefix(dispatch.BriefPath, ".aether/data/")
		fileContents, err := os.ReadFile(filepath.Join(dataDir, briefRel))
		if err != nil {
			t.Fatalf("failed to read worker brief file for %s: %v", dispatch.Name, err)
		}
		if !bytes.Equal(fileContents, []byte(dispatch.Brief)) {
			t.Fatalf("worker brief file for %s does not byte-match manifest brief field", dispatch.Name)
		}
		if strings.Contains(string(fileContents), "## Pheromone Signals") {
			sawPheromoneSection = true
		}
	}
	if !sawPheromoneSection {
		t.Fatal("expected at least one worker brief file to contain the Pheromone Signals heading")
	}

	// Prove the base-only render does NOT itself contain the pheromone
	// section -- the file changed because writeCodexBuildArtifacts now writes
	// the composed brief, not because the heading appears unconditionally.
	var reloadedState colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &reloadedState); err != nil {
		t.Fatalf("failed to reload colony state: %v", err)
	}
	base := renderCodexBuildWorkerBrief(root, reloadedState.Plan.Phases[0], manifest.Dispatches[0], time.Now().UTC())
	if strings.Contains(base, "## Pheromone Signals") {
		t.Fatal("base-only render unexpectedly contains the Pheromone Signals heading")
	}
}

// TestDispatchEntryCarriesBriefPath proves every dispatch entry in both the
// result envelope and the persisted manifest names the file holding its
// brief, and that the named file resolves to something on disk under the
// store base path.
func TestDispatchEntryCarriesBriefPath(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceBuildJSONOutput(t)

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

	goal := "Prove every dispatch entry names its brief file"
	researchID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "brief_path coverage",
					Description: "Every manifest dispatch entry names the file holding its brief",
					Status:      colony.PhaseReady,
					Tasks: []colony.Task{
						{ID: &researchID, Goal: "Research the missing build orchestration gaps", Status: colony.TaskPending},
					},
					SuccessCriteria: []string{"Build artifacts exist"},
				},
			},
		},
	})

	rootCmd.SetArgs([]string{"build", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(stdout.(*bytes.Buffer).Bytes(), &envelope); err != nil {
		t.Fatalf("failed to parse build output: %v\n%s", err, stdout.(*bytes.Buffer).String())
	}
	result := envelope["result"].(map[string]interface{})
	dispatches := result["dispatches"].([]interface{})
	if len(dispatches) == 0 {
		t.Fatal("expected at least one dispatch in the build result")
	}
	basePath := store.BasePath()
	for _, raw := range dispatches {
		dispatch := raw.(map[string]interface{})
		briefPath, ok := dispatch["brief_path"].(string)
		if !ok || strings.TrimSpace(briefPath) == "" {
			t.Fatalf("dispatch %v missing brief_path", dispatch["name"])
		}
		if strings.Contains(briefPath, "..") {
			t.Fatalf("brief_path %s escapes the store base path", briefPath)
		}
		rel := strings.TrimPrefix(briefPath, ".aether/data/")
		full := filepath.Join(basePath, rel)
		if _, err := os.Stat(full); err != nil {
			t.Fatalf("brief_path %s does not resolve to an existing file under %s: %v", briefPath, basePath, err)
		}
	}

	var manifest codexBuildManifest
	if err := store.LoadJSON("build/phase-1/manifest.json", &manifest); err != nil {
		t.Fatalf("failed to load build manifest: %v", err)
	}
	for _, dispatch := range manifest.Dispatches {
		if strings.TrimSpace(dispatch.BriefPath) == "" {
			t.Fatalf("manifest dispatch %s missing brief_path", dispatch.Name)
		}
	}
}

func TestBuildPlanOnlyPrintsDispatchManifestWithoutMutatingState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceBuildJSONOutput(t)
	setupRuntimeSkillAssignmentHub(t)

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

	goal := "Expose wrapper-spawn build plans"
	taskOneID := "1.1"
	taskTwoID := "1.2"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Wrapper bridge",
					Description: "Let Claude and OpenCode spawn workers from a runtime manifest",
					Status:      colony.PhaseReady,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Define the structured build manifest", Status: colony.TaskPending},
						{ID: &taskTwoID, Goal: "Use the manifest in wrappers", Status: colony.TaskPending, DependsOn: []string{taskOneID}},
					},
					SuccessCriteria: []string{"Wrappers do not parse visual output"},
				},
			},
		},
	})

	rootCmd.SetArgs([]string{"build", "1", "--plan-only"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build --plan-only returned error: %v", err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(stdout.(*bytes.Buffer).Bytes(), &envelope); err != nil {
		t.Fatalf("failed to parse plan-only output: %v\n%s", err, stdout.(*bytes.Buffer).String())
	}
	if envelope["ok"] != true {
		t.Fatalf("expected ok:true, got %v", envelope)
	}
	result := envelope["result"].(map[string]interface{})
	if result["plan_only"] != true {
		t.Fatalf("plan_only = %v, want true", result["plan_only"])
	}
	if got := result["dispatch_mode"].(string); got != "plan-only" {
		t.Fatalf("dispatch_mode = %q, want plan-only", got)
	}
	wrapperContract := result["wrapper_contract"].(map[string]interface{})
	if got := wrapperContract["source_command"].(string); got != "aether build <phase> --plan-only" {
		t.Fatalf("wrapper_contract source_command = %q, want TS host build command", got)
	}
	if got := result["colony_mode"].(string); got != "colony" {
		t.Fatalf("colony_mode = %q, want colony", got)
	}
	if got := int(result["dispatch_count"].(float64)); got != 7 {
		t.Fatalf("dispatch_count = %d, want 7", got)
	}
	dispatches := result["dispatches"].([]interface{})
	if len(dispatches) != 7 {
		t.Fatalf("dispatches = %d, want 7", len(dispatches))
	}
	for _, raw := range dispatches {
		dispatch := raw.(map[string]interface{})
		if dispatch["status"].(string) != "planned" {
			t.Fatalf("dispatch status = %q, want planned", dispatch["status"])
		}
		if strings.TrimSpace(dispatch["agent_name"].(string)) == "" {
			t.Fatalf("dispatch missing agent_name: %+v", dispatch)
		}
		if int(dispatch["execution_wave"].(float64)) <= 0 {
			t.Fatalf("dispatch missing execution_wave: %+v", dispatch)
		}
		assertDispatchHasRuntimeSkillAssignment(t, dispatch)
	}

	manifest := result["dispatch_manifest"].(map[string]interface{})
	if manifest["plan_only"] != true {
		t.Fatalf("manifest plan_only = %v, want true", manifest["plan_only"])
	}
	if manifest["dispatch_mode"].(string) != "plan-only" {
		t.Fatalf("manifest dispatch_mode = %q, want plan-only", manifest["dispatch_mode"])
	}
	if manifest["colony_mode"].(string) != "colony" {
		t.Fatalf("manifest colony_mode = %q, want colony", manifest["colony_mode"])
	}
	if manifest["checkpoint"].(string) != "" || manifest["claims_path"].(string) != "" {
		t.Fatalf("plan-only manifest should not claim artifact paths: %+v", manifest)
	}
	if strings.TrimSpace(manifest["attempt_id"].(string)) == "" || strings.TrimSpace(manifest["attempt_path"].(string)) == "" {
		t.Fatalf("plan-only manifest should identify its durable attempt: %+v", manifest)
	}
	if workerBriefs := manifest["worker_briefs"].([]interface{}); len(workerBriefs) != 0 {
		t.Fatalf("plan-only manifest should not write worker briefs, got %v", workerBriefs)
	}
	manifestDispatches := manifest["dispatches"].([]interface{})
	assertDispatchHasRuntimeSkillAssignment(t, manifestDispatches[0].(map[string]interface{}))
	executionPlan := manifest["execution_plan"].([]interface{})
	if len(executionPlan) != 7 {
		t.Fatalf("execution_plan = %d, want 7 steps: %#v", len(executionPlan), executionPlan)
	}
	wantStages := []string{"design", "wave", "wave", "probe", "measurement", "resilience", "verification"}
	var gotStages []string
	for _, raw := range executionPlan {
		step := raw.(map[string]interface{})
		stage := step["stage"].(string)
		if stage == "wave" {
			gotStages = append(gotStages, stage)
			continue
		}
		gotStages = append(gotStages, stage)
	}
	if strings.Join(gotStages, ",") != strings.Join(wantStages, ",") {
		t.Fatalf("execution stages = %v, want %v", gotStages, wantStages)
	}

	for _, rel := range []string{
		"checkpoints/pre-build-phase-1.json",
		"last-build-claims.json",
	} {
		if _, err := os.Stat(filepath.Join(dataDir, rel)); !os.IsNotExist(err) {
			t.Fatalf("plan-only unexpectedly wrote %s (err=%v)", rel, err)
		}
	}
	for _, rel := range []string{
		"build/phase-1/manifest.json",
		strings.TrimPrefix(manifest["attempt_path"].(string), ".aether/data/"),
		"build/phase-1/latest-attempt.json",
	} {
		if _, err := os.Stat(filepath.Join(dataDir, rel)); err != nil {
			t.Fatalf("plan-only did not persist durable coordination record %s: %v", rel, err)
		}
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload colony state: %v", err)
	}
	if state.State != colony.StateREADY {
		t.Fatalf("state = %s, want READY", state.State)
	}
	if state.CurrentPhase != 0 {
		t.Fatalf("current_phase = %d, want 0", state.CurrentPhase)
	}
	if state.BuildStartedAt != nil {
		t.Fatal("BuildStartedAt should remain nil")
	}
	if state.Plan.Phases[0].Status != colony.PhaseReady {
		t.Fatalf("phase status = %s, want ready", state.Plan.Phases[0].Status)
	}
}

func TestBuildQueenLedWrapperContractUsesHostSourceCommand(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	setupRuntimeSkillAssignmentHub(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Expose queen-led wrapper contract"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:          1,
			Name:        "Wrapper bridge",
			Description: "Let the Queen dispatch workers from a runtime manifest",
			Status:      colony.PhaseReady,
			Tasks:       []colony.Task{{ID: &taskID, Goal: "Build the wrapper bridge", Status: colony.TaskPending}},
		}}},
	})

	result, _, _, _, err := runCodexBuildQueenLed(root, 1, nil, codexBuildOptions{})
	if err != nil {
		t.Fatalf("runCodexBuildQueenLed returned error: %v", err)
	}
	wrapperContract := result["wrapper_contract"].(map[string]interface{})
	if got := wrapperContract["source_command"].(string); got != "aether build <phase> --plan-only" {
		t.Fatalf("wrapper_contract source_command = %q, want TS host build command", got)
	}
}

func TestBuildPlanOnlyUsesPriorPhaseEvidenceWithoutMutatingState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Plan the next phase without committing repairs"
	phaseOneTaskID := "1.1"
	phaseTwoTaskID := "2.1"
	now := time.Now().UTC()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 2,
		ColonyDepth:  "light",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Completed but stale task rows",
					Status: colony.PhaseCompleted,
					Tasks:  []colony.Task{{ID: &phaseOneTaskID, Goal: "Already completed externally", Status: colony.TaskPending}},
				},
				{
					ID:     2,
					Name:   "Next phase",
					Status: colony.PhaseReady,
					Tasks:  []colony.Task{{ID: &phaseTwoTaskID, Goal: "Plan the next implementation", Status: colony.TaskPending}},
				},
			},
		},
	})

	if err := store.SaveJSON("build/phase-1/manifest.json", codexBuildManifest{
		Phase:        1,
		PhaseName:    "Completed but stale task rows",
		Goal:         goal,
		Root:         root,
		ColonyDepth:  "light",
		DispatchMode: "external-task",
		GeneratedAt:  now.Format(time.RFC3339),
		State:        string(colony.StateBUILT),
		Tasks:        []codexBuildTaskPlan{{ID: phaseOneTaskID, Goal: "Already completed externally", Status: colony.TaskCompleted}},
		Dispatches: []codexBuildDispatch{
			{Stage: "wave", Wave: 1, ExecutionWave: 11, Caste: "builder", Name: "Forge-prior", Task: "Already completed externally", Status: "completed", TaskID: phaseOneTaskID, Outputs: []string{"main.go"}},
			{Stage: "verification", ExecutionWave: 12, Caste: "watcher", Name: "Keen-prior", Task: "Verify prior phase", Status: "completed", Outputs: []string{"main_test.go"}},
		},
	}); err != nil {
		t.Fatalf("failed to seed prior manifest: %v", err)
	}
	if err := store.SaveJSON("last-build-claims.json", codexBuildClaims{
		FilesModified: []string{"main.go"},
		BuildPhase:    1,
		Timestamp:     now.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("failed to seed prior claims: %v", err)
	}

	statePath := filepath.Join(dataDir, "COLONY_STATE.json")
	before, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read state before plan-only: %v", err)
	}

	result, _, _, _, err := runCodexBuildPlanOnly(root, 2, nil)
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnly returned error: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)
	if manifest.Phase != 2 {
		t.Fatalf("manifest phase = %d, want 2", manifest.Phase)
	}

	after, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read state after plan-only: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("build --plan-only mutated COLONY_STATE.json\nbefore:\n%s\nafter:\n%s", string(before), string(after))
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	if state.Plan.Phases[0].Tasks[0].Status == colony.TaskCompleted {
		t.Fatal("plan-only should not persist prior task repair before build-finalize")
	}
	if state.Plan.Phases[1].Tasks[0].Status != colony.TaskPending {
		t.Fatalf("current task status = %s, want pending", state.Plan.Phases[1].Tasks[0].Status)
	}
}

func TestBuildPlanOnlyExecutionPlanRunsWatcherAfterSpecialists(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	goal := "Expose the real build worker sequence"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:          1,
				Name:        "Execution contract",
				Description: "Run implementation before independent specialist checks and final watcher verification",
				Mode:        colony.PhaseModePrototype,
				Status:      colony.PhaseReady,
				Tasks:       []colony.Task{{ID: &taskID, Goal: "Implement the build sequence contract", Status: colony.TaskPending}},
			}},
		},
	})

	result, _, _, _, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{HeavyFlag: true})
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnlyWithOptions returned error: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)
	if len(manifest.ExecutionPlan) == 0 {
		t.Fatal("manifest execution_plan should not be empty")
	}

	gotStages := make([]string, 0, len(manifest.ExecutionPlan))
	for _, step := range manifest.ExecutionPlan {
		gotStages = append(gotStages, step.Stage)
		if step.WorkerCount < 1 {
			t.Fatalf("execution step %+v has no workers", step)
		}
	}
	wantStages := []string{"wave", "probe", "measurement", "resilience", "verification"}
	if strings.Join(gotStages, ",") != strings.Join(wantStages, ",") {
		t.Fatalf("execution stages = %v, want %v", gotStages, wantStages)
	}

	last := manifest.ExecutionPlan[len(manifest.ExecutionPlan)-1]
	if last.Stage != "verification" || !containsString(last.Castes, "watcher") {
		t.Fatalf("final execution step = %+v, want watcher verification", last)
	}
	contract, ok := result["dispatch_contract"].(map[string]interface{})
	if !ok {
		t.Fatalf("dispatch_contract missing or wrong type: %T", result["dispatch_contract"])
	}
	if got := intValue(contract["wave_count"]); got != len(manifest.ExecutionPlan) {
		t.Fatalf("dispatch_contract wave_count = %d, want execution plan length %d", got, len(manifest.ExecutionPlan))
	}
	if got := intValue(contract["worker_count"]); got != len(manifest.Dispatches) {
		t.Fatalf("dispatch_contract worker_count = %d, want dispatch count %d", got, len(manifest.Dispatches))
	}
}

func TestBuildPlanOnlyManifestQueenExecutionPolicyExposesSpawnBudget(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	goal := "Expose Queen spawn budget metadata"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:          1,
				Name:        "Worker budget contract",
				Description: "Publish minimal Queen spawn limits without exposing prompts or provider details",
				Mode:        colony.PhaseModePrototype,
				Status:      colony.PhaseReady,
				Tasks:       []colony.Task{{ID: &taskID, Goal: "Validate spawn budget contract", Status: colony.TaskPending}},
			}},
		},
	})

	result, _, _, _, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{HeavyFlag: true})
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnlyWithOptions returned error: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal dispatch manifest: %v", err)
	}
	var manifestJSON map[string]interface{}
	if err := json.Unmarshal(encoded, &manifestJSON); err != nil {
		t.Fatalf("unmarshal dispatch manifest: %v", err)
	}

	rawPolicy, ok := manifestJSON["queen_execution_policy"].(map[string]interface{})
	if !ok {
		t.Fatalf("manifest missing queen_execution_policy: %#v", manifestJSON)
	}
	if got := stringValue(rawPolicy["review_depth"]); got != string(colony.VerificationDepthHeavy) {
		t.Fatalf("queen_execution_policy.review_depth = %q, want %q", got, colony.VerificationDepthHeavy)
	}
	if got := stringValue(rawPolicy["verification_depth"]); got != string(colony.VerificationDepthHeavy) {
		t.Fatalf("queen_execution_policy.verification_depth = %q, want %q", got, colony.VerificationDepthHeavy)
	}

	rawBudget, ok := rawPolicy["spawn_budget"].(map[string]interface{})
	if !ok {
		t.Fatalf("queen_execution_policy.spawn_budget missing from manifest policy: %#v", rawPolicy)
	}
	for _, key := range []string{
		"max_workers",
		"selected_workers",
		"worker_count",
		"max_selected_castes",
		"selected_castes",
		"pruned_workers",
		"pruned_castes",
		"required_castes",
		"overflow_required_workers",
		"relevance_threshold",
		"selected_reasons",
		"budget_unit",
		"reason",
		"flow_type",
		"risk_level",
		"castes",
		"counts",
	} {
		if _, ok := rawBudget[key]; !ok {
			t.Fatalf("spawn_budget missing stable JSON key %q: %#v", key, rawBudget)
		}
	}
	for key := range rawBudget {
		if strings.ContainsAny(key, "ABCDEFGHIJKLMNOPQRSTUVWXYZ-") {
			t.Fatalf("spawn_budget key %q should use stable snake_case JSON", key)
		}
	}
	if _, ok := rawBudget["provider_diagnostics"]; ok {
		t.Fatalf("spawn_budget must not include provider_diagnostics: %#v", rawBudget)
	}
	if got := intValue(rawBudget["worker_count"]); got != len(manifest.Dispatches) {
		t.Fatalf("spawn_budget.worker_count = %d, want dispatch count %d", got, len(manifest.Dispatches))
	}
	if got := intValue(rawBudget["selected_workers"]); got != len(manifest.Dispatches) {
		t.Fatalf("spawn_budget.selected_workers = %d, want dispatch count %d", got, len(manifest.Dispatches))
	}
	if got := intValue(rawBudget["max_workers"]); got != len(manifest.Dispatches) {
		t.Fatalf("spawn_budget.max_workers = %d, want concrete dispatch count %d", got, len(manifest.Dispatches))
	}
	if got := stringValue(rawBudget["budget_unit"]); got != "caste" {
		t.Fatalf("spawn_budget.budget_unit = %q, want caste", got)
	}
	if got := intValue(rawBudget["max_selected_castes"]); got < 1 {
		t.Fatalf("spawn_budget.max_selected_castes = %d, want positive caste budget", got)
	}
	if got := intValue(rawBudget["selected_castes"]); got != len(stringSliceValue(rawBudget["castes"])) {
		t.Fatalf("spawn_budget.selected_castes = %d, want castes length %d", got, len(stringSliceValue(rawBudget["castes"])))
	}
	if got := intValue(rawBudget["pruned_castes"]); got < 0 {
		t.Fatalf("spawn_budget.pruned_castes = %d, want non-negative", got)
	}
	if got := intValue(rawBudget["pruned_workers"]); got != intValue(rawBudget["pruned_castes"]) {
		t.Fatalf("spawn_budget.pruned_workers = %d, want pruned_castes %d for caste budget unit", got, intValue(rawBudget["pruned_castes"]))
	}
	if got := intValue(rawBudget["overflow_required_workers"]); got < 0 {
		t.Fatalf("spawn_budget.overflow_required_workers = %d, want non-negative", got)
	}
	for _, caste := range []string{"builder", "watcher"} {
		if !containsString(stringSliceValue(rawBudget["castes"]), caste) {
			t.Fatalf("spawn_budget.castes missing %s: %#v", caste, rawBudget["castes"])
		}
	}
	counts, ok := rawBudget["counts"].(map[string]interface{})
	if !ok {
		t.Fatalf("spawn_budget.counts missing or wrong type: %#v", rawBudget)
	}
	if got := intValue(counts["builder"]); got < 1 {
		t.Fatalf("spawn_budget.counts.builder = %d, want at least 1", got)
	}
	if strings.TrimSpace(stringValue(rawBudget["reason"])) == "" {
		t.Fatalf("spawn_budget.reason missing or empty: %#v", rawBudget)
	}
	if selectedReasons, ok := rawBudget["selected_reasons"].(map[string]interface{}); !ok || len(selectedReasons) == 0 {
		t.Fatalf("spawn_budget.selected_reasons missing or empty: %#v", rawBudget)
	}
}

func TestCodexBuildPlanOnlySpawnBudgetPreservesSafetyCastesUnderLightAndHeavy(t *testing.T) {
	tests := []struct {
		name      string
		phase     colony.Phase
		options   codexBuildOptions
		wantDepth colony.VerificationDepth
	}{
		{
			name: "security light keeps required safety castes",
			phase: colony.Phase{
				ID:          1,
				Name:        "Security hardening",
				Description: "Protect privileged configuration before production rollout",
				Mode:        colony.PhaseModeProduction,
			},
			options:   codexBuildOptions{LightFlag: true},
			wantDepth: colony.VerificationDepthLight,
		},
		{
			name: "security heavy keeps required safety castes",
			phase: colony.Phase{
				ID:          1,
				Name:        "Security hardening",
				Description: "Protect privileged configuration before production rollout",
				Mode:        colony.PhaseModeProduction,
			},
			options:   codexBuildOptions{HeavyFlag: true},
			wantDepth: colony.VerificationDepthHeavy,
		},
		{
			name: "final review light keeps required safety castes",
			phase: colony.Phase{
				ID:          1,
				Name:        "Final review",
				Description: "Complete final signoff before handoff",
				Mode:        colony.PhaseModeProduction,
			},
			options:   codexBuildOptions{LightFlag: true},
			wantDepth: colony.VerificationDepthLight,
		},
		{
			name: "final review heavy keeps required safety castes",
			phase: colony.Phase{
				ID:          1,
				Name:        "Final review",
				Description: "Complete final signoff before handoff",
				Mode:        colony.PhaseModeProduction,
			},
			options:   codexBuildOptions{HeavyFlag: true},
			wantDepth: colony.VerificationDepthHeavy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)

			dataDir := setupBuildFlowTest(t)
			root := filepath.Dir(filepath.Dir(dataDir))
			goal := "Preserve build safety castes"
			taskID := "1.1"
			tt.phase.Status = colony.PhaseReady
			tt.phase.Tasks = []colony.Task{{
				ID:     &taskID,
				Goal:   "Build release evidence and address blockers",
				Status: colony.TaskPending,
			}}
			createTestColonyState(t, dataDir, colony.ColonyState{
				Version:      "3.0",
				Goal:         &goal,
				State:        colony.StateREADY,
				ColonyDepth:  "full",
				CurrentPhase: 0,
				Plan:         colony.Plan{Phases: []colony.Phase{tt.phase}},
			})

			result, _, _, _, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, tt.options)
			if err != nil {
				t.Fatalf("runCodexBuildPlanOnlyWithOptions returned error: %v", err)
			}
			manifest := result["dispatch_manifest"].(codexBuildManifest)
			policy := manifest.QueenExecutionPolicy
			if policy.ReviewDepth != string(tt.wantDepth) {
				t.Fatalf("queen_execution_policy.review_depth = %q, want %q", policy.ReviewDepth, tt.wantDepth)
			}
			if policy.VerificationDepth != string(tt.wantDepth) {
				t.Fatalf("queen_execution_policy.verification_depth = %q, want %q", policy.VerificationDepth, tt.wantDepth)
			}
			if policy.SpawnBudget == nil {
				t.Fatalf("queen_execution_policy.spawn_budget missing for %s", tt.name)
			}

			for _, caste := range []string{"builder", "watcher", "probe", "gatekeeper", "auditor"} {
				if !containsString(policy.SpawnBudget.RequiredCastes, caste) {
					t.Fatalf("spawn_budget.required_castes missing %s: %+v", caste, policy.SpawnBudget.RequiredCastes)
				}
				if !containsString(policy.SpawnBudget.Castes, caste) {
					t.Fatalf("spawn_budget.castes missing %s: %+v", caste, policy.SpawnBudget.Castes)
				}
				if !buildManifestHasCaste(manifest, caste) {
					t.Fatalf("manifest dispatches missing %s after %s pruning: %v", caste, tt.wantDepth, buildManifestCastes(manifest))
				}
			}
		})
	}
}

func TestCodexBuildPlanOnlySpawnBudgetSeparatesCasteBudgetFromWorkerCount(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	goal := "Separate caste budget from worker dispatch count"
	taskIDs := []string{"1.1", "1.2", "1.3"}
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:          1,
				Name:        "Security release hardening",
				Description: "Production release signoff with multiple implementation tasks",
				Mode:        colony.PhaseModeProduction,
				Status:      colony.PhaseReady,
				Tasks: []colony.Task{
					{ID: &taskIDs[0], Goal: "Implement budget manifest contract", Status: colony.TaskPending},
					{ID: &taskIDs[1], Goal: "Add release hardening checks", Status: colony.TaskPending},
					{ID: &taskIDs[2], Goal: "Verify security signoff evidence", Status: colony.TaskPending},
				},
			}},
		},
	})

	result, _, _, _, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{HeavyFlag: true})
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnlyWithOptions returned error: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)
	budget := manifest.QueenExecutionPolicy.SpawnBudget
	if budget == nil {
		t.Fatalf("spawn_budget missing from manifest policy")
	}
	if budget.WorkerCount != len(manifest.Dispatches) {
		t.Fatalf("worker_count = %d, want dispatch count %d", budget.WorkerCount, len(manifest.Dispatches))
	}
	if budget.MaxWorkers != budget.WorkerCount || budget.SelectedWorkers != budget.WorkerCount {
		t.Fatalf("worker fields should use concrete dispatch units: %+v", budget)
	}
	if budget.BudgetUnit != "caste" {
		t.Fatalf("budget_unit = %q, want caste", budget.BudgetUnit)
	}
	if budget.MaxSelectedCastes <= 0 {
		t.Fatalf("max_selected_castes should expose Queen caste budget: %+v", budget)
	}
	if budget.WorkerCount <= budget.MaxSelectedCastes {
		t.Fatalf("fixture should prove worker_count can exceed max_selected_castes: %+v", budget)
	}
	if budget.SelectedCastes != len(budget.Castes) {
		t.Fatalf("selected_castes = %d, want castes length %d", budget.SelectedCastes, len(budget.Castes))
	}
	if len(budget.PolicyAddedCastes) == 0 {
		t.Fatalf("policy_added_castes should expose build-policy castes added after Queen selection: %+v", budget)
	}
	if budget.PrunedCastes == nil || budget.PrunedWorkers == nil {
		t.Fatalf("pruned budget counts should be present even when zero: %+v", budget)
	}
	if *budget.PrunedWorkers != *budget.PrunedCastes {
		t.Fatalf("pruned worker count should mirror caste count for caste budget unit: %+v", budget)
	}
	if budget.OverflowRequiredWorkers == nil || *budget.OverflowRequiredWorkers != 0 {
		t.Fatalf("overflow_required_workers = %v, want explicit zero for this fixture", budget.OverflowRequiredWorkers)
	}
}

func TestCodexBuildPlanOnlySpawnBudgetExplainsPrunedCastes(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	goal := "Explain pruned Queen castes"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:          1,
				Name:        "Broad release verification and migration hardening",
				Description: "Implement, test, audit, benchmark, document, refactor, package, and release a secure migration with legacy cleanup",
				Mode:        colony.PhaseModeProduction,
				Status:      colony.PhaseReady,
				Tasks:       []colony.Task{{ID: &taskID, Goal: "Implement release migration hardening and package safeguards", Status: colony.TaskPending}},
			}},
		},
	})

	result, _, _, _, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{HeavyFlag: true})
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnlyWithOptions returned error: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)
	budget := manifest.QueenExecutionPolicy.SpawnBudget
	if budget == nil {
		t.Fatal("spawn_budget missing from manifest policy")
	}
	if budget.PrunedCastes == nil || *budget.PrunedCastes == 0 {
		t.Fatalf("pruned_castes should explain Queen pruning in broad fixture: %+v", budget)
	}
	if budget.PrunedWorkers == nil || *budget.PrunedWorkers != *budget.PrunedCastes {
		t.Fatalf("pruned_workers should mirror pruned_castes for caste budget unit: %+v", budget)
	}
	if len(budget.SkippedCastes) == 0 {
		t.Fatalf("skipped_castes should list Queen-pruned castes: %+v", budget)
	}
	if len(budget.PrunedReasons) == 0 {
		t.Fatalf("pruned_reasons should explain Queen-pruned castes: %+v", budget)
	}
	for _, caste := range budget.SkippedCastes {
		reason := budget.PrunedReasons[caste]
		if !strings.Contains(reason, "not spawned") {
			t.Fatalf("pruned reason for %s should explain not spawned decision: %q", caste, reason)
		}
	}
}

func TestCodexBuildPlanOnlyPhaseFiveSafetyVerificationKeepsRequiredCastes(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	goal := "Full safety verification"
	taskID := "5.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:          5,
				Name:        "Full Safety Verification",
				Description: "Run the release-oriented verification set and inspect contract-sensitive output so adaptive pruning does not weaken state, security, release, or final safeguards.",
				Mode:        colony.PhaseModeProduction,
				Status:      colony.PhaseReady,
				Tasks:       []colony.Task{{ID: &taskID, Goal: "Run focused release/security/final safeguard checks and manually inspect any failures before closing.", Status: colony.TaskPending}},
			}},
		},
	})

	result, _, _, _, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{HeavyFlag: true})
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnlyWithOptions returned error: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)
	budget := manifest.QueenExecutionPolicy.SpawnBudget
	if budget == nil {
		t.Fatal("spawn_budget missing from manifest policy")
	}
	for _, caste := range []string{"builder", "watcher", "probe", "gatekeeper", "auditor"} {
		if !containsString(budget.RequiredCastes, caste) {
			t.Fatalf("required_castes missing %s: %+v", caste, budget.RequiredCastes)
		}
		if !containsString(budget.Castes, caste) {
			t.Fatalf("castes missing required %s: %+v", caste, budget.Castes)
		}
		if !buildManifestHasCaste(manifest, caste) {
			t.Fatalf("manifest dispatches missing required %s: %v", caste, buildManifestCastes(manifest))
		}
	}
}

func TestSuggestedBuildCasteDoesNotTreatInspectAsSpec(t *testing.T) {
	task := colony.Task{
		Goal: "Run focused release/security/final safeguard checks and manually inspect any failures before closing.",
	}

	if got := suggestedBuildCaste(task); got != "builder" {
		t.Fatalf("suggestedBuildCaste(inspect task) = %q, want builder", got)
	}
}

func TestSuggestedBuildCasteKeepsSpecificationTasksWithScout(t *testing.T) {
	task := colony.Task{
		Goal: "Research API specifications before implementation",
	}

	if got := suggestedBuildCaste(task); got != "scout" {
		t.Fatalf("suggestedBuildCaste(specification task) = %q, want scout", got)
	}
}

func TestBuildPlanOnlyIncludesRuntimeProviderDiagnostics(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	goal := "Surface provider diagnostics"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "standard",
		CurrentPhase: 0,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Provider diagnostics",
			Status: colony.PhaseReady,
			Tasks:  []colony.Task{{ID: &taskID, Goal: "Validate diagnostics ownership", Status: colony.TaskPending}},
		}}},
	})

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &buildProviderDiagnosticInvoker{} }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	result, _, _, _, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{})
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnlyWithOptions returned error: %v", err)
	}

	diagnostic := stringValue(result["provider_diagnostics"])
	if !strings.Contains(diagnostic, "runtime-owned provider detail") {
		t.Fatalf("provider_diagnostics = %q, want runtime-owned availability detail", diagnostic)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)
	if manifest.ProviderDiagnostics != diagnostic {
		t.Fatalf("manifest provider diagnostics = %q, want %q", manifest.ProviderDiagnostics, diagnostic)
	}
	contract := result["dispatch_contract"].(map[string]interface{})
	if visibility := stringSliceValue(contract["fallback_visibility"]); !containsString(visibility, "provider_diagnostics") {
		t.Fatalf("fallback_visibility missing provider_diagnostics: %v", visibility)
	}
}

func TestBuildPlanOnlyAddsAmbassadorForIntegrationPhases(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	goal := "Wire external service safely"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "standard",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:          1,
				Name:        "OpenAI webhook integration",
				Description: "Connect an external API without leaking secrets",
				Status:      colony.PhaseReady,
				Tasks: []colony.Task{{
					ID:          &taskID,
					Goal:        "Implement SDK client wrapper for the third-party webhook",
					Status:      colony.TaskPending,
					Constraints: []string{"OAuth credentials must come from environment variables"},
				}},
			}},
		},
	})

	result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnly returned error: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)
	var ambassador *codexBuildDispatch
	for i := range manifest.Dispatches {
		if manifest.Dispatches[i].Caste == "ambassador" {
			ambassador = &manifest.Dispatches[i]
			break
		}
	}
	if ambassador == nil {
		t.Fatalf("expected ambassador dispatch for integration phase, got %#v", manifest.Dispatches)
	}
	if ambassador.Stage != "integration" || ambassador.ExecutionWave != 4 {
		t.Fatalf("ambassador dispatch = %+v, want integration execution wave 4", *ambassador)
	}
	if got := codexAgentNameForCaste(ambassador.Caste); got != "aether-ambassador" {
		t.Fatalf("ambassador agent = %q, want aether-ambassador", got)
	}
}

func TestBuildPlanOnlyUsesQueenSelectedSecurityCastes(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	goal := "Secure auth refresh"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "standard",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:          1,
				Name:        "Auth token rotation",
				Description: "Implement secure token refresh and rotation",
				Mode:        colony.PhaseModeProduction,
				Status:      colony.PhaseReady,
				Tasks: []colony.Task{{
					ID:     &taskID,
					Goal:   "Implement token rotation endpoint with crypto",
					Status: colony.TaskPending,
				}},
			}},
		},
	})

	result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnly returned error: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)
	for _, caste := range []string{"architect", "gatekeeper"} {
		if !buildManifestHasCaste(manifest, caste) {
			t.Fatalf("expected Queen-selected %s dispatch for security phase, got %v", caste, buildManifestCastes(manifest))
		}
	}
}

func TestBuildPlanOnlyKeepsRoutineUIQueenSelectionLean(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	goal := "Build a routine settings panel"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:          3,
				Name:        "Settings UI panel",
				Description: "Build a settings panel for user preferences",
				Mode:        colony.PhaseModePrototype,
				Status:      colony.PhaseReady,
				Tasks: []colony.Task{{
					ID:     &taskID,
					Goal:   "Implement SettingsPanel component with form controls",
					Status: colony.TaskPending,
				}},
			}},
		},
	})

	result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnly returned error: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)
	if got, want := buildManifestCastes(manifest), []string{"builder", "probe", "measurer", "chaos", "watcher"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("dispatch castes = %v, want lean Queen plan %v", got, want)
	}
	for _, caste := range []string{"archaeologist", "oracle", "architect", "gatekeeper"} {
		if buildManifestHasCaste(manifest, caste) {
			t.Fatalf("routine UI phase should not include %s; got %v", caste, buildManifestCastes(manifest))
		}
	}
}

func TestBuildPlanOnlyStandardReviewSuppressesQueenPerformanceCastes(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	goal := "Optimize query performance"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "standard",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:          1,
				Name:        "Performance optimization",
				Description: "Optimize query latency and reduce memory usage",
				Mode:        colony.PhaseModePrototype,
				Status:      colony.PhaseReady,
				Tasks: []colony.Task{{
					ID:     &taskID,
					Goal:   "Benchmark and optimize slow queries",
					Status: colony.TaskPending,
				}},
			}},
		},
	})

	result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnly returned error: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)
	for _, caste := range []string{"measurer", "chaos"} {
		if buildManifestHasCaste(manifest, caste) {
			t.Fatalf("standard review should suppress Queen-selected %s, got %v", caste, buildManifestCastes(manifest))
		}
	}
}

func TestBuildPlanOnlyHeavyReviewAllowsPolicyMeasurerAndChaos(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	goal := "Optimize query performance"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:          4,
				Name:        "Performance optimization",
				Description: "Optimize query latency and reduce memory usage",
				Mode:        colony.PhaseModePrototype,
				Status:      colony.PhaseReady,
				Tasks: []colony.Task{{
					ID:     &taskID,
					Goal:   "Benchmark and optimize slow queries",
					Status: colony.TaskPending,
				}},
			}},
		},
	})

	result, _, _, _, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{HeavyFlag: true})
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnlyWithOptions returned error: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)
	for _, caste := range []string{"measurer", "chaos"} {
		if !buildManifestHasCaste(manifest, caste) {
			t.Fatalf("heavy full-depth review should allow policy %s, got %v", caste, buildManifestCastes(manifest))
		}
	}
}

func TestBuildPlanOnlyCLIForwardsVerificationDepth(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceBuildJSONOutput(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Optimize query performance"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:          4,
				Name:        "Performance optimization",
				Description: "Optimize query latency and reduce memory usage",
				Mode:        colony.PhaseModePrototype,
				Status:      colony.PhaseReady,
				Tasks: []colony.Task{{
					ID:     &taskID,
					Goal:   "Benchmark and optimize slow queries",
					Status: colony.TaskPending,
				}},
			}},
		},
	})

	rootCmd.SetArgs([]string{"build", "1", "--plan-only", "--verification-depth", "heavy"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build --plan-only --verification-depth returned error: %v", err)
	}
	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if result["review_depth"].(string) != string(colony.VerificationDepthHeavy) {
		t.Fatalf("review_depth = %q, want %q", result["review_depth"], colony.VerificationDepthHeavy)
	}
	for _, caste := range []string{"measurer", "chaos"} {
		if !buildEnvelopeHasCaste(result, caste) {
			t.Fatalf("expected CLI plan-only to forward heavy depth and include %s, got %v", caste, buildEnvelopeCastes(result))
		}
	}
}

func TestBuildCLIForwardsVerificationDepth(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceBuildJSONOutput(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Optimize query performance"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:          1,
				Name:        "Performance optimization",
				Description: "Optimize query latency and reduce memory usage",
				Mode:        colony.PhaseModePrototype,
				Status:      colony.PhaseReady,
				Tasks: []colony.Task{{
					ID:     &taskID,
					Goal:   "Benchmark and optimize slow queries",
					Status: colony.TaskPending,
				}},
			}},
		},
	})

	rootCmd.SetArgs([]string{"build", "1", "--synthetic", "--verification-depth", "heavy"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build --verification-depth returned error: %v", err)
	}
	rawOutput := stdout.(*bytes.Buffer).String()
	if strings.TrimSpace(rawOutput) == "" {
		t.Fatalf("expected build JSON output on stdout, got empty; stderr: %s", stderr.(*bytes.Buffer).String())
	}
	env := parseEnvelope(t, rawOutput)
	result := env["result"].(map[string]interface{})
	if result["review_depth"].(string) != string(colony.VerificationDepthHeavy) {
		t.Fatalf("review_depth = %q, want %q", result["review_depth"], colony.VerificationDepthHeavy)
	}
	for _, caste := range []string{"measurer", "chaos"} {
		if !buildEnvelopeHasCaste(result, caste) {
			t.Fatalf("expected CLI build to forward heavy depth and include %s, got %v", caste, buildEnvelopeCastes(result))
		}
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload state: %v", err)
	}
	dispatchCount := int(result["dispatch_count"].(float64))
	eventText := strings.Join(state.Events, "\n")
	wantEvent := fmt.Sprintf("Dispatched %d workers for phase 1", dispatchCount)
	if !strings.Contains(eventText, wantEvent) {
		t.Fatalf("build_dispatched event should match actual dispatch count %q, got %v", wantEvent, state.Events)
	}
}

func TestBuildFinalizeRecordsExternalTaskResultsForContinue(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceBuildJSONOutput(t)

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

	goal := "Finalize wrapper-spawned agents"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "standard",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:          1,
				Name:        "Wrapper finalize",
				Description: "Record external Task tool worker results as build evidence",
				Status:      colony.PhaseReady,
				Tasks:       []colony.Task{{ID: &taskID, Goal: "Create wrapper evidence", Status: colony.TaskPending}},
			}},
		},
	})

	result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnly returned error: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)
	if err := os.WriteFile(filepath.Join(root, "wrapper-evidence.txt"), []byte("external work\n"), 0644); err != nil {
		t.Fatalf("failed to write claimed file: %v", err)
	}
	writeClaimFileForTest(t, root, "cmd/main.go")

	dispatchResults := make([]codexExternalBuildWorkerResult, 0, len(manifest.Dispatches))
	for _, dispatch := range manifest.Dispatches {
		worker := codexExternalBuildWorkerResult{
			Stage:         dispatch.Stage,
			Wave:          dispatch.Wave,
			ExecutionWave: normalizedDispatchWave(dispatch),
			Caste:         dispatch.Caste,
			Name:          dispatch.Name,
			TaskID:        dispatch.TaskID,
			Status:        "completed",
			Summary:       dispatch.Name + " completed externally",
			Duration:      1.25,
			Handoff: codex.WorkerHandoff{
				CommandsRun:            []string{"go test ./..."},
				VerificationStatus:     "pass",
				NextWorkerInstructions: []string{"work complete"},
			},
		}
		if dispatch.Caste == "builder" {
			worker.FilesCreated = []string{"wrapper-evidence.txt"}
			worker.FilesModified = []string{"cmd/main.go"}
			worker.TestsWritten = []string{"wrapper-evidence.txt"}
		}
		dispatchResults = append(dispatchResults, worker)
	}
	completion := codexExternalBuildCompletion{
		DispatchManifest: &manifest,
		Dispatches:       dispatchResults,
	}
	completionData, err := json.MarshalIndent(completion, "", "  ")
	if err != nil {
		t.Fatalf("marshal completion: %v", err)
	}
	completionPath := filepath.Join(root, "completion.json")
	if err := os.WriteFile(completionPath, completionData, 0644); err != nil {
		t.Fatalf("write completion: %v", err)
	}

	rootCmd.SetArgs([]string{"build-finalize", "1", "--completion-file", completionPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build-finalize returned error: %v", err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(stdout.(*bytes.Buffer).Bytes(), &envelope); err != nil {
		t.Fatalf("failed to parse finalize output: %v\n%s", err, stdout.(*bytes.Buffer).String())
	}
	if envelope["ok"] != true {
		t.Fatalf("expected ok:true, got %v", envelope)
	}
	finalizeResult := envelope["result"].(map[string]interface{})
	if finalizeResult["dispatch_mode"].(string) != "external-task" {
		t.Fatalf("dispatch_mode = %q, want external-task", finalizeResult["dispatch_mode"])
	}
	if finalizeResult["next"].(string) != "aether continue" {
		t.Fatalf("next = %q, want aether continue", finalizeResult["next"])
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	if state.State != colony.StateBUILT {
		t.Fatalf("state = %s, want BUILT", state.State)
	}
	if state.CurrentPhase != 1 {
		t.Fatalf("current_phase = %d, want 1", state.CurrentPhase)
	}
	if state.Plan.Phases[0].Status != colony.PhaseInProgress {
		t.Fatalf("phase status = %s, want in_progress", state.Plan.Phases[0].Status)
	}
	if state.BuildStartedAt == nil {
		t.Fatal("expected BuildStartedAt to be set")
	}
	if state.Plan.Phases[0].Tasks[0].Status != colony.TaskCompleted {
		t.Fatalf("task status = %s, want completed", state.Plan.Phases[0].Tasks[0].Status)
	}

	var finalManifest codexBuildManifest
	if err := store.LoadJSON("build/phase-1/manifest.json", &finalManifest); err != nil {
		t.Fatalf("failed to load final manifest: %v", err)
	}
	if finalManifest.PlanOnly {
		t.Fatal("final manifest should not be plan_only")
	}
	if finalManifest.DispatchMode != "external-task" {
		t.Fatalf("manifest dispatch mode = %q, want external-task", finalManifest.DispatchMode)
	}
	if len(finalManifest.Dispatches) != len(manifest.Dispatches) {
		t.Fatalf("final manifest dispatches = %d, want %d", len(finalManifest.Dispatches), len(manifest.Dispatches))
	}
	if len(finalManifest.Tasks) != 1 || finalManifest.Tasks[0].Status != colony.TaskCompleted {
		t.Fatalf("final manifest task status = %+v, want completed", finalManifest.Tasks)
	}
	for _, dispatch := range finalManifest.Dispatches {
		if dispatch.Status != "completed" {
			t.Fatalf("dispatch %s status = %s, want completed", dispatch.Name, dispatch.Status)
		}
		if dispatch.TaskID == "" {
			continue
		}
		if len(dispatch.Outputs) == 0 {
			t.Fatalf("completed task dispatch %s has no output evidence", dispatch.Name)
		}
		outputRel := strings.TrimPrefix(dispatch.Outputs[0], ".aether/data/")
		outputData, err := os.ReadFile(filepath.Join(dataDir, outputRel))
		if err != nil {
			t.Fatalf("read output evidence for %s: %v", dispatch.Name, err)
		}
		if !strings.Contains(string(outputData), "## Recorded Outcome") || !strings.Contains(string(outputData), "wrapper-evidence.txt") {
			t.Fatalf("output evidence for %s does not contain final outcome evidence:\n%s", dispatch.Name, string(outputData))
		}
	}

	var claims codexBuildClaims
	if err := store.LoadJSON("last-build-claims.json", &claims); err != nil {
		t.Fatalf("failed to load claims: %v", err)
	}
	if claims.BuildPhase != 1 {
		t.Fatalf("claims phase = %d, want 1", claims.BuildPhase)
	}
	if len(claims.FilesCreated) != 1 || claims.FilesCreated[0] != "wrapper-evidence.txt" {
		t.Fatalf("claims files created = %v, want wrapper-evidence.txt", claims.FilesCreated)
	}
	if len(claims.TaskClaims) != 1 || claims.TaskClaims[0].TaskID != taskID {
		t.Fatalf("task claims = %+v, want task %s", claims.TaskClaims, taskID)
	}

	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	entries, err := spawnTree.Parse()
	if err != nil {
		t.Fatalf("parse spawn tree: %v", err)
	}
	if len(entries) != len(manifest.Dispatches) {
		t.Fatalf("spawn entries = %d, want %d", len(entries), len(manifest.Dispatches))
	}
	for _, entry := range entries {
		if entry.Status != "completed" {
			t.Fatalf("spawn entry %s status = %s, want completed", entry.AgentName, entry.Status)
		}
	}
}

func TestBuildWaveExecutionPlansRespectParallelMode(t *testing.T) {
	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-1", Task: "Task one"},
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-2", Task: "Task two"},
		{Stage: "wave", Wave: 2, Caste: "builder", Name: "Forge-3", Task: "Task three"},
	}

	inRepo := buildWaveExecutionPlans(dispatches, colony.ModeInRepo)
	if len(inRepo) != 2 {
		t.Fatalf("in-repo wave plans = %d, want 2", len(inRepo))
	}
	if inRepo[0].Strategy != "serial" {
		t.Fatalf("wave 1 strategy = %q, want serial", inRepo[0].Strategy)
	}
	if !strings.Contains(inRepo[0].Reason, "main working tree") {
		t.Fatalf("wave 1 reason = %q, want shared working tree guidance", inRepo[0].Reason)
	}
	if inRepo[1].Strategy != "serial" || inRepo[1].WorkerCount != 1 {
		t.Fatalf("wave 2 plan = %+v, want single-task serial", inRepo[1])
	}

	worktree := buildWaveExecutionPlans(dispatches, colony.ModeWorktree)
	if len(worktree) != 2 {
		t.Fatalf("worktree wave plans = %d, want 2", len(worktree))
	}
	if worktree[0].Strategy != "parallel" {
		t.Fatalf("worktree wave 1 strategy = %q, want parallel", worktree[0].Strategy)
	}
	if !strings.Contains(worktree[0].Reason, "isolated worktrees") {
		t.Fatalf("worktree wave 1 reason = %q, want isolated worktree guidance", worktree[0].Reason)
	}
}

func TestBuildFinalizeAcceptsVerificationOnlyOutputEvidence(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	goal := "Verification-only finalizer evidence"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:          1,
				Name:        "Verification-only phase",
				Description: "Run verification checks before release",
				Mode:        colony.PhaseModeProduction,
				Status:      colony.PhaseReady,
				Tasks:       []colony.Task{{ID: &taskID, Goal: "Run verification commands", Status: colony.TaskPending}},
			}},
		},
	})
	if err := os.WriteFile(filepath.Join(root, "verification.log"), []byte("verification passed\n"), 0644); err != nil {
		t.Fatalf("write verification evidence: %v", err)
	}

	result, _, _, _, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{HeavyFlag: true})
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnlyWithOptions returned error: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)
	dispatchResults := make([]codexExternalBuildWorkerResult, 0, len(manifest.Dispatches))
	for _, dispatch := range manifest.Dispatches {
		worker := codexExternalBuildWorkerResult{
			Stage:         dispatch.Stage,
			Wave:          dispatch.Wave,
			ExecutionWave: normalizedDispatchWave(dispatch),
			Caste:         dispatch.Caste,
			Name:          dispatch.Name,
			TaskID:        dispatch.TaskID,
			Status:        "completed",
			Summary:       dispatch.Name + " completed verification-only work",
			Handoff: codex.WorkerHandoff{
				CommandsRun:            []string{"go test ./..."},
				VerificationStatus:     "pass",
				NextWorkerInstructions: []string{"work complete"},
			},
		}
		if dispatch.TaskID != "" {
			worker.Outputs = []string{"verification.log"}
		}
		dispatchResults = append(dispatchResults, worker)
	}
	completion := codexExternalBuildCompletion{
		DispatchManifest: &manifest,
		Dispatches:       dispatchResults,
		// FilesCreated/FilesModified must be explicit empty arrays, not a
		// nil zero value: the completion-packet schema (generated from
		// codexBuildClaims's non-omitempty []string fields) requires both
		// keys present as arrays, even for a legitimate verification-only
		// submission with no file claims.
		Claims: &codexBuildClaims{FilesCreated: []string{}, FilesModified: []string{}},
	}

	_, state, _, finalDispatches, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("runCodexBuildFinalize returned error: %v", err)
	}
	if state.State != colony.StateBUILT {
		t.Fatalf("state = %s, want BUILT", state.State)
	}
	foundTaskOutput := false
	for _, dispatch := range finalDispatches {
		if dispatch.TaskID == "" {
			continue
		}
		if len(dispatch.Outputs) == 0 {
			t.Fatalf("verification-only task dispatch %s lost output evidence", dispatch.Name)
		}
		foundTaskOutput = true
	}
	if !foundTaskOutput {
		t.Fatal("expected at least one task dispatch with verification-only output evidence")
	}
}

func TestBuildSupportsTaskScopedRedispatch(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceBuildJSONOutput(t)

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

	goal := "Redispatch only the missing task"
	taskOneID := "1.1"
	taskTwoID := "1.2"
	now := time.Now().UTC()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		ColonyDepth:    "full",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Targeted redispatch",
					Status: colony.PhaseInProgress,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Keep the completed task closed", Status: colony.TaskCompleted},
						{ID: &taskTwoID, Goal: "Redispatch only the missing task", Status: colony.TaskInProgress, DependsOn: []string{taskOneID}},
					},
				},
			},
		},
	})

	rootCmd.SetArgs([]string{"build", "1", "--task", taskTwoID})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(stdout.(*bytes.Buffer).Bytes(), &envelope); err != nil {
		t.Fatalf("failed to parse build output: %v\n%s", err, stdout.(*bytes.Buffer).String())
	}
	result := envelope["result"].(map[string]interface{})
	selectedTasks := result["selected_tasks"].([]interface{})
	if len(selectedTasks) != 1 || selectedTasks[0].(string) != taskTwoID {
		t.Fatalf("selected_tasks = %v, want [%s]", selectedTasks, taskTwoID)
	}

	var manifest codexBuildManifest
	if err := store.LoadJSON("build/phase-1/manifest.json", &manifest); err != nil {
		t.Fatalf("failed to load build manifest: %v", err)
	}
	if len(manifest.SelectedTasks) != 1 || manifest.SelectedTasks[0] != taskTwoID {
		t.Fatalf("manifest selected tasks = %v, want [%s]", manifest.SelectedTasks, taskTwoID)
	}
	if len(manifest.Dispatches) != 2 {
		t.Fatalf("expected 2 manifest dispatches for targeted redispatch, got %d", len(manifest.Dispatches))
	}
	for _, dispatch := range manifest.Dispatches {
		if dispatch.TaskID != "" && dispatch.TaskID != taskTwoID {
			t.Fatalf("unexpected task-scoped dispatch %+v", dispatch)
		}
		switch dispatch.Stage {
		case "prep", "research", "design", "integration", "probe", "measurement":
			t.Fatalf("unexpected full-phase specialist during targeted redispatch: %+v", dispatch)
		}
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload colony state: %v", err)
	}
	if state.Plan.Phases[0].Tasks[0].Status != colony.TaskCompleted {
		t.Fatalf("task 1 status = %s, want completed", state.Plan.Phases[0].Tasks[0].Status)
	}
	if state.Plan.Phases[0].Tasks[1].Status != colony.TaskCompleted {
		t.Fatalf("task 2 status = %s, want completed", state.Plan.Phases[0].Tasks[1].Status)
	}
}

func TestBuildRepairsCompletedPriorPhaseTasksFromTrustedManifest(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Repair completed phase task statuses before next build"
	phaseOneTaskID := "1.1"
	phaseOneSecondTaskID := "1.2"
	phaseTwoTaskID := "2.1"
	now := time.Now().UTC()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 2,
		ColonyDepth:  "light",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Already closed phase",
					Status: colony.PhaseCompleted,
					Tasks: []colony.Task{
						{ID: &phaseOneTaskID, Goal: "Finish the first prior task", Status: colony.TaskPending},
						{ID: &phaseOneSecondTaskID, Goal: "Finish the second prior task", Status: colony.TaskInProgress, DependsOn: []string{phaseOneTaskID}},
					},
				},
				{
					ID:     2,
					Name:   "Next phase",
					Status: colony.PhaseReady,
					Tasks:  []colony.Task{{ID: &phaseTwoTaskID, Goal: "Start only after prior tasks are reconciled", Status: colony.TaskPending}},
				},
			},
		},
	})

	if err := store.SaveJSON("build/phase-1/manifest.json", codexBuildManifest{
		Phase:        1,
		PhaseName:    "Already closed phase",
		Goal:         goal,
		Root:         root,
		ColonyDepth:  "light",
		DispatchMode: "external-task",
		GeneratedAt:  now.Format(time.RFC3339),
		State:        string(colony.StateBUILT),
		ClaimsPath:   displayDataPath("last-build-claims.json"),
		Tasks: []codexBuildTaskPlan{
			{ID: phaseOneTaskID, Goal: "Finish the first prior task", Status: colony.TaskCompleted},
			{ID: phaseOneSecondTaskID, Goal: "Finish the second prior task", Status: colony.TaskCompleted, DependsOn: []string{phaseOneTaskID}},
		},
		Dispatches: []codexBuildDispatch{
			{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-prior-1", Task: "Finish the first prior task", Status: "completed", TaskID: phaseOneTaskID, Outputs: []string{"main.go"}},
			{Stage: "wave", Wave: 2, Caste: "builder", Name: "Forge-prior-2", Task: "Finish the second prior task", Status: "completed", TaskID: phaseOneSecondTaskID, Outputs: []string{"main.go"}},
			{Stage: "verification", Caste: "watcher", Name: "Keen-prior-3", Task: "Verify prior phase", Status: "completed", Outputs: []string{"main_test.go"}},
		},
	}); err != nil {
		t.Fatalf("failed to seed prior manifest: %v", err)
	}
	if err := store.SaveJSON("last-build-claims.json", codexBuildClaims{
		FilesModified: []string{"main.go"},
		BuildPhase:    1,
		Timestamp:     now.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("failed to seed prior claims: %v", err)
	}

	if _, err := runCodexBuild(root, 2, nil, true); err != nil {
		t.Fatalf("build phase 2 returned error: %v", err)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	prior := state.Plan.Phases[0]
	if prior.Tasks[0].Status != colony.TaskCompleted || prior.Tasks[1].Status != colony.TaskCompleted {
		t.Fatalf("prior completed phase task statuses = %s/%s, want completed/completed", prior.Tasks[0].Status, prior.Tasks[1].Status)
	}
}

func TestBuildRejectsSimulatedPriorPhaseManifestForTaskRepair(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Reject simulated completed phase task repair"
	phaseOneTaskID := "1.1"
	phaseTwoTaskID := "2.1"
	now := time.Now().UTC()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 2,
		ColonyDepth:  "light",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Simulated prior phase",
					Status: colony.PhaseCompleted,
					Tasks:  []colony.Task{{ID: &phaseOneTaskID, Goal: "Do not trust simulated work", Status: colony.TaskPending}},
				},
				{
					ID:     2,
					Name:   "Next phase",
					Status: colony.PhaseReady,
					Tasks:  []colony.Task{{ID: &phaseTwoTaskID, Goal: "Start only after real prior evidence", Status: colony.TaskPending}},
				},
			},
		},
	})

	if err := store.SaveJSON("build/phase-1/manifest.json", codexBuildManifest{
		Phase:        1,
		PhaseName:    "Simulated prior phase",
		Goal:         goal,
		Root:         root,
		ColonyDepth:  "light",
		DispatchMode: "simulated",
		GeneratedAt:  now.Format(time.RFC3339),
		State:        string(colony.StateBUILT),
		Tasks:        []codexBuildTaskPlan{{ID: phaseOneTaskID, Goal: "Do not trust simulated work", Status: colony.TaskCompleted}},
		Dispatches: []codexBuildDispatch{
			{Stage: "wave", Wave: 1, Caste: "builder", Name: "Fake-prior-1", Task: "Do not trust simulated work", Status: "completed", TaskID: phaseOneTaskID, Outputs: []string{"main.go"}},
		},
	}); err != nil {
		t.Fatalf("failed to seed simulated prior manifest: %v", err)
	}

	_, err := runCodexBuild(root, 2, nil, true)
	if err == nil {
		t.Fatal("expected build phase 2 to reject simulated prior manifest, got nil error")
	}
	if !strings.Contains(err.Error(), "simulated") || !strings.Contains(err.Error(), "cannot repair persisted task state") {
		t.Fatalf("error = %q, want simulated repair rejection", err.Error())
	}

	var state colony.ColonyState
	if loadErr := store.LoadJSON("COLONY_STATE.json", &state); loadErr != nil {
		t.Fatalf("failed to reload state: %v", loadErr)
	}
	if state.Plan.Phases[0].Tasks[0].Status == colony.TaskCompleted {
		t.Fatal("simulated manifest should not mark prior task completed")
	}
}

func TestBuildRecoversMissingPlanFromPersistedPlanningArtifact(t *testing.T) {
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

	goal := "Recover build after the saved plan vanished"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 2,
		ColonyDepth:  "full",
		Plan:         colony.Plan{Phases: []colony.Phase{}},
		Events: []string{
			"2026-04-21T07:50:00Z phase-1-complete: Audit complete.",
			"2026-04-21T08:20:00Z phase-2-complete: Standard designed.",
		},
	})
	if err := store.SaveJSON("planning/phase-plan.json", codexWorkerPlanArtifact{
		Confidence: codexPlanConfidence{Overall: 88},
		Phases: []codexWorkerPlanPhase{
			{Name: "Audit", Tasks: []codexWorkerPlanTask{{Goal: "Audit the existing notes"}}},
			{Name: "Design", Tasks: []codexWorkerPlanTask{{Goal: "Define the frontmatter standard"}}},
			{
				Name:        "Standardize core references",
				Description: "Apply the saved schema to the highest-value notes first.",
				Tasks: []codexWorkerPlanTask{
					{Goal: "Standardize pattern notes"},
					{Goal: "Standardize device specs"},
				},
				SuccessCriteria: []string{"Core notes share the same schema"},
			},
		},
	}); err != nil {
		t.Fatalf("failed to save planning artifact: %v", err)
	}

	result, err := runCodexBuild(root, 3, nil, false)
	if err != nil {
		t.Fatalf("runCodexBuild returned error: %v", err)
	}
	if next := result["next"].(string); next != "aether continue" {
		t.Fatalf("next = %q, want aether continue", next)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload colony state: %v", err)
	}
	if len(state.Plan.Phases) != 3 {
		t.Fatalf("phase count = %d, want 3", len(state.Plan.Phases))
	}
	if state.CurrentPhase != 3 {
		t.Fatalf("current_phase = %d, want 3", state.CurrentPhase)
	}
	if state.State != colony.StateBUILT {
		t.Fatalf("state = %s, want BUILT", state.State)
	}
	if state.Plan.Phases[0].Status != colony.PhaseCompleted {
		t.Fatalf("phase 1 status = %s, want completed", state.Plan.Phases[0].Status)
	}
	if state.Plan.Phases[1].Status != colony.PhaseCompleted {
		t.Fatalf("phase 2 status = %s, want completed", state.Plan.Phases[1].Status)
	}
	if state.Plan.Phases[2].Status != colony.PhaseInProgress {
		t.Fatalf("phase 3 status = %s, want in_progress", state.Plan.Phases[2].Status)
	}
	if state.Plan.Phases[2].Tasks[0].Status != colony.TaskCompleted {
		t.Fatalf("phase 3 task 1 status = %s, want completed", state.Plan.Phases[2].Tasks[0].Status)
	}
	if state.Plan.Phases[2].Tasks[1].Status != colony.TaskCompleted {
		t.Fatalf("phase 3 task 2 status = %s, want completed", state.Plan.Phases[2].Tasks[1].Status)
	}
}

func TestBuildRejectsDifferentActivePhase(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Do not dispatch a different active phase"
	activeTaskID := "1.1"
	nextTaskID := "2.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Already active",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &activeTaskID, Goal: "Finish the active work", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Not yet active",
					Status: colony.PhaseReady,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Future work", Status: colony.TaskPending}},
				},
			},
		},
	})

	var errBuf bytes.Buffer
	stderr = &errBuf

	rootCmd.SetArgs([]string{"build", "2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	if !strings.Contains(errBuf.String(), "phase 1 is already active") {
		t.Fatalf("expected active-phase rejection, got: %s", errBuf.String())
	}
}

func TestBuildAllocatesUniqueNamesWhenSpawnHistoryCollides(t *testing.T) {
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

	goal := "Avoid spawn tree collisions"
	taskID := "1.1"
	phase := colony.Phase{
		ID:          1,
		Name:        "Collision handling",
		Description: "Ensure new build workers do not reuse old spawn names",
		Status:      colony.PhaseReady,
		Tasks: []colony.Task{
			{ID: &taskID, Goal: "Implement collision-safe build dispatch names", Status: colony.TaskPending},
		},
	}
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{phase}},
	})

	baseDispatches := plannedBuildDispatches(phase, "standard")
	if len(baseDispatches) == 0 {
		t.Fatal("expected planned dispatches")
	}

	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := spawnTree.RecordSpawn("Queen", baseDispatches[0].Caste, baseDispatches[0].Name, "Old worker", 1); err != nil {
		t.Fatalf("failed to seed spawn tree: %v", err)
	}
	if err := spawnTree.UpdateStatus(baseDispatches[0].Name, "completed", "old run"); err != nil {
		t.Fatalf("failed to complete seeded spawn: %v", err)
	}

	rootCmd.SetArgs([]string{"build", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	var manifest codexBuildManifest
	if err := store.LoadJSON("build/phase-1/manifest.json", &manifest); err != nil {
		t.Fatalf("failed to load build manifest: %v", err)
	}

	if manifest.Dispatches[0].Name == baseDispatches[0].Name {
		t.Fatalf("expected collided worker name to be renamed, still got %q", manifest.Dispatches[0].Name)
	}
	if !strings.HasPrefix(manifest.Dispatches[0].Name, baseDispatches[0].Name+"-r") {
		t.Fatalf("expected retry-style suffix on renamed worker, got %q", manifest.Dispatches[0].Name)
	}
}

// seedBuildAttemptRecord writes a minimal buildAttemptRecord and its
// latest-attempt pointer directly to the store, bypassing beginBuildAttempt's
// ColonyState/workspace-fingerprint requirements, so tests can construct an
// attempt in an arbitrary status for a given phase.
func seedBuildAttemptRecord(t *testing.T, phaseNum int, status string, dispatches []codexBuildDispatch) {
	t.Helper()
	attemptID := fmt.Sprintf("attempt-test-phase-%d-%s", phaseNum, status)
	attemptRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseNum), "attempts", attemptID+".json"))
	now := time.Now().UTC().Format(time.RFC3339Nano)
	record := buildAttemptRecord{
		SchemaVersion: buildAttemptSchemaVersion,
		ID:            attemptID,
		Phase:         phaseNum,
		Status:        status,
		StartedAt:     now,
		UpdatedAt:     now,
		Dispatches:    dispatches,
	}
	if err := store.SaveJSON(attemptRel, record); err != nil {
		t.Fatalf("failed to seed build attempt record: %v", err)
	}
	if err := store.SaveJSON(latestBuildAttemptPointerPath(phaseNum), latestBuildAttemptPointer{
		SchemaVersion: buildAttemptSchemaVersion,
		AttemptID:     attemptID,
		Path:          displayDataPath(attemptRel),
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("failed to seed latest build attempt pointer: %v", err)
	}
}

func dispatchNames(dispatches []codexBuildDispatch) []string {
	names := make([]string, len(dispatches))
	for i, dispatch := range dispatches {
		names[i] = dispatch.Name
	}
	sort.Strings(names)
	return names
}

// TestEnsureUniqueBuildDispatchNamesStableAcrossRePlan is the headline D-09
// test: planning the same phase's still-open attempt twice in a row, without
// finalizing it, must produce identical worker names both times -- no -rN
// suffixes -- so results never need manual remapping between re-plans.
func TestEnsureUniqueBuildDispatchNamesStableAcrossRePlan(t *testing.T) {
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

	goal := "Keep worker names stable across re-plans"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Re-plan stability",
					Description: "Re-planning the same unfinalized attempt keeps worker names",
					Status:      colony.PhaseReady,
					Tasks: []colony.Task{
						{ID: &taskID, Goal: "Implement stable naming", Status: colony.TaskPending},
					},
					SuccessCriteria: []string{"Names stay stable"},
				},
			},
		},
	})

	_, _, _, firstDispatches, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatalf("first plan-only run failed: %v", err)
	}
	_, _, _, secondDispatches, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatalf("second plan-only run (re-plan) failed: %v", err)
	}

	firstNames := dispatchNames(firstDispatches)
	secondNames := dispatchNames(secondDispatches)
	if len(firstNames) == 0 {
		t.Fatal("expected at least one dispatch from the first plan-only run")
	}
	if !reflect.DeepEqual(firstNames, secondNames) {
		t.Fatalf("worker names changed across re-plan: first=%v second=%v", firstNames, secondNames)
	}
	retrySuffix := regexp.MustCompile(`-r[0-9]+$`)
	for _, name := range secondNames {
		if retrySuffix.MatchString(name) {
			t.Fatalf("re-plan produced a retry-suffixed name %q, want stable names", name)
		}
	}
}

// TestEnsureUniqueBuildDispatchNamesSuffixesCollisionFromDifferentPhase proves
// a name reused from a genuinely different phase's attempt still forces a
// suffix -- the phase-scoped exclusion in ensureUniqueBuildDispatchNames must
// not leak across phases (T-163.1-20).
func TestEnsureUniqueBuildDispatchNamesSuffixesCollisionFromDifferentPhase(t *testing.T) {
	saveGlobals(t)
	dataDir := t.TempDir() + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := spawnTree.RecordSpawn("Queen", "builder", "Hammer-44", "Phase 1 task", 1); err != nil {
		t.Fatalf("failed to seed spawn tree: %v", err)
	}
	// Phase 1 has its own active, unfinalized attempt that used this name --
	// that exclusion must not apply when planning a DIFFERENT phase.
	seedBuildAttemptRecord(t, 1, buildAttemptAwaiting, []codexBuildDispatch{{Name: "Hammer-44", Caste: "builder"}})

	dispatches := []codexBuildDispatch{{Name: "Hammer-44", Caste: "builder"}}
	allocated, err := ensureUniqueBuildDispatchNames(dispatches, 2)
	if err != nil {
		t.Fatalf("ensureUniqueBuildDispatchNames: %v", err)
	}
	if allocated[0].Name == "Hammer-44" {
		t.Fatalf("expected a same-name worker from a different phase to be renamed, still got %q", allocated[0].Name)
	}
	if !strings.HasPrefix(allocated[0].Name, "Hammer-44-r") {
		t.Fatalf("expected retry-style suffix, got %q", allocated[0].Name)
	}
}

// TestEnsureUniqueBuildDispatchNamesSuffixesCollisionFromSealedAttempt proves
// a name collision with a worker from a previously sealed (built) attempt of
// THIS SAME phase still forces a suffix -- reuse is scoped to unfinalized
// attempts only (T-163.1-20).
func TestEnsureUniqueBuildDispatchNamesSuffixesCollisionFromSealedAttempt(t *testing.T) {
	saveGlobals(t)
	dataDir := t.TempDir() + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := spawnTree.RecordSpawn("Queen", "builder", "Hammer-44", "Phase 1 task", 1); err != nil {
		t.Fatalf("failed to seed spawn tree: %v", err)
	}
	seedBuildAttemptRecord(t, 1, buildAttemptBuilt, []codexBuildDispatch{{Name: "Hammer-44", Caste: "builder"}})

	dispatches := []codexBuildDispatch{{Name: "Hammer-44", Caste: "builder"}}
	allocated, err := ensureUniqueBuildDispatchNames(dispatches, 1)
	if err != nil {
		t.Fatalf("ensureUniqueBuildDispatchNames: %v", err)
	}
	if allocated[0].Name == "Hammer-44" {
		t.Fatalf("expected a same-name worker from a sealed (built) attempt of the same phase to be renamed, still got %q", allocated[0].Name)
	}
	if !strings.HasPrefix(allocated[0].Name, "Hammer-44-r") {
		t.Fatalf("expected retry-style suffix, got %q", allocated[0].Name)
	}
}

// TestEnsureUniqueBuildDispatchNamesDistinguishesWithinRunCollisions proves
// the within-run collision guard survives the D-09 change: two dispatches
// produced by the same plan run that would share a name must still end up
// with distinct names.
func TestEnsureUniqueBuildDispatchNamesDistinguishesWithinRunCollisions(t *testing.T) {
	saveGlobals(t)
	dataDir := t.TempDir() + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	dispatches := []codexBuildDispatch{
		{Name: "Hammer-44", Caste: "builder"},
		{Name: "Hammer-44", Caste: "watcher"},
	}
	allocated, err := ensureUniqueBuildDispatchNames(dispatches, 1)
	if err != nil {
		t.Fatalf("ensureUniqueBuildDispatchNames: %v", err)
	}
	if allocated[0].Name == allocated[1].Name {
		t.Fatalf("expected two same-named dispatches in one run to get distinct names, both got %q", allocated[0].Name)
	}
}

type buildFailInvoker struct{}

func (f *buildFailInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	return codex.WorkerResult{}, context.DeadlineExceeded
}

func (f *buildFailInvoker) IsAvailable(ctx context.Context) bool { return false }

func (f *buildFailInvoker) ValidateAgent(path string) error { return nil }

type buildProviderDiagnosticInvoker struct{}

func (i *buildProviderDiagnosticInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	return codex.WorkerResult{}, context.DeadlineExceeded
}

func (i *buildProviderDiagnosticInvoker) IsAvailable(ctx context.Context) bool { return false }

func (i *buildProviderDiagnosticInvoker) ValidateAgent(path string) error { return nil }

func (i *buildProviderDiagnosticInvoker) Availability(context.Context) codex.AvailabilityStatus {
	return codex.AvailabilityStatus{
		Platform:  codex.PlatformCodex,
		Binary:    "codex",
		Available: false,
		Category:  codex.AvailabilityCategoryAuthInactive,
		Reason:    "runtime-owned provider detail",
	}
}

type recordedWorkerCall struct {
	TaskID  string
	Caste   string
	Timeout time.Duration
}

type timeoutRecordingInvoker struct {
	mu       sync.Mutex
	calls    []recordedWorkerCall
	onInvoke func(codex.WorkerConfig)
}

func (i *timeoutRecordingInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	if i.onInvoke != nil {
		i.onInvoke(config)
	}
	i.mu.Lock()
	i.calls = append(i.calls, recordedWorkerCall{
		TaskID:  config.TaskID,
		Caste:   config.Caste,
		Timeout: config.Timeout,
	})
	i.mu.Unlock()
	return codex.WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "completed",
		Summary:    "recorded worker completed",
		Duration:   time.Millisecond,
	}, nil
}

func (i *timeoutRecordingInvoker) IsAvailable(ctx context.Context) bool { return true }

func (i *timeoutRecordingInvoker) ValidateAgent(path string) error { return nil }

func (i *timeoutRecordingInvoker) hasCall(taskID string, timeout time.Duration) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	for _, call := range i.calls {
		if call.TaskID == taskID && call.Timeout == timeout {
			return true
		}
	}
	return false
}

func (i *timeoutRecordingInvoker) allTimeoutsEqual(timeout time.Duration) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	if len(i.calls) == 0 {
		return false
	}
	for _, call := range i.calls {
		if call.Timeout != timeout {
			return false
		}
	}
	return true
}

type unavailableMutatingInvoker struct {
	mutated bool
	mutate  func()
}

func (i *unavailableMutatingInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	return codex.WorkerResult{}, context.DeadlineExceeded
}

func (i *unavailableMutatingInvoker) IsAvailable(ctx context.Context) bool {
	if !i.mutated && i.mutate != nil {
		i.mutated = true
		i.mutate()
	}
	return false
}

func (i *unavailableMutatingInvoker) ValidateAgent(path string) error { return nil }

func TestBuildCommandExposesWorkerTimeoutFlag(t *testing.T) {
	if buildCmd.Flags().Lookup("worker-timeout") == nil {
		t.Fatal("expected build command to expose --worker-timeout")
	}
}

func TestBuildUsesWorkerTimeoutOverride(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Build honors timeout override"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:     1,
				Name:   "Timeout phase",
				Status: colony.PhaseReady,
				Tasks:  []colony.Task{{ID: &taskID, Goal: "Record timeout", Status: colony.TaskPending}},
			}},
		},
	})

	recorder := &timeoutRecordingInvoker{}
	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return recorder }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	if _, err := runCodexBuildWithOptions(root, 1, nil, false, codexBuildOptions{WorkerTimeout: 19 * time.Minute}); err != nil {
		t.Fatalf("runCodexBuildWithOptions returned error: %v", err)
	}
	if !recorder.allTimeoutsEqual(19 * time.Minute) {
		t.Fatalf("expected all build worker timeouts to be 19m, got %+v", recorder.calls)
	}
}

func TestBuildFinalizationDoesNotOverwritePausedState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Do not overwrite pause"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:     1,
				Name:   "Pause during build",
				Status: colony.PhaseReady,
				Tasks:  []colony.Task{{ID: &taskID, Goal: "Pause while worker runs", Status: colony.TaskPending}},
			}},
		},
	})

	mutated := false
	recorder := &timeoutRecordingInvoker{
		onInvoke: func(config codex.WorkerConfig) {
			if mutated {
				return
			}
			mutated = true
			var state colony.ColonyState
			if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
				t.Fatalf("load state during invoke: %v", err)
			}
			pausedAt := time.Now().UTC().Format(time.RFC3339)
			state.Paused = true
			state.PausedAt = &pausedAt
			if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
				t.Fatalf("pause state during invoke: %v", err)
			}
		},
	}
	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return recorder }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	_, err := runCodexBuildWithOptions(root, 1, nil, false, codexBuildOptions{})
	if !errors.Is(err, errRuntimeStateSuperseded) {
		t.Fatalf("expected superseded build error, got %v", err)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload state: %v", err)
	}
	if !state.Paused {
		t.Fatalf("expected paused state to be preserved, got %+v", state)
	}
	if state.State == colony.StateBUILT {
		t.Fatalf("stale build overwrote state to BUILT")
	}
}

func TestBuildRollbackDoesNotOverwritePausedState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Do not rollback over pause"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:     1,
				Name:   "Rollback pause",
				Status: colony.PhaseReady,
				Tasks:  []colony.Task{{ID: &taskID, Goal: "Fail after pause", Status: colony.TaskPending}},
			}},
		},
	})

	invoker := &unavailableMutatingInvoker{mutate: func() {
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			t.Fatalf("load state during availability: %v", err)
		}
		pausedAt := time.Now().UTC().Format(time.RFC3339)
		state.Paused = true
		state.PausedAt = &pausedAt
		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			t.Fatalf("pause state during availability: %v", err)
		}
	}}
	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	if _, err := runCodexBuild(root, 1, nil, false); err == nil {
		t.Fatal("expected build failure")
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload state: %v", err)
	}
	if !state.Paused {
		t.Fatalf("expected paused state to survive rollback, got %+v", state)
	}
	if state.State == colony.StateREADY && state.BuildStartedAt == nil {
		t.Fatalf("stale rollback restored pre-build READY state")
	}
}

func TestBuildRollsBackStateWhenDispatchFails(t *testing.T) {
	saveGlobals(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir to root: %v", err)
	}
	defer os.Chdir(oldDir)

	goal := "Rollback failed build dispatches"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Rollback phase",
					Status: colony.PhaseReady,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Try the failing build", Status: colony.TaskPending}},
				},
			},
		},
	})

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &buildFailInvoker{} }
	defer func() { newCodexWorkerInvoker = originalInvoker }()

	_, err = runCodexBuild(root, 1, nil, false)
	if err == nil {
		t.Fatal("expected build failure")
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	if state.State != colony.StateREADY {
		t.Fatalf("state = %s, want READY after rollback", state.State)
	}
	if state.CurrentPhase != 0 {
		t.Fatalf("current phase = %d, want 0 after rollback", state.CurrentPhase)
	}
	if state.BuildStartedAt != nil {
		t.Fatal("expected BuildStartedAt to be cleared by rollback")
	}
	if state.Plan.Phases[0].Status != colony.PhaseReady {
		t.Fatalf("phase status = %s, want ready after rollback", state.Plan.Phases[0].Status)
	}

	contextData, readErr := os.ReadFile(filepath.Join(root, ".aether", "CONTEXT.md"))
	if readErr != nil {
		t.Fatalf("expected CONTEXT.md after rollback: %v", readErr)
	}
	if !strings.Contains(string(contextData), "worker dispatcher is unavailable") {
		t.Fatalf("expected rollback context summary, got:\n%s", string(contextData))
	}
}

func TestBuildAllowsRetryWhenBuiltPhaseHasFailedDispatches(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir to root: %v", err)
	}
	defer os.Chdir(oldDir)

	goal := "Retry a poisoned built phase"
	taskID := "1.1"
	startedAt := mustParseRFC3339(t, "2026-04-17T12:00:00Z")
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateBUILT,
		CurrentPhase: 1,
		BuildStartedAt: func() *time.Time {
			ts := startedAt
			return &ts
		}(),
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Retry phase",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Recover the failed build", Status: colony.TaskInProgress}},
				},
			},
		},
	})

	if err := store.SaveJSON("build/phase-1/manifest.json", codexBuildManifest{
		Phase:        1,
		PhaseName:    "Retry phase",
		DispatchMode: "real",
		Dispatches: []codexBuildDispatch{
			{Name: "Brick-60", Caste: "builder", Status: "failed", Task: "Recover the failed build"},
			{Name: "Sentinel-29", Caste: "watcher", Status: "failed", Task: "Verify the failed build"},
		},
	}); err != nil {
		t.Fatalf("failed to seed manifest: %v", err)
	}
	if err := store.SaveJSON("last-build-claims.json", codexBuildClaims{
		BuildPhase: 1,
		Timestamp:  startedAt.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("failed to seed empty claims: %v", err)
	}

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &codex.FakeInvoker{} }
	defer func() { newCodexWorkerInvoker = originalInvoker }()

	if _, err := runCodexBuild(root, 1, nil, false); err != nil {
		t.Fatalf("build retry returned error: %v", err)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	if state.State != colony.StateBUILT {
		t.Fatalf("state = %s, want BUILT after retry", state.State)
	}
	if state.CurrentPhase != 1 {
		t.Fatalf("current phase = %d, want 1", state.CurrentPhase)
	}

	var manifest codexBuildManifest
	if err := store.LoadJSON("build/phase-1/manifest.json", &manifest); err != nil {
		t.Fatalf("failed to reload manifest: %v", err)
	}
	if len(manifest.Dispatches) == 0 {
		t.Fatal("expected retried dispatches in manifest")
	}
	if len(manifest.WorkerBriefs) == 0 {
		t.Fatal("expected retried build to regenerate worker briefs")
	}
	for _, dispatch := range manifest.Dispatches {
		if dispatch.Status == "failed" {
			t.Fatalf("expected retried dispatches to avoid seeded failed status, got %+v", dispatch)
		}
	}
}

func floatPtr(v float64) *float64 { return &v }

type terminalBuildResultInvoker struct {
	status string
}

func (i *terminalBuildResultInvoker) Invoke(_ context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	return codex.WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     i.status,
		Summary:    "controlled terminal worker result",
	}, nil
}

func (i *terminalBuildResultInvoker) IsAvailable(context.Context) bool { return true }
func (i *terminalBuildResultInvoker) ValidateAgent(string) error       { return nil }

type noChangePlatformInvoker struct{}

func (i *noChangePlatformInvoker) Invoke(_ context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	return codex.WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "completed",
		Summary:    "claimed success without changing the workspace",
	}, nil
}

func (i *noChangePlatformInvoker) IsAvailable(context.Context) bool { return true }
func (i *noChangePlatformInvoker) ValidateAgent(string) error       { return nil }
func (i *noChangePlatformInvoker) Platform() codex.Platform         { return codex.PlatformCodex }

func TestBuildDoesNotAdvanceWhenWorkersFailOrTimeout(t *testing.T) {
	for _, status := range []string{"failed", "blocked", "timeout"} {
		t.Run(status, func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)

			dataDir := setupBuildFlowTest(t)
			root := filepath.Dir(filepath.Dir(dataDir))
			withWorkingDir(t, root)
			goal := "Reject false build completion"
			taskID := "1.1"
			createTestColonyState(t, dataDir, colony.ColonyState{
				Version: "3.0",
				Goal:    &goal,
				State:   colony.StateREADY,
				Plan: colony.Plan{Phases: []colony.Phase{{
					ID:     1,
					Name:   "Truthful build",
					Status: colony.PhaseReady,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Implement the feature", Status: colony.TaskPending}},
				}}},
			})

			originalInvoker := newCodexWorkerInvoker
			newCodexWorkerInvoker = func() codex.WorkerInvoker {
				return &terminalBuildResultInvoker{status: status}
			}
			t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

			_, err := runCodexBuild(root, 1, nil, false)
			if err == nil || !strings.Contains(err.Error(), "did not complete cleanly") {
				t.Fatalf("build error = %v, want terminal-result failure", err)
			}

			var state colony.ColonyState
			if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
				t.Fatalf("reload colony state: %v", err)
			}
			if state.State == colony.StateBUILT {
				t.Fatalf("failed worker advanced colony to BUILT: %+v", state)
			}
			if strings.Contains(strings.Join(state.Events, "\n"), "|build_completed|") {
				t.Fatalf("failed worker emitted build_completed: %v", state.Events)
			}
		})
	}
}

func TestBuildDoesNotAdvanceOnProviderBackedNoOp(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	goal := "Reject a no-op implementation"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "No-op build",
			Status: colony.PhaseReady,
			Tasks:  []colony.Task{{ID: &taskID, Goal: "Implement the feature", Status: colony.TaskPending}},
		}}},
	})

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &noChangePlatformInvoker{} }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	_, err := runCodexBuild(root, 1, nil, false)
	if err == nil || !strings.Contains(err.Error(), "without observed file changes") {
		t.Fatalf("build error = %v, want no-op evidence failure", err)
	}
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload colony state: %v", err)
	}
	if state.State == colony.StateBUILT {
		t.Fatal("provider-backed no-op advanced colony to BUILT")
	}
}

func TestDiscoveryBuildAcceptsDurableReadOnlyTaskEvidence(t *testing.T) {
	dispatches := []codexBuildDispatch{
		{TaskID: "1.1", Caste: "scout", Status: "completed", Summary: "Mapped the current command surface."},
		{TaskID: "1.2", Caste: "scout", Status: "completed", Summary: "Recorded the active risks and boundaries."},
		{Caste: "watcher", Status: "completed", Summary: "Discovery evidence is internally consistent."},
	}
	phase := colony.Phase{
		ID:   1,
		Name: "Discovery and boundaries",
		Mode: colony.PhaseModeDiscovery,
		Tasks: []colony.Task{
			{Goal: "Read the current implementation"},
			{Goal: "Capture risks and constraints"},
		},
	}
	if err := validateRuntimeBuildDispatchResults(phase, dispatches, &codex.ClaimsSummary{}, true); err != nil {
		t.Fatalf("read-only discovery evidence was rejected: %v", err)
	}

	dispatches[0].Summary = ""
	if err := validateRuntimeBuildDispatchResults(phase, dispatches, &codex.ClaimsSummary{}, true); err == nil {
		t.Fatal("discovery phase advanced without a durable task summary")
	}
}

func mustParseRFC3339(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("failed to parse timestamp %q: %v", value, err)
	}
	return parsed
}

func TestResolvePheromoneSection_GroupsSignalsByType(t *testing.T) {
	saveGlobals(t)
	dataDir := t.TempDir() + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	recent := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{Type: "FOCUS", Content: json.RawMessage(`{"text":"security"}`), Active: true, Strength: floatPtr(0.8), CreatedAt: recent},
			{Type: "REDIRECT", Content: json.RawMessage(`{"text":"avoid global state"}`), Active: true, Strength: floatPtr(0.9), CreatedAt: recent},
			{Type: "FEEDBACK", Content: json.RawMessage(`{"text":"prefer interfaces"}`), Active: true, Strength: floatPtr(0.7), CreatedAt: recent},
		},
	}
	if err := store.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("failed to save pheromones: %v", err)
	}

	section := resolvePheromoneSection()
	if section == "" {
		t.Fatal("expected non-empty pheromone section when signals exist")
	}
	if !strings.Contains(section, "### Active Pheromone Signals") {
		t.Fatalf("missing section header in pheromone section:\n%s", section)
	}
	if !strings.Contains(section, "FOCUS") {
		t.Fatalf("missing FOCUS type in pheromone section:\n%s", section)
	}
	if !strings.Contains(section, "REDIRECT") {
		t.Fatalf("missing REDIRECT type in pheromone section:\n%s", section)
	}
	if !strings.Contains(section, "FEEDBACK") {
		t.Fatalf("missing FEEDBACK type in pheromone section:\n%s", section)
	}
	if !strings.Contains(section, "security") {
		t.Fatalf("missing signal content in pheromone section:\n%s", section)
	}
}

func TestResolvePheromoneSection_ReturnsEmptyWhenNoSignals(t *testing.T) {
	saveGlobals(t)
	dataDir := t.TempDir() + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	pf := colony.PheromoneFile{Signals: []colony.PheromoneSignal{}}
	if err := store.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("failed to save pheromones: %v", err)
	}

	section := resolvePheromoneSection()
	if section != "" {
		t.Fatalf("expected empty pheromone section when no signals, got:\n%s", section)
	}
}

func TestResolveSkillSection_FormatsMatchedSkills(t *testing.T) {
	saveGlobals(t)

	tmpDir := t.TempDir()
	hubDir := tmpDir + "/hub"
	skillsDir := filepath.Join(hubDir, "system", "skills", "colony", "test-skill")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}
	skillContent := "---\nname: test-skill\ntype: colony\ncategory: testing\nagent_roles:\n  - builder\n---\nThis is the test skill content."
	if err := os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("failed to write skill: %v", err)
	}

	t.Setenv("AETHER_HUB_DIR", hubDir)

	section := resolveSkillSection("builder", "testing task")
	if section == "" {
		t.Fatal("expected non-empty skill section when a matching skill exists")
	}
	if !strings.Contains(section, "### Skill: test-skill") {
		t.Fatalf("missing skill header in skill section:\n%s", section)
	}
	if !strings.Contains(section, "This is the test skill content") {
		t.Fatalf("missing skill content in skill section:\n%s", section)
	}
}

func TestResolveSkillSection_ReturnsEmptyWhenNoMatches(t *testing.T) {
	saveGlobals(t)

	tmpDir := t.TempDir()
	hubDir := tmpDir + "/hub"
	if err := os.MkdirAll(hubDir, 0755); err != nil {
		t.Fatalf("failed to create hub dir: %v", err)
	}

	t.Setenv("AETHER_HUB_DIR", hubDir)

	section := resolveSkillSection("builder", "some task")
	if section != "" {
		t.Fatalf("expected empty skill section when no skills exist, got:\n%s", section)
	}
}

func setupRuntimeSkillAssignmentHub(t *testing.T) {
	t.Helper()
	hubDir := filepath.Join(t.TempDir(), "hub")
	skillsDir := filepath.Join(hubDir, "system", "skills", "colony", "runtime-assignment")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}
	skillContent := `---
name: runtime-assignment
type: colony
category: testing
roles: archaeologist, architect, ambassador, auditor, builder, chaos, gatekeeper, measurer, oracle, probe, route_setter, scout, watcher
---
Runtime assignment skill content.`
	if err := os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("failed to write skill: %v", err)
	}
	t.Setenv("AETHER_HUB_DIR", hubDir)
}

func assertDispatchHasRuntimeSkillAssignment(t *testing.T, dispatch map[string]interface{}) {
	t.Helper()
	if count := intValue(dispatch["skill_count"]); count < 1 {
		t.Fatalf("dispatch missing skill_count: %+v", dispatch)
	}
	if names := stringSliceValue(dispatch["matched_skills"]); !containsString(names, "runtime-assignment") {
		t.Fatalf("dispatch missing runtime-assignment skill: %+v", dispatch)
	}
	if section := strings.TrimSpace(stringValue(dispatch["skill_section"])); !strings.Contains(section, "### Skill: runtime-assignment") {
		t.Fatalf("dispatch missing runtime skill section: %+v", dispatch)
	}
}

func buildManifestHasCaste(manifest codexBuildManifest, caste string) bool {
	for _, dispatch := range manifest.Dispatches {
		if dispatch.Caste == caste {
			return true
		}
	}
	return false
}

func buildManifestCastes(manifest codexBuildManifest) []string {
	castes := make([]string, 0, len(manifest.Dispatches))
	for _, dispatch := range manifest.Dispatches {
		castes = append(castes, dispatch.Caste)
	}
	return castes
}

func buildEnvelopeHasCaste(result map[string]interface{}, caste string) bool {
	for _, got := range buildEnvelopeCastes(result) {
		if got == caste {
			return true
		}
	}
	return false
}

func buildEnvelopeCastes(result map[string]interface{}) []string {
	rawDispatches, _ := result["dispatches"].([]interface{})
	castes := make([]string, 0, len(rawDispatches))
	for _, raw := range rawDispatches {
		dispatch, _ := raw.(map[string]interface{})
		if dispatch == nil {
			continue
		}
		caste, _ := dispatch["caste"].(string)
		if caste == "" {
			continue
		}
		castes = append(castes, caste)
	}
	return castes
}

// TestBuildInRepo_VerifiesGitClaimsForCompletedWorkers proves that in-repo
// builds verify completed worker claims against actual git state.
// After the fix in task 04-01, completed workers have their claims
// checked via applyObservedClaims, not trusted blindly.
func TestBuildInRepo_VerifiesGitClaimsForCompletedWorkers(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	runGit(t, root, "checkout", "-b", "main")

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")

	goal := "Verify in-repo claims are git-verified for completed workers"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 0,
		ColonyDepth:  "light",
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:     1,
				Name:   "Claims verification",
				Status: colony.PhaseReady,
				Tasks:  []colony.Task{{ID: &taskID, Goal: "Create a file and verify claims", Status: colony.TaskPending}},
			}},
		},
	})

	// Create an invoker that creates a file and reports it as completed
	invoker := &inRepoClaimsInvoker{
		root: root,
	}
	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	result, err := runCodexBuild(root, 1, nil, false)
	if err != nil {
		t.Fatalf("runCodexBuild returned error: %v", err)
	}

	dispatches, ok := result["dispatches"].([]map[string]interface{})
	if !ok || len(dispatches) == 0 {
		t.Fatalf("expected dispatches, got %v", result["dispatches"])
	}

	// Verify the worker completed
	dispatch := dispatches[0]
	if status, _ := dispatch["status"].(string); status != "completed" {
		t.Fatalf("expected completed status, got %q", status)
	}

	// Verify the claims file was written and contains the file
	var claims codexBuildClaims
	if err := store.LoadJSON("last-build-claims.json", &claims); err != nil {
		t.Fatalf("failed to load claims: %v", err)
	}

	foundClaimed := false
	for _, f := range claims.FilesCreated {
		if f == "pkg/feature.txt" {
			foundClaimed = true
			break
		}
	}
	if !foundClaimed {
		for _, f := range claims.FilesModified {
			if f == "pkg/feature.txt" {
				foundClaimed = true
				break
			}
		}
	}
	if !foundClaimed {
		t.Fatalf("expected pkg/feature.txt in claims, got FilesCreated=%v FilesModified=%v", claims.FilesCreated, claims.FilesModified)
	}

	// Verify the file exists on disk (proving git verification checked real state)
	if _, err := os.Stat(filepath.Join(root, "pkg", "feature.txt")); err != nil {
		t.Fatalf("expected pkg/feature.txt to exist on disk: %v", err)
	}
}

// inRepoClaimsInvoker is a test invoker that creates a file in-repo
// and reports completion with claimed files.
type inRepoClaimsInvoker struct {
	root string
}

func (i *inRepoClaimsInvoker) Invoke(_ context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	target := filepath.Join(cfg.Root, "pkg", "feature.txt")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return codex.WorkerResult{}, err
	}
	if err := os.WriteFile(target, []byte("in-repo build change\n"), 0644); err != nil {
		return codex.WorkerResult{}, err
	}
	return codex.WorkerResult{
		WorkerName:   cfg.WorkerName,
		Caste:        cfg.Caste,
		TaskID:       cfg.TaskID,
		Status:       "completed",
		Summary:      "in-repo build completed",
		FilesCreated: []string{"pkg/feature.txt"},
	}, nil
}

func (i *inRepoClaimsInvoker) IsAvailable(_ context.Context) bool { return true }
func (i *inRepoClaimsInvoker) ValidateAgent(_ string) error       { return nil }

func TestDispatchManifestAllCastes(t *testing.T) {
	castes := []string{
		"builder", "watcher", "scout", "oracle", "architect",
		"route_setter", "queen", "colonizer",
		"surveyor-nest", "surveyor-disciplines", "surveyor-pathogens", "surveyor-provisions",
		"keeper", "tracker", "probe", "weaver", "auditor",
		"chaos", "archaeologist", "gatekeeper", "includer", "measurer",
		"sage", "ambassador", "chronicler", "medic",
	}

	seenFiles := make(map[string]string)
	seenNames := make(map[string]string)

	for _, caste := range castes {
		t.Run(caste, func(t *testing.T) {
			file := codexAgentFileForCaste(caste)
			name := codexAgentNameForCaste(caste)

			if file == "" {
				t.Fatalf("empty file for caste %q", caste)
			}
			if name == "" {
				t.Fatalf("empty name for caste %q", caste)
			}
			if !strings.HasSuffix(file, ".toml") {
				t.Errorf("file %q for caste %q missing .toml suffix", file, caste)
			}
			if strings.Contains(name, ".toml") {
				t.Errorf("name %q for caste %q should not contain .toml", name, caste)
			}

			// Check uniqueness
			if prevCaste, exists := seenFiles[file]; exists {
				t.Errorf("caste %q and %q both map to file %q", prevCaste, caste, file)
			}
			if prevCaste, exists := seenNames[name]; exists {
				t.Errorf("caste %q and %q both map to name %q", prevCaste, caste, name)
			}
			seenFiles[file] = caste
			seenNames[name] = caste
		})
	}

	if len(seenFiles) != len(castes) {
		t.Errorf("expected %d unique files, got %d", len(castes), len(seenFiles))
	}
}

// TestBuildWorkerBriefOmitsHeartbeat locks in the removal of the heartbeat
// protocol. It previously asserted the opposite. The instruction asked a worker
// to write a file "roughly every 30 seconds" during its own turn; a model has no
// timer and cannot act between turns, so it was never satisfiable. Reinstating
// it would spend ~380 chars of every prompt teaching workers to ignore an
// instruction.
func TestBuildWorkerBriefOmitsHeartbeat(t *testing.T) {
	saveGlobals(t)

	tmpDir := t.TempDir()
	dispatch := codexBuildDispatch{
		Name:  "Hammer-23",
		Caste: "builder",
		Task:  "Implement feature X",
	}
	phase := colony.Phase{
		ID:          1,
		Name:        "Test Phase",
		Description: "Testing heartbeat removal",
	}

	brief := renderCodexBuildWorkerBrief(tmpDir, phase, dispatch, time.Now())

	if strings.Contains(brief, "Heartbeat Protocol") {
		t.Error("worker brief reinstated the 'Heartbeat Protocol' section; a model has no timer and cannot honour it")
	}
	if strings.Contains(brief, "heartbeat-") {
		t.Error("worker brief reinstated a heartbeat file reference")
	}
	if !strings.Contains(brief, "Implement feature X") {
		t.Error("worker brief lost its assignment")
	}
}

// TestBuildWorkerBriefOmitsPlaybooks locks in the removal of playbook injection
// from worker prompts. It previously asserted the opposite.
//
// Measured on a real brief before removal: playbook content was 5,733 of 7,485
// characters (76.6%) against an assignment of 79. The injected text was
// orchestrator guidance truncated at 2,800 chars, so a Builder received the
// opening of build-wave.md telling it "YOU (the Queen) will spawn workers
// directly" — a role contradiction at five times the mass of its actual task.
//
// Playbooks are still loaded for the orchestrator. They are simply not injected
// into individual worker prompts, which is not what they were written for.
func TestBuildWorkerBriefOmitsPlaybooks(t *testing.T) {
	saveGlobals(t)

	tmpDir := t.TempDir()
	playbookPath := filepath.Join(tmpDir, ".aether", "docs", "command-playbooks", "build-wave.md")
	if err := os.MkdirAll(filepath.Dir(playbookPath), 0755); err != nil {
		t.Fatalf("mkdir playbook dir: %v", err)
	}
	if err := os.WriteFile(playbookPath, []byte("# Build Wave\n\nYOU (the Queen) will spawn workers directly.\n"), 0644); err != nil {
		t.Fatalf("write playbook: %v", err)
	}

	dispatch := codexBuildDispatch{
		Name:  "Hammer-24",
		Caste: "builder",
		Task:  "Implement feature X",
	}
	phase := colony.Phase{ID: 1, Name: "Test Phase"}

	brief := renderCodexBuildWorkerBrief(tmpDir, phase, dispatch, time.Now())

	if strings.Contains(brief, "## Relevant Playbooks") {
		t.Error("worker brief reinstated the Relevant Playbooks section")
	}
	if strings.Contains(brief, "YOU (the Queen) will spawn workers directly") {
		t.Fatalf("worker brief injected orchestrator guidance into a worker prompt:\n%s", brief)
	}
	if !strings.Contains(brief, "Implement feature X") {
		t.Error("worker brief lost its assignment")
	}
}

// TestBuildWorkerBriefIsMostlyTask is the regression lock that matters. It
// asserts a proportion rather than the presence of any one section, so any
// future addition that pushes framework scaffolding past half the prompt fails
// here regardless of what that addition is called.
func TestBuildWorkerBriefIsMostlyTask(t *testing.T) {
	saveGlobals(t)

	tmpDir := t.TempDir()
	dispatch := codexBuildDispatch{
		Name:  "Hammer-25",
		Caste: "builder",
		Task:  "Add the exporter call to commands.rs and pass its result to the dashboard view",
	}
	phase := colony.Phase{
		ID:              1,
		Name:            "Wire the exporter",
		Description:     "Connect the vault exporter to the dashboard command",
		SuccessCriteria: []string{"Dashboard renders exporter output"},
	}

	brief := renderCodexBuildWorkerBrief(tmpDir, phase, dispatch, time.Now())

	taskChars := 0
	for _, section := range splitBriefSections(brief) {
		switch section.Name {
		case "Assignment", "Phase Objective", "Phase Success Criteria",
			"Task Success Criteria", "Dependencies", "Task Constraints", "Hints":
			taskChars += section.Chars
		}
	}

	if len(brief) == 0 {
		t.Fatal("empty worker brief")
	}
	share := float64(taskChars) / float64(len(brief)) * 100
	if share < 40 {
		t.Errorf("task-relevant content is %.1f%% of the worker brief (%d of %d chars); framework scaffolding now outweighs the task",
			share, taskChars, len(brief))
	}
}

func TestBuildWorkerBriefIncludesCodegraphContext(t *testing.T) {
	saveGlobals(t)

	tmpDir := t.TempDir()
	graphDir := filepath.Join(tmpDir, ".aether", "data")
	if err := os.MkdirAll(graphDir, 0755); err != nil {
		t.Fatalf("mkdir graph dir: %v", err)
	}
	graph := codegraph.CodeGraph{
		Files: []codegraph.FileNode{
			{Path: "src/app.ts", Language: "typescript"},
			{Path: "src/util.ts", Language: "typescript"},
		},
		Edges: []codegraph.DepEdge{{Source: "src/app.ts", Target: "src/util.ts", Type: "import"}},
	}
	if err := graph.Save(filepath.Join(graphDir, "codebase-graph.json")); err != nil {
		t.Fatalf("save graph: %v", err)
	}

	phase := colony.Phase{
		ID:   1,
		Name: "Graph Phase",
		Tasks: []colony.Task{{
			Goal:  "Update app shell",
			Hints: []string{"src/app.ts"},
		}},
	}
	dispatch := codexBuildDispatch{
		Name:      "Hammer-25",
		Caste:     "builder",
		Task:      "Update app shell",
		TaskID:    buildTaskID(phase.Tasks[0], 0),
		TaskIndex: 0,
	}

	brief := renderCodexBuildWorkerBrief(tmpDir, phase, dispatch, time.Now())

	if !strings.Contains(brief, "## Codebase Graph Context") {
		t.Fatalf("worker brief missing codegraph context:\n%s", brief)
	}
	if !strings.Contains(brief, "src/util.ts") {
		t.Fatalf("worker brief missing related dependency file:\n%s", brief)
	}
}

// TestBuildWorkerBriefIncludesSurveyAndResearch pins CONTEXT-01 and
// CONTEXT-04: territory survey findings and phase research findings must be
// demonstrably present in a build worker's actual prompt, not merely
// resolvable by functions nothing calls. Neither requirement had a test
// before this one — a grep for existing coverage
// (`grep -n 'func Test' cmd/codex_build_test.go`) confirmed OmitsHeartbeat,
// OmitsPlaybooks, IsMostlyTask, and IncludesCodegraphContext exist, and none
// of them asserts on resolveSurveySection or resolvePhaseResearchSection.
// The fixtures below are built from real files on disk in a temp colony,
// not stubbed resolvers — the requirement is that content reaches the
// prompt, and a stubbed resolver would prove only that a stub was called.
func TestBuildWorkerBriefIncludesSurveyAndResearch(t *testing.T) {
	t.Run("survey pointer list reaches the brief with real paths", func(t *testing.T) {
		saveGlobals(t)

		tmpDir := t.TempDir()
		dataDir := filepath.Join(tmpDir, ".aether", "data")
		surveyDir := filepath.Join(dataDir, "survey")
		if err := os.MkdirAll(surveyDir, 0755); err != nil {
			t.Fatalf("mkdir survey dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(surveyDir, "BLUEPRINT.md"), []byte("# Blueprint\n\nThe survey found a Go monorepo with cmd/ and pkg/."), 0644); err != nil {
			t.Fatalf("write survey doc: %v", err)
		}
		s, err := storage.NewStore(dataDir)
		if err != nil {
			t.Fatalf("new store: %v", err)
		}
		store = s

		dispatch := codexBuildDispatch{Name: "Hammer-26", Caste: "builder", Task: "Wire the exporter"}
		phase := colony.Phase{ID: 1, Name: "Test Phase"}

		brief := renderCodexBuildWorkerBrief(tmpDir, phase, dispatch, time.Now())

		if !strings.Contains(brief, "### Territory Survey") {
			t.Fatalf("worker brief missing Territory Survey section:\n%s", brief)
		}
		if !strings.Contains(brief, ".aether/data/survey/BLUEPRINT.md") {
			t.Fatalf("worker brief missing the delivered repo-relative survey path:\n%s", brief)
		}
	})

	t.Run("phase research reaches the brief with its content", func(t *testing.T) {
		saveGlobals(t)

		tmpDir := t.TempDir()
		dataDir := filepath.Join(tmpDir, ".aether", "data")
		researchDir := filepath.Join(dataDir, "phase-research")
		if err := os.MkdirAll(researchDir, 0755); err != nil {
			t.Fatalf("mkdir research dir: %v", err)
		}
		researchBody := "## Key Patterns\n\nUse the widget factory pattern for the exporter wiring."
		if err := os.WriteFile(filepath.Join(researchDir, "phase-1-research.md"), []byte(researchBody), 0644); err != nil {
			t.Fatalf("write research doc: %v", err)
		}
		s, err := storage.NewStore(dataDir)
		if err != nil {
			t.Fatalf("new store: %v", err)
		}
		store = s

		dispatch := codexBuildDispatch{Name: "Hammer-27", Caste: "builder", Task: "Wire the exporter"}
		phase := colony.Phase{ID: 1, Name: "Test Phase"}

		brief := renderCodexBuildWorkerBrief(tmpDir, phase, dispatch, time.Now())

		if !strings.Contains(brief, "## Phase Research") {
			t.Fatalf("worker brief missing Phase Research section:\n%s", brief)
		}
		if !strings.Contains(brief, "widget factory pattern") {
			t.Fatalf("worker brief missing the delivered research content:\n%s", brief)
		}
	})

	t.Run("staleness notice from task 1 reaches the brief alongside the survey", func(t *testing.T) {
		saveGlobals(t)

		tmpDir := t.TempDir()
		dataDir := filepath.Join(tmpDir, ".aether", "data")
		surveyDir := filepath.Join(dataDir, "survey")
		if err := os.MkdirAll(surveyDir, 0755); err != nil {
			t.Fatalf("mkdir survey dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(surveyDir, "BLUEPRINT.md"), []byte("# Blueprint\n\nsurvey content"), 0644); err != nil {
			t.Fatalf("write survey doc: %v", err)
		}
		s, err := storage.NewStore(dataDir)
		if err != nil {
			t.Fatalf("new store: %v", err)
		}
		store = s
		if err := store.SaveJSON("COLONY_STATE.json", colony.ColonyState{}); err != nil {
			t.Fatalf("save colony state: %v", err)
		}

		dispatch := codexBuildDispatch{Name: "Hammer-28", Caste: "builder", Task: "Wire the exporter"}
		phase := colony.Phase{ID: 1, Name: "Test Phase"}

		brief := renderCodexBuildWorkerBrief(tmpDir, phase, dispatch, time.Now())

		if !strings.Contains(brief, "never been surveyed") || !strings.Contains(brief, "/ant-colonize") {
			t.Fatalf("worker brief missing the survey staleness notice:\n%s", brief)
		}
	})

	t.Run("neither artifact present means neither section appears", func(t *testing.T) {
		saveGlobals(t)

		tmpDir := t.TempDir()
		dataDir := filepath.Join(tmpDir, ".aether", "data")
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			t.Fatalf("mkdir data dir: %v", err)
		}
		s, err := storage.NewStore(dataDir)
		if err != nil {
			t.Fatalf("new store: %v", err)
		}
		store = s

		dispatch := codexBuildDispatch{Name: "Hammer-29", Caste: "builder", Task: "Wire the exporter"}
		phase := colony.Phase{ID: 1, Name: "Test Phase"}

		brief := renderCodexBuildWorkerBrief(tmpDir, phase, dispatch, time.Now())

		if strings.Contains(brief, "### Territory Survey") {
			t.Errorf("worker brief has a Territory Survey heading with no survey data on disk:\n%s", brief)
		}
		if strings.Contains(brief, "## Phase Research") {
			t.Errorf("worker brief has a Phase Research heading with no research data on disk:\n%s", brief)
		}
	})
}

func TestBuildDispatchStartsHeartbeatMonitor(t *testing.T) {
	saveGlobals(t)

	ctx := context.Background()
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("create data dir: %v", err)
	}

	taskID := "1.1"
	phase := colony.Phase{
		ID:     1,
		Name:   "Test Phase",
		Status: colony.PhaseReady,
		Tasks: []colony.Task{
			{ID: &taskID, Goal: "Test task", Status: colony.TaskPending},
		},
	}

	dispatches := []codexBuildDispatch{
		{Name: "Hammer-23", Caste: "builder", Task: "Test task", TaskID: taskID},
	}

	invoker := &codex.FakeInvoker{}
	results, _, _, err := executeCodexBuildDispatches(ctx, tmpDir, phase, dispatches, time.Now(), invoker, colony.ModeInRepo, 0, 3, false, nil)
	if err != nil {
		t.Fatalf("execute dispatches: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// After dispatch completes, heartbeat files should be cleaned up
	matches, _ := filepath.Glob(filepath.Join(dataDir, "heartbeat-*.json"))
	if len(matches) > 0 {
		t.Errorf("expected heartbeat files cleaned up after dispatch, found: %v", matches)
	}
}
