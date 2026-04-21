package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

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

func TestStatusNoColonyVisual(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

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

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	output := buf.String()
	for _, want := range []string{"📊", "C O L O N Y   S T A T U S", "No colony initialized in this repo.", "aether init", "aether lay-eggs"} {
		if !strings.Contains(output, want) {
			t.Errorf("visual no-colony status missing %q\n%s", want, output)
		}
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

func TestStatusOutput_DefaultScopeDisplay(t *testing.T) {
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

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Scope: project") {
		t.Fatalf("expected default project scope in output, got:\n%s", output)
	}
}

func TestStatusOutput_MetaScopeDisplay(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	state.Scope = colony.ScopeMeta
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
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
	if !strings.Contains(output, "Scope: meta") {
		t.Fatalf("expected meta scope in output, got:\n%s", output)
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

func TestStatusShowsInlinePheromoneStrength(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	now := time.Now().UTC()
	redirectStrength := 0.91
	focusStrength := 0.83
	redirectExpiresAt := now.Add(48 * time.Hour).Format(time.RFC3339)
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "sig_redirect", Type: "REDIRECT", Priority: "high", Source: "test", CreatedAt: now.Format(time.RFC3339), ExpiresAt: &redirectExpiresAt, Active: true, Strength: &redirectStrength, Content: []byte(`{"text":"Avoid global state"}`)},
			{ID: "sig_focus", Type: "FOCUS", Priority: "normal", Source: "test", CreatedAt: now.Format(time.RFC3339), Active: true, Strength: &focusStrength, Content: []byte(`{"text":"Focus on lifecycle output"}`)},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("failed to save pheromones: %v", err)
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
	for _, want := range []string{"Active Pheromones", "Strength", "Life", "Avoid global state", "Focus on lifecycle output", "0.91", "0.83", "phase-scoped", "ttl"} {
		if !strings.Contains(output, want) {
			t.Errorf("status output missing %q\n%s", want, output)
		}
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

func TestStatusShowsRecentInstinctsWithConfidence(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	instincts := colony.InstinctsFile{
		Version: "1.0",
		Instincts: []colony.InstinctEntry{
			{
				ID:         "inst_old",
				Trigger:    "old trigger",
				Action:     "old action",
				Domain:     "go",
				Confidence: 0.61,
				TrustScore: 0.61,
				TrustTier:  "emerging",
				Provenance: colony.InstinctProvenance{Source: "obs_old", CreatedAt: "2026-04-19T10:00:00Z"},
			},
			{
				ID:         "inst_mid",
				Trigger:    "mid trigger",
				Action:     "mid action",
				Domain:     "testing",
				Confidence: 0.74,
				TrustScore: 0.74,
				TrustTier:  "trusted",
				Provenance: colony.InstinctProvenance{Source: "obs_mid", CreatedAt: "2026-04-20T10:00:00Z"},
			},
			{
				ID:         "inst_new",
				Trigger:    "new trigger",
				Action:     "new action",
				Domain:     "runtime",
				Confidence: 0.92,
				TrustScore: 0.92,
				TrustTier:  "canonical",
				Provenance: colony.InstinctProvenance{Source: "obs_new", CreatedAt: "2026-04-21T10:00:00Z"},
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
	if !strings.Contains(output, "Recent Instincts") {
		t.Fatalf("expected Recent Instincts section, got:\n%s", output)
	}
	for _, want := range []string{"new action", "mid action", "old action", "0.92", "0.74", "0.61"} {
		if !strings.Contains(output, want) {
			t.Errorf("status output missing %q\n%s", want, output)
		}
	}
	if strings.Index(output, "new action") > strings.Index(output, "mid action") {
		t.Fatalf("expected newest instinct before older instinct, got:\n%s", output)
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

func TestStatusPrefersCurrentRunWorkersOverStaleHistory(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	var buf bytes.Buffer
	stdout = &buf

	goal := "Show only the current run"
	now := time.Now().UTC()
	taskID := "task-1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Live phase",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Keep the colony moving", Status: colony.TaskInProgress}},
				},
			},
		},
	})

	spawnData := strings.Join([]string{
		fmt.Sprintf("%s|Queen|builder|Ghost-41|Old worker|1|spawned", now.Add(-110*time.Second).Format(time.RFC3339)),
		fmt.Sprintf("%s|Ghost-41|active|Old active", now.Add(-100*time.Second).Format(time.RFC3339)),
		fmt.Sprintf("%s|Queen|builder|Hammer-1|Current worker|1|spawned", now.Add(-10*time.Second).Format(time.RFC3339)),
		fmt.Sprintf("%s|Hammer-1|active|Current active", now.Add(-5*time.Second).Format(time.RFC3339)),
	}, "\n") + "\n"
	if err := store.AtomicWrite("spawn-tree.txt", []byte(spawnData)); err != nil {
		t.Fatalf("write spawn-tree.txt: %v", err)
	}

	runState := fmt.Sprintf(`{
  "current_run_id": "run-current",
  "runs": [
    {
      "id": "run-old",
      "command": "build",
      "started_at": %q,
      "ended_at": %q,
      "status": "completed"
    },
    {
      "id": "run-current",
      "command": "build",
      "started_at": %q,
      "status": "active"
    }
  ]
}
`, now.Add(-2*time.Minute).Format(time.RFC3339), now.Add(-90*time.Second).Format(time.RFC3339), now.Add(-30*time.Second).Format(time.RFC3339))
	if err := store.AtomicWrite("spawn-runs.json", []byte(runState)); err != nil {
		t.Fatalf("write spawn-runs.json: %v", err)
	}

	rootCmd.SetArgs([]string{"status"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Hammer-1") {
		t.Fatalf("expected current run worker in status output, got:\n%s", output)
	}
	if strings.Contains(output, "Ghost-41") {
		t.Fatalf("status should not show stale worker from older run, got:\n%s", output)
	}
}

func TestStatusShowsSpawnSummaryFromSpawnTree(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	var buf bytes.Buffer
	stdout = &buf

	goal := "Show spawn summary"
	now := mustParseRFC3339(t, "2026-04-21T10:15:00Z")
	taskID := "task-1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Live phase",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Keep the colony moving", Status: colony.TaskInProgress}},
				},
			},
		},
	})

	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := spawnTree.RecordSpawn("Queen", "builder", "Hammer-1", "Keep building", 1); err != nil {
		t.Fatalf("record active spawn: %v", err)
	}
	if err := spawnTree.UpdateStatus("Hammer-1", "active", "Running"); err != nil {
		t.Fatalf("mark active: %v", err)
	}
	if err := spawnTree.RecordSpawn("Queen", "watcher", "Keen-2", "Verify the slice", 1); err != nil {
		t.Fatalf("record completed spawn: %v", err)
	}
	if err := spawnTree.UpdateStatus("Keen-2", "completed", "Verified"); err != nil {
		t.Fatalf("mark completed: %v", err)
	}
	if err := spawnTree.RecordSpawn("Queen", "scout", "Map-3", "Investigate blocker", 1); err != nil {
		t.Fatalf("record blocked spawn: %v", err)
	}
	if err := spawnTree.UpdateStatus("Map-3", "blocked", "Waiting on input"); err != nil {
		t.Fatalf("mark blocked: %v", err)
	}

	rootCmd.SetArgs([]string{"status"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	output := buf.String()
	for _, want := range []string{"Spawn Activity", "1 active", "1 completed", "1 blocked", "Active Workers", "Recent Outcomes", "Hammer-1", "Keen-2", "Map-3"} {
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

func TestStatusShowsProofSummaryAndRoute(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	var buf bytes.Buffer
	stdout = &buf

	goal := "Expose proof in status"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Proof slice",
					Status: colony.PhaseReady,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Add the proof command", Status: colony.TaskPending}},
				},
			},
		},
	})

	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	output := buf.String()
	for _, want := range []string{"Proof", "Context:", "Inspect: aether proof", "aether proof"} {
		if !strings.Contains(output, want) {
			t.Fatalf("status output missing %q\n%s", want, output)
		}
	}
	if strings.Contains(output, "## Colony State") {
		t.Fatalf("status should not dump full proof ledger\n%s", output)
	}
}
