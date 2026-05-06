package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestChamberCompareWithRealData(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create a chamber manifest
	chamberDir := filepath.Join(tmpDir, ".aether", "chambers", "test-chamber")
	os.MkdirAll(chamberDir, 0755)
	manifest := map[string]interface{}{
		"name":             "test-chamber",
		"goal":             "test goal",
		"milestone":        "v1.0",
		"phases_completed": 1,
		"total_phases":     3,
	}
	manifestData, marshalErr := json.MarshalIndent(manifest, "", "  ")
	if marshalErr != nil {
		t.Fatalf("failed to marshal manifest: %v", marshalErr)
	}
	os.WriteFile(filepath.Join(chamberDir, "manifest.json"), manifestData, 0644)

	// Create a colony state with matching goal but different phases_completed (2 vs 1)
	goal := "test goal"
	taskID := "t-1"
	state := colony.ColonyState{
		Goal:  &goal,
		State: colony.StateEXECUTING,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskID, Goal: "done", Status: colony.TaskCompleted}}},
				{ID: 2, Name: "Phase 2", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskID, Goal: "done", Status: colony.TaskCompleted}}},
				{ID: 3, Name: "Phase 3", Status: colony.PhaseInProgress, Tasks: []colony.Task{{ID: &taskID, Goal: "working", Status: colony.TaskPending}}},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"chamber-compare", "--name", "test-chamber"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})

	if result["chamber"] != "test-chamber" {
		t.Errorf("chamber = %v, want test-chamber", result["chamber"])
	}

	matches := result["matches"].([]interface{})
	diffs := result["diffs"].([]interface{})

	// Matches should NOT be empty -- goal should match
	if len(matches) == 0 {
		t.Errorf("expected non-empty matches, got: %v", matches)
	}

	// Diffs should NOT be empty -- milestone, phases_completed, total_phases differ
	if len(diffs) == 0 {
		t.Errorf("expected non-empty diffs, got: %v", diffs)
	}

	// Verify goal is in matches
	foundGoalMatch := false
	for _, m := range matches {
		entry := m.(map[string]interface{})
		if entry["field"] == "goal" {
			foundGoalMatch = true
			break
		}
	}
	if !foundGoalMatch {
		t.Errorf("expected 'goal' in matches, got: %v", matches)
	}

	// Verify phases_completed is in diffs
	foundPhaseDiff := false
	for _, d := range diffs {
		entry := d.(map[string]interface{})
		if entry["field"] == "phases_completed" {
			foundPhaseDiff = true
			break
		}
	}
	if !foundPhaseDiff {
		t.Errorf("expected 'phases_completed' in diffs, got: %v", diffs)
	}

	totalCompared, ok := result["total_compared"].(float64)
	if !ok || totalCompared != 4 {
		t.Errorf("total_compared = %v, want 4", result["total_compared"])
	}
}

func TestChamberCompareNoChamber(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test goal"
	s.SaveJSON("COLONY_STATE.json", colony.ColonyState{Goal: &goal})

	rootCmd.SetArgs([]string{"chamber-compare", "--name", "nonexistent"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})

	if result["chamber"] != "nonexistent" {
		t.Errorf("chamber = %v, want nonexistent", result["chamber"])
	}

	errMsg, ok := result["error"].(string)
	if !ok {
		t.Fatalf("expected error string, got: %T %v", result["error"], result["error"])
	}
	if !strings.Contains(errMsg, "not found") {
		t.Errorf("error = %q, want to contain 'not found'", errMsg)
	}

	matches := result["matches"].([]interface{})
	diffs := result["diffs"].([]interface{})
	if len(matches) != 0 {
		t.Errorf("matches = %v, want empty for nonexistent chamber", matches)
	}
	if len(diffs) != 0 {
		t.Errorf("diffs = %v, want empty for nonexistent chamber", diffs)
	}
}

func TestChamberCompareMatchingState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create a chamber manifest
	chamberDir := filepath.Join(tmpDir, ".aether", "chambers", "match-chamber")
	os.MkdirAll(chamberDir, 0755)
	manifest := map[string]interface{}{
		"name":             "match-chamber",
		"goal":             "shared goal",
		"milestone":        "",
		"phases_completed": 3,
		"total_phases":     3,
	}
	manifestData, marshalErr := json.MarshalIndent(manifest, "", "  ")
	if marshalErr != nil {
		t.Fatalf("failed to marshal manifest: %v", marshalErr)
	}
	os.WriteFile(filepath.Join(chamberDir, "manifest.json"), manifestData, 0644)

	// Create a colony state with identical goal and phases_completed = 3, total_phases = 3
	goal := "shared goal"
	taskID := "t-1"
	state := colony.ColonyState{
		Goal:  &goal,
		State: colony.StateCOMPLETED,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskID, Goal: "done", Status: colony.TaskCompleted}}},
				{ID: 2, Name: "Phase 2", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskID, Goal: "done", Status: colony.TaskCompleted}}},
				{ID: 3, Name: "Phase 3", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskID, Goal: "done", Status: colony.TaskCompleted}}},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"chamber-compare", "--name", "match-chamber"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})

	matches := result["matches"].([]interface{})
	diffs := result["diffs"].([]interface{})

	// All fields should match: goal (same), milestone (both empty), phases_completed (3), total_phases (3)
	// milestone: manifest has "" (empty string via json), colony has no milestone field -> compare to ""
	if len(matches) == 0 {
		t.Errorf("expected non-empty matches for identical state, got: %v", matches)
	}

	// No diffs expected
	if len(diffs) != 0 {
		t.Errorf("expected empty diffs for matching state, got: %v", diffs)
	}
}

