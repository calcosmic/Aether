package cmd

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/storage"
)

// TestSpendLedgerPersistsAcrossProcesses covers single-workflow persistence
// and the schema-mismatch bullet: a ledger saved through one store must
// reload intact through a freshly constructed store over the same
// directory -- proving the row survived on disk, not merely in RAM -- and a
// ledger whose schema_version does not match the current constant must be
// skipped rather than partially read.
func TestSpendLedgerPersistsAcrossProcesses(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, _ := newTestStore(t)
	store = s

	ledger := spendLedger{
		Phase:      7,
		PhaseName:  "spend-ledger",
		Workflow:   spendWorkflowBuild,
		RunID:      "run-1",
		RecordedAt: time.Now().UTC().Format(time.RFC3339),
		Rows: []spendRow{
			{
				AgentName: "Mason-67",
				Caste:     "builder",
				Task:      "Implement the ledger",
				Status:    "completed",
				// Anthropic's documented disjoint-column example
				// (pkg/codex/usage.go's own doc comment): 50 input,
				// 100,000 cache read, 2,000 cache creation, 500 output.
				Usage: codex.WorkerUsage{
					InputTokens:         50,
					CachedInputTokens:   100000,
					CacheCreationTokens: 2000,
					OutputTokens:        500,
					Source:              codex.UsageSourceProvider,
				},
			},
		},
	}
	if err := saveSpendLedger(ledger); err != nil {
		t.Fatalf("saveSpendLedger: %v", err)
	}

	// Simulate the process boundary honestly: construct a second store over
	// the same base directory with storage.NewStore rather than reusing the
	// same in-memory handle, which would prove only that a struct still
	// exists in RAM.
	dataDir := s.BasePath()
	second, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("construct second store: %v", err)
	}
	store = second

	loaded, ok := loadSpendLedger(7, spendWorkflowBuild)
	if !ok {
		t.Fatalf("loadSpendLedger returned ok=false")
	}
	if len(loaded.Rows) != 1 {
		t.Fatalf("Rows = %d, want 1", len(loaded.Rows))
	}
	row := loaded.Rows[0]
	if row.Usage.InputTokens != 50 {
		t.Fatalf("InputTokens = %d, want 50", row.Usage.InputTokens)
	}
	if row.Usage.CachedInputTokens != 100000 {
		t.Fatalf("CachedInputTokens = %d, want 100000", row.Usage.CachedInputTokens)
	}
	if row.Usage.CacheCreationTokens != 2000 {
		t.Fatalf("CacheCreationTokens = %d, want 2000", row.Usage.CacheCreationTokens)
	}
	if row.Usage.OutputTokens != 500 {
		t.Fatalf("OutputTokens = %d, want 500", row.Usage.OutputTokens)
	}
	if row.Usage.BilledTotalTokens() != 102550 {
		t.Fatalf("BilledTotalTokens() = %d, want 102550", row.Usage.BilledTotalTokens())
	}
	if row.Usage.Source != codex.UsageSourceProvider {
		t.Fatalf("Source = %q, want %q", row.Usage.Source, codex.UsageSourceProvider)
	}
	if row.Caste != "builder" {
		t.Fatalf("Caste = %q, want builder", row.Caste)
	}

	// Schema-mismatch bullet: a ledger whose schema_version is not the
	// current constant is skipped rather than silently reading fields that
	// may have moved.
	mismatched := ledger
	mismatched.Phase = 8
	mismatched.SchemaVersion = spendLedgerSchemaVersion + 1
	rel := spendLedgerRelForPhaseWorkflow(mismatched.Phase, mismatched.Workflow)
	if err := second.SaveJSON(rel, mismatched); err != nil {
		t.Fatalf("seed schema-mismatched ledger: %v", err)
	}
	if _, ok := loadSpendLedger(8, spendWorkflowBuild); ok {
		t.Fatalf("loadSpendLedger returned ok=true for a schema-mismatched ledger")
	}
	if _, ok := loadSpendLedgersForPhase(8); ok {
		t.Fatalf("loadSpendLedgersForPhase returned ok=true when its only ledger is schema-mismatched")
	}
}

