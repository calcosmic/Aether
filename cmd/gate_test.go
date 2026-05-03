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
	prior := []GateCheckResult{
		{Name: "spawn_gate", Status: "passed", Timestamp: time.Now().UTC().Format(time.RFC3339)},
	}
	result := shouldSkipGate(prior, "spawn_gate")
	if !result {
		t.Error("should skip spawn_gate when it previously passed")
	}
}

func TestShouldSkipGate_TestsNeverSkipped(t *testing.T) {
	prior := []GateCheckResult{
		{Name: "tests_pass", Status: "passed", Timestamp: time.Now().UTC().Format(time.RFC3339)},
	}
	result := shouldSkipGate(prior, "tests_pass")
	if result {
		t.Error("tests_pass should never be skipped, even when previously passed")
	}
}

func TestShouldSkipGate_FailedGateNotSkipped(t *testing.T) {
	prior := []GateCheckResult{
		{Name: "spawn_gate", Status: "failed", Timestamp: time.Now().UTC().Format(time.RFC3339)},
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
	// Build lookup by name (merge logic uses map, order not guaranteed)
	byName := make(map[string]colony.GateResultEntry, len(readBack))
	for _, e := range readBack {
		byName[e.Name] = e
	}
	if e, ok := byName["spawn_gate"]; !ok || !e.Passed {
		t.Errorf("spawn_gate entry missing or not passed: %+v", e)
	}
	if e, ok := byName["tests_pass"]; !ok || e.Passed {
		t.Errorf("tests_pass entry missing or passed: %+v", e)
	}
	if e, ok := byName["tests_pass"]; !ok || e.Detail != "2 tests failed" {
		t.Errorf("detail mismatch: got %q", e.Detail)
	}
}

// TestGateResultsWrite_MergesEntries verifies that sequential gateResultsWrite
// calls accumulate entries instead of replacing them (CR-01 fix).
func TestGateResultsWrite_MergesEntries(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	// Create a minimal COLONY_STATE.json
	state := map[string]interface{}{
		"version": "3.0",
		"goal":    "test merge",
		"state":   "READY",
	}
	stateData, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(dir, "COLONY_STATE.json"), stateData, 0644)

	ts := time.Now().UTC().Format(time.RFC3339)

	// Write first entry
	if err := gateResultsWrite([]colony.GateResultEntry{
		{Name: "spawn_gate", Passed: true, Timestamp: ts},
	}); err != nil {
		t.Fatalf("first write failed: %v", err)
	}

	// Write second entry (different gate name)
	if err := gateResultsWrite([]colony.GateResultEntry{
		{Name: "tests_pass", Passed: false, Timestamp: ts, Detail: "1 test failed"},
	}); err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	readBack := gateResultsRead()
	if len(readBack) != 2 {
		t.Fatalf("expected 2 entries after sequential writes, got %d", len(readBack))
	}

	var foundSpawn, foundTests bool
	for _, r := range readBack {
		if r.Name == "spawn_gate" && r.Passed {
			foundSpawn = true
		}
		if r.Name == "tests_pass" && !r.Passed && r.Detail == "1 test failed" {
			foundTests = true
		}
	}
	if !foundSpawn {
		t.Error("spawn_gate entry not found or incorrect")
	}
	if !foundTests {
		t.Error("tests_pass entry not found or incorrect")
	}
}

