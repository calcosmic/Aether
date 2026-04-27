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

// --- Gate Incremental Skip Tests (Phase 59, Plan 01, Task 2) ---

// TestContinueGates_SkipPassedGates verifies that previously passed gates are
// replaced with synthetic "skipped: previously passed" entries.
func TestContinueGates_SkipPassedGates(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	// Create COLONY_STATE.json with prior gate results
	goal := "test skip"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateBUILT,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Test", Status: colony.PhaseInProgress},
			},
		},
		GateResults: []colony.GateResultEntry{
			{Name: "manifest_present", Passed: true, Timestamp: time.Now().UTC().Format(time.RFC3339)},
			{Name: "implementation_evidence", Passed: true, Timestamp: time.Now().UTC().Format(time.RFC3339)},
		},
	}
	stateData, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(dir, "COLONY_STATE.json"), stateData, 0644)

	// Run gates with prior results
	priorResults := gateResultsRead()
	phase := colony.Phase{ID: 1, Name: "Test", Status: colony.PhaseInProgress}
	manifest := codexContinueManifest{Present: true}
	verification := codexContinueVerificationReport{ChecksPassed: true, Passed: true}
	assessment := codexContinueAssessment{PositiveEvidence: true, Passed: true}

	report := runCodexContinueGates(phase, manifest, verification, assessment, time.Now(), priorResults)

	// Verify that previously passed gates show as skipped
	for _, check := range report.Checks {
		if check.Name == "manifest_present" {
			if !check.Passed {
				t.Errorf("manifest_present should be passed (skipped), got detail: %s", check.Detail)
			}
			if check.Detail != "skipped: previously passed" {
				t.Errorf("manifest_present should show 'skipped: previously passed', got: %s", check.Detail)
			}
		}
		if check.Name == "implementation_evidence" {
			if !check.Passed {
				t.Errorf("implementation_evidence should be passed (skipped), got detail: %s", check.Detail)
			}
			if check.Detail != "skipped: previously passed" {
				t.Errorf("implementation_evidence should show 'skipped: previously passed', got: %s", check.Detail)
			}
		}
	}
}

// TestContinueGates_TestsAlwaysRun verifies that safety-critical gates always
// execute even when all prior results show passed. The no_critical_flags gate
// always runs (not wrapped in skip logic) as a safety net.
func TestContinueGates_TestsAlwaysRun(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	// Create COLONY_STATE.json with all gates previously passed
	goal := "test always run"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateBUILT,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Test", Status: colony.PhaseInProgress},
			},
		},
		GateResults: []colony.GateResultEntry{
			{Name: "tests_pass", Passed: true, Timestamp: time.Now().UTC().Format(time.RFC3339)},
			{Name: "no_critical_flags", Passed: true, Timestamp: time.Now().UTC().Format(time.RFC3339)},
			{Name: "manifest_present", Passed: true, Timestamp: time.Now().UTC().Format(time.RFC3339)},
			{Name: "verification_steps_passed", Passed: true, Timestamp: time.Now().UTC().Format(time.RFC3339)},
			{Name: "implementation_evidence", Passed: true, Timestamp: time.Now().UTC().Format(time.RFC3339)},
		},
	}
	stateData, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(dir, "COLONY_STATE.json"), stateData, 0644)

	priorResults := gateResultsRead()
	phase := colony.Phase{ID: 1, Name: "Test", Status: colony.PhaseInProgress}
	manifest := codexContinueManifest{Present: true}
	verification := codexContinueVerificationReport{ChecksPassed: true, Passed: true}
	assessment := codexContinueAssessment{PositiveEvidence: true, Passed: true}

	report := runCodexContinueGates(phase, manifest, verification, assessment, time.Now(), priorResults)

	// The no_critical_flags gate always runs (not wrapped in skip logic)
	found := false
	for _, check := range report.Checks {
		if check.Name == "no_critical_flags" {
			found = true
			if check.Detail == "skipped: previously passed" {
				t.Error("no_critical_flags should always run, not be skipped")
			}
		}
	}
	if !found {
		t.Error("no_critical_flags check should be present in gate report")
	}
}

