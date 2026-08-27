package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// refusalFixturePhase is a two-task phase whose tasks declare nothing, so the
// only thing that can refuse a receipt is the receipt itself.
func refusalFixturePhase() colony.Phase {
	a, b := "1.1", "1.2"
	return colony.Phase{
		ID:   1,
		Name: "Receipt refusal fixture",
		Tasks: []colony.Task{
			{ID: &a, Goal: "First step", Status: colony.TaskPending},
			{ID: &b, Goal: "Second step", Status: colony.TaskPending},
		},
	}
}

// TestRefusedReceiptsAreVisibleToTheOwner is the permanent regression lock for
// the 195 review's first warning (WR-01).
//
// The receipt boundary refuses a malformed or out-of-scope receipt by one of
// twelve named rules, and the file that implements it says in its own header
// that a refusal is "never a silent drop". Both in-repo lanes then threw the
// refusal list away (`admission, _ :=` / `claims, completedTaskIDs, _ :=`), so
// a wrapper mis-shaping every receipt it submitted was indistinguishable from a
// worker that legitimately finished nothing.
//
// This test submits receipts that are refused for reasons NOT covered by the
// two rules the worktree lane already printed, and fails if the owner is told
// nothing.
func TestRefusedReceiptsAreVisibleToTheOwner(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	root := dataDir[:len(dataDir)-len("/.aether/data")]

	phase := refusalFixturePhase()

	// The worker really did write this file, and its own result reports it.
	if err := os.WriteFile(filepath.Join(root, "kept.go"), []byte("package fixture\n"), 0644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}

	passHandoff := codex.WorkerHandoff{
		VerificationStatus: "pass",
		CommandsRun:        []string{"go build ./..."},
	}
	dispatch := codexBuildDispatch{
		Name:           "Mason-1",
		Caste:          "builder",
		Stage:          "wave",
		TaskID:         "1.1",
		CoveredTaskIDs: []string{"1.1"},
		JobName:        "automatic-two-step",
		Status:         "failed",
		Outputs:        []string{"kept.go"},
		TaskReceipts: []codex.TaskReceipt{
			{
				// Out of scope: 1.2 was never assigned to this dispatch.
				TaskID:        "1.2",
				Status:        codex.TaskReceiptStatusCompleted,
				Summary:       "also did the second step",
				FilesCreated:  []string{"kept.go"},
				FilesModified: []string{},
				TestsWritten:  []string{},
				Handoff:       passHandoff,
			},
			{
				// In scope, but claims a file the worker's own result never
				// reported touching.
				TaskID:        "1.1",
				Status:        codex.TaskReceiptStatusCompleted,
				Summary:       "did the first step",
				FilesCreated:  []string{"never-reported.go"},
				FilesModified: []string{},
				TestsWritten:  []string{},
				Handoff:       passHandoff,
			},
		},
	}

	stderr = &bytes.Buffer{}
	resolved := resolveCoherentJobDispatchReceipts(root, phase, []codexBuildDispatch{dispatch})

	if len(resolved) != 1 {
		t.Fatalf("resolveCoherentJobDispatchReceipts returned %d dispatches, want 1", len(resolved))
	}
	if len(resolved[0].CompletedTaskIDs) != 0 {
		t.Fatalf("no receipt here is admissible, yet %v was credited", resolved[0].CompletedTaskIDs)
	}

	out := stderr.(*bytes.Buffer).String()
	if strings.TrimSpace(out) == "" {
		t.Fatal("both receipts were refused and the owner was told nothing; a wrapper mis-shaping every receipt looks exactly like a worker that finished nothing")
	}
	for _, want := range []string{"1.2", "never-reported.go", "Mason-1"} {
		if !strings.Contains(out, want) {
			t.Errorf("refusal output never mentions %q; got:\n%s", want, out)
		}
	}
}

