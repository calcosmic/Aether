package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// --- RecoveryBudget tests ---

func TestRecoveryBudget_ConsumeRetry(t *testing.T) {
	budget := newRecoveryBudget(1)
	if !budget.consume("retry") {
		t.Error("first consume should succeed")
	}
	if budget.RetriesUsed != 1 {
		t.Errorf("expected retries_used=1, got %d", budget.RetriesUsed)
	}
}

func TestRecoveryBudget_ConsumeExhaustion(t *testing.T) {
	budget := newRecoveryBudget(1)
	budget.TotalBudget = 1 // Override to test exhaustion
	if !budget.consume("retry") {
		t.Error("first consume should succeed")
	}
	if budget.consume("retry") {
		t.Error("second consume should fail -- budget exhausted")
	}
}

func TestRecoveryBudget_WaveReset(t *testing.T) {
	budget := newRecoveryBudget(1)
	budget.consume("retry")
	budget.consume("peer_reassignment")

	budget.resetForWave(2)

	if budget.Wave != 2 {
		t.Errorf("expected wave=2, got %d", budget.Wave)
	}
	if budget.RetriesUsed != 0 {
		t.Errorf("expected retries_used=0 after reset, got %d", budget.RetriesUsed)
	}
	if budget.ReassignsUsed != 0 {
		t.Errorf("expected reassigns_used=0 after reset, got %d", budget.ReassignsUsed)
	}
	if budget.TotalBudget != 3 {
		t.Errorf("expected total_budget=3 after reset, got %d", budget.TotalBudget)
	}
}

func TestRecoveryBudget_Persistence(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	budget := newRecoveryBudget(1)
	budget.consume("retry")
	budget.consume("peer_reassignment")

	// Persist
	if err := persistBudgetToRecoveryLog(1, budget); err != nil {
		t.Fatalf("persistBudgetToRecoveryLog: %v", err)
	}

	// Read back
	loaded := budgetFromRecoveryLog(1, 1)
	if loaded == nil {
		t.Fatal("budgetFromRecoveryLog returned nil")
	}
	if loaded.RetriesUsed != 1 {
		t.Errorf("expected retries_used=1, got %d", loaded.RetriesUsed)
	}
	if loaded.ReassignsUsed != 1 {
		t.Errorf("expected reassigns_used=1, got %d", loaded.ReassignsUsed)
	}
	if loaded.Wave != 1 {
		t.Errorf("expected wave=1, got %d", loaded.Wave)
	}
}

// --- orchestrateRecovery tests ---

func TestOrchestrateRecovery_RecoverableSequence(t *testing.T) {
	cb := NewCircuitBreaker(3)
	budget := newRecoveryBudget(1)

	ctx := RecoveryContext{
		Phase:        1,
		Wave:         1,
		WorkerName:   "Builder-1",
		TaskID:       "task-1",
		Caste:        "builder",
		Status:       "timeout",
		ErrorMessage: "worker timed out after 300s",
		Dispatches: []codex.WorkerDispatch{
			{WorkerName: "Builder-1", Caste: "builder", TaskID: "task-1"},
			{WorkerName: "Builder-2", Caste: "builder", TaskID: "task-2"},
		},
		CircuitBreaker:  cb,
		Budget:          budget,
		RecoveryHistory: nil,
	}

	// First call: should return retry
	outcome := orchestrateRecovery(ctx)
	if outcome.Action.Type != "retry" {
		t.Errorf("expected first action 'retry', got %q", outcome.Action.Type)
	}
	if outcome.Classification != Recoverable {
		t.Errorf("expected classification 'recoverable', got %q", outcome.Classification)
	}
	if outcome.Exhausted {
		t.Error("should not be exhausted after first action")
	}

	// Second call: retry done, should return peer_reassignment
	ctx.RecoveryHistory = []RecoveryAction{outcome.Action}
	outcome = orchestrateRecovery(ctx)
	if outcome.Action.Type != "peer_reassignment" {
		t.Errorf("expected second action 'peer_reassignment', got %q", outcome.Action.Type)
	}
	if outcome.Action.PeerName != "Builder-2" {
		t.Errorf("expected peer 'Builder-2', got %q", outcome.Action.PeerName)
	}

	// Third call: peer done, should return fixer_dispatch
	ctx.RecoveryHistory = append(ctx.RecoveryHistory, outcome.Action)
	outcome = orchestrateRecovery(ctx)
	if outcome.Action.Type != "fixer_dispatch" {
		t.Errorf("expected third action 'fixer_dispatch', got %q", outcome.Action.Type)
	}

	// Fourth call: fixer done, should return escalate
	ctx.RecoveryHistory = append(ctx.RecoveryHistory, outcome.Action)
	outcome = orchestrateRecovery(ctx)
	if outcome.Action.Type != "escalate" {
		t.Errorf("expected fourth action 'escalate', got %q", outcome.Action.Type)
	}
	if !outcome.Exhausted {
		t.Error("should be exhausted after escalate")
	}
}

