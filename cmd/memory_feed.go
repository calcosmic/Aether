package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/learn"
	"github.com/calcosmic/Aether/pkg/storage"
)

// middenCategoryWorkerFailed is the midden.json category every failed build
// worker's record is filed under.
const middenCategoryWorkerFailed = "worker_failed"

// workerOutcomeFacts is the one shape both build lanes (the wrapper-driven
// delegate lane in cmd/codex_build_finalize.go and the in-process native
// lane in cmd/codex_build.go) construct before calling
// feedMemoryFromWorkerOutcome, so this plan's whole boundary sees a single
// argument shape rather than each lane passing its own positional
// arguments.
type workerOutcomeFacts struct {
	Workflow   string
	PhaseID    int
	WorkerName string
	Caste      string
	Status     string
	Summary    string
	Blockers   []string
	Handoff    codex.WorkerHandoff
	// Failed and Succeeded are derived from the MERGED dispatch status (the
	// trust boundary mergeExternalBuildResults / the native dispatch loop
	// already crossed), never recomputed from a worker's raw self-report.
	Failed    bool
	Succeeded bool
	// AttemptID and JobName bind this outcome to the exact build attempt and
	// (when grouped) job that produced it (201-08's identity work). Resolved
	// from the phase's own latest durable build attempt record
	// (loadLatestBuildAttempt) rather than threaded through
	// codex.WorkerDispatch -- both build lanes already call
	// recordDispatchWorkerOutcome only after commitBuildStart has made that
	// attempt durable, so "the latest build attempt for this phase" IS the
	// attempt this outcome belongs to.
	AttemptID string
	JobName   string
}

// recordDispatchWorkerOutcome is the single boundary both build lanes call
// after a worker's terminal result is known. It preserves the existing
// worker-handoff persistence (persistDispatchWorkerHandoff) unchanged --
// including its return value, which this function returns as its own error
// -- and additionally feeds the failure log (midden.json) and the
// observation log (learning-observations.json) from the same facts. A
// memory-feed failure is warned to stderr and never returned: bookkeeping
// must never fail a build.
//
// TestEveryBuildLaneFeedsMemoryThroughOneBoundary (cmd/memory_feed_test.go)
// asserts this is the ONLY function in the build lanes that calls
// persistDispatchWorkerHandoff -- a future third lane that persists a
// handoff without going through this function fails that guard by name.
func recordDispatchWorkerOutcome(dispatch codex.WorkerDispatch, result codex.DispatchResult) error {
	err := persistDispatchWorkerHandoff(dispatch, result)

	facts := workerOutcomeFacts{
		Workflow:   dispatch.Workflow,
		PhaseID:    dispatch.Phase,
		WorkerName: dispatch.WorkerName,
		Caste:      dispatch.Caste,
		Status:     result.Status,
	}
	if result.WorkerResult != nil {
		facts.Summary = result.WorkerResult.Summary
		facts.Blockers = append([]string{}, result.WorkerResult.Blockers...)
		facts.Handoff = result.WorkerResult.Handoff
	}
	// isTerminalExternalBuildStatus / isSuccessfulExternalBuildStatus are the
	// SAME status-classification helpers the external build-finalize lane
	// already uses (cmd/codex_build_finalize.go) -- reused here rather than
	// re-derived, so "did this worker succeed" can never drift between the
	// merge path and the memory-feed path.
	facts.Failed = isTerminalExternalBuildStatus(facts.Status) && !isSuccessfulExternalBuildStatus(facts.Status)
	facts.Succeeded = isSuccessfulExternalBuildStatus(facts.Status)

	// Bind this outcome to the exact attempt and (when grouped) job it came
	// from (CAP-003/CAP-004, 201-08's identity work): the phase's own
	// latest durable build attempt record already carries both, per
	// dispatch, via attemptCoherentJobWaves' own source fields
	// (cmd/coherent_jobs.go). A phase with no durable attempt yet (a
	// synthetic/unit-test dispatch, for instance) leaves both empty rather
	// than fabricating an identity nothing actually bound.
	if _, attempt, ok := loadLatestBuildAttempt(dispatch.Phase); ok {
		facts.AttemptID = attempt.ID
		for _, d := range attempt.Dispatches {
			if d.Name == dispatch.WorkerName {
				facts.JobName = strings.TrimSpace(d.JobName)
				break
			}
		}
	}

	// Record which instincts this worker was actually given -- the input the
	// QUEEN.md promotion gate has always been missing (198.1-03, FEED-03).
	// Placed here, after the handoff persist and before the memory feed, so
	// Plan 01's one-boundary AST guard (TestEveryBuildLaneFeedsMemoryThroughOneBoundary)
	// stays the guarantee that both build lanes record deliveries.
	recordInstinctDeliveries(dispatch.Phase, dispatch.Workflow, capsuleForDispatch(dispatch))

	feedMemoryFromWorkerOutcome(facts)

	return err
}

