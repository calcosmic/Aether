package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
)

// Phase 196 plan 07 — D-01 as amended (owner, 2026-08-27).
//
// This is the line the owner actually reads at the end of every build and
// every check. His own ruling, in full:
//
//	Cost: 1.4M tokens across 3 workers
//	  Builder Mason-67    1.2M  measured
//	  Watcher Keen-12     220K  measured
//	  Scout Roam-90          —  not reported
//	(1 worker's tool did not report usage)
//
// A worker whose tool reported nothing shows NO number — not a zero, not an
// estimate wearing a label. The total counts only the measured workers and
// says so. No currency figure appears anywhere.
//
// Assertions on an unreported worker are made on the PARSED FIGURE CELL, never
// on the whole row: deterministic worker names carry digits of their own
// (Scout Roam-90 is in the owner's own example), so a whole-row "contains no
// digit" check fails on correct output and invites the next person to weaken
// it. This is the same cell-parsing discipline plan 196-06 established for the
// detail view, reusing its separator.

// spendMoneyRe matches a monetary amount: a currency symbol next to a figure,
// or a figure next to a currency word. Asserted on the RENDERED OUTPUT, never
// on the source file, so a comment can neither satisfy nor break the check.
var spendMoneyRe = regexp.MustCompile(`[$£€¥]\s*[0-9]|[0-9]\s*(?i:USD|EUR|GBP|dollars?|cents?)\b|(?i:\bUSD\b)`)

// spendCostLineJargon is the repository vocabulary that must never reach this
// block. Every one of these words means nothing to someone who has not opened
// a file here, and CLAUDE.md makes translating them a hard rule on every
// owner-facing surface.
var spendCostLineJargon = []string{
	"colony", "caste", "pheromone", "midden", "instinct", "hive", "ratchet",
	"orphan", "entomb", "allowlist", "queen", "ledger", "closeout", "wrapper",
	"dispatch", "manifest", "packet", "brief", "seal", "brood",
}

// costLineRowCells parses the rendered row naming worker into its cells.
func costLineRowCells(t *testing.T, block, worker string) []string {
	t.Helper()
	var match []string
	for _, line := range strings.Split(stripANSI(block), "\n") {
		if !strings.HasPrefix(line, "  ") || !strings.Contains(line, worker) {
			continue
		}
		cells := spendRowCellSeparatorRe.Split(strings.TrimSpace(line), -1)
		if len(cells) < 3 {
			t.Fatalf("row for %q parsed into %d cell(s), want at least 3: %q", worker, len(cells), line)
		}
		if match != nil {
			t.Fatalf("worker %q matched more than one rendered row in:\n%s", worker, block)
		}
		match = cells
	}
	if match == nil {
		t.Fatalf("no rendered row names worker %q; block was:\n%s", worker, block)
	}
	return match
}

// costLineFigureCell is the middle column: the worker's figure, or the dash
// sentinel meaning the tool reported nothing.
func costLineFigureCell(t *testing.T, block, worker string) string {
	t.Helper()
	cells := costLineRowCells(t, block, worker)
	return cells[len(cells)-2]
}

func TestCostLineShowsPerWorkerAndTotal(t *testing.T) {
	setupSpendTestStore(t)
	seedSpendLedgerForTest(t, 196, spendWorkflowBuild,
		measuredSpendRowForTest("Mason-67", "builder", 1_200_000),
		measuredSpendRowForTest("Keen-12", "watcher", 220_000),
	)

	block := renderSpendCostLine(196)

	if !strings.Contains(block, "Cost:") {
		t.Errorf("the block does not open with the run total; block was:\n%s", block)
	}
	if !strings.Contains(block, "1.4M") {
		t.Errorf("the run total is not shown in the owner's compact shape (1.4M); block was:\n%s", block)
	}
	if !strings.Contains(block, "2 workers") {
		t.Errorf("the block does not say how many workers ran; block was:\n%s", block)
	}
	if got := costLineFigureCell(t, block, "Mason-67"); got != "1.2M" {
		t.Errorf("Mason-67's figure cell = %q, want %q", got, "1.2M")
	}
	if got := costLineFigureCell(t, block, "Keen-12"); got != "220K" {
		t.Errorf("Keen-12's figure cell = %q, want %q", got, "220K")
	}
	for _, worker := range []string{"Mason-67", "Keen-12"} {
		cells := costLineRowCells(t, block, worker)
		if cells[len(cells)-1] != spendMarkMeasured {
			t.Errorf("%s is marked %q, want %q", worker, cells[len(cells)-1], spendMarkMeasured)
		}
	}
	for _, jargon := range spendCostLineJargon {
		if strings.Contains(strings.ToLower(stripANSI(block)), jargon) {
			t.Errorf("the cost line uses the repo-invented word %q, which means nothing to the reader:\n%s", jargon, block)
		}
	}
}