// TestSpendLedgerContinueDoesNotEraseBuild covers the two-workflow bullets:
// a phase runs build then continue as its normal cycle, and the build
// carries most of the tokens. Keying by phase alone would let the continue
// erase the build's rows on every ordinary cycle -- this asserts the build
// file survives, and that loadSpendLedgersForPhase returns both in
// build-then-continue order with the combined row count of both.
func TestSpendLedgerContinueDoesNotEraseBuild(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, _ := newTestStore(t)
	store = s

	build := spendLedger{
		Phase:      7,
		Workflow:   spendWorkflowBuild,
		RecordedAt: time.Now().UTC().Format(time.RFC3339),
		Rows: []spendRow{
			{AgentName: "Mason-1", Caste: "builder", Status: "completed",
				Usage: codex.WorkerUsage{TotalTokens: 30000, Source: codex.UsageSourceProvider}},
			{AgentName: "Watcher-2", Caste: "watcher", Status: "completed",
				Usage: codex.WorkerUsage{TotalTokens: 1000, Source: codex.UsageSourceProvider}},
		},
	}
	if err := saveSpendLedger(build); err != nil {
		t.Fatalf("save build ledger: %v", err)
	}

	cont := spendLedger{
		Phase:      7,
		Workflow:   spendWorkflowContinue,
		RecordedAt: time.Now().UTC().Format(time.RFC3339),
		Rows: []spendRow{
			{AgentName: "Watcher-3", Caste: "watcher", Status: "completed",
				Usage: codex.WorkerUsage{TotalTokens: 4000, Source: codex.UsageSourceProvider}},
		},
	}
	if err := saveSpendLedger(cont); err != nil {
		t.Fatalf("save continue ledger: %v", err)
	}

	reloadedBuild, ok := loadSpendLedger(7, spendWorkflowBuild)
	if !ok {
		t.Fatalf("loadSpendLedger(build) ok=false")
	}
	if len(reloadedBuild.Rows) != 2 {
		t.Fatalf("build Rows = %d, want 2 (continue must not erase build)", len(reloadedBuild.Rows))
	}

	ledgers, ok := loadSpendLedgersForPhase(7)
	if !ok {
		t.Fatalf("loadSpendLedgersForPhase ok=false")
	}
	if len(ledgers) != 2 {
		t.Fatalf("loadSpendLedgersForPhase returned %d ledgers, want 2", len(ledgers))
	}
	if ledgers[0].Workflow != spendWorkflowBuild || ledgers[1].Workflow != spendWorkflowContinue {
		t.Fatalf("ledgers not in build-then-continue order: %q, %q", ledgers[0].Workflow, ledgers[1].Workflow)
	}
	totalRows := len(ledgers[0].Rows) + len(ledgers[1].Rows)
	if totalRows != 3 {
		t.Fatalf("combined row count = %d, want 3", totalRows)
	}
}

// TestSpendLedgerSameWorkflowRerunReplaces covers the replace-within-a-
// workflow rule: saving the same two-row build ledger twice must leave the
// loaded build row count at 2, not 4. This test and
// TestSpendLedgerContinueDoesNotEraseBuild are a pair -- one forbids losing
// rows across workflows, the other forbids doubling them within one, and
// neither is safe to keep without the other.
func TestSpendLedgerSameWorkflowRerunReplaces(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, _ := newTestStore(t)
	store = s

	build := spendLedger{
		Phase:      9,
		Workflow:   spendWorkflowBuild,
		RecordedAt: time.Now().UTC().Format(time.RFC3339),
		Rows: []spendRow{
			{AgentName: "Mason-1", Caste: "builder", Status: "completed",
				Usage: codex.WorkerUsage{TotalTokens: 1000, Source: codex.UsageSourceProvider}},
			{AgentName: "Mason-2", Caste: "builder", Status: "completed",
				Usage: codex.WorkerUsage{TotalTokens: 2000, Source: codex.UsageSourceProvider}},
		},
	}
	if err := saveSpendLedger(build); err != nil {
		t.Fatalf("save first build ledger: %v", err)
	}
	if err := saveSpendLedger(build); err != nil {
		t.Fatalf("save second (rerun) build ledger: %v", err)
	}

	loaded, ok := loadSpendLedger(9, spendWorkflowBuild)
	if !ok {
		t.Fatalf("loadSpendLedger ok=false")
	}
	if len(loaded.Rows) != 2 {
		t.Fatalf("Rows = %d, want 2 (rerun must replace, not double)", len(loaded.Rows))
	}
}

// TestSpendLedgerLoadEdgeCases covers the remaining loadSpendLedgersForPhase
// behavior bullets: a phase with no ledger of either workflow returns
// ok == false with no error-shaped panic, and a phase with only one
// workflow present returns that one ledger with ok == true.
func TestSpendLedgerLoadEdgeCases(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, _ := newTestStore(t)
	store = s

	if _, ok := loadSpendLedgersForPhase(999); ok {
		t.Fatalf("loadSpendLedgersForPhase(999) ok=true for a phase with no ledgers")
	}

	onlyContinue := spendLedger{
		Phase:      12,
		Workflow:   spendWorkflowContinue,
		RecordedAt: time.Now().UTC().Format(time.RFC3339),
		Rows: []spendRow{
			{AgentName: "Watcher-1", Caste: "watcher", Status: "completed",
				Usage: codex.WorkerUsage{TotalTokens: 500, Source: codex.UsageSourceProvider}},
		},
	}
	if err := saveSpendLedger(onlyContinue); err != nil {
		t.Fatalf("save continue-only ledger: %v", err)
	}

	ledgers, ok := loadSpendLedgersForPhase(12)
	if !ok {
		t.Fatalf("loadSpendLedgersForPhase(12) ok=false with one workflow present")
	}
	if len(ledgers) != 1 {
		t.Fatalf("loadSpendLedgersForPhase(12) returned %d ledgers, want 1", len(ledgers))
	}
	if ledgers[0].Workflow != spendWorkflowContinue {
		t.Fatalf("loaded ledger workflow = %q, want continue", ledgers[0].Workflow)
	}
}