// feedMemoryFromWorkerOutcome is the shared fan-out from one worker outcome
// into both memory stores this plan feeds: the failure log (midden.json,
// this task) and the observation log (learning-observations.json, Task 2).
func feedMemoryFromWorkerOutcome(facts workerOutcomeFacts) {
	if facts.Failed {
		message := middenMessageForFailedWorker(facts)
		if err := recordWorkerFailureToMidden(middenCategoryWorkerFailed, "aether build", message, middenTagsForFacts(facts)); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not record build worker failure to memory: %v\n", err)
		}
	}

	// A worker-reported blocker becomes a durable, attempt-bound record in
	// the SAME store every advancement, status, and closure read already
	// resolves blockers from (CAP-003/CAP-004) -- see recordWorkerBlockerFlag.
	// Triggered on any blocker the worker reported, independent of Failed:
	// a worker can report a blocker without its terminal status being a
	// build failure.
	if len(facts.Blockers) > 0 {
		if err := recordWorkerBlockerFlag(facts, sanitizedWorkerSentence(facts)); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not record worker blocker to memory: %v\n", err)
		}
	}

	sourceType := observationSourceTypeForOutcome(facts.Succeeded)
	rejected := 0
	firstReason := ""
	noteRejection := func(reason string) {
		rejected++
		if firstReason == "" {
			firstReason = reason
		}
	}
	for _, sentence := range facts.Handoff.DoNotRepeat {
		if ok, reason := captureWorkerObservation(sentence, "redirect", sourceType, "single_phase"); !ok {
			noteRejection(reason)
		}
	}
	for _, sentence := range facts.Handoff.NextWorkerInstructions {
		if ok, reason := captureWorkerObservation(sentence, "pattern", sourceType, "single_phase"); !ok {
			noteRejection(reason)
		}
	}
	if rejected > 0 {
		fmt.Fprintf(os.Stderr, "warning: %d worker sentence(s) rejected from memory; first reason: %s\n", rejected, firstReason)
	}
}

// middenMessageForFailedWorker builds the midden entry message for a failed
// worker. The worker's own sentence comes FIRST, before the attribution --
// the colony-prime capsule's "## Recent Failures" section
// (cmd/colony_prime_context.go) truncates each entry's message at 160
// characters, so anything after the sentence may be cut, but the sentence
// itself never is.
func middenMessageForFailedWorker(facts workerOutcomeFacts) string {
	sanitized := sanitizedWorkerSentence(facts)
	attribution := fmt.Sprintf("%s phase %d, worker %s (%s), status %s", facts.Workflow, facts.PhaseID, facts.WorkerName, facts.Caste, facts.Status)
	return sanitized + " — " + attribution
}

// sanitizedWorkerSentence is the ONE place a worker's own reported sentence
// (blocker, known failure, or summary -- see firstNonEmptyWorkerSentence) is
// sanitised before it is ever stored or replayed into a later brief (D-12's
// prohibition: never store or replay worker-authored text unsanitised).
// Both the failure-record path (middenMessageForFailedWorker) and the
// blocker-truth path (recordWorkerBlockerFlag) call this rather than each
// sanitising their own copy.
func sanitizedWorkerSentence(facts workerOutcomeFacts) string {
	sentence := firstNonEmptyWorkerSentence(facts)
	// Worker text is untrusted input that ends up inside another worker's
	// prompt (T-198.1-01) -- sanitise before it is ever stored.
	sanitized, err := colony.SanitizeSignalContent(sentence)
	if err != nil {
		return "the worker's reported reason could not be safely recorded"
	}
	return sanitized
}

// middenAttemptTagPrefix and middenJobTagPrefix mark the attempt identifier
// and job name a failure record carries (CAP-003) inside MiddenEntry.Tags --
// the one existing extension point appendMiddenEntry already exposes, so
// carrying this identity needs no change to appendMiddenEntry's own
// long-standing (category, source, message, tags) signature or its other
// callers (cmd/autopilot_retry_record.go, cmd/midden_cmds.go).
const (
	middenAttemptTagPrefix = "attempt:"
	middenJobTagPrefix     = "job:"
)

// middenTagsForFacts builds the tag list every midden entry this boundary
// writes carries: workflow and caste (unchanged from before this plan),
// plus the attempt identifier and job name (when known) so a failure record
// names the exact attempt it came from (CAP-003).
func middenTagsForFacts(facts workerOutcomeFacts) []string {
	tags := []string{facts.Workflow, facts.Caste}
	if id := strings.TrimSpace(facts.AttemptID); id != "" {
		tags = append(tags, middenAttemptTagPrefix+id)
	}
	if job := strings.TrimSpace(facts.JobName); job != "" {
		tags = append(tags, middenJobTagPrefix+job)
	}
	return tags
}

