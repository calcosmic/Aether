package codex

import (
	"fmt"
	"strings"
	"time"
)

// WorkerHandoff carries structured relay data from one worker to the next.
type WorkerHandoff struct {
	ChangedFiles           stringList `json:"changed_files,omitempty"`
	CommandsRun            stringList `json:"commands_run,omitempty"`
	VerificationStatus     string     `json:"verification_status,omitempty"`
	KnownFailures          stringList `json:"known_failures,omitempty"`
	OpenDecisions          stringList `json:"open_decisions,omitempty"`
	Assumptions            stringList `json:"assumptions,omitempty"`
	NextWorkerInstructions stringList `json:"next_worker_instructions,omitempty"`
	DoNotRepeat            stringList `json:"do_not_repeat,omitempty"`
	Freshness              string     `json:"freshness,omitempty"`
}

// HandoffFieldsSummary is the single canonical description of every field a
// WorkerHandoff carries. renderResponseContract (native-Codex dispatch path)
// and the wrapper-facing build/continue brief composers (Claude Code and
// OpenCode dispatch paths) all reference this one constant instead of
// hand-copying the sentence, so the schema stated to a worker can never drift
// from the schema ValidateWorkerHandoff actually enforces.
const HandoffFieldsSummary = "changed_files, commands_run, verification_status, known_failures, open_decisions, assumptions, next_worker_instructions, do_not_repeat, and freshness (an RFC3339 timestamp for when evidence was collected, or \"not-run\")"

// IsEmptyWorkerHandoff reports whether a handoff carries no relay content at
// all. Handoffs are the memory the next phase's workers receive; a
// content-free record occupies a slot in that memory while telling the next
// worker nothing, which is worse than no record.
func IsEmptyWorkerHandoff(h WorkerHandoff) bool {
	return len(h.ChangedFiles) == 0 &&
		len(h.CommandsRun) == 0 &&
		strings.TrimSpace(h.VerificationStatus) == "" &&
		len(h.KnownFailures) == 0 &&
		len(h.OpenDecisions) == 0 &&
		len(h.Assumptions) == 0 &&
		len(h.NextWorkerInstructions) == 0 &&
		len(h.DoNotRepeat) == 0
}

// ValidateWorkerHandoff checks that a WorkerHandoff is structurally valid.
func ValidateWorkerHandoff(h WorkerHandoff) error {
	status := strings.ToLower(strings.TrimSpace(h.VerificationStatus))
	switch status {
	case "", "pass", "passed", "fail", "failed", "partial", "not_run", "not-run", "not run", "unknown":
	default:
		return fmt.Errorf("verification_status must be pass, fail, partial, not_run, or unknown")
	}
	// freshness accepts any string. The shipped handoff contract promises
	// "timestamp or statement", and workers (LLMs) routinely send prose like
	// "Evidence collected after latest edit." Rejecting the whole completion
	// packet over phrasing was the single most expensive downstream failure
	// mode; NormalizeWorkerHandoff coerces non-RFC3339 statements to the
	// receipt time so stored records stay lexicographically sortable.
	return nil
}

// NormalizeWorkerHandoff returns a normalized copy of the handoff.
func NormalizeWorkerHandoff(root string, h WorkerHandoff) WorkerHandoff {
	h.ChangedFiles = normalizeClaimPaths(root, h.ChangedFiles)
	h.CommandsRun = compactStrings(h.CommandsRun)
	h.KnownFailures = compactStrings(h.KnownFailures)
	h.OpenDecisions = compactStrings(h.OpenDecisions)
	h.Assumptions = compactStrings(h.Assumptions)
	h.NextWorkerInstructions = compactStrings(h.NextWorkerInstructions)
	h.DoNotRepeat = compactStrings(h.DoNotRepeat)
	switch strings.ToLower(strings.TrimSpace(h.VerificationStatus)) {
	case "passed":
		h.VerificationStatus = "pass"
	case "failed":
		h.VerificationStatus = "fail"
	case "not run", "not-run":
		h.VerificationStatus = "not_run"
	default:
		h.VerificationStatus = strings.ToLower(strings.TrimSpace(h.VerificationStatus))
	}
	if strings.TrimSpace(h.VerificationStatus) == "" {
		h.VerificationStatus = "unknown"
	}
	freshness := strings.TrimSpace(h.Freshness)
	switch {
	case freshness == "":
		h.Freshness = time.Now().UTC().Format(time.RFC3339)
	case strings.EqualFold(freshness, "not-run"), strings.EqualFold(freshness, "not_run"), strings.EqualFold(freshness, "not run"):
		h.Freshness = "not-run"
	default:
		if _, err := time.Parse(time.RFC3339, freshness); err != nil {
			// A prose statement ("Evidence collected after latest edit.") is
			// contract-legal but not sortable; stamp the receipt time instead.
			h.Freshness = time.Now().UTC().Format(time.RFC3339)
		}
	}
	return h
}

func workerHandoffIsEmpty(h WorkerHandoff) bool {
	return len(h.ChangedFiles) == 0 &&
		len(h.CommandsRun) == 0 &&
		strings.TrimSpace(h.VerificationStatus) == "" &&
		len(h.KnownFailures) == 0 &&
		len(h.OpenDecisions) == 0 &&
		len(h.Assumptions) == 0 &&
		len(h.NextWorkerInstructions) == 0 &&
		len(h.DoNotRepeat) == 0 &&
		strings.TrimSpace(h.Freshness) == ""
}
