package cmd

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/calcosmic/Aether/pkg/trace"
)

func TestAnalyzeTraceDiagnostics_Empty(t *testing.T) {
	diag := analyzeTraceDiagnostics(nil)
	if diag.EntryCount != 0 {
		t.Errorf("expected 0 entries, got %d", diag.EntryCount)
	}
	if diag.HealthScore != 100 {
		t.Errorf("expected health score 100 for empty trace, got %d", diag.HealthScore)
	}
}

func TestAnalyzeTraceDiagnostics_BasicFields(t *testing.T) {
	entries := []trace.TraceEntry{
		{RunID: "run_1", Timestamp: "2026-01-01T00:00:00Z", Level: trace.TraceLevelState, Topic: "state.transition", Payload: map[string]interface{}{"from": "initialized", "to": "planning"}},
		{RunID: "run_1", Timestamp: "2026-01-01T00:01:00Z", Level: trace.TraceLevelState, Topic: "state.transition", Payload: map[string]interface{}{"from": "planning", "to": "building"}},
	}
	diag := analyzeTraceDiagnostics(entries)
	if diag.RunID != "run_1" {
		t.Errorf("expected run_id run_1, got %s", diag.RunID)
	}
	if diag.EntryCount != 2 {
		t.Errorf("expected 2 entries, got %d", diag.EntryCount)
	}
	if diag.Duration != "1m0s" {
		t.Errorf("expected duration 1m0s, got %s", diag.Duration)
	}
}

func TestFindErrorClusters_ThreeOrMore(t *testing.T) {
	entries := []trace.TraceEntry{
		{Level: trace.TraceLevelError, Topic: "error.add", Payload: map[string]interface{}{"phase": float64(1), "severity": "high", "error_id": "e1"}},
		{Level: trace.TraceLevelError, Topic: "error.add", Payload: map[string]interface{}{"phase": float64(1), "severity": "high", "error_id": "e2"}},
		{Level: trace.TraceLevelError, Topic: "error.add", Payload: map[string]interface{}{"phase": float64(1), "severity": "low", "error_id": "e3"}},
	}
	clusters := findErrorClusters(entries)
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}
	if clusters[0].Phase != 1 {
		t.Errorf("expected phase 1, got %d", clusters[0].Phase)
	}
	if clusters[0].Count != 3 {
		t.Errorf("expected 3 errors, got %d", clusters[0].Count)
	}
}

func TestFindErrorClusters_BelowThreshold(t *testing.T) {
	entries := []trace.TraceEntry{
		{Level: trace.TraceLevelError, Topic: "error.add", Payload: map[string]interface{}{"phase": float64(2), "severity": "low"}},
		{Level: trace.TraceLevelError, Topic: "error.add", Payload: map[string]interface{}{"phase": float64(2), "severity": "low"}},
	}
	clusters := findErrorClusters(entries)
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster (below threshold but included), got %d", len(clusters))
	}
	if clusters[0].Count != 2 {
		t.Errorf("expected 2 errors, got %d", clusters[0].Count)
	}
}

func TestFindStateGaps_MissingTransition(t *testing.T) {
	entries := []trace.TraceEntry{
		{Level: trace.TraceLevelState, Topic: "state.transition", Timestamp: "2026-01-01T00:00:00Z", Payload: map[string]interface{}{"from": "initialized", "to": "planning"}},
		// Missing: planning -> building
	}
	gaps := findStateGaps(entries)
	if len(gaps) == 0 {
		t.Error("expected state gaps but found none")
	}
	found := false
	for _, g := range gaps {
		if g.After == "planning" && g.Missing == "building" {
			found = true
		}
	}
	if !found {
		t.Error("expected gap: planning -> building missing")
	}
}

func TestFindStateGaps_CompleteChain(t *testing.T) {
	entries := []trace.TraceEntry{
		{Level: trace.TraceLevelState, Topic: "state.transition", Timestamp: "2026-01-01T00:00:00Z", Payload: map[string]interface{}{"from": "initialized", "to": "planning"}},
		{Level: trace.TraceLevelState, Topic: "state.transition", Timestamp: "2026-01-01T00:01:00Z", Payload: map[string]interface{}{"from": "planning", "to": "building"}},
		{Level: trace.TraceLevelState, Topic: "state.transition", Timestamp: "2026-01-01T00:02:00Z", Payload: map[string]interface{}{"from": "building", "to": "verifying"}},
		{Level: trace.TraceLevelState, Topic: "state.transition", Timestamp: "2026-01-01T00:03:00Z", Payload: map[string]interface{}{"from": "verifying", "to": "active"}},
		{Level: trace.TraceLevelState, Topic: "state.transition", Timestamp: "2026-01-01T00:04:00Z", Payload: map[string]interface{}{"from": "active", "to": "building"}},
		{Level: trace.TraceLevelState, Topic: "state.transition", Timestamp: "2026-01-01T00:05:00Z", Payload: map[string]interface{}{"from": "active", "to": "sealed"}},
	}
	gaps := findStateGaps(entries)
	if len(gaps) != 0 {
		t.Errorf("expected no gaps for complete chain, got %d", len(gaps))
	}
}

