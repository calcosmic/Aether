package cmd

import (
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
)

// buildProvenanceManifestFixture is a minimal manifest whose phase really owns
// tasks 1.1 and 1.2, so a receipt naming anything else can be recognised as
// naming a task that does not exist.
func buildProvenanceManifestFixture() *codexBuildManifest {
	return &codexBuildManifest{
		Phase: 1,
		Tasks: []codexBuildTaskPlan{
			{ID: "1.1", Goal: "implement the thing"},
			{ID: "1.2", Goal: "implement the other thing"},
		},
		Dispatches: []codexBuildDispatch{
			{Name: "Mason-1", Caste: "builder", TaskID: "1.1", CoveredTaskIDs: []string{"1.1", "1.2"}},
		},
	}
}

// TestOneUnverifiedReceiptCannotDisarmThePhantomBuildGuard is the permanent
// regression lock for the 195 review's second critical finding (CR-02).
//
// The phantom-build guard exists to reject a completion packet in which no
// worker actually changed anything. Phase 195 added a carve-out so that a
// worker whose own overall run failed could still prove it had honestly
// finished part of a grouped job. That carve-out did `return nil` -- it
// terminated the guard for the ENTIRE packet on the strength of a single
// task_receipts object whose only qualification was its own self-reported
// shape. Nothing was checked against the phase or the repository.
//
// This test reproduces the packet the review proved would be accepted: a
// builder reporting success while having changed no file at all, plus a
// reviewer worker carrying one receipt for a task that does not exist. Before
// this phase, that packet was rejected as "the build produced no changes".
func TestOneUnverifiedReceiptCannotDisarmThePhantomBuildGuard(t *testing.T) {
	results := []codexExternalBuildWorkerResult{
		{
			Name:   "Mason-1",
			Caste:  "builder",
			TaskID: "1.1",
			Status: "completed",
			// No files created, modified, or tested: nothing happened.
		},
		{
			Name:   "Falcon-2",
			Caste:  "watcher",
			Status: "failed",
			TaskReceipts: []codex.TaskReceipt{{
				TaskID:  "totally-made-up",
				Status:  codex.TaskReceiptStatusCompleted,
				Summary: "x",
				Handoff: codex.WorkerHandoff{VerificationStatus: "pass", CommandsRun: []string{"echo hi"}},
			}},
		},
	}

	if err := validateBuildProvenance(results); err == nil {
		t.Fatal("a packet where no worker changed a single file was accepted because one unverified receipt object was present; the whole-packet guard was disarmed by a self-assertion")
	}
	if err := validateBuildProvenanceForManifest(buildProvenanceManifestFixture(), results); err == nil {
		t.Fatal("the manifest-scoped guard also accepted a packet that changed nothing, on the strength of a receipt naming a task the phase does not have")
	}
}

// TestReceiptCarveOutStillAcceptsRealPartialWork proves the CR-02 fix keeps
// D-08's actual intent: a grouped worker whose own run failed, but which
// really did finish and name files for part of its job, is still honest
// provenance and must not be rejected as a phantom build.
func TestReceiptCarveOutStillAcceptsRealPartialWork(t *testing.T) {
	results := []codexExternalBuildWorkerResult{
		{
			Name:   "Mason-1",
			Caste:  "builder",
			TaskID: "1.1",
			Status: "failed",
			TaskReceipts: []codex.TaskReceipt{{
				TaskID:        "1.1",
				Status:        codex.TaskReceiptStatusCompleted,
				Summary:       "finished the first half before the run died",
				FilesModified: []string{"cmd/thing.go"},
				Handoff:       codex.WorkerHandoff{VerificationStatus: "pass", CommandsRun: []string{"go build ./..."}},
			}},
		},
	}
	if err := validateBuildProvenance(results); err != nil {
		t.Fatalf("a failed grouped worker with real, file-naming per-task proof must still satisfy provenance: %v", err)
	}
	if err := validateBuildProvenanceForManifest(buildProvenanceManifestFixture(), results); err != nil {
		t.Fatalf("manifest-scoped guard rejected genuine partial work: %v", err)
	}
}

// TestReceiptCarveOutRefusesAReceiptNamingNoFile closes the other half of the
// hole: even on the right worker, for a task that really exists, a receipt
// that names no file is not proof that the build changed anything.
func TestReceiptCarveOutRefusesAReceiptNamingNoFile(t *testing.T) {
	results := []codexExternalBuildWorkerResult{
		{
			Name:   "Mason-1",
			Caste:  "builder",
			TaskID: "1.1",
			Status: "failed",
			TaskReceipts: []codex.TaskReceipt{{
				TaskID:  "1.1",
				Status:  codex.TaskReceiptStatusCompleted,
				Summary: "trust me",
				Handoff: codex.WorkerHandoff{VerificationStatus: "pass", CommandsRun: []string{"echo hi"}},
			}},
		},
	}
	if err := validateBuildProvenance(results); err == nil {
		t.Fatal("a receipt naming no file satisfied the phantom-build guard")
	}
}

