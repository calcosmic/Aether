package cmd

import (
	"fmt"
	"sort"
	"strings"
)

// Recommendation-first Oracle synthesis (202-12, LIVE-07, D-10).
//
// A research run that ends with a wall of evidence and no stated
// recommendation has done the work and withheld the result -- the owner
// reads top-down and stops when they have what they need, so the order of
// the document is the product (202-CONTEXT.md D-10). The counterpart risk
// is a run that stopped early producing a document that reads like a
// settled answer.
//
// renderOracleFinalSynthesis renders the fixed section order every saved
// Oracle document now follows: the recommendation; the confidence, in
// ordinary words; what the run could not settle; the sources; then the
// round-by-round evidence trail. Every claim it renders in the
// recommendation section is copied from a finding the run actually
// recorded (oracleSynthesisConclusions), never composed fresh -- a claim on
// the page is always a claim the run made.
//
// evidenceTrail carries the existing per-template write-up
// (writeOracleSynthesisReport's output, already read by callers for the
// pre-existing empty-run check) as the document's closing section. The
// round-by-round questions/findings/sources detail that report already
// contains is not re-derived here a second time -- it is the evidence
// trail.

// oracleSynthesisConclusion is one claim the run is prepared to stand
// behind: a finding the run recorded, together with the source IDs it
// already cited. Never invented at render time -- Text and SourceIDs are
// copied straight from plan.Questions[].KeyFindings.
type oracleSynthesisConclusion struct {
	QuestionID string
	Text       string
	SourceIDs  []string
}

// oracleSynthesisConclusions collects every recorded finding across every
// question as a conclusion candidate. A finding with no source citation is
// kept (Oracle's worker responses do not always attach evidence to every
// finding) -- only a finding that cites a source ID absent from
// plan.Sources is refused, by validateOracleSynthesisConclusions.
func oracleSynthesisConclusions(plan oraclePlanFile) []oracleSynthesisConclusion {
	var out []oracleSynthesisConclusion
	for _, q := range plan.Questions {
		for _, f := range q.KeyFindings {
			text := strings.TrimSpace(f.Text)
			if text == "" {
				continue
			}
			out = append(out, oracleSynthesisConclusion{
				QuestionID: q.ID,
				Text:       text,
				SourceIDs:  append([]string(nil), f.SourceIDs...),
			})
		}
	}
	return out
}

// validateOracleSynthesisConclusions refuses to render when a conclusion
// cites a source ID plan.Sources does not actually contain -- "never state
// a conclusion the gathered sources do not support" (plan prohibition).
func validateOracleSynthesisConclusions(conclusions []oracleSynthesisConclusion, plan oraclePlanFile) error {
	for _, c := range conclusions {
		for _, sid := range c.SourceIDs {
			sid = strings.TrimSpace(sid)
			if sid == "" {
				continue
			}
			if _, ok := plan.Sources[sid]; !ok {
				return fmt.Errorf("synthesis conclusion %q cites source %q, which is not recorded in this run's sources", c.Text, sid)
			}
		}
	}
	return nil
}

// oracleSynthesisSourceRound finds the earliest round (iteration) whose
// recorded finding cited this source -- "the round that produced it." A
// source no finding currently cites reports round 0.
func oracleSynthesisSourceRound(sourceID string, plan oraclePlanFile) int {
	round := 0
	found := false
	for _, q := range plan.Questions {
		for _, f := range q.KeyFindings {
			for _, sid := range f.SourceIDs {
				if sid != sourceID {
					continue
				}
				if !found || f.Iteration < round {
					round = f.Iteration
					found = true
				}
			}
		}
	}
	return round
}

// oracleConfidenceExplanation states, in ordinary words, what the recorded
// confidence figure against its target means -- never a second number, only
// a sentence.
func oracleConfidenceExplanation(confidence, target int) string {
	switch {
	case confidence >= target && target > 0:
		return "this meets the run's own target -- the recommendation above is well-supported."
	case confidence >= 70:
		return "this is good confidence, short of the target -- worth a second look before treating it as final."
	case confidence >= 40:
		return "this is moderate confidence -- treat the recommendation as a strong lead, not a settled answer."
	default:
		return "this is low confidence -- more research is needed before acting on this."
	}
}

