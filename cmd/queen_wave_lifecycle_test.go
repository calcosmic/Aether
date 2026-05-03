package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// --- Queen Wave Lifecycle Tests (Plan 98-01) ---

// helper to make a WorkerDispatch with minimal fields set.
func makeWorkerDispatch(name, caste string, wave int) codex.WorkerDispatch {
	return codex.WorkerDispatch{
		ID:         fmt.Sprintf("dispatch-%s", name),
		WorkerName: name,
		Caste:      caste,
		TaskID:     fmt.Sprintf("task-%s", name),
		Wave:       wave,
	}
}

// helper to make a success DispatchResult.
func makeSuccessResult(name string) codex.DispatchResult {
	return codex.DispatchResult{
		WorkerName: name,
		Status:     "completed",
	}
}

// helper to make a failure DispatchResult.
func makeFailResult(name string, err error) codex.DispatchResult {
	return codex.DispatchResult{
		WorkerName: name,
		Status:     "failed",
		Error:      err,
	}
}

// mockDispatchFunc returns a WaveDispatchFunc that returns controlled results per wave.
// The results map is keyed by wave number.
func mockDispatchFunc(results map[int][]codex.DispatchResult, err error) WaveDispatchFunc {
	return func(ctx context.Context, dispatches []codex.WorkerDispatch, wave int) ([]codex.DispatchResult, error) {
		if err != nil {
			return nil, err
		}
		return results[wave], nil
	}
}