func TestOrchestrateRecovery_BlockingEscalatesImmediately(t *testing.T) {
	cb := NewCircuitBreaker(3)
	budget := newRecoveryBudget(1)

	ctx := RecoveryContext{
		Phase:          1,
		Wave:           1,
		WorkerName:     "Builder-1",
		TaskID:         "task-1",
		Caste:          "builder",
		Status:         "bad_task_spec",
		ErrorMessage:   "invalid task specification",
		Dispatches:     []codex.WorkerDispatch{},
		CircuitBreaker: cb,
		Budget:         budget,
	}

	outcome := orchestrateRecovery(ctx)
	if outcome.Classification != Blocking {
		t.Errorf("expected classification 'blocking', got %q", outcome.Classification)
	}
	if outcome.Action.Type != "escalate" {
		t.Errorf("expected action 'escalate', got %q", outcome.Action.Type)
	}
	if !outcome.Exhausted {
		t.Error("blocking failure should be exhausted immediately")
	}
	// No budget consumed for blocking
	if budget.RetriesUsed != 0 {
		t.Errorf("blocking should not consume budget, got retries_used=%d", budget.RetriesUsed)
	}
}

func TestOrchestrateRecovery_RequiresAttemptSequence(t *testing.T) {
	cb := NewCircuitBreaker(3)
	budget := newRecoveryBudget(1)

	ctx := RecoveryContext{
		Phase:        1,
		Wave:         1,
		WorkerName:   "Builder-1",
		TaskID:       "task-1",
		Caste:        "builder",
		Status:       "failed",
		ErrorMessage: "generic failure",
		Dispatches: []codex.WorkerDispatch{
			{WorkerName: "Builder-1", Caste: "builder", TaskID: "task-1"},
			{WorkerName: "Builder-2", Caste: "builder", TaskID: "task-2"},
		},
		CircuitBreaker:  cb,
		Budget:          budget,
		RecoveryHistory: nil,
	}

	// First call: should return retry (no peer for requires-attempt)
	outcome := orchestrateRecovery(ctx)
	if outcome.Classification != RequiresAttempt {
		t.Errorf("expected classification 'requires-attempt', got %q", outcome.Classification)
	}
	if outcome.Action.Type != "retry" {
		t.Errorf("expected first action 'retry', got %q", outcome.Action.Type)
	}

	// Second call: retry done, should return fixer_dispatch (no peer for requires-attempt)
	ctx.RecoveryHistory = []RecoveryAction{outcome.Action}
	outcome = orchestrateRecovery(ctx)
	if outcome.Action.Type != "fixer_dispatch" {
		t.Errorf("expected second action 'fixer_dispatch', got %q", outcome.Action.Type)
	}

	// Third call: fixer done, should escalate
	ctx.RecoveryHistory = append(ctx.RecoveryHistory, outcome.Action)
	outcome = orchestrateRecovery(ctx)
	if outcome.Action.Type != "escalate" {
		t.Errorf("expected third action 'escalate', got %q", outcome.Action.Type)
	}
	if !outcome.Exhausted {
		t.Error("should be exhausted after escalate")
	}
}

func TestOrchestrateRecovery_BudgetExhaustion(t *testing.T) {
	cb := NewCircuitBreaker(3)
	budget := newRecoveryBudget(1)
	// Exhaust the budget
	budget.consume("retry")
	budget.consume("retry")
	budget.consume("retry")

	ctx := RecoveryContext{
		Phase:          1,
		Wave:           1,
		WorkerName:     "Builder-1",
		TaskID:         "task-1",
		Caste:          "builder",
		Status:         "timeout",
		ErrorMessage:   "worker timed out",
		Dispatches:     []codex.WorkerDispatch{},
		CircuitBreaker: cb,
		Budget:         budget,
	}

	outcome := orchestrateRecovery(ctx)
	if outcome.Action.Type != "escalate" {
		t.Errorf("expected 'escalate' when budget exhausted, got %q", outcome.Action.Type)
	}
	if !outcome.Exhausted {
		t.Error("should be exhausted when budget is consumed")
	}
}

func TestOrchestrateRecovery_CircuitBreakerBypassesRetry(t *testing.T) {
	cb := NewCircuitBreaker(1)
	// Trip the breaker for Builder-1
	cb.RecordFailure("Builder-1")

	budget := newRecoveryBudget(1)

	ctx := RecoveryContext{
		Phase:        1,
		Wave:         1,
		WorkerName:   "Builder-1",
		TaskID:       "task-1",
		Caste:        "builder",
		Status:       "timeout",
		ErrorMessage: "worker timed out",
		Dispatches: []codex.WorkerDispatch{
			{WorkerName: "Builder-1", Caste: "builder", TaskID: "task-1"},
			{WorkerName: "Builder-2", Caste: "builder", TaskID: "task-2"},
		},
		CircuitBreaker:  cb,
		Budget:          budget,
		RecoveryHistory: nil,
	}

	outcome := orchestrateRecovery(ctx)
	// Should skip retry and go to peer_reassignment
	if outcome.Action.Type != "peer_reassignment" {
		t.Errorf("expected 'peer_reassignment' when breaker tripped, got %q", outcome.Action.Type)
	}
	if outcome.Action.PeerName != "Builder-2" {
		t.Errorf("expected peer 'Builder-2', got %q", outcome.Action.PeerName)
	}
}

