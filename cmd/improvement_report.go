package cmd

// LEARN-08 (204-10-PLAN.md): the improvement report. A system that grades
// its own improvement will find a way to grade itself well -- that is not
// cynicism about this system in particular, it is the reason hard failures
// here are non-gameable. Two things worth knowing are opposites and must
// never be combined into one figure: how often the system finished a
// genuinely useful task (VerifiedUsefulSuccess), and how often a person had
// to step in to prevent something wrong (PreventableInterventions). A single
// number blending both can always be improved by getting worse at one and
// better at the other -- so improvementReport declares NO combined score
// field, deliberately, and never will. The next person to read this file and
// reach for a convenient "overall health" percentage: don't. Render the two
// figures, separately, every time.
//
// Both figures are read from the durable episode ledger (cmd/episode_ledger.go
// -- recordEpisodeOutcome/readEpisodeLedger) rather than any live, TTL-bounded
// feed, and hard-gate failures are read from what a terminal record already
// recorded, never recomputed here. As of 204-04/WINDOWS.md entry 38, no
// production lane yet populates EvidenceIDs, HardGateResults or Interventions
// on a real run -- the boundary callers that would populate them
// (emitColonyLiveOutcomeRecorded, emitColonyLiveInterventionRecorded) exist
// but have no caller yet. Until that gap closes, a real window of production
// episodes reports every episode as unclassified, honestly, rather than
// inventing a figure the ledger was never given the facts to support.
import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// reportWindow bounds which ledger records an improvement report covers, by
// each record's own recorded timestamp (episodeLedgerRecordSortTimestamp --
// a record's EndedAt when it has one, otherwise its StartedAt). Both bounds
// are RFC3339 strings; an empty bound is unbounded on that side, and a
// wholly empty window covers every record it is given -- deliberately, so a
// window is never mistaken for a hidden default filter.
type reportWindow struct {
	// Start is inclusive; a record whose timestamp is before Start is
	// excluded. Empty means unbounded below.
	Start string
	// End is exclusive; a record whose timestamp is at or after End is
	// excluded. Empty means unbounded above.
	End string
}

// reportRatio carries a count over a total as two plain integers -- never a
// single derived float -- so every classification and every threshold
// comparison in this file can cross-multiply the integers instead of
// comparing a rounded percentage (Planner Assumption W, 204-10-PLAN.md: "a
// percentage is for reading, never for deciding").
type reportRatio struct {
	Count int
	Total int
}

// ratioOf constructs a reportRatio from a count and a total.
func ratioOf(count, total int) reportRatio {
	return reportRatio{Count: count, Total: total}
}

// reportPercentDecimalPlaces is the number of decimal places a displayed
// percentage rounds to. Named so the rounding call site never carries a bare
// numeral.
const reportPercentDecimalPlaces = 1

// Percent derives the display-only percentage for r, rounded to
// reportPercentDecimalPlaces with ties rounding to the nearest even digit
// (banker's rounding), computed explicitly rather than relying on fmt's
// default rounding verb -- Planner Assumption W. A zero total reports 0.0,
// never a division error and never a silently perfect 100.0. This value is
// for display ONLY: TestPercentagesAreNeverCompared structurally forbids any
// comparison operator in this file being applied to a Percent() result.
func (r reportRatio) Percent() float64 {
	if r.Total <= 0 {
		return 0
	}
	raw := (float64(r.Count) / float64(r.Total)) * 100
	return roundHalfToEven(raw, reportPercentDecimalPlaces)
}

// roundHalfToEven rounds value to the given number of decimal places, with a
// value falling exactly halfway between two representable values rounding
// to whichever is even -- the convention Planner Assumption W names
// explicitly rather than leaving to Go's default float formatting (which
// rounds half away from zero).
func roundHalfToEven(value float64, decimals int) float64 {
	mult := math.Pow(10, float64(decimals))
	scaled := value * mult
	floor := math.Floor(scaled)
	diff := scaled - floor
	const halfwayEpsilon = 1e-9
	var rounded float64
	switch {
	case diff > 0.5+halfwayEpsilon:
		rounded = floor + 1
	case diff < 0.5-halfwayEpsilon:
		rounded = floor
	default:
		// Exactly halfway (within floating-point epsilon): round to even.
		if math.Mod(floor, 2) == 0 {
			rounded = floor
		} else {
			rounded = floor + 1
		}
	}
	return rounded / mult
}

