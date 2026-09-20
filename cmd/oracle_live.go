package cmd

import (
	"strings"

	"github.com/calcosmic/Aether/pkg/events"
)

// Oracle's live-event emission (202-11, LIVE-06/CEC-05).
//
// oracle_loop.go already tracks everything a watcher would want to see --
// the round number, the round cap, the active question, the overall
// confidence, the open gaps and the contradictions -- at oracleStateFile's
// own existing mutation points (202-CLASSIC-SYNTHESIS.md's SYN-202-11: "the
// real gap is visibility, not capability"). Nothing here adds a new field
// to that state; every function below just reads what the loop already
// assigned and publishes it through the one live-event boundary
// (cmd/live_events.go's emitColonyLive), mirroring
// cmd/ceremony_emitter.go's emitOracleIteration/emitOraclePhaseTransition
// call-site shape and their error-tolerant pattern: a failed emit returns
// silently and never changes the research round's outcome -- emitColonyLive
// itself already guarantees that (nil store, empty topic, marshal failure,
// or bus-publish failure all no-op).
//
// Oracle has no per-worker dispatch of its own -- one round is one pass of
// the same research effort, not a new worker -- so a running round is
// represented as a single, repeatedly-replaced worker row (WorkerID
// "oracle", caste "oracle") rather than a new live-event shape. This lets
// cmd/watch_dashboard.go's existing generic worker-block and
// confidence/contradiction rendering show a running Oracle round with no
// second screen (202-11 Task 3).

// oracleLiveWorkerID is the stable synthetic worker identity every
// round-scoped live event carries, so the dashboard's existing per-worker
// row (never a new rendering path) tracks one continuously-updated "Oracle"
// row across the whole run.
const oracleLiveWorkerID = "oracle"

// oracleLiveEpisodeID derives a stable episode identifier from the one
// oracleStateFile field that is set once when a run begins and preserved
// unchanged across every `oracle iterate` resume: StartedAt. Nothing new is
// persisted to reach this -- StartedAt is already an existing field.
func oracleLiveEpisodeID(state oracleStateFile) string {
	started := strings.TrimSpace(state.StartedAt)
	if started == "" {
		return "oracle"
	}
	return "oracle-" + started
}

// emitOracleLiveRound marks one round beginning: the round number (Wave),
// its cap (RoundCap), the phase, the active question and the confidence as
// it stood before this round's answer -- every field read directly from
// state, which the loop has already updated for this round by the time
// this is called (the round increment and active-question assignment have
// both already happened).
func emitOracleLiveRound(state oracleStateFile) {
	emitColonyLive(events.LiveTopicWorkerStarted, events.ColonyLivePayload{
		EpisodeID:   oracleLiveEpisodeID(state),
		EpisodeKind: events.EpisodeKindOracle,
		Wave:        state.Iteration,
		RoundCap:    state.MaxIterations,
		WorkerID:    oracleLiveWorkerID,
		Caste:       "oracle",
		WorkerName:  "Oracle",
		Status:      strings.TrimSpace(state.Phase),
		Question:    strings.TrimSpace(state.ActiveQuestionText),
		Confidence:  float64(state.OverallConfidence),
	})
}

// emitOracleLiveConfidence marks a confidence change: previous, new and
// target. previous is the value the caller held before recomputing
// state.OverallConfidence -- state itself only ever carries the current
// value, so the previous figure must be handed in by the caller.
func emitOracleLiveConfidence(state oracleStateFile, previous int) {
	emitColonyLive(events.LiveTopicConfidenceChanged, events.ColonyLivePayload{
		EpisodeID:          oracleLiveEpisodeID(state),
		EpisodeKind:        events.EpisodeKindOracle,
		Confidence:         float64(state.OverallConfidence),
		TargetConfidence:   float64(state.TargetConfidence),
		PreviousConfidence: float64(previous),
	})
}

// emitOracleLiveContradiction announces one newly-observed contradiction.
// Call once per genuinely new entry -- a contradiction already recorded in
// state.Contradictions before the merge must not be announced again.
func emitOracleLiveContradiction(state oracleStateFile, contradiction string) {
	emitColonyLive(events.LiveTopicContradictionFound, events.ColonyLivePayload{
		EpisodeID:      oracleLiveEpisodeID(state),
		EpisodeKind:    events.EpisodeKindOracle,
		Contradictions: []string{contradiction},
	})
}

