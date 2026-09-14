package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/events"
)

// emitColonyLive is the ONE emission boundary for every live-colony event.
// Every lifecycle lane -- build, continue, plan, swarm, oracle, recovery --
// must publish a live.* topic through this function and no other.
// cmd/live_projection_test.go's TestEveryLiveEventGoesThroughOneBoundary
// enforces this structurally, by walking the package's own parsed syntax
// tree, rather than by review: a future lane that grows its own live-event
// writer fails that test by name.
//
// Emission failure is never allowed to change the outcome of the work being
// described. A nil store, an empty topic, a marshal failure, or a bus
// publish failure all return silently -- no error is surfaced to the
// caller, and the caller's own work proceeds exactly as if emission had
// never been attempted.
func emitColonyLive(topic string, payload events.ColonyLivePayload) {
	if store == nil || strings.TrimSpace(topic) == "" {
		return
	}
	payload.SchemaVersion = events.ColonyLiveSchemaVersion
	if colonyLiveSchemaVersionOverride != "" {
		// Test-only seam (see doc comment below): lets a test produce a
		// genuine, boundary-emitted event carrying an unrecognized schema
		// version, so the replay reducer's version-skip path is proven
		// against real published events rather than a hand-typed JSON
		// fixture. Empty in production; never set outside a test.
		payload.SchemaVersion = colonyLiveSchemaVersionOverride
	}
	payload.Sequence = nextLiveSequence(payload.EpisodeID)

	raw, err := payload.RawMessage()
	if err != nil {
		return
	}

	if colonyLiveEmissionFailureOverride {
		// Test-only seam (see doc comment below): simulates a real
		// bus.Publish failure at the exact same point a genuine one would
		// occur -- used by TestFailedLiveEmitNeverChangesLaneOutcome to prove
		// every lifecycle lane's own result and durable state are byte-
		// identical whether or not the live-event publish itself succeeds.
		// Empty (false) in production; never set outside a test.
		return
	}

	bus := events.NewBus(store, events.DefaultConfig())
	_, _ = bus.Publish(context.Background(), topic, raw, "aether-live")
}

// colonyLiveSchemaVersionOverride is the test-only seam emitColonyLive
// checks above. It must never be set outside a test, and every test that
// sets it must restore it to "" (e.g. via t.Cleanup) before returning.
var colonyLiveSchemaVersionOverride string

// colonyLiveEmissionFailureOverride is the test-only seam emitColonyLive
// checks above, right before the point it would otherwise publish. It must
// never be set outside a test, and every test that sets it must restore it
// to false (e.g. via t.Cleanup) before returning.
var colonyLiveEmissionFailureOverride bool

var (
	liveSequenceMu       sync.Mutex
	liveSequenceCounters = map[string]*int64{}
)

// nextLiveSequence returns the next monotonic sequence number for the given
// episode, starting at 1. Each episode keeps its own independent, strictly
// increasing series -- concurrent episodes emitting at the same time never
// interfere with each other's numbering, and the projection's ordering of
// events sharing an identical (second-precision) timestamp falls back to
// this sequence number.
func nextLiveSequence(episodeID string) int64 {
	liveSequenceMu.Lock()
	counter, ok := liveSequenceCounters[episodeID]
	if !ok {
		counter = new(int64)
		liveSequenceCounters[episodeID] = counter
	}
	liveSequenceMu.Unlock()
	return atomic.AddInt64(counter, 1)
}

// Per-lane emission helpers (202-03). Every lifecycle lane -- planning,
// building, checking, recovery -- speaks on the live stream through one of
// these thin, typed wrappers, never by constructing an events.ColonyLivePayload
// and calling emitColonyLive directly at its own call site. Each helper fills
// only the payload fields its moment genuinely knows, from the
// already-available domain value handed to it (a codex.WorkerDispatch, a
// codex.DispatchResult, a check name and outcome, a recovery state) --
// mirroring cmd/ceremony_emitter.go's emitBuildCeremonyWorkerStarting /
// emitBuildCeremonyWorkerFinished mapping style so the two vocabularies stay
// recognizably parallel. An absent field means unknown, never a zero
// measurement -- no helper invents a value its input does not carry.

