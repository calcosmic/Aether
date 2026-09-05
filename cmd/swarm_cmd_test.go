package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

type swarmTestInvoker struct {
	blockedCaste string
	configs      []codex.WorkerConfig
}

func (i *swarmTestInvoker) Invoke(_ context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	i.configs = append(i.configs, cfg)

	response := swarmWorkerResponse{
		Role:    cfg.Caste,
		Status:  "completed",
		Summary: cfg.Caste + " completed the swarm pass.",
		Evidence: []string{
			"cmd/swarm_cmd.go",
		},
		Recommendation: "Run aether continue if the active colony phase was waiting on this bug fix.",
		Verification: []string{
			"go test ./...",
		},
	}
	result := codex.WorkerResult{
		WorkerName: cfg.WorkerName,
		Caste:      cfg.Caste,
		TaskID:     cfg.TaskID,
		Status:     "completed",
		Summary:    response.Summary,
		Duration:   time.Second,
	}

	switch cfg.Caste {
	case "tracker":
		response.Summary = "Tracked the failure to an unchecked nil access in the auth handler."
		response.Findings = []string{"panic originates from a missing nil guard in the auth handler"}
		response.RootCause = "auth handler dereferences a missing session dependency"
		result.Summary = response.Summary
	case "scout":
		response.Summary = "Found the relevant handler and the existing test pattern used by nearby modules."
		response.Findings = []string{"pkg/auth/handler.go matches the failure path", "pkg/auth/handler_test.go has the nearest regression pattern"}
		result.Summary = response.Summary
	case "archaeologist":
		response.Summary = "Git history shows the regression was introduced during a recent handler cleanup."
		response.Findings = []string{"recent auth cleanup removed the nil guard"}
		result.Summary = response.Summary
	case "builder":
		response.Summary = "Added the missing nil guard and a regression test."
		response.ProposedFix = "Restore the nil guard in pkg/auth/handler.go and cover it in pkg/auth/handler_test.go."
		response.FilesTouched = []string{"pkg/auth/handler.go"}
		response.TestsWritten = []string{"pkg/auth/handler_test.go"}
		response.Verification = []string{"go test ./pkg/auth"}
		result.Summary = response.Summary
		result.FilesModified = append(result.FilesModified, response.FilesTouched...)
		result.TestsWritten = append(result.TestsWritten, response.TestsWritten...)
	case "watcher":
		response.Summary = "Verified the fix with the focused auth test suite."
		response.Verification = []string{"go test ./pkg/auth"}
		result.Summary = response.Summary
	}

	if cfg.Caste == i.blockedCaste {
		response.Status = "blocked"
		response.Summary = cfg.Caste + " hit a blocking issue."
		response.Recommendation = "Resolve the missing fixture before retrying the swarm."
		result.Status = "blocked"
		result.Summary = response.Summary
		result.Blockers = []string{"missing test fixture"}
	}

	if strings.TrimSpace(cfg.ResponsePath) == "" {
		return codex.WorkerResult{}, context.Canceled
	}
	if err := os.MkdirAll(filepath.Dir(cfg.ResponsePath), 0755); err != nil {
		return codex.WorkerResult{}, err
	}
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return codex.WorkerResult{}, err
	}
	if err := os.WriteFile(cfg.ResponsePath, append(data, '\n'), 0644); err != nil {
		return codex.WorkerResult{}, err
	}
	return result, nil
}

func (i *swarmTestInvoker) IsAvailable(_ context.Context) bool { return true }
func (i *swarmTestInvoker) ValidateAgent(_ string) error       { return nil }

