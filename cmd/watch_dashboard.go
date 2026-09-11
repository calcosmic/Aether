package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
)

// colonyLiveDrillSelector narrows renderColonyLiveDashboard's worker detail
// to one worker identity or one wave, leaving the header line unchanged.
// The zero value selects nothing -- the default view (current wave in
// depth, every other wave compressed to a counted line).
type colonyLiveDrillSelector struct {
	WorkerID string
	Wave     int
}

// parseColonyLiveDrillSelector reads the watch command's own positional
// argument (cobra.MaximumNArgs(1) on watchCmd): a bare positive integer
// selects a wave, any other non-empty value selects a worker identity by
// WorkerID or WorkerName (case-insensitive). No argument, more than one, or
// a blank argument selects nothing -- the default view.
func parseColonyLiveDrillSelector(args []string) colonyLiveDrillSelector {
	if len(args) != 1 {
		return colonyLiveDrillSelector{}
	}
	arg := strings.TrimSpace(args[0])
	if arg == "" {
		return colonyLiveDrillSelector{}
	}
	if wave, err := strconv.Atoi(arg); err == nil && wave > 0 {
		return colonyLiveDrillSelector{Wave: wave}
	}
	return colonyLiveDrillSelector{WorkerID: arg}
}

// matches reports whether a worker-identity drill selector names row.
func (d colonyLiveDrillSelector) matches(row colonyLiveWorkerRow) bool {
	return strings.EqualFold(row.WorkerID, d.WorkerID) || strings.EqualFold(row.WorkerName, d.WorkerID)
}

// colonyLiveWaveCount is one compressed-wave line: a wave number and how
// many workers it carries, rendered instead of that wave's own worker
// detail once it is no longer the focused wave.
type colonyLiveWaveCount struct {
	wave  int
	count int
}

// renderColonyLiveDashboard renders the D-01 live cockpit screen: one
// compact colony header line naming the project, phase, standing and
// episode; a stage marker naming the focused wave or worker; one detailed
// block per worker in the focused set; every other wave compressed to one
// counted line; confidence against target when the episode carries one;
// contradictions and signals consulted; recovery state; the event ticker;
// and the reported cost block last, read verbatim from
// renderSpendCostLineFromLedgers -- never recomputed here.
//
// Every field is read from snapshot (the pure replay projection) or from
// state/ledgers (durable records this function receives, never fetches
// itself). Rendering the same four inputs twice always produces
// byte-identical output: nothing here reads wall-clock time.
func renderColonyLiveDashboard(snapshot colonyLiveSnapshot, state colony.ColonyState, ledgers []spendLedger, drill colonyLiveDrillSelector) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("watch"), "Watch"))
	b.WriteString(visualDividerStr())
	b.WriteString(renderColonyLiveDashboardHeader(snapshot, state))

	if snapshot.StartedAt != "" {
		fmt.Fprintf(&b, "Started: %s | Elapsed: %.0fs\n", snapshot.StartedAt, snapshot.ElapsedSeconds)
	}
	b.WriteString("\n")

	focusWave := snapshot.Wave
	focusLabel := fmt.Sprintf("Wave %d", focusWave)
	switch {
	case drill.WorkerID != "":
		focusLabel = "Worker " + drill.WorkerID
	case drill.Wave > 0:
		focusWave = drill.Wave
		focusLabel = fmt.Sprintf("Wave %d", focusWave)
	}
	b.WriteString(renderStageMarker(focusLabel))

	inFocus, compressed := partitionColonyLiveWorkers(snapshot.Workers, focusWave, drill)
	if len(inFocus) == 0 {
		b.WriteString("No workers active in this wave yet.\n")
	} else {
		for _, w := range inFocus {
			b.WriteString(renderColonyLiveWorkerBlock(w, snapshot.LastTimestamp))
		}
	}

	for _, wave := range compressed {
		fmt.Fprintf(&b, "Wave %d: %d worker(s)\n", wave.wave, wave.count)
	}

	if snapshot.Confidence > 0 || snapshot.TargetConfidence > 0 {
		fmt.Fprintf(&b, "\nConfidence: %.2f / target %.2f\n", snapshot.Confidence, snapshot.TargetConfidence)
	}
	if len(snapshot.Contradictions) > 0 {
		b.WriteString("\nContradictions:\n")
		for _, c := range snapshot.Contradictions {
			fmt.Fprintf(&b, "  - %s\n", c)
		}
	}
	if len(snapshot.Signals) > 0 {
		b.WriteString("\nSignals consulted:\n")
		for _, sig := range snapshot.Signals {
			fmt.Fprintf(&b, "  - %s\n", sig)
		}
	}
	if snapshot.RecoveryState != "" {
		fmt.Fprintf(&b, "\nRecovery state: %s\n", snapshot.RecoveryState)
	}

	b.WriteString("\n")
	b.WriteString(renderColonyLiveTicker(snapshot.Ticker))

	b.WriteString(renderSpendCostLineFromLedgers(ledgers))

	return b.String()
}

