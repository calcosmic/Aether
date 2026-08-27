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
