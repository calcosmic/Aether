package cmd

import (
	"bytes"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
)

// The spend view is an INSPECTION path, and CLAUDE.md makes "an inspection or
// --dry-run command must not mutate state" a named rule with its own history
// in this repository: consolidation-phase-end --dry-run and
// consolidation-seal --dry-run both wrote to instincts.json for months while
// their own flag documentation read "Report without modifying". These tests
// therefore prove the property against the whole store by content hash rather
// than trusting the command's help text, reusing
// snapshotStoreFileHashesForTest / assertHashSnapshotsEqualForTest from
// spawn_reap_test.go instead of inventing a second snapshot mechanism.

// spendRowCellSeparatorRe splits a rendered row into its cells. Cells are
// separated by two or more spaces; nothing inside a cell ever contains a run
// of two spaces (the job name is whitespace-collapsed before rendering), so
// this is a total parse of the row shape rather than a guess.
var spendRowCellSeparatorRe = regexp.MustCompile(`\s{2,}`)

// setupSpendTestStore installs a fresh store and captures stdout/stderr.
func setupSpendTestStore(t *testing.T) (*bytes.Buffer, string) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf
	return &buf, tmpDir
}

// seedSpendLedgerForTest writes one workflow's ledger through the ledger's own
// save entry point, so the fixture is a real ledger and not a hand-shaped file
// the loader would never have produced.
func seedSpendLedgerForTest(t *testing.T, phase int, workflow string, rows ...spendRow) {
	t.Helper()
	if err := saveSpendLedger(spendLedger{
		Phase:      phase,
		PhaseName:  "see what it cost",
		Workflow:   workflow,
		RecordedAt: "2026-08-27T10:00:00Z",
		Rows:       rows,
	}); err != nil {
		t.Fatalf("seed %s ledger: %v", workflow, err)
	}
}

// measuredSpendRowForTest builds a row whose provider reported an exact total.
func measuredSpendRowForTest(name, caste string, total int64) spendRow {
	return spendRow{
		AgentName: name,
		Caste:     caste,
		Task:      "a task",
		Status:    "completed",
		Usage: codex.WorkerUsage{
			TotalTokens: total,
			Source:      codex.UsageSourceProvider,
		},
	}
}

// runSpendForTest executes the command and returns its stdout.
func runSpendForTest(t *testing.T, buf *bytes.Buffer, args ...string) string {
	t.Helper()
	buf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs(append([]string{"spend"}, args...))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("spend failed: %v", err)
	}
	if code := int(renderedCommandExitCode.Load()); code != 0 {
		t.Fatalf("spend exited %d", code)
	}
	return buf.String()
}

// spendRowCellsForWorker returns the parsed cells of the rendered row naming
// worker. Assertions are made on the FIGURE CELL, never on the whole line:
// deterministic worker names carry digits of their own (Scout Roam-90), so a
// whole-line "contains no digit" check fails on correct output and would be
// weakened by the next person who hits it.
func spendRowCellsForWorker(t *testing.T, output, worker string) []string {
	t.Helper()
	var match []string
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, "  ") {
			continue
		}
		if !strings.Contains(line, worker) {
			continue
		}
		cells := spendRowCellSeparatorRe.Split(strings.TrimSpace(line), -1)
		if len(cells) < 3 {
			t.Fatalf("row for %q parsed into %d cell(s), want at least 3: %q", worker, len(cells), line)
		}
		if match != nil {
			t.Fatalf("worker %q matched more than one rendered row", worker)
		}
		match = cells
	}
	if match == nil {
		t.Fatalf("no rendered row names worker %q; output was:\n%s", worker, output)
	}
	return match
}

// spendRenderedRowLines returns every rendered worker row line.
func spendRenderedRowLines(output string) []string {
	var rows []string
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "  ") && strings.TrimSpace(line) != "" {
			rows = append(rows, line)
		}
	}
	return rows
}

