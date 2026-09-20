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
	return validateBuildProvenanceWithKnownTasks(results, nil)
}

// validateBuildProvenanceWithKnownTasks is validateBuildProvenance with the
// phase's own task IDs supplied, so the failed-worker receipt carve-out below
// can check that a receipt names a task this phase actually has. knownTaskIDs
// may be nil when no manifest is available, in which case the task-ID check is
// skipped and the carve-out's other requirements still apply.
func validateBuildProvenanceWithKnownTasks(results []codexExternalBuildWorkerResult, knownTaskIDs map[string]struct{}) error {
	if len(results) == 0 {
		return fmt.Errorf("build provenance: no worker results provided")
	}

	completedCount := 0
	receiptEvidenceCount := 0
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
			//
			// CR-02 (195-REVIEW.md): this used to `return nil`, which ended
			// the guard for the WHOLE packet the moment any single result
			// carried a structurally-plausible receipt -- so a reviewer's
			// invented receipt excused a builder that had changed nothing.
			// The relaxation is now scoped: it counts, per worker, only for
			// an implementation worker whose own run did not succeed, and
			// only for a receipt that names a real file (and, when the phase's
			// tasks are known, a real task). It never short-circuits the rest
			// of the loop.
			//
			// NEW-01 (195-REVIEW.iter2.md): the worker test is
			// isReceiptCarveOutWorker, NOT isBuildImplementationWorker.
			// The latter answers "yes" for any result carrying a task ID at
			// all, so a reviewer worker that happened to be given one tripped
			// the carve-out the code-writing worker is the only owner of.
			if !isSuccessfulExternalBuildStatus(status) && isReceiptCarveOutWorker(r) &&
				hasGenuineTaskReceiptEvidence(r, knownTaskIDs) {
				receiptEvidenceCount++
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

	// NEW-01 (195-REVIEW.iter2.md): the receipt carve-out may only excuse
	// SAFE-01 ("nobody finished"), never SAFE-02 ("somebody finished and
	// changed nothing"). Placing it ahead of both errors meant a worker that
	// reported success while touching no file was waved through on a
	// DIFFERENT worker's evidence, and then credited for every task it
	// covered -- strictly weaker than the guard before this phase. A failed
	// worker's honest partial proof says nothing whatsoever about whether the
	// worker beside it did its own job, so it is folded into the SAFE-01
	// branch only.
	if completedCount == 0 {
		if receiptEvidenceCount > 0 {
			return nil
		}
		return fmt.Errorf("build provenance: no workers completed successfully -- all %d worker(s) are in a non-success state", len(results))
	}
	return fmt.Errorf("build provenance: %d worker(s) completed but none reported file changes (created, modified, or tests) or evidenced no-change verification -- the build produced no changes", completedCount)
}

// hasGenuineTaskReceiptEvidence reports whether a worker result -- regardless
// of its own overall terminal status -- carries at least one task_receipts
// entry with genuine, evidenced completion proof: a task ID this phase really
// has (when knownTaskIDs is supplied), a successful receipt status, a
// non-empty summary, a passing handoff with at least one concrete commands_run
// entry, and AT LEAST ONE NAMED FILE. It mirrors the structural checks
// admitCoherentJobTaskReceipts enforces (cmd/coherent_job_receipts.go)
// closely enough to prove SOME of a failed grouped job's work is real,
// without duplicating that function's manifest/phase-scoped rules --
// admitCoherentJobTaskReceipts (and finalizeCoherentJobTaskReceiptEvidence)
// remain the only functions that decide exactly which tasks the evidence
// actually credits. A result with zero task_receipts, or only structurally
// hollow ones, still falls through to the ordinary phantom-build rejection.
//
// CR-02 (195-REVIEW.md): the file requirement is what stops this function
// being satisfiable by pure self-assertion. The guard it feeds asks exactly
// one question -- "did this build change anything?" -- and a receipt naming no
// file is not an answer to it.
func hasGenuineTaskReceiptEvidence(r codexExternalBuildWorkerResult, knownTaskIDs map[string]struct{}) bool {
	for _, receipt := range r.TaskReceipts {
		taskID := strings.TrimSpace(receipt.TaskID)
		if taskID == "" {
			continue
		}
		if len(knownTaskIDs) > 0 {
			if _, known := knownTaskIDs[taskID]; !known {
				continue
			}
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
		if len(receipt.FilesCreated) == 0 && len(receipt.FilesModified) == 0 && len(receipt.TestsWritten) == 0 {
			continue
		}
		return true
	}
	return false
}

// manifestKnownTaskIDs collects every task ID the manifest's phase actually
// owns, from both its task plan and its dispatches' covered-task lists, so a
// receipt naming a task that exists nowhere in the phase can be recognised.
func manifestKnownTaskIDs(manifest *codexBuildManifest) map[string]struct{} {
	if manifest == nil {
		return nil
	}
	known := make(map[string]struct{}, len(manifest.Tasks)+len(manifest.Dispatches))
	for _, task := range manifest.Tasks {
		if id := strings.TrimSpace(task.ID); id != "" {
			known[id] = struct{}{}
		}
	}
	for _, dispatch := range manifest.Dispatches {
		for _, id := range dispatchCoveredTaskIDs(dispatch) {
			known[id] = struct{}{}
		}
	}
	return known
}

func validateBuildProvenanceForManifest(manifest *codexBuildManifest, results []codexExternalBuildWorkerResult) error {
	err := validateBuildProvenanceWithKnownTasks(results, manifestKnownTaskIDs(manifest))
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

// isReceiptCarveOutWorker reports whether a worker result belongs to the one
// role the failed-worker receipt carve-out exists for: the worker that writes
// the code. It deliberately asks ONLY about the caste, never about the
// presence of a task ID.
//
// NEW-01 (195-REVIEW.iter2.md): isBuildImplementationWorker treats any result
// carrying a task ID as an implementation worker, which is right for the
// question it was written for (which results must show file evidence) and
// wrong for this one. A reviewer handed a task ID is still a reviewer, and a
// reviewer's receipts are never evidence that the build produced changes.
func isReceiptCarveOutWorker(result codexExternalBuildWorkerResult) bool {
	switch strings.ToLower(strings.TrimSpace(result.Caste)) {
	case "", "builder":
		return true
	default:
		return false
	}
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