func TestFindStalledPhases(t *testing.T) {
	entries := []trace.TraceEntry{
		{Level: trace.TraceLevelPhase, Timestamp: "2026-01-01T00:00:00Z", Payload: map[string]interface{}{"phase": float64(1), "status": "started"}},
		{Level: trace.TraceLevelPhase, Timestamp: "2026-01-01T00:01:00Z", Payload: map[string]interface{}{"phase": float64(1), "status": "completed"}},
		{Level: trace.TraceLevelPhase, Timestamp: "2026-01-01T00:02:00Z", Payload: map[string]interface{}{"phase": float64(2), "status": "started"}},
		// Phase 2 has no further activity — latest overall is at :02, last phase 2 activity is :02
		// We need a later entry to create a stall
		{Level: trace.TraceLevelState, Timestamp: "2026-01-01T01:00:00Z", Topic: "state.transition", Payload: map[string]interface{}{"from": "x", "to": "y"}},
	}
	stalled := findStalledPhases(entries)
	if len(stalled) == 0 {
		t.Error("expected stalled phase 2 but found none")
	}
	found := false
	for _, s := range stalled {
		if s.Phase == 2 {
			found = true
		}
	}
	if !found {
		t.Error("expected phase 2 to be stalled")
	}
}

func TestSumTokens(t *testing.T) {
	entries := []trace.TraceEntry{
		{Level: trace.TraceLevelToken, Payload: map[string]interface{}{"input_tokens": float64(100), "output_tokens": float64(50), "usd_cost": float64(0.05)}},
		{Level: trace.TraceLevelToken, Payload: map[string]interface{}{"input_tokens": float64(200), "output_tokens": float64(100), "usd_cost": float64(0.10)}},
	}
	totals := sumTokens(entries)
	if totals.InputTokens != 300 {
		t.Errorf("expected 300 input tokens, got %d", totals.InputTokens)
	}
	if totals.OutputTokens != 150 {
		t.Errorf("expected 150 output tokens, got %d", totals.OutputTokens)
	}
	if totals.TotalCost < 0.149 || totals.TotalCost > 0.151 {
		t.Errorf("expected ~$0.15 cost, got $%.4f", totals.TotalCost)
	}
	if totals.Entries != 2 {
		t.Errorf("expected 2 token entries, got %d", totals.Entries)
	}
}

func TestGenerateDiagnosticSuggestions(t *testing.T) {
	diag := TraceDiagnostic{
		ErrorGroups: []ErrorCluster{
			{Phase: 3, Count: 5},
		},
		StateGaps: []StateGap{
			{After: "planning", Missing: "building", Message: "gap"},
		},
		Stalled: []StalledPhase{
			{Phase: 2, StallAge: "1h0m0s"},
		},
	}
	suggestions := generateDiagnosticSuggestions(diag)
	if len(suggestions) < 3 {
		t.Errorf("expected at least 3 suggestions, got %d", len(suggestions))
	}
}

func TestComputeHealthScore(t *testing.T) {
	// Perfect score
	perfect := TraceDiagnostic{}
	if computeHealthScore(perfect) != 100 {
		t.Error("expected 100 for clean trace")
	}

	// Deductions
	withErrors := TraceDiagnostic{
		ErrorGroups: []ErrorCluster{{Count: 5}},
	}
	score := computeHealthScore(withErrors)
	if score >= 100 {
		t.Errorf("expected score < 100 with errors, got %d", score)
	}

	// Floor at 0
	terrible := TraceDiagnostic{
		ErrorGroups: []ErrorCluster{{Count: 20}, {Count: 15}, {Count: 10}},
		StateGaps:   []StateGap{{}, {}, {}, {}, {}, {}, {}, {}, {}, {}},
		Stalled:     []StalledPhase{{}, {}, {}, {}, {}},
	}
	score = computeHealthScore(terrible)
	if score != 0 {
		t.Errorf("expected score 0 for terrible trace, got %d", score)
	}
}

func TestRenderTraceDiagnostic(t *testing.T) {
	diag := TraceDiagnostic{
		RunID:       "run_test",
		EntryCount:  10,
		Duration:    "5m0s",
		HealthScore: 85,
		Timeline: []StateTransition{
			{Timestamp: "2026-01-01T00:00:00Z", From: "initialized", To: "planning"},
		},
	}
	output := renderTraceDiagnostic(diag)
	if output == "" {
		t.Error("expected non-empty output")
	}
	if !contains(output, "run_test") {
		t.Error("expected run_id in output")
	}
	if !contains(output, "85/100") {
		t.Error("expected health score in output")
	}
}

func TestRenderTraceDiagnosticJSON(t *testing.T) {
	diag := TraceDiagnostic{
		RunID:       "run_json",
		EntryCount:  5,
		HealthScore: 90,
	}
	output := renderTraceDiagnosticJSON(diag)
	var parsed TraceDiagnostic
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("invalid JSON output: %v", err)
	}
	if parsed.RunID != "run_json" {
		t.Errorf("expected run_id run_json, got %s", parsed.RunID)
	}
}

func TestLoadTraceExport(t *testing.T) {
	entries := []trace.TraceEntry{
		{ID: "t1", RunID: "r1", Level: trace.TraceLevelState, Topic: "test"},
	}
	data, _ := json.MarshalIndent(entries, "", "  ")
	tmpFile, err := os.CreateTemp("", "trace-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write(data)
	tmpFile.Close()

	loaded, err := loadTraceExport(tmpFile.Name())
	if err != nil {
		t.Fatalf("loadTraceExport failed: %v", err)
	}
	if len(loaded) != 1 {
		t.Errorf("expected 1 entry, got %d", len(loaded))
	}
	if loaded[0].RunID != "r1" {
		t.Errorf("expected run_id r1, got %s", loaded[0].RunID)
	}
}

func TestLoadTraceExport_InvalidPath(t *testing.T) {
	_, err := loadTraceExport("/nonexistent/path.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadTraceExport_InvalidJSON(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "trace-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("not json")
	tmpFile.Close()

	_, err = loadTraceExport(tmpFile.Name())
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestUniqueStrings(t *testing.T) {
	input := []string{"a", "b", "a", "c", "b"}
	result := uniqueStrings(input)
	if len(result) != 3 {
		t.Errorf("expected 3 unique strings, got %d", len(result))
	}
}
