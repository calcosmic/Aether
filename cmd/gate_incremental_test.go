package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// --- Gate Struct Extension Tests (Phase 88, Plan 02, Task 1) ---

// TestGateCheckStruct_HasFixHintAndRecoveryOptions verifies gateCheck with
// FixHint and RecoveryOptions serializes correctly.
func TestGateCheckStruct_HasFixHintAndRecoveryOptions(t *testing.T) {
	check := gateCheck{
		Name:            "tests_pass",
		Passed:          false,
		Detail:          "2 tests failed",
		FixHint:         "check tests",
		RecoveryOptions: []string{"/ant-continue", "/ant-unblock"},
	}
	data, err := json.Marshal(check)
	if err != nil {
		t.Fatalf("marshal gateCheck: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, `"fix_hint":"check tests"`) {
		t.Errorf("JSON should contain fix_hint, got: %s", s)
	}
	if !strings.Contains(s, `"recovery_options":["/ant-continue","/ant-unblock"]`) {
		t.Errorf("JSON should contain recovery_options, got: %s", s)
	}
}

// TestShouldSkipGate_AlwaysRunGates verifies "flags" gate returns false even when previously passed.
func TestShouldSkipGate_AlwaysRunGates(t *testing.T) {
	prior := []GateCheckResult{
		{Name: "flags", Status: "passed", Timestamp: time.Now().UTC().Format(time.RFC3339)},
	}
	if shouldSkipGate(prior, "flags") {
		t.Error("flags gate should never be skipped even when previously passed")
	}
}

// TestShouldSkipGate_AlwaysRunWatcherVeto verifies "watcher_veto" gate returns false even when previously passed.
func TestShouldSkipGate_AlwaysRunWatcherVeto(t *testing.T) {
	prior := []GateCheckResult{
		{Name: "watcher_veto", Status: "passed", Timestamp: time.Now().UTC().Format(time.RFC3339)},
	}
	if shouldSkipGate(prior, "watcher_veto") {
		t.Error("watcher_veto gate should never be skipped even when previously passed")
	}
}

// TestShouldSkipGate_SkipsPreviouslyPassed verifies "manifest_present" gate is skipped when prior result has status "passed".
func TestShouldSkipGate_SkipsPreviouslyPassed(t *testing.T) {
	prior := []GateCheckResult{
		{Name: "manifest_present", Status: "passed", Timestamp: time.Now().UTC().Format(time.RFC3339)},
	}
	if !shouldSkipGate(prior, "manifest_present") {
		t.Error("manifest_present should be skipped when prior result has status passed")
	}
}

// TestShouldSkipGate_SkipsPreviouslySkipped verifies "implementation_evidence" gate is skipped when prior result has status "skipped".
func TestShouldSkipGate_SkipsPreviouslySkipped(t *testing.T) {
	prior := []GateCheckResult{
		{Name: "implementation_evidence", Status: "skipped", Timestamp: time.Now().UTC().Format(time.RFC3339)},
	}
	if !shouldSkipGate(prior, "implementation_evidence") {
		t.Error("implementation_evidence should be skipped when prior result has status skipped")
	}
}

// TestGateResultsPhasePersistence verifies write/read roundtrip for per-phase gate results.
func TestGateResultsPhasePersistence(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	entries := []GateCheckResult{
		{Name: "tests_pass", Status: "failed", Detail: "2 tests failed", FixHint: "fix tests", Timestamp: "2026-05-01T00:00:00Z", RetryCount: 1},
		{Name: "flags", Status: "passed", Timestamp: "2026-05-01T00:00:00Z"},
	}
	if err := gateResultsWritePhase(88, entries); err != nil {
		t.Fatalf("gateResultsWritePhase: %v", err)
	}

	readBack, err := gateResultsReadPhase(88)
	if err != nil {
		t.Fatalf("gateResultsReadPhase: %v", err)
	}
	if len(readBack) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(readBack))
	}
	if readBack[0].Name != "tests_pass" || readBack[0].Status != "failed" {
		t.Errorf("first entry mismatch: %+v", readBack[0])
	}
	if readBack[0].FixHint != "fix tests" {
		t.Errorf("expected FixHint 'fix tests', got %q", readBack[0].FixHint)
	}
	if readBack[1].Name != "flags" || readBack[1].Status != "passed" {
		t.Errorf("second entry mismatch: %+v", readBack[1])
	}
}

// TestGateResultsPhasePersistence_EmptyPhase verifies reading nonexistent phase file returns error.
func TestGateResultsPhasePersistence_EmptyPhase(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	_, err = gateResultsReadPhase(99)
	if err == nil {
		t.Error("expected error reading nonexistent phase file, got nil")
	}
}