func TestChamberCompareNoColonyState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create a chamber manifest but no COLONY_STATE.json
	chamberDir := filepath.Join(tmpDir, ".aether", "chambers", "no-state-chamber")
	os.MkdirAll(chamberDir, 0755)
	manifest := map[string]interface{}{
		"name":             "no-state-chamber",
		"goal":             "orphan goal",
		"milestone":        "",
		"phases_completed": 0,
		"total_phases":     0,
	}
	manifestData, marshalErr := json.MarshalIndent(manifest, "", "  ")
	if marshalErr != nil {
		t.Fatalf("failed to marshal manifest: %v", marshalErr)
	}
	os.WriteFile(filepath.Join(chamberDir, "manifest.json"), manifestData, 0644)

	rootCmd.SetArgs([]string{"chamber-compare", "--name", "no-state-chamber"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})

	// Without colony state, current values default to "" or 0.
	// Manifest has goal="orphan goal" which differs from current state, so goal goes to diffs. Milestone, phases_completed, and total_phases match against defaults., milestone="", phases=0, total=0 -- all match against defaults.
	matches := result["matches"].([]interface{})
	if len(matches) == 0 {
		t.Errorf("expected matches from manifest even without colony state, got: %v", matches)
	}

	errMsg, ok := result["error"].(string)
	if !ok || !strings.Contains(errMsg, "colony state not available") {
		t.Errorf("error = %v, want 'colony state not available'", result["error"])
	}
}

func TestTunnelsCompareTwoChambers(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	store = s

	for _, chamber := range []struct {
		name      string
		goal      string
		milestone string
		phases    int
		entombed  string
	}{
		{name: "first", goal: "Initial release", milestone: "Sealed Chambers", phases: 2, entombed: "2026-05-01T00:00:00Z"},
		{name: "second", goal: "Visual restoration", milestone: "Crowned Anthill", phases: 4, entombed: "2026-05-03T00:00:00Z"},
	} {
		chamberDir := filepath.Join(tmpDir, ".aether", "chambers", chamber.name)
		if err := os.MkdirAll(chamberDir, 0755); err != nil {
			t.Fatalf("mkdir chamber: %v", err)
		}
		manifest := map[string]interface{}{
			"name":             chamber.name,
			"goal":             chamber.goal,
			"milestone":        chamber.milestone,
			"phases_completed": chamber.phases,
			"total_phases":     chamber.phases,
			"entombed_at":      chamber.entombed,
		}
		data, err := json.Marshal(manifest)
		if err != nil {
			t.Fatalf("marshal manifest: %v", err)
		}
		if err := os.WriteFile(filepath.Join(chamberDir, "manifest.json"), data, 0644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
	}

	rootCmd.SetArgs([]string{"tunnels", "first", "second"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("tunnels returned error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["mode"] != "compare" {
		t.Fatalf("mode = %v, want compare", result["mode"])
	}
	growth := result["growth"].(map[string]interface{})
	if got := growth["phases_diff"]; got != float64(2) {
		t.Fatalf("phases_diff = %v, want 2", got)
	}
}

func TestTunnelsImportSignalsFromChamberArchive(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	store = s

	chamberDir := filepath.Join(tmpDir, ".aether", "chambers", "source-chamber")
	if err := os.MkdirAll(chamberDir, 0755); err != nil {
		t.Fatalf("mkdir chamber: %v", err)
	}
	manifest := []byte(`{"name":"source-chamber","goal":"source","milestone":"Crowned Anthill","phases_completed":1,"total_phases":1}`)
	if err := os.WriteFile(filepath.Join(chamberDir, "manifest.json"), manifest, 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	archive := `<?xml version="1.0" encoding="UTF-8"?>
<colony-archive xmlns="http://aether.colony/schemas/archive/1.0" colony_id="source-chamber" sealed_at="2026-05-01T00:00:00Z" version="1.0">
  <pheromones xmlns="http://aether.colony/schemas/pheromones" version="1.0" generated_at="2026-05-01T00:00:00Z" count="1">
    <signal id="sig-1" type="FOCUS" priority="normal" source="source" created_at="2026-05-01T00:00:00Z" active="true">
      <content><text>Carry this signal forward</text></content>
    </signal>
  </pheromones>
</colony-archive>`
	if err := os.WriteFile(filepath.Join(chamberDir, "colony-archive.xml"), []byte(archive), 0644); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	rootCmd.SetArgs([]string{"tunnels", "source-chamber", "--import-signals"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("tunnels returned error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["mode"] != "import" {
		t.Fatalf("mode = %v, want import", result["mode"])
	}
	if result["imported"] != float64(1) {
		t.Fatalf("imported = %v, want 1", result["imported"])
	}
	var pheromones colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pheromones); err != nil {
		t.Fatalf("load pheromones: %v", err)
	}
	if len(pheromones.Signals) != 1 {
		t.Fatalf("signals = %d, want 1", len(pheromones.Signals))
	}
	if !strings.HasPrefix(pheromones.Signals[0].ID, "source-chamber:") {
		t.Fatalf("signal id = %q, want chamber prefix", pheromones.Signals[0].ID)
	}
}
