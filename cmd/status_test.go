package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
)

func TestGenerateProgressBar(t *testing.T) {
	tests := []struct {
		current int
		total   int
		width   int
		want    string
	}{
		{0, 0, 20, "[░░░░░░░░░░░░░░░░░░░░]"},
		{5, 10, 20, "[██████████░░░░░░░░░░]"},
		{10, 10, 20, "[████████████████████]"},
		{0, 10, 20, "[░░░░░░░░░░░░░░░░░░░░]"},
		{3, 10, 20, "[██████░░░░░░░░░░░░░░]"},
		{7, 10, 20, "[██████████████░░░░░░]"},
		{15, 10, 20, "[████████████████████]"}, // current > total caps
		{1, 4, 20, "[█████░░░░░░░░░░░░░░░]"},
	}

	for _, tt := range tests {
		got := generateProgressBar(tt.current, tt.total, tt.width)
		if got != tt.want {
			t.Errorf("generateProgressBar(%d, %d, %d) = %q, want %q", tt.current, tt.total, tt.width, got, tt.want)
		}
	}
}

func TestStatusNoColony(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	// Create temp dir with .aether/data/ but no COLONY_STATE.json
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	rootCmd.SetArgs([]string{"status"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No colony initialized") {
		t.Errorf("expected 'No colony initialized', got: %q", output)
	}
}

func TestStatusOutput(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	// Set AETHER_ROOT so PersistentPreRunE initializes store
	origRoot := os.Getenv("AETHER_ROOT")
	// We need to point to the parent dir, not .aether/data directly
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	// Override store since setupTestStore already created it
	store = s

	rootCmd.SetArgs([]string{"status"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	output := buf.String()

	// Check essential sections exist
	checks := []string{
		"C O L O N Y   S T A T U S",
		"Goal:",
		"Progress",
		"Phase:",
		"Tasks:",
		"Instincts:",
		"Flags:",
		"State:",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("output missing %q\ngot:\n%s", check, output)
		}
	}

	// Check progress bar format
	if !strings.Contains(output, "[Phase 2/3]") {
		t.Errorf("expected '[Phase 2/3]' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "[Tasks 2/4]") {
		t.Errorf("expected '[Tasks 2/4]' in output, got:\n%s", output)
	}

	// Check instinct counts
	if !strings.Contains(output, "2 learned") {
		t.Errorf("expected '2 learned' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "1 strong") {
		t.Errorf("expected '1 strong' in output, got:\n%s", output)
	}
}

func TestStatusPheromoneSummary(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	store = s

	rootCmd.SetArgs([]string{"status"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	output := buf.String()

	// Check pheromone table structure
	if !strings.Contains(output, "Active Pheromones") {
		t.Errorf("output missing 'Active Pheromones'\ngot:\n%s", output)
	}
	if !strings.Contains(output, "FOCUS") {
		t.Errorf("output missing FOCUS row")
	}
	if !strings.Contains(output, "REDIRECT") {
		t.Errorf("output missing REDIRECT row")
	}
	if !strings.Contains(output, "FEEDBACK") {
		t.Errorf("output missing FEEDBACK row")
	}
}

func TestStatusUsesStandaloneInstincts(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to load colony state: %v", err)
	}
	state.Memory.Instincts = []colony.Instinct{}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("failed to save colony state: %v", err)
	}

	instincts := colony.InstinctsFile{
		Version: "1.0",
		Instincts: []colony.InstinctEntry{
			{
				ID:         "inst_live_1",
				Trigger:    "trigger one",
				Action:     "action one",
				Domain:     "go",
				Confidence: 0.9,
				TrustScore: 0.9,
				TrustTier:  "canonical",
				Provenance: colony.InstinctProvenance{Source: "obs_1", CreatedAt: "2026-04-01T00:00:00Z"},
			},
			{
				ID:         "inst_live_2",
				Trigger:    "trigger two",
				Action:     "action two",
				Domain:     "testing",
				Confidence: 0.6,
				TrustScore: 0.6,
				TrustTier:  "emerging",
				Provenance: colony.InstinctProvenance{Source: "obs_2", CreatedAt: "2026-04-02T00:00:00Z"},
			},
		},
	}
	if err := s.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatalf("failed to save instincts: %v", err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	store = s
	rootCmd.SetArgs([]string{"status"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "2 learned") {
		t.Errorf("expected standalone instincts count in output, got:\n%s", output)
	}
	if !strings.Contains(output, "1 strong") {
		t.Errorf("expected strong instinct count from instincts.json, got:\n%s", output)
	}
}

func TestStatusOutput_ParallelModeDefault(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	store = s

	rootCmd.SetArgs([]string{"status"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	output := buf.String()

	// When parallel_mode is not set in state, should default to "in-repo"
	if !strings.Contains(output, "Parallel:") {
		t.Errorf("status output missing 'Parallel:' line\ngot:\n%s", output)
	}
	if !strings.Contains(output, "in-repo") {
		t.Errorf("status output missing default 'in-repo' parallel mode\ngot:\n%s", output)
	}
}

func TestStatusOutput_ParallelModeWorktree(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	// Load the existing colony state and set parallel_mode to worktree
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to load colony state: %v", err)
	}
	state.ParallelMode = colony.ModeWorktree
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("failed to save colony state: %v", err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	store = s

	rootCmd.SetArgs([]string{"status"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Parallel:") {
		t.Errorf("status output missing 'Parallel:' line\ngot:\n%s", output)
	}
	if !strings.Contains(output, "worktree") {
		t.Errorf("status output missing 'worktree' parallel mode\ngot:\n%s", output)
	}
}

func TestStatusMemoryHealth(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	store = s

	rootCmd.SetArgs([]string{"status"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Memory Health") {
		t.Errorf("output missing 'Memory Health'\ngot:\n%s", output)
	}
	if !strings.Contains(output, "Wisdom Entries") {
		t.Errorf("output missing 'Wisdom Entries' row")
	}
	if !strings.Contains(output, "Recent Failures") {
		t.Errorf("output missing 'Recent Failures' row")
	}
}

func TestStatusShowsActiveWorkersFromSpawnTree(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	spawnTree := agent.NewSpawnTree(s, "spawn-tree.txt")
	if err := spawnTree.RecordSpawn("Queen", "surveyor", "Atlas-55", "Map provisions and external trails", 1); err != nil {
		t.Fatalf("failed to record spawn: %v", err)
	}
	if err := spawnTree.RecordSpawn("Queen", "surveyor", "Map-91", "Map architecture and chamber layout", 1); err != nil {
		t.Fatalf("failed to record spawn: %v", err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	store = s
	rootCmd.SetArgs([]string{"status"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	output := buf.String()
	for _, want := range []string{"Active Workers", "Atlas-55", "Map-91", "2 active workers", "in-flight command"} {
		if !strings.Contains(output, want) {
			t.Errorf("status output missing %q\n%s", want, output)
		}
	}
}

func TestStatusPausedColonyIgnoresStaleSpawnTreeWorkers(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to load colony state: %v", err)
	}
	state.State = colony.State("PAUSED")
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("failed to save colony state: %v", err)
	}

	spawnTree := agent.NewSpawnTree(s, "spawn-tree.txt")
	if err := spawnTree.RecordSpawn("Queen", "builder", "Ghost-41", "Old worker from previous session", 1); err != nil {
		t.Fatalf("failed to record ghost spawn: %v", err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	store = s
	rootCmd.SetArgs([]string{"status"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	output := buf.String()
	for _, unwanted := range []string{"Active Workers", "Ghost-41", "active workers", "in-flight command"} {
		if strings.Contains(output, unwanted) {
			t.Fatalf("status should not show stale workers for paused colony; found %q in output:\n%s", unwanted, output)
		}
	}
}

func TestStatusCompletedColonyShowsFullTaskProgress(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	var buf bytes.Buffer
	stdout = &buf

	goal := "Show completed final phase cleanly"
	taskID := "task-1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateCOMPLETED,
		CurrentPhase: 1,
		Milestone:    "Crowned Anthill",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Finish line",
					Status: colony.PhaseCompleted,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Wrap up remaining task markers", Status: colony.TaskPending}},
				},
			},
		},
	})

	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[Tasks 1/1]") {
		t.Fatalf("expected completed colony to show full task progress, got:\n%s", output)
	}
}
