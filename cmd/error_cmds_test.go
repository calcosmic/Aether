package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// --- error-add tests ---

func TestErrorAdd(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test goal"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Errors:  colony.Errors{Records: []colony.ErrorRecord{}},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"error-add", "--category", "build", "--severity", "critical", "--description", "build failed on import"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	id, ok := result["id"].(string)
	if !ok || id == "" {
		t.Errorf("id = %v, want non-empty string", result["id"])
	}
	if result["category"] != "build" {
		t.Errorf("category = %v, want build", result["category"])
	}

	// Verify the record was persisted
	var updated colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &updated)
	if len(updated.Errors.Records) != 1 {
		t.Fatalf("records count = %d, want 1", len(updated.Errors.Records))
	}
	if updated.Errors.Records[0].Category != "build" {
		t.Errorf("record category = %q, want build", updated.Errors.Records[0].Category)
	}
	if updated.Errors.Records[0].Severity != "critical" {
		t.Errorf("record severity = %q, want critical", updated.Errors.Records[0].Severity)
	}
	if updated.Errors.Records[0].Description != "build failed on import" {
		t.Errorf("record description = %q, want 'build failed on import'", updated.Errors.Records[0].Description)
	}
}

func TestErrorAddWithPhase(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test goal"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Errors:  colony.Errors{Records: []colony.ErrorRecord{}},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"error-add", "--category", "test", "--severity", "warning", "--description", "flaky test", "--phase", "3"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	// Verify the record was persisted with phase
	var updated colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &updated)
	if len(updated.Errors.Records) != 1 {
		t.Fatalf("records count = %d, want 1", len(updated.Errors.Records))
	}
	if updated.Errors.Records[0].Phase == nil || *updated.Errors.Records[0].Phase != 3 {
		t.Errorf("phase = %v, want 3", updated.Errors.Records[0].Phase)
	}
}

func TestErrorAddMissingArgs(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"error-add"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for missing args, got: %v", env["ok"])
	}
}

func TestErrorAddCapsAt50(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test goal"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Errors:  colony.Errors{Records: []colony.ErrorRecord{}},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	// Add 52 errors to test the cap at 50
	for i := 0; i < 52; i++ {
		buf.Reset()
		rootCmd.SetArgs([]string{"error-add", "--category", "build", "--severity", "warning", "--description", "filler error"})
		rootCmd.Execute()
	}

	// Verify only 50 records remain
	var updated colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &updated)
	if len(updated.Errors.Records) != 50 {
		t.Errorf("records count = %d, want 50 (capped)", len(updated.Errors.Records))
	}
}

// --- error-flag-pattern tests ---

func TestErrorFlagPatternNew(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"error-flag-pattern", "--name", "build-loop", "--description", "Build keeps looping", "--severity", "critical"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["created"] != true {
		t.Errorf("created = %v, want true", result["created"])
	}
	if result["pattern"] != "build-loop" {
		t.Errorf("pattern = %v, want build-loop", result["pattern"])
	}

	// Verify the file was created
	var patterns map[string]interface{}
	s.LoadJSON("error-patterns.json", &patterns)
	patternsList := patterns["patterns"].([]interface{})
	if len(patternsList) != 1 {
		t.Fatalf("patterns count = %d, want 1", len(patternsList))
	}
	p := patternsList[0].(map[string]interface{})
	if p["name"] != "build-loop" {
		t.Errorf("pattern name = %v, want build-loop", p["name"])
	}
	if p["severity"] != "critical" {
		t.Errorf("pattern severity = %v, want critical", p["severity"])
	}
}

