package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// TestNoChangeReceiptIsNotCreditedOnSelfAssertionAlone is the permanent
// regression lock for NEW-02 (195-REVIEW.iter2.md).
//
// The previous round refused a receipt that names no file -- unless the
// receipt spelled its own status "completed_no_change". Everything that
// exemption had to clear was written by the worker itself: a summary, a
// verification status, and one commands_run string that nothing runs, records
// or corroborates. So the hole the round closed reopened one word later: a
// task that declares no files of its own was credited against a project
// directory that does not even exist.
//
// The rule this locks: free credit must be backed by something the worker
// cannot assert about itself. A no-change receipt is now only credited when
// the task names its own files AND those files are really present in the
// project right now.
func TestNoChangeReceiptIsNotCreditedOnSelfAssertionAlone(t *testing.T) {
	// A task with no evidence artifacts and no file-shaped hints: there is
	// nothing in the project this claim could ever be checked against.
	phase := colony.Phase{ID: 1, Tasks: []colony.Task{receiptTestTask("1.1", "make the thing true")}}
	dispatch := codexBuildDispatch{
		Name:           "Mason-1",
		Status:         "failed",
		CoveredTaskIDs: []string{"1.1"},
	}
	root := filepath.Join(t.TempDir(), "nonexistent-root")

	receipt := codex.TaskReceipt{
		TaskID:  "1.1",
		Status:  codex.TaskReceiptStatusCompletedNoChange,
		Summary: "already true",
		Handoff: passingHandoff("go test ./... (i promise)"),
	}

	admission, _ := admitCoherentJobTaskReceipts(root, phase, dispatch, nil, []codex.TaskReceipt{receipt})
	claims, completed, violations := finalizeCoherentJobTaskReceiptEvidence(root, phase, dispatch, admission)
	if len(completed) != 0 {
		t.Fatalf("a no-change receipt credited task(s) %v against a project directory that does not exist, on nothing but its own sentence", completed)
	}
	if len(claims) != 0 {
		t.Fatalf("expected no task claims, got %d", len(claims))
	}
	if !containsRule(violations, violationRuleTaskReceiptUnevidenced) {
		t.Fatalf("expected refusal rule %q, got %v", violationRuleTaskReceiptUnevidenced, violationRules(violations))
	}

	// The same hazard through the lane both build paths actually use.
	dispatches := []codexBuildDispatch{{
		Name:           "Mason-1",
		Status:         "failed",
		CoveredTaskIDs: []string{"1.1"},
		TaskReceipts:   []codex.TaskReceipt{receipt},
	}}
	resolved := resolveCoherentJobDispatchReceipts(root, phase, dispatches)
	if len(resolved[0].CompletedTaskIDs) != 0 {
		t.Fatalf("the in-repo lane credited %v from a no-change receipt with nothing behind it", resolved[0].CompletedTaskIDs)
	}
	if _, ok := completedBuildTaskIDs(resolved)["1.1"]; ok {
		t.Fatal("task 1.1 reached the credited set on the strength of one self-reported word")
	}
}

// TestGenuineNoChangeReceiptIsStillCreditedWhenTheProjectBacksIt is the other
// direction: the honest "this was already true, and here is the file that
// proves it" case must still be credited. The task names its own file, that
// file is really there, and nothing was changed -- so the receipt names no
// file of its own and is still accepted.
func TestGenuineNoChangeReceiptIsStillCreditedWhenTheProjectBacksIt(t *testing.T) {
	phase := colony.Phase{ID: 1, Tasks: []colony.Task{receiptTestTask("1.1", "make sure the config is right", "config/settings.yaml")}}
	dispatch := codexBuildDispatch{
		Name:           "Mason-1",
		Status:         "failed",
		CoveredTaskIDs: []string{"1.1"},
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatalf("seed config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "settings.yaml"), []byte("already: true\n"), 0o644); err != nil {
		t.Fatalf("seed config file: %v", err)
	}
	// WR-15 (owner decision, 2026-08-27): the file being present is no longer
	// the whole bar -- the check this receipt names is re-run by the program
	// and has to pass, so the project must contain a runnable check.
	seedGoCheck(t, root, "recheck", true)

	receipt := codex.TaskReceipt{
		TaskID:  "1.1",
		Status:  codex.TaskReceiptStatusCompletedNoChange,
		Summary: "the setting was already correct; nothing to change",
		Handoff: passingHandoff("go test ./..."),
	}

	admission, violations := admitCoherentJobTaskReceipts(root, phase, dispatch, nil, []codex.TaskReceipt{receipt})
	if len(violations) != 0 {
		t.Fatalf("expected an honest no-change receipt to be admitted, got %v", violationRules(violations))
	}
	_, completed, finalViolations := finalizeCoherentJobTaskReceiptEvidence(root, phase, dispatch, admission)
	if len(finalViolations) != 0 {
		t.Fatalf("expected no refusals for a no-change receipt the project itself backs, got %v", violationRules(finalViolations))
	}
	if len(completed) != 1 || completed[0] != "1.1" {
		t.Fatalf("expected task 1.1 credited when the file it names is really present, got %v", completed)
	}

	// And the same receipt against a project where that file is absent is
	// refused: the backing is the project's own state, not the sentence.
	absentRoot := t.TempDir()
	admission, _ = admitCoherentJobTaskReceipts(absentRoot, phase, dispatch, nil, []codex.TaskReceipt{receipt})
	_, completed, violations = finalizeCoherentJobTaskReceiptEvidence(absentRoot, phase, dispatch, admission)
	if len(completed) != 0 {
		t.Fatalf("a no-change receipt was credited even though the file its task names is missing from the project, got %v", completed)
	}
	if !containsRule(violations, violationRuleTaskReceiptRootEvidenceMissing) {
		t.Fatalf("expected %q, got %v", violationRuleTaskReceiptRootEvidenceMissing, violationRules(violations))
	}
}