// String renders r as "N out of M (P%)" -- the count and total always shown
// alongside, never replaced by, the percentage.
func (r reportRatio) String() string {
	return fmt.Sprintf("%d out of %d (%.1f%%)", r.Count, r.Total, r.Percent())
}

// improvementReportSuccessTerminalStatus is the run-status vocabulary's own
// success value (cmd/spawn_runs.go's summarizeRunStatus: "completed" is the
// default and only success outcome in that closed vocabulary; "active",
// "blocked" and "failed" are its other members). Named here so the
// classification below never carries a bare string literal.
const improvementReportSuccessTerminalStatus = "completed"

// Named classification thresholds. Every classification and comparison in
// this file compares integers cross-multiplied against these constants,
// never a Percent() result -- see roundHalfToEven's doc comment and
// TestPercentagesAreNeverCompared.
const (
	// improvementReportSuccessThresholdCount / Total: a verified-success
	// ratio classifies as "meeting the bar" once it reaches 80%.
	improvementReportSuccessThresholdCount = 4
	improvementReportSuccessThresholdTotal = 5

	// improvementReportInterventionThresholdCount / Total: a preventable-
	// intervention ratio classifies as "elevated" once it reaches 20%.
	improvementReportInterventionThresholdCount = 1
	improvementReportInterventionThresholdTotal = 5
)

// improvementReportSuccessClassification names what a verified-success ratio
// says about the reporting window, never a bare percentage.
func improvementReportSuccessClassification(r reportRatio) string {
	if r.Total == 0 {
		return "no episodes in this window"
	}
	if r.Count*improvementReportSuccessThresholdTotal >= improvementReportSuccessThresholdCount*r.Total {
		return "meeting the bar"
	}
	return "below the bar"
}

// improvementReportInterventionClassification names what a preventable-
// intervention ratio says about the reporting window, never a bare
// percentage.
func improvementReportInterventionClassification(r reportRatio) string {
	if r.Total == 0 {
		return "no episodes in this window"
	}
	if r.Count*improvementReportInterventionThresholdTotal >= improvementReportInterventionThresholdCount*r.Total {
		return "elevated"
	}
	return "within bounds"
}

// improvementReportHardFailure is one hard-gate failure recorded on one
// episode's terminal record -- the gate that failed, and, when the gate
// matches a declared eval-gate sentinel (cmd/eval_gates.go), the
// consequence that sentinel's own entry names. A gate with no matching
// sentinel is still reported, with an empty Consequence: the hard-failure
// list is never filtered down to only sentinel-named gates, only enriched by
// them.
type improvementReportHardFailure struct {
	EpisodeID   string
	Gate        string
	Consequence string
}

// assembleHardFailureList reads every closed episode's own recorded
// HardGateResults and returns a failing entry for each gate recorded false.
// This function takes only the recorded results and the sentinel list to
// enrich a consequence string with -- there is no parameter, flag or branch
// here capable of excluding, downgrading or offsetting an entry with a
// success recorded elsewhere. A hard failure survives regardless of how many
// unrelated episodes in records succeeded.
func assembleHardFailureList(records []episodeLedgerRecord, sentinels []evalGateSentinel) []improvementReportHardFailure {
	consequenceByGate := make(map[string]string, len(sentinels))
	for _, s := range sentinels {
		consequenceByGate[s.Test] = s.Consequence
	}

	var failures []improvementReportHardFailure
	for _, episodeID := range episodeLedgerEpisodeIDs(records) {
		terminal, ok := episodeLedgerTerminalRecord(records, episodeID)
		if !ok {
			continue
		}
		gates := make([]string, 0, len(terminal.HardGateResults))
		for gate := range terminal.HardGateResults {
			gates = append(gates, gate)
		}
		sort.Strings(gates)
		for _, gate := range gates {
			if terminal.HardGateResults[gate] {
				continue
			}
			failures = append(failures, improvementReportHardFailure{
				EpisodeID:   episodeID,
				Gate:        gate,
				Consequence: consequenceByGate[gate],
			})
		}
	}
	if failures == nil {
		failures = []improvementReportHardFailure{}
	}
	return failures
}