func TestOrchestrateRecovery_RecoveryLogEntries(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	cb := NewCircuitBreaker(3)
	budget := newRecoveryBudget(1)

	ctx := RecoveryContext{
		Phase:          1,
		Wave:           1,
		WorkerName:     "Builder-1",
		TaskID:         "task-1",
		Caste:          "builder",
		Status:         "timeout",
		ErrorMessage:   "worker timed out",
		Dispatches:     []codex.WorkerDispatch{},
		CircuitBreaker: cb,
		Budget:         budget,
	}

	outcome := orchestrateRecovery(ctx)
	if len(outcome.LogEntries) == 0 {
		t.Error("expected at least one recovery log entry")
	}

	entry := outcome.LogEntries[0]
	if entry.ActionTaken != "retry" {
		t.Errorf("expected action_taken='retry', got %q", entry.ActionTaken)
	}
	if entry.Failure.WorkerName != "Builder-1" {
		t.Errorf("expected worker='Builder-1', got %q", entry.Failure.WorkerName)
	}
	if entry.Failure.Status != "timeout" {
		t.Errorf("expected status='timeout', got %q", entry.Failure.Status)
	}
}

func TestOrchestrateRecovery_FixerContext(t *testing.T) {
	cb := NewCircuitBreaker(3)
	budget := newRecoveryBudget(1)

	// Simulate retry and peer already attempted
	retryAction := RecoveryAction{
		Type:       "retry",
		WorkerName: "Builder-1",
		Detail:     "retry failed: timeout",
	}
	peerAction := RecoveryAction{
		Type:       "peer_reassignment",
		WorkerName: "Builder-1",
		PeerName:   "Builder-2",
		Detail:     "peer also failed: timeout",
	}

	ctx := RecoveryContext{
		Phase:           1,
		Wave:            1,
		WorkerName:      "Builder-1",
		TaskID:          "task-1",
		Caste:           "builder",
		Status:          "timeout",
		ErrorMessage:    "worker timed out",
		Dispatches:      []codex.WorkerDispatch{},
		CircuitBreaker:  cb,
		Budget:          budget,
		RecoveryHistory: []RecoveryAction{retryAction, peerAction},
	}

	outcome := orchestrateRecovery(ctx)
	if outcome.Action.Type != "fixer_dispatch" {
		t.Fatalf("expected 'fixer_dispatch', got %q", outcome.Action.Type)
	}
	// Detail should include recovery history context
	if !strings.Contains(outcome.Action.Detail, "retry") {
		t.Errorf("fixer dispatch detail should mention retry history, got: %q", outcome.Action.Detail)
	}
	if !strings.Contains(outcome.Action.Detail, "peer reassignment") {
		t.Errorf("fixer dispatch detail should mention peer reassignment history, got: %q", outcome.Action.Detail)
	}
}

func TestOrchestrateRecovery_BudgetRemaining(t *testing.T) {
	cb := NewCircuitBreaker(3)
	budget := newRecoveryBudget(1)

	ctx := RecoveryContext{
		Phase:          1,
		Wave:           1,
		WorkerName:     "Builder-1",
		TaskID:         "task-1",
		Caste:          "builder",
		Status:         "timeout",
		ErrorMessage:   "worker timed out",
		Dispatches:     []codex.WorkerDispatch{},
		CircuitBreaker: cb,
		Budget:         budget,
	}

	outcome := orchestrateRecovery(ctx)
	if outcome.Action.BudgetRemaining != 2 {
		t.Errorf("expected budget_remaining=2 after first consume, got %d", outcome.Action.BudgetRemaining)
	}
}

func TestBudgetFromRecoveryLog_NoExistingFile(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	budget := budgetFromRecoveryLog(99, 1)
	if budget == nil {
		t.Fatal("expected non-nil budget for missing file")
	}
	if budget.Wave != 1 {
		t.Errorf("expected wave=1, got %d", budget.Wave)
	}
	if budget.TotalBudget != 3 {
		t.Errorf("expected total_budget=3, got %d", budget.TotalBudget)
	}
}

// --- Task 1: Build finalize recovery wiring tests ---

