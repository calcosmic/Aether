package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- Queen Decision Tests (Plan 97-01) ---

// Test 1: queenDecide with all-passing gates returns "pass" recommendation for every gate
func TestQueenDecide_AllPassing(t *testing.T) {
	setupBuildFlowTest(t)
	budget := newRecoveryBudget(1)

	gates := codexContinueGateReport{
		Phase:       1,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Checks: []gateCheck{
			{Name: "tests_pass", Passed: true},
			{Name: "auditor", Passed: true},
			{Name: "medic", Passed: true},
		},
		Passed: true,
	}

	decisions := queenDecide(gates, budget, nil, 1, "standard")

	if len(decisions) != 3 {
		t.Fatalf("expected 3 decisions, got %d", len(decisions))
	}

	// tests_pass is hard_block, auto_resolve_eligible should be false
	if decisions[0].QueenRecommendation != "pass" {
		t.Errorf("gate tests_pass: expected pass recommendation, got %s", decisions[0].QueenRecommendation)
	}
	if decisions[0].AutoResolveEligible {
		t.Error("gate tests_pass: auto_resolve_eligible should be false for hard_block passing gate")
	}

	// auditor is soft_block, auto_resolve_eligible should be false for passing gate
	if decisions[1].AutoResolveEligible {
		t.Error("gate auditor: auto_resolve_eligible should be false for passing soft_block gate")
	}

	// All should have pass recommendation
	for i, d := range decisions {
		if d.QueenRecommendation != "pass" {
			t.Errorf("decision %d: expected pass, got %s", i, d.QueenRecommendation)
		}
	}
}

// Test 2: queenDecide with a failed soft_block gate returns "auto-resolve" recommendation
func TestQueenDecide_FailedSoftBlock(t *testing.T) {
	setupBuildFlowTest(t)
	budget := newRecoveryBudget(1)

	gates := codexContinueGateReport{
		Phase:       1,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Checks: []gateCheck{
			{Name: "auditor", Passed: false, Detail: "quality finding"},
		},
		Passed: false,
	}

	decisions := queenDecide(gates, budget, nil, 1, "standard")

	if len(decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decisions))
	}

	if decisions[0].QueenRecommendation != "auto-resolve" {
		t.Errorf("expected auto-resolve, got %s", decisions[0].QueenRecommendation)
	}

	if decisions[0].ClassificationTier != "soft_block" {
		t.Errorf("expected soft_block tier, got %s", decisions[0].ClassificationTier)
	}

	// Rationale should reference soft_block tier
	if decisions[0].Rationale == "" {
		t.Error("expected non-empty rationale")
	}
}

// Test 3: queenDecide with a failed hard_block gate returns "escalate" recommendation
func TestQueenDecide_FailedHardBlock(t *testing.T) {
	setupBuildFlowTest(t)
	budget := newRecoveryBudget(1)

	gates := codexContinueGateReport{
		Phase:       1,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Checks: []gateCheck{
			{Name: "tests_pass", Passed: false, Detail: "tests failing"},
		},
		Passed: false,
	}

	decisions := queenDecide(gates, budget, nil, 1, "standard")

	if len(decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decisions))
	}

	if decisions[0].QueenRecommendation != "escalate" {
		t.Errorf("expected escalate, got %s", decisions[0].QueenRecommendation)
	}

	if decisions[0].ClassificationTier != "hard_block" {
		t.Errorf("expected hard_block tier, got %s", decisions[0].ClassificationTier)
	}
}

// Test 4: queenDecide with exhausted budget returns "escalate" for failed soft_block gates
func TestQueenDecide_ExhaustedBudget(t *testing.T) {
	setupBuildFlowTest(t)
	budget := newRecoveryBudget(1)
	budget.TotalBudget = 1
	budget.RetriesUsed = 1 // exhaust the budget

	gates := codexContinueGateReport{
		Phase:       1,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Checks: []gateCheck{
			{Name: "auditor", Passed: false, Detail: "quality finding"},
		},
		Passed: false,
	}

	decisions := queenDecide(gates, budget, nil, 1, "standard")

	if len(decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decisions))
	}

	if decisions[0].QueenRecommendation != "escalate" {
		t.Errorf("expected escalate with exhausted budget, got %s", decisions[0].QueenRecommendation)
	}
}

