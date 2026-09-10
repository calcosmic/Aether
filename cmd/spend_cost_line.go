package cmd

import (
	"fmt"
	"strings"
	"time"
)

// The one cost line every build and every check ends with (Phase 196, COST-01,
// COST-02). This file is the ONLY renderer of it. Both the platform-driven
// lane and the direct in-process lane call the function below; nothing else in
// the tree may render a second one, and cmd/ceremony_closeout_spend_test.go
// counts occurrences per lane rather than merely asserting presence, because a
// test that checks the block appears cannot catch a second one appearing.
//
// The shape is the owner's own, recorded as D-01 as amended in
// .planning/phases/196-see-what-it-cost/196-CONTEXT.md, and is not open to
// redesign here:
//
//	Cost: 1.4M tokens across 3 workers
//	  Builder Mason-67    1.2M  measured
//	  Watcher Keen-12     220K  measured
//	  Scout Roam-90          —  not reported
//	(1 worker's tool did not report usage)
//
// Three properties are load-bearing and each has a test that fails without it:
//
//  1. A worker whose tool reported nothing shows NO number. Not a zero, which
//     reads as "this worker was free", and not an estimate wearing a label:
//     prompt-character count is the only estimate mechanism in the tree, so a
//     marked estimate would be exactly the thing this phase's third success
//     criterion forbids. The dash is the only honest cell.
//  2. The total counts ONLY the measured workers, and the block says so. The
//     owner accepted what that costs him — no sense of an unreported worker's
//     size, and a total that is incomplete rather than approximate — because a
//     number he cannot trust is worse than no number.
//  3. No currency symbol and no monetary amount appear anywhere, on any input,
//     including rows whose shared usage type carries the provider's own
//     reported cost. The type keeps that field; this renderer never shows it.
//
// The measured/unreported split is spendRowReportedUsage (cmd/spend_cmd.go),
// deliberately shared with the detail view, so the headline and the detail
// view can never disagree about which workers counted.

// spendCostLineHeading names the block on screen and is what the exactly-one
// tests count. Plain English: nothing here is a word this repository invented.
//
// It says PHASE, not "this run", because the figures below span both the
// building pass and the checking pass for the phase, and every attempt at each
// -- all of them recorded separately, none erasing another, so at the end of a
// check the block reports the whole phase rather than the last screen's worth.
//
// It says THE HELPERS, not simply "the phase", because that is the honest scope
// of what is counted (WR-07). The coordinator's own back-and-forth is real spend
// and is NOT in these figures: on this repository's own corpus it is 142,581 of
// 142,833 usage-bearing lines, so a heading claiming to state the phase's cost
// while omitting it would understate by far more than it reports. The block says
// so in the footnote below rather than leaving the reader to infer it.
const spendCostLineHeading = "What The Helpers Cost"

// spendCostLineSessionNote states the one exclusion the total makes that has
// nothing to do with an unreported worker. It is printed on every rendered
// block that has rows, because it is true of every one of them.
//
// It carries no em dash, deliberately: the em dash is spendNotReportedFigure,
// the sentinel that means "this worker has no figure", and putting one in prose
// directly under a column of them invites the reader to see a row that is not
// there.
const spendCostLineSessionNote = "(The coordinator's own back-and-forth, the main session itself, is not counted here. These figures are the helpers it sent.)\n"

// spendCompactTokenFigure renders a token count at the magnitude the owner
// wrote — 1.2M, 220K — rather than as raw digits. This is the HEADLINE; the
// exact recorded figure for every worker is one command away in the detail
// view, which deliberately abbreviates nothing.
//
// Magnitudes are TRUNCATED, never rounded up, so a headline can never claim a
// run cost more than its rows say it did.
func spendCompactTokenFigure(count int64) string {
	if count < 0 {
		count = -count
	}
	switch {
	case count >= 1_000_000:
		whole := count / 1_000_000
		tenths := (count % 1_000_000) / 100_000
		return fmt.Sprintf("%d.%dM", whole, tenths)
	case count >= 10_000:
		return fmt.Sprintf("%dK", count/1_000)
	case count >= 1_000:
		whole := count / 1_000
		tenths := (count % 1_000) / 100
		return fmt.Sprintf("%d.%dK", whole, tenths)
	default:
		return fmt.Sprintf("%d", count)
	}
}

// spendCostLineFigure is the figure column for one row: the compact magnitude
// for a worker whose tool reported one, and the dash sentinel for every worker
// whose tool did not — including a worker carrying a local estimate.
func spendCostLineFigure(row spendRow) string {
	if !spendRowReportedUsage(row) {
		return spendNotReportedFigure
	}
	return spendCompactTokenFigure(row.Usage.BilledTotalTokens())
}