func TestFilterFailedDispatches(t *testing.T) {
	dispatches := []codexBuildDispatch{
		{Name: "Builder-1", Caste: "builder", Status: "completed", TaskID: "task-1"},
		{Name: "Builder-2", Caste: "builder", Status: "timeout", TaskID: "task-2", Summary: "timed out"},
		{Name: "Builder-3", Caste: "builder", Status: "failed", TaskID: "task-3", Summary: "generic failure"},
		{Name: "Watcher-1", Caste: "watcher", Status: "completed", TaskID: "task-4"},
	}

	failed := filterFailedDispatches(dispatches)
	if len(failed) != 2 {
		t.Fatalf("expected 2 failed dispatches, got %d", len(failed))
	}
	if failed[0].Name != "Builder-2" {
		t.Errorf("expected first failed dispatch 'Builder-2', got %q", failed[0].Name)
	}
	if failed[1].Name != "Builder-3" {
		t.Errorf("expected second failed dispatch 'Builder-3', got %q", failed[1].Name)
	}
}

func TestFilterFailedDispatches_AllCompleted(t *testing.T) {
	dispatches := []codexBuildDispatch{
		{Name: "Builder-1", Caste: "builder", Status: "completed", TaskID: "task-1"},
		{Name: "Builder-2", Caste: "builder", Status: "completed", TaskID: "task-2"},
	}

	failed := filterFailedDispatches(dispatches)
	if len(failed) != 0 {
		t.Errorf("expected 0 failed dispatches, got %d", len(failed))
	}
}

func TestEffectiveWave(t *testing.T) {
	tests := []struct {
		name       string
		dispatches []codexBuildDispatch
		want       int
	}{
		{
			name: "wave from dispatches",
			dispatches: []codexBuildDispatch{
				{Name: "B-1", Wave: 2},
				{Name: "B-2", Wave: 2},
			},
			want: 2,
		},
		{
			name: "zero waves default to 1",
			dispatches: []codexBuildDispatch{
				{Name: "B-1", Wave: 0},
				{Name: "B-2", Wave: 0},
			},
			want: 1,
		},
		{
			name:       "empty dispatches default to 1",
			dispatches: []codexBuildDispatch{},
			want:       1,
		},
		{
			name: "mixed waves picks first non-zero",
			dispatches: []codexBuildDispatch{
				{Name: "B-1", Wave: 0},
				{Name: "B-2", Wave: 3},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := effectiveWave(tt.dispatches)
			if got != tt.want {
				t.Errorf("effectiveWave() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestBuildToWorkerDispatches(t *testing.T) {
	dispatches := []codexBuildDispatch{
		{Name: "Builder-1", Caste: "builder", TaskID: "task-1"},
		{Name: "Watcher-1", Caste: "watcher", TaskID: "task-2"},
	}

	result := buildToWorkerDispatches(dispatches)
	if len(result) != 2 {
		t.Fatalf("expected 2 worker dispatches, got %d", len(result))
	}
	if result[0].WorkerName != "Builder-1" {
		t.Errorf("expected WorkerName 'Builder-1', got %q", result[0].WorkerName)
	}
	if result[0].Caste != "builder" {
		t.Errorf("expected Caste 'builder', got %q", result[0].Caste)
	}
	if result[0].TaskID != "task-1" {
		t.Errorf("expected TaskID 'task-1', got %q", result[0].TaskID)
	}
	if result[1].WorkerName != "Watcher-1" {
		t.Errorf("expected WorkerName 'Watcher-1', got %q", result[1].WorkerName)
	}
}

func TestBuildFinalize_RecoveryForFailedDispatch(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	// Set up circuit breaker
	origCB := globalCircuitBreaker
	cb := NewCircuitBreaker(3)
	globalCircuitBreaker = cb
	defer func() { globalCircuitBreaker = origCB }()

	// Create a colony state with a plan phase
	state := colony.ColonyState{
		Goal:         strPtr("Test goal"),
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Test Phase", Status: colony.PhaseInProgress, Tasks: []colony.Task{
					{ID: strPtr("task-1"), Status: colony.TaskInProgress},
					{ID: strPtr("task-2"), Status: colony.TaskInProgress},
				}},
			},
		},
	}
	createTestColonyState(t, dataDir, state)

	// Create completion with one failed (timeout) dispatch
	manifest := codexBuildManifest{
		Phase:    1,
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{
			{Name: "Builder-1", Caste: "builder", TaskID: "task-1", Task: "Build feature", Status: "pending", Wave: 1},
			{Name: "Builder-2", Caste: "builder", TaskID: "task-2", Task: "Build feature 2", Status: "pending", Wave: 1},
		},
		SelectedTasks: []string{"task-1", "task-2"},
	}
	completion := codexExternalBuildCompletion{
		DispatchManifest: &manifest,
		Results: []codexExternalBuildWorkerResult{
			{Name: "Builder-1", Status: "timeout", Summary: "worker timed out after 300s", Handoff: codex.WorkerHandoff{Freshness: "not-run"}},
			{Name: "Builder-2", Status: "completed", Summary: "done", FilesModified: []string{"cmd/test.go"}, Handoff: codex.WorkerHandoff{Freshness: "not-run"}},
		},
	}

	completionPath := filepath.Join(dataDir, "completion.json")
	raw, _ := json.Marshal(completion)
	if err := os.WriteFile(completionPath, raw, 0644); err != nil {
		t.Fatalf("failed to write completion file: %v", err)
	}

	result, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("runCodexBuildFinalize failed: %v", err)
	}

	recoveryRaw, ok := result["recovery_instructions"]
	if !ok {
		t.Fatal("expected 'recovery_instructions' in result for failed dispatch")
	}
	recoveryInstructions, ok := recoveryRaw.([]map[string]interface{})
	if !ok {
		t.Fatalf("expected recovery_instructions to be []map[string]interface{}, got %T", recoveryRaw)
	}
	if len(recoveryInstructions) != 1 {
		t.Fatalf("expected 1 recovery instruction, got %d", len(recoveryInstructions))
	}
	instruction := recoveryInstructions[0]
	if instruction["worker"] != "Builder-1" {
		t.Errorf("expected worker 'Builder-1', got %v", instruction["worker"])
	}
	// timeout is Recoverable, so first action is "retry"
	if instruction["action"] != "retry" {
		t.Errorf("expected action 'retry', got %v", instruction["action"])
	}
}

