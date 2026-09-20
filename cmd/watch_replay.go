package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/storage"
)

// colonyLiveEpisodeStartEntry returns the entry that marks episodeID's own
// beginning, out of entries (already decoded and sorted chronologically by
// sortColonyLiveEntries). It prefers the earliest boundary-start event
// (live.episode.started, or -- for a lane like Swarm's investigation wave
// that never opens an episode-level boundary -- the earliest
// live.wave.started) and falls back to the episode's own earliest recorded
// event of any kind for a lane (Oracle) that emits neither boundary. ok is
// false only when the episode has no recorded events at all.
func colonyLiveEpisodeStartEntry(entries []colonyLiveDecodedEvent, episodeID string) (colonyLiveDecodedEvent, bool) {
	var boundary, any colonyLiveDecodedEvent
	haveBoundary, haveAny := false, false
	for _, entry := range entries {
		if entry.payload.EpisodeID != episodeID {
			continue
		}
		if !haveAny {
			any = entry
			haveAny = true
		}
		if !haveBoundary && (entry.event.Topic == events.LiveTopicEpisodeStarted || entry.event.Topic == events.LiveTopicWaveStarted) {
			boundary = entry
			haveBoundary = true
		}
	}
	if haveBoundary {
		return boundary, true
	}
	return any, haveAny
}

// mostRecentlyStartedLiveEpisode finds the episode whose own start event
// (colonyLiveEpisodeStartEntry) is the chronologically LATEST across every
// persisted episode -- ties broken by that event's own sequence number,
// then by its event ID, reusing the exact (timestamp, sequence, event ID)
// total order colonyLiveEntryAfterCursor already defines for resuming a
// replay. This is deliberately independent of which episode the single
// most-recent EVENT belongs to: an older episode can still be receiving
// events after a newer one has already begun, and "what just ran" means
// the newest EPISODE, not the newest event. latestLiveEpisodeID
// (cmd/watch_live.go) shares this exact selection logic through
// latestStartedLiveEpisodeAmong below -- it additionally prefers the
// subset of episodes that are still open, falling back to this same
// "latest started, among all episodes" answer with nothing open, so the
// live and replay branches can never name different episodes when nothing
// is running. Returns "" when no live event has ever been recorded.
func mostRecentlyStartedLiveEpisode(s *storage.Store) string {
	raw := readColonyLiveEventsRaw(s, time.Time{})
	if len(raw) == 0 {
		return ""
	}
	decoded := decodeColonyLiveEvents(raw)
	if len(decoded) == 0 {
		return ""
	}
	sortColonyLiveEntries(decoded)

	seen := map[string]bool{}
	episodeIDs := make([]string, 0, len(decoded))
	for _, entry := range decoded {
		id := entry.payload.EpisodeID
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		episodeIDs = append(episodeIDs, id)
	}

	return latestStartedLiveEpisodeAmong(decoded, episodeIDs)
}

// latestStartedLiveEpisodeAmong returns whichever episode ID in episodeIDs
// has the chronologically LATEST own start event
// (colonyLiveEpisodeStartEntry), ties broken by that event's own sequence
// number then its event ID -- the exact (timestamp, sequence, event ID)
// total order colonyLiveEntryAfterCursor already defines. entries must
// already be decoded and sorted (sortColonyLiveEntries). Returns "" when
// none of episodeIDs has any recorded start entry at all. This is the one
// shared "latest started episode" selection both
// mostRecentlyStartedLiveEpisode (the replay branch, called over every
// episode) and latestLiveEpisodeID (the live branch, called over the open
// subset and, as its fallback, every episode) call -- there is exactly one
// implementation of this rule in the package.
func latestStartedLiveEpisodeAmong(entries []colonyLiveDecodedEvent, episodeIDs []string) string {
	best := ""
	var bestEntry colonyLiveDecodedEvent
	for _, id := range episodeIDs {
		entry, ok := colonyLiveEpisodeStartEntry(entries, id)
		if !ok {
			continue
		}
		cursor := colonyLiveResumeCursor{
			Timestamp: bestEntry.event.Timestamp,
			Sequence:  bestEntry.payload.Sequence,
			EventID:   bestEntry.event.ID,
		}
		if best == "" || colonyLiveEntryAfterCursor(entry, cursor) {
			best = id
			bestEntry = entry
		}
	}
	return best
}

// colonyReplayOutcomeStatus is a closed episode's own terminal status: the
// most recent non-empty Status carried by its (already episode-scoped and
// bounded) ticker, which for a normally-closed episode is its own
// episode.ended/wave.ended status. Falls back to "completed" only when the
// episode carries no status at all (a lane with no terminal boundary event
// of its own) -- never invented beyond that.
func colonyReplayOutcomeStatus(snapshot colonyLiveSnapshot) string {
	for i := len(snapshot.Ticker) - 1; i >= 0; i-- {
		if status := strings.TrimSpace(snapshot.Ticker[i].Status); status != "" {
			return status
		}
	}
	return "completed"
}

// colonyLiveLastTransition names an interrupted episode's own last recorded
// event: a plain-English description (colonyLiveTickerDescription, the same
// vocabulary the ticker already renders -- never a raw topic string) and
// the timestamp it happened at.
func colonyLiveLastTransition(snapshot colonyLiveSnapshot) (description, at string) {
	if len(snapshot.Ticker) == 0 {
		return "", ""
	}
	last := snapshot.Ticker[len(snapshot.Ticker)-1]
	return colonyLiveTickerDescription(last), last.Timestamp
}

