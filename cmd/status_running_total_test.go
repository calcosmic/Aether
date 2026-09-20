package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 201 plan 06 — Task 3. The status dashboard's colony-wide running
// total: elapsed time and reported cost, summed through the SAME durable
// per-phase facts every other screen in this repository already reads --
// loadSpendLedgersForPhase for cost (the identical loader
// cmd/spend_cost_line.go uses) and loadLatestBuildAttempt for elapsed (the
// identical attempt-bound source Task 1 reads from) -- never a second
// accounting path. The unreported/unmeasured counts are named rather than
// folded silently into the total.

// statusRunningTotalFixture seeds a colony spanning three phases with a
// deliberate mix:
//   - phase 1: fully measured cost AND a fully timestamped attempt.
//   - phase 2: one measured row plus one unreported row, and an attempt
//     missing its end timestamp (unmeasured elapsed).
//   - phase 3: no ledger rows and no attempt at all -- proving an absent
//     phase contributes nothing to either total or either count, the same
//     distinction Task 1 established for one phase's own cost-and-time
//     block (a genuinely absent attempt says nothing, a present-but-
//     incomplete one says "unmeasured").
func statusRunningTotalFixture(t *testing.T) colony.ColonyState {
	t.Helper()
	seedSpendLedgerForTest(t, 1, spendWorkflowBuild,
		measuredSpendRowForTest("Mason-67", "builder", 500_000),
		measuredSpendRowForTest("Keen-12", "watcher", 300_000),
	)
	seedSpendElapsedAttemptForTest(t, 1, "attempt-running-total-phase-1",
		"2026-01-01T00:00:00Z", "2026-01-01T00:10:00Z")

	seedSpendLedgerForTest(t, 2, spendWorkflowBuild,
		measuredSpendRowForTest("Roam-90", "scout", 200_000),
		// Nothing at all was recorded for this worker -- unreported.
		spendRow{AgentName: "Guess-11", Caste: "scout", Task: "research", Status: "completed"},
	)
	seedSpendElapsedAttemptForTest(t, 2, "attempt-running-total-phase-2",
		"2026-01-01T01:00:00Z", "")

	return colony.ColonyState{
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Phase one"},
			{ID: 2, Name: "Phase two"},
			{ID: 3, Name: "Phase three, never run"},
		}},
	}
}

func TestStatusRunningTotalEqualsTheSumOfItsRows(t *testing.T) {
	setupSpendTestStore(t)
	state := statusRunningTotalFixture(t)

	total := computeColonyRunningSpendTotal(state)

	if want := 10 * time.Minute; total.ElapsedMeasured != want {
		t.Errorf("ElapsedMeasured = %s, want %s (phase 1's 10 minutes only -- phase 2 is unmeasured, phase 3 has no attempt at all)", total.ElapsedMeasured, want)
	}
	if total.ElapsedUnmeasured != 1 {
		t.Errorf("ElapsedUnmeasured = %d, want 1 (phase 2's attempt is missing its end timestamp)", total.ElapsedUnmeasured)
	}
	if want := int64(500_000 + 300_000 + 200_000); total.CostTokens != want {
		t.Errorf("CostTokens = %d, want %d (the three measured rows summed by hand)", total.CostTokens, want)
	}
	if total.CostReportedRows != 3 {
		t.Errorf("CostReportedRows = %d, want 3", total.CostReportedRows)
	}
	if total.CostUnreportedRows != 1 {
		t.Errorf("CostUnreportedRows = %d, want 1 (Guess-11's row carries no usage)", total.CostUnreportedRows)
	}

	visual := renderColonyRunningSpendTotal(total)
	if !strings.Contains(visual, "unreported") {
		t.Errorf("rendered total does not name the unreported count:\n%s", visual)
	}
	if !strings.Contains(visual, "unmeasured") {
		t.Errorf("rendered total does not name the unmeasured count:\n%s", visual)
	}
	if !strings.Contains(visual, "10m0s") {
		t.Errorf("rendered total does not show the summed elapsed time (10m0s):\n%s", visual)
	}
	if !strings.Contains(visual, "1.0M") {
		t.Errorf("rendered total does not show the summed cost (1.0M):\n%s", visual)
	}
}

// TestStatusRunningTotalWritesNothing proves the running total is a pure
// read: no ledger file, no lock file, no state mutation. The first call
// against fixture paths never touched before legitimately creates the
// storage layer's own first-touch lock bookkeeping (pkg/storage.FileLocker
// -- the same documented exception cmd/memory_details_render_test.go's own
// snapshotDirFiles carries), so this test warms every path once before
// taking its "before" snapshot -- isolating "does calling this AGAIN write
// anything" (the actual no-mutation property D-06 requires) from a
// storage-layer concern this task did not touch.
func TestStatusRunningTotalWritesNothing(t *testing.T) {
	_, tmpDir := setupSpendTestStore(t)
	state := statusRunningTotalFixture(t)

	// Warm every path the compute pass will touch.
	_ = computeColonyRunningSpendTotal(state)

	before := snapshotStoreFileHashesForTest(t, tmpDir)
	total := computeColonyRunningSpendTotal(state)
	_ = renderColonyRunningSpendTotal(total)
	after := snapshotStoreFileHashesForTest(t, tmpDir)

	assertHashSnapshotsEqualForTest(t, "colony running spend total", before, after)
}
