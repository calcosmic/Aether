package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// criterionStateNeedsOwnerConfirmation marks a success criterion that no
// deterministic source could prove and no reviewer was dispatched to judge
// (D-05, 193-CONTEXT.md): the phase still advances, but `aether seal` blocks
// until the owner has answered it through the existing decision-answer path
// (`aether decision-answer`) -- no reviewer worker is ever spawned for it.
const criterionStateNeedsOwnerConfirmation = "needs_owner_confirmation"

// ownerConfirmationQuestionText builds the stable, deterministic question
// text an outstanding owner-confirmation criterion is recorded and matched
// against. It is designed to be typed back verbatim through
// `aether decision-answer --question "<this text>" --answer "..."`, and
// matched the same way a worker's open_decisions question is matched:
// normalizeDecisionText (via answeredDecisionTexts/handoffDecisionID) is the
// single shared normalization both paths already use, so an owner's answer
// resolves this criterion the same way it resolves a worker's question.
func ownerConfirmationQuestionText(phaseID int, taskID, criterion string) string {
	criterion = strings.TrimSpace(criterion)
	taskID = strings.TrimSpace(taskID)
	if taskID != "" {
		return fmt.Sprintf("Phase %d task %s: no program check or reviewer could verify %q -- please confirm it is actually true.", phaseID, taskID, criterion)
	}
	return fmt.Sprintf("Phase %d: no program check or reviewer could verify %q -- please confirm it is actually true.", phaseID, criterion)
}

// ownerConfirmationAnswered reports whether the owner has already answered
// this criterion's outstanding confirmation, by matching its stable question
// text against the resolved pending-decision entries the same way a worker's
// open_decisions question is matched (answeredDecisionTexts).
func ownerConfirmationAnswered(phaseID int, taskID, criterion string) bool {
	answered := answeredDecisionTexts()
	return answered[normalizeDecisionText(ownerConfirmationQuestionText(phaseID, taskID, criterion))]
}

// shellQuote wraps s in single quotes so it is safe to paste into a POSIX
// shell command line, escaping any embedded single quote as the standard
// close-quote/escaped-literal-quote/reopen-quote sequence: a single quote, a
// backslash-escaped single quote, then a single quote. That sequence is not
// written out literally here because gofmt rewrites an adjacent pair of single
// quotes inside a comment into a Unicode right double quote, which would
// silently corrupt it. See shellQuote's body and its tests for the exact
// bytes. This is
// deliberately NOT Go's %q: %q escapes for Go source syntax, not a shell --
// it leaves $, backticks, and other shell metacharacters untouched, so a
// criterion's own free-form text (authored by the planning LLM, possibly
// influenced by external content) could otherwise expand a variable or
// execute a command substitution when the reader -- explicitly
// non-technical, per this repo's own CLAUDE.md -- copies the shown command
// into a terminal (CR-02, 193-REVIEW.md).
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ownerConfirmationCommand is the exact `aether decision-answer` invocation
// the owner runs to resolve one outstanding criterion -- surfaced verbatim
// by the owner_confirmation_pending gate's RecoveryOptions and by the seal
// blocker text, so the owner never has to construct the question text by
// hand. The question text is shell-quoted (shellQuote), not Go-quoted
// (%q), because it embeds untrusted criterion text into a command the
// owner is told to paste into a real shell (CR-02, 193-REVIEW.md).
func ownerConfirmationCommand(phaseID int, taskID, criterion string) string {
	return fmt.Sprintf("aether decision-answer --question %s --answer 'confirmed' --phase %d", shellQuote(ownerConfirmationQuestionText(phaseID, taskID, criterion)), phaseID)
}

// isLastPhaseOfActivePlan reports whether phaseID is the last phase of the
// currently active colony plan -- the point at which advancing a phase and
// sealing the colony are the same act, so the owner_confirmation_pending gate
// (cmd/codex_continue.go) must actually be able to block there. Loads the
// active colony state itself (rather than adding a colony.ColonyState
// parameter to runCodexContinueGates, which has 20 existing call sites)
// since store is a package-level global the same way loadActiveColonyState's
// other callers already assume.
func isLastPhaseOfActivePlan(phaseID int) bool {
	state, err := loadActiveColonyState()
	if err != nil || len(state.Plan.Phases) == 0 {
		return false
	}
	return state.Plan.Phases[len(state.Plan.Phases)-1].ID == phaseID
}

// outstandingOwnerConfirmations returns every criterion in criteria whose
// State is criterionStateNeedsOwnerConfirmation and which the owner has not
// yet answered through the decision-answer path.
func outstandingOwnerConfirmations(phaseID int, criteria []codexCriterionVerification) []codexCriterionVerification {
	var outstanding []codexCriterionVerification
	for _, c := range criteria {
		if c.State != criterionStateNeedsOwnerConfirmation {
			continue
		}
		if ownerConfirmationAnswered(phaseID, c.TaskID, c.Criterion) {
			continue
		}
		outstanding = append(outstanding, c)
	}
	return outstanding
}

// ownerConfirmationSealBlockers scans every phase's persisted continue
// verification report (build/phase-<N>/verification.json) for an outstanding
// needs_owner_confirmation criterion the owner has not yet answered, and
// returns them shaped as colony.FlagEntry blockers so they flow through the
// exact same checkSealBlockers/renderBlockerSummary/renderRecoveryMenu route
// every other seal blocker already uses (D-05) -- reusing the existing
// --force/--reason override contract rather than inventing a second one.
// Computed live, not persisted: there is nothing to keep in sync when
// ownerConfirmationAnswered later reports the owner has resolved one.
func ownerConfirmationSealBlockers(state colony.ColonyState) []colony.FlagEntry {
	if store == nil {
		return nil
	}
	var blockers []colony.FlagEntry
	for _, phase := range state.Plan.Phases {
		verificationRel := continuePlanArtifactsPath(phase.ID, "verification.json")
		var report codexContinueVerificationReport
		if err := store.LoadJSON(verificationRel, &report); err != nil {
			continue
		}
		phaseID := phase.ID
		for _, c := range outstandingOwnerConfirmations(phase.ID, report.Criteria) {
			command := ownerConfirmationCommand(phase.ID, c.TaskID, c.Criterion)
			blockers = append(blockers, colony.FlagEntry{
				ID:          fmt.Sprintf("owner-confirm-%d-%s", phase.ID, handoffDecisionID(ownerConfirmationQuestionText(phase.ID, c.TaskID, c.Criterion))),
				Type:        "blocker",
				Description: fmt.Sprintf("Phase %d: %q needs your confirmation -- no program check or reviewer could verify it. Run: %s", phase.ID, c.Criterion, command),
				Phase:       &phaseID,
				Source:      "owner_confirmation",
				CreatedAt:   report.GeneratedAt,
				// This blocker is computed live, never persisted to
				// pending-decisions.json (see this function's doc comment),
				// so the generic `aether flag-resolve --id <ID>` line
				// renderBlockerSummary would otherwise print can never
				// resolve it -- carry its real recovery command instead
				// (WR-01, 193-REVIEW.md).
				RecoveryCommand: command,
			})
		}
	}
	return blockers
}