func TestSwarmDestroyRunsWorkerWavesAndReturnsStructuredResult(t *testing.T) {
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

	goal := "Destroy a stubborn auth bug"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:     1,
				Name:   "Bug fix",
				Status: colony.PhaseReady,
				Tasks:  []colony.Task{{ID: &taskID, Goal: "Fix the auth bug", Status: colony.TaskPending}},
			}},
		},
	})

	originalInvoker := newSwarmWorkerInvoker
	invoker := &swarmTestInvoker{}
	newSwarmWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	defer func() { newSwarmWorkerInvoker = originalInvoker }()

	rootCmd.SetArgs([]string{"swarm", "Auth panic when session is missing"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := result["mode"]; got != "destroy" {
		t.Fatalf("mode = %v, want destroy", got)
	}
	if got := result["status"]; got != "completed" {
		t.Fatalf("status = %v, want completed", got)
	}
	// Trio + Queen-selected gatekeeper for an auth bug; Scout and
	// Archaeologist are relevance-selected now and this wording carries no
	// research or history signal (TestSwarmTrivialBugSkipsHistoryAndResearch).
	if got := result["worker_count"]; got != float64(4) {
		t.Fatalf("worker_count = %v, want 4", got)
	}
	if got := result["autopilot_available"]; got != true {
		t.Fatalf("autopilot_available = %v, want true", got)
	}
	if got := result["root_cause"]; got == "" {
		t.Fatalf("expected root_cause in result, got %v", result)
	}
	if got := result["solution"]; got == "" {
		t.Fatalf("expected solution in result, got %v", result)
	}
	if got := result["next"]; got != "aether status" {
		t.Fatalf("next = %v, want aether status", got)
	}

	if len(invoker.configs) != 4 {
		t.Fatalf("expected 4 worker configs, got %d", len(invoker.configs))
	}
	for _, cfg := range invoker.configs {
		if strings.TrimSpace(cfg.ResponsePath) == "" {
			t.Fatalf("expected response path for %s", cfg.WorkerName)
		}
		if strings.TrimSpace(cfg.PheromoneSection) != "" {
			t.Fatalf("expected empty pheromone section in swarm test without signals, got %q", cfg.PheromoneSection)
		}
	}

	spawnTreeData, err := os.ReadFile(filepath.Join(dataDir, "spawn-tree.txt"))
	if err != nil {
		t.Fatalf("read spawn-tree: %v", err)
	}
	for _, caste := range []string{"tracker", "gatekeeper", "builder", "watcher"} {
		if !strings.Contains(string(spawnTreeData), "|Swarm|"+caste+"|") {
			t.Fatalf("spawn tree missing %s entry:\n%s", caste, string(spawnTreeData))
		}
	}
}

func TestAllSwarmPlansUseQueenSelectedGatekeeperForAuthBug(t *testing.T) {
	root := t.TempDir()

	plans := allSwarmPlans(root, "Auth token regression in session permissions")

	if !swarmPlansHaveCaste(plans, "gatekeeper") {
		t.Fatalf("swarm plans missing Queen-selected gatekeeper: %+v", plans)
	}
	for _, caste := range []string{"tracker", "builder", "watcher"} {
		if !swarmPlansHaveCaste(plans, caste) {
			t.Fatalf("swarm plans missing required %s: %+v", caste, plans)
		}
	}
}

func TestSwarmDestroySurfacesBlockedWorkers(t *testing.T) {
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

	goal := "Destroy a stubborn auth bug"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	})

	originalInvoker := newSwarmWorkerInvoker
	newSwarmWorkerInvoker = func() codex.WorkerInvoker {
		return &swarmTestInvoker{blockedCaste: "watcher"}
	}
	defer func() { newSwarmWorkerInvoker = originalInvoker }()

	rootCmd.SetArgs([]string{"swarm", "Auth panic when session is missing"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := result["status"]; got != "blocked" {
		t.Fatalf("status = %v, want blocked", got)
	}
	blockers, ok := result["blockers"].([]interface{})
	if !ok || len(blockers) == 0 {
		t.Fatalf("expected blockers in result, got %v", result["blockers"])
	}
	if !strings.Contains(stringValue(blockers[0]), "missing test fixture") {
		t.Fatalf("unexpected blocker payload: %v", blockers)
	}
}

func TestSwarmPlanOnlyPrintsManifestAndPersistsIssuanceOnly(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Plan visible swarm workers"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	})

	rootCmd.SetArgs([]string{"swarm", "--plan-only", "Auth panic when session is missing"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm --plan-only returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := result["dispatch_mode"]; got != "plan-only" {
		t.Fatalf("dispatch_mode = %v, want plan-only", got)
	}
	if got, _ := result["requires_finalizer"].(bool); !got {
		t.Fatalf("requires_finalizer = %v, want true", result["requires_finalizer"])
	}
	manifest := result["swarm_manifest"].(map[string]interface{})
	if got := manifest["finalizer_command"]; !strings.Contains(stringValue(got), "swarm-finalize") {
		t.Fatalf("finalizer_command = %v", got)
	}
	swarmID := strings.TrimSpace(stringValue(manifest["swarm_id"]))
	if swarmID == "" {
		t.Fatal("swarm manifest did not include a swarm_id")
	}
	workers := result["workers"].([]interface{})
	if len(workers) != 4 {
		t.Fatalf("workers = %d, want 4", len(workers))
	}
	if !workerMapsHaveCaste(workers, "gatekeeper") {
		t.Fatalf("workers missing Queen-selected gatekeeper: %+v", workers)
	}
	issuancePath := filepath.Join(dataDir, "swarms", swarmID, "issuance.json")
	var issuance map[string]interface{}
	if err := store.LoadJSON(filepath.ToSlash(filepath.Join("swarms", swarmID, "issuance.json")), &issuance); err != nil {
		t.Fatalf("plan-only did not persist a runtime issuance at %s: %v", issuancePath, err)
	}
	if got := strings.TrimSpace(stringValue(issuance["manifest_sha256"])); got == "" {
		t.Fatalf("issuance has no manifest digest: %#v", issuance)
	}
	if got := strings.TrimSpace(stringValue(issuance["status"])); got != "issued" {
		t.Fatalf("issuance status = %q, want issued: %#v", got, issuance)
	}
	entries, err := os.ReadDir(filepath.Join(dataDir, "swarms", swarmID))
	if err != nil {
		t.Fatalf("read issued swarm directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "issuance.json" {
		t.Fatalf("plan-only wrote artifacts beyond the issuance record: %+v", entries)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "spawn-tree.txt")); !os.IsNotExist(err) {
		t.Fatalf("plan-only should not write spawn-tree, stat err=%v", err)
	}
}

func TestSwarmFinalizeRecordsExternalTaskResults(t *testing.T) {
	tests := []struct {
		status      string
		wantOutcome string
	}{
		{status: "completed", wantOutcome: "completed"},
		{status: "passed", wantOutcome: "completed"},
		{status: "code_written", wantOutcome: "completed"},
		{status: "blocked", wantOutcome: "blocked"},
		{status: "failed", wantOutcome: "failed"},
		{status: "timeout", wantOutcome: "failed"},
	}

	for _, tc := range tests {
		t.Run(tc.status, func(t *testing.T) {
			saveGlobals(t)
			dataDir := setupBuildFlowTest(t)
			root := filepath.Dir(filepath.Dir(dataDir))
			withWorkingDir(t, root)
			goal := "Finalize visible swarm workers"
			createTestColonyState(t, dataDir, colony.ColonyState{
				Version: "3.0",
				Goal:    &goal,
				State:   colony.StateREADY,
			})

			manifest := issuedSwarmManifestForTest(t, root, "Auth panic when session is missing")
			dispatches := validExternalSwarmResults(manifest, tc.status)
			result, err := runSwarmFinalize(root, externalSwarmCompletion{
				SwarmManifest: &manifest,
				Dispatches:    dispatches,
			})
			if err != nil {
				t.Fatalf("runSwarmFinalize(%s): %v", tc.status, err)
			}
			if got := result["dispatch_mode"]; got != "external-task" {
				t.Fatalf("dispatch_mode = %v, want external-task", got)
			}
			if got := result["status"]; got != tc.wantOutcome {
				t.Fatalf("status = %v, want %s", got, tc.wantOutcome)
			}
			workers := result["workers"].([]map[string]interface{})
			if len(workers) != len(manifest.Dispatches) {
				t.Fatalf("workers = %d, want %d", len(workers), len(manifest.Dispatches))
			}
			for i, plan := range manifest.Dispatches {
				for field, want := range map[string]string{
					"name": plan.Name, "caste": plan.Caste, "role": plan.Role, "task": plan.Task,
				} {
					if got := strings.TrimSpace(stringValue(workers[i][field])); got != strings.TrimSpace(want) {
						t.Errorf("worker %d %s = %q, want manifest value %q", i, field, got, want)
					}
				}
			}
			if _, err := os.Stat(filepath.Join(dataDir, "swarms", manifest.SwarmID, "result.json")); err != nil {
				t.Fatalf("expected swarm result artifact: %v", err)
			}
		})
	}
}

func TestSwarmFinalizeRejectsUnboundOrNonTerminalEvidenceWithoutMutation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*swarmManifest, *[]swarmWorkerExecution)
	}{
		{name: "duplicate name", mutate: func(_ *swarmManifest, results *[]swarmWorkerExecution) {
			(*results)[1].Name = (*results)[0].Name
		}},
		{name: "duplicate role mapping", mutate: func(_ *swarmManifest, results *[]swarmWorkerExecution) {
			(*results)[1].Role = (*results)[0].Role
		}},
		{name: "surplus worker", mutate: func(_ *swarmManifest, results *[]swarmWorkerExecution) {
			extra := (*results)[0]
			extra.Name = "surplus-worker"
			extra.Role = "surplus-role"
			extra.Response.Role = "surplus-role"
			*results = append(*results, extra)
		}},
		{name: "missing worker", mutate: func(_ *swarmManifest, results *[]swarmWorkerExecution) {
			*results = (*results)[:len(*results)-1]
		}},
		{name: "forged caste", mutate: func(_ *swarmManifest, results *[]swarmWorkerExecution) {
			(*results)[0].Caste = "forged-caste"
		}},
		{name: "forged role", mutate: func(_ *swarmManifest, results *[]swarmWorkerExecution) {
			(*results)[0].Role = "forged-role"
		}},
		{name: "forged task", mutate: func(_ *swarmManifest, results *[]swarmWorkerExecution) {
			(*results)[0].Task = "forged task"
		}},
		{name: "forged name", mutate: func(_ *swarmManifest, results *[]swarmWorkerExecution) {
			(*results)[0].Name = "forged-worker"
		}},
		{name: "nested role mismatch", mutate: func(_ *swarmManifest, results *[]swarmWorkerExecution) {
			(*results)[0].Response.Role = "forged-response-role"
		}},
		{name: "blank status", mutate: setExternalSwarmResultStatus("")},
		{name: "planned status", mutate: setExternalSwarmResultStatus("planned")},
		{name: "spawned status", mutate: setExternalSwarmResultStatus("spawned")},
		{name: "running status", mutate: setExternalSwarmResultStatus("running")},
		{name: "unknown status", mutate: setExternalSwarmResultStatus("mysterious")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			saveGlobals(t)
			dataDir := setupBuildFlowTest(t)
			root := filepath.Dir(filepath.Dir(dataDir))
			withWorkingDir(t, root)
			goal := "Reject untrusted external swarm evidence"
			createTestColonyState(t, dataDir, colony.ColonyState{
				Version: "3.0",
				Goal:    &goal,
				State:   colony.StateREADY,
			})

			manifest := issuedSwarmManifestForTest(t, root, "Auth panic when session is missing")
			results := validExternalSwarmResults(manifest, "completed")
			tc.mutate(&manifest, &results)
			before := snapshotProjectDataTree(t, dataDir)
			_, err := runSwarmFinalize(root, externalSwarmCompletion{
				SwarmManifest: &manifest,
				Dispatches:    results,
			})
			if err == nil {
				t.Errorf("runSwarmFinalize accepted %s", tc.name)
			}
			after := snapshotProjectDataTree(t, dataDir)
			if !reflect.DeepEqual(before, after) {
				t.Errorf("rejected %s mutated .aether/data\nbefore: %#v\nafter:  %#v", tc.name, before, after)
			}
		})
	}
}