// TestLedgerMeasuredEstimatedSubtotalsAreSeparate is the fixed 10,000 /
// 20,000 / 500 case: two provider rows and one estimate row must report
// MeasuredTokens == 30000 and EstimatedTokens == 500 as two separate
// fields -- no field on spendTotals holds 30,500 labelled as measured. A
// session-transcript row counts toward MeasuredTokens and SessionRows, and
// does NOT count toward ProviderRows.
func TestLedgerMeasuredEstimatedSubtotalsAreSeparate(t *testing.T) {
	ledger := spendLedger{
		Phase:    10,
		Workflow: spendWorkflowBuild,
		Rows: []spendRow{
			{AgentName: "Mason-1", Usage: codex.WorkerUsage{TotalTokens: 10000, Source: codex.UsageSourceProvider}},
			{AgentName: "Mason-2", Usage: codex.WorkerUsage{TotalTokens: 20000, Source: codex.UsageSourceProvider}},
			{AgentName: "Mason-3", Usage: codex.WorkerUsage{TotalTokens: 500, Source: codex.UsageSourceEstimate}},
		},
	}
	totals := computeSpendTotals([]spendLedger{ledger})
	if totals.MeasuredTokens != 30000 {
		t.Fatalf("MeasuredTokens = %d, want 30000", totals.MeasuredTokens)
	}
	if totals.EstimatedTokens != 500 {
		t.Fatalf("EstimatedTokens = %d, want 500", totals.EstimatedTokens)
	}
	if totals.GrandTotalTokens != 30500 {
		t.Fatalf("GrandTotalTokens = %d, want 30500", totals.GrandTotalTokens)
	}

	sessionLedger := spendLedger{
		Phase:    10,
		Workflow: spendWorkflowContinue,
		Rows: []spendRow{
			{AgentName: "Watcher-1", Usage: codex.WorkerUsage{TotalTokens: 750, Source: codex.UsageSourceSessionTranscript}},
		},
	}
	sessionTotals := computeSpendTotals([]spendLedger{sessionLedger})
	if sessionTotals.MeasuredTokens != 750 {
		t.Fatalf("session-transcript MeasuredTokens = %d, want 750", sessionTotals.MeasuredTokens)
	}
	if sessionTotals.SessionRows != 1 {
		t.Fatalf("SessionRows = %d, want 1", sessionTotals.SessionRows)
	}
	if sessionTotals.ProviderRows != 0 {
		t.Fatalf("ProviderRows = %d, want 0 for a session-transcript row", sessionTotals.ProviderRows)
	}
}