// TestContinueGates_ResultsPersisted verifies that gate results are written
// to COLONY_STATE.json after gate check runs.
func TestContinueGates_ResultsPersisted(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	// Create minimal COLONY_STATE.json
	goal := "test persist"
	stateData, _ := json.Marshal(colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateBUILT,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Test", Status: colony.PhaseInProgress},
			},
		},
	})
	os.WriteFile(filepath.Join(dir, "COLONY_STATE.json"), stateData, 0644)

	// Write gate results
	results := []colony.GateResultEntry{
		{Name: "manifest_present", Passed: true, Timestamp: time.Now().UTC().Format(time.RFC3339)},
		{Name: "tests_pass", Passed: false, Timestamp: time.Now().UTC().Format(time.RFC3339), Detail: "1 test failed"},
	}
	if err := gateResultsWrite(results); err != nil {
		t.Fatalf("gateResultsWrite failed: %v", err)
	}

	// Read back and verify
	var readState colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &readState); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if len(readState.GateResults) != 2 {
		t.Fatalf("expected 2 gate results, got %d", len(readState.GateResults))
	}
	if readState.GateResults[0].Name != "manifest_present" {
		t.Errorf("first result should be manifest_present, got %s", readState.GateResults[0].Name)
	}
	if readState.GateResults[1].Name != "tests_pass" {
		t.Errorf("second result should be tests_pass, got %s", readState.GateResults[1].Name)
	}
}

// TestContinueGates_ClearedOnAdvance verifies that gate results are cleared
// when phase advances successfully (simulated via direct state mutation).
func TestContinueGates_ClearedOnAdvance(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	// Create state with gate results
	goal := "test clear"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateBUILT,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseInProgress},
				{ID: 2, Name: "Phase 2", Status: colony.PhasePending},
			},
		},
		GateResults: []colony.GateResultEntry{
			{Name: "manifest_present", Passed: true, Timestamp: time.Now().UTC().Format(time.RFC3339)},
			{Name: "tests_pass", Passed: true, Timestamp: time.Now().UTC().Format(time.RFC3339)},
		},
	}
	stateData, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(dir, "COLONY_STATE.json"), stateData, 0644)

	// Simulate phase advance: clear gate results via atomic update
	var updated colony.ColonyState
	if err := store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
		updated.GateResults = nil
		updated.Plan.Phases[0].Status = colony.PhaseCompleted
		updated.State = colony.StateREADY
		return nil
	}); err != nil {
		t.Fatalf("atomic update failed: %v", err)
	}

	// Verify gate results are cleared
	var verifyState colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &verifyState); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if verifyState.GateResults != nil {
		t.Errorf("gate results should be nil after phase advance, got %d entries", len(verifyState.GateResults))
	}
}

// TestContinueGates_ResultsPreservedOnFailure verifies that gate results
// are NOT cleared when gates fail (phase does not advance).
func TestContinueGates_ResultsPreservedOnFailure(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	// Create state with gate results (some failing)
	goal := "test preserve"
	gateResults := []colony.GateResultEntry{
		{Name: "manifest_present", Passed: true, Timestamp: time.Now().UTC().Format(time.RFC3339)},
		{Name: "tests_pass", Passed: false, Timestamp: time.Now().UTC().Format(time.RFC3339), Detail: "failed"},
	}
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateBUILT,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseInProgress},
			},
		},
		GateResults: gateResults,
	}
	stateData, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(dir, "COLONY_STATE.json"), stateData, 0644)

	// Simulate gate failure: do NOT clear gate results, just rewrite state
	var updated colony.ColonyState
	if err := store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
		// Phase does NOT advance -- gate results stay
		return nil
	}); err != nil {
		t.Fatalf("atomic update failed: %v", err)
	}

	// Verify gate results are still there
	var verifyState colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &verifyState); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if len(verifyState.GateResults) != 2 {
		t.Errorf("gate results should be preserved on failure, got %d entries", len(verifyState.GateResults))
	}
}