func issuedSwarmManifestForTest(t *testing.T, root, target string) swarmManifest {
	t.Helper()
	result, err := runSwarmPlanOnly(root, target)
	if err != nil {
		t.Fatalf("issue swarm manifest: %v", err)
	}
	manifest, ok := result["swarm_manifest"].(swarmManifest)
	if !ok {
		t.Fatalf("swarm_manifest type = %T, want swarmManifest", result["swarm_manifest"])
	}
	return manifest
}

func validExternalSwarmResults(manifest swarmManifest, status string) []swarmWorkerExecution {
	results := make([]swarmWorkerExecution, 0, len(manifest.Dispatches))
	for _, plan := range manifest.Dispatches {
		response := swarmWorkerResponse{
			Role:           plan.Role,
			Status:         status,
			Summary:        plan.Role + " completed externally.",
			Recommendation: "Continue with the active colony lifecycle.",
			Verification:   []string{"go test ./..."},
		}
		if plan.Role == "tracker" {
			response.RootCause = "missing session guard"
		}
		if plan.Role == "builder" {
			response.ProposedFix = "restore the missing session guard"
			response.FilesTouched = []string{"pkg/auth/handler.go"}
			response.TestsWritten = []string{"pkg/auth/handler_test.go"}
		}
		results = append(results, swarmWorkerExecution{
			Name: plan.Name, Caste: plan.Caste, Role: plan.Role, Task: plan.Task,
			Status: status, Summary: response.Summary,
			Files:    append([]string{}, response.FilesTouched...),
			Tests:    append([]string{}, response.TestsWritten...),
			Response: response,
		})
	}
	return results
}