// isVerifiedUsefulSuccess reports whether terminal describes a verified
// useful-task success: its terminal result is this project's own success
// status, every hard-gate result it recorded is passing (a terminal record
// with no hard-gate results at all records no evidence of failure, so it is
// not excluded on that basis alone), and it carries at least one evidence
// identifier -- evidence being what separates a verified success from a
// merely claimed one.
func isVerifiedUsefulSuccess(terminal episodeLedgerRecord) bool {
	if terminal.TerminalResult != improvementReportSuccessTerminalStatus {
		return false
	}
	if len(terminal.EvidenceIDs) == 0 {
		return false
	}
	for _, passing := range terminal.HardGateResults {
		if !passing {
			return false
		}
	}
	return true
}

// improvementReportInterventionEntry is one categorised owner intervention
// recorded on one episode -- the category coming from the free-form category
// string an intervention_recorded ledger record already carries
// (episodeLedgerRecord.Interventions, written by
// emitColonyLiveInterventionRecorded).
type improvementReportInterventionEntry struct {
	EpisodeID string
	Category  string
}

// collectPreventableInterventions returns one entry per non-empty category
// string recorded on every intervention_recorded record in records. An
// episode's own records showing the runtime would otherwise have proceeded
// is exactly what an intervention_recorded record IS -- it is written only
// when an owner action interrupted a run that was not itself already
// terminal (emitColonyLiveInterventionRecorded's own call sites).
func collectPreventableInterventions(records []episodeLedgerRecord) []improvementReportInterventionEntry {
	var entries []improvementReportInterventionEntry
	for _, r := range records {
		if r.RecordKind != episodeLedgerRecordKindIntervention {
			continue
		}
		for _, category := range r.Interventions {
			category = strings.TrimSpace(category)
			if category == "" {
				continue
			}
			entries = append(entries, improvementReportInterventionEntry{
				EpisodeID: r.EpisodeID,
				Category:  category,
			})
		}
	}
	if entries == nil {
		entries = []improvementReportInterventionEntry{}
	}
	return entries
}

// improvementReportRow is one episode's row in the report: whether it
// counted as a verified success, whether it counted as a preventable
// intervention (both may be true for the same episode -- neither excludes
// the other), and the intervention categories it recorded, if any.
type improvementReportRow struct {
	EpisodeID               string
	VerifiedSuccess         bool
	PreventableIntervention bool
	InterventionCategories  []string
}

// improvementReport is the two-figure report LEARN-08 requires. There is
// deliberately no combined score field -- see this file's top-of-file
// comment. Every count is over TotalEpisodes, the number of distinct
// episodes in Window; UnclassifiedEpisodes names every episode matching
// neither figure, so a window is never silently short a classification.
type improvementReport struct {
	Window reportWindow

	VerifiedUsefulSuccess    reportRatio
	PreventableInterventions reportRatio

	TotalEpisodes        int
	UnclassifiedEpisodes []string
	HardFailures         []improvementReportHardFailure
	Rows                 []improvementReportRow
}

// recordsInWindow filters records to those whose own sort timestamp
// (episodeLedgerRecordSortTimestamp) falls within window. An entirely empty
// window returns records unfiltered.
func recordsInWindow(records []episodeLedgerRecord, window reportWindow) []episodeLedgerRecord {
	if window.Start == "" && window.End == "" {
		return records
	}
	var filtered []episodeLedgerRecord
	for _, r := range records {
		ts := episodeLedgerRecordSortTimestamp(r)
		if window.Start != "" && ts < window.Start {
			continue
		}
		if window.End != "" && ts >= window.End {
			continue
		}
		filtered = append(filtered, r)
	}
	if filtered == nil {
		filtered = []episodeLedgerRecord{}
	}
	return filtered
}

