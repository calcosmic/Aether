package cmd

// LEARN-03 (204-02-PLAN.md Task 1, this phase's tracer; ruling (a) and
// SYN-204-03/05/06 in 204-CLASSIC-SYNTHESIS.md): the first real production
// writer into cmd/recruitment_credit.go's evidence-gated credit ledger --
// the exact outcome-weighted tuning the owner asked for in the immediately
// preceding phase (Phase 203), previously running against a permanently
// empty store because nothing ever called recordRecruitmentCredit outside
// its own test. A delivered instinct earns credit only when the phase's own
// durable build attempt records BOTH a changed decision (a knowledge delta
// of kind "decision") and a measured effect (a free-check comparison
// against the phase's own previous attempt) -- never on delivery into a
// brief or a passing phase alone (Assumption C/D in 204-02-PLAN.md).

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
)

// buildKnowledgeDeltaKindDecision is the buildAttemptKnowledgeDelta.Kind
// value (cmd/codex_build_finalize.go) that names a changed-decision delta,
// as distinct from a "learning" delta -- the only kind this file's credit
// derivation reads. Named here rather than inlined so no future comparison
// against this vocabulary silently drifts from a typo.
const buildKnowledgeDeltaKindDecision = "decision"

// phaseApplicationCreditSummary is the runtime-facing summary of one
// recordPhaseApplicationCredit call. Returned by value only, never as a Go
// error -- this pass can never itself fail the phase-end check that calls
// it, mirroring phaseEndConsolidationSummary's and
// pheromoneOutcomeTuningResult's own non-blocking shape (D-05's
// enrichment-not-gate discipline, extended here). Ran=false means this pass
// could not even start (no store, or the phase's build attempt could not be
// read); every field after Reason is meaningless in that case.
type phaseApplicationCreditSummary struct {
	Ran        bool
	Reason     string
	Considered int
	Recorded   int
	Pending    int
	Helpful    int
	Neutral    int
	Harmful    int
}

// phaseApplicationDecisionID is the deterministic, content-derived changed-
// decision identifier for one decision-kind knowledge delta -- a stable
// string built from the attempt's own ID and that delta's zero-based index
// within the decision-kind subset, following recruitmentCreditRecordID's
// own idiom (cmd/recruitment_credit.go) of deriving identity from stable
// inputs rather than a fresh UUID or timestamp. Declared as its own named
// helper so nothing constructs this format inline.
func phaseApplicationDecisionID(attemptID string, index int) string {
	return fmt.Sprintf("decision:%s:%d", strings.TrimSpace(attemptID), index)
}

// phaseApplicationEffectEvidenceID is the deterministic effect-evidence
// identifier for one attempt's free-check report -- derived from the
// attempt's own ID alone, since a build attempt carries at most one
// buildFreeCheckReport (buildAttemptRecord.FreeChecks).
func phaseApplicationEffectEvidenceID(attemptID string) string {
	return fmt.Sprintf("effect:%s", strings.TrimSpace(attemptID))
}

