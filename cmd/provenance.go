package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/codex"
)

// validateBuildProvenance checks that at least one worker completed successfully
// and reported file changes (created, modified, or tests written). It rejects
// phantom builds where no worker actually produced output changes.
//
// SAFE-01: Rejects builds where all workers are in a non-success state.
// SAFE-02: Rejects builds where all completed workers have zero file changes.
func validateBuildProvenance(results []codexExternalBuildWorkerResult) error {
	if len(results) == 0 {
		return fmt.Errorf("build provenance: no worker results provided")
	}

	completedCount := 0
	for _, r := range results {
		status := normalizeExternalBuildStatus(r.Status)
		if !isSuccessfulExternalBuildStatus(status) || !isBuildImplementationWorker(r) {
			// D-08 (195-CONTEXT.md): a failed/blocked/timeout/interrupted
			// worker can still carry genuine, evidenced per-task completion
			// proof (task_receipts) for part of a grouped job before it
			// failed -- that is real provenance, not a phantom build, even
			// though the WORKER's own overall status never reached success.
			// The shared receipt trust boundary
			// (cmd/coherent_job_receipts.go) still decides exactly which
			// tasks that proof actually credits; this only proves SOME of
			// the work is real, the same carve-out completed_no_change
			// already gets via the loop below.
			if hasGenuineTaskReceiptEvidence(r) {
				return nil
			}
			continue
		}
		completedCount++
		if len(r.FilesModified) > 0 || len(r.FilesCreated) > 0 || len(r.TestsWritten) > 0 {
			return nil // At least one valid provenance entry found
		}
		// An honest completed_no_change result satisfies provenance WITH its
		// evidence (ruling D6): the work was to verify, and the commands_run
		// prove somebody did. Evidence-free no-change still falls through to
		// rejection — the phantom-build guard keeps its teeth (SAFE-02).
		if isNoChangeExternalBuildStatus(status) && len(noChangeEvidenceMissing(r)) == 0 {
			return nil
		}
	}

	if completedCount == 0 {
		return fmt.Errorf("build provenance: no workers completed successfully -- all %d worker(s) are in a non-success state", len(results))
	}
	return fmt.Errorf("build provenance: %d worker(s) completed but none reported file changes (created, modified, or tests) or evidenced no-change verification -- the build produced no changes", completedCount)
}

// hasGenuineTaskReceiptEvidence reports whether a worker result -- regardless
// of its own overall terminal status -- carries at least one task_receipts
// entry with genuine, evidenced completion proof: a successful receipt
// status, a non-empty summary, and a passing handoff with at least one
// concrete commands_run entry. It mirrors the structural checks
// admitCoherentJobTaskReceipts enforces (cmd/coherent_job_receipts.go)
// closely enough to prove SOME of a failed grouped job's work is real,
// without duplicating that function's manifest/phase-scoped rules --
// admitCoherentJobTaskReceipts (and finalizeCoherentJobTaskReceiptEvidence)
// remain the only functions that decide exactly which tasks the evidence
// actually credits. A result with zero task_receipts, or only structurally
// hollow ones, still falls through to the ordinary phantom-build rejection.
func hasGenuineTaskReceiptEvidence(r codexExternalBuildWorkerResult) bool {
	for _, receipt := range r.TaskReceipts {
		if strings.TrimSpace(receipt.TaskID) == "" {
			continue
		}
		status := strings.ToLower(strings.TrimSpace(receipt.Status))
		if status != codex.TaskReceiptStatusCompleted && status != codex.TaskReceiptStatusCompletedNoChange {
			continue
		}
		if strings.TrimSpace(receipt.Summary) == "" {
			continue
		}
		if strings.ToLower(strings.TrimSpace(receipt.Handoff.VerificationStatus)) != "pass" {
			continue
		}
		if len(receipt.Handoff.CommandsRun) == 0 {
			continue
		}
		return true
	}
	return false
}

