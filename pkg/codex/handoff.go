package codex

import (
	"fmt"
	"strings"
	"time"
)

// WorkerHandoff carries structured relay data from one worker to the next.
type WorkerHandoff struct {
	ChangedFiles           []string `json:"changed_files,omitempty"`
	CommandsRun            []string `json:"commands_run,omitempty"`
	VerificationStatus     string   `json:"verification_status,omitempty"`
	KnownFailures          []string `json:"known_failures,omitempty"`
	OpenDecisions          []string `json:"open_decisions,omitempty"`
	Assumptions            []string `json:"assumptions,omitempty"`
	NextWorkerInstructions []string `json:"next_worker_instructions,omitempty"`
	DoNotRepeat            []string `json:"do_not_repeat,omitempty"`
	Freshness              string   `json:"freshness,omitempty"`
}

// ValidateWorkerHandoff checks that a WorkerHandoff is structurally valid.
func ValidateWorkerHandoff(h WorkerHandoff) error {
	status := strings.ToLower(strings.TrimSpace(h.VerificationStatus))
	switch status {
	case "", "pass", "passed", "fail", "failed", "partial", "not_run", "not-run", "not run", "unknown":
	default:
		return fmt.Errorf("verification_status must be pass, fail, partial, not_run, or unknown")
	}
	freshness := strings.TrimSpace(h.Freshness)
	if freshness == "" || freshness == "not-run" {
		return nil
	}
	if _, err := time.Parse(time.RFC3339, freshness); err != nil {
		return fmt.Errorf("freshness must be RFC3339 or not-run: %w", err)
	}
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
	if strings.TrimSpace(h.Freshness) == "" {
		h.Freshness = time.Now().UTC().Format(time.RFC3339)
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
