package events

import "encoding/json"

// Live-colony topic vocabulary (live/v1). This is the one versioned typed
// event model every lifecycle lane -- build, continue, plan, swarm, oracle,
// recovery -- publishes through and every renderer (aether watch) reads
// through. See cmd/live_events.go's emitColonyLive for the single emission
// boundary and cmd/live_projection.go's replayColonyLiveSnapshot for the
// single replay reducer. Do not build a second event bus, revive the
// retired cmd/event_types.go trio, or add a topic here without also adding
// it to ColonyLiveTopics().
const (
	LiveTopicEpisodeStarted     = "live.episode.started"
	LiveTopicEpisodeEnded       = "live.episode.ended"
	LiveTopicWaveStarted        = "live.wave.started"
	LiveTopicWaveEnded          = "live.wave.ended"
	LiveTopicWorkerStarted      = "live.worker.started"
	LiveTopicWorkerProgress     = "live.worker.progress"
	LiveTopicWorkerFinished     = "live.worker.finished"
	LiveTopicQuestionChanged    = "live.question.changed"
	LiveTopicConfidenceChanged  = "live.confidence.changed"
	LiveTopicContradictionFound = "live.contradiction.found"
	LiveTopicFindingRecorded    = "live.finding.recorded"
	LiveTopicSignalConsulted    = "live.signal.consulted"
	LiveTopicCheckStarted       = "live.check.started"
	LiveTopicCheckPassed        = "live.check.passed"
	LiveTopicCheckFailed        = "live.check.failed"
	LiveTopicRecoveryChanged    = "live.recovery.changed"
)

// Episode-kind vocabulary. EpisodeKind on ColonyLivePayload is a free-form
// string field, but every lifecycle lane that opens an episode (202-03) uses
// one of these declared values, and cmd/live_lane_coverage_test.go's
// TestEveryLifecycleLaneEmitsLiveEvents derives its lane inventory from
// ColonyLiveEpisodeKinds() rather than a hand-typed list, so a kind declared
// here with no lane driving it through a real public entry point fails that
// test by name.
const (
	EpisodeKindSwarm    = "swarm"
	EpisodeKindBuild    = "build"
	EpisodeKindContinue = "continue"
	EpisodeKindPlan     = "plan"
	EpisodeKindRecovery = "recovery"
)

// ColonyLiveEpisodeKinds returns every declared episode-kind constant. A
// kind added to the const block above must also be added here, mirroring
// ColonyLiveTopics()'s own completeness contract.
func ColonyLiveEpisodeKinds() []string {
	return []string{
		EpisodeKindSwarm,
		EpisodeKindBuild,
		EpisodeKindContinue,
		EpisodeKindPlan,
		EpisodeKindRecovery,
	}
}

// ColonyLiveSchemaVersion is the current wire-shape version of
// ColonyLivePayload. Every event this model publishes carries this value in
// its SchemaVersion field so a future incompatible change can be detected
// and named by the replay reducer -- an unrecognized version is skipped with
// a note, never a fatal replay error.
const ColonyLiveSchemaVersion = "live/v1"

// ColonyLivePayload is the one versioned typed live-colony event shape. It
// is the only model any live renderer reads, and the only payload
// emitColonyLive (cmd/live_events.go) is permitted to publish live.* topics
// with.
//
// No field on this payload may ever carry a currency amount. Reported cost
// is read exclusively from the existing spend ledger authority
// (renderSpendCostLine / loadSpendLedgersForPhase) -- see
// 202-CLASSIC-SYNTHESIS.md's cost/spend authority ruling. Adding a
// currency-shaped field here would reopen the exact "unread money field"
// hazard Phase 196 already fixed once.
type ColonyLivePayload struct {
	SchemaVersion    string   `json:"schema_version"`
	EpisodeID        string   `json:"episode_id"`
	EpisodeKind      string   `json:"episode_kind,omitempty"`
	Sequence         int64    `json:"sequence"`
	Wave             int      `json:"wave,omitempty"`
	WorkerID         string   `json:"worker_id,omitempty"`
	ParentWorkerID   string   `json:"parent_worker_id,omitempty"`
	Caste            string   `json:"caste,omitempty"`
	WorkerName       string   `json:"worker_name,omitempty"`
	Workspace        string   `json:"workspace,omitempty"`
	Lens             string   `json:"lens,omitempty"`
	Question         string   `json:"question,omitempty"`
	Confidence       float64  `json:"confidence,omitempty"`
	TargetConfidence float64  `json:"target_confidence,omitempty"`
	Contradictions   []string `json:"contradictions,omitempty"`
	Findings         []string `json:"findings,omitempty"`
	Signals          []string `json:"signals,omitempty"`
	CheckName        string   `json:"check_name,omitempty"`
	RecoveryState    string   `json:"recovery_state,omitempty"`
	Status           string   `json:"status,omitempty"`
	ElapsedSeconds   float64  `json:"elapsed_seconds,omitempty"`
}

// RawMessage marshals the payload for events.Bus.Publish, mirroring
// CeremonyPayload.RawMessage.
func (p ColonyLivePayload) RawMessage() (json.RawMessage, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

// ColonyLiveTopics returns every declared live.* topic constant. A topic
// added to the const block above must also be added here -- a test in
// colony_live_test.go asserts the two counts agree so the vocabulary can
// never silently drift.
func ColonyLiveTopics() []string {
	return []string{
		LiveTopicEpisodeStarted,
		LiveTopicEpisodeEnded,
		LiveTopicWaveStarted,
		LiveTopicWaveEnded,
		LiveTopicWorkerStarted,
		LiveTopicWorkerProgress,
		LiveTopicWorkerFinished,
		LiveTopicQuestionChanged,
		LiveTopicConfidenceChanged,
		LiveTopicContradictionFound,
		LiveTopicFindingRecorded,
		LiveTopicSignalConsulted,
		LiveTopicCheckStarted,
		LiveTopicCheckPassed,
		LiveTopicCheckFailed,
		LiveTopicRecoveryChanged,
	}
}