func TestBuildFinalize_RecoveryForBlockingDispatch(t *testing.T) {
	// "blocked" status is terminal in build flow and maps to RequiresAttempt in the classifier.
	// To test blocking escalation, we use a failed dispatch and verify the orchestrator returns
	// a recovery action (retry for RequiresAttempt). The blocking classification itself is tested
	// in TestOrchestrateRecovery_BlockingEscalatesImmediately with direct orchestrator calls.
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	origCB := globalCircuitBreaker
	cb := NewCircuitBreaker(3)
	globalCircuitBreaker = cb
	defer func() { globalCircuitBreaker = origCB }()

	state := colony.ColonyState{
		Goal:         strPtr("Test goal"),
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Test Phase", Status: colony.PhaseInProgress, Tasks: []colony.Task{
					{ID: strPtr("task-1"), Status: colony.TaskInProgress},
					{ID: strPtr("task-2"), Status: colony.TaskInProgress},
				}},
			},
		},
	}
	createTestColonyState(t, dataDir, state)

	// Blocked dispatch needs at least one completed worker to pass provenance validation.
	// Provenance (SAFE-01, SAFE-02) rejects builds where no worker completed with file modifications.
	manifest := codexBuildManifest{
		Phase:    1,
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{
			{Name: "Builder-1", Caste: "builder", TaskID: "task-1", Task: "Build feature", Status: "pending", Wave: 1},
			{Name: "Builder-2", Caste: "builder", TaskID: "task-2", Task: "Build feature 2", Status: "pending", Wave: 1},
		},
		SelectedTasks: []string{"task-1", "task-2"},
	}
	completion := codexExternalBuildCompletion{
		DispatchManifest: &manifest,
		Results: []codexExternalBuildWorkerResult{
			{Name: "Builder-1", Status: "blocked", Summary: "worker was blocked", Handoff: codex.WorkerHandoff{Freshness: "not-run"}},
			{Name: "Builder-2", Status: "completed", Summary: "done", FilesModified: []string{"cmd/test.go"}, Handoff: codex.WorkerHandoff{Freshness: "not-run"}},
		},
	}

	result, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("runCodexBuildFinalize failed: %v", err)
	}

	recoveryRaw, ok := result["recovery_instructions"]
	if !ok {
		t.Fatal("expected 'recovery_instructions' in result for blocked dispatch")
	}
	recoveryInstructions := recoveryRaw.([]map[string]interface{})
	if len(recoveryInstructions) != 1 {
		t.Fatalf("expected 1 recovery instruction, got %d", len(recoveryInstructions))
	}
	// "blocked" maps to RequiresAttempt which returns "retry" as first action
	if recoveryInstructions[0]["action"] != "retry" {
		t.Errorf("expected action 'retry' for blocked failure (requires-attempt), got %v", recoveryInstructions[0]["action"])
	}
}

func TestBuildFinalize_NoRecoveryForCompletedDispatches(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	origCB := globalCircuitBreaker
	cb := NewCircuitBreaker(3)
	globalCircuitBreaker = cb
	defer func() { globalCircuitBreaker = origCB }()

	state := colony.ColonyState{
		Goal:         strPtr("Test goal"),
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Test Phase", Status: colony.PhaseInProgress, Tasks: []colony.Task{
					{ID: strPtr("task-1"), Status: colony.TaskInProgress},
				}},
			},
		},
	}
	createTestColonyState(t, dataDir, state)

	manifest := codexBuildManifest{
		Phase:    1,
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{
			{Name: "Builder-1", Caste: "builder", TaskID: "task-1", Task: "Build feature", Status: "pending", Wave: 1},
		},
		SelectedTasks: []string{"task-1"},
	}
	completion := codexExternalBuildCompletion{
		DispatchManifest: &manifest,
		Results: []codexExternalBuildWorkerResult{
			{Name: "Builder-1", Status: "completed", Summary: "all done", FilesModified: []string{"cmd/test.go"}, Handoff: codex.WorkerHandoff{Freshness: "not-run"}},
		},
	}

	result, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("runCodexBuildFinalize failed: %v", err)
	}

	_, ok := result["recovery_instructions"]
	if ok {
		t.Error("expected no 'recovery_instructions' when all dispatches completed")
	}
}

