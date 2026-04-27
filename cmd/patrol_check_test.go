package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// setupPatrolData creates a temp .aether/data/ directory, sets COLONY_DATA_DIR,
// and returns the data directory path.
func setupPatrolData(t *testing.T) string {
	t.Helper()
	orig := os.Getenv("COLONY_DATA_DIR")
	t.Cleanup(func() { os.Setenv("COLONY_DATA_DIR", orig) })

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}
	os.Setenv("COLONY_DATA_DIR", dataDir)
	return dataDir
}

// runPatrolCheck executes the patrol-check subcommand and returns parsed result.
func runPatrolCheck(t *testing.T, dataDir string) *PatrolResult {
	t.Helper()

	store = nil

	var buf bytes.Buffer
	oldStdout := stdout
	oldStderr := stderr
	t.Cleanup(func() {
		stdout = oldStdout
		stderr = oldStderr
	})

	stdout = &buf
	stderr = &buf

	rootCmd.SetArgs([]string{"patrol-check"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("rootCmd.Execute: %v", err)
	}

	rootCmd.SetArgs([]string{})
	store = nil

	out := buf.String()
	var envelope struct {
		OK     bool         `json:"ok"`
		Result PatrolResult `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("failed to parse output: %v\nOutput:\n%s", err, out)
	}
	if !envelope.OK {
		t.Fatalf("patrol-check returned ok=false: %s", out)
	}
	return &envelope.Result
}

// findCheck finds a check by name in the result.
func findCheck(result *PatrolResult, name string) (PatrolCheck, bool) {
	for _, c := range result.Checks {
		if c.Name == name {
			return c, true
		}
	}
	return PatrolCheck{}, false
}

// findFile finds a file check by path in a check's files.
func findFile(check PatrolCheck, path string) (PatrolCheckFile, bool) {
	for _, f := range check.Files {
		if f.Path == path {
			return f, true
		}
	}
	return PatrolCheckFile{}, false
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestPatrolCheckAllHealthy(t *testing.T) {
	dataDir := setupPatrolData(t)

	goal := "Test colony"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 5,
	})

	strength := 0.8
	sp := 5
	writeJSONFile(t, dataDir, "pheromones.json", colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "sig-1", Type: "FOCUS", Active: true, Strength: &strength, SourcePhase: &sp},
		},
	})

	writeJSONFile(t, dataDir, "session.json", map[string]interface{}{
		"session_id":  "test-session",
		"colony_goal": goal,
	})

	result := runPatrolCheck(t, dataDir)

	if result.OverallStatus != "healthy" {
		t.Errorf("overall_status: expected healthy, got %s", result.OverallStatus)
	}

	jv, ok := findCheck(result, "json_validity")
	if !ok {
		t.Fatal("json_validity check not found")
	}
	if jv.Status != "healthy" {
		t.Errorf("json_validity status: expected healthy, got %s", jv.Status)
	}
	if len(jv.Files) != 3 {
		t.Errorf("json_validity files: expected 3, got %d", len(jv.Files))
	}

	spCheck, ok := findCheck(result, "stale_pheromones")
	if !ok {
		t.Fatal("stale_pheromones check not found")
	}
	if spCheck.Status != "healthy" {
		t.Errorf("stale_pheromones status: expected healthy, got %s", spCheck.Status)
	}
	if spCheck.StaleCount != 0 {
		t.Errorf("stale_pheromones stale_count: expected 0, got %d", spCheck.StaleCount)
	}

	ib, ok := findCheck(result, "interrupted_builds")
	if !ok {
		t.Fatal("interrupted_builds check not found")
	}
	if ib.Status != "healthy" {
		t.Errorf("interrupted_builds status: expected healthy, got %s", ib.Status)
	}
}

func TestPatrolCheckInvalidJSON(t *testing.T) {
	dataDir := setupPatrolData(t)

	goal := "Test"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY, CurrentPhase: 1,
	})

	writeFile(t, dataDir, "pheromones.json", []byte(`{not valid json`))

	writeJSONFile(t, dataDir, "session.json", map[string]interface{}{
		"session_id": "test-session", "colony_goal": goal,
	})

	result := runPatrolCheck(t, dataDir)

	if result.OverallStatus != "error" {
		t.Errorf("overall_status: expected error, got %s", result.OverallStatus)
	}

	jv, ok := findCheck(result, "json_validity")
	if !ok {
		t.Fatal("json_validity check not found")
	}
	if jv.Status != "error" {
		t.Errorf("json_validity status: expected error, got %s", jv.Status)
	}

	f, ok := findFile(jv, "pheromones.json")
	if !ok {
		t.Fatal("pheromones.json file not found in json_validity check")
	}
	if f.Status != "error" {
		t.Errorf("pheromones.json status: expected error, got %s", f.Status)
	}
	if f.Severity != "warning" {
		t.Errorf("pheromones.json severity: expected warning, got %s", f.Severity)
	}
}

func TestPatrolCheckMissingFile(t *testing.T) {
	dataDir := setupPatrolData(t)

	goal := "Test"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY, CurrentPhase: 1,
	})
	writeJSONFile(t, dataDir, "pheromones.json", colony.PheromoneFile{Signals: []colony.PheromoneSignal{}})
	// session.json intentionally missing

	result := runPatrolCheck(t, dataDir)

	jv, ok := findCheck(result, "json_validity")
	if !ok {
		t.Fatal("json_validity check not found")
	}

	f, ok := findFile(jv, "session.json")
	if !ok {
		t.Fatal("session.json file not found in json_validity check")
	}
	if f.Status != "missing" {
		t.Errorf("session.json status: expected missing, got %s", f.Status)
	}
	if f.Severity != "info" {
		t.Errorf("session.json severity: expected info, got %s", f.Severity)
	}
}

func TestPatrolCheckEmptyFile(t *testing.T) {
	dataDir := setupPatrolData(t)

	goal := "Test"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY, CurrentPhase: 1,
	})
	writeFile(t, dataDir, "pheromones.json", []byte(""))
	writeJSONFile(t, dataDir, "session.json", map[string]interface{}{
		"session_id": "test-session", "colony_goal": goal,
	})

	result := runPatrolCheck(t, dataDir)

	jv, ok := findCheck(result, "json_validity")
	if !ok {
		t.Fatal("json_validity check not found")
	}

	f, ok := findFile(jv, "pheromones.json")
	if !ok {
		t.Fatal("pheromones.json file not found in json_validity check")
	}
	if f.Status != "empty" {
		t.Errorf("pheromones.json status: expected empty, got %s", f.Status)
	}
	if f.Severity != "info" {
		t.Errorf("pheromones.json severity: expected info, got %s", f.Severity)
	}
}

func TestPatrolCheckStalePheromones(t *testing.T) {
	dataDir := setupPatrolData(t)

	goal := "Test"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateEXECUTING, CurrentPhase: 5,
	})

	stalePhase := 2
	strength := 0.8
	writeJSONFile(t, dataDir, "pheromones.json", colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "stale-sig", Type: "FOCUS", Active: true, Strength: &strength, SourcePhase: &stalePhase},
		},
	})
	writeJSONFile(t, dataDir, "session.json", map[string]interface{}{
		"session_id": "test-session", "colony_goal": goal,
	})

	result := runPatrolCheck(t, dataDir)

	if result.OverallStatus != "warning" {
		t.Errorf("overall_status: expected warning, got %s", result.OverallStatus)
	}

	sc, ok := findCheck(result, "stale_pheromones")
	if !ok {
		t.Fatal("stale_pheromones check not found")
	}
	if sc.Status != "warning" {
		t.Errorf("stale_pheromones status: expected warning, got %s", sc.Status)
	}
	if sc.StaleCount != 1 {
		t.Errorf("stale_count: expected 1, got %d", sc.StaleCount)
	}
	if len(sc.StaleSignals) != 1 {
		t.Fatalf("stale_signals: expected 1, got %d", len(sc.StaleSignals))
	}
	if sc.StaleSignals[0].ID != "stale-sig" {
		t.Errorf("stale signal ID: expected stale-sig, got %s", sc.StaleSignals[0].ID)
	}
}

func TestPatrolCheckZeroStrength(t *testing.T) {
	dataDir := setupPatrolData(t)

	goal := "Test"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateEXECUTING, CurrentPhase: 5,
	})

	zeroStrength := 0.0
	sp := 5
	writeJSONFile(t, dataDir, "pheromones.json", colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "zero-sig", Type: "REDIRECT", Active: true, Strength: &zeroStrength, SourcePhase: &sp},
		},
	})
	writeJSONFile(t, dataDir, "session.json", map[string]interface{}{
		"session_id": "test-session", "colony_goal": goal,
	})

	result := runPatrolCheck(t, dataDir)

	sc, ok := findCheck(result, "stale_pheromones")
	if !ok {
		t.Fatal("stale_pheromones check not found")
	}
	if sc.Status != "warning" {
		t.Errorf("stale_pheromones status: expected warning, got %s", sc.Status)
	}
	if sc.StaleCount != 1 {
		t.Errorf("stale_count: expected 1, got %d", sc.StaleCount)
	}
	if len(sc.StaleSignals) != 1 {
		t.Fatalf("stale_signals: expected 1, got %d", len(sc.StaleSignals))
	}
	if sc.StaleSignals[0].ID != "zero-sig" {
		t.Errorf("stale signal ID: expected zero-sig, got %s", sc.StaleSignals[0].ID)
	}
}

func TestPatrolCheckNoStaleSignals(t *testing.T) {
	dataDir := setupPatrolData(t)

	goal := "Test"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateEXECUTING, CurrentPhase: 3,
	})

	strength := 0.9
	sp := 3
	writeJSONFile(t, dataDir, "pheromones.json", colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "active-sig", Type: "FOCUS", Active: true, Strength: &strength, SourcePhase: &sp},
		},
	})
	writeJSONFile(t, dataDir, "session.json", map[string]interface{}{
		"session_id": "test-session", "colony_goal": goal,
	})

	result := runPatrolCheck(t, dataDir)

	sc, ok := findCheck(result, "stale_pheromones")
	if !ok {
		t.Fatal("stale_pheromones check not found")
	}
	if sc.Status != "healthy" {
		t.Errorf("stale_pheromones status: expected healthy, got %s", sc.Status)
	}
	if sc.StaleCount != 0 {
		t.Errorf("stale_count: expected 0, got %d", sc.StaleCount)
	}
}

func TestPatrolCheckInterruptedBuild(t *testing.T) {
	dataDir := setupPatrolData(t)

	goal := "Test"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY, CurrentPhase: 1,
	})
	writeJSONFile(t, dataDir, "pheromones.json", colony.PheromoneFile{Signals: []colony.PheromoneSignal{}})
	writeJSONFile(t, dataDir, "session.json", map[string]interface{}{
		"session_id": "test-session", "colony_goal": goal,
	})
	writeJSONFile(t, dataDir, "build_manifest_123.json", map[string]interface{}{
		"phase": 1, "status": "interrupted",
	})

	result := runPatrolCheck(t, dataDir)

	if result.OverallStatus != "warning" {
		t.Errorf("overall_status: expected warning, got %s", result.OverallStatus)
	}

	ib, ok := findCheck(result, "interrupted_builds")
	if !ok {
		t.Fatal("interrupted_builds check not found")
	}
	if ib.Status != "warning" {
		t.Errorf("interrupted_builds status: expected warning, got %s", ib.Status)
	}
	if len(ib.Artifacts) != 1 {
		t.Errorf("artifacts: expected 1, got %d", len(ib.Artifacts))
	}
}

func TestPatrolCheckNoInterrupt(t *testing.T) {
	dataDir := setupPatrolData(t)

	goal := "Test"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY, CurrentPhase: 1,
	})
	writeJSONFile(t, dataDir, "pheromones.json", colony.PheromoneFile{Signals: []colony.PheromoneSignal{}})
	writeJSONFile(t, dataDir, "session.json", map[string]interface{}{
		"session_id": "test-session", "colony_goal": goal,
	})

	result := runPatrolCheck(t, dataDir)

	ib, ok := findCheck(result, "interrupted_builds")
	if !ok {
		t.Fatal("interrupted_builds check not found")
	}
	if ib.Status != "healthy" {
		t.Errorf("interrupted_builds status: expected healthy, got %s", ib.Status)
	}
	if len(ib.Artifacts) != 0 {
		t.Errorf("artifacts: expected 0, got %d", len(ib.Artifacts))
	}
}
