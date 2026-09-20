package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// The read-only per-worker spend view (Phase 196, COST-03).
//
// Two properties are load-bearing and are proved by cmd/spend_cmd_test.go
// rather than promised by this comment:
//
//  1. It mutates NOTHING. CLAUDE.md makes "an inspection or --dry-run command
//     must not mutate state" a named rule with its own history here --
//     consolidation-phase-end --dry-run and consolidation-seal --dry-run both
//     wrote to instincts.json for months while their own documentation said
//     they would not. Everything below reads: loadSpendLedgersForPhase and
//     loadColonyState are both pure reads, and nothing here calls a save.
//     loadActiveColonyState is deliberately NOT used: it can persist a legacy-
//     shape repair, which would make this command a writer on some stores and
//     not others.
//
//  2. It computes no figure of its own. Every per-worker number is
//     row.Usage.BilledTotalTokens() -- the single mandated entry point, never
//     a re-derived sum (that shape produced this subsystem's 186x undercount)
//     -- and the total and the measured-worker count come straight off
//     computeSpendTotals. Nothing here derives a count from a character or a
//     prompt length; D-01 as amended forbids that anywhere.

// spendNotReportedFigure is D-01 as amended: a worker whose tool reported no
// usage shows NO number. Not a zero, which reads as "this worker was free",
// and not an estimate wearing a label, because prompt-character-count is the
// only estimate mechanism in the tree and success criterion 3 forbids exactly
// that. A dash is the only honest cell.
const spendNotReportedFigure = "—"

const (
	spendMarkMeasured    = "measured"
	spendMarkNotReported = "not reported"

	// spendCellSeparator is what makes a rendered row parse unambiguously into
	// three cells: no cell ever contains a run of two spaces, so a reader (and
	// the test that guards D-01's dash sentinel) can isolate the figure column
	// without matching on the whole line -- deterministic worker names carry
	// digits of their own, so a whole-line digit check would fail on correct
	// output.
	spendCellSeparator = "  "
)

// spendWorkflowHeading translates the two internal workflow words into
// ordinary English. "build" and "continue" are this repository's own
// vocabulary for two stages an owner thinks of as making the change and
// checking it.
func spendWorkflowHeading(workflow string) string {
	switch workflow {
	case spendWorkflowBuild:
		return "While building:"
	case spendWorkflowContinue:
		return "While checking the work:"
	default:
		return "Other work:"
	}
}

// spendRowReportedUsage reports whether a row carries a figure that may be
// shown. It must agree with computeSpendTotals' own measured classification
// (not estimated, not empty); TestSpendShowsNoNumberForUnreportedRows fails if
// the two ever drift, because the rendered total is taken from
// computeSpendTotals while the per-row marks are taken from here.
func spendRowReportedUsage(row spendRow) bool {
	return !row.Usage.Estimated() && !row.Usage.Empty()
}

// spendRowFigure returns the cell a row's figure column shows.
func spendRowFigure(row spendRow) string {
	if !spendRowReportedUsage(row) {
		return spendNotReportedFigure
	}
	return strconv.FormatInt(row.Usage.BilledTotalTokens(), 10)
}

// spendWorkerDescription is the identity cell: the caste glyph and its
// human-readable role, then the worker's own name, then the grouped job it
// owned if it owned one. Whitespace inside the job name is collapsed so a cell
// can never contain a run of two spaces, which is what keeps the rendered row
// unambiguously three cells wide.
//
// attempt is the run label spendAttemptLabels worked out for this row, and is
// empty for every worker that ran only once in this phase. It is filled in
// only when one name occurs in more than one attempt, which the runtime makes
// routine: a re-run of a phase gives its workers the same deterministic names,
// so two rows reading "Builder Mason-67" would otherwise look like one worker
// billed twice rather than one worker run twice.
//
// It returns the rendered cell and its plain equivalent. The plain form exists
// only so the column can be padded: casteIdentity colours the role label with
// ANSI escapes, and padding on the coloured string would count the escape
// bytes as visible width and break the alignment it was meant to create.
func spendWorkerDescription(row spendRow, attempt string) (rendered, plain string) {
	name := strings.TrimSpace(row.AgentName)
	if name == "" {
		name = "(unnamed worker)"
	}
	suffix := " " + name
	if attempt = strings.TrimSpace(attempt); attempt != "" {
		suffix += " (" + attempt + ")"
	}
	if job := strings.Join(strings.Fields(row.JobName), " "); job != "" {
		suffix += " (job: " + job + ")"
	}
	emoji := casteEmoji(row.Caste)
	if emoji != "🐜" {
		emoji += "🐜"
	}
	label := casteLabel(row.Caste)
	return casteIdentity(row.Caste) + suffix, emoji + " " + label + suffix
}

