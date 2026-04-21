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
)

// setupBuildFlowTest creates a temp directory with store initialized for testing.
func setupBuildFlowTest(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create temp data dir: %v", err)
	}
	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	t.Cleanup(func() {
		if origRoot == "" {
			os.Unsetenv("AETHER_ROOT")
			return
		}
		os.Setenv("AETHER_ROOT", origRoot)
	})

	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	t.Cleanup(func() {
		stdout = os.Stdout
		stderr = os.Stderr
	})

	return dataDir
}

// createTestColonyState creates a minimal COLONY_STATE.json for testing.
func createTestColonyState(t *testing.T, dataDir string, state colony.ColonyState) {
	t.Helper()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "COLONY_STATE.json"), data, 0644); err != nil {
		t.Fatalf("failed to write state: %v", err)
	}
}

// TestVersionCheckCachedFirstRun tests version check without cache.
func TestVersionCheckCachedFirstRun(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	os.Setenv("AETHER_ROOT", tmpDir)
	t.Cleanup(func() { os.Unsetenv("AETHER_ROOT") })

	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	t.Cleanup(func() {
		stdout = os.Stdout
		stderr = os.Stderr
	})

	rootCmd.SetArgs([]string{"version-check-cached"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("version-check-cached returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"cached":false`) {
		t.Errorf("expected cached:false on first run, got: %s", output)
	}

	// Verify cache file was created
	if _, err := os.Stat(filepath.Join(dataDir, ".version-check-cache")); os.IsNotExist(err) {
		t.Error("version cache file was not created")
	}
}

// TestVersionCheckCachedFromCache tests version check reads from cache.
func TestVersionCheckCachedFromCache(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	os.Setenv("AETHER_ROOT", tmpDir)
	t.Cleanup(func() { os.Unsetenv("AETHER_ROOT") })

	// Write a recent cache entry
	cache := versionCacheEntry{
		Version:  "test-version",
		CachedAt: time.Now().UTC().Format(time.RFC3339),
	}
	cacheData, _ := json.Marshal(cache)
	os.WriteFile(filepath.Join(dataDir, ".version-check-cache"), cacheData, 0644)

	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	t.Cleanup(func() {
		stdout = os.Stdout
		stderr = os.Stderr
	})

	rootCmd.SetArgs([]string{"version-check-cached"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("version-check-cached returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"cached":true`) {
		t.Errorf("expected cached:true from cache, got: %s", output)
	}
	if !strings.Contains(output, "test-version") {
		t.Errorf("expected cached version 'test-version', got: %s", output)
	}
}

// TestMilestoneDetect tests milestone detection from colony state.
func TestMilestoneDetect(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)

	goal := "test goal"
	name := "test-colony"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:    "2.0",
		Goal:       &goal,
		ColonyName: &name,
		State:      colony.StateREADY,
		Milestone:  "Open Chambers",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: "completed"},
				{ID: 2, Name: "Phase 2", Status: "in_progress"},
			},
		},
	})

	rootCmd.SetArgs([]string{"milestone-detect"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("milestone-detect returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, "Open Chambers") {
		t.Errorf("expected milestone 'Open Chambers', got: %s", output)
	}
}

// TestMilestoneDetectAutoDetect tests automatic milestone detection.
func TestMilestoneDetectAutoDetect(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)

	goal := "test goal"
	name := "test-colony"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:    "2.0",
		Goal:       &goal,
		ColonyName: &name,
		State:      colony.StateREADY,
		Milestone:  "", // No milestone set
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: "completed"},
				{ID: 2, Name: "Phase 2", Status: "completed"},
				{ID: 3, Name: "Phase 3", Status: "completed"},
				{ID: 4, Name: "Phase 4", Status: "in_progress"},
			},
		},
	})

	rootCmd.SetArgs([]string{"milestone-detect"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("milestone-detect returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	// 3/4 complete = 75% = "Ventilated Nest"
	if !strings.Contains(output, "Ventilated Nest") {
		t.Errorf("expected auto-detected milestone, got: %s", output)
	}
}

// TestUpdateProgress tests updating a phase status.
func TestUpdateProgress(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)

	goal := "test goal"
	name := "test-colony"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "2.0",
		Goal:         &goal,
		ColonyName:   &name,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: "in_progress"},
				{ID: 2, Name: "Phase 2", Status: "pending"},
			},
		},
	})

	rootCmd.SetArgs([]string{"update-progress", "--phase", "1", "--status", "completed"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("update-progress returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"updated":true`) {
		t.Errorf("expected updated:true, got: %s", output)
	}

	// Verify state was persisted
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}
	if state.Plan.Phases[0].Status != "completed" {
		t.Errorf("expected phase 1 status 'completed', got %q", state.Plan.Phases[0].Status)
	}
}