func TestBuildFinalize_BudgetPersisted(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	origCB := globalCircuitBreaker
	cb := NewCircuitBreaker(3)
	globalCircuitBreaker = cb
	defer func() { globalCircuitBreaker = origCB }()

	state := colony.ColonyState{
		Goal:         strPtr("Test goal"),
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Test Phase", Status: colony.PhaseInProgress, Tasks: []colony.Task{
					{ID: strPtr("task-1"), Status: colony.TaskInProgress},
					{ID: strPtr("task-2"), Status: colony.TaskInProgress},
				}},
			},
		},
	}
	createTestColonyState(t, dataDir, state)

	// Need at least one completed worker with files to pass provenance validation.
	manifest := codexBuildManifest{
		Phase:    1,
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{
			{Name: "Builder-1", Caste: "builder", TaskID: "task-1", Task: "Build feature", Status: "pending", Wave: 1},
			{Name: "Builder-2", Caste: "builder", TaskID: "task-2", Task: "Build feature 2", Status: "pending", Wave: 1},
		},
		SelectedTasks: []string{"task-1", "task-2"},
	}
	completion := codexExternalBuildCompletion{
		DispatchManifest: &manifest,
		Results: []codexExternalBuildWorkerResult{
			{Name: "Builder-1", Status: "timeout", Summary: "timed out", Handoff: codex.WorkerHandoff{Freshness: "not-run"}},
			{Name: "Builder-2", Status: "completed", Summary: "done", FilesModified: []string{"cmd/test.go"}, Handoff: codex.WorkerHandoff{Freshness: "not-run"}},
		},
	}

	_, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("runCodexBuildFinalize failed: %v", err)
	}

	// Verify budget was persisted to recovery-log file
	budget := budgetFromRecoveryLog(1, 1)
	if budget == nil {
		t.Fatal("expected budget to be persisted in recovery-log file")
	}
	if budget.RetriesUsed != 1 {
		t.Errorf("expected retries_used=1 after one failed dispatch, got %d", budget.RetriesUsed)
	}
}

func TestBuildFinalize_MultipleFailedDispatches(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	origCB := globalCircuitBreaker
	cb := NewCircuitBreaker(3)
	globalCircuitBreaker = cb
	defer func() { globalCircuitBreaker = origCB }()

	state := colony.ColonyState{
		Goal:         strPtr("Test goal"),
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Test Phase", Status: colony.PhaseInProgress, Tasks: []colony.Task{
					{ID: strPtr("task-1"), Status: colony.TaskInProgress},
					{ID: strPtr("task-2"), Status: colony.TaskInProgress},
					{ID: strPtr("task-3"), Status: colony.TaskInProgress},
				}},
			},
		},
	}
	createTestColonyState(t, dataDir, state)

	manifest := codexBuildManifest{
		Phase:    1,
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{
			{Name: "Builder-1", Caste: "builder", TaskID: "task-1", Task: "Build feature 1", Status: "pending", Wave: 1},
			{Name: "Builder-2", Caste: "builder", TaskID: "task-2", Task: "Build feature 2", Status: "pending", Wave: 1},
			{Name: "Builder-3", Caste: "builder", TaskID: "task-3", Task: "Build feature 3", Status: "pending", Wave: 1},
		},
		SelectedTasks: []string{"task-1", "task-2", "task-3"},
	}
	completion := codexExternalBuildCompletion{
		DispatchManifest: &manifest,
		Results: []codexExternalBuildWorkerResult{
			{Name: "Builder-1", Status: "timeout", Summary: "timed out", Handoff: codex.WorkerHandoff{Freshness: "not-run"}},
			{Name: "Builder-2", Status: "failed", Summary: "generic failure", Handoff: codex.WorkerHandoff{Freshness: "not-run"}},
			{Name: "Builder-3", Status: "completed", Summary: "done", FilesModified: []string{"cmd/test.go"}, Handoff: codex.WorkerHandoff{Freshness: "not-run"}},
		},
	}

	result, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("runCodexBuildFinalize failed: %v", err)
	}

	recoveryRaw, ok := result["recovery_instructions"]
	if !ok {
		t.Fatal("expected 'recovery_instructions' in result")
	}
	recoveryInstructions := recoveryRaw.([]map[string]interface{})
	if len(recoveryInstructions) != 2 {
		t.Fatalf("expected 2 recovery instructions (2 failed dispatches), got %d", len(recoveryInstructions))
	}
	// Budget should be decremented for both
	budget := budgetFromRecoveryLog(1, 1)
	if budget == nil {
		t.Fatal("expected budget to be persisted")
	}
	if budget.totalUsed() != 2 {
		t.Errorf("expected total budget used=2, got %d", budget.totalUsed())
	}
}

