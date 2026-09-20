package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

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
// deliberately reuse agent.SpawnEntry's vocabulary (AgentName/Caste/Task/
// Status) so a row joins to a spawn-tree entry by name with no translation
// layer. Usage is never omitempty -- a dispatch with a zero-value usage must
// still be visible as a row; a dispatch never vanishes from the ledger.
//
// There is deliberately NO ParentName and NO ToolCount here, and neither
// should be re-added without a production writer landing in the same change.
// Both existed on the salvaged version of this struct and neither was ever
// assigned by writeSpendRowsForRun -- the only production writer. ParentName
// carried a display-only roll-up (spendRollupByParent) whose test seeded the
// field by hand, so it passed over a shape production cannot emit and would
// have returned a single "(unattributed)" bucket against a real ledger;
// ToolCount was declared, never written and never read. Both were removed in
// the Phase 196 closeout, before this ledger's first production write, so no
// schema bump and no migration was owed -- the same argument JobName was
// added under. A field on a durable record that production never fills is a
// schema lie, and TestNothingThisPhaseAddedIsUncalled now fails on one.
type spendRow struct {
	AgentName string `json:"name"`
	Caste     string `json:"caste"`
	Task      string `json:"task"`
	// JobName is the grouped job this worker owned, mirroring
	// codexBuildDispatch.JobName (cmd/codex_build.go). Phase 195 made a single
	// worker able to own a chain of dependent tasks, so a ledger keyed only by
	// worker cannot say which job a worker's tokens belong to. A worker that
	// owned one ungrouped task records nothing here rather than a placeholder,
	// so the key is absent from the serialized form entirely.
	//
	// It is added now, before this ledger has ever been written to in
	// production, precisely so no schema version bump and no migration is owed
	// for a one-line field.
	JobName string `json:"job_name,omitempty"`
	// Status is the DISPATCH status -- the same vocabulary
	// normalizeRuntimeDispatchStatus and dispatchStatusIcon already use across
	// this package -- and never the task-receipt status
	// (codex.TaskReceiptStatusCompleted), which is task-scoped completion
	// evidence rather than worker-scoped outcome. Two vocabularies for one idea
	// is this repository's documented failure mode; saveSpendLedger refuses any
	// word outside spendRowStatusVocabulary by name rather than storing it.
	Status string            `json:"status"`
	Usage  codex.WorkerUsage `json:"usage"`
	// RunID names the attempt at this phase that this row was filed by.
	//
	// A phase is routinely built more than once -- the recovery command
	// redispatches only the unfinished tasks and finalizes the same phase
	// again, and a blocked check hands the owner `build --force`. Without this
	// field the writer could not tell a re-finalize of one attempt (which must
	// replace its rows) from a second attempt at the phase (which must add to
	// them), so it replaced in both cases and a retry erased everything the
	// first attempt spent. Empty on rows written before this field existed;
	// see writeSpendRowsForRun for how those are treated.
	RunID string `json:"run_id,omitempty"`
}

// spendRowStatusVocabulary is the closed set of dispatch statuses a ledger row
// may carry, in the fixed order the refusal message lists them. It is the
// vocabulary dispatchStatusIcon (cmd/codex_visuals.go) already renders, not a
// second one invented here.
func spendRowStatusVocabulary() []string {
	return []string{
		"completed",
		"completed_no_change",
		"blocked",
		"interrupted",
		"failed",
		"timeout",
		"manually-reconciled",
		"superseded",
		"starting",
		"spawned",
		"active",
		"running",
	}
}

