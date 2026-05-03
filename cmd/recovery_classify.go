package cmd

import (
	"strings"
)

// FailureClassification represents how a worker failure should be handled.
// Classification is deterministic and code-level -- never user-configurable.
type FailureClassification string

const (
	Recoverable     FailureClassification = "recoverable"
	RequiresAttempt FailureClassification = "requires-attempt"
	Blocking        FailureClassification = "blocking"
)

// FailureType distinguishes environmental (transient) from fundamental (systemic) failures.
type FailureType string

const (
	Transient FailureType = "transient"
	Systemic  FailureType = "systemic"
)

// failureClassificationEntry records a failure pattern's classification, type, and rationale.
type failureClassificationEntry struct {
	Classification FailureClassification
	FailureType    FailureType
	Rationale      string
}

// failureClassifications maps worker status values and error patterns
// to deterministic failure classifications.
// This is a read-only constant -- no configuration can change these values.
var failureClassifications = map[string]failureClassificationEntry{
	// Transient failures (recoverable): environmental hiccups -- per D-08
	"timeout":          {Recoverable, Transient, "Worker timed out -- environment may recover on retry"},
	"context_overflow": {Recoverable, Transient, "Context window exceeded -- shorter retry may succeed"},
	"resource_limit":   {Recoverable, Transient, "Temporary resource constraint -- retriable"},
	"cancelled":        {Recoverable, Transient, "Worker was cancelled -- retriable with fresh invocation"},
	// Systemic failures (blocking): fundamental problems -- per D-09
	"bad_task_spec":      {Blocking, Systemic, "Task specification is invalid -- retrying won't help"},
	"missing_dependency": {Blocking, Systemic, "Required dependency not found -- must be fixed first"},
	"invalid_file_path":  {Blocking, Systemic, "File path error -- structural issue in task definition"},
	"structural_error":   {Blocking, Systemic, "Code structure error -- requires human inspection"},
	// Requires-attempt: ambiguous, try once -- per D-10
	"partial_completion":   {RequiresAttempt, Transient, "Worker completed some but not all tasks -- one retry may help"},
	"unparseable_output":   {RequiresAttempt, Systemic, "Worker output was garbled -- may indicate deeper issue"},
	"failed":              {RequiresAttempt, Systemic, "Generic failure -- safe middle ground, one attempt allowed"},
	"blocked":             {RequiresAttempt, Systemic, "Worker was blocked -- may resolve on retry or may indicate conflict"},
	"manually-reconciled": {RequiresAttempt, Systemic, "Manual reconciliation -- one retry may stabilize"},
}

// classifyWorkerFailure returns the classification, failure type, and rationale
// for a worker failure. Classification is deterministic -- never LLM-inferred.
// Unknown patterns default to requires-attempt per D-11.
func classifyWorkerFailure(status string, errMsg string) (FailureClassification, FailureType, string) {
	normalized := strings.ToLower(strings.TrimSpace(status))

	// Direct status match from registry
	if entry, ok := failureClassifications[normalized]; ok {
		return entry.Classification, entry.FailureType, entry.Rationale
	}

	// Error message pattern matching for ambiguous statuses
	lower := strings.ToLower(errMsg)
	switch {
	case strings.Contains(lower, "context window"),
		strings.Contains(lower, "token limit"),
		strings.Contains(lower, "max tokens"):
		return Recoverable, Transient, "Context overflow detected from error message"
	case strings.Contains(lower, "no such file"),
		strings.Contains(lower, "file not found"):
		return Blocking, Systemic, "Missing file detected from error message"
	case strings.Contains(lower, "permission denied"):
		return Blocking, Systemic, "Permission error detected from error message"
	}

	// Default: requires-attempt (safe middle ground per D-11)
	return RequiresAttempt, Systemic, "Unknown failure pattern -- defaulting to requires-attempt for safety"
}

// FailureRecord captures a worker failure with classification metadata.
// Phase 96 will consume these records to drive retry/reassign/fixer dispatch.
type FailureRecord struct {
	WorkerName     string                `json:"worker_name"`
	TaskID         string                `json:"task_id,omitempty"`
	Caste          string                `json:"caste,omitempty"`
	Phase          int                   `json:"phase"`
	Status         string                `json:"status"`
	Classification FailureClassification `json:"classification"`
	FailureType    FailureType           `json:"failure_type"`
	ErrorMessage   string                `json:"error_message"`
	Timestamp      string                `json:"timestamp"`
	RetryCount     int                   `json:"retry_count,omitempty"`
}

// RecoveryLogEntry records a recovery action taken for a worker failure.
// Each entry captures what was tried, what happened, and the original failure context.
type RecoveryLogEntry struct {
	ID            string        `json:"id"`
	Failure       FailureRecord `json:"failure"`
	ActionTaken   string        `json:"action_taken"`
	Outcome       string        `json:"outcome"`
	AttemptNumber int           `json:"attempt_number"`
	Timestamp     string        `json:"timestamp"`
	Detail        string        `json:"detail,omitempty"`
}

// RecoveryLogFile wraps per-phase recovery log entries for persistence.
type RecoveryLogFile struct {
	Phase   int                `json:"phase"`
	Entries []RecoveryLogEntry `json:"entries"`
}