// middenEntryAttemptID and middenEntryJobName are the read side of
// middenTagsForFacts: the one way to recover a failure record's attempt
// identity from its tags. cmd/work_repair.go's failureBornRepairSignal
// (D-12) reads middenEntryAttemptID to find this same run's own failure
// evidence for the repair wave's brief.
func middenEntryAttemptID(entry colony.MiddenEntry) string {
	return middenEntryTagValue(entry, middenAttemptTagPrefix)
}

func middenEntryJobName(entry colony.MiddenEntry) string {
	return middenEntryTagValue(entry, middenJobTagPrefix)
}

func middenEntryTagValue(entry colony.MiddenEntry, prefix string) string {
	for _, tag := range entry.Tags {
		if strings.HasPrefix(tag, prefix) {
			return strings.TrimPrefix(tag, prefix)
		}
	}
	return ""
}

// recordWorkerBlockerFlag persists a worker-reported blocker as a durable,
// attempt-bound record in pending-decisions.json -- the exact store
// advancement (checkUnresolvedBlockerFlags, cmd/gate.go), status
// (readBlockerSnapshot, cmd/blocker_snapshot.go), and closure
// (LifecycleFacts.Blockers, cmd/lifecycle_facts.go) already read blockers
// from, so no second store is introduced (CAP-004). Source is
// "escalation" -- the same value swarm-strike and checkpoint escalations
// already write (cmd/swarm_strikes.go, cmd/autopilot_checkpoints.go) -- so
// the existing single escalated-blocker counting function
// (readBlockerSnapshotEvidence, cmd/blocker_snapshot.go) counts a
// worker-reported blocker without any change to that function at all
// (CAP-051: one counting path). Dedup is on (AttemptID, Description) within
// the unresolved set, mirroring appendMiddenEntryOnce's own established
// dedup discipline, so an idempotent re-finalize or a repeated completion
// packet never grows pending-decisions.json unbounded.
func recordWorkerBlockerFlag(facts workerOutcomeFacts, sanitizedMessage string) error {
	if store == nil {
		return nil
	}
	trimmed := strings.TrimSpace(sanitizedMessage)
	if trimmed == "" {
		return nil
	}
	attemptID := strings.TrimSpace(facts.AttemptID)
	description := fmt.Sprintf("%s — %s phase %d, worker %s (%s)", trimmed, facts.Workflow, facts.PhaseID, facts.WorkerName, facts.Caste)

	var ff colony.FlagsFile
	return store.UpdateJSONAtomically("pending-decisions.json", &ff, func() error {
		if ff.Decisions == nil {
			ff.Decisions = []colony.FlagEntry{}
		}
		for _, existing := range ff.Decisions {
			if existing.Resolved {
				continue
			}
			if existing.Type == "blocker" && existing.AttemptID == attemptID && existing.Description == description {
				return nil
			}
		}
		var phasePtr *int
		if facts.PhaseID > 0 {
			phaseID := facts.PhaseID
			phasePtr = &phaseID
		}
		ff.Decisions = append(ff.Decisions, colony.FlagEntry{
			ID:          generateFlagID(),
			Type:        "blocker",
			Description: description,
			Source:      "escalation",
			Phase:       phasePtr,
			AttemptID:   attemptID,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
			Resolved:    false,
		})
		return nil
	})
}

// firstNonEmptyWorkerSentence picks the worker's own explanation for a
// failure: the first blocker it reported, else the first known failure from
// its handoff, else its summary, else a literal fallback so the message is
// never empty.
func firstNonEmptyWorkerSentence(facts workerOutcomeFacts) string {
	for _, b := range facts.Blockers {
		if trimmed := strings.TrimSpace(b); trimmed != "" {
			return trimmed
		}
	}
	for _, f := range facts.Handoff.KnownFailures {
		if trimmed := strings.TrimSpace(f); trimmed != "" {
			return trimmed
		}
	}
	if trimmed := strings.TrimSpace(facts.Summary); trimmed != "" {
		return trimmed
	}
	return "the worker reported no reason"
}

// recordWorkerFailureToMidden delegates to appendMiddenEntryOnce against the
// global store. Returns nil when there is no store to write to (mirroring
// persistDispatchWorkerHandoff's own nil-store no-op).
func recordWorkerFailureToMidden(category, source, message string, tags []string) error {
	if store == nil {
		return nil
	}
	_, err := appendMiddenEntryOnce(store, category, source, message, tags)
	return err
}