// TestLedgerGrandTotalEqualsSumOfRows asserts the invariant over varied
// input, in the style of TestQueenOrchestratePreservesSafetyCastes
// (cmd/caste_relevance_test.go): for each table entry,
// computeSpendTotals(set).GrandTotalTokens equals an independently
// accumulated sum of row.Usage.BilledTotalTokens() over every row in the
// set, and equals MeasuredTokens + EstimatedTokens. At least two entries
// carry both workflows.
func TestLedgerGrandTotalEqualsSumOfRows(t *testing.T) {
	tests := []struct {
		name    string
		ledgers []spendLedger
	}{
		{
			name: "single-workflow-all-provider",
			ledgers: []spendLedger{
				{Phase: 1, Workflow: spendWorkflowBuild, Rows: []spendRow{
					{Usage: codex.WorkerUsage{TotalTokens: 1000, Source: codex.UsageSourceProvider}},
					{Usage: codex.WorkerUsage{TotalTokens: 2000, Source: codex.UsageSourceProvider}},
				}},
			},
		},
		{
			name: "single-workflow-mixed-source",
			ledgers: []spendLedger{
				{Phase: 2, Workflow: spendWorkflowBuild, Rows: []spendRow{
					{Usage: codex.WorkerUsage{TotalTokens: 5000, Source: codex.UsageSourceProvider}},
					{Usage: codex.WorkerUsage{TotalTokens: 300, Source: codex.UsageSourceEstimate}},
					{Usage: codex.WorkerUsage{TotalTokens: 800, Source: codex.UsageSourceSessionTranscript}},
				}},
			},
		},
		{
			name: "build-plus-continue-small",
			ledgers: []spendLedger{
				{Phase: 3, Workflow: spendWorkflowBuild, Rows: []spendRow{
					{Usage: codex.WorkerUsage{TotalTokens: 30500, Source: codex.UsageSourceProvider}},
				}},
				{Phase: 3, Workflow: spendWorkflowContinue, Rows: []spendRow{
					{Usage: codex.WorkerUsage{TotalTokens: 4000, Source: codex.UsageSourceProvider}},
				}},
			},
		},
		{
			name: "build-plus-continue-many-parents",
			ledgers: []spendLedger{
				{Phase: 4, Workflow: spendWorkflowBuild, Rows: []spendRow{
					{Usage: codex.WorkerUsage{TotalTokens: 1200, Source: codex.UsageSourceProvider}},
					{Usage: codex.WorkerUsage{TotalTokens: 600, Source: codex.UsageSourceEstimate}},
					{Usage: codex.WorkerUsage{TotalTokens: 150, Source: codex.UsageSourceProvider}},
				}},
				{Phase: 4, Workflow: spendWorkflowContinue, Rows: []spendRow{
					{Usage: codex.WorkerUsage{TotalTokens: 900, Source: codex.UsageSourceSessionTranscript}},
					{Usage: codex.WorkerUsage{TotalTokens: 300, Source: codex.UsageSourceProvider}},
				}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var wantSum int64
			for _, ledger := range tc.ledgers {
				for _, row := range ledger.Rows {
					wantSum += row.Usage.BilledTotalTokens()
				}
			}

			totals := computeSpendTotals(tc.ledgers)
			if totals.GrandTotalTokens != wantSum {
				t.Fatalf("GrandTotalTokens = %d, want %d (independent row sum)", totals.GrandTotalTokens, wantSum)
			}
			if totals.GrandTotalTokens != totals.MeasuredTokens+totals.EstimatedTokens {
				t.Fatalf("GrandTotalTokens (%d) != MeasuredTokens+EstimatedTokens (%d)",
					totals.GrandTotalTokens, totals.MeasuredTokens+totals.EstimatedTokens)
			}
		})
	}
}

// TestLedgerPhaseTotalAddsBuildAndContinue is the literal 30,500 + 4,000 =
// 34,500 case: a phase's cost is its workflows added together, never the
// last one written. The row-count assertion proves the continue-arriving
// worker is counted, not dropped.
func TestLedgerPhaseTotalAddsBuildAndContinue(t *testing.T) {
	build := spendLedger{Phase: 40, Workflow: spendWorkflowBuild, Rows: []spendRow{
		{Usage: codex.WorkerUsage{TotalTokens: 30500, Source: codex.UsageSourceProvider}},
	}}
	cont := spendLedger{Phase: 40, Workflow: spendWorkflowContinue, Rows: []spendRow{
		{Usage: codex.WorkerUsage{TotalTokens: 4000, Source: codex.UsageSourceProvider}},
	}}

	totals := computeSpendTotals([]spendLedger{build, cont})
	if totals.GrandTotalTokens != 34500 {
		t.Fatalf("GrandTotalTokens = %d, want 34500", totals.GrandTotalTokens)
	}

	if totals.MeasuredRows != 2 {
		t.Fatalf("MeasuredRows = %d, want 2 -- the continue-arriving worker must be counted, not dropped", totals.MeasuredRows)
	}
}

// TestLedgerGrandTotalFromRawColumnsWithoutHelper is FIX 1-2 from the Phase
// 196 salvage assessment: the aggregation invariant proved against numbers
// this test worked out itself, not against the function the implementation
// calls.
//
// TestLedgerGrandTotalEqualsSumOfRows above accumulates its "independent"
// expectation with row.Usage.BilledTotalTokens() -- the same helper
// computeSpendTotals uses. If that helper were ever wrong, that test would
// pass while enshrining the error. That is the exact shape of the 186x
// undercount this repository already shipped, in this same subsystem: an
// arithmetic test whose expectation is produced by the arithmetic it is
// checking cannot fail when that arithmetic is wrong.
//
// So every expected number below is a literal, hand-summed here from the
// four raw disjoint columns, and nothing in this function calls
// BilledTotalTokens(), TotalInputTokens() or billedTotal(). That absence is
// itself enforced, structurally, by
// TestRawColumnInvariantDoesNotCallTheBilledTotalHelper.
func TestLedgerGrandTotalFromRawColumnsWithoutHelper(t *testing.T) {
	// Anthropic's documented disjoint-column example. Hand-summed:
	//   50 input + 100,000 cache read + 2,000 cache creation + 500 output
	//   = 102,550.
	// TotalTokens is deliberately left zero on every row so the total can
	// only come from the four columns, never from a figure the fixture
	// pre-computed for the implementation.
	anthropicExample := codex.WorkerUsage{
		InputTokens:         50,
		CachedInputTokens:   100000,
		CacheCreationTokens: 2000,
		OutputTokens:        500,
		Source:              codex.UsageSourceProvider,
	}

	t.Run("one provider row totals the four raw columns", func(t *testing.T) {
		totals := computeSpendTotals([]spendLedger{
			{Phase: 50, Workflow: spendWorkflowBuild, Rows: []spendRow{
				{AgentName: "Mason-67", Status: "completed", Usage: anthropicExample},
			}},
		})
		if totals.GrandTotalTokens != 102550 {
			t.Fatalf("GrandTotalTokens = %d, want 102550 (50 + 100000 + 2000 + 500, summed by hand)", totals.GrandTotalTokens)
		}
		if totals.MeasuredTokens != 102550 {
			t.Fatalf("MeasuredTokens = %d, want 102550", totals.MeasuredTokens)
		}
		if totals.EstimatedTokens != 0 {
			t.Fatalf("EstimatedTokens = %d, want 0", totals.EstimatedTokens)
		}
	})

	t.Run("across both workflows the phase total is the hand-summed set", func(t *testing.T) {
		ledgers := []spendLedger{
			{Phase: 51, Workflow: spendWorkflowBuild, Rows: []spendRow{
				// 102,550 (above).
				{AgentName: "Mason-67", Status: "completed", Usage: anthropicExample},
				// 1,200 + 0 + 0 + 300 = 1,500.
				{AgentName: "Keen-12", Status: "completed", Usage: codex.WorkerUsage{
					InputTokens:  1200,
					OutputTokens: 300,
					Source:       codex.UsageSourceSessionTranscript,
				}},
			}},
			{Phase: 51, Workflow: spendWorkflowContinue, Rows: []spendRow{
				// 4,000 + 0 + 0 + 0 = 4,000, and it is an estimate, so it
				// must land in its own subtotal and never in the measured one.
				{AgentName: "Roam-90", Status: "completed", Usage: codex.WorkerUsage{
					InputTokens: 4000,
					Source:      codex.UsageSourceEstimate,
				}},
			}},
		}

		totals := computeSpendTotals(ledgers)

		// 102,550 + 1,500 = 104,050, worked out here, not by the code.
		if totals.MeasuredTokens != 104050 {
			t.Fatalf("MeasuredTokens = %d, want 104050 (102550 + 1500)", totals.MeasuredTokens)
		}
		if totals.EstimatedTokens != 4000 {
			t.Fatalf("EstimatedTokens = %d, want 4000", totals.EstimatedTokens)
		}
		// 104,050 + 4,000 = 108,050.
		if totals.GrandTotalTokens != 108050 {
			t.Fatalf("GrandTotalTokens = %d, want 108050 (104050 + 4000)", totals.GrandTotalTokens)
		}
		if totals.MeasuredRows != 2 {
			t.Fatalf("MeasuredRows = %d, want 2", totals.MeasuredRows)
		}
		if totals.EstimatedRows != 1 {
			t.Fatalf("EstimatedRows = %d, want 1", totals.EstimatedRows)
		}
	})
}

// TestRawColumnInvariantDoesNotCallTheBilledTotalHelper makes the
// independence of the test above executable rather than aspirational.
//
// CLAUDE.md's Definition of Done: a requirement is satisfied only when a
// command exists that fails when it is unmet. "The invariant test must not
// call the helper it is checking" is a requirement, so this parses the test
// file's syntax tree and fails if any call inside
// TestLedgerGrandTotalFromRawColumnsWithoutHelper resolves to one of the
// token-summing helpers. It walks the AST rather than searching text, so a
// comment mentioning the helper (there are several above) can neither
// satisfy nor break it.
func TestRawColumnInvariantDoesNotCallTheBilledTotalHelper(t *testing.T) {
	const guardedFunc = "TestLedgerGrandTotalFromRawColumnsWithoutHelper"
	forbidden := map[string]bool{
		"BilledTotalTokens": true,
		"TotalInputTokens":  true,
		"billedTotal":       true,
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "spend_ledger_test.go", nil, 0)
	if err != nil {
		t.Fatalf("parse spend_ledger_test.go: %v", err)
	}

	var found bool
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Name.Name != guardedFunc {
			continue
		}
		found = true
		ast.Inspect(fn, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fun := call.Fun.(type) {
			case *ast.SelectorExpr:
				if fun.Sel != nil && forbidden[fun.Sel.Name] {
					t.Errorf("%s calls %s() -- its expected totals must be hand-computed literals, "+
						"never produced by the arithmetic it is checking (the 186x-undercount shape)",
						guardedFunc, fun.Sel.Name)
				}
			case *ast.Ident:
				if forbidden[fun.Name] {
					t.Errorf("%s calls %s() -- its expected totals must be hand-computed literals",
						guardedFunc, fun.Name)
				}
			}
			return true
		})
	}
	if !found {
		t.Fatalf("%s not found in spend_ledger_test.go -- the independent aggregation invariant was removed", guardedFunc)
	}
}