// TestSpendDoesNotMutate is the named lock from 196-CONTEXT.md's "What Phase
// 196 still builds from scratch" item 4. It hashes every file under the store
// before and after, runs the command twice, and requires both the hashes and
// the output to be unchanged -- so an incidental write ANYWHERE, not merely in
// the file the author happened to think of, fails this test.
func TestSpendDoesNotMutate(t *testing.T) {
	buf, _ := setupSpendTestStore(t)

	seedSpendLedgerForTest(t, 196, spendWorkflowBuild,
		measuredSpendRowForTest("Mason-67", "builder", 1200000),
	)
	seedSpendLedgerForTest(t, 196, spendWorkflowContinue,
		measuredSpendRowForTest("Keen-12", "watcher", 220000),
	)
	if err := store.SaveJSON("COLONY_STATE.json", map[string]interface{}{
		"goal":          "see what it cost",
		"current_phase": 196,
	}); err != nil {
		t.Fatalf("seed colony state: %v", err)
	}

	before := snapshotStoreFileHashesForTest(t, store.BasePath())

	firstOutput := runSpendForTest(t, buf)
	afterFirst := snapshotStoreFileHashesForTest(t, store.BasePath())
	assertHashSnapshotsEqualForTest(t, "after first spend run", before, afterFirst)

	secondOutput := runSpendForTest(t, buf)
	afterSecond := snapshotStoreFileHashesForTest(t, store.BasePath())
	assertHashSnapshotsEqualForTest(t, "after second spend run", before, afterSecond)

	if firstOutput != secondOutput {
		t.Fatalf("spend output is not byte-identical across two runs:\nfirst:  %q\nsecond: %q", firstOutput, secondOutput)
	}

	t.Run("the snapshot would catch a write", func(t *testing.T) {
		seedSpendLedgerForTest(t, 196, spendWorkflowBuild,
			measuredSpendRowForTest("Mason-67", "builder", 1200001),
		)
		after := snapshotStoreFileHashesForTest(t, store.BasePath())
		same := len(after) == len(before)
		if same {
			for name, hash := range before {
				if after[name] != hash {
					same = false
					break
				}
			}
		}
		if same {
			t.Fatal("the hash snapshot did not notice a real ledger write -- the no-mutation proof above would pass vacuously")
		}
	})
}

// TestSpendShowsEveryWorkerRow proves every figure the command prints is a
// figure some ledger row actually recorded. The expected values are the seeded
// literals, written out here by hand; nothing in this test calls the renderer,
// computeSpendTotals, or BilledTotalTokens to produce what it then asserts.
func TestSpendShowsEveryWorkerRow(t *testing.T) {
	buf, _ := setupSpendTestStore(t)

	seedSpendLedgerForTest(t, 196, spendWorkflowBuild,
		measuredSpendRowForTest("Mason-67", "builder", 1200000),
		measuredSpendRowForTest("Keen-12", "watcher", 220000),
	)

	output := runSpendForTest(t, buf, "--phase", "196")

	for worker, wantFigure := range map[string]string{
		"Mason-67": "1200000",
		"Keen-12":  "220000",
	} {
		cells := spendRowCellsForWorker(t, output, worker)
		if cells[1] != wantFigure {
			t.Errorf("worker %s figure cell = %q, want the seeded literal %q", worker, cells[1], wantFigure)
		}
		if cells[2] != "measured" {
			t.Errorf("worker %s mark = %q, want %q", worker, cells[2], "measured")
		}
	}

	// The total is the ledger's own measured subtotal: 1200000 + 220000,
	// written here as a literal rather than recomputed from the seeds.
	if !strings.Contains(output, "1420000") {
		t.Errorf("output does not carry the measured total 1420000:\n%s", output)
	}

	// No figure appears that was not seeded. Every number-shaped figure cell
	// must be one of the seeded row values or the seeded total.
	seeded := map[string]bool{"1200000": true, "220000": true, "1420000": true}
	for _, line := range spendRenderedRowLines(output) {
		cells := spendRowCellSeparatorRe.Split(strings.TrimSpace(line), -1)
		if len(cells) < 2 {
			continue
		}
		if _, err := strconv.ParseInt(cells[1], 10, 64); err != nil {
			continue
		}
		if !seeded[cells[1]] {
			t.Errorf("figure %q appears in the output but was never seeded into a ledger row", cells[1])
		}
	}
}

