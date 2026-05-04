package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
)

// RecoveryBudget tracks per-wave recovery action consumption.
// Per D-10: default total budget is 3, resets when a new wave starts.
// Per D-11: lives in the recovery-log file, not COLONY_STATE.json.
type RecoveryBudget struct {
	TotalBudget         int `json:"total_budget"`
	RetriesUsed         int `json:"retries_used"`
	ReassignsUsed       int `json:"reassigns_used"`
	FixerDispatchesUsed int `json:"fixer_dispatches_used"`
	Wave                int `json:"wave"`
}

// consume decrements the appropriate counter for the given action type.
// Returns false if the total budget is exhausted (all used counters sum >= total_budget).
func (b *RecoveryBudget) consume(action string) bool {
	used := b.totalUsed()
	if used >= b.TotalBudget {
		return false
	}
	switch action {
	case "retry":
		b.RetriesUsed++
	case "peer_reassignment":
		b.ReassignsUsed++
	case "fixer_dispatch":
		b.FixerDispatchesUsed++
	}
	return true
}

// totalUsed returns the sum of all consumed action counts.
func (b *RecoveryBudget) totalUsed() int {
	return b.RetriesUsed + b.ReassignsUsed + b.FixerDispatchesUsed
}

// resetForWave clears all counters and sets the new wave number.
// Per D-10: budget resets when a new wave starts.
func (b *RecoveryBudget) resetForWave(wave int) {
	b.RetriesUsed = 0
	b.ReassignsUsed = 0
	b.FixerDispatchesUsed = 0
	b.TotalBudget = 3
	b.Wave = wave
}

// remaining returns how many recovery actions are still available.
func (b *RecoveryBudget) remaining() int {
	r := b.TotalBudget - b.totalUsed()
	if r < 0 {
		return 0
	}
	return r
}

// newRecoveryBudget creates a new budget with default total of 3 for the given wave.
func newRecoveryBudget(wave int) *RecoveryBudget {
	return &RecoveryBudget{
		TotalBudget: 3,
		Wave:        wave,
	}
}

// budgetFromRecoveryLog reads the recovery-log file and returns the budget.
// If the file has no budget (legacy), or the file does not exist, returns a fresh default budget.
func budgetFromRecoveryLog(phaseNum int, wave int) *RecoveryBudget {
	file, err := recoveryLogReadPhase(phaseNum)
	if err != nil {
		return newRecoveryBudget(wave)
	}
	if file.RecoveryBudget != nil {
		return file.RecoveryBudget
	}
	return newRecoveryBudget(wave)
}

// persistBudgetToRecoveryLog writes the budget back to the recovery-log file.
// Reads existing entries first to preserve them, then writes the budget alongside.
func persistBudgetToRecoveryLog(phaseNum int, budget *RecoveryBudget) error {
	file, err := recoveryLogReadPhase(phaseNum)
	if err != nil {
		file = RecoveryLogFile{Phase: phaseNum, Entries: []RecoveryLogEntry{}}
	}
	file.RecoveryBudget = budget
	rel := fmt.Sprintf("recovery-log-%d.json", phaseNum)
	return store.SaveJSON(rel, file)
}

// RecoveryAction represents a single recovery action to be taken.
type RecoveryAction struct {
	Type            string // "retry", "peer_reassignment", "fixer_dispatch", "escalate"
	WorkerName      string
	PeerName        string // set when Type == "peer_reassignment"
	Detail          string
	BudgetRemaining int
}

// RecoveryContext provides all input needed for the orchestrator to make a recovery decision.
type RecoveryContext struct {
	Phase           int
	Wave            int
	WorkerName      string
	TaskID          string
	Caste           string
	Status          string
	ErrorMessage    string
	Dispatches      []codex.WorkerDispatch
	CircuitBreaker  *CircuitBreaker
	Budget          *RecoveryBudget
	RecoveryHistory []RecoveryAction
}

// RecoveryOutcome captures the orchestrator's decision for a single failure.
type RecoveryOutcome struct {
	Classification FailureClassification
	FailureType    FailureType
	Rationale      string
	Action         RecoveryAction
	Exhausted      bool // true when no further recovery possible
	LogEntries     []RecoveryLogEntry
}

