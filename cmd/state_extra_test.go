package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// --- state-checkpoint tests ---

func TestStateCheckpoint(t *testing.T) {
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
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"state-checkpoint", "--name", "before-rebuild"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["checkpoint"] != "before-rebuild" {
		t.Errorf("checkpoint = %v, want before-rebuild", result["checkpoint"])
	}
	if result["path"] != "checkpoints/before-rebuild.json" {
		t.Errorf("path = %v, want checkpoints/before-rebuild.json", result["path"])
	}

	// Verify the checkpoint file was created and has matching content
	var checkpoint colony.ColonyState
	if err := s.LoadJSON("checkpoints/before-rebuild.json", &checkpoint); err != nil {
		t.Fatalf("checkpoint file not created: %v", err)
	}
	if *checkpoint.Goal != "test goal" {
		t.Errorf("checkpoint goal = %q, want %q", *checkpoint.Goal, "test goal")
	}
}

func TestStateCheckpointMissingName(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"state-checkpoint"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for missing --name, got: %v", env["ok"])
	}
}

func TestStateCheckpointNoStateFile(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"state-checkpoint", "--name", "test"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false when COLONY_STATE.json missing, got: %v", env["ok"])
	}
}

// --- state-write tests ---

func TestStateWrite(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "original goal"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"state-write", "--field", "version", "--value", "4.0"})

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
	if result["field"] != "version" {
		t.Errorf("field = %v, want version", result["field"])
	}
	if result["value"] != "4.0" {
		t.Errorf("value = %v, want 4.0", result["value"])
	}

	// Verify the file was actually updated
	data, _ := s.ReadFile("COLONY_STATE.json")
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	if m["version"] != "4.0" {
		t.Errorf("version in file = %v, want 4.0", m["version"])
	}
}

func TestStateWritePositionalArg(t *testing.T) {
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
	}
	s.SaveJSON("COLONY_STATE.json", state)

	// state-write should accept a positional JSON blob and write it directly
	rootCmd.SetArgs([]string{"state-write", `{"goal":"updated goal","version":"3.0"}`})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected state-write to accept positional JSON, got error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	// Verify the file was updated with the new JSON
	data, _ := s.ReadFile("COLONY_STATE.json")
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	if m["goal"] != "updated goal" {
		t.Errorf("goal in file = %v, want 'updated goal'", m["goal"])
	}
}

func TestStateWriteMissingField(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"state-write"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for missing --field, got: %v", env["ok"])
	}
}

// --- phase-insert tests ---

func TestPhaseInsert(t *testing.T) {
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
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseCompleted, Tasks: []colony.Task{}},
				{ID: 2, Name: "Phase 2", Status: colony.PhasePending, Tasks: []colony.Task{}},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"phase-insert", "--after", "1", "--name", "Fix Bug", "--description", "Fix critical bug"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["inserted"] != true {
		t.Errorf("inserted = %v, want true", result["inserted"])
	}
	if result["phase_id"] != float64(3) {
		t.Errorf("phase_id = %v, want 3", result["phase_id"])
	}
	if result["after"] != float64(1) {
		t.Errorf("after = %v, want 1", result["after"])
	}

	// Verify the phase was inserted at the right position
	var updated colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &updated)
	if len(updated.Plan.Phases) != 3 {
		t.Fatalf("phase count = %d, want 3", len(updated.Plan.Phases))
	}
	if updated.Plan.Phases[1].Name != "Fix Bug" {
		t.Errorf("inserted phase name = %q, want 'Fix Bug'", updated.Plan.Phases[1].Name)
	}
	if updated.Plan.Phases[1].Status != colony.PhasePending {
		t.Errorf("inserted phase status = %q, want pending", updated.Plan.Phases[1].Status)
	}
}

func TestPhaseInsertAtEnd(t *testing.T) {
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
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseCompleted, Tasks: []colony.Task{}},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"phase-insert", "--after", "1", "--name", "Phase 2", "--description", "Second phase"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["inserted"] != true {
		t.Errorf("inserted = %v, want true", result["inserted"])
	}

	var updated colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &updated)
	if len(updated.Plan.Phases) != 2 {
		t.Fatalf("phase count = %d, want 2", len(updated.Plan.Phases))
	}
	if updated.Plan.Phases[1].Name != "Phase 2" {
		t.Errorf("phase at index 1 = %q, want 'Phase 2'", updated.Plan.Phases[1].Name)
	}
}

func TestPhaseInsertInvalidAfter(t *testing.T) {
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
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Tasks: []colony.Task{}},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"phase-insert", "--after", "5", "--name", "Bad", "--description", "Invalid"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for invalid after index, got: %v", env["ok"])
	}
}

// --- validate-oracle-state tests ---

