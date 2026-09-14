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
	"fmt"
	"strings"
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
