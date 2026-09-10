package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Before this file existed, "does reviewer judgement land at build-end or at
// the check step" was never a single fact -- it was whatever the build-side
// caste selection happened to send, independently reconciled a second time by
// whatever the check-side caste selection happened to send (SYN-201-03). Two
// independently computed sets that usually agree still drift the same way a
// second independently-derived boundary would, which is exactly why the same
// phase can end up reviewed twice. Recording the choice once, with a reason,
// is what lets a later plan gate both review passes on one stored fact
// instead of two guesses that happen to match today.
//
// verificationBoundaryChoiceCheckStep and verificationBoundaryChoiceBuildEnd
// are the closed vocabulary for where reviewer judgement lands.
// verificationBoundaryChoiceCheckStep is the default (D-01, D-02): with no
// strong signal, review happens at `aether continue`, not at build end.
const (
	verificationBoundaryChoiceCheckStep = "check_step"
	verificationBoundaryChoiceBuildEnd  = "build_end"
)

// verificationBoundarySourceQueen and verificationBoundarySourceDeterministic
// mirror queenCasteJudgement.Source (cmd/queen_judgement.go): "queen" when a
// proposal was supplied and reconciled, "deterministic" when no proposal
// arrived and the default applies.
const (
	verificationBoundarySourceQueen         = "queen"
	verificationBoundarySourceDeterministic = "deterministic"
)

// verificationBoundaryDecision is the reconciled, validated record of where
// reviewer judgement lands for one phase attempt. Choice and Source are
// always populated; Reason, Refused, and RefusedWhy are empty/false unless
// they apply.
type verificationBoundaryDecision struct {
	// Choice is one of verificationBoundaryChoiceCheckStep or
	// verificationBoundaryChoiceBuildEnd -- never any other value.
	Choice string `json:"choice"`
	// Reason is the plain-English reason a "queen" source proposal carried.
	// Empty for a "deterministic" source, and empty for a refused proposal
	// (RefusedWhy carries that story instead).
	Reason string `json:"reason,omitempty"`
	// Source is verificationBoundarySourceQueen or
	// verificationBoundarySourceDeterministic.
	Source string `json:"source"`
	// Refused is true when the proposal could not be honoured as offered
	// (an unreasoned build-end request, or an unrecognised choice) and the
	// decision fell back to the check-step default instead.
	Refused bool `json:"refused,omitempty"`
	// RefusedWhy names what was missing or wrong with the proposal. Set only
	// when Refused is true.
	RefusedWhy string `json:"refused_why,omitempty"`
}

// Summary renders the decision as a line a non-specialist can read, following
// the plain-English style of queenCasteJudgement.Summary(). It carries no
// list fields, so two calls on the same decision value always produce the
// same string -- there is nothing for iteration order to disturb.
func (d verificationBoundaryDecision) Summary() string {
	var b strings.Builder
	if d.Choice == verificationBoundaryChoiceBuildEnd {
		b.WriteString("Review lands at build end")
	} else {
		b.WriteString("Review lands at the check step")
	}
	if d.Source == verificationBoundarySourceQueen {
		b.WriteString(" -- the Queen's choice")
	} else {
		b.WriteString(" -- the default with no proposal offered")
	}
	if strings.TrimSpace(d.Reason) != "" {
		b.WriteString(fmt.Sprintf(": %s", d.Reason))
	}
	b.WriteString(".")
	if d.Refused {
		b.WriteString(fmt.Sprintf(" A proposal was refused: %s.", d.RefusedWhy))
	}
	return b.String()
}

// queenApplyVerificationBoundary reconciles the Queen's proposed boundary
// choice into a validated verificationBoundaryDecision. It is a pure
// function: it never writes state, never dispatches, and never consults a
// verification-depth flag -- phase and state are accepted for parity with
// queenApplyJudgement's call shape and future evidence-bearing rules, not
// consulted by the current reconciliation logic.
//
// An empty proposal is not an error: it means no judgement was offered, so
// the check-step default applies (source "deterministic", D-01). A proposal
// is normalized by trimming surrounding whitespace and lowercasing ASCII
// letters, then compared for EXACT equality against the two known choices --
// no folding, no prefix matching, no Unicode normalisation of any kind. A
// build-end proposal requires a non-blank reason after trimming (D-02); with
// none, the proposal is refused BY NAME and the decision falls back to the
// check step. An unrecognised proposal string is refused the same way.
func queenApplyVerificationBoundary(proposed string, reason string, phase colony.Phase, state colony.ColonyState) verificationBoundaryDecision {
	normalized := strings.ToLower(strings.TrimSpace(proposed))
	trimmedReason := strings.TrimSpace(reason)

	if normalized == "" {
		return verificationBoundaryDecision{
			Choice: verificationBoundaryChoiceCheckStep,
			Source: verificationBoundarySourceDeterministic,
		}
	}

	switch normalized {
	case verificationBoundaryChoiceCheckStep:
		return verificationBoundaryDecision{
			Choice: verificationBoundaryChoiceCheckStep,
			Source: verificationBoundarySourceQueen,
			Reason: trimmedReason,
		}
	case verificationBoundaryChoiceBuildEnd:
		if trimmedReason == "" {
			return verificationBoundaryDecision{
				Choice:     verificationBoundaryChoiceCheckStep,
				Source:     verificationBoundarySourceDeterministic,
				Refused:    true,
				RefusedWhy: "build-end review requires a stated reason, and none was given",
			}
		}
		return verificationBoundaryDecision{
			Choice: verificationBoundaryChoiceBuildEnd,
			Source: verificationBoundarySourceQueen,
			Reason: trimmedReason,
		}
	default:
		return verificationBoundaryDecision{
			Choice:     verificationBoundaryChoiceCheckStep,
			Source:     verificationBoundarySourceDeterministic,
			Refused:    true,
			RefusedWhy: fmt.Sprintf("%q is not a recognised verification boundary choice", strings.TrimSpace(proposed)),
		}
	}
}

// verificationBoundaryForAttempt is the single read path every consumer of a
// recorded verification-boundary decision uses. It reads the stored decision
// for the exact attempt and returns ok=false when none has been recorded yet
// (including for an attempt written before this field existed) -- it never
// calls queenApplyVerificationBoundary to recompute one. Consumers read;
// they never derive (TestOneFunctionDerivesTheVerificationBoundary).
func verificationBoundaryForAttempt(attemptRel string) (verificationBoundaryDecision, bool) {
	if store == nil || strings.TrimSpace(attemptRel) == "" {
		return verificationBoundaryDecision{}, false
	}
	var record buildAttemptRecord
	if err := store.LoadJSON(attemptRel, &record); err != nil {
		return verificationBoundaryDecision{}, false
	}
	if record.VerificationBoundary == nil {
		return verificationBoundaryDecision{}, false
	}
	return *record.VerificationBoundary, true
}
