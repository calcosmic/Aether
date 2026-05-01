package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/events"
)

// --- readUnblockAttempts tests ---

func TestReadUnblockAttempts_NoFile(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	attempts := readUnblockAttempts(99)
	if attempts != 0 {
		t.Errorf("readUnblockAttempts with no file: expected 0, got %d", attempts)
	}
}

func TestReadUnblockAttempts_ExistingFile(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	// Write gate results file with attempts
	fileData := gateResultsFile{
		Attempts: 3,
		Results: []GateCheckResult{
			{Name: "test_gate", Status: "failed", Timestamp: "2026-05-01T10:00:00Z"},
		},
	}
	data, _ := json.MarshalIndent(fileData, "", "  ")
	if err := os.WriteFile(filepath.Join(dataDir, "gate-results-5.json"), data, 0644); err != nil {
		t.Fatalf("failed to write gate results: %v", err)
	}

	attempts := readUnblockAttempts(5)
	if attempts != 3 {
		t.Errorf("readUnblockAttempts: expected 3, got %d", attempts)
	}
}

// --- incrementUnblockAttempts tests ---

func TestIncrementUnblockAttempts_NewFile(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := incrementUnblockAttempts(7); err != nil {
		t.Fatalf("incrementUnblockAttempts: %v", err)
	}

	attempts := readUnblockAttempts(7)
	if attempts != 1 {
		t.Errorf("after increment: expected 1, got %d", attempts)
	}
}

func TestIncrementUnblockAttempts_ExistingFile(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	// Write initial file with 1 attempt
	fileData := gateResultsFile{
		Attempts: 1,
		Results: []GateCheckResult{
			{Name: "gate1", Status: "failed", Timestamp: "2026-05-01T10:00:00Z"},
		},
	}
	data, _ := json.MarshalIndent(fileData, "", "  ")
	if err := os.WriteFile(filepath.Join(dataDir, "gate-results-8.json"), data, 0644); err != nil {
		t.Fatalf("failed to write gate results: %v", err)
	}

	if err := incrementUnblockAttempts(8); err != nil {
		t.Fatalf("incrementUnblockAttempts: %v", err)
	}

	attempts := readUnblockAttempts(8)
	if attempts != 2 {
		t.Errorf("after increment from 1: expected 2, got %d", attempts)
	}
}

// --- checkAttemptCap tests ---