func TestValidateOracleState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	s.SaveJSON("oracle/state.json", map[string]string{"status": "active"})
	s.SaveJSON("oracle/plan.json", map[string]string{"plan": "research"})

	rootCmd.SetArgs([]string{"validate-oracle-state"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["valid"] != true {
		t.Errorf("valid = %v, want true", result["valid"])
	}
	files := result["files"].(map[string]interface{})
	if files["state.json"] != true {
		t.Errorf("state.json valid = %v, want true", files["state.json"])
	}
	if files["plan.json"] != true {
		t.Errorf("plan.json valid = %v, want true", files["plan.json"])
	}
}

func TestValidateOracleStateMissing(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"validate-oracle-state"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["valid"] != false {
		t.Errorf("valid = %v, want false when files missing", result["valid"])
	}
	issues := result["issues"].([]interface{})
	if len(issues) != 2 {
		t.Errorf("issues count = %d, want 2", len(issues))
	}
}

func TestValidateOracleStateInvalidJSON(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	s.AtomicWrite("oracle/state.json", []byte("not json"))
	s.SaveJSON("oracle/plan.json", map[string]string{"plan": "research"})

	rootCmd.SetArgs([]string{"validate-oracle-state"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["valid"] != false {
		t.Errorf("valid = %v, want false for invalid JSON", result["valid"])
	}
}

// --- view-state tests ---

func TestViewStateInit(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"view-state-init"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["initialized"] != true {
		t.Errorf("initialized = %v, want true", result["initialized"])
	}
}

func TestViewStateGetNotFound(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"view-state-get", "--key", "nonexistent"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["found"] != false {
		t.Errorf("found = %v, want false", result["found"])
	}
	if result["value"] != nil {
		t.Errorf("value = %v, want nil", result["value"])
	}
}

func TestViewStateSetAndGet(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Set a value
	rootCmd.SetArgs([]string{"view-state-set", "--key", "theme", "--value", "dark"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["set"] != true {
		t.Errorf("set = %v, want true", result["set"])
	}

	// Now get it back
	buf.Reset()
	rootCmd.SetArgs([]string{"view-state-get", "--key", "theme"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env = parseEnvelope(t, buf.String())
	result = env["result"].(map[string]interface{})
	if result["found"] != true {
		t.Errorf("found = %v, want true", result["found"])
	}
	if result["value"] != "dark" {
		t.Errorf("value = %v, want dark", result["value"])
	}
}

func TestViewStateToggle(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"view-state-toggle", "--key", "sidebar"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["value"] != true {
		t.Errorf("value = %v, want true (default false -> toggle to true)", result["value"])
	}

	// Toggle again should flip to false
	buf.Reset()
	rootCmd.SetArgs([]string{"view-state-toggle", "--key", "sidebar"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env = parseEnvelope(t, buf.String())
	result = env["result"].(map[string]interface{})
	if result["value"] != false {
		t.Errorf("value = %v, want false (true -> toggle to false)", result["value"])
	}
}

func TestViewStateExpand(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"view-state-expand", "--section", "details"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["expanded"] != true {
		t.Errorf("expanded = %v, want true", result["expanded"])
	}
	if result["section"] != "details" {
		t.Errorf("section = %v, want details", result["section"])
	}

	// Verify the key was set correctly
	buf.Reset()
	rootCmd.SetArgs([]string{"view-state-get", "--key", "expanded_details"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env = parseEnvelope(t, buf.String())
	result = env["result"].(map[string]interface{})
	if result["value"] != true {
		t.Errorf("expanded_details = %v, want true", result["value"])
	}
}

func TestViewStateCollapse(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"view-state-collapse", "--section", "details"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["expanded"] != false {
		t.Errorf("expanded = %v, want false", result["expanded"])
	}
}

// --- grave tests ---

func TestGraveAdd(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"grave-add", "--agent", "builder-1", "--reason", "timeout", "--phase", "2"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["agent"] != "builder-1" {
		t.Errorf("agent = %v, want builder-1", result["agent"])
	}
	if result["buried"] != true {
		t.Errorf("buried = %v, want true", result["buried"])
	}
}

func TestGraveAddWithoutPhase(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"grave-add", "--agent", "builder-2", "--reason", "panic", "--phase", ""})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	// Verify the entry was saved
	var entries []GraveEntry
	s.LoadJSON("graveyard.json", &entries)
	if len(entries) != 1 {
		t.Fatalf("entries count = %d, want 1", len(entries))
	}
	if entries[0].Agent != "builder-2" {
		t.Errorf("agent = %q, want builder-2", entries[0].Agent)
	}
	if entries[0].Reason != "panic" {
		t.Errorf("reason = %q, want panic", entries[0].Reason)
	}
	if entries[0].Phase != "" {
		t.Errorf("phase = %q, want empty string", entries[0].Phase)
	}
	if entries[0].Created == "" {
		t.Error("created_at should not be empty")
	}
}

func TestGraveCheckFound(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	entries := []GraveEntry{
		{Agent: "builder-1", Reason: "timeout", Phase: "1", Created: "2026-01-01T00:00:00Z"},
		{Agent: "builder-2", Reason: "panic", Created: "2026-01-02T00:00:00Z"},
	}
	s.SaveJSON("graveyard.json", entries)

	rootCmd.SetArgs([]string{"grave-check", "--agent", "builder-1"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["found"] != true {
		t.Errorf("found = %v, want true", result["found"])
	}
	matching := result["entries"].([]interface{})
	if len(matching) != 1 {
		t.Errorf("entries count = %d, want 1", len(matching))
	}
}

func TestGraveCheckNotFound(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"grave-check", "--agent", "nonexistent"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["found"] != false {
		t.Errorf("found = %v, want false", result["found"])
	}
	entries := result["entries"].([]interface{})
	if len(entries) != 0 {
		t.Errorf("entries count = %d, want 0", len(entries))
	}
}

func TestGraveCheckNoFile(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"grave-check", "--agent", "builder-1"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["found"] != false {
		t.Errorf("found = %v, want false when no graveyard file", result["found"])
	}
}

// --- nil store tests ---
// Note: PersistentPreRunE always initializes the store before RunE runs,
// so nil store can only occur if PersistentPreRunE is bypassed.
// The nil checks exist as defensive guards in the code.

func TestStateCheckpointNilStore(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Verify the command works when store IS initialized (the normal path)
	goal := "test"
	state := colony.ColonyState{Version: "3.0", Goal: &goal}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"state-checkpoint", "--name", "test"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}
}
