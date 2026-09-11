package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/storage"
)

// colonyLiveTickerLimit bounds the most-recent-events tail carried on a
// snapshot for cockpit rendering.
const colonyLiveTickerLimit = 20

// colonyLiveEventBusFile is the persisted JSONL filename events.Bus writes
// live.* (and every other) event to, relative to the store's base path.
// Matches events.DefaultConfig().JSONLFile.
const colonyLiveEventBusFile = "event-bus.jsonl"

// colonyLiveWorkerRow is one worker's current picture inside a live
// episode: identity, lineage, workspace, lens, and the question/findings it
// has surfaced so far. Ordered by first-seen (worker.started) sequence.
type colonyLiveWorkerRow struct {
	WorkerID       string   `json:"worker_id"`
	ParentWorkerID string   `json:"parent_worker_id,omitempty"`
	Caste          string   `json:"caste"`
	WorkerName     string   `json:"worker_name"`
	Wave           int      `json:"wave,omitempty"`
	Workspace      string   `json:"workspace,omitempty"`
	Lens           string   `json:"lens,omitempty"`
	Question       string   `json:"question,omitempty"`
	Findings       []string `json:"findings,omitempty"`
	Status         string   `json:"status,omitempty"`

	// StartedAt is this worker's own worker.started event timestamp -- the
	// only source an "elapsed time" figure for this specific worker may
	// ever be computed from (never wall-clock "now").
	StartedAt string `json:"started_at,omitempty"`
	// Finished is true once a worker.finished event has been folded for
	// this worker. A row with Finished == false has no terminal event of
	// its own, whatever Status currently reads (worker.progress and
	// question/finding updates also write Status without ever finishing
	// the worker) -- this is the one field the "never call an unfinished
	// worker finished" rule (202-06 Task 3) reads.
	Finished bool `json:"finished,omitempty"`
	// InterruptedReason is set only when a still-open worker (Finished ==
	// false) is reclassified as interrupted because the spawn run that
	// owned it reached a terminal status without ever emitting this
	// worker's own worker.finished event. It names that run's own recorded
	// terminal status -- never a guess, never "completed".
	InterruptedReason string `json:"interrupted_reason,omitempty"`
}

// colonyLiveTickerEntry is one entry in the bounded most-recent-events tail
// carried on the snapshot for cockpit rendering.
type colonyLiveTickerEntry struct {
	Topic     string `json:"topic"`
	Timestamp string `json:"timestamp"`
	WorkerID  string `json:"worker_id,omitempty"`
	Caste     string `json:"caste,omitempty"`
	Status    string `json:"status,omitempty"`
}

// colonyLiveSnapshot is the pure-replay projection of a live episode's
// persisted events. It is the only shape any renderer -- live or replay --
// reads a live-colony picture through; replaying the same persisted file
// twice must always produce a byte-identical snapshot, and driving the
// projection from the persisted event file alone, with no live process,
// must reproduce the same snapshot field for field as the live path that
// emitted the events.
type colonyLiveSnapshot struct {
	SchemaVersion    string                  `json:"schema_version"`
	EpisodeID        string                  `json:"episode_id"`
	EpisodeKind      string                  `json:"episode_kind,omitempty"`
	Open             bool                    `json:"open"`
	StartedAt        string                  `json:"started_at,omitempty"`
	ElapsedSeconds   float64                 `json:"elapsed_seconds,omitempty"`
	Wave             int                     `json:"wave,omitempty"`
	Workers          []colonyLiveWorkerRow   `json:"workers,omitempty"`
	Confidence       float64                 `json:"confidence,omitempty"`
	TargetConfidence float64                 `json:"target_confidence,omitempty"`
	Contradictions   []string                `json:"contradictions,omitempty"`
	Signals          []string                `json:"signals,omitempty"`
	RecoveryState    string                  `json:"recovery_state,omitempty"`
	Ticker           []colonyLiveTickerEntry `json:"ticker,omitempty"`

	// SkippedEventCount / SkippedSchemaVersions record events whose schema
	// version this reducer does not recognize -- skipped, never fatal.
	SkippedEventCount     int      `json:"skipped_event_count,omitempty"`
	SkippedSchemaVersions []string `json:"skipped_schema_versions,omitempty"`

	LastEventID   string `json:"last_event_id,omitempty"`
	LastTimestamp string `json:"last_timestamp,omitempty"`
	LastSequence  int64  `json:"last_sequence,omitempty"`
}

