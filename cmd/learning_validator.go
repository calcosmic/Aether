package cmd

// WINDOWS.md entry 44 (REOPENED by 204-REVIEW.md CR-01, 2026-09-15): the
// automatic hypothesis-to-validated promoter. Before this file,
// learn.StatusValidated was written only by the hand-run `learning-validate`
// command (cmd/learning_cmds.go's learningValidateCmd) -- an owner-typed
// command a worker's own genuinely-proven lesson never reaches on its own.
// The filtering machinery that renders "verified" versus "not yet verified"
// content (cmd/learning_status_vocabulary.go's learningVerifiedEntries/
// learningUnverifiedEntries) was already correct and wired at both render
// sites (cmd/colony_prime_context.go, cmd/autopilot_lessons.go) -- the gap
// this file intended to close was that nothing durable ever moved a real
// entry INTO the verified set automatically.
//
// promoteHelpfulHypotheses was intended to close that gap using evidence
// this phase's earlier plans already built: cmd/application_evidence.go's
// nine-state guidanceApplicationState vocabulary, specifically its terminal
// guidanceApplicationStateHelpful state. IT CANNOT DO SO IN PRODUCTION
// TODAY, for two compounding reasons confirmed by grep across cmd/
// (excluding _test.go files) at review time:
//
//  1. Every non-test writer of a GuidanceApplications record
//     (recordGuidanceApplicationState, cmd/application_evidence.go:541)
//     keys it by an Instinct's own ID (colony.InstinctEntry.ID) --
//     cmd/instinct_application.go:69,75 and cmd/application_evidence.go's
//     claim/contradiction/ignored writers all pass inst.ID or
//     claim.GuidanceID, itself always sourced from an Instinct. A
//     learn.Entry (pkg/learn), the store this function reads, has an
//     entirely separate ID generator (pkg/learn/colony_store.go's
//     generateID) and no production writer ever links the two --
//     learn.Entry.ParentID exists only for hypothesis-revision lineage
//     (one learn.Entry to another), never entry-to-instinct, and no
//     Lineage.SourceID writer ever names an Instinct either (confirmed by
//     grep: cmd/codex_continue_finalize.go's worker-lesson capture and
//     cmd/oracle_promote.go's promoteOracleFindingAsLearning both create a
//     brand-new learn.Entry with a freshly generated ID and no lineage back
//     to any instinct).
//  2. Independently of (1), no non-test production call site anywhere in
//     cmd/ ever writes guidanceApplicationStateHelpful (or
//     …StateNeutral/…StateHarmful) at all -- confirmed by the same grep.
//     The "helpful" outcome QUEEN promotion actually reads
//     (recruitmentCreditOutcomeHelpful, cmd/recruitment_credit.go) lives in
//     a different list (recruitmentCreditFile.Entries) than the one this
//     function reads (recruitmentCreditFile.GuidanceApplications) and is
//     never bridged into the guidance-application vocabulary's terminal
//     states.
//
// Net effect: learningEntryHasHelpfulApplication returns false for every
// learn.Entry, on every check, forever, so Promoted can never be non-empty
// in production even though Ran is reported true. Do not "fix" this by
// fabricating an identifier correlation a real production writer does not
// produce (that is exactly the "false certificate... a fixture built in a
// shape the runtime cannot produce" failure mode this repo's own CLAUDE.md
// names by name) -- a genuine fix requires either a real production writer
// linking a learn.Entry to the Instinct it came from (option (a) in
// 204-REVIEW.md's CR-01), or the learning-propose/-validate pipeline itself
// writing/reading GuidanceApplications under learn.Entry.ID from a real
// call site (option (b)), AND a real writer of the Helpful/Neutral/Harmful
// states themselves. Neither exists yet. See
// cmd/learning_validator_test.go's TestHypothesisPromotionNeverCrossesTheIdentifierGap
// for the test that locks in this honest, current limitation.
import (
	"fmt"
	"os"

	"github.com/calcosmic/Aether/pkg/learn"
)

// learningValidationSummary is the non-blocking summary of one
// promoteHelpfulHypotheses call, mirroring improvementPassSummary's and
// phaseApplicationCreditSummary's own shape: this function never returns a
// Go error a caller could propagate into an abort.
type learningValidationSummary struct {
	Ran        bool
	Considered int
	// Promoted carries the actual learn.Entry IDs this call moved from
	// hypothesis to validated -- not just a count, mirroring
	// phaseEndConsolidationSummary.QueenPromoted's own "name what actually
	// happened" convention.
	Promoted []string
	Failures []string
}

func (s *learningValidationSummary) recordFailure(id, reason string) {
	if id != "" {
		reason = fmt.Sprintf("%s: %s", id, reason)
	}
	s.Failures = append(s.Failures, reason)
	fmt.Fprintf(os.Stderr, "learning validator: %s\n", reason)
}