// TestGateResultsWrite_UpsertsExistingEntry verifies that writing a gate result
// with the same name updates (upserts) the existing entry instead of duplicating.
func TestGateResultsWrite_UpsertsExistingEntry(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	state := map[string]interface{}{
		"version": "3.0",
		"goal":    "test upsert",
		"state":   "READY",
	}
	stateData, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(dir, "COLONY_STATE.json"), stateData, 0644)

	ts := time.Now().UTC().Format(time.RFC3339)

	// Write gate A as passed
	if err := gateResultsWrite([]colony.GateResultEntry{
		{Name: "tests_pass", Passed: true, Timestamp: ts},
	}); err != nil {
		t.Fatalf("first write failed: %v", err)
	}

	// Write gate A again as failed with detail (upsert)
	if err := gateResultsWrite([]colony.GateResultEntry{
		{Name: "tests_pass", Passed: false, Timestamp: ts, Detail: "3 tests failed"},
	}); err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	readBack := gateResultsRead()
	if len(readBack) != 1 {
		t.Fatalf("expected 1 entry after upsert, got %d", len(readBack))
	}
	if readBack[0].Name != "tests_pass" {
		t.Errorf("expected name tests_pass, got %s", readBack[0].Name)
	}
	if readBack[0].Passed {
		t.Error("expected Passed=false after upsert")
	}
	if readBack[0].Detail != "3 tests failed" {
		t.Errorf("expected detail '3 tests failed', got %q", readBack[0].Detail)
	}
}