// renderSpendCostLine is the whole block for one phase, read from that phase's
// recorded rows across both the build and the check, plus the elapsed time
// for the exact attempt this closeout belongs to (Phase 201, D-06).
//
// It is a pure read: loadSpendLedgersForPhase, loadLatestBuildAttempt and
// computeSpendTotals all only read, and nothing here saves anything. The
// total it prints is computeSpendTotals' own MeasuredTokens; this function
// performs no arithmetic on token counts of its own beyond choosing a
// magnitude to display, and the elapsed figure is spendElapsedFigure's own
// subtraction of the attempt's two recorded timestamps -- never a
// recomputation from anything already rendered.
//
// When no build attempt exists at all for this phase, no elapsed line is
// rendered -- not even the sentinel. That is a different fact from an
// attempt existing with an incomplete timestamp, which DOES render the
// sentinel: "nothing recorded about this attempt" and "this attempt did not
// finish measuring" are not the same claim, and only the second one is what
// the sentinel means.
func renderSpendCostLine(phase int) string {
	ledgers, _ := loadSpendLedgersForPhase(phase)
	elapsedFigure := ""
	if _, attempt, ok := loadLatestBuildAttempt(phase); ok {
		elapsedFigure = spendElapsedFigure(attempt.StartedAt, attempt.CompletedAt)
	}
	return renderSpendCostLineBlock(ledgers, elapsedFigure)
}

// spendElapsedFigure is the elapsed-time figure for one attempt, parsed from
// its own StartedAt and CompletedAt fields (RFC3339Nano, the format every
// production writer of buildAttemptRecord uses) and from nothing else.
// Either timestamp being absent, unparseable, or CompletedAt preceding
// StartedAt renders the identical dash sentinel the unreported cost figure
// uses -- an inferred or defaulted duration is exactly what D-06 forbids,
// and zero is a measurement, not an admission that nothing was measured.
func spendElapsedFigure(startedAt, completedAt string) string {
	start, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(startedAt))
	if err != nil {
		return spendNotReportedFigure
	}
	end, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(completedAt))
	if err != nil {
		return spendNotReportedFigure
	}
	elapsed := end.Sub(start)
	if elapsed < 0 {
		return spendNotReportedFigure
	}
	return elapsed.Round(time.Second).String()
}

// renderSpendCostLineFromLedgers is the ledgers-only render: the shape every
// existing caller and every existing test in this file uses, with no elapsed
// line at all (elapsedFigure == ""). renderSpendCostLine(phase) is the only
// caller that ever passes a non-empty elapsedFigure, because it is the only
// one with an attempt record to read one from.
func renderSpendCostLineFromLedgers(ledgers []spendLedger) string {
	return renderSpendCostLineBlock(ledgers, "")
}

// renderSpendCostLineBlock is the shared render body. elapsedFigure is the
// already-computed elapsed-time figure for the one attempt this block
// belongs to, or the empty string when the caller has no attempt to speak
// of at all (as opposed to an attempt whose figure is the dash sentinel,
// which is a non-empty string and DOES render a line). Leaves room for plan
// 201-12 to append the largest timing segment onto this same "Elapsed: "
// line without adding a second block.
func renderSpendCostLineBlock(ledgers []spendLedger, elapsedFigure string) string {
	var b strings.Builder
	b.WriteString(renderStageMarker(spendCostLineHeading))
	if elapsedFigure != "" {
		fmt.Fprintf(&b, "Elapsed: %s\n", elapsedFigure)
	}

	rows := spendRowsAcross(ledgers)
	if len(rows) == 0 {
		// No digits at all here, deliberately: any number on a screen that has
		// nothing to report would be read as a measurement of something.
		b.WriteString("No token use was recorded for this phase, so there is no figure to show.\n")
		return b.String()
	}

	totals := computeSpendTotals(ledgers)
	measured := totals.MeasuredRows
	unreported := len(rows) - measured

	distinct := map[string]struct{}{}
	for _, row := range rows {
		distinct[row.AgentName] = struct{}{}
	}
	b.WriteString(spendCostLineTotalSentence(totals.MeasuredTokens, len(rows), len(distinct), measured, unreported))
	b.WriteString("\n")

	attempts := spendAttemptLabels(rows)
	identityWidth, figureWidth := spendCostLineColumnWidths(rows)
	for _, row := range rows {
		mark := spendMarkNotReported
		if spendRowReportedUsage(row) {
			mark = spendMarkMeasured
		}
		rendered, plain := spendWorkerDescription(row, attempts[spendAttemptKey(row)])
		pad := identityWidth - spendDisplayWidth(plain)
		if pad < 0 {
			pad = 0
		}
		fmt.Fprintf(&b, "  %s%s%s%*s%s%s\n",
			rendered, strings.Repeat(" ", pad), spendCellSeparator,
			figureWidth, spendCostLineFigure(row),
			spendCellSeparator, mark,
		)
	}

	// The footnote exists only to explain the dashes above it. When every tool
	// reported, there are no dashes and the footnote would be a dangling
	// explanation of nothing, so it is omitted entirely.
	if unreported > 0 {
		b.WriteString(spendCostLineFootnote(unreported))
	}
	b.WriteString(spendCostLineSessionNote)
	return b.String()
}