// learningEntryHasHelpfulApplication reports whether ANY guidance
// application record for guidanceID has reached the helpful state, across
// every phase that ever recorded one -- a lesson proven helpful once is
// proof enough that it is no longer merely a guess, regardless of which
// phase produced that proof. Reads recruitmentCreditPath directly (the
// same file recordGuidanceApplicationState writes), the way
// sweepIgnoredGuidanceApplications already reads it: a missing or
// unreadable file is read as "no records yet" rather than an error, since
// a fresh colony with no credit history is normal starting state, not
// corruption.
func learningEntryHasHelpfulApplication(guidanceID string) bool {
	if store == nil {
		return false
	}
	var file recruitmentCreditFile
	if err := store.LoadJSON(recruitmentCreditPath, &file); err != nil {
		return false
	}
	for _, rec := range file.GuidanceApplications {
		if rec.GuidanceID == guidanceID && rec.State == guidanceApplicationStateHelpful {
			return true
		}
	}
	return false
}

// promoteHelpfulHypotheses is the automatic promoter WINDOWS.md entry 44
// names: every learned entry still in hypothesis status is promoted to
// learn.StatusValidated if and only if a corroborated guidance application
// record for that entry's OWN identifier (entry.ID) has reached
// guidanceApplicationStateHelpful -- never merely rendered, consulted, or
// even acted on (acting on guidance is not the same as being helped by
// it), and never a worker's own uncorroborated claim (guidanceClaimRecord
// is a wholly separate, never-consulted table).
//
// The write itself goes through the exact same learnStore.Replace call the
// hand-run `learning-validate` command already uses
// (cmd/learning_cmds.go's learningValidateCmd) -- never a second writer --
// so `grep -rc 'StatusValidated' cmd/ --include='*.go' | grep -v _test.go`
// finds it written in exactly two places: that command, and here.
//
// Deliberately excludes a legacy entry with an entirely empty Status
// (learningUnverifiedEntries treats "hypothesis" and "" identically for
// rendering purposes -- and pkg/learn/colony_store.go's own loadEntries
// already normalizes a genuinely empty on-disk status to StatusHypothesis
// before this function (or anything else) ever sees the entry, so there is
// no distinguishable "truly empty" case left at this layer to exclude; a
// disproven entry is a separate declared status and never reaches this
// set at all).
//
// This function contains no comparison against a learn.Status* constant of
// its own -- it selects candidates entirely through
// learningUnverifiedEntries, the one shared predicate
// TestOneLearningStatusVocabulary requires every hypothesis/verified
// selection in this package to route through.
//
// Non-blocking: a read or write failure warns to stderr and is recorded
// into summary.Failures, exactly like runAutomaticImprovementPass's and
// recordPhaseApplicationCredit's own contract. phaseID is accepted (and
// unused beyond signature parity with this pass's sibling automatic steps)
// because a hypothesis's proof of helpfulness is not itself scoped to any
// one phase -- a lesson that helped once, on any phase, has stopped being
// merely a guess.
//
// CURRENT LIMITATION (WINDOWS.md entry 44, reopened by 204-REVIEW.md CR-01):
// this function runs on every check (Ran is true) but summary.Promoted can
// never be non-empty in today's running program. learningEntryHasHelpfulApplication
// keys its lookup on entry.ID, and no production writer anywhere in cmd/
// ever records a GuidanceApplications entry under a learn.Entry's own
// identifier -- every real writer keys by an Instinct's ID instead, and no
// real writer records the terminal Helpful/Neutral/Harmful states at all
// yet. See this file's package doc comment for the full grep-confirmed
// account. This is recorded, not silently accepted: fixing it for real
// requires a genuine identifier bridge or a new production writer, not a
// change to the selection/gating logic below, which is already correct.
func promoteHelpfulHypotheses(phaseID int) learningValidationSummary {
	_ = phaseID
	summary := learningValidationSummary{}
	if store == nil {
		return summary
	}
	summary.Ran = true

	learnStore := learn.NewColonyStore(store)
	entries, err := learnStore.List(learn.EntryFilter{})
	if err != nil {
		summary.recordFailure("", fmt.Sprintf("could not read the learning store: %v", err))
		return summary
	}

	for _, entry := range learningUnverifiedEntries(entries) {
		summary.Considered++

		if !learningEntryHasHelpfulApplication(entry.ID) {
			continue
		}

		entry.Status = learn.StatusValidated
		if err := learnStore.Replace(entry.ID, entry); err != nil {
			summary.recordFailure(entry.ID, fmt.Sprintf("could not promote to validated: %v", err))
			continue
		}
		summary.Promoted = append(summary.Promoted, entry.ID)
	}

	return summary
}
