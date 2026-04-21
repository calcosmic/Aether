package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/trace"
)

func TestTraceReplayFiltersByRunID(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	_ = setupBuildFlowTest(t)

	// Seed trace.jsonl with entries for two runs
	entries := []trace.TraceEntry{
		{RunID: "run_a", Level: trace.TraceLevelState, Topic: "state.transition", Source: "test", Timestamp: "2026-04-21T10:00:00Z"},
		{RunID: "run_b", Level: trace.TraceLevelPhase, Topic: "phase.start", Source: "test", Timestamp: "2026-04-21T10:01:00Z"},
		{RunID: "run_a", Level: trace.TraceLevelError, Topic: "error.add", Source: "test", Timestamp: "2026-04-21T10:02:00Z"},
	}
	for _, e := range entries {
		if err := store.AppendJSONL("trace.jsonl", e); err != nil {
			t.Fatalf("seed trace: %v", err)
		}
	}

	rootCmd.SetArgs([]string{"trace-replay", "--run-id", "run_a"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("trace-replay returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, `"run_id":"run_a"`) {
		t.Errorf("expected run_a in output, got: %s", output)
	}
	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			Entries []trace.TraceEntry `json:"entries"`
			Count   int                `json:"count"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if result.Result.Count != 2 {
		t.Errorf("expected 2 entries for run_a, got %d", result.Result.Count)
	}
}

func TestTraceReplayFiltersByLevel(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	_ = setupBuildFlowTest(t)

	entries := []trace.TraceEntry{
		{RunID: "run_c", Level: trace.TraceLevelState, Topic: "state.transition", Source: "test", Timestamp: "2026-04-21T10:00:00Z"},
		{RunID: "run_c", Level: trace.TraceLevelPhase, Topic: "phase.start", Source: "test", Timestamp: "2026-04-21T10:01:00Z"},
		{RunID: "run_c", Level: trace.TraceLevelError, Topic: "error.add", Source: "test", Timestamp: "2026-04-21T10:02:00Z"},
	}
	for _, e := range entries {
		if err := store.AppendJSONL("trace.jsonl", e); err != nil {
			t.Fatalf("seed trace: %v", err)
		}
	}

	rootCmd.SetArgs([]string{"trace-replay", "--run-id", "run_c", "--level", "state,phase"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("trace-replay returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			Entries []trace.TraceEntry `json:"entries"`
			Count   int                `json:"count"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if result.Result.Count != 2 {
		t.Errorf("expected 2 entries for state+phase filter, got %d", result.Result.Count)
	}
	for _, e := range result.Result.Entries {
		if e.Level != trace.TraceLevelState && e.Level != trace.TraceLevelPhase {
			t.Errorf("unexpected level %q in filtered result", e.Level)
		}
	}
}

func TestTraceExportWritesFile(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)

	entries := []trace.TraceEntry{
		{RunID: "run_d", Level: trace.TraceLevelState, Topic: "state.transition", Source: "test", Timestamp: "2026-04-21T10:00:00Z"},
	}
	for _, e := range entries {
		if err := store.AppendJSONL("trace.jsonl", e); err != nil {
			t.Fatalf("seed trace: %v", err)
		}
	}

	outPath := filepath.Join(dataDir, "export.json")
	rootCmd.SetArgs([]string{"trace-export", "--run-id", "run_d", "--output", outPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("trace-export returned error: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read export file: %v", err)
	}
	var exported []trace.TraceEntry
	if err := json.Unmarshal(data, &exported); err != nil {
		t.Fatalf("unmarshal export: %v", err)
	}
	if len(exported) != 1 {
		t.Errorf("expected 1 exported entry, got %d", len(exported))
	}
	if exported[0].RunID != "run_d" {
		t.Errorf("expected run_d, got %q", exported[0].RunID)
	}
}

func TestTraceEndToEndInitAndStateMutate(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)

	goal := "test goal"
	name := "test-colony"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		ColonyName:   &name,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		RunID:        strPtr("run_e2e_1"),
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: "in_progress"},
			},
		},
	})

	// Ensure tracer is initialized
	tracer = trace.NewTracer(store)

	// Simulate state mutation (READY -> EXECUTING)
	rootCmd.SetArgs([]string{"state-mutate", "--field", "state", "--value", "EXECUTING"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("state-mutate returned error: %v", err)
	}

	// Verify trace.jsonl has the state transition entry
	lines, err := store.ReadJSONL("trace.jsonl")
	if err != nil {
		t.Fatalf("read trace.jsonl: %v", err)
	}
	if len(lines) == 0 {
		t.Fatal("expected trace entries after state mutation")
	}

	var found bool
	for _, line := range lines {
		var entry trace.TraceEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.Topic == "state.transition" && entry.RunID == "run_e2e_1" {
			found = true
			if entry.Payload["from"] != "READY" || entry.Payload["to"] != "EXECUTING" {
				t.Errorf("unexpected payload: %v", entry.Payload)
			}
		}
	}
	if !found {
		t.Error("expected state.transition trace entry for run_e2e_1")
	}
}

