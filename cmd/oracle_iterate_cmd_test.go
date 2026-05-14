package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOracleIteratePlanOnlyReturnsManifest(t *testing.T) {
	resetRootCmd(t)
	cleanup := setupOracleTestDir(t)
	defer cleanup()

	out, _ := runCmd(t, []string{"oracle-iterate", "--plan-only", "--topic", "test-topic"})

	var result oracleIterationResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("unmarshal result: %v\noutput: %s", err, out)
	}
	if !result.OK {
		t.Fatalf("expected ok=true, got false")
	}
	m := result.IterationManifest
	if m.Topic != "test-topic" {
		t.Errorf("topic=%q, want test-topic", m.Topic)
	}
	if m.Depth == "" {
		t.Error("depth is empty")
	}
	if m.MaxIterations <= 0 {
		t.Errorf("max_iterations=%d, want >0", m.MaxIterations)
	}
	if m.ConfidenceTarget <= 0 {
		t.Errorf("confidence_target=%d, want >0", m.ConfidenceTarget)
	}
	if len(m.Workers) == 0 {
		t.Error("workers array is empty")
	}
	if m.Workers[0].Caste != "oracle" {
		t.Errorf("worker caste=%q, want oracle", m.Workers[0].Caste)
	}
}

func TestOracleIterateRequiresTopic(t *testing.T) {
	resetRootCmd(t)
	cleanup := setupOracleTestDir(t)
	defer cleanup()

	_, errOut := runCmd(t, []string{"oracle-iterate", "--plan-only"})

	if !strings.Contains(errOut, "--topic is required") {
		t.Errorf("expected error about --topic in stderr, got: %s", errOut)
	}
}

func TestOracleIterateRespectsDepthFlag(t *testing.T) {
	resetRootCmd(t)
	cleanup := setupOracleTestDir(t)
	defer cleanup()

	out, _ := runCmd(t, []string{"oracle-iterate", "--plan-only", "--topic", "x", "--depth", "quick"})

	var result oracleIterationResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, out)
	}
	if result.IterationManifest.MaxIterations != 5 {
		t.Errorf("max_iterations=%d, want 5 for quick depth", result.IterationManifest.MaxIterations)
	}
	if result.IterationManifest.ConfidenceTarget != 60 {
		t.Errorf("confidence_target=%d, want 60 for quick depth", result.IterationManifest.ConfidenceTarget)
	}
}

func TestOracleIterateFinalizeWritesState(t *testing.T) {
	resetRootCmd(t)
	cleanup := setupOracleTestDir(t)
	defer cleanup()

	completion := map[string]any{
		"iteration_manifest": map[string]any{
			"topic":             "finalize-test",
			"depth":             "balanced",
			"max_iterations":    15,
			"confidence_target": 85,
			"current_iteration": 2,
		},
		"dispatches": []map[string]any{
			{"worker": "Oracle-01", "status": "completed", "summary": "Found patterns", "confidence_delta": 10},
		},
		"current_confidence": 70,
		"current_iteration":  2,
		"should_continue":    true,
	}
	compPath := writeTempJSON(t, completion)

	out, _ := runCmd(t, []string{"oracle-iterate-finalize", "--completion-file", compPath})

	var result oracleFinalizeResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, out)
	}
	if !result.OK {
		t.Fatalf("expected ok=true")
	}

	state, err := loadOracleState()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.Topic != "finalize-test" {
		t.Errorf("topic=%q, want finalize-test", state.Topic)
	}
	if state.CurrentIteration != 2 {
		t.Errorf("current_iteration=%d, want 2", state.CurrentIteration)
	}
	if state.CurrentConfidence != 70 {
		t.Errorf("current_confidence=%d, want 70", state.CurrentConfidence)
	}
	if len(state.History) == 0 {
		t.Error("history is empty")
	}
}

func TestOracleIterateFinalizeStopsAtConfidenceTarget(t *testing.T) {
	resetRootCmd(t)
	cleanup := setupOracleTestDir(t)
	defer cleanup()

	completion := map[string]any{
		"iteration_manifest": map[string]any{
			"topic":             "stop-test",
			"depth":             "balanced",
			"max_iterations":    15,
			"confidence_target": 85,
			"current_iteration": 3,
		},
		"dispatches":        []map[string]any{},
		"current_confidence": 90,
		"current_iteration":  3,
		"should_continue":    true,
	}
	compPath := writeTempJSON(t, completion)

	out, _ := runCmd(t, []string{"oracle-iterate-finalize", "--completion-file", compPath})

	var result oracleFinalizeResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, out)
	}
	if result.ShouldContinue {
		t.Error("expected should_continue=false when confidence >= target")
	}
}

func TestOracleIterateFinalizeStopsAtMaxIterations(t *testing.T) {
	resetRootCmd(t)
	cleanup := setupOracleTestDir(t)
	defer cleanup()

	completion := map[string]any{
		"iteration_manifest": map[string]any{
			"topic":             "max-test",
			"depth":             "quick",
			"max_iterations":    5,
			"confidence_target": 60,
			"current_iteration": 5,
		},
		"dispatches":        []map[string]any{},
		"current_confidence": 50,
		"current_iteration":  5,
		"should_continue":    true,
	}
	compPath := writeTempJSON(t, completion)

	out, _ := runCmd(t, []string{"oracle-iterate-finalize", "--completion-file", compPath})

	var result oracleFinalizeResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, out)
	}
	if result.ShouldContinue {
		t.Error("expected should_continue=false when iteration >= max")
	}
}

func TestOracleIterateResumeFromExistingState(t *testing.T) {
	resetRootCmd(t)
	cleanup := setupOracleTestDir(t)
	defer cleanup()

	existing := &oracleState{
		Topic:             "resume-test",
		Depth:             "balanced",
		MaxIterations:     15,
		ConfidenceTarget:  85,
		CurrentIteration:  4,
		CurrentConfidence: 65,
		ShouldContinue:    true,
	}
	if err := saveOracleState(existing); err != nil {
		t.Fatalf("seed state: %v", err)
	}

	out, _ := runCmd(t, []string{"oracle-iterate", "--plan-only", "--topic", "resume-test"})

	var result oracleIterationResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, out)
	}
	if result.IterationManifest.CurrentIteration != 4 {
		t.Errorf("current_iteration=%d, want 4 (resumed)", result.IterationManifest.CurrentIteration)
	}
}

// --- test helpers ---

func setupOracleTestDir(t *testing.T) func() {
	t.Helper()
	tmpDir := t.TempDir()
	oracleDir := filepath.Join(tmpDir, ".aether", "data", "oracle")
	if err := os.MkdirAll(oracleDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	originalWD, _ := os.Getwd()
	os.Chdir(tmpDir)
	return func() {
		os.Chdir(originalWD)
	}
}

// runCmd executes a rootCmd subcommand and returns (stdout, stderr).
func runCmd(t *testing.T, args []string) (string, string) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	oldStdout, oldStderr := stdout, stderr
	stdout, stderr = &outBuf, &errBuf
	defer func() { stdout, stderr = oldStdout, oldStderr }()

	rootCmd.SetArgs(args)
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	_ = rootCmd.Execute()
	return outBuf.String(), errBuf.String()
}

func writeTempJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	f, err := os.CreateTemp("", "oracle-completion-*.json")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	return f.Name()
}
