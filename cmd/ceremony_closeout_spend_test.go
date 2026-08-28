package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 196 plan 07 — "exactly one cost line ends every build and every check,
// on every lane."
//
// These tests COUNT occurrences rather than asserting presence. A test that
// only checks the block appears cannot catch a second one appearing, and two
// cost lines disagreeing with each other is worse than one: the reader has no
// way to tell which is the run's actual cost.
//
// A run can end on two different screens, and only one of them per run:
//
//   - the wrapper lane, where the platform spawns the workers itself, then
//     runs the finalizer and finally `aether ceremony closeout`;
//   - the direct lane, where `aether build <n>` and `aether continue` run the
//     workers in-process and print their own ending screen.
//
// Each is driven end to end below and each must yield exactly one block.

// countCostLineBlocks counts the rendered cost blocks in any output, with
// colour stripped so the count is not defeated by escape codes.
func countCostLineBlocks(output string) int {
	return strings.Count(stripANSI(output), spendCostLineHeading)
}

// seedCostLineColonyForTest installs a store holding a colony at phase 1 with
// recorded rows for it, so every lane below has something real to render.
func seedCostLineColonyForTest(t *testing.T) string {
	t.Helper()
	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s

	goal := "See what it cost"
	if err := s.SaveJSON("COLONY_STATE.json", colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateBUILT,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "See what it cost"},
			{ID: 2, Name: "Next phase"},
		}},
	}); err != nil {
		t.Fatalf("seed colony state: %v", err)
	}
	seedSpendLedgerForTest(t, 1, spendWorkflowBuild,
		measuredSpendRowForTest("Mason-67", "builder", 1_200_000),
		spendRow{AgentName: "Roam-90", Caste: "scout", Task: "research", Status: "completed"},
	)
	seedSpendLedgerForTest(t, 1, spendWorkflowContinue,
		measuredSpendRowForTest("Keen-12", "watcher", 220_000),
	)
	return tmpDir
}

// costLineCompletionFileForTest writes the completion packet a finalizer hands
// to the closeout, naming phase 1.
func costLineCompletionFileForTest(t *testing.T, workflow string) string {
	t.Helper()
	return writeCeremonyTestJSON(t, map[string]interface{}{
		"phase":      1,
		"phase_name": "See what it cost",
		"workflow":   workflow,
		"dispatches": []map[string]interface{}{
			{"name": "Mason-67", "caste": "builder", "status": "completed", "summary": "wrote it"},
		},
	})
}

func TestBuildEndsWithOneCostLine(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	seedCostLineColonyForTest(t)

	_, visual := renderCeremonyCloseout("build", costLineCompletionFileForTest(t, "build"))

	if got := countCostLineBlocks(visual); got != 1 {
		t.Fatalf("the build's ending screen carries %d cost line block(s), want exactly 1:\n%s", got, visual)
	}
	clean := stripANSI(visual)
	if !strings.Contains(clean, "1.2M") {
		t.Errorf("the build's ending screen does not show the recorded total:\n%s", visual)
	}
	if !strings.Contains(clean, "did not report usage") {
		t.Errorf("the build's ending screen does not mark the worker whose tool reported nothing:\n%s", visual)
	}
	if match := spendMoneyRe.FindString(clean); match != "" {
		t.Errorf("the build's ending screen shows a monetary amount (%q):\n%s", match, visual)
	}
}

func TestContinueEndsWithOneCostLine(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	seedCostLineColonyForTest(t)

	_, visual := renderCeremonyCloseout("continue", costLineCompletionFileForTest(t, "continue"))

	if got := countCostLineBlocks(visual); got != 1 {
		t.Fatalf("the check's ending screen carries %d cost line block(s), want exactly 1:\n%s", got, visual)
	}
	if match := spendMoneyRe.FindString(stripANSI(visual)); match != "" {
		t.Errorf("the check's ending screen shows a monetary amount (%q):\n%s", match, visual)
	}
}

func TestCloseoutCostLineCountIsAssertedNotAssumed(t *testing.T) {
	// Anti-vacuity: the counter must actually be able to see two blocks. A
	// counting test whose counter cannot count past one proves nothing.
	saveGlobals(t)
	resetRootCmd(t)
	seedCostLineColonyForTest(t)

	_, visual := renderCeremonyCloseout("build", costLineCompletionFileForTest(t, "build"))
	if got := countCostLineBlocks(visual + visual); got != 2 {
		t.Fatalf("the counter reported %d for a doubled screen, want 2 — it cannot detect a second cost line", got)
	}
	if got := countCostLineBlocks("no cost line here"); got != 0 {
		t.Fatalf("the counter reported %d for a screen with no cost line, want 0", got)
	}
}

func TestNoLaneRendersTwoCostLines(t *testing.T) {
	t.Run("the wrapper lane: the finalizer stays silent and the ending screen speaks once", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		seedCostLineColonyForTest(t)

		var lane bytes.Buffer

		// The finalizer runs as a machine surface and must not print the block
		// itself; if it did, the wrapper lane would end with two.
		finalizeVisual := renderContinueVisual(
			colony.ColonyState{Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "See what it cost"}}}, CurrentPhase: 1},
			colony.Phase{ID: 1, Name: "See what it cost"},
			nil, false, nil,
			map[string]interface{}{"phase": 1},
			colony.VerificationDepthStandard,
		)
		lane.WriteString(finalizeVisual)

		_, closeoutVisual := renderCeremonyCloseout("continue", costLineCompletionFileForTest(t, "continue"))
		lane.WriteString(closeoutVisual)

		if got := countCostLineBlocks(lane.String()); got != 1 {
			t.Fatalf("driving the wrapper lane yielded %d cost line block(s), want exactly 1:\n%s", got, lane.String())
		}
	})

	t.Run("the direct lane: the build's own ending screen speaks once", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))
		withTestWorkspace(t, root)
		withWorkingDir(t, root)
		t.Setenv("AETHER_OUTPUT_MODE", "visual")

		goal := "See what the direct lane cost"
		taskID := "1.1"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version: "3.0",
			Goal:    &goal,
			State:   colony.StateREADY,
			Plan: colony.Plan{Phases: []colony.Phase{{
				ID:     1,
				Name:   "Direct lane",
				Status: colony.PhaseReady,
				Tasks:  []colony.Task{{ID: &taskID, Goal: "Write the thing", Status: colony.TaskPending}},
			}}},
		})

		var buf bytes.Buffer
		stdout = &buf
		rootCmd.SetArgs([]string{"build", "1"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("build returned error: %v", err)
		}
		if got := countCostLineBlocks(buf.String()); got != 1 {
			t.Fatalf("driving the direct build lane yielded %d cost line block(s), want exactly 1:\n%s", got, buf.String())
		}
	})

	t.Run("the direct lane: the check's own ending screen speaks once", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		root, _, _, _ := setupIntermediateContinueState(t, "Direct check lane")
		_ = root
		t.Setenv("AETHER_OUTPUT_MODE", "visual")

		var buf bytes.Buffer
		stdout = &buf
		rootCmd.SetArgs([]string{"continue", "--verification-depth", "light"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("continue returned error: %v", err)
		}
		if got := countCostLineBlocks(buf.String()); got != 1 {
			t.Fatalf("driving the direct check lane yielded %d cost line block(s), want exactly 1:\n%s", got, buf.String())
		}
	})
}