func TestUnreportedWorkerShowsNoNumber(t *testing.T) {
	setupSpendTestStore(t)
	seedSpendLedgerForTest(t, 196, spendWorkflowBuild,
		measuredSpendRowForTest("Mason-67", "builder", 1_200_000),
		// Nothing at all was recorded for this worker.
		spendRow{AgentName: "Roam-90", Caste: "scout", Task: "research", Status: "completed"},
		// An ESTIMATE is treated exactly like an absence: D-01 as amended
		// forbids rendering an estimated figure at all, and prompt-character
		// count is the only estimate mechanism in the tree.
		spendRow{
			AgentName: "Guess-11", Caste: "scout", Task: "research", Status: "completed",
			Usage: codex.WorkerUsage{TotalTokens: 999_000, Source: codex.UsageSourceEstimate},
		},
	)

	block := renderSpendCostLine(196)

	for _, worker := range []string{"Roam-90", "Guess-11"} {
		if got := costLineFigureCell(t, block, worker); got != spendNotReportedFigure {
			t.Errorf("%s's figure cell = %q, want exactly the dash sentinel %q — a zero reads as a measurement of nothing rather than an absence of measurement",
				worker, got, spendNotReportedFigure)
		}
		cells := costLineRowCells(t, block, worker)
		if cells[len(cells)-1] != spendMarkNotReported {
			t.Errorf("%s is marked %q, want %q", worker, cells[len(cells)-1], spendMarkNotReported)
		}
	}
	// The estimated figure itself must not reach the block in any form.
	if strings.Contains(stripANSI(block), "999") {
		t.Errorf("an estimated figure reached the cost line:\n%s", block)
	}
	// And the worker whose name carries digits still parses and still shows
	// its real figure — the reason the assertion is on the cell, not the row.
	if got := costLineFigureCell(t, block, "Mason-67"); got != "1.2M" {
		t.Errorf("Mason-67's figure cell = %q, want %q", got, "1.2M")
	}
}

func TestTotalCountsOnlyMeasuredWorkers(t *testing.T) {
	setupSpendTestStore(t)
	// Two measured literals and one unreported worker. The expected total is
	// the two literals added by hand here, never a number the renderer
	// computed for itself.
	seedSpendLedgerForTest(t, 196, spendWorkflowBuild,
		measuredSpendRowForTest("Mason-67", "builder", 1_200_000),
		measuredSpendRowForTest("Keen-12", "watcher", 220_000),
		spendRow{AgentName: "Roam-90", Caste: "scout", Task: "research", Status: "completed"},
	)

	block := stripANSI(renderSpendCostLine(196))

	// 1,200,000 + 220,000 = 1,420,000 -> 1.4M. The unreported worker adds
	// nothing, so a total of anything else means it was counted somehow.
	if !strings.Contains(block, "1.4M") {
		t.Errorf("the total is not the two measured literals summed (1.4M); block was:\n%s", block)
	}
	if !strings.Contains(block, "The total counts") {
		t.Errorf("the block never says what the total counts; block was:\n%s", block)
	}
	if !strings.Contains(block, "2 whose tools reported") {
		t.Errorf("the block does not say the total counts only the 2 measured workers; block was:\n%s", block)
	}
	// The unreported worker is still listed — never quietly dropped to make
	// the total look complete.
	if !strings.Contains(block, "Roam-90") {
		t.Errorf("the unreported worker was dropped from the breakdown; block was:\n%s", block)
	}
	if !strings.Contains(block, "1 worker's tool did not report usage") {
		t.Errorf("the footnote does not count the unreported worker; block was:\n%s", block)
	}
}

func TestFootnoteAbsentWhenEveryWorkerMeasured(t *testing.T) {
	setupSpendTestStore(t)
	seedSpendLedgerForTest(t, 196, spendWorkflowBuild,
		measuredSpendRowForTest("Mason-67", "builder", 1_200_000),
		measuredSpendRowForTest("Keen-12", "watcher", 220_000),
	)

	block := stripANSI(renderSpendCostLine(196))

	if strings.Contains(block, "did not report usage") {
		t.Errorf("a fully measured run still carries the unreported footnote, which explains nothing:\n%s", block)
	}
	if strings.Contains(block, spendNotReportedFigure) {
		t.Errorf("a fully measured run shows the dash sentinel somewhere:\n%s", block)
	}
	if !strings.Contains(block, "The total counts") {
		t.Errorf("the block never says what the total counts; block was:\n%s", block)
	}
}

