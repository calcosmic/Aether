package cmd

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPendingDecisionAdd(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"pending-decision-add", "--description", "choose a database"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["added"] != true {
		t.Errorf("added = %v, want true", result["added"])
	}
	id := result["id"].(string)
	if id == "" {
		t.Error("id should not be empty")
	}
}

func TestPendingDecisionAddWithFlags(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{
		"pending-decision-add",
		"--type", "architectural",
		"--description", "which ORM to use",
		"--phase", "2",
		"--source", "team discussion",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	id := result["id"].(string)

	// Verify it was actually saved
	var buf2 bytes.Buffer
	stdout = &buf2
	rootCmd.SetArgs([]string{"pending-decision-list"})
	rootCmd.Execute()

	env2 := parseEnvelope(t, buf2.String())
	r2 := env2["result"].(map[string]interface{})
	decisions := r2["decisions"].([]interface{})
	if len(decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decisions))
	}
	d := decisions[0].(map[string]interface{})
	if d["id"] != id {
		t.Errorf("id = %v, want %v", d["id"], id)
	}
	if d["description"] != "which ORM to use" {
		t.Errorf("description = %v, want 'which ORM to use'", d["description"])
	}
	if d["type"] != "architectural" {
		t.Errorf("type = %v, want 'architectural'", d["type"])
	}
}

func TestPendingDecisionListEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"pending-decision-list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["total"] != float64(0) {
		t.Errorf("total = %v, want 0", result["total"])
	}
	if result["unresolved"] != float64(0) {
		t.Errorf("unresolved = %v, want 0", result["unresolved"])
	}
	decisions := result["decisions"].([]interface{})
	if len(decisions) != 0 {
		t.Errorf("decisions = %v, want empty", decisions)
	}
}

func TestPendingDecisionListFilterUnresolved(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create two decisions, resolve one
	rootCmd.SetArgs([]string{"pending-decision-add", "--description", "open decision"})
	rootCmd.Execute()

	var buf2 bytes.Buffer
	stdout = &buf2
	rootCmd.SetArgs([]string{"pending-decision-add", "--description", "to be resolved"})
	rootCmd.Execute()

	// Get the ID of the second decision
	env2 := parseEnvelope(t, buf2.String())
	id2 := env2["result"].(map[string]interface{})["id"].(string)

	// Resolve the second one
	var buf3 bytes.Buffer
	stderr = &buf3
	rootCmd.SetArgs([]string{"pending-decision-resolve", "--id", id2, "--resolution", "we decided"})
	rootCmd.Execute()

	// List all
	var buf4 bytes.Buffer
	stdout = &buf4
	rootCmd.SetArgs([]string{"pending-decision-list"})
	rootCmd.Execute()

	env4 := parseEnvelope(t, buf4.String())
	r4 := env4["result"].(map[string]interface{})
	if r4["total"] != float64(2) {
		t.Errorf("total = %v, want 2", r4["total"])
	}
	if r4["unresolved"] != float64(1) {
		t.Errorf("unresolved = %v, want 1", r4["unresolved"])
	}

	// List only unresolved
	var buf5 bytes.Buffer
	stdout = &buf5
	rootCmd.SetArgs([]string{"pending-decision-list", "--unresolved"})
	rootCmd.Execute()

	env5 := parseEnvelope(t, buf5.String())
	r5 := env5["result"].(map[string]interface{})
	decisions := r5["decisions"].([]interface{})
	if len(decisions) != 1 {
		t.Errorf("unresolved decisions = %v, want 1", len(decisions))
	}
}

func TestPendingDecisionListFilterType(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"pending-decision-add", "--type", "arch", "--description", "arch decision"})
	rootCmd.Execute()

	var buf2 bytes.Buffer
	stdout = &buf2
	rootCmd.SetArgs([]string{"pending-decision-add", "--type", "tech", "--description", "tech decision"})
	rootCmd.Execute()

	var buf3 bytes.Buffer
	stdout = &buf3
	rootCmd.SetArgs([]string{"pending-decision-list", "--type", "arch"})
	rootCmd.Execute()

	env3 := parseEnvelope(t, buf3.String())
	r3 := env3["result"].(map[string]interface{})
	decisions := r3["decisions"].([]interface{})
	if len(decisions) != 1 {
		t.Errorf("arch decisions = %v, want 1", len(decisions))
	}
}

func TestPendingDecisionResolve(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"pending-decision-add", "--description", "resolve me"})
	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	id := env["result"].(map[string]interface{})["id"].(string)

	var buf2 bytes.Buffer
	stdout = &buf2
	rootCmd.SetArgs([]string{"pending-decision-resolve", "--id", id, "--resolution", "done"})
	rootCmd.Execute()

	env2 := parseEnvelope(t, buf2.String())
	if env2["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env2["ok"])
	}
	r2 := env2["result"].(map[string]interface{})
	if r2["resolved"] != true {
		t.Errorf("resolved = %v, want true", r2["resolved"])
	}
}

func TestPendingDecisionResolveNotFound(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"pending-decision-resolve", "--id", "pd_nonexistent", "--resolution", "nope"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for non-existent id, got: %v", env["ok"])
	}
}

func TestPendingDecisionResolveRejectsStaleScopedDecision(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "Resolve scoped pending decision"
	currentSession := "session_current_pending"
	oldSession := "session_old_pending"
	initializedAt := time.Date(2026, 5, 12, 9, 0, 0, 0, time.UTC)
	if err := store.SaveJSON("COLONY_STATE.json", colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		State:         colony.StateREADY,
		SessionID:     &currentSession,
		InitializedAt: &initializedAt,
	}); err != nil {
		t.Fatalf("seed colony state: %v", err)
	}
	if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{Decisions: []PendingDecision{{
		ID:          "pd_old_decision",
		Type:        clarificationDecisionType,
		Description: "old decision",
		Resolved:    false,
		CreatedAt:   "2026-05-11T10:00:00Z",
		GoalHash:    pendingDecisionGoalHash(goal),
		SessionID:   oldSession,
	}}}); err != nil {
		t.Fatalf("seed pending decisions: %v", err)
	}

	rootCmd.SetArgs([]string{"pending-decision-resolve", "--id", "pd_old_decision", "--resolution", "done"})
	rootCmd.Execute()

	env := parseEnvelope(t, errBuf.String())
	if env["ok"] != false {
		t.Fatalf("expected ok:false for stale decision resolve, got %v", env["ok"])
	}
	var file PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
		t.Fatalf("load pending decisions: %v", err)
	}
	if file.Decisions[0].Resolved {
		t.Fatal("stale scoped decision should not be resolved")
	}
}

func TestPendingDecisionAddMissingDescription(t *testing.T) {
	resetRootCmd(t)
	var outBuf, errBuf bytes.Buffer
	saveGlobals(t)
	forceJSONOutputModeForTest(t)
	stdout = &outBuf
	stderr = &errBuf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Explicitly set --description to empty to override any persisted flag value
	rootCmd.SetArgs([]string{"pending-decision-add", "--description", ""})

	rootCmd.Execute()

	errOutput := errBuf.String()
	if errOutput == "" {
		t.Fatal("expected error output on stderr but got none")
	}

	env := parseEnvelope(t, errOutput)
	if env["ok"] != false {
		t.Errorf("expected ok:false for missing description, got: %v", env["ok"])
	}
}