// colonyLiveResumeCursor identifies the last persisted event a prior
// replay already folded, letting a restarted process continue folding
// forward through only the events persisted since, without re-reading or
// double-counting anything it already projected. The three fields together
// match sortColonyLiveEntries' own ordering exactly, so "after the cursor"
// means precisely the same thing here as it does during a full replay's
// sort.
type colonyLiveResumeCursor struct {
	Timestamp string
	Sequence  int64
	EventID   string
}

// colonyLiveEntryAfterCursor reports whether entry sorts strictly after
// cursor under the same (timestamp, sequence, event ID) ordering
// sortColonyLiveEntries uses. A zero-value cursor (no EventID) means
// "nothing has been folded yet" -- every entry is after it.
func colonyLiveEntryAfterCursor(entry colonyLiveDecodedEvent, cursor colonyLiveResumeCursor) bool {
	if cursor.EventID == "" {
		return true
	}
	if entry.event.Timestamp != cursor.Timestamp {
		return entry.event.Timestamp > cursor.Timestamp
	}
	if entry.payload.Sequence != cursor.Sequence {
		return entry.payload.Sequence > cursor.Sequence
	}
	return entry.event.ID > cursor.EventID
}

// colonyLiveDecodedEvent pairs a raw persisted event with its decoded
// ColonyLivePayload, used internally while sorting and folding.
type colonyLiveDecodedEvent struct {
	event   events.Event
	payload events.ColonyLivePayload
}

// replayColonyLiveSnapshot is the pure reducer that folds persisted live
// events into a live snapshot. It is the only path any renderer -- live or
// replay -- may read a live-colony picture through.
//
// episodeID scopes the replay to a single episode; an empty episodeID
// resolves to whichever episode the chronologically-latest persisted event
// belongs to. An absent or empty persisted file, or a nil store, yields an
// empty snapshot and a nil error -- never an error.
func replayColonyLiveSnapshot(ctx context.Context, s *storage.Store, episodeID string, since time.Time) (colonyLiveSnapshot, error) {
	_ = ctx
	snapshot := colonyLiveSnapshot{SchemaVersion: events.ColonyLiveSchemaVersion, EpisodeID: episodeID}
	if s == nil {
		return snapshot, nil
	}

	raw := readColonyLiveEventsRaw(s, since)
	if len(raw) == 0 {
		return snapshot, nil
	}

	decoded := decodeColonyLiveEvents(raw)
	sortColonyLiveEntries(decoded)

	if episodeID == "" && len(decoded) > 0 {
		episodeID = decoded[len(decoded)-1].payload.EpisodeID
		snapshot.EpisodeID = episodeID
	}
	if episodeID != "" {
		decoded = filterColonyLiveEntriesByEpisode(decoded, episodeID)
	}

	return foldColonyLiveEvents(snapshot, decoded), nil
}

// replayColonyLiveSnapshotResume continues folding from a previously
// folded snapshot, forward through only the events persisted since that
// snapshot's own last-folded event -- the mechanism a restarted watch
// process uses to pick back up after a simulated process restart. Folding
// forward from previous's cursor must always produce the exact same result
// as folding the whole persisted file from the beginning; no event
// previous already folded is ever counted twice.
func replayColonyLiveSnapshotResume(ctx context.Context, s *storage.Store, episodeID string, previous colonyLiveSnapshot) (colonyLiveSnapshot, error) {
	_ = ctx
	if s == nil {
		return previous, nil
	}

	raw := readColonyLiveEventsRaw(s, time.Time{})
	if len(raw) == 0 {
		return previous, nil
	}

	decoded := decodeColonyLiveEvents(raw)
	sortColonyLiveEntries(decoded)
	if episodeID != "" {
		decoded = filterColonyLiveEntriesByEpisode(decoded, episodeID)
	}

	cursor := colonyLiveResumeCursor{
		Timestamp: previous.LastTimestamp,
		Sequence:  previous.LastSequence,
		EventID:   previous.LastEventID,
	}
	forward := make([]colonyLiveDecodedEvent, 0, len(decoded))
	for _, entry := range decoded {
		if colonyLiveEntryAfterCursor(entry, cursor) {
			forward = append(forward, entry)
		}
	}
	if len(forward) == 0 {
		return previous, nil
	}

	return foldColonyLiveEvents(previous, forward), nil
}

