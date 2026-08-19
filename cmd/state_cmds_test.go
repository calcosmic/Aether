package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/tidwall/gjson"
)

func TestSetNestedFieldJSON_NumericValue(t *testing.T) {
	// Simulate a COLONY_STATE.json with a plan that has a confidence field.
	state := map[string]interface{}{
		"version": "1.0.0",
		"plan": map[string]interface{}{
			"confidence": 50.0,
			"phases": []interface{}{
				map[string]interface{}{
					"id":     1,
					"status": "pending",
				},
			},
		},
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to marshal test state: %v", err)
	}

	// Set plan.confidence to "85" via the --field path.
	// The current buggy code uses sjson.SetBytes which JSON-encodes the string,
	// producing "85" (a quoted string) instead of 85 (a number).
	result, err := setNestedFieldJSON(data, "plan.confidence", "85")
	if err != nil {
		t.Fatalf("setNestedFieldJSON returned error: %v", err)
	}

	// Assert that plan.confidence is numeric 85, not a quoted string.
	confidenceResult := gjson.GetBytes(result, "plan.confidence")
	if !confidenceResult.Exists() {
		t.Fatal("plan.confidence does not exist in result")
	}
	if confidenceResult.Type != gjson.Number {
		t.Errorf("plan.confidence should be a number, got type %v with raw value %q", confidenceResult.Type, confidenceResult.Raw)
	}
	if confidenceResult.Int() != 85 {
		t.Errorf("plan.confidence = %v, want 85", confidenceResult.Int())
	}
}

func TestSetNestedFieldJSON_DeepNestedPath(t *testing.T) {
	// Test setting a deeply nested field like plan.phases.0.status.
	state := map[string]interface{}{
		"version": "1.0.0",
		"plan": map[string]interface{}{
			"phases": []interface{}{
				map[string]interface{}{
					"id":     1,
					"status": "pending",
				},
			},
		},
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to marshal test state: %v", err)
	}

	result, err := setNestedFieldJSON(data, "plan.phases.0.status", "in-progress")
	if err != nil {
		t.Fatalf("setNestedFieldJSON returned error: %v", err)
	}

	statusResult := gjson.GetBytes(result, "plan.phases.0.status")
	if !statusResult.Exists() {
		t.Fatal("plan.phases.0.status does not exist in result")
	}
	if statusResult.Str != "in-progress" {
		t.Errorf("plan.phases.0.status = %q, want %q", statusResult.Str, "in-progress")
	}
	// Also verify it's a string, not a double-quoted string.
	if statusResult.Raw != `"in-progress"` {
		t.Errorf("plan.phases.0.status raw = %q, want %q", statusResult.Raw, `"in-progress"`)
	}
}

func TestSetNestedFieldJSON_NumericArrayElement(t *testing.T) {
	// Test setting an array element to a numeric value.
	state := map[string]interface{}{
		"version": "1.0.0",
		"plan": map[string]interface{}{
			"phases": []interface{}{
				map[string]interface{}{
					"id":  1,
					"seq": 0,
				},
			},
		},
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to marshal test state: %v", err)
	}

	result, err := setNestedFieldJSON(data, "plan.phases.0.seq", "42")
	if err != nil {
		t.Fatalf("setNestedFieldJSON returned error: %v", err)
	}

	seqResult := gjson.GetBytes(result, "plan.phases.0.seq")
	if !seqResult.Exists() {
		t.Fatal("plan.phases.0.seq does not exist in result")
	}
	if seqResult.Type != gjson.Number {
		t.Errorf("plan.phases.0.seq should be a number, got type %v with raw value %q", seqResult.Type, seqResult.Raw)
	}
	if seqResult.Int() != 42 {
		t.Errorf("plan.phases.0.seq = %v, want 42", seqResult.Int())
	}
}

func TestStateMutateBracket(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "build the thing"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateEXECUTING,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "phase one", Status: "pending"},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	// Bracket notation should set plan.phases[0].status to "completed".
	// Currently fails because reFieldSet regex only accepts [\w.]+
	// which does not include [ or ].
	rootCmd.SetArgs([]string{"state-mutate", `.plan.phases[0].status = "completed"`})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected cobra error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected state-mutate to succeed with bracket notation, got: %v", env)
	}

	// Verify the phase status was actually updated in the file.
	var updated colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &updated)
	if len(updated.Plan.Phases) == 0 {
		t.Fatal("expected at least 1 phase in state")
	}
	if updated.Plan.Phases[0].Status != "completed" {
		t.Errorf("phases[0].status = %q, want %q", updated.Plan.Phases[0].Status, "completed")
	}
}

