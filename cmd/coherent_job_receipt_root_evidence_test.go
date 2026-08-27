package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// TestReceiptClaimingNoFileIsNeverCredited is the permanent regression lock for
// the 195 review's first critical finding (CR-01): the stage-2 root-evidence
// check was guarded by `if len(paths) > 0`, so a receipt that named NO file at
// all skipped the check entirely and fell straight through to task credit.
// Combined with stage 1 skipping its task-binding check for a task that
// declares no paths of its own, a worker could mark a task complete on the
// strength of one self-asserted sentence, against a root that does not even
// exist.
//
// The rule this locks: a `completed` receipt must name at least one file that
// is actually present in the root checkout. Naming nothing is a refusal, not a
// pass. Only `completed_no_change` -- whose evidence IS its commands_run, and
// which stage 1 already forces to carry a passing verification plus at least
// one concrete command -- may credit a task without naming a file.
func TestReceiptClaimingNoFileIsNeverCredited(t *testing.T) {
	// A task with no evidence artifacts and no file-shaped hints: stage 1's
	// declared-path binding has nothing to bind against, which is exactly the
	// combination the finding reproduced.
	phase := colony.Phase{ID: 1, Tasks: []colony.Task{receiptTestTask("1.1", "make the thing true")}}
	dispatch := codexBuildDispatch{
		Name:           "Mason-1",
		Status:         "failed",
		CoveredTaskIDs: []string{"1.1"},
	}
	// Deliberately a root that does not exist: nothing here can possibly be
	// root-backed evidence.
	root := filepath.Join(t.TempDir(), "nonexistent-root")

	receipt := codex.TaskReceipt{
		TaskID:  "1.1",
		Status:  codex.TaskReceiptStatusCompleted,
		Summary: "done",
		Handoff: passingHandoff("echo ok"),
	}

	admission, _ := admitCoherentJobTaskReceipts(root, phase, dispatch, nil, []codex.TaskReceipt{receipt})
	claims, completed, violations := finalizeCoherentJobTaskReceiptEvidence(root, phase, dispatch, admission)

	if len(completed) != 0 {
		t.Fatalf("a receipt naming no file credited task(s) %v against a nonexistent root; a receipt that points at nothing is not evidence of anything", completed)
	}
	if len(claims) != 0 {
		t.Fatalf("expected no task claims, got %d", len(claims))
	}
	if !containsRule(violations, violationRuleTaskReceiptUnevidenced) {
		t.Fatalf("expected refusal rule %q naming the task, got %v", violationRuleTaskReceiptUnevidenced, violationRules(violations))
	}
}

// TestNoChangeReceiptStillCreditsWithoutNamingFilesItself proves the CR-01
// fix is the narrow rule and not a blanket ban: an honest
// completed_no_change receipt -- the D6 "the behaviour was already true"
// case -- still credits its task without naming a file of its own.
//
// NEW-02 (195-REVIEW.iter2.md) narrowed what "honest" means here. This test
// used to hand credit to a task that named no file anywhere, against an empty
// directory, on the strength of the receipt's own status word; it therefore
// asserted the very hole the review found. What it locks now is the same
// intent with real backing: the task names its own file, that file is really
// in the project, and the receipt still names nothing because nothing changed.
func TestNoChangeReceiptStillCreditsWithoutNamingFilesItself(t *testing.T) {
	phase := colony.Phase{ID: 1, Tasks: []colony.Task{receiptTestTask("1.1", "make the thing true", "thing.go")}}
	dispatch := codexBuildDispatch{
		Name:           "Mason-1",
		Status:         "failed",
		CoveredTaskIDs: []string{"1.1"},
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "thing.go"), []byte("package thing\n"), 0o644); err != nil {
		t.Fatalf("seed root file: %v", err)
	}
	// WR-15 (owner decision, 2026-08-27): "honest" now also means the check
	// this receipt names really passes when the program re-runs it, so the
	// project has to contain a check that can actually be run.
	seedGoCheck(t, root, "thing", true)

	receipt := codex.TaskReceipt{
		TaskID:  "1.1",
		Status:  codex.TaskReceiptStatusCompletedNoChange,
		Summary: "already true; verified",
		Handoff: passingHandoff("go test ./..."),
	}

	admission, violations := admitCoherentJobTaskReceipts(root, phase, dispatch, nil, []codex.TaskReceipt{receipt})
	if len(violations) != 0 {
		t.Fatalf("expected an honest no-change receipt to be admitted, got %v", violationRules(violations))
	}
	_, completed, finalViolations := finalizeCoherentJobTaskReceiptEvidence(root, phase, dispatch, admission)
	if len(finalViolations) != 0 {
		t.Fatalf("expected no refusals for an honest no-change receipt, got %v", violationRules(finalViolations))
	}
	if len(completed) != 1 || completed[0] != "1.1" {
		t.Fatalf("expected task 1.1 to be credited by its no-change receipt, got %v", completed)
	}
}