// emitColonyLiveEpisodeStarted marks the beginning of one lifecycle episode
// (a build, a check pass, a planning run, ...). episodeKind should be one of
// the events.EpisodeKind* constants.
//
// 204-04 (LEARN-02, SYN-204-04): this is one of the two episode-boundary
// functions every lifecycle lane already calls to open/close a live
// episode -- extending it (rather than adding a second call site anywhere)
// is what puts the durable ledger on every lane with no new wiring per
// lane. See recordEpisodeLedgerOpen's own doc comment for the durable half.
func emitColonyLiveEpisodeStarted(episodeID, episodeKind string) {
	emitColonyLive(events.LiveTopicEpisodeStarted, events.ColonyLivePayload{
		EpisodeID:   episodeID,
		EpisodeKind: episodeKind,
		Status:      "starting",
	})
	recordEpisodeLedgerOpen(episodeID, episodeKind)
}

// emitColonyLiveEpisodeEnded closes the episode episodeID opened. status is
// the episode's own terminal status (e.g. "completed", "failed", "blocked",
// "interrupted") -- whatever the caller's own outcome value already is.
// See recordEpisodeLedgerClose's own doc comment for the durable half.
func emitColonyLiveEpisodeEnded(episodeID, episodeKind, status string) {
	emitColonyLive(events.LiveTopicEpisodeEnded, events.ColonyLivePayload{
		EpisodeID:   episodeID,
		EpisodeKind: episodeKind,
		Status:      status,
	})
	recordEpisodeLedgerClose(episodeID, episodeKind, status)
}

// recordEpisodeLedgerOpen (204-04, LEARN-02) writes the durable open record
// for episodeID through recordEpisodeOutcome -- the SAME opening moment
// emitColonyLiveEpisodeStarted already announces on the live stream, never
// a second call site. A ledger write failure is warned to stderr and never
// returned: bookkeeping must never fail the run it is describing, the same
// rule recordDispatchWorkerOutcome and runPhaseEndConsolidation already
// follow.
func recordEpisodeLedgerOpen(episodeID, episodeKind string) {
	episodeID = strings.TrimSpace(episodeID)
	if episodeID == "" || store == nil {
		return
	}
	_, _, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind:     episodeLedgerRecordKindOpened,
		EpisodeID:      episodeID,
		EpisodeKind:    episodeKind,
		RuntimeVersion: resolveVersion(),
		PolicyVersion:  episodeLedgerPolicyVersion(),
		StartedAt:      time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to record durable episode-open record for %q: %v\n", episodeID, err)
	}
}

// recordEpisodeLedgerClose (204-04, LEARN-02) writes the durable terminal
// record for episodeID. Elapsed time is computed from the ALREADY-STORED
// open record's own StartedAt, never from a separate clock reading taken
// here -- a resumed or replayed episode must never report an elapsed
// figure a full replay would not have produced (the live view already has
// a recorded defect of exactly that shape; this must not reproduce it).
func recordEpisodeLedgerClose(episodeID, episodeKind, status string) {
	episodeID = strings.TrimSpace(episodeID)
	if episodeID == "" || store == nil {
		return
	}
	now := time.Now().UTC()
	var elapsed float64
	if records, err := episodeLedgerForEpisode(episodeID); err == nil {
		if open, ok := episodeLedgerOpenRecord(records, episodeID); ok {
			if startedAt, parseErr := time.Parse(time.RFC3339, open.StartedAt); parseErr == nil {
				elapsed = now.Sub(startedAt).Seconds()
			}
		}
	}
	_, _, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind:     episodeLedgerRecordKindClosed,
		EpisodeID:      episodeID,
		EpisodeKind:    episodeKind,
		RuntimeVersion: resolveVersion(),
		PolicyVersion:  episodeLedgerPolicyVersion(),
		EndedAt:        now.Format(time.RFC3339),
		ElapsedSeconds: elapsed,
		TerminalResult: status,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to record durable episode-close record for %q: %v\n", episodeID, err)
	}
}