// buildImprovementReport is pure over its arguments: given the same records,
// sentinels and window it always returns the same report. A window
// containing zero episodes reports VerifiedUsefulSuccess and
// PreventableInterventions both as 0 out of 0, never a perfect score and
// never an absent report.
func buildImprovementReport(records []episodeLedgerRecord, sentinels []evalGateSentinel, window reportWindow) improvementReport {
	windowed := recordsInWindow(records, window)
	episodeIDs := episodeLedgerEpisodeIDs(windowed)
	sort.Strings(episodeIDs)

	interventionsByEpisode := map[string][]string{}
	for _, entry := range collectPreventableInterventions(windowed) {
		interventionsByEpisode[entry.EpisodeID] = append(interventionsByEpisode[entry.EpisodeID], entry.Category)
	}

	successCount := 0
	interventionCount := 0
	var unclassified []string
	rows := make([]improvementReportRow, 0, len(episodeIDs))

	for _, id := range episodeIDs {
		terminal, hasTerminal := episodeLedgerTerminalRecord(windowed, id)
		success := hasTerminal && isVerifiedUsefulSuccess(terminal)
		categories := interventionsByEpisode[id]
		hasIntervention := len(categories) > 0

		if success {
			successCount++
		}
		if hasIntervention {
			interventionCount++
		}
		if !success && !hasIntervention {
			unclassified = append(unclassified, id)
		}
		rows = append(rows, improvementReportRow{
			EpisodeID:               id,
			VerifiedSuccess:         success,
			PreventableIntervention: hasIntervention,
			InterventionCategories:  categories,
		})
	}
	if unclassified == nil {
		unclassified = []string{}
	}

	total := len(episodeIDs)
	return improvementReport{
		Window:                   window,
		VerifiedUsefulSuccess:    ratioOf(successCount, total),
		PreventableInterventions: ratioOf(interventionCount, total),
		TotalEpisodes:            total,
		UnclassifiedEpisodes:     unclassified,
		HardFailures:             assembleHardFailureList(windowed, sentinels),
		Rows:                     rows,
	}
}

// renderImprovementReport renders report in the shared voice: every content
// line opens with a glyph from voiceGlyph, no word this project invented
// appears without an inline translation, each figure appears as a count over
// a total with the percentage beside it (never instead of it), and the two
// figures render in fully separate paragraphs -- never one line, never one
// table row -- so TestTwoFiguresAreNeverCombined can assert structurally
// that no rendered line carries both. This function renders inside an
// already-registered screen surface rather than declaring a new voiced
// screen family of its own.
func renderImprovementReport(report improvementReport) string {
	var sb strings.Builder

	sb.WriteString(voiceLine("status", "How the system is doing, in two separate numbers that are never added together") + "\n")
	sb.WriteString("\n")

	sb.WriteString(voiceLine("done", fmt.Sprintf(
		"Finished something genuinely useful: %s -- %s.",
		report.VerifiedUsefulSuccess.String(),
		improvementReportSuccessClassification(report.VerifiedUsefulSuccess),
	)) + "\n")
	sb.WriteString("\n")

	sb.WriteString(voiceLine("warning", fmt.Sprintf(
		"A person had to step in to prevent something going wrong: %s -- %s.",
		report.PreventableInterventions.String(),
		improvementReportInterventionClassification(report.PreventableInterventions),
	)) + "\n")
	sb.WriteString("\n")

	sb.WriteString(voiceLine("status", fmt.Sprintf("%d run(s) counted in this reporting window.", report.TotalEpisodes)) + "\n")
	if len(report.UnclassifiedEpisodes) > 0 {
		sb.WriteString(voiceLine("warning", fmt.Sprintf(
			"%d run(s) matched neither figure and are counted honestly as unclassified rather than guessed at: %s.",
			len(report.UnclassifiedEpisodes), strings.Join(report.UnclassifiedEpisodes, ", "),
		)) + "\n")
	}

	if len(report.HardFailures) == 0 {
		sb.WriteString(voiceLine("done", "No hard checks failed in this window.") + "\n")
	} else {
		sb.WriteString(voiceLine("failed", fmt.Sprintf("%d hard check failure(s) -- these are never excluded, downgraded or offset by any success:", len(report.HardFailures))) + "\n")
		for _, f := range report.HardFailures {
			line := fmt.Sprintf("Run %s: %s failed.", f.EpisodeID, f.Gate)
			if f.Consequence != "" {
				line += " " + f.Consequence
			}
			sb.WriteString(voiceLine("failed", line) + "\n")
		}
	}

	return sb.String()
}
