package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
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