// spendCurrencyWords are the whole words that mark a field as carrying a
// money amount. Matching whole words rather than substrings keeps
// "RecordedAt" and "WorkerCount" from tripping a naive "cent"/"count"
// search.
var spendCurrencyWords = map[string]bool{
	"usd": true, "dollar": true, "dollars": true, "currency": true,
	"price": true, "prices": true, "pricing": true, "cost": true,
	"costs": true, "money": true, "cent": true, "cents": true,
	"fee": true, "fees": true, "charge": true, "charges": true,
	"billing": true, "paid": true, "payment": true, "rate": true,
	"rates": true,
}

// spendNameCarriesCurrency splits an identifier or JSON key into its words
// (camelCase and snake_case both) and reports whether any word names money.
func spendNameCarriesCurrency(name string) bool {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "usd") || strings.Contains(lower, "dollar") || strings.Contains(name, "$") {
		return true
	}
	var words []string
	var current strings.Builder
	for i, r := range name {
		switch {
		case r == '_' || r == '-' || r == '.':
			words = append(words, current.String())
			current.Reset()
		case unicode.IsUpper(r) && i > 0:
			words = append(words, current.String())
			current.Reset()
			current.WriteRune(unicode.ToLower(r))
		default:
			current.WriteRune(unicode.ToLower(r))
		}
	}
	words = append(words, current.String())
	for _, word := range words {
		if spendCurrencyWords[word] {
			return true
		}
	}
	return false
}