// spendColumnWidths returns the padded widths of the identity and figure
// columns, measured in runes over already-rendered strings. These are display
// widths over text, never an input to any token count.
func spendColumnWidths(ledgers []spendLedger) (identity, figure int) {
	figure = len([]rune(spendNotReportedFigure))
	rows := spendRowsAcross(ledgers)
	attempts := spendAttemptLabels(rows)
	for _, row := range rows {
		_, plain := spendWorkerDescription(row, attempts[spendAttemptKey(row)])
		if w := len([]rune(plain)); w > identity {
			identity = w
		}
		if w := len([]rune(spendRowFigure(row))); w > figure {
			figure = w
		}
	}
	return identity, figure
}

// renderSpendText is the view an owner reads. Figures are printed exactly as
// recorded -- no thousands separators and no 1.2M-style shortening -- because
// abbreviating is arithmetic, and the whole promise of this view is that every
// number in it can be checked against the recorded row unchanged.
func renderSpendText(phase int, ledgers []spendLedger) string {
	var b strings.Builder

	rows := spendRowsAcross(ledgers)
	if len(rows) == 0 {
		b.WriteString(fmt.Sprintf(
			"Nothing has been recorded for phase %d yet, so there is no token use to show.\n",
			phase,
		))
		b.WriteString("Token use is recorded when a build or a check finishes; run one and ask again.\n")
		return b.String()
	}

	totals := computeSpendTotals(ledgers)
	identityWidth, figureWidth := spendColumnWidths(ledgers)
	attempts := spendAttemptLabels(rows)

	b.WriteString(fmt.Sprintf("Token use for phase %d, worker by worker, as each worker's own tool reported it.\n", phase))

	for _, ledger := range ledgers {
		if len(ledger.Rows) == 0 {
			continue
		}
		b.WriteString("\n")
		b.WriteString(spendWorkflowHeading(ledger.Workflow) + "\n")
		for _, row := range ledger.Rows {
			mark := spendMarkNotReported
			if spendRowReportedUsage(row) {
				mark = spendMarkMeasured
			}
			rendered, plain := spendWorkerDescription(row, attempts[spendAttemptKey(row)])
			pad := identityWidth - len([]rune(plain))
			if pad < 0 {
				pad = 0
			}
			b.WriteString(fmt.Sprintf(
				"  %s%s%s%*s%s%s\n",
				rendered, strings.Repeat(" ", pad), spendCellSeparator,
				figureWidth, spendRowFigure(row),
				spendCellSeparator, mark,
			))
		}
	}

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(
		"Total: %d tokens, counting only the %s whose tools reported a figure.\n",
		totals.MeasuredTokens, spendWorkerWord(totals.MeasuredRows),
	))
	if unreported := len(rows) - totals.MeasuredRows; unreported > 0 {
		b.WriteString(fmt.Sprintf(
			"%s did not report usage, so %s no number above and %s not in the total.\n",
			capitalizeFirstRune(spendWorkerWord(unreported)+"'s tool"),
			pluralizeHasHave(unreported), pluralizeIsAre(unreported),
		))
	}
	b.WriteString("\nThis is a read-only view; running it changes nothing.\n")
	return b.String()
}

// spendWorkerWord renders a worker count with its noun, so the sentences above
// read naturally at one and at many.
func spendWorkerWord(count int) string {
	if count == 1 {
		return "1 worker"
	}
	return fmt.Sprintf("%d workers", count)
}

