package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
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

// recoveryLogWritePhase persists recovery log entries to recovery-log-{N}.json.
func recoveryLogWritePhase(phaseNum int, entries []RecoveryLogEntry) error {
	rel := fmt.Sprintf("recovery-log-%d.json", phaseNum)
	file := RecoveryLogFile{
		Phase:   phaseNum,
		Entries: entries,
	}
	return store.SaveJSON(rel, file)
}

// recoveryLogReadPhase reads recovery log entries from recovery-log-{N}.json.
func recoveryLogReadPhase(phaseNum int) (RecoveryLogFile, error) {
	rel := fmt.Sprintf("recovery-log-%d.json", phaseNum)
	var file RecoveryLogFile
	if err := store.LoadJSON(rel, &file); err != nil {
		return RecoveryLogFile{}, err
	}
	return file, nil
}

// --- Cobra CLI subcommands for failure classification and recovery logs ---

var failureClassifyCmd = &cobra.Command{
	Use:          "failure-classify",
	Short:        "Show failure classification rules and rationale",
	Long:         "Display all failure classifications (recoverable, requires-attempt, blocking) with rationale.\nUse --json for structured output.",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			outputOK(failureClassifications)
			return nil
		}
		renderFailureClassifyTable()
		return nil
	},
}

func renderFailureClassifyTable() {
	t := table.NewWriter()
	t.AppendHeader(table.Row{"Pattern", "Classification", "Failure Type", "Rationale"})

	type entry struct {
		pattern string
		failureClassificationEntry
	}
	var entries []entry
	for pattern, e := range failureClassifications {
		entries = append(entries, entry{pattern, e})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Classification != entries[j].Classification {
			return entries[i].Classification < entries[j].Classification
		}
		return entries[i].pattern < entries[j].pattern
	})

	for _, e := range entries {
		t.AppendRow(table.Row{e.pattern, string(e.Classification), string(e.FailureType), e.Rationale})
	}
	fmt.Fprintln(stdout, t.Render())
}

var recoveryLogReadCmd = &cobra.Command{
	Use:          "recovery-log-read",
	Short:        "Read the recovery log for a phase",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		phaseNum, _ := cmd.Flags().GetInt("phase")
		if phaseNum <= 0 {
			outputErrorMessage("--phase is required")
			return nil
		}
		file, err := recoveryLogReadPhase(phaseNum)
		if err != nil {
			outputOK(map[string]interface{}{"entries": []RecoveryLogEntry{}, "phase": phaseNum, "total": 0})
			return nil
		}
		outputOK(map[string]interface{}{
			"entries": file.Entries,
			"phase":   file.Phase,
			"total":   len(file.Entries),
		})
		return nil
	},
}

var recoveryLogWriteCmd = &cobra.Command{
	Use:          "recovery-log-write",
	Short:        "Write a recovery log entry for a phase",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		phaseNum, _ := cmd.Flags().GetInt("phase")
		if phaseNum <= 0 {
			outputErrorMessage("--phase is required")
			return nil
		}
		worker := mustGetString(cmd, "worker")
		if worker == "" {
			outputErrorMessage("--worker is required")
			return nil
		}
		status := mustGetString(cmd, "status")
		if status == "" {
			outputErrorMessage("--status is required")
			return nil
		}
		errMsg, _ := cmd.Flags().GetString("error")
		action := mustGetString(cmd, "action")
		if action == "" {
			outputErrorMessage("--action is required")
			return nil
		}
		outcome := mustGetString(cmd, "outcome")
		if outcome == "" {
			outputErrorMessage("--outcome is required")
			return nil
		}
		attempt, _ := cmd.Flags().GetInt("attempt")

		classification, failureType, rationale := classifyWorkerFailure(status, errMsg)

		entry := RecoveryLogEntry{
			ID: fmt.Sprintf("rl_%d", time.Now().UnixNano()),
			Failure: FailureRecord{
				WorkerName:     worker,
				Phase:          phaseNum,
				Status:         status,
				Classification: classification,
				FailureType:    failureType,
				ErrorMessage:   errMsg,
				Timestamp:      time.Now().UTC().Format(time.RFC3339),
			},
			ActionTaken:   action,
			Outcome:       outcome,
			AttemptNumber: attempt,
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
			Detail:        rationale,
		}

		// Read existing log and append
		existing, _ := recoveryLogReadPhase(phaseNum)
		entries := append(existing.Entries, entry)

		if err := recoveryLogWritePhase(phaseNum, entries); err != nil {
			outputError(1, "failed to write recovery log entry", err)
			return nil
		}

		outputOK(entry)
		return nil
	},
}

func init() {
	failureClassifyCmd.Flags().Bool("json", false, "Output as JSON")
	rootCmd.AddCommand(failureClassifyCmd)

	recoveryLogReadCmd.Flags().Int("phase", 0, "Phase number")
	rootCmd.AddCommand(recoveryLogReadCmd)

	recoveryLogWriteCmd.Flags().Int("phase", 0, "Phase number")
	recoveryLogWriteCmd.Flags().String("worker", "", "Worker name")
	recoveryLogWriteCmd.Flags().String("status", "", "Worker status")
	recoveryLogWriteCmd.Flags().String("error", "", "Error message")
	recoveryLogWriteCmd.Flags().String("action", "", "Action taken")
	recoveryLogWriteCmd.Flags().String("outcome", "", "Outcome")
	recoveryLogWriteCmd.Flags().Int("attempt", 1, "Attempt number")
	rootCmd.AddCommand(recoveryLogWriteCmd)
}
