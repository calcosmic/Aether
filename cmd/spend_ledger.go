package cmd

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/calcosmic/Aether/pkg/codex"
)

// spendLedgerSchemaVersion guards loadSpendLedger against reading a ledger
// shaped by a future, incompatible version of this file. A mismatch is
// treated the same as an absent file -- skipped, never partially read.
const spendLedgerSchemaVersion = 1

// The two workflows a phase runs, in the fixed order every aggregate and
// every rendered report uses -- so output is deterministic.
const (
	spendWorkflowBuild    = "build"
	spendWorkflowContinue = "continue"
)

// spendLedgerWorkflows returns the two workflows in their fixed order.
func spendLedgerWorkflows() []string {
	return []string{spendWorkflowBuild, spendWorkflowContinue}
}

// spendLedgerRelForPhaseWorkflow returns the store-relative path for a
// phase's ledger of one workflow: spend/phase-<N>-<workflow>.json. It
// returns the empty string for any workflow outside spendLedgerWorkflows(),
// so a typo can never silently create a third ledger nothing aggregates.
//
// The file is keyed by phase AND workflow, and whole-document replace
// applies WITHIN a workflow only. This is the decision that was got wrong
// first, so it is documented in full here.
//
// A re-run of build-finalize against the same phase must replace the build
// rows, never append them -- that is the no-double-count invariant SPEND-05
// exists to hold. But a phase routinely runs build and then continue
// (CLAUDE.md's Typical Workflow), and those are two different sets of
// workers doing two different jobs. Keying by phase alone would let the
// continue erase the build's rows on every normal cycle, which is where the
// majority of a phase's tokens live -- SPEND-01's "survives the run" would
// fail against the very next operation, and SPEND-08's headline line would
// understate real cost every time. Two files, each replaced by its own
// workflow, satisfy both rules at once. Retention still needs no pruning:
// the file count is bounded by two per planned phase.
func spendLedgerRelForPhaseWorkflow(phase int, workflow string) string {
	valid := false
	for _, w := range spendLedgerWorkflows() {
		if w == workflow {
			valid = true
			break
		}
	}
	if !valid {
		return ""
	}
	return filepath.ToSlash(filepath.Join("spend", fmt.Sprintf("phase-%d-%s.json", phase, workflow)))
}

// spendRow is one worker dispatch's entry in a phase's ledger. Field names
// deliberately reuse agent.SpawnEntry's vocabulary (AgentName/Caste/
// ParentName/Task/Status) so a row joins to a spawn-tree entry by name with
// no translation layer. Usage is never omitempty -- a dispatch with a
// zero-value usage must still be visible as a row; a dispatch never
// vanishes from the ledger.
type spendRow struct {
	AgentName  string            `json:"name"`
	Caste      string            `json:"caste"`
	ParentName string            `json:"parent"`
	Task       string            `json:"task"`
	Status     string            `json:"status"`
	ToolCount  int               `json:"tool_count,omitempty"`
	Usage      codex.WorkerUsage `json:"usage"`
}

// spendLedger is one workflow's durable ledger for one phase.
type spendLedger struct {
	SchemaVersion int        `json:"schema_version"`
	Phase         int        `json:"phase"`
	PhaseName     string     `json:"phase_name,omitempty"`
	Workflow      string     `json:"workflow"`
	RunID         string     `json:"run_id,omitempty"`
	RecordedAt    string     `json:"recorded_at"`
	Rows          []spendRow `json:"rows"`
}

// saveSpendLedger writes entry's workflow ledger through pkg/storage.Store.
// Writing one workflow must never read, rewrite or delete the other
// workflow's file -- store.SaveJSON at spendLedgerRelForPhaseWorkflow
// touches only entry.Workflow's own path.
func saveSpendLedger(entry spendLedger) error {
	if store == nil {
		return fmt.Errorf("save spend ledger: store is not initialized")
	}
	rel := spendLedgerRelForPhaseWorkflow(entry.Phase, entry.Workflow)
	if rel == "" {
		return fmt.Errorf("save spend ledger: workflow %q is not one of %v", entry.Workflow, spendLedgerWorkflows())
	}
	entry.SchemaVersion = spendLedgerSchemaVersion
	return store.SaveJSON(rel, entry)
}

// loadSpendLedger is the single-workflow read. ok is false when the store
// is nil, the workflow is unknown, the file is absent, the JSON is
// malformed, or SchemaVersion differs from the current constant. A
// schema-mismatched or corrupt ledger is never returned partially
// populated alongside ok == true.
func loadSpendLedger(phase int, workflow string) (spendLedger, bool) {
	if store == nil {
		return spendLedger{}, false
	}
	rel := spendLedgerRelForPhaseWorkflow(phase, workflow)
	if rel == "" {
		return spendLedger{}, false
	}
	var ledger spendLedger
	if err := store.LoadJSON(rel, &ledger); err != nil {
		return spendLedger{}, false
	}
	if ledger.SchemaVersion != spendLedgerSchemaVersion {
		return spendLedger{}, false
	}
	return ledger, true
}

// loadSpendLedgersForPhase is the whole-phase read every report and the
// closeout line use. It calls loadSpendLedger once per entry in
// spendLedgerWorkflows(), keeps the ones that loaded, and returns them in
// that fixed order with ok true when at least one loaded. This is the
// function that makes a phase's reported cost the sum of its build and its
// continue rather than whichever finalized last.
func loadSpendLedgersForPhase(phase int) ([]spendLedger, bool) {
	var ledgers []spendLedger
	for _, workflow := range spendLedgerWorkflows() {
		if ledger, ok := loadSpendLedger(phase, workflow); ok {
			ledgers = append(ledgers, ledger)
		}
	}
	return ledgers, len(ledgers) > 0
}