// recordPhaseApplicationCredit is the phase's tracer: the first real
// production writer into cmd/recruitment_credit.go's evidence-gated credit
// ledger. It has exactly one call site -- runPhaseEndConsolidation
// (cmd/consolidation_lifecycle.go), immediately after
// recordInstinctApplicationsForPhase(phaseID) -- which is itself already
// invoked from both check lanes (cmd/codex_continue.go's runCodexContinue,
// cmd/codex_continue_finalize.go's runCodexContinueFinalize). That one call
// site is what puts this function on both lanes; it has no Cobra entry
// point of its own and is never called directly from a command.
//
// Contributions are every instinct this phase's own delivery ledger
// (instinct-deliveries.json, recordInstinctDeliveries in
// cmd/instinct_application.go) recorded as delivered on phaseID, excluding
// any instinct now archived or absent from instincts.json
// (loadInstinctFileOrEmpty). Each contribution earns credit only against a
// changed decision this phase's own latest durable build attempt actually
// recorded (a KnowledgeDeltas entry of kind "decision") -- a contribution
// with no recorded decision earns no record at all, matching
// recordRecruitmentCredit's own first refusal branch (never a default or
// inferred credit). The outcome for every (contribution, decision) pair is
// derived from comparing that attempt's own free-check report against the
// most recent EARLIER attempt that also carries one -- never inferred from
// the phase having merely advanced (Assumption C).
//
// Never returns a Go error and never blocks phase advancement: a store that
// cannot be read, or a phase with no durable build attempt at all, is
// reported via Ran=false and Reason, matching phaseEndConsolidationSummary's
// and pheromoneOutcomeTuningResult's own non-blocking contract.
func recordPhaseApplicationCredit(phaseID int) phaseApplicationCreditSummary {
	summary := phaseApplicationCreditSummary{}
	if store == nil {
		summary.Reason = "no store initialized"
		return summary
	}

	// LEARN-03 (204-06-PLAN.md Task 2): ignored is recorded at phase close
	// rather than at worker close, because a later worker in the same phase
	// may still consult guidance an earlier one did not. This runs exactly
	// once per phase-end call, on every return path below, via defer --
	// every guidance rendered on this phase with no consulted, acted-on,
	// contradicted, or claimed-but-unverified record by the time this
	// function returns is recorded as ignored.
	defer sweepIgnoredGuidanceApplications(phaseID)

	var deliveries instinctDeliveryFile
	if err := store.LoadJSON(instinctDeliveriesPath, &deliveries); err != nil {
		// No delivery ledger yet is a fresh colony's normal starting state,
		// not a failure to report -- the pass ran cleanly and found nothing
		// to credit this phase.
		summary.Ran = true
		return summary
	}

	instinctFile := loadInstinctFileOrEmpty(store)
	knownInstincts := make(map[string]bool, len(instinctFile.Instincts))
	for _, inst := range instinctFile.Instincts {
		if !inst.Archived {
			knownInstincts[inst.ID] = true
		}
	}

	var contributions []string
	seenContribution := map[string]bool{}
	for _, d := range deliveries.Deliveries {
		if d.Phase != phaseID {
			continue
		}
		if !knownInstincts[d.InstinctID] {
			continue
		}
		if seenContribution[d.InstinctID] {
			continue
		}
		seenContribution[d.InstinctID] = true
		contributions = append(contributions, d.InstinctID)
	}
	summary.Ran = true
	summary.Considered = len(contributions)
	if len(contributions) == 0 {
		return summary
	}

	_, latestAttempt, ok := loadLatestBuildAttempt(phaseID)
	if !ok {
		summary.Reason = "no durable build attempt recorded for this phase"
		return summary
	}

	var decisionIDs []string
	decisionIndex := 0
	for _, delta := range latestAttempt.KnowledgeDeltas {
		if delta.Kind != buildKnowledgeDeltaKindDecision {
			continue
		}
		decisionIDs = append(decisionIDs, phaseApplicationDecisionID(latestAttempt.ID, decisionIndex))
		decisionIndex++
	}
	if len(decisionIDs) == 0 {
		// A contribution with no recorded decision earns no record at all
		// (recordRecruitmentCredit's own first refusal branch) -- every
		// contribution above was considered, but nothing is written.
		return summary
	}

	effectEvidenceID, outcome := phaseApplicationEffectAndOutcome(phaseID, latestAttempt)

	for _, contributionID := range contributions {
		for _, decisionID := range decisionIDs {
			record, credited, err := recordRecruitmentCredit(contributionID, recruitmentContributionMemoryItem, decisionID, effectEvidenceID, outcome, "")
			if err != nil {
				summary.Reason = err.Error()
				continue
			}
			if !credited {
				continue
			}
			summary.Recorded++
			switch record.Outcome {
			case recruitmentCreditOutcomePending:
				summary.Pending++
			case recruitmentCreditOutcomeHelpful:
				summary.Helpful++
			case recruitmentCreditOutcomeNeutral:
				summary.Neutral++
			case recruitmentCreditOutcomeHarmful:
				summary.Harmful++
			}
		}
	}

	return summary
}