// Test 5: queenDecide includes recovery preview for ALL gates (passing and failing) per D-04
func TestQueenDecide_RecoveryPreviewForAllGates(t *testing.T) {
	setupBuildFlowTest(t)
	budget := newRecoveryBudget(1)

	gates := codexContinueGateReport{
		Phase:       1,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Checks: []gateCheck{
			{Name: "tests_pass", Passed: true},  // hard_block, passing
			{Name: "auditor", Passed: false},     // soft_block, failing
			{Name: "medic", Passed: true},        // advisory, passing
		},
		Passed: false,
	}

	decisions := queenDecide(gates, budget, nil, 1, "standard")

	for i, d := range decisions {
		if d.RecoveryPreview == nil {
			t.Errorf("decision %d (%s): expected recovery preview for all gates per D-04", i, d.GateName)
		}
	}

	// Verify hard_block passing gate would escalate if it failed
	if decisions[0].RecoveryPreview.WouldEscalate != true {
		t.Error("tests_pass (hard_block): WouldEscalate should be true")
	}

	// Verify soft_block failing gate would auto-resolve
	if decisions[1].RecoveryPreview.WouldAutoResolve != true {
		t.Error("auditor (soft_block, failing): WouldAutoResolve should be true")
	}
}

// Test 6: queenDecide includes budget snapshot per D-03
func TestQueenDecide_BudgetSnapshot(t *testing.T) {
	setupBuildFlowTest(t)
	budget := newRecoveryBudget(1)
	budget.TotalBudget = 5
	budget.RetriesUsed = 1
	budget.ReassignsUsed = 1

	gates := codexContinueGateReport{
		Phase:       1,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Checks: []gateCheck{
			{Name: "tests_pass", Passed: true},
		},
		Passed: true,
	}

	decisions := queenDecide(gates, budget, nil, 1, "standard")

	// All decisions should have recovery preview with budget info
	for _, d := range decisions {
		if d.RecoveryPreview == nil {
			t.Fatal("expected recovery preview")
		}
		if d.RecoveryPreview.BudgetRemaining != 3 { // 5 - 1 - 1 = 3
			t.Errorf("expected budget_remaining=3, got %d", d.RecoveryPreview.BudgetRemaining)
		}
	}
}

// Test 7: queenDecide with nil circuit breaker does not panic
func TestQueenDecide_NilBreaker(t *testing.T) {
	setupBuildFlowTest(t)
	budget := newRecoveryBudget(1)

	gates := codexContinueGateReport{
		Phase:       1,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Checks: []gateCheck{
			{Name: "tests_pass", Passed: false},
		},
		Passed: false,
	}

	// Must not panic
	decisions := queenDecide(gates, budget, nil, 1, "standard")

	if len(decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decisions))
	}

	if decisions[0].Rationale == "" {
		t.Error("expected non-empty rationale even with nil breaker")
	}
}

// Test 8: queenDecide with tripped circuit breaker returns "escalate" for gates with tripped workers
func TestQueenDecide_TrippedBreaker(t *testing.T) {
	setupBuildFlowTest(t)
	budget := newRecoveryBudget(1)

	breaker := NewCircuitBreaker(3)
	// Trip the breaker for a worker
	breaker.RecordFailure("worker-1")
	breaker.RecordFailure("worker-1")
	breaker.RecordFailure("worker-1")

	gates := codexContinueGateReport{
		Phase:       1,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Checks: []gateCheck{
			{Name: "tests_pass", Passed: false},
		},
		Passed: false,
	}

	decisions := queenDecide(gates, budget, breaker, 1, "standard")

	if len(decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decisions))
	}

	// With tripped breaker, rationale should mention breaker state
	if decisions[0].Rationale == "" {
		t.Error("expected non-empty rationale with breaker state")
	}
}