// spendCostLineTotalSentence is the first line: the run total and how many
// workers ran, followed by a plain statement of what the total counts. It
// always says what the total counts, so the reader never has to infer the
// exclusion from the footnote.
//
// When no tool reported anything at all there is no total to state, so none is
// stated — the sentence says the cost is not known rather than showing a zero.
// runs is the number of ROWS and distinctWorkers the number of distinct worker
// names among them. They differ when a phase was built or checked more than
// once: the runtime derives a worker's name from the phase and its role, so a
// retry re-runs the SAME name. Saying "across 4 workers" when two workers each
// ran twice overstates the team; the rows below are labelled "(attempt 1)" and
// "(attempt 2)", so the noun is what has to carry the distinction here.
func spendCostLineTotalSentence(measuredTokens int64, runs, distinctWorkers, measured, unreported int) string {
	workers := runs
	if measured == 0 {
		// The one-worker wording is separate because the general sentence
		// reads "any of the 1 worker" at a count of one, and this block is
		// read by someone who has never opened a file here.
		if workers == 1 {
			return "Cost: not known. The one worker that ran had no figure reported by its tool, so there is no total to show."
		}
		return fmt.Sprintf(
			"Cost: not known. No tool reported a figure for any of the %s, so there is no total to show.",
			spendRunWord(runs, distinctWorkers),
		)
	}
	if unreported == 0 {
		if workers == 1 {
			return fmt.Sprintf(
				"Cost: %s tokens for the one worker that ran, whose tool reported a figure.",
				spendCompactTokenFigure(measuredTokens),
			)
		}
		return fmt.Sprintf(
			"Cost: %s tokens across %s. The total counts all %d, because every tool reported a figure.",
			spendCompactTokenFigure(measuredTokens), spendRunWord(runs, distinctWorkers), workers,
		)
	}
	return fmt.Sprintf(
		"Cost: %s tokens across %s. The total counts only the %d whose tools reported a figure.",
		spendCompactTokenFigure(measuredTokens), spendRunWord(runs, distinctWorkers), measured,
	)
}

// spendRunWord names what the total is spread across. Ordinarily that is
// workers. When a name recurs -- the phase was built or checked more than once,
// and the runtime regenerates the same name for the same role -- calling them
// workers would count one worker twice, so they are named as runs instead and
// the number of distinct workers is stated alongside.
func spendRunWord(runs, distinctWorkers int) string {
	if distinctWorkers <= 0 || distinctWorkers == runs {
		return spendWorkerWord(runs)
	}
	return fmt.Sprintf("%d worker runs by %s", runs, spendWorkerWord(distinctWorkers))
}

// spendCostLineFootnote counts the workers whose tools reported nothing and
// says plainly that their use is not in the total above.
func spendCostLineFootnote(unreported int) string {
	if unreported == 1 {
		return "(1 worker's tool did not report usage, so its use is not in the total.)\n"
	}
	return fmt.Sprintf("(%d workers' tools did not report usage, so their use is not in the total.)\n", unreported)
}

// spendDisplayWidth is how many terminal columns an already-rendered cell
// occupies. A plain counting of code points misaligns the breakdown, because a
// caste glyph occupies two columns while an emoji variation selector occupies
// none: "👁️🐜 Watcher" and "🔨🐜 Builder" carry different numbers of code
// points for the same width on screen.
//
// This measures TEXT for layout. It is never an input to any token count.
func spendDisplayWidth(text string) int {
	columns := 0
	for _, r := range text {
		switch {
		case r == 0xFE0F || r == 0xFE0E || r == 0x200D:
			// Variation selectors and the zero-width joiner draw nothing.
		case r >= 0x1F000 || (r >= 0x2600 && r <= 0x27BF):
			columns += 2
		default:
			columns++
		}
	}
	return columns
}

// spendCostLineColumnWidths measures the padded widths of the identity and
// figure columns over already-rendered text. These are display widths and are
// never an input to any token count.
func spendCostLineColumnWidths(rows []spendRow) (identity, figure int) {
	figure = spendDisplayWidth(spendNotReportedFigure)
	attempts := spendAttemptLabels(rows)
	for _, row := range rows {
		_, plain := spendWorkerDescription(row, attempts[spendAttemptKey(row)])
		if w := spendDisplayWidth(plain); w > identity {
			identity = w
		}
		if w := spendDisplayWidth(spendCostLineFigure(row)); w > figure {
			figure = w
		}
	}
	return identity, figure
}

// appendSpendCostLine puts the block at the end of an ending screen.
//
// Call sites are the terminal surfaces only — the platform-driven ending
// screen (cmd/ceremony_cmd.go) and the two direct in-process commands
// (cmd/codex_workflow_cmds.go). The finalizers deliberately do NOT call it:
// on the platform-driven lane the finalizer runs first and the ending screen
// follows it, so a finalizer that also printed the block would end that lane
// with two.
func appendSpendCostLine(visual string, phase int) string {
	if phase <= 0 {
		return visual
	}
	block := renderSpendCostLine(phase)
	if block == "" {
		return visual
	}
	if visual != "" && !strings.HasSuffix(visual, "\n") {
		visual += "\n"
	}
	return visual + "\n" + block
}
