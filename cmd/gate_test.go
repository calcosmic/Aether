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
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

func TestGateCheck_TaskComplete_AllPass(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	// Create a minimal COLONY_STATE.json
	state := map[string]interface{}{
		"version": "3.0",
		"goal":    "test gate-check",
		"state":   "READY",
		"errors": map[string]interface{}{
			"records":          []interface{}{},
			"flagged_patterns": []interface{}{},
		},
	}
	stateData, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(dir, "COLONY_STATE.json"), stateData, 0644)

	// Run gate-check for task-complete
	result := runGateCheck("task-complete", "1.1", 0)

	if !result.Allowed {
		t.Errorf("expected allowed=true, got false: %s", result.Reason)
	}
	for _, c := range result.Checks {
		if c.Name == "tests_pass" && !c.Passed {
			t.Logf("tests_pass check: %s (expected in temp dir without tests)", c.Detail)
		}
		if c.Name == "no_critical_flags" && !c.Passed {
			t.Errorf("no_critical_flags should pass with empty errors: %s", c.Detail)
		}
	}
}

func TestGateCheck_PhaseAdvance_PendingTasks(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	// Create state with one incomplete task
	taskID := "1.1"
	state := map[string]interface{}{
		"version": "3.0",
		"goal":    "test gate-check",
		"state":   "READY",
		"plan": map[string]interface{}{
			"phases": []interface{}{
				map[string]interface{}{
					"id":     1,
					"name":   "Test Phase",
					"status": "in_progress",
					"tasks": []interface{}{
						map[string]interface{}{
							"id":     taskID,
							"goal":   "Do something",
							"status": "code_written",
						},
					},
				},
			},
		},
		"errors": map[string]interface{}{
			"records":          []interface{}{},
			"flagged_patterns": []interface{}{},
		},
	}
	stateData, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(dir, "COLONY_STATE.json"), stateData, 0644)

	result := runGateCheck("phase-advance", "", 1)

	if result.Allowed {
		t.Error("expected allowed=false when tasks are not completed")
	}

	// Find the all_tasks_completed check
	found := false
	for _, c := range result.Checks {
		if c.Name == "all_tasks_completed" {
			found = true
			if c.Passed {
				t.Error("all_tasks_completed should fail with pending tasks")
			}
		}
	}
	if !found {
		t.Error("missing all_tasks_completed check")
	}
}

func TestGateCheck_NoCriticalFlags(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	// State with a critical error record
	state := map[string]interface{}{
		"version": "3.0",
		"goal":    "test gate-check",
		"state":   "READY",
		"errors": map[string]interface{}{
			"records": []interface{}{
				map[string]interface{}{
					"id":          "err-1",
					"category":    "build",
					"severity":    "CRITICAL",
					"description": "Build failed",
				},
			},
			"flagged_patterns": []interface{}{},
		},
	}
	stateData, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(dir, "COLONY_STATE.json"), stateData, 0644)

	check := checkNoCriticalFlags()
	if check.Passed {
		t.Error("expected no_critical_flags to fail with CRITICAL error record")
	}
}

func TestEnforceGuard_Blocked(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	// State with critical error — guard should block
	state := map[string]interface{}{
		"version": "3.0",
		"goal":    "test guard",
		"state":   "READY",
		"errors": map[string]interface{}{
			"records": []interface{}{
				map[string]interface{}{
					"id":          "err-1",
					"category":    "test",
					"severity":    "CRITICAL",
					"description": "Test failure",
				},
			},
			"flagged_patterns": []interface{}{},
		},
	}
	stateData, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(dir, "COLONY_STATE.json"), stateData, 0644)

	err = enforceGuard("task-complete:1.1")
	if err == nil {
		t.Error("expected guard to block with critical errors")
	}
}

func TestEnforceGuard_InvalidFormat(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	err = enforceGuard("invalid-format")
	if err == nil {
		t.Error("expected error for invalid guard format")
	}
}

func TestResolveTestCommand_GoProject(t *testing.T) {
	// Save and clear AETHER_ROOT so ResolveAetherRoot uses git to find repo root
	origRoot := os.Getenv("AETHER_ROOT")
	os.Unsetenv("AETHER_ROOT")
	defer os.Setenv("AETHER_ROOT", origRoot)

	// Since this test runs inside the Aether repo (which has go.mod),
	// it should detect Go and return the test command.
	cmd := resolveTestCommand()
	if cmd != "go test ./..." {
		t.Errorf("expected 'go test ./...', got %q", cmd)
	}
}