func setExternalSwarmResultStatus(status string) func(*swarmManifest, *[]swarmWorkerExecution) {
	return func(_ *swarmManifest, results *[]swarmWorkerExecution) {
		(*results)[0].Status = status
		(*results)[0].Response.Status = status
	}
}

func TestSwarmFinalizeRejectsTraversalIDBeforeAnyMutationPublicBoundary(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	goal := "Reject an unsafe external swarm identifier"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	})

	manifest := buildSwarmManifest(root, "Auth panic when session is missing", "plan-only", time.Now().UTC())
	results := validExternalSwarmResults(manifest, "completed")
	manifest.SwarmID = "../../../escaped-swarm"
	completion := externalSwarmCompletion{SwarmManifest: &manifest, Dispatches: results}
	data, err := json.MarshalIndent(completion, "", "  ")
	if err != nil {
		t.Fatalf("marshal traversal completion: %v", err)
	}
	completionPath := filepath.Join(t.TempDir(), "swarm-completion.json")
	if err := os.WriteFile(completionPath, append(data, '\n'), 0644); err != nil {
		t.Fatalf("write traversal completion: %v", err)
	}

	before := snapshotProjectDataTree(t, root)
	rootCmd.SetArgs([]string{"swarm-finalize", "--completion-file", completionPath})
	if err := rootCmd.Execute(); err == nil {
		t.Fatal("public swarm-finalize accepted a traversal swarm_id")
	}
	after := snapshotProjectDataTree(t, root)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("rejected traversal mutated the workspace\nbefore: %#v\nafter:  %#v", before, after)
	}
}

func TestSwarmFinalizeRequiresIssuedManifestAndProtectsReplay(t *testing.T) {
	newFixture := func(t *testing.T) (string, string) {
		t.Helper()
		saveGlobals(t)
		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))
		withWorkingDir(t, root)
		goal := "Trust one issued external swarm manifest"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version: "3.0",
			Goal:    &goal,
			State:   colony.StateREADY,
		})
		return dataDir, root
	}

	t.Run("unissued manifest is rejected without mutation", func(t *testing.T) {
		dataDir, root := newFixture(t)
		manifest := buildSwarmManifest(root, "Auth panic when session is missing", "plan-only", time.Now().UTC())
		completion := externalSwarmCompletion{SwarmManifest: &manifest, Dispatches: validExternalSwarmResults(manifest, "completed")}
		before := snapshotProjectDataTree(t, dataDir)
		if _, err := runSwarmFinalize(root, completion); err == nil {
			t.Fatal("swarm-finalize accepted a manifest with no runtime issuance")
		}
		after := snapshotProjectDataTree(t, dataDir)
		if !reflect.DeepEqual(before, after) {
			t.Fatalf("unissued manifest rejection mutated durable state\nbefore: %#v\nafter:  %#v", before, after)
		}
	})

	t.Run("altered issued manifest is rejected without mutation", func(t *testing.T) {
		dataDir, root := newFixture(t)
		manifest := issuedSwarmManifestForTest(t, root, "Auth panic when session is missing")
		manifest.Target = "a caller-rewritten target"
		completion := externalSwarmCompletion{SwarmManifest: &manifest, Dispatches: validExternalSwarmResults(manifest, "completed")}
		before := snapshotProjectDataTree(t, dataDir)
		if _, err := runSwarmFinalize(root, completion); err == nil {
			t.Fatal("swarm-finalize accepted an altered issued manifest")
		}
		after := snapshotProjectDataTree(t, dataDir)
		if !reflect.DeepEqual(before, after) {
			t.Fatalf("altered manifest rejection mutated durable state\nbefore: %#v\nafter:  %#v", before, after)
		}
	})

	t.Run("exact replay is read-only and changed completion is rejected", func(t *testing.T) {
		dataDir, root := newFixture(t)
		manifest := issuedSwarmManifestForTest(t, root, "Auth panic when session is missing")
		completion := externalSwarmCompletion{SwarmManifest: &manifest, Dispatches: validExternalSwarmResults(manifest, "completed")}
		first, err := runSwarmFinalize(root, completion)
		if err != nil {
			t.Fatalf("first issued finalization: %v", err)
		}

		beforeReplay := snapshotProjectDataTree(t, dataDir)
		replayed, err := runSwarmFinalize(root, completion)
		if err != nil {
			t.Fatalf("exact replay: %v", err)
		}
		afterReplay := snapshotProjectDataTree(t, dataDir)
		if !reflect.DeepEqual(beforeReplay, afterReplay) {
			t.Fatalf("exact replay mutated durable state\nbefore: %#v\nafter:  %#v", beforeReplay, afterReplay)
		}
		firstJSON, _ := json.Marshal(first)
		replayJSON, _ := json.Marshal(replayed)
		if !bytes.Equal(firstJSON, replayJSON) {
			t.Fatalf("exact replay returned a different result\nfirst:  %s\nreplay: %s", firstJSON, replayJSON)
		}

		changed := completion
		changed.Dispatches = append([]swarmWorkerExecution{}, completion.Dispatches...)
		changed.Dispatches[0].Summary = "caller changed the terminal packet"
		beforeChanged := snapshotProjectDataTree(t, dataDir)
		if _, err := runSwarmFinalize(root, changed); err == nil {
			t.Fatal("swarm-finalize accepted a changed replay")
		}
		afterChanged := snapshotProjectDataTree(t, dataDir)
		if !reflect.DeepEqual(beforeChanged, afterChanged) {
			t.Fatalf("changed replay rejection mutated durable state\nbefore: %#v\nafter:  %#v", beforeChanged, afterChanged)
		}
	})
}