func validateBuildProvenanceForManifest(manifest *codexBuildManifest, results []codexExternalBuildWorkerResult) error {
	err := validateBuildProvenance(results)
	if err == nil {
		return nil
	}
	if manifest != nil && manifest.PhaseMode == "discovery" && hasExternalDiscoveryEvidence(results) {
		return nil
	}
	if manifest == nil || !isVerificationOnlyBuildManifest(*manifest) {
		return err
	}

	completedCount := 0
	for _, r := range results {
		if normalizeExternalBuildStatus(r.Status) != "completed" || !isBuildImplementationWorker(r) {
			continue
		}
		completedCount++
		if len(r.Outputs) > 0 {
			return nil
		}
	}
	if completedCount == 0 {
		return err
	}
	return fmt.Errorf("build provenance: verification-only phase completed but no implementation workers reported output evidence")
}

func hasExternalDiscoveryEvidence(results []codexExternalBuildWorkerResult) bool {
	foundTask := false
	for _, result := range results {
		if strings.TrimSpace(result.TaskID) == "" {
			continue
		}
		foundTask = true
		if normalizeExternalBuildStatus(result.Status) != "completed" || strings.TrimSpace(result.Summary) == "" {
			return false
		}
	}
	return foundTask
}

func isVerificationOnlyBuildManifest(manifest codexBuildManifest) bool {
	if len(manifest.Tasks) == 0 {
		return false
	}
	for _, task := range manifest.Tasks {
		goal := strings.ToLower(strings.TrimSpace(task.Goal))
		if !(strings.HasPrefix(goal, "run ") || strings.HasPrefix(goal, "verify ")) {
			return false
		}
		if containsBuildMutationVerb(goal) {
			return false
		}
	}
	return true
}

func containsBuildMutationVerb(text string) bool {
	for _, needle := range []string{
		"implement ",
		"create ",
		"fix ",
		"update ",
		"add ",
		"write ",
		"modify ",
		"refactor ",
		"delete ",
		"remove ",
	} {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func isBuildImplementationWorker(result codexExternalBuildWorkerResult) bool {
	if result.TaskID != "" {
		return true
	}
	switch result.Caste {
	case "", "builder":
		return true
	default:
		return false
	}
}

// traceContinueProvenance verifies that every completed worker dispatch has
// non-empty Outputs, ensuring claims trace back to valid worker results.
//
// SAFE-03: Rejects claims where completed dispatches have missing provenance.
// SAFE-04: Rejects claims when no completed dispatches exist (stale/missing).
//
// Per D-03: rejection causes halt. There is no warn-and-allow path.
func traceContinueProvenance(dispatches []codexBuildDispatch) error {
	if len(dispatches) == 0 {
		return fmt.Errorf("continue provenance: no worker dispatches found -- build did not produce verifiable results")
	}

	completedCount := 0
	for _, d := range dispatches {
		if !isSuccessfulExternalBuildStatus(d.Status) {
			continue
		}
		completedCount++
		// Two successes are exempt from the file-outputs requirement.
		// completed_no_change: its evidence is the verification it ran,
		// already enforced by the finalizer's no_change_evidence gate before
		// this status could be stored (ruling D6). manually-reconciled: an
		// operator reconciled it by hand, so it has no worker outputs by
		// definition -- demanding them halts the manual recovery path with a
		// phantom-build accusation.
		if isNoChangeExternalBuildStatus(d.Status) || d.Status == "manually-reconciled" {
			continue
		}
		if len(d.Outputs) == 0 {
			return fmt.Errorf("continue provenance: worker %q claims completion but has no file outputs -- possible phantom build", d.Name)
		}
	}

	if completedCount == 0 {
		return fmt.Errorf("continue provenance: no completed worker dispatches found -- build did not produce verifiable results")
	}

	return nil
}

func traceContinueProvenanceForManifest(manifest codexContinueManifest) error {
	if manifest.Present && manifest.Data.PhaseMode == "discovery" {
		if hasDurableDiscoveryDispatchEvidence(manifest.Data.Dispatches) {
			return nil
		}
		return fmt.Errorf("continue provenance: discovery phase has no durable completed task summaries")
	}
	return traceContinueProvenance(manifest.Data.Dispatches)
}