func TestResolveTestCommand_NoProject(t *testing.T) {
	// Save and clear AETHER_ROOT, then set to empty temp dir
	origRoot := os.Getenv("AETHER_ROOT")
	os.Unsetenv("AETHER_ROOT")
	defer os.Setenv("AETHER_ROOT", origRoot)

	// resolveTestCommand uses ResolveAetherRoot which finds the git repo root.
	// Just verify it doesn't panic.
	_ = resolveTestCommand()
}

// --- Gate Integration Tests (Phase 22) ---

func TestPreBuildGates(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)

	goal := "Gate test"
	taskID := "task-gate"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Gate test", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: &taskID, Goal: "Gate task", Status: colony.TaskPending}}},
			},
		},
	})

	// Fresh state with no critical flags: should pass
	if err := runPreBuildGates(dataDir, 1); err != nil {
		t.Errorf("pre-build gates should pass with no critical flags: %v", err)
	}

	// Add a critical error record: should fail
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	state.Errors.Records = append(state.Errors.Records, colony.ErrorRecord{
		ID:        "1",
		Severity:  "CRITICAL",
		Category:  "test",
		Description:   "critical error",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}
	if err := runPreBuildGates(dataDir, 1); err == nil {
		t.Error("pre-build gates should fail with critical flags")
	}
}

func TestPreContinueGates(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)

	goal := "Gate test"
	taskID := "task-gate"
	now := time.Now().UTC()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateBUILT,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Gate test", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskID, Goal: "Gate task", Status: colony.TaskCompleted}}},
			},
		},
		BuildStartedAt: &now,
	})

	// No critical flags: should pass
	if err := runPreContinueGates(dataDir, 1); err != nil {
		t.Errorf("pre-continue gates should pass with no critical flags: %v", err)
	}

	// Add a critical error record: should fail
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	state.Errors.Records = append(state.Errors.Records, colony.ErrorRecord{
		ID:        "1",
		Severity:  "CRITICAL",
		Category:  "test",
		Description:   "critical error",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}
	if err := runPreContinueGates(dataDir, 1); err == nil {
		t.Error("pre-continue gates should fail with critical flags")
	}
}

// --- Gate Recovery Template Tests (Phase 59, Plan 01) ---

func TestGateRecoveryTemplates_HasAllGateNames(t *testing.T) {
	expectedGates := []string{
		"verification_loop", "spawn_gate", "anti_pattern", "complexity",
		"gatekeeper", "auditor", "tdd_evidence", "runtime",
		"flags", "watcher_veto", "medic", "tests_pass",
	}
	for _, name := range expectedGates {
		if _, ok := gateRecoveryTemplates[name]; !ok {
			t.Errorf("gateRecoveryTemplates missing entry for %q", name)
		}
	}
}

func TestGateRecoveryTemplate_KnownGate(t *testing.T) {
	result := gateRecoveryTemplate("spawn_gate")
	if !strings.Contains(result, "ant-build") {
		t.Errorf("spawn_gate template should contain 'ant-build', got: %s", result)
	}
	if !strings.Contains(result, "ant-continue") {
		t.Errorf("spawn_gate template should contain 'ant-continue', got: %s", result)
	}
}

func TestGateRecoveryTemplate_UnknownGate(t *testing.T) {
	result := gateRecoveryTemplate("nonexistent_gate")
	if !strings.Contains(result, "No specific recovery instructions") {
		t.Errorf("unknown gate should return fallback message, got: %s", result)
	}
}

func TestShouldSkipGate_PassedGateSkipped(t *testing.T) {
	prior := []colony.GateResultEntry{
		{Name: "spawn_gate", Passed: true, Timestamp: time.Now().UTC().Format(time.RFC3339)},
	}
	result := shouldSkipGate(prior, "spawn_gate")
	if !result {
		t.Error("should skip spawn_gate when it previously passed")
	}
}

func TestShouldSkipGate_TestsNeverSkipped(t *testing.T) {
	prior := []colony.GateResultEntry{
		{Name: "tests_pass", Passed: true, Timestamp: time.Now().UTC().Format(time.RFC3339)},
	}
	result := shouldSkipGate(prior, "tests_pass")
	if result {
		t.Error("tests_pass should never be skipped, even when previously passed")
	}
}