func TestCostLineHasNoMonetaryFigure(t *testing.T) {
	setupSpendTestStore(t)
	// The shared usage type keeps the provider's own reported cost. The
	// guarantee is not that the field is gone — it is that the renderer never
	// shows it. Feed it one and prove it does not reach the output.
	seedSpendLedgerForTest(t, 196, spendWorkflowBuild,
		spendRow{
			AgentName: "Mason-67", Caste: "builder", Task: "write it", Status: "completed",
			Usage: codex.WorkerUsage{
				TotalTokens: 1_200_000,
				USDCost:     12.34,
				Source:      codex.UsageSourceProvider,
			},
		},
		measuredSpendRowForTest("Keen-12", "watcher", 220_000),
	)

	block := stripANSI(renderSpendCostLine(196))

	if strings.Contains(block, "12.34") {
		t.Errorf("the provider's reported cost reached the rendered line:\n%s", block)
	}
	if match := spendMoneyRe.FindString(block); match != "" {
		t.Errorf("the cost line shows a monetary amount (%q) — D-01 and Phase 174's D-02 both forbid one:\n%s", match, block)
	}
	for _, symbol := range []string{"$", "£", "€", "¥"} {
		if strings.Contains(block, symbol) {
			t.Errorf("the cost line contains the currency symbol %q:\n%s", symbol, block)
		}
	}

	t.Run("the guard would catch a money figure if one appeared", func(t *testing.T) {
		planted := block + "\nEstimated spend: $12.34\n"
		if spendMoneyRe.FindString(planted) == "" {
			t.Error("the money guard does not recognise a planted currency figure, so it proves nothing")
		}
	})
}

func TestCostLineWithNoRowsShowsNoNumber(t *testing.T) {
	setupSpendTestStore(t)

	block := stripANSI(renderSpendCostLine(196))

	if !strings.Contains(block, "No token use was recorded") {
		t.Errorf("a phase with nothing recorded does not say so plainly; block was:\n%s", block)
	}
	if strings.ContainsAny(block, "0123456789") {
		t.Errorf("a phase with nothing recorded still shows a number, which reads as a measurement:\n%s", block)
	}
	if strings.Contains(block, spendNotReportedFigure) {
		t.Errorf("a phase with nothing recorded renders a worker row; there are no workers to render:\n%s", block)
	}
	for _, jargon := range spendCostLineJargon {
		if strings.Contains(strings.ToLower(block), jargon) {
			t.Errorf("the empty-phase sentence uses the repo-invented word %q:\n%s", jargon, block)
		}
	}
}

func TestCostLineShowsNoTotalWhenNothingWasMeasured(t *testing.T) {
	setupSpendTestStore(t)
	seedSpendLedgerForTest(t, 196, spendWorkflowBuild,
		spendRow{AgentName: "Mason-67", Caste: "builder", Task: "write it", Status: "completed"},
		spendRow{AgentName: "Roam-90", Caste: "scout", Task: "research", Status: "completed"},
	)

	block := stripANSI(renderSpendCostLine(196))

	for _, worker := range []string{"Mason-67", "Roam-90"} {
		if got := costLineFigureCell(t, block, worker); got != spendNotReportedFigure {
			t.Errorf("%s's figure cell = %q, want the dash sentinel", worker, got)
		}
	}
	// The total line must carry no figure of its own: there is nothing to
	// total. The word "Cost:" still opens the block so the reader is told
	// where the answer would have been.
	totalLine := ""
	for _, line := range strings.Split(block, "\n") {
		if strings.HasPrefix(line, "Cost:") {
			totalLine = line
			break
		}
	}
	if totalLine == "" {
		t.Fatalf("no total line at all; block was:\n%s", block)
	}
	for _, magnitude := range []string{"K", "M", "0 tokens"} {
		if strings.Contains(totalLine, magnitude) {
			t.Errorf("the total line shows a figure (%q) when no tool reported one: %q", magnitude, totalLine)
		}
	}
}

