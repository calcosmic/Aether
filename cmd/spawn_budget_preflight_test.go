package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
)

// Phase 183. From a live run on 2026-08-14:
//
//	"This build started at 17 of 20 helpers already consumed — spent by the
//	 preceding /ant-continue in the same run. A 10-worker build cannot fit."
//	 ... "Queen hit the whole-run helper budget: 27 of 20 helpers already
//	 spawned in this run" ... "37 of 20"
//
// Two things were wrong, and neither is the one it looked like.
//
// The budget is NOT run-scoped across commands: beginRuntimeSpawnRun starts a
// fresh run for build, continue, plan and colonize alike. The carry-over came
// from ghosts. When a worker's completion fails to record, its entry stays
// live, and the tamper rules from Phase 173 deliberately raise the count to
// include live entries outside the current window. The report shows exactly
// that failure: spawn-log returned recorded:true, the matching spawn-complete
// returned `spawn_tree: agent "Chip-49" not found`, and re-running spawn-log
// consumed a second slot for the same worker.
//
// And the ceiling was checked per spawn but never against the whole plan, so a
// ten-worker build with three slots left began anyway and failed in the middle.

func TestSpawnCompleteReleasesBudgetWhenEntryIsMissing(t *testing.T) {
	saveGlobals(t)
	setupBuildFlowTest(t)

	tree := agent.NewSpawnTree(store, "spawn-tree.txt")
	if _, err := tree.BeginRun("build", time.Now().UTC()); err != nil {
		t.Fatalf("begin run: %v", err)
	}

	// A worker the ledger has no record of -- the exact condition that left
	// Chip-49 live and cost a second budget slot.
	err := spawnCompleteTolerant(tree, "Ghost-1", "completed", "finished", time.Now().UTC())
	if err != nil {
		t.Fatalf("completing an unrecorded worker must not fail -- failing is what leaks the budget slot: %v", err)
	}
}

func TestBudgetPreflightRefusesAPlanThatCannotFit(t *testing.T) {
	ok, msg := spawnBudgetPreflight(10, spawnTreeBudget{Max: 20, Consumed: 17, Remaining: 3})
	if ok {
		t.Fatal("a ten-worker plan with three slots left was allowed to begin; it will stall in the middle")
	}
	if !strings.Contains(msg, "10") || !strings.Contains(msg, "3") {
		t.Fatalf("the refusal must say how many were planned and how many remain, got: %q", msg)
	}
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "budget_consumed") || strings.Contains(lower, "spawn_tree") {
		t.Fatalf("the refusal must read as plain English, got: %q", msg)
	}
}

func TestBudgetPreflightAllowsAPlanThatFits(t *testing.T) {
	ok, msg := spawnBudgetPreflight(4, spawnTreeBudget{Max: 20, Consumed: 2, Remaining: 18})
	if !ok {
		t.Fatalf("a four-worker plan with eighteen slots left was refused: %q", msg)
	}
	if msg != "" {
		t.Fatalf("a plan that fits should say nothing, got: %q", msg)
	}
}

// The ceiling is a ceiling, not a suggestion. If a plan is allowed to begin,
// the count it is allowed to reach must never exceed the maximum -- the failure
// record showed 27 and 37 against a cap of 20.
func TestBudgetPreflightNeverAllowsMoreThanTheCap(t *testing.T) {
	for _, tc := range []struct {
		planned  int
		consumed int
	}{{27, 0}, {10, 17}, {21, 0}, {1, 20}} {
		state := spawnTreeBudget{Max: 20, Consumed: tc.consumed}
		state.Remaining = state.Max - state.Consumed
		if state.Remaining < 0 {
			state.Remaining = 0
		}
		ok, _ := spawnBudgetPreflight(tc.planned, state)
		if ok && tc.planned+tc.consumed > state.Max {
			t.Fatalf("allowed %d workers with %d already spent against a cap of %d", tc.planned, tc.consumed, state.Max)
		}
	}
}
