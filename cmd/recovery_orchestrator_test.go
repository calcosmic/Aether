package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
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
		Phase:        1,
		Wave:         1,
		WorkerName:   "Builder-1",
		TaskID:       "task-1",
		Caste:        "builder",
		Status:       "bad_task_spec",
		ErrorMessage: "invalid task specification",
		Dispatches:   []codex.WorkerDispatch{},
		CircuitBreaker: cb,
		Budget:       budget,
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
		Phase:        1,
		Wave:         1,
		WorkerName:   "Builder-1",
		TaskID:       "task-1",
		Caste:        "builder",
		Status:       "timeout",
		ErrorMessage: "worker timed out",
		Dispatches:   []codex.WorkerDispatch{},
		CircuitBreaker: cb,
		Budget:       budget,
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
		Phase:        1,
		Wave:         1,
		WorkerName:   "Builder-1",
		TaskID:       "task-1",
		Caste:        "builder",
		Status:       "timeout",
		ErrorMessage: "worker timed out",
		Dispatches:   []codex.WorkerDispatch{},
		CircuitBreaker: cb,
		Budget:       budget,
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
		Phase:        1,
		Wave:         1,
		WorkerName:   "Builder-1",
		TaskID:       "task-1",
		Caste:        "builder",
		Status:       "timeout",
		ErrorMessage: "worker timed out",
		Dispatches:   []codex.WorkerDispatch{},
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
		Phase:        1,
		Wave:         1,
		WorkerName:   "Builder-1",
		TaskID:       "task-1",
		Caste:        "builder",
		Status:       "timeout",
		ErrorMessage: "worker timed out",
		Dispatches:   []codex.WorkerDispatch{},
		CircuitBreaker: cb,
		Budget:       budget,
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