// episodeLedgerPolicyVersion is the schema version of the permission and
// admission policy currently in force, per this plan's own action text.
// codex.PermissionProfileSchemaVersion is the one declared schema-version
// constant this codebase carries for that policy today.
func episodeLedgerPolicyVersion() string {
	return fmt.Sprintf("%d", codex.PermissionProfileSchemaVersion)
}

// emitColonyLiveOutcomeRecorded (204-04, LEARN-02) publishes
// LiveTopicOutcomeRecorded and, through the same recordEpisodeOutcome call
// the episode boundary itself uses, writes a durable terminal record for a
// caller that already holds a fuller episodeLedgerRecord (real evidence,
// hard-gate, or usage facts recordEpisodeLedgerClose's own two-argument
// signature does not receive).
func emitColonyLiveOutcomeRecorded(episodeID, episodeKind string, record episodeLedgerRecord) {
	emitColonyLive(events.LiveTopicOutcomeRecorded, events.ColonyLivePayload{
		EpisodeID:   episodeID,
		EpisodeKind: episodeKind,
		Status:      record.TerminalResult,
	})
	record.RecordKind = episodeLedgerRecordKindClosed
	record.EpisodeID = episodeID
	if record.EpisodeKind == "" {
		record.EpisodeKind = episodeKind
	}
	if _, _, err := recordEpisodeOutcome(record); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to record durable outcome for %q: %v\n", episodeID, err)
	}
}

// emitColonyLiveInterventionRecorded (204-04, LEARN-02) publishes
// LiveTopicInterventionRecorded and writes a durable intervention_recorded
// record naming what the owner did (e.g. "declined a forced reviewer").
func emitColonyLiveInterventionRecorded(episodeID, episodeKind, intervention string) {
	intervention = strings.TrimSpace(intervention)
	if intervention == "" {
		return
	}
	emitColonyLive(events.LiveTopicInterventionRecorded, events.ColonyLivePayload{
		EpisodeID:   episodeID,
		EpisodeKind: episodeKind,
		Reason:      intervention,
	})
	if _, _, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind:    episodeLedgerRecordKindIntervention,
		EpisodeID:     episodeID,
		EpisodeKind:   episodeKind,
		StartedAt:     time.Now().UTC().Format(time.RFC3339),
		Interventions: []string{intervention},
	}); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to record durable intervention for %q: %v\n", episodeID, err)
	}
}

// emitColonyLiveWaveStarted marks the beginning of one dispatch wave inside
// an already-open episode.
func emitColonyLiveWaveStarted(episodeID, episodeKind string, wave int) {
	emitColonyLive(events.LiveTopicWaveStarted, events.ColonyLivePayload{
		EpisodeID:   episodeID,
		EpisodeKind: episodeKind,
		Wave:        wave,
		Status:      "starting",
	})
}

// emitColonyLiveWaveEnded closes the wave wave opened. status is the wave's
// own terminal status.
func emitColonyLiveWaveEnded(episodeID, episodeKind string, wave int, status string) {
	emitColonyLive(events.LiveTopicWaveEnded, events.ColonyLivePayload{
		EpisodeID:   episodeID,
		EpisodeKind: episodeKind,
		Wave:        wave,
		Status:      status,
	})
}

// emitColonyLiveWorkerStarted records one worker beginning its dispatch.
// Worker identity, caste, wave and workspace all come directly from
// dispatch. When dispatch names a parent (dispatch.ParentWorkerID is
// non-empty), that parent is recorded on the payload -- lineage read from
// the dispatch's own field, never derived from a rendered string.
func emitColonyLiveWorkerStarted(episodeID, episodeKind string, dispatch codex.WorkerDispatch) {
	emitColonyLive(events.LiveTopicWorkerStarted, events.ColonyLivePayload{
		EpisodeID:      episodeID,
		EpisodeKind:    episodeKind,
		Wave:           dispatch.Wave,
		WorkerID:       dispatch.WorkerName,
		ParentWorkerID: dispatch.ParentWorkerID,
		Caste:          dispatch.Caste,
		WorkerName:     dispatch.WorkerName,
		Workspace:      dispatch.Root,
		Status:         "active",
	})
}

