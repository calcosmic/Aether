package cmd

import (
	"fmt"
	"os"
	"time"
)

// QueenDecision represents the queen's recommendation for a single gate.
// Per D-01: each decision includes gate name, status, classification tier,
// queen recommendation, rationale, and auto-resolve eligibility.
type QueenDecision struct {
	GateName            string                `json:"gate_name"`
	Status              string                `json:"status"`               // "passed", "failed"
	ClassificationTier  string                `json:"classification_tier"`  // "hard_block", "soft_block", "advisory", ""
	QueenRecommendation string                `json:"queen_recommendation"` // "pass", "auto-resolve", "dispatch-fixer", "escalate"
	Rationale           string                `json:"rationale"`
	AutoResolveEligible bool                  `json:"auto_resolve_eligible"`
	RecoveryPreview     *QueenRecoveryPreview `json:"recovery_preview,omitempty"`
}

// QueenRecoveryPreview describes what would happen if recovery is needed for a gate.
// Per D-02, D-04: generated for ALL gates (passing and failing).
type QueenRecoveryPreview struct {
	Classification    string `json:"classification"`     // FailureClassification: "recoverable", "requires-attempt", "blocking"
	FirstAction       string `json:"first_action"`       // "retry", "escalate", "fixer_dispatch"
	BudgetRemaining   int    `json:"budget_remaining"`
	WouldAutoResolve  bool   `json:"would_auto_resolve"`  // based on tier == softBlock and budget > 0
	WouldEscalate     bool   `json:"would_escalate"`      // true for hardBlock or exhausted budget
}

// QueenStateFile persists the queen's decision list, budget snapshot, and escalation log.
// Per D-11: stored as queen-state-{N}.json in the data directory.
type QueenStateFile struct {
	Phase                 int              `json:"phase"`
	GeneratedAt           string           `json:"generated_at"`
	Decisions             []QueenDecision  `json:"decisions"`
	BudgetSnapshot        *RecoveryBudget  `json:"budget_snapshot,omitempty"`
	EscalationLog         []EscalationEntry `json:"escalation_log,omitempty"`
	BreakerTrippedWorkers []string         `json:"breaker_tripped_workers,omitempty"`
}

// EscalationEntry records a circuit breaker escalation event.
// Per D-12: captures breaker state, tripped workers, and action taken.
type EscalationEntry struct {
	Timestamp         string   `json:"timestamp"`
	GateName          string   `json:"gate_name,omitempty"`
	WorkerName        string   `json:"worker_name,omitempty"`
	BreakerTripped    []string `json:"breaker_tripped_workers"`
	EscalationAction  string   `json:"escalation_action"` // "escalate_to_human", "skip_retry"
	Rationale         string   `json:"rationale"`
}

// queenDecide produces a decision list for every gate in the gate report.
// Per D-05: this is a pure function -- it does NOT call budget.consume() or breaker.Reset().
// Per D-10: single-invocation contract (no goroutines, no daemon).
//
// Returns one QueenDecision per gate in gates.Checks.
func queenDecide(gates codexContinueGateReport, budget *RecoveryBudget, breaker *CircuitBreaker, phaseNum int, reviewDepth string) []QueenDecision {
	remaining := 0
	used := 0
	total := 0
	if budget != nil {
		remaining = budget.remaining()
		used = budget.totalUsed()
		total = budget.TotalBudget
	}

	var trippedWorkers []string
	if breaker != nil {
		trippedWorkers = breaker.TrippedWorkers()
	}

	decisions := make([]QueenDecision, 0, len(gates.Checks))
	for _, check := range gates.Checks {
		tier, tierRationale := gateClassify(check.Name)

		status := "passed"
		if !check.Passed {
			status = "failed"
		}

		// Determine queen recommendation
		recommendation := determineRecommendation(status, tier, remaining)

		// Determine auto-resolve eligibility
		autoResolveEligible := tier == softBlock && status == "failed"

		// Build rationale
		rationale := buildRationale(check.Name, status, tier, tierRationale, recommendation, remaining, used, total, trippedWorkers)

		// Build recovery preview (per D-04: ALL gates get a preview)
		preview := buildRecoveryPreview(tier, status, remaining)

		decisions = append(decisions, QueenDecision{
			GateName:            check.Name,
			Status:              status,
			ClassificationTier:  string(tier),
			QueenRecommendation: recommendation,
			Rationale:           rationale,
			AutoResolveEligible: autoResolveEligible,
			RecoveryPreview:     preview,
		})
	}

	return decisions
}