// normalizeSpendRowStatus folds a row status onto exactly one word from
// spendRowStatusVocabulary, using the same synonym mappings
// normalizeCloseoutWorkerStatus (cmd/closeout_cmd.go) and
// normalizeRuntimeDispatchStatus (cmd/dispatch_runtime.go) already apply, and
// reports whether the result is in the vocabulary.
//
// The empty string is refused rather than folded. normalizeRuntimeDispatchStatus
// reads an unstated status as "failed", which is the right default when
// summarizing a dispatch that never reported -- but recording a worker as
// failed in a durable cost ledger because its writer forgot to state an
// outcome would put a wrong fact on disk. The writer is made to say.
func normalizeSpendRowStatus(status string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(status))
	if normalized == "" {
		return "", false
	}
	switch normalized {
	case "complete", "done", "success", "succeeded", "passed", "code_written":
		normalized = "completed"
	default:
		normalized = normalizeRuntimeDispatchStatus(normalized)
	}
	for _, known := range spendRowStatusVocabulary() {
		if known == normalized {
			return normalized, true
		}
	}
	return "", false
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
//
// Every row's status is validated against spendRowStatusVocabulary BEFORE
// anything is written. A row carrying a word outside it is refused by name,
// naming the worker too, and no file is created or replaced -- a partially
// written ledger would be worse than a refused one, because the next read
// would return it as fact.
//
// Validation folds synonyms onto the canonical word on a COPY of the rows, so
// a caller's own slice is never rewritten underneath it.
func saveSpendLedger(entry spendLedger) error {
	if store == nil {
		return fmt.Errorf("save spend ledger: store is not initialized")
	}
	rel := spendLedgerRelForPhaseWorkflow(entry.Phase, entry.Workflow)
	if rel == "" {
		return fmt.Errorf("save spend ledger: workflow %q is not one of %v", entry.Workflow, spendLedgerWorkflows())
	}

	rows := make([]spendRow, len(entry.Rows))
	copy(rows, entry.Rows)
	// Keyed by run id AND worker name, never by name alone -- see the comment
	// at the check below.
	measuredByWorker := map[string]bool{}
	for i := range rows {
		worker := strings.TrimSpace(rows[i].AgentName)
		if worker == "" {
			worker = "(unnamed worker)"
		}
		// Two rows may share a name -- a worker never vanishes from this
		// ledger, so a name collision cannot be answered by dropping one. Two
		// rows that both carry a MEASUREMENT under one name is the harm: the
		// run's total then counts one measurement twice, which is what a run of
		// 1,000 tokens reporting 2.0K looked like (WR-02).
		//
		// The question is scoped to ONE ATTEMPT, and that scope is the whole
		// correctness of it (NEW-01). deterministicAntName is a pure function of
		// caste and phase, so re-running a phase gives the retry's workers the
		// SAME names as the first attempt's -- and a ledger that keeps every
		// attempt's rows (CR-02) is then holding exactly the pair an
		// unscoped rule forbids. Refusing that pair refused the whole retry:
		// nothing was written, and the owner was shown the first attempt's
		// figure under a sentence claiming it counted every worker. Two measured
		// rows under one name in one attempt is still a double count; the same
		// two across two attempts is two runs of one worker, which is what a
		// retry IS.
		if !rows[i].Usage.Empty() {
			key := strings.TrimSpace(rows[i].RunID) + "\x00" + worker
			if measuredByWorker[key] {
				return fmt.Errorf(
					"save spend ledger: two rows filed under worker %s in the same run both carry a token figure — one measurement would be counted twice in this phase's total",
					worker,
				)
			}
			measuredByWorker[key] = true
		}
		normalized, ok := normalizeSpendRowStatus(rows[i].Status)
		if !ok {
			if strings.TrimSpace(rows[i].Status) == "" {
				return fmt.Errorf(
					"save spend ledger: worker %s has no status — a ledger row must state its dispatch outcome, one of %v",
					worker, spendRowStatusVocabulary(),
				)
			}
			return fmt.Errorf(
				"save spend ledger: worker %s has status %q, which is not a dispatch status — expected one of %v",
				worker, rows[i].Status, spendRowStatusVocabulary(),
			)
		}
		rows[i].Status = normalized
	}
	entry.Rows = rows

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
//
// Every field here is a token count or a row count. None is a money amount,
// and none may become one. Phase 174's D-02 and Phase 196's D-01 both forbid
// a currency figure on the cost line, and this struct is what a headline
// renderer reads. The salvaged version of this file carried ProviderUSD and
// ProviderUSDRows, relaying the provider's own reported cost; neither
// computed a price from a rate and nothing read either of them, which is
// precisely why they were removed -- an unread money field on the totals
// struct is a standing invitation to render it. They can return the day
// something actually asks for one, with the test that keeps it off the
// headline. TestSpendLedgerCarriesNoCurrencyField enforces this.
type spendTotals struct {
	MeasuredTokens   int64 `json:"measured_tokens"`
	EstimatedTokens  int64 `json:"estimated_tokens"`
	GrandTotalTokens int64 `json:"grand_total_tokens"`
	MeasuredRows     int   `json:"measured_rows"`
	EstimatedRows    int   `json:"estimated_rows"`
	ProviderRows     int   `json:"provider_rows"`
	SessionRows      int   `json:"session_rows"`
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

// spendAttemptKey identifies one row's (attempt, worker) pair. It is the key
// spendAttemptLabels answers on, and the same pair saveSpendLedger scopes its
// double-count refusal to.
func spendAttemptKey(row spendRow) string {
	return strings.TrimSpace(row.RunID) + "\x00" + strings.TrimSpace(row.AgentName)
}

// spendAttemptLabels works out which rows need to say which attempt they are.
//
// A phase can be built or checked more than once -- a blocked check hands the
// owner `build --force`, and recovery redispatches the unfinished tasks -- and
// every attempt's rows are kept (CR-02). The worker names do NOT change between
// attempts: deterministicAntName is a pure function of caste and phase. So the
// breakdown can hold two rows both reading "Builder Mason-67", which without a
// label reads as one worker billed twice rather than one worker run twice.
//
// Only names that actually occur in more than one attempt get a label, so a
// phase built once -- the ordinary case -- is unchanged. Attempts are numbered
// in the order they appear in the rows, which is the order they happened,
// because each run appends its own rows after the ones already recorded.
//
// It reads rows and returns text. It computes nothing about tokens.
func spendAttemptLabels(rows []spendRow) map[string]string {
	runsByWorker := map[string][]string{}
	for _, row := range rows {
		worker := strings.TrimSpace(row.AgentName)
		run := strings.TrimSpace(row.RunID)
		seen := false
		for _, known := range runsByWorker[worker] {
			if known == run {
				seen = true
				break
			}
		}
		if !seen {
			runsByWorker[worker] = append(runsByWorker[worker], run)
		}
	}
	labels := map[string]string{}
	for worker, runs := range runsByWorker {
		if len(runs) < 2 {
			continue
		}
		for i, run := range runs {
			labels[run+"\x00"+worker] = fmt.Sprintf("attempt %d", i+1)
		}
	}
	return labels
}

// computeSpendTotals classifies every row across every supplied ledger
// using row.Usage.Estimated(): an estimated row adds to EstimatedTokens/
// EstimatedRows; every other non-empty row adds to MeasuredTokens/
// MeasuredRows. ProviderRows and SessionRows are counted independently of
// that split.
//
// It reads no money figure off any row. codex.WorkerUsage keeps its own
// provider-reported USDCost for callers outside this phase, and this
// function deliberately walks past it: no currency amount enters the totals,
// so none can leave them (D-01, D-02).
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
		}
		if row.Usage.Source == codex.UsageSourceSessionTranscript {
			totals.SessionRows++
		}
	}
	totals.GrandTotalTokens = totals.MeasuredTokens + totals.EstimatedTokens
	return totals
}