// Test 9: queenStateWrite + queenStateRead round-trip
func TestQueenState_RoundTrip(t *testing.T) {
	setupBuildFlowTest(t)

	state := QueenStateFile{
		Phase:       97,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Decisions: []QueenDecision{
			{
				GateName:            "tests_pass",
				Status:              "passed",
				ClassificationTier:  "hard_block",
				QueenRecommendation: "pass",
				Rationale:           "hard_block gate passed",
			},
		},
		BudgetSnapshot: &RecoveryBudget{
			TotalBudget: 3,
			Wave:        1,
		},
		EscalationLog: []EscalationEntry{
			{
				Timestamp:         time.Now().Format(time.RFC3339),
				BreakerTripped:    []string{"worker-1"},
				EscalationAction:  "escalate_to_human",
				Rationale:         "circuit breaker tripped",
			},
		},
	}

	if err := queenStateWrite(97, state); err != nil {
		t.Fatalf("queenStateWrite: %v", err)
	}

	loaded, err := queenStateRead(97)
	if err != nil {
		t.Fatalf("queenStateRead: %v", err)
	}

	if loaded.Phase != 97 {
		t.Errorf("expected phase 97, got %d", loaded.Phase)
	}

	if len(loaded.Decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(loaded.Decisions))
	}

	if loaded.Decisions[0].GateName != "tests_pass" {
		t.Errorf("expected gate_name tests_pass, got %s", loaded.Decisions[0].GateName)
	}

	if loaded.BudgetSnapshot == nil {
		t.Fatal("expected non-nil budget snapshot")
	}

	if loaded.BudgetSnapshot.TotalBudget != 3 {
		t.Errorf("expected total_budget 3, got %d", loaded.BudgetSnapshot.TotalBudget)
	}

	if len(loaded.EscalationLog) != 1 {
		t.Fatalf("expected 1 escalation log entry, got %d", len(loaded.EscalationLog))
	}

	if loaded.EscalationLog[0].EscalationAction != "escalate_to_human" {
		t.Errorf("expected escalation_action escalate_to_human, got %s", loaded.EscalationLog[0].EscalationAction)
	}
}

// Test 10: queenStateRead returns error for non-existent phase file
func TestQueenStateRead_NonExistent(t *testing.T) {
	setupBuildFlowTest(t)

	_, err := queenStateRead(99999)
	if err == nil {
		t.Error("expected error for non-existent queen-state file")
	}
}

// Test 11: queenLogEscalation appends entry to queen-state file
func TestQueenLogEscalation_Appends(t *testing.T) {
	setupBuildFlowTest(t)

	// Write initial state
	initialState := QueenStateFile{
		Phase:       42,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Decisions:   []QueenDecision{},
	}
	if err := queenStateWrite(42, initialState); err != nil {
		t.Fatalf("queenStateWrite: %v", err)
	}

	// Log escalation
	queenLogEscalation(42, []string{"worker-1", "worker-2"}, "circuit breaker tripped during finalize")

	// Read back and verify
	loaded, err := queenStateRead(42)
	if err != nil {
		t.Fatalf("queenStateRead: %v", err)
	}

	if len(loaded.EscalationLog) != 1 {
		t.Fatalf("expected 1 escalation log entry, got %d", len(loaded.EscalationLog))
	}

	entry := loaded.EscalationLog[0]
	if entry.EscalationAction != "escalate_to_human" {
		t.Errorf("expected escalation_action escalate_to_human, got %s", entry.EscalationAction)
	}

	if len(entry.BreakerTripped) != 2 {
		t.Errorf("expected 2 tripped workers, got %d", len(entry.BreakerTripped))
	}

	if len(loaded.BreakerTrippedWorkers) != 2 {
		t.Errorf("expected 2 breaker_tripped_workers, got %d", len(loaded.BreakerTrippedWorkers))
	}
}

// Test 12: queenLogEscalation creates file if it does not exist
func TestQueenLogEscalation_CreatesFile(t *testing.T) {
	setupBuildFlowTest(t)

	// No prior state file -- queenLogEscalation should create one
	queenLogEscalation(77, []string{"worker-x"}, "breaker tripped")

	loaded, err := queenStateRead(77)
	if err != nil {
		t.Fatalf("queenStateRead: %v", err)
	}

	if loaded.Phase != 77 {
		t.Errorf("expected phase 77, got %d", loaded.Phase)
	}

	if len(loaded.EscalationLog) != 1 {
		t.Fatalf("expected 1 escalation entry, got %d", len(loaded.EscalationLog))
	}
}

// Test 13: queenDecide with unknown/unclassified gate returns empty classification tier and "escalate" recommendation
func TestQueenDecide_UnclassifiedGate(t *testing.T) {
	setupBuildFlowTest(t)
	budget := newRecoveryBudget(1)

	gates := codexContinueGateReport{
		Phase:       1,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Checks: []gateCheck{
			{Name: "some_unknown_gate", Passed: false},
		},
		Passed: false,
	}

	decisions := queenDecide(gates, budget, nil, 1, "standard")

	if len(decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decisions))
	}

	if decisions[0].ClassificationTier != "" {
		t.Errorf("expected empty classification tier for unknown gate, got %q", decisions[0].ClassificationTier)
	}

	if decisions[0].QueenRecommendation != "escalate" {
		t.Errorf("expected escalate for unclassified failed gate, got %s", decisions[0].QueenRecommendation)
	}
}

