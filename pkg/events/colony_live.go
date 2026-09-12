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
	// LiveTopicGapTargeted (202-11, LIVE-06/CEC-05) announces one newly
	// added open research gap -- the same "new entries only" discipline
	// LiveTopicContradictionFound already uses, emitted immediately beside
	// it at Oracle's state.OpenGaps merge point.
	LiveTopicGapTargeted = "live.gap.targeted"
	// LiveTopicRecruitAdmitted (203-02, BIO-01/02/06) is emitted exactly
	// once per admitted recruitment, through emitColonyLive's one boundary
	// (cmd/live_events.go) -- never a second live-event family. Rendered
	// inline in the same terminal every other live event already renders
	// through (D-04, carried from Phase 202's D-01/D-02).
	LiveTopicRecruitAdmitted = "live.recruit.admitted"
	// LiveTopicRecruitRefused (203-02, BIO-02) is emitted exactly once per
	// refused recruitment attempt. A refusal never blocks or pauses the
	// caller (D-03); this topic is how the owner sees it happen, both
	// inline and later in the end-of-run summary (D-06).
	LiveTopicRecruitRefused = "live.recruit.refused"
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
	// EpisodeKindOracle (202-11, LIVE-06/CEC-05) is Oracle's own research
	// run. Oracle has no per-worker dispatch of its own -- one round is one
	// pass of the same research effort, not a new worker -- so a running
	// research episode is represented as a single, repeatedly-replaced
	// worker row (see cmd/oracle_live.go) rather than a new live-event
	// shape.
	EpisodeKindOracle = "oracle"
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
		EpisodeKindOracle,
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
	SchemaVersion string `json:"schema_version"`
	EpisodeID     string `json:"episode_id"`
	EpisodeKind   string `json:"episode_kind,omitempty"`
	Sequence      int64  `json:"sequence"`
	Wave          int    `json:"wave,omitempty"`
	// RoundCap (202-11, LIVE-06) is Oracle's own round-cap number for the
	// wave/round this event describes (oracleStateFile.MaxIterations) --
	// generic enough to carry any future lane's own "up to N of these"
	// figure, but only Oracle sets it today.
	RoundCap         int     `json:"round_cap,omitempty"`
	WorkerID         string  `json:"worker_id,omitempty"`
	ParentWorkerID   string  `json:"parent_worker_id,omitempty"`
	Caste            string  `json:"caste,omitempty"`
	WorkerName       string  `json:"worker_name,omitempty"`
	Workspace        string  `json:"workspace,omitempty"`
	Lens             string  `json:"lens,omitempty"`
	Question         string  `json:"question,omitempty"`
	Confidence       float64 `json:"confidence,omitempty"`
	TargetConfidence float64 `json:"target_confidence,omitempty"`
	// PreviousConfidence (202-11, LIVE-06) is the confidence value a
	// live.confidence.changed event is moving FROM -- Confidence above is
	// always the new value. Only a confidence-changed event sets this.
	PreviousConfidence float64  `json:"previous_confidence,omitempty"`
	Contradictions     []string `json:"contradictions,omitempty"`
	Findings           []string `json:"findings,omitempty"`
	Signals            []string `json:"signals,omitempty"`
	CheckName          string   `json:"check_name,omitempty"`
	RecoveryState      string   `json:"recovery_state,omitempty"`
	Status             string   `json:"status,omitempty"`
	ElapsedSeconds     float64  `json:"elapsed_seconds,omitempty"`
	// Reason (203-02, BIO-01/02) carries the worker's own stated why on a
	// live.recruit.admitted event, and the refusal reason class plus detail
	// on a live.recruit.refused event. Never a currency amount -- see this
	// struct's own doc comment above.
	Reason string `json:"reason,omitempty"`
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
		LiveTopicGapTargeted,
		LiveTopicRecruitAdmitted,
		LiveTopicRecruitRefused,
	}
}