// determineRecommendation returns the queen's recommendation based on gate status, tier, and budget.
func determineRecommendation(status string, tier GateClassificationTier, budgetRemaining int) string {
	if status == "passed" {
		return "pass"
	}

	// Failed gate -- determine action based on tier and budget
	switch tier {
	case hardBlock:
		return "escalate"
	case softBlock:
		if budgetRemaining > 0 {
			return "auto-resolve"
		}
		return "escalate"
	case advisory:
		return "escalate"
	default: // unclassified
		return "escalate"
	}
}

// buildRationale constructs a human-readable rationale for the queen's decision.
func buildRationale(gateName, status string, tier GateClassificationTier, tierRationale, recommendation string, remaining, used, total int, trippedWorkers []string) string {
	tierStr := string(tier)
	if tierStr == "" {
		tierStr = "unclassified"
	}

	var rationale string
	if status == "passed" {
		rationale = fmt.Sprintf("%s gate passed", tierStr)
	} else {
		rationale = fmt.Sprintf("%s gate failed; %s; budget %d/%d", tierStr, recommendation, used, total)
	}

	// Append breaker state if any workers are tripped
	if len(trippedWorkers) > 0 {
		rationale += fmt.Sprintf("; breaker tripped: %v", trippedWorkers)
	}

	return rationale
}

// buildRecoveryPreview creates a recovery preview for a gate.
// Per D-04: generated for ALL gates (passing and failing).
func buildRecoveryPreview(tier GateClassificationTier, status string, budgetRemaining int) *QueenRecoveryPreview {
	preview := &QueenRecoveryPreview{
		BudgetRemaining: budgetRemaining,
	}

	switch tier {
	case hardBlock:
		preview.Classification = "blocking"
		preview.FirstAction = "escalate"
		preview.WouldEscalate = true
		preview.WouldAutoResolve = false
	case softBlock:
		preview.Classification = "recoverable"
		preview.FirstAction = "retry"
		preview.WouldAutoResolve = status == "failed" && budgetRemaining > 0
		preview.WouldEscalate = status == "failed" && budgetRemaining == 0
	case advisory:
		preview.Classification = "recoverable"
		preview.FirstAction = "escalate"
		preview.WouldEscalate = status == "failed"
		preview.WouldAutoResolve = false
	default: // unclassified
		preview.Classification = "requires-attempt"
		preview.FirstAction = "escalate"
		preview.WouldEscalate = status == "failed"
		preview.WouldAutoResolve = false
	}

	return preview
}

// queenStateWrite persists the queen state file to queen-state-{phaseNum}.json.
// Follows the same pattern as gateResultsWritePhase.
func queenStateWrite(phaseNum int, state QueenStateFile) error {
	rel := fmt.Sprintf("queen-state-%d.json", phaseNum)
	return store.SaveJSON(rel, state)
}

// queenStateRead reads the queen state file from queen-state-{phaseNum}.json.
// Returns an error if the file does not exist or cannot be read.
func queenStateRead(phaseNum int) (QueenStateFile, error) {
	var state QueenStateFile
	rel := fmt.Sprintf("queen-state-%d.json", phaseNum)
	err := store.LoadJSON(rel, &state)
	return state, err
}

// queenLogEscalation appends an escalation entry to the queen-state file.
// Per D-12: records breaker state, tripped workers, and action taken.
// Creates the file if it does not exist.
func queenLogEscalation(phaseNum int, trippedWorkers []string, rationale string) {
	// Read existing state (create fresh if file doesn't exist)
	state, err := queenStateRead(phaseNum)
	if err != nil {
		state = QueenStateFile{
			Phase:       phaseNum,
			GeneratedAt: time.Now().Format(time.RFC3339),
			Decisions:   []QueenDecision{},
		}
	}

	// Append escalation entry
	state.EscalationLog = append(state.EscalationLog, EscalationEntry{
		Timestamp:        time.Now().Format(time.RFC3339),
		BreakerTripped:   trippedWorkers,
		EscalationAction: "escalate_to_human",
		Rationale:        rationale,
	})

	// Update breaker tripped workers list
	state.BreakerTrippedWorkers = trippedWorkers

	// Write back
	if err := queenStateWrite(phaseNum, state); err != nil {
		// Log but don't fail -- escalation logging is best-effort
		fmt.Fprintf(os.Stderr, "warning: failed to persist escalation log: %v\n", err)
	}
}