func TestStateMutateExpressionStringNotDoubleQuoted(t *testing.T) {
	// Regression: expression mode with a quoted string value should store "READY"
	// not "\"READY\"" (literal escaped quotes embedded in the string).
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
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "phase one", Status: "pending"},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"state-mutate", `.state = "EXECUTING"`})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var updated colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &updated)
	if updated.State != colony.StateEXECUTING {
		t.Errorf("state = %q, want %q", updated.State, colony.StateEXECUTING)
	}
}

// TestStateMutateExpressionNumericStillWorks used to prove
// `.current_phase = 3` succeeded via the expression syntax with no --guard
// flag anywhere -- exactly the bypass CR-03 (188-REVIEW.md) closes:
// state-mutate's `--field current_phase` path already required a matching
// `--guard phase-advance:<N>`, but the free-form jq-like expression syntax
// reached the same destructive field through executeExpression, a
// completely separate code path with no field-name-specific validation at
// all. This test now asserts the expression syntax is refused exactly like
// --field is, proving both paths share one gated way to move current_phase.
// See TestStateMutateExpressionCurrentPhaseSucceedsWithMatchingGuard for the
// (still working, now gated) success case, and
// TestStateMutateExpressionNonCurrentPhaseFieldRemainsUnguarded for proof
// this does not over-block other fields.
func TestStateMutateExpressionNumericStillWorks(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	stderr = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	s.SaveJSON("COLONY_STATE.json", phaseAdvanceReadyState())
	beforeData, _ := s.ReadFile("COLONY_STATE.json")

	// No --guard at all -- the exact shape this test used to prove succeeded.
	rootCmd.SetArgs([]string{"state-mutate", `.current_phase = 2`})
	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] == true {
		t.Fatalf("expected `.current_phase = N` with no --guard to be refused, got: %v", env)
	}

	afterData, _ := s.ReadFile("COLONY_STATE.json")
	if string(beforeData) != string(afterData) {
		t.Error("COLONY_STATE.json changed on disk despite the refused, unguarded expression-syntax current_phase mutation")
	}
	var updated colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &updated)
	if updated.CurrentPhase != 1 {
		t.Errorf("current_phase = %d, want unchanged 1", updated.CurrentPhase)
	}
}

// TestStateMutateExpressionCurrentPhaseSucceedsWithMatchingGuard proves the
// expression syntax is still usable for current_phase -- just gated the
// same way --field is -- and that numeric values still round-trip through
// SetRawBytes (raw JSON, not a quoted string), the original property
// TestStateMutateExpressionNumericStillWorks protected.
func TestStateMutateExpressionCurrentPhaseSucceedsWithMatchingGuard(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	stderr = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	s.SaveJSON("COLONY_STATE.json", phaseAdvanceReadyState())

	rootCmd.SetArgs([]string{"state-mutate", "--guard", "phase-advance:2", `.current_phase = 2`})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected cobra error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected a matching phase-advance guard to allow the expression-syntax mutation, got: %v", env)
	}

	var updated colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &updated)
	if updated.CurrentPhase != 2 {
		t.Errorf("current_phase = %d, want 2", updated.CurrentPhase)
	}
}

// TestStateMutateExpressionNonCurrentPhaseFieldRemainsUnguarded proves the
// new guard does not over-block: expression-syntax mutations of any OTHER
// field must keep working with no --guard at all, exactly as before.
func TestStateMutateExpressionNonCurrentPhaseFieldRemainsUnguarded(t *testing.T) {
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
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "phase one", Status: "pending"},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"state-mutate", `.milestone = "Brood Stable"`})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected an unguarded, non-current_phase expression mutation to still succeed, got: %v", env)
	}

	var updated colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &updated)
	if updated.Milestone != "Brood Stable" {
		t.Errorf("milestone = %q, want %q", updated.Milestone, "Brood Stable")
	}
}

// --- T-188-07: state-mutate --field current_phase must require a matching
// phase-advance guard. These use newTestStoreWithRoot (not newTestStore)
// because a --guard value routes through enforceGuard -> runGateCheck ->
// checkTestsPass, which shells out to a resolved test command; without an
// isolated AETHER_ROOT that resolution can walk up to this very repo's
// CLAUDE.md and try to run `go test ./...` recursively from inside a test.