// oracleSynthesisGapHint translates identifyGaps's categorical reason into a
// plain-English statement of what evidence would settle the question --
// never a new investigation, only naming the kind of evidence still absent.
func oracleSynthesisGapHint(reason string) string {
	switch reason {
	case "unanswered question":
		return "a first answer with a recorded source"
	case "no findings":
		return "at least one finding backed by a source"
	case "below target confidence":
		return "more evidence to close the remaining confidence gap"
	case "open gap from state":
		return "the specific evidence named above"
	default:
		return "more evidence"
	}
}

// renderOracleFinalSynthesis renders the recommendation-first document body
// in fixed section order. Returns an error, naming the offending
// conclusion, when a recorded finding cites a source the run's own
// plan.Sources does not contain -- callers must not save a document that
// makes an unsupported claim.
func renderOracleFinalSynthesis(state oracleStateFile, plan oraclePlanFile, evidenceTrail string) (string, error) {
	conclusions := oracleSynthesisConclusions(plan)
	if err := validateOracleSynthesisConclusions(conclusions, plan); err != nil {
		return "", err
	}

	var b strings.Builder

	// --- Section 1: Recommendation ---
	b.WriteString("## Recommendation\n\n")
	if label := oracleResearchPartialLabel(state); label != "" {
		b.WriteString(label)
		b.WriteString("\n\n")
	}
	hasRecommendation := strings.TrimSpace(state.Recommendation) != "" || len(conclusions) > 0
	if !hasRecommendation {
		b.WriteString("No recommendation yet.")
		if what := strings.TrimSpace(state.Summary); what != "" {
			fmt.Fprintf(&b, " What this run did establish: %s", what)
		} else {
			b.WriteString(" This run has not gathered enough evidence to recommend anything.")
		}
		b.WriteString("\n")
	} else {
		if rec := strings.TrimSpace(state.Recommendation); rec != "" {
			b.WriteString(rec)
			b.WriteString("\n")
		}
		if len(conclusions) > 0 {
			b.WriteString("\nBecause:\n")
			for _, c := range conclusions {
				fmt.Fprintf(&b, "- %s", c.Text)
				if len(c.SourceIDs) > 0 {
					fmt.Fprintf(&b, " (%s)", strings.Join(c.SourceIDs, ", "))
				}
				b.WriteString("\n")
			}
		}
	}
	b.WriteString("\n")

	// --- Section 2: Confidence ---
	b.WriteString("## Confidence\n\n")
	fmt.Fprintf(&b, "%d%% / %d%% target -- %s\n\n", state.OverallConfidence, state.TargetConfidence, oracleConfidenceExplanation(state.OverallConfidence, state.TargetConfidence))

	// --- Section 3: What is still unsettled ---
	b.WriteString("## What Is Still Unsettled\n\n")
	gaps := identifyGaps(plan, state)
	if len(gaps) == 0 {
		b.WriteString("Nothing outstanding -- every tracked question reached its target.\n\n")
	} else {
		for _, gap := range gaps {
			fmt.Fprintf(&b, "- %s -- would be settled by %s\n", gap.Question, oracleSynthesisGapHint(gap.Reason))
		}
		b.WriteString("\n")
	}

	// --- Section 4: Sources ---
	b.WriteString("## Sources\n\n")
	if len(plan.Sources) == 0 {
		b.WriteString("None recorded.\n\n")
	} else {
		ids := make([]string, 0, len(plan.Sources))
		for id := range plan.Sources {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			src := plan.Sources[id]
			round := oracleSynthesisSourceRound(id, plan)
			fmt.Fprintf(&b, "- [%s] %s -- %s (%s, round %d)\n", id, emptyFallback(src.Title, src.URL), emptyFallback(src.URL, "no location recorded"), emptyFallback(src.Type, "codebase"), round)
		}
		b.WriteString("\n")
	}

	// --- Section 5: Round-by-round evidence trail ---
	b.WriteString("## Evidence Trail\n\n")
	trail := strings.TrimSpace(evidenceTrail)
	if trail != "" {
		b.WriteString(trail)
		b.WriteString("\n")
	} else {
		b.WriteString("No round-by-round detail recorded.\n")
	}

	return strings.TrimSpace(b.String()) + "\n", nil
}