func TestCostLineSpansBuildAndCheckRows(t *testing.T) {
	setupSpendTestStore(t)
	seedSpendLedgerForTest(t, 196, spendWorkflowBuild,
		measuredSpendRowForTest("Mason-67", "builder", 1_200_000),
	)
	seedSpendLedgerForTest(t, 196, spendWorkflowContinue,
		measuredSpendRowForTest("Keen-12", "watcher", 220_000),
	)

	block := stripANSI(renderSpendCostLine(196))

	if !strings.Contains(block, "Mason-67") || !strings.Contains(block, "Keen-12") {
		t.Errorf("the line does not span both the build's and the check's workers:\n%s", block)
	}
	if !strings.Contains(block, "1.4M") {
		t.Errorf("the total is not the two files added together (1.4M):\n%s", block)
	}
}

func TestCompactTokenFigureMatchesTheOwnersShape(t *testing.T) {
	// The exact figures from D-01's worked example, plus the boundaries.
	for _, tc := range []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
		{1_000, "1.0K"},
		{1_500, "1.5K"},
		{9_999, "9.9K"},
		{10_000, "10K"},
		{220_000, "220K"},
		{999_999, "999K"},
		{1_000_000, "1.0M"},
		{1_200_000, "1.2M"},
		{1_420_000, "1.4M"},
		{12_300_000, "12.3M"},
	} {
		if got := spendCompactTokenFigure(tc.in); got != tc.want {
			t.Errorf("spendCompactTokenFigure(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestNoResolvedFieldIsPopulatedThenDropped is WR-07's ratchet.
//
// The resolver filled SessionUsage/SessionReported with the orchestrating
// session's own turns and documented them as "kept apart rather than folded into
// any worker's row or silently discarded" — and then its only production
// consumer discarded them. A field that is written and never read is a claim the
// code does not keep.
func TestNoResolvedFieldIsPopulatedThenDropped(t *testing.T) {
	dir := filepath.Dir(goldenTestdataDir())
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}

	written := map[string]string{}
	read := map[string]bool{}
	fields := map[string]bool{}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		// Collect the field names wrapperUsageResolution declares.
		ast.Inspect(file, func(n ast.Node) bool {
			spec, ok := n.(*ast.TypeSpec)
			if !ok || spec.Name.Name != "wrapperUsageResolution" {
				return true
			}
			structType, ok := spec.Type.(*ast.StructType)
			if !ok {
				return true
			}
			for _, f := range structType.Fields.List {
				for _, ident := range f.Names {
					fields[ident.Name] = true
				}
			}
			return true
		})
		// Every selector on a field is a read unless it is an assignment target.
		assigned := map[ast.Node]bool{}
		ast.Inspect(file, func(n ast.Node) bool {
			assign, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}
			for _, lhs := range assign.Lhs {
				assigned[lhs] = true
			}
			return true
		})
		ast.Inspect(file, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if assigned[ast.Node(sel)] {
				written[sel.Sel.Name] = name
				return true
			}
			read[sel.Sel.Name] = true
			return true
		})
	}

	if len(fields) == 0 {
		t.Fatal("wrapperUsageResolution was not found at all; this guard would pass over an empty set")
	}
	for field := range fields {
		if written[field] == "" {
			continue
		}
		if !read[field] {
			t.Errorf("wrapperUsageResolution.%s is filled in %s and read by no production code — either something must report it or it must not be gathered",
				field, written[field])
		}
	}
}

// TestCostBlockSaysWhatItCounts is the owner-facing half of WR-07.
//
// The block was headed "What This Phase Has Cost" while its total excluded the
// largest component of a phase's real spend — the coordinator's own
// back-and-forth, which on this repository's own corpus is 142,581 of 142,833
// usage-bearing lines. The heading has to say what the figures under it measure.
func TestCostBlockSaysWhatItCounts(t *testing.T) {
	ledger := spendLedger{
		Phase:    5,
		Workflow: spendWorkflowBuild,
		Rows: []spendRow{{
			AgentName: "Mason-67", Caste: "builder", Status: "completed",
			Usage: codex.WorkerUsage{InputTokens: 1000, Source: codex.UsageSourceProvider},
		}},
	}
	block := stripANSI(renderSpendCostLineFromLedgers([]spendLedger{ledger}))

	if strings.Contains(block, "What This Phase Has Cost") {
		t.Errorf("the block claims to say what the phase cost while counting only the workers it sent; the coordinator's own use is not in these figures:\n%s", block)
	}
	mentions := false
	for _, phrase := range []string{"coordinator", "main session"} {
		if strings.Contains(strings.ToLower(block), phrase) {
			mentions = true
		}
	}
	if !mentions {
		t.Errorf("nothing in the block tells the owner that the coordinator's own token use is not counted in it:\n%s", block)
	}
}