// renderColonyLiveDashboardHeader is the one compact colony header line:
// project name, current phase, standing and episode identity. state and
// snapshot are both durable/replayed records -- nothing here is invented,
// and an absent colony name falls back to a plain-English placeholder
// rather than an empty cell.
func renderColonyLiveDashboardHeader(snapshot colonyLiveSnapshot, state colony.ColonyState) string {
	project := ""
	if state.ColonyName != nil {
		project = strings.TrimSpace(*state.ColonyName)
	}
	project = emptyFallback(project, "colony")
	standing := emptyFallback(string(state.State), "unknown")
	episode := fmt.Sprintf("%s (%s)", emptyFallback(snapshot.EpisodeID, "unknown"), emptyFallback(snapshot.EpisodeKind, "unknown"))
	return fmt.Sprintf("%s -- phase %d -- %s -- episode %s\n", project, state.CurrentPhase, standing, episode)
}

// partitionColonyLiveWorkers splits workers into the focused set (rendered
// in depth by the caller) and every other wave (rendered as one counted
// line each, ascending by wave number).
//
// A worker-identity drill selector focuses on exactly that one worker,
// wherever its own wave is, and compresses nothing -- no counted lines are
// returned in that case, matching "drill-down by worker identity renders
// that worker's full detail and leaves the header line unchanged" with no
// other wave noise added.
func partitionColonyLiveWorkers(workers []colonyLiveWorkerRow, focusWave int, drill colonyLiveDrillSelector) ([]colonyLiveWorkerRow, []colonyLiveWaveCount) {
	if drill.WorkerID != "" {
		for _, w := range workers {
			if drill.matches(w) {
				return []colonyLiveWorkerRow{w}, nil
			}
		}
		return nil, nil
	}

	inFocus := make([]colonyLiveWorkerRow, 0, len(workers))
	counts := make(map[int]int, len(workers))
	for _, w := range workers {
		if w.Wave == focusWave {
			inFocus = append(inFocus, w)
			continue
		}
		counts[w.Wave]++
	}

	waves := make([]int, 0, len(counts))
	for wave := range counts {
		waves = append(waves, wave)
	}
	sort.Ints(waves)

	compressed := make([]colonyLiveWaveCount, 0, len(waves))
	for _, wave := range waves {
		compressed = append(compressed, colonyLiveWaveCount{wave: wave, count: counts[wave]})
	}
	return inFocus, compressed
}

// renderColonyLiveWorkerBlock renders one worker's full detail: identity
// (caste glyph + label via casteIdentity, never a local copy), lineage,
// workspace, lens or current question, findings so far, and -- for a
// worker with no terminal event of its own -- elapsed time computed from
// its own start event against referenceTimestamp (the snapshot's own last
// replayed event, never wall-clock "now"). A field the row does not carry
// is omitted entirely.
func renderColonyLiveWorkerBlock(w colonyLiveWorkerRow, referenceTimestamp string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s -- wave %d -- %s\n", casteIdentity(w.Caste), emptyFallback(w.WorkerName, w.WorkerID), w.Wave, colonyLiveWorkerStatusLabel(w))
	if w.ParentWorkerID != "" {
		fmt.Fprintf(&b, "  lineage: parent %s\n", w.ParentWorkerID)
	}
	if w.Workspace != "" {
		fmt.Fprintf(&b, "  workspace: %s\n", w.Workspace)
	}
	if w.Lens != "" {
		fmt.Fprintf(&b, "  lens: %s\n", w.Lens)
	}
	if w.Question != "" {
		fmt.Fprintf(&b, "  question: %s\n", w.Question)
	}
	if !w.Finished {
		if elapsed, ok := colonyLiveWorkerElapsedSeconds(w, referenceTimestamp); ok {
			fmt.Fprintf(&b, "  elapsed: %.0fs\n", elapsed)
		}
	}
	if w.InterruptedReason != "" {
		fmt.Fprintf(&b, "  interrupted: the run that dispatched this worker ended (%s) before it reported its own result\n", w.InterruptedReason)
	}
	for _, finding := range w.Findings {
		fmt.Fprintf(&b, "  finding: %s\n", finding)
	}
	return b.String()
}