// orchestrateRecovery is the core auto-recovery decision function.
// It classifies the failure and returns the next recovery action based on
// the classification, recovery history, circuit breaker state, and budget.
//
// Per D-01: Recovery sequence is classification-dependent.
// Per D-06: The orchestrator is a pure function -- classify, decide, log, return.
func orchestrateRecovery(ctx RecoveryContext) RecoveryOutcome {
	classification, failType, rationale := classifyWorkerFailure(ctx.Status, ctx.ErrorMessage)

	// Build the failure record for logging
	now := time.Now().UTC().Format(time.RFC3339)
	failureRecord := FailureRecord{
		WorkerName:     ctx.WorkerName,
		TaskID:         ctx.TaskID,
		Caste:          ctx.Caste,
		Phase:          ctx.Phase,
		Status:         ctx.Status,
		Classification: classification,
		FailureType:    failType,
		ErrorMessage:   ctx.ErrorMessage,
		Timestamp:      now,
		RetryCount:     countRetries(ctx.RecoveryHistory),
	}

	switch classification {
	case Blocking:
		// Per D-04: immediate escalation, no retry, no reassignment, no budget consumed
		return RecoveryOutcome{
			Classification: classification,
			FailureType:    failType,
			Rationale:      rationale,
			Action: RecoveryAction{
				Type:            "escalate",
				WorkerName:      ctx.WorkerName,
				Detail:          rationale,
				BudgetRemaining: ctx.Budget.remaining(),
			},
			Exhausted: true,
			LogEntries: []RecoveryLogEntry{
				{
					ID:            fmt.Sprintf("ro-%d-%d", ctx.Phase, time.Now().UnixNano()),
					Failure:       failureRecord,
					ActionTaken:   "escalate",
					Outcome:       "blocking failure -- no recovery attempted",
					AttemptNumber: 0,
					Timestamp:     now,
					Detail:        rationale,
				},
			},
		}

	case RequiresAttempt:
		return sequenceRequiresAttempt(ctx, classification, failType, rationale, failureRecord, now)

	case Recoverable:
		return sequenceRecoverable(ctx, classification, failType, rationale, failureRecord, now)
	}

	// Unknown classification -- escalate safely
	return RecoveryOutcome{
		Classification: classification,
		FailureType:    failType,
		Rationale:      rationale,
		Action: RecoveryAction{
			Type:       "escalate",
			WorkerName: ctx.WorkerName,
			Detail:     rationale,
		},
		Exhausted: true,
	}
}

// sequenceRequiresAttempt handles the requires-attempt recovery sequence.
// Per D-03: retry once, then Fixer dispatch. No peer reassignment.
// Per D-07: retry -> fixer_dispatch. No peer.
func sequenceRequiresAttempt(ctx RecoveryContext, classification FailureClassification, failType FailureType, rationale string, failureRecord FailureRecord, now string) RecoveryOutcome {
	hasRetry := hasActionType(ctx.RecoveryHistory, "retry")
	hasFixer := hasActionType(ctx.RecoveryHistory, "fixer_dispatch")

	if !hasRetry {
		// Check budget
		if ctx.Budget.totalUsed() >= ctx.Budget.TotalBudget {
			return escalateOutcome(ctx, classification, failType, rationale, failureRecord, now, "budget exhausted")
		}
		// Check circuit breaker
		if ctx.CircuitBreaker != nil && !ctx.CircuitBreaker.Allow(ctx.WorkerName) {
			// Breaker tripped -- skip retry, go to fixer (for requires-attempt, no peer)
			return fixerDispatchOutcome(ctx, classification, failType, rationale, failureRecord, now)
		}
		ctx.Budget.consume("retry")
		return RecoveryOutcome{
			Classification: classification,
			FailureType:    failType,
			Rationale:      rationale,
			Action: RecoveryAction{
				Type:            "retry",
				WorkerName:      ctx.WorkerName,
				Detail:          "requires-attempt: retrying once",
				BudgetRemaining: ctx.Budget.remaining(),
			},
			Exhausted: false,
			LogEntries: []RecoveryLogEntry{
				{
					ID:            fmt.Sprintf("ro-%d-%d", ctx.Phase, time.Now().UnixNano()),
					Failure:       failureRecord,
					ActionTaken:   "retry",
					Outcome:       "requires-attempt: retrying once",
					AttemptNumber: countRetries(ctx.RecoveryHistory) + 1,
					Timestamp:     now,
					Detail:        rationale,
				},
			},
		}
	}

	if !hasFixer {
		// Check budget
		if ctx.Budget.totalUsed() >= ctx.Budget.TotalBudget {
			return escalateOutcome(ctx, classification, failType, rationale, failureRecord, now, "budget exhausted")
		}
		return fixerDispatchOutcome(ctx, classification, failType, rationale, failureRecord, now)
	}

	// All recovery attempted
	return escalateOutcome(ctx, classification, failType, rationale, failureRecord, now, "all recovery actions exhausted")
}