func TestSwarmThreeStrikeRecoveryExactTargetRetryReEscalates(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	goal := "Stop retrying an architectural auth failure"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Existing stabilization work",
			Status: colony.PhaseReady,
			Tasks:  []colony.Task{},
		}}},
	})

	invoker := &swarmTestInvoker{blockedCaste: "watcher"}
	originalInvoker := newSwarmWorkerInvoker
	newSwarmWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newSwarmWorkerInvoker = originalInvoker })

	target := "Auth panic when session is missing"
	configsPerAttempt := 0
	for attempt := 1; attempt <= 3; attempt++ {
		result, err := runSwarmCompatibility(root, target, false, false)
		if err != nil {
			t.Fatalf("attempt %d: %v", attempt, err)
		}
		if got := result["status"]; got != "blocked" {
			t.Fatalf("attempt %d status = %v, want blocked", attempt, got)
		}
		if attempt == 1 {
			configsPerAttempt = len(invoker.configs)
			if configsPerAttempt == 0 {
				t.Fatal("first swarm attempt did not dispatch workers")
			}
		}
		if got, want := len(invoker.configs), attempt*configsPerAttempt; got != want {
			t.Fatalf("attempt %d dispatched %d total workers, want %d", attempt, got, want)
		}
		flags := activeSwarmEscalationFlags(store)
		wantFlags := 0
		if attempt == 3 {
			wantFlags = 1
		}
		if len(flags) != wantFlags {
			t.Fatalf("attempt %d active escalation flags = %d, want %d: %+v", attempt, len(flags), wantFlags, flags)
		}
	}

	history, err := evaluateSwarmStrikeHistory(store, target)
	if err != nil {
		t.Fatalf("evaluate third-attempt history: %v", err)
	}
	if history.StrikeCount != 3 {
		t.Fatalf("strike count after three attempts = %d, want 3", history.StrikeCount)
	}
	evidenceIDs := swarmStrikeEvidenceIDs(history.Evidence)
	flags := activeSwarmEscalationFlags(store)
	for _, id := range evidenceIDs {
		if !strings.Contains(flags[0].Description, id) {
			t.Errorf("escalation description missing evidence id %q: %s", id, flags[0].Description)
		}
	}

	beforeFourth := len(invoker.configs)
	planRefusal, err := runSwarmCompatibility(root, target, false, true)
	if err != nil {
		t.Fatalf("fourth plan-only attempt: %v", err)
	}
	if got := planRefusal["status"]; got != "architectural_concern" {
		t.Fatalf("fourth plan-only status = %v, want architectural_concern", got)
	}
	if got := len(invoker.configs); got != beforeFourth {
		t.Fatalf("fourth plan-only attempt dispatched workers: configs %d -> %d", beforeFourth, got)
	}

	refusal, err := runSwarmCompatibility(root, target, false, false)
	if err != nil {
		t.Fatalf("fourth attempt: %v", err)
	}
	if got := len(invoker.configs); got != beforeFourth {
		t.Fatalf("fourth attempt dispatched workers: configs %d -> %d", beforeFourth, got)
	}
	if got := refusal["status"]; got != "architectural_concern" {
		t.Fatalf("fourth status = %v, want architectural_concern", got)
	}
	wantNext := `aether insert-phase "Auth panic when session is missing"`
	if got := refusal["next"]; got != wantNext {
		t.Fatalf("fourth next = %q, want %q", got, wantNext)
	}
	for _, id := range evidenceIDs {
		if !strings.Contains(renderSwarmCompatibilityVisual(refusal), id) {
			t.Errorf("refusal visual missing evidence id %q", id)
		}
	}

	differentTargetResult, err := runSwarmCompatibility(root, "Database panic after migration", false, false)
	if err != nil {
		t.Fatalf("different target attempt: %v", err)
	}
	wantAfterDifferentTarget := beforeFourth + intValue(differentTargetResult["worker_count"])
	if got := len(invoker.configs); got != wantAfterDifferentTarget {
		t.Fatalf("different target did not dispatch normally: configs = %d, want %d", got, wantAfterDifferentTarget)
	}

	flagsBeforeRecovery, ok := loadFlagsFile(store)
	if !ok {
		t.Fatal("load flags before public recovery")
	}
	unrelatedFlagID := swarmEscalationFlagID(swarmTargetFingerprint("Database panic after migration"))
	flagsBeforeRecovery.Decisions = append(flagsBeforeRecovery.Decisions, colony.FlagEntry{
		ID:          unrelatedFlagID,
		Type:        "blocker",
		Description: "Unrelated target must remain active.",
		Source:      "escalation",
		CreatedAt:   time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err := store.SaveJSON("pending-decisions.json", flagsBeforeRecovery); err != nil {
		t.Fatalf("seed unrelated escalation: %v", err)
	}

	// Execute the exact public recovery command emitted by the refusal. This
	// is the real root-command seam; no fake completed swarm row is seeded.
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"insert-phase", target})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute emitted recovery command %q: %v", wantNext, err)
	}
	insertEnvelope := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	insertResult := insertEnvelope["result"].(map[string]interface{})
	if inserted, _ := insertResult["inserted"].(bool); !inserted {
		t.Fatalf("emitted recovery command did not insert a phase: %v", insertResult)
	}

	recoveredHistory, err := evaluateSwarmStrikeHistory(store, target)
	if err != nil {
		t.Fatalf("evaluate recovered history: %v", err)
	}
	if recoveredHistory.StrikeCount != 0 || recoveredHistory.LatestRecovery == nil {
		t.Fatalf("history after public recovery = %+v, want zero strikes with recovery evidence", recoveredHistory)
	}
	recovery := recoveredHistory.LatestRecovery

	// Normalization differences do not create a different scope. The very
	// next retry must resolve the old flag before it dispatches real workers.
	retryTarget := "  AUTH PANIC WHEN SESSION IS MISSING!!!  "
	beforeRetry := len(invoker.configs)
	retry, err := runSwarmCompatibility(root, retryTarget, false, false)
	if err != nil {
		t.Fatalf("exact normalized target retry: %v", err)
	}
	if got := retry["status"]; got != "blocked" {
		t.Fatalf("recovered retry status = %v, want blocked from the test watcher", got)
	}
	if got := len(invoker.configs); got != beforeRetry+configsPerAttempt {
		t.Fatalf("recovered retry dispatched %d configs, want %d", got-beforeRetry, configsPerAttempt)
	}

	flagsFile, ok := loadFlagsFile(store)
	if !ok {
		t.Fatal("load escalation flags after recovered retry")
	}
	stableID := swarmEscalationFlagID(swarmTargetFingerprint(target))
	var resolved *colony.FlagEntry
	for i := range flagsFile.Decisions {
		if flagsFile.Decisions[i].ID == stableID {
			resolved = &flagsFile.Decisions[i]
			break
		}
	}
	if resolved == nil || !resolved.Resolved {
		t.Fatalf("stable escalation flag was not resolved before retry dispatch: %+v", flagsFile.Decisions)
	}
	var unrelated *colony.FlagEntry
	for i := range flagsFile.Decisions {
		if flagsFile.Decisions[i].ID == unrelatedFlagID {
			unrelated = &flagsFile.Decisions[i]
			break
		}
	}
	if unrelated == nil || unrelated.Resolved {
		t.Fatalf("same-target reconciliation changed unrelated blocker: %+v", flagsFile.Decisions)
	}
	for _, want := range []string{recovery.SwarmID, "phase " + fmt.Sprint(recovery.InsertedPhaseID)} {
		if !strings.Contains(strings.ToLower(resolved.Resolution), strings.ToLower(want)) {
			t.Errorf("resolution %q missing recovery reference %q", resolved.Resolution, want)
		}
	}
	if flags := activeSwarmEscalationFlags(store); len(flags) != 1 || flags[0].ID != unrelatedFlagID {
		t.Fatalf("recovered retry changed active escalation scope: %+v", flags)
	}

	// The recovered retry above is strike one in a fresh epoch. Two more real
	// failures must recreate the same stable blocker, and the next attempt
	// must refuse without dispatching anything.
	for attempt := 2; attempt <= 3; attempt++ {
		result, err := runSwarmCompatibility(root, target, false, false)
		if err != nil {
			t.Fatalf("new epoch attempt %d: %v", attempt, err)
		}
		if got := result["status"]; got != "blocked" {
			t.Fatalf("new epoch attempt %d status = %v, want blocked", attempt, got)
		}
	}
	newEpochFlags := activeSwarmEscalationFlags(store)
	if len(newEpochFlags) != 2 ||
		(newEpochFlags[0].ID != stableID && newEpochFlags[1].ID != stableID) {
		t.Fatalf("fresh three-strike epoch flags = %+v, want stable target plus unrelated blocker", newEpochFlags)
	}
	beforeNewRefusal := len(invoker.configs)
	newRefusal, err := runSwarmCompatibility(root, target, false, false)
	if err != nil {
		t.Fatalf("new epoch refusal: %v", err)
	}
	if got := newRefusal["status"]; got != "architectural_concern" {
		t.Fatalf("new epoch fourth status = %v, want architectural_concern", got)
	}
	if got := len(invoker.configs); got != beforeNewRefusal {
		t.Fatalf("new epoch refusal dispatched workers: configs %d -> %d", beforeNewRefusal, got)
	}
}