// colonyLiveWorkerStatusLabel is the one word this row uses for its own
// status. A worker reclassified as interrupted (InterruptedReason set) is
// ALWAYS labelled "interrupted" here, regardless of what word the owning
// run's own terminal status used -- a still-open worker is never called
// finished just because the run around it ended.
func colonyLiveWorkerStatusLabel(w colonyLiveWorkerRow) string {
	if w.InterruptedReason != "" {
		return "interrupted"
	}
	return emptyFallback(w.Status, "active")
}

// colonyLiveWorkerElapsedSeconds computes a worker's own elapsed time from
// its own recorded start event (StartedAt) against referenceTimestamp --
// the snapshot's own last replayed event timestamp, never time.Now(), so
// repeated renders of the same snapshot are always byte-identical. Returns
// false when either timestamp is absent or unparseable, or when the
// computed elapsed would be negative (a malformed or out-of-order fixture).
func colonyLiveWorkerElapsedSeconds(w colonyLiveWorkerRow, referenceTimestamp string) (float64, bool) {
	if w.StartedAt == "" || referenceTimestamp == "" {
		return 0, false
	}
	started, err := time.Parse(time.RFC3339, w.StartedAt)
	if err != nil {
		return 0, false
	}
	last, err := time.Parse(time.RFC3339, referenceTimestamp)
	if err != nil {
		return 0, false
	}
	elapsed := last.Sub(started).Seconds()
	if elapsed < 0 {
		return 0, false
	}
	return elapsed, true
}

// renderColonyLiveTicker renders the bounded most-recent-events tail
// (snapshot.Ticker, already bounded to colonyLiveTickerLimit entries by the
// reducer) as one line per entry, oldest first / newest last -- the exact
// order the reducer's own append-then-trim already leaves the slice in. An
// episode with fewer events than the bound renders exactly that many lines
// and no filler. Returns "" when the ticker is empty, so an idle/fresh
// episode's dashboard carries no empty section.
func renderColonyLiveTicker(ticker []colonyLiveTickerEntry) string {
	if len(ticker) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(renderStageMarker("Recent Activity"))
	for _, entry := range ticker {
		b.WriteString(renderColonyLiveTickerLine(entry))
	}
	return b.String()
}

// renderColonyLiveTickerLine renders one ticker entry: its timestamp, its
// caste glyph via casteEmoji when the event names a caste, and a
// plain-English description of the transition the event actually carried.
func renderColonyLiveTickerLine(entry colonyLiveTickerEntry) string {
	glyph := ""
	if entry.Caste != "" {
		glyph = casteEmoji(entry.Caste) + " "
	}
	return fmt.Sprintf("  %s %s%s\n", entry.Timestamp, glyph, colonyLiveTickerDescription(entry))
}

// colonyLiveTickerDescription maps one ticker entry's topic to a
// plain-English description of the transition it recorded -- never the raw
// dotted topic string, which is repository vocabulary, not ordinary
// English. Every declared live.* topic (events.ColonyLiveTopics()) has its
// own case; the default branch exists only as a non-jargon fallback for a
// topic this function has not been taught yet.
func colonyLiveTickerDescription(entry colonyLiveTickerEntry) string {
	who := emptyFallback(entry.WorkerID, "a worker")
	switch entry.Topic {
	case events.LiveTopicEpisodeStarted:
		return "the episode started"
	case events.LiveTopicEpisodeEnded:
		return "the episode ended (" + emptyFallback(entry.Status, "unknown") + ")"
	case events.LiveTopicWaveStarted:
		return "a wave started"
	case events.LiveTopicWaveEnded:
		return "a wave ended (" + emptyFallback(entry.Status, "unknown") + ")"
	case events.LiveTopicWorkerStarted:
		return who + " started"
	case events.LiveTopicWorkerProgress:
		return who + " is working"
	case events.LiveTopicWorkerFinished:
		return who + " finished (" + emptyFallback(entry.Status, "unknown") + ")"
	case events.LiveTopicQuestionChanged:
		return who + "'s question changed"
	case events.LiveTopicConfidenceChanged:
		return "confidence moved"
	case events.LiveTopicContradictionFound:
		return "a contradiction appeared"
	case events.LiveTopicFindingRecorded:
		return who + " recorded a finding"
	case events.LiveTopicSignalConsulted:
		return "a signal was consulted"
	case events.LiveTopicCheckStarted:
		return "a check started"
	case events.LiveTopicCheckPassed:
		return "a check passed"
	case events.LiveTopicCheckFailed:
		return "a check failed"
	case events.LiveTopicRecoveryChanged:
		return "recovery state changed (" + emptyFallback(entry.Status, "unknown") + ")"
	default:
		return "activity recorded"
	}
}
