package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestOrchestratorDecompose(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s

	rootCmd.SetArgs([]string{"orchestrator-decompose", "--phase", "1"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("orchestrator-decompose returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			Phase     int    `json:"phase"`
			TaskCount int    `json:"task_count"`
			Tasks     []struct {
				ID    string `json:"id"`
				Goal  string `json:"goal"`
				Caste string `json:"caste"`
			} `json:"tasks"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("failed to parse JSON: %v\noutput: %s", err, output)
	}

	if envelope.Result.Phase != 1 {
		t.Errorf("phase = %d, want 1", envelope.Result.Phase)
	}
	if envelope.Result.TaskCount != 2 {
		t.Errorf("task_count = %d, want 2", envelope.Result.TaskCount)
	}
	for _, task := range envelope.Result.Tasks {
		if task.Caste == "" {
			t.Errorf("task %s has empty caste", task.ID)
		}
	}
}

func TestOrchestratorAssign(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s

	rootCmd.SetArgs([]string{"orchestrator-assign", "--phase", "2"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("orchestrator-assign returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			Phase       int `json:"phase"`
			Assignments []struct {
				TaskID string `json:"task_id"`
				Goal   string `json:"goal"`
				Caste  string `json:"caste"`
			} `json:"assignments"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("failed to parse JSON: %v\noutput: %s", err, output)
	}

	if envelope.Result.Phase != 2 {
		t.Errorf("phase = %d, want 2", envelope.Result.Phase)
	}
	if len(envelope.Result.Assignments) == 0 {
		t.Error("expected assignments, got none")
	}
}

func TestOrchestratorStatus_Idle(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s

	rootCmd.SetArgs([]string{"orchestrator-status"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("orchestrator-status returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"idle"`) {
		t.Errorf("expected idle status, got: %s", output)
	}
}

func TestOrchestratorStatus_Active(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s

	// Write state with active orchestrator state
	var state map[string]interface{}
	raw, err := os.ReadFile("testdata/colony_state.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal testdata: %v", err)
	}

	state["orchestrator_state"] = map[string]interface{}{
		"phase":      2,
		"status":     "dispatching",
		"task_count": 4,
		"completed":  2,
		"failed":     0,
	}

	stateBytes, _ := json.Marshal(state)
	if err := os.WriteFile(tmpDir+"/.aether/data/COLONY_STATE.json", stateBytes, 0644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	rootCmd.SetArgs([]string{"orchestrator-status"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("orchestrator-status returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"dispatching"`) {
		t.Errorf("expected dispatching status, got: %s", output)
	}
}

func TestOrchestratorDecompose_NoStore(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var errBuf bytes.Buffer
	stderr = &errBuf

	// Point to an empty temp dir with no COLONY_STATE.json
	tmpDir := t.TempDir()
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	rootCmd.SetArgs([]string{"orchestrator-decompose", "--phase", "1"})

	_ = rootCmd.Execute()

	output := errBuf.String()
	if !strings.Contains(output, "COLONY_STATE.json not found") {
		t.Errorf("expected 'COLONY_STATE.json not found' in stderr, got: %s", output)
	}
}

func TestOrchestratorDecompose_PhaseNotFound(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s

	rootCmd.SetArgs([]string{"orchestrator-decompose", "--phase", "99"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected nil error for missing phase, got: %v", err)
	}

	if !strings.Contains(errBuf.String(), "phase 99 not found") {
		t.Errorf("expected 'phase 99 not found' in stderr, got: %s", errBuf.String())
	}
}

func TestOrchestratorCommandsRegistered(t *testing.T) {
	commands := []string{
		"orchestrator-decompose",
		"orchestrator-assign",
		"orchestrator-status",
	}

	for _, name := range commands {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("command %q not registered in rootCmd", name)
		}
	}
}
