package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestPhaseJSON(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s
	// Phase 2 is the current phase in testdata
	rootCmd.SetArgs([]string{"phase", "--json"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("phase --json returned error: %v", err)
	}

	output := buf.String()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %s", err, output)
	}
	if envelope["ok"] != true {
		t.Errorf("expected ok=true, got: %v", envelope["ok"])
	}
	result, ok := envelope["result"].(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}
	if result["name"] != "Core Features" {
		t.Errorf("expected name='Core Features', got: %v", result["name"])
	}
	if result["status"] != "in_progress" {
		t.Errorf("expected status='in_progress', got: %v", result["status"])
	}
	tasks, ok := result["tasks"].([]interface{})
	if !ok {
		t.Fatalf("result.tasks is not an array, got: %T", result["tasks"])
	}
	if len(tasks) != 4 {
		t.Errorf("expected 4 tasks, got %d", len(tasks))
	}
}

func TestPhaseJSONNotFound(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s
	rootCmd.SetArgs([]string{"phase", "--number", "99", "--json"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("phase --number 99 --json returned unexpected error: %v", err)
	}

	output := buf.String()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("error output is not valid JSON: %v, got: %s", err, output)
	}
	if envelope["ok"] != false {
		t.Errorf("expected ok=false for not found, got: %v", envelope["ok"])
	}
}

func TestPhaseJSONSpecificNumber(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s
	rootCmd.SetArgs([]string{"phase", "--number", "1", "--json"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("phase --number 1 --json returned error: %v", err)
	}

	output := buf.String()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %s", err, output)
	}
	result := envelope["result"].(map[string]interface{})
	if result["name"] != "Foundation" {
		t.Errorf("expected name='Foundation', got: %v", result["name"])
	}
	if result["status"] != "completed" {
		t.Errorf("expected status='completed', got: %v", result["status"])
	}
}

func TestPhaseCurrentPhase(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s
	rootCmd.SetArgs([]string{"phase"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("phase returned error: %v", err)
	}

	output := buf.String()
	// Current phase is 2 ("Core Features")
	if !strings.Contains(output, "🧱") {
		t.Errorf("expected visual banner, got: %s", output)
	}
	if !strings.Contains(output, "Phase 2") {
		t.Errorf("expected 'Phase 2', got: %s", output)
	}
	if !strings.Contains(output, "Core Features") {
		t.Errorf("expected 'Core Features', got: %s", output)
	}
	if !strings.Contains(output, "in_progress") {
		t.Errorf("expected 'in_progress' status, got: %s", output)
	}
	// Task table should be present
	if !strings.Contains(output, "Implement status command") {
		t.Errorf("expected task 'Implement status command', got: %s", output)
	}
}

func TestPhaseSpecificNumber(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s
	rootCmd.SetArgs([]string{"phase", "--number", "1"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("phase --number 1 returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Phase 1") {
		t.Errorf("expected 'Phase 1', got: %s", output)
	}
	if !strings.Contains(output, "Foundation") {
		t.Errorf("expected 'Foundation', got: %s", output)
	}
	if !strings.Contains(output, "completed") {
		t.Errorf("expected 'completed' status, got: %s", output)
	}
}

func TestPhaseInvalidNumber(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s
	rootCmd.SetArgs([]string{"phase", "--number", "99"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("phase --number 99 returned unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "not found") {
		t.Errorf("expected 'not found' for invalid phase, got: %s", output)
	}
}

func TestPhaseTaskCompletion(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s
	rootCmd.SetArgs([]string{"phase", "--number", "2"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("phase returned error: %v", err)
	}

	output := buf.String()
	// Phase 2 has 2/4 completed (50%)
	if !strings.Contains(output, "2/4") {
		t.Errorf("expected '2/4 tasks', got: %s", output)
	}
	if !strings.Contains(output, "50%") {
		t.Errorf("expected '50%%' completion, got: %s", output)
	}
}
