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
// patrolCheckResult helpers
// ---------------------------------------------------------------------------

type patrolCheckResult struct {
	Checks        []patrolCheck `json:"checks"`
	OverallStatus string        `json:"overall_status"`
}

type patrolCheck struct {
	Name         string      `json:"name"`
	Status       string      `json:"status"`
	Files        []fileCheck `json:"files,omitempty"`
	StaleCount   int         `json:"stale_count,omitempty"`
	StaleSignals []patrolStaleInfo `json:"stale_signals,omitempty"`
	Artifacts    []string    `json:"artifacts,omitempty"`
}

type fileCheck struct {
	Path     string `json:"path"`
	Status   string `json:"status"`
	Severity string `json:"severity,omitempty"`
	Details  string `json:"details,omitempty"`
	Size     *int   `json:"size,omitempty"`
}

type patrolStaleInfo struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

// setupPatrolData creates a temp .aether/data/ directory, sets COLONY_DATA_DIR,
// and returns the data directory path. The caller must use t.TempDir() cleanup.
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

// runPatrolCheck executes the patrol-check subcommand and returns parsed JSON.
func runPatrolCheck(t *testing.T, dataDir string) *patrolCheckResult {
	t.Helper()

	// Reset store so PersistentPreRunE reinitializes it from COLONY_DATA_DIR
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

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("rootCmd.Execute: %v", err)
	}

	// Reset rootCmd args so other tests don't inherit them
	rootCmd.SetArgs([]string{})
	// Reset store
	store = nil

	out := buf.String()
	var envelope struct {
		OK     bool             `json:"ok"`
		Result patrolCheckResult `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("failed to parse output as JSON: %v\nOutput:\n%s", err, out)
	}
	if !envelope.OK {
		t.Fatalf("patrol-check returned ok=false: %s", out)
	}
	return &envelope.Result
}

// ---------------------------------------------------------------------------
// TestPatrolCheckAllHealthy
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

	rootCmd.SetArgs([]string{"patrol-check"})
	result := runPatrolCheck(t, dataDir)

	if result.OverallStatus != "healthy" {
		t.Errorf("overall_status: expected healthy, got %s", result.OverallStatus)
	}

	checkNames := make(map[string]patrolCheck)
	for _, c := range result.Checks {
		checkNames[c.Name] = c
	}

	jv, ok := checkNames["json_validity"]
	if !ok {
		t.Fatal("json_validity check not found")
	}
	if jv.Status != "healthy" {
		t.Errorf("json_validity status: expected healthy, got %s", jv.Status)
	}
	if len(jv.Files) != 3 {
		t.Errorf("json_validity files: expected 3, got %d", len(jv.Files))
	}

	spCheck, ok := checkNames["stale_pheromones"]
	if !ok {
		t.Fatal("stale_pheromones check not found")
	}
	if spCheck.Status != "healthy" {
		t.Errorf("stale_pheromones status: expected healthy, got %s", spCheck.Status)
	}
	if spCheck.StaleCount != 0 {
		t.Errorf("stale_pheromones stale_count: expected 0, got %d", spCheck.StaleCount)
	}

	ib, ok := checkNames["interrupted_builds"]
	if !ok {
		t.Fatal("interrupted_builds check not found")
	}
	if ib.Status != "healthy" {
		t.Errorf("interrupted_builds status: expected healthy, got %s", ib.Status)
	}
}

// ---------------------------------------------------------------------------
// TestPatrolCheckInvalidJSON
// ---------------------------------------------------------------------------

func TestPatrolCheckInvalidJSON(t *testing.T) {
	dataDir := setupPatrolData(t)

	goal := "Test"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
	})

	writeFile(t, dataDir, "pheromones.json", []byte(`{not valid json`))

	writeJSONFile(t, dataDir, "session.json", map[string]interface{}{
		"session_id":  "test-session",
		"colony_goal": goal,
	})

	rootCmd.SetArgs([]string{"patrol-check"})
	result := runPatrolCheck(t, dataDir)

	if result.OverallStatus != "error" {
		t.Errorf("overall_status: expected error, got %s", result.OverallStatus)
	}

	for _, c := range result.Checks {
		if c.Name == "json_validity" {
			if c.Status != "error" {
				t.Errorf("json_validity status: expected error, got %s", c.Status)
			}
			for _, f := range c.Files {
				if f.Path == "pheromones.json" {
					if f.Status != "error" {
						t.Errorf("pheromones.json status: expected error, got %s", f.Status)
					}
					if f.Severity != "warning" {
						t.Errorf("pheromones.json severity: expected warning, got %s", f.Severity)
					}
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestPatrolCheckMissingFile
// ---------------------------------------------------------------------------

func TestPatrolCheckMissingFile(t *testing.T) {
	dataDir := setupPatrolData(t)

	goal := "Test"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
	})

	writeJSONFile(t, dataDir, "pheromones.json", colony.PheromoneFile{Signals: []colony.PheromoneSignal{}})
	// Do NOT write session.json

	rootCmd.SetArgs([]string{"patrol-check"})
	result := runPatrolCheck(t, dataDir)

	for _, c := range result.Checks {
		if c.Name == "json_validity" {
			for _, f := range c.Files {
				if f.Path == "session.json" {
					if f.Status != "missing" {
						t.Errorf("session.json status: expected missing, got %s", f.Status)
					}
					if f.Severity != "info" {
						t.Errorf("session.json severity: expected info, got %s", f.Severity)
					}
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestPatrolCheckEmptyFile
// ---------------------------------------------------------------------------

func TestPatrolCheckEmptyFile(t *testing.T) {
	dataDir := setupPatrolData(t)

	goal := "Test"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
	})

	writeFile(t, dataDir, "pheromones.json", []byte(""))

	writeJSONFile(t, dataDir, "session.json", map[string]interface{}{
		"session_id":  "test-session",
		"colony_goal": goal,
	})

	rootCmd.SetArgs([]string{"patrol-check"})
	result := runPatrolCheck(t, dataDir)

	for _, c := range result.Checks {
		if c.Name == "json_validity" {
			for _, f := range c.Files {
				if f.Path == "pheromones.json" {
					if f.Status != "empty" {
						t.Errorf("pheromones.json status: expected empty, got %s", f.Status)
					}
					if f.Severity != "info" {
						t.Errorf("pheromones.json severity: expected info, got %s", f.Severity)
					}
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestPatrolCheckStalePheromones
// ---------------------------------------------------------------------------

func TestPatrolCheckStalePheromones(t *testing.T) {
	dataDir := setupPatrolData(t)

	goal := "Test"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 5,
	})

	stalePhase := 2
	strength := 0.8
	writeJSONFile(t, dataDir, "pheromones.json", colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "stale-sig", Type: "FOCUS", Active: true, Strength: &strength, SourcePhase: &stalePhase},
		},
	})

	writeJSONFile(t, dataDir, "session.json", map[string]interface{}{
		"session_id":  "test-session",
		"colony_goal": goal,
	})

	rootCmd.SetArgs([]string{"patrol-check"})
	result := runPatrolCheck(t, dataDir)

	if result.OverallStatus != "warning" {
		t.Errorf("overall_status: expected warning, got %s", result.OverallStatus)
	}

	for _, c := range result.Checks {
		if c.Name == "stale_pheromones" {
			if c.Status != "warning" {
				t.Errorf("stale_pheromones status: expected warning, got %s", c.Status)
			}
			if c.StaleCount != 1 {
				t.Errorf("stale_pheromones stale_count: expected 1, got %d", c.StaleCount)
			}
			if len(c.StaleSignals) != 1 {
				t.Fatalf("stale_pheromones stale_signals: expected 1, got %d", len(c.StaleSignals))
			}
			if c.StaleSignals[0].ID != "stale-sig" {
				t.Errorf("stale signal ID: expected stale-sig, got %s", c.StaleSignals[0].ID)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestPatrolCheckZeroStrength
// ---------------------------------------------------------------------------

func TestPatrolCheckZeroStrength(t *testing.T) {
	dataDir := setupPatrolData(t)

	goal := "Test"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 5,
	})

	zeroStrength := 0.0
	sp := 5
	writeJSONFile(t, dataDir, "pheromones.json", colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "zero-sig", Type: "REDIRECT", Active: true, Strength: &zeroStrength, SourcePhase: &sp},
		},
	})

	writeJSONFile(t, dataDir, "session.json", map[string]interface{}{
		"session_id":  "test-session",
		"colony_goal": goal,
	})

	rootCmd.SetArgs([]string{"patrol-check"})
	result := runPatrolCheck(t, dataDir)

	for _, c := range result.Checks {
		if c.Name == "stale_pheromones" {
			if c.Status != "warning" {
				t.Errorf("stale_pheromones status: expected warning, got %s", c.Status)
			}
			if c.StaleCount != 1 {
				t.Errorf("stale_pheromones stale_count: expected 1, got %d", c.StaleCount)
			}
			if len(c.StaleSignals) != 1 {
				t.Fatalf("stale_pheromones stale_signals: expected 1, got %d", len(c.StaleSignals))
			}
			if c.StaleSignals[0].ID != "zero-sig" {
				t.Errorf("stale signal ID: expected zero-sig, got %s", c.StaleSignals[0].ID)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestPatrolCheckNoStaleSignals
// ---------------------------------------------------------------------------

func TestPatrolCheckNoStaleSignals(t *testing.T) {
	dataDir := setupPatrolData(t)

	goal := "Test"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 3,
	})

	strength := 0.9
	sp := 3
	writeJSONFile(t, dataDir, "pheromones.json", colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "active-sig", Type: "FOCUS", Active: true, Strength: &strength, SourcePhase: &sp},
		},
	})

	writeJSONFile(t, dataDir, "session.json", map[string]interface{}{
		"session_id":  "test-session",
		"colony_goal": goal,
	})

	rootCmd.SetArgs([]string{"patrol-check"})
	result := runPatrolCheck(t, dataDir)

	for _, c := range result.Checks {
		if c.Name == "stale_pheromones" {
			if c.Status != "healthy" {
				t.Errorf("stale_pheromones status: expected healthy, got %s", c.Status)
			}
			if c.StaleCount != 0 {
				t.Errorf("stale_pheromones stale_count: expected 0, got %d", c.StaleCount)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestPatrolCheckInterruptedBuild
// ---------------------------------------------------------------------------

func TestPatrolCheckInterruptedBuild(t *testing.T) {
	dataDir := setupPatrolData(t)

	goal := "Test"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
	})

	writeJSONFile(t, dataDir, "pheromones.json", colony.PheromoneFile{Signals: []colony.PheromoneSignal{}})
	writeJSONFile(t, dataDir, "session.json", map[string]interface{}{
		"session_id":  "test-session",
		"colony_goal": goal,
	})

	writeJSONFile(t, dataDir, "build_manifest_123.json", map[string]interface{}{
		"phase":  1,
		"status": "interrupted",
	})

	rootCmd.SetArgs([]string{"patrol-check"})
	result := runPatrolCheck(t, dataDir)

	if result.OverallStatus != "warning" {
		t.Errorf("overall_status: expected warning, got %s", result.OverallStatus)
	}

	for _, c := range result.Checks {
		if c.Name == "interrupted_builds" {
			if c.Status != "warning" {
				t.Errorf("interrupted_builds status: expected warning, got %s", c.Status)
			}
			if len(c.Artifacts) != 1 {
				t.Errorf("interrupted_builds artifacts: expected 1, got %d", len(c.Artifacts))
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestPatrolCheckNoInterrupt
// ---------------------------------------------------------------------------

func TestPatrolCheckNoInterrupt(t *testing.T) {
	dataDir := setupPatrolData(t)

	goal := "Test"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
	})

	writeJSONFile(t, dataDir, "pheromones.json", colony.PheromoneFile{Signals: []colony.PheromoneSignal{}})
	writeJSONFile(t, dataDir, "session.json", map[string]interface{}{
		"session_id":  "test-session",
		"colony_goal": goal,
	})

	rootCmd.SetArgs([]string{"patrol-check"})
	result := runPatrolCheck(t, dataDir)

	for _, c := range result.Checks {
		if c.Name == "interrupted_builds" {
			if c.Status != "healthy" {
				t.Errorf("interrupted_builds status: expected healthy, got %s", c.Status)
			}
			if len(c.Artifacts) != 0 {
				t.Errorf("interrupted_builds artifacts: expected 0, got %d", len(c.Artifacts))
			}
		}
	}
}
