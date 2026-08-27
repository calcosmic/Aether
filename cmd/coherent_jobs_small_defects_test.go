package cmd

import (
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
)

// TestJobPlannerDoesNotMutateItsCallersProposal locks IN-01 (195-REVIEW.md).
// The job planner's own documentation calls it pure, but normalizing a
// proposal wrote trimmed task IDs back through the caller's slice, so the
// caller's data changed underneath it.
func TestJobPlannerDoesNotMutateItsCallersProposal(t *testing.T) {
	original := []string{" 1.1 ", "1.2\t"}
	proposal := coherentJobProposal{Name: "job", TaskIDs: original}

	normalized := normalizeCoherentJobProposal(proposal)

	if original[0] != " 1.1 " || original[1] != "1.2\t" {
		t.Fatalf("the planner rewrote its caller's own task list: %q", original)
	}
	if normalized.TaskIDs[0] != "1.1" || normalized.TaskIDs[1] != "1.2" {
		t.Fatalf("normalization stopped working: %q", normalized.TaskIDs)
	}
}

// TestSentenceCaseKeepsWholeCharacters locks IN-02 (195-REVIEW.md): capitalizing
// an owner-facing sentence by slicing its first BYTE splits any character that
// takes more than one byte to store, turning it into unreadable rubbish on
// screen. Every sentence in this position is plain English today; this makes
// the first one that is not safe.
func TestSentenceCaseKeepsWholeCharacters(t *testing.T) {
	cases := map[string]string{
		"":                     "",
		"one worker":           "One worker",
		"élan is the only one": "Élan is the only one",
		"日本語のぶん":               "日本語のぶん",
	}
	for input, want := range cases {
		if got := sentenceCase(input); got != want {
			t.Errorf("sentenceCase(%q) = %q, want %q", input, got, want)
		}
	}
}

// TestTwoUnnamedWorkersStillConflictOverOneFile locks IN-03 (195-REVIEW.md):
// two workers that both arrive without a task id compared equal, so a genuine
// collision between them over the same file was not refused.
func TestTwoUnnamedWorkersStillConflictOverOneFile(t *testing.T) {
	dispatches := []codex.WorkerDispatch{
		{WorkerName: "Anvil-1", Caste: "builder", Wave: 1, DeclaredPaths: []string{"cmd/foo.go"}},
		{WorkerName: "Hammer-2", Caste: "builder", Wave: 1, DeclaredPaths: []string{"cmd/foo.go"}},
	}
	if err := validateDeclaredWorktreeOwnership(dispatches); err == nil {
		t.Fatal("two different workers both claiming cmd/foo.go in the same round were not refused, because neither carried a task id")
	}

	// The same worker listed twice is not a conflict with itself.
	same := []codex.WorkerDispatch{
		{WorkerName: "Anvil-1", Caste: "builder", Wave: 1, DeclaredPaths: []string{"cmd/foo.go"}},
		{WorkerName: "Anvil-1", Caste: "builder", Wave: 1, DeclaredPaths: []string{"cmd/foo.go"}},
	}
	if err := validateDeclaredWorktreeOwnership(same); err != nil {
		t.Fatalf("one worker was refused for colliding with itself: %v", err)
	}
}