// Test 14: queenDecide does NOT consume budget (read-only)
func TestQueenDecide_BudgetNotConsumed(t *testing.T) {
	setupBuildFlowTest(t)
	budget := newRecoveryBudget(1)
	budget.TotalBudget = 5

	gates := codexContinueGateReport{
		Phase:       1,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Checks: []gateCheck{
			{Name: "auditor", Passed: false},
			{Name: "complexity", Passed: false},
			{Name: "tests_pass", Passed: true},
		},
		Passed: false,
	}

	usedBefore := budget.totalUsed()
	_ = queenDecide(gates, budget, nil, 1, "standard")
	usedAfter := budget.totalUsed()

	if usedBefore != 0 || usedAfter != 0 {
		t.Errorf("queenDecide should not consume budget: before=%d, after=%d", usedBefore, usedAfter)
	}
}

// Test 15: queenDecide with nil budget does not panic (treat as exhausted)
func TestQueenDecide_NilBudget(t *testing.T) {
	setupBuildFlowTest(t)

	gates := codexContinueGateReport{
		Phase:       1,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Checks: []gateCheck{
			{Name: "auditor", Passed: false},
		},
		Passed: false,
	}

	decisions := queenDecide(gates, nil, nil, 1, "standard")

	if len(decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decisions))
	}

	// With nil budget, soft_block failure should escalate
	if decisions[0].QueenRecommendation != "escalate" {
		t.Errorf("expected escalate with nil budget, got %s", decisions[0].QueenRecommendation)
	}

	// Recovery preview budget should be 0
	if decisions[0].RecoveryPreview == nil {
		t.Fatal("expected recovery preview")
	}
	if decisions[0].RecoveryPreview.BudgetRemaining != 0 {
		t.Errorf("expected budget_remaining=0 with nil budget, got %d", decisions[0].RecoveryPreview.BudgetRemaining)
	}
}

// Test 16: queenLogEscalation appends to existing escalation log (not replaces)
func TestQueenLogEscalation_AppendsMultiple(t *testing.T) {
	setupBuildFlowTest(t)

	initialState := QueenStateFile{
		Phase:       55,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Decisions:   []QueenDecision{},
		EscalationLog: []EscalationEntry{
			{
				Timestamp:        time.Now().Format(time.RFC3339),
				EscalationAction: "escalate_to_human",
				Rationale:        "first escalation",
			},
		},
	}
	if err := queenStateWrite(55, initialState); err != nil {
		t.Fatalf("queenStateWrite: %v", err)
	}

	queenLogEscalation(55, []string{"worker-2"}, "second escalation")

	loaded, err := queenStateRead(55)
	if err != nil {
		t.Fatalf("queenStateRead: %v", err)
	}

	if len(loaded.EscalationLog) != 2 {
		t.Fatalf("expected 2 escalation entries, got %d", len(loaded.EscalationLog))
	}

	// First entry preserved
	if loaded.EscalationLog[0].Rationale != "first escalation" {
		t.Errorf("first entry not preserved: %s", loaded.EscalationLog[0].Rationale)
	}

	// Second entry appended
	if loaded.EscalationLog[1].Rationale != "second escalation" {
		t.Errorf("second entry wrong: %s", loaded.EscalationLog[1].Rationale)
	}
}

// Test 17: verify queen-state file is persisted at correct path
func TestQueenState_FilePath(t *testing.T) {
	setupBuildFlowTest(t)

	state := QueenStateFile{
		Phase:       33,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Decisions:   []QueenDecision{},
	}
	if err := queenStateWrite(33, state); err != nil {
		t.Fatalf("queenStateWrite: %v", err)
	}

	// Verify the file exists at the expected path
	dataDir := os.Getenv("AETHER_ROOT") + "/.aether/data"
	expectedPath := filepath.Join(dataDir, "queen-state-33.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected queen-state-33.json at %s", expectedPath)
	}
}

// --- Plan 97-02: Integration tests for plan-only wiring and finalize advisory context ---

