package cmd

import (
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// TestGroupedWorkerGetsTimeForEveryTask is Phase 210 blocker 8 (French
// Basics, 2026-09-26): phase 2's four related tasks were folded into one
// builder (the coherent-job rule), but that builder kept the single-task
// ten-minute limit and was cut off mid-way through authoring ~100 cards. A
// grouped worker gets the base limit per task it covers, capped so a runaway
// worker is still stopped.
func TestGroupedWorkerGetsTimeForEveryTask(t *testing.T) {
	saveGlobals(t)
	_, root := setupTestStore(t)
	t.Setenv("AETHER_ROOT", root)

	phase := colony.Phase{ID: 2, Name: "Author the deck"}
	dispatches := []codexBuildDispatch{
		{Name: "Solo-1", Caste: "builder", Task: "one task", TaskID: "2.1"},
		{Name: "Anvil-88", Caste: "builder", Task: "four tasks", TaskID: "2.1", CoveredTaskIDs: []string{"2.1", "2.2", "2.3", "2.4"}},
		{Name: "Big-9", Caste: "builder", Task: "six tasks", TaskID: "3.1", CoveredTaskIDs: []string{"3.1", "3.2", "3.3", "3.4", "3.5", "3.6"}},
	}
	built, err := buildCodexWorkerDispatches(root, phase, dispatches, time.Now().UTC(), &codex.FakeInvoker{}, "", codex.DefaultWorkerTimeout, nil)
	if err != nil {
		t.Fatalf("buildCodexWorkerDispatches: %v", err)
	}
	want := map[string]time.Duration{
		"Solo-1":   codex.DefaultWorkerTimeout,
		"Anvil-88": 4 * codex.DefaultWorkerTimeout,
		"Big-9":    groupedWorkerTimeoutCap,
	}
	for _, dispatch := range built {
		if dispatch.Timeout != want[dispatch.WorkerName] {
			t.Errorf("%s timeout = %v, want %v", dispatch.WorkerName, dispatch.Timeout, want[dispatch.WorkerName])
		}
	}
	if groupedWorkerTimeoutCap > internalWorkerTimeoutMax {
		t.Fatalf("grouped cap %v exceeds the worker adapter's own ceiling %v", groupedWorkerTimeoutCap, internalWorkerTimeoutMax)
	}
}
