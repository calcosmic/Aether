package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// gateResultsFile wraps gate results with unblock attempt tracking.
// The unblock_attempts field is co-located with gate data for simplicity.
type gateResultsFile struct {
	Attempts int               `json:"unblock_attempts,omitempty"`
	Results  []GateCheckResult `json:"results"`
}

// globalCircuitBreaker is the package-level circuit breaker used by Fixer dispatch.
// It can be overridden in tests. In production, a fresh breaker is created per dispatch session.
var globalCircuitBreaker *CircuitBreaker

// DefaultMaxUnblockAttempts is the default attempt cap per phase (D-06).
const DefaultMaxUnblockAttempts = 1

// validFixerModes lists the acceptable Fixer autonomy modes (D-01).
var validFixerModes = map[string]bool{
	"full":    true,
	"propose": true,
	"advise":  true,
}

// readUnblockAttempts reads the unblock_attempts count from gate-results-{phaseNum}.json.
// Returns 0 if the file does not exist or the field is missing.
func readUnblockAttempts(phaseNum int) int {
	fileData, err := readGateResultsPhase(phaseNum)
	if err != nil {
		return 0
	}
	return fileData.Attempts
}

// readGateResultsPhase reads the full gate results file including attempt metadata.
// Supports both the legacy plain array format and the newer gateResultsFile wrapper format.
func readGateResultsPhase(phaseNum int) (*gateResultsFile, error) {
	rel := fmt.Sprintf("gate-results-%d.json", phaseNum)

	// Read raw content to detect format
	raw, err := store.LoadRawJSON(rel)
	if err != nil {
		return nil, err
	}

	// Try wrapper format first (newer format with unblock_attempts)
	// The wrapper is a JSON object, while the legacy format is a JSON array.
	if len(raw) > 0 && raw[0] == '{' {
		var wrapped gateResultsFile
		if err := json.Unmarshal(raw, &wrapped); err != nil {
			return nil, fmt.Errorf("failed to unmarshal gate results file: %w", err)
		}
		return &wrapped, nil
	}

	// Fall back to plain array format (legacy) -- wrap with zero attempts
	var results []GateCheckResult
	if err := json.Unmarshal(raw, &results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gate results array: %w", err)
	}
	return &gateResultsFile{Results: results}, nil
}

// incrementUnblockAttempts increments the unblock attempt count for a phase.
// Creates the file if it doesn't exist.
func incrementUnblockAttempts(phaseNum int) error {
	rel := fmt.Sprintf("gate-results-%d.json", phaseNum)

	fileData, err := readGateResultsPhase(phaseNum)
	if err != nil {
		// No existing file -- create with 1 attempt and empty results
		fileData = &gateResultsFile{
			Attempts: 0,
			Results:  []GateCheckResult{},
		}
	}

	fileData.Attempts++

	return store.SaveJSON(rel, fileData)
}

// checkAttemptCap returns an error if the unblock attempt count has reached or exceeded maxAttempts.
func checkAttemptCap(phaseNum int, maxAttempts int) error {
	if maxAttempts <= 0 {
		maxAttempts = DefaultMaxUnblockAttempts
	}
	current := readUnblockAttempts(phaseNum)
	if current >= maxAttempts {
		return fmt.Errorf("Max unblock attempts (%d) reached for Phase %d. Human intervention required.", maxAttempts, phaseNum)
	}
	return nil
}

// isFixerDispatchBlocked checks if Fixer dispatch is blocked by the circuit breaker
// for any failed gate in the given phase. Returns (blocked, message).
func isFixerDispatchBlocked(phaseNum int) (bool, string) {
	cb := globalCircuitBreaker
	if cb == nil {
		return false, ""
	}

	fileData, err := readGateResultsPhase(phaseNum)
	if err != nil {
		return false, ""
	}

	for _, r := range fileData.Results {
		if r.Status == "failed" {
			key := gateRetryKey(phaseNum, r.Name)
			if !cb.Allow(key) {
				return true, fmt.Sprintf("Circuit breaker tripped for Phase %d -- manual intervention required", phaseNum)
			}
		}
	}

	return false, ""
}