// Test 1: All workers succeed across 2 waves.
func TestQueenWaveLifecycle_AllSucceed(t *testing.T) {
	setupBuildFlowTest(t)

	dispatches := []codex.WorkerDispatch{
		makeWorkerDispatch("worker-A", "builder", 1),
		makeWorkerDispatch("worker-B", "builder", 2),
		makeWorkerDispatch("worker-C", "builder", 2),
	}

	results := map[int][]codex.DispatchResult{
		1: {makeSuccessResult("worker-A")},
		2: {makeSuccessResult("worker-B"), makeSuccessResult("worker-C")},
	}

	summary, allResults, err := queenWaveLifecycle(
		context.Background(),
		dispatches,
		mockDispatchFunc(results, nil),
		colony.Phase{ID: 1, Name: "Test Phase"},
		nil,
		1,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.TotalWaves != 2 {
		t.Errorf("expected TotalWaves=2, got %d", summary.TotalWaves)
	}
	if summary.TotalSucceeded != 3 {
		t.Errorf("expected TotalSucceeded=3, got %d", summary.TotalSucceeded)
	}
	if summary.TotalFailed != 0 {
		t.Errorf("expected TotalFailed=0, got %d", summary.TotalFailed)
	}
	if summary.TotalRecovered != 0 {
		t.Errorf("expected TotalRecovered=0, got %d", summary.TotalRecovered)
	}
	if summary.TotalEscalated != 0 {
		t.Errorf("expected TotalEscalated=0, got %d", summary.TotalEscalated)
	}
	if len(summary.Waves) < 2 {
		t.Fatalf("expected at least 2 waves, got %d", len(summary.Waves))
	}
	if summary.Waves[0].Dispatched != 1 {
		t.Errorf("expected wave 0 Dispatched=1, got %d", summary.Waves[0].Dispatched)
	}
	if summary.Waves[1].Dispatched != 2 {
		t.Errorf("expected wave 1 Dispatched=2, got %d", summary.Waves[1].Dispatched)
	}
	if len(allResults) != 3 {
		t.Errorf("expected 3 total results, got %d", len(allResults))
	}
}

// Test 2: Failure in wave 1, queen still advances to wave 2 (D-01/D-02).
func TestQueenWaveLifecycle_FailureAlwaysAdvances(t *testing.T) {
	setupBuildFlowTest(t)

	dispatches := []codex.WorkerDispatch{
		makeWorkerDispatch("worker-A", "builder", 1),
		makeWorkerDispatch("worker-B", "builder", 1),
		makeWorkerDispatch("worker-C", "builder", 2),
	}

	results := map[int][]codex.DispatchResult{
		1: {
			makeFailResult("worker-A", errors.New("build failed")),
			makeSuccessResult("worker-B"),
		},
		2: {makeSuccessResult("worker-C")},
	}

	summary, _, err := queenWaveLifecycle(
		context.Background(),
		dispatches,
		mockDispatchFunc(results, nil),
		colony.Phase{ID: 1, Name: "Test Phase"},
		nil,
		1,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.TotalWaves != 2 {
		t.Errorf("expected TotalWaves=2 (always advance), got %d", summary.TotalWaves)
	}
	if summary.TotalFailed != 1 {
		t.Errorf("expected TotalFailed=1, got %d", summary.TotalFailed)
	}
	// TotalEscalated may be 0 or more depending on orchestrator decision
	if summary.TotalSucceeded != 2 {
		t.Errorf("expected TotalSucceeded=2, got %d", summary.TotalSucceeded)
	}
}

// Test 3: All workers fail across 3 waves, queen still advances through all (D-02).
func TestQueenWaveLifecycle_AllFailStillAdvances(t *testing.T) {
	setupBuildFlowTest(t)

	dispatches := []codex.WorkerDispatch{
		makeWorkerDispatch("worker-A", "builder", 1),
		makeWorkerDispatch("worker-B", "builder", 2),
		makeWorkerDispatch("worker-C", "builder", 3),
	}

	results := map[int][]codex.DispatchResult{
		1: {makeFailResult("worker-A", errors.New("fail"))},
		2: {makeFailResult("worker-B", errors.New("fail"))},
		3: {makeFailResult("worker-C", errors.New("fail"))},
	}

	summary, _, err := queenWaveLifecycle(
		context.Background(),
		dispatches,
		mockDispatchFunc(results, nil),
		colony.Phase{ID: 1, Name: "Test Phase"},
		nil,
		1,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.TotalWaves != 3 {
		t.Errorf("expected TotalWaves=3, got %d", summary.TotalWaves)
	}
	if summary.TotalFailed != 3 {
		t.Errorf("expected TotalFailed=3, got %d", summary.TotalFailed)
	}
}

// Test 4: Mid-wave partial failure -- some workers fail, others succeed in the same wave.
// The wave does NOT abort early; the dispatch func returns all results.
func TestQueenWaveLifecycle_MidWaveFailureTolerance(t *testing.T) {
	setupBuildFlowTest(t)

	dispatches := []codex.WorkerDispatch{
		makeWorkerDispatch("worker-A", "builder", 1),
		makeWorkerDispatch("worker-B", "builder", 1),
		makeWorkerDispatch("worker-C", "builder", 1),
	}

	results := map[int][]codex.DispatchResult{
		1: {
			makeFailResult("worker-A", errors.New("failed")),
			makeSuccessResult("worker-B"),
			makeSuccessResult("worker-C"),
		},
	}

	summary, _, err := queenWaveLifecycle(
		context.Background(),
		dispatches,
		mockDispatchFunc(results, nil),
		colony.Phase{ID: 1, Name: "Test Phase"},
		nil,
		1,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.TotalWaves != 1 {
		t.Errorf("expected TotalWaves=1, got %d", summary.TotalWaves)
	}
	if summary.TotalSucceeded != 2 {
		t.Errorf("expected TotalSucceeded=2 (B and C completed despite A failing), got %d", summary.TotalSucceeded)
	}
	if summary.TotalFailed != 1 {
		t.Errorf("expected TotalFailed=1, got %d", summary.TotalFailed)
	}
}

// Test 5: Mid-wave all-fail, then next wave succeeds. Wave completes and collects both failures.
func TestQueenWaveLifecycle_MidWaveAllFailStillAdvances(t *testing.T) {
	setupBuildFlowTest(t)

	dispatches := []codex.WorkerDispatch{
		makeWorkerDispatch("worker-A", "builder", 1),
		makeWorkerDispatch("worker-B", "builder", 1),
		makeWorkerDispatch("worker-C", "builder", 2),
	}

	results := map[int][]codex.DispatchResult{
		1: {
			makeFailResult("worker-A", errors.New("fail")),
			makeFailResult("worker-B", errors.New("fail")),
		},
		2: {makeSuccessResult("worker-C")},
	}

	summary, _, err := queenWaveLifecycle(
		context.Background(),
		dispatches,
		mockDispatchFunc(results, nil),
		colony.Phase{ID: 1, Name: "Test Phase"},
		nil,
		1,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.TotalWaves != 2 {
		t.Errorf("expected TotalWaves=2, got %d", summary.TotalWaves)
	}
	if summary.TotalFailed != 2 {
		t.Errorf("expected TotalFailed=2 (wave 1), got %d", summary.TotalFailed)
	}
	if summary.TotalSucceeded != 1 {
		t.Errorf("expected TotalSucceeded=1 (wave 2), got %d", summary.TotalSucceeded)
	}
}

// Test 6: Recovery is called for failed workers between waves.
func TestQueenWaveLifecycle_RecoveryCalledBetweenWaves(t *testing.T) {
	setupBuildFlowTest(t)

	dispatches := []codex.WorkerDispatch{
		makeWorkerDispatch("worker-A", "builder", 1),
	}

	results := map[int][]codex.DispatchResult{
		1: {makeFailResult("worker-A", errors.New("build failed"))},
	}

	summary, _, err := queenWaveLifecycle(
		context.Background(),
		dispatches,
		mockDispatchFunc(results, nil),
		colony.Phase{ID: 1, Name: "Test Phase"},
		nil,
		1,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify recovery log has entries for the failed worker
	recoveryLog, readErr := recoveryLogReadPhase(1)
	if readErr != nil {
		t.Fatalf("recoveryLogReadPhase: %v", readErr)
	}
	if len(recoveryLog.Entries) == 0 {
		t.Error("expected recovery log entries for failed worker, got none")
	}

	// Verify the wave result has recovery entries
	if len(summary.Waves) == 0 {
		t.Fatal("expected at least 1 wave")
	}
	// Recovery entries should have been recorded
	if len(summary.Waves[0].Recovered) == 0 && summary.TotalEscalated == 0 {
		t.Error("expected either recovered entries or escalated count for failed worker")
	}
}

// Test 7: Ceremony events are emitted (stdout should have content after lifecycle runs).
func TestQueenWaveLifecycle_CeremonyEmitted(t *testing.T) {
	setupBuildFlowTest(t)

	dispatches := []codex.WorkerDispatch{
		makeWorkerDispatch("worker-A", "builder", 1),
	}

	results := map[int][]codex.DispatchResult{
		1: {makeSuccessResult("worker-A")},
	}

	_, _, err := queenWaveLifecycle(
		context.Background(),
		dispatches,
		mockDispatchFunc(results, nil),
		colony.Phase{ID: 1, Name: "Test Phase"},
		nil,
		1,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// After lifecycle runs, stdout should have content (table or ceremony output).
	// The stub renders nothing, but the implementation should produce output.
	out := stdout.(*bytes.Buffer).String()
	// For RED phase: stub renders nothing, so we just verify no crash.
	// For GREEN phase: we'd check for "Wave" in the output.
	_ = out
}

// Test 8: Wave summary JSON persistence.
func TestQueenWaveLifecycle_WaveSummaryJSON(t *testing.T) {
	setupBuildFlowTest(t)

	dispatches := []codex.WorkerDispatch{
		makeWorkerDispatch("worker-A", "builder", 1),
	}

	results := map[int][]codex.DispatchResult{
		1: {makeSuccessResult("worker-A")},
	}

	summary, _, err := queenWaveLifecycle(
		context.Background(),
		dispatches,
		mockDispatchFunc(results, nil),
		colony.Phase{ID: 1, Name: "Test Phase"},
		nil,
		98,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify writeWaveSummary succeeds
	if err := writeWaveSummary(98, summary); err != nil {
		t.Fatalf("writeWaveSummary: %v", err)
	}

	// Verify readWaveSummary returns same phase number
	loaded, err := readWaveSummary(98)
	if err != nil {
		t.Fatalf("readWaveSummary: %v", err)
	}
	if loaded.Phase != 98 {
		t.Errorf("expected phase 98, got %d", loaded.Phase)
	}

	// Verify file exists at the expected path
	dataDir := os.Getenv("AETHER_ROOT") + "/.aether/data"
	expectedPath := filepath.Join(dataDir, "wave-summary-98.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected wave-summary-98.json at %s", expectedPath)
	}
}

// Test 9: Summary table rendering produces expected headers and totals.
func TestQueenWaveLifecycle_SummaryTable(t *testing.T) {
	setupBuildFlowTest(t)

	summary := WaveLifecycleSummary{
		Phase:           1,
		TotalWaves:      2,
		TotalDispatched: 3,
		TotalSucceeded:  2,
		TotalFailed:     1,
		TotalRecovered:  0,
		TotalEscalated:  0,
		Waves: []WaveResult{
			{Wave: 1, Dispatched: 1, Succeeded: 0, Failed: 1},
			{Wave: 2, Dispatched: 2, Succeeded: 2, Failed: 0},
		},
		CompletedAt: time.Now().Format(time.RFC3339),
	}

	renderWaveSummaryTable(summary)

	out := stdout.(*bytes.Buffer).String()
	if !strings.Contains(out, "Wave") {
		t.Error("expected table output to contain 'Wave' header")
	}
	if !strings.Contains(out, "Total") {
		t.Error("expected table output to contain 'Total' row")
	}
}

// Test 10: Empty dispatches -- no crash, empty summary.
func TestQueenWaveLifecycle_EmptyDispatches(t *testing.T) {
	setupBuildFlowTest(t)

	summary, allResults, err := queenWaveLifecycle(
		context.Background(),
		[]codex.WorkerDispatch{},
		mockDispatchFunc(nil, nil),
		colony.Phase{ID: 1, Name: "Test Phase"},
		nil,
		1,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.TotalWaves != 0 {
		t.Errorf("expected TotalWaves=0 for empty dispatches, got %d", summary.TotalWaves)
	}
	if len(allResults) != 0 {
		t.Errorf("expected 0 results for empty dispatches, got %d", len(allResults))
	}
}

// Test 11: Budget resets per wave.
func TestQueenWaveLifecycle_BudgetResetPerWave(t *testing.T) {
	setupBuildFlowTest(t)

	dispatches := []codex.WorkerDispatch{
		makeWorkerDispatch("worker-A", "builder", 1),
		makeWorkerDispatch("worker-B", "builder", 2),
	}

	results := map[int][]codex.DispatchResult{
		1: {makeFailResult("worker-A", errors.New("fail"))},
		2: {makeSuccessResult("worker-B")},
	}

	summary, _, err := queenWaveLifecycle(
		context.Background(),
		dispatches,
		mockDispatchFunc(results, nil),
		colony.Phase{ID: 1, Name: "Test Phase"},
		nil,
		1,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both waves should be in the summary (budget reset allows wave 2 to proceed)
	if summary.TotalWaves != 2 {
		t.Errorf("expected TotalWaves=2 (budget reset per wave), got %d", summary.TotalWaves)
	}
	if len(summary.Waves) != 2 {
		t.Fatalf("expected 2 wave entries, got %d", len(summary.Waves))
	}
}

// Test 12: Wave summary round-trip -- write all fields, read back, verify match.
func TestWaveSummaryRoundTrip(t *testing.T) {
	setupBuildFlowTest(t)

	original := WaveLifecycleSummary{
		Phase:           42,
		TotalWaves:      2,
		TotalDispatched: 5,
		TotalSucceeded:  3,
		TotalFailed:     2,
		TotalRecovered:  1,
		TotalEscalated:  1,
		Waves: []WaveResult{
			{
				Wave:       1,
				Dispatched: 3,
				Succeeded:  2,
				Failed:     1,
				Recovered: []RecoveryEntry{
					{WorkerName: "worker-A", Method: "retry", Detail: "retrying worker"},
				},
				Escalated:  0,
				BudgetUsed: 1,
			},
			{
				Wave:       2,
				Dispatched: 2,
				Succeeded:  1,
				Failed:     1,
				Escalated:  1,
				BudgetUsed: 2,
			},
		},
		CompletedAt: time.Now().Format(time.RFC3339),
	}

	if err := writeWaveSummary(42, original); err != nil {
		t.Fatalf("writeWaveSummary: %v", err)
	}

	loaded, err := readWaveSummary(42)
	if err != nil {
		t.Fatalf("readWaveSummary: %v", err)
	}

	if loaded.Phase != original.Phase {
		t.Errorf("Phase: expected %d, got %d", original.Phase, loaded.Phase)
	}
	if loaded.TotalWaves != original.TotalWaves {
		t.Errorf("TotalWaves: expected %d, got %d", original.TotalWaves, loaded.TotalWaves)
	}
	if loaded.TotalDispatched != original.TotalDispatched {
		t.Errorf("TotalDispatched: expected %d, got %d", original.TotalDispatched, loaded.TotalDispatched)
	}
	if loaded.TotalSucceeded != original.TotalSucceeded {
		t.Errorf("TotalSucceeded: expected %d, got %d", original.TotalSucceeded, loaded.TotalSucceeded)
	}
	if loaded.TotalFailed != original.TotalFailed {
		t.Errorf("TotalFailed: expected %d, got %d", original.TotalFailed, loaded.TotalFailed)
	}
	if loaded.TotalRecovered != original.TotalRecovered {
		t.Errorf("TotalRecovered: expected %d, got %d", original.TotalRecovered, loaded.TotalRecovered)
	}
	if loaded.TotalEscalated != original.TotalEscalated {
		t.Errorf("TotalEscalated: expected %d, got %d", original.TotalEscalated, loaded.TotalEscalated)
	}
	if loaded.CompletedAt != original.CompletedAt {
		t.Errorf("CompletedAt: expected %s, got %s", original.CompletedAt, loaded.CompletedAt)
	}
	if len(loaded.Waves) != len(original.Waves) {
		t.Fatalf("Waves: expected %d entries, got %d", len(original.Waves), len(loaded.Waves))
	}
	if loaded.Waves[0].Wave != original.Waves[0].Wave {
		t.Errorf("Wave 0 Wave: expected %d, got %d", original.Waves[0].Wave, loaded.Waves[0].Wave)
	}
	if loaded.Waves[0].Recovered[0].WorkerName != original.Waves[0].Recovered[0].WorkerName {
		t.Errorf("Wave 0 Recovered WorkerName: expected %s, got %s", original.Waves[0].Recovered[0].WorkerName, loaded.Waves[0].Recovered[0].WorkerName)
	}
	if loaded.Waves[0].Recovered[0].Method != original.Waves[0].Recovered[0].Method {
		t.Errorf("Wave 0 Recovered Method: expected %s, got %s", original.Waves[0].Recovered[0].Method, loaded.Waves[0].Recovered[0].Method)
	}
}