func TestTraceEndToEndResumeGeneratesNewRunID(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)

	goal := "test goal"
	name := "test-colony"
	oldRunID := "run_old"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		ColonyName:   &name,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		RunID:        &oldRunID,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: "in_progress"},
			},
		},
	})

	// Write a stale session.json
	session := colony.SessionFile{
		SessionID:      "sess_1",
		StartedAt:      "2026-04-01T00:00:00Z",
		ColonyGoal:     goal,
		CurrentPhase:   1,
		BaselineCommit: "abc123",
	}
	if err := store.SaveJSON("session.json", session); err != nil {
		t.Fatalf("save session: %v", err)
	}

	tracer = trace.NewTracer(store)

	rootCmd.SetArgs([]string{"resume-colony"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("resume-colony returned error: %v", err)
	}

	// Verify trace.jsonl has a resume intervention entry
	lines, err := store.ReadJSONL("trace.jsonl")
	if err != nil {
		t.Fatalf("read trace.jsonl: %v", err)
	}

	var found bool
	for _, line := range lines {
		var entry trace.TraceEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.Topic == "resume.spawn-clear" && entry.Level == trace.TraceLevelIntervention {
			found = true
			if entry.Payload["reason"] != "stale_session" {
				t.Errorf("unexpected reason: %v", entry.Payload["reason"])
			}
		}
	}
	if !found {
		t.Error("expected resume.spawn-clear trace entry after stale session resume")
	}
}