// TestGateResultsWrite_MergesMultipleEntriesAtOnce verifies that batch writes
// merge correctly with existing entries.
func TestGateResultsWrite_MergesMultipleEntriesAtOnce(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	state := map[string]interface{}{
		"version": "3.0",
		"goal":    "test batch merge",
		"state":   "READY",
	}
	stateData, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(dir, "COLONY_STATE.json"), stateData, 0644)

	ts := time.Now().UTC().Format(time.RFC3339)

	// Write batch 1: A and B
	if err := gateResultsWrite([]colony.GateResultEntry{
		{Name: "gate_a", Passed: true, Timestamp: ts},
		{Name: "gate_b", Passed: true, Timestamp: ts},
	}); err != nil {
		t.Fatalf("batch 1 failed: %v", err)
	}

	// Write batch 2: B (updated) and C (new)
	if err := gateResultsWrite([]colony.GateResultEntry{
		{Name: "gate_b", Passed: false, Timestamp: ts, Detail: "now failing"},
		{Name: "gate_c", Passed: true, Timestamp: ts},
	}); err != nil {
		t.Fatalf("batch 2 failed: %v", err)
	}

	readBack := gateResultsRead()
	if len(readBack) != 3 {
		t.Fatalf("expected 3 entries after batch merge, got %d", len(readBack))
	}

	names := make(map[string]colony.GateResultEntry)
	for _, r := range readBack {
		names[r.Name] = r
	}

	if names["gate_a"].Passed != true {
		t.Error("gate_a should still be passed")
	}
	if names["gate_b"].Passed != false || names["gate_b"].Detail != "now failing" {
		t.Error("gate_b should be updated to failed with detail")
	}
	if names["gate_c"].Passed != true {
		t.Error("gate_c should be passed")
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
		rootCmd.SetArgs([]string{"should-skip-gate", "--name", "spawn_gate", "--phase", "1"})
	stdout = &buf

		// Write per-phase gate results file
		phaseResults := []GateCheckResult{
			{Name: "spawn_gate", Status: "passed", Timestamp: time.Now().UTC().Format(time.RFC3339)},
		}
		phaseData, _ := json.Marshal(phaseResults)
		os.WriteFile(filepath.Join(dir, "gate-results-1.json"), phaseData, 0644)

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

// TestIncrementalGateChecking_SkipsPriorPassed verifies the full incremental
// gate checking flow: passed non-test gates are skipped, failed gates are
// re-checked, and tests_pass is never skipped regardless of prior results.
func TestIncrementalGateChecking_SkipsPriorPassed(t *testing.T) {
	ts := time.Now().UTC().Format(time.RFC3339)
	prior := []GateCheckResult{
		{Name: "spawn_gate", Status: "passed", Timestamp: ts},
		{Name: "state_gate", Status: "passed", Timestamp: ts},
		{Name: "build_gate", Status: "failed", Timestamp: ts},
		{Name: "tests_pass", Status: "passed", Timestamp: ts},
	}

	// Passed non-test gates should be skipped
	if !shouldSkipGate(prior, "spawn_gate") {
		t.Error("expected spawn_gate (passed) to be skipped")
	}
	if !shouldSkipGate(prior, "state_gate") {
		t.Error("expected state_gate (passed) to be skipped")
	}

	// Failed gate should NOT be skipped (must re-check)
	if shouldSkipGate(prior, "build_gate") {
		t.Error("expected build_gate (failed) to NOT be skipped")
	}

	// tests_pass is NEVER skipped even when previously passed
	if shouldSkipGate(prior, "tests_pass") {
		t.Error("expected tests_pass to never be skipped even when previously passed")
	}

	// Unknown gate (not in prior results) should NOT be skipped
	if shouldSkipGate(prior, "unknown_gate") {
		t.Error("expected unknown_gate to NOT be skipped")
	}
}

// --- Gate Classification Tests (Phase 93, Plan 01) ---

func TestGateClassifications_CoversAllNamedGates(t *testing.T) {
	for name := range gateRecoveryTemplates {
		tier, rationale := gateClassify(name)
		if tier == "" {
			t.Errorf("gateClassifications missing entry for %q", name)
		}
		if rationale == "" {
			t.Errorf("gateClassifications has empty rationale for %q", name)
		}
	}
}

func TestGateClassifications_CoversAllAlwaysRunGates(t *testing.T) {
	for name := range alwaysRunGates {
		tier, _ := gateClassify(name)
		if tier == "" {
			t.Errorf("gateClassifications missing entry for always-run gate %q", name)
		}
	}
}

func TestGateClassifications_HardBlockImmutability(t *testing.T) {
	hardBlockGates := []string{"gatekeeper", "watcher_veto", "flags", "tests_pass", "no_critical_flags"}
	for _, name := range hardBlockGates {
		tier, _ := gateClassify(name)
		if tier != hardBlock {
			t.Errorf("expected %q to be hard_block, got %q", name, tier)
		}
	}
}

func TestGateClassify_UnknownGate(t *testing.T) {
	tier, rationale := gateClassify("nonexistent_gate")
	if tier != "" {
		t.Errorf("expected empty tier for unknown gate, got %q", tier)
	}
	if rationale != "" {
		t.Errorf("expected empty rationale for unknown gate, got %q", rationale)
	}
}

func TestIsHardBlockGate_HardGates(t *testing.T) {
	hardGates := []string{"gatekeeper", "watcher_veto", "flags", "tests_pass", "no_critical_flags"}
	for _, name := range hardGates {
		if !isHardBlockGate(name) {
			t.Errorf("expected isHardBlockGate(%q) to be true", name)
		}
	}
}

func TestIsHardBlockGate_SoftGates(t *testing.T) {
	softGates := []string{"auditor", "complexity", "tdd_evidence"}
	for _, name := range softGates {
		if isHardBlockGate(name) {
			t.Errorf("expected isHardBlockGate(%q) to be false", name)
		}
	}
}

func TestQueenAnnotation_JSONRoundtrip(t *testing.T) {
	original := GateCheckResult{
		Name:      "auditor",
		Status:    "failed",
		Detail:    "quality score below 60",
		Timestamp: "2026-05-03T12:00:00Z",
		QueenAnnotation: &QueenAnnotation{
			Decision:     "auto-resolved",
			Rationale:    "score 58 is within tolerance",
			Timestamp:    "2026-05-03T12:00:00Z",
			QueenVersion: "1.0.27",
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded GateCheckResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.QueenAnnotation == nil {
		t.Fatal("expected QueenAnnotation to be non-nil after roundtrip")
	}
	if decoded.QueenAnnotation.Decision != "auto-resolved" {
		t.Errorf("expected Decision 'auto-resolved', got %q", decoded.QueenAnnotation.Decision)
	}
	if decoded.QueenAnnotation.Rationale != "score 58 is within tolerance" {
		t.Errorf("expected Rationale 'score 58 is within tolerance', got %q", decoded.QueenAnnotation.Rationale)
	}
	if decoded.QueenAnnotation.Timestamp != "2026-05-03T12:00:00Z" {
		t.Errorf("expected Timestamp '2026-05-03T12:00:00Z', got %q", decoded.QueenAnnotation.Timestamp)
	}
	if decoded.QueenAnnotation.QueenVersion != "1.0.27" {
		t.Errorf("expected QueenVersion '1.0.27', got %q", decoded.QueenAnnotation.QueenVersion)
	}
	if decoded.Detail != "quality score below 60" {
		t.Errorf("expected Detail preserved, got %q", decoded.Detail)
	}
}

func TestGateCheckResult_BackwardCompatible_NoAnnotation(t *testing.T) {
	oldJSON := `{"name":"tests_pass","status":"failed","detail":"2 tests failed","timestamp":"2026-05-01T00:00:00Z","retry_count":0}`

	var result GateCheckResult
	if err := json.Unmarshal([]byte(oldJSON), &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result.QueenAnnotation != nil {
		t.Error("expected QueenAnnotation to be nil for old JSON without queen_annotation field")
	}
	if result.Detail != "2 tests failed" {
		t.Errorf("expected Detail '2 tests failed', got %q", result.Detail)
	}
}

func TestGateClassifyCmd_JSONOutput(t *testing.T) {
	gateCmdTestSetup(t)

	var buf bytes.Buffer
	rootCmd.SetArgs([]string{"gate-classify", "--json"})
	stdout = &buf
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"gatekeeper"`) {
		t.Errorf("expected JSON output to contain 'gatekeeper', got: %s", output)
	}
	if !strings.Contains(output, `"auditor"`) {
		t.Errorf("expected JSON output to contain 'auditor', got: %s", output)
	}
}

func TestGateClassifyCmd_TableOutput(t *testing.T) {
	gateCmdTestSetup(t)

	var buf bytes.Buffer
	rootCmd.SetArgs([]string{"gate-classify"})
	stdout = &buf
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "gatekeeper") {
		t.Errorf("expected table output to contain 'gatekeeper', got: %s", output)
	}
	if !strings.Contains(output, "hard_block") {
		t.Errorf("expected table output to contain 'hard_block', got: %s", output)
	}
	if !strings.Contains(output, "soft_block") {
		t.Errorf("expected table output to contain 'soft_block', got: %s", output)
	}
}

// --- Auto-Resolve Tests (Phase 95, Plan 01) ---

func TestAutoResolveSoftBlock(t *testing.T) {
	report := codexContinueGateReport{
		Phase: 1,
		Checks: []gateCheck{
			{Name: "auditor", Passed: false, Detail: "quality score below threshold"},
		},
		Passed:         false,
		BlockingIssues: []string{"quality score below threshold"},
	}

	updated, resolved := autoResolveSoftBlockGates(1, report, "standard")

	if len(resolved) != 1 || resolved[0] != "auditor" {
		t.Errorf("expected auditor in resolved list, got %v", resolved)
	}
	if !updated.Checks[0].Passed {
		t.Error("expected auditor check to be flipped to Passed=true")
	}
	if !updated.Passed {
		t.Error("expected overall report.Passed=true after resolving all soft_block gates")
	}
}

func TestAutoResolveHardBlockNever(t *testing.T) {
	report := codexContinueGateReport{
		Phase: 1,
		Checks: []gateCheck{
			{Name: "gatekeeper", Passed: false, Detail: "CVE found"},
		},
		Passed:         false,
		BlockingIssues: []string{"CVE found"},
	}

	updated, resolved := autoResolveSoftBlockGates(1, report, "standard")

	if len(resolved) != 0 {
		t.Errorf("expected empty resolved list for hard_block gate, got %v", resolved)
	}
	if updated.Checks[0].Passed {
		t.Error("hard_block gate should remain failed")
	}
	if updated.Passed {
		t.Error("report should remain failed with hard_block gate")
	}
}

func TestAutoResolveAdvisoryIgnored(t *testing.T) {
	report := codexContinueGateReport{
		Phase: 1,
		Checks: []gateCheck{
			{Name: "medic", Passed: false, Detail: "health issue"},
		},
		Passed:         false,
		BlockingIssues: []string{"health issue"},
	}

	updated, resolved := autoResolveSoftBlockGates(1, report, "standard")

	if len(resolved) != 0 {
		t.Errorf("expected empty resolved list for advisory gate, got %v", resolved)
	}
	if updated.Checks[0].Passed {
		t.Error("advisory gate should remain as-is (not auto-resolved)")
	}
}

func TestAutoResolveDepthMultiplier(t *testing.T) {
	if m := autoResolveDepthMultiplier(colony.VerificationDepthLight); m != 1.5 {
		t.Errorf("expected light multiplier 1.5, got %f", m)
	}
	if m := autoResolveDepthMultiplier(colony.VerificationDepthStandard); m != 1.0 {
		t.Errorf("expected standard multiplier 1.0, got %f", m)
	}
	if m := autoResolveDepthMultiplier(colony.VerificationDepthHeavy); m != 0.0 {
		t.Errorf("expected heavy multiplier 0.0, got %f", m)
	}
}

func TestAutoResolveHeavySkipsAll(t *testing.T) {
	report := codexContinueGateReport{
		Phase: 1,
		Checks: []gateCheck{
			{Name: "auditor", Passed: false, Detail: "quality score below threshold"},
			{Name: "complexity", Passed: false, Detail: "too complex"},
		},
		Passed:         false,
		BlockingIssues: []string{"quality score below threshold", "too complex"},
	}

	updated, resolved := autoResolveSoftBlockGates(1, report, "heavy")

	if len(resolved) != 0 {
		t.Errorf("expected no auto-resolved gates at heavy depth, got %v", resolved)
	}
	if updated.Checks[0].Passed {
		t.Error("auditor should remain failed at heavy depth")
	}
	if updated.Checks[1].Passed {
		t.Error("complexity should remain failed at heavy depth")
	}
}

func TestAnnotateGateResultPreservesOriginal(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	// Seed gate results with original data
	original := []GateCheckResult{
		{
			Name:            "auditor",
			Status:          "failed",
			Detail:          "original detail",
			FixHint:         "fix it",
			RecoveryOptions: []string{"retry"},
			Timestamp:       time.Now().UTC().Format(time.RFC3339),
		},
	}
	if err := gateResultsWritePhase(1, original); err != nil {
		t.Fatalf("failed to write gate results: %v", err)
	}

	// Annotate
	annotation := QueenAnnotation{
		Decision:     "auto-resolved",
		Rationale:    "finding below threshold",
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		QueenVersion: "1.0.27",
	}
	if err := annotateGateResult(1, "auditor", annotation); err != nil {
		t.Fatalf("annotateGateResult failed: %v", err)
	}

	// Read back and verify
	results, err := gateResultsReadPhase(1)
	if err != nil {
		t.Fatalf("failed to read gate results: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]

	// Original fields must be preserved
	if r.Detail != "original detail" {
		t.Errorf("Detail modified: got %q, want 'original detail'", r.Detail)
	}
	if r.FixHint != "fix it" {
		t.Errorf("FixHint modified: got %q, want 'fix it'", r.FixHint)
	}
	if len(r.RecoveryOptions) != 1 || r.RecoveryOptions[0] != "retry" {
		t.Errorf("RecoveryOptions modified: got %v", r.RecoveryOptions)
	}

	// Annotation must be set
	if r.QueenAnnotation == nil {
		t.Fatal("expected QueenAnnotation to be set")
	}
	if r.QueenAnnotation.Decision != "auto-resolved" {
		t.Errorf("expected Decision 'auto-resolved', got %q", r.QueenAnnotation.Decision)
	}
}

func TestAutoResolveMixedResults(t *testing.T) {
	report := codexContinueGateReport{
		Phase: 1,
		Checks: []gateCheck{
			{Name: "auditor", Passed: false, Detail: "quality issue"},
			{Name: "complexity", Passed: false, Detail: "too complex"},
			{Name: "gatekeeper", Passed: false, Detail: "CVE found"},
		},
		Passed:         false,
		BlockingIssues: []string{"quality issue", "too complex", "CVE found"},
	}

	updated, resolved := autoResolveSoftBlockGates(1, report, "standard")

	// Two soft_block gates should be resolved
	if len(resolved) != 2 {
		t.Errorf("expected 2 resolved gates, got %d: %v", len(resolved), resolved)
	}
	resolvedSet := make(map[string]bool)
	for _, r := range resolved {
		resolvedSet[r] = true
	}
	if !resolvedSet["auditor"] || !resolvedSet["complexity"] {
		t.Errorf("expected auditor and complexity resolved, got %v", resolved)
	}

	// Hard_block gate should remain failed
	if updated.Checks[2].Passed {
		t.Error("gatekeeper (hard_block) should remain failed")
	}

	// Overall should be failed because hard_block still blocking
	if updated.Passed {
		t.Error("report.Passed should be false because gatekeeper still blocks")
	}
}

func TestAutoResolveUnclassifiedGate(t *testing.T) {
	report := codexContinueGateReport{
		Phase: 1,
		Checks: []gateCheck{
			{Name: "unknown_structural_gate", Passed: false, Detail: "something failed"},
		},
		Passed:         false,
		BlockingIssues: []string{"something failed"},
	}

	updated, resolved := autoResolveSoftBlockGates(1, report, "standard")

	if len(resolved) != 0 {
		t.Errorf("expected empty resolved for unclassified gate, got %v", resolved)
	}
	if updated.Checks[0].Passed {
		t.Error("unclassified gate should remain failed (fail-open for safety)")
	}
}

func TestAutoResolveEmptyReport(t *testing.T) {
	report := codexContinueGateReport{
		Phase:          1,
		Checks:         []gateCheck{},
		Passed:         true,
		BlockingIssues: nil,
	}

	updated, resolved := autoResolveSoftBlockGates(1, report, "standard")

	if len(resolved) != 0 {
		t.Errorf("expected empty resolved for all-passed report, got %v", resolved)
	}
	if !updated.Passed {
		t.Error("report should remain passed when all gates are already passing")
	}
}

func TestGateAutoResolveCmdJSON(t *testing.T) {
	gateCmdTestSetup(t)

	var buf bytes.Buffer
	rootCmd.SetArgs([]string{"gate-auto-resolve", "--json"})
	stdout = &buf
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := buf.String()

	// Parse JSON output (wrapped in {"ok":true,"result":...})
	var wrapper struct {
		OK     bool                   `json:"ok"`
		Result map[string]interface{} `json:"result"`
	}
	if err := json.Unmarshal([]byte(output), &wrapper); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}
	if !wrapper.OK {
		t.Fatalf("expected ok=true, got output: %s", output)
	}

	// All 6 soft_block gates should be present
	expectedGates := []string{"auditor", "complexity", "tdd_evidence", "anti_pattern", "verification_loop", "spawn_gate"}
	for _, gate := range expectedGates {
		if _, ok := wrapper.Result[gate]; !ok {
			t.Errorf("expected gate %q in JSON output, not found", gate)
		}
	}
}

func TestGateAutoResolveCmdTable(t *testing.T) {
	gateCmdTestSetup(t)

	var buf bytes.Buffer
	rootCmd.SetArgs([]string{"gate-auto-resolve"})
	stdout = &buf
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := buf.String()

	// Table should contain gate names
	if !strings.Contains(output, "auditor") {
		t.Errorf("expected table output to contain 'auditor', got: %s", output)
	}
	if !strings.Contains(output, "complexity") {
		t.Errorf("expected table output to contain 'complexity', got: %s", output)
	}
}