func TestShouldSkipGate_FailedGateNotSkipped(t *testing.T) {
	prior := []colony.GateResultEntry{
		{Name: "spawn_gate", Passed: false, Timestamp: time.Now().UTC().Format(time.RFC3339)},
	}
	result := shouldSkipGate(prior, "spawn_gate")
	if result {
		t.Error("should not skip spawn_gate when it previously failed")
	}
}

func TestShouldSkipGate_NoPriorResults(t *testing.T) {
	result := shouldSkipGate(nil, "spawn_gate")
	if result {
		t.Error("should not skip any gate when no prior results exist")
	}
}

func TestGateResultsWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	// Create a minimal COLONY_STATE.json
	state := map[string]interface{}{
		"version": "3.0",
		"goal":    "test gate results",
		"state":   "READY",
	}
	stateData, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(dir, "COLONY_STATE.json"), stateData, 0644)

	entries := []colony.GateResultEntry{
		{Name: "spawn_gate", Passed: true, Timestamp: time.Now().UTC().Format(time.RFC3339)},
		{Name: "tests_pass", Passed: false, Timestamp: time.Now().UTC().Format(time.RFC3339), Detail: "2 tests failed"},
	}

	if err := gateResultsWrite(entries); err != nil {
		t.Fatalf("gateResultsWrite failed: %v", err)
	}

	readBack := gateResultsRead()
	if len(readBack) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(readBack))
	}
	if readBack[0].Name != "spawn_gate" || !readBack[0].Passed {
		t.Errorf("first entry mismatch: %+v", readBack[0])
	}
	if readBack[1].Name != "tests_pass" || readBack[1].Passed {
		t.Errorf("second entry mismatch: %+v", readBack[1])
	}
	if readBack[1].Detail != "2 tests failed" {
		t.Errorf("detail mismatch: got %q", readBack[1].Detail)
	}
}

func TestGateResultsRead_NoFile(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	result := gateResultsRead()
	if result != nil {
		t.Errorf("expected nil when no state file, got %v", result)
	}
}

func TestFormatSkipSummary_MixedResults(t *testing.T) {
	prior := []colony.GateResultEntry{
		{Name: "spawn_gate", Passed: true, Timestamp: time.Now().UTC().Format(time.RFC3339)},
		{Name: "anti_pattern", Passed: true, Timestamp: time.Now().UTC().Format(time.RFC3339)},
		{Name: "tests_pass", Passed: false, Timestamp: time.Now().UTC().Format(time.RFC3339), Detail: "failed"},
		{Name: "gatekeeper", Passed: false, Timestamp: time.Now().UTC().Format(time.RFC3339), Detail: "CVE found"},
		{Name: "auditor", Passed: false, Timestamp: time.Now().UTC().Format(time.RFC3339), Detail: "low score"},
	}
	summary := formatSkipSummary(prior)
	if !strings.Contains(summary, "Skipping 2 passed gates") {
		t.Errorf("summary should mention 2 passed gates, got: %s", summary)
	}
	if !strings.Contains(summary, "re-checking 3 failures") {
		t.Errorf("summary should mention 3 failures, got: %s", summary)
	}
}

func TestFormatSkipSummary_NoPriorResults(t *testing.T) {
	summary := formatSkipSummary(nil)
	if summary != "" {
		t.Errorf("expected empty string for nil results, got: %s", summary)
	}
}

// --- CLI Subcommand Tests (Phase 59, Plan 01, Task 3) ---

// gateCmdTestSetup prepares a test environment for gate CLI subcommands.
// It disables rootCmd's PersistentPreRunE (which would overwrite the test store)
// and restores it on cleanup.
func gateCmdTestSetup(t *testing.T) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	origPreRun := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error { return nil }
	t.Cleanup(func() {
		rootCmd.PersistentPreRunE = origPreRun
	})
}

func TestGateResultsReadCmd_EmptyState(t *testing.T) {
	gateCmdTestSetup(t)
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	var buf bytes.Buffer
	rootCmd.SetArgs([]string{"gate-results-read"})
	stdout = &buf
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "[]" {
		t.Errorf("expected '[]', got %q", output)
	}
}

