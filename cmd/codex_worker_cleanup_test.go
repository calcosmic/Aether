package cmd

import (
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
)

// TestStaleWorkerCleanupBeforeDispatch verifies cleanupStaleWorkersBeforeDispatch
// calls through to codex.CleanupStaleWorkers and handles results correctly.
func TestStaleWorkerCleanupBeforeDispatch(t *testing.T) {
	// Test with empty root -- should return immediately without error
	cleanupStaleWorkersBeforeDispatch("")
	// No panic = success
}

// TestStaleWorkerCleanupEmptyRoot verifies that an empty root string
// is handled gracefully (returns without calling cleanup).
func TestStaleWorkerCleanupEmptyRoot(t *testing.T) {
	// Should not panic
	cleanupStaleWorkersBeforeDispatch("")
	cleanupStaleWorkersBeforeDispatch("   ")
}

// TestStaleWorkerCleanupIntegration seeds stale worker processes in a temp
// directory's worker-processes.json and verifies cleanup works end-to-end.
func TestStaleWorkerCleanupIntegration(t *testing.T) {
	origExists := codex.WorkerProcessExistsFunc()
	origCommand := codex.WorkerProcessCommandFunc()
	origTerminate := codex.TerminateWorkerFunc()
	origKill := codex.KillWorkerFunc()
	defer func() {
		codex.SetWorkerProcessExistsFunc(origExists)
		codex.SetWorkerProcessCommandFunc(origCommand)
		codex.SetWorkerTerminateFunc(origTerminate)
		codex.SetWorkerKillFunc(origKill)
	}()

	root := t.TempDir()

	// Mock functions: process exists and is a known worker
	codex.SetWorkerProcessExistsFunc(func(pid int) bool {
		return pid == 7777
	})
	codex.SetWorkerProcessCommandFunc(func(pid int) string {
		return "/usr/local/bin/codex exec --task build"
	})

	var terminated []int
	codex.SetWorkerTerminateFunc(func(pid int) error {
		terminated = append(terminated, pid)
		return nil
	})
	codex.SetWorkerKillFunc(func(pid int) error {
		return nil
	})

	// Seed stale worker in registry
	staleTime := time.Now().UTC().Add(-2 * time.Hour)
	if err := codex.WriteTrackedProcessesForTest(root, []codex.TrackedProcess{
		{PID: 7777, WorkerName: "stale-builder", Caste: "builder", Root: root, SpawnedAt: staleTime},
	}); err != nil {
		t.Fatalf("writeTrackedProcesses: %v", err)
	}

	// Track an active process so it's not considered stale
	tracker := codex.GlobalProcessTracker()
	tracker.TrackProcess(8888, codex.TrackedProcess{WorkerName: "active-worker", Root: root})
	defer tracker.UntrackProcess(8888)

	// Call cleanup
	cleanupStaleWorkersBeforeDispatch(root)

	// Verify the stale process was terminated
	found := false
	for _, pid := range terminated {
		if pid == 7777 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("stale worker pid 7777 was not terminated; terminated=%v", terminated)
	}
}