func pluralizeHasHave(count int) string {
	if count == 1 {
		return "it has"
	}
	return "they have"
}

func pluralizeIsAre(count int) string {
	if count == 1 {
		return "it is"
	}
	return "they are"
}

func capitalizeFirstRune(text string) string {
	if text == "" {
		return text
	}
	return strings.ToUpper(text[:1]) + text[1:]
}

// spendWorkerJSON is one row on the machine surface. Tokens is a pointer so an
// unreported worker serializes as null rather than as 0 -- the same rule the
// text view follows, applied to the JSON, so no consumer can read a zero as a
// measurement.
type spendWorkerJSON struct {
	Name     string `json:"name"`
	Caste    string `json:"caste"`
	Role     string `json:"role"`
	Job      string `json:"job,omitempty"`
	Workflow string `json:"workflow"`
	Status   string `json:"status"`
	Tokens   *int64 `json:"tokens"`
	Reported bool   `json:"reported"`
}

func spendWorkersJSON(ledgers []spendLedger) []spendWorkerJSON {
	out := make([]spendWorkerJSON, 0)
	for _, ledger := range ledgers {
		for _, row := range ledger.Rows {
			entry := spendWorkerJSON{
				Name:     row.AgentName,
				Caste:    row.Caste,
				Role:     casteLabel(row.Caste),
				Job:      row.JobName,
				Workflow: ledger.Workflow,
				Status:   row.Status,
				Reported: spendRowReportedUsage(row),
			}
			if entry.Reported {
				tokens := row.Usage.BilledTotalTokens()
				entry.Tokens = &tokens
			}
			out = append(out, entry)
		}
	}
	return out
}

func writeSpendEnvelope(result map[string]interface{}) {
	if shouldRenderVisualOutput(stdout) {
		if text, ok := result["_text"].(string); ok {
			delete(result, "_text")
			writeVisualOutput(stdout, text)
			return
		}
	}
	delete(result, "_text")
	data, err := json.Marshal(map[string]interface{}{"ok": true, "result": result})
	if err != nil {
		outputError(2, fmt.Sprintf("failed to marshal spend result: %v", err), nil)
		return
	}
	fmt.Fprintln(stdout, string(data))
}

// resolveSpendPhase returns the phase to report on: the --phase flag when
// given, otherwise the colony's current phase read through loadColonyState,
// which is a pure read.
func resolveSpendPhase(flagPhase int) (int, error) {
	if flagPhase > 0 {
		return flagPhase, nil
	}
	state, err := loadColonyState()
	if err != nil {
		return 0, err
	}
	if state == nil || state.CurrentPhase <= 0 {
		return 0, fmt.Errorf("no phase is active, so there is nothing to report on — pass --phase <number> to look at a particular one")
	}
	return state.CurrentPhase, nil
}

var spendCmd = &cobra.Command{
	Use:   "spend",
	Short: "Show what each worker's tools reported this run cost, in tokens",
	Long: "Show, worker by worker, the tokens each worker's own tool reported for a phase.\n\n" +
		"This reads and prints. It changes nothing, and it shows no number for a worker\n" +
		"whose tool reported none.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		phase, err := resolveSpendPhase(mustGetInt(cmd, "phase"))
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		ledgers, _ := loadSpendLedgersForPhase(phase)
		totals := computeSpendTotals(ledgers)
		workers := spendWorkersJSON(ledgers)

		writeSpendEnvelope(map[string]interface{}{
			"phase":              phase,
			"workers":            workers,
			"worker_count":       len(workers),
			"measured_tokens":    totals.MeasuredTokens,
			"measured_workers":   totals.MeasuredRows,
			"unreported_workers": len(workers) - totals.MeasuredRows,
			"_text":              renderSpendText(phase, ledgers),
		})
		return nil
	},
}

func init() {
	spendCmd.Flags().Int("phase", 0, "Phase to report on (defaults to the colony's current phase)")
	rootCmd.AddCommand(spendCmd)
}