// phaseApplicationEffectAndOutcome derives the effect-evidence identifier
// and the outcome for phaseID's latest durable build attempt, by comparing
// its own free-check report against the most recent EARLIER attempt (from
// listBuildAttemptsForPhase, strictly preceding latestAttempt by attempt
// ID) that also carries one. No baseline to measure against -- the latest
// attempt itself carries no free-check report, or no earlier attempt
// carries one either -- means an empty effect-evidence identifier and
// recruitmentCreditOutcomePending, never a default or inferred earned
// outcome (Assumption C's own pending rule).
//
// harmful wins over helpful when both are true in the same comparison
// (a regression is the more consequential fact than an unrelated
// improvement in the same pair of reports).
func phaseApplicationEffectAndOutcome(phaseID int, latestAttempt buildAttemptRecord) (string, recruitmentCreditOutcome) {
	if latestAttempt.FreeChecks == nil {
		return "", recruitmentCreditOutcomePending
	}

	// The attempt ID format ("attempt-<fixed-width timestamp>-<pid>",
	// deriveBuildAttemptID) is lexicographically sortable, so a plain string
	// comparison of the ID alone -- never the report's own contents -- finds
	// the most recent EARLIER attempt among every attempt this phase has
	// ever recorded.
	var earlierID string
	var earlier *buildFreeCheckReport
	for _, attempt := range listBuildAttemptsForPhase(phaseID) {
		if attempt.ID >= latestAttempt.ID {
			continue
		}
		if attempt.FreeChecks == nil {
			continue
		}
		if earlier == nil || attempt.ID > earlierID {
			earlierID = attempt.ID
			earlier = attempt.FreeChecks
		}
	}
	if earlier == nil {
		// No baseline to measure against -- a changed decision alone is
		// pending, never a default or inferred earned outcome.
		return "", recruitmentCreditOutcomePending
	}

	effectEvidenceID := phaseApplicationEffectEvidenceID(latestAttempt.ID)

	earlierFailed := make(map[string]bool, len(earlier.Failed))
	for _, name := range earlier.Failed {
		earlierFailed[name] = true
	}
	latestFailed := make(map[string]bool, len(latestAttempt.FreeChecks.Failed))
	for _, name := range latestAttempt.FreeChecks.Failed {
		latestFailed[name] = true
	}

	harmful := false
	for name := range latestFailed {
		if !earlierFailed[name] {
			harmful = true
			break
		}
	}
	if harmful {
		return effectEvidenceID, recruitmentCreditOutcomeHarmful
	}

	helpful := false
	for name := range earlierFailed {
		if !latestFailed[name] {
			helpful = true
			break
		}
	}
	if helpful {
		return effectEvidenceID, recruitmentCreditOutcomeHelpful
	}

	return effectEvidenceID, recruitmentCreditOutcomeNeutral
}

// ---------------------------------------------------------------------------
// LEARN-03 (204-06-PLAN.md Task 1): the nine-state guidance application
// vocabulary. recordPhaseApplicationCredit above proves one thing well --
// that a delivered instinct's text was genuinely present in a worker's
// capsule -- and records a success for every one of them on any phase that
// merely advanced, because reaching that code means the phase advanced.
// There is no path anywhere in this system that records a failure, an
// ignore, or a contradiction for a piece of guidance. This section adds
// that: nine declared states (available, rendered, consulted, acted on,
// ignored, contradicted, helpful, neutral, harmful) with a transition rule,
// so the program can tell advice that helped apart from advice that was
// merely shown.
// ---------------------------------------------------------------------------

// guidanceApplicationState is the declared, closed vocabulary a piece of
// guidance's recorded application may reach. Exactly nine members --
// TestGuidanceStateVocabularyIsClosed asserts guidanceApplicationStateVocabulary
// carries all nine and no more.
//
// The last three (helpful, neutral, harmful) deliberately reuse the same
// three words recruitmentCreditOutcome already declares
// (cmd/recruitment_credit.go) -- and are a SEPARATE Go type, never an alias
// of it, because the two answer different questions that can legitimately
// disagree: a recruitmentCreditOutcome is about a DECISION (did the changed
// decision this contribution produced turn out to help, per
// recordRecruitmentCredit's evidence-gated ledger); a
// guidanceApplicationState is about a DELIVERY (what happened to this
// specific piece of guidance once it reached a worker -- was it even read,
// let alone acted on). The same contribution can be credited helpful at the
// decision level while its own delivery-level application never advances
// past "rendered" (nobody ever claimed to consult it), and vice versa.
type guidanceApplicationState string

const (
	guidanceApplicationStateAvailable    guidanceApplicationState = "available"
	guidanceApplicationStateRendered     guidanceApplicationState = "rendered"
	guidanceApplicationStateConsulted    guidanceApplicationState = "consulted"
	guidanceApplicationStateActedOn      guidanceApplicationState = "acted_on"
	guidanceApplicationStateIgnored      guidanceApplicationState = "ignored"
	guidanceApplicationStateContradicted guidanceApplicationState = "contradicted"
	guidanceApplicationStateHelpful      guidanceApplicationState = "helpful"
	guidanceApplicationStateNeutral      guidanceApplicationState = "neutral"
	guidanceApplicationStateHarmful      guidanceApplicationState = "harmful"
)

// guidanceApplicationStateVocabulary is the declared, closed set of every
// state a guidance application may reach. Mirrors the
// recruitmentCreditOutcomeVocabulary / recruitmentContributionKindVocabulary
// completeness convention (cmd/recruitment_credit.go): a state added to the
// const block above must also be added here, or TestGuidanceStateVocabularyIsClosed
// fails.
var guidanceApplicationStateVocabulary = []guidanceApplicationState{
	guidanceApplicationStateAvailable,
	guidanceApplicationStateRendered,
	guidanceApplicationStateConsulted,
	guidanceApplicationStateActedOn,
	guidanceApplicationStateIgnored,
	guidanceApplicationStateContradicted,
	guidanceApplicationStateHelpful,
	guidanceApplicationStateNeutral,
	guidanceApplicationStateHarmful,
}