// TestRefusedReceiptsAreVisibleOnTheWorktreeLane covers the same gap on the
// external worktree lane, which never read outcome.Violations at all.
func TestRefusedReceiptsAreVisibleOnTheWorktreeLane(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	root := dataDir[:len(dataDir)-len("/.aether/data")]

	checkout := t.TempDir()
	if err := os.WriteFile(filepath.Join(checkout, "kept.go"), []byte("package fixture\n"), 0644); err != nil {
		t.Fatalf("write worker checkout file: %v", err)
	}

	phase := refusalFixturePhase()
	state := colony.ColonyState{
		Worktrees: []colony.WorktreeEntry{{
			ID:     "wt-1",
			Branch: "aether/phase-1/mason-1",
			Path:   checkout,
			Status: colony.WorktreeInProgress,
			Phase:  1,
			Agent:  "Mason-1",
		}},
	}
	dispatch := codexBuildDispatch{
		Name:           "Mason-1",
		CoveredTaskIDs: []string{"1.1"},
		Status:         "failed",
		Outputs:        []string{"kept.go"},
		TaskReceipts: []codex.TaskReceipt{{
			// Out of scope: 1.2 was never assigned to this dispatch.
			TaskID:        "1.2",
			Status:        codex.TaskReceiptStatusCompleted,
			Summary:       "also did the second step",
			FilesCreated:  []string{"kept.go"},
			FilesModified: []string{},
			TestsWritten:  []string{},
			Handoff: codex.WorkerHandoff{
				VerificationStatus: "pass",
				CommandsRun:        []string{"go build ./..."},
			},
		}},
	}

	stderr = &bytes.Buffer{}
	resolved := resolveWorktreeExternalDispatchReceipts(root, phase, state, 1, []codexBuildDispatch{dispatch})
	if len(resolved[0].CompletedTaskIDs) != 0 {
		t.Fatalf("an out-of-scope receipt was credited: %v", resolved[0].CompletedTaskIDs)
	}
	out := stderr.(*bytes.Buffer).String()
	if !strings.Contains(out, "1.2") {
		t.Fatalf("the worktree lane refused task 1.2's receipt and told the owner nothing; got:\n%s", out)
	}
}

// TestReceiptAliasesAreAcceptedOnEveryLane is the permanent regression lock for
// the 195 review's eighth warning (WR-08).
//
// The runtime's own handoff validator explicitly accepts "passed" as a spelling
// of "pass". The direct worker path normalized it before checking; the
// wrapper/external path decoded the receipt straight off the submitted packet
// and tested the raw text, so an identical receipt was credited on one path and
// refused on the other. Two lanes crediting different work from identical
// evidence is exactly what this phase set out to stop.
func TestReceiptAliasesAreAcceptedOnEveryLane(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	root := dataDir[:len(dataDir)-len("/.aether/data")]

	if err := os.WriteFile(filepath.Join(root, "kept.go"), []byte("package fixture\n"), 0644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}

	phase := refusalFixturePhase()
	dispatch := codexBuildDispatch{
		Name:           "Mason-1",
		Caste:          "builder",
		Stage:          "wave",
		TaskID:         "1.1",
		CoveredTaskIDs: []string{"1.1"},
		JobName:        "automatic-two-step",
		Status:         "failed",
		Outputs:        []string{"kept.go"},
		TaskReceipts: []codex.TaskReceipt{{
			TaskID:        "1.1",
			Status:        "Completed",
			Summary:       "did the first step",
			FilesCreated:  []string{"kept.go"},
			FilesModified: []string{},
			TestsWritten:  []string{},
			Handoff: codex.WorkerHandoff{
				// "passed" is an alias the runtime's own handoff validator
				// accepts. So is "not run" for "not_run".
				VerificationStatus: "passed",
				CommandsRun:        []string{"go build ./..."},
			},
		}},
	}

	resolved := resolveCoherentJobDispatchReceipts(root, phase, []codexBuildDispatch{dispatch})
	if len(resolved[0].CompletedTaskIDs) != 1 || resolved[0].CompletedTaskIDs[0] != "1.1" {
		t.Fatalf("a receipt spelling its passing check \"passed\" instead of \"pass\" was refused on the wrapper path; credited = %v", resolved[0].CompletedTaskIDs)
	}
}