// emitColonyLiveWorkerProgress records an in-flight progress note for an
// already-started worker. message is whatever the caller's own progress
// text already is (e.g. a WorkerProgressEvent.Message) -- never invented.
// It rides the payload's Question field, the same free-text "what this
// worker is doing right now" slot replayColonyLiveSnapshot's reducer
// already folds a worker.progress event's text into.
func emitColonyLiveWorkerProgress(episodeID, episodeKind string, dispatch codex.WorkerDispatch, message string) {
	emitColonyLive(events.LiveTopicWorkerProgress, events.ColonyLivePayload{
		EpisodeID:   episodeID,
		EpisodeKind: episodeKind,
		Wave:        dispatch.Wave,
		WorkerID:    dispatch.WorkerName,
		Caste:       dispatch.Caste,
		WorkerName:  dispatch.WorkerName,
		Status:      "running",
		Question:    strings.TrimSpace(message),
	})
}

// firstNonEmptyStringSlice wraps a single already-trimmed message into a
// one-element slice, or returns nil for an empty message -- kept local to
// this file since it exists purely to keep the check-failed mapping below a
// one-liner.
func firstNonEmptyStringSlice(message string) []string {
	if strings.TrimSpace(message) == "" {
		return nil
	}
	return []string{message}
}

// emitColonyLiveWorkerFinished records one worker's terminal result.
// Identity comes from dispatch; status comes from result -- the same two
// values cmd/ceremony_emitter.go's emitBuildCeremonyWorkerFinished already
// reads at its own call sites.
func emitColonyLiveWorkerFinished(episodeID, episodeKind string, dispatch codex.WorkerDispatch, result codex.DispatchResult) {
	status := strings.TrimSpace(result.Status)
	if status == "" {
		status = "failed"
	}
	emitColonyLive(events.LiveTopicWorkerFinished, events.ColonyLivePayload{
		EpisodeID:      episodeID,
		EpisodeKind:    episodeKind,
		Wave:           dispatch.Wave,
		WorkerID:       dispatch.WorkerName,
		ParentWorkerID: dispatch.ParentWorkerID,
		Caste:          dispatch.Caste,
		WorkerName:     dispatch.WorkerName,
		Workspace:      dispatch.Root,
		Status:         status,
	})
}

// emitColonyLiveCheckStarted marks the beginning of one named deterministic
// check (build, types, lint, tests, ...) inside a check episode.
func emitColonyLiveCheckStarted(episodeID, episodeKind, checkName string) {
	emitColonyLive(events.LiveTopicCheckStarted, events.ColonyLivePayload{
		EpisodeID:   episodeID,
		EpisodeKind: episodeKind,
		CheckName:   checkName,
		Status:      "starting",
	})
}

// emitColonyLiveCheckPassed records one check's passing terminal result.
func emitColonyLiveCheckPassed(episodeID, episodeKind, checkName string) {
	emitColonyLive(events.LiveTopicCheckPassed, events.ColonyLivePayload{
		EpisodeID:   episodeID,
		EpisodeKind: episodeKind,
		CheckName:   checkName,
		Status:      "passed",
	})
}

// emitColonyLiveCheckFailed records one check's failing terminal result.
// summary is whatever the check's own already-computed failure summary is.
func emitColonyLiveCheckFailed(episodeID, episodeKind, checkName, summary string) {
	emitColonyLive(events.LiveTopicCheckFailed, events.ColonyLivePayload{
		EpisodeID:   episodeID,
		EpisodeKind: episodeKind,
		CheckName:   checkName,
		Status:      "failed",
		Findings:    firstNonEmptyStringSlice(summary),
	})
}

// emitColonyLiveRecoveryChanged records one recovery state transition.
// recoveryState is the orchestrator's own already-decided action type (e.g.
// "retry", "peer_reassignment", "fixer_dispatch", "escalate").
func emitColonyLiveRecoveryChanged(episodeID, episodeKind, recoveryState string) {
	emitColonyLive(events.LiveTopicRecoveryChanged, events.ColonyLivePayload{
		EpisodeID:     episodeID,
		EpisodeKind:   episodeKind,
		RecoveryState: recoveryState,
	})
}

