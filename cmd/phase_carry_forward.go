package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// phaseCarryForwardBudgetChars bounds the failed/flagged item list inside
// resolvePreviousPhaseCarryForward. This is its OWN allowance, deliberately
// kept outside colonyPrimeBudgetChars / colonyPrimeCompactBudgetChars
// (cmd/colony_prime_context.go) and outside surveyDigestBudgetChars
// (cmd/helpers.go) -- the folded worker-turnaround todo caps total brief
// growth at exactly these two new named slots (D-01, D-09), so this
// constant must never draw from either existing budget. The persisted
// closing summary (outcome.md, D-11) is appended after this budget is spent
// and is never truncated against it -- D-11 requires it word-for-word, and a
// truncated "closing summary" would no longer be the text the owner read.
const phaseCarryForwardBudgetChars = 2000

// resolvePreviousPhaseCarryForward tells a builder starting a new phase what
// happened in the phase immediately before it (WIRE-07, D-09..D-11): what
// failed, what a reviewer flagged, and the closing summary the owner read --
// never the full verification.json / review.json reports those come from.
//
// It resolves the immediately preceding phase from the plan's own phase
// order -- the phase at position i-1 where the current phase sits at
// position i, not currentPhaseID-1, since phase IDs are not always
// contiguous. Records are then read through the store using the same
// read-and-guard shape criterion_owner_confirmation.go already uses for
// verification.json: a missing or unreadable file is skipped, never an
// error.
//
// From the verification report, only failed (non-skipped) checks and unmet
// or unconfirmed criteria are selected. From the review report, only
// blocking issues are selected. Each becomes one sentence. When nothing
// failed and nothing was flagged, the section collapses to a single line
// naming how many checks passed and that the review was clean -- never the
// full reports either way. The persisted outcome.md text (D-11), when
// present, is appended verbatim as the closing summary, labelled in plain
// English.
//
// Every sentence drawn from verification.json/review.json is sanitised
// through colony.SanitizeSignalContent before entering the section --
// reviewer- and worker-authored report text is untrusted input on its way
// into a new helper's instructions (T-198.2-22), the same boundary 198.1
// established for replayed failure-log text. A sentence the sanitiser
// rejects is named in the omission line, never silently dropped, matching
// the omission handling for a sentence that does not fit the budget.
//
// Returns the empty string when there is no preceding phase, or the
// preceding phase has no persisted verification.json, review.json or
// outcome.md at all (WIRE-07 unclassified truth: no carry-forward section
// at all, not an empty heading).
func resolvePreviousPhaseCarryForward(currentPhaseID int) string {
	if store == nil {
		return ""
	}
	state, err := loadActiveColonyState()
	if err != nil {
		return ""
	}
	prevID, ok := precedingPhaseID(state, currentPhaseID)
	if !ok {
		return ""
	}

	var verification codexContinueVerificationReport
	hasVerification := store.LoadJSON(continuePlanArtifactsPath(prevID, "verification.json"), &verification) == nil

	var review codexContinueReviewReport
	hasReview := store.LoadJSON(continuePlanArtifactsPath(prevID, "review.json"), &review) == nil

	outcomeBytes, outcomeErr := store.ReadFile(continuePlanArtifactsPath(prevID, "outcome.md"))
	outcomeText := strings.TrimSpace(string(outcomeBytes))
	hasOutcome := outcomeErr == nil && outcomeText != ""

	if !hasVerification && !hasReview && !hasOutcome {
		return ""
	}

	type flaggedItem struct {
		label    string
		sentence string
	}
	var items []flaggedItem
	passedCount := 0

	if hasVerification {
		for _, step := range verification.Steps {
			if step.Skipped {
				continue
			}
			if step.Passed {
				passedCount++
				continue
			}
			items = append(items, flaggedItem{
				label:    step.Name,
				sentence: fmt.Sprintf("Check %q failed: %s", step.Name, firstNonEmpty(step.Summary, "no summary recorded")),
			})
		}
		for _, c := range verification.Criteria {
			if c.Passed && c.State == "" {
				continue
			}
			if c.Passed {
				continue
			}
			items = append(items, flaggedItem{
				label:    c.Criterion,
				sentence: fmt.Sprintf("Criterion %q was not met: %s", c.Criterion, firstNonEmpty(c.Summary, "not met")),
			})
		}
	}

	if hasReview {
		for _, issue := range review.BlockingIssues {
			items = append(items, flaggedItem{
				label:    "review",
				sentence: fmt.Sprintf("Review flagged: %s", issue),
			})
		}
	}

	var body strings.Builder
	fmt.Fprintf(&body, "## What Happened Last Phase (Phase %d)\n\n", prevID)

	if len(items) == 0 {
		if hasVerification || hasReview {
			line := fmt.Sprintf("Phase %d: all %d checks passed", prevID, passedCount)
			if hasReview {
				line += ", review clean."
			} else {
				line += "."
			}
			body.WriteString(line)
			body.WriteString("\n")
		}
	} else {
		used := 0
		var omitted []string
		for _, item := range items {
			sanitized, sanErr := colony.SanitizeSignalContent(item.sentence)
			if sanErr != nil {
				omitted = append(omitted, fmt.Sprintf("%s (content rejected: %v)", item.label, sanErr))
				continue
			}
			remaining := phaseCarryForwardBudgetChars - used
			if remaining <= 0 || len(sanitized) > remaining {
				omitted = append(omitted, fmt.Sprintf("%s (over budget)", item.label))
				continue
			}
			fmt.Fprintf(&body, "- %s\n", sanitized)
			used += len(sanitized)
		}
		if len(omitted) > 0 {
			fmt.Fprintf(&body, "_Not included here — read directly if relevant: %s_\n", strings.Join(omitted, ", "))
		}
	}

	if hasOutcome {
		body.WriteString("\nWhat the owner was told when that phase finished:\n\n")
		body.WriteString(outcomeText)
		body.WriteString("\n")
	}

	return strings.TrimSpace(body.String())
}

// precedingPhaseID returns the phase ID at the position immediately before
// currentPhaseID within state.Plan.Phases -- the phase's own recorded order,
// not currentPhaseID-1, since phase IDs are not always contiguous in this
// repository's own plans (e.g. 198, 198.1, 198.2 are distinct integer IDs
// with no arithmetic relationship). Returns ok=false when currentPhaseID is
// the first phase, or is not found in the plan at all.
func precedingPhaseID(state colony.ColonyState, currentPhaseID int) (int, bool) {
	phases := state.Plan.Phases
	for i, p := range phases {
		if p.ID != currentPhaseID {
			continue
		}
		if i == 0 {
			return 0, false
		}
		return phases[i-1].ID, true
	}
	return 0, false
}