func TestSwarmRecoveryPersistenceFailurePublicIsNonMutating(t *testing.T) {
	dataDir, root, target, invoker := seedSwarmRecoveryPublicEscalation(t)
	beforeState, err := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("read state before recovery failure: %v", err)
	}
	beforeDispatch := len(invoker.configs)

	originalPersist := persistCorrectiveSwarmRecovery
	persistCorrectiveSwarmRecovery = func(*storage.Store, swarmResultRecord) error {
		return errors.New("forced recovery persistence failure")
	}
	t.Cleanup(func() { persistCorrectiveSwarmRecovery = originalPersist })

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"insert-phase", target})
	if err := rootCmd.Execute(); err == nil {
		t.Fatal("recovery persistence failure returned success")
	}
	if got := len(invoker.configs); got != beforeDispatch {
		t.Fatalf("failed recovery insertion dispatched workers: configs %d -> %d", beforeDispatch, got)
	}
	afterState, err := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("read state after recovery failure: %v", err)
	}
	if !bytes.Equal(beforeState, afterState) {
		t.Fatal("failed recovery persistence still committed the corrective phase")
	}
	history, err := evaluateSwarmStrikeHistory(store, target)
	if err != nil {
		t.Fatalf("evaluate history after recovery failure: %v", err)
	}
	if history.StrikeCount != 3 || history.LatestRecovery != nil {
		t.Fatalf("failed recovery persistence changed history: %+v", history)
	}
	if _, err := runSwarmCompatibility(root, target, false, false); err != nil {
		t.Fatalf("unchanged refused retry: %v", err)
	}
}

