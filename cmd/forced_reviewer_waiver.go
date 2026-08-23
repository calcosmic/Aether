package cmd

import (
	"fmt"
	"strings"
)

// This file holds the owner's ONLY way to decline a forced reviewer (D-03,
// .planning/phases/194-the-queen-decides-the-team/194-CONTEXT.md). It follows
// cmd/criterion_owner_confirmation.go line for line in structure: a stable
// deterministic question text, a matcher against resolved decision-answer
// entries, and a shell-quoted command shown to the owner -- no state of its
// own. This module never writes a decision; recording one happens only
// through the existing `aether decision-answer` command
// (cmd/handoff_decisions_cmd.go), the same path a worker's open_decisions
// question already goes through. T-194-11.

// forcedReviewerWaiverReviewerLabel names the reviewer a signal would force,
// the way the owner reads it -- "security reviewer" or "quality reviewer",
// never the registry identifier (gatekeeper/auditor). Reuses
// forcedReviewerPlainLabel (cmd/codex_build.go) via the signal's own Caste so
// this sentence and the pre-build announcement never describe the same
// reviewer two different ways.
func forcedReviewerWaiverReviewerLabel(signal string) string {
	signal = strings.TrimSpace(signal)
	for _, s := range queenRiskSignalTable {
		if s.Name == signal {
			return forcedReviewerPlainLabel(s.Caste)
		}
	}
	return "reviewer"
}

// forcedReviewerWaiverQuestionText builds the stable, deterministic sentence
// a waiver answer is matched against, in the shape: "Phase 7: a security
// reviewer is being added because this touches logins and passwords. Waive
// it?". phaseID and plainEnglish (unique per signal -- see
// queenRiskSignalTable's own PlainEnglish values) together make this text
// unique per phase AND per signal: that uniqueness IS the scoping rule in
// D-03, exactly as ownerConfirmationQuestionText scopes by phase and task.
// signal only selects the reviewer label (forcedReviewerWaiverReviewerLabel)
// -- the raw signal identifier (e.g. "credentials/auth") never appears in
// text the owner is shown, per CLAUDE.md's plain-English mandate.
func forcedReviewerWaiverQuestionText(phaseID int, signal string, plainEnglish string) string {
	plainEnglish = strings.TrimSpace(plainEnglish)
	return fmt.Sprintf(
		"Phase %d: a %s is being added because this touches %s. Waive it?",
		phaseID, forcedReviewerWaiverReviewerLabel(signal), plainEnglish,
	)
}

// forcedReviewerWaiverCommand is the exact `aether decision-answer`
// invocation the owner runs to decline one forced reviewer -- surfaced
// verbatim on the check-in card so the owner never has to construct the
// question text by hand. The question text is shell-quoted (shellQuote,
// cmd/criterion_owner_confirmation.go), not Go-quoted (%q), because it is
// pasted into a real shell by a reader this repo's own CLAUDE.md describes as
// non-technical (CR-02, 193-REVIEW.md; T-194-12). The answer is a
// placeholder the owner replaces with their own reason -- D-03 requires the
// reason be written down, and this module never writes one on the owner's
// behalf.
func forcedReviewerWaiverCommand(phaseID int, signal string, plainEnglish string) string {
	question := forcedReviewerWaiverQuestionText(phaseID, signal, plainEnglish)
	return fmt.Sprintf(
		"aether decision-answer --question %s --answer '<say why here>' --phase %d",
		shellQuote(question), phaseID,
	)
}

// forcedReviewerWaiver reports whether the owner has already answered this
// phase-and-signal's waiver question through the decision-answer path, and
// the owner's recorded reason. It matches the normalised question text
// against resolved pending-decision entries the SAME way
// answeredDecisionTexts/ownerConfirmationAnswered already do (matching either
// the parsed clarification question or the raw description) -- reading the
// pending-decision file directly rather than introducing a second
// normalisation, because answeredDecisionTexts only returns presence and
// this caller also needs the reason text for the card (D-03: "the card shows
// waived by owner: <reason>").
func forcedReviewerWaiver(phaseID int, signal string) (waived bool, reason string) {
	signal = strings.TrimSpace(signal)
	plainEnglish := ""
	for _, s := range queenRiskSignalTable {
		if s.Name == signal {
			plainEnglish = s.PlainEnglish
			break
		}
	}
	if plainEnglish == "" {
		return false, ""
	}
	target := normalizeDecisionText(forcedReviewerWaiverQuestionText(phaseID, signal, plainEnglish))
	if target == "" {
		return false, ""
	}
	file := loadPendingDecisionFile()
	for _, decision := range file.Decisions {
		if !decision.Resolved || strings.TrimSpace(decision.Resolution) == "" {
			continue
		}
		question, _ := parseClarificationDescription(decision.Description)
		if normalizeDecisionText(question) == target || normalizeDecisionText(decision.Description) == target {
			return true, strings.TrimSpace(decision.Resolution)
		}
	}
	return false, ""
}

// applyForcedReviewerWaivers filters hits at the SIGNAL level -- before
// collapseToForcedReviewers merges signals into castes -- so a waived
// credentials signal on a phase that also carries a live payments signal
// still forces the security reviewer, with a reason naming payments alone
// (D-03's "one signal for one phase" rule; key_links in
// .planning/phases/194-the-queen-decides-the-team/194-07-PLAN.md). Called
// from inside queenForcedContinueReviewers (cmd/queen_risk_signals.go) --
// the one function both continue lanes already go through -- so the waiver
// applies identically on the in-process and wrapper lanes with no second
// implementation, and because that function unions plan-wording hits with
// changed-file hits BEFORE this filter runs, a signal waived from plan
// wording stays waived when the same signal is re-detected from the files
// the builder changed.
func applyForcedReviewerWaivers(phaseID int, hits []riskSignalHit) (kept []riskSignalHit, waived []riskSignalHit) {
	for _, hit := range hits {
		if ok, reason := forcedReviewerWaiver(phaseID, hit.Signal.Name); ok {
			hit.WaiverReason = reason
			waived = append(waived, hit)
			continue
		}
		kept = append(kept, hit)
	}
	return kept, waived
}