func TestRecoveryLogFile_BackwardCompatibility(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	// Write a legacy recovery log file without recovery_budget field
	legacyData := map[string]interface{}{
		"phase":   1,
		"entries": []interface{}{},
	}
	raw, _ := json.MarshalIndent(legacyData, "", "  ")
	if err := os.WriteFile(filepath.Join(dataDir, "recovery-log-1.json"), raw, 0644); err != nil {
		t.Fatalf("failed to write legacy file: %v", err)
	}

	// Reading should succeed without panic
	file, err := recoveryLogReadPhase(1)
	if err != nil {
		t.Fatalf("recoveryLogReadPhase failed on legacy file: %v", err)
	}
	if file.Phase != 1 {
		t.Errorf("expected phase=1, got %d", file.Phase)
	}
	if file.RecoveryBudget != nil {
		t.Error("expected nil RecoveryBudget for legacy file")
	}
}

// --- Task 2: Continue finalize gate recovery wiring tests ---

// evaluateGateRecovery simulates the continue finalize gate recovery evaluation.
// This mirrors the logic that will be wired into runCodexContinueFinalize.
func evaluateGateRecovery(phaseNum int, gates codexContinueGateReport, cb *CircuitBreaker) []map[string]interface{} {
	var instructions []map[string]interface{}
	budget := budgetFromRecoveryLog(phaseNum, 1)
	if budget == nil {
		budget = newRecoveryBudget(1)
	}

	for _, c := range gates.Checks {
		if c.Passed {
			continue
		}
		tier, _ := gateClassify(c.Name)
		// Per D-04: blocking failures escalate immediately, no orchestrator
		if tier == hardBlock {
			instructions = append(instructions, map[string]interface{}{
				"gate":           c.Name,
				"classification": "hard_block",
				"action":         "escalate",
				"detail":         "hard_block gate failure requires human intervention",
			})
			continue
		}

		// Build recovery context from gate failure
		ctx := RecoveryContext{
			Phase:          phaseNum,
			Wave:           1,
			WorkerName:     fmt.Sprintf("gate-%s", c.Name),
			Caste:          "watcher",
			Status:         "failed",
			ErrorMessage:   c.Detail,
			Budget:         budget,
			CircuitBreaker: cb,
		}
		outcome := orchestrateRecovery(ctx)

		// Persist recovery log entries
		if len(outcome.LogEntries) > 0 {
			existingLog, _ := recoveryLogReadPhase(phaseNum)
			existingLog.Entries = append(existingLog.Entries, outcome.LogEntries...)
			existingLog.Phase = phaseNum
			_ = recoveryLogWritePhase(phaseNum, existingLog.Entries)
		}

		instructions = append(instructions, map[string]interface{}{
			"gate":           c.Name,
			"classification": string(outcome.Classification),
			"action":         outcome.Action.Type,
			"detail":         outcome.Action.Detail,
			"exhausted":      outcome.Exhausted,
			"rationale":      outcome.Rationale,
		})
	}

	_ = persistBudgetToRecoveryLog(phaseNum, budget)
	return instructions
}

func TestContinueFinalize_GateRecovery_HardBlockSkipsOrchestrator(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	cb := NewCircuitBreaker(3)

	gates := codexContinueGateReport{
		Passed: false,
		Checks: []gateCheck{
			{Name: "tests_pass", Passed: false, Detail: "2 tests failed"},
			{Name: "gatekeeper", Passed: false, Detail: "CVE-2024-1234 found"},
		},
	}

	instructions := evaluateGateRecovery(1, gates, cb)

	if len(instructions) != 2 {
		t.Fatalf("expected 2 instructions, got %d", len(instructions))
	}

	// Both are hard_block gates -- should escalate immediately
	for _, inst := range instructions {
		if inst["action"] != "escalate" {
			t.Errorf("expected action 'escalate' for hard_block gate %v, got %v", inst["gate"], inst["action"])
		}
		if inst["classification"] != "hard_block" {
			t.Errorf("expected classification 'hard_block', got %v", inst["classification"])
		}
	}
}

func TestContinueFinalize_GateRecovery_SoftBlockAfterAutoResolve(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	cb := NewCircuitBreaker(3)

	gates := codexContinueGateReport{
		Passed: false,
		Checks: []gateCheck{
			{Name: "auditor", Passed: false, Detail: "code quality issues found"},
			{Name: "complexity", Passed: false, Detail: "high complexity in module X"},
		},
	}

	instructions := evaluateGateRecovery(1, gates, cb)

	if len(instructions) != 2 {
		t.Fatalf("expected 2 instructions, got %d", len(instructions))
	}

	// Soft block gates should go through orchestrator and get a recovery action
	for _, inst := range instructions {
		action, ok := inst["action"].(string)
		if !ok {
			t.Fatalf("expected action to be string, got %T", inst["action"])
		}
		// Gate failures use status "failed" which maps to RequiresAttempt -> first action is "retry"
		if action != "retry" {
			t.Errorf("expected action 'retry' for soft_block gate %v, got %v", inst["gate"], action)
		}
		classification, ok := inst["classification"].(string)
		if !ok {
			t.Fatalf("expected classification to be string, got %T", inst["classification"])
		}
		if classification != "requires-attempt" {
			t.Errorf("expected classification 'requires-attempt' for soft_block gate, got %v", classification)
		}
	}
}