func TestTraceSummaryAndInspect(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	_ = setupBuildFlowTest(t)

	// Seed a rich trace for run_e2e_full
	runID := "run_e2e_full"
	entries := []trace.TraceEntry{
		{RunID: runID, Level: trace.TraceLevelState, Topic: "state.transition", Source: "init", Timestamp: "2026-04-21T10:00:00Z", Payload: map[string]interface{}{"from": "IDLE", "to": "READY"}},
		{RunID: runID, Level: trace.TraceLevelPhase, Topic: "phase.start", Source: "build", Timestamp: "2026-04-21T10:01:00Z", Payload: map[string]interface{}{"phase": float64(1), "status": "in_progress"}},
		{RunID: runID, Level: trace.TraceLevelToken, Topic: "token.usage", Source: "agent-pool", Timestamp: "2026-04-21T10:02:00Z", Payload: map[string]interface{}{"model": "claude-sonnet-4-20250514", "input_tokens": float64(1000), "output_tokens": float64(500), "usd_cost": 0.009}},
		{RunID: runID, Level: trace.TraceLevelError, Topic: "error.add", Source: "build", Timestamp: "2026-04-21T10:03:00Z", Payload: map[string]interface{}{"phase": float64(1), "error_id": "err1", "severity": "warning"}},
		{RunID: runID, Level: trace.TraceLevelArtifact, Topic: "build.worker", Source: "worker", Timestamp: "2026-04-21T10:04:00Z", Payload: map[string]interface{}{"worker": "Builder-1", "status": "completed", "files_modified": 3, "summary": "done"}},
		{RunID: runID, Level: trace.TraceLevelIntervention, Topic: "resume.spawn-clear", Source: "resume-colony", Timestamp: "2026-04-21T10:05:00Z", Payload: map[string]interface{}{"reason": "stale_session"}},
		{RunID: runID, Level: trace.TraceLevelPhase, Topic: "phase.complete", Source: "build", Timestamp: "2026-04-21T10:06:00Z", Payload: map[string]interface{}{"phase": float64(1), "status": "completed"}},
	}
	for _, e := range entries {
		if err := store.AppendJSONL("trace.jsonl", e); err != nil {
			t.Fatalf("seed trace: %v", err)
		}
	}

	// Test trace-summary
	rootCmd.SetArgs([]string{"trace-summary", "--run-id", runID})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("trace-summary returned error: %v", err)
	}
	var summaryResult struct {
		OK     bool                   `json:"ok"`
		Result map[string]interface{} `json:"result"`
	}
	if err := json.Unmarshal([]byte(stdout.(*bytes.Buffer).String()), &summaryResult); err != nil {
		t.Fatalf("unmarshal summary: %v", err)
	}
	if summaryResult.Result["total_entries"] != float64(7) {
		t.Errorf("expected 7 total entries, got %v", summaryResult.Result["total_entries"])
	}
	if summaryResult.Result["phase_count"] != float64(1) {
		t.Errorf("expected 1 phase, got %v", summaryResult.Result["phase_count"])
	}

	tokenUsage, ok := summaryResult.Result["token_usage"].(map[string]interface{})
	if !ok {
		t.Fatal("expected token_usage in summary")
	}
	if tokenUsage["total_input_tokens"] != float64(1000) {
		t.Errorf("expected 1000 input tokens, got %v", tokenUsage["total_input_tokens"])
	}
	if tokenUsage["total_usd_cost"] != float64(0.009) {
		t.Errorf("expected $0.009 cost, got %v", tokenUsage["total_usd_cost"])
	}

	// Test trace-inspect --focus error
	resetRootCmd(t)
	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"trace-inspect", "--run-id", runID, "--focus", "error"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("trace-inspect returned error: %v", err)
	}
	var inspectResult struct {
		OK     bool                   `json:"ok"`
		Result map[string]interface{} `json:"result"`
	}
	if err := json.Unmarshal([]byte(stdout.(*bytes.Buffer).String()), &inspectResult); err != nil {
		t.Fatalf("unmarshal inspect: %v", err)
	}
	if inspectResult.Result["count"] != float64(1) {
		t.Errorf("expected 1 error entry, got %v", inspectResult.Result["count"])
	}

	// Test trace-replay --level state filtering
	resetRootCmd(t)
	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"trace-replay", "--run-id", runID, "--level", "state"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("trace-replay returned error: %v", err)
	}
	var replayResult struct {
		OK     bool `json:"ok"`
		Result struct {
			Count int `json:"count"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(stdout.(*bytes.Buffer).String()), &replayResult); err != nil {
		t.Fatalf("unmarshal replay: %v", err)
	}
	if replayResult.Result.Count != 1 {
		t.Errorf("expected 1 state entry, got %d", replayResult.Result.Count)
	}
}

func TestTraceRotateCommand(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)

	// Create a trace.jsonl larger than 1MB to trigger rotation with max-size-mb=1
	largeContent := strings.Repeat("a", 2*1024*1024)
	if err := os.WriteFile(filepath.Join(dataDir, "trace.jsonl"), []byte(largeContent), 0644); err != nil {
		t.Fatalf("write trace.jsonl: %v", err)
	}

	rootCmd.SetArgs([]string{"trace-rotate", "--max-size-mb", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("trace-rotate returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	var result struct {
		OK     bool                   `json:"ok"`
		Result map[string]interface{} `json:"result"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("unmarshal rotate output: %v", err)
	}
	if !result.Result["rotated"].(bool) {
		t.Errorf("expected rotation to occur, got: %v", result.Result["rotated"])
	}

	// Verify old trace was rotated and new one exists
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		t.Fatalf("read data dir: %v", err)
	}
	var foundRotated bool
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "trace.") && strings.HasSuffix(entry.Name(), ".jsonl") && entry.Name() != "trace.jsonl" {
			foundRotated = true
			break
		}
	}
	if !foundRotated {
		t.Error("expected rotated trace file to exist")
	}
	if _, err := os.Stat(filepath.Join(dataDir, "trace.jsonl")); err != nil {
		t.Errorf("expected new trace.jsonl to exist: %v", err)
	}
}

func TestTraceRotateNoOpWhenUnderLimit(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)

	// Create a tiny trace.jsonl
	if err := os.WriteFile(filepath.Join(dataDir, "trace.jsonl"), []byte("{}\n"), 0644); err != nil {
		t.Fatalf("write trace.jsonl: %v", err)
	}

	rootCmd.SetArgs([]string{"trace-rotate", "--max-size-mb", "50"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("trace-rotate returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	var result struct {
		OK     bool                   `json:"ok"`
		Result map[string]interface{} `json:"result"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("unmarshal rotate output: %v", err)
	}
	if result.Result["rotated"].(bool) {
		t.Errorf("expected no rotation, got: %v", result.Result["rotated"])
	}
}

