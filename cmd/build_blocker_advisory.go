package cmd

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