// guidanceApplicationStateNames returns the string form of every declared
// state, for display and for refusal messages that must name all nine.
func guidanceApplicationStateNames() []string {
	names := make([]string, 0, len(guidanceApplicationStateVocabulary))
	for _, s := range guidanceApplicationStateVocabulary {
		names = append(names, string(s))
	}
	return names
}

func guidanceApplicationStateDeclared(state guidanceApplicationState) bool {
	for _, s := range guidanceApplicationStateVocabulary {
		if s == state {
			return true
		}
	}
	return false
}

// guidanceApplicationTransitionRule is one state's transition contract:
// which states must already be reached before it may be recorded (Requires),
// and which already-reached states refuse it outright as a mutually
// exclusive terminal reading of the same delivery (Excludes). Every
// transition rule this system enforces is declared once, here, inside
// guidanceApplicationPredecessors -- recordGuidanceApplicationState below
// contains no comparison against a named state constant outside this map
// (TestGuidanceStateTransitionsRequireTheirPredecessors's own AST check).
type guidanceApplicationTransitionRule struct {
	Requires []guidanceApplicationState
	Excludes []guidanceApplicationState
}

// guidanceApplicationPredecessors declares every state's transition rule.
// Available has none. Rendered requires available. Consulted requires
// rendered. Acted on requires consulted. Ignored requires rendered and is
// refused when consulted is already present (mutually exclusive terminal
// readings of the same delivery -- TestIgnoredAndConsultedAreMutuallyExclusive).
// Contradicted requires consulted. Helpful, neutral and harmful each require
// acted on.
var guidanceApplicationPredecessors = map[guidanceApplicationState]guidanceApplicationTransitionRule{
	guidanceApplicationStateAvailable: {},
	guidanceApplicationStateRendered: {
		Requires: []guidanceApplicationState{guidanceApplicationStateAvailable},
	},
	guidanceApplicationStateConsulted: {
		Requires: []guidanceApplicationState{guidanceApplicationStateRendered},
	},
	guidanceApplicationStateActedOn: {
		Requires: []guidanceApplicationState{guidanceApplicationStateConsulted},
	},
	guidanceApplicationStateIgnored: {
		Requires: []guidanceApplicationState{guidanceApplicationStateRendered},
		Excludes: []guidanceApplicationState{guidanceApplicationStateConsulted},
	},
	guidanceApplicationStateContradicted: {
		Requires: []guidanceApplicationState{guidanceApplicationStateConsulted},
	},
	guidanceApplicationStateHelpful: {
		Requires: []guidanceApplicationState{guidanceApplicationStateActedOn},
	},
	guidanceApplicationStateNeutral: {
		Requires: []guidanceApplicationState{guidanceApplicationStateActedOn},
	},
	guidanceApplicationStateHarmful: {
		Requires: []guidanceApplicationState{guidanceApplicationStateActedOn},
	},
}

// guidanceApplicationRecord is one recorded fact: this guidance reached this
// state, on this phase, for this contribution kind, with (optionally) the
// evidence identifier that justified the transition.
type guidanceApplicationRecord struct {
	RecordID   string                      `json:"record_id"`
	GuidanceID string                      `json:"guidance_id"`
	Kind       recruitmentContributionKind `json:"kind"`
	Phase      int                         `json:"phase"`
	State      guidanceApplicationState    `json:"state"`
	EvidenceID string                      `json:"evidence_id,omitempty"`
	RecordedAt string                      `json:"recorded_at"`
}

// guidanceClaimRecord is one recorded fact: a worker's own claim to have
// consulted or acted on this guidance, which the runtime could NOT
// independently corroborate (Assumption K, 204-06-PLAN.md Task 2). Declared
// here as a sibling type of guidanceApplicationRecord because
// recruitmentCreditFile (cmd/recruitment_credit.go) carries both as sibling
// arrays in the same file; the writer that populates it
// (recordGuidanceClaimUnverified) is added in Task 2.
type guidanceClaimRecord struct {
	RecordID   string `json:"record_id"`
	GuidanceID string `json:"guidance_id"`
	Phase      int    `json:"phase"`
	RecordedAt string `json:"recorded_at"`
}

// guidanceApplicationRecordID is the deterministic, content-addressed record
// identifier over the guidance identifier, the phase, and the state --
// following recruitmentCreditRecordID's own identity idiom
// (cmd/recruitment_credit.go): the same (guidanceID, phase, state) always
// resolves to the same record, so a repeat write is a replay, never a
// second record.
func guidanceApplicationRecordID(guidanceID string, phaseID int, state guidanceApplicationState) string {
	return fmt.Sprintf("guidance:%s:%d:%s", strings.TrimSpace(guidanceID), phaseID, state)
}

