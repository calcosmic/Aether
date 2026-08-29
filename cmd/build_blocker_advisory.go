package cmd

import "strings"

// buildBlockerAdvisoryQuestion is the single carry-on-or-stop question D-08
// requires whenever at least one blocker signal is present and the run can
// actually be asked. It never changes with the signals present -- the
// signals themselves are what varies, one line each above this question.
const buildBlockerAdvisoryQuestion = "Carry on with the build, or stop and deal with this first?"

// buildBlockerSignal names one thing genuinely blocking a build from
// starting cleanly (D-09), plus its plain-English reason for the owner.
type buildBlockerSignal struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// buildStartBlockerSignals returns the union of exactly three predicates
// D-09 names as "hard stops" -- never FOCUS/FEEDBACK notes, never ordinary
// flags:
//
//  1. An unwaived forced reviewer still waiting on the owner.
//  2. An unanswered question -- an orchestrator boundary question, or a
//     worker's unanswered handoff question -- collapsed into one signal,
//     since D-09 names "an open owner question" as a single hard stop
//     regardless of which of the two records raised it.
//  3. The last check-and-advance (`continue`) on this phase ended blocked.
//
// The first two reuse buildHasPendingOwnerDecision's own two predicates --
// not re-derived -- so the team check-in card and this heads-up can never
// disagree about what is pending. The third is lastContinueEndedBlocked,
// reading the structured saved report.
func buildStartBlockerSignals(manifest codexBuildManifest) []buildBlockerSignal {
	var signals []buildBlockerSignal

	allForcedHits := riskSignalHitsFromRecords(manifest.ForcedReviewers)
	liveForcedHits, _ := applyForcedReviewerWaivers(manifest.Phase, allForcedHits)
	if len(liveForcedHits) > 0 {
		signals = append(signals, buildBlockerSignal{
			Name:   "forced-reviewer",
			Reason: "a forced reviewer is still waiting for the owner's check-in decision",
		})
	}

	if manifest.BoundaryQuestionCount > 0 {
		signals = append(signals, buildBlockerSignal{
			Name:   "unanswered-question",
			Reason: "an unanswered planning question is still waiting for the owner",
		})
	} else if len(pendingHandoffDecisions(manifest.Phase)) > 0 {
		signals = append(signals, buildBlockerSignal{
			Name:   "unanswered-question",
			Reason: "a worker left a question only the owner can answer",
		})
	}

	if blocked, reason := lastContinueEndedBlocked(manifest.Phase); blocked {
		signals = append(signals, buildBlockerSignal{
			Name:   "last-continue-blocked",
			Reason: reason,
		})
	}

	return signals
}

// buildBlockerAdvisory is decideBuildBlockerAdvisory's total, structured
// verdict: the named signals present (possibly none), and whether the run
// should ask the owner what to do about them.
type buildBlockerAdvisory struct {
	Signals []buildBlockerSignal
	Ask     bool
}

// decideBuildBlockerAdvisory is the pure D-08/D-09 policy, mirroring
// decideBuildCheckin's shape: printing is unconditional whenever at least
// one signal is present; only ASKING is gated by interactivity. A naive
// reuse of decideBuildCheckin's own non-interactive branch would lose this
// exact distinction -- that policy's non-interactive branch means "print
// nothing and don't ask" for the check-in card, but D-09 requires the
// heads-up to still print even when the run cannot ask.
//
// Total and side-effect free: never touches the store, never reads rendered
// text, never dispatches anything.
func decideBuildBlockerAdvisory(signals []buildBlockerSignal, nonInteractive bool) buildBlockerAdvisory {
	if len(signals) == 0 {
		return buildBlockerAdvisory{}
	}
	return buildBlockerAdvisory{
		Signals: signals,
		Ask:     !nonInteractive,
	}
}

// renderBuildBlockerAdvisory renders the D-08 heads-up in house style: an
// emoji heading with a plain-English parenthetical, one line per named
// signal (already plain English -- every repo word was translated when the
// signal was named), and the single carry-on-or-stop question when the run
// can ask. Never a bordered table. Returns "" when there is nothing to say.
func renderBuildBlockerAdvisory(advisory buildBlockerAdvisory) string {
	if len(advisory.Signals) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("🚧 Something is stuck (this build is starting, but here's what's still outstanding)\n")
	for _, signal := range advisory.Signals {
		b.WriteString("  - " + signal.Reason + "\n")
	}
	if advisory.Ask {
		b.WriteString("\n" + buildBlockerAdvisoryQuestion + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