// appendMiddenEntryOnce is appendMiddenEntry with a dedup guard: an
// unacknowledged entry already carrying the same category and message is
// not appended a second time. This is what makes recordDispatchWorkerOutcome
// safe to call from an idempotent re-finalize or a repeated completion
// packet without growing midden.json unbounded (T-198.1-02).
//
// WR-02 (198.1-REVIEW.md): the dedup key is the exact (category, message)
// pair -- there is no separate identity field. This means EVERY caller's
// message string must carry enough attribution (phase ID, worker name,
// status, target, etc.) that two genuinely distinct failures can never
// render into byte-identical text, while a genuine repeat of the same
// failure keeps rendering identically so it collapses as intended. A future
// caller that builds a leaner message risks silently suppressing a real,
// distinct failure as if it were a duplicate of an earlier one.
func appendMiddenEntryOnce(s *storage.Store, category, source, message string, tags []string) (bool, error) {
	if mf, err := loadMiddenFile(s); err == nil {
		for _, entry := range mf.Entries {
			if entry.Category != category || entry.Message != message {
				continue
			}
			if entry.Acknowledged == nil || !*entry.Acknowledged {
				return false, nil
			}
		}
	}
	if err := appendMiddenEntry(s, category, source, message, tags); err != nil {
		return false, err
	}
	return true, nil
}

// resolveRecentFailuresSection renders the same unacknowledged-failure
// content the colony-prime capsule's "## Recent Failures" section carries
// (cmd/colony_prime_context.go), for composeBuildManifestBrief's
// self-contained (includeSteeringSections=true) caller -- the one caller
// with no accompanying capsule of its own (see composeBuildManifestBrief's
// doc comment). Mirrors resolvePheromoneSection's shape: nil-store guard,
// "only render when there is content" contract, same truncation and cap the
// capsule renderer uses.
func resolveRecentFailuresSection() string {
	if store == nil {
		return ""
	}
	mf, err := loadMiddenFile(store)
	if err != nil || len(mf.Entries) == 0 {
		return ""
	}
	unacked := make([]colony.MiddenEntry, 0, len(mf.Entries))
	for _, entry := range mf.Entries {
		if entry.Acknowledged != nil && *entry.Acknowledged {
			continue
		}
		unacked = append(unacked, entry)
	}
	if len(unacked) == 0 {
		return ""
	}
	sort.SliceStable(unacked, func(i, j int) bool {
		return unacked[i].Timestamp > unacked[j].Timestamp
	})
	const recentFailuresLimit = 5
	shown := unacked
	remaining := 0
	if len(shown) > recentFailuresLimit {
		remaining = len(shown) - recentFailuresLimit
		shown = shown[:recentFailuresLimit]
	}
	var b strings.Builder
	b.WriteString("## Recent Failures\n\n")
	for _, entry := range shown {
		fmt.Fprintf(&b, "- [%s] %s\n", entry.Category, truncateString(entry.Message, 160))
	}
	if remaining > 0 {
		fmt.Fprintf(&b, "+%d more unacknowledged\n", remaining)
	}
	return b.String()
}

// captureWorkerObservation is the observation half of the memory feed
// (Task 2). It trims and sanitises the worker-authored sentence, refuses it
// if it fails the admissibility gate (too short/long, or names no file,
// command, or error), and otherwise captures it via the same observation
// service `aether memory-capture` uses -- never a second, parallel store.
// Returns (accepted, reason); reason is empty on acceptance.
func captureWorkerObservation(content, wisdomType, sourceType, evidenceType string) (bool, string) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return false, "content is empty"
	}
	sanitized, err := colony.SanitizeSignalContent(trimmed)
	if err != nil {
		return false, err.Error()
	}
	if ok, reason := learn.IsAdmissibleInstinctContent(sanitized); !ok {
		return false, reason
	}
	if store == nil {
		return false, "no store initialized"
	}
	bus := events.NewBus(store, events.DefaultConfig())
	svc := learn.NewObservationService(store, bus)
	colonyName := pipelineConfigForStore().ColonyName
	if _, err := svc.CaptureWithTrust(context.Background(), sanitized, wisdomType, colonyName, sourceType, evidenceType); err != nil {
		return false, err.Error()
	}
	return true, ""
}

// observationSourceTypeForOutcome maps a worker's success/failure into the
// trust-scoring source type its lessons are captured under -- one named
// function so this mapping is stated once rather than an inline conditional
// repeated by every plan that feeds an observation from a worker outcome.
func observationSourceTypeForOutcome(succeeded bool) string {
	if succeeded {
		return "success_pattern"
	}
	return "error_resolution"
}
