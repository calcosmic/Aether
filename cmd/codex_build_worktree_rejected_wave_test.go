package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// rejectedWaveFixture builds the exact same-wave collision the 195 review's
// fifth critical finding describes: one worker that finished cleanly and owns
// cmd/foo.go, and a second worker in the same wave that failed part-way, never
// declared cmd/foo.go, but wrote it anyway and submitted a receipt claiming it.
// The failing worker deliberately sits at the HIGHER outcome index, which is
// what decided the winner before the fix.
func rejectedWaveFixture(t *testing.T) (root string, phase colony.Phase, outcomes []*worktreeWaveOutcome) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	dataDir, root := newCalVaultWorktreeRepo(t)

	contested := "cmd/foo.go"
	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0o755); err != nil {
		t.Fatalf("make cmd dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, contested), []byte("package cmd // original\n"), 0o644); err != nil {
		t.Fatalf("seed contested file: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "seed contested file")

	owningID, strayID := "1.1", "1.2"
	owning := colony.Task{ID: &owningID, Goal: "own the contested file", Status: colony.TaskPending, Hints: []string{contested}}
	// Declares nothing at all: the combination that lets a receipt claim any
	// path the worker's own result reported touching.
	stray := colony.Task{ID: &strayID, Goal: "do something else entirely", Status: colony.TaskPending}
	phase = colony.Phase{ID: 1, Name: "Rejected wave", Status: colony.PhaseReady, Tasks: []colony.Task{owning, stray}}

	goal := "Two workers, one file"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateEXECUTING, ColonyDepth: "standard", CurrentPhase: 1,
		ParallelMode: colony.ModeWorktree,
		Plan:         colony.Plan{Phases: []colony.Phase{phase}},
	})

	startedAt := time.Now().UTC()

	owningDispatch := codex.WorkerDispatch{
		WorkerName: "Anvil-1", Caste: "builder", TaskID: owningID,
		CoveredTaskIDs: []string{owningID}, DeclaredPaths: []string{contested},
	}
	owningSession, err := allocateBuildWorktree(root, 1, owningDispatch, startedAt)
	if err != nil {
		t.Fatalf("allocate owning worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(owningSession.AbsPath, contested), []byte("package cmd // written by the declared owner\n"), 0o644); err != nil {
		t.Fatalf("write owning copy: %v", err)
	}

	strayDispatch := codex.WorkerDispatch{
		WorkerName: "Hammer-2", Caste: "builder", TaskID: strayID,
		CoveredTaskIDs: []string{strayID},
	}
	straySession, err := allocateBuildWorktree(root, 1, strayDispatch, startedAt.Add(time.Millisecond))
	if err != nil {
		t.Fatalf("allocate stray worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(straySession.AbsPath, contested), []byte("package cmd // written by the worker that failed\n"), 0o644); err != nil {
		t.Fatalf("write stray copy: %v", err)
	}

	outcomes = []*worktreeWaveOutcome{
		{
			dispatch: owningDispatch,
			session:  owningSession,
			touched:  []string{contested},
			result: codex.DispatchResult{
				Status: "completed",
				WorkerResult: &codex.WorkerResult{
					WorkerName: "Anvil-1", Caste: "builder", TaskID: owningID,
					Status: "completed", Summary: "wrote the file it declared",
					FilesModified: []string{contested},
					Handoff:       codex.WorkerHandoff{VerificationStatus: "pass", CommandsRun: []string{"go build ./..."}},
				},
			},
		},
		{
			dispatch: strayDispatch,
			session:  straySession,
			touched:  []string{contested},
			result: codex.DispatchResult{
				Status: "failed",
				WorkerResult: &codex.WorkerResult{
					WorkerName: "Hammer-2", Caste: "builder", TaskID: strayID,
					Status: "failed", Summary: "died after touching a file it does not own",
					FilesModified: []string{contested},
					TaskReceipts: []codex.TaskReceipt{{
						TaskID: strayID, Status: codex.TaskReceiptStatusCompleted,
						Summary:       "finished " + strayID,
						FilesModified: []string{contested},
						Handoff:       codex.WorkerHandoff{VerificationStatus: "pass", CommandsRun: []string{"go test ./..."}},
					}},
					Handoff: codex.WorkerHandoff{VerificationStatus: "fail", CommandsRun: []string{"go test ./..."}},
				},
			},
		},
	}
	return root, phase, outcomes
}

// TestFailedWorkerReceiptPathsAreVisibleToConflictDetection is the first half
// of the permanent lock for the 195 review's fifth critical finding (CR-05).
//
// Same-wave ownership conflicts were computed only from workers that finished
// cleanly. A worker that failed part-way still gets its receipted files copied
// into the project, so its writes were invisible to the very check that exists
// to stop two workers overwriting each other.
func TestFailedWorkerReceiptPathsAreVisibleToConflictDetection(t *testing.T) {
	_, _, outcomes := rejectedWaveFixture(t)

	conflicts, conflictWorkers := detectWorktreeWaveConflicts(outcomes)
	if len(conflicts) == 0 {
		t.Fatal("no conflict reported when a failed worker's receipt claims a file another same-wave worker declared and wrote; index order alone would decide the winner")
	}
	if !conflictWorkers[1] {
		t.Fatalf("the failed worker whose receipt claims the contested file was not flagged as part of the conflict: %v", conflicts)
	}
}

// TestRejectedWaveNeverSyncsAFailedWorkersReceipt is the second half: even
// once the conflict is seen, the failed worker's receipt-scoped copy-back must
// not run inside a wave the runtime just rejected. The project's own copy of
// the contested file must be exactly what it was before the wave -- neither
// worker's version -- and both branches must be preserved so nothing is lost.
func TestRejectedWaveNeverSyncsAFailedWorkersReceipt(t *testing.T) {
	root, phase, outcomes := rejectedWaveFixture(t)
	contested := filepath.Join(root, "cmd", "foo.go")

	before, err := os.ReadFile(contested)
	if err != nil {
		t.Fatalf("read contested file before the wave: %v", err)
	}

	ledger := newWorktreeReceiptLedger()
	reconcileWorktreeWave(root, phase, 1, outcomes, NewCircuitBreaker(3), ledger)

	after, err := os.ReadFile(contested)
	if err != nil {
		t.Fatalf("read contested file after the wave: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("a rejected wave still wrote into the project: cmd/foo.go was %q before and %q after", string(before), string(after))
	}

	for _, outcome := range outcomes {
		if outcome.result.Status == "completed" {
			t.Fatalf("worker %s was reported completed out of a wave the runtime rejected for conflicts", outcome.dispatch.WorkerName)
		}
	}

	resolved := ledger.entries["Hammer-2"]
	if len(resolved.SyncedPaths) != 0 {
		t.Fatalf("the failed worker's receipt was copied into the project from inside a rejected wave: %v", resolved.SyncedPaths)
	}
	if len(resolved.CompletedTaskIDs) != 0 {
		t.Fatalf("the failed worker was credited for %v inside a rejected wave", resolved.CompletedTaskIDs)
	}

	// Nothing is destroyed: both workers' own copies stay on their branches.
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload colony state: %v", err)
	}
	preserved := 0
	for _, entry := range state.Worktrees {
		if entry.Status == colony.WorktreeOrphaned {
			preserved++
		}
	}
	if preserved != 2 {
		t.Fatalf("expected both workers' copies to be preserved for recovery, got %d preserved of %d tracked", preserved, len(state.Worktrees))
	}
}

// TestConflictFreeWaveStillSyncsAFailedWorkersReceipt proves the CR-05 fix is a
// gate and not a removal: when the wave has no conflict at all, a failed
// worker's proven work is still copied back and still credited, exactly as
// D-08/D-09 intend.
func TestConflictFreeWaveStillSyncsAFailedWorkersReceipt(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir, root := newCalVaultWorktreeRepo(t)

	taskID := "1.1"
	proven := taskFileName(taskID)
	task := colony.Task{ID: &taskID, Goal: "write the proven file", Status: colony.TaskPending, Hints: []string{proven}}
	phase := colony.Phase{ID: 1, Name: "Clean wave", Status: colony.PhaseReady, Tasks: []colony.Task{task}}

	goal := "One worker, no collision"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateEXECUTING, ColonyDepth: "standard", CurrentPhase: 1,
		ParallelMode: colony.ModeWorktree,
		Plan:         colony.Plan{Phases: []colony.Phase{phase}},
	})

	dispatch := codex.WorkerDispatch{
		WorkerName: "Hammer-2", Caste: "builder", TaskID: taskID,
		CoveredTaskIDs: []string{taskID}, DeclaredPaths: []string{proven},
	}
	session, err := allocateBuildWorktree(root, 1, dispatch, time.Now().UTC())
	if err != nil {
		t.Fatalf("allocate worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(session.AbsPath, proven), []byte("package fixture\n"), 0o644); err != nil {
		t.Fatalf("write proven file: %v", err)
	}

	outcomes := []*worktreeWaveOutcome{{
		dispatch: dispatch,
		session:  session,
		touched:  []string{proven},
		result: codex.DispatchResult{
			Status: "failed",
			WorkerResult: &codex.WorkerResult{
				WorkerName: "Hammer-2", Caste: "builder", TaskID: taskID,
				Status: "failed", Summary: "died after finishing its one task",
				FilesModified: []string{proven},
				TaskReceipts:  []codex.TaskReceipt{worktreeReceiptForTask(taskID)},
				Handoff:       codex.WorkerHandoff{VerificationStatus: "fail", CommandsRun: []string{"go test ./..."}},
			},
		},
	}}

	ledger := newWorktreeReceiptLedger()
	reconcileWorktreeWave(root, phase, 1, outcomes, NewCircuitBreaker(3), ledger)

	if _, err := os.Stat(filepath.Join(root, proven)); err != nil {
		t.Fatalf("a conflict-free wave failed to copy the failed worker's proven file into the project: %v", err)
	}
	resolved := ledger.entries["Hammer-2"]
	if len(resolved.CompletedTaskIDs) != 1 || resolved.CompletedTaskIDs[0] != taskID {
		t.Fatalf("a conflict-free wave lost the failed worker's honest partial credit: %v", resolved.CompletedTaskIDs)
	}
}
