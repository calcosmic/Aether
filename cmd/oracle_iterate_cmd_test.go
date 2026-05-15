package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/storage"
)

func TestOracleIteratePlanOnlyReturnsManifest(t *testing.T) {
	resetRootCmd(t)
	cleanup := setupOracleTestDir(t)
	defer cleanup()

	out, _ := runCmd(t, []string{"oracle-iterate", "--plan-only", "--topic", "test-topic"})

	var envelope struct {
		OK     bool                `json:"ok"`
		Result oracleIterationResult `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("unmarshal envelope: %v\noutput: %s", err, out)
	}
	if !envelope.OK {
		t.Fatalf("expected ok=true, got false")
	}
	result := envelope.Result
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

	var envelope struct {
		OK     bool                `json:"ok"`
		Result oracleIterationResult `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, out)
	}
	result := envelope.Result
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

	var envelope struct {
		OK     bool               `json:"ok"`
		Result oracleFinalizeResult `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, out)
	}
	if !envelope.OK {
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

	var envelope struct {
		OK     bool               `json:"ok"`
		Result oracleFinalizeResult `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, out)
	}
	result := envelope.Result
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

	var envelope struct {
		OK     bool               `json:"ok"`
		Result oracleFinalizeResult `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, out)
	}
	result := envelope.Result
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

	var envelope struct {
		OK     bool                `json:"ok"`
		Result oracleIterationResult `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, out)
	}
	result := envelope.Result
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
	// Reset the global store so that direct saveOracleState/loadOracleState calls
	// fall back to plain file I/O instead of using a stale store from a prior test.
	origStore := store
	store = nil
	return func() {
		os.Chdir(originalWD)
		store = origStore
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

// --- Atomic storage contract tests (Task 2) ---

func TestOracleStateConcurrentWrites(t *testing.T) {
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)

	// Seed initial state
	state := &oracleState{
		Topic:            "concurrent-test",
		CurrentIteration: 1,
		ShouldContinue:   true,
	}
	if err := s.SaveJSON("oracle/state.json", state); err != nil {
		t.Fatalf("seed state: %v", err)
	}

	// Spawn 10 goroutines, each saving a different iteration number
	done := make(chan error, 10)
	for i := 1; i <= 10; i++ {
		go func(iter int) {
			st := &oracleState{
				Topic:            "concurrent-test",
				CurrentIteration: iter,
				ShouldContinue:   true,
			}
			done <- s.SaveJSON("oracle/state.json", st)
		}(i)
	}
	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			t.Fatalf("concurrent write error: %v", err)
		}
	}

	// Verify loaded state is valid JSON (not corrupted/mixed)
	data, err := s.ReadFile("oracle/state.json")
	if err != nil {
		t.Fatalf("read after concurrent writes: %v", err)
	}
	var loaded oracleState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("state corrupted after concurrent writes: %v", err)
	}
	if loaded.Topic != "concurrent-test" {
		t.Errorf("topic = %q, want concurrent-test", loaded.Topic)
	}
}

func TestOracleStateReadDuringWrite(t *testing.T) {
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)

	state := &oracleState{
		Topic:            "read-during-write",
		CurrentIteration: 1,
		ShouldContinue:   true,
	}
	if err := s.SaveJSON("oracle/state.json", state); err != nil {
		t.Fatalf("seed state: %v", err)
	}

	writeDone := make(chan struct{})
	readDone := make(chan struct{})
	readErrs := make(chan error, 50)

	// Writer loop
	go func() {
		defer close(writeDone)
		for i := 1; i <= 50; i++ {
			st := &oracleState{
				Topic:            "read-during-write",
				CurrentIteration: i,
				ShouldContinue:   true,
			}
			if err := s.SaveJSON("oracle/state.json", st); err != nil {
				t.Errorf("write error: %v", err)
				return
			}
		}
	}()

	// Reader loop
	go func() {
		defer close(readDone)
		for i := 0; i < 50; i++ {
			data, err := s.ReadFile("oracle/state.json")
			if err != nil {
				readErrs <- fmt.Errorf("read error: %w", err)
				continue
			}
			var loaded oracleState
			if err := json.Unmarshal(data, &loaded); err != nil {
				readErrs <- fmt.Errorf("corrupted read: %w", err)
			}
		}
	}()

	<-writeDone
	<-readDone
	close(readErrs)

	for err := range readErrs {
		t.Fatalf("read-during-write failure: %v", err)
	}
}

func TestOracleStateStoreInitCreatesDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".aether", "data")

	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	state := &oracleState{
		Topic:            "init-test",
		CurrentIteration: 1,
		ShouldContinue:   true,
	}
	if err := s.SaveJSON("oracle/state.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	// Verify directory was created
	statePath := filepath.Join(dataDir, "oracle", "state.json")
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("state file not created at expected path: %v", err)
	}
}

func TestOracleStateMissingReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".aether", "data")

	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	_, err = s.ReadFile("oracle/state.json")
	if err == nil {
		t.Fatal("expected error for missing state file, got nil")
	}
}

func TestOracleStateInterruptRecovery(t *testing.T) {
	resetRootCmd(t)
	cleanup := setupOracleTestDir(t)
	defer cleanup()

	// Seed state with a pending iteration
	existing := &oracleState{
		Topic:             "interrupt-test",
		Depth:             "balanced",
		MaxIterations:     15,
		ConfidenceTarget:  85,
		CurrentIteration:  3,
		CurrentConfidence: 65,
		ShouldContinue:    true,
		PendingIteration:  3,
		PendingStartTime:  time.Now().UTC().Format(time.RFC3339),
		LastWorkerStatus:  "dispatched",
	}
	if err := saveOracleState(existing); err != nil {
		t.Fatalf("seed state: %v", err)
	}

	out, _ := runCmd(t, []string{"oracle-iterate", "--plan-only", "--topic", "interrupt-test"})

	var envelope struct {
		OK     bool                  `json:"ok"`
		Result oracleIterationResult `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, out)
	}
	if !envelope.OK {
		t.Fatalf("expected ok=true")
	}
	if !envelope.Result.IterationManifest.Resuming {
		t.Error("expected resuming=true when pending iteration matches current")
	}

	// Verify finalize clears the pending marker
	completion := map[string]any{
		"iteration_manifest": map[string]any{
			"topic":             "interrupt-test",
			"depth":             "balanced",
			"max_iterations":    15,
			"confidence_target": 85,
			"current_iteration": 3,
		},
		"dispatches":        []map[string]any{},
		"current_confidence": 70,
		"current_iteration":  3,
		"should_continue":    true,
	}
	compPath := writeTempJSON(t, completion)
	_, _ = runCmd(t, []string{"oracle-iterate-finalize", "--completion-file", compPath})

	state, err := loadOracleState()
	if err != nil {
		t.Fatalf("load state after finalize: %v", err)
	}
	if state.PendingIteration != 0 {
		t.Errorf("pending_iteration=%d, want 0 after finalize", state.PendingIteration)
	}
	if state.PendingStartTime != "" {
		t.Error("pending_start_time should be empty after finalize")
	}
}

func TestOracleStateStalePendingAutoClears(t *testing.T) {
	resetRootCmd(t)
	cleanup := setupOracleTestDir(t)
	defer cleanup()

	// Seed state with a stale pending iteration (>1 hour old)
	existing := &oracleState{
		Topic:             "stale-test",
		Depth:             "balanced",
		MaxIterations:     15,
		ConfidenceTarget:  85,
		CurrentIteration:  3,
		CurrentConfidence: 65,
		ShouldContinue:    true,
		PendingIteration:  3,
		PendingStartTime:  time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
		LastWorkerStatus:  "dispatched",
	}
	if err := saveOracleState(existing); err != nil {
		t.Fatalf("seed state: %v", err)
	}

	out, _ := runCmd(t, []string{"oracle-iterate", "--plan-only", "--topic", "stale-test"})

	var envelope struct {
		OK     bool                  `json:"ok"`
		Result oracleIterationResult `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, out)
	}
	if !envelope.OK {
		t.Fatalf("expected ok=true")
	}
	if envelope.Result.IterationManifest.Resuming {
		t.Error("expected resuming=false when pending is stale")
	}

	// Verify stale marker was replaced with a fresh one (not left as stale)
	state, err := loadOracleState()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.PendingIteration != 3 {
		t.Errorf("pending_iteration=%d, want 3 (fresh marker written)", state.PendingIteration)
	}
	// The new pending start time should be recent, not 2 hours old
	if state.PendingStartTime == "" {
		t.Fatal("pending_start_time should not be empty after plan-only")
	}
	startTime, _ := time.Parse(time.RFC3339, state.PendingStartTime)
	if time.Since(startTime) > 5*time.Minute {
		t.Error("pending_start_time should be recent after stale clear + refresh")
	}
}