// spendTotals is the whole-phase arithmetic summary over every row across
// every supplied ledger. MeasuredTokens and EstimatedTokens are
// deliberately two fields and are never summed into a field presented as a
// measurement (SPEND-04). GrandTotalTokens exists only as the honest
// arithmetic total of every row, measured or estimated alike.
type spendTotals struct {
	MeasuredTokens   int64   `json:"measured_tokens"`
	EstimatedTokens  int64   `json:"estimated_tokens"`
	GrandTotalTokens int64   `json:"grand_total_tokens"`
	MeasuredRows     int     `json:"measured_rows"`
	EstimatedRows    int     `json:"estimated_rows"`
	ProviderRows     int     `json:"provider_rows"`
	SessionRows      int     `json:"session_rows"`
	ProviderUSD      float64 `json:"provider_usd"`
	ProviderUSDRows  int     `json:"provider_usd_rows"`
}

// spendRowsAcross returns every row from every supplied ledger,
// concatenated in the supplied ledger order. This is the one place the
// whole-phase row set is formed; every aggregate below takes []spendLedger
// and starts here, so no caller can accidentally aggregate one workflow
// and call it the phase.
func spendRowsAcross(ledgers []spendLedger) []spendRow {
	var rows []spendRow
	for _, ledger := range ledgers {
		rows = append(rows, ledger.Rows...)
	}
	return rows
}

// computeSpendTotals classifies every row across every supplied ledger
// using row.Usage.Estimated(): an estimated row adds to EstimatedTokens/
// EstimatedRows; every other non-empty row adds to MeasuredTokens/
// MeasuredRows. ProviderRows and SessionRows are counted independently of
// that split. ProviderUSD accumulates only rows where row.Usage.Measured()
// is true and USDCost is greater than zero -- USD is relayed from the
// provider, never computed here (D-02: no per-model rate table is ever
// built in this file).
func computeSpendTotals(ledgers []spendLedger) spendTotals {
	var totals spendTotals
	for _, row := range spendRowsAcross(ledgers) {
		tokens := row.Usage.BilledTotalTokens()
		switch {
		case row.Usage.Estimated():
			totals.EstimatedTokens += tokens
			totals.EstimatedRows++
		case !row.Usage.Empty():
			totals.MeasuredTokens += tokens
			totals.MeasuredRows++
		}
		if row.Usage.Measured() {
			totals.ProviderRows++
			if row.Usage.USDCost > 0 {
				totals.ProviderUSD += row.Usage.USDCost
				totals.ProviderUSDRows++
			}
		}
		if row.Usage.Source == codex.UsageSourceSessionTranscript {
			totals.SessionRows++
		}
	}
	totals.GrandTotalTokens = totals.MeasuredTokens + totals.EstimatedTokens
	return totals
}

// spendPerWorkerAverageTokens divides the phase's GrandTotalTokens -- the
// same total every row contributes via BilledTotalTokens() -- by the total
// row count. When includeEstimates is false and the row set contains at
// least one estimated row, the derived metric is refused by name rather
// than silently blending a guess into a headline number.
func spendPerWorkerAverageTokens(ledgers []spendLedger, includeEstimates bool) (float64, error) {
	rows := spendRowsAcross(ledgers)
	if len(rows) == 0 {
		return 0, fmt.Errorf("per-worker average refused: no rows to average")
	}
	totals := computeSpendTotals(ledgers)
	if !includeEstimates && totals.EstimatedRows > 0 {
		return 0, fmt.Errorf(
			"per-worker average refused: %d of %d rows are estimates, not measurements — pass --include-estimates to compute it anyway",
			totals.EstimatedRows, len(rows),
		)
	}
	return float64(totals.GrandTotalTokens) / float64(len(rows)), nil
}

// spendParentRollup is one parent's roll-up entry in spendRollupByParent's
// result.
type spendParentRollup struct {
	Parent      string `json:"parent"`
	WorkerCount int    `json:"worker_count"`
	TotalTokens int64  `json:"total_tokens"`
}

// spendRollupByParent returns a slice sorted by parent name, one entry per
// distinct row.ParentName across every supplied ledger. Rows with an empty
// parent group under the literal "(unattributed)" so they are still
// visible and still counted. This is D-07's display-only roll-up over
// agent.SpawnEntry.ParentName; the deliberately-deferred self/subtree
// dual-column view belongs to Phase 177.
func spendRollupByParent(ledgers []spendLedger) []spendParentRollup {
	byParent := map[string]*spendParentRollup{}
	var order []string
	for _, row := range spendRowsAcross(ledgers) {
		parent := row.ParentName
		if parent == "" {
			parent = "(unattributed)"
		}
		entry, ok := byParent[parent]
		if !ok {
			entry = &spendParentRollup{Parent: parent}
			byParent[parent] = entry
			order = append(order, parent)
		}
		entry.WorkerCount++
		entry.TotalTokens += row.Usage.BilledTotalTokens()
	}
	sort.Strings(order)
	rollups := make([]spendParentRollup, 0, len(order))
	for _, parent := range order {
		rollups = append(rollups, *byParent[parent])
	}
	return rollups
}