// TestUpdateProgressInvalidStatus tests that invalid status is rejected.
func TestUpdateProgressInvalidStatus(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)

	goal := "test goal"
	name := "test-colony"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:    "2.0",
		Goal:       &goal,
		ColonyName: &name,
		State:      colony.StateEXECUTING,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: "in_progress"},
			},
		},
	})

	rootCmd.SetArgs([]string{"update-progress", "--phase", "1", "--status", "invalid"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("update-progress returned error: %v", err)
	}

	errOutput := stderr.(*bytes.Buffer).String()
	if !strings.Contains(errOutput, "invalid status") {
		t.Errorf("expected invalid status error, got: %s", errOutput)
	}
}

// TestUpdateProgressWithTask tests updating a specific task within a phase.
func TestUpdateProgressWithTask(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)

	goal := "test goal"
	name := "test-colony"
	taskID := "task-1-1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:    "2.0",
		Goal:       &goal,
		ColonyName: &name,
		State:      colony.StateEXECUTING,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Phase 1",
					Status: "in_progress",
					Tasks: []colony.Task{
						{ID: &taskID, Goal: "Do thing", Status: "pending"},
					},
				},
			},
		},
	})

	rootCmd.SetArgs([]string{"update-progress", "--phase", "1", "--status", "completed", "--task", "task-1-1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("update-progress returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}

	// Verify task was updated
	var state colony.ColonyState
	store.LoadJSON("COLONY_STATE.json", &state)
	if state.Plan.Phases[0].Tasks[0].Status != "completed" {
		t.Errorf("expected task status 'completed', got %q", state.Plan.Phases[0].Tasks[0].Status)
	}
}

// TestPrintNextUpExecuting tests print-next-up during EXECUTING state.
func TestPrintNextUpExecuting(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)

	goal := "test goal"
	name := "test-colony"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "2.0",
		Goal:         &goal,
		ColonyName:   &name,
		State:        colony.StateEXECUTING,
		CurrentPhase: 2,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: "completed"},
				{ID: 2, Name: "Phase 2", Status: "in_progress"},
			},
		},
	})

	rootCmd.SetArgs([]string{"print-next-up"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("print-next-up returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, "aether continue") {
		t.Errorf("expected aether continue suggestion, got: %s", output)
	}
}

// TestPrintNextUpCompleted tests print-next-up for completed colony.
func TestPrintNextUpCompleted(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)

	goal := "test goal"
	name := "test-colony"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:    "2.0",
		Goal:       &goal,
		ColonyName: &name,
		State:      colony.StateCOMPLETED,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: "completed"},
			},
		},
	})

	rootCmd.SetArgs([]string{"print-next-up"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("print-next-up returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, `aether seal`) {
		t.Errorf("expected aether seal suggestion, got: %s", output)
	}
}

// TestDataSafetyStats tests data safety stats reporting.
func TestDataSafetyStats(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	os.Setenv("AETHER_ROOT", tmpDir)
	t.Cleanup(func() { os.Unsetenv("AETHER_ROOT") })

	// Create test files
	os.WriteFile(filepath.Join(dataDir, "COLONY_STATE.json"), []byte(`{"version":"2.0"}`), 0644)
	os.WriteFile(filepath.Join(dataDir, "pheromones.json"), []byte(`{"signals":[{"id":"1"},{"id":"2"}]}`), 0644)
	os.WriteFile(filepath.Join(dataDir, "session.json"), []byte(`{"colony_goal":"test goal"}`), 0644)

	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	t.Cleanup(func() {
		stdout = os.Stdout
		stderr = os.Stderr
	})

	rootCmd.SetArgs([]string{"data-safety-stats"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("data-safety-stats returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"state_exists":true`) {
		t.Errorf("expected state_exists:true, got: %s", output)
	}
	if !strings.Contains(output, `"pheromones_count":2`) {
		t.Errorf("expected pheromones_count:2, got: %s", output)
	}
	if !strings.Contains(output, `"session_active":true`) {
		t.Errorf("expected session_active:true, got: %s", output)
	}
}

// TestDataSafetyStatsNoDataDir tests data safety stats with empty data dir.
func TestDataSafetyStatsNoDataDir(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	tmpDir := t.TempDir()
	os.Setenv("AETHER_ROOT", tmpDir)
	t.Cleanup(func() { os.Unsetenv("AETHER_ROOT") })

	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	t.Cleanup(func() {
		stdout = os.Stdout
		stderr = os.Stderr
	})

	rootCmd.SetArgs([]string{"data-safety-stats"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("data-safety-stats returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	// PersistentPreRunE auto-creates the data dir, so check for empty state instead
	if !strings.Contains(output, `"state_exists":false`) {
		t.Errorf("expected state_exists:false for empty dir, got: %s", output)
	}
	if !strings.Contains(output, `"pheromones_count":0`) {
		t.Errorf("expected pheromones_count:0 for empty dir, got: %s", output)
	}
}

// TestBuildFlowCommandsRegistered verifies all build flow commands are registered.
func TestBuildFlowCommandsRegistered(t *testing.T) {
	expectedCommands := []string{
		"version-check-cached",
		"milestone-detect",
		"update-progress",
		"print-next-up",
		"data-safety-stats",
	}

	for _, name := range expectedCommands {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("command %q not registered in rootCmd", name)
		}
	}
}