// TestReceiptCarveOutIsScopedToImplementationWorkers proves the relaxation is
// per-worker: a reviewer's receipts are not a licence to accept a build in
// which the worker who was supposed to write the code wrote nothing.
func TestReceiptCarveOutIsScopedToImplementationWorkers(t *testing.T) {
	results := []codexExternalBuildWorkerResult{
		{Name: "Mason-1", Caste: "builder", TaskID: "1.1", Status: "completed"},
		{
			Name:   "Falcon-2",
			Caste:  "watcher",
			Status: "failed",
			TaskReceipts: []codex.TaskReceipt{{
				TaskID:        "1.1",
				Status:        codex.TaskReceiptStatusCompleted,
				Summary:       "reviewed it",
				FilesModified: []string{"cmd/thing.go"},
				Handoff:       codex.WorkerHandoff{VerificationStatus: "pass", CommandsRun: []string{"go build ./..."}},
			}},
		},
	}
	if err := validateBuildProvenance(results); err == nil {
		t.Fatal("a reviewer worker's receipts disarmed the phantom-build guard for a build whose implementation worker changed nothing")
	}
}

// TestZeroFileCompletedWorkerIsRejectedEvenBesideARealPartial is the
// permanent regression lock for NEW-01 (195-REVIEW.iter2.md).
//
// The first fix round scoped WHICH worker may trip the failed-worker receipt
// carve-out, but kept the relaxation as a packet-wide `return nil` placed
// ahead of the "completed but reported no file changes" rejection. A builder
// that reported success while changing nothing therefore got full credit for
// every task it covered, as long as any OTHER worker in the same packet
// carried one genuine receipt. That is strictly worse than the behaviour
// before this phase, where the same packet was rejected outright.
//
// Both directions are asserted, because either one alone proves nothing: the
// carve-out must keep accepting a genuine partial, and must stop excusing a
// do-nothing worker that sits beside one.
func TestZeroFileCompletedWorkerIsRejectedEvenBesideARealPartial(t *testing.T) {
	genuinePartial := codexExternalBuildWorkerResult{
		Name:   "Mason-2",
		Caste:  "builder",
		TaskID: "1.2",
		Status: "failed",
		TaskReceipts: []codex.TaskReceipt{{
			TaskID:        "1.2",
			Status:        codex.TaskReceiptStatusCompleted,
			Summary:       "finished this one before the run died",
			FilesModified: []string{"cmd/real.go"},
			Handoff:       codex.WorkerHandoff{VerificationStatus: "pass", CommandsRun: []string{"go build ./..."}},
		}},
	}

	// Direction 1: the do-nothing worker must not be excused by its neighbour.
	phantomBesidePartial := []codexExternalBuildWorkerResult{
		{
			Name:   "Mason-1",
			Caste:  "builder",
			TaskID: "1.1",
			Status: "completed",
			// Zero files created, modified or tested: nothing happened here.
		},
		genuinePartial,
	}
	if err := validateBuildProvenance(phantomBesidePartial); err == nil {
		t.Fatal("a worker that reported success while changing no file was accepted because a DIFFERENT worker carried a real receipt; the phantom-build guard must judge the do-nothing worker on its own evidence")
	}
	if err := validateBuildProvenanceForManifest(buildProvenanceManifestFixture(), phantomBesidePartial); err == nil {
		t.Fatal("the manifest-scoped guard also excused a zero-file completed worker on another worker's evidence")
	}

	// Direction 2: the genuine partial on its own is still honest provenance.
	if err := validateBuildProvenance([]codexExternalBuildWorkerResult{genuinePartial}); err != nil {
		t.Fatalf("a failed grouped worker with real, file-naming per-task proof must still satisfy provenance: %v", err)
	}

	// Direction 3: a real partial beside a worker that really did change
	// something is accepted, so the tightened rule has not made a normal
	// mixed packet unfinalizable.
	realWorkBesidePartial := []codexExternalBuildWorkerResult{
		{
			Name:          "Mason-1",
			Caste:         "builder",
			TaskID:        "1.1",
			Status:        "completed",
			FilesModified: []string{"cmd/other.go"},
		},
		genuinePartial,
	}
	if err := validateBuildProvenance(realWorkBesidePartial); err != nil {
		t.Fatalf("a packet where one worker really changed a file and another partially succeeded must be accepted: %v", err)
	}
}

// TestReceiptCarveOutIgnoresAReviewerCarryingATaskID closes the scoping hole
// the iteration-2 review named: the carve-out asked
// isBuildImplementationWorker, which answers "yes" for ANY result carrying a
// task id at all, so a reviewer worker with a task id could trip it. The
// sibling test above only ever exercised a watcher with an EMPTY task id, so
// it proved less than its name claimed.
func TestReceiptCarveOutIgnoresAReviewerCarryingATaskID(t *testing.T) {
	results := []codexExternalBuildWorkerResult{
		{
			Name:   "Falcon-2",
			Caste:  "watcher",
			TaskID: "1.1",
			Status: "failed",
			TaskReceipts: []codex.TaskReceipt{{
				TaskID:        "1.1",
				Status:        codex.TaskReceiptStatusCompleted,
				Summary:       "reviewed it",
				FilesModified: []string{"cmd/thing.go"},
				Handoff:       codex.WorkerHandoff{VerificationStatus: "pass", CommandsRun: []string{"go build ./..."}},
			}},
		},
	}
	if err := validateBuildProvenance(results); err == nil {
		t.Fatal("a reviewer worker carrying a task id tripped the failed-worker receipt carve-out; the carve-out exists for the worker that writes the code, not for whoever checks it")
	}
}