func TestSwarmPartialEscalationRepairPublicFlow(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	goal := "Repair a partially persisted escalation through the public flow"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Existing stabilization work",
			Status: colony.PhaseReady,
			Tasks:  []colony.Task{},
		}}},
	})

	target := "Auth panic when session is missing"
	invoker := &swarmTestInvoker{blockedCaste: "watcher"}
	originalInvoker := newSwarmWorkerInvoker
	newSwarmWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newSwarmWorkerInvoker = originalInvoker })

	history, pendingPath := seedPartialSwarmEscalationFailure(
		t,
		store,
		target,
		time.Now().UTC().Add(-3*time.Minute),
	)
	if _, err := runSwarmCompatibility(root, target, false, false); err == nil {
		t.Error("public same-target command succeeded while the flag store was still failing")
	}
	if got := len(invoker.configs); got != 0 {
		t.Fatalf("failed public repair invoked %d workers, want zero", got)
	}
	if err := os.Remove(pendingPath); err != nil {
		t.Fatalf("restore pending-decision storage: %v", err)
	}

	refusal, err := runSwarmCompatibility(root, target, false, false)
	if err != nil {
		t.Fatalf("public repair retry: %v", err)
	}
	if got := refusal["status"]; got != "architectural_concern" {
		t.Fatalf("public repair status = %v, want architectural_concern", got)
	}
	wantNext := swarmInsertPhaseCommand(target)
	if got := refusal["next"]; got != wantNext {
		t.Fatalf("public repair next = %q, want %q", got, wantNext)
	}
	if got := len(invoker.configs); got != 0 {
		t.Fatalf("successful public repair invoked %d workers before recovery, want zero", got)
	}
	stableID := swarmEscalationFlagID(history.TargetFingerprint)
	if flags := activeSwarmEscalationFlags(store); len(flags) != 1 || flags[0].ID != stableID {
		t.Fatalf("public repair flags = %+v, want one active stable flag %q", flags, stableID)
	}

	// Execute the exact corrective command emitted by the repaired concern.
	// It must authenticate against that stable same-target blocker and append a
	// typed recovery epoch before the target can dispatch again.
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"insert-phase", target})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute emitted recovery command %q: %v", wantNext, err)
	}
	insertEnvelope := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	insertResult := insertEnvelope["result"].(map[string]interface{})
	if inserted, _ := insertResult["inserted"].(bool); !inserted {
		t.Fatalf("emitted recovery command did not insert a phase: %v", insertResult)
	}
	recovered, err := evaluateSwarmStrikeHistory(store, target)
	if err != nil {
		t.Fatalf("evaluate repaired recovery epoch: %v", err)
	}
	if recovered.StrikeCount != 0 || recovered.LatestRecovery == nil {
		t.Fatalf("repaired recovery history = %+v, want zero strikes with recovery evidence", recovered)
	}

	retry, err := runSwarmCompatibility(root, "  AUTH PANIC WHEN SESSION IS MISSING!!!  ", false, false)
	if err != nil {
		t.Fatalf("exact-target retry after repaired recovery: %v", err)
	}
	if got := retry["status"]; got != "blocked" {
		t.Fatalf("recovered exact-target retry status = %v, want blocked test outcome", got)
	}
	if got := len(invoker.configs); got == 0 {
		t.Fatal("recovered exact-target retry did not become dispatchable")
	}
	flagsFile, ok := loadFlagsFile(store)
	if !ok {
		t.Fatal("load repaired escalation after recovery retry")
	}
	for _, flag := range flagsFile.Decisions {
		if flag.ID == stableID {
			if !flag.Resolved {
				t.Fatalf("repaired stable escalation was not resolved before dispatch: %+v", flag)
			}
			return
		}
	}
	t.Fatalf("repaired stable escalation %q disappeared after recovery retry", stableID)
}

func TestSwarmRecoveryReconciliationFailurePublicDispatchesNobody(t *testing.T) {
	dataDir, root, target, invoker := seedSwarmRecoveryPublicEscalation(t)

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"insert-phase", target})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute public recovery command: %v", err)
	}
	history, err := evaluateSwarmStrikeHistory(store, target)
	if err != nil {
		t.Fatalf("evaluate public recovery: %v", err)
	}
	if history.LatestRecovery == nil {
		t.Fatal("public recovery command did not create verified recovery evidence")
	}

	// Corrupt only the blocker persistence seam after the recovery is valid.
	// History evaluation remains read-only and succeeds; reconciliation must
	// fail before either the direct or plan-only lane creates work.
	if err := os.WriteFile(filepath.Join(dataDir, "pending-decisions.json"), []byte("{"), 0644); err != nil {
		t.Fatalf("corrupt pending decisions fixture: %v", err)
	}
	beforeDispatch := len(invoker.configs)
	if _, err := runSwarmCompatibility(root, target, false, false); err == nil {
		t.Fatal("recovery retry succeeded despite blocker reconciliation failure")
	}
	if got := len(invoker.configs); got != beforeDispatch {
		t.Fatalf("reconciliation failure dispatched workers: configs %d -> %d", beforeDispatch, got)
	}
	if _, err := runSwarmCompatibility(root, target, false, true); err == nil {
		t.Fatal("plan-only recovery retry succeeded despite blocker reconciliation failure")
	}
	if got := len(invoker.configs); got != beforeDispatch {
		t.Fatalf("plan-only reconciliation failure dispatched workers: configs %d -> %d", beforeDispatch, got)
	}
}