// TestSpendLedgerCarriesNoCurrencyField is FIX 1-1 and decision D-01 made
// executable: no field on the ledger or its totals may relay a money amount,
// so no later renderer can reach one and put it on the headline line.
//
// The salvaged branch carried spendTotals.ProviderUSD and ProviderUSDRows.
// Neither computed a price from a rate -- both relayed the provider's own
// reported figure -- but nothing read them either, and an unread money field
// sitting on the totals struct is a standing invitation to render it.
//
// Two assertions, deliberately: the Go types by reflection (so a field added
// with no JSON tag is still caught) and the marshalled JSON keys (so a field
// renamed in Go but still serialized as money is caught). Neither searches
// the source text, so a comment can never satisfy or break this test.
//
// codex.WorkerUsage is explicitly out of scope and skipped by name. It is the
// shared provider-usage type on pkg/codex, it keeps its own USDCost field for
// callers outside this phase, and this plan is forbidden from changing it.
// The boundary this test defends is the ledger's own types -- above all
// spendTotals, which is what a headline renderer reads.
func TestSpendLedgerCarriesNoCurrencyField(t *testing.T) {
	usageType := reflect.TypeOf(codex.WorkerUsage{})

	var walkType func(t *testing.T, typ reflect.Type, path string, seen map[reflect.Type]bool)
	walkType = func(t *testing.T, typ reflect.Type, path string, seen map[reflect.Type]bool) {
		for typ.Kind() == reflect.Ptr || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array {
			typ = typ.Elem()
		}
		if typ.Kind() != reflect.Struct || typ == usageType || seen[typ] {
			return
		}
		seen[typ] = true
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			where := path + "." + field.Name
			if spendNameCarriesCurrency(field.Name) {
				t.Errorf("field %s names a money amount; the ledger and its totals must carry none (D-01: no currency figure may reach the cost line)", where)
			}
			tag := strings.Split(field.Tag.Get("json"), ",")[0]
			if tag != "" && spendNameCarriesCurrency(tag) {
				t.Errorf("field %s serializes as %q, which names a money amount", where, tag)
			}
			walkType(t, field.Type, where, seen)
		}
	}

	walkType(t, reflect.TypeOf(spendTotals{}), "spendTotals", map[reflect.Type]bool{})
	walkType(t, reflect.TypeOf(spendLedger{}), "spendLedger", map[reflect.Type]bool{})
	walkType(t, reflect.TypeOf(spendRow{}), "spendRow", map[reflect.Type]bool{})

	// The serialized form, on a fully populated value so no omitempty tag can
	// hide a key from this walk.
	var walkJSON func(t *testing.T, value interface{}, path string)
	walkJSON = func(t *testing.T, value interface{}, path string) {
		object, ok := value.(map[string]interface{})
		if !ok {
			if list, isList := value.([]interface{}); isList {
				for i, item := range list {
					walkJSON(t, item, fmt.Sprintf("%s[%d]", path, i))
				}
			}
			return
		}
		for key, nested := range object {
			// The row's usage object is codex.WorkerUsage, out of scope above
			// and out of scope here for the same reason.
			if key == "usage" {
				continue
			}
			if spendNameCarriesCurrency(key) {
				t.Errorf("serialized key %s.%s names a money amount", path, key)
			}
			walkJSON(t, nested, path+"."+key)
		}
	}

	totals := computeSpendTotals([]spendLedger{
		{Phase: 60, Workflow: spendWorkflowBuild, Rows: []spendRow{
			{AgentName: "Mason-67", Status: "completed", Usage: codex.WorkerUsage{
				InputTokens: 1000, OutputTokens: 200, USDCost: 12.34, Source: codex.UsageSourceProvider,
			}},
			{AgentName: "Roam-90", Status: "completed", Usage: codex.WorkerUsage{
				InputTokens: 500, Source: codex.UsageSourceEstimate,
			}},
			{AgentName: "Keen-12", Status: "completed", Usage: codex.WorkerUsage{
				InputTokens: 700, Source: codex.UsageSourceSessionTranscript,
			}},
		}},
	})

	for _, subject := range []struct {
		name  string
		value interface{}
	}{
		{"spendTotals", totals},
		{"spendLedger", spendLedger{
			SchemaVersion: spendLedgerSchemaVersion,
			Phase:         60,
			PhaseName:     "see-what-it-cost",
			Workflow:      spendWorkflowBuild,
			RunID:         "run-1",
			RecordedAt:    "2026-08-27T00:00:00Z",
			Rows: []spendRow{{
				AgentName: "Mason-67", Caste: "builder",
				Task: "Land the ledger", Status: "completed",
				Usage: codex.WorkerUsage{InputTokens: 50, USDCost: 9.99, Source: codex.UsageSourceProvider},
			}},
		}},
	} {
		raw, err := json.Marshal(subject.value)
		if err != nil {
			t.Fatalf("marshal %s: %v", subject.name, err)
		}
		var decoded interface{}
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("unmarshal %s: %v", subject.name, err)
		}
		walkJSON(t, decoded, subject.name)
	}
}

