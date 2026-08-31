package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
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
	if got := result["next"]; got != "aether build 1" {
		t.Fatalf("next = %v, want aether build 1", got)
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

func TestSwarmPlanOnlyPrintsManifestWithoutMutatingState(t *testing.T) {
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
	workers := result["workers"].([]interface{})
	if len(workers) != 4 {
		t.Fatalf("workers = %d, want 4", len(workers))
	}
	if !workerMapsHaveCaste(workers, "gatekeeper") {
		t.Fatalf("workers missing Queen-selected gatekeeper: %+v", workers)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "swarms")); !os.IsNotExist(err) {
		t.Fatalf("plan-only should not create swarm artifacts, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "spawn-tree.txt")); !os.IsNotExist(err) {
		t.Fatalf("plan-only should not write spawn-tree, stat err=%v", err)
	}
}

func TestSwarmFinalizeRecordsExternalTaskResults(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Finalize visible swarm workers"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	})

	planResult, err := runSwarmPlanOnly(root, "Auth panic when session is missing")
	if err != nil {
		t.Fatalf("runSwarmPlanOnly: %v", err)
	}
	manifest := planResult["swarm_manifest"].(swarmManifest)
	dispatches := make([]swarmWorkerExecution, 0, len(manifest.Dispatches))
	for _, plan := range manifest.Dispatches {
		response := swarmWorkerResponse{
			Role:           plan.Role,
			Status:         "completed",
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
		dispatches = append(dispatches, swarmWorkerExecution{
			Name:     plan.Name,
			Caste:    plan.Caste,
			Role:     plan.Role,
			Task:     plan.Task,
			Status:   "completed",
			Summary:  response.Summary,
			Files:    append([]string{}, response.FilesTouched...),
			Tests:    append([]string{}, response.TestsWritten...),
			Response: response,
		})
	}

	completionPath := filepath.Join(t.TempDir(), "swarm-completion.json")
	data, err := json.Marshal(map[string]interface{}{
		"swarm_manifest": manifest,
		"dispatches":     dispatches,
	})
	if err != nil {
		t.Fatalf("marshal completion: %v", err)
	}
	if err := os.WriteFile(completionPath, data, 0644); err != nil {
		t.Fatalf("write completion: %v", err)
	}

	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"swarm-finalize", "--completion-file", completionPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm-finalize returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := result["dispatch_mode"]; got != "external-task" {
		t.Fatalf("dispatch_mode = %v, want external-task", got)
	}
	if got := result["status"]; got != "completed" {
		t.Fatalf("status = %v, want completed", got)
	}
	resultPath := filepath.Join(dataDir, "swarms", manifest.SwarmID, "result.json")
	if _, err := os.Stat(resultPath); err != nil {
		t.Fatalf("expected swarm result artifact: %v", err)
	}
	spawnTreeData, err := os.ReadFile(filepath.Join(dataDir, "spawn-tree.txt"))
	if err != nil {
		t.Fatalf("read spawn-tree: %v", err)
	}
	for _, caste := range []string{"tracker", "builder", "watcher"} {
		if !strings.Contains(string(spawnTreeData), "|Swarm|"+caste+"|") {
			t.Fatalf("spawn tree missing %s entry:\n%s", caste, string(spawnTreeData))
		}
	}
}

func TestSwarmThreeStrikeEscalatesOnceAndFourthAttemptRefuses(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	goal := "Stop retrying an architectural auth failure"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
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

	if err := saveSwarmResultRecord(store, swarmResultRecord{
		SwarmID:     "swarm-reset-success",
		Target:      target,
		Status:      "completed",
		CompletedAt: time.Now().UTC().Add(time.Second).Format(time.RFC3339Nano),
	}); err != nil {
		t.Fatalf("seed reset result: %v", err)
	}
	beforeResetAttempt := len(invoker.configs)
	if _, err := runSwarmCompatibility(root, target, false, false); err != nil {
		t.Fatalf("post-success attempt: %v", err)
	}
	if got := len(invoker.configs); got != beforeResetAttempt+configsPerAttempt {
		t.Fatalf("completed result did not reset dispatch guard: configs = %d, want %d", got, beforeResetAttempt+configsPerAttempt)
	}
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

	manifest := buildSwarmManifest(root, target, "plan-only", time.Now().UTC())
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