// emitColonyLiveSignalConsulted records the pheromone signals a lane
// consulted while making a decision. signals is whatever the caller's own
// already-resolved signal list is.
func emitColonyLiveSignalConsulted(episodeID, episodeKind string, signals []string) {
	emitColonyLive(events.LiveTopicSignalConsulted, events.ColonyLivePayload{
		EpisodeID:   episodeID,
		EpisodeKind: episodeKind,
		Signals:     append([]string{}, signals...),
	})
}

// emitColonyLiveRecruitAdmitted records one admitted recruitment (203-02,
// BIO-01/02/06). childName is the deterministic name the caller just
// recorded in the spawn tree. Reason carries the worker's own stated why for
// wanting help -- never invented, always the caller's own intent.Reason.
func emitColonyLiveRecruitAdmitted(intent recruitmentIntent, childName string) {
	episodeID, episodeKind := currentLiveRecruitmentEpisode()
	emitColonyLive(events.LiveTopicRecruitAdmitted, events.ColonyLivePayload{
		EpisodeID:      episodeID,
		EpisodeKind:    episodeKind,
		ParentWorkerID: intent.ParentName,
		Caste:          intent.Caste,
		WorkerName:     childName,
		Workspace:      intent.Workspace,
		Reason:         intent.Reason,
		Status:         "admitted",
	})
	// 203-14 (D-04): one inline line in the working session, at the SAME
	// moment the live event is published -- this is the ONE funnel both the
	// native `aether recruit` lane (cmd/recruitment.go) and the Go in-repo
	// build lane (cmd/recruitment_lane.go) already call, so wiring the
	// inline line here covers both lanes without adding a second call site
	// to either file (both outside 203-14's declared files_modified; see
	// 203-14-SUMMARY.md Deviations).
	emitInlineRecruitLine(intent.Caste, childName, intent.Reason, recruitmentInlineCostFigure(childName))
}

// emitColonyLiveRecruitRefused records one refused recruitment attempt
// (203-02, BIO-02). No child identity is invented for a refusal -- WorkerName
// stays empty, matching this codebase's "absent field means unknown, never a
// zero measurement" convention. reasonClass/detail are spawnDecisionResult's
// own Reason/Detail from the single spawnCanSpawnDecision call this
// recruitment made.
func emitColonyLiveRecruitRefused(intent recruitmentIntent, reasonClass, detail string) {
	episodeID, episodeKind := currentLiveRecruitmentEpisode()
	reason := strings.TrimSpace(reasonClass)
	if trimmedDetail := strings.TrimSpace(detail); trimmedDetail != "" {
		if reason != "" {
			reason = reason + ": " + trimmedDetail
		} else {
			reason = trimmedDetail
		}
	}
	emitColonyLive(events.LiveTopicRecruitRefused, events.ColonyLivePayload{
		EpisodeID:      episodeID,
		EpisodeKind:    episodeKind,
		ParentWorkerID: intent.ParentName,
		Caste:          intent.Caste,
		Workspace:      intent.Workspace,
		Reason:         reason,
		Status:         "refused",
	})
	// 203-14 (D-03/D-06): one inline line the moment the refusal happens --
	// same single-funnel rationale as emitColonyLiveRecruitAdmitted above.
	emitInlineRefusalLine(intent.Caste, reasonClass, detail)
}

// currentLiveRecruitmentEpisode resolves which already-open episode (a
// build, a check) a recruitment happening mid-task should render inside --
// mirroring currentLiveRecoveryEpisode's own build-then-continue precedence,
// but with no synthetic phase-derived fallback: a recruitment issued outside
// any open episode (a worker invoked directly via Bash, at top level) simply
// carries an empty EpisodeID, which is "unknown", never a fabricated one.
func currentLiveRecruitmentEpisode() (string, string) {
	if id := currentLiveBuildEpisode(); id != "" {
		return id, events.EpisodeKindBuild
	}
	if id := currentLiveContinueEpisode(); id != "" {
		return id, events.EpisodeKindContinue
	}
	return "", ""
}