// TestSpendRowCarriesJobName is FIX 1-3: Phase 195 made a single worker able
// to own a grouped chain of tasks (codexBuildDispatch.JobName /
// CoveredTaskIDs), and a ledger keyed only by worker cannot say which grouped
// job a worker's tokens belong to. The row records the job name, and it
// survives the disk round trip like every other column.
func TestSpendRowCarriesJobName(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, _ := newTestStore(t)
	store = s

	// The schema version must still be the salvaged value. JobName was added
	// before this ledger was ever written to in production precisely so no
	// version bump and no migration is owed for a one-line field. If this
	// assertion ever has to change, the field was added too late.
	if spendLedgerSchemaVersion != 1 {
		t.Fatalf("spendLedgerSchemaVersion = %d, want 1 -- JobName was added before the first production write, so no bump is owed",
			spendLedgerSchemaVersion)
	}

	ledger := spendLedger{
		Phase:      70,
		Workflow:   spendWorkflowBuild,
		RecordedAt: "2026-08-27T00:00:00Z",
		Rows: []spendRow{
			{
				AgentName: "Mason-67",
				Caste:     "builder",
				Task:      "Wire the coherent jobs into build planning",
				JobName:   "coherent-jobs",
				Status:    "completed",
				Usage:     codex.WorkerUsage{TotalTokens: 1200, Source: codex.UsageSourceProvider},
			},
		},
	}
	if err := saveSpendLedger(ledger); err != nil {
		t.Fatalf("saveSpendLedger: %v", err)
	}

	loaded, ok := loadSpendLedger(70, spendWorkflowBuild)
	if !ok {
		t.Fatalf("loadSpendLedger ok=false")
	}
	if len(loaded.Rows) != 1 {
		t.Fatalf("Rows = %d, want 1", len(loaded.Rows))
	}
	if loaded.Rows[0].JobName != "coherent-jobs" {
		t.Fatalf("JobName = %q, want %q -- a grouped job's attribution must survive the round trip",
			loaded.Rows[0].JobName, "coherent-jobs")
	}

	// And it is really on disk under its own key, not reconstructed in RAM.
	raw, err := os.ReadFile(filepath.Join(s.BasePath(), "spend", "phase-70-build.json"))
	if err != nil {
		t.Fatalf("read ledger file: %v", err)
	}
	var onDisk struct {
		Rows []map[string]json.RawMessage `json:"rows"`
	}
	if err := json.Unmarshal(raw, &onDisk); err != nil {
		t.Fatalf("unmarshal ledger file: %v", err)
	}
	if len(onDisk.Rows) != 1 {
		t.Fatalf("on-disk rows = %d, want 1", len(onDisk.Rows))
	}
	if _, present := onDisk.Rows[0]["job_name"]; !present {
		t.Fatalf("serialized row has no job_name key: %s", raw)
	}
}