// buildReplayWatchResult replays the persisted live events, selects the
// most recently STARTED episode (mostRecentlyStartedLiveEpisode), and
// produces the replay-backed watch result: the episode's own kind and
// identifier, how many workers ran, its terminal outcome (or, for an
// episode with a start event and no matching end, that it was interrupted
// and what its own last recorded transition was), its elapsed time (read
// from the replayed snapshot's own start/end events, never wall-clock
// "now"), its cost block (loadSpendLedgersForPhase /
// renderSpendCostLineFromLedgers -- the one existing spend authority, never
// recomputed here), and the single next command (the lifecycle
// projection's own next action, exactly as buildIdleWatchResult already
// takes it -- never chosen by this function).
//
// Resolving this result performs no write of any kind: every read here
// (readColonyLiveEventsRaw, loadSpendLedgersForPhase, loadLifecycleFacts)
// is a plain, already-proven read-only path.
func buildReplayWatchResult(ctx context.Context, root string, s *storage.Store, now time.Time) map[string]interface{} {
	captured := now.UTC().Format(time.RFC3339Nano)
	episodeID := mostRecentlyStartedLiveEpisode(s)
	if episodeID == "" {
		return map[string]interface{}{
			"schema_version":  LifecycleResultSchemaVersion,
			"mode":            "replay_watch",
			"command":         "watch",
			"live_capability": "recorded",
			"captured_at":     captured,
		}
	}

	snapshot, err := replayColonyLiveSnapshot(ctx, s, episodeID, time.Time{})
	if err != nil {
		snapshot = colonyLiveSnapshot{EpisodeID: episodeID}
	}

	interrupted := snapshot.Open
	outcome, lastTransition, lastTransitionAt := "", "", ""
	if interrupted {
		lastTransition, lastTransitionAt = colonyLiveLastTransition(snapshot)
	} else {
		outcome = colonyReplayOutcomeStatus(snapshot)
	}

	state, _ := readColonyStateWithoutWriting()
	ledgers, _ := loadSpendLedgersForPhase(state.CurrentPhase)
	costBlock := renderSpendCostLineFromLedgers(ledgers)

	facts, factsErr := loadLifecycleFacts(root, s, now)
	if factsErr != nil {
		facts = unavailableLifecycleFacts(root, now, factsErr.Error())
	}
	projection := projectLifecycle(facts, LifecycleViewCompact, detectPlatform())
	nextCommand := lifecycleStatusActionCommand(projection.NextAction)

	tickerBlock := renderColonyLiveTicker(snapshot.Ticker)

	return map[string]interface{}{
		"schema_version":      LifecycleResultSchemaVersion,
		"mode":                "replay_watch",
		"command":             "watch",
		"live_capability":     "recorded",
		"episode_id":          snapshot.EpisodeID,
		"episode_kind":        snapshot.EpisodeKind,
		"worker_count":        len(snapshot.Workers),
		"interrupted":         interrupted,
		"outcome":             outcome,
		"last_transition":     lastTransition,
		"last_transition_at":  lastTransitionAt,
		"started_at":          snapshot.StartedAt,
		"elapsed_seconds":     snapshot.ElapsedSeconds,
		"next":                nextCommand,
		"ticker_block":        tickerBlock,
		"cost_block":          costBlock,
		"captured_at":         captured,
		"projection_revision": projection.ProjectionRevision,
	}
}

// renderReplayWatchVisual renders buildReplayWatchResult's map as the
// replay watch screen: an opening line stating plainly that this describes
// the last thing that ran and that nothing is running now, then the
// episode's identity, worker count, outcome (or interruption), elapsed
// time, the next suggested command, the episode's own recent-events ticker
// (reusing renderColonyLiveTicker -- never a second renderer), and the
// reported-cost block last.
func renderReplayWatchVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("watch"), "Watch"))
	b.WriteString(visualDividerStr())

	episodeID := stringValue(result["episode_id"])
	if episodeID == "" {
		b.WriteString("Nothing has been recorded yet, and nothing is running now.\n")
		return b.String()
	}

	b.WriteString("This is the last thing that ran -- nothing is running now.\n")
	fmt.Fprintf(&b, "Episode %s (%s)\n", episodeID, emptyFallback(stringValue(result["episode_kind"]), "unknown"))
	fmt.Fprintf(&b, "Workers: %d\n", intValue(result["worker_count"]))

	if boolValue(result["interrupted"]) {
		fmt.Fprintf(&b, "Outcome: interrupted -- last recorded: %s at %s\n",
			emptyFallback(stringValue(result["last_transition"]), "unknown"),
			emptyFallback(stringValue(result["last_transition_at"]), "unknown"))
	} else {
		fmt.Fprintf(&b, "Outcome: %s\n", emptyFallback(stringValue(result["outcome"]), "unknown"))
	}

	if startedAt := stringValue(result["started_at"]); startedAt != "" {
		fmt.Fprintf(&b, "Started: %s | Elapsed: %.0fs\n", startedAt, floatValue(result["elapsed_seconds"]))
	}

	if next := stringValue(result["next"]); next != "" {
		fmt.Fprintf(&b, "\nNext: %s\n", next)
	}
	b.WriteString("\n")

	b.WriteString(stringValue(result["ticker_block"]))
	b.WriteString(stringValue(result["cost_block"]))

	return b.String()
}