// TestSpendEmptyLedgerShowsNoFigure: with nothing recorded the command says so
// in plain words rather than inventing a zero. Asserted structurally -- no
// rendered row and no total line -- rather than by scanning the whole output
// for digits, because the phase number itself is a digit run.
func TestSpendEmptyLedgerShowsNoFigure(t *testing.T) {
	buf, _ := setupSpendTestStore(t)

	output := runSpendForTest(t, buf, "--phase", "196")

	if rows := spendRenderedRowLines(output); len(rows) != 0 {
		t.Errorf("empty ledger rendered %d worker row(s), want none: %v", len(rows), rows)
	}
	if strings.Contains(output, "Total:") {
		t.Errorf("empty ledger rendered a total line, which would be an invented figure:\n%s", output)
	}
	if !strings.Contains(strings.ToLower(output), "nothing has been recorded") {
		t.Errorf("empty ledger output does not say in plain words that nothing was recorded:\n%s", output)
	}
}

// TestSpendShowsNoNumberForUnreportedRows is D-01 as amended: a worker whose
// tool reported nothing shows NO number -- not a zero, not a guess wearing an
// "estimated" label -- and is not counted in the total.
func TestSpendShowsNoNumberForUnreportedRows(t *testing.T) {
	buf, _ := setupSpendTestStore(t)

	seedSpendLedgerForTest(t, 196, spendWorkflowBuild,
		measuredSpendRowForTest("Mason-67", "builder", 1200000),
		spendRow{
			AgentName: "Roam-90",
			Caste:     "scout",
			Task:      "research",
			Status:    "completed",
			Usage:     codex.WorkerUsage{},
		},
		spendRow{
			AgentName: "Quill-31",
			Caste:     "chronicler",
			Task:      "write it up",
			Status:    "completed",
			Usage: codex.WorkerUsage{
				TotalTokens: 80000,
				Source:      codex.UsageSourceEstimate,
			},
		},
	)

	output := runSpendForTest(t, buf, "--phase", "196")

	for _, worker := range []string{"Roam-90", "Quill-31"} {
		cells := spendRowCellsForWorker(t, output, worker)
		if cells[1] != "—" {
			t.Errorf("worker %s figure cell = %q, want the dash sentinel %q -- D-01 as amended forbids any number here", worker, cells[1], "—")
		}
		if cells[2] != "not reported" {
			t.Errorf("worker %s mark = %q, want %q", worker, cells[2], "not reported")
		}
	}

	// The estimate's own figure must not appear anywhere: a marked estimate is
	// exactly what D-01's amendment removed.
	if strings.Contains(output, "80000") {
		t.Errorf("the estimated figure 80000 was rendered; D-01 as amended forbids rendering an estimate at all:\n%s", output)
	}

	measuredCells := spendRowCellsForWorker(t, output, "Mason-67")
	if measuredCells[1] != "1200000" {
		t.Errorf("measured worker figure cell = %q, want %q", measuredCells[1], "1200000")
	}
	if !strings.Contains(output, "Total: 1200000") {
		t.Errorf("the total does not equal the single measured worker's figure -- unreported rows leaked into it:\n%s", output)
	}
	if !strings.Contains(output, "2 workers") {
		t.Errorf("the output does not say how many workers reported nothing:\n%s", output)
	}
}

// TestSpendShowsBuildAndContinueRows: the ledger is keyed by phase AND
// workflow, and a phase routinely runs both. Neither set may erase the other
// in the view.
func TestSpendShowsBuildAndContinueRows(t *testing.T) {
	buf, _ := setupSpendTestStore(t)

	seedSpendLedgerForTest(t, 196, spendWorkflowBuild,
		measuredSpendRowForTest("Mason-67", "builder", 1200000),
	)
	seedSpendLedgerForTest(t, 196, spendWorkflowContinue,
		measuredSpendRowForTest("Keen-12", "watcher", 220000),
	)

	output := runSpendForTest(t, buf, "--phase", "196")

	buildCells := spendRowCellsForWorker(t, output, "Mason-67")
	if buildCells[1] != "1200000" {
		t.Errorf("build worker figure cell = %q, want %q", buildCells[1], "1200000")
	}
	continueCells := spendRowCellsForWorker(t, output, "Keen-12")
	if continueCells[1] != "220000" {
		t.Errorf("continue worker figure cell = %q, want %q", continueCells[1], "220000")
	}
	if !strings.Contains(output, "Total: 1420000") {
		t.Errorf("the total does not span both workflows:\n%s", output)
	}
}
