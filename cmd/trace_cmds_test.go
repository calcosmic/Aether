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