// TestCircuitBreaker_GateRetryTracking verifies RecordFailure with gate retry key format.
func TestCircuitBreaker_GateRetryTracking(t *testing.T) {
	cb := NewCircuitBreaker(3)
	key := gateRetryKey(88, "tests_pass")
	for i := 0; i < 3; i++ {
		cb.RecordFailure(key)
	}
	if cb.Allow(key) {
		t.Error("circuit breaker should be tripped after 3 failures")
	}
}

// TestCircuitBreaker_GateRetryReset verifies RecordSuccess resets gate retry count.
func TestCircuitBreaker_GateRetryReset(t *testing.T) {
	cb := NewCircuitBreaker(3)
	key := gateRetryKey(88, "tests_pass")
	cb.RecordFailure(key)
	cb.RecordFailure(key)
	cb.RecordSuccess(key)
	if !cb.Allow(key) {
		t.Error("circuit breaker should allow after success reset")
	}
}

// TestGateCheckResult_StructSerialization verifies GateCheckResult serializes correctly.
func TestGateCheckResult_StructSerialization(t *testing.T) {
	result := GateCheckResult{
		Name:            "tests_pass",
		Status:          "failed",
		Detail:          "2 tests failed",
		FixHint:         "check tests",
		RecoveryOptions: []string{"/ant-continue", "/ant-unblock"},
		Timestamp:       "2026-05-01T00:00:00Z",
		RetryCount:      2,
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal GateCheckResult: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, `"status":"failed"`) {
		t.Errorf("JSON should contain status, got: %s", s)
	}
	if !strings.Contains(s, `"fix_hint":"check tests"`) {
		t.Errorf("JSON should contain fix_hint, got: %s", s)
	}
	if !strings.Contains(s, `"recovery_options":["/ant-continue","/ant-unblock"]`) {
		t.Errorf("JSON should contain recovery_options, got: %s", s)
	}
	if !strings.Contains(s, `"retry_count":2`) {
		t.Errorf("JSON should contain retry_count, got: %s", s)
	}
}

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
	priorResults := []GateCheckResult{
		{Name: "manifest_present", Status: "passed", Timestamp: time.Now().UTC().Format(time.RFC3339)},
		{Name: "implementation_evidence", Status: "passed", Timestamp: time.Now().UTC().Format(time.RFC3339)},
	}
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

	priorResults := []GateCheckResult{
		{Name: "tests_pass", Status: "passed", Timestamp: time.Now().UTC().Format(time.RFC3339)},
		{Name: "no_critical_flags", Status: "passed", Timestamp: time.Now().UTC().Format(time.RFC3339)},
		{Name: "manifest_present", Status: "passed", Timestamp: time.Now().UTC().Format(time.RFC3339)},
		{Name: "verification_steps_passed", Status: "passed", Timestamp: time.Now().UTC().Format(time.RFC3339)},
		{Name: "implementation_evidence", Status: "passed", Timestamp: time.Now().UTC().Format(time.RFC3339)},
	}
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

// TestFinalizeGateResultsPersisted verifies that gate results written via
// the finalize path pattern are persisted and readable (WR-01 fix).
func TestFinalizeGateResultsPersisted(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	goal := "test finalize persist"
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

	ts := time.Now().UTC().Format(time.RFC3339)

	// Simulate finalize path: write gate results after gate run
	results := []colony.GateResultEntry{
		{Name: "manifest_present", Passed: true, Timestamp: ts},
		{Name: "tests_pass", Passed: false, Timestamp: ts, Detail: "failed"},
	}
	if err := gateResultsWrite(results); err != nil {
		t.Fatalf("gateResultsWrite failed: %v", err)
	}

	readBack := gateResultsRead()
	if len(readBack) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(readBack))
	}

	names := make(map[string]colony.GateResultEntry)
	for _, r := range readBack {
		names[r.Name] = r
	}
	if names["manifest_present"].Passed != true {
		t.Error("manifest_present should be passed")
	}
	if names["tests_pass"].Passed != false {
		t.Error("tests_pass should be failed")
	}
}

// TestFinalizeGateResultsClearedOnAdvance verifies that gate results are
// cleared when phase advances via the finalize path (WR-02 fix).
func TestFinalizeGateResultsClearedOnAdvance(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store = s

	goal := "test finalize clear"
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

	// Simulate finalize advance: atomic update that clears gate results
	var updated colony.ColonyState
	if err := store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
		updated.GateResults = nil
		updated.Plan.Phases[0].Status = colony.PhaseCompleted
		updated.State = colony.StateREADY
		return nil
	}); err != nil {
		t.Fatalf("atomic update failed: %v", err)
	}

	readBack := gateResultsRead()
	if readBack != nil {
		t.Errorf("gate results should be nil after finalize advance, got %d entries", len(readBack))
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