// errGuidanceApplicationNoChange is the internal replay sentinel
// recordGuidanceApplicationState returns from its own UpdateJSONAtomically
// mutate closure to abort the write on a replay -- mirroring
// errRecruitmentCreditNoChange's role in recordRecruitmentCredit
// (cmd/recruitment_credit.go).
var errGuidanceApplicationNoChange = errors.New("guidance application state already recorded")

// guidanceReachedStates returns the set of states already recorded for
// (guidanceID, phaseID) among records -- the read side every transition
// check and every idempotency check below is built from.
func guidanceReachedStates(records []guidanceApplicationRecord, guidanceID string, phaseID int) map[guidanceApplicationState]bool {
	reached := make(map[guidanceApplicationState]bool, len(guidanceApplicationStateVocabulary))
	for _, rec := range records {
		if rec.GuidanceID == guidanceID && rec.Phase == phaseID {
			reached[rec.State] = true
		}
	}
	return reached
}

// findGuidanceApplicationRecord returns the existing record for
// (guidanceID, phaseID, state) among records, if one is already there.
func findGuidanceApplicationRecord(records []guidanceApplicationRecord, guidanceID string, phaseID int, state guidanceApplicationState) (guidanceApplicationRecord, bool) {
	for _, rec := range records {
		if rec.GuidanceID == guidanceID && rec.Phase == phaseID && rec.State == state {
			return rec, true
		}
	}
	return guidanceApplicationRecord{}, false
}

// guidanceApplicationTransitionRefusal checks state's transition rule
// (guidanceApplicationPredecessors) against reached, and returns a non-nil
// error naming the exact missing predecessor or the exact excluded
// conflicting state when the transition is refused. Contains no comparison
// against a named state constant -- only generic membership checks over the
// rule's own Requires/Excludes lists -- so every transition rule this system
// enforces stays declared exactly once, in the map above.
func guidanceApplicationTransitionRefusal(state guidanceApplicationState, reached map[guidanceApplicationState]bool, guidanceID string, phaseID int) error {
	rule := guidanceApplicationPredecessors[state]
	for _, excluded := range rule.Excludes {
		if reached[excluded] {
			return fmt.Errorf(
				"guidance application state %q is refused for guidance %q on phase %d: %q is already recorded, and these are mutually exclusive terminal readings of the same delivery",
				state, guidanceID, phaseID, excluded,
			)
		}
	}
	for _, predecessor := range rule.Requires {
		if !reached[predecessor] {
			return fmt.Errorf(
				"guidance application state %q requires %q to already be recorded for guidance %q on phase %d, but it is not",
				state, predecessor, guidanceID, phaseID,
			)
		}
	}
	return nil
}

// recordGuidanceApplicationState is the single writer for every guidance
// application state transition. It refuses an undeclared state by name,
// listing all nine declared values; refuses a state whose transition rule is
// not satisfied (missing predecessor, or an excluded state already
// recorded), naming the specific reason; returns the existing record with
// written=false on a repeat; and stores through the credit ledger's own
// atomic-update store (recruitmentCreditPath, cmd/recruitment_credit.go) as
// a sibling array in the SAME file -- never a second data file -- so one
// read gives a reader the whole picture of what a contribution did.
func recordGuidanceApplicationState(guidanceID string, kind recruitmentContributionKind, phaseID int, state guidanceApplicationState, evidenceID string) (guidanceApplicationRecord, bool, error) {
	if store == nil {
		return guidanceApplicationRecord{}, false, fmt.Errorf("no store initialized")
	}
	guidanceID = strings.TrimSpace(guidanceID)
	if guidanceID == "" {
		return guidanceApplicationRecord{}, false, fmt.Errorf("guidance application state requires a non-empty guidance id")
	}
	if !guidanceApplicationStateDeclared(state) {
		return guidanceApplicationRecord{}, false, fmt.Errorf(
			"guidance application state %q is not in the declared vocabulary %v", state, guidanceApplicationStateNames(),
		)
	}

	var file recruitmentCreditFile
	var result guidanceApplicationRecord
	updateErr := store.UpdateJSONAtomically(recruitmentCreditPath, &file, func() error {
		reached := guidanceReachedStates(file.GuidanceApplications, guidanceID, phaseID)

		if existing, ok := findGuidanceApplicationRecord(file.GuidanceApplications, guidanceID, phaseID, state); ok {
			result = existing
			return errGuidanceApplicationNoChange
		}

		if err := guidanceApplicationTransitionRefusal(state, reached, guidanceID, phaseID); err != nil {
			return err
		}

		rec := guidanceApplicationRecord{
			RecordID:   guidanceApplicationRecordID(guidanceID, phaseID, state),
			GuidanceID: guidanceID,
			Kind:       kind,
			Phase:      phaseID,
			State:      state,
			EvidenceID: strings.TrimSpace(evidenceID),
			RecordedAt: time.Now().UTC().Format(time.RFC3339),
		}
		file.GuidanceApplications = append(file.GuidanceApplications, rec)
		result = rec
		return nil
	})
	if updateErr != nil {
		if errors.Is(updateErr, errGuidanceApplicationNoChange) {
			return result, false, nil
		}
		return guidanceApplicationRecord{}, false, updateErr
	}
	return result, true, nil
}