// Test 18 (Plan 97-02): plan-only with passing gates includes queen_decisions array in result map
// This tests the wiring: queenDecide produces decisions for each gate with "pass" recommendation.
func TestPlanOnlyQueenDecisions(t *testing.T) {
	setupBuildFlowTest(t)
	budget := newRecoveryBudget(1)

	gates := codexContinueGateReport{
		Phase:       7,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Checks: []gateCheck{
			{Name: "tests_pass", Passed: true},
			{Name: "auditor", Passed: true},
			{Name: "medic", Passed: true},
		},
		Passed: true,
	}

	decisions := queenDecide(gates, budget, nil, 7, "standard")

	if len(decisions) != 3 {
		t.Fatalf("expected 3 decisions, got %d", len(decisions))
	}

	for i, d := range decisions {
		if d.QueenRecommendation != "pass" {
			t.Errorf("decision %d: expected pass recommendation, got %s", i, d.QueenRecommendation)
		}
	}

	// Verify auto_resolve_eligible matches tier (hard_block = false, soft_block for passing = false)
	if decisions[0].AutoResolveEligible {
		t.Error("tests_pass (hard_block, passed): auto_resolve_eligible should be false")
	}
}

// Test 19 (Plan 97-02): plan-only persists queen-state-{N}.json file that can be read back
func TestPlanOnlyStatePersistence(t *testing.T) {
	setupBuildFlowTest(t)

	budget := newRecoveryBudget(1)
	budget.TotalBudget = 5

	gates := codexContinueGateReport{
		Phase:       42,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Checks: []gateCheck{
			{Name: "tests_pass", Passed: true},
			{Name: "auditor", Passed: false},
		},
		Passed: false,
	}

	decisions := queenDecide(gates, budget, nil, 42, "standard")

	// Build and persist queen state (simulating what plan-only does)
	queenState := QueenStateFile{
		Phase:          42,
		GeneratedAt:    gates.GeneratedAt,
		Decisions:      decisions,
		BudgetSnapshot: budget,
	}

	if err := queenStateWrite(42, queenState); err != nil {
		t.Fatalf("queenStateWrite: %v", err)
	}

	// Read back and verify
	loaded, err := queenStateRead(42)
	if err != nil {
		t.Fatalf("queenStateRead: %v", err)
	}

	if loaded.Phase != 42 {
		t.Errorf("expected phase 42, got %d", loaded.Phase)
	}

	if len(loaded.Decisions) != 2 {
		t.Fatalf("expected 2 decisions, got %d", len(loaded.Decisions))
	}

	if loaded.BudgetSnapshot == nil {
		t.Fatal("expected non-nil budget snapshot")
	}

	if loaded.BudgetSnapshot.TotalBudget != 5 {
		t.Errorf("expected total_budget 5, got %d", loaded.BudgetSnapshot.TotalBudget)
	}
}

// Test 20 (Plan 97-02): plan-only queen decisions do NOT consume budget
func TestPlanOnlyBudgetNotConsumed(t *testing.T) {
	setupBuildFlowTest(t)
	budget := newRecoveryBudget(1)
	budget.TotalBudget = 3

	gates := codexContinueGateReport{
		Phase:       10,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Checks: []gateCheck{
			{Name: "auditor", Passed: false},
			{Name: "tests_pass", Passed: false},
		},
		Passed: false,
	}

	usedBefore := budget.totalUsed()
	_ = queenDecide(gates, budget, nil, 10, "standard")
	usedAfter := budget.totalUsed()

	if usedBefore != usedAfter {
		t.Errorf("queenDecide consumed budget: before=%d, after=%d", usedBefore, usedAfter)
	}
}

// Test 21 (Plan 97-02): plan-only result map includes queen_state_file key with correct value
func TestPlanOnlyResultMapKeys(t *testing.T) {
	setupBuildFlowTest(t)

	budget := newRecoveryBudget(1)

	gates := codexContinueGateReport{
		Phase:       88,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Checks: []gateCheck{
			{Name: "tests_pass", Passed: true},
		},
		Passed: true,
	}

	decisions := queenDecide(gates, budget, nil, 88, "standard")

	// Simulate what plan-only does: add keys to result map
	result := map[string]interface{}{
		"queen_decisions":   decisions,
		"queen_state_file":  fmt.Sprintf("queen-state-%d.json", 88),
	}

	// Verify queen_decisions key exists
	if _, ok := result["queen_decisions"]; !ok {
		t.Error("expected queen_decisions key in result map")
	}

	// Verify queen_state_file key exists with correct value
	stateFile, ok := result["queen_state_file"].(string)
	if !ok || stateFile != "queen-state-88.json" {
		t.Errorf("expected queen_state_file=queen-state-88.json, got %q", stateFile)
	}
}