// phaseAdvanceReadyState returns a fixture colony where phase 2's
// phase-advance preconditions genuinely pass: phase 1 already completed,
// phase 2 has one task and it is completed, and there are no critical error
// records. This is the one fixture shape shared by the "guard matches"
// success case and the "guard target mismatch" refusal case below.
func phaseAdvanceReadyState() colony.ColonyState {
	goal := "current_phase guard test"
	taskID := "2.1"
	return colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "phase one", Status: colony.PhaseCompleted},
				{ID: 2, Name: "phase two", Status: colony.PhaseReady, Tasks: []colony.Task{
					{ID: &taskID, Goal: "finish phase two", Status: colony.TaskCompleted},
				}},
			},
		},
	}
}

func TestStateMutateCurrentPhaseRequiresGuard(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	stderr = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	s.SaveJSON("COLONY_STATE.json", phaseAdvanceReadyState())
	beforeData, _ := s.ReadFile("COLONY_STATE.json")

	// No --guard at all.
	rootCmd.SetArgs([]string{"state-mutate", "--field", "current_phase", "--value", "2"})
	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] == true {
		t.Fatalf("expected state-mutate --field current_phase with no --guard to be refused, got: %v", env)
	}

	afterData, _ := s.ReadFile("COLONY_STATE.json")
	if string(beforeData) != string(afterData) {
		t.Error("COLONY_STATE.json changed on disk despite the refused, unguarded current_phase mutation")
	}
	var updated colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &updated)
	if updated.CurrentPhase != 1 {
		t.Errorf("current_phase = %d, want unchanged 1", updated.CurrentPhase)
	}
}

func TestStateMutateCurrentPhaseRejectsWrongGuardType(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	stderr = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	s.SaveJSON("COLONY_STATE.json", phaseAdvanceReadyState())
	beforeData, _ := s.ReadFile("COLONY_STATE.json")

	// A task-complete guard is the wrong kind of guard for current_phase.
	rootCmd.SetArgs([]string{"state-mutate", "--field", "current_phase", "--value", "2", "--guard", "task-complete:2.1"})
	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] == true {
		t.Fatalf("expected current_phase with a task-complete guard to be refused, got: %v", env)
	}

	afterData, _ := s.ReadFile("COLONY_STATE.json")
	if string(beforeData) != string(afterData) {
		t.Error("COLONY_STATE.json changed on disk despite the refused, wrong-guard-type mutation")
	}
	var updated colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &updated)
	if updated.CurrentPhase != 1 {
		t.Errorf("current_phase = %d, want unchanged 1", updated.CurrentPhase)
	}
}

func TestStateMutateCurrentPhaseRejectsMismatchedGuardTarget(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	stderr = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	s.SaveJSON("COLONY_STATE.json", phaseAdvanceReadyState())
	beforeData, _ := s.ReadFile("COLONY_STATE.json")

	// The guard is a genuine, passing phase-advance guard -- but for phase 2,
	// while --value asks to set current_phase to 3. The mismatch must refuse.
	rootCmd.SetArgs([]string{"state-mutate", "--field", "current_phase", "--value", "3", "--guard", "phase-advance:2"})
	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] == true {
		t.Fatalf("expected mismatched guard target (phase-advance:2 for --value 3) to be refused, got: %v", env)
	}

	afterData, _ := s.ReadFile("COLONY_STATE.json")
	if string(beforeData) != string(afterData) {
		t.Error("COLONY_STATE.json changed on disk despite the refused, mismatched-target mutation")
	}
	var updated colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &updated)
	if updated.CurrentPhase != 1 {
		t.Errorf("current_phase = %d, want unchanged 1", updated.CurrentPhase)
	}
}

func TestStateMutateCurrentPhaseSucceedsWithMatchingGuard(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	stderr = &buf

	s, tmpDir := newTestStoreWithRoot(t)
	defer os.RemoveAll(tmpDir)
	store = s

	s.SaveJSON("COLONY_STATE.json", phaseAdvanceReadyState())

	// A genuine, matching phase-advance guard for the exact phase being set
	// must still work exactly as it does today.
	rootCmd.SetArgs([]string{"state-mutate", "--field", "current_phase", "--value", "2", "--guard", "phase-advance:2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected cobra error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected a matching phase-advance guard to succeed, got: %v", env)
	}

	var updated colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &updated)
	if updated.CurrentPhase != 2 {
		t.Errorf("current_phase = %d, want 2", updated.CurrentPhase)
	}
}