func TestSwarmRecoveryArbitraryInsertPublicDoesNotAuthorize(t *testing.T) {
	tests := []struct {
		name string
		args func(target string) []string
	}{
		{
			name: "different positional target",
			args: func(string) []string {
				return []string{"insert-phase", "Database panic after migration"}
			},
		},
		{
			name: "explicit description without emitted positional target",
			args: func(target string) []string {
				return []string{
					"phase-insert",
					"--after", "1",
					"--name", "General corrective work",
					"--description", target,
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, target, _ := seedSwarmRecoveryPublicEscalation(t)
			stdout = &bytes.Buffer{}
			stderr = &bytes.Buffer{}
			rootCmd.SetArgs(tc.args(target))
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("ordinary phase insertion failed: %v", err)
			}
			env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
			result := env["result"].(map[string]interface{})
			if _, recovered := result["swarm_recovery_event"]; recovered {
				t.Fatalf("ordinary insertion created swarm recovery evidence: %v", result)
			}
			history, err := evaluateSwarmStrikeHistory(store, target)
			if err != nil {
				t.Fatalf("evaluate escalated target after ordinary insertion: %v", err)
			}
			if history.StrikeCount != 3 || history.LatestRecovery != nil {
				t.Fatalf("ordinary insertion changed escalated history: %+v", history)
			}
		})
	}
}

func seedSwarmRecoveryPublicEscalation(t *testing.T) (dataDir, root, target string, invoker *swarmTestInvoker) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir = setupBuildFlowTest(t)
	root = filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	goal := "Recover a repeatedly failing swarm through the public command"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Existing work",
			Status: colony.PhaseReady,
			Tasks:  []colony.Task{},
		}}},
	})

	target = "Auth panic when session is missing"
	invoker = &swarmTestInvoker{blockedCaste: "watcher"}
	originalInvoker := newSwarmWorkerInvoker
	newSwarmWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newSwarmWorkerInvoker = originalInvoker })

	for attempt := 1; attempt <= 3; attempt++ {
		result, err := runSwarmCompatibility(root, target, false, false)
		if err != nil {
			t.Fatalf("seed attempt %d: %v", attempt, err)
		}
		if got := result["status"]; got != "blocked" {
			t.Fatalf("seed attempt %d status = %v, want blocked", attempt, got)
		}
	}
	if flags := activeSwarmEscalationFlags(store); len(flags) != 1 {
		t.Fatalf("seed escalation flags = %+v, want one", flags)
	}
	return dataDir, root, target, invoker
}

func TestSwarmFourthAttemptShellQuotesTarget(t *testing.T) {
	target := "Fix $SESSION and `refresh` for \"admin\""
	command := swarmInsertPhaseCommand(target)
	want := "aether insert-phase \"Fix \\$SESSION and \\`refresh\\` for \\\"admin\\\"\""
	if command != want {
		t.Fatalf("insert command = %q, want %q", command, want)
	}
	quotedTarget := strings.TrimPrefix(command, "aether insert-phase ")
	out, err := exec.Command("/bin/sh", "-c", `set -- `+quotedTarget+`; printf '%s\n%s' "$#" "$1"`).CombinedOutput()
	if err != nil {
		t.Fatalf("shell parse guided command: %v: %s", err, out)
	}
	if got, want := string(out), "1\n"+target; got != want {
		t.Fatalf("guided command did not preserve one literal argument: got %q want %q", got, want)
	}
}

func TestSwarmThreeStrikeExternalFinalizeReplayKeepsOneEscalation(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	goal := "Finalize an externally dispatched failing swarm"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	})

	target := "Auth panic when session is missing"
	base := time.Now().UTC().Add(-2 * time.Minute)
	for i := 0; i < 2; i++ {
		if _, err := persistSwarmResultOutcome(store, swarmResultRecord{
			SwarmID:     []string{"swarm-prior-1", "swarm-prior-2"}[i],
			Target:      target,
			Status:      "failed",
			CompletedAt: base.Add(time.Duration(i) * time.Minute).Format(time.RFC3339Nano),
		}); err != nil {
			t.Fatalf("seed prior external strike %d: %v", i+1, err)
		}
	}

	manifest := issuedSwarmManifestForTest(t, root, target)
	dispatches := make([]swarmWorkerExecution, 0, len(manifest.Dispatches))
	for _, plan := range manifest.Dispatches {
		status := "completed"
		blockers := []string(nil)
		if plan.Role == "watcher" {
			status = "blocked"
			blockers = []string{"verification fixture unavailable"}
		}
		dispatches = append(dispatches, swarmWorkerExecution{
			Name:     plan.Name,
			Caste:    plan.Caste,
			Role:     plan.Role,
			Task:     plan.Task,
			Status:   status,
			Summary:  plan.Role + " completed externally",
			Blockers: blockers,
			Response: swarmWorkerResponse{
				Role:    plan.Role,
				Status:  status,
				Summary: plan.Role + " completed externally",
			},
		})
	}
	completion := externalSwarmCompletion{SwarmManifest: &manifest, Dispatches: dispatches}
	for replay := 1; replay <= 2; replay++ {
		result, err := runSwarmFinalize(root, completion)
		if err != nil {
			t.Fatalf("external finalize replay %d: %v", replay, err)
		}
		if got := result["status"]; got != "blocked" {
			t.Fatalf("external finalize replay %d status = %v, want blocked", replay, got)
		}
		if flags := activeSwarmEscalationFlags(store); len(flags) != 1 {
			t.Fatalf("external finalize replay %d flags = %d, want 1: %+v", replay, len(flags), flags)
		}
	}
}

func swarmPlansHaveCaste(plans []swarmWorkerPlan, caste string) bool {
	for _, plan := range plans {
		if plan.Caste == caste {
			return true
		}
	}
	return false
}

func workerMapsHaveCaste(workers []interface{}, caste string) bool {
	for _, worker := range workers {
		entry, ok := worker.(map[string]interface{})
		if ok && entry["caste"] == caste {
			return true
		}
	}
	return false
}