// readColonyLiveEventsRaw reads the persisted live-event JSONL file
// directly via os.ReadFile, deliberately bypassing storage.Store's locking
// read path (store.ReadJSONL / events.Bus.Query). Every code path that
// resolves the watch mode runs on EVERY `aether watch` invocation,
// including the plain idle path -- taking a lock there would create a
// lock-file side effect on a command that must stay strictly read-only
// (TestWatchIdle199ReadOnly). This mirrors the established discipline in
// cmd/lifecycle_facts.go's readLifecycleJSON/readLifecycleActors, which
// exist for exactly the same reason.
func readColonyLiveEventsRaw(s *storage.Store, since time.Time) []events.Event {
	if s == nil {
		return nil
	}
	path := filepath.Join(s.BasePath(), colonyLiveEventBusFile)
	data, err := os.ReadFile(path)
	if err != nil {
		// Absent file (no episode ever recorded) -- an empty, not an
		// error, result.
		return nil
	}

	now := events.FormatTimestamp(time.Now().UTC())
	var sinceStamp string
	if !since.IsZero() {
		sinceStamp = events.FormatTimestamp(since.UTC())
	}

	lines := bytes.Split(data, []byte{'\n'})
	result := make([]events.Event, 0, len(lines))
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		var evt events.Event
		if err := json.Unmarshal(trimmed, &evt); err != nil {
			continue // malformed line -- skip, never abort the replay
		}
		if !strings.HasPrefix(evt.Topic, "live.") {
			continue
		}
		if evt.ExpiresAt != "" && evt.ExpiresAt <= now {
			continue // expired
		}
		if sinceStamp != "" && evt.Timestamp < sinceStamp {
			continue
		}
		result = append(result, evt)
	}
	return result
}

func decodeColonyLiveEvents(raw []events.Event) []colonyLiveDecodedEvent {
	decoded := make([]colonyLiveDecodedEvent, 0, len(raw))
	for _, evt := range raw {
		var payload events.ColonyLivePayload
		if err := json.Unmarshal(evt.Payload, &payload); err != nil {
			// Not a valid ColonyLivePayload at all (malformed JSON) --
			// skip rather than abort the surrounding replay.
			continue
		}
		decoded = append(decoded, colonyLiveDecodedEvent{event: evt, payload: payload})
	}
	return decoded
}

func filterColonyLiveEntriesByEpisode(entries []colonyLiveDecodedEvent, episodeID string) []colonyLiveDecodedEvent {
	filtered := make([]colonyLiveDecodedEvent, 0, len(entries))
	for _, entry := range entries {
		if entry.payload.EpisodeID == episodeID {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// sortColonyLiveEntries sorts strictly by timestamp, then sequence number,
// then event identifier -- so two events sharing an identical
// (second-precision) timestamp always replay in one specified order, and
// repeated replays of the same file produce identical snapshots.
func sortColonyLiveEntries(entries []colonyLiveDecodedEvent) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.event.Timestamp != b.event.Timestamp {
			return a.event.Timestamp < b.event.Timestamp
		}
		if a.payload.Sequence != b.payload.Sequence {
			return a.payload.Sequence < b.payload.Sequence
		}
		return a.event.ID < b.event.ID
	})
}