func TestGateResultsWriteCmd_WithNamePassed(t *testing.T) {
	gateCmdTestSetup(t)
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	// Create minimal state
	stateData, _ := json.Marshal(map[string]interface{}{
		"version": "3.0",
		"state":   "READY",
	})
	os.WriteFile(filepath.Join(dir, "COLONY_STATE.json"), stateData, 0644)

	var buf bytes.Buffer
	rootCmd.SetArgs([]string{"gate-results-write", "--name", "spawn_gate", "--passed"})
	stdout = &buf
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true in output, got %q", output)
	}

	// Verify entry was persisted
	var readState colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &readState); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if len(readState.GateResults) != 1 {
		t.Fatalf("expected 1 gate result, got %d", len(readState.GateResults))
	}
	if readState.GateResults[0].Name != "spawn_gate" || !readState.GateResults[0].Passed {
		t.Errorf("unexpected gate result: %+v", readState.GateResults[0])
	}
}

func TestGateResultsWriteCmd_WithDetail(t *testing.T) {
	gateCmdTestSetup(t)
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	stateData, _ := json.Marshal(map[string]interface{}{
		"version": "3.0",
		"state":   "READY",
	})
	os.WriteFile(filepath.Join(dir, "COLONY_STATE.json"), stateData, 0644)

	var buf bytes.Buffer
	rootCmd.SetArgs([]string{"gate-results-write", "--name", "spawn_gate", "--passed=false", "--detail", "missing files"})
	stdout = &buf
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	// Verify detail preserved
	var readState colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &readState); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if readState.GateResults[0].Detail != "missing files" {
		t.Errorf("expected detail 'missing files', got %q", readState.GateResults[0].Detail)
	}
}

func TestGateResultsWriteCmd_MissingName(t *testing.T) {
	gateCmdTestSetup(t)
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	var buf bytes.Buffer
	rootCmd.SetArgs([]string{"gate-results-write", "--passed"})
	stdout = &buf
	stderr = &buf
	// Should output error about --name being required, not crash
	_ = rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "--name is required") {
		t.Errorf("expected --name required error, got %q", output)
	}
}

func TestShouldSkipGateCmd_PassedGate(t *testing.T) {
	gateCmdTestSetup(t)
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	// Create state with passed spawn_gate
	stateData, _ := json.Marshal(colony.ColonyState{
		Version: "3.0",
		State:   colony.StateREADY,
		GateResults: []colony.GateResultEntry{
			{Name: "spawn_gate", Passed: true, Timestamp: time.Now().UTC().Format(time.RFC3339)},
		},
	})
	os.WriteFile(filepath.Join(dir, "COLONY_STATE.json"), stateData, 0644)

	var buf bytes.Buffer
	rootCmd.SetArgs([]string{"should-skip-gate", "--name", "spawn_gate"})
	stdout = &buf
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "true" {
		t.Errorf("expected 'true', got %q", output)
	}
}

func TestShouldSkipGateCmd_TestsNeverSkipped(t *testing.T) {
	gateCmdTestSetup(t)
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	// Create state with tests_pass passed
	stateData, _ := json.Marshal(colony.ColonyState{
		Version: "3.0",
		State:   colony.StateREADY,
		GateResults: []colony.GateResultEntry{
			{Name: "tests_pass", Passed: true, Timestamp: time.Now().UTC().Format(time.RFC3339)},
		},
	})
	os.WriteFile(filepath.Join(dir, "COLONY_STATE.json"), stateData, 0644)

	var buf bytes.Buffer
	rootCmd.SetArgs([]string{"should-skip-gate", "--name", "tests_pass"})
	stdout = &buf
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "false" {
		t.Errorf("expected 'false' (tests never skipped), got %q", output)
	}
}

func TestGateRecoveryTemplateCmd_KnownGate(t *testing.T) {
	gateCmdTestSetup(t)

	var buf bytes.Buffer
	rootCmd.SetArgs([]string{"gate-recovery-template", "--name", "spawn_gate"})
	stdout = &buf
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "ant-build") {
		t.Errorf("expected recovery template containing 'ant-build', got %q", output)
	}
}

func TestGateRecoveryTemplateCmd_UnknownGate(t *testing.T) {
	gateCmdTestSetup(t)

	var buf bytes.Buffer
	rootCmd.SetArgs([]string{"gate-recovery-template", "--name", "nonexistent"})
	stdout = &buf
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if !strings.Contains(output, "No specific recovery instructions") {
		t.Errorf("expected fallback message, got %q", output)
	}
}