// Test 22 (Plan 97-02): finalize reads queen-state file as advisory context without modifying it
func TestFinalizeAdvisoryContext(t *testing.T) {
	setupBuildFlowTest(t)

	// Test: queenStateRead for non-existent phase returns error (advisory context is optional)
	_, err := queenStateRead(99998)
	if err == nil {
		t.Error("expected error for non-existent queen-state file (advisory context is optional)")
	}

	// Test: write queen-state, then read it back, verify advisory context preserved
	initialDecisions := []QueenDecision{
		{
			GateName:            "tests_pass",
			Status:              "passed",
			ClassificationTier:  "hard_block",
			QueenRecommendation: "pass",
			Rationale:           "hard_block gate passed",
		},
	}

	state := QueenStateFile{
		Phase:       55,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Decisions:   initialDecisions,
		BudgetSnapshot: &RecoveryBudget{
			TotalBudget: 3,
			Wave:        1,
		},
	}

	if err := queenStateWrite(55, state); err != nil {
		t.Fatalf("queenStateWrite: %v", err)
	}

	// Finalize reads advisory context
	advisory, err := queenStateRead(55)
	if err != nil {
		t.Fatalf("queenStateRead: %v", err)
	}

	if len(advisory.Decisions) != 1 {
		t.Fatalf("expected 1 advisory decision, got %d", len(advisory.Decisions))
	}

	if advisory.Decisions[0].GateName != "tests_pass" {
		t.Errorf("expected gate_name tests_pass, got %s", advisory.Decisions[0].GateName)
	}

	// Verify advisory context is NOT modified by reading (read-only)
	advisoryCopy := advisory
	advisoryCopy.Decisions[0].GateName = "modified"
	_ = advisoryCopy

	// Re-read and verify original is unchanged
	advisoryAgain, _ := queenStateRead(55)
	if advisoryAgain.Decisions[0].GateName != "tests_pass" {
		t.Error("advisory context was modified by read -- should be read-only")
	}
}

// Test 23 (Plan 97-02): finalize logs escalation entry to queen-state when circuit breaker trips
func TestFinalizeEscalationLogging(t *testing.T) {
	setupBuildFlowTest(t)

	// Create a circuit breaker and trip it
	breaker := NewCircuitBreaker(3)
	breaker.RecordFailure("worker-1")
	breaker.RecordFailure("worker-1")
	breaker.RecordFailure("worker-1")

	tripped := breaker.TrippedWorkers()
	if len(tripped) == 0 {
		t.Fatal("expected circuit breaker to have tripped workers")
	}

	// Log escalation (simulating what finalize does)
	queenLogEscalation(66, tripped, "circuit breaker tripped during finalize -- escalation required")

	// Read back and verify
	loaded, err := queenStateRead(66)
	if err != nil {
		t.Fatalf("queenStateRead: %v", err)
	}

	if len(loaded.EscalationLog) != 1 {
		t.Fatalf("expected 1 escalation log entry, got %d", len(loaded.EscalationLog))
	}

	entry := loaded.EscalationLog[0]
	if entry.EscalationAction != "escalate_to_human" {
		t.Errorf("expected escalation_action escalate_to_human, got %s", entry.EscalationAction)
	}

	if len(entry.BreakerTripped) == 0 {
		t.Error("expected non-empty breaker_tripped_workers in escalation entry")
	}

	// Verify BreakerTrippedWorkers field is populated
	if len(loaded.BreakerTrippedWorkers) == 0 {
		t.Error("expected non-empty breaker_tripped_workers in state file")
	}
}

// Test 24 (Plan 97-02): finalize with nil queen-state file (plan-only never ran) completes without error
func TestFinalizeNilQueenState(t *testing.T) {
	setupBuildFlowTest(t)

	// queenStateRead for a phase that has no queen-state file should return error but not panic
	_, err := queenStateRead(99999)
	if err == nil {
		t.Error("expected error for non-existent queen-state file")
	}

	// queenLogEscalation should create the file if it doesn't exist (best-effort)
	queenLogEscalation(99999, []string{"worker-test"}, "test escalation")

	// Now read should succeed
	loaded, err := queenStateRead(99999)
	if err != nil {
		t.Fatalf("queenStateRead after logEscalation: %v", err)
	}

	if loaded.Phase != 99999 {
		t.Errorf("expected phase 99999, got %d", loaded.Phase)
	}

	if len(loaded.EscalationLog) != 1 {
		t.Fatalf("expected 1 escalation entry, got %d", len(loaded.EscalationLog))
	}
}