// sequenceRecoverable handles the recoverable (transient) failure sequence.
// Per D-02: retry -> peer_reassignment -> fixer_dispatch -> escalate.
// Per D-07: retry -> peer reassign -> fixer_dispatch.
func sequenceRecoverable(ctx RecoveryContext, classification FailureClassification, failType FailureType, rationale string, failureRecord FailureRecord, now string) RecoveryOutcome {
	hasRetry := hasActionType(ctx.RecoveryHistory, "retry")
	hasPeer := hasActionType(ctx.RecoveryHistory, "peer_reassignment")
	hasFixer := hasActionType(ctx.RecoveryHistory, "fixer_dispatch")

	// Check budget first
	if ctx.Budget.totalUsed() >= ctx.Budget.TotalBudget {
		return escalateOutcome(ctx, classification, failType, rationale, failureRecord, now, "budget exhausted")
	}

	if !hasRetry {
		// Check circuit breaker before retry
		if ctx.CircuitBreaker != nil && !ctx.CircuitBreaker.Allow(ctx.WorkerName) {
			// Per Pitfall 2: tripped worker skips retry, goes to peer
			return peerReassignmentOutcome(ctx, classification, failType, rationale, failureRecord, now)
		}
		ctx.Budget.consume("retry")
		return RecoveryOutcome{
			Classification: classification,
			FailureType:    failType,
			Rationale:      rationale,
			Action: RecoveryAction{
				Type:            "retry",
				WorkerName:      ctx.WorkerName,
				Detail:          "recoverable: retrying worker",
				BudgetRemaining: ctx.Budget.remaining(),
			},
			Exhausted: false,
			LogEntries: []RecoveryLogEntry{
				{
					ID:            fmt.Sprintf("ro-%d-%d", ctx.Phase, time.Now().UnixNano()),
					Failure:       failureRecord,
					ActionTaken:   "retry",
					Outcome:       "recoverable: retrying worker",
					AttemptNumber: countRetries(ctx.RecoveryHistory) + 1,
					Timestamp:     now,
					Detail:        rationale,
				},
			},
		}
	}

	if !hasPeer {
		return peerReassignmentOutcome(ctx, classification, failType, rationale, failureRecord, now)
	}

	if !hasFixer {
		// Check budget
		if ctx.Budget.totalUsed() >= ctx.Budget.TotalBudget {
			return escalateOutcome(ctx, classification, failType, rationale, failureRecord, now, "budget exhausted")
		}
		return fixerDispatchOutcome(ctx, classification, failType, rationale, failureRecord, now)
	}

	// All recovery attempted
	return escalateOutcome(ctx, classification, failType, rationale, failureRecord, now, "all recovery actions exhausted")
}

// peerReassignmentOutcome returns the peer reassignment action if a peer is available,
// or falls through to Fixer dispatch if no peer exists.
func peerReassignmentOutcome(ctx RecoveryContext, classification FailureClassification, failType FailureType, rationale string, failureRecord FailureRecord, now string) RecoveryOutcome {
	// Check budget
	if ctx.Budget.totalUsed() >= ctx.Budget.TotalBudget {
		return escalateOutcome(ctx, classification, failType, rationale, failureRecord, now, "budget exhausted")
	}

	// Find the current dispatch for peer lookup
	var currentDispatch codex.WorkerDispatch
	found := false
	for _, d := range ctx.Dispatches {
		if d.WorkerName == ctx.WorkerName {
			currentDispatch = d
			found = true
			break
		}
	}

	if found && ctx.CircuitBreaker != nil {
		peer := findSameCastePeer(ctx.Dispatches, currentDispatch, ctx.CircuitBreaker)
		if peer != nil {
			ctx.Budget.consume("peer_reassignment")
			return RecoveryOutcome{
				Classification: classification,
				FailureType:    failType,
				Rationale:      rationale,
				Action: RecoveryAction{
					Type:            "peer_reassignment",
					WorkerName:      ctx.WorkerName,
					PeerName:        peer.WorkerName,
					Detail:          fmt.Sprintf("reassigning to same-caste peer %s", peer.WorkerName),
					BudgetRemaining: ctx.Budget.remaining(),
				},
				Exhausted: false,
				LogEntries: []RecoveryLogEntry{
					{
						ID:            fmt.Sprintf("ro-%d-%d", ctx.Phase, time.Now().UnixNano()),
						Failure:       failureRecord,
						ActionTaken:   "peer_reassignment",
						Outcome:       fmt.Sprintf("reassigned to %s", peer.WorkerName),
						AttemptNumber: countRetries(ctx.RecoveryHistory) + 1,
						Timestamp:     now,
						Detail:        fmt.Sprintf("original worker %s failed; reassigned to %s", ctx.WorkerName, peer.WorkerName),
					},
				},
			}
		}
	}

	// No peer available -- fall through to fixer dispatch
	if ctx.Budget.totalUsed() >= ctx.Budget.TotalBudget {
		return escalateOutcome(ctx, classification, failType, rationale, failureRecord, now, "budget exhausted")
	}
	return fixerDispatchOutcome(ctx, classification, failType, rationale, failureRecord, now)
}