func TestErrorFlagPatternUpdate(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create initial pattern
	s.SaveJSON("error-patterns.json", map[string]interface{}{
		"version":  float64(1),
		"patterns": []interface{}{map[string]interface{}{"name": "build-loop", "description": "desc", "severity": "warning", "occurrences": float64(1), "first_seen": "2026-01-01T00:00:00Z", "last_seen": "2026-01-01T00:00:00Z"}},
	})

	rootCmd.SetArgs([]string{"error-flag-pattern", "--name", "build-loop", "--description", "updated desc"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["updated"] != true {
		t.Errorf("updated = %v, want true", result["updated"])
	}
	if result["occurrences"] != float64(2) {
		t.Errorf("occurrences = %v, want 2", result["occurrences"])
	}
}

func TestErrorFlagPatternMissingName(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"error-flag-pattern"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for missing name, got: %v", env["ok"])
	}
}

// --- error-summary tests ---

func TestErrorSummary(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test goal"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Errors: colony.Errors{
			Records: []colony.ErrorRecord{
				{ID: "1", Category: "build", Severity: "critical", Description: "err1", Timestamp: "2026-01-01T00:00:00Z"},
				{ID: "2", Category: "build", Severity: "warning", Description: "err2", Timestamp: "2026-01-01T00:00:01Z"},
				{ID: "3", Category: "test", Severity: "warning", Description: "err3", Timestamp: "2026-01-01T00:00:02Z"},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"error-summary"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["total"] != float64(3) {
		t.Errorf("total = %v, want 3", result["total"])
	}
	byCategory := result["by_category"].(map[string]interface{})
	if byCategory["build"] != float64(2) {
		t.Errorf("by_category.build = %v, want 2", byCategory["build"])
	}
	if byCategory["test"] != float64(1) {
		t.Errorf("by_category.test = %v, want 1", byCategory["test"])
	}
	bySeverity := result["by_severity"].(map[string]interface{})
	if bySeverity["critical"] != float64(1) {
		t.Errorf("by_severity.critical = %v, want 1", bySeverity["critical"])
	}
	if bySeverity["warning"] != float64(2) {
		t.Errorf("by_severity.warning = %v, want 2", bySeverity["warning"])
	}
}

func TestErrorSummaryNoStateFile(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"error-summary"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false when no state file, got: %v", env["ok"])
	}
}

// --- error-pattern-check tests ---

func TestErrorPatternCheck(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test goal"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Errors: colony.Errors{
			Records: []colony.ErrorRecord{
				{ID: "1", Category: "build", Severity: "critical", Description: "err1", Timestamp: "2026-01-01T00:00:00Z"},
				{ID: "2", Category: "build", Severity: "warning", Description: "err2", Timestamp: "2026-01-01T00:00:01Z"},
				{ID: "3", Category: "build", Severity: "warning", Description: "err3", Timestamp: "2026-01-01T00:00:02Z"},
				{ID: "4", Category: "test", Severity: "info", Description: "err4", Timestamp: "2026-01-01T00:00:03Z"},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"error-pattern-check"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].([]interface{})
	// Only "build" has 3+ entries
	if len(result) != 1 {
		t.Fatalf("pattern groups = %d, want 1", len(result))
	}
	group := result[0].(map[string]interface{})
	if group["category"] != "build" {
		t.Errorf("group category = %v, want build", group["category"])
	}
	if group["count"] != float64(3) {
		t.Errorf("group count = %v, want 3", group["count"])
	}
}

func TestErrorPatternCheckBelowThreshold(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test goal"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Errors: colony.Errors{
			Records: []colony.ErrorRecord{
				{ID: "1", Category: "build", Severity: "critical", Description: "err1", Timestamp: "2026-01-01T00:00:00Z"},
				{ID: "2", Category: "test", Severity: "warning", Description: "err2", Timestamp: "2026-01-01T00:00:01Z"},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"error-pattern-check"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].([]interface{})
	if len(result) != 0 {
		t.Errorf("pattern groups = %d, want 0 (all below threshold of 3)", len(result))
	}
}

func TestErrorPatternCheckNoStateFile(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"error-pattern-check"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false when no state file, got: %v", env["ok"])
	}
}

// parseErrorPatternsFile is a test helper to read error-patterns.json
func parseErrorPatternsFile(t *testing.T, store interface {
	LoadJSON(string, interface{}) error
}) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := store.LoadJSON("error-patterns.json", &result); err != nil {
		t.Fatalf("failed to load error-patterns.json: %v", err)
	}
	return result
}

// Helper to remarshal a generic map back to typed struct for inspection
func remarshal(t *testing.T, input, output interface{}) {
	t.Helper()
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("remarshal: %v", err)
	}
	if err := json.Unmarshal(data, output); err != nil {
		t.Fatalf("remarshal: %v", err)
	}
}