func TestContinueFinalize_GateRecovery_MixedGates(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	cb := NewCircuitBreaker(3)

	gates := codexContinueGateReport{
		Passed: false,
		Checks: []gateCheck{
			{Name: "tests_pass", Passed: false, Detail: "tests failed"},
			{Name: "auditor", Passed: false, Detail: "quality issues"},
			{Name: "verification_loop", Passed: true, Detail: ""},
		},
	}

	instructions := evaluateGateRecovery(1, gates, cb)

	if len(instructions) != 2 {
		t.Fatalf("expected 2 instructions (only failed gates), got %d", len(instructions))
	}

	// First: hard_block gate (tests_pass)
	if instructions[0]["gate"] != "tests_pass" {
		t.Errorf("expected first gate 'tests_pass', got %v", instructions[0]["gate"])
	}
	if instructions[0]["action"] != "escalate" {
		t.Errorf("expected 'escalate' for hard_block gate, got %v", instructions[0]["action"])
	}

	// Second: soft_block gate (auditor) -- goes through orchestrator
	if instructions[1]["gate"] != "auditor" {
		t.Errorf("expected second gate 'auditor', got %v", instructions[1]["gate"])
	}
	if instructions[1]["action"] != "retry" {
		t.Errorf("expected 'retry' for soft_block gate, got %v", instructions[1]["action"])
	}
}

func TestContinueFinalize_GateRecovery_RecoveryInstructionsInOutput(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	state := colony.ColonyState{
		Goal:         strPtr("Test goal"),
		State:        colony.StateBUILT,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Test Phase", Status: colony.PhaseInProgress, Tasks: []colony.Task{
					{ID: strPtr("task-1"), Status: colony.TaskInProgress},
				}},
			},
		},
	}
	createTestColonyState(t, dataDir, state)

	// Create the continue manifest so validation passes
	manifest := codexContinueManifest{
		Present: true,
		Path:    "build/phase-1/manifest.json",
		Data: codexBuildManifest{
			Dispatches: []codexBuildDispatch{
				{Name: "Builder-1", Caste: "builder", TaskID: "task-1", Status: "completed", Outputs: []string{"cmd/test.go"}},
			},
		},
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "build", "phase-1"), 0755); err != nil {
		t.Fatalf("failed to create manifest dir: %v", err)
	}
	manifestData, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(dataDir, "build", "phase-1", "manifest.json"), manifestData, 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Set up gates that will fail
	gates := codexContinueGateReport{
		Passed: false,
		Checks: []gateCheck{
			{Name: "auditor", Passed: false, Detail: "quality issues found"},
		},
		BlockingIssues: []string{"auditor gate failed"},
	}

	// Test that finalizeBlockedExternalContinue includes recovery_instructions
	// when passed gate recovery instructions
	gateRecoveryInstructions := []map[string]interface{}{
		{
			"gate":           "auditor",
			"classification": "recoverable",
			"action":         "retry",
			"detail":         "recoverable: retrying worker",
			"exhausted":      false,
			"rationale":      "gate failure classified as recoverable",
		},
	}

	now := time.Now().UTC()
	result, _, err := finalizeBlockedExternalContinue(
		state,
		state.Plan.Phases[0],
		manifest,
		codexContinueVerificationReport{},
		codexContinueAssessment{},
		gates,
		nil,
		"",
		nil,
		now,
		"build/phase-1/verification.json",
		"build/phase-1/gates.json",
		gateRecoveryInstructions,
	)
	if err != nil {
		t.Fatalf("finalizeBlockedExternalContinue failed: %v", err)
	}

	// Verify recovery_instructions appear in the result
	recoveryRaw, ok := result["recovery_instructions"]
	if !ok {
		t.Fatal("expected 'recovery_instructions' in blocked result")
	}
	recoveryInstructions, ok := recoveryRaw.([]map[string]interface{})
	if !ok {
		t.Fatalf("expected []map[string]interface{}, got %T", recoveryRaw)
	}
	if len(recoveryInstructions) != 1 {
		t.Fatalf("expected 1 recovery instruction, got %d", len(recoveryInstructions))
	}
	if recoveryInstructions[0]["gate"] != "auditor" {
		t.Errorf("expected gate 'auditor', got %v", recoveryInstructions[0]["gate"])
	}
	if recoveryInstructions[0]["action"] != "retry" {
		t.Errorf("expected action 'retry', got %v", recoveryInstructions[0]["action"])
	}
}
