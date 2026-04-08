package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// plan-granularity get tests
// ---------------------------------------------------------------------------

func TestPlanGranularityGet_NoState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"plan-granularity", "get"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok, got: %v", env)
	}
	result := env["result"].(map[string]interface{})
	if result["granularity"] != "none" {
		t.Errorf("expected granularity 'none', got %v", result["granularity"])
	}
	if result["source"] != "default" {
		t.Errorf("expected source 'default', got %v", result["source"])
	}
}

func TestPlanGranularityGet_EmptyState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"plan-granularity", "get"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["granularity"] != "none" {
		t.Errorf("expected granularity 'none', got %v", result["granularity"])
	}
	if result["source"] != "default" {
		t.Errorf("expected source 'default', got %v", result["source"])
	}
}

func TestPlanGranularityGet_WithValue(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test"
	state := colony.ColonyState{
		Version:         "3.0",
		Goal:            &goal,
		State:           colony.StateREADY,
		PlanGranularity: colony.GranularitySprint,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"plan-granularity", "get"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["granularity"] != "sprint" {
		t.Errorf("expected granularity 'sprint', got %v", result["granularity"])
	}
	if result["source"] != "state" {
		t.Errorf("expected source 'state', got %v", result["source"])
	}
}

// ---------------------------------------------------------------------------
// plan-granularity set tests
// ---------------------------------------------------------------------------

func TestPlanGranularitySet_Valid(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"plan-granularity", "set", "--granularity", "sprint"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok, got: %v", env)
	}
	result := env["result"].(map[string]interface{})
	if result["granularity"] != "sprint" {
		t.Errorf("expected granularity 'sprint', got %v", result["granularity"])
	}
	if result["source"] != "cli" {
		t.Errorf("expected source 'cli', got %v", result["source"])
	}

	// Verify persisted
	var loaded colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &loaded)
	if loaded.PlanGranularity != colony.GranularitySprint {
		t.Errorf("persisted granularity = %q, want %q", loaded.PlanGranularity, colony.GranularitySprint)
	}
}

func TestPlanGranularitySet_Invalid(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"plan-granularity", "set", "--granularity", "invalid"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected cobra error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok=false for invalid granularity, got: %v", env)
	}
	if env["code"] != float64(1) {
		t.Errorf("expected code 1, got: %v", env["code"])
	}
}

// ---------------------------------------------------------------------------
// Status granularity display test
// ---------------------------------------------------------------------------

func TestStatusOutput_Granularity(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test colony goal for granularity"
	state := colony.ColonyState{
		Version:         "3.0",
		Goal:            &goal,
		State:           colony.StateEXECUTING,
		CurrentPhase:    1,
		PlanGranularity: colony.GranularityMilestone,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: "in_progress", Tasks: []colony.Task{}},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"status"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Granularity:") {
		t.Errorf("status output missing 'Granularity:'\ngot:\n%s", output)
	}
	if !strings.Contains(output, "milestone") {
		t.Errorf("status output missing 'milestone' granularity label\ngot:\n%s", output)
	}
	if !strings.Contains(output, "4-7") {
		t.Errorf("status output missing range '4-7' for milestone\ngot:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// state-mutate plan_granularity tests
// ---------------------------------------------------------------------------

func TestStateMutate_PlanGranularity_Valid(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"state-mutate", "--field", "plan_granularity", "--value", "quarter"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok, got: %v", env)
	}

	var loaded colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &loaded)
	if loaded.PlanGranularity != colony.GranularityQuarter {
		t.Errorf("persisted plan_granularity = %q, want %q", loaded.PlanGranularity, colony.GranularityQuarter)
	}
}

func TestStateMutate_PlanGranularity_Invalid(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"state-mutate", "--field", "plan_granularity", "--value", "bad"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected cobra error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok=false for invalid plan_granularity, got: %v", env)
	}
	if env["code"] != float64(1) {
		t.Errorf("expected code 1, got: %v", env["code"])
	}
}

// ---------------------------------------------------------------------------
// Granularity range in get output
// ---------------------------------------------------------------------------

func TestPlanGranularityGet_WithRange(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test"
	state := colony.ColonyState{
		Version:         "3.0",
		Goal:            &goal,
		State:           colony.StateREADY,
		PlanGranularity: colony.GranularityQuarter,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"plan-granularity", "get"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["min"] != float64(8) {
		t.Errorf("expected min 8, got %v", result["min"])
	}
	if result["max"] != float64(12) {
		t.Errorf("expected max 12, got %v", result["max"])
	}
}

// ---------------------------------------------------------------------------
// JSON round-trip for plan_granularity field
// ---------------------------------------------------------------------------

func TestPlanGranularityJSONRoundTrip(t *testing.T) {
	goal := "test"
	state := colony.ColonyState{
		Version:         "3.0",
		Goal:            &goal,
		State:           colony.StateREADY,
		PlanGranularity: colony.GranularityMajor,
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded colony.ColonyState
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.PlanGranularity != colony.GranularityMajor {
		t.Errorf("round-trip: got %q, want %q", decoded.PlanGranularity, colony.GranularityMajor)
	}
}