// ---------------------------------------------------------------------------
// LEARN-03 (204-06-PLAN.md Task 2): checking a worker's claim against what
// the runtime can see for itself (Assumption K). A worker cannot literally
// cite an instinct's internal ID -- colony-prime never renders one into a
// worker's capsule (cmd/colony_prime_context.go's "- [trigger] action
// (confidence: X.XX)" format carries no ID) -- so a claim is detected by the
// SAME signal recordInstinctDeliveries already uses for delivery: the
// guidance's own action text appearing in the worker's own words.
// ---------------------------------------------------------------------------

// guidanceClaim is one guidance identifier a worker's own handoff names as
// consulted or acted on -- extracted from untrusted input
// (guidanceClaimFromHandoff) and never itself proof of anything; it is
// checked against durable evidence by corroborateGuidanceClaim before it is
// ever recorded as more than a claim.
type guidanceClaim struct {
	GuidanceID string
	Kind       recruitmentContributionKind
}

// guidanceClaimFromHandoff extracts, from a worker's own handoff, the
// guidance identifiers that worker says it consulted or acted on. It is
// parsing untrusted input: every candidate is checked against instincts.json
// (loadInstinctFileOrEmpty) -- only a non-archived instinct whose own action
// text is found, verbatim, inside one of the worker's own retrospective
// fields (NextWorkerInstructions, DoNotRepeat -- the same two fields
// feedMemoryFromWorkerOutcome already treats as the worker's own lesson
// content, cmd/memory_feed.go) is extracted; anything else is discarded
// rather than invented.
func guidanceClaimFromHandoff(handoff codex.WorkerHandoff) []guidanceClaim {
	if store == nil {
		return nil
	}
	file := loadInstinctFileOrEmpty(store)
	if len(file.Instincts) == 0 {
		return nil
	}

	texts := make([]string, 0, len(handoff.NextWorkerInstructions)+len(handoff.DoNotRepeat))
	texts = append(texts, handoff.NextWorkerInstructions...)
	texts = append(texts, handoff.DoNotRepeat...)
	if len(texts) == 0 {
		return nil
	}

	var claims []guidanceClaim
	for _, inst := range file.Instincts {
		if inst.Archived {
			continue
		}
		action := strings.TrimSpace(inst.Action)
		if action == "" {
			continue
		}
		for _, text := range texts {
			if strings.Contains(text, action) {
				claims = append(claims, guidanceClaim{GuidanceID: inst.ID, Kind: recruitmentContributionMemoryItem})
				break
			}
		}
	}
	return claims
}

// guidanceContradictionMarker is the fixed phrase a worker's own recorded
// decision must contain, immediately followed by the guidance's own action
// text, for that decision to count as an explicit contradiction. A
// contradiction is parsed conservatively from an explicit marker, never
// inferred from prose -- the same discipline this repository already
// applies to worker-reported evidence generally (never believed, always
// re-checked).
const guidanceContradictionMarker = "against its own guidance: "

