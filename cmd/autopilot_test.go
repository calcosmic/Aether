package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestAutopilotInitCreatesState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"autopilot-init", "--phases", "5"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("autopilot-init returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", out["ok"])
	}
	result := out["result"].(map[string]interface{})
	if result["initialized"] != true {
		t.Fatalf("expected initialized=true, got: %v", result["initialized"])
	}
	if result["total_phases"] != float64(5) {
		t.Fatalf("expected total_phases=5, got: %v", result["total_phases"])
	}

	// Verify filesystem state
	statePath := tmpDir + "/.aether/data/autopilot/state.json"
	raw, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read state.json: %v", err)
	}
	var state autopilotState
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("parse state.json: %v", err)
	}
	if state.TotalPhases != 5 {
		t.Fatalf("expected TotalPhases=5, got %d", state.TotalPhases)
	}
	if state.Status != "initialized" {
		t.Fatalf("expected Status=initialized, got %q", state.Status)
	}
}

func TestAutopilotStatusNotInitialized(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"autopilot-status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("autopilot-status returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", out["ok"])
	}
	result := out["result"].(map[string]interface{})
	if result["active"] != false {
		t.Fatalf("expected active=false when not initialized, got: %v", result["active"])
	}
}

func TestAutopilotUpdateMutatesState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Initialize first
	rootCmd.SetArgs([]string{"autopilot-init", "--phases", "3"})
	_ = rootCmd.Execute()

	resetRootCmd(t)
	stdout = &buf
	buf.Reset()

	rootCmd.SetArgs([]string{"autopilot-update", "--phase", "1", "--status", "completed"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("autopilot-update returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", out["ok"])
	}
	result := out["result"].(map[string]interface{})
	if result["updated"] != true {
		t.Fatalf("expected updated=true, got: %v", result["updated"])
	}
	if result["phase"] != float64(1) {
		t.Fatalf("expected phase=1, got: %v", result["phase"])
	}

	// Verify state
	statePath := tmpDir + "/.aether/data/autopilot/state.json"
	raw, _ := os.ReadFile(statePath)
	var state autopilotState
	json.Unmarshal(raw, &state)
	if len(state.Phases) != 1 {
		t.Fatalf("expected 1 phase entry, got %d", len(state.Phases))
	}
	if state.Phases[0].Status != "completed" {
		t.Fatalf("expected phase status completed, got %q", state.Phases[0].Status)
	}
	if state.Status != "running" {
		t.Fatalf("expected autopilot status running, got %q", state.Status)
	}
}

func TestAutopilotStop(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Initialize
	rootCmd.SetArgs([]string{"autopilot-init", "--phases", "2"})
	_ = rootCmd.Execute()

	resetRootCmd(t)
	stdout = &buf
	buf.Reset()

	rootCmd.SetArgs([]string{"autopilot-stop"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("autopilot-stop returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", out["ok"])
	}
	result := out["result"].(map[string]interface{})
	if result["stopped"] != true {
		t.Fatalf("expected stopped=true, got: %v", result["stopped"])
	}

	// Verify state
	statePath := tmpDir + "/.aether/data/autopilot/state.json"
	raw, _ := os.ReadFile(statePath)
	var state autopilotState
	json.Unmarshal(raw, &state)
	if state.Status != "stopped" {
		t.Fatalf("expected status=stopped, got %q", state.Status)
	}
}
