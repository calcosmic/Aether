package cmd

import (
	"strings"
	"testing"
	"time"

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
				AgentName:  "Mason-67",
				Caste:      "builder",
				ParentName: "Queen",
				Task:       "Implement the ledger",
				Status:     "completed",
				ToolCount:  4,
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
	if row.ParentName != "Queen" {
		t.Fatalf("ParentName = %q, want Queen", row.ParentName)
	}
	if row.ToolCount != 4 {
		t.Fatalf("ToolCount = %d, want 4", row.ToolCount)
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
// set, equals MeasuredTokens + EstimatedTokens, and the spendRollupByParent
// totals sum to the same figure. At least two entries carry both
// workflows.
func TestLedgerGrandTotalEqualsSumOfRows(t *testing.T) {
	tests := []struct {
		name    string
		ledgers []spendLedger
	}{
		{
			name: "single-workflow-all-provider",
			ledgers: []spendLedger{
				{Phase: 1, Workflow: spendWorkflowBuild, Rows: []spendRow{
					{ParentName: "Queen", Usage: codex.WorkerUsage{TotalTokens: 1000, Source: codex.UsageSourceProvider}},
					{ParentName: "Queen", Usage: codex.WorkerUsage{TotalTokens: 2000, Source: codex.UsageSourceProvider}},
				}},
			},
		},
		{
			name: "single-workflow-mixed-source",
			ledgers: []spendLedger{
				{Phase: 2, Workflow: spendWorkflowBuild, Rows: []spendRow{
					{ParentName: "Queen", Usage: codex.WorkerUsage{TotalTokens: 5000, Source: codex.UsageSourceProvider}},
					{ParentName: "Queen", Usage: codex.WorkerUsage{TotalTokens: 300, Source: codex.UsageSourceEstimate}},
					{ParentName: "Scout-1", Usage: codex.WorkerUsage{TotalTokens: 800, Source: codex.UsageSourceSessionTranscript}},
				}},
			},
		},
		{
			name: "build-plus-continue-small",
			ledgers: []spendLedger{
				{Phase: 3, Workflow: spendWorkflowBuild, Rows: []spendRow{
					{ParentName: "Queen", Usage: codex.WorkerUsage{TotalTokens: 30500, Source: codex.UsageSourceProvider}},
				}},
				{Phase: 3, Workflow: spendWorkflowContinue, Rows: []spendRow{
					{ParentName: "Queen", Usage: codex.WorkerUsage{TotalTokens: 4000, Source: codex.UsageSourceProvider}},
				}},
			},
		},
		{
			name: "build-plus-continue-many-parents",
			ledgers: []spendLedger{
				{Phase: 4, Workflow: spendWorkflowBuild, Rows: []spendRow{
					{ParentName: "Queen", Usage: codex.WorkerUsage{TotalTokens: 1200, Source: codex.UsageSourceProvider}},
					{ParentName: "Mason-1", Usage: codex.WorkerUsage{TotalTokens: 600, Source: codex.UsageSourceEstimate}},
					{ParentName: "", Usage: codex.WorkerUsage{TotalTokens: 150, Source: codex.UsageSourceProvider}},
				}},
				{Phase: 4, Workflow: spendWorkflowContinue, Rows: []spendRow{
					{ParentName: "Queen", Usage: codex.WorkerUsage{TotalTokens: 900, Source: codex.UsageSourceSessionTranscript}},
					{ParentName: "Watcher-1", Usage: codex.WorkerUsage{TotalTokens: 300, Source: codex.UsageSourceProvider}},
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

			var rollupSum int64
			for _, r := range spendRollupByParent(tc.ledgers) {
				rollupSum += r.TotalTokens
			}
			if rollupSum != totals.GrandTotalTokens {
				t.Fatalf("parent roll-up sum = %d, want %d", rollupSum, totals.GrandTotalTokens)
			}
		})
	}
}

// TestLedgerRefusesDerivedMetricOverEstimates covers
// spendPerWorkerAverageTokens's refusal: a mixed set with includeEstimates
// false returns an error naming the estimate count and --include-estimates;
// with true it returns the grand total divided by the row count without
// error; over an all-measured set it returns a value and no error even
// when includeEstimates is false.
func TestLedgerRefusesDerivedMetricOverEstimates(t *testing.T) {
	ledgers := []spendLedger{
		{Phase: 20, Workflow: spendWorkflowBuild, Rows: []spendRow{
			{Usage: codex.WorkerUsage{TotalTokens: 1000, Source: codex.UsageSourceProvider}},
			{Usage: codex.WorkerUsage{TotalTokens: 500, Source: codex.UsageSourceEstimate}},
		}},
	}

	_, err := spendPerWorkerAverageTokens(ledgers, false)
	if err == nil {
		t.Fatalf("expected an error when estimates are present and includeEstimates is false")
	}
	if !strings.Contains(err.Error(), "1 of 2") {
		t.Fatalf("error does not name the estimate/row counts (want \"1 of 2\"): %v", err)
	}
	if !strings.Contains(err.Error(), "--include-estimates") {
		t.Fatalf("error does not name --include-estimates: %v", err)
	}

	avg, err := spendPerWorkerAverageTokens(ledgers, true)
	if err != nil {
		t.Fatalf("spendPerWorkerAverageTokens(true) returned error: %v", err)
	}
	if avg != 750 {
		t.Fatalf("avg = %v, want 750", avg)
	}

	allMeasured := []spendLedger{
		{Phase: 21, Workflow: spendWorkflowBuild, Rows: []spendRow{
			{Usage: codex.WorkerUsage{TotalTokens: 1000, Source: codex.UsageSourceProvider}},
			{Usage: codex.WorkerUsage{TotalTokens: 2000, Source: codex.UsageSourceProvider}},
		}},
	}
	avg2, err := spendPerWorkerAverageTokens(allMeasured, false)
	if err != nil {
		t.Fatalf("spendPerWorkerAverageTokens over an all-measured set returned error: %v", err)
	}
	if avg2 != 1500 {
		t.Fatalf("avg2 = %v, want 1500", avg2)
	}
}

// TestLedgerParentRollupCountsEachWorkerOnce covers spendRollupByParent:
// three rows sharing the parent "Queen" (arriving from both workflows)
// return a single "Queen" entry totalling those three rows, an
// empty-parent row groups under "(unattributed)", and the sum across all
// parent entries equals GrandTotalTokens -- no worker counted twice, none
// dropped for arriving from the other workflow.
func TestLedgerParentRollupCountsEachWorkerOnce(t *testing.T) {
	ledgers := []spendLedger{
		{Phase: 30, Workflow: spendWorkflowBuild, Rows: []spendRow{
			{ParentName: "Queen", Usage: codex.WorkerUsage{TotalTokens: 1000, Source: codex.UsageSourceProvider}},
			{ParentName: "Queen", Usage: codex.WorkerUsage{TotalTokens: 2000, Source: codex.UsageSourceProvider}},
		}},
		{Phase: 30, Workflow: spendWorkflowContinue, Rows: []spendRow{
			{ParentName: "Queen", Usage: codex.WorkerUsage{TotalTokens: 500, Source: codex.UsageSourceProvider}},
			{ParentName: "", Usage: codex.WorkerUsage{TotalTokens: 250, Source: codex.UsageSourceEstimate}},
		}},
	}

	rollups := spendRollupByParent(ledgers)

	var queen, unattributed *spendParentRollup
	for i := range rollups {
		switch rollups[i].Parent {
		case "Queen":
			queen = &rollups[i]
		case "(unattributed)":
			unattributed = &rollups[i]
		}
	}
	if queen == nil {
		t.Fatalf("expected a Queen roll-up entry")
	}
	if queen.WorkerCount != 3 {
		t.Fatalf("Queen WorkerCount = %d, want 3", queen.WorkerCount)
	}
	if queen.TotalTokens != 3500 {
		t.Fatalf("Queen TotalTokens = %d, want 3500", queen.TotalTokens)
	}
	if unattributed == nil {
		t.Fatalf("expected an (unattributed) roll-up entry for the empty-parent row")
	}
	if unattributed.WorkerCount != 1 || unattributed.TotalTokens != 250 {
		t.Fatalf("(unattributed) entry = %+v, want WorkerCount=1 TotalTokens=250", *unattributed)
	}

	var sum int64
	for _, r := range rollups {
		sum += r.TotalTokens
	}
	total := computeSpendTotals(ledgers)
	if sum != total.GrandTotalTokens {
		t.Fatalf("roll-up sum = %d, want %d (GrandTotalTokens)", sum, total.GrandTotalTokens)
	}
}

// TestLedgerPhaseTotalAddsBuildAndContinue is the literal 30,500 + 4,000 =
// 34,500 case: a phase's cost is its workflows added together, never the
// last one written. The parent roll-up assertion proves the
// continue-arriving worker is counted, not dropped.
func TestLedgerPhaseTotalAddsBuildAndContinue(t *testing.T) {
	build := spendLedger{Phase: 40, Workflow: spendWorkflowBuild, Rows: []spendRow{
		{ParentName: "Queen", Usage: codex.WorkerUsage{TotalTokens: 30500, Source: codex.UsageSourceProvider}},
	}}
	cont := spendLedger{Phase: 40, Workflow: spendWorkflowContinue, Rows: []spendRow{
		{ParentName: "Queen", Usage: codex.WorkerUsage{TotalTokens: 4000, Source: codex.UsageSourceProvider}},
	}}

	totals := computeSpendTotals([]spendLedger{build, cont})
	if totals.GrandTotalTokens != 34500 {
		t.Fatalf("GrandTotalTokens = %d, want 34500", totals.GrandTotalTokens)
	}

	rollups := spendRollupByParent([]spendLedger{build, cont})
	if len(rollups) != 1 {
		t.Fatalf("expected a single Queen roll-up entry, got %d", len(rollups))
	}
	if rollups[0].WorkerCount != 2 {
		t.Fatalf("WorkerCount = %d, want 2 -- the continue-arriving worker must be counted, not dropped", rollups[0].WorkerCount)
	}
	if rollups[0].TotalTokens != 34500 {
		t.Fatalf("TotalTokens = %d, want 34500", rollups[0].TotalTokens)
	}
}