// corroborateGuidanceClaim is the independent check behind a worker's own
// claim to have consulted or acted on a piece of guidance (Assumption K): it
// looks only at records the runtime wrote itself, never the worker's own
// free-text fields directly (TestCorroborationNeverReadsTheWorkersOwnText) --
//
//   - the handoff's own recorded changed-file list (facts.Handoff.ChangedFiles,
//     a durable list of paths, not prose)
//   - the "decision"-kind knowledge delta deriveBuildKnowledgeDeltas
//     (cmd/build_knowledge_deltas.go) derives fresh from this worker's OWN
//     already-persisted handoff record -- a runtime-derived, sanitized
//     summary, never the raw field
//   - the "decision"-kind deltas the phase's latest durable build attempt
//     already carries (loadLatestBuildAttempt), the SAME durable decision
//     records recordPhaseApplicationCredit itself reads
//
// This mirrors the discipline this repository already applies to
// builder-reported evidence (CLAUDE.md's Coherent Jobs section: a worker's
// claim is checked against files actually present, never taken on its own
// word): a claim is re-run against durable evidence, never believed.
//
// Returns acted_on when this worker's own changed files are non-empty AND a
// decision delta freshly derived from this worker's own handoff mentions the
// guidance's action text; consulted when the phase's already-durable latest
// attempt carries a decision delta mentioning it (evidence someone recorded
// this decision this phase, even if not corroborated as this worker's own
// action); ok=false when neither is found.
func corroborateGuidanceClaim(claim guidanceClaim, facts workerOutcomeFacts) (guidanceApplicationState, string, bool) {
	if store == nil {
		return "", "", false
	}
	action := guidanceActionTextByID(claim.GuidanceID)
	if action == "" {
		return "", "", false
	}

	if len(facts.Handoff.ChangedFiles) > 0 {
		deltas := deriveBuildKnowledgeDeltas(facts.PhaseID, []codexBuildDispatch{{Name: facts.WorkerName}})
		for i, delta := range deltas {
			if delta.Kind == buildKnowledgeDeltaKindDecision && strings.Contains(delta.Summary, action) {
				return guidanceApplicationStateActedOn, fmt.Sprintf("handoff-decision:%s:%d", facts.WorkerName, i), true
			}
		}
	}

	if _, latest, ok := loadLatestBuildAttempt(facts.PhaseID); ok {
		decisionIndex := 0
		for _, delta := range latest.KnowledgeDeltas {
			if delta.Kind != buildKnowledgeDeltaKindDecision {
				continue
			}
			if strings.Contains(delta.Summary, action) {
				return guidanceApplicationStateConsulted, phaseApplicationDecisionID(latest.ID, decisionIndex), true
			}
			decisionIndex++
		}
	}

	return "", "", false
}

// guidanceActionTextByID looks up a non-archived instinct's own trimmed
// action text by ID, or "" if it is archived, absent, or blank.
func guidanceActionTextByID(guidanceID string) string {
	if store == nil {
		return ""
	}
	file := loadInstinctFileOrEmpty(store)
	for _, inst := range file.Instincts {
		if inst.Archived || inst.ID != guidanceID {
			continue
		}
		return strings.TrimSpace(inst.Action)
	}
	return ""
}

// guidanceContradiction is one guidance identifier a worker's own recorded
// decision explicitly went against (guidanceContradictionMarker), together
// with the evidence identifier that decision's delta corresponds to.
type guidanceContradiction struct {
	GuidanceID string
	EvidenceID string
}

// guidanceContradictionsFromWorkerOutcome scans the "decision"-kind
// knowledge deltas deriveBuildKnowledgeDeltas derives fresh from this
// worker's own already-persisted handoff record for the explicit
// contradiction marker, immediately followed by a known, non-archived
// instinct's own action text.
func guidanceContradictionsFromWorkerOutcome(facts workerOutcomeFacts) []guidanceContradiction {
	if store == nil {
		return nil
	}
	deltas := deriveBuildKnowledgeDeltas(facts.PhaseID, []codexBuildDispatch{{Name: facts.WorkerName}})
	if len(deltas) == 0 {
		return nil
	}
	file := loadInstinctFileOrEmpty(store)
	var out []guidanceContradiction
	for i, delta := range deltas {
		if delta.Kind != buildKnowledgeDeltaKindDecision {
			continue
		}
		for _, inst := range file.Instincts {
			if inst.Archived {
				continue
			}
			action := strings.TrimSpace(inst.Action)
			if action == "" {
				continue
			}
			if strings.Contains(delta.Summary, guidanceContradictionMarker+action) {
				out = append(out, guidanceContradiction{
					GuidanceID: inst.ID,
					EvidenceID: fmt.Sprintf("handoff-decision:%s:%d", facts.WorkerName, i),
				})
			}
		}
	}
	return out
}

// recordGuidanceClaimUnverified records a worker's own uncorroborated claim
// to have consulted or acted on guidanceID -- neither accepted nor discarded
// (Assumption K). Idempotent per (guidanceID, phase): a repeat claim writes
// nothing new.
func recordGuidanceClaimUnverified(guidanceID string, phaseID int) (guidanceClaimRecord, bool, error) {
	if store == nil {
		return guidanceClaimRecord{}, false, fmt.Errorf("no store initialized")
	}
	guidanceID = strings.TrimSpace(guidanceID)
	if guidanceID == "" {
		return guidanceClaimRecord{}, false, fmt.Errorf("guidance claim requires a non-empty guidance id")
	}

	recordID := fmt.Sprintf("guidance-claim:%s:%d", guidanceID, phaseID)
	var file recruitmentCreditFile
	var result guidanceClaimRecord
	updateErr := store.UpdateJSONAtomically(recruitmentCreditPath, &file, func() error {
		for _, existing := range file.GuidanceClaims {
			if existing.RecordID == recordID {
				result = existing
				return errGuidanceApplicationNoChange
			}
		}
		rec := guidanceClaimRecord{
			RecordID:   recordID,
			GuidanceID: guidanceID,
			Phase:      phaseID,
			RecordedAt: time.Now().UTC().Format(time.RFC3339),
		}
		file.GuidanceClaims = append(file.GuidanceClaims, rec)
		result = rec
		return nil
	})
	if updateErr != nil {
		if errors.Is(updateErr, errGuidanceApplicationNoChange) {
			return result, false, nil
		}
		return guidanceClaimRecord{}, false, updateErr
	}
	return result, true, nil
}