// emitOracleLiveGapTargeted announces one newly-added open gap -- the same
// "new entries only" discipline emitOracleLiveContradiction uses, at the
// state.OpenGaps merge immediately beside it.
func emitOracleLiveGapTargeted(state oracleStateFile, gap string) {
	emitColonyLive(events.LiveTopicGapTargeted, events.ColonyLivePayload{
		EpisodeID:   oracleLiveEpisodeID(state),
		EpisodeKind: events.EpisodeKindOracle,
		Findings:    []string{gap},
	})
}

// emitOracleLiveRoundEnded marks one round ending, carrying the confidence
// it ended at.
func emitOracleLiveRoundEnded(state oracleStateFile) {
	emitColonyLive(events.LiveTopicWorkerProgress, events.ColonyLivePayload{
		EpisodeID:   oracleLiveEpisodeID(state),
		EpisodeKind: events.EpisodeKindOracle,
		Wave:        state.Iteration,
		WorkerID:    oracleLiveWorkerID,
		Caste:       "oracle",
		WorkerName:  "Oracle",
		Status:      "round_ended",
		Confidence:  float64(state.OverallConfidence),
	})
}

// openOracleLiveEpisode opens Oracle's own episode boundary (202-17,
// CR-01/CEC-05/LIVE-06) -- the same emitColonyLiveEpisodeStarted /
// emitColonyLiveEpisodeEnded pair cmd/codex_build.go's and
// cmd/codex_continue.go's own lanes already use, so a genuinely iterating
// Oracle round can go positive in foldColonyLiveEvents's start/end balance
// like any other lane instead of never opening a boundary at all. The
// episode ID deliberately reuses oracleLiveEpisodeID(state), derived from
// state.StartedAt, which does not change across an `oracle iterate` resume
// of the same run -- so a resumed run rejoins the same episode rather than
// minting a second one every time the process restarts. The returned close
// function is intended to be deferred by the caller with the run's own
// terminal status (see oracleLiveTerminalStatus) so every return path closes
// the episode it opened.
func openOracleLiveEpisode(state oracleStateFile) (string, func(status string)) {
	episodeID := oracleLiveEpisodeID(state)
	emitColonyLiveEpisodeStarted(episodeID, events.EpisodeKindOracle)
	return episodeID, func(status string) {
		emitColonyLiveEpisodeEnded(episodeID, events.EpisodeKindOracle, status)
	}
}

// oracleLiveTerminalStatus derives the status an Oracle live episode closes
// with from the loop's own reported outcome -- never a status the run did
// not itself report. finalizeOracleLoop already writes a "status" value
// into every terminal result map it returns (e.g. "complete", "stopped",
// "blocked", "max_iterations_reached"); this reads that value back rather
// than re-deriving it. A non-nil error means the loop never reached
// finalizeOracleLoop at all (state/plan failed to load, invoker
// unavailable, or a mid-round write failed) and closes as "failed". A nil
// error with no readable "status" value should not happen in practice, but
// still closes as "interrupted" rather than leaving the episode open
// forever.
func oracleLiveTerminalStatus(result map[string]interface{}, err error) string {
	if err != nil {
		return "failed"
	}
	if result != nil {
		if status, ok := result["status"].(string); ok && strings.TrimSpace(status) != "" {
			return status
		}
	}
	return "interrupted"
}

// newOracleNotes returns the entries present in after but not before, in
// after's own order -- the "genuinely new" diff emitOracleLiveContradiction
// and emitOracleLiveGapTargeted's call sites use so an unchanged,
// already-recorded entry is never announced twice.
func newOracleNotes(before, after []string) []string {
	seen := make(map[string]bool, len(before))
	for _, v := range before {
		seen[v] = true
	}
	added := make([]string, 0)
	for _, v := range after {
		if seen[v] {
			continue
		}
		seen[v] = true
		added = append(added, v)
	}
	return added
}