// TestFileNamingReceiptStillNeedsThatFileInRoot keeps the positive path honest
// from the other direction: a `completed` receipt that names a file which IS
// present in root is credited, and the same receipt against a root where the
// file is absent is refused by name.
func TestFileNamingReceiptStillNeedsThatFileInRoot(t *testing.T) {
	phase := colony.Phase{ID: 1, Tasks: []colony.Task{receiptTestTask("1.1", "make the thing true")}}
	dispatch := codexBuildDispatch{
		Name:           "Mason-1",
		Status:         "failed",
		CoveredTaskIDs: []string{"1.1"},
	}
	receipt := codex.TaskReceipt{
		TaskID:        "1.1",
		Status:        codex.TaskReceiptStatusCompleted,
		Summary:       "wrote it",
		FilesModified: []string{"thing.go"},
		Handoff:       passingHandoff("go build ./..."),
	}
	aggregate := []string{"thing.go"}

	presentRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(presentRoot, "thing.go"), []byte("package thing\n"), 0o644); err != nil {
		t.Fatalf("seed root file: %v", err)
	}
	admission, _ := admitCoherentJobTaskReceipts(presentRoot, phase, dispatch, aggregate, []codex.TaskReceipt{receipt})
	_, completed, _ := finalizeCoherentJobTaskReceiptEvidence(presentRoot, phase, dispatch, admission)
	if len(completed) != 1 || completed[0] != "1.1" {
		t.Fatalf("expected task 1.1 credited when its named file is really in root, got %v", completed)
	}

	absentRoot := t.TempDir()
	admission, _ = admitCoherentJobTaskReceipts(absentRoot, phase, dispatch, aggregate, []codex.TaskReceipt{receipt})
	_, completed, violations := finalizeCoherentJobTaskReceiptEvidence(absentRoot, phase, dispatch, admission)
	if len(completed) != 0 {
		t.Fatalf("expected no credit when the named file is absent from root, got %v", completed)
	}
	if !containsRule(violations, violationRuleTaskReceiptRootEvidenceMissing) {
		t.Fatalf("expected %q, got %v", violationRuleTaskReceiptRootEvidenceMissing, violationRules(violations))
	}
}

// TestInRepoLaneNeverCreditsAFileLessReceipt runs the same hazard through the
// native/in-repo caller both build lanes use, so the lock covers the wiring and
// not only the two functions in isolation.
func TestInRepoLaneNeverCreditsAFileLessReceipt(t *testing.T) {
	phase := colony.Phase{ID: 1, Tasks: []colony.Task{receiptTestTask("1.1", "make the thing true")}}
	dispatches := []codexBuildDispatch{{
		Name:           "Mason-1",
		Status:         "failed",
		CoveredTaskIDs: []string{"1.1"},
		TaskReceipts: []codex.TaskReceipt{{
			TaskID:  "1.1",
			Status:  codex.TaskReceiptStatusCompleted,
			Summary: "done",
			Handoff: passingHandoff("echo ok"),
		}},
	}}

	resolved := resolveCoherentJobDispatchReceipts(filepath.Join(t.TempDir(), "nonexistent-root"), phase, dispatches)
	if len(resolved[0].CompletedTaskIDs) != 0 {
		t.Fatalf("in-repo lane credited %v from a receipt that names no file", resolved[0].CompletedTaskIDs)
	}
	credited := completedBuildTaskIDs(resolved)
	if _, ok := credited["1.1"]; ok {
		t.Fatalf("task 1.1 reached completedBuildTaskIDs on the strength of a self-asserted sentence")
	}
}