// activeLiveBuildEpisode carries the current build's live-episode ID across
// the direct build lane's own call chain (cmd/codex_build.go's
// runCodexBuildWithOptions down into cmd/codex_build_worktree.go's per-wave
// dispatch loops), mirroring cmd/ceremony_emitter.go's
// activeBuildCeremony/setActiveBuildCeremony/currentBuildCeremony pattern --
// the same problem (a value the top of the call chain knows and a deeply
// nested dispatch loop needs) solved the same way, rather than threading a
// new parameter through every function in between.
var (
	activeLiveBuildEpisodeMu sync.RWMutex
	activeLiveBuildEpisodeID string
)

// setActiveLiveBuildEpisode sets the current build's live-episode ID and
// returns a restore function the caller must defer, so a nested or
// re-entrant build never leaks its episode ID into an unrelated one.
func setActiveLiveBuildEpisode(episodeID string) func() {
	activeLiveBuildEpisodeMu.Lock()
	previous := activeLiveBuildEpisodeID
	activeLiveBuildEpisodeID = episodeID
	activeLiveBuildEpisodeMu.Unlock()
	return func() {
		activeLiveBuildEpisodeMu.Lock()
		activeLiveBuildEpisodeID = previous
		activeLiveBuildEpisodeMu.Unlock()
	}
}

// currentLiveBuildEpisode returns the active build's live-episode ID, or ""
// when no build has one active (e.g. a call path never wrapped by
// setActiveLiveBuildEpisode) -- emitColonyLive still emits in that case,
// simply with an empty EpisodeID, exactly like any other unknown field.
func currentLiveBuildEpisode() string {
	activeLiveBuildEpisodeMu.RLock()
	defer activeLiveBuildEpisodeMu.RUnlock()
	return activeLiveBuildEpisodeID
}

// activeLiveContinueEpisode is continue's own instance of the same
// active-episode carrier build uses above -- runCodexContinueVerification
// (cmd/codex_continue.go) reads it from inside runDeterministicFloor's own
// result handling, several call frames below where the check episode ID is
// known.
var (
	activeLiveContinueEpisodeMu sync.RWMutex
	activeLiveContinueEpisodeID string
)

func setActiveLiveContinueEpisode(episodeID string) func() {
	activeLiveContinueEpisodeMu.Lock()
	previous := activeLiveContinueEpisodeID
	activeLiveContinueEpisodeID = episodeID
	activeLiveContinueEpisodeMu.Unlock()
	return func() {
		activeLiveContinueEpisodeMu.Lock()
		activeLiveContinueEpisodeID = previous
		activeLiveContinueEpisodeMu.Unlock()
	}
}

func currentLiveContinueEpisode() string {
	activeLiveContinueEpisodeMu.RLock()
	defer activeLiveContinueEpisodeMu.RUnlock()
	return activeLiveContinueEpisodeID
}

// currentLiveRecoveryEpisode resolves the episode a recovery decision for
// phase should be recorded against: the active build episode when one is
// set, otherwise the active continue (check) episode when one is set,
// otherwise a phase-derived fallback identifier for the (rarer) case where
// a recovery decision fires with no live episode open at all -- so a
// standalone recovery decision is still recorded rather than silently
// dropped.
//
// The episode kind returned alongside the ID is the OWNING episode's own
// kind (events.EpisodeKindBuild or events.EpisodeKindContinue), never
// events.EpisodeKindRecovery, except in the no-open-episode fallback case.
// This is deliberate: the projection's snapshot.EpisodeKind must keep
// naming the run the owner is actually watching (a build, or a check) --
// recovery is a detail layered onto that episode via RecoveryState, never
// a screen of its own.
//
// Build is checked before continue. In the runtime today this precedence
// can never actually matter: cmd/codex_build.go and cmd/codex_continue.go
// each clear their own carrier via a deferred restore before the other
// lane's call chain can run, so at most one of the two carriers is ever
// non-empty at a time a recovery decision fires.
func currentLiveRecoveryEpisode(phase int) (string, string) {
	if id := currentLiveBuildEpisode(); id != "" {
		return id, events.EpisodeKindBuild
	}
	if id := currentLiveContinueEpisode(); id != "" {
		return id, events.EpisodeKindContinue
	}
	return fmt.Sprintf("recovery-phase-%d", phase), events.EpisodeKindRecovery
}