func TestCheckAttemptCap_NotExceeded(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	// Write gate results with 1 attempt (less than cap of 3)
	fileData := gateResultsFile{
		Attempts: 1,
		Results:  []GateCheckResult{},
	}
	data, _ := json.MarshalIndent(fileData, "", "  ")
	if err := os.WriteFile(filepath.Join(dataDir, "gate-results-1.json"), data, 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	err := checkAttemptCap(1, 3)
	if err != nil {
		t.Errorf("checkAttemptCap(1, 3): expected nil, got %v", err)
	}
}

func TestCheckAttemptCap_ExactlyAtCap(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	// Write gate results with 3 attempts (at cap of 3)
	fileData := gateResultsFile{
		Attempts: 3,
		Results:  []GateCheckResult{},
	}
	data, _ := json.MarshalIndent(fileData, "", "  ")
	if err := os.WriteFile(filepath.Join(dataDir, "gate-results-3.json"), data, 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	err := checkAttemptCap(3, 3)
	if err == nil {
		t.Error("checkAttemptCap(3, 3): expected error at cap")
	}
	if !strings.Contains(err.Error(), "Max unblock attempts") {
		t.Errorf("error should mention 'Max unblock attempts', got: %v", err)
	}
}

func TestCheckAttemptCap_Exceeded(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	// Write gate results with 5 attempts (exceeds cap of 3)
	fileData := gateResultsFile{
		Attempts: 5,
		Results:  []GateCheckResult{},
	}
	data, _ := json.MarshalIndent(fileData, "", "  ")
	if err := os.WriteFile(filepath.Join(dataDir, "gate-results-5.json"), data, 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	err := checkAttemptCap(5, 3)
	if err == nil {
		t.Error("checkAttemptCap(5, 3): expected error when exceeded")
	}
}

// --- isFixerDispatchBlocked tests ---

func TestIsFixerDispatchBlocked_NoneBlocked(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	results := []GateCheckResult{
		{Name: "gate1", Status: "failed", Timestamp: "2026-05-01T10:00:00Z"},
	}
	fileData := gateResultsFile{Results: results}
	data, _ := json.MarshalIndent(fileData, "", "  ")
	if err := os.WriteFile(filepath.Join(dataDir, "gate-results-10.json"), data, 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	blocked, msg := isFixerDispatchBlocked(10)
	if blocked {
		t.Errorf("isFixerDispatchBlocked: expected false, got true with msg: %s", msg)
	}
}

func TestIsFixerDispatchBlocked_CircuitBreakerTripped(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	results := []GateCheckResult{
		{Name: "gate1", Status: "failed", Timestamp: "2026-05-01T10:00:00Z"},
	}
	fileData := gateResultsFile{Results: results}
	data, _ := json.MarshalIndent(fileData, "", "  ")
	if err := os.WriteFile(filepath.Join(dataDir, "gate-results-11.json"), data, 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Trip the circuit breaker for this gate
	cb := NewCircuitBreaker(1)
	cb.RecordFailure(gateRetryKey(11, "gate1"))

	// Replace global circuit breaker temporarily
	origCB := globalCircuitBreaker
	globalCircuitBreaker = cb
	defer func() { globalCircuitBreaker = origCB }()

	blocked, msg := isFixerDispatchBlocked(11)
	if !blocked {
		t.Error("isFixerDispatchBlocked: expected true when circuit breaker tripped")
	}
	if !strings.Contains(msg, "Circuit breaker tripped") {
		t.Errorf("blocked message should mention 'Circuit breaker tripped', got: %s", msg)
	}
	if !strings.Contains(msg, "Phase 11") {
		t.Errorf("blocked message should mention 'Phase 11', got: %s", msg)
	}
}

// --- dispatchFixer tests ---

func TestDispatchFixer_CircuitBreakerTripped(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	results := []GateCheckResult{
		{Name: "gate1", Status: "failed", Timestamp: "2026-05-01T10:00:00Z"},
	}
	fileData := gateResultsFile{Results: results}
	data, _ := json.MarshalIndent(fileData, "", "  ")
	if err := os.WriteFile(filepath.Join(dataDir, "gate-results-20.json"), data, 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	origCB := globalCircuitBreaker
	cb := NewCircuitBreaker(1)
	cb.RecordFailure(gateRetryKey(20, "gate1"))
	globalCircuitBreaker = cb
	defer func() { globalCircuitBreaker = origCB }()

	err := dispatchFixer(20, "propose")
	if err == nil {
		t.Fatal("dispatchFixer: expected error when circuit breaker tripped")
	}
	if !strings.Contains(err.Error(), "Circuit breaker tripped") {
		t.Errorf("error should mention circuit breaker, got: %v", err)
	}
}

func TestDispatchFixer_AttemptCapExceeded(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	// Write gate results with max attempts already used
	fileData := gateResultsFile{
		Attempts: 1,
		Results: []GateCheckResult{
			{Name: "gate1", Status: "failed", Timestamp: "2026-05-01T10:00:00Z"},
		},
	}
	data, _ := json.MarshalIndent(fileData, "", "  ")
	if err := os.WriteFile(filepath.Join(dataDir, "gate-results-21.json"), data, 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	err := dispatchFixer(21, "propose")
	if err == nil {
		t.Fatal("dispatchFixer: expected error when attempt cap exceeded")
	}
	if !strings.Contains(err.Error(), "Max unblock attempts") {
		t.Errorf("error should mention 'Max unblock attempts', got: %v", err)
	}
}

func TestDispatchFixer_InvalidMode(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	err := dispatchFixer(22, "invalid_mode")
	if err == nil {
		t.Fatal("dispatchFixer: expected error for invalid mode")
	}
	if !strings.Contains(err.Error(), "valid modes") {
		t.Errorf("error should mention 'valid modes', got: %v", err)
	}
}

func TestDispatchFixer_Success(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	results := []GateCheckResult{
		{Name: "gate1", Status: "failed", Detail: "tests broke", FixHint: "fix tests", Timestamp: "2026-05-01T10:00:00Z"},
	}
	fileData := gateResultsFile{Results: results}
	data, _ := json.MarshalIndent(fileData, "", "  ")
	if err := os.WriteFile(filepath.Join(dataDir, "gate-results-23.json"), data, 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	err := dispatchFixer(23, "propose")
	if err != nil {
		t.Fatalf("dispatchFixer: unexpected error: %v", err)
	}

	// Check that attempts were incremented
	attempts := readUnblockAttempts(23)
	if attempts != 1 {
		t.Errorf("after dispatch: expected 1 attempt, got %d", attempts)
	}
}

func TestDispatchFixer_EmitsLoopBreakEvent(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	results := []GateCheckResult{
		{Name: "gate1", Status: "failed", Timestamp: "2026-05-01T10:00:00Z"},
	}
	fileData := gateResultsFile{Results: results}
	data, _ := json.MarshalIndent(fileData, "", "  ")
	if err := os.WriteFile(filepath.Join(dataDir, "gate-results-24.json"), data, 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	err := dispatchFixer(24, "propose")
	if err != nil {
		t.Fatalf("dispatchFixer: unexpected error: %v", err)
	}

	// Verify loop break event was persisted to event bus
	persisted := readPersistedCeremonyEvents(t)
	found := false
	for _, evt := range persisted {
		if evt.Topic == events.CeremonyTopicLoopBreak {
			found = true
			break
		}
	}
	if !found {
		t.Error("dispatchFixer: expected loop_break event to be emitted")
	}
}

// --- resolveFixedGates tests ---

func TestResolveFixedGates_MarksAddressedGates(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	results := []GateCheckResult{
		{Name: "gate1", Status: "failed", Detail: "broken", FixHint: "fix it", Timestamp: "2026-05-01T10:00:00Z"},
		{Name: "gate2", Status: "failed", Detail: "also broken", Timestamp: "2026-05-01T10:00:01Z"},
		{Name: "gate3", Status: "passed", Timestamp: "2026-05-01T10:00:02Z"},
	}
	fileData := gateResultsFile{Results: results}
	data, _ := json.MarshalIndent(fileData, "", "  ")
	if err := os.WriteFile(filepath.Join(dataDir, "gate-results-30.json"), data, 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	err := resolveFixedGates(30, []string{"gate1"})
	if err != nil {
		t.Fatalf("resolveFixedGates: %v", err)
	}

	// Read back and verify
	updated, err := readGateResultsPhase(30)
	if err != nil {
		t.Fatalf("readGateResultsPhase: %v", err)
	}

	for _, r := range updated.Results {
		if r.Name == "gate1" {
			if r.Status != "passed" {
				t.Errorf("gate1 should be 'passed', got %q", r.Status)
			}
			if r.FixHint != "" {
				t.Error("gate1 FixHint should be cleared")
			}
			if r.Detail != "" {
				t.Error("gate1 Detail should be cleared")
			}
		}
		if r.Name == "gate2" {
			if r.Status != "failed" {
				t.Errorf("gate2 should remain 'failed', got %q", r.Status)
			}
		}
		if r.Name == "gate3" {
			if r.Status != "passed" {
				t.Errorf("gate3 should remain 'passed', got %q", r.Status)
			}
		}
	}
}

func TestResolveFixedGates_PreservesUnaddressed(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	results := []GateCheckResult{
		{Name: "gate1", Status: "failed", Timestamp: "2026-05-01T10:00:00Z"},
		{Name: "gate2", Status: "failed", Timestamp: "2026-05-01T10:00:01Z"},
	}
	fileData := gateResultsFile{Results: results}
	data, _ := json.MarshalIndent(fileData, "", "  ")
	if err := os.WriteFile(filepath.Join(dataDir, "gate-results-31.json"), data, 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	err := resolveFixedGates(31, []string{"gate1"})
	if err != nil {
		t.Fatalf("resolveFixedGates: %v", err)
	}

	updated, err := readGateResultsPhase(31)
	if err != nil {
		t.Fatalf("readGateResultsPhase: %v", err)
	}

	for _, r := range updated.Results {
		if r.Name == "gate2" {
			if r.Status != "failed" {
				t.Errorf("gate2 should remain 'failed', got %q", r.Status)
			}
		}
	}
}

func TestResolveFixedGates_IgnoresUnknownGates(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	results := []GateCheckResult{
		{Name: "gate1", Status: "failed", Timestamp: "2026-05-01T10:00:00Z"},
	}
	fileData := gateResultsFile{Results: results}
	data, _ := json.MarshalIndent(fileData, "", "  ")
	if err := os.WriteFile(filepath.Join(dataDir, "gate-results-32.json"), data, 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Address a gate that doesn't exist -- should not error
	err := resolveFixedGates(32, []string{"nonexistent_gate"})
	if err != nil {
		t.Fatalf("resolveFixedGates with unknown gate: should not error, got: %v", err)
	}
}

// --- recordFixerFailure tests ---

func TestRecordFixerFailure(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	results := []GateCheckResult{
		{Name: "gate1", Status: "failed", Timestamp: "2026-05-01T10:00:00Z"},
	}
	fileData := gateResultsFile{Results: results}
	data, _ := json.MarshalIndent(fileData, "", "  ")
	if err := os.WriteFile(filepath.Join(dataDir, "gate-results-40.json"), data, 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	origCB := globalCircuitBreaker
	cb := NewCircuitBreaker(3)
	globalCircuitBreaker = cb
	defer func() { globalCircuitBreaker = origCB }()

	recordFixerFailure(40, "gate1: unable to fix")

	// Verify circuit breaker recorded failure
	if cb.FailureCount(gateRetryKey(40, "gate1")) != 1 {
		t.Error("recordFixerFailure: expected circuit breaker failure count of 1")
	}

	// Verify loop break event was persisted to event bus
	persisted := readPersistedCeremonyEvents(t)
	found := false
	for _, evt := range persisted {
		if evt.Topic == events.CeremonyTopicLoopBreak {
			found = true
			break
		}
	}
	if !found {
		t.Error("recordFixerFailure: expected loop_break event to be emitted")
	}
}