// TestSpendRowWithoutJobNameOmitsIt is the other half of FIX 1-3: a worker
// that owned one ungrouped task records no job name at all rather than a
// placeholder, and the key is absent from the serialized form entirely.
//
// The assertion is made on the marshalled bytes rather than on the struct
// value, because a struct value would still read as the empty string if the
// omitempty tag silently stopped working.
func TestSpendRowWithoutJobNameOmitsIt(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, _ := newTestStore(t)
	store = s

	ungrouped := spendRow{
		AgentName: "Keen-12",
		Caste:     "watcher",
		Task:      "Check the build",
		Status:    "completed",
		Usage:     codex.WorkerUsage{TotalTokens: 220, Source: codex.UsageSourceProvider},
	}

	marshalled, err := json.Marshal(ungrouped)
	if err != nil {
		t.Fatalf("marshal row: %v", err)
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(marshalled, &keys); err != nil {
		t.Fatalf("unmarshal row: %v", err)
	}
	if _, present := keys["job_name"]; present {
		t.Fatalf("an ungrouped row serialized a job_name key: %s", marshalled)
	}

	if err := saveSpendLedger(spendLedger{
		Phase:      71,
		Workflow:   spendWorkflowContinue,
		RecordedAt: "2026-08-27T00:00:00Z",
		Rows:       []spendRow{ungrouped},
	}); err != nil {
		t.Fatalf("saveSpendLedger: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(s.BasePath(), "spend", "phase-71-continue.json"))
	if err != nil {
		t.Fatalf("read ledger file: %v", err)
	}
	var onDisk struct {
		Rows []map[string]json.RawMessage `json:"rows"`
	}
	if err := json.Unmarshal(raw, &onDisk); err != nil {
		t.Fatalf("unmarshal ledger file: %v", err)
	}
	if _, present := onDisk.Rows[0]["job_name"]; present {
		t.Fatalf("an ungrouped row wrote a job_name key to disk: %s", raw)
	}
}

// TestSpendLedgerRefusesUnknownRowStatus is FIX 1-4: the row's status is the
// dispatch status, drawn from the vocabulary the rest of the runtime already
// normalizes (dispatchStatusIcon / normalizeRuntimeDispatchStatus), not a
// second free-form one. This repository's documented failure mode is two
// vocabularies for one idea, so an unknown word is refused by name on save
// and nothing is written.
func TestSpendLedgerRefusesUnknownRowStatus(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, _ := newTestStore(t)
	store = s

	t.Run("an unknown status is refused by name and writes no file", func(t *testing.T) {
		err := saveSpendLedger(spendLedger{
			Phase:      80,
			Workflow:   spendWorkflowBuild,
			RecordedAt: "2026-08-27T00:00:00Z",
			Rows: []spendRow{
				{AgentName: "Mason-67", Caste: "builder", Status: "completed",
					Usage: codex.WorkerUsage{TotalTokens: 100, Source: codex.UsageSourceProvider}},
				{AgentName: "Roam-90", Caste: "scout", Status: "mostly-finished-ish",
					Usage: codex.WorkerUsage{TotalTokens: 200, Source: codex.UsageSourceProvider}},
			},
		})
		if err == nil {
			t.Fatalf("expected saveSpendLedger to refuse an unknown row status")
		}
		if !strings.Contains(err.Error(), "mostly-finished-ish") {
			t.Fatalf("refusal does not name the offending status: %v", err)
		}
		if !strings.Contains(err.Error(), "Roam-90") {
			t.Fatalf("refusal does not name the worker: %v", err)
		}
		if _, statErr := os.Stat(filepath.Join(s.BasePath(), "spend", "phase-80-build.json")); !os.IsNotExist(statErr) {
			t.Fatalf("a refused ledger was written to disk anyway (stat err = %v)", statErr)
		}
		if _, ok := loadSpendLedger(80, spendWorkflowBuild); ok {
			t.Fatalf("a refused ledger loaded back")
		}
	})

	t.Run("an empty status is refused rather than silently read as failed", func(t *testing.T) {
		err := saveSpendLedger(spendLedger{
			Phase:      81,
			Workflow:   spendWorkflowBuild,
			RecordedAt: "2026-08-27T00:00:00Z",
			Rows: []spendRow{
				{AgentName: "Keen-12", Caste: "watcher",
					Usage: codex.WorkerUsage{TotalTokens: 100, Source: codex.UsageSourceProvider}},
			},
		})
		if err == nil {
			t.Fatalf("expected saveSpendLedger to refuse a row with no status")
		}
		if !strings.Contains(err.Error(), "Keen-12") {
			t.Fatalf("refusal does not name the worker: %v", err)
		}
		if _, statErr := os.Stat(filepath.Join(s.BasePath(), "spend", "phase-81-build.json")); !os.IsNotExist(statErr) {
			t.Fatalf("a refused ledger was written to disk anyway (stat err = %v)", statErr)
		}
	})

	t.Run("every accepted status stores as one canonical word", func(t *testing.T) {
		// "success" and "passed" are the runtime's own synonyms for a finished
		// worker. They are accepted and folded, so the file never carries two
		// words for one idea.
		if err := saveSpendLedger(spendLedger{
			Phase:      82,
			Workflow:   spendWorkflowBuild,
			RecordedAt: "2026-08-27T00:00:00Z",
			Rows: []spendRow{
				{AgentName: "Mason-1", Status: "success",
					Usage: codex.WorkerUsage{TotalTokens: 10, Source: codex.UsageSourceProvider}},
				{AgentName: "Mason-2", Status: "PASSED",
					Usage: codex.WorkerUsage{TotalTokens: 20, Source: codex.UsageSourceProvider}},
				{AgentName: "Mason-3", Status: "timed_out",
					Usage: codex.WorkerUsage{TotalTokens: 30, Source: codex.UsageSourceProvider}},
			},
		}); err != nil {
			t.Fatalf("saveSpendLedger refused a known synonym: %v", err)
		}
		loaded, ok := loadSpendLedger(82, spendWorkflowBuild)
		if !ok {
			t.Fatalf("loadSpendLedger ok=false")
		}
		want := []string{"completed", "completed", "timeout"}
		for i, row := range loaded.Rows {
			if row.Status != want[i] {
				t.Fatalf("row %d status = %q, want %q", i, row.Status, want[i])
			}
		}
	})

	t.Run("validating a save does not mutate the caller's rows", func(t *testing.T) {
		rows := []spendRow{
			{AgentName: "Mason-9", Status: "success",
				Usage: codex.WorkerUsage{TotalTokens: 10, Source: codex.UsageSourceProvider}},
		}
		if err := saveSpendLedger(spendLedger{
			Phase: 83, Workflow: spendWorkflowBuild, RecordedAt: "2026-08-27T00:00:00Z", Rows: rows,
		}); err != nil {
			t.Fatalf("saveSpendLedger: %v", err)
		}
		if rows[0].Status != "success" {
			t.Fatalf("saveSpendLedger rewrote the caller's row status to %q", rows[0].Status)
		}
	})
}