// foldColonyLiveEvents folds entries into snapshot. snapshot may be a
// fresh zero value (a full replay from the beginning) or a previously
// folded snapshot (a resume) -- in the resume case, its existing workers
// and open/closed balance are seeded as the starting accumulator so
// forward-only entries update the same rows a full replay would, rather
// than starting over.
func foldColonyLiveEvents(snapshot colonyLiveSnapshot, entries []colonyLiveDecodedEvent) colonyLiveSnapshot {
	workerIndex := make(map[string]int, len(snapshot.Workers))
	for i, w := range snapshot.Workers {
		workerIndex[w.WorkerID] = i
	}
	openBalance := 0
	if snapshot.Open {
		openBalance = 1
	}

	for _, entry := range entries {
		evt, payload := entry.event, entry.payload

		if payload.SchemaVersion != "" && payload.SchemaVersion != events.ColonyLiveSchemaVersion {
			snapshot.SkippedEventCount++
			if !containsString(snapshot.SkippedSchemaVersions, payload.SchemaVersion) {
				snapshot.SkippedSchemaVersions = append(snapshot.SkippedSchemaVersions, payload.SchemaVersion)
			}
			continue
		}

		snapshot.LastEventID = evt.ID
		snapshot.LastTimestamp = evt.Timestamp
		snapshot.LastSequence = payload.Sequence
		if snapshot.EpisodeKind == "" && payload.EpisodeKind != "" {
			snapshot.EpisodeKind = payload.EpisodeKind
		}

		switch evt.Topic {
		case events.LiveTopicEpisodeStarted, events.LiveTopicWaveStarted:
			openBalance++
			if snapshot.StartedAt == "" {
				snapshot.StartedAt = evt.Timestamp
			}
			if payload.Wave > 0 {
				snapshot.Wave = payload.Wave
			}
		case events.LiveTopicEpisodeEnded, events.LiveTopicWaveEnded:
			if openBalance > 0 {
				openBalance--
			}
			if payload.ElapsedSeconds > 0 {
				snapshot.ElapsedSeconds = payload.ElapsedSeconds
			}
		case events.LiveTopicWorkerStarted:
			key := firstNonEmpty(payload.WorkerID, payload.WorkerName)
			row := colonyLiveWorkerRow{
				WorkerID:       key,
				ParentWorkerID: payload.ParentWorkerID,
				Caste:          payload.Caste,
				WorkerName:     payload.WorkerName,
				Wave:           payload.Wave,
				Workspace:      payload.Workspace,
				Lens:           payload.Lens,
				Question:       payload.Question,
				Status:         firstNonEmpty(payload.Status, "active"),
				StartedAt:      evt.Timestamp,
			}
			if payload.Wave > 0 {
				snapshot.Wave = payload.Wave
			}
			if idx, ok := workerIndex[key]; ok {
				snapshot.Workers[idx] = row
			} else {
				workerIndex[key] = len(snapshot.Workers)
				snapshot.Workers = append(snapshot.Workers, row)
			}
		case events.LiveTopicWorkerProgress, events.LiveTopicWorkerFinished, events.LiveTopicQuestionChanged, events.LiveTopicFindingRecorded:
			key := firstNonEmpty(payload.WorkerID, payload.WorkerName)
			if idx, ok := workerIndex[key]; ok {
				if payload.Question != "" {
					snapshot.Workers[idx].Question = payload.Question
				}
				if len(payload.Findings) > 0 {
					snapshot.Workers[idx].Findings = append(snapshot.Workers[idx].Findings, payload.Findings...)
				}
				if payload.Status != "" {
					snapshot.Workers[idx].Status = payload.Status
				}
				if evt.Topic == events.LiveTopicWorkerFinished {
					snapshot.Workers[idx].Finished = true
					snapshot.Workers[idx].InterruptedReason = ""
				}
			}
		case events.LiveTopicConfidenceChanged:
			snapshot.Confidence = payload.Confidence
			snapshot.TargetConfidence = payload.TargetConfidence
		case events.LiveTopicContradictionFound:
			snapshot.Contradictions = append(snapshot.Contradictions, payload.Contradictions...)
		case events.LiveTopicSignalConsulted:
			snapshot.Signals = append(snapshot.Signals, payload.Signals...)
		case events.LiveTopicRecoveryChanged:
			snapshot.RecoveryState = payload.RecoveryState
		}

		snapshot.Ticker = append(snapshot.Ticker, colonyLiveTickerEntry{
			Topic:     evt.Topic,
			Timestamp: evt.Timestamp,
			WorkerID:  firstNonEmpty(payload.WorkerID, payload.WorkerName),
			Caste:     payload.Caste,
			Status:    payload.Status,
		})
	}

	if len(snapshot.Ticker) > colonyLiveTickerLimit {
		snapshot.Ticker = snapshot.Ticker[len(snapshot.Ticker)-colonyLiveTickerLimit:]
	}

	// Open is derived from a start/end balance across episode- and
	// wave-level boundary events, never from wall-clock inference: a
	// started boundary with no matching ended boundary yet leaves the
	// balance positive, so the episode/wave is reported open.
	snapshot.Open = openBalance > 0

	if snapshot.Open && snapshot.StartedAt != "" {
		if started, err := time.Parse(time.RFC3339, snapshot.StartedAt); err == nil {
			if last, err := time.Parse(time.RFC3339, snapshot.LastTimestamp); err == nil {
				snapshot.ElapsedSeconds = last.Sub(started).Seconds()
			}
		}
	}

	return snapshot
}
