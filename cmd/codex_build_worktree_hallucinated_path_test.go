package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// TestOneHallucinatedReceiptPathDoesNotBlockACleanWorker is the permanent
// regression lock for NEW-04 (195-REVIEW.iter2.md).
//
// Making a failed worker's receipted files visible to the same-wave collision
// check was right, but the paths were fed in raw: no check that the receipt
// covered a task this worker was actually given, that it reported success, or
// that the worker's own result ever said it touched the file. So a failed
// worker naming a file it never went near -- an ordinary AI output error --
// collided with the real owner and cancelled the whole round, throwing away
// every other worker's finished work with it. Those paths were never
// copy-back candidates in the first place; they are refused before they reach
// the project.
func TestOneHallucinatedReceiptPathDoesNotBlockACleanWorker(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir, root := newCalVaultWorktreeRepo(t)

	owned := "cmd/foo.go"
	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0o755); err != nil {
		t.Fatalf("make cmd dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, owned), []byte("package cmd // original\n"), 0o644); err != nil {
		t.Fatalf("seed owned file: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "seed owned file")

	owningID, strayID := "1.1", "1.2"
	owning := colony.Task{ID: &owningID, Goal: "own the file", Status: colony.TaskPending, Hints: []string{owned}}
	strayFile := "cmd/bar.go"
	stray := colony.Task{ID: &strayID, Goal: "write something else", Status: colony.TaskPending, Hints: []string{strayFile}}
	phase := colony.Phase{ID: 1, Name: "Hallucinated path", Status: colony.PhaseReady, Tasks: []colony.Task{owning, stray}}

	goal := "One bad path must not cancel the round"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateEXECUTING, ColonyDepth: "standard", CurrentPhase: 1,
		ParallelMode: colony.ModeWorktree,
		Plan:         colony.Plan{Phases: []colony.Phase{phase}},
	})

	startedAt := time.Now().UTC()

	owningDispatch := codex.WorkerDispatch{
		WorkerName: "Anvil-1", Caste: "builder", TaskID: owningID,
		CoveredTaskIDs: []string{owningID}, DeclaredPaths: []string{owned},
	}
	owningSession, err := allocateBuildWorktree(root, 1, owningDispatch, startedAt)
	if err != nil {
		t.Fatalf("allocate owning worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(owningSession.AbsPath, owned), []byte("package cmd // written by the declared owner\n"), 0o644); err != nil {
		t.Fatalf("write owning copy: %v", err)
	}

	strayDispatch := codex.WorkerDispatch{
		WorkerName: "Hammer-2", Caste: "builder", TaskID: strayID,
		CoveredTaskIDs: []string{strayID}, DeclaredPaths: []string{strayFile},
	}
	straySession, err := allocateBuildWorktree(root, 1, strayDispatch, startedAt.Add(time.Millisecond))
	if err != nil {
		t.Fatalf("allocate stray worktree: %v", err)
	}

	outcomes := []*worktreeWaveOutcome{
		{
			dispatch: owningDispatch,
			session:  owningSession,
			touched:  []string{owned},
			result: codex.DispatchResult{
				Status: "completed",
				WorkerResult: &codex.WorkerResult{
					WorkerName: "Anvil-1", Caste: "builder", TaskID: owningID,
					Status: "completed", Summary: "wrote the file it declared",
					FilesModified: []string{owned},
					Handoff:       codex.WorkerHandoff{VerificationStatus: "pass", CommandsRun: []string{"go build ./..."}},
				},
			},
		},
		{
			dispatch: strayDispatch,
			session:  straySession,
			// It touched nothing at all: the receipt below is simply wrong.
			touched: nil,
			result: codex.DispatchResult{
				Status: "failed",
				WorkerResult: &codex.WorkerResult{
					WorkerName: "Hammer-2", Caste: "builder", TaskID: strayID,
					Status: "failed", Summary: "died having written nothing",
					TaskReceipts: []codex.TaskReceipt{{
						TaskID: strayID, Status: codex.TaskReceiptStatusCompleted,
						Summary:       "finished " + strayID,
						FilesModified: []string{owned}, // never touched; invented
						Handoff:       codex.WorkerHandoff{VerificationStatus: "pass", CommandsRun: []string{"go test ./..."}},
					}},
					Handoff: codex.WorkerHandoff{VerificationStatus: "fail", CommandsRun: []string{"go test ./..."}},
				},
			},
		},
	}

	conflicts, _ := detectWorktreeWaveConflicts(root, outcomes)
	if len(conflicts) != 0 {
		t.Fatalf("a file the failed worker's own result never reported touching was treated as a real collision, so the whole round is cancelled: %v", conflicts)
	}

	reconcileWorktreeWave(root, phase, 1, outcomes, NewCircuitBreaker(3), newWorktreeReceiptLedger())

	// The decisive evidence is the project itself: a cancelled round copies
	// nothing back at all, so the file would still hold its original text.
	after, err := os.ReadFile(filepath.Join(root, owned))
	if err != nil {
		t.Fatalf("read owned file after the wave: %v", err)
	}
	if string(after) != "package cmd // written by the declared owner\n" {
		t.Fatalf("the clean worker's finished work never reached the project; cmd/foo.go is %q", string(after))
	}
	if outcomes[0].result.Status == "blocked" {
		t.Fatalf("the worker that finished cleanly was held back because another worker named a file it never touched: %v", outcomes[0].result.Error)
	}
}

// TestAnOutOfScopeReceiptPathDoesNotBlockACleanWorker is the same rule from
// the other side: a receipt for a task this worker was never given could
// never be admitted, so it must not cancel the round either.
func TestAnOutOfScopeReceiptPathDoesNotBlockACleanWorker(t *testing.T) {
	root, _, outcomes := rejectedWaveFixture(t)

	// Re-point the failed worker's receipt at a task it does not cover. Its
	// own result still reports the contested file, so only the scope rule can
	// save the round here.
	outcomes[1].result.WorkerResult.TaskReceipts[0].TaskID = "9.9"

	conflicts, _ := detectWorktreeWaveConflicts(root, outcomes)
	if len(conflicts) != 0 {
		t.Fatalf("a receipt for a task the worker was never assigned was treated as a real collision: %v", conflicts)
	}
}
