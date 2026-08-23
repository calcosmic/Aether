package cmd

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// This file holds the owner's ONLY way to decline a forced reviewer (D-03,
// .planning/phases/194-the-queen-decides-the-team/194-CONTEXT.md). It follows
// cmd/criterion_owner_confirmation.go line for line in structure: a stable
// deterministic question text, a matcher against resolved decision-answer
// entries, and a shell-quoted command shown to the owner. Recording an
// answer happens only through the existing `aether decision-answer` command
// (cmd/handoff_decisions_cmd.go), the same path a worker's open_decisions
// question already goes through. T-194-11.
//
// CR-01 (194-REVIEW.md) closed the gap in that last sentence: before this
// fix, `aether decision-answer` had no authentication and no requirement
// that its --question text match anything the program had actually shown
// anyone -- since forcedReviewerWaiverQuestionText is fully deterministic
// from public information (phaseID + one of five fixed PlainEnglish
// strings, all readable in queenRiskSignalTable or the rendered check-in
// card), ANY process able to invoke the `aether` binary could forge the
// exact command and silently waive a reviewer with no owner ever having
// seen the question. The fix is two functions in this file plus one call
// added at the CLI boundary (decisionAnswerCmd.RunE,
// cmd/handoff_decisions_cmd.go):
//
//  1. ensureForcedReviewerWaiverPendingDecision writes the pending,
//     UNRESOLVED question into pending-decisions.json the moment (and ONLY
//     the moment) the check-in card renders a LIVE forced reviewer
//     (cmd/ceremony_team_checkin.go) -- this is the runtime's own record
//     that it actually showed this exact question to whoever is looking at
//     the card.
//  2. resolveForcedReviewerWaiverPendingDecision is the only way a
//     forced-reviewer-shaped --question can be answered: it looks up that
//     SAME pending row and marks it resolved. It never creates a new
//     resolved entry the way the general clarification path
//     (recordDecisionAnswer) does -- so a --question with no matching
//     runtime-created row is refused and nothing is recorded.
//
// TestDecisionAnswerCannotForgeAForcedReviewerWaiver
// (cmd/forced_reviewer_waiver_test.go) proves the forgery path this closes:
// calling the decision-answer COMMAND (not just recordDecisionAnswer) with a
// correctly-shaped question and no prior card render still leaves the
// forced reviewer in the dispatch list.

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

// ensureForcedReviewerWaiverPendingDecision writes the pending, UNRESOLVED
// decision row the owner's decline command (forcedReviewerWaiverCommand)
// can later resolve -- CR-01's fix (194-REVIEW.md). Called ONLY from the
// check-in card's render step (cmd/ceremony_team_checkin.go), once per LIVE
// forced hit -- a hit applyForcedReviewerWaivers has already filtered out
// (an already-waived signal) is never re-offered a fresh row here. Writing
// a row that already exists for this exact phase+signal question (pending
// OR already resolved) is a no-op: the card can render many times over a
// session, and this must stay idempotent rather than piling up duplicate
// rows on every render.
func ensureForcedReviewerWaiverPendingDecision(phaseID int, signalName, plainEnglish string) error {
	if store == nil {
		return nil
	}
	question := forcedReviewerWaiverQuestionText(phaseID, signalName, plainEnglish)
	target := normalizeDecisionText(question)
	if target == "" {
		return nil
	}

	file := loadPendingDecisionFile()
	for _, d := range file.Decisions {
		q, _ := parseClarificationDescription(d.Description)
		if normalizeDecisionText(q) == target || normalizeDecisionText(d.Description) == target {
			return nil // already exists (pending or resolved) -- nothing to do
		}
	}

	decision := PendingDecision{
		ID:          fmt.Sprintf("pd_%d", time.Now().UnixNano()),
		Type:        clarificationDecisionType,
		Description: formatClarificationDescription(question, nil),
		Source:      "forced-reviewer-waiver",
		Resolved:    false,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	if phaseID > 0 {
		decision.Phase = &phaseID
	}
	stampPendingDecisionScope(&decision, loadCurrentPendingDecisionScope())

	file.Decisions = append(file.Decisions, decision)
	return store.SaveJSON(pendingDecisionsFile, file)
}

// forcedReviewerWaiverPhasePrefix parses the phase number
// forcedReviewerWaiverQuestionText embeds at the start of every waiver
// question ("Phase 7: ..."). Used by forcedReviewerWaiverSignalForQuestion
// to recover the phase a candidate --question text claims to be about,
// independent of (and never trusting) any --phase flag the caller also
// passed.
var forcedReviewerWaiverPhasePrefix = regexp.MustCompile(`^Phase (\d+):`)

// forcedReviewerWaiverSignalForQuestion reports whether a candidate
// question text is shaped like a forced-reviewer waiver question, and if
// so, which phase and signal it names. This is CR-01's forgery check: the
// ONLY way to be sure a --question is (or is not) this shape is to
// regenerate the exact deterministic sentence the runtime would have
// produced for every signal at the phase the text itself claims, and
// compare -- never to pattern-match the prose, which a forger could dress
// up to look close enough. The phase is read from the question text itself
// (not a --phase flag the caller also supplied), because a forger could
// set an unrelated or absent --phase to try to dodge this check.
func forcedReviewerWaiverSignalForQuestion(question string) (phaseID int, signalName string, ok bool) {
	question = strings.TrimSpace(question)
	m := forcedReviewerWaiverPhasePrefix.FindStringSubmatch(question)
	if m == nil {
		return 0, "", false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, "", false
	}
	target := normalizeDecisionText(question)
	for _, s := range queenRiskSignalTable {
		candidate := forcedReviewerWaiverQuestionText(n, s.Name, s.PlainEnglish)
		if normalizeDecisionText(candidate) == target {
			return n, s.Name, true
		}
	}
	return 0, "", false
}

// resolveForcedReviewerWaiverPendingDecision is the ONLY way a
// forced-reviewer-shaped question can be answered (CR-01, 194-REVIEW.md):
// it finds the SINGLE pending, unresolved row the runtime itself created
// for this exact phase+question
// (ensureForcedReviewerWaiverPendingDecision, called from the check-in
// card's render step) and marks it resolved with the owner's answer.
// Unlike recordDecisionAnswer, this function NEVER creates a new entry --
// if no matching unresolved row exists, the waiver did not originate from
// something the runtime actually showed anyone, and is refused. found is
// false in that case; no file write happens.
func resolveForcedReviewerWaiverPendingDecision(question, answer string, phaseID int) (decision PendingDecision, found bool, err error) {
	if store == nil {
		return PendingDecision{}, false, nil
	}
	target := normalizeDecisionText(question)
	if target == "" {
		return PendingDecision{}, false, nil
	}

	var file PendingDecisionFile
	if loadErr := store.LoadJSON(pendingDecisionsFile, &file); loadErr != nil {
		return PendingDecision{}, false, nil
	}
	for i := range file.Decisions {
		d := &file.Decisions[i]
		if d.Resolved {
			continue
		}
		if phaseID > 0 && (d.Phase == nil || *d.Phase != phaseID) {
			continue
		}
		q, _ := parseClarificationDescription(d.Description)
		if normalizeDecisionText(q) != target && normalizeDecisionText(d.Description) != target {
			continue
		}
		d.Resolved = true
		d.Resolution = answer
		d.ResolvedAt = time.Now().UTC().Format(time.RFC3339)
		if saveErr := store.SaveJSON(pendingDecisionsFile, file); saveErr != nil {
			return PendingDecision{}, false, saveErr
		}
		return *d, true, nil
	}
	return PendingDecision{}, false, nil
}