// fixerDispatchOutcome returns the Fixer dispatch action with recovery history context.
// Per D-08: Fixer receives context about the recovery sequence so far.
func fixerDispatchOutcome(ctx RecoveryContext, classification FailureClassification, failType FailureType, rationale string, failureRecord FailureRecord, now string) RecoveryOutcome {
	ctx.Budget.consume("fixer_dispatch")
	historySummary := recoveryHistorySummary(ctx.RecoveryHistory)

	return RecoveryOutcome{
		Classification: classification,
		FailureType:    failType,
		Rationale:      rationale,
		Action: RecoveryAction{
			Type:            "fixer_dispatch",
			WorkerName:      ctx.WorkerName,
			Detail:          historySummary,
			BudgetRemaining: ctx.Budget.remaining(),
		},
		Exhausted: false,
		LogEntries: []RecoveryLogEntry{
			{
				ID:            fmt.Sprintf("ro-%d-%d", ctx.Phase, time.Now().UnixNano()),
				Failure:       failureRecord,
				ActionTaken:   "fixer_dispatch",
				Outcome:       "dispatching Fixer as recovery strategy",
				AttemptNumber: countRetries(ctx.RecoveryHistory) + 1,
				Timestamp:     now,
				Detail:        historySummary,
			},
		},
	}
}

// escalateOutcome returns the escalate action. No further recovery possible.
func escalateOutcome(ctx RecoveryContext, classification FailureClassification, failType FailureType, rationale string, failureRecord FailureRecord, now string, reason string) RecoveryOutcome {
	return RecoveryOutcome{
		Classification: classification,
		FailureType:    failType,
		Rationale:      rationale,
		Action: RecoveryAction{
			Type:            "escalate",
			WorkerName:      ctx.WorkerName,
			Detail:          fmt.Sprintf("%s: %s", reason, rationale),
			BudgetRemaining: ctx.Budget.remaining(),
		},
		Exhausted: true,
		LogEntries: []RecoveryLogEntry{
			{
				ID:            fmt.Sprintf("ro-%d-%d", ctx.Phase, time.Now().UnixNano()),
				Failure:       failureRecord,
				ActionTaken:   "escalate",
				Outcome:       reason,
				AttemptNumber: countRetries(ctx.RecoveryHistory),
				Timestamp:     now,
				Detail:        fmt.Sprintf("escalated: %s (%s)", reason, rationale),
			},
		},
	}
}

// recoveryHistorySummary builds a string describing all previous recovery actions.
// Per D-08: Fixer receives context about the recovery sequence so far.
func recoveryHistorySummary(actions []RecoveryAction) string {
	if len(actions) == 0 {
		return "no previous recovery actions"
	}
	var parts []string
	for _, a := range actions {
		switch a.Type {
		case "retry":
			parts = append(parts, fmt.Sprintf("retry failed for %s", a.WorkerName))
		case "peer_reassignment":
			parts = append(parts, fmt.Sprintf("peer reassignment from %s to %s failed", a.WorkerName, a.PeerName))
		case "fixer_dispatch":
			parts = append(parts, fmt.Sprintf("fixer dispatch for %s failed", a.WorkerName))
		}
	}
	return strings.Join(parts, "; ")
}

// hasActionType checks if a given action type exists in the recovery history.
func hasActionType(history []RecoveryAction, actionType string) bool {
	for _, a := range history {
		if a.Type == actionType {
			return true
		}
	}
	return false
}

// countRetries counts the number of retry actions in the recovery history.
func countRetries(history []RecoveryAction) int {
	count := 0
	for _, a := range history {
		if a.Type == "retry" {
			count++
		}
	}
	return count
}