// dispatchFixer is the main dispatch function for the Fixer agent.
// It validates mode, checks circuit breaker and attempt cap, increments attempts,
// emits telemetry, and outputs dispatch instruction JSON.
func dispatchFixer(phaseNum int, fixerMode string) error {
	// Validate mode
	if !validFixerModes[fixerMode] {
		return fmt.Errorf("invalid fixer mode %q: valid modes are full, propose, advise", fixerMode)
	}

	// Check circuit breaker
	if blocked, msg := isFixerDispatchBlocked(phaseNum); blocked {
		return fmt.Errorf("%s", msg)
	}

	// Check attempt cap
	if err := checkAttemptCap(phaseNum, DefaultMaxUnblockAttempts); err != nil {
		return err
	}

	// Read gate results for dispatch context
	fileData, err := readGateResultsPhase(phaseNum)
	if err != nil {
		return fmt.Errorf("failed to read gate results for phase %d: %w", phaseNum, err)
	}

	// Increment attempt count
	if err := incrementUnblockAttempts(phaseNum); err != nil {
		return fmt.Errorf("failed to increment unblock attempts: %w", err)
	}

	// Build list of failed gates
	var failedGates []map[string]string
	for _, r := range fileData.Results {
		if r.Status == "failed" {
			gate := map[string]string{
				"name":     r.Name,
				"detail":   r.Detail,
				"fix_hint": r.FixHint,
			}
			failedGates = append(failedGates, gate)
		}
	}

	// Build fix hint summary
	var hints []string
	for _, r := range fileData.Results {
		if r.Status == "failed" && r.FixHint != "" {
			hints = append(hints, fmt.Sprintf("%s: %s", r.Name, r.FixHint))
		}
	}
	fixHint := strings.Join(hints, "; ")

	// Emit fixer_dispatch loop break event (LOOP-04)
	emitLoopBreakEvent("fixer_dispatch",
		fmt.Sprintf("%d failed gates in Phase %d", len(failedGates), phaseNum),
		fmt.Sprintf("dispatching Fixer in %s mode (attempt %d)", fixerMode, fileData.Attempts),
		"aether-unblock")

	// Output dispatch instruction JSON for wrapper to consume
	dispatch := map[string]interface{}{
		"mode":         "fixer_dispatch",
		"phase":        phaseNum,
		"fixer_mode":   fixerMode,
		"failed_gates": failedGates,
		"fix_hint":     fixHint,
		"attempt":      fileData.Attempts,
	}

	data, err := json.MarshalIndent(dispatch, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dispatch instruction: %w", err)
	}

	if shouldRenderVisualOutput(stderr) {
		fmt.Fprintf(stderr, "Fixer dispatched (attempt %d) -- %s mode\n", fileData.Attempts, fixerMode)
		fmt.Fprintf(stderr, "Investigating %d failed gate(s) in Phase %d\n", len(failedGates), phaseNum)
	}
	fmt.Fprintln(stdout, string(data))

	return nil
}

// resolveFixedGates updates gate results for gates that the Fixer addressed.
// Addressed gates are marked as "passed" with cleared detail/fixHint/recoveryOptions.
// Gates not in addressedGateNames are left unchanged (GATE-07 per D-07).
// Unknown gate names are silently ignored (T-89-03 mitigation).
func resolveFixedGates(phaseNum int, addressedGateNames []string) error {
	fileData, err := readGateResultsPhase(phaseNum)
	if err != nil {
		return fmt.Errorf("failed to read gate results for phase %d: %w", phaseNum, err)
	}

	// Build set of addressed gate names for O(1) lookup
	addressed := make(map[string]bool, len(addressedGateNames))
	for _, name := range addressedGateNames {
		addressed[name] = true
	}

	resolvedCount := 0
	now := time.Now().UTC().Format(time.RFC3339)

	for i := range fileData.Results {
		if !addressed[fileData.Results[i].Name] {
			continue
		}
		fileData.Results[i].Status = "passed"
		fileData.Results[i].Timestamp = now
		fileData.Results[i].Detail = ""
		fileData.Results[i].FixHint = ""
		fileData.Results[i].RecoveryOptions = nil
		resolvedCount++
	}

	// Write updated results back
	rel := fmt.Sprintf("gate-results-%d.json", phaseNum)
	if err := store.SaveJSON(rel, fileData); err != nil {
		return fmt.Errorf("failed to write gate results for phase %d: %w", phaseNum, err)
	}

	// Emit fixer_complete loop break event (LOOP-04)
	emitLoopBreakEvent("fixer_complete",
		fmt.Sprintf("%d gates resolved in Phase %d", resolvedCount, phaseNum),
		fmt.Sprintf("gate results updated for %d addressed gates", resolvedCount),
		"aether-unblock")

	return nil
}

// recordFixerFailure records a Fixer failure in the circuit breaker and emits telemetry.
func recordFixerFailure(phaseNum int, errMsg string) {
	// Record failure in circuit breaker for the relevant gate key
	cb := globalCircuitBreaker
	if cb != nil {
		// Parse the gate name from the error message (format: "gate_name: error detail")
		parts := strings.SplitN(errMsg, ":", 2)
		if len(parts) >= 1 {
			gateName := strings.TrimSpace(parts[0])
			if gateName != "" {
				cb.RecordFailure(gateRetryKey(phaseNum, gateName))
			}
		}
	}

	// Emit fixer_failed loop break event (LOOP-04)
	emitLoopBreakEvent("fixer_failed",
		fmt.Sprintf("Fixer failed for Phase %d: %s", phaseNum, errMsg),
		"manual intervention may be required",
		"aether-unblock")
}