// recordGuidanceStatesForWorkerOutcome is the fan-out from one worker
// outcome into the guidance application ledger, called from
// recordDispatchWorkerOutcome (cmd/memory_feed.go) immediately after the
// existing recordInstinctDeliveries call and before feedMemoryFromWorkerOutcome.
// Placing it there keeps the existing one-boundary guard
// (TestEveryBuildLaneFeedsMemoryThroughOneBoundary) as the guarantee that
// both build lanes record these states -- there is no second call site.
//
// For each guidance claim extracted from this worker's own handoff
// (guidanceClaimFromHandoff): a corroborated claim is recorded consulted
// (always -- the predecessor acted_on itself requires), and additionally
// acted_on when corroboration reached that far; an uncorroborated claim is
// recorded as an unverified claim (recordGuidanceClaimUnverified) --
// neither consulted nor ignored. Separately, any decision this worker's own
// handoff explicitly recorded as going against a piece of guidance is
// recorded contradicted (which also requires consulted first).
func recordGuidanceStatesForWorkerOutcome(facts workerOutcomeFacts) {
	if store == nil {
		return
	}

	for _, claim := range guidanceClaimFromHandoff(facts.Handoff) {
		state, evidenceID, corroborated := corroborateGuidanceClaim(claim, facts)
		if !corroborated {
			if _, _, err := recordGuidanceClaimUnverified(claim.GuidanceID, facts.PhaseID); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not record guidance claimed-but-unverified state: %v\n", err)
			}
			continue
		}
		if _, _, err := recordGuidanceApplicationState(claim.GuidanceID, claim.Kind, facts.PhaseID, guidanceApplicationStateConsulted, evidenceID); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not record guidance consulted state: %v\n", err)
			continue
		}
		if state == guidanceApplicationStateActedOn {
			if _, _, err := recordGuidanceApplicationState(claim.GuidanceID, claim.Kind, facts.PhaseID, guidanceApplicationStateActedOn, evidenceID); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not record guidance acted-on state: %v\n", err)
			}
		}
	}

	for _, contradiction := range guidanceContradictionsFromWorkerOutcome(facts) {
		if _, _, err := recordGuidanceApplicationState(contradiction.GuidanceID, recruitmentContributionMemoryItem, facts.PhaseID, guidanceApplicationStateConsulted, contradiction.EvidenceID); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not record guidance consulted state ahead of contradiction: %v\n", err)
			continue
		}
		if _, _, err := recordGuidanceApplicationState(contradiction.GuidanceID, recruitmentContributionMemoryItem, facts.PhaseID, guidanceApplicationStateContradicted, contradiction.EvidenceID); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not record guidance contradicted state: %v\n", err)
		}
	}
}

// sweepIgnoredGuidanceApplications records ignored for every guidance
// rendered on phaseID with no consulted, acted-on, contradicted, or
// claimed-but-unverified record -- called once, unconditionally, from
// recordPhaseApplicationCredit via defer, so it runs at phase close exactly
// once per phase-end pass regardless of which of that function's own early
// returns fires.
func sweepIgnoredGuidanceApplications(phaseID int) {
	if store == nil {
		return
	}
	var file recruitmentCreditFile
	if err := store.LoadJSON(recruitmentCreditPath, &file); err != nil {
		return
	}

	renderedKind := map[string]recruitmentContributionKind{}
	settled := map[string]bool{}
	for _, rec := range file.GuidanceApplications {
		if rec.Phase != phaseID {
			continue
		}
		if rec.State == guidanceApplicationStateRendered {
			renderedKind[rec.GuidanceID] = rec.Kind
		}
		if rec.State == guidanceApplicationStateConsulted || rec.State == guidanceApplicationStateActedOn || rec.State == guidanceApplicationStateContradicted {
			settled[rec.GuidanceID] = true
		}
	}
	for _, rec := range file.GuidanceClaims {
		if rec.Phase == phaseID {
			settled[rec.GuidanceID] = true
		}
	}

	for guidanceID, kind := range renderedKind {
		if settled[guidanceID] {
			continue
		}
		if _, _, err := recordGuidanceApplicationState(guidanceID, kind, phaseID, guidanceApplicationStateIgnored, ""); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not record guidance ignored state: %v\n", err)
		}
	}
}
